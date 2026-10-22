// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// TestCompleteTextRunes checks that completion edits use rune
// positions for the seed and inserted text, with non-ASCII text.
func TestCompleteTextRunes(t *testing.T) {
	b := core.NewBody()
	ed := NewEditor(b)
	ed.Lines.SetText([]byte("x := café.Fü\n"))
	ed.Complete = core.NewComplete()
	ed.Complete.EditFunc = func(data any, text string, cursorPos int, comp complete.Completion, seed string) complete.Edit {
		return complete.Edit{NewText: "café.Füße()", CursorAdjust: -1}
	}
	ed.Complete.Seed = "café.Fü"
	ed.Complete.SrcLn = 0
	ed.Complete.SrcCh = 12 // end of line, in runes
	ed.completeText("Füße")
	assert.Equal(t, "x := café.Füße()", strings.TrimRight(ed.Lines.String(), "\n"))
	assert.Equal(t, textpos.Pos{Line: 0, Char: 15}, ed.CursorPos) // inside the parens
}

// TestTypeOverAutoInserted checks that typing an auto-inserted closing
// bracket types over it, including after typing other text, so that
// typing code character by character does not duplicate closers.
func TestTypeOverAutoInserted(t *testing.T) {
	b := core.NewBody()
	ed := NewEditor(b)
	ed.Lines.SetText([]byte("\n"))
	typeText := func(s string) {
		for _, r := range s {
			ed.HandleEvent(events.NewKey(events.KeyChord, r, 0, 0))
		}
	}
	line := func() string { return string(ed.Lines.Line(0)) }

	typeText("f(")
	assert.Equal(t, "f()", line())
	typeText("a, g(b")
	assert.Equal(t, "f(a, g(b)", line())
	typeText(")")
	assert.Equal(t, "f(a, g(b)", line())
	typeText(")") // nothing pending: inserted
	assert.Equal(t, "f(a, g(b))", line())
	assert.Equal(t, 10, ed.CursorPos.Char)

	ed.Lines.SetText([]byte("\n"))
	ed.SetCursorShow(textpos.Pos{})
	typeText("x := m[k] + f(\"s\", 1)")
	assert.Equal(t, "x := m[k] + f(\"s\", 1)", line())
}
