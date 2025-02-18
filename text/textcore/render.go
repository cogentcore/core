// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"fmt"
	"image"
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
)

// NeedsLayout indicates that the [Base] needs a new layout pass.
func (ed *Base) NeedsLayout() {
	ed.NeedsRender()
}

// todo: manage scrollbar ourselves!
// func (ed *Base) renderLayout() {
// 	chg := ed.ManageOverflow(3, true)
// 	ed.layoutAllLines()
// 	ed.ConfigScrolls()
// 	if chg {
// 		ed.Frame.NeedsLayout() // required to actually update scrollbar vs not
// 	}
// }

func (ed *Base) RenderWidget() {
	if ed.StartRender() {
		// if ed.needsLayout {
		// 	ed.renderLayout()
		// 	ed.needsLayout = false
		// }
		// if ed.targetSet {
		// 	ed.scrollCursorToTarget()
		// }
		ed.PositionScrolls()
		ed.renderLines()
		if ed.StateIs(states.Focused) {
			ed.startCursor()
		} else {
			ed.stopCursor()
		}
		ed.RenderChildren()
		ed.RenderScrolls()
		ed.EndRender()
	} else {
		ed.stopCursor()
	}
}

// renderBBox is the bounding box for the text render area (ContentBBox)
func (ed *Base) renderBBox() image.Rectangle {
	return ed.Geom.ContentBBox
}

// renderLineStartEnd returns the starting and ending lines to render,
// based on the scroll position. Also returns the starting upper left position
// for rendering the first line.
func (ed *Base) renderLineStartEnd() (stln, edln int, spos math32.Vector2) {
	spos = ed.Geom.Pos.Content
	stln = int(math32.Floor(ed.scrollPos))
	spos.Y += (float32(stln) - ed.scrollPos) * ed.charSize.Y
	edln = min(ed.linesSize.Y, stln+ed.visSize.Y+1)
	return
}

// posIsVisible returns true if given position is visible,
// in terms of the vertical lines in view.
func (ed *Base) posIsVisible(pos textpos.Pos) bool {
	if ed.Lines == nil {
		return false
	}
	vpos := ed.Lines.PosToView(ed.viewId, pos)
	sp := int(math32.Floor(ed.scrollPos))
	return vpos.Line >= sp && vpos.Line < sp+ed.visSize.Y
}

// renderLines renders the visible lines and line numbers.
func (ed *Base) renderLines() {
	ed.RenderStandardBox()
	bb := ed.renderBBox()
	stln, edln, spos := ed.renderLineStartEnd()
	pc := &ed.Scene.Painter
	pc.PushContext(nil, render.NewBoundsRect(bb, sides.NewFloats()))
	sh := ed.Scene.TextShaper

	if ed.hasLineNumbers {
		ed.renderLineNumbersBox()
		li := 0
		for ln := stln; ln <= edln; ln++ {
			sp := ed.Lines.PosFromView(ed.viewId, textpos.Pos{Line: ln})
			if sp.Char == 0 { // this means it is the start of a source line
				ed.renderLineNumber(spos, li, sp.Line)
			}
			li++
		}
	}

	ed.renderDepthBackground(spos, stln, edln)
	if ed.hasLineNumbers {
		tbb := bb
		tbb.Min.X += int(ed.lineNumberPixels())
		pc.PushContext(nil, render.NewBoundsRect(tbb, sides.NewFloats()))
	}

	buf := ed.Lines
	ctx := &core.AppearanceSettings.Text
	ts := ed.Lines.Settings.TabSize
	rpos := spos
	rpos.X += ed.lineNumberPixels()
	sz := ed.charSize
	sz.X *= float32(ed.linesSize.X)
	vsel := buf.RegionToView(ed.viewId, ed.SelectRegion)
	buf.Lock()
	for ln := stln; ln < edln; ln++ {
		tx := buf.ViewMarkupLine(ed.viewId, ln)
		vlr := buf.ViewLineRegionLocked(ed.viewId, ln)
		vseli := vlr.Intersect(vsel, ed.linesSize.X)
		indent := 0
		for si := range tx { // tabs encoded as single chars at start
			sn, rn := rich.SpanLen(tx[si])
			if rn == 1 && tx[si][sn] == '\t' {
				lpos := rpos
				ic := float32(ts*indent) * ed.charSize.X
				lpos.X += ic
				lsz := sz
				lsz.X -= ic
				lns := sh.WrapLines(tx[si:si+1], &ed.Styles.Font, &ed.Styles.Text, ctx, lsz)
				pc.TextLines(lns, lpos)
				indent++
			} else {
				break
			}
		}
		rtx := tx[indent:]
		lpos := rpos
		ic := float32(ts*indent) * ed.charSize.X
		lpos.X += ic
		lsz := sz
		lsz.X -= ic
		lns := sh.WrapLines(rtx, &ed.Styles.Font, &ed.Styles.Text, ctx, lsz)
		if !vseli.IsNil() {
			lns.SelectRegion(textpos.Range{Start: vseli.Start.Char, End: vseli.End.Char})
		}
		pc.TextLines(lns, lpos)
		rpos.Y += ed.charSize.Y
	}
	buf.Unlock()
	if ed.hasLineNumbers {
		pc.PopContext()
	}
	pc.PopContext()
}

// renderLineNumbersBox renders the background for the line numbers in the LineNumberColor
func (ed *Base) renderLineNumbersBox() {
	if !ed.hasLineNumbers {
		return
	}
	pc := &ed.Scene.Painter
	bb := ed.renderBBox()
	spos := math32.FromPoint(bb.Min)
	epos := math32.FromPoint(bb.Max)
	epos.X = spos.X + ed.lineNumberPixels()

	sz := epos.Sub(spos)
	pc.Fill.Color = ed.LineNumberColor
	pc.RoundedRectangleSides(spos.X, spos.Y, sz.X, sz.Y, ed.Styles.Border.Radius.Dots())
	pc.PathDone()
}

// renderLineNumber renders given line number at given li index.
func (ed *Base) renderLineNumber(pos math32.Vector2, li, ln int) {
	if !ed.hasLineNumbers || ed.Lines == nil {
		return
	}
	pos.Y += float32(li) * ed.charSize.Y

	sty := &ed.Styles
	pc := &ed.Scene.Painter
	sh := ed.Scene.TextShaper
	fst := sty.Font

	fst.Background = nil
	lfmt := fmt.Sprintf("%d", ed.lineNumberDigits)
	lfmt = "%" + lfmt + "d"
	lnstr := fmt.Sprintf(lfmt, ln+1)

	if ed.CursorPos.Line == ln {
		fst.SetFillColor(colors.ToUniform(colors.Scheme.Primary.Base))
		fst.Weight = rich.Bold
	} else {
		fst.SetFillColor(colors.ToUniform(colors.Scheme.OnSurfaceVariant))
	}
	sz := ed.charSize
	sz.X *= float32(ed.lineNumberOffset)
	tx := rich.NewText(&fst, []rune(lnstr))
	lns := sh.WrapLines(tx, &fst, &sty.Text, &core.AppearanceSettings.Text, sz)
	pc.TextLines(lns, pos)

	// render circle
	lineColor, has := ed.Lines.LineColor(ln)
	if has {
		pos.X += float32(ed.lineNumberDigits) * ed.charSize.X
		r := 0.5 * ed.charSize.X
		center := pos.AddScalar(r)

		// cut radius in half so that it doesn't look too big
		r /= 2

		pc.Fill.Color = lineColor
		pc.Circle(center.X, center.Y, r)
		pc.PathDone()
	}
}

func (ed *Base) lineNumberPixels() float32 {
	return float32(ed.lineNumberOffset) * ed.charSize.X
}

// TODO: make viewDepthColors HCT based?

// viewDepthColors are changes in color values from default background for different
// depths. For dark mode, these are increments, for light mode they are decrements.
var viewDepthColors = []color.RGBA{
	{0, 0, 0, 0},
	{5, 5, 0, 0},
	{15, 15, 0, 0},
	{5, 15, 0, 0},
	{0, 15, 5, 0},
	{0, 15, 15, 0},
	{0, 5, 15, 0},
	{5, 0, 15, 0},
	{5, 0, 5, 0},
}

// renderDepthBackground renders the depth background color.
func (ed *Base) renderDepthBackground(pos math32.Vector2, stln, edln int) {
	if !ed.Lines.Settings.DepthColor || ed.IsDisabled() || !ed.StateIs(states.Focused) {
		return
	}
	pos.X += ed.lineNumberPixels()
	buf := ed.Lines
	bbmax := float32(ed.Geom.ContentBBox.Max.X)
	pc := &ed.Scene.Painter
	sty := &ed.Styles
	isDark := matcolor.SchemeIsDark
	nclrs := len(viewDepthColors)
	for ln := stln; ln <= edln; ln++ {
		sp := ed.Lines.PosFromView(ed.viewId, textpos.Pos{Line: ln})
		depth := buf.LineLexDepth(sp.Line)
		if depth == 0 {
			continue
		}
		var vdc color.RGBA
		if isDark { // reverse order too
			vdc = viewDepthColors[nclrs-1-depth%nclrs]
		} else {
			vdc = viewDepthColors[depth%nclrs]
		}
		bg := gradient.Apply(sty.Background, func(c color.Color) color.Color {
			if isDark { // reverse order too
				return colors.Add(c, vdc)
			}
			return colors.Sub(c, vdc)
		})
		spos := pos
		spos.Y += float32(ln-stln) * ed.charSize.Y
		epos := spos
		epos.Y += ed.charSize.Y
		epos.X = bbmax
		pc.FillBox(spos, epos.Sub(spos), bg)
	}
}

// renderHighlights renders the highlight regions as a
// highlighted background color.
func (ed *Base) renderHighlights(stln, edln int) {
	// for _, reg := range ed.Highlights {
	// 	reg := ed.Lines.AdjustRegion(reg)
	// 	if reg.IsNil() || (stln >= 0 && (reg.Start.Line > edln || reg.End.Line < stln)) {
	// 		continue
	// 	}
	// 	ed.renderRegionBox(reg, ed.HighlightColor)
	// }
}

// renderScopelights renders a highlight background color for regions
// in the Scopelights list.
func (ed *Base) renderScopelights(stln, edln int) {
	// for _, reg := range ed.scopelights {
	// 	reg := ed.Lines.AdjustRegion(reg)
	// 	if reg.IsNil() || (stln >= 0 && (reg.Start.Line > edln || reg.End.Line < stln)) {
	// 		continue
	// 	}
	// 	ed.renderRegionBox(reg, ed.HighlightColor)
	// }
}

// PixelToCursor finds the cursor position that corresponds to the given pixel
// location (e.g., from mouse click), in scene-relative coordinates.
func (ed *Base) PixelToCursor(pt image.Point) textpos.Pos {
	stln, _, spos := ed.renderLineStartEnd()
	spos.X += ed.lineNumberPixels()
	ptf := math32.FromPoint(pt)
	cp := ptf.Sub(spos).Div(ed.charSize)
	vpos := textpos.Pos{Line: stln + int(math32.Floor(cp.Y)), Char: int(math32.Round(cp.X))}
	tx := ed.Lines.ViewMarkupLine(ed.viewId, vpos.Line)
	indent := 0
	for si := range tx { // tabs encoded as single chars at start
		sn, rn := rich.SpanLen(tx[si])
		if rn == 1 && tx[si][sn] == '\t' {
			indent++
		} else {
			break
		}
	}
	if indent == 0 {
		return ed.Lines.PosFromView(ed.viewId, vpos)
	}
	ts := ed.Lines.Settings.TabSize
	ic := indent * ts
	if vpos.Char >= ic {
		vpos.Char -= (ic - indent)
		return ed.Lines.PosFromView(ed.viewId, vpos)
	}
	ip := vpos.Char / ts
	vpos.Char = ip
	return ed.Lines.PosFromView(ed.viewId, vpos)
}

// charStartPos returns the starting (top left) render coords for the
// given source text position.
func (ed *Base) charStartPos(pos textpos.Pos) math32.Vector2 {
	if ed.Lines == nil {
		return math32.Vector2{}
	}
	vpos := ed.Lines.PosToView(ed.viewId, pos)
	spos := ed.Geom.Pos.Content
	spos.X += ed.lineNumberPixels() - ed.Geom.Scroll.X
	spos.Y += (float32(vpos.Line) - ed.scrollPos) * ed.charSize.Y
	tx := ed.Lines.ViewMarkupLine(ed.viewId, vpos.Line)
	ts := ed.Lines.Settings.TabSize
	indent := 0
	for si := range tx { // tabs encoded as single chars at start
		sn, rn := rich.SpanLen(tx[si])
		if rn == 1 && tx[si][sn] == '\t' {
			if vpos.Char == si {
				spos.X += float32(indent*ts) * ed.charSize.X
				return spos
			}
			indent++
		} else {
			break
		}
	}
	spos.X += float32(indent*ts+(vpos.Char-indent)) * ed.charSize.X
	return spos
}
