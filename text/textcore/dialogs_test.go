// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTextDialog tests the dialog that shows text in an editor.
func TestTextDialog(t *testing.T) {
	b := core.NewBody()
	core.RenderImage(b, core.Locked(b, func() {
		ed := TextDialog(b, "Text", "some text\n")
		assert.Equal(t, "some text\n", ed.Lines.String())
	}))
}

// TestDiffEditorSaveApply tests saving the A and B sides of a diff
// editor, which asks first for an existing file, and applying a diff by
// double clicking on its line number.
func TestDiffEditorSaveApply(t *testing.T) {
	dir := t.TempDir()
	b := core.NewBody()
	var de *DiffEditor
	core.RenderImage(b, core.Locked(b, func() {
		de = DiffEditorDialog(b, "diff", []string{"one", "two"}, []string{"one", "three"}, "a.txt", "b.txt", "", "")
	}), func() {
		de.AsyncLock() // (it is in the window of the dialog)
		defer de.AsyncUnlock()
		fa := filepath.Join(dir, "a.txt")
		de.saveFileA(core.Filename(fa))
		got, err := os.ReadFile(fa)
		require.NoError(t, err)
		assert.Equal(t, "one\ntwo", string(got))
		fb := filepath.Join(dir, "b.txt")
		de.saveFileB(core.Filename(fb))
		got, err = os.ReadFile(fb)
		require.NoError(t, err)
		assert.Equal(t, "one\nthree", string(got))
		de.saveFileB(core.Filename(fb)) // exists: a dialog asks first
		assert.Error(t, de.saveFile(true, core.Filename(filepath.Join(dir, "none", "x.txt"))))

		ta, tb := de.textEditors()
		assert.Same(t, de, ta.diffEditor())
		// a double click on the line number of the second line applies its diff
		p := ta.Geom.ContentBBox.Min.Add(image.Pt(2, int(ta.charSize.Y*1.5)))
		ev := events.NewMouse(events.DoubleClick, events.Left, p, 0)
		ev.SetLocalOff(image.Point{})
		ta.HandleEvent(ev)
		_ = tb
	})
}

// TestCodeErrorsRender tests rendering the backgrounds of code errors,
// including ones over wrapped lines, and their tooltips.
func TestCodeErrorsRender(t *testing.T) {
	b := core.NewBody()
	ed := NewEditor(b)
	ed.Lines.SetString("first line with an error\nsecond\n")
	ed.ErrorColor = colors.Uniform(colors.Red)
	core.RenderImage(b, core.Locked(b, func() {
		ed.SetCodeErrors(CodeError{Region: textpos.Region{Start: pos(0, 6), End: pos(1, 3)}, Msg: "bad"})
		ed.NeedsRender()
	}), core.Locked(b, func() {
		tip, _ := ed.WidgetTooltip(image.Pt(-1, -1))
		assert.Equal(t, "bad", tip, "the first error without a position")
		at := mouseAt(ed, pos(0, 8))
		tip, _ = ed.WidgetTooltip(at)
		assert.Equal(t, "bad", tip)
		tip, _ = ed.WidgetTooltip(mouseAt(ed, pos(1, 5)))
		assert.NotEqual(t, "bad", tip, "no error there")
		ed.renderCodeErrors()
		ed.SetCodeErrors()
		tip, _ = ed.WidgetTooltip(at)
		assert.NotEqual(t, "bad", tip)
	}))
}

// TestQReplacePromptISearchBack tests the query replace prompt, and
// backspace in interactive search.
func TestQReplacePromptISearchBack(t *testing.T) {
	editorTest(t, "alpha beta alpha\n", func(ed *Editor) {
		ed.SelectRegion = textpos.Region{Start: pos(0, 0), End: pos(0, 5)}
		ed.QReplacePrompt()

		ed.SelectReset()
		ed.iSearchStart()
		core.TypeText(ed, "alp")
		assert.Equal(t, "alp", ed.ISearch.Find)
		ed.iSearchBackspace()
		assert.Equal(t, "al", ed.ISearch.Find)
		ed.iSearchBackspace()
		ed.iSearchBackspace()
		assert.Equal(t, "", ed.ISearch.Find)
		core.TypeText(ed, "zz")
		ed.iSearchBackspace()
		assert.Equal(t, "z", ed.ISearch.Find)
		core.SendKeyFunction(ed, keymap.Abort)
		assert.False(t, ed.ISearch.On)

		// a backspace right after starting again goes back to no search
		ed.iSearchStart()
		ed.ISearch.Find = PrevISearchString
		ed.iSearchBackspace()
		assert.Equal(t, "", ed.ISearch.Find)
	})
}

// TestScrollChanged tests scrolling the editor with its scroll bars.
func TestScrollChanged(t *testing.T) {
	text := ""
	for range 60 {
		text += "line\n"
	}
	editorTest(t, text, func(ed *Editor) {
		sb := core.NewSlider()
		sb.Value = 10 * ed.charSize.Y
		ed.ScrollChanged(math32.Y, sb)
		assert.InDelta(t, 10, ed.scrollPos, 0.01)
		assert.True(t, ed.isScrolling)
		sb.Value = 0
		ed.ScrollChanged(math32.X, sb)
	})
}
