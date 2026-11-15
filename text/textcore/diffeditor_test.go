// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiffEditorNavigateApply tests moving between the diffs of a diff
// editor and applying them in both directions, with undo.
func TestDiffEditorNavigateApply(t *testing.T) {
	b := core.NewBody()
	de := NewDiffEditor(b)
	astr := []string{"same", "a one", "same 2", "a two", "end"}
	bstr := []string{"same", "b one", "same 2", "b two", "end"}
	de.DiffStrings(astr, bstr)
	core.RenderImage(b, core.Locked(b, func() {
		tva, tvb := de.textEditors()
		tva.SetCursorShow(textpos.Pos{})
		assert.True(t, de.nextDiff(0))
		assert.Equal(t, 1, tva.CursorPos.Line)
		assert.Equal(t, 1, tvb.CursorPos.Line, "both editors move")
		assert.True(t, de.nextDiff(0))
		assert.Equal(t, 3, tva.CursorPos.Line)
		assert.False(t, de.nextDiff(0), "no more")
		assert.True(t, de.prevDiff(1))
		assert.Equal(t, 1, tvb.CursorPos.Line)
		assert.False(t, de.prevDiff(1), "no more")

		assert.False(t, de.applyDiff(0, 0), "an equal line")
		assert.True(t, de.applyDiff(0, 1))
		assert.Equal(t, "b one", string(de.linesA.Line(1)))
		assert.True(t, de.applyDiff(1, 3))
		assert.Equal(t, "a two", string(de.linesB.Line(3)))
		assert.NoError(t, de.undoDiff(0))
		assert.Equal(t, "a one", string(de.linesA.Line(1)))
		assert.Error(t, de.undoDiff(0), "nothing more to undo")

		tva.SetCursorShow(textpos.Pos{Line: 2})
		tva.SendInput()
		assert.Equal(t, 2, tvb.CursorPos.Line, "the cursors are synced")
	}))
}

// TestDiffFiles tests opening a diff editor dialog of two files, and
// saving its edits.
func TestDiffFiles(t *testing.T) {
	dir := t.TempDir()
	fa, fb := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	require.NoError(t, os.WriteFile(fa, []byte("one\ntwo\n"), 0644))
	require.NoError(t, os.WriteFile(fb, []byte("one\nthree\n"), 0644))
	b := core.NewBody()
	var de *DiffEditor
	core.RenderImage(b, core.Locked(b, func() {
		var err error
		de, err = DiffFiles(b, fa, fb)
		require.NoError(t, err)
		_, err = DiffFiles(b, fa, filepath.Join(dir, "none"))
		assert.Error(t, err)
	}), func() {
		de.AsyncLock() // (it is in the window of the dialog)
		defer de.AsyncUnlock()
		assert.Contains(t, de.linesB.String(), "three")
		fc := filepath.Join(dir, "c.txt")
		de.saveAs(true, core.Filename(fc))
		got, err := os.ReadFile(fc)
		require.NoError(t, err)
		assert.Equal(t, "one\nthree\n", string(got), "true saves B")
	})
}

// TestTwinEditors tests the twin editors, whose cursors are synced.
func TestTwinEditors(t *testing.T) {
	dir := t.TempDir()
	fa := filepath.Join(dir, "a.go")
	require.NoError(t, os.WriteFile(fa, []byte("package a\n"), 0644))
	b := core.NewBody()
	te := NewTwinEditors(b)
	te.BufferA.SetString("one\ntwo\nthree\n")
	te.BufferB.SetString("1\n2\n3\n")
	te.SetFiles(fa, fa)
	core.RenderImage(b, core.Locked(b, func() {
		ea, eb := te.Editors()
		ea.SetCursorShow(textpos.Pos{Line: 2})
		ea.SendInput()
		assert.Equal(t, 2, eb.CursorPos.Line)
	}))
}

// TestEditorLinks tests moving between, and opening, the links of an
// editor.
func TestEditorLinks(t *testing.T) {
	b := core.NewBody()
	ed := NewEditor(b)
	sty := rich.NewStyle()
	mk := func(parts ...string) ([]rune, rich.Text) {
		var tx rich.Text
		for i, p := range parts {
			if i%2 == 0 {
				tx.AddSpanString(sty, p)
			} else {
				tx.AddLink(sty, "https://"+p, p)
			}
		}
		return tx.Join(), tx
	}
	l0, m0 := mk("plain")
	l1, m1 := mk("", "a.org", " see")
	l2, m2 := mk("none")
	l3, m3 := mk("", "c.org")
	ed.Lines.AppendTextMarkup([][]rune{l0, l1, l2, l3}, []rich.Text{m0, m1, m2, m3})
	var opened []string
	ed.LinkHandler = func(tl *rich.Hyperlink) { opened = append(opened, tl.URL) }
	core.RenderImage(b, core.Locked(b, func() {
		ed.SetCursorShow(textpos.Pos{Line: 2})
		assert.True(t, ed.CursorNextLink(false))
		assert.Equal(t, 3, ed.CursorPos.Line)
		assert.False(t, ed.CursorNextLink(false))
		assert.True(t, ed.CursorNextLink(true), "wraps around")
		assert.Equal(t, 1, ed.CursorPos.Line)
		assert.True(t, ed.CursorPrevLink(true), "wraps around")
		assert.Equal(t, 3, ed.CursorPos.Line)

		lk, ln := ed.OpenLinkAt(textpos.Pos{Line: 1, Char: 2})
		require.NotNil(t, lk)
		assert.Equal(t, 1, ln)
		assert.Equal(t, []string{"https://a.org"}, opened)
		lk, _ = ed.OpenLinkAt(textpos.Pos{Line: 2, Char: 1})
		assert.Nil(t, lk)
	}))
}

// TestEditorSaveClose tests saving the text of an editor to a file, and
// closing it.
func TestEditorSaveClose(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "a.txt")
	b := core.NewBody()
	ed := NewEditor(b)
	ed.Lines.SetString("hello\n")
	core.RenderImage(b, core.Locked(b, func() {
		assert.Error(t, ed.Save(), "no filename")
		ed.SaveAs(core.Filename(fn))
		got, err := os.ReadFile(fn)
		require.NoError(t, err)
		assert.Equal(t, "hello\n", string(got))
		assert.False(t, ed.IsNotSaved())

		ed.InsertAtCursor([]byte("x"))
		assert.True(t, ed.IsNotSaved())
		ed.SaveQuiet()
		assert.False(t, ed.IsNotSaved())
		got, _ = os.ReadFile(fn)
		assert.Equal(t, "xhello\n", string(got))

		closed := false
		assert.True(t, ed.Close(func(canceled bool) { closed = !canceled }))
		assert.True(t, closed)

		ln := lines.NewLines()
		ln.SetString("y\n")
		ed.SetLines(ln)
		assert.Equal(t, "y\n", ed.Lines.String())
	}))
}
