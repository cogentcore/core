// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"strings"

	"goki.dev/enums"
)

// Modifiers are used as bitflags representing a set of modifier keys.
type Modifiers int64 //enums:bitflag

const (
	Shift Modifiers = iota
	Control
	Alt
	Meta // called "Command" on OS X
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

// ModsFmString returns the modifiers corresponding to given string
// and the remainder of the string after modifiers have been stripped
func ModsFmString(cs string) (Modifiers, string) {
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
