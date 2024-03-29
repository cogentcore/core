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
	Shift Modifiers = iota
	Control
	Alt
	// called "Command" on OS X
	Meta
)

// ModsString returns the string representation of the modifiers
func ModsString(mods Modifiers) string {
	modstr := ""
	for m := Shift; m < ModifiersN; m++ {
		if mods.HasFlag(m) {
			modstr += m.BitIndexString() + "+"
		}
	}
	return modstr
}

// ModsFromString returns the modifiers corresponding to given string
// and the remainder of the string after modifiers have been stripped
func ModsFromString(cs string) (Modifiers, string) {
	var mods Modifiers
	for m := Shift; m < ModifiersN; m++ {
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
