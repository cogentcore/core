// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package key

import (
	"fmt"
	"strings"
	"unicode"
)

// Chord represents the key chord associated with a given key function -- it
// is linked to the KeyChordEdit in the giv ValueView system so you can just
// type keys to set key chords.
type Chord string

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
	// if goosi.TheApp.Platform() == goosi.MacOS {
	// 	sc = strings.Replace(sc, "Command+", "Meta+", -1)
	// } else {
	sc = strings.Replace(sc, "Command+", "Control+", -1)
	// }
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
func (ch Chord) Decode() (r rune, mods Modifiers, err error) {
	cs := string(ch)
	mods, cs = ModsFmString(cs)
	rs := ([]rune)(cs)
	if len(rs) == 1 {
		r = rs[0]
	} else {
		err = fmt.Errorf("goosi/events/key.DecodeChord got more/less than one rune: %v from remaining chord: %v\n", rs, cs)
	}
	return
}

// Shortcut transforms chord string into short form suitable for display to users
func (ch Chord) Shortcut() string {
	cs := strings.Replace(string(ch), "Control+", "^", -1) // ⌃ doesn't look as good
	// switch goosi.TheApp.Platform() {
	// case goosi.MacOS:
	cs = strings.Replace(cs, "Shift+", "⇧", -1)
	cs = strings.Replace(cs, "Meta+", "⌘", -1)
	cs = strings.Replace(cs, "Alt+", "⌥", -1)
	// case goosi.Windows:
	// 	cs = strings.Replace(cs, "Shift+", "↑", -1)
	// 	cs = strings.Replace(cs, "Meta+", "Win+", -1) // todo: actual windows key
	// default:
	// 	cs = strings.Replace(cs, "Meta+", "", 1)
	// }
	cs = strings.Replace(cs, "DeleteBackspace", "⌫", -1)
	cs = strings.Replace(cs, "DeleteForward", "⌦", -1)
	return cs
}
