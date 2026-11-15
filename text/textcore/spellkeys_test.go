// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spellKey returns a key chord event of the given key function, or of
// the given rune if kf is keymap.None.
func spellKey(t *testing.T, kf keymap.Functions, r rune) events.Event {
	t.Helper()
	if kf == keymap.None {
		return events.NewKey(events.KeyChord, r, key.CodeA, 0)
	}
	rn, code, mods, err := kf.Chord().Decode()
	require.NoError(t, err)
	return events.NewKey(events.KeyChord, rn, code, mods)
}

// TestSpellKeyPositions tests checking the spelling of the word before or
// at the cursor for the keys that end or edit a word, at each kind of
// cursor position.
func TestSpellKeyPositions(t *testing.T) {
	spellTest(t, "tset at end\nmore tset words\n", func(ed *Editor) {
		check := func(p textpos.Pos, kf keymap.Functions, r rune, want bool, msg string) {
			t.Helper()
			ed.Lines.SetTags(p.Line, nil)
			ed.SetCursorShow(p)
			ed.iSpellKeyInput(spellKey(t, kf, r))
			assert.Equal(t, want, hasSpellErr(ed, p.Line), msg)
		}
		check(pos(0, 4), keymap.MoveRight, 0, true, "right at the end of a word")
		check(pos(1, 9), keymap.MoveRight, 0, true, "right at a break after a word")
		check(pos(1, 15), keymap.MoveRight, 0, false, "right at the end of the line, of a known word")
		check(pos(1, 2), keymap.MoveRight, 0, false, "right within a word")
		check(pos(1, 9), keymap.FocusNext, 0, true, "tab after a word")
		check(pos(1, 7), keymap.Delete, 0, true, "delete within a word")
		check(pos(1, 9), keymap.Backspace, 0, true, "backspace after a word")
		check(pos(1, 7), keymap.None, 'x', true, "typing within a word")
		check(pos(1, 9), keymap.None, '.', true, "punctuation after a word")

		// right at the start of a line checks the word at the end of the previous one
		ed.Lines.SetTags(0, nil)
		ed.SetCursorShow(pos(1, 0))
		ed.iSpellKeyInput(spellKey(t, keymap.MoveRight, 0))
	})
}
