// Copyright (c) 2023, Cogent Core. All rights reserved.
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
	RunChordDecode("a", t)
	RunChordDecode("Control+A", t)
	RunChordDecode("ReturnEnter", t)
	RunChordDecode("KeypadEnter", t)
	RunChordDecode("Backspace", t)
	RunChordDecode("Escape", t)
}
