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

	"github.com/goki/gi/oswin"
	"github.com/goki/ki/kit"
)

// key.Event is a low-level immediately-generated key event, tracking press
// and release of keys -- suitable for fine-grained tracking of key events --
// see also key.ChordEvent for events that are generated only on key press,
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
	Code Codes

	// Modifiers is bitflags representing a set of modifier keys: ModShift,
	// ModAlt, etc. -- bit positions are key.Modifiers
	Modifiers int32

	// Action is the key action taken: Press, Release, or None (for key repeats).
	Action Actions

	// TODO: add a Device ID, for multiple input devices?
}

// key.ChordEvent reports events that are generated only on keyboard release,
// and that include the full chord information about all the modifier keys
// that were present when a non-modifier key was released -- these are
// generally appropriate for most uses
type ChordEvent struct {
	Event
}

func (ev *Event) String() string {
	if ev.Rune >= 0 {
		return fmt.Sprintf("Type: %v  Action: %v  Chord: %v  Rune: %d hex: %X  Mods: %v  Time: %v", ev.Type(), ev.Action, ev.Chord(), ev.Rune, ev.Rune, ModsString(ev.Modifiers), ev.Time())
	}
	return fmt.Sprintf("Type: %v  Action: %v  Code: %v  Mods: %v  Time: %v", ev.Type(), ev.Action, ev.Code, ModsString(ev.Modifiers), ev.Time())
}

// SetModifierBits sets the bitflags based on a list of key.Modifiers
func SetModifierBits(modBits *int32, mods ...Modifiers) {
	for _, m := range mods {
		*modBits |= (1 << uint32(m))
	}
}

// HasAnyModifierBits tests whether any of given modifier(s) bits were set
func HasAnyModifierBits(modBits int32, mods ...Modifiers) bool {
	for _, m := range mods {
		if modBits&(1<<uint32(m)) != 0 {
			return true
		}
	}
	return false
}

// HasAllModifiersBits tests whether all of given modifier(s) bits were set
func HasAllModifierBits(modBits int32, mods ...Modifiers) bool {
	for _, m := range mods {
		if modBits&(1<<uint32(m)) == 0 {
			return false
		}
	}
	return true
}

// SetModifiers sets the bitflags based on a list of key.Modifiers
func (e *Event) SetModifiers(mods ...Modifiers) {
	SetModifierBits(&e.Modifiers, mods...)
}

// HasAnyModifier tests whether any of given modifier(s) were set
func (e *Event) HasAnyModifier(mods ...Modifiers) bool {
	return HasAnyModifierBits(e.Modifiers, mods...)
}

// HasAllModifiers tests whether all of given modifier(s) were set
func (e *Event) HasAllModifier(mods ...Modifiers) bool {
	return HasAllModifierBits(e.Modifiers, mods...)
}

// key.Chord represents the key chord associated with a given key function -- it
// is linked to the KeyChordEdit in the giv ValueView system so you can just
// type keys to set key chords.
type Chord string

// Chord returns a string representation of the keyboard event suitable for
// keyboard function maps, etc -- printable runes are sent directly, and
// non-printable ones are converted to their corresponding code names without
// the "Code" prefix.
func (e *Event) Chord() Chord {
	modstr := ModsString(e.Modifiers)
	if modstr != "" && e.Code == CodeSpacebar { // modified space is not regular space
		return Chord(modstr + "Spacebar")
	}
	if unicode.IsPrint(e.Rune) {
		if len(modstr) > 0 {
			return Chord(modstr + string(unicode.ToUpper(e.Rune))) // all modded keys are uppercase!
		} else {
			return Chord(string(e.Rune))
		}
	}
	// now convert code
	codestr := strings.TrimPrefix(interface{}(e.Code).(fmt.Stringer).String(), "Code")
	return Chord(modstr + codestr)
}

// Decode decodes a chord string into rune and modifiers (set as bit flags)
func (ch Chord) Decode() (r rune, mods int32, err error) {
	cs := string(ch)
	mods, cs = ModsFmString(cs)
	rs := ([]rune)(cs)
	if len(rs) == 1 {
		r = rs[0]
	} else {
		err = fmt.Errorf("gi.oswin.key.DecodeChord got more/less than one rune: %v from remaining chord: %v\n", rs, cs)
	}
	return
}

// Shortcut transforms chord string into short form suitable for display to users
func (ch Chord) Shortcut() string {
	cs := strings.Replace(string(ch), "Control+", "^", -1) // ⌃ doesn't look as good
	switch oswin.TheApp.Platform() {
	case oswin.MacOS:
		cs = strings.Replace(cs, "Shift+", "⇧", -1)
		cs = strings.Replace(cs, "Meta+", "⌘", -1)
		cs = strings.Replace(cs, "Alt+", "⌥", -1)
	case oswin.Windows:
		cs = strings.Replace(cs, "Shift+", "↑", -1)
		cs = strings.Replace(cs, "Meta+", "Win+", -1) // todo: actual windows key
	default:
		cs = strings.Replace(cs, "Meta+", "", 1)
	}
	cs = strings.Replace(cs, "DeleteBackspace", "⌫", -1)
	cs = strings.Replace(cs, "DeleteForward", "⌦", -1)
	return cs
}

// OSShortcut translates Command into either Control or Meta depending on platform
func (ch Chord) OSShortcut() Chord {
	sc := string(ch)
	if oswin.TheApp.Platform() == oswin.MacOS {
		sc = strings.Replace(sc, "Command+", "Meta+", -1)
	} else {
		sc = strings.Replace(sc, "Command+", "Control+", -1)
	}
	return Chord(sc)
}

// CodeIsModifier returns true if given code is a modifier key
func CodeIsModifier(c Codes) bool {
	if c >= CodeLeftControl && c <= CodeRightMeta {
		return true
	}
	return false
}

// Actions is the action taken on the key
type Actions int32

const (
	NoAction Actions = iota
	Press
	Release

	ActionsN
)

//go:generate stringer -type=Actions

var KiT_Actions = kit.Enums.AddEnum(ActionsN, kit.NotBitFlag, nil)

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

var KiT_Modifiers = kit.Enums.AddEnum(ModifiersN, kit.BitFlag, nil)

// ModsString returns the string representation of the modifiers
func ModsString(mods int32) string {
	modstr := ""
	for m := Shift; m < ModifiersN; m++ {
		if mods&(1<<uint32(m)) != 0 {
			modstr += interface{}(m).(fmt.Stringer).String() + "+"
		}
	}
	return modstr
}

// ModsFmString returns the modifiers corresponding to given string
// and the remainder of the string after modifiers have been stripped
func ModsFmString(cs string) (int32, string) {
	var mods int32
	for m := Shift; m < ModifiersN; m++ {
		mstr := interface{}(m).(fmt.Stringer).String() + "+"
		if strings.HasPrefix(cs, mstr) {
			mods |= (1 << uint32(m))
			cs = strings.TrimPrefix(cs, mstr)
		}
	}
	return mods, cs
}

// Codes is the identity of a key relative to a notional "standard" keyboard.
type Codes uint32

// Physical key codes.
//
// For standard key codes, its value matches USB HID key codes.
// TODO: add missing codes.
const (
	CodeUnknown Codes = 0

	CodeA Codes = 4
	CodeB Codes = 5
	CodeC Codes = 6
	CodeD Codes = 7
	CodeE Codes = 8
	CodeF Codes = 9
	CodeG Codes = 10
	CodeH Codes = 11
	CodeI Codes = 12
	CodeJ Codes = 13
	CodeK Codes = 14
	CodeL Codes = 15
	CodeM Codes = 16
	CodeN Codes = 17
	CodeO Codes = 18
	CodeP Codes = 19
	CodeQ Codes = 20
	CodeR Codes = 21
	CodeS Codes = 22
	CodeT Codes = 23
	CodeU Codes = 24
	CodeV Codes = 25
	CodeW Codes = 26
	CodeX Codes = 27
	CodeY Codes = 28
	CodeZ Codes = 29

	Code1 Codes = 30
	Code2 Codes = 31
	Code3 Codes = 32
	Code4 Codes = 33
	Code5 Codes = 34
	Code6 Codes = 35
	Code7 Codes = 36
	Code8 Codes = 37
	Code9 Codes = 38
	Code0 Codes = 39

	CodeReturnEnter        Codes = 40
	CodeEscape             Codes = 41
	CodeDeleteBackspace    Codes = 42
	CodeTab                Codes = 43
	CodeSpacebar           Codes = 44
	CodeHyphenMinus        Codes = 45 // -
	CodeEqualSign          Codes = 46 // =
	CodeLeftSquareBracket  Codes = 47 // [
	CodeRightSquareBracket Codes = 48 // ]
	CodeBackslash          Codes = 49 // \
	CodeSemicolon          Codes = 51 // ;
	CodeApostrophe         Codes = 52 // '
	CodeGraveAccent        Codes = 53 // `
	CodeComma              Codes = 54 // ,
	CodeFullStop           Codes = 55 // .
	CodeSlash              Codes = 56 // /
	CodeCapsLock           Codes = 57

	CodeF1  Codes = 58
	CodeF2  Codes = 59
	CodeF3  Codes = 60
	CodeF4  Codes = 61
	CodeF5  Codes = 62
	CodeF6  Codes = 63
	CodeF7  Codes = 64
	CodeF8  Codes = 65
	CodeF9  Codes = 66
	CodeF10 Codes = 67
	CodeF11 Codes = 68
	CodeF12 Codes = 69

	CodePause         Codes = 72
	CodeInsert        Codes = 73
	CodeHome          Codes = 74
	CodePageUp        Codes = 75
	CodeDeleteForward Codes = 76
	CodeEnd           Codes = 77
	CodePageDown      Codes = 78

	CodeRightArrow Codes = 79
	CodeLeftArrow  Codes = 80
	CodeDownArrow  Codes = 81
	CodeUpArrow    Codes = 82

	CodeKeypadNumLock     Codes = 83
	CodeKeypadSlash       Codes = 84 // /
	CodeKeypadAsterisk    Codes = 85 // *
	CodeKeypadHyphenMinus Codes = 86 // -
	CodeKeypadPlusSign    Codes = 87 // +
	CodeKeypadEnter       Codes = 88
	CodeKeypad1           Codes = 89
	CodeKeypad2           Codes = 90
	CodeKeypad3           Codes = 91
	CodeKeypad4           Codes = 92
	CodeKeypad5           Codes = 93
	CodeKeypad6           Codes = 94
	CodeKeypad7           Codes = 95
	CodeKeypad8           Codes = 96
	CodeKeypad9           Codes = 97
	CodeKeypad0           Codes = 98
	CodeKeypadFullStop    Codes = 99  // .
	CodeKeypadEqualSign   Codes = 103 // =

	CodeF13 Codes = 104
	CodeF14 Codes = 105
	CodeF15 Codes = 106
	CodeF16 Codes = 107
	CodeF17 Codes = 108
	CodeF18 Codes = 109
	CodeF19 Codes = 110
	CodeF20 Codes = 111
	CodeF21 Codes = 112
	CodeF22 Codes = 113
	CodeF23 Codes = 114
	CodeF24 Codes = 115

	CodeHelp Codes = 117

	CodeMute       Codes = 127
	CodeVolumeUp   Codes = 128
	CodeVolumeDown Codes = 129

	CodeLeftControl  Codes = 224
	CodeLeftShift    Codes = 225
	CodeLeftAlt      Codes = 226
	CodeLeftMeta     Codes = 227 // Command on mac, win key on windows, ? on linux
	CodeRightControl Codes = 228
	CodeRightShift   Codes = 229
	CodeRightAlt     Codes = 230
	CodeRightMeta    Codes = 231

	// The following codes are not part of the standard USB HID Usage IDs for
	// keyboards. See http://www.usb.org/developers/hidpage/Hut1_12v2.pdf
	//
	// Usage IDs are uint16s, so these non-standard values start at 0x10000.

	// CodeCompose is the Code for a compose key, sometimes called a multi key,
	// used to input non-ASCII characters such as Ã± being composed of n and ~.
	//
	// See https://en.wikipedia.org/wiki/Compose_key
	CodeCompose Codes = 0x10000
)

// note: have to use official go stringer for this, not custom one, which doesn't currently
// handle discontinuities
// // go: generate stringer -type=Codes

// TODO: Given we use runes outside the unicode space, should we provide a
// printing function? Related: it's a little unfortunate that printing a
// key.Event with %v gives not very readable output like:
//	{100 7 key.Modifiers() Press}

func (ev *Event) Type() oswin.EventType {
	return oswin.KeyEvent
}

func (ev *Event) HasPos() bool {
	return false
}

func (ev *Event) Pos() image.Point {
	return image.ZP
}

func (ev *Event) OnFocus() bool {
	return true
}

// check for interface implementation
var _ oswin.Event = &Event{}

func (ev *ChordEvent) Type() oswin.EventType {
	return oswin.KeyChordEvent
}

var CodeRuneMap = map[Codes]rune{
	CodeA: 'A',
	CodeB: 'B',
	CodeC: 'C',
	CodeD: 'D',
	CodeE: 'E',
	CodeF: 'F',
	CodeG: 'G',
	CodeH: 'H',
	CodeI: 'I',
	CodeJ: 'J',
	CodeK: 'K',
	CodeL: 'L',
	CodeM: 'M',
	CodeN: 'N',
	CodeO: 'O',
	CodeP: 'P',
	CodeQ: 'Q',
	CodeR: 'R',
	CodeS: 'S',
	CodeT: 'T',
	CodeU: 'U',
	CodeV: 'V',
	CodeW: 'W',
	CodeX: 'X',
	CodeY: 'Y',
	CodeZ: 'Z',

	Code1: '1',
	Code2: '2',
	Code3: '3',
	Code4: '4',
	Code5: '5',
	Code6: '6',
	Code7: '7',
	Code8: '8',
	Code9: '9',
	Code0: '0',

	CodeTab:                '\t',
	CodeSpacebar:           ' ',
	CodeHyphenMinus:        '-',
	CodeEqualSign:          '=',
	CodeLeftSquareBracket:  '[',
	CodeRightSquareBracket: ']',
	CodeBackslash:          '\\',
	CodeSemicolon:          ';',
	CodeApostrophe:         '\'',
	CodeGraveAccent:        '`',
	CodeComma:              ',',
	CodeFullStop:           '.',
	CodeSlash:              '/',

	CodeKeypadSlash:       '/',
	CodeKeypadAsterisk:    '*',
	CodeKeypadHyphenMinus: '-',
	CodeKeypadPlusSign:    '+',
	CodeKeypadFullStop:    '.',
	CodeKeypadEqualSign:   '=',
}
