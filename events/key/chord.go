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

// Chord represents the key chord associated with a given key function; it
// is linked to the [cogentcore.org/core/core.KeyChordValue] so you can just
// type keys to set key chords.
type Chord string

// SystemPlatform is the string version of [cogentcore.org/core/system.App.SystemPlatform],
// which is set by system during initialization so that this package can conditionalize
// shortcut formatting based on the underlying system platform without import cycles.
var SystemPlatform string

// NewChord returns a string representation of the keyboard event suitable for
// keyboard function maps, etc. Printable runes are sent directly, and
// non-printable ones are converted to their corresponding code names without
// the "Code" prefix.
func NewChord(rn rune, code Codes, mods Modifiers) Chord {
	modstr := mods.ModifiersString()
	if modstr != "" && code == CodeSpacebar { // modified space is not regular space
		return Chord(modstr + "Spacebar")
	}
	if unicode.IsPrint(rn) {
		if len(modstr) > 0 {
			return Chord(modstr + string(unicode.ToUpper(rn))) // all modded keys are uppercase!
		}
		return Chord(string(rn))
	}
	// now convert code
	codestr := strings.TrimPrefix(code.String(), "Code")
	return Chord(modstr + codestr)
}

// PlatformChord translates Command into either Control or Meta depending on the platform
func (ch Chord) PlatformChord() Chord {
	sc := string(ch)
	if SystemPlatform == "MacOS" {
		sc = strings.ReplaceAll(sc, "Command+", "Meta+")
	} else {
		sc = strings.ReplaceAll(sc, "Command+", "Control+")
	}
	return Chord(sc)
}

// CodeIsModifier returns true if given code is a modifier key
func CodeIsModifier(c Codes) bool {
	return c >= CodeLeftControl && c <= CodeRightMeta
}

// IsMulti returns true if the Chord represents a multi-key sequence
func (ch Chord) IsMulti() bool {
	return strings.Contains(string(ch), " ")
}

// Chords returns the multiple keys represented in a multi-key sequence
func (ch Chord) Chords() []Chord {
	ss := strings.Fields(string(ch))
	nc := len(ss)
	if nc <= 1 {
		return []Chord{ch}
	}
	cc := make([]Chord, nc)
	for i, s := range ss {
		cc[i] = Chord(s)
	}
	return cc
}

// Decode decodes a chord string into rune and modifiers (set as bit flags)
func (ch Chord) Decode() (r rune, code Codes, mods Modifiers, err error) {
	cs := string(ch.PlatformChord())
	cs, _, _ = strings.Cut(cs, "\n") // we only care about the first chord
	mods, cs = ModifiersFromString(cs)
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
	err = fmt.Errorf("system/events/key.DecodeChord got more/less than one rune: %v from remaining chord: %v", rs, string(cs))
	return
}

// Label transforms the chord string into a short form suitable for display to users.
func (ch Chord) Label() string {
	cs := string(ch.PlatformChord())
	cs = strings.ReplaceAll(cs, "Control", "Ctrl")
	switch SystemPlatform {
	case "MacOS":
		if runtime.GOOS == "js" { // no font to display symbol on web
			cs = strings.ReplaceAll(cs, "Meta+", "Cmd+")
		} else {
			cs = strings.ReplaceAll(cs, "Meta+", "⌘")
			// need to have + after ⌘ when before other modifiers
			cs = strings.ReplaceAll(cs, "⌘Alt", "⌘+Alt")
			cs = strings.ReplaceAll(cs, "⌘Shift", "⌘+Shift")
		}
	case "Windows":
		cs = strings.ReplaceAll(cs, "Meta+", "Win+")
	}
	cs = strings.ReplaceAll(cs, "\n", " or ")
	return cs
}
