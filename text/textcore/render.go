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
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
)

func (ed *Base) reLayout() {
	if ed.Lines == nil {
		return
	}
	lns := ed.Lines.ViewLines(ed.viewId)
	if lns == ed.linesSize.Y {
		return
	}
	ed.layoutAllLines()
	chg := ed.ManageOverflow(1, true)
	// fmt.Println(chg)
	if chg {
		ed.NeedsLayout()
		if !ed.HasScroll[math32.Y] {
			ed.scrollPos = 0
		}
	}
}

func (ed *Base) RenderWidget() {
	if ed.StartRender() {
		ed.reLayout()
		if ed.targetSet {
			ed.scrollCursorToTarget()
		}
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
	return vpos.Line >= sp && vpos.Line < sp+ed.visSize.Y
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
	for ln := stln; ln <= edln; ln++ {
		ed.renderLine(ln, rpos, vsel, hlts)
		rpos.Y += ed.charSize.Y
	}
	buf.Unlock()
	if ed.hasLineNumbers {
		pc.PopContext()
	}
	pc.PopContext()
}

// renderLine renders given line, dealing with tab stops etc
func (ed *Base) renderLine(ln int, rpos math32.Vector2, vsel textpos.Region, hlts []textpos.Region) {
	buf := ed.Lines
	sh := ed.Scene.TextShaper
	pc := &ed.Scene.Painter
	sz := ed.charSize
	sz.X *= float32(ed.linesSize.X)
	vlr := buf.ViewLineRegionLocked(ed.viewId, ln)
	vseli := vlr.Intersect(vsel, ed.linesSize.X)
	tx := buf.ViewMarkupLine(ed.viewId, ln)
	ctx := &rich.DefaultSettings
	ts := ed.Lines.Settings.TabSize
	indent := 0

	rendSpan := func(stx rich.Text, pos math32.Vector2, ssz math32.Vector2, coff int) {
		lns := sh.WrapLines(stx, &ed.Styles.Font, &ed.Styles.Text, ctx, ssz)
		lns.SetGlyphXAdvance(math32.ToFixed(ed.charSize.X))
		if !vseli.IsNil() {
			lns.SelectRegion(textpos.Range{Start: vseli.Start.Char - coff, End: vseli.End.Char - coff})
		}
		for _, hlrg := range hlts {
			hlsi := vlr.Intersect(hlrg, ed.linesSize.X)
			if !hlsi.IsNil() {
				lns.HighlightRegion(textpos.Range{Start: hlsi.Start.Char - coff, End: hlsi.End.Char - coff})
			}
		}
		pc.TextLines(lns, pos)
	}

	for si := range tx { // tabs encoded as single chars at start
		sn, rn := rich.SpanLen(tx[si])
		if rn == 1 && tx[si][sn] == '\t' {
			lpos := rpos
			ic := float32(ts*indent) * ed.charSize.X
			lpos.X += ic
			lsz := sz
			lsz.X -= ic
			rendSpan(tx[si:si+1], lpos, lsz, indent)
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
	hasTab := false
	for si := range rtx {
		sn, rn := rich.SpanLen(tx[si])
		if rn > 0 && tx[si][sn] == '\t' {
			hasTab = true
			break
		}
	}
	if !hasTab {
		rendSpan(rtx, lpos, lsz, indent)
		return
	}
	coff := indent
	cc := ts * indent
	scc := cc
	for si := range rtx {
		sn, rn := rich.SpanLen(rtx[si])
		if rn == 0 {
			continue
		}
		spos := lpos
		spos.X += float32(cc-scc) * ed.charSize.X
		if rtx[si][sn] != '\t' {
			ssz := ed.charSize.Mul(math32.Vec2(float32(rn), 1))
			rendSpan(rtx[si:si+1], spos, ssz, coff)
			cc += rn
			coff += rn
			continue
		}
		for range rn {
			tcc := ((cc / 8) + 1) * 8
			spos.X += float32(tcc-cc) * ed.charSize.X
			cc = tcc
			rendSpan(rtx[si:si+1], spos, ed.charSize, coff)
			coff++
		}
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

	fst.SetBackground(nil)
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
	lns := sh.WrapLines(tx, &fst, &sty.Text, &rich.DefaultSettings, sz)
	pc.TextLines(lns, pos)

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
		pc.PathDone()
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
	vln := stln + int(math32.Floor(cp.Y))
	vpos := textpos.Pos{Line: vln, Char: 0}
	srcp := ed.Lines.PosFromView(ed.viewId, vpos)
	stp := ed.charStartPos(srcp)
	if ptf.X < stp.X {
		return srcp
	}
	scc := srcp.Char
	hc := 0.5 * ed.charSize.X
	vll := ed.Lines.ViewLineLen(ed.viewId, vln)
	for cc := range vll {
		srcp.Char = scc + cc
		edp := ed.charStartPos(textpos.Pos{Line: srcp.Line, Char: scc + cc + 1})
		if ptf.X >= stp.X-hc && ptf.X < edp.X-hc {
			return srcp
		}
		stp = edp
	}
	srcp.Char = scc + vll
	return srcp
}

// charStartPos returns the starting (top left) render coords for the
// given source text position.
func (ed *Base) charStartPos(pos textpos.Pos) math32.Vector2 {
	if ed.Lines == nil {
		return math32.Vector2{}
	}
	vpos := ed.Lines.PosToView(ed.viewId, pos)
	spos := ed.Geom.Pos.Content
	spos.X += ed.LineNumberPixels() - ed.Geom.Scroll.X
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
	rtx := tx[indent:]
	lpos := spos
	lpos.X += float32(ts*indent) * ed.charSize.X
	coff := indent
	cc := ts * indent
	scc := cc
	for si := range rtx {
		sn, rn := rich.SpanLen(rtx[si])
		if rn == 0 {
			continue
		}
		spos := lpos
		spos.X += float32(cc-scc) * ed.charSize.X
		if rtx[si][sn] != '\t' {
			rc := vpos.Char - coff
			if rc >= 0 && rc < rn {
				spos.X += float32(rc) * ed.charSize.X
				return spos
			}
			cc += rn
			coff += rn
			continue
		}
		for ri := range rn {
			if ri == vpos.Char-coff {
				return spos
			}
			tcc := ((cc / 8) + 1) * 8
			cc = tcc
			coff++
		}
	}
	spos = lpos
	spos.X += float32(cc-scc) * ed.charSize.X
	return spos
}
