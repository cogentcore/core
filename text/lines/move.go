// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"fmt"

	"cogentcore.org/core/text/textpos"
)

// moveForward moves given source position forward given number of rune steps.
func (ls *Lines) moveForward(pos textpos.Pos, steps int) textpos.Pos {
	if !ls.isValidPos(pos) {
		return pos
	}
	for range steps {
		pos.Char++
		llen := len(ls.lines[pos.Line])
		if pos.Char > llen {
			if pos.Line < len(ls.lines)-1 {
				pos.Char = 0
				pos.Line++
			} else {
				pos.Char = llen
				break
			}
		}
	}
	return pos
}

// moveBackward moves given source position backward given number of rune steps.
func (ls *Lines) moveBackward(pos textpos.Pos, steps int) textpos.Pos {
	if !ls.isValidPos(pos) {
		return pos
	}
	for range steps {
		pos.Char--
		if pos.Char < 0 {
			if pos.Line > 0 {
				pos.Line--
				pos.Char = len(ls.lines[pos.Line])
			} else {
				pos.Char = 0
				break
			}
		}
	}
	return pos
}

// moveForwardWord moves given source position forward given number of word steps.
func (ls *Lines) moveForwardWord(pos textpos.Pos, steps int) textpos.Pos {
	if !ls.isValidPos(pos) {
		return pos
	}
	nstep := 0
	for nstep < steps {
		op := pos.Char
		np, ns := textpos.ForwardWord(ls.lines[pos.Line], op, steps)
		nstep += ns
		pos.Char = np
		if np == op || pos.Line >= len(ls.lines)-1 {
			break
		}
		if nstep < steps {
			pos.Line++
			pos.Char = 0
		}
	}
	return pos
}

// moveBackwardWord moves given source position backward given number of word steps.
func (ls *Lines) moveBackwardWord(pos textpos.Pos, steps int) textpos.Pos {
	if !ls.isValidPos(pos) {
		return pos
	}
	nstep := 0
	for nstep < steps {
		op := pos.Char
		np, ns := textpos.BackwardWord(ls.lines[pos.Line], op, steps)
		nstep += ns
		pos.Char = np
		if pos.Line == 0 {
			break
		}
		if nstep < steps {
			pos.Line--
			pos.Char = len(ls.lines[pos.Line])
		}
	}
	return pos
}

// moveDown moves given source position down given number of display line steps,
// always attempting to use the given column position if the line is long enough.
func (ls *Lines) moveDown(vw *view, pos textpos.Pos, steps, col int) textpos.Pos {
	if !ls.isValidPos(pos) {
		return pos
	}
	vp := ls.posToView(vw, pos)
	nvp := vp
	nvp.Line = ls.viewLinesMoved(vw, vp.Line, steps)
	nvp.Char = cellChar(ls.viewLineCells(vw, nvp.Line), col)
	dp := ls.posFromView(vw, nvp)
	return dp
}

// moveUp moves given source position up given number of display line steps,
// always attempting to use the given column position if the line is long enough.
func (ls *Lines) moveUp(vw *view, pos textpos.Pos, steps, col int) textpos.Pos {
	if !ls.isValidPos(pos) {
		return pos
	}
	vp := ls.posToView(vw, pos)
	nvp := vp
	nvp.Line = ls.viewLinesMoved(vw, vp.Line, -steps)
	nvp.Char = cellChar(ls.viewLineCells(vw, nvp.Line), col)
	dp := ls.posFromView(vw, nvp)
	return dp
}

// viewLinesMoved returns the view line that is the given number of
// steps (negative for up) from the given view line, within range,
// skipping view lines that only contain virtual text.
func (ls *Lines) viewLinesMoved(vw *view, vl, steps int) int {
	dir := 1
	if steps < 0 {
		dir, steps = -1, -steps
	}
	nl := vl
	for steps > 0 {
		next := nl + dir
		for next >= 0 && next < vw.viewLines && ls.isVirtualViewLine(vw, next) {
			next += dir
		}
		if next < 0 || next >= vw.viewLines {
			break
		}
		nl = next
		steps--
	}
	return nl
}

// moveLineStart moves given source position to start of view line.
func (ls *Lines) moveLineStart(vw *view, pos textpos.Pos) textpos.Pos {
	if !ls.isValidPos(pos) {
		return pos
	}
	vp := ls.posToView(vw, pos)
	vp.Char = 0
	return ls.posFromView(vw, vp)
}

// moveLineEnd moves given source position to end of view line.
func (ls *Lines) moveLineEnd(vw *view, pos textpos.Pos) textpos.Pos {
	if !ls.isValidPos(pos) {
		return pos
	}
	vp := ls.posToView(vw, pos)
	vp.Char = ls.viewLineLen(vw, vp.Line)
	return ls.posFromView(vw, vp)
}

// transposeChar swaps the character at the cursor with the one before it.
func (ls *Lines) transposeChar(vw *view, pos textpos.Pos) bool {
	if !ls.isValidPos(pos) {
		return false
	}
	vp := ls.posToView(vw, pos)
	pvp := vp
	pvp.Char--
	if pvp.Char < 0 {
		return false
	}
	ppos := ls.posFromView(vw, pvp)
	if pos.Line >= len(ls.lines) || pos.Char >= len(ls.lines[pos.Line]) {
		fmt.Println("transpose err:", pos, "lines:", len(ls.lines))
		return false
	}
	chr := ls.lines[pos.Line][pos.Char]
	pchr := ls.lines[ppos.Line][ppos.Char]
	repl := string([]rune{chr, pchr})
	pos.Char++
	ls.replaceText(ppos, pos, ppos, repl, ReplaceMatchCase)
	return true
}

// wordsOfLine returns the ranges of the words of the given line.
func wordsOfLine(txt []rune) []textpos.Range {
	var wds []textpos.Range
	st := -1
	for i, r := range txt {
		brk := textpos.RuneIsWordBreak(r)
		switch {
		case !brk && st < 0:
			st = i
		case brk && st >= 0:
			wds = append(wds, textpos.Range{Start: st, End: i})
			st = -1
		}
	}
	if st >= 0 {
		wds = append(wds, textpos.Range{Start: st, End: len(txt)})
	}
	return wds
}

// transposeWord swaps the word at or before the given position with the
// next word on its line, or with the word before it if it is the last
// word of the line, as in "one tw|o three" to "one three two|". It returns
// the position after the moved words, and false if there are not two
// words to swap.
func (ls *Lines) transposeWord(pos textpos.Pos) (textpos.Pos, bool) {
	if !ls.isValidPos(pos) {
		return pos, false
	}
	txt := ls.lines[pos.Line]
	wds := wordsOfLine(txt)
	if len(wds) < 2 {
		return pos, false
	}
	wi := 0 // the word at or before pos
	for i, w := range wds {
		if w.Start <= pos.Char {
			wi = i
		}
	}
	if wi == len(wds)-1 {
		wi--
	}
	w1, w2 := wds[wi], wds[wi+1]
	repl := string(txt[w2.Start:w2.End]) + string(txt[w1.End:w2.Start]) + string(txt[w1.Start:w1.End])
	st := textpos.Pos{Line: pos.Line, Char: w1.Start}
	ed := textpos.Pos{Line: pos.Line, Char: w2.End}
	ls.replaceText(st, ed, st, repl, ReplaceNoMatchCase)
	return ed, true
}
