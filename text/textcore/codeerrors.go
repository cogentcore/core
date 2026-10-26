// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"image"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// CodeError is an error of the text of an editor at a region of it, such
// as an error of running the text as code, which [Base.SetCodeErrors]
// shows in the editor.
type CodeError struct {

	// Region is the region of the text that the error is about.
	Region textpos.Region

	// Msg is the message of the error.
	Msg string
}

// SetCodeErrors sets the errors of the text of the editor, such as those of
// running it as code, replacing any previous ones. Their regions are shown
// with the background color [Base.ErrorColor], and a dotted underline (the
// [token.Error] tag), and their messages are the tooltip of the editor at
// their regions. The regions follow the edits of the text. No errors
// clears them.
func (ed *Base) SetCodeErrors(errs ...CodeError) {
	if ed.Lines != nil {
		for _, ce := range ed.codeErrors {
			reg := ed.Lines.AdjustRegion(ce.Region)
			if !reg.IsNil() {
				ed.Lines.RemoveTag(reg.Start, token.Error)
			}
		}
		for i := range errs {
			reg := &errs[i].Region
			reg.TimeNow() // so that it can be adjusted for later edits
			for ln := reg.Start.Line; ln <= reg.End.Line; ln++ {
				st, end := 0, ed.Lines.LineLen(ln)
				if ln == reg.Start.Line {
					st = reg.Start.Char
				}
				if ln == reg.End.Line {
					end = reg.End.Char
				}
				if end > st {
					ed.Lines.AddTag(ln, st, end, token.Error)
				}
			}
		}
	}
	ed.codeErrors = errs
	ed.Restyle() // for the tooltip ability (see [Base.WidgetTooltip])
}

// CodeErrors returns the errors of the text of the editor
// (see [Base.SetCodeErrors]).
func (ed *Base) CodeErrors() []CodeError { return ed.codeErrors }

// codeErrorAt returns the error whose region contains the given text
// position, if any.
func (ed *Base) codeErrorAt(pos textpos.Pos) (CodeError, bool) {
	for _, ce := range ed.codeErrors {
		reg := ed.Lines.AdjustRegion(ce.Region)
		if !reg.IsNil() && reg.Contains(pos) {
			return ce, true
		}
	}
	return CodeError{}, false
}

// WidgetTooltip returns the message of the code error at the given
// position, if any (see [Base.SetCodeErrors]), and otherwise the tooltip
// of the editor. Without a position (as when styling, to know whether
// the editor has a tooltip), it returns the message of the first error.
func (ed *Base) WidgetTooltip(pos image.Point) (string, image.Point) {
	if ed.Lines != nil && len(ed.codeErrors) > 0 {
		if pos.X < 0 {
			return ed.codeErrors[0].Msg, pos
		}
		tp := ed.PixelToCursor(pos.Sub(ed.Geom.ContentBBox.Min))
		if ce, ok := ed.codeErrorAt(tp); ok {
			return ce.Msg, pos
		}
	}
	return ed.Frame.WidgetTooltip(pos)
}

// renderCodeErrors renders the backgrounds of the regions of the code
// errors (see [Base.SetCodeErrors]), in [Base.ErrorColor]. It must not
// be called with the Lines locked, as charStartPos locks them.
func (ed *Base) renderCodeErrors() {
	if len(ed.codeErrors) == 0 || ed.ErrorColor == nil {
		return
	}
	pc := &ed.Scene.Painter
	var off math32.Vector2 // charStartPos is in the coordinates of the window
	if ed.Scene != nil {
		off = math32.FromPoint(ed.Scene.SceneGeom.Pos)
	}
	fill := func(start math32.Vector2, endX float32) {
		if endX > start.X {
			pc.FillBox(start.Sub(off), math32.Vec2(endX-start.X, ed.charSize.Y), ed.ErrorColor)
		}
	}
	for _, ce := range ed.codeErrors {
		reg := ed.Lines.AdjustRegion(ce.Region)
		if reg.IsNil() {
			continue
		}
		for ln := reg.Start.Line; ln <= reg.End.Line; ln++ {
			st, end := 0, ed.Lines.LineLen(ln)
			if ln == reg.Start.Line {
				st = reg.Start.Char
			}
			if ln == reg.End.Line {
				end = min(reg.End.Char, end)
			}
			if end <= st {
				continue
			}
			// A rectangle for each visual line of the region, which can wrap.
			rowStart := ed.charStartPos(textpos.Pos{Line: ln, Char: st})
			for c := st + 1; c <= end; c++ {
				p := ed.charStartPos(textpos.Pos{Line: ln, Char: c})
				switch {
				case p.Y != rowStart.Y: // c starts a new visual line
					prev := ed.charStartPos(textpos.Pos{Line: ln, Char: c - 1})
					fill(rowStart, prev.X+ed.charSize.X)
					rowStart = p
				case c == end:
					fill(rowStart, p.X)
				}
			}
		}
	}
}
