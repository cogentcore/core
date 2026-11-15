// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLinesViewRegions tests the regions of view lines, and converting
// regions between view and source coordinates.
func TestLinesViewRegions(t *testing.T) {
	lns, vid := newTestLines(t, "abc\n\tdéf\nlast")
	reg := lns.ViewLineRegion(vid, 1)
	assert.Equal(t, textpos.Pos{Line: 1}, reg.Start)
	assert.Equal(t, textpos.Pos{Line: 1, Char: lns.ViewLineLen(vid, 1)}, reg.End)
	assert.Equal(t, reg, lns.ViewLineRegionNoLock(vid, 1))
	assert.Equal(t, textpos.Region{}, lns.ViewLineRegion(vid+100, 1), "no such view")
	assert.Equal(t, textpos.Region{}, lns.ViewLineRegionNoLock(vid+100, 1))
	src := textpos.Region{Start: textpos.Pos{Line: 0, Char: 1}, End: textpos.Pos{Line: 2, Char: 2}}
	assert.Equal(t, src, lns.RegionFromView(vid, lns.RegionToView(vid, src)))
	assert.Equal(t, textpos.Region{}, lns.RegionFromView(vid+100, src))
	assert.Equal(t, 4, lns.ViewCell(vid, textpos.Pos{Line: 1, Char: 1}), "after a tab")
	assert.Equal(t, 0, lns.ViewCell(vid+100, textpos.Pos{Line: 1, Char: 1}))
	assert.NotNil(t, lns.Metadata())
}

// TestLinesAdjustRegion tests adjusting a region for later edits.
func TestLinesAdjustRegion(t *testing.T) {
	lns, _ := newTestLines(t, "one two three\nfour\n")
	reg := textpos.Region{Start: textpos.Pos{Char: 8}, End: textpos.Pos{Char: 13}}
	reg.TimeNow()
	time.Sleep(time.Millisecond)
	lns.InsertText(textpos.Pos{Char: 0}, []rune("zero "))
	adj := lns.AdjustRegion(reg)
	assert.Equal(t, textpos.Pos{Char: 13}, adj.Start)
	assert.Equal(t, textpos.Pos{Char: 18}, adj.End)
	assert.Equal(t, "three", string(lns.Region(adj.Start, adj.End).ToBytes()))
}

// TestLinesClose tests closing lines, which sends a close event to
// its views.
func TestLinesClose(t *testing.T) {
	lns, vid := newTestLines(t, "abc\n")
	closed := false
	lns.OnClose(vid, func(e events.Event) { closed = true })
	lns.Close()
	assert.True(t, closed)
	assert.Equal(t, 0, lns.NumLines())
}

// TestFileRegionBytes tests getting the lines of a file, with the
// comments before them.
func TestFileRegionBytes(t *testing.T) {
	fn := filepath.Join(t.TempDir(), "a.go")
	src := "package a\n\n// F does\n// things.\nfunc F() {\n}\n\nvar x = 1\n"
	require.NoError(t, os.WriteFile(fn, []byte(src), 0o644))
	b, err := FileBytes(fn)
	require.NoError(t, err)
	assert.Equal(t, src, string(b))
	_, err = FileBytes(filepath.Join(t.TempDir(), "missing.go"))
	assert.Error(t, err)

	assert.Equal(t, src, string(FileRegionBytes(fn, 0, 0, false, 0)))
	assert.Equal(t, "func F() {\n}\n", string(FileRegionBytes(fn, 4, 5, false, 0)))
	assert.Equal(t, "// F does\n// things.\nfunc F() {\n}\n", string(FileRegionBytes(fn, 4, 5, true, 10)))
	assert.Equal(t, "// things.\nfunc F() {\n}\n", string(FileRegionBytes(fn, 4, 5, true, 0)), "only one line back")
	assert.Equal(t, "var x = 1\n\n", string(FileRegionBytes(fn, 7, 0, true, 10)), "to the end")
	assert.Nil(t, FileRegionBytes(filepath.Join(t.TempDir(), "missing.go"), 1, 2, false, 0))

	lns := [][]byte{[]byte("/*"), []byte(" a comment"), []byte("*/"), []byte("code")}
	assert.Equal(t, 0, PreCommentStart(lns, 3, "//", "/*", "*/", 10), "a block comment")
}
