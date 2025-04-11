// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"cogentcore.org/core/system"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
)

// openLink opens given link, using the LinkHandler if non-nil,
// or the default system.TheApp.OpenURL() which will open a browser.
func (ed *Base) openLink(tl *rich.Hyperlink) {
	if ed.LinkHandler != nil {
		ed.LinkHandler(tl)
	} else {
		system.TheApp.OpenURL(tl.URL)
	}
}

// linkAt returns hyperlink at given source position, if one exists there,
// otherwise returns nil.
func (ed *Base) linkAt(pos textpos.Pos) (*rich.Hyperlink, int) {
	lk := ed.Lines.LinkAt(pos)
	if lk == nil {
		return nil, -1
	}
	return lk, pos.Line
}

// OpenLinkAt opens a link at given cursor position, if one exists there.
// returns the link if found, else nil. Also highlights the selected link.
func (ed *Base) OpenLinkAt(pos textpos.Pos) (*rich.Hyperlink, int) {
	tl, ln := ed.linkAt(pos)
	if tl == nil {
		return nil, -1
	}
	ed.HighlightsReset()
	ed.highlightLink(tl, ln)
	ed.openLink(tl)
	return tl, pos.Line
}

// highlightLink highlights given hyperlink
func (ed *Base) highlightLink(lk *rich.Hyperlink, ln int) textpos.Region {
	reg := textpos.NewRegion(ln, lk.Range.Start, ln, lk.Range.End)
	ed.HighlightRegion(reg)
	return reg
}

// CursorNextLink moves cursor to next link. wraparound wraps around to top of
// buffer if none found -- returns true if found
func (ed *Base) CursorNextLink(wraparound bool) bool {
	if ed.NumLines() == 0 {
		return false
	}
	ed.validateCursor()
	nl, ln := ed.Lines.NextLink(ed.CursorPos)
	if nl == nil {
		if !wraparound {
			return false
		}
		nl, ln = ed.Lines.NextLink(textpos.Pos{}) // wraparound
		if nl == nil {
			return false
		}
	}
	ed.HighlightsReset()
	reg := ed.highlightLink(nl, ln)
	ed.SetCursorTarget(reg.Start)
	ed.savePosHistory(reg.Start)
	ed.NeedsRender()
	return true
}

// CursorPrevLink moves cursor to next link. wraparound wraps around to bottom of
// buffer if none found -- returns true if found
func (ed *Base) CursorPrevLink(wraparound bool) bool {
	if ed.NumLines() == 0 {
		return false
	}
	ed.validateCursor()
	nl, ln := ed.Lines.PrevLink(ed.CursorPos)
	if nl == nil {
		if !wraparound {
			return false
		}
		nl, ln = ed.Lines.PrevLink(ed.Lines.EndPos()) // wraparound
		if nl == nil {
			return false
		}
	}
	ed.HighlightsReset()
	reg := ed.highlightLink(nl, ln)
	ed.SetCursorTarget(reg.Start)
	ed.savePosHistory(reg.Start)
	ed.NeedsRender()
	return true
}
