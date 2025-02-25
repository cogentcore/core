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
	// width is the current line width in rune characters, used for line wrapping.
	width int

	// viewLines is the total number of line-wrapped lines.
	viewLines int

	// vlineStarts are the positions in the original [Lines.lines] source for
	// the start of each view line. This slice is viewLines in length.
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
	sl := ls.lines[vs.Line]
	if vl == vw.viewLines-1 {
		return len(sl) - vs.Char
	}
	np := vw.vlineStarts[vl+1]
	if np.Line == vs.Line {
		return np.Char - vs.Char
	}
	return len(sl) + 1 - vs.Char
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
	vp := pos
	vl := ls.validViewLine(vw, pos.Line)
	vp.Line = vl
	vlen := ls.viewLineLen(vw, vl)
	if pos.Char < vlen {
		return vp
	}
	nl := vl + 1
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
	vp.Char = min(vp.Char, vlen-1)
	pos := vp
	sp := vw.vlineStarts[vl]
	pos.Line = sp.Line
	pos.Char = sp.Char + vp.Char
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
	vw := &view{width: width}
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
