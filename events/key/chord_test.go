// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package key

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChordDecode(t *testing.T) {
	RunChordDecode(t, "a")
	RunChordDecode(t, "Control+A")
	RunChordDecode(t, "ReturnEnter")
	RunChordDecode(t, "KeypadEnter")
	RunChordDecode(t, "Backspace")
	RunChordDecode(t, "Escape")
}

func RunChordDecode(t *testing.T, ch Chord) {
	r, code, mods, err := ch.Decode()
	if assert.NoError(t, err) {
		nch := NewChord(r, code, mods)
		assert.Equal(t, ch, nch)
	}
}
