// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"testing"

	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

func TestRuneWidth(t *testing.T) {
	// CJK, fullwidth forms, and emoji are wide; halfwidth forms
	// and zero width chars are not
	for _, r := range "中한にＡ😀" {
		assert.Equal(t, 2, RuneWidth(r), "%q", r)
	}
	for _, r := range "aé\tｱ\u0301\u200b" {
		assert.Equal(t, 1, RuneWidth(r), "%q", r)
	}
}

func TestLineCells(t *testing.T) {
	lns, vid := NewLinesFromBytes("dummy.txt", 80, []byte("\tab中文\tx\n"))
	lns.Lock()
	defer lns.Unlock()
	ts := lns.Settings.TabSize
	cells := LineCells(lns.ViewMarkupLine(vid, 0), ts)
	// the tab at the start takes the tab size, the other tab goes to the
	// next multiple of 8, and the wide chars take 2 cells
	tab := ((ts+6)/8 + 1) * 8
	assert.Equal(t, []int{0, ts, ts + 1, ts + 2, ts + 4, ts + 6, tab, tab + 1}, cells)

	vw := lns.view(vid)
	assert.Equal(t, ts+4, lns.viewCell(vw, textpos.Pos{Line: 0, Char: 4}))
	assert.Equal(t, tab+3, lns.viewCell(vw, textpos.Pos{Line: 0, Char: 9}), "beyond the end")

	// a column in the second half of a wide char is after it
	assert.Equal(t, 3, cellChar(cells, ts+2))
	assert.Equal(t, 4, cellChar(cells, ts+3))
	assert.Equal(t, 6, cellChar(cells, tab-1))
	assert.Equal(t, 7, cellChar(cells, tab+5))
}

func TestWrapWide(t *testing.T) {
	src := "中文中文中文中文\nab 中文中文中文 cd\n"
	lns, vid := NewLinesFromBytes("dummy.txt", 10, []byte(src))
	lns.Lock()
	defer lns.Unlock()
	vw := lns.view(vid)
	// the lines are wrapped at their width in cells: a word of wide
	// chars is broken into lines that fill the width
	assert.Equal(t, []textpos.Pos{{0, 0}, {0, 5}, {1, 0}, {1, 3}, {1, 8}}, vw.vlineStarts)
	for vl := range vw.viewLines {
		cells := lns.viewLineCells(vw, vl)
		assert.LessOrEqual(t, cells[len(cells)-1], 10, "view line %d", vl)
	}
}

func TestMoveWide(t *testing.T) {
	lns, vid := NewLinesFromBytes("dummy.txt", 80, []byte("abcdef\n中文中文\nabcdef\n"))
	lns.Lock()
	defer lns.Unlock()
	vw := lns.view(vid)
	// moving up and down keeps the column in cells, with a column in
	// the second half of a wide char after it
	down := func(pos textpos.Pos) textpos.Pos {
		col := lns.viewCell(vw, lns.posToView(vw, pos))
		return lns.moveDown(vw, pos, 1, col)
	}
	assert.Equal(t, textpos.Pos{Line: 1, Char: 2}, down(textpos.Pos{Line: 0, Char: 4}))
	assert.Equal(t, textpos.Pos{Line: 1, Char: 2}, down(textpos.Pos{Line: 0, Char: 3}))
	assert.Equal(t, textpos.Pos{Line: 1, Char: 3}, down(textpos.Pos{Line: 0, Char: 6}))
	assert.Equal(t, textpos.Pos{Line: 2, Char: 4}, down(textpos.Pos{Line: 1, Char: 2}))
	col := lns.viewCell(vw, textpos.Pos{Line: 1, Char: 2})
	assert.Equal(t, textpos.Pos{Line: 0, Char: 4}, lns.moveUp(vw, textpos.Pos{Line: 1, Char: 2}, 1, col))
}
