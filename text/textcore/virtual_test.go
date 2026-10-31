// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"image"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// newWideEditor returns a new editor that is wide enough
// for the test lines to not wrap.
func newWideEditor(b *core.Body) *Editor {
	ed := NewEditor(b)
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Ch(40)
	})
	return ed
}

// clickPoint returns the point passed to [Base.PixelToCursor], as for a
// mouse click, in the first half of the char at the given render coords,
// as returned by [Base.CharStartPos], for an editor that is not scrolled.
// A click in the first half of a char goes to the position before it.
func clickPoint(ed *Base, csp math32.Vector2) image.Point {
	pt := csp.Add(ed.charSize.Mul(math32.Vec2(0.25, 0.5)))
	pt = pt.Sub(ed.Geom.Pos.Content).Sub(math32.FromPoint(ed.Scene.SceneGeom.Pos))
	return pt.ToPoint()
}

// TestEditorVirtualIME tests the composition of an input method as virtual
// text in an editor: the text after it is shown after it, the candidate
// window position is at its cursor, and clicks on it go to its position.
func TestEditorVirtualIME(t *testing.T) {
	b := core.NewBody()
	ed := newWideEditor(b)
	ed.Lines.SetString("hello world\nsecond line\n")
	pos := textpos.Pos{Line: 0, Char: 6}
	b.AssertRender(t, "virtual/ime", func() {
		ed.SetCursorShow(pos)
		ed.SetVirtualText("ime", lines.VirtualText{Pos: pos, Text: rich.NewText(ed.CompositionStyle(), []rune("compose"))})
	}, func() {
		cw := ed.charSize.X
		p6 := ed.CharStartPos(pos)
		p7 := ed.CharStartPos(textpos.Pos{Line: 0, Char: 7})
		assert.Equal(t, 8*cw, p7.X-p6.X) // "compose" and "w"

		cur, ok := ed.VirtualCharStartPos("ime", pos, 3)
		assert.True(t, ok)
		assert.Equal(t, p6.X+3*cw, cur.X)
		assert.Equal(t, p6.Y, cur.Y)

		assert.Equal(t, textpos.Pos{Line: 0, Char: 7}, ed.PixelToCursor(clickPoint(&ed.Base, p7)))
		assert.Equal(t, pos, ed.PixelToCursor(clickPoint(&ed.Base, cur)))
		assert.Equal(t, "hello world\nsecond line\n", ed.Lines.String())
	})
}

// TestEditorVirtualInlay tests language server inlay hints, which are
// shown in all editors of the same text, unlike the composition of an
// input method, which is shown only in its editor.
func TestEditorVirtualInlay(t *testing.T) {
	b := core.NewBody()
	ed := newWideEditor(b)
	ed.Lines.SetString("foo(1, 2)\n")
	twin := newWideEditor(b)
	twin.SetLines(ed.Lines)
	b.AssertRender(t, "virtual/inlay", func() {
		ed.Lines.SetVirtualText("inlay", -1,
			lines.VirtualText{Pos: textpos.Pos{Line: 0, Char: 4}, Text: rich.NewText(ed.HintStyle(), []rune("a: "))},
			lines.VirtualText{Pos: textpos.Pos{Line: 0, Char: 7}, Text: rich.NewText(ed.HintStyle(), []rune("b: "))})
		ed.SetVirtualText("ime", lines.VirtualText{Pos: textpos.Pos{Line: 0, Char: 9}, Text: rich.NewText(ed.CompositionStyle(), []rune("x"))})
	}, func() {
		end := textpos.Pos{Line: 0, Char: 9}
		assert.Equal(t, textpos.Pos{Line: 0, Char: 15}, ed.Lines.PosToView(ed.viewId, end))
		assert.Equal(t, textpos.Pos{Line: 0, Char: 15}, twin.Lines.PosToView(twin.viewId, end))
		assert.Equal(t, 17, ed.Lines.ViewLineLen(ed.viewId, 0))     // with "x"
		assert.Equal(t, 16, twin.Lines.ViewLineLen(twin.viewId, 0)) // without

		// typing moves the cursor over the hints, which are not text
		ed.SetCursorShow(textpos.Pos{Line: 0, Char: 4})
		ed.HandleEvent(events.NewKey(events.KeyChord, 'z', 0, 0))
		assert.Equal(t, "foo(z1, 2)\n", ed.Lines.String())
		assert.Equal(t, textpos.Pos{Line: 0, Char: 5}, ed.CursorPos)
	})
}

// TestEditorVirtualSuggestion tests a multi-line suggested completion,
// whose lines the cursor skips, and which is accepted as one edit.
func TestEditorVirtualSuggestion(t *testing.T) {
	b := core.NewBody()
	ed := newWideEditor(b)
	ed.Lines.SetString("func f() {\n}\n")
	sug := lines.VirtualText{Pos: textpos.Pos{Line: 0, Char: 10}, Text: rich.NewText(ed.SuggestionStyle(), []rune("\n\ta := 1\n\treturn a"))}
	b.AssertRender(t, "virtual/suggestion", func() {
		ed.SetCursorShow(sug.Pos)
		ed.SetVirtualText("suggestion", sug)
	}, func() {
		assert.Equal(t, 4, ed.Lines.ViewLines(ed.viewId))
		ed.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeDownArrow, 0))
		assert.Equal(t, 1, ed.CursorPos.Line)
		ed.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeUpArrow, 0))
		assert.Equal(t, 0, ed.CursorPos.Line)
		ed.SetCursorShow(sug.Pos)
	})

	// accept
	ed.ClearVirtualText("suggestion")
	ed.Lines.InsertText(sug.Pos, sug.Text.Join())
	assert.Equal(t, "func f() {\n\ta := 1\n\treturn a\n}\n", ed.Lines.String())
}
