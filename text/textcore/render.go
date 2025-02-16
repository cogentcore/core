// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"fmt"
	"image"
	"image/color"

	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ptext"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/lines"
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
		ed.renderAllLines()
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

// textStyleProperties returns the styling properties for text based on HiStyle Markup
func (ed *Base) textStyleProperties() map[string]any {
	if ed.Buffer == nil {
		return nil
	}
	return ed.Buffer.Highlighter.CSSProperties
}

// renderStartPos is absolute rendering start position from our content pos with scroll
// This can be offscreen (left, up) based on scrolling.
func (ed *Base) renderStartPos() math32.Vector2 {
	return ed.Geom.Pos.Content.Add(ed.Geom.Scroll)
}

// renderBBox is the render region
func (ed *Base) renderBBox() image.Rectangle {
	bb := ed.Geom.ContentBBox
	spc := ed.Styles.BoxSpace().Size().ToPointCeil()
	// bb.Min = bb.Min.Add(spc)
	bb.Max = bb.Max.Sub(spc)
	return bb
}

// charStartPos returns the starting (top left) render coords for the given
// position -- makes no attempt to rationalize that pos (i.e., if not in
// visible range, position will be out of range too)
func (ed *Base) charStartPos(pos textpos.Pos) math32.Vector2 {
	spos := ed.renderStartPos()
	// todo:
	// spos.X += ed.LineNumberOffset
	// if pos.Line >= len(ed.offsets) {
	// 	if len(ed.offsets) > 0 {
	// 		pos.Line = len(ed.offsets) - 1
	// 	} else {
	// 		return spos
	// 	}
	// } else {
	// 	spos.Y += ed.offsets[pos.Line]
	// }
	// if pos.Line >= len(ed.renders) {
	// 	return spos
	// }
	// rp := &ed.renders[pos.Line]
	// if len(rp.Spans) > 0 {
	// 	// note: Y from rune pos is baseline
	// 	rrp, _, _, _ := ed.renders[pos.Line].RuneRelPos(pos.Ch)
	// 	spos.X += rrp.X
	// 	spos.Y += rrp.Y - ed.renders[pos.Line].Spans[0].RelPos.Y // relative
	// }
	return spos
}

// charStartPosVisible returns the starting pos for given position
// that is currently visible, based on bounding boxes.
func (ed *Base) charStartPosVisible(pos textpos.Pos) math32.Vector2 {
	spos := ed.charStartPos(pos)
	// todo:
	// bb := ed.renderBBox()
	// bbmin := math32.FromPoint(bb.Min)
	// bbmin.X += ed.LineNumberOffset
	// bbmax := math32.FromPoint(bb.Max)
	// spos.SetMax(bbmin)
	// spos.SetMin(bbmax)
	return spos
}

// charEndPos returns the ending (bottom right) render coords for the given
// position -- makes no attempt to rationalize that pos (i.e., if not in
// visible range, position will be out of range too)
func (ed *Base) charEndPos(pos textpos.Pos) math32.Vector2 {
	spos := ed.renderStartPos()
	// todo:
	// pos.Line = min(pos.Line, ed.NumLines-1)
	// if pos.Line < 0 {
	// 	spos.Y += float32(ed.linesSize.Y)
	// 	spos.X += ed.LineNumberOffset
	// 	return spos
	// }
	// if pos.Line >= len(ed.offsets) {
	// 	spos.Y += float32(ed.linesSize.Y)
	// 	spos.X += ed.LineNumberOffset
	// 	return spos
	// }
	// spos.Y += ed.offsets[pos.Line]
	// spos.X += ed.LineNumberOffset
	// r := ed.renders[pos.Line]
	// if len(r.Spans) > 0 {
	// 	// note: Y from rune pos is baseline
	// 	rrp, _, _, _ := r.RuneEndPos(pos.Ch)
	// 	spos.X += rrp.X
	// 	spos.Y += rrp.Y - r.Spans[0].RelPos.Y // relative
	// }
	// spos.Y += ed.lineHeight // end of that line
	return spos
}

// lineBBox returns the bounding box for given line
func (ed *Base) lineBBox(ln int) math32.Box2 {
	tbb := ed.renderBBox()
	// var bb math32.Box2
	// bb.Min = ed.renderStartPos()
	// bb.Min.X += ed.LineNumberOffset
	// bb.Max = bb.Min
	// bb.Max.Y += ed.lineHeight
	// bb.Max.X = float32(tbb.Max.X)
	// if ln >= len(ed.offsets) {
	// 	if len(ed.offsets) > 0 {
	// 		ln = len(ed.offsets) - 1
	// 	} else {
	// 		return bb
	// 	}
	// } else {
	// 	bb.Min.Y += ed.offsets[ln]
	// 	bb.Max.Y += ed.offsets[ln]
	// }
	// if ln >= len(ed.renders) {
	// 	return bb
	// }
	// rp := &ed.renders[ln]
	// bb.Max = bb.Min.Add(rp.BBox.Size())
	// return bb
	return tbb
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
	// if ed.Buffer == nil {
	// 	return
	// }
	// if !ed.Buffer.Options.DepthColor || ed.IsDisabled() || !ed.StateIs(states.Focused) {
	// 	return
	// }
	// buf := ed.Buffer
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
	// 	reg := ed.Buffer.AdjustRegion(reg)
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
	// 	reg := ed.Buffer.AdjustRegion(reg)
	// 	if reg.IsNil() || (stln >= 0 && (reg.Start.Line > edln || reg.End.Line < stln)) {
	// 		continue
	// 	}
	// 	ed.renderRegionBox(reg, ed.HighlightColor)
	// }
}

// renderRegionBox renders a region in background according to given background
func (ed *Base) renderRegionBox(reg lines.Region, bg image.Image) {
	// ed.renderRegionBoxStyle(reg, &ed.Styles, bg, false)
}

// renderRegionBoxStyle renders a region in given style and background
func (ed *Base) renderRegionBoxStyle(reg lines.Region, sty *styles.Style, bg image.Image, fullWidth bool) {
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

// renderAllLines displays all the visible lines on the screen,
// after StartRender has already been called.
func (ed *Base) renderAllLines() {
	ed.RenderStandardBox()
	pc := &ed.Scene.Painter
	bb := ed.renderBBox()
	pos := ed.renderStartPos()
	stln := -1
	edln := -1
	for ln := 0; ln < ed.NumLines; ln++ {
		if ln >= len(ed.offsets) || ln >= len(ed.renders) {
			break
		}
		lst := pos.Y + ed.offsets[ln]
		led := lst + math32.Max(ed.renders[ln].BBox.Size().Y, ed.lineHeight)
		if int(math32.Ceil(led)) < bb.Min.Y {
			continue
		}
		if int(math32.Floor(lst)) > bb.Max.Y {
			continue
		}
		if stln < 0 {
			stln = ln
		}
		edln = ln
	}

	if stln < 0 || edln < 0 { // shouldn't happen.
		return
	}
	pc.PushContext(nil, render.NewBoundsRect(bb, sides.NewFloats()))

	if ed.hasLineNumbers {
		ed.renderLineNumbersBoxAll()
		nln := 1 + edln - stln
		ed.lineNumberRenders = slicesx.SetLength(ed.lineNumberRenders, nln)
		li := 0
		for ln := stln; ln <= edln; ln++ {
			ed.renderLineNumber(li, ln, false) // don't re-render std fill boxes
			li++
		}
	}

	ed.renderDepthBackground(stln, edln)
	ed.renderHighlights(stln, edln)
	ed.renderScopelights(stln, edln)
	ed.renderSelect()
	if ed.hasLineNumbers {
		tbb := bb
		tbb.Min.X += int(ed.LineNumberOffset)
		pc.PushContext(nil, render.NewBoundsRect(tbb, sides.NewFloats()))
	}
	for ln := stln; ln <= edln; ln++ {
		lst := pos.Y + ed.offsets[ln]
		lp := pos
		lp.Y = lst
		lp.X += ed.LineNumberOffset
		if lp.Y+ed.fontAscent > float32(bb.Max.Y) {
			break
		}
		pc.Text(&ed.renders[ln], lp) // not top pos; already has baseline offset
	}
	if ed.hasLineNumbers {
		pc.PopContext()
	}
	pc.PopContext()
}

// renderLineNumbersBoxAll renders the background for the line numbers in the LineNumberColor
func (ed *Base) renderLineNumbersBoxAll() {
	if !ed.hasLineNumbers {
		return
	}
	pc := &ed.Scene.Painter
	bb := ed.renderBBox()
	spos := math32.FromPoint(bb.Min)
	epos := math32.FromPoint(bb.Max)
	epos.X = spos.X + ed.LineNumberOffset

	sz := epos.Sub(spos)
	pc.Fill.Color = ed.LineNumberColor
	pc.RoundedRectangleSides(spos.X, spos.Y, sz.X, sz.Y, ed.Styles.Border.Radius.Dots())
	pc.PathDone()
}

// renderLineNumber renders given line number; called within context of other render.
// if defFill is true, it fills box color for default background color (use false for
// batch mode).
func (ed *Base) renderLineNumber(li, ln int, defFill bool) {
	if !ed.hasLineNumbers || ed.Buffer == nil {
		return
	}
	bb := ed.renderBBox()
	tpos := math32.Vector2{
		X: float32(bb.Min.X), // + spc.Pos().X
		Y: ed.charEndPos(textpos.Pos{Ln: ln}).Y - ed.fontDescent,
	}
	if tpos.Y > float32(bb.Max.Y) {
		return
	}

	sc := ed.Scene
	sty := &ed.Styles
	fst := sty.FontRender()
	pc := &sc.Painter

	fst.Background = nil
	lfmt := fmt.Sprintf("%d", ed.lineNumberDigits)
	lfmt = "%" + lfmt + "d"
	lnstr := fmt.Sprintf(lfmt, ln+1)

	if ed.CursorPos.Line == ln {
		fst.Color = colors.Scheme.Primary.Base
		fst.Weight = styles.WeightBold
		// need to open with new weight
		fst.Font = ptext.OpenFont(fst, &ed.Styles.UnitContext)
	} else {
		fst.Color = colors.Scheme.OnSurfaceVariant
	}
	lnr := &ed.lineNumberRenders[li]
	lnr.SetString(lnstr, fst, &sty.UnitContext, &sty.Text, true, 0, 0)

	pc.Text(lnr, tpos)

	// render circle
	lineColor := ed.Buffer.LineColors[ln]
	if lineColor != nil {
		start := ed.charStartPos(textpos.Pos{Ln: ln})
		end := ed.charEndPos(textpos.Pos{Ln: ln + 1})

		if ln < ed.NumLines-1 {
			end.Y -= ed.lineHeight
		}
		if end.Y >= float32(bb.Max.Y) {
			return
		}

		// starts at end of line number text
		start.X = tpos.X + lnr.BBox.Size().X
		// ends at end of line number offset
		end.X = float32(bb.Min.X) + ed.LineNumberOffset

		r := (end.X - start.X) / 2
		center := start.AddScalar(r)

		// cut radius in half so that it doesn't look too big
		r /= 2

		pc.Fill.Color = lineColor
		pc.Circle(center.X, center.Y, r)
		pc.PathDone()
	}
}

// firstVisibleLine finds the first visible line, starting at given line
// (typically cursor -- if zero, a visible line is first found) -- returns
// stln if nothing found above it.
func (ed *Base) firstVisibleLine(stln int) int {
	bb := ed.renderBBox()
	if stln == 0 {
		perln := float32(ed.linesSize.Y) / float32(ed.NumLines)
		stln = int(ed.Geom.Scroll.Y/perln) - 1
		if stln < 0 {
			stln = 0
		}
		for ln := stln; ln < ed.NumLines; ln++ {
			lbb := ed.lineBBox(ln)
			if int(math32.Ceil(lbb.Max.Y)) > bb.Min.Y { // visible
				stln = ln
				break
			}
		}
	}
	lastln := stln
	for ln := stln - 1; ln >= 0; ln-- {
		cpos := ed.charStartPos(textpos.Pos{Ln: ln})
		if int(math32.Ceil(cpos.Y)) < bb.Min.Y { // top just offscreen
			break
		}
		lastln = ln
	}
	return lastln
}

// lastVisibleLine finds the last visible line, starting at given line
// (typically cursor) -- returns stln if nothing found beyond it.
func (ed *Base) lastVisibleLine(stln int) int {
	bb := ed.renderBBox()
	lastln := stln
	for ln := stln + 1; ln < ed.NumLines; ln++ {
		pos := textpos.Pos{Ln: ln}
		cpos := ed.charStartPos(pos)
		if int(math32.Floor(cpos.Y)) > bb.Max.Y { // just offscreen
			break
		}
		lastln = ln
	}
	return lastln
}

// PixelToCursor finds the cursor position that corresponds to the given pixel
// location (e.g., from mouse click) which has had ScBBox.Min subtracted from
// it (i.e, relative to upper left of text area)
func (ed *Base) PixelToCursor(pt image.Point) textpos.Pos {
	if ed.NumLines == 0 {
		return textpos.PosZero
	}
	bb := ed.renderBBox()
	sty := &ed.Styles
	yoff := float32(bb.Min.Y)
	xoff := float32(bb.Min.X)
	stln := ed.firstVisibleLine(0)
	cln := stln
	fls := ed.charStartPos(textpos.Pos{Ln: stln}).Y - yoff
	if pt.Y < int(math32.Floor(fls)) {
		cln = stln
	} else if pt.Y > bb.Max.Y {
		cln = ed.NumLines - 1
	} else {
		got := false
		for ln := stln; ln < ed.NumLines; ln++ {
			ls := ed.charStartPos(textpos.Pos{Ln: ln}).Y - yoff
			es := ls
			es += math32.Max(ed.renders[ln].BBox.Size().Y, ed.lineHeight)
			if pt.Y >= int(math32.Floor(ls)) && pt.Y < int(math32.Ceil(es)) {
				got = true
				cln = ln
				break
			}
		}
		if !got {
			cln = ed.NumLines - 1
		}
	}
	// fmt.Printf("cln: %v  pt: %v\n", cln, pt)
	if cln >= len(ed.renders) {
		return textpos.Pos{Ln: cln, Ch: 0}
	}
	lnsz := ed.Buffer.LineLen(cln)
	if lnsz == 0 || sty.Font.Face == nil {
		return textpos.Pos{Ln: cln, Ch: 0}
	}
	scrl := ed.Geom.Scroll.Y
	nolno := float32(pt.X - int(ed.LineNumberOffset))
	sc := int((nolno + scrl) / sty.Font.Face.Metrics.Ch)
	sc -= sc / 4
	sc = max(0, sc)
	cch := sc

	lnst := ed.charStartPos(textpos.Pos{Ln: cln})
	lnst.Y -= yoff
	lnst.X -= xoff
	rpt := math32.FromPoint(pt).Sub(lnst)

	si, ri, ok := ed.renders[cln].PosToRune(rpt)
	if ok {
		cch, _ := ed.renders[cln].SpanPosToRuneIndex(si, ri)
		return textpos.Pos{Ln: cln, Ch: cch}
	}

	return textpos.Pos{Ln: cln, Ch: cch}
}
