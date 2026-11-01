// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// TestEditorCompose tests composing text with an input method, as sent
// by the drivers: the text being composed is shown at the cursor without
// changing the text, and the committed text follows its end as KeyChords.
func TestEditorCompose(t *testing.T) {
	b := core.NewBody()
	ed := newWideEditor(b)
	ed.Lines.SetString("hello world\n")
	pos := textpos.Pos{Line: 0, Char: 6}
	compose := func(text string, start, end int) {
		ed.HandleEvent(events.NewTextComposition(text, start, end))
	}
	b.AssertRender(t, "ime/compose", func() {
		ed.SetFocus()
		ed.SetCursorShow(pos)
		compose("にほん", 3, 3)
		compose("日本", 0, 2) // converted, with the clause selected
	}, func() {
		assert.Equal(t, "hello world\n", ed.Lines.String())
		vts := ed.VirtualText(imeLayer)
		if assert.Len(t, vts, 1) {
			assert.Equal(t, pos, vts[0].Pos)
			assert.Equal(t, "日本", string(vts[0].Text.Join()))
			assert.Len(t, vts[0].Text, 1) // all selected
		}
		// the input method cursor is at the end of the selected clause,
		// of two wide chars
		cur, ok := ed.VirtualCharStartPos(imeLayer, pos, 2)
		assert.True(t, ok)
		assert.Equal(t, ed.CharStartPos(pos).X+4*ed.charSize.X, cur.X)
	}, func() {
		// commit
		compose("", 0, 0)
		for _, r := range "日本" {
			ed.HandleEvent(events.NewKey(events.KeyChord, r, 0, 0))
		}
		assert.Nil(t, ed.VirtualText(imeLayer))
		assert.Equal(t, "hello 日本world\n", ed.Lines.String())
		assert.Equal(t, textpos.Pos{Line: 0, Char: 8}, ed.CursorPos)

		// composing replaces the selection, and canceling leaves it
		// deleted, as typing does
		ed.SelectRegion = textpos.Region{Start: textpos.Pos{Line: 0, Char: 0}, End: textpos.Pos{Line: 0, Char: 5}}
		ed.SetCursorShow(textpos.Pos{Line: 0, Char: 5})
		compose("x", 1, 1)
		assert.Equal(t, " 日本world\n", ed.Lines.String())
		if vts := ed.VirtualText(imeLayer); assert.Len(t, vts, 1) {
			assert.Equal(t, textpos.Pos{Line: 0, Char: 0}, vts[0].Pos)
		}
		compose("", 0, 0)
		assert.Equal(t, " 日本world\n", ed.Lines.String())
		assert.Nil(t, ed.VirtualText(imeLayer))
	})
}
