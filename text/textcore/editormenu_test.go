// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTextEditor returns a body with an editor of plain text.
func newTextEditor(text string) (*core.Body, *Editor) {
	b := core.NewBody()
	ed := NewEditor(b)
	ed.Lines.SetFilename("plain.txt")
	ed.Lines.SetString(text)
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Em(30)
		s.Min.Y.Em(10)
	})
	return b, ed
}

// TestEditorContextMenuActions tests the context menus of an editable editor
// (cut and paste) and of a read-only one of generated code (making it
// editable).
func TestEditorContextMenuActions(t *testing.T) {
	b, ed := newTextEditor("hello world\n")
	core.RenderImage(b, core.Locked(b, func() {
		ed.SetFocus()
		ed.SelectRegion = textpos.Region{Start: pos(0, 0), End: pos(0, 5)}
		m := core.NewScene()
		ed.contextMenu(m, image.Point{})
		core.ButtonByText(m, "Cut").Send(events.Click)
		assert.Equal(t, " world\n", ed.Lines.String())
		ed.SetCursorShow(pos(0, 6))
		m = core.NewScene()
		ed.contextMenu(m, image.Point{})
		core.ButtonByText(m, "Paste").Send(events.Click)
		assert.Equal(t, " worldhello\n", ed.Lines.String())
		assert.NotNil(t, core.ButtonByText(m, "Save"), "the save function")

		ed.SetReadOnly(true)
		ed.Lines.FileInfo().Generated = true
		m = core.NewScene()
		ed.contextMenu(m, image.Point{})
		assert.Nil(t, core.ButtonByText(m, "Cut"), "read-only")
		core.ButtonByText(m, "Set editable").Send(events.Click)
		assert.False(t, ed.IsReadOnly())
		assert.False(t, ed.Lines.FileInfo().Generated)
		ed.SetReadOnly(true)
		m = core.NewScene()
		ed.contextMenu(m, image.Point{})
		core.ButtonByText(m, "Clear").Send(events.Click)
		assert.Equal(t, "\n", ed.Lines.String())
	}))
}

// TestJumpToLinePrompt tests jumping to a line number entered in a dialog.
func TestJumpToLinePrompt(t *testing.T) {
	b, ed := newTextEditor("1\n2\n3\n4\n5\n")
	core.RenderImage(b, core.Locked(b, func() {
		ed.JumpToLinePrompt()
	}), func() {
		time.Sleep(50 * time.Millisecond)
	}, core.Locked(b, func() {
		d := core.TopScene(b)
		require.NotNil(t, d)
		require.Equal(t, "Jump to line", d.Body.Title)
		var tf *core.TextField
		for cw := range d.AllWidgets() {
			if f, ok := cw.(*core.TextField); ok {
				tf = f
			}
		}
		require.NotNil(t, tf)
		tf.SetText("4")
		core.ButtonByText(d, "Jump").Send(events.Click)
		assert.Equal(t, 3, ed.CursorPos.Line, "line 4")
	}))
}

// TestInsertBracket tests that typing an opening bracket inserts the
// closing one in plain text where it is followed by space or the end of
// the line, and a brace at the end of a line opens a block of lines.
func TestInsertBracket(t *testing.T) {
	b, ed := newTextEditor("a b\nend \n")
	core.RenderImage(b, core.Locked(b, func() {
		ed.SetFocus()
		ed.SetCursorShow(pos(0, 3))
		core.SendKey(ed, '(', key.Code9, key.Shift)
		assert.Equal(t, "a b()", string(ed.Lines.Line(0)), "at the end of the line")
		ed.SetCursorShow(pos(0, 0))
		core.SendKey(ed, '[', key.CodeLeftSquareBracket)
		assert.Equal(t, "[a b()", string(ed.Lines.Line(0)), "not before a word")
		ed.SetCursorShow(pos(1, 4))
		core.SendKey(ed, '{', key.CodeLeftSquareBracket, key.Shift)
		assert.Equal(t, "end {", string(ed.Lines.Line(1)))
		assert.Equal(t, "}", string(ed.Lines.Line(3)), "a block, after a space")
		ed.SetCursorShow(pos(0, 1))
		core.SendKey(ed, '{', key.CodeLeftSquareBracket, key.Shift)
		assert.Equal(t, "[{a b()", string(ed.Lines.Line(0)), "not before a word")
	}))
}
