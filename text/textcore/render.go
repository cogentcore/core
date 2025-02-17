// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"fmt"
	"image"
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
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

func (ed *Base) renderBBox() image.Rectangle {
	return ed.Geom.ContentBBox
}

// charStartPos returns the starting (top left) render coords for the
// given source text position.
func (ed *Base) charStartPos(pos textpos.Pos) math32.Vector2 {
	if ed.Lines == nil {
		return math32.Vector2{}
	}
	vpos := ed.Lines.PosToView(ed.viewId, pos)
	spos := ed.Geom.Pos.Content
	spos.X += ed.lineNumberPixels() + float32(vpos.Char)*ed.charSize.X
	spos.Y += (float32(vpos.Line) - ed.scrollPos) * ed.charSize.Y
	return spos
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
	pos := ed.Geom.Pos.Content
	stln := int(math32.Floor(ed.scrollPos))
	edln := min(ed.linesSize.Y, stln+ed.visSize.Y+1)
	// fmt.Println("render lines size:", ed.linesSize.Y, edln, "stln:", stln, "bb:", bb, "pos:", pos)

	pc := &ed.Scene.Painter
	pc.PushContext(nil, render.NewBoundsRect(bb, sides.NewFloats()))
	sh := ed.Scene.TextShaper

	if ed.hasLineNumbers {
		ed.renderLineNumbersBox()
		li := 0
		for ln := stln; ln <= edln; ln++ {
			ed.renderLineNumber(li, ln, false) // don't re-render std fill boxes
			li++
		}
	}

	ed.renderDepthBackground(stln, edln)
	// ed.renderHighlights(stln, edln)
	// ed.renderScopelights(stln, edln)
	// ed.renderSelect()
	if ed.hasLineNumbers {
		tbb := bb
		tbb.Min.X += int(ed.lineNumberPixels())
		pc.PushContext(nil, render.NewBoundsRect(tbb, sides.NewFloats()))
	}

	buf := ed.Lines
	buf.Lock()
	rpos := pos
	rpos.X += ed.lineNumberPixels()
	sz := ed.charSize
	sz.X *= float32(ed.linesSize.X)
	for ln := stln; ln < edln; ln++ {
		tx := buf.ViewMarkupLine(ed.viewId, ln)
		lns := sh.WrapLines(tx, &ed.Styles.Font, &ed.Styles.Text, &core.AppearanceSettings.Text, sz)
		pc.TextLines(lns, rpos)
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

// renderLineNumber renders given line number; called within context of other render.
// if defFill is true, it fills box color for default background color (use false for
// batch mode).
func (ed *Base) renderLineNumber(li, ln int, defFill bool) {
	if !ed.hasLineNumbers || ed.Lines == nil {
		return
	}
	bb := ed.renderBBox()
	spos := math32.FromPoint(bb.Min)
	spos.Y += float32(li) * ed.charSize.Y

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
	pc.TextLines(lns, spos)

	// render circle
	lineColor, has := ed.Lines.LineColor(ln)
	if has {
		spos.X += float32(ed.lineNumberDigits) * ed.charSize.X
		r := 0.5 * ed.charSize.X
		center := spos.AddScalar(r)

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
func (ed *Base) renderDepthBackground(stln, edln int) {
	// if ed.Lines == nil {
	// 	return
	// }
	// if !ed.Lines.Options.DepthColor || ed.IsDisabled() || !ed.StateIs(states.Focused) {
	// 	return
	// }
	// buf := ed.Lines
	//
	// bb := ed.renderBBox()
	// bln := buf.NumLines()
	// sty := &ed.Styles
	// isDark := matcolor.SchemeIsDark
	// nclrs := len(viewDepthColors)
	// lstdp := 0
	// for ln := stln; ln <= edln; ln++ {
	// 	lst := ed.charStartPos(textpos.Pos{Ln: ln}).Y // note: charstart pos includes descent
	// 	led := lst + math32.Max(ed.renders[ln].BBox.Size().Y, ed.lineHeight)
	// 	if int(math32.Ceil(led)) < bb.Min.Y {
	// 		continue
	// 	}
	// 	if int(math32.Floor(lst)) > bb.Max.Y {
	// 		continue
	// 	}
	// 	if ln >= bln { // may be out of sync
	// 		continue
	// 	}
	// 	ht := buf.HiTags(ln)
	// 	lsted := 0
	// 	for ti := range ht {
	// 		lx := &ht[ti]
	// 		if lx.Token.Depth > 0 {
	// 			var vdc color.RGBA
	// 			if isDark { // reverse order too
	// 				vdc = viewDepthColors[nclrs-1-lx.Token.Depth%nclrs]
	// 			} else {
	// 				vdc = viewDepthColors[lx.Token.Depth%nclrs]
	// 			}
	// 			bg := gradient.Apply(sty.Background, func(c color.Color) color.Color {
	// 				if isDark { // reverse order too
	// 					return colors.Add(c, vdc)
	// 				}
	// 				return colors.Sub(c, vdc)
	// 			})
	//
	// 			st := min(lsted, lx.St)
	// 			reg := lines.Region{Start: textpos.Pos{Ln: ln, Ch: st}, End: textpos.Pos{Ln: ln, Ch: lx.Ed}}
	// 			lsted = lx.Ed
	// 			lstdp = lx.Token.Depth
	// 			ed.renderRegionBoxStyle(reg, sty, bg, true) // full width alway
	// 		}
	// 	}
	// 	if lstdp > 0 {
	// 		ed.renderRegionToEnd(textpos.Pos{Ln: ln, Ch: lsted}, sty, sty.Background)
	// 	}
	// }
}

// todo: select and highlights handled by lines shaped directly.

// renderSelect renders the selection region as a selected background color.
func (ed *Base) renderSelect() {
	// if !ed.HasSelection() {
	// 	return
	// }
	// ed.renderRegionBox(ed.SelectRegion, ed.SelectColor)
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

// renderRegionBox renders a region in background according to given background
func (ed *Base) renderRegionBox(reg textpos.Region, bg image.Image) {
	// ed.renderRegionBoxStyle(reg, &ed.Styles, bg, false)
}

// renderRegionBoxStyle renders a region in given style and background
func (ed *Base) renderRegionBoxStyle(reg textpos.Region, sty *styles.Style, bg image.Image, fullWidth bool) {
	// st := reg.Start
	// end := reg.End
	// spos := ed.charStartPosVisible(st)
	// epos := ed.charStartPosVisible(end)
	// epos.Y += ed.lineHeight
	// bb := ed.renderBBox()
	// stx := math32.Ceil(float32(bb.Min.X) + ed.LineNumberOffset)
	// if int(math32.Ceil(epos.Y)) < bb.Min.Y || int(math32.Floor(spos.Y)) > bb.Max.Y {
	// 	return
	// }
	// ex := float32(bb.Max.X)
	// if fullWidth {
	// 	epos.X = ex
	// }
	//
	// pc := &ed.Scene.Painter
	// stsi, _, _ := ed.wrappedLineNumber(st)
	// edsi, _, _ := ed.wrappedLineNumber(end)
	// if st.Line == end.Line && stsi == edsi {
	// 	pc.FillBox(spos, epos.Sub(spos), bg) // same line, done
	// 	return
	// }
	// // on diff lines: fill to end of stln
	// seb := spos
	// seb.Y += ed.lineHeight
	// seb.X = ex
	// pc.FillBox(spos, seb.Sub(spos), bg)
	// sfb := seb
	// sfb.X = stx
	// if sfb.Y < epos.Y { // has some full box
	// 	efb := epos
	// 	efb.Y -= ed.lineHeight
	// 	efb.X = ex
	// 	pc.FillBox(sfb, efb.Sub(sfb), bg)
	// }
	// sed := epos
	// sed.Y -= ed.lineHeight
	// sed.X = stx
	// pc.FillBox(sed, epos.Sub(sed), bg)
}

// renderRegionToEnd renders a region in given style and background, to end of line from start
func (ed *Base) renderRegionToEnd(st textpos.Pos, sty *styles.Style, bg image.Image) {
	// spos := ed.charStartPosVisible(st)
	// epos := spos
	// epos.Y += ed.lineHeight
	// vsz := epos.Sub(spos)
	// if vsz.X <= 0 || vsz.Y <= 0 {
	// 	return
	// }
	// pc := &ed.Scene.Painter
	// pc.FillBox(spos, epos.Sub(spos), bg) // same line, done
}

// PixelToCursor finds the cursor position that corresponds to the given pixel
// location (e.g., from mouse click) which has had ScBBox.Min subtracted from
// it (i.e, relative to upper left of text area)
func (ed *Base) PixelToCursor(pt image.Point) textpos.Pos {
	return textpos.Pos{}
	// if ed.NumLines == 0 {
	// 	return textpos.PosZero
	// }
	// bb := ed.renderBBox()
	// sty := &ed.Styles
	// yoff := float32(bb.Min.Y)
	// xoff := float32(bb.Min.X)
	// stln := ed.firstVisibleLine(0)
	// cln := stln
	// fls := ed.charStartPos(textpos.Pos{Ln: stln}).Y - yoff
	// if pt.Y < int(math32.Floor(fls)) {
	// 	cln = stln
	// } else if pt.Y > bb.Max.Y {
	// 	cln = ed.NumLines - 1
	// } else {
	// 	got := false
	// 	for ln := stln; ln < ed.NumLines; ln++ {
	// 		ls := ed.charStartPos(textpos.Pos{Ln: ln}).Y - yoff
	// 		es := ls
	// 		es += math32.Max(ed.renders[ln].BBox.Size().Y, ed.lineHeight)
	// 		if pt.Y >= int(math32.Floor(ls)) && pt.Y < int(math32.Ceil(es)) {
	// 			got = true
	// 			cln = ln
	// 			break
	// 		}
	// 	}
	// 	if !got {
	// 		cln = ed.NumLines - 1
	// 	}
	// }
	// // fmt.Printf("cln: %v  pt: %v\n", cln, pt)
	// if cln >= len(ed.renders) {
	// 	return textpos.Pos{Ln: cln, Ch: 0}
	// }
	// lnsz := ed.Lines.LineLen(cln)
	// if lnsz == 0 || sty.Font.Face == nil {
	// 	return textpos.Pos{Ln: cln, Ch: 0}
	// }
	// scrl := ed.Geom.Scroll.Y
	// nolno := float32(pt.X - int(ed.LineNumberOffset))
	// sc := int((nolno + scrl) / sty.Font.Face.Metrics.Ch)
	// sc -= sc / 4
	// sc = max(0, sc)
	// cch := sc
	//
	// lnst := ed.charStartPos(textpos.Pos{Ln: cln})
	// lnst.Y -= yoff
	// lnst.X -= xoff
	// rpt := math32.FromPoint(pt).Sub(lnst)
	//
	// si, ri, ok := ed.renders[cln].PosToRune(rpt)
	// if ok {
	// 	cch, _ := ed.renders[cln].SpanPosToRuneIndex(si, ri)
	// 	return textpos.Pos{Ln: cln, Ch: cch}
	// }
	//
	// return textpos.Pos{Ln: cln, Ch: cch}
}
