// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
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
	vl := vw.viewLines
	vp := ls.posToView(vw, pos)
	nvp := vp
	nvp.Line = min(nvp.Line+steps, vl-1)
	nvp.Char = col
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
	nvp.Line = max(nvp.Line-steps, 0)
	nvp.Char = col
	dp := ls.posFromView(vw, nvp)
	return dp
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
