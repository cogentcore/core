// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"image"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mouseAt returns the point of a mouse event, as the editor gets it, in
// the first half of the char at the given text position.
func mouseAt(ed *Editor, p textpos.Pos) image.Point {
	return clickPoint(&ed.Base, ed.CharStartPos(p)).Add(ed.Geom.ContentBBox.Min)
}

// sendMouse sends a mouse event of the given type at the given text position.
func sendMouse(ed *Editor, typ events.Types, but events.Buttons, p textpos.Pos, mods ...key.Modifiers) {
	var m key.Modifiers
	for _, md := range mods {
		m.SetFlag(true, md)
	}
	ev := events.NewMouse(typ, but, mouseAt(ed, p), m)
	ev.SetLocalOff(image.Point{}) // the scene is at the origin of the window
	ed.HandleEvent(ev)
}

// mouseMove sends a mouse move event at the given text position.
func mouseMove(ed *Editor, p textpos.Pos) {
	ev := events.NewMouseMove(events.NoButton, mouseAt(ed, p), image.Point{}, 0)
	ev.SetLocalOff(image.Point{})
	ed.HandleEvent(ev)
}

// TestEditorMouseClicks tests that clicks move the cursor, shift clicks
// extend the selection, double clicks select a word, triple clicks select
// a line, and middle clicks paste.
func TestEditorMouseClicks(t *testing.T) {
	editorTest(t, "one two three\nfour five\n", func(ed *Editor) {
		sendMouse(ed, events.Click, events.Left, pos(0, 4))
		assert.Equal(t, pos(0, 4), ed.CursorPos)
		assert.False(t, ed.HasSelection())

		sendMouse(ed, events.Click, events.Left, pos(1, 2), key.Shift)
		assert.Equal(t, pos(1, 2), ed.CursorPos)
		assert.True(t, ed.HasSelection(), "shift extends the selection")
		assert.Equal(t, pos(0, 4), ed.SelectRegion.Start)

		sendMouse(ed, events.Click, events.Left, pos(0, 5))
		assert.False(t, ed.HasSelection(), "a plain click resets the selection")
		sendMouse(ed, events.DoubleClick, events.Left, pos(0, 5))
		assert.Equal(t, "two", string(ed.Selection().ToBytes()))
		assert.Equal(t, pos(0, 4), ed.CursorPos, "at the start of the word")

		ed.SetCursorShow(pos(1, 1))
		sendMouse(ed, events.TripleClick, events.Left, pos(1, 1))
		assert.Equal(t, "four five", string(ed.Selection().ToBytes()))

		ed.Clipboard().Write(nil)
		ed.SelectReset()
		ed.SetCursorShow(pos(0, 0))
		ed.SelectRegion = textpos.Region{Start: pos(0, 0), End: pos(0, 3)}
		ed.Copy(true)
		sendMouse(ed, events.Click, events.Middle, pos(1, 0))
		assert.Equal(t, "oneone two three", string(ed.Lines.Line(0)), "a middle click pastes at the cursor")
	})
}

// TestEditorMouseDrag tests selecting text by sliding (dragging) the mouse.
func TestEditorMouseDrag(t *testing.T) {
	editorTest(t, "alpha beta gamma\n", func(ed *Editor) {
		sendMouse(ed, events.SlideStart, events.Left, pos(0, 6))
		assert.True(t, ed.isScrolling)
		sendMouse(ed, events.SlideMove, events.Left, pos(0, 10))
		sendMouse(ed, events.SlideStop, events.Left, pos(0, 10))
		assert.False(t, ed.isScrolling)
		assert.Equal(t, pos(0, 10), ed.CursorPos)
		require.True(t, ed.HasSelection())
		assert.Contains(t, string(ed.Selection().ToBytes()), "beta")

		ed.selectModeToggle()
		ed.SelectReset()
		ed.SetCursorShow(pos(0, 0))
		sendMouse(ed, events.SlideStart, events.Left, pos(0, 11), key.Shift)
		assert.True(t, ed.HasSelection(), "shift extends from the cursor")
	})
}

// TestEditorLinkCursor tests that the mouse cursor is a pointer over a
// link, and that clicking on a link opens it instead of moving the cursor.
func TestEditorLinkCursor(t *testing.T) {
	b := core.NewBody()
	ed := NewEditor(b)
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Em(30)
	})
	sty := rich.NewStyle()
	tx := rich.Text{}
	tx.AddSpanString(sty, "see ")
	tx.AddLink(sty, "https://example.com", "link")
	tx.AddSpanString(sty, " here")
	ed.Lines.AppendTextMarkup([][]rune{tx.Join()}, []rich.Text{tx})
	opened := ""
	ed.LinkHandler = func(tl *rich.Hyperlink) { opened = tl.URL }
	core.RenderImage(b, core.Locked(b, func() {
		ed.SetFocus()
		mouseMove(ed, pos(0, 5))
		assert.Equal(t, cursors.Pointer, ed.Styles.Cursor)
		mouseMove(ed, pos(0, 10))
		assert.Equal(t, cursors.Text, ed.Styles.Cursor)
		sendMouse(ed, events.Click, events.Left, pos(0, 5))
		assert.Equal(t, "https://example.com", opened)
	}))
}

// TestEditorContextMenu tests the items of the context menu, for an
// editable and a read-only editor, and its position.
func TestEditorContextMenu(t *testing.T) {
	editorTest(t, "copy me\n", func(ed *Editor) {
		texts := func() []string {
			m := core.NewScene()
			ed.contextMenu(m, image.Point{})
			var ts []string
			for _, c := range m.Children {
				switch w := c.(type) {
				case *core.Button:
					ts = append(ts, w.Text)
				case *core.FuncButton:
					ts = append(ts, w.Text)
				}
			}
			return ts
		}
		assert.Contains(t, texts(), "Cut")
		assert.Contains(t, texts(), "Paste")
		ed.SetReadOnly(true)
		assert.Contains(t, texts(), "Clear")
		assert.NotContains(t, texts(), "Cut")
		ed.Lines.FileInfo().Generated = true
		assert.Contains(t, texts(), "Set editable")
		ed.SetReadOnly(false)

		ed.SetCursorShow(pos(0, 2))
		cp := ed.ContextMenuPos(nil)
		assert.Greater(t, cp.Y, int(ed.CharStartPos(pos(0, 2)).Y), "below the cursor")
		ev := events.NewMouse(events.ContextMenu, events.Right, image.Pt(7, 9), 0)
		ev.SetLocalOff(image.Point{})
		assert.Equal(t, image.Pt(7, 9), ed.ContextMenuPos(ev))
		ed.ShowContextMenu(ev)
	})
}

// TestEditorJumpTranspose tests jumping to a line, the jump to line
// prompt, and transposing words (which does nothing for one word).
func TestEditorJumpTranspose(t *testing.T) {
	editorTest(t, "a\nb\nc\nd\n", func(ed *Editor) {
		ed.jumpToLine(3)
		assert.Equal(t, pos(2, 0), ed.CursorPos)
		ed.JumpToLinePrompt()
		core.SendKeyFunction(ed, keymap.TransposeWord)
		assert.Equal(t, "a\nb\nc\nd\n", ed.Lines.String())

		assert.Equal(t, []byte("a\nb\nc\nd\n"), ed.WidgetValue())
		assert.NoError(t, ed.SetWidgetValue("x\ny"))
		assert.Equal(t, "x\ny\n", ed.Lines.String())
		assert.True(t, ed.posIsVisible(pos(0, 0)))
	})
}

// TestEditorTransposeWord tests transposing words with the key function.
func TestEditorTransposeWord(t *testing.T) {
	editorTest(t, "one two three\n", func(ed *Editor) {
		ed.SetCursorShow(pos(0, 5))
		core.SendKeyFunction(ed, keymap.TransposeWord)
		assert.Equal(t, "one three two\n", ed.Lines.String())
		assert.Equal(t, pos(0, 13), ed.CursorPos)
		core.SendKeyFunction(ed, keymap.Undo)
		assert.Equal(t, "one two three\n", ed.Lines.String())
	})
}

// TestEditorPasteHistory tests the history of copied text, and the menu
// of it for pasting.
func TestEditorPasteHistory(t *testing.T) {
	prev := editorClipboardHistory
	defer func() { editorClipboardHistory = prev }()
	editorClipboardHistory = nil
	editorTest(t, "one two three\n", func(ed *Editor) {
		core.SendKeyFunction(ed, keymap.PasteHist) // nothing copied yet
		long := strings.Repeat("x", editorClipHistoryChooserLength+5)
		addBaseClipboardHistory([]byte(long))
		ed.SelectRegion = textpos.Region{Start: pos(0, 4), End: pos(0, 7)}
		ed.Copy(false)
		cl := editorClipHistoryChooserList()
		require.Len(t, cl, 2)
		assert.Equal(t, "two", cl[0], "the latest first")
		assert.Len(t, cl[1], editorClipHistoryChooserLength, "shortened")
		ed.SelectReset()
		ed.SetCursorShow(pos(0, 0))
		core.SendKeyFunction(ed, keymap.PasteHist)
		b := ed.Scene.Body
		pops := core.Popups(b)
		require.NotEmpty(t, pops, "the menu")
		bt := core.ButtonByText(pops[len(pops)-1], "two")
		require.NotNil(t, bt)
		bt.Send(events.Click)
		assert.Equal(t, "twoone two three\n", ed.Lines.String(), "pasted at the cursor")
		assert.Equal(t, pos(0, 3), ed.CursorPos)
	})
}
