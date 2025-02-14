// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"testing"

	_ "cogentcore.org/core/system/driver"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

func TestMove(t *testing.T) {
	src := `The [rich.Text](http://rich.text.com) type is the standard representation for formatted text, used as the input to the "shaped" package for text layout and rendering. It is encoded purely using "[]rune" slices for each span, with the _style_ information **represented** with special rune values at the start of each span. This is an efficient and GPU-friendly pure-value format that avoids any issues of style struct pointer management etc.
It provides basic font styling properties.
The "n" newline is used to mark the end of a paragraph, and in general text will be automatically wrapped to fit a given size, in the "shaped" package. If the text starting after a newline has a ParagraphStart decoration, then it will be styled according to the "text.Style" paragraph styles (indent and paragraph spacing). The HTML parser sets this as appropriate based on "<br>" vs "<p>" tags.
`

	lns, vid := NewLinesFromBytes("dummy.md", 80, []byte(src))
	vw := lns.view(vid)

	// ft0 := string(vw.markup[0].Join())
	// ft1 := string(vw.markup[1].Join())
	// ft2 := string(vw.markup[2].Join())
	// fmt.Println(ft0)
	// fmt.Println(ft1)
	// fmt.Println(ft2)

	fwdTests := []struct {
		pos   textpos.Pos
		steps int
		tpos  textpos.Pos
	}{
		{textpos.Pos{0, 0}, 1, textpos.Pos{0, 1}},
		{textpos.Pos{0, 0}, 8, textpos.Pos{0, 8}},
		{textpos.Pos{0, 380}, 1, textpos.Pos{0, 381}},
		{textpos.Pos{0, 438}, 1, textpos.Pos{0, 439}},
		{textpos.Pos{0, 439}, 1, textpos.Pos{0, 440}},
		{textpos.Pos{0, 440}, 1, textpos.Pos{1, 0}},
		{textpos.Pos{0, 439}, 2, textpos.Pos{1, 0}},
		{textpos.Pos{2, 393}, 1, textpos.Pos{2, 394}},
		{textpos.Pos{2, 395}, 1, textpos.Pos{2, 395}},
		{textpos.Pos{2, 395}, 10, textpos.Pos{2, 395}},
	}
	for _, test := range fwdTests {
		tp := lns.moveForward(test.pos, test.steps)
		// ln := lns.lines[tp.Line]
		// fmt.Println(test.pos, test.steps, tp, string(ln[tp.Char:min(len(ln), tp.Char+8)]))
		assert.Equal(t, test.tpos, tp)
	}

	bkwdTests := []struct {
		pos   textpos.Pos
		steps int
		tpos  textpos.Pos
	}{
		{textpos.Pos{0, 0}, 1, textpos.Pos{0, 0}},
		{textpos.Pos{0, 0}, 8, textpos.Pos{0, 0}},
		{textpos.Pos{0, 380}, 1, textpos.Pos{0, 379}},
		{textpos.Pos{0, 438}, 1, textpos.Pos{0, 437}},
		{textpos.Pos{0, 439}, 1, textpos.Pos{0, 438}},
		{textpos.Pos{0, 440}, 1, textpos.Pos{0, 439}},
		{textpos.Pos{1, 0}, 1, textpos.Pos{0, 440}},
		{textpos.Pos{1, 0}, 2, textpos.Pos{0, 439}},
		{textpos.Pos{2, 393}, 1, textpos.Pos{2, 392}},
		{textpos.Pos{2, 395}, 1, textpos.Pos{2, 394}},
	}
	for _, test := range bkwdTests {
		tp := lns.moveBackward(test.pos, test.steps)
		// ln := lns.lines[tp.Line]
		// fmt.Println(test.pos, test.steps, tp, string(ln[tp.Char:min(len(ln), tp.Char+8)]))
		assert.Equal(t, test.tpos, tp)
	}

	fwdWordTests := []struct {
		pos   textpos.Pos
		steps int
		tpos  textpos.Pos
	}{
		{textpos.Pos{0, 0}, 1, textpos.Pos{0, 3}},
		{textpos.Pos{0, 3}, 1, textpos.Pos{0, 9}},
		{textpos.Pos{0, 0}, 2, textpos.Pos{0, 9}},
		{textpos.Pos{0, 382}, 1, textpos.Pos{0, 389}},
		{textpos.Pos{0, 438}, 1, textpos.Pos{0, 439}},
		{textpos.Pos{0, 439}, 1, textpos.Pos{0, 439}},
		{textpos.Pos{0, 440}, 1, textpos.Pos{1, 2}},
		{textpos.Pos{0, 440}, 2, textpos.Pos{1, 11}},
		{textpos.Pos{2, 390}, 1, textpos.Pos{2, 394}},
		{textpos.Pos{2, 395}, 1, textpos.Pos{2, 394}},
		{textpos.Pos{2, 395}, 5, textpos.Pos{2, 394}},
	}
	for _, test := range fwdWordTests {
		tp := lns.moveForwardWord(test.pos, test.steps)
		// ln := lns.lines[tp.Line]
		// fmt.Println(test.pos, test.steps, tp, string(ln[min(tp.Char, test.pos.Char):tp.Char]))
		assert.Equal(t, test.tpos, tp)
	}

	bkwdWordTests := []struct {
		pos   textpos.Pos
		steps int
		tpos  textpos.Pos
	}{
		{textpos.Pos{0, 0}, 1, textpos.Pos{0, 0}},
		{textpos.Pos{0, 0}, 1, textpos.Pos{0, 0}},
		{textpos.Pos{0, 3}, 1, textpos.Pos{0, 0}},
		{textpos.Pos{0, 3}, 5, textpos.Pos{0, 0}},
		{textpos.Pos{0, 9}, 2, textpos.Pos{0, 0}},
		{textpos.Pos{0, 382}, 1, textpos.Pos{0, 377}},
		{textpos.Pos{1, 0}, 1, textpos.Pos{0, 435}},
		{textpos.Pos{1, 0}, 2, textpos.Pos{0, 424}},
		{textpos.Pos{2, 395}, 1, textpos.Pos{2, 389}},
	}
	for _, test := range bkwdWordTests {
		tp := lns.moveBackwardWord(test.pos, test.steps)
		// ln := lns.lines[tp.Line]
		// fmt.Println(test.pos, test.steps, tp, string(ln[min(tp.Char, test.pos.Char):max(tp.Char, test.pos.Char)]))
		assert.Equal(t, test.tpos, tp)
	}

	downTests := []struct {
		pos   textpos.Pos
		steps int
		col   int
		tpos  textpos.Pos
	}{
		{textpos.Pos{0, 0}, 1, 50, textpos.Pos{0, 128}},
		{textpos.Pos{0, 0}, 2, 50, textpos.Pos{0, 206}},
		{textpos.Pos{0, 0}, 4, 60, textpos.Pos{0, 371}},
		{textpos.Pos{0, 0}, 5, 60, textpos.Pos{0, 440}},
		{textpos.Pos{0, 371}, 2, 60, textpos.Pos{1, 42}},
		{textpos.Pos{1, 30}, 1, 60, textpos.Pos{2, 60}},
	}
	for _, test := range downTests {
		tp := lns.moveDown(vw, test.pos, test.steps, test.col)
		// sp := test.pos
		// stln := lns.lines[sp.Line]
		// fmt.Println(sp, test.steps, tp, string(stln[min(test.col, len(stln)-1):min(test.col+5, len(stln))]))
		// ln := lns.lines[tp.Line]
		// fmt.Println(test.pos, test.steps, tp, string(ln[tp.Char:min(tp.Char+5, len(ln))]))
		assert.Equal(t, test.tpos, tp)
	}

	upTests := []struct {
		pos   textpos.Pos
		steps int
		col   int
		tpos  textpos.Pos
	}{
		{textpos.Pos{0, 128}, 1, 50, textpos.Pos{0, 50}},
		{textpos.Pos{0, 128}, 2, 50, textpos.Pos{0, 50}},
		{textpos.Pos{0, 206}, 1, 50, textpos.Pos{0, 128}},
		{textpos.Pos{0, 371}, 1, 60, textpos.Pos{0, 294}},
		{textpos.Pos{1, 5}, 1, 60, textpos.Pos{0, 440}},
		{textpos.Pos{1, 5}, 1, 20, textpos.Pos{0, 410}},
		{textpos.Pos{1, 5}, 2, 60, textpos.Pos{0, 371}},
		{textpos.Pos{1, 5}, 3, 50, textpos.Pos{0, 284}},
	}
	for _, test := range upTests {
		tp := lns.moveUp(vw, test.pos, test.steps, test.col)
		// sp := test.pos
		// stln := lns.lines[sp.Line]
		// fmt.Println(sp, test.steps, tp, string(stln[min(test.col, len(stln)-1):min(test.col+5, len(stln))]))
		// ln := lns.lines[tp.Line]
		// fmt.Println(test.pos, test.steps, tp, string(ln[tp.Char:min(tp.Char+5, len(ln))]))
		assert.Equal(t, test.tpos, tp)
	}
}
