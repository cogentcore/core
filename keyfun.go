// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "github.com/goki/goki/ki/kit"

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
	"Control+V":           KeyFunPageDown,
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
	"Control+ReturnEnter": KeyFunAccept,
	"Escape":              KeyFunAbort,
	"Control+G":           KeyFunCancelSelect,
	// "Control+DownArrow": KeyFunExtendSelect,
	"Control+Spacebar": KeyFunSelectText,
	"DeleteBackspace":  KeyFunBackspace,
	"DeleteForward":    KeyFunDelete,
	"Control+D":        KeyFunDelete,
	"Control+H":        KeyFunBackspace,
	"Control+K":        KeyFunKill,
	"Control+M":        KeyFunDuplicate,
	"Control+I":        KeyFunInsert,
	"Control+O":        KeyFunInsertAfter,
	"Control+Alt+I":    KeyFunGoGiEditor,
	"Control+Alt+E":    KeyFunGoGiEditor,
	"Shift+Meta+=":     KeyFunZoomIn,
	"Meta+=":           KeyFunZoomIn,
	"Meta+-":           KeyFunZoomOut,
	"Control+=":     KeyFunZoomIn,
	"Shift+Control++":     KeyFunZoomIn,
	"Shift+Meta+-":     KeyFunZoomOut,
	"Control+-":     KeyFunZoomOut,
	"Shift+Control+_":     KeyFunZoomOut,
	"Control+Alt+P":    KeyFunPrefs,
	"F5":               KeyFunRefresh,
}

// ActiveKeyMap points to the active map -- users can set this to an
// alternative map in Prefs
var ActiveKeyMap *KeyMap = &DefaultKeyMap

// translate chord into keyboard function -- use oswin key.ChordString to get chord
func KeyFun(chord string) KeyFunctions {
	kf := KeyFunNil
	if chord != "" {
		kf = (*ActiveKeyMap)[chord]
		// fmt.Printf("chord: %v = %v\n", chord, kf)
	}
	return kf
}
