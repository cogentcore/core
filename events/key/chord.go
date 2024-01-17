// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package key

import (
	"fmt"
	"runtime"
	"strings"
	"unicode"
)

// TODO: consider adding chaining methods for constructing chords

// Chord represents the key chord associated with a given key function -- it
// is linked to the KeyChordEdit in the giv ValueView system so you can just
// type keys to set key chords.
type Chord string

func (ch Chord) String() string {
	return string(ch)
}

// NewChord returns a string representation of the keyboard event suitable for
// keyboard function maps, etc. Printable runes are sent directly, and
// non-printable ones are converted to their corresponding code names without
// the "Code" prefix.
func NewChord(rn rune, code Codes, mods Modifiers) Chord {
	modstr := ModsString(mods)
	if modstr != "" && code == CodeSpacebar { // modified space is not regular space
		return Chord(modstr + "Spacebar")
	}
	if unicode.IsPrint(rn) {
		if len(modstr) > 0 {
			return Chord(modstr + string(unicode.ToUpper(rn))) // all modded keys are uppercase!
		} else {
			return Chord(string(rn))
		}
	}
	// now convert code
	codestr := strings.TrimPrefix(any(code).(fmt.Stringer).String(), "Code")
	return Chord(modstr + codestr)
}

// OSShortcut translates Command into either Control or Meta depending on platform
func (ch Chord) OSShortcut() Chord {
	sc := string(ch)
	if runtime.GOOS == "darwin" {
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

// Decode decodes a chord string into rune and modifiers (set as bit flags)
func (ch Chord) Decode() (r rune, code Codes, mods Modifiers, err error) {
	cs := string(ch)
	mods, cs = ModsFmString(cs)
	rs := ([]rune)(cs)
	if len(rs) == 1 {
		r = rs[0]
		return
	}
	cstr := string(cs)
	code.SetString(cstr)
	if code != CodeUnknown {
		r = 0
		return
	}
	err = fmt.Errorf("goosi/events/key.DecodeChord got more/less than one rune: %v from remaining chord: %v", rs, string(cs))
	return
}

// Shortcut transforms chord string into short form suitable for display to users
func (ch Chord) Shortcut() string {
	// TODO: is this smart stuff actually helpful, or would it be much easier for
	// the user if they could just read the shortcuts in English?
	cs := strings.ReplaceAll(string(ch), "Control", "Ctrl")
	switch runtime.GOOS {
	case "darwin":
		cs = strings.ReplaceAll(cs, "Ctrl+", "^") // ⌃ doesn't look as good
		cs = strings.ReplaceAll(cs, "Shift+", "⇧")
		cs = strings.ReplaceAll(cs, "Meta+", "⌘")
		cs = strings.ReplaceAll(cs, "Alt+", "⌥")
	case "windows":
		cs = strings.ReplaceAll(cs, "Meta+", "Win+") // todo: actual windows key
	}
	cs = strings.ReplaceAll(cs, "Backspace", "⌫")
	cs = strings.ReplaceAll(cs, "Delete", "⌦")
	return cs
}
