// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

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
	"control+p":                KeyFunMoveUp,
	"down_arrow":               KeyFunMoveDown,
	"control+n":                KeyFunMoveDown,
	"right_arrow":              KeyFunMoveRight,
	"control+f":                KeyFunMoveRight,
	"left_arrow":               KeyFunMoveLeft,
	"control+b":                KeyFunMoveLeft,
	"control+up_arrow":         KeyFunPageUp,
	"control+u":                KeyFunPageUp,
	"control+down_arrow":       KeyFunPageDown,
	"control+v":                KeyFunPageDown,
	"control+right_arrow":      KeyFunPageRight,
	"control+left_arrow":       KeyFunPageLeft,
	"home":                     KeyFunHome,
	"kp_home":                  KeyFunHome,
	"control+a":                KeyFunHome,
	"super+control+left_arrow": KeyFunHome,
	"end":                       KeyFunEnd,
	"kp_end":                    KeyFunEnd,
	"control+e":                 KeyFunEnd,
	"super+control+right_arrow": KeyFunEnd,
	"tab":       KeyFunFocusNext,
	"shift_tab": KeyFunFocusPrev,
	"return":    KeyFunSelectItem,
	"escape":    KeyFunAbort,
	"control+g": KeyFunCancelSelect,
	// "control+down_arrow": KeyFunExtendSelect,
	"control+space": KeyFunSelectText,
	"left_shift":    KeyFunShift,
	"right_shift":   KeyFunShift,
	"left_super":    KeyFunCtrl,
	"right_super":   KeyFunCtrl,
	"backspace":     KeyFunBackspace,
	"delete":        KeyFunDelete,
	"control+d":     KeyFunDelete,
	"control+h":     KeyFunBackspace,
	"control+k":     KeyFunKill,
}

// users can set this to an alternative map
var ActiveKeyMap *KeyMap = &DefaultKeyMap

// translate key string into a function
func KeyFun(key, chord string) KeyFunctions {
	kf := KeyFunNil
	if key != "" {
		kf = (*ActiveKeyMap)[key]
		// fmt.Printf("key: %v = %v\n", key, kf)
	}
	if chord != "" {
		kf = (*ActiveKeyMap)[chord]
		// fmt.Printf("chord: %v = %v\n", chord, kf)
	}
	return kf
}
