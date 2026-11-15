// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLinesFile tests opening, saving, reverting, and checking files
// for changes on disk.
func TestLinesFile(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "a.go")
	require.NoError(t, os.WriteFile(fn, []byte("package a\n"), 0644))

	lns := NewLines()
	require.NoError(t, lns.Open(fn))
	assert.Equal(t, fn, lns.Filename())
	assert.Equal(t, "package a\n", lns.String())
	assert.Equal(t, fileinfo.Go, lns.FileInfo().Known)
	assert.True(t, lns.ConfigKnown())
	assert.NotNil(t, lns.ParseFileState())
	lp, fss := lns.ParseState()
	assert.NotNil(t, lp)
	assert.NotNil(t, fss)
	assert.NoError(t, lns.Stat())
	assert.False(t, lns.ChangedOnDisk())
	assert.False(t, lns.FileModCheck())

	lns.InsertText(textpos.Pos{Line: 0, Char: 9}, []rune("\nvar x = 1"))
	assert.True(t, lns.IsNotSaved())
	assert.Equal(t, filepath.Join(dir, "#a.go#"), lns.AutosaveFilename())
	require.NoError(t, os.WriteFile(lns.AutosaveFilename(), []byte("x"), 0644))
	assert.True(t, lns.AutosaveCheck())
	lns.AutosaveDelete()
	assert.False(t, lns.AutosaveCheck())

	fn2 := filepath.Join(dir, "b.go")
	require.NoError(t, lns.SaveFile(fn2))
	assert.Equal(t, fn2, lns.Filename())
	assert.False(t, lns.IsNotSaved())
	got, _ := os.ReadFile(fn2)
	assert.Equal(t, "package a\nvar x = 1\n", string(got))

	// a change on disk of an unedited file reverts it
	later := time.Now().Add(time.Hour)
	require.NoError(t, os.WriteFile(fn2, []byte("package b\nvar x = 1\n"), 0644))
	require.NoError(t, os.Chtimes(fn2, later, later))
	assert.True(t, lns.ChangedOnDisk())
	assert.True(t, lns.FileModCheck())
	assert.Equal(t, "package b\nvar x = 1\n", lns.String())

	// an edited file prompts
	prompted := false
	lns.FileModPromptFunc = func() { prompted = true }
	lns.InsertText(textpos.Pos{}, []rune("// x\n"))
	later = later.Add(time.Hour)
	require.NoError(t, os.Chtimes(fn2, later, later))
	assert.True(t, lns.FileModCheck())
	assert.True(t, prompted)
	lns.SetFileModOK(true)
	assert.False(t, lns.FileModCheck(), "ok to edit")

	assert.True(t, lns.Revert())
	assert.Equal(t, "package b\nvar x = 1\n", lns.String())
	assert.False(t, NewLines().Revert(), "no file")

	lns.ClearNotSaved()
	lns.EditDone()
	assert.False(t, lns.IsChanged())
	lns.SetReadOnly(true)
	assert.True(t, lns.readOnly)
}

// TestLinesOpenFS tests opening a file from a file system.
func TestLinesOpenFS(t *testing.T) {
	fsys := fstest.MapFS{"dir/a.md": {Data: []byte("# Title\n\ntext\n")}}
	lns := NewLines()
	require.NoError(t, lns.OpenFS(fsys, "dir/a.md"))
	assert.Equal(t, "# Title\n\ntext\n", lns.String())
	assert.Equal(t, fileinfo.Markdown, lns.FileInfo().Known)
	assert.Error(t, lns.OpenFS(fsys, "none.md"))
}

// TestLinesLanguage tests setting the language of the lines directly.
func TestLinesLanguage(t *testing.T) {
	lns := NewLines()
	lns.SetString("x = 1\n")
	lns.SetFileExt(".py")
	assert.True(t, lns.Highlighter.Has)
	assert.Equal(t, "#", lns.Settings.CommentLine[:1])
	lns.SetLanguage(fileinfo.Go)
	assert.Equal(t, "//", lns.Settings.CommentLine[:2])
	lns.SetHighlighting("emacs")
	assert.Equal(t, "emacs", string(lns.Highlighter.StyleName))
}

// TestLinesLinks tests the hyperlinks of text appended with markup.
func TestLinesLinks(t *testing.T) {
	lns := NewLines()
	lns.Defaults()
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
	l0, m0 := mk("see ", "a.org", " and ", "b.org")
	l1, m1 := mk("none")
	l2, m2 := mk("", "c.org")
	lns.AppendTextMarkup([][]rune{l0, l1, l2}, []rich.Text{m0, m1, m2})
	assert.Len(t, lns.Links(), 2, "lines with links")

	lk := lns.LinkAt(textpos.Pos{Line: 0, Char: 5})
	require.NotNil(t, lk)
	assert.Equal(t, "https://a.org", lk.URL)
	assert.Nil(t, lns.LinkAt(textpos.Pos{Line: 1, Char: 1}))

	lk, ln := lns.NextLink(textpos.Pos{Line: 1})
	require.NotNil(t, lk)
	assert.Equal(t, 2, ln)
	assert.Equal(t, "https://c.org", lk.URL)
	lk, ln = lns.PrevLink(textpos.Pos{Line: 2, Char: 1})
	require.NotNil(t, lk)
	assert.Equal(t, 0, ln)
	lk, ln = lns.NextLink(textpos.Pos{Line: 2, Char: 2})
	assert.Nil(t, lk)
	assert.Equal(t, -1, ln)

	lns.AppendTextMarkupAsync([][]rune{l1}, []rich.Text{m1})
	assert.Equal(t, 4, lns.NumLines()-1)
}

// TestLinesNextLinkSameLine tests moving to the next link on the same
// line.
func TestLinesNextLinkSameLine(t *testing.T) {
	lns := NewLines()
	lns.Defaults()
	sty := rich.NewStyle()
	var tx rich.Text
	tx.AddLink(sty, "https://a.org", "a.org")
	tx.AddSpanString(sty, " and ")
	tx.AddLink(sty, "https://b.org", "b.org")
	lns.AppendTextMarkup([][]rune{tx.Join()}, []rich.Text{tx})
	lk, ln := lns.NextLink(textpos.Pos{Line: 0, Char: 1})
	require.NotNil(t, lk)
	assert.Equal(t, 0, ln)
	assert.Equal(t, "https://b.org", lk.URL)

	// a link at the position is the current one, which is skipped
	// (editors check for a link at the start on wraparound).
	lk, _ = lns.NextLink(textpos.Pos{})
	require.NotNil(t, lk)
	assert.Equal(t, "https://b.org", lk.URL)
	lk, _ = lns.PrevLink(textpos.Pos{Line: 0, Char: 12}) // in b.org
	require.NotNil(t, lk)
	assert.Equal(t, "https://a.org", lk.URL, "the closest previous link")
}

// TestDiffSelected tests applying selected diffs from A to B and back,
// with undo.
func TestDiffSelected(t *testing.T) {
	astr := []string{"a one", "a two", "a three"}
	bstr := []string{"a one", "b two", "a three", "b four"}
	ds := NewDiffSelected(astr, bstr)
	require.NotEmpty(t, ds.Diffs)
	for i := range ds.Diffs {
		if ds.Diffs[i].Tag != 'e' {
			ds.AtoB(i)
		}
	}
	assert.Equal(t, astr, ds.B.Edit)
	for ds.B.Undo() {
	}
	assert.Equal(t, bstr, ds.B.Edit)
	assert.False(t, ds.B.Undo())

	for i := range ds.Diffs {
		if ds.Diffs[i].Tag != 'e' {
			ds.BtoA(i)
		}
	}
	assert.Equal(t, bstr, ds.A.Edit)

	rv := ds.Diffs.Reverse()
	assert.Len(t, rv, len(ds.Diffs))
	pt := ds.Diffs.ToPatch(bstr)
	assert.Equal(t, 2, pt.NumBlines(), "b two, b four")
	assert.Equal(t, bstr, pt.Apply(astr))
}

// TestLinesSetLanguageFileInfo tests that setting the language sets the
// file info, which editors use to set up completion.
func TestLinesSetLanguageFileInfo(t *testing.T) {
	lns := NewLines()
	lns.SetLanguage(fileinfo.Go)
	assert.Equal(t, fileinfo.Go, lns.FileInfo().Known)
}

// TestLinesPatchLastLine tests patching lines with a replacement of
// fewer lines that runs through the last line, as in [Lines.Revert].
func TestLinesPatchLastLine(t *testing.T) {
	a := NewLines()
	a.Defaults()
	a.SetString("x\ny")
	b := NewLines()
	b.Defaults()
	b.SetString("z")
	a.PatchFrom(b, a.Diffs(b))
	assert.Equal(t, b.String(), a.String())

	a.SetString("a\nx")
	b.SetString("b\ny")
	a.PatchFrom(b, a.Diffs(b))
	assert.Equal(t, b.String(), a.String(), "the same number of lines")
}

// TestLinesPatchRandom tests that patching lines with the diffs to other
// lines makes them the same, for random edits of random lines.
func TestLinesPatchRandom(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	words := []string{"a", "b", "c", "", "d e"}
	randLines := func() string {
		ls := make([]string, rnd.Intn(6))
		for i := range ls {
			ls[i] = words[rnd.Intn(len(words))]
		}
		return strings.Join(ls, "\n")
	}
	for range 500 {
		as, bs := randLines(), randLines()
		a := NewLines()
		a.Defaults()
		a.SetString(as)
		b := NewLines()
		b.Defaults()
		b.SetString(bs)
		a.PatchFrom(b, a.Diffs(b))
		if !assert.Equal(t, b.String(), a.String(), "patching %q to %q", as, bs) {
			return
		}
	}
}
