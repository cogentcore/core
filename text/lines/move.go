// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/text/textpos"
)

// displayPos returns the local display position of rune
// at given source line and char: wrapped line, char.
// returns -1, -1 for an invalid source position.
func (ls *Lines) displayPos(pos textpos.Pos) textpos.Pos {
	if errors.Log(ls.isValidPos(pos)) != nil {
		return textpos.Pos{-1, -1}
	}
	return ls.layout[pos.Line][pos.Char].ToPos()
}

// displayToPos finds the closest source line, char position for given
// local display position within given source line, for wrapped
// lines with nbreaks > 0. The result will be on the target line
// if there is text on that line, but the Char position may be
// less than the target depending on the line length.
func (ls *Lines) displayToPos(ln int, pos textpos.Pos) textpos.Pos {
	nb := ls.nbreaks[ln]
	sz := len(ls.lines[ln])
	if sz == 0 {
		return textpos.Pos{ln, 0}
	}
	pos.Char = min(pos.Char, sz-1)
	if nb == 0 {
		return textpos.Pos{ln, pos.Char}
	}
	if pos.Line >= nb { // nb is len-1 already
		pos.Line = nb
	}
	lay := ls.layout[ln]
	sp := ls.width*pos.Line + pos.Char // initial guess for starting position
	sp = min(sp, sz-1)
	// first get to the correct line
	for sp < sz-1 && lay[sp].Line < int16(pos.Line) {
		sp++
	}
	for sp > 0 && lay[sp].Line > int16(pos.Line) {
		sp--
	}
	if lay[sp].Line != int16(pos.Line) {
		return textpos.Pos{ln, sp}
	}
	// now get to the correct char
	for sp < sz-1 && lay[sp].Line == int16(pos.Line) && lay[sp].Char < int16(pos.Char) {
		sp++
	}
	if lay[sp].Line != int16(pos.Line) { // went too far
		return textpos.Pos{ln, sp - 1}
	}
	for sp > 0 && lay[sp].Line == int16(pos.Line) && lay[sp].Char > int16(pos.Char) {
		sp--
	}
	if lay[sp].Line != int16(pos.Line) { // went too far
		return textpos.Pos{ln, sp + 1}
	}
	return textpos.Pos{ln, sp}
}

// moveForward moves given source position forward given number of rune steps.
func (ls *Lines) moveForward(pos textpos.Pos, steps int) textpos.Pos {
	if errors.Log(ls.isValidPos(pos)) != nil {
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
	if errors.Log(ls.isValidPos(pos)) != nil {
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
	if errors.Log(ls.isValidPos(pos)) != nil {
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
	if errors.Log(ls.isValidPos(pos)) != nil {
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
func (ls *Lines) moveDown(pos textpos.Pos, steps, col int) textpos.Pos {
	if errors.Log(ls.isValidPos(pos)) != nil {
		return pos
	}
	nl := len(ls.lines)
	nsteps := 0
	for nsteps < steps {
		gotwrap := false
		if nbreak := ls.nbreaks[pos.Line]; nbreak > 0 {
			dp := ls.displayPos(pos)
			if dp.Line < nbreak {
				dp.Line++
				dp.Char = col // shoot for col
				pos = ls.displayToPos(pos.Line, dp)
				adp := ls.displayPos(pos)
				ns := adp.Line - dp.Line
				if ns > 0 {
					nsteps += ns
					gotwrap = true
				}
			}
		}
		if !gotwrap { // go to next source line
			if pos.Line >= nl-1 {
				pos.Line = nl - 1
				break
			}
			pos.Char = col // try for col
			pos.Char = min(len(ls.lines[pos.Line]), pos.Char)
			nsteps++
		}
	}
	return pos
}
