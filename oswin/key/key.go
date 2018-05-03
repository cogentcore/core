// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/mobile/event:
//
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package key defines an event for physical keyboard keys, for the GoGi GUI
// system.
//
// On-screen software keyboards do not send key events.
package key

import (
	"fmt"
	"image"
	"strings"
	"unicode"

	"github.com/goki/goki/gi/oswin"
	"github.com/goki/goki/ki/kit"
)

// key.Event is a low-level immediately-generated key event, tracking press
// and release of keys -- suitable for fine-grained tracking of key events --
// see also key.ChordEvent for events that are generated only on keyboard release,
// and that include the full chord information about all the modifier keys
// that were present when a non-modifier key was released
type Event struct {
	oswin.EventBase

	// Rune is the meaning of the key event as determined by the
	// operating system. The mapping is determined by system-dependent
	// current layout, modifiers, lock-states, etc.
	//
	// If non-negative, it is a Unicode codepoint: pressing the 'a' key
	// generates different Runes 'a' or 'A' (but the same Code) depending on
	// the state of the shift key.
	//
	// If -1, the key does not generate a Unicode codepoint. To distinguish
	// them, look at Code.
	Rune rune

	// Code is the identity of the physical key relative to a notional
	// "standard" keyboard, independent of current layout, modifiers,
	// lock-states, etc
	//
	// For standard key codes, its value matches USB HID key codes.
	// Compare its value to uint32-typed constants in this package, such
	// as CodeLeftShift and CodeEscape.
	//
	// Pressing the regular '2' key and number-pad '2' key (with Num-Lock)
	// generate different Codes (but the same Rune).
	Code Code

	// Modifiers is bitflags representing a set of modifier keys: ModShift,
	// ModAlt, etc. -- bit positions are key.Modifiers
	Modifiers int32

	// Action is the key action taken: Press, Release, or None (for key repeats).
	Action Action

	// TODO: add a Device ID, for multiple input devices?
}

// key.ChordEvent reports events that are generated only on keyboard release,
// and that include the full chord information about all the modifier keys
// that were present when a non-modifier key was released -- these are
// generally appropriate for most uses
type ChordEvent struct {
	Event
}

func (e Event) String() string {
	if e.Rune >= 0 {
		return fmt.Sprintf("key.Event{%q (%v), %v, %v}", e.Rune, e.Code, e.Modifiers, e.Action)
	}
	return fmt.Sprintf("key.Event{(%v), %v, %v}", e.Code, e.Modifiers, e.Action)
}

// SetModifiers sets the bitflags based on a list of key.Modifiers
func (e *Event) SetModifiers(mods ...Modifiers) {
	for _, m := range mods {
		e.Modifiers |= (1 << uint32(m))
	}
}

// ChordString returns a string representation of the keyboard event suitable
// for keyboard function maps, etc -- printable runes are sent directly, and
// non-printable ones are converted to their corresponding code names without
// the "Code" prefix
func (e *Event) ChordString() string {
	modstr := ""
	for m := Shift; m < ModifiersN; m++ {
		if e.Modifiers&(1<<uint32(m)) != 0 {
			modstr += interface{}(m).(fmt.Stringer).String() + "+"
		}
	}
	if modstr != "" && e.Code == CodeSpacebar { // modified space is not regular space
		return modstr + "Spacebar"
	}
	if unicode.IsPrint(e.Rune) {
	   if len(modstr) > 0 {
	      return modstr + string(unicode.ToUpper(e.Rune)) // all modded keys are uppercase!
	      } else {
		return modstr + string(e.Rune)
		}
	}
	// now convert code
	codestr := strings.TrimPrefix(interface{}(e.Code).(fmt.Stringer).String(), "Code")
	return modstr + codestr
}

// CodeIsModifier returns true if given code is a modifier key
func CodeIsModifier(c Code) bool {
	if c >= CodeLeftControl && c <= CodeRightGUI {
		return true
	}
	return false
}

// Action is the action taken on the key
type Action int32

const (
	None Action = iota
	Press
	Release

	ActionN
)

//go:generate stringer -type=Action

var KiT_Action = kit.Enums.AddEnum(ActionN, false, nil)

// Modifiers are used as bitflags representing a set of modifier keys -- see
// SetModifiers method
type Modifiers int32

const (
	Shift Modifiers = iota
	Control
	Alt
	Meta // called "Command" on OS X

	ModifiersN
)

//go:generate stringer -type=Modifiers

var KiT_Modifiers = kit.Enums.AddEnum(ModifiersN, true, nil) // true = bitflag

// Code is the identity of a key relative to a notional "standard" keyboard.
type Code uint32

// Physical key codes.
//
// For standard key codes, its value matches USB HID key codes.
// TODO: add missing codes.
const (
	CodeUnknown Code = 0

	CodeA Code = 4
	CodeB Code = 5
	CodeC Code = 6
	CodeD Code = 7
	CodeE Code = 8
	CodeF Code = 9
	CodeG Code = 10
	CodeH Code = 11
	CodeI Code = 12
	CodeJ Code = 13
	CodeK Code = 14
	CodeL Code = 15
	CodeM Code = 16
	CodeN Code = 17
	CodeO Code = 18
	CodeP Code = 19
	CodeQ Code = 20
	CodeR Code = 21
	CodeS Code = 22
	CodeT Code = 23
	CodeU Code = 24
	CodeV Code = 25
	CodeW Code = 26
	CodeX Code = 27
	CodeY Code = 28
	CodeZ Code = 29

	Code1 Code = 30
	Code2 Code = 31
	Code3 Code = 32
	Code4 Code = 33
	Code5 Code = 34
	Code6 Code = 35
	Code7 Code = 36
	Code8 Code = 37
	Code9 Code = 38
	Code0 Code = 39

	CodeReturnEnter        Code = 40
	CodeEscape             Code = 41
	CodeDeleteBackspace    Code = 42
	CodeTab                Code = 43
	CodeSpacebar           Code = 44
	CodeHyphenMinus        Code = 45 // -
	CodeEqualSign          Code = 46 // =
	CodeLeftSquareBracket  Code = 47 // [
	CodeRightSquareBracket Code = 48 // ]
	CodeBackslash          Code = 49 // \
	CodeSemicolon          Code = 51 // ;
	CodeApostrophe         Code = 52 // '
	CodeGraveAccent        Code = 53 // `
	CodeComma              Code = 54 // ,
	CodeFullStop           Code = 55 // .
	CodeSlash              Code = 56 // /
	CodeCapsLock           Code = 57

	CodeF1  Code = 58
	CodeF2  Code = 59
	CodeF3  Code = 60
	CodeF4  Code = 61
	CodeF5  Code = 62
	CodeF6  Code = 63
	CodeF7  Code = 64
	CodeF8  Code = 65
	CodeF9  Code = 66
	CodeF10 Code = 67
	CodeF11 Code = 68
	CodeF12 Code = 69

	CodePause         Code = 72
	CodeInsert        Code = 73
	CodeHome          Code = 74
	CodePageUp        Code = 75
	CodeDeleteForward Code = 76
	CodeEnd           Code = 77
	CodePageDown      Code = 78

	CodeRightArrow Code = 79
	CodeLeftArrow  Code = 80
	CodeDownArrow  Code = 81
	CodeUpArrow    Code = 82

	CodeKeypadNumLock     Code = 83
	CodeKeypadSlash       Code = 84 // /
	CodeKeypadAsterisk    Code = 85 // *
	CodeKeypadHyphenMinus Code = 86 // -
	CodeKeypadPlusSign    Code = 87 // +
	CodeKeypadEnter       Code = 88
	CodeKeypad1           Code = 89
	CodeKeypad2           Code = 90
	CodeKeypad3           Code = 91
	CodeKeypad4           Code = 92
	CodeKeypad5           Code = 93
	CodeKeypad6           Code = 94
	CodeKeypad7           Code = 95
	CodeKeypad8           Code = 96
	CodeKeypad9           Code = 97
	CodeKeypad0           Code = 98
	CodeKeypadFullStop    Code = 99  // .
	CodeKeypadEqualSign   Code = 103 // =

	CodeF13 Code = 104
	CodeF14 Code = 105
	CodeF15 Code = 106
	CodeF16 Code = 107
	CodeF17 Code = 108
	CodeF18 Code = 109
	CodeF19 Code = 110
	CodeF20 Code = 111
	CodeF21 Code = 112
	CodeF22 Code = 113
	CodeF23 Code = 114
	CodeF24 Code = 115

	CodeHelp Code = 117

	CodeMute       Code = 127
	CodeVolumeUp   Code = 128
	CodeVolumeDown Code = 129

	CodeLeftControl  Code = 224
	CodeLeftShift    Code = 225
	CodeLeftAlt      Code = 226
	CodeLeftGUI      Code = 227 // Command on mac, ? on windows, ? on linux
	CodeRightControl Code = 228
	CodeRightShift   Code = 229
	CodeRightAlt     Code = 230
	CodeRightGUI     Code = 231

	// The following codes are not part of the standard USB HID Usage IDs for
	// keyboards. See http://www.usb.org/developers/hidpage/Hut1_12v2.pdf
	//
	// Usage IDs are uint16s, so these non-standard values start at 0x10000.

	// CodeCompose is the Code for a compose key, sometimes called a multi key,
	// used to input non-ASCII characters such as ñ being composed of n and ~.
	//
	// See https://en.wikipedia.org/wiki/Compose_key
	CodeCompose Code = 0x10000
)

// // go: generate stringer -type=Code

// TODO: Given we use runes outside the unicode space, should we provide a
// printing function? Related: it's a little unfortunate that printing a
// key.Event with %v gives not very readable output like:
//	{100 7 key.Modifiers() Press}

func (ev Event) Type() oswin.EventType {
	return oswin.KeyEvent
}

func (ev Event) HasPos() bool {
	return false
}

func (ev Event) Pos() image.Point {
	return image.ZP
}

func (ev Event) OnFocus() bool {
	return true
}

// check for interface implementation
var _ oswin.Event = &Event{}

func (ev ChordEvent) Type() oswin.EventType {
	return oswin.KeyChordEvent
}
