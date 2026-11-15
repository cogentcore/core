// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// editorTest makes an editor with the given text in a new window,
// and calls the given function with the editor, which has the focus,
// with the body locked.
func editorTest(t *testing.T, text string, fun func(ed *Editor)) {
	t.Helper()
	b := core.NewBody()
	ed := NewEditor(b)
	ed.Lines.SetString(text)
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Em(30)
		s.Min.Y.Em(10)
	})
	core.RenderImage(b, core.Locked(b, func() {
		ed.SetFocus()
		fun(ed)
	}))
}

// pos returns a text position.
func pos(ln, ch int) textpos.Pos {
	return textpos.Pos{Line: ln, Char: ch}
}

// TestEditorNavKeys tests moving the cursor with the navigation keys.
func TestEditorNavKeys(t *testing.T) {
	editorTest(t, "one two three\nfour five\nsix\n", func(ed *Editor) {
		key := func(kf keymap.Functions) { core.SendKeyFunction(ed, kf) }
		ed.SetCursorShow(pos(0, 0))
		key(keymap.MoveRight)
		assert.Equal(t, pos(0, 1), ed.CursorPos)
		key(keymap.WordRight)
		assert.Equal(t, pos(0, 3), ed.CursorPos)
		key(keymap.WordLeft)
		assert.Equal(t, pos(0, 0), ed.CursorPos)
		key(keymap.MoveDown)
		assert.Equal(t, 1, ed.CursorPos.Line)
		key(keymap.MoveUp)
		assert.Equal(t, 0, ed.CursorPos.Line)
		key(keymap.End)
		assert.Equal(t, pos(0, 13), ed.CursorPos)
		key(keymap.Home)
		assert.Equal(t, pos(0, 0), ed.CursorPos)
		key(keymap.MoveLeft)
		assert.Equal(t, pos(0, 0), ed.CursorPos, "stays at the start")
		key(keymap.DocEnd)
		assert.Equal(t, ed.Lines.EndPos(), ed.CursorPos)
		key(keymap.DocHome)
		assert.Equal(t, pos(0, 0), ed.CursorPos)
		key(keymap.PageDown)
		assert.Greater(t, ed.CursorPos.Line, 0)
		key(keymap.PageUp)
		assert.Equal(t, 0, ed.CursorPos.Line)
		key(keymap.Recenter)
		key(keymap.Recenter)
	})
}

// TestEditorSelectKeys tests selecting text with shift and the
// selection keys.
func TestEditorSelectKeys(t *testing.T) {
	editorTest(t, "hello world\nnext\n", func(ed *Editor) {
		ed.SetCursorShow(pos(0, 0))
		core.SendKey(ed, 0, key.CodeRightArrow, key.Shift)
		core.SendKey(ed, 0, key.CodeRightArrow, key.Shift)
		require.True(t, ed.HasSelection())
		assert.Equal(t, "he", string(ed.Selection().ToBytes()))

		core.SendKeyFunction(ed, keymap.CancelSelect)
		assert.False(t, ed.HasSelection())

		core.SendKeyFunction(ed, keymap.SelectAll)
		assert.Equal(t, "hello world\nnext", string(ed.Selection().ToBytes()))
		ed.SelectReset()

		ed.SetCursorShow(pos(0, 6))
		core.SendKeyFunction(ed, keymap.SelectMode)
		core.SendKeyFunction(ed, keymap.End)
		assert.Equal(t, "world", string(ed.Selection().ToBytes()))
		core.SendKeyFunction(ed, keymap.SelectMode)

		ed.SetCursorShow(pos(1, 2))
		ed.selectWord()
		assert.Equal(t, "next", string(ed.Selection().ToBytes()))
	})
}

// TestEditorEditKeys tests the editing keys.
func TestEditorEditKeys(t *testing.T) {
	editorTest(t, "one two three\n", func(ed *Editor) {
		key := func(kf keymap.Functions) { core.SendKeyFunction(ed, kf) }
		ed.SetCursorShow(pos(0, 13))
		key(keymap.BackspaceWord)
		assert.Equal(t, "one two\n", ed.Lines.String(), "deletes the word and the space before it")
		key(keymap.Backspace)
		assert.Equal(t, "one tw\n", ed.Lines.String())
		key(keymap.Home)
		key(keymap.DeleteWord)
		assert.Equal(t, " tw\n", ed.Lines.String())
		key(keymap.Delete)
		assert.Equal(t, "tw\n", ed.Lines.String())
		key(keymap.MoveRight)
		key(keymap.Transpose)
		assert.Equal(t, "wt\n", ed.Lines.String())
		ed.SetCursorShow(pos(0, 1))
		key(keymap.Kill)
		assert.Equal(t, "w\n", ed.Lines.String())

		key(keymap.Undo)
		assert.Equal(t, "one two three\n", ed.Lines.String(), "the quick edits are one undo group")
		key(keymap.Redo)
		assert.Equal(t, "w\n", ed.Lines.String())

		ed.Lines.SetString("wt\n")
		ed.SetCursorShow(pos(0, 2))
		core.TypeText(ed, " x")
		assert.Equal(t, "wt x\n", ed.Lines.String())
		key(keymap.Enter)
		core.TypeText(ed, "y")
		assert.Equal(t, "wt x\ny\n", ed.Lines.String())
		ed.SetCursorShow(pos(1, 0))
		key(keymap.FocusNext)
		assert.Equal(t, "y", string(ed.Lines.Line(1)), "auto indents to the level of the previous line")
		ed.SetCursorShow(pos(1, 1))
		key(keymap.FocusNext)
		assert.Equal(t, "y\t", string(ed.Lines.Line(1)), "inserts a tab")
		ed.Lines.IndentLine(1, 2)
		ed.SetCursorShow(pos(1, 3))
		key(keymap.FocusPrev)
		assert.Equal(t, "\ty\t", string(ed.Lines.Line(1)), "shift tab unindents")
	})
}

// TestEditorBrackets tests the automatic closing of brackets and
// typing over the inserted ones.
func TestEditorBrackets(t *testing.T) {
	editorTest(t, "x\n", func(ed *Editor) {
		ed.SetCursorShow(pos(0, 1))
		core.TypeText(ed, "(")
		assert.Equal(t, "x()\n", ed.Lines.String())
		core.TypeText(ed, "a)")
		assert.Equal(t, "x(a)\n", ed.Lines.String(), "types over the closing paren")
		core.TypeText(ed, "[")
		assert.Equal(t, "x(a)[]\n", ed.Lines.String())
	})
}

// TestEditorClipboard tests cutting, copying, and pasting with the
// clipboard, including rectangles and the paste history.
func TestEditorClipboard(t *testing.T) {
	editorTest(t, "abcd\nefgh\nijkl\n", func(ed *Editor) {
		clip := func() string {
			return ed.Clipboard().Read([]string{mimedata.TextPlain}).Text(mimedata.TextPlain)
		}
		ed.SelectRegion = textpos.Region{Start: pos(0, 1), End: pos(0, 3)}
		core.SendKeyFunction(ed, keymap.Copy)
		assert.Equal(t, "bc", clip())
		assert.False(t, ed.HasSelection(), "copy resets the selection")

		ed.SelectRegion = textpos.Region{Start: pos(1, 0), End: pos(1, 2)}
		core.SendKeyFunction(ed, keymap.Cut)
		assert.Equal(t, "ef", clip())
		assert.Equal(t, "abcd\ngh\nijkl\n", ed.Lines.String())

		ed.SetCursorShow(pos(2, 0))
		core.SendKeyFunction(ed, keymap.Paste)
		assert.Equal(t, "abcd\ngh\nefijkl\n", ed.Lines.String())

		ed.SelectRegion = textpos.Region{Start: pos(0, 1), End: pos(2, 3)}
		ed.CopyRect(true)
		ed.SetCursorShow(pos(0, 0))
		ed.SelectRegion = textpos.Region{Start: pos(0, 0), End: pos(1, 1)}
		ed.CutRect()
		assert.Equal(t, "bcd\nh\nefijkl\n", ed.Lines.String())
		ed.SetCursorShow(pos(0, 0))
		ed.PasteRect()
		assert.Equal(t, "abcd\ngh\nefijkl\n", ed.Lines.String())

		ed.SelectRegion = textpos.Region{Start: pos(0, 0), End: pos(0, 4)}
		ed.ReCaseSelection(0)
		assert.NotEqual(t, "", string(ed.Lines.Line(0)))
	})
}

// TestEditorISearch tests the interactive search.
func TestEditorISearch(t *testing.T) {
	editorTest(t, "foo bar\nbar foo\nfoo\n", func(ed *Editor) {
		ed.SetCursorShow(pos(0, 0))
		core.SendKeyFunction(ed, keymap.Search)
		assert.True(t, ed.ISearch.On)
		core.TypeText(ed, "foo")
		assert.Equal(t, "foo", ed.ISearch.Find)
		assert.Len(t, ed.ISearch.Matches, 3)
		first := ed.CursorPos
		core.SendKeyFunction(ed, keymap.Search)
		assert.NotEqual(t, first, ed.CursorPos, "moves to the next match")
		core.SendKeyFunction(ed, keymap.Backspace)
		assert.Equal(t, "fo", ed.ISearch.Find)
		core.SendKeyFunction(ed, keymap.Abort)
		assert.False(t, ed.ISearch.On)
	})
}

// TestEditorQReplace tests the query replace.
func TestEditorQReplace(t *testing.T) {
	editorTest(t, "foo bar\nbar foo\nfoo\n", func(ed *Editor) {
		ed.SetCursorShow(pos(0, 0))
		ed.QReplaceStart("foo", "baz", false)
		assert.True(t, ed.QReplace.On)
		assert.Len(t, ed.QReplace.Matches, 3)
		core.SendKey(ed, 'y', 0)
		assert.Equal(t, "baz bar\nbar foo\nfoo\n", ed.Lines.String())
		core.SendKey(ed, 'n', 0)
		core.SendKey(ed, '!', 0)
		assert.Equal(t, "baz bar\nbar foo\nbaz\n", ed.Lines.String())

		ed.QReplaceStart("bar", "x", false)
		ed.QReplaceReplaceAll(0)
		assert.Equal(t, "baz x\nx foo\nbaz\n", ed.Lines.String())
		ed.escPressed()
		assert.False(t, ed.QReplace.On)
	})
}

// TestEditorHistory tests moving through the cursor position history.
func TestEditorHistory(t *testing.T) {
	editorTest(t, "a\nb\nc\nd\n", func(ed *Editor) {
		ed.SetCursorShow(pos(1, 0))
		ed.savePosHistory(ed.CursorPos)
		ed.SetCursorShow(pos(3, 0))
		ed.savePosHistory(ed.CursorPos)
		ed.SetCursorShow(pos(2, 0))
		core.SendKeyFunction(ed, keymap.HistPrev)
		assert.Equal(t, 3, ed.CursorPos.Line)
		core.SendKeyFunction(ed, keymap.HistPrev)
		assert.Equal(t, 1, ed.CursorPos.Line)
		core.SendKeyFunction(ed, keymap.HistNext)
		assert.Equal(t, 3, ed.CursorPos.Line)
	})
}

// TestEditorScroll tests scrolling the editor to lines and the cursor.
func TestEditorScroll(t *testing.T) {
	text := ""
	for range 100 {
		text += "line\n"
	}
	editorTest(t, text, func(ed *Editor) {
		assert.False(t, ed.IsScrolledToEnd())
		ed.SetCursorShow(pos(90, 0))
		ed.scrollCursorToTop()
		assert.Greater(t, ed.ScrollPos(), float32(0))
		ed.scrollCursorToBottom()
		ed.scrollCursorToCenter()
		ed.SetScrollPos(0)
		assert.Equal(t, float32(0), ed.ScrollPos())
		assert.Greater(t, ed.VisibleSize().Y, 0)
		assert.Greater(t, ed.CharSize().X, float32(0))
		ed.scrollToRight(10)
		ed.SetCursorShow(ed.Lines.EndPos())
		ed.scrollCursorToBottom()
		assert.True(t, ed.IsScrolledToEnd())
	})
}

// TestEditorAutoIndent tests the auto indent of a language with it.
func TestEditorAutoIndent(t *testing.T) {
	editorTest(t, "func f() {\n}\n", func(ed *Editor) {
		ed.Lines.SetLanguage(fileinfo.Go)
		ed.SetCursorShow(pos(0, 10))
		core.SendKeyFunction(ed, keymap.Enter)
		core.TypeText(ed, "x")
		assert.Equal(t, "\tx", string(ed.Lines.Line(1)))
	})
}

// TestEditorCodeErrors tests setting code errors, shown as tooltips.
func TestEditorCodeErrors(t *testing.T) {
	editorTest(t, "x := 1\n", func(ed *Editor) {
		ed.SetCodeErrors(CodeError{Region: textpos.Region{Start: pos(0, 0), End: pos(0, 1)}, Msg: "x is unused"})
		require.Len(t, ed.CodeErrors(), 1)
		ce, ok := ed.codeErrorAt(pos(0, 0))
		assert.True(t, ok)
		assert.Equal(t, "x is unused", ce.Msg)
		_, ok = ed.codeErrorAt(pos(0, 4))
		assert.False(t, ok)
		ed.InsertAtCursor([]byte("// "))
		_, ok = ed.codeErrorAt(pos(0, 3))
		assert.True(t, ok, "the region follows the edit")
		ed.SetCodeErrors()
		assert.Empty(t, ed.CodeErrors())
	})
}

// TestEditorHighlights tests highlighting regions.
func TestEditorHighlights(t *testing.T) {
	editorTest(t, "some text\n", func(ed *Editor) {
		ed.HighlightRegion(textpos.Region{Start: pos(0, 0), End: pos(0, 4)})
		assert.Len(t, ed.Highlights, 1)
		ed.HighlightsReset()
		assert.Empty(t, ed.Highlights)
	})
}
