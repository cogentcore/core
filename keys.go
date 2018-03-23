// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

/*
   directly copied from https://github.com/skelterjohn/go.wde

   Copyright 2012 the go.wde authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	// "fmt"
	"sort"
	"strings"
)

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
	KeyFunCancelSelect
	KeyFunExtendSelect
	KeyFunSelectText
	KeyFunEditItem
	KeyFunCopy
	KeyFunCut
	KeyFunPaste
	KeyFunBackspace
	KeyFunDelete
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
	"home":                     KeyFunHome,
	"kp_home":                  KeyFunHome,
	"control+a":                KeyFunHome,
	"super+control+left_arrow": KeyFunHome,
	"end":                       KeyFunEnd,
	"kp_end":                    KeyFunEnd,
	"control+e":                 KeyFunEnd,
	"super+control+right_arrow": KeyFunEnd,
	"tab":                KeyFunFocusNext,
	"shift_tab":          KeyFunFocusPrev,
	"return":             KeyFunSelectItem,
	"control+g":          KeyFunCancelSelect,
	"control+down_arrow": KeyFunExtendSelect,
	"control+space":      KeyFunSelectText,
	"left_shift":         KeyFunShift,
	"right_shift":        KeyFunShift,
	"left_super":         KeyFunCtrl,
	"right_super":        KeyFunCtrl,
	"backspace":          KeyFunBackspace,
	"delete":             KeyFunDelete,
	"control+d":          KeyFunDelete,
	"control+h":          KeyFunBackspace,
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

////////////////////////////////////////////////////////////////////////////////////////
//  Basic Keys

const (
	KeyFunction     = "function"
	KeyLeftSuper    = "left_super"
	KeyRightSuper   = "right_super"
	KeyLeftAlt      = "left_alt"
	KeyRightAlt     = "right_alt"
	KeyLeftControl  = "left_control"
	KeyRightControl = "right_control"
	KeyLeftShift    = "left_shift"
	KeyRightShift   = "right_shift"
	KeyUpArrow      = "up_arrow"
	KeyDownArrow    = "down_arrow"
	KeyLeftArrow    = "left_arrow"
	KeyRightArrow   = "right_arrow"
	KeyInsert       = "insert"
	KeyTab          = "tab"
	KeySpace        = "space"
	KeyA            = "a"
	KeyB            = "b"
	KeyC            = "c"
	KeyD            = "d"
	KeyE            = "e"
	KeyF            = "f"
	KeyG            = "g"
	KeyH            = "h"
	KeyI            = "i"
	KeyJ            = "j"
	KeyK            = "k"
	KeyL            = "l"
	KeyM            = "m"
	KeyN            = "n"
	KeyO            = "o"
	KeyP            = "p"
	KeyQ            = "q"
	KeyR            = "r"
	KeyS            = "s"
	KeyT            = "t"
	KeyU            = "u"
	KeyV            = "v"
	KeyW            = "w"
	KeyX            = "x"
	KeyY            = "y"
	KeyZ            = "z"
	Key1            = "1"
	Key2            = "2"
	Key3            = "3"
	Key4            = "4"
	Key5            = "5"
	Key6            = "6"
	Key7            = "7"
	Key8            = "8"
	Key9            = "9"
	Key0            = "0"
	KeyPadEnd       = "kp_end"
	KeyPadDown      = "kp_down"
	KeyPadNext      = "kp_next"
	KeyPadLeft      = "kp_left"
	KeyPadBegin     = "kp_begin"
	KeyPadRight     = "kp_right"
	KeyPadHome      = "kp_home"
	KeyPadUp        = "kp_up"
	KeyPadPrior     = "kp_prior"
	KeyPadInsert    = "kp_insert"
	KeyPadSlash     = "kp_slash"
	KeyPadStar      = "kp_star"
	KeyPadMinus     = "kp_minus"
	KeyPadPlus      = "kp_plus"
	KeyPadDot       = "kp_dot"
	KeyPadEqual     = "kp_equal"
	KeyPadEnter     = "kp_enter"
	KeyBackTick     = "`"
	KeyF1           = "f1"
	KeyF2           = "f2"
	KeyF3           = "f3"
	KeyF4           = "f4"
	KeyF5           = "f5"
	KeyF6           = "f6"
	KeyF7           = "f7"
	KeyF8           = "f8"
	KeyF9           = "f9"
	KeyF10          = "f10"
	KeyF11          = "f11"
	KeyF12          = "f12"
	KeyF13          = "f13"
	KeyF14          = "f14"
	KeyF15          = "f15"
	KeyF16          = "f16"
	KeyMinus        = "-"
	KeyEqual        = "="
	KeyLeftBracket  = "["
	KeyRightBracket = "]"
	KeyBackslash    = `\`
	KeySemicolon    = ";"
	KeyQuote        = "'"
	KeyComma        = ","
	KeyPeriod       = "."
	KeySlash        = "/"
	KeyReturn       = "return"
	KeyEscape       = "escape"
	KeyNumlock      = "numlock"
	KeyDelete       = "delete"
	KeyBackspace    = "backspace"
	KeyHome         = "home"
	KeyEnd          = "end"
	KeyPrior        = "prior"
	KeyNext         = "next"
	KeyCapsLock     = "caps"
)

var chordPrecedence = []string{
	"super",
	"shift",
	"alt",
	"control",
	"function",
}

var chordIndices map[string]int

func init() {
	chordIndices = map[string]int{}
	for i, k := range chordPrecedence {
		// we give these negative values so that when a lookup is done on something
		// that is not in this list, it gets 0 (the default), and comes after each
		// of the keys indicated in chordPrecedence.
		chordIndices[k] = i - len(chordPrecedence)
	}
}

type ChordSorter []string

func (c ChordSorter) Len() int {
	return len(c)
}
func (c ChordSorter) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c ChordSorter) Less(i, j int) (less bool) {
	ip := chordIndices[c[i]]
	jp := chordIndices[c[j]]
	if ip == 0 && jp == 0 {
		if len(c[i]) != len(c[j]) {
			return len(c[i]) > len(c[j])
		}
		return c[i] < c[j]
	}
	return ip < jp
}

func ConstructChord(keys map[string]bool) (chord string) {
	unikeys := map[string]bool{}
	for key := range keys {
		if !strings.HasSuffix(key, "_arrow") {
			if strings.HasPrefix(key, "left_") && chordIndices[key[5:]] != 0 {
				key = key[5:]
			}
			if strings.HasPrefix(key, "right_") && chordIndices[key[6:]] != 0 {
				key = key[6:]
			}
		}
		unikeys[key] = true
	}
	if len(unikeys) <= 1 {
		return
	}
	allkeys := ChordSorter{}
	for key := range unikeys {
		allkeys = append(allkeys, key)
	}

	sort.Sort(allkeys)
	chord = strings.Join(allkeys, "+")
	return
}
