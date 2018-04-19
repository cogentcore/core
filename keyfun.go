// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "fmt"

////////////////////////////////////////////////////////////////////////////////////////
//  Mapping Keys to Functions

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
	KeyFunSelectItem
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
	KeyFunctionsN
)

//go:generate stringer -type=KeyFunctions

// todo: need to have multiple functions possible per key, depending on context?

type KeyMap map[string]KeyFunctions

// the default map has emacs-style navigation etc
var DefaultKeyMap = KeyMap{
	"up_arrow":                 KeyFunMoveUp,
	"Control+p":                KeyFunMoveUp,
	"down_arrow":               KeyFunMoveDown,
	"Control+n":                KeyFunMoveDown,
	"right_arrow":              KeyFunMoveRight,
	"Control+f":                KeyFunMoveRight,
	"left_arrow":               KeyFunMoveLeft,
	"Control+b":                KeyFunMoveLeft,
	"Control+up_arrow":         KeyFunPageUp,
	"Control+u":                KeyFunPageUp,
	"Control+down_arrow":       KeyFunPageDown,
	"Control+v":                KeyFunPageDown,
	"Control+right_arrow":      KeyFunPageRight,
	"Control+left_arrow":       KeyFunPageLeft,
	"home":                     KeyFunHome,
	"kp_home":                  KeyFunHome,
	"Control+a":                KeyFunHome,
	"super+Control+left_arrow": KeyFunHome,
	"end":                       KeyFunEnd,
	"kp_end":                    KeyFunEnd,
	"Control+e":                 KeyFunEnd,
	"super+Control+right_arrow": KeyFunEnd,
	"tab":            KeyFunFocusNext,
	"shift+tab":      KeyFunFocusPrev,
	"return":         KeyFunSelectItem,
	"Control+return": KeyFunSelectItem,
	"escape":         KeyFunAbort,
	"Control+g":      KeyFunCancelSelect,
	// "Control+down_arrow": KeyFunExtendSelect,
	"Control+space": KeyFunSelectText,
	"left_shift":    KeyFunShift,
	"right_shift":   KeyFunShift,
	"left_super":    KeyFunCtrl,
	"right_super":   KeyFunCtrl,
	"backspace":     KeyFunBackspace,
	"delete":        KeyFunDelete,
	"Control+d":     KeyFunDelete,
	"Control+h":     KeyFunBackspace,
	"Control+k":     KeyFunKill,
	"Control+m":     KeyFunDuplicate,
	"Control+i":     KeyFunInsert,
	"Control+o":     KeyFunInsertAfter,
	"Alt+Control+i": KeyFunGoGiEditor,
	"Alt+Control+e": KeyFunGoGiEditor,
}

// users can set this to an alternative map
var ActiveKeyMap *KeyMap = &DefaultKeyMap

// translate chord into keyboard function -- use oswin key.ChordString to get chord
func KeyFun(chord string) KeyFunctions {
	kf := KeyFunNil
	if chord != "" {
		kf = (*ActiveKeyMap)[chord]
		fmt.Printf("chord: %v = %v\n", chord, kf)
	}
	return kf
}
