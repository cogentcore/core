// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"
	"image"
	"image/color"

	"cogentcore.org/core/cam/hct"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor/textbuf"
)

// Rendering Notes: all rendering is done in Render call.
// Layout must be called whenever content changes across lines.

func (ed *Editor) SetNeedsLayout(updt bool) {
	if updt {
		ed.SetNeedsRender(updt)
		ed.SetFlag(true, EditorNeedsLayout)
	}
}

func (ed *Editor) Render() {
	if ed.PushBounds() {
		ed.ApplyStyle()
		if ed.Is(EditorNeedsLayout) {
			ed.LayoutAll()
			ed.ConfigScrolls()
			ed.SetFlag(false, EditorNeedsLayout)
		}
		ed.PositionScrolls()
		ed.RenderAllLines()
		if ed.Is(EditorTargetSet) {
			ed.ScrollCursorToTarget()
		}
		if ed.StateIs(states.Focused) {
			ed.StartCursor()
		} else {
			ed.StopCursor()
		}
		ed.RenderChildren()
		ed.PopBounds()
		ed.RenderScrolls()
	} else {
		ed.StopCursor()
	}
}

// TextStyleProps returns the styling properties for text based on HiStyle Markup
func (ed *Editor) TextStyleProps() ki.Props {
	if ed.Buf == nil || !ed.Buf.Hi.HasHi() {
		return nil
	}
	return ed.Buf.Hi.CSSProps
}

// CharStartPos returns the starting (top left) render coords for the given
// position -- makes no attempt to rationalize that pos (i.e., if not in
// visible range, position will be out of range too)
func (ed *Editor) CharStartPos(pos lex.Pos) mat32.Vec2 {
	spos := ed.RenderStartPos()
	spos.X += ed.LineNoOff
	if pos.Ln >= len(ed.Offs) {
		if len(ed.Offs) > 0 {
			pos.Ln = len(ed.Offs) - 1
		} else {
			return spos
		}
	} else {
		spos.Y += ed.Offs[pos.Ln] + ed.FontDescent
	}
	if pos.Ln >= len(ed.Renders) {
		return spos
	}
	rp := &ed.Renders[pos.Ln]
	if len(rp.Spans) > 0 {
		// note: Y from rune pos is baseline
		rrp, _, _, _ := ed.Renders[pos.Ln].RuneRelPos(pos.Ch)
		spos.X += rrp.X
		spos.Y += rrp.Y - ed.Renders[pos.Ln].Spans[0].RelPos.Y // relative
	}
	return spos
}

// CharStartPosVis returns the starting pos for given position
// that is currently visible, based on bounding boxes.
func (ed *Editor) CharStartPosVis(pos lex.Pos) mat32.Vec2 {
	spos := ed.CharStartPos(pos)
	bb := ed.Geom.ContentBBox
	bbmin := mat32.V2FromPoint(bb.Min)
	bbmin.X += ed.LineNoOff
	bbmax := mat32.V2FromPoint(bb.Max)
	spos.SetMax(bbmin)
	spos.SetMin(bbmax)
	return spos
}

// CharEndPos returns the ending (bottom right) render coords for the given
// position -- makes no attempt to rationalize that pos (i.e., if not in
// visible range, position will be out of range too)
func (ed *Editor) CharEndPos(pos lex.Pos) mat32.Vec2 {
	spos := ed.RenderStartPos()
	pos.Ln = min(pos.Ln, ed.NLines-1)
	if pos.Ln < 0 {
		spos.Y += float32(ed.LinesSize.Y)
		spos.X += ed.LineNoOff
		return spos
	}
	if pos.Ln >= len(ed.Offs) {
		spos.Y += float32(ed.LinesSize.Y)
		spos.X += ed.LineNoOff
		return spos
	}
	spos.Y += ed.Offs[pos.Ln] + ed.FontDescent
	spos.X += ed.LineNoOff
	if len(ed.Renders[pos.Ln].Spans) > 0 {
		// note: Y from rune pos is baseline
		rrp, _, _, _ := ed.Renders[pos.Ln].RuneEndPos(pos.Ch)
		spos.X += rrp.X
		spos.Y += rrp.Y - ed.Renders[pos.Ln].Spans[0].RelPos.Y // relative
	}
	spos.Y += ed.LineHeight // end of that line
	return spos
}

// ViewDepthOffsets are changes in color values from default background for different
// depths.  For dark mode, these are increments, for light mode they are decrements.
var ViewDepthColors = []color.RGBA{
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

// RenderDepthBg renders the depth background color.
func (ed *Editor) RenderDepthBg(stln, edln int) {
	if ed.Buf == nil {
		return
	}
	if !ed.Buf.Opts.DepthColor || ed.IsDisabled() || !ed.StateIs(states.Focused) {
		return
	}
	buf := ed.Buf
	buf.MarkupMu.RLock() // needed for HiTags access
	defer buf.MarkupMu.RUnlock()

	bb := ed.Geom.ContentBBox
	sty := &ed.Styles
	isDark := matcolor.SchemeIsDark
	nclrs := len(ViewDepthColors)
	lstdp := 0
	for ln := stln; ln <= edln; ln++ {
		lst := ed.CharStartPos(lex.Pos{Ln: ln}).Y // note: charstart pos includes descent
		led := lst + mat32.Max(ed.Renders[ln].Size.Y, ed.LineHeight)
		if int(mat32.Ceil(led)) < bb.Min.Y {
			continue
		}
		if int(mat32.Floor(lst)) > bb.Max.Y {
			continue
		}
		if ln >= len(buf.HiTags) { // may be out of sync
			continue
		}
		ht := buf.HiTags[ln]
		lsted := 0
		for ti := range ht {
			lx := &ht[ti]
			if lx.Tok.Depth > 0 {
				var vdc color.RGBA
				if isDark { // reverse order too
					vdc = ViewDepthColors[nclrs-1-lx.Tok.Depth%nclrs]
				} else {
					vdc = ViewDepthColors[lx.Tok.Depth%nclrs]
				}
				bg := colors.Apply(sty.Background, func(c color.Color) color.Color {
					if isDark { // reverse order too
						return colors.Add(c, vdc)
					}
					return colors.Sub(c, vdc)
				})

				st := min(lsted, lx.St)
				reg := textbuf.Region{Start: lex.Pos{Ln: ln, Ch: st}, End: lex.Pos{Ln: ln, Ch: lx.Ed}}
				lsted = lx.Ed
				lstdp = lx.Tok.Depth
				ed.RenderRegionBoxSty(reg, sty, bg, true) // full width alway
			}
		}
		if lstdp > 0 {
			ed.RenderRegionToEnd(lex.Pos{Ln: ln, Ch: lsted}, sty, sty.Background)
		}
	}
}

// RenderSelect renders the selection region as a selected background color.
func (ed *Editor) RenderSelect() {
	if !ed.HasSelection() {
		return
	}
	ed.RenderRegionBox(ed.SelectReg, ed.SelectColor)
}

// RenderHighlights renders the highlight regions as a
// highlighted background color.
func (ed *Editor) RenderHighlights(stln, edln int) {
	for _, reg := range ed.Highlights {
		reg := ed.Buf.AdjustReg(reg)
		if reg.IsNil() || (stln >= 0 && (reg.Start.Ln > edln || reg.End.Ln < stln)) {
			continue
		}
		ed.RenderRegionBox(reg, ed.HighlightColor)
	}
}

// RenderScopelights renders a highlight background color for regions
// in the Scopelights list.
func (ed *Editor) RenderScopelights(stln, edln int) {
	for _, reg := range ed.Scopelights {
		reg := ed.Buf.AdjustReg(reg)
		if reg.IsNil() || (stln >= 0 && (reg.Start.Ln > edln || reg.End.Ln < stln)) {
			continue
		}
		ed.RenderRegionBox(reg, ed.HighlightColor)
	}
}

// RenderRegionBox renders a region in background according to given background
func (ed *Editor) RenderRegionBox(reg textbuf.Region, bg image.Image) {
	ed.RenderRegionBoxSty(reg, &ed.Styles, bg, false)
}

// RenderRegionBoxSty renders a region in given style and background
func (ed *Editor) RenderRegionBoxSty(reg textbuf.Region, sty *styles.Style, bg image.Image, fullWidth bool) {
	st := reg.Start
	end := reg.End
	spos := ed.CharStartPosVis(st)
	epos := ed.CharStartPosVis(end)
	epos.Y += ed.LineHeight
	bb := ed.Geom.ContentBBox
	stx := mat32.Ceil(float32(bb.Min.X) + ed.LineNoOff)
	if int(mat32.Ceil(epos.Y)) < bb.Min.Y || int(mat32.Floor(spos.Y)) > bb.Max.Y {
		return
	}
	ex := float32(ed.Geom.ContentBBox.Max.X)
	if fullWidth {
		epos.X = ex
	}

	pc := &ed.Scene.PaintContext
	stsi, _, _ := ed.WrappedLineNo(st)
	edsi, _, _ := ed.WrappedLineNo(end)
	if st.Ln == end.Ln && stsi == edsi {
		pc.FillBox(spos, epos.Sub(spos), bg) // same line, done
		return
	}
	// on diff lines: fill to end of stln
	seb := spos
	seb.Y += ed.LineHeight
	seb.X = ex
	pc.FillBox(spos, seb.Sub(spos), bg)
	sfb := seb
	sfb.X = stx
	if sfb.Y < epos.Y { // has some full box
		efb := epos
		efb.Y -= ed.LineHeight
		efb.X = ex
		pc.FillBox(sfb, efb.Sub(sfb), bg)
	}
	sed := epos
	sed.Y -= ed.LineHeight
	sed.X = stx
	pc.FillBox(sed, epos.Sub(sed), bg)
}

// RenderRegionToEnd renders a region in given style and background, to end of line from start
func (ed *Editor) RenderRegionToEnd(st lex.Pos, sty *styles.Style, bg image.Image) {
	spos := ed.CharStartPosVis(st)
	epos := spos
	epos.Y += ed.LineHeight
	vsz := epos.Sub(spos)
	if vsz.X <= 0 || vsz.Y <= 0 {
		return
	}
	pc := &ed.Scene.PaintContext
	pc.FillBox(spos, epos.Sub(spos), bg) // same line, done
}

// RenderStartPos is absolute rendering start position from our content pos with scroll
// This can be offscreen (left, up) based on scrolling.
func (ed *Editor) RenderStartPos() mat32.Vec2 {
	pos := ed.Geom.Pos.Content.Add(ed.Geom.Scroll)
	return pos
}

// RenderAllLines displays all the visible lines on the screen,
// after PushBounds has already been called.
func (ed *Editor) RenderAllLines() {
	pc := &ed.Scene.PaintContext
	pc.Lock()
	sty := &ed.Styles
	bb := ed.Geom.ContentBBox
	bbmin := mat32.V2FromPoint(bb.Min)
	bbmax := mat32.V2FromPoint(bb.Max)
	pc.FillBox(bbmin, bbmax.Sub(bbmin), sty.Background)
	pos := ed.RenderStartPos()
	stln := -1
	edln := -1
	for ln := 0; ln < ed.NLines; ln++ {
		if ln >= len(ed.Offs) || ln >= len(ed.Renders) {
			break
		}
		lst := pos.Y + ed.Offs[ln]
		led := lst + mat32.Max(ed.Renders[ln].Size.Y, ed.LineHeight)
		if int(mat32.Ceil(led)) < bb.Min.Y {
			continue
		}
		if int(mat32.Floor(lst)) > bb.Max.Y {
			continue
		}
		if stln < 0 {
			stln = ln
		}
		edln = ln
	}

	if stln < 0 || edln < 0 { // shouldn't happen.
		pc.Unlock()
		return
	}

	if ed.HasLineNos() {
		ed.RenderLineNosBoxAll()
		for ln := stln; ln <= edln; ln++ {
			ed.RenderLineNo(ln, false, false) // don't re-render std fill boxes, no separate vp upload
		}
	}

	ed.RenderDepthBg(stln, edln)
	ed.RenderHighlights(stln, edln)
	ed.RenderScopelights(stln, edln)
	ed.RenderSelect()
	if ed.HasLineNos() {
		tbb := bb
		tbb.Min.X += int(ed.LineNoOff)
		pc.Unlock()
		pc.PushBounds(tbb)
		pc.Lock()
	}
	for ln := stln; ln <= edln; ln++ {
		lst := pos.Y + ed.Offs[ln]
		lp := pos
		lp.Y = lst
		lp.X += ed.LineNoOff
		if lp.Y+ed.LineHeight > bbmax.Y {
			break
		}
		ed.Renders[ln].Render(pc, lp) // not top pos -- already has baseline offset
	}
	pc.Unlock()
	if ed.HasLineNos() {
		pc.PopBounds()
	}
}

// RenderLineNosBoxAll renders the background for the line numbers in the LineNumberColor
func (ed *Editor) RenderLineNosBoxAll() {
	if !ed.HasLineNos() {
		return
	}
	pc := &ed.Scene.PaintContext
	bb := ed.Geom.ContentBBox
	spos := mat32.V2FromPoint(bb.Min)
	epos := mat32.V2FromPoint(bb.Max)
	epos.X = spos.X + ed.LineNoOff
	pc.FillBox(spos, epos.Sub(spos), ed.LineNumberColor)
}

// RenderLineNosBox renders the background for the line numbers in given range, in the LineNumberColor
func (ed *Editor) RenderLineNosBox(st, end int) {
	if !ed.HasLineNos() {
		return
	}
	pc := &ed.Scene.PaintContext
	// sty := &ed.Styles
	// spc := sty.BoxSpace()
	bb := ed.Geom.ContentBBox
	spos := ed.CharStartPos(lex.Pos{Ln: st})
	spos.X = float32(bb.Min.X)
	epos := ed.CharEndPos(lex.Pos{Ln: end + 1})
	epos.Y -= ed.LineHeight
	epos.X = spos.X + ed.LineNoOff
	pc.FillBox(spos, epos.Sub(spos), ed.LineNumberColor)
}

// RenderLineNo renders given line number -- called within context of other render
// if defFill is true, it fills box color for default background color (use false for batch mode)
// and if vpUpload is true it uploads the rendered region to scene directly
// (only if totally separate from other updates)
func (ed *Editor) RenderLineNo(ln int, defFill bool, vpUpload bool) {
	if !ed.HasLineNos() || ed.Buf == nil {
		return
	}

	sc := ed.Scene
	sty := &ed.Styles
	fst := sty.FontRender()
	pc := &sc.PaintContext
	bb := ed.Geom.ContentBBox

	// render fillbox
	sbox := ed.CharStartPos(lex.Pos{Ln: ln})
	sbox.X = float32(bb.Min.X)
	ebox := ed.CharEndPos(lex.Pos{Ln: ln + 1})
	if ln < ed.NLines-1 {
		ebox.Y -= ed.LineHeight
	}
	if ebox.Y >= float32(bb.Max.Y) {
		return
	}
	ebox.X = sbox.X + ed.LineNoOff
	bsz := ebox.Sub(sbox)
	lclr, hasLClr := ed.Buf.LineColors[ln]
	actClr := lclr
	if ed.CursorPos.Ln == ln {
		if hasLClr { // split the diff!
			bszhlf := bsz
			bszhlf.X /= 2
			pc.FillBoxColor(sbox, bszhlf, lclr)
			nsp := sbox
			nsp.X += bszhlf.X
			pc.FillBox(nsp, bszhlf, ed.SelectColor)
		} else {
			actClr = colors.ToUniform(ed.SelectColor)
			pc.FillBox(sbox, bsz, ed.SelectColor)
		}
	} else if hasLClr {
		pc.FillBoxColor(sbox, bsz, lclr)
	} else if defFill {
		actClr = colors.ToUniform(ed.LineNumberColor)
		pc.FillBox(sbox, bsz, ed.LineNumberColor)
	}

	fst.Background = nil
	lfmt := fmt.Sprintf("%d", ed.LineNoDigs)
	lfmt = "%" + lfmt + "d"
	lnstr := fmt.Sprintf(lfmt, ln+1)

	if hct.ContrastRatio(actClr, fst.Color) < hct.ContrastAA {
		fst.Color = hct.ContrastColor(actClr, hct.ContrastAA)
	}
	ed.LineNoRender.SetString(lnstr, fst, &sty.UnContext, &sty.Text, true, 0, 0)
	pos := mat32.Vec2{}
	lst := ed.CharStartPos(lex.Pos{Ln: ln}).Y // note: charstart pos includes descent
	pos.Y = lst + ed.FontAscent - ed.FontDescent
	pos.X = float32(bb.Min.X) // + spc.Pos().X

	ed.LineNoRender.Render(pc, pos)
	// todo: need an SvgRender interface that just takes an svg file or object
	// and renders it to a given bitmap, and then just keep that around.
	// if icnm, ok := ed.Buf.LineIcons[ln]; ok {
	// 	ic := ed.Buf.Icons[icnm]
	// 	ic.Par = ed
	// 	ic.Scene = ed.Sc
	// 	// pos.X += 20 // todo
	// 	sic := ic.SVGIcon()
	// 	sic.Resize(image.Point{20, 20})
	// 	sic.FullRenderTree()
	// 	ist := sbox.ToPointFloor()
	// 	ied := ebox.ToPointFloor()
	// 	ied.X += int(spc)
	// 	ist.X = ied.X - 20
	// 	r := image.Rectangle{Min: ist, Max: ied}
	// 	sic.Sty.BackgroundColor.SetName("black")
	// 	sic.FillScene()
	// 	draw.Draw(ed.Sc.Pixels, r, sic.Pixels, image.Point{}, draw.Over)
	// }
	// if vpUpload {
	// 	tBBox := image.Rectangle{sbox.ToPointFloor(), ebox.ToPointCeil()}
	// 	winoff := bb.Min.Sub(bb.Min)
	// 	tScBBox := tBBox.Add(winoff)
	// 	sc.This().(gi.Scene).ScUploadRegion(tBBox, tScBBox)
	// }
}

// FirstVisibleLine finds the first visible line, starting at given line
// (typically cursor -- if zero, a visible line is first found) -- returns
// stln if nothing found above it.
func (ed *Editor) FirstVisibleLine(stln int) int {
	bb := ed.Geom.ContentBBox
	if stln == 0 {
		perln := float32(ed.LinesSize.Y) / float32(ed.NLines)
		// stln = int(float32(bb.Min.Y-ed.ObjBBox.Min.Y)/perln) - 1 // todo: scroll
		stln = int(ed.Geom.Scroll.Y/perln) - 1
		if stln < 0 {
			stln = 0
		}
		for ln := stln; ln < ed.NLines; ln++ {
			cpos := ed.CharStartPos(lex.Pos{Ln: ln})
			if int(mat32.Floor(cpos.Y)) >= bb.Min.Y { // top definitely on screen
				stln = ln
				break
			}
		}
	}
	lastln := stln
	for ln := stln - 1; ln >= 0; ln-- {
		cpos := ed.CharStartPos(lex.Pos{Ln: ln})
		if int(mat32.Ceil(cpos.Y)) < bb.Min.Y { // top just offscreen
			break
		}
		lastln = ln
	}
	return lastln
}

// LastVisibleLine finds the last visible line, starting at given line
// (typically cursor) -- returns stln if nothing found beyond it.
func (ed *Editor) LastVisibleLine(stln int) int {
	bb := ed.Geom.ContentBBox
	lastln := stln
	for ln := stln + 1; ln < ed.NLines; ln++ {
		pos := lex.Pos{Ln: ln}
		cpos := ed.CharStartPos(pos)
		if int(mat32.Floor(cpos.Y)) > bb.Max.Y { // just offscreen
			break
		}
		lastln = ln
	}
	return lastln
}

// PixelToCursor finds the cursor position that corresponds to the given pixel
// location (e.g., from mouse click) which has had ScBBox.Min subtracted from
// it (i.e, relative to upper left of text area)
func (ed *Editor) PixelToCursor(pt image.Point) lex.Pos {
	if ed.NLines == 0 {
		return lex.PosZero
	}
	bb := ed.Geom.ContentBBox
	sty := &ed.Styles
	yoff := float32(bb.Min.Y)
	stln := ed.FirstVisibleLine(0)
	cln := stln
	fls := ed.CharStartPos(lex.Pos{Ln: stln}).Y - yoff
	if pt.Y < int(mat32.Floor(fls)) {
		cln = stln
	} else if pt.Y > bb.Max.Y {
		cln = ed.NLines - 1
	} else {
		got := false
		for ln := stln; ln < ed.NLines; ln++ {
			ls := ed.CharStartPos(lex.Pos{Ln: ln}).Y - yoff
			es := ls
			es += mat32.Max(ed.Renders[ln].Size.Y, ed.LineHeight)
			if pt.Y >= int(mat32.Floor(ls)) && pt.Y < int(mat32.Ceil(es)) {
				got = true
				cln = ln
				break
			}
		}
		if !got {
			cln = ed.NLines - 1
		}
	}
	// fmt.Printf("cln: %v  pt: %v\n", cln, pt)
	lnsz := ed.Buf.LineLen(cln)
	if lnsz == 0 {
		return lex.Pos{Ln: cln, Ch: 0}
	}
	xoff := float32(bb.Min.X)
	scrl := ed.Geom.Scroll.Y
	nolno := float32(pt.X - int(ed.LineNoOff))
	if sty.Font.Face == nil {
		return lex.Pos{Ln: cln, Ch: 0}
	}
	sc := int((nolno + scrl) / sty.Font.Face.Metrics.Ch)
	sc -= sc / 4
	sc = max(0, sc)
	cch := sc

	si := 0
	spoff := 0
	if cln >= len(ed.Renders) {
		return lex.Pos{Ln: cln, Ch: 0}
	}
	nspan := len(ed.Renders[cln].Spans)
	lstY := ed.CharStartPos(lex.Pos{Ln: cln}).Y - yoff
	if nspan > 1 {
		si = int((float32(pt.Y) - lstY) / ed.LineHeight)
		si = min(si, nspan-1)
		si = max(si, 0)
		for i := 0; i < si; i++ {
			spoff += len(ed.Renders[cln].Spans[i].Text)
		}
		// fmt.Printf("si: %v  spoff: %v\n", si, spoff)
	}

	ri := sc
	if si >= nspan {
		return lex.Pos{Ln: cln, Ch: spoff}
	}
	rsz := len(ed.Renders[cln].Spans[si].Text)
	if rsz == 0 {
		return lex.Pos{Ln: cln, Ch: spoff}
	}
	// fmt.Printf("sc: %v  rsz: %v\n", sc, rsz)

	c, _ := ed.Renders[cln].SpanPosToRuneIdx(si, rsz-1) // end
	rsp := mat32.Floor(ed.CharStartPos(lex.Pos{Ln: cln, Ch: c}).X - xoff)
	rep := mat32.Ceil(ed.CharEndPos(lex.Pos{Ln: cln, Ch: c}).X - xoff)
	if int(rep) < pt.X { // end of line
		if si == nspan-1 {
			c++
		}
		return lex.Pos{Ln: cln, Ch: c}
	}

	tooBig := false
	got := false
	if ri < rsz {
		for rii := ri; rii < rsz; rii++ {
			c, _ := ed.Renders[cln].SpanPosToRuneIdx(si, rii)
			rsp = mat32.Floor(ed.CharStartPos(lex.Pos{Ln: cln, Ch: c}).X - xoff)
			rep = mat32.Ceil(ed.CharEndPos(lex.Pos{Ln: cln, Ch: c}).X - xoff)
			// fmt.Printf("trying c: %v for pt: %v xoff: %v rsp: %v, rep: %v\n", c, pt, xoff, rsp, rep)
			if pt.X >= int(rsp) && pt.X < int(rep) {
				cch = c
				got = true
				// fmt.Printf("got cch: %v for pt: %v rsp: %v, rep: %v\n", cch, pt, rsp, rep)
				break
			} else if int(rep) > pt.X {
				cch = c
				tooBig = true
				break
			}
		}
	} else {
		tooBig = true
	}
	if !got && tooBig {
		ri = rsz - 1
		// fmt.Printf("too big: %v\n", ri)
		for rii := ri; rii >= 0; rii-- {
			c, _ := ed.Renders[cln].SpanPosToRuneIdx(si, rii)
			rsp := mat32.Floor(ed.CharStartPos(lex.Pos{Ln: cln, Ch: c}).X - xoff)
			rep := mat32.Ceil(ed.CharEndPos(lex.Pos{Ln: cln, Ch: c}).X - xoff)
			// fmt.Printf("too big: trying c: %v for pt: %v rsp: %v, rep: %v\n", c, pt, rsp, rep)
			if pt.X >= int(rsp) && pt.X < int(rep) {
				got = true
				cch = c
				// fmt.Printf("got cch: %v for pt: %v rsp: %v, rep: %v\n", cch, pt, rsp, rep)
				break
			}
		}
	}
	return lex.Pos{Ln: cln, Ch: cch}
}
