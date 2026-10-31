// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
)

// view provides a view onto a shared [Lines] text buffer,
// with a representation of view lines that are the wrapped versions of
// the original [Lines.lines] source lines, with wrapping according to
// the view width. Views are managed by the Lines.
type view struct {
	// id is the unique id of the view in its [Lines].
	id int

	// width is the current line width in rune characters, used for line wrapping.
	width int

	// viewLines is the total number of line-wrapped lines.
	viewLines int

	// vlineStarts are the positions of the start of each view line, as the
	// source line and the display char position in that line, which includes
	// any [VirtualText] (see [Lines.srcToDisplay]). This slice is viewLines in length.
	vlineStarts []textpos.Pos

	// markup is the view-specific version of the [Lines.markup] markup for
	// each view line (len = viewLines).
	markup []rich.Text

	// lineToVline maps the source [Lines.lines] indexes to the wrapped
	// viewLines. Each slice value contains the index into the viewLines space,
	// such that vlineStarts of that index is the start of the original source line.
	// Any subsequent vlineStarts with the same Line and Char > 0 following this
	// starting line represent additional wrapped content from the same source line.
	lineToVline []int

	// listeners is used for sending Change, Input, and Close events to views.
	listeners events.Listeners

	// runAsync, if set, runs the listeners for an event that is sent from
	// a background goroutine (see [Lines.SetAsyncRunner]).
	runAsync func(f func())
}

// viewLineLen returns the length in chars (runes) of the given view line.
func (ls *Lines) viewLineLen(vw *view, vl int) int {
	n := len(vw.vlineStarts)
	if n == 0 {
		return 0
	}
	if vl < 0 {
		vl = 0
	}
	if vl >= n {
		vl = n - 1
	}
	vs := vw.vlineStarts[vl]
	sl := ls.displayLineLen(vw, vs.Line)
	if vl == vw.viewLines-1 {
		return sl + 1 - vs.Char
	}
	np := vw.vlineStarts[vl+1]
	if np.Line == vs.Line {
		return np.Char - vs.Char
	}
	return sl + 1 - vs.Char
}

// viewLinesRange returns the start and end view lines for given
// source line number, using only lineToVline. ed is inclusive.
func (ls *Lines) viewLinesRange(vw *view, ln int) (st, ed int) {
	n := len(vw.lineToVline)
	st = vw.lineToVline[ln]
	if ln+1 < n {
		ed = vw.lineToVline[ln+1] - 1
	} else {
		ed = vw.viewLines - 1
	}
	return
}

// validViewLine returns a view line that is in range based on given
// source line.
func (ls *Lines) validViewLine(vw *view, ln int) int {
	if ln < 0 {
		return 0
	} else if ln >= len(vw.lineToVline) {
		return vw.viewLines - 1
	}
	return vw.lineToVline[ln]
}

// posToView returns the view position in terms of viewLines and Char
// offset into that view line for given source line, char position.
// Is robust to out-of-range positions.
func (ls *Lines) posToView(vw *view, pos textpos.Pos) textpos.Pos {
	if pos.Line < 0 || pos.Line >= len(ls.lines) {
		return ls.displayToView(vw, pos)
	}
	pos.Char = ls.srcToDisplay(vw, pos.Line, pos.Char)
	vp := ls.displayToView(vw, pos)
	// A position at the start of a view line that starts with virtual
	// text is shown at the end of the previous view line, as the cursor
	// is shown before virtual text.
	if vp.Char == 0 && vp.Line > 0 && vp.Line < len(vw.vlineStarts) {
		prev := vw.vlineStarts[vp.Line-1]
		if prev.Line == pos.Line && ls.isVirtualDisplay(vw, pos.Line, pos.Char) {
			return textpos.Pos{Line: vp.Line - 1, Char: pos.Char - prev.Char}
		}
	}
	return vp
}

// displayToView returns the view position for the given position
// in terms of the source line and display char in that line.
func (ls *Lines) displayToView(vw *view, pos textpos.Pos) textpos.Pos {
	vp := pos
	vl := ls.validViewLine(vw, pos.Line)
	vp.Line = vl
	vlen := ls.viewLineLen(vw, vl)
	if pos.Char < vlen {
		return vp
	}
	nl := vl + 1
	if nl == vw.viewLines {
		vp.Char = pos.Char
		return vp
	}
	for nl < vw.viewLines && vw.vlineStarts[nl].Line == pos.Line {
		np := vw.vlineStarts[nl]
		vlen := ls.viewLineLen(vw, nl)
		if pos.Char >= np.Char && pos.Char < np.Char+vlen {
			np.Line = nl
			np.Char = pos.Char - np.Char
			return np
		}
		nl++
	}
	return vp
}

// posFromView returns the original source position from given
// view position in terms of viewLines and Char offset into that view line.
// If the Char position is beyond the end of the line, it returns the
// end of the given line.
func (ls *Lines) posFromView(vw *view, vp textpos.Pos) textpos.Pos {
	n := len(vw.vlineStarts)
	if n == 0 {
		return textpos.Pos{}
	}
	vl := vp.Line
	if vl < 0 {
		vl = 0
	} else if vl >= n {
		vl = n - 1
	}
	vlen := ls.viewLineLen(vw, vl)
	if vlen == 0 {
		vlen = 1
	}
	sp := vw.vlineStarts[vl]
	if vl+1 < n && vw.vlineStarts[vl+1].Line == sp.Line && ls.isVirtualDisplay(vw, sp.Line, vw.vlineStarts[vl+1].Char) {
		vlen++ // the end of the view line is before the virtual text that starts the next one
	}
	vp.Char = min(vp.Char, vlen-1)
	pos := vp
	pos.Line = sp.Line
	pos.Char = ls.displayToSrc(vw, sp.Line, sp.Char+vp.Char)
	return pos
}

// regionToView converts the given region in source coordinates into view coordinates.
func (ls *Lines) regionToView(vw *view, reg textpos.Region) textpos.Region {
	return textpos.Region{Start: ls.posToView(vw, reg.Start), End: ls.posToView(vw, reg.End)}
}

// regionFromView converts the given region in view coordinates into source coordinates.
func (ls *Lines) regionFromView(vw *view, reg textpos.Region) textpos.Region {
	return textpos.Region{Start: ls.posFromView(vw, reg.Start), End: ls.posFromView(vw, reg.End)}
}

// viewLineRegion returns the region in view coordinates of the given view line.
func (ls *Lines) viewLineRegion(vw *view, vln int) textpos.Region {
	llen := ls.viewLineLen(vw, vln)
	return textpos.Region{Start: textpos.Pos{Line: vln}, End: textpos.Pos{Line: vln, Char: llen}}
}

// initViews ensures that the views map is constructed.
func (ls *Lines) initViews() {
	if ls.views == nil {
		ls.views = make(map[int]*view)
	}
}

// view returns view for given unique view id. nil if not found.
func (ls *Lines) view(vid int) *view {
	ls.initViews()
	return ls.views[vid]
}

// newView makes a new view with next available id, using given initial width.
func (ls *Lines) newView(width int) (*view, int) {
	ls.initViews()
	mxi := 0
	for i := range ls.views {
		mxi = max(i, mxi)
	}
	id := mxi + 1
	vw := &view{id: id, width: width}
	ls.views[id] = vw
	ls.layoutViewLines(vw)
	return vw, id
}

// deleteView deletes view with given view id.
func (ls *Lines) deleteView(vid int) {
	delete(ls.views, vid)
}

// ViewMarkupLine returns the markup [rich.Text] line for given view and
// view line number. This must be called under the mutex Lock! It is the
// api for rendering the lines.
func (ls *Lines) ViewMarkupLine(vid, line int) rich.Text {
	vw := ls.view(vid)
	if line >= 0 && len(vw.markup) > line {
		return vw.markup[line]
	}
	return rich.Text{}
}
