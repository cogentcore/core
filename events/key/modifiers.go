// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package key

//go:generate core generate

import (
	"strings"

	"cogentcore.org/core/enums"
)

// Modifiers are used as bitflags representing a set of modifier keys.
type Modifiers int64 //enums:bitflag

const (
	// Control is the "Control" (Ctrl) key.
	Control Modifiers = iota

	// Meta is the system meta key (the "Command" key on macOS
	// and the Windows key on Windows).
	Meta

	// Alt is the "Alt" ("Option" on macOS) key.
	Alt

	// Shift is the "Shift" key.
	Shift
)

// ModifiersString returns the string representation of the modifiers using
// plus symbols as seperators. The order is given by Modifiers order:
// Control, Meta, Alt, Shift.
func (mo Modifiers) ModifiersString() string {
	modstr := ""
	for _, m := range ModifiersValues() {
		if mo.HasFlag(m) {
			modstr += m.BitIndexString() + "+"
		}
	}
	return modstr
}

// ModifiersFromString returns the modifiers corresponding to given string
// and the remainder of the string after modifiers have been stripped. This only
// works for modifiers followed with a plus sign (ex: "Control+C" will work, but
// "Control" will not). See [Modifiers.SetString] for simple setting of modifiers
// without the plus sign.
func ModifiersFromString(cs string) (Modifiers, string) {
	var mods Modifiers
	for _, m := range ModifiersValues() {
		mstr := m.BitIndexString() + "+"
		if strings.HasPrefix(cs, mstr) {
			mods.SetFlag(true, m)
			cs = strings.TrimPrefix(cs, mstr)
		}
	}
	return mods, cs
}

// HasAnyModifier tests whether any of given modifier(s) were set
func HasAnyModifier(flags Modifiers, mods ...enums.BitFlag) bool {
	for _, m := range mods {
		if flags.HasFlag(m) {
			return true
		}
	}
	return false
}

// HasAllModifiers tests whether all of given modifier(s) were set
func HasAllModifiers(flags Modifiers, mods ...enums.BitFlag) bool {
	for _, m := range mods {
		if !flags.HasFlag(m) {
			return false
		}
	}
	return true
}
