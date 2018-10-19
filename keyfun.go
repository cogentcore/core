// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// https://en.wikipedia.org/wiki/Table_of_keyboard_shortcuts

// KeyFuns are functions that keyboard events can perform in the GUI --
// seems possible to keep this flat and consistent across different contexts,
// as long as the functions can be appropriately reinterpreted for each
// context.
type KeyFuns int32

const (
	KeyFunNil KeyFuns = iota
	KeyFunMoveUp
	KeyFunMoveDown
	KeyFunMoveRight
	KeyFunMoveLeft
	KeyFunPageUp
	KeyFunPageDown
	KeyFunPageRight
	KeyFunPageLeft
	KeyFunHome    // start-of-line
	KeyFunEnd     // end-of-line
	KeyFunDocHome // start-of-doc -- Control / Alt / Shift +Home
	KeyFunDocEnd  // end-of-doc Control / Alt / Shift +End
	KeyFunWordRight
	KeyFunWordLeft
	KeyFunFocusNext // Tab
	KeyFunFocusPrev // Shift-Tab
	KeyFunEnter     // Enter / return key -- has various special functions
	KeyFunAccept    // Ctrl+Enter = accept any changes and close dialog / move to next
	KeyFunCancelSelect
	KeyFunSelectMode
	KeyFunSelectAll
	KeyFunAbort
	KeyFunEditItem
	KeyFunCopy
	KeyFunCut
	KeyFunPaste
	KeyFunPasteHist // from history
	KeyFunBackspace
	KeyFunBackspaceWord
	KeyFunDelete
	KeyFunDeleteWord
	KeyFunKill
	KeyFunDuplicate
	KeyFunUndo
	KeyFunRedo
	KeyFunInsert
	KeyFunInsertAfter
	KeyFunGoGiEditor
	KeyFunZoomOut
	KeyFunZoomIn
	KeyFunPrefs
	KeyFunRefresh
	KeyFunRecenter // Ctrl+L in emacs
	KeyFunComplete
	KeyFunSearch // Ctrl+S in emacs -- more interactive type of search
	KeyFunFind   // Command+F full-dialog find
	KeyFunJump   // jump to line
	KeyFunHistPrev
	KeyFunHistNext
	KeyFunsN
)

//go:generate stringer -type=KeyFuns

var KiT_KeyFuns = kit.Enums.AddEnumAltLower(KeyFunsN, false, StylePropProps, "KeyFun")

func (kf KeyFuns) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(kf) }
func (kf *KeyFuns) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(kf, b) }

// KeyMap is a map between a key sequence (chord) and a specific KeyFun
// function.  This mapping must be unique, in that each chord has unique
// KeyFun, but multiple chords can trigger the same function.
type KeyMap map[key.Chord]KeyFuns

// ActiveKeyMap points to the active map -- users can set this to an
// alternative map in Prefs
var ActiveKeyMap *KeyMap

// SetActiveKeyMap sets the current ActiveKeyMap, calling Update on the map
// prior to setting it to ensure that it is a valid, complete map
func SetActiveKeyMap(km *KeyMap) {
	km.Update()
	ActiveKeyMap = km
}

// SetActiveKeyMapName sets the current ActiveKeyMap by name from those
// defined in AvailKeyMaps, calling Update on the map prior to setting it to
// ensure that it is a valid, complete map
func SetActiveKeyMapName(mapnm KeyMapName) {
	km, _, ok := AvailKeyMaps.MapByName(mapnm)
	if ok {
		SetActiveKeyMap(km)
	} else {
		log.Printf("gi.SetActiveKeyMapName: key map named: %v not found, using default: %v\n", mapnm, DefaultKeyMap)
		km, _, ok = AvailKeyMaps.MapByName(DefaultKeyMap)
		if ok {
			SetActiveKeyMap(km)
		} else {
			log.Printf("gi.SetActiveKeyMapName: ok, this is bad: DefaultKeyMap not found either -- size of AvailKeyMaps: %v -- trying first one\n", len(AvailKeyMaps))
			if len(AvailKeyMaps) > 0 {
				SetActiveKeyMap(&AvailKeyMaps[0].Map)
			}
		}
	}
}

// KeyFun translates chord into keyboard function -- use oswin key.Chord
// to get chord
func KeyFun(chord key.Chord) KeyFuns {
	kf := KeyFunNil
	if chord != "" {
		kf = (*ActiveKeyMap)[chord]
		// fmt.Printf("chord: %v = %v\n", chord, kf)
	}
	return kf
}

// KeyMapItem records one element of the key map -- used for organizing the map.
type KeyMapItem struct {
	Key key.Chord `desc:"the key chord that activates a function"`
	Fun KeyFuns   `desc:"the function of that key"`
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
func (km *KeyMap) ChordForFun(kf KeyFuns) key.Chord {
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
	dkm, _, _ := AvailKeyMaps.MapByName(DefaultKeyMap)

	dkms := dkm.ToSlice()
	kms := km.ToSlice()

	addkm := make([]KeyMapItem, 0)

	if len(kms) == 0 { // set custom to match default
		for _, dki := range dkms {
			addkm = append(addkm, dki)
			fmt.Println(dki.Fun.String())
		}
		for _, ai := range addkm {
			(*km)[ai.Key] = ai.Fun
		}
		return
	}

	sort.Slice(dkms, func(i, j int) bool {
		return dkms[i].Fun < dkms[j].Fun
	})
	sort.Slice(kms, func(i, j int) bool {
		return kms[i].Fun < kms[j].Fun
	})

	mi := 0
	for _, dki := range dkms {
		if mi >= len(kms) {
			break
		}
		mmi := kms[mi]
		if dki.Fun < mmi.Fun {
			fmt.Printf("warning - %v has no key mapping", dki.Fun)
			addkm = append(addkm, dki)
			s := dki.Fun.String()
			s = strings.TrimPrefix(s, "KeyFun")
			s = "- Not Set - " + s
			addkm[len(addkm)-1].Key = key.Chord(s)
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

// Shortcuts is a map between a key chord and a specific Action that can be
// triggered.  This mapping must be unique, in that each chord has unique
// Action, and generally each Action only has a single chord as well, though
// this is not strictly enforced.  Shortcuts are evaluated *after* the
// standard KeyMap event processing, so any conflicts are resolved in favor of
// the local widget's key event processing, with the shortcut only operating
// when no conflicting widgets are in focus.  Shortcuts are always window-wide
// and are intended for global window / toolbar actions.  Widget-specific key
// functions should be be handeled directly within widget key event
// processing.
type Shortcuts map[key.Chord]*Action

/////////////////////////////////////////////////////////////////////////////////
// KeyMaps -- list of KeyMap's

// KeyMapName has an associated ValueView for selecting from the list of
// available key map names, for use in preferences etc.
type KeyMapName string

// DefaultKeyMap is the overall default keymap -- reinitialized in gimain init()
// depending on platform
var DefaultKeyMap = KeyMapName("MacEmacs")

// KeyMapsItem is an entry in a KeyMaps list
type KeyMapsItem struct {
	Name string `width:"20" desc:"name of keymap"`
	Desc string `desc:"description of keymap -- good idea to include source it was derived from"`
	Map  KeyMap `desc:"to edit key sequence click button and type new key combination; to edit function mapped to key sequence choose from menu"`
}

// KeyMaps is a list of KeyMap's -- users can edit these in Prefs -- to create
// a custom one, just duplicate an existing map, rename, and customize
type KeyMaps []KeyMapsItem

var KiT_KeyMaps = kit.Types.AddType(&KeyMaps{}, KeyMapsProps)

// AvailKeyMaps is the current list of available keymaps for use -- can be
// loaded / saved / edited with preferences.  This is set to StdKeyMaps at
// startup.
var AvailKeyMaps KeyMaps

func init() {
	AvailKeyMaps.CopyFrom(StdKeyMaps)
}

// MapByName returns a keymap and index by name -- returns false and emits a
// message to stdout if not found
func (km *KeyMaps) MapByName(name KeyMapName) (*KeyMap, int, bool) {
	for i, it := range *km {
		if it.Name == string(name) {
			return &it.Map, i, true
		}
	}
	fmt.Printf("gi.KeyMaps.MapByName: key map named: %v not found\n", name)
	return nil, -1, false
}

// PrefsKeyMapsFileName is the name of the preferences file in GoGi prefs
// directory for saving / loading the default AvailKeyMaps key maps list
var PrefsKeyMapsFileName = "key_maps_prefs.json"

// OpenJSON opens keymaps from a JSON-formatted file.
func (km *KeyMaps) OpenJSON(filename FileName) error {
	*km = make(KeyMaps, 0, 10) // reset
	b, err := ioutil.ReadFile(string(filename))
	if err != nil {
		PromptDialog(nil, DlgOpts{Title: "File Not Found", Prompt: err.Error()}, true, false, nil, nil)
		log.Println(err)
		return err
	}
	return json.Unmarshal(b, km)
}

// SaveJSON saves keymaps to a JSON-formatted file.
func (km *KeyMaps) SaveJSON(filename FileName) error {
	b, err := json.MarshalIndent(km, "", "  ")
	if err != nil {
		log.Println(err) // unlikely
		return err
	}
	err = ioutil.WriteFile(string(filename), b, 0644)
	if err != nil {
		PromptDialog(nil, DlgOpts{Title: "Could not Save to File", Prompt: err.Error()}, true, false, nil, nil)
		log.Println(err)
	}
	return err
}

// OpenPrefs opens KeyMaps from GoGi standard prefs directory, using PrefsKeyMapsFileName
func (km *KeyMaps) OpenPrefs() error {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, PrefsKeyMapsFileName)
	AvailKeyMapsChanged = false
	return km.OpenJSON(FileName(pnm))
}

// SavePrefs saves KeyMaps to GoGi standard prefs directory, using PrefsKeyMapsFileName
func (km *KeyMaps) SavePrefs() error {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, PrefsKeyMapsFileName)
	AvailKeyMapsChanged = false
	return km.SaveJSON(FileName(pnm))
}

// CopyFrom copies keymaps from given other map
func (km *KeyMaps) CopyFrom(cp KeyMaps) {
	*km = make(KeyMaps, 0, len(cp)) // reset
	b, _ := json.Marshal(cp)
	json.Unmarshal(b, km)
}

// RevertToStd reverts this map to using the StdKeyMaps that are compiled into
// the program and have all the lastest key functions bound to standard
// values.
func (km *KeyMaps) RevertToStd() {
	km.CopyFrom(StdKeyMaps)
	AvailKeyMapsChanged = true
}

// ViewStd shows the standard maps that are compiled into the program and have
// all the lastest key functions bound to standard values.  Useful for
// comparing against custom maps.
func (km *KeyMaps) ViewStd() {
	TheViewIFace.KeyMapsView(&StdKeyMaps)
}

// AvailKeyMapsChanged is used to update giv.KeyMapsView toolbars via
// following menu, toolbar props update methods -- not accurate if editing any
// other map but works for now..
var AvailKeyMapsChanged = false

// KeyMapsProps define the ToolBar and MenuBar for TableView of KeyMaps, e.g., giv.KeyMapsView
var KeyMapsProps = ki.Props{
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenPrefs", ki.Props{}},
			{"SavePrefs", ki.Props{
				"shortcut": "Command+S",
				"updtfunc": func(kmi interface{}, act *Action) {
					act.SetActiveState(AvailKeyMapsChanged)
				},
			}},
			{"sep-file", ki.BlankProp{}},
			{"OpenJSON", ki.Props{
				"label":    "Open from file",
				"desc":     "You can save and open key maps to / from files to share, experiment, transfer, etc",
				"shortcut": "Command+O",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
			{"SaveJSON", ki.Props{
				"label": "Save to file",
				"desc":  "You can save and open key maps to / from files to share, experiment, transfer, etc",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
			{"RevertToStd", ki.Props{
				"desc":    "This reverts the keymaps to using the StdKeyMaps that are compiled into the program and have all the lastest key functions defined.  If you have edited your maps, and are finding things not working, it is a good idea to save your current maps and try this, or at least do ViewStdMaps to see the current standards.  <b>Your current map edits will be lost if you proceed!</b>  Continue?",
				"confirm": true,
			}},
		}},
		{"Edit", "Copy Cut Paste Dupe"},
		{"Window", "Windows"},
	},
	"ToolBar": ki.PropSlice{
		{"SavePrefs", ki.Props{
			"desc": "saves KeyMaps to GoGi standard prefs directory, in file key_maps_prefs.json, which will be loaded automatically at startup if prefs SaveKeyMaps is checked (should be if you're using custom keymaps)",
			"icon": "file-save",
			"updtfunc": func(kmi interface{}, act *Action) {
				act.SetActiveStateUpdt(AvailKeyMapsChanged)
			},
		}},
		{"sep-file", ki.BlankProp{}},
		{"OpenJSON", ki.Props{
			"label": "Open from file",
			"icon":  "file-open",
			"desc":  "You can save and open key maps to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"SaveJSON", ki.Props{
			"label": "Save to file",
			"icon":  "file-save",
			"desc":  "You can save and open key maps to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"sep-std", ki.BlankProp{}},
		{"ViewStd", ki.Props{
			"desc":    "Shows the standard maps that are compiled into the program and have all the lastest key functions bound to standard key chords.  Useful for comparing against custom maps.",
			"confirm": true,
		}},
		{"RevertToStd", ki.Props{
			"icon":    "update",
			"desc":    "This reverts the keymaps to using the StdKeyMaps that are compiled into the program and have all the lastest key functions bound to standard key chords.  If you have edited your maps, and are finding things not working, it is a good idea to save your current maps and try this, or at least do ViewStdMaps to see the current standards.  <b>Your current map edits will be lost if you proceed!</b>  Continue?",
			"confirm": true,
		}},
	},
}

// note: shift and meta modifiers for navigation keys do select + move

// note: where multiple shortcuts exist for a given function, any shortcut
// display of such items in menus will randomly display one of the
// options. This can be considered a feature, not a bug!

// StdKeyMaps is the original compiled-in set of standard keymaps that have
// the lastest key functions bound to standard key chords.
var StdKeyMaps = KeyMaps{
	{"MacStd", "Standard Mac KeyMap", KeyMap{
		"UpArrow":                 KeyFunMoveUp,
		"Shift+UpArrow":           KeyFunMoveUp,
		"Meta+UpArrow":            KeyFunMoveUp,
		"Control+P":               KeyFunMoveUp,
		"Shift+Control+P":         KeyFunMoveUp,
		"Meta+Control+P":          KeyFunMoveUp,
		"DownArrow":               KeyFunMoveDown,
		"Shift+DownArrow":         KeyFunMoveDown,
		"Meta+DownArrow":          KeyFunMoveDown,
		"Control+N":               KeyFunMoveDown,
		"Shift+Control+N":         KeyFunMoveDown,
		"Meta+Control+N":          KeyFunMoveDown,
		"RightArrow":              KeyFunMoveRight,
		"Shift+RightArrow":        KeyFunMoveRight,
		"Meta+RightArrow":         KeyFunEnd,
		"Control+F":               KeyFunMoveRight,
		"Shift+Control+F":         KeyFunMoveRight,
		"Meta+Control+F":          KeyFunMoveRight,
		"LeftArrow":               KeyFunMoveLeft,
		"Shift+LeftArrow":         KeyFunMoveLeft,
		"Meta+LeftArrow":          KeyFunHome,
		"Control+B":               KeyFunMoveLeft,
		"Shift+Control+B":         KeyFunMoveLeft,
		"Meta+Control+B":          KeyFunMoveLeft,
		"Control+UpArrow":         KeyFunPageUp,
		"Control+U":               KeyFunPageUp,
		"Control+DownArrow":       KeyFunPageDown,
		"Shift+Control+V":         KeyFunPageDown,
		"Alt+√":                   KeyFunPageDown,
		"Control+RightArrow":      KeyFunWordRight,
		"Control+LeftArrow":       KeyFunWordLeft,
		"Alt+RightArrow":          KeyFunWordRight,
		"Alt+LeftArrow":           KeyFunWordLeft,
		"Home":                    KeyFunHome,
		"Control+A":               KeyFunHome,
		"End":                     KeyFunEnd,
		"Control+E":               KeyFunEnd,
		"Tab":                     KeyFunFocusNext,
		"Shift+Tab":               KeyFunFocusPrev,
		"ReturnEnter":             KeyFunEnter,
		"KeypadEnter":             KeyFunEnter,
		"Shift+Control+A":         KeyFunSelectAll,
		"Meta+A":                  KeyFunSelectAll,
		"Control+G":               KeyFunCancelSelect,
		"Control+Spacebar":        KeyFunSelectMode,
		"Control+ReturnEnter":     KeyFunAccept,
		"Escape":                  KeyFunAbort,
		"DeleteBackspace":         KeyFunBackspace,
		"Control+DeleteBackspace": KeyFunBackspaceWord,
		"Alt+DeleteBackspace":     KeyFunBackspaceWord,
		"DeleteForward":           KeyFunDelete,
		"Control+DeleteForward":   KeyFunDeleteWord,
		"Alt+DeleteForward":       KeyFunDeleteWord,
		"Control+D":               KeyFunDelete,
		"Control+H":               KeyFunBackspace,
		"Control+K":               KeyFunKill,
		"Alt+∑":                   KeyFunCopy,
		"Meta+C":                  KeyFunCopy,
		"Control+W":               KeyFunCut,
		"Meta+X":                  KeyFunCut,
		"Control+Y":               KeyFunPaste,
		"Control+V":               KeyFunPaste,
		"Meta+V":                  KeyFunPaste,
		"Shift+Meta+V":            KeyFunPasteHist,
		"Control+M":               KeyFunDuplicate,
		"Control+Z":               KeyFunUndo,
		"Meta+Z":                  KeyFunUndo,
		"Shift+Control+Z":         KeyFunRedo,
		"Shift+Meta+Z":            KeyFunRedo,
		"Control+I":               KeyFunInsert,
		"Control+O":               KeyFunInsertAfter,
		"Control+Alt+I":           KeyFunGoGiEditor,
		"Control+Alt+E":           KeyFunGoGiEditor,
		"Shift+Control+I":         KeyFunGoGiEditor,
		"Shift+Meta+=":            KeyFunZoomIn,
		"Meta+=":                  KeyFunZoomIn,
		"Meta+-":                  KeyFunZoomOut,
		"Control+=":               KeyFunZoomIn,
		"Shift+Control++":         KeyFunZoomIn,
		"Shift+Meta+-":            KeyFunZoomOut,
		"Control+-":               KeyFunZoomOut,
		"Shift+Control+_":         KeyFunZoomOut,
		"Control+Alt+P":           KeyFunPrefs,
		"F5":                      KeyFunRefresh,
		"Control+L":               KeyFunRecenter,
		"Control+.":               KeyFunComplete,
		"Control+S":               KeyFunSearch,
		"Meta+F":                  KeyFunFind,
		"Control+J":               KeyFunJump,
		"Meta+[":                  KeyFunHistPrev,
		"Meta+]":                  KeyFunHistNext,
		"Control+[":               KeyFunHistPrev,
		"Control+]":               KeyFunHistNext,
	}},
	{"MacEmacs", "Mac with emacs-style navigation -- emacs wins in conflicts", KeyMap{
		"UpArrow":                 KeyFunMoveUp,
		"Shift+UpArrow":           KeyFunMoveUp,
		"Meta+UpArrow":            KeyFunMoveUp,
		"Control+P":               KeyFunMoveUp,
		"Shift+Control+P":         KeyFunMoveUp,
		"Meta+Control+P":          KeyFunMoveUp,
		"DownArrow":               KeyFunMoveDown,
		"Shift+DownArrow":         KeyFunMoveDown,
		"Meta+DownArrow":          KeyFunMoveDown,
		"Control+N":               KeyFunMoveDown,
		"Shift+Control+N":         KeyFunMoveDown,
		"Meta+Control+N":          KeyFunMoveDown,
		"RightArrow":              KeyFunMoveRight,
		"Shift+RightArrow":        KeyFunMoveRight,
		"Meta+RightArrow":         KeyFunEnd,
		"Control+F":               KeyFunMoveRight,
		"Shift+Control+F":         KeyFunMoveRight,
		"Meta+Control+F":          KeyFunMoveRight,
		"LeftArrow":               KeyFunMoveLeft,
		"Shift+LeftArrow":         KeyFunMoveLeft,
		"Meta+LeftArrow":          KeyFunHome,
		"Control+B":               KeyFunMoveLeft,
		"Shift+Control+B":         KeyFunMoveLeft,
		"Meta+Control+B":          KeyFunMoveLeft,
		"Control+UpArrow":         KeyFunPageUp,
		"Control+U":               KeyFunPageUp,
		"Control+DownArrow":       KeyFunPageDown,
		"Shift+Control+V":         KeyFunPageDown,
		"Alt+√":                   KeyFunPageDown,
		"Control+V":               KeyFunPageDown,
		"Control+RightArrow":      KeyFunWordRight,
		"Control+LeftArrow":       KeyFunWordLeft,
		"Alt+RightArrow":          KeyFunWordRight,
		"Alt+LeftArrow":           KeyFunWordLeft,
		"Home":                    KeyFunHome,
		"Control+A":               KeyFunHome,
		"End":                     KeyFunEnd,
		"Control+E":               KeyFunEnd,
		"Meta+Home":               KeyFunDocHome,
		"Meta+H":                  KeyFunDocHome,
		"Meta+L":                  KeyFunDocEnd,
		"Control+Alt+E":           KeyFunDocEnd,
		"Alt+Ƒ":                   KeyFunWordRight,
		"Alt+∫":                   KeyFunWordLeft,
		"Tab":                     KeyFunFocusNext,
		"Shift+Tab":               KeyFunFocusPrev,
		"ReturnEnter":             KeyFunEnter,
		"KeypadEnter":             KeyFunEnter,
		"Shift+Control+A":         KeyFunSelectAll,
		"Meta+A":                  KeyFunSelectAll,
		"Control+G":               KeyFunCancelSelect,
		"Control+Spacebar":        KeyFunSelectMode,
		"Control+ReturnEnter":     KeyFunAccept,
		"Escape":                  KeyFunAbort,
		"DeleteBackspace":         KeyFunBackspace,
		"Control+DeleteBackspace": KeyFunBackspaceWord,
		"Alt+DeleteBackspace":     KeyFunBackspaceWord,
		"DeleteForward":           KeyFunDelete,
		"Control+DeleteForward":   KeyFunDeleteWord,
		"Alt+DeleteForward":       KeyFunDeleteWord,
		"Control+D":               KeyFunDelete,
		"Control+H":               KeyFunBackspace,
		"Control+K":               KeyFunKill,
		"Alt+∑":                   KeyFunCopy,
		"Meta+C":                  KeyFunCopy,
		"Control+W":               KeyFunCut,
		"Meta+X":                  KeyFunCut,
		"Control+Y":               KeyFunPaste,
		"Meta+V":                  KeyFunPaste,
		"Shift+Meta+V":            KeyFunPasteHist,
		"Shift+Control+Y":         KeyFunPasteHist,
		"Control+M":               KeyFunDuplicate,
		"Control+Z":               KeyFunUndo,
		"Meta+Z":                  KeyFunUndo,
		"Control+/":               KeyFunUndo,
		"Shift+Control+Z":         KeyFunRedo,
		"Shift+Meta+Z":            KeyFunRedo,
		"Control+I":               KeyFunInsert,
		"Control+O":               KeyFunInsertAfter,
		"Control+Alt+I":           KeyFunGoGiEditor,
		"Shift+Control+I":         KeyFunGoGiEditor,
		"Shift+Meta+=":            KeyFunZoomIn,
		"Meta+=":                  KeyFunZoomIn,
		"Meta+-":                  KeyFunZoomOut,
		"Control+=":               KeyFunZoomIn,
		"Shift+Control++":         KeyFunZoomIn,
		"Shift+Meta+-":            KeyFunZoomOut,
		"Control+-":               KeyFunZoomOut,
		"Shift+Control+_":         KeyFunZoomOut,
		"Control+Alt+P":           KeyFunPrefs,
		"F5":                      KeyFunRefresh,
		"Control+L":               KeyFunRecenter,
		"Control+.":               KeyFunComplete,
		"Control+S":               KeyFunSearch,
		"Meta+F":                  KeyFunFind,
		"Control+J":               KeyFunJump,
		"Meta+[":                  KeyFunHistPrev,
		"Meta+]":                  KeyFunHistNext,
		"Control+[":               KeyFunHistPrev,
		"Control+]":               KeyFunHistNext,
	}},
	{"LinuxStd", "Standard Linux KeyMap", KeyMap{
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
		"Control+RightArrow": KeyFunWordRight,
		"Control+LeftArrow":  KeyFunWordLeft,
		"Home":               KeyFunHome,
		"Alt+LeftArrow":      KeyFunHome,
		"End":                KeyFunEnd,
		// "Control+E":           KeyFunEnd, // Search Google
		"Alt+RightArrow":  KeyFunEnd,
		"Tab":             KeyFunFocusNext,
		"Shift+Tab":       KeyFunFocusPrev,
		"ReturnEnter":     KeyFunEnter,
		"KeypadEnter":     KeyFunEnter,
		"Control+A":       KeyFunSelectAll,
		"Shift+Control+A": KeyFunCancelSelect,
		//"Control+Spacebar":    KeyFunSelectMode, // change input method / keyboard
		"Control+ReturnEnter":     KeyFunAccept,
		"Escape":                  KeyFunAbort,
		"DeleteBackspace":         KeyFunBackspace,
		"Control+DeleteBackspace": KeyFunBackspaceWord,
		"DeleteForward":           KeyFunDelete,
		// "Control+D":           KeyFunDelete, // Bookmark
		// "Control+H":       KeyFunBackspace, // Help
		"Control+K": KeyFunKill,
		"Control+C": KeyFunCopy,
		// "Control+W":       KeyFunCut, // Close Current Tab
		"Control+X":       KeyFunCut,
		"Control+V":       KeyFunPaste,
		"Shift+Control+V": KeyFunPasteHist,
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
		"Control+L":       KeyFunRecenter,
		"Control+.":       KeyFunComplete,
		"Control+F":       KeyFunFind,
		"Control+J":       KeyFunJump,
		"Control+[":       KeyFunHistPrev,
		"Control+]":       KeyFunHistNext,
	}},
	{"LinuxEmacs", "Linux with emacs-style navigation -- emacs wins in conflicts", KeyMap{
		"UpArrow":                 KeyFunMoveUp,
		"Shift+UpArrow":           KeyFunMoveUp,
		"Meta+UpArrow":            KeyFunMoveUp,
		"Control+P":               KeyFunMoveUp,
		"Shift+Control+P":         KeyFunMoveUp,
		"Meta+Control+P":          KeyFunMoveUp,
		"DownArrow":               KeyFunMoveDown,
		"Shift+DownArrow":         KeyFunMoveDown,
		"Meta+DownArrow":          KeyFunMoveDown,
		"Control+N":               KeyFunMoveDown,
		"Shift+Control+N":         KeyFunMoveDown,
		"Meta+Control+N":          KeyFunMoveDown,
		"RightArrow":              KeyFunMoveRight,
		"Shift+RightArrow":        KeyFunMoveRight,
		"Meta+RightArrow":         KeyFunEnd,
		"Control+F":               KeyFunMoveRight,
		"Shift+Control+F":         KeyFunMoveRight,
		"Meta+Control+F":          KeyFunMoveRight,
		"LeftArrow":               KeyFunMoveLeft,
		"Shift+LeftArrow":         KeyFunMoveLeft,
		"Meta+LeftArrow":          KeyFunHome,
		"Control+B":               KeyFunMoveLeft,
		"Shift+Control+B":         KeyFunMoveLeft,
		"Meta+Control+B":          KeyFunMoveLeft,
		"Control+UpArrow":         KeyFunPageUp,
		"Control+U":               KeyFunPageUp,
		"Control+DownArrow":       KeyFunPageDown,
		"Shift+Control+V":         KeyFunPageDown,
		"Alt+√":                   KeyFunPageDown,
		"Control+RightArrow":      KeyFunWordRight,
		"Control+LeftArrow":       KeyFunWordLeft,
		"Home":                    KeyFunHome,
		"Control+A":               KeyFunHome,
		"Shift+Control+A":         KeyFunHome,
		"Alt+LeftArrow":           KeyFunHome,
		"End":                     KeyFunEnd,
		"Control+E":               KeyFunEnd,
		"Shift+Control+E":         KeyFunEnd,
		"Alt+RightArrow":          KeyFunEnd,
		"Tab":                     KeyFunFocusNext,
		"Shift+Tab":               KeyFunFocusPrev,
		"ReturnEnter":             KeyFunEnter,
		"KeypadEnter":             KeyFunEnter,
		"Meta+A":                  KeyFunSelectAll,
		"Control+G":               KeyFunCancelSelect,
		"Control+Spacebar":        KeyFunSelectMode,
		"Control+ReturnEnter":     KeyFunAccept,
		"Escape":                  KeyFunAbort,
		"DeleteBackspace":         KeyFunBackspace,
		"Control+DeleteBackspace": KeyFunBackspaceWord,
		"DeleteForward":           KeyFunDelete,
		"Control+D":               KeyFunDelete,
		"Control+H":               KeyFunBackspace,
		"Control+K":               KeyFunKill,
		"Alt+∑":                   KeyFunCopy,
		"Control+C":               KeyFunCopy,
		"Meta+C":                  KeyFunCopy,
		"Control+W":               KeyFunCut,
		"Control+X":               KeyFunCut,
		"Meta+X":                  KeyFunCut,
		"Control+Y":               KeyFunPaste,
		"Control+V":               KeyFunPaste,
		"Meta+V":                  KeyFunPaste,
		"Shift+Meta+V":            KeyFunPasteHist,
		"Shift+Control+Y":         KeyFunPasteHist,
		"Control+M":               KeyFunDuplicate,
		"Control+I":               KeyFunInsert,
		"Control+O":               KeyFunInsertAfter,
		"Control+Alt+I":           KeyFunGoGiEditor,
		"Control+Alt+E":           KeyFunGoGiEditor,
		"Shift+Control+I":         KeyFunGoGiEditor,
		"Shift+Meta+=":            KeyFunZoomIn,
		"Meta+=":                  KeyFunZoomIn,
		"Meta+-":                  KeyFunZoomOut,
		"Control+=":               KeyFunZoomIn,
		"Shift+Control++":         KeyFunZoomIn,
		"Shift+Meta+-":            KeyFunZoomOut,
		"Control+-":               KeyFunZoomOut,
		"Shift+Control+_":         KeyFunZoomOut,
		"Control+Alt+P":           KeyFunPrefs,
		"F5":                      KeyFunRefresh,
		"Control+L":               KeyFunRecenter,
		"Control+.":               KeyFunComplete,
		"Control+S":               KeyFunSearch,
		"Meta+F":                  KeyFunFind,
		"Control+J":               KeyFunJump,
		"Control+[":               KeyFunHistPrev,
		"Control+]":               KeyFunHistNext,
	}},
	{"WindowsStd", "Standard Windows KeyMap", KeyMap{
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
		"Control+RightArrow": KeyFunWordRight,
		"Control+LeftArrow":  KeyFunWordLeft,
		"Home":               KeyFunHome,
		"Alt+LeftArrow":      KeyFunHome,
		"End":                KeyFunEnd,
		// "Control+E":           KeyFunEnd, // Search Google
		"Alt+RightArrow":  KeyFunEnd,
		"Tab":             KeyFunFocusNext,
		"Shift+Tab":       KeyFunFocusPrev,
		"ReturnEnter":     KeyFunEnter,
		"KeypadEnter":     KeyFunEnter,
		"Control+A":       KeyFunSelectAll,
		"Shift+Control+A": KeyFunCancelSelect,
		//"Control+Spacebar":    KeyFunSelectMode, // change input method / keyboard
		"Control+ReturnEnter":     KeyFunAccept,
		"Escape":                  KeyFunAbort,
		"DeleteBackspace":         KeyFunBackspace,
		"Control+DeleteBackspace": KeyFunBackspaceWord,
		"DeleteForward":           KeyFunDelete,
		// "Control+D":           KeyFunDelete, // Bookmark
		// "Control+H":       KeyFunBackspace, // Help
		"Control+K": KeyFunKill,
		"Control+C": KeyFunCopy,
		// "Control+W":       KeyFunCut, // Close Current Tab
		"Control+X":       KeyFunCut,
		"Control+V":       KeyFunPaste,
		"Shift+Control+V": KeyFunPasteHist,
		"Control+M":       KeyFunDuplicate,
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
		"Control+L":       KeyFunRecenter,
		"Control+.":       KeyFunComplete,
		"Control+[":       KeyFunHistPrev,
		"Control+]":       KeyFunHistNext,
	}},
	{"ChromeStd", "Standard chrome-browser and linux-under-chrome bindings", KeyMap{
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
		"Control+RightArrow": KeyFunWordRight,
		"Control+LeftArrow":  KeyFunWordLeft,
		"Home":               KeyFunHome,
		"Alt+LeftArrow":      KeyFunHome,
		"End":                KeyFunEnd,
		// "Control+E":           KeyFunEnd, // Search Google
		"Alt+RightArrow":  KeyFunEnd,
		"Tab":             KeyFunFocusNext,
		"Shift+Tab":       KeyFunFocusPrev,
		"ReturnEnter":     KeyFunEnter,
		"KeypadEnter":     KeyFunEnter,
		"Control+A":       KeyFunSelectAll,
		"Shift+Control+A": KeyFunCancelSelect,
		//"Control+Spacebar":    KeyFunSelectMode, // change input method / keyboard
		"Control+ReturnEnter":     KeyFunAccept,
		"Escape":                  KeyFunAbort,
		"DeleteBackspace":         KeyFunBackspace,
		"Control+DeleteBackspace": KeyFunBackspaceWord,
		"DeleteForward":           KeyFunDelete,
		// "Control+D":           KeyFunDelete, // Bookmark
		// "Control+H":       KeyFunBackspace, // Help
		"Control+K": KeyFunKill,
		"Control+C": KeyFunCopy,
		// "Control+W":       KeyFunCut, // Close Current Tab
		"Control+X":       KeyFunCut,
		"Control+V":       KeyFunPaste,
		"Shift+Control+V": KeyFunPasteHist,
		"Control+M":       KeyFunDuplicate,
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
		"Control+L":       KeyFunRecenter,
		"Control+.":       KeyFunComplete,
		"Control+F":       KeyFunFind,
		"Control+J":       KeyFunJump,
		"Control+[":       KeyFunHistPrev,
		"Control+]":       KeyFunHistNext,
	}},
}
