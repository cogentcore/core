// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
)

// view provides a view onto a shared [Lines] text buffer, with different
// with and markup layout for each view. Views are managed by the Lines.
type view struct {
	// width is the current line width in rune characters, used for line wrapping.
	width int

	// totalLines is the total number of display lines, including line breaks.
	// this is updated during markup.
	totalLines int

	// nbreaks are the number of display lines per source line (0 if it all fits on
	// 1 display line).
	nbreaks []int

	// layout is a mapping from lines rune index to display line and char,
	// within the scope of each line. E.g., Line=0 is first display line,
	// 1 is one after the first line break, etc.
	layout [][]textpos.Pos16

	// markup is the layout-specific version of the [rich.Text] markup,
	// specific to the width of this view.
	markup []rich.Text

	// listeners is used for sending Change and Input events
	listeners events.Listeners
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
	ls.layoutAll(vw)
	return vw, id
}

// deleteView deletes view with given view id.
func (ls *Lines) deleteView(vid int) {
	delete(ls.views, vid)
}
