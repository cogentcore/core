// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"fmt"
	"image"
	"image/color"
	"slices"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/textpos"
)

func (ed *Base) reLayout() {
	if ed.Lines == nil {
		return
	}
	prevLines := ed.linesSize.Y
	lns := ed.Lines.ViewLines(ed.viewId)
	if lns == prevLines {
		return
	}
	ed.layoutAllLines()
	chg := ed.ManageOverflow(1, true)
	if ed.Styles.Grow.Y == 0 && lns < maxGrowLines || prevLines < maxGrowLines {
		chg = prevLines != ed.linesSize.Y || chg
	}
	if chg {
		// fmt.Println(chg, lns, prevLines, ed.visSize.Y, ed.linesSize.Y)
		ed.NeedsLayout()
		if !ed.HasScroll[math32.Y] {
			ed.scrollPos = 0
		}
	}
}

func (ed *Base) RenderWidget() {
	if ed.Lines != nil {
		ed.Lines.StartDeferredMarkup()
	}
	if ed.StartRender() {
		ed.reLayout()
		if ed.targetSet {
			ed.scrollCursorToTarget()
		}
		if !ed.isScrolling {
			ed.scrollCursorToCenterIfHidden()
		}
		ed.PositionScrolls()
		ed.renderLines()
		ed.RenderChildren()
		ed.RenderScrolls()
		ed.updateCursorPosition()
		ed.EndRender()
	}
}

// renderBBox is the bounding box for the text render area (ContentBBox)
func (ed *Base) renderBBox() image.Rectangle {
	return ed.Geom.ContentBBox
}

// renderLineStartEnd returns the starting and ending (inclusive) lines to render
// based on the scroll position. Also returns the starting upper left position
// for rendering the first line.
func (ed *Base) renderLineStartEnd() (stln, edln int, spos math32.Vector2) {
	spos = ed.Geom.Pos.Content
	stln = int(math32.Floor(ed.scrollPos))
	spos.Y += (float32(stln) - ed.scrollPos) * ed.charSize.Y
	edln = min(ed.linesSize.Y-1, stln+ed.visSize.Y)
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
	return vpos.Line >= sp && vpos.Line <= sp+ed.visSize.Y
}

// renderLines renders the visible lines and line numbers.
func (ed *Base) renderLines() {
	ed.RenderStandardBox()
	if ed.Lines == nil {
		return
	}
	bb := ed.renderBBox()
	stln, edln, spos := ed.renderLineStartEnd()
	pc := &ed.Scene.Painter
	pc.PushContext(nil, render.NewBoundsRect(bb, sides.NewFloats()))

	if ed.hasLineNumbers {
		ed.renderLineNumbersBox()
		li := 0
		lastln := -1
		for ln := stln; ln <= edln; ln++ {
			sp := ed.Lines.PosFromView(ed.viewId, textpos.Pos{Line: ln})
			if sp.Char == 0 && sp.Line != lastln { // Char=0 is start of source line
				// but also get 0 for out-of-range..
				ed.renderLineNumber(spos, li, sp.Line)
				lastln = sp.Line
			}
			li++
		}
	}

	ed.renderDepthBackground(spos, stln, edln)
	ed.renderCodeErrors()
	if ed.hasLineNumbers {
		tbb := bb
		tbb.Min.X += int(ed.LineNumberPixels())
		pc.PushContext(nil, render.NewBoundsRect(tbb, sides.NewFloats()))
	}

	buf := ed.Lines
	rpos := spos
	rpos.X += ed.LineNumberPixels()
	vsel := buf.RegionToView(ed.viewId, ed.SelectRegion)
	rtoview := func(rs []textpos.Region) []textpos.Region {
		if len(rs) == 0 {
			return nil
		}
		hlts := make([]textpos.Region, 0, len(rs))
		for _, reg := range rs {
			reg := ed.Lines.AdjustRegion(reg)
			if !reg.IsNil() {
				hlts = append(hlts, buf.RegionToView(ed.viewId, reg))
			}
		}
		return hlts
	}
	hlts := rtoview(ed.Highlights)
	slts := rtoview(ed.scopelights)
	hlts = append(hlts, slts...)
	buf.Lock()
	li := 0
	for ln := stln; ln <= edln; ln++ {
		ed.renderLine(li, ln, rpos, vsel, hlts)
		rpos.Y += ed.charSize.Y
		li++
	}
	buf.Unlock()
	if ed.hasLineNumbers {
		pc.PopContext()
	}
	pc.PopContext()
}

type renderCache struct {
	// tx is the rich text of the line, including the styles of its spans.
	tx  []rune
	lns *shaped.Lines
}

// renderLine renders given line, dealing with tab stops etc
func (ed *Base) renderLine(li, ln int, rpos math32.Vector2, vsel textpos.Region, hlts []textpos.Region) {
	buf := ed.Lines
	sh := ed.Scene.TextShaper()
	pc := &ed.Scene.Painter
	sz := ed.charSize
	sz.X *= float32(ed.linesSize.X)
	vlr := buf.ViewLineRegionNoLock(ed.viewId, ln)
	vseli := vlr.Intersect(vsel, ed.linesSize.X)
	tx := buf.ViewMarkupLine(ed.viewId, ln)
	ts := ed.Lines.Settings.TabSize
	indent := 0
	sty, tsty := ed.Styles.NewRichText()

	shapeTab := func(stx rich.Text, ssz math32.Vector2) *shaped.Lines {
		if ed.tabRender != nil {
			return ed.tabRender.Clone()
		}
		lns := sh.WrapLines(stx, sty, tsty, ssz)
		ed.tabRender = lns
		return lns
	}
	shapeSpan := func(stx rich.Text, ssz math32.Vector2) *shaped.Lines {
		txt := slices.Concat(stx...) // with the styles, which change with highlighting
		rc := ed.lineRenders[li]
		if rc.lns != nil && slices.Compare(rc.tx, txt) == 0 {
			return rc.lns
		}
		lns := sh.WrapLines(stx, sty, tsty, ssz)
		ed.lineRenders[li] = renderCache{tx: txt, lns: lns}
		return lns
	}

	// The chars of the line are positioned in cells, where wide chars
	// take two cells, and tabs go to tab stops.
	cells := lines.LineCells(tx, ts)
	cellX := func(ci int) float32 {
		return float32(cells[min(ci, len(cells)-1)]) * ed.charSize.X
	}

	// rendSpan renders the given shaped span, starting at char coff,
	// where tabCells is the width in cells of a tab, which is 0 otherwise.
	rendSpan := func(lns *shaped.Lines, pos math32.Vector2, coff, tabCells int) {
		lns.SelectReset()
		lns.HighlightReset()
		if tabCells > 0 {
			lns.SetGlyphXAdvance(math32.ToFixed(float32(tabCells)*ed.charSize.X), nil)
		} else {
			lns.SetGlyphXAdvance(math32.ToFixed(ed.charSize.X), lines.RuneWidth)
		}
		if !vseli.IsNil() {
			lns.SelectRegion(textpos.Range{Start: vseli.Start.Char - coff, End: vseli.End.Char - coff})
		}
		for _, hlrg := range hlts {
			hlsi := vlr.Intersect(hlrg, ed.linesSize.X)
			if !hlsi.IsNil() {
				lns.HighlightRegion(textpos.Range{Start: hlsi.Start.Char - coff, End: hlsi.End.Char - coff})
			}
		}
		pc.DrawText(lns, pos)
	}
	// rendTab renders the tab at char ci.
	rendTab := func(stx rich.Text, ci int) {
		pos := rpos
		pos.X += cellX(ci)
		tsz := sz
		tsz.X -= cellX(ci)
		rendSpan(shapeTab(stx, tsz), pos, ci, cells[ci+1]-cells[ci])
	}

	for si := range tx { // tabs encoded as single chars at start
		sn, rn := rich.SpanLen(tx[si])
		if rn == 1 && tx[si][sn] == '\t' {
			rendTab(tx[si:si+1], indent)
			indent++
		} else {
			break
		}
	}
	rtx := tx[indent:]
	hasTab := false
	for si := range rtx {
		sn, rn := rich.SpanLen(rtx[si])
		if rn > 0 && rtx[si][sn] == '\t' {
			hasTab = true
			break
		}
	}
	if !hasTab {
		lpos := rpos
		lpos.X += cellX(indent)
		lsz := sz
		lsz.X -= cellX(indent)
		rendSpan(shapeSpan(rtx, lsz), lpos, indent, 0)
		return
	}
	coff := indent
	for si := range rtx {
		sn, rn := rich.SpanLen(rtx[si])
		if rn == 0 {
			continue
		}
		if rtx[si][sn] == '\t' {
			for range rn {
				rendTab(rtx[si:si+1], coff)
				coff++
			}
			continue
		}
		spos := rpos
		spos.X += cellX(coff)
		ssz := sz
		ssz.X -= cellX(coff)
		rendSpan(shapeSpan(rtx[si:si+1], ssz), spos, coff, 0)
		coff += rn
	}
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
	epos.X = spos.X + ed.LineNumberPixels()

	sz := epos.Sub(spos)
	pc.Fill.Color = ed.LineNumberColor
	pc.RoundedRectangleSides(spos.X, spos.Y, sz.X, sz.Y, ed.Styles.Border.Radius.Dots())
	pc.Draw()
}

// renderLineNumber renders given line number at given li index.
func (ed *Base) renderLineNumber(pos math32.Vector2, li, ln int) {
	if !ed.hasLineNumbers || ed.Lines == nil {
		return
	}
	pos.Y += float32(li) * ed.charSize.Y

	pc := &ed.Scene.Painter
	sh := ed.Scene.TextShaper()
	sty, tsty := ed.Styles.NewRichText()

	sty.SetBackground(nil)
	lfmt := fmt.Sprintf("%d", ed.lineNumberDigits)
	lfmt = "%" + lfmt + "d"
	lnstr := fmt.Sprintf(lfmt, ln+1)

	if ed.CursorPos.Line == ln {
		sty.SetFillColor(colors.ToUniform(colors.Scheme.Primary.Base))
		sty.Weight = rich.Bold
	} else {
		sty.SetFillColor(colors.ToUniform(colors.Scheme.OnSurfaceVariant))
	}
	sz := ed.charSize
	sz.X *= float32(ed.lineNumberOffset)
	var lns *shaped.Lines
	rc := ed.lineNoRenders[li]
	tx := rich.NewText(sty, []rune(lnstr))
	if rc.lns != nil && slices.Compare(rc.tx, tx[0]) == 0 { // captures styling
		lns = rc.lns
	} else {
		lns = sh.WrapLines(tx, sty, tsty, sz)
		ed.lineNoRenders[li] = renderCache{tx: tx[0], lns: lns}
	}
	pc.DrawText(lns, pos)

	// render circle
	lineColor, has := ed.Lines.LineColor(ln)
	if has {
		pos.X += float32(ed.lineNumberDigits) * ed.charSize.X
		r := 0.7 * ed.charSize.X
		center := pos.AddScalar(r)
		center.Y += 0.3 * ed.charSize.Y
		center.X += 0.3 * ed.charSize.X
		pc.Fill.Color = lineColor
		pc.Circle(center.X, center.Y, r)
		pc.Draw()
	}
}

func (ed *Base) LineNumberPixels() float32 {
	return float32(ed.lineNumberOffset) * ed.charSize.X
}

// TODO: make viewDepthColors HCT based?

// viewDepthColors are changes in color values from default background for different
// depths. For dark mode, these are increments, for light mode they are decrements.
var viewDepthColors = []color.RGBA{
	{0, 0, 0, 0},
	{4, 4, 0, 0},
	{8, 8, 0, 0},
	{4, 8, 0, 0},
	{0, 8, 4, 0},
	{0, 8, 8, 0},
	{0, 4, 8, 0},
	{4, 0, 8, 0},
	{4, 0, 4, 0},
}

// renderDepthBackground renders the depth background color.
func (ed *Base) renderDepthBackground(pos math32.Vector2, stln, edln int) {
	if !ed.Lines.Settings.DepthColor || ed.IsDisabled() || !ed.StateIs(states.Focused) {
		return
	}
	pos.X += ed.LineNumberPixels()
	buf := ed.Lines
	bbmax := float32(ed.Geom.ContentBBox.Max.X)
	pc := &ed.Scene.Painter
	sty := &ed.Styles
	isDark := matcolor.SchemeIsDark
	nclrs := len(viewDepthColors)
	for ln := stln; ln <= edln; ln++ {
		sp := ed.Lines.PosFromView(ed.viewId, textpos.Pos{Line: ln})
		depth := buf.LineLexDepth(sp.Line)
		if depth <= 0 {
			continue
		}
		var vdc color.RGBA
		if isDark { // reverse order too
			vdc = viewDepthColors[(nclrs-1)-(depth%nclrs)]
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

// PixelToCursor finds the cursor position that corresponds to the given pixel
// location (e.g., from mouse click), in widget-relative coordinates.
func (ed *Base) PixelToCursor(pt image.Point) textpos.Pos {
	if ed.Lines == nil {
		return textpos.PosErr
	}
	stln, _, spos := ed.renderLineStartEnd()
	ptf := math32.FromPoint(pt)
	ptf.X += ed.Geom.Pos.Content.X
	ptf.Y -= (spos.Y - ed.Geom.Pos.Content.Y) // fractional bit
	cp := ptf.Div(ed.charSize)
	if cp.Y < 0 {
		return textpos.PosErr
	}
	// The chars of the view line are found in view coordinates, which
	// include any virtual text, which maps to its source position.
	// A point in the second half of a char is after it.
	vln := stln + int(math32.Floor(cp.Y))
	vpos := textpos.Pos{Line: vln, Char: 0}
	stp := ed.viewCharStartPos(vpos)
	if ptf.X < stp.X {
		return ed.Lines.PosFromView(ed.viewId, vpos)
	}
	col := (ptf.X - stp.X) / ed.charSize.X
	cells := ed.viewLineCells(vln)
	vll := ed.Lines.ViewLineLen(ed.viewId, vln)
	for cc := range min(vll, len(cells)-1) {
		if 2*col < float32(cells[cc]+cells[cc+1]) {
			vpos.Char = cc
			return ed.Lines.PosFromView(ed.viewId, vpos)
		}
	}
	vpos.Char = vll
	return ed.Lines.PosFromView(ed.viewId, vpos)
}

// CharStartPos returns the starting (top left) render coords for the
// given source text position, in the same scene-based coordinates the
// editor uses to render text (see [Editor.charStartPos]). Popups such
// as the completion popup anchor themselves relative to this position.
func (ed *Base) CharStartPos(pos textpos.Pos) math32.Vector2 {
	return ed.charStartPos(pos)
}

// charStartPos returns the starting (top left) render coords for the
// given source text position.
func (ed *Base) charStartPos(pos textpos.Pos) math32.Vector2 {
	if ed.Lines == nil {
		return math32.Vector2{}
	}
	return ed.viewCharStartPos(ed.Lines.PosToView(ed.viewId, pos))
}

// viewLineCells returns the cell column of the start of each char
// of the given view line, followed by its end (see [lines.LineCells]).
func (ed *Base) viewLineCells(vln int) []int {
	ed.Lines.Lock() // required by ViewMarkupLine; never called from renderLines
	tx := ed.Lines.ViewMarkupLine(ed.viewId, vln)
	ed.Lines.Unlock()
	return lines.LineCells(tx, ed.Lines.Settings.TabSize)
}

// viewCharStartPos returns the starting (top left) render coords for the
// given view position, in view lines and display chars, which include
// any virtual text. Positions beyond the end of the line are at its end.
func (ed *Base) viewCharStartPos(vpos textpos.Pos) math32.Vector2 {
	if ed.Lines == nil {
		return math32.Vector2{}
	}
	scpos := image.Point{}
	if ed.Scene != nil {
		scpos = ed.Scene.SceneGeom.Pos
	}
	spos := ed.Geom.Pos.Content.Add(math32.FromPoint(scpos))
	spos.X += ed.LineNumberPixels() - ed.Geom.Scroll.X
	spos.Y += (float32(vpos.Line) - ed.scrollPos) * ed.charSize.Y
	cells := ed.viewLineCells(vpos.Line)
	ci := min(max(vpos.Char, 0), len(cells)-1)
	spos.X += float32(cells[ci]) * ed.charSize.X
	return spos
}
