// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log"
	"sort"

	"github.com/goki/ki/kit"
)

// KeyFunctions are functions that keyboard events can perform in the GUI --
// seems possible to keep this flat and consistent across different contexts,
// as long as the functions can be appropriately reinterpreted for each
// context.
type KeyFunctions int64

const (
	KeyFunNil KeyFunctions = iota
	KeyFunMoveUp
	KeyFunMoveDown
	KeyFunMoveRight
	KeyFunMoveLeft
	KeyFunPageUp
	KeyFunPageDown
	KeyFunPageRight
	KeyFunPageLeft
	KeyFunHome
	KeyFunEnd
	KeyFunFocusNext
	KeyFunFocusPrev
	KeyFunSelectItem // enter
	KeyFunCancelSelect
	KeyFunSelectMode
	KeyFunSelectAll
	KeyFunAccept // accept any changes and close dialog / move to next
	KeyFunAbort
	KeyFunEditItem
	KeyFunCopy
	KeyFunCut
	KeyFunPaste
	KeyFunBackspace
	KeyFunDelete
	KeyFunKill
	KeyFunDuplicate
	KeyFunUndo
	KeyFunRedo
	KeyFunInsert
	KeyFunInsertAfter
	KeyFunGoGiEditor
	// either shift key
	KeyFunShift
	// the control key: command for mac, ctrl for others?
	KeyFunCtrl
	KeyFunZoomOut
	KeyFunZoomIn
	KeyFunPrefs
	KeyFunRefresh
	KeyFunComplete
	KeyFunctionsN
)

//go:generate stringer -type=KeyFunctions

var KiT_KeyFunctions = kit.Enums.AddEnumAltLower(KeyFunctionsN, false, StylePropProps, "KeyFun")

func (ev KeyFunctions) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *KeyFunctions) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// KeyMap is a map between a key sequence (chord) and a specific KeyFun
// function.  This mapping must be unique, in that each chord has unique
// KeyFun, but multiple chords can trigger the same function.
type KeyMap map[string]KeyFunctions

// SetActiveKeyMap sets the current ActiveKeyMap, calling Update on the map
// prior to setting it to ensure that it is a valid, complete map
func SetActiveKeyMap(km *KeyMap) {
	km.Update()
	ActiveKeyMap = km
}

// KeyFun translates chord into keyboard function -- use oswin key.ChordString
// to get chord
func KeyFun(chord string) KeyFunctions {
	kf := KeyFunNil
	if chord != "" {
		kf = (*ActiveKeyMap)[chord]
		// fmt.Printf("chord: %v = %v\n", chord, kf)
	}
	return kf
}

// StdKeyMaps collects a list of standard keymap options
var StdKeyMaps = []*KeyMap{
	&MacKeyMap,
	&MacEmacsKeyMap,
	&LinuxKeyMap,
	&LinuxEmacsKeyMap,
	&ChromeKeyMap,
}

// StdKeyMapNames lists names of maps in same order as StdKeyMaps
var StdKeyMapNames = []string{
	"MacKeyMap",
	"MacEmacsKeyMap",
	"LinuxKeyMap",
	"LinuxEmacsKeyMap",
	"ChromeKeyMap",
}

// DefaultKeyMap is the overall default keymap -- reinitialized in gimain init()
// depending on platform
var DefaultKeyMap = &MacEmacsKeyMap

// ActiveKeyMap points to the active map -- users can set this to an
// alternative map in Prefs
var ActiveKeyMap = DefaultKeyMap

// StdKeyMapByName returns a standard keymap and index within StdKeyMaps* tables by name.
func StdKeyMapByName(name string) (*KeyMap, int) {
	for i, nm := range StdKeyMapNames {
		if nm == name {
			return StdKeyMaps[i], i
		}
	}
	fmt.Printf("gi.StdKeyMapByName key map named: %v not found\n", name)
	return nil, -1
}

// StdKeyMapName returns name of a standard key map
func StdKeyMapName(km *KeyMap) string {
	for i, sk := range StdKeyMaps {
		if sk == km {
			return StdKeyMapNames[i]
		}
	}
	fmt.Printf("gi.StdKeyMapName key map not found\n")
	return ""
}

// KeyMapItem records one element of the key map -- used for organizing the map.
type KeyMapItem struct {
	Key string       `desc:"the key chord that activates a function"`
	Fun KeyFunctions `desc:"the function of that key"`
}

// ToSlice copies this keymap to a slice of KeyMapItem's
func (km *KeyMap) ToSlice() []KeyMapItem {
	kms := make([]KeyMapItem, len(*km))
	idx := 0
	for key, fun := range *km {
		kms[idx] = KeyMapItem{key, fun}
		idx++
	}
	return kms
}

// ChordForFun returns first key chord trigger for given KeyFun in map
func (km *KeyMap) ChordForFun(kf KeyFunctions) string {
	for key, fun := range *km {
		if fun == kf {
			return key
		}
	}
	return ""
}

// Update ensures that the given keymap has at least one entry for every
// defined KeyFun, grabbing ones from the default map if not, and also
// eliminates any Nil entries which might reflect out-of-date functions
func (km *KeyMap) Update() {
	for key, val := range *km {
		if val == KeyFunNil {
			log.Printf("KeyMap: key function is nil -- probably renamed, for key: %v\n", key)
			delete(*km, key)
		}
	}

	dkms := DefaultKeyMap.ToSlice()
	kms := km.ToSlice()

	sort.Slice(dkms, func(i, j int) bool {
		return dkms[i].Fun < dkms[j].Fun
	})
	sort.Slice(kms, func(i, j int) bool {
		return kms[i].Fun < kms[j].Fun
	})

	addkm := make([]KeyMapItem, 0)

	mi := 0
	for _, dki := range dkms {
		if mi >= len(kms) {
			break
		}
		mmi := kms[mi]
		if dki.Fun < mmi.Fun {
			addkm = append(addkm, dki)
		} else if dki.Fun > mmi.Fun { // shouldn't happen but..
			mi++
		} else {
			mi++
		}
	}

	for _, ai := range addkm {
		(*km)[ai.Key] = ai.Fun
	}
}

/////////////////////////////////////////////////////////////////////////////////
// Shortcuts

// Shortcuts is a map between a key sequence (chord) and a specific Action
// that can be triggered.  This mapping must be unique, in that each chord has
// unique Action, and generally each Action only has a single chord as well,
// though this is not strictly enforced.  Shortcuts are evaluated *after* the
// standard KeyMap event processing, so any conflicts are resolved in favor of
// the local widget's key event processing, with the shortcut only operating
// when no conflicting widgets are in focus.  Shortcuts are always window-wide
// and are intended for global window / toolbar actions.  Widget-specific key
// functions should be be handeled directly within widget key event
// processing.
type Shortcuts map[string]*Action

/////////////////////////////////////////////////////////////////////////////////
// Std Keymaps

// note: shift and meta modifiers for navigation keys do select + move

// todo: use ! to indicate preferred shortcut for menus

// MacEmacsKeyMap defines emacs-based navigation for mac
var MacEmacsKeyMap = KeyMap{
	"UpArrow":             KeyFunMoveUp,
	"Shift+UpArrow":       KeyFunMoveUp,
	"Meta+UpArrow":        KeyFunMoveUp,
	"Control+P":           KeyFunMoveUp,
	"Shift+Control+P":     KeyFunMoveUp,
	"Meta+Control+P":      KeyFunMoveUp,
	"DownArrow":           KeyFunMoveDown,
	"Shift+DownArrow":     KeyFunMoveDown,
	"Meta+DownArrow":      KeyFunMoveDown,
	"Control+N":           KeyFunMoveDown,
	"Shift+Control+N":     KeyFunMoveDown,
	"Meta+Control+N":      KeyFunMoveDown,
	"RightArrow":          KeyFunMoveRight,
	"Shift+RightArrow":    KeyFunMoveRight,
	"Meta+RightArrow":     KeyFunEnd,
	"Control+F":           KeyFunMoveRight,
	"Shift+Control+F":     KeyFunMoveRight,
	"Meta+Control+F":      KeyFunMoveRight,
	"LeftArrow":           KeyFunMoveLeft,
	"Shift+LeftArrow":     KeyFunMoveLeft,
	"Meta+LeftArrow":      KeyFunHome,
	"Control+B":           KeyFunMoveLeft,
	"Shift+Control+B":     KeyFunMoveLeft,
	"Meta+Control+B":      KeyFunMoveLeft,
	"Control+UpArrow":     KeyFunPageUp,
	"Control+U":           KeyFunPageUp,
	"Control+DownArrow":   KeyFunPageDown,
	"Shift+Control+V":     KeyFunPageDown,
	"Alt+V":               KeyFunPageDown,
	"Control+RightArrow":  KeyFunPageRight,
	"Control+LeftArrow":   KeyFunPageLeft,
	"Home":                KeyFunHome,
	"Control+A":           KeyFunHome,
	"Alt+LeftArrow":       KeyFunHome,
	"End":                 KeyFunEnd,
	"Control+E":           KeyFunEnd,
	"Alt+RightArrow":      KeyFunEnd,
	"Tab":                 KeyFunFocusNext,
	"Shift+Tab":           KeyFunFocusPrev,
	"ReturnEnter":         KeyFunSelectItem,
	"KeypadEnter":         KeyFunSelectItem,
	"Shift+Control+A":     KeyFunSelectAll,
	"Meta+A":              KeyFunSelectAll,
	"Control+G":           KeyFunCancelSelect,
	"Control+Spacebar":    KeyFunSelectMode,
	"Control+ReturnEnter": KeyFunAccept,
	"Escape":              KeyFunAbort,
	"DeleteBackspace":     KeyFunBackspace,
	"DeleteForward":       KeyFunDelete,
	"Control+D":           KeyFunDelete,
	"Control+H":           KeyFunBackspace,
	"Control+K":           KeyFunKill,
	"Alt+W":               KeyFunCopy,
	"Control+C":           KeyFunCopy,
	"Meta+C":              KeyFunCopy,
	"Control+W":           KeyFunCut,
	"Control+X":           KeyFunCut,
	"Meta+X":              KeyFunCut,
	"Control+Y":           KeyFunPaste,
	"Control+V":           KeyFunPaste,
	"Meta+V":              KeyFunPaste,
	"Control+M":           KeyFunDuplicate,
	"Control+Z":           KeyFunUndo,
	"Meta+Z":              KeyFunUndo,
	"Control+/":           KeyFunUndo,
	"Shift+Control+Z":     KeyFunRedo,
	"Shift+Meta+Z":        KeyFunRedo,
	"Control+I":           KeyFunInsert,
	"Control+O":           KeyFunInsertAfter,
	"Control+Alt+I":       KeyFunGoGiEditor,
	"Control+Alt+E":       KeyFunGoGiEditor,
	"Shift+Control+I":     KeyFunGoGiEditor,
	"Shift+Meta+=":        KeyFunZoomIn,
	"Meta+=":              KeyFunZoomIn,
	"Meta+-":              KeyFunZoomOut,
	"Control+=":           KeyFunZoomIn,
	"Shift+Control++":     KeyFunZoomIn,
	"Shift+Meta+-":        KeyFunZoomOut,
	"Control+-":           KeyFunZoomOut,
	"Shift+Control+_":     KeyFunZoomOut,
	"Control+Alt+P":       KeyFunPrefs,
	"F5":                  KeyFunRefresh,
	"Control+.":           KeyFunComplete,
}

// todo: following maps need work

// MacKeyMap defines standard mac keys
var MacKeyMap = KeyMap{
	"UpArrow":             KeyFunMoveUp,
	"Shift+UpArrow":       KeyFunMoveUp,
	"Meta+UpArrow":        KeyFunMoveUp,
	"Control+P":           KeyFunMoveUp,
	"Shift+Control+P":     KeyFunMoveUp,
	"Meta+Control+P":      KeyFunMoveUp,
	"DownArrow":           KeyFunMoveDown,
	"Shift+DownArrow":     KeyFunMoveDown,
	"Meta+DownArrow":      KeyFunMoveDown,
	"Control+N":           KeyFunMoveDown,
	"Shift+Control+N":     KeyFunMoveDown,
	"Meta+Control+N":      KeyFunMoveDown,
	"RightArrow":          KeyFunMoveRight,
	"Shift+RightArrow":    KeyFunMoveRight,
	"Meta+RightArrow":     KeyFunEnd,
	"Control+F":           KeyFunMoveRight,
	"Shift+Control+F":     KeyFunMoveRight,
	"Meta+Control+F":      KeyFunMoveRight,
	"LeftArrow":           KeyFunMoveLeft,
	"Shift+LeftArrow":     KeyFunMoveLeft,
	"Meta+LeftArrow":      KeyFunHome,
	"Control+B":           KeyFunMoveLeft,
	"Shift+Control+B":     KeyFunMoveLeft,
	"Meta+Control+B":      KeyFunMoveLeft,
	"Control+UpArrow":     KeyFunPageUp,
	"Control+U":           KeyFunPageUp,
	"Control+DownArrow":   KeyFunPageDown,
	"Shift+Control+V":     KeyFunPageDown,
	"Alt+V":               KeyFunPageDown,
	"Control+RightArrow":  KeyFunPageRight,
	"Control+LeftArrow":   KeyFunPageLeft,
	"Home":                KeyFunHome,
	"Control+A":           KeyFunHome,
	"Alt+LeftArrow":       KeyFunHome,
	"End":                 KeyFunEnd,
	"Control+E":           KeyFunEnd,
	"Alt+RightArrow":      KeyFunEnd,
	"Tab":                 KeyFunFocusNext,
	"Shift+Tab":           KeyFunFocusPrev,
	"ReturnEnter":         KeyFunSelectItem,
	"KeypadEnter":         KeyFunSelectItem,
	"Shift+Control+A":     KeyFunSelectAll,
	"Meta+A":              KeyFunSelectAll,
	"Control+G":           KeyFunCancelSelect,
	"Control+Spacebar":    KeyFunSelectMode,
	"Control+ReturnEnter": KeyFunAccept,
	"Escape":              KeyFunAbort,
	"DeleteBackspace":     KeyFunBackspace,
	"DeleteForward":       KeyFunDelete,
	"Control+D":           KeyFunDelete,
	"Control+H":           KeyFunBackspace,
	"Control+K":           KeyFunKill,
	"Alt+W":               KeyFunCopy,
	"Control+C":           KeyFunCopy,
	"Meta+C":              KeyFunCopy,
	"Control+W":           KeyFunCut,
	"Control+X":           KeyFunCut,
	"Meta+X":              KeyFunCut,
	"Control+Y":           KeyFunPaste,
	"Control+V":           KeyFunPaste,
	"Meta+V":              KeyFunPaste,
	"Control+M":           KeyFunDuplicate,
	"Control+Z":           KeyFunUndo,
	"Meta+Z":              KeyFunUndo,
	"Shift+Control+Z":     KeyFunRedo,
	"Shift+Meta+Z":        KeyFunRedo,
	"Control+I":           KeyFunInsert,
	"Control+O":           KeyFunInsertAfter,
	"Control+Alt+I":       KeyFunGoGiEditor,
	"Control+Alt+E":       KeyFunGoGiEditor,
	"Shift+Control+I":     KeyFunGoGiEditor,
	"Shift+Meta+=":        KeyFunZoomIn,
	"Meta+=":              KeyFunZoomIn,
	"Meta+-":              KeyFunZoomOut,
	"Control+=":           KeyFunZoomIn,
	"Shift+Control++":     KeyFunZoomIn,
	"Shift+Meta+-":        KeyFunZoomOut,
	"Control+-":           KeyFunZoomOut,
	"Shift+Control+_":     KeyFunZoomOut,
	"Control+Alt+P":       KeyFunPrefs,
	"F5":                  KeyFunRefresh,
	"Control+.":           KeyFunComplete,
}

// LinuxKeyMap is a standard key map for generic Linux style bindings
var LinuxKeyMap = KeyMap{
	"UpArrow": KeyFunMoveUp,
	// "Control+P":           KeyFunMoveUp, // Print
	"Shift+UpArrow": KeyFunMoveUp,
	"DownArrow":     KeyFunMoveDown,
	// "Control+N":           KeyFunMoveDown, // New
	"Shift+DownArrow":  KeyFunMoveDown,
	"RightArrow":       KeyFunMoveRight,
	"Shift+RightArrow": KeyFunMoveRight,
	// "Control+F":           KeyFunMoveRight, // Find
	"LeftArrow":       KeyFunMoveLeft,
	"Shift+LeftArrow": KeyFunMoveLeft,
	// "Control+B":           KeyFunMoveLeft, // bold
	"Control+UpArrow": KeyFunPageUp,
	// "Control+U":           KeyFunPageUp, // Underline
	"Control+DownArrow":  KeyFunPageDown,
	"Control+RightArrow": KeyFunPageRight,
	"Control+LeftArrow":  KeyFunPageLeft,
	"Home":               KeyFunHome,
	"Alt+LeftArrow":      KeyFunHome,
	"End":                KeyFunEnd,
	// "Control+E":           KeyFunEnd, // Search Google
	"Alt+RightArrow":  KeyFunEnd,
	"Tab":             KeyFunFocusNext,
	"Shift+Tab":       KeyFunFocusPrev,
	"ReturnEnter":     KeyFunSelectItem,
	"KeypadEnter":     KeyFunSelectItem,
	"Control+A":       KeyFunSelectAll,
	"Shift+Control+A": KeyFunCancelSelect,
	//"Control+Spacebar":    KeyFunSelectMode, // change input method / keyboard
	"Control+ReturnEnter": KeyFunAccept,
	"Escape":              KeyFunAbort,
	"DeleteBackspace":     KeyFunBackspace,
	"DeleteForward":       KeyFunDelete,
	// "Control+D":           KeyFunDelete, // Bookmark
	// "Control+H":       KeyFunBackspace, // Help
	"Control+K": KeyFunKill,
	"Control+C": KeyFunCopy,
	// "Control+W":       KeyFunCut, // Close Current Tab
	"Control+X":       KeyFunCut,
	"Control+V":       KeyFunPaste,
	"Control+M":       KeyFunDuplicate,
	"Control+Z":       KeyFunUndo,
	"Shift+Control+Z": KeyFunRedo,
	// "Control+I":       KeyFunInsert, // Italic
	// "Control+O":       KeyFunInsertAfter, // Open
	"Shift+Control+I": KeyFunGoGiEditor,
	"Control+=":       KeyFunZoomIn,
	"Shift+Control++": KeyFunZoomIn,
	"Control+-":       KeyFunZoomOut,
	"Shift+Control+_": KeyFunZoomOut,
	"Shift+Control+P": KeyFunPrefs,
	"Control+Alt+P":   KeyFunPrefs,
	"F5":              KeyFunRefresh,
	"Control+.":       KeyFunComplete,
}

// LinuxEmacsKeyMap defines standard emacs-based navigation for linux
var LinuxEmacsKeyMap = KeyMap{
	"UpArrow":            KeyFunMoveUp,
	"Shift+UpArrow":      KeyFunMoveUp,
	"Meta+UpArrow":       KeyFunMoveUp,
	"Control+P":          KeyFunMoveUp,
	"Shift+Control+P":    KeyFunMoveUp,
	"Meta+Control+P":     KeyFunMoveUp,
	"DownArrow":          KeyFunMoveDown,
	"Shift+DownArrow":    KeyFunMoveDown,
	"Meta+DownArrow":     KeyFunMoveDown,
	"Control+N":          KeyFunMoveDown,
	"Shift+Control+N":    KeyFunMoveDown,
	"Meta+Control+N":     KeyFunMoveDown,
	"RightArrow":         KeyFunMoveRight,
	"Shift+RightArrow":   KeyFunMoveRight,
	"Meta+RightArrow":    KeyFunEnd,
	"Control+F":          KeyFunMoveRight,
	"Shift+Control+F":    KeyFunMoveRight,
	"Meta+Control+F":     KeyFunMoveRight,
	"LeftArrow":          KeyFunMoveLeft,
	"Shift+LeftArrow":    KeyFunMoveLeft,
	"Meta+LeftArrow":     KeyFunHome,
	"Control+B":          KeyFunMoveLeft,
	"Shift+Control+B":    KeyFunMoveLeft,
	"Meta+Control+B":     KeyFunMoveLeft,
	"Control+UpArrow":    KeyFunPageUp,
	"Control+U":          KeyFunPageUp,
	"Control+DownArrow":  KeyFunPageDown,
	"Shift+Control+V":    KeyFunPageDown,
	"Alt+V":              KeyFunPageDown,
	"Control+RightArrow": KeyFunPageRight,
	"Control+LeftArrow":  KeyFunPageLeft,
	"Home":               KeyFunHome,
	"Control+A":          KeyFunHome,
	"Shift+Control+A":    KeyFunHome,
	"Alt+LeftArrow":      KeyFunHome,
	"End":                KeyFunEnd,
	"Control+E":          KeyFunEnd,
	"Shift+Control+E":    KeyFunEnd,
	"Alt+RightArrow":     KeyFunEnd,
	"Tab":                KeyFunFocusNext,
	"Shift+Tab":          KeyFunFocusPrev,
	"ReturnEnter":        KeyFunSelectItem,
	"KeypadEnter":        KeyFunSelectItem,
	// "Shift+Control+A":     KeyFunSelectAll,
	"Meta+A":              KeyFunSelectAll,
	"Control+G":           KeyFunCancelSelect,
	"Control+Spacebar":    KeyFunSelectMode,
	"Control+ReturnEnter": KeyFunAccept,
	"Escape":              KeyFunAbort,
	"DeleteBackspace":     KeyFunBackspace,
	"DeleteForward":       KeyFunDelete,
	"Control+D":           KeyFunDelete,
	"Control+H":           KeyFunBackspace,
	"Control+K":           KeyFunKill,
	"Alt+W":               KeyFunCopy,
	"Control+C":           KeyFunCopy,
	"Meta+C":              KeyFunCopy,
	"Control+W":           KeyFunCut,
	"Control+X":           KeyFunCut,
	"Meta+X":              KeyFunCut,
	"Control+Y":           KeyFunPaste,
	"Control+V":           KeyFunPaste,
	"Meta+V":              KeyFunPaste,
	"Control+M":           KeyFunDuplicate,
	"Control+I":           KeyFunInsert,
	"Control+O":           KeyFunInsertAfter,
	"Control+Alt+I":       KeyFunGoGiEditor,
	"Control+Alt+E":       KeyFunGoGiEditor,
	"Shift+Control+I":     KeyFunGoGiEditor,
	"Shift+Meta+=":        KeyFunZoomIn,
	"Meta+=":              KeyFunZoomIn,
	"Meta+-":              KeyFunZoomOut,
	"Control+=":           KeyFunZoomIn,
	"Shift+Control++":     KeyFunZoomIn,
	"Shift+Meta+-":        KeyFunZoomOut,
	"Control+-":           KeyFunZoomOut,
	"Shift+Control+_":     KeyFunZoomOut,
	"Control+Alt+P":       KeyFunPrefs,
	"F5":                  KeyFunRefresh,
	"Control+.":           KeyFunComplete,
}

// WindowsKeyMap is a standard key map for generic Windows style bindings
var WindowsKeyMap = KeyMap{
	"UpArrow": KeyFunMoveUp,
	// "Control+P":           KeyFunMoveUp, // Print
	"Shift+UpArrow": KeyFunMoveUp,
	"DownArrow":     KeyFunMoveDown,
	// "Control+N":           KeyFunMoveDown, // New
	"Shift+DownArrow":  KeyFunMoveDown,
	"RightArrow":       KeyFunMoveRight,
	"Shift+RightArrow": KeyFunMoveRight,
	// "Control+F":           KeyFunMoveRight, // Find
	"LeftArrow":       KeyFunMoveLeft,
	"Shift+LeftArrow": KeyFunMoveLeft,
	// "Control+B":           KeyFunMoveLeft, // bold
	"Control+UpArrow": KeyFunPageUp,
	// "Control+U":           KeyFunPageUp, // Underline
	"Control+DownArrow":  KeyFunPageDown,
	"Control+RightArrow": KeyFunPageRight,
	"Control+LeftArrow":  KeyFunPageLeft,
	"Home":               KeyFunHome,
	"Alt+LeftArrow":      KeyFunHome,
	"End":                KeyFunEnd,
	// "Control+E":           KeyFunEnd, // Search Google
	"Alt+RightArrow":  KeyFunEnd,
	"Tab":             KeyFunFocusNext,
	"Shift+Tab":       KeyFunFocusPrev,
	"ReturnEnter":     KeyFunSelectItem,
	"KeypadEnter":     KeyFunSelectItem,
	"Control+A":       KeyFunSelectAll,
	"Shift+Control+A": KeyFunCancelSelect,
	//"Control+Spacebar":    KeyFunSelectMode, // change input method / keyboard
	"Control+ReturnEnter": KeyFunAccept,
	"Escape":              KeyFunAbort,
	"DeleteBackspace":     KeyFunBackspace,
	"DeleteForward":       KeyFunDelete,
	// "Control+D":           KeyFunDelete, // Bookmark
	// "Control+H":       KeyFunBackspace, // Help
	"Control+K": KeyFunKill,
	"Control+C": KeyFunCopy,
	// "Control+W":       KeyFunCut, // Close Current Tab
	"Control+X": KeyFunCut,
	"Control+V": KeyFunPaste,
	"Control+M": KeyFunDuplicate,
	// "Control+I":       KeyFunInsert, // Italic
	// "Control+O":       KeyFunInsertAfter, // Open
	"Shift+Control+I": KeyFunGoGiEditor,
	"Control+=":       KeyFunZoomIn,
	"Shift+Control++": KeyFunZoomIn,
	"Control+-":       KeyFunZoomOut,
	"Shift+Control+_": KeyFunZoomOut,
	"Shift+Control+P": KeyFunPrefs,
	"Control+Alt+P":   KeyFunPrefs,
	"F5":              KeyFunRefresh,
	"Control+.":       KeyFunComplete,
}

// ChromeKeyMap is a standard key map for google / chrome style bindings
var ChromeKeyMap = KeyMap{
	"UpArrow": KeyFunMoveUp,
	// "Control+P":           KeyFunMoveUp, // Print
	"Shift+UpArrow": KeyFunMoveUp,
	"DownArrow":     KeyFunMoveDown,
	// "Control+N":           KeyFunMoveDown, // New
	"Shift+DownArrow":  KeyFunMoveDown,
	"RightArrow":       KeyFunMoveRight,
	"Shift+RightArrow": KeyFunMoveRight,
	// "Control+F":           KeyFunMoveRight, // Find
	"LeftArrow":       KeyFunMoveLeft,
	"Shift+LeftArrow": KeyFunMoveLeft,
	// "Control+B":           KeyFunMoveLeft, // bold
	"Control+UpArrow": KeyFunPageUp,
	// "Control+U":           KeyFunPageUp, // Underline
	"Control+DownArrow":  KeyFunPageDown,
	"Control+RightArrow": KeyFunPageRight,
	"Control+LeftArrow":  KeyFunPageLeft,
	"Home":               KeyFunHome,
	"Alt+LeftArrow":      KeyFunHome,
	"End":                KeyFunEnd,
	// "Control+E":           KeyFunEnd, // Search Google
	"Alt+RightArrow":  KeyFunEnd,
	"Tab":             KeyFunFocusNext,
	"Shift+Tab":       KeyFunFocusPrev,
	"ReturnEnter":     KeyFunSelectItem,
	"KeypadEnter":     KeyFunSelectItem,
	"Control+A":       KeyFunSelectAll,
	"Shift+Control+A": KeyFunCancelSelect,
	//"Control+Spacebar":    KeyFunSelectMode, // change input method / keyboard
	"Control+ReturnEnter": KeyFunAccept,
	"Escape":              KeyFunAbort,
	"DeleteBackspace":     KeyFunBackspace,
	"DeleteForward":       KeyFunDelete,
	// "Control+D":           KeyFunDelete, // Bookmark
	// "Control+H":       KeyFunBackspace, // Help
	"Control+K": KeyFunKill,
	"Control+C": KeyFunCopy,
	// "Control+W":       KeyFunCut, // Close Current Tab
	"Control+X": KeyFunCut,
	"Control+V": KeyFunPaste,
	"Control+M": KeyFunDuplicate,
	// "Control+I":       KeyFunInsert, // Italic
	// "Control+O":       KeyFunInsertAfter, // Open
	"Shift+Control+I": KeyFunGoGiEditor,
	"Control+=":       KeyFunZoomIn,
	"Shift+Control++": KeyFunZoomIn,
	"Control+-":       KeyFunZoomOut,
	"Shift+Control+_": KeyFunZoomOut,
	"Shift+Control+P": KeyFunPrefs,
	"Control+Alt+P":   KeyFunPrefs,
	"F5":              KeyFunRefresh,
	"Control+.":       KeyFunComplete,
}
