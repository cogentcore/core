// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"strings"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

func runeLines(s ...string) [][]rune {
	rs := make([][]rune, len(s))
	for i, l := range s {
		rs[i] = []rune(l)
	}
	return rs
}

func TestReplaceLinesMarkup(t *testing.T) {
	lns := NewLines()
	lns.Defaults()
	vid := lns.NewView(10)
	red := rich.NewStyle().SetFillColor(colors.Red)
	mu := func(s ...string) []rich.Text {
		m := make([]rich.Text, len(s))
		for i, l := range s {
			m[i] = rich.NewText(red, []rune(l))
		}
		return m
	}

	lns.ReplaceLinesMarkup(0, -1, runeLines("one", "two words wrap", "three"), mu("one", "two words wrap", "three"))
	assert.Equal(t, "one\ntwo words wrap\nthree\n", lns.String())
	assert.Equal(t, 4, lns.ViewLines(vid))
	sty, _ := lns.markup[1].Span(0)
	assert.Equal(t, red.FillColor(), sty.FillColor(), "markup is kept")

	// replace the tail, which only lays out the new lines
	lns.ReplaceLinesMarkup(2, -1, runeLines("3", "4 is a long line"), nil)
	assert.Equal(t, "one\ntwo words wrap\n3\n4 is a long line\n", lns.String())
	assert.Equal(t, 6, lns.ViewLines(vid))
	assert.Equal(t, textpos.Pos{Line: 3, Char: 7}, lns.PosFromView(vid, textpos.Pos{Line: 5}))

	// delete from the start, as for limiting a scrollback history
	lns.ReplaceLinesMarkup(0, 2, nil, nil)
	assert.Equal(t, "3\n4 is a long line\n", lns.String())
	assert.Equal(t, 3, lns.ViewLines(vid))
	assert.Equal(t, textpos.Pos{Line: 1, Char: 7}, lns.PosFromView(vid, textpos.Pos{Line: 2}))

	// in the middle
	lns.ReplaceLinesMarkup(1, 1, runeLines("x", "y"), nil)
	assert.Equal(t, "3\nx\ny\n4 is a long line\n", lns.String())
	assert.Equal(t, 5, lns.ViewLines(vid))

	lns.ReplaceLinesMarkup(0, -1, nil, nil)
	assert.Equal(t, 1, lns.NumLines(), "always one line")
	assert.Equal(t, "\n", lns.String())

	// many lines, appended in steps, have the same layout as all at once
	var all []string
	for i := range 50 {
		all = append(all, strings.Repeat("ab ", i%7+1))
	}
	for i := 0; i < 50; i += 5 {
		lns.ReplaceLinesMarkup(i, -1, runeLines(all[i:i+5]...), nil)
	}
	inc := lns.ViewLines(vid)
	lns.SetString(strings.Join(all, "\n"))
	assert.Equal(t, lns.ViewLines(vid), inc)
}
