// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log"
	"sort"

	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  KeyFun is for mapping Keys to Functions

// functions that keyboard events can perform in the gui
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
	KeyFunAccept     // accept any changes and close dialog / move to next
	KeyFunAbort
	KeyFunCancelSelect
	KeyFunExtendSelect
	KeyFunSelectText
	KeyFunSelectAll
	KeyFunEditItem
	KeyFunCopy
	KeyFunCut
	KeyFunPaste
	KeyFunBackspace
	KeyFunDelete
	KeyFunKill
	KeyFunDuplicate
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
	KeyFunctionsN
)

//go:generate stringer -type=KeyFunctions

var KiT_KeyFunctions = kit.Enums.AddEnumAltLower(KeyFunctionsN, false, StylePropProps, "KeyFun")

func (ev KeyFunctions) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *KeyFunctions) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// todo: need to have multiple functions possible per key, depending on context?

type KeyMap map[string]KeyFunctions

// the default map has emacs-style navigation etc
var DefaultKeyMap = KeyMap{
	"UpArrow":             KeyFunMoveUp,
	"Control+P":           KeyFunMoveUp,
	"DownArrow":           KeyFunMoveDown,
	"Control+N":           KeyFunMoveDown,
	"RightArrow":          KeyFunMoveRight,
	"Control+F":           KeyFunMoveRight,
	"LeftArrow":           KeyFunMoveLeft,
	"Control+B":           KeyFunMoveLeft,
	"Control+UpArrow":     KeyFunPageUp,
	"Control+U":           KeyFunPageUp,
	"Control+DownArrow":   KeyFunPageDown,
	"Shift+Control+V":     KeyFunPageDown,
	"Alt+V":               KeyFunPageDown,
	"Control+RightArrow":  KeyFunPageRight,
	"Control+LeftArrow":   KeyFunPageLeft,
	"Home":                KeyFunHome,
	"Control+A":           KeyFunHome,
	"Meta+LeftArrow":      KeyFunHome,
	"End":                 KeyFunEnd,
	"Control+E":           KeyFunEnd,
	"Meta+RightArrow":     KeyFunEnd,
	"Tab":                 KeyFunFocusNext,
	"Shift+Tab":           KeyFunFocusPrev,
	"ReturnEnter":         KeyFunSelectItem,
	"KeypadEnter":         KeyFunSelectItem,
	"Shift+Control+A":     KeyFunSelectAll,
	"Meta+A":              KeyFunSelectAll,
	"Control+ReturnEnter": KeyFunAccept,
	"Escape":              KeyFunAbort,
	"Control+G":           KeyFunCancelSelect,
	"Alt+DownArrow":       KeyFunExtendSelect,
	"Control+Spacebar":    KeyFunSelectText,
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
}

// ActiveKeyMap points to the active map -- users can set this to an
// alternative map in Prefs
var ActiveKeyMap *KeyMap = &DefaultKeyMap

// KeyMapItem records one element of the key map -- used for organizing the map
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
