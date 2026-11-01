// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"testing"

	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// TestLayoutViewLinesRange tests that the layout of a range of lines
// after their markup changes is that of all of the lines.
func TestLayoutViewLinesRange(t *testing.T) {
	src := "package main\n\nfunc main() {\n\tx := 1\n\tfmt.Println(x, \"a long string that wraps around\")\n}\n"
	ls, vid := NewLinesFromBytes("dummy.go", 20, []byte(src))
	ls.SetVirtualText("inlay", -1, VirtualText{Pos: textpos.Pos{Line: 3, Char: 2}, Text: rich.NewText(rich.NewStyle(), []rune(" int"))})
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	// check compares the layout of the view with a full layout
	check := func(what string) {
		t.Helper()
		got := *vw
		got.markup, got.vlineStarts, got.lineToVline = append([]rich.Text{}, vw.markup...), append([]textpos.Pos{}, vw.vlineStarts...), append([]int{}, vw.lineToVline...)
		ls.layoutViewLines(vw)
		assert.Equal(t, vw.vlineStarts, got.vlineStarts, what)
		assert.Equal(t, vw.lineToVline, got.lineToVline, what)
		assert.Equal(t, vw.viewLines, got.viewLines, what)
		assert.Equal(t, len(vw.markup), len(got.markup), what)
		for i := range vw.markup {
			assert.Equal(t, vw.markup[i].Join(), got.markup[i].Join(), what)
		}
	}
	for ln := range ls.numLines() {
		ls.markupLines(ln, ln)
		check("the markup of a line")
	}
	// a line that wraps into more view lines, which moves the lines after it
	ls.lines[3] = []rune("\tx := 1 + 2 + 3 + 4 + 5 + 6")
	ls.markupLines(3, 3)
	check("a line with more view lines")
	ls.lines[3] = []rune("\tx := 1")
	ls.markupLines(2, 4)
	check("a range of lines with fewer view lines")
}
