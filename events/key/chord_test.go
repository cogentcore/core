// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package key

import (
	"fmt"
	"testing"
)

func RunChordDecode(ch Chord, t *testing.T) {
	r, code, mods, err := ch.Decode()
	if err != nil {
		t.Error(err.Error())
		return
	}
	fmt.Println("ch:", ch, "r:", r, "code:", code.String(), "mods:", mods.String())
	nch := NewChord(r, code, mods)
	if nch != ch {
		t.Error("ChordDecode error: orig:", ch.String(), "new:", nch.String())
	}
}

func TestChordDecode(t *testing.T) {
	RunChordDecode(Chord("a"), t)
	RunChordDecode(Chord("Control+A"), t)
	RunChordDecode(Chord("ReturnEnter"), t)
	RunChordDecode(Chord("KeypadEnter"), t)
	RunChordDecode(Chord("Backspace"), t)
	RunChordDecode(Chord("Escape"), t)
}
