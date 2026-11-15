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
	// The unexported functions tested here require the lock, which also
	// keeps the background markup from changing the layout during the test.
	lns.Lock()
	defer lns.Unlock()
	vw := lns.view(vid)

	// for ln := range vw.viewLines {
	// 	ft := string(vw.markup[ln].Join())
	// 	fmt.Println(ft)
	// }

	posTests := []struct {
		pos  textpos.Pos
		vpos textpos.Pos
	}{
		{textpos.Pos{Line: 0, Char: 0}, textpos.Pos{Line: 0, Char: 0}},
		{textpos.Pos{Line: 0, Char: 90}, textpos.Pos{Line: 1, Char: 12}},
		{textpos.Pos{Line: 0, Char: 180}, textpos.Pos{Line: 2, Char: 24}},
		{textpos.Pos{Line: 0, Char: 260}, textpos.Pos{Line: 3, Char: 26}},
		{textpos.Pos{Line: 0, Char: 438}, textpos.Pos{Line: 5, Char: 48}},
		{textpos.Pos{Line: 1, Char: 10}, textpos.Pos{Line: 6, Char: 10}},
		{textpos.Pos{Line: 1, Char: 41}, textpos.Pos{Line: 6, Char: 41}},
		{textpos.Pos{Line: 2, Char: 42}, textpos.Pos{Line: 7, Char: 42}},
	}
	for _, test := range posTests {
		vp := lns.posToView(vw, test.pos)
		// ln := vw.markup[vp.Line]
		// txt := ln.Join()
		// fmt.Println(test.pos, vp, string(txt[vp.Char:min(len(txt), vp.Char+8)]))
		assert.Equal(t, test.vpos, vp)

		sp := lns.posFromView(vw, vp)
		assert.Equal(t, test.pos, sp)
	}

	vposTests := []struct {
		vpos textpos.Pos
		pos  textpos.Pos
	}{
		{textpos.Pos{Line: 0, Char: 0}, textpos.Pos{Line: 0, Char: 0}},
		{textpos.Pos{Line: 0, Char: 90}, textpos.Pos{Line: 0, Char: 77}},
		{textpos.Pos{Line: 1, Char: 90}, textpos.Pos{Line: 0, Char: 155}},
	}
	for _, test := range vposTests {
		sp := lns.posFromView(vw, test.vpos)
		// txt := lns.lines[sp.Line]
		// fmt.Println(test.vpos, sp, string(txt[sp.Char:min(len(txt), sp.Char+8)]))
		assert.Equal(t, test.pos, sp)
	}

	fwdTests := []struct {
		pos   textpos.Pos
		steps int
		tpos  textpos.Pos
	}{
		{textpos.Pos{Line: 0, Char: 0}, 1, textpos.Pos{Line: 0, Char: 1}},
		{textpos.Pos{Line: 0, Char: 0}, 8, textpos.Pos{Line: 0, Char: 8}},
		{textpos.Pos{Line: 0, Char: 380}, 1, textpos.Pos{Line: 0, Char: 381}},
		{textpos.Pos{Line: 0, Char: 438}, 1, textpos.Pos{Line: 0, Char: 439}},
		{textpos.Pos{Line: 0, Char: 439}, 1, textpos.Pos{Line: 0, Char: 440}},
		{textpos.Pos{Line: 0, Char: 440}, 1, textpos.Pos{Line: 1, Char: 0}},
		{textpos.Pos{Line: 0, Char: 439}, 2, textpos.Pos{Line: 1, Char: 0}},
		{textpos.Pos{Line: 2, Char: 393}, 1, textpos.Pos{Line: 2, Char: 394}},
		{textpos.Pos{Line: 2, Char: 395}, 1, textpos.Pos{Line: 2, Char: 395}},
		{textpos.Pos{Line: 2, Char: 395}, 10, textpos.Pos{Line: 2, Char: 395}},
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
		{textpos.Pos{Line: 0, Char: 0}, 1, textpos.Pos{Line: 0, Char: 0}},
		{textpos.Pos{Line: 0, Char: 0}, 8, textpos.Pos{Line: 0, Char: 0}},
		{textpos.Pos{Line: 0, Char: 380}, 1, textpos.Pos{Line: 0, Char: 379}},
		{textpos.Pos{Line: 0, Char: 438}, 1, textpos.Pos{Line: 0, Char: 437}},
		{textpos.Pos{Line: 0, Char: 439}, 1, textpos.Pos{Line: 0, Char: 438}},
		{textpos.Pos{Line: 0, Char: 440}, 1, textpos.Pos{Line: 0, Char: 439}},
		{textpos.Pos{Line: 1, Char: 0}, 1, textpos.Pos{Line: 0, Char: 440}},
		{textpos.Pos{Line: 1, Char: 0}, 2, textpos.Pos{Line: 0, Char: 439}},
		{textpos.Pos{Line: 2, Char: 393}, 1, textpos.Pos{Line: 2, Char: 392}},
		{textpos.Pos{Line: 2, Char: 395}, 1, textpos.Pos{Line: 2, Char: 394}},
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
		{textpos.Pos{Line: 0, Char: 0}, 1, textpos.Pos{Line: 0, Char: 3}},
		{textpos.Pos{Line: 0, Char: 3}, 1, textpos.Pos{Line: 0, Char: 9}},
		{textpos.Pos{Line: 0, Char: 0}, 2, textpos.Pos{Line: 0, Char: 9}},
		{textpos.Pos{Line: 0, Char: 382}, 1, textpos.Pos{Line: 0, Char: 389}},
		{textpos.Pos{Line: 0, Char: 438}, 1, textpos.Pos{Line: 0, Char: 439}},
		{textpos.Pos{Line: 0, Char: 439}, 1, textpos.Pos{Line: 0, Char: 439}},
		{textpos.Pos{Line: 0, Char: 440}, 1, textpos.Pos{Line: 1, Char: 2}},
		{textpos.Pos{Line: 0, Char: 440}, 2, textpos.Pos{Line: 1, Char: 11}},
		{textpos.Pos{Line: 2, Char: 390}, 1, textpos.Pos{Line: 2, Char: 394}},
		{textpos.Pos{Line: 2, Char: 395}, 1, textpos.Pos{Line: 2, Char: 394}},
		{textpos.Pos{Line: 2, Char: 395}, 5, textpos.Pos{Line: 2, Char: 394}},
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
		{textpos.Pos{Line: 0, Char: 0}, 1, textpos.Pos{Line: 0, Char: 0}},
		{textpos.Pos{Line: 0, Char: 0}, 1, textpos.Pos{Line: 0, Char: 0}},
		{textpos.Pos{Line: 0, Char: 3}, 1, textpos.Pos{Line: 0, Char: 0}},
		{textpos.Pos{Line: 0, Char: 3}, 5, textpos.Pos{Line: 0, Char: 0}},
		{textpos.Pos{Line: 0, Char: 9}, 2, textpos.Pos{Line: 0, Char: 0}},
		{textpos.Pos{Line: 0, Char: 382}, 1, textpos.Pos{Line: 0, Char: 377}},
		{textpos.Pos{Line: 1, Char: 0}, 1, textpos.Pos{Line: 0, Char: 435}},
		{textpos.Pos{Line: 1, Char: 0}, 2, textpos.Pos{Line: 0, Char: 424}},
		{textpos.Pos{Line: 2, Char: 395}, 1, textpos.Pos{Line: 2, Char: 389}},
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
		{textpos.Pos{Line: 0, Char: 0}, 1, 50, textpos.Pos{Line: 0, Char: 128}},
		{textpos.Pos{Line: 0, Char: 0}, 2, 50, textpos.Pos{Line: 0, Char: 206}},
		{textpos.Pos{Line: 0, Char: 0}, 4, 60, textpos.Pos{Line: 0, Char: 371}},
		{textpos.Pos{Line: 0, Char: 0}, 5, 60, textpos.Pos{Line: 0, Char: 440}},
		{textpos.Pos{Line: 0, Char: 371}, 2, 60, textpos.Pos{Line: 1, Char: 42}},
		{textpos.Pos{Line: 1, Char: 30}, 1, 60, textpos.Pos{Line: 2, Char: 60}},
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
		{textpos.Pos{Line: 0, Char: 128}, 1, 50, textpos.Pos{Line: 0, Char: 50}},
		{textpos.Pos{Line: 0, Char: 128}, 2, 50, textpos.Pos{Line: 0, Char: 50}},
		{textpos.Pos{Line: 0, Char: 206}, 1, 50, textpos.Pos{Line: 0, Char: 128}},
		{textpos.Pos{Line: 0, Char: 371}, 1, 60, textpos.Pos{Line: 0, Char: 294}},
		{textpos.Pos{Line: 1, Char: 5}, 1, 60, textpos.Pos{Line: 0, Char: 440}},
		{textpos.Pos{Line: 1, Char: 5}, 1, 20, textpos.Pos{Line: 0, Char: 410}},
		{textpos.Pos{Line: 1, Char: 5}, 2, 60, textpos.Pos{Line: 0, Char: 371}},
		{textpos.Pos{Line: 1, Char: 5}, 3, 50, textpos.Pos{Line: 0, Char: 284}},
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
