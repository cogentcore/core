// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textview

import (
	"fmt"
	"image"
	"image/color"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/textview/textbuf"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/lex"
)

// Rendering Notes: all rendering is done in Render call.
// Layout must be called whenever content changes across
// lines.

func (tv *View) Render(sc *gi.Scene) {
	if tv.PushBounds(sc) {
		tv.RenderAllLinesInBounds()
		if tv.ScrollToCursorOnRender {
			tv.ScrollToCursorOnRender = false
			tv.CursorPos = tv.ScrollToCursorPos
			tv.ScrollCursorToTop()
		}
		if tv.StateIs(states.Focused) {
			tv.StartCursor()
		} else {
			tv.StopCursor()
		}
		tv.RenderChildren(sc)
		tv.RenderScrolls(sc)
		tv.PopBounds(sc)
	} else {
		tv.StopCursor()
	}
}

// HiStyle applies the highlighting styles from buffer markup style
func (tv *View) HiStyle() {
	// STYTODO: need to figure out what to do with histyle
	if !tv.Buf.Hi.HasHi() {
		return
	}
	tv.CSS = tv.Buf.Hi.CSSProps
	if chp, ok := ki.SubProps(tv.CSS, ".chroma"); ok {
		for ky, vl := range chp { // apply to top level
			tv.SetProp(ky, vl)
		}
	}
}

// CharStartPos returns the starting (top left) render coords for the given
// position -- makes no attempt to rationalize that pos (i.e., if not in
// visible range, position will be out of range too)
func (tv *View) CharStartPos(pos lex.Pos) mat32.Vec2 {
	spos := tv.RenderStartPos()
	spos.X += tv.LineNoOff
	if pos.Ln >= len(tv.Offs) {
		if len(tv.Offs) > 0 {
			pos.Ln = len(tv.Offs) - 1
		} else {
			return spos
		}
	} else {
		spos.Y += tv.Offs[pos.Ln] + mat32.FromFixed(tv.Styles.Font.Face.Face.Metrics().Descent)
	}
	if len(tv.Renders[pos.Ln].Spans) > 0 {
		// note: Y from rune pos is baseline
		rrp, _, _, _ := tv.Renders[pos.Ln].RuneRelPos(pos.Ch)
		spos.X += rrp.X
		spos.Y += rrp.Y - tv.Renders[pos.Ln].Spans[0].RelPos.Y // relative
	}
	return spos
}

// CharEndPos returns the ending (bottom right) render coords for the given
// position -- makes no attempt to rationalize that pos (i.e., if not in
// visible range, position will be out of range too)
func (tv *View) CharEndPos(pos lex.Pos) mat32.Vec2 {
	spos := tv.RenderStartPos()
	pos.Ln = min(pos.Ln, tv.NLines-1)
	if pos.Ln < 0 {
		spos.Y += float32(tv.LinesSize.Y)
		spos.X += tv.LineNoOff
		return spos
	}
	// if pos.Ln >= tv.NLines {
	// 	spos.Y += float32(tv.LinesSize.Y)
	// 	spos.X += tv.LineNoOff
	// 	return spos
	// }
	spos.Y += tv.Offs[pos.Ln] + mat32.FromFixed(tv.Styles.Font.Face.Face.Metrics().Descent)
	spos.X += tv.LineNoOff
	if len(tv.Renders[pos.Ln].Spans) > 0 {
		// note: Y from rune pos is baseline
		rrp, _, _, _ := tv.Renders[pos.Ln].RuneEndPos(pos.Ch)
		spos.X += rrp.X
		spos.Y += rrp.Y - tv.Renders[pos.Ln].Spans[0].RelPos.Y // relative
	}
	spos.Y += tv.LineHeight // end of that line
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

// RenderDepthBg renders the depth background color
func (tv *View) RenderDepthBg(stln, edln int) {
	if tv.Buf == nil {
		return
	}
	if !tv.Buf.Opts.DepthColor || tv.IsDisabled() || !tv.StateIs(states.Focused) {
		return
	}
	tv.Buf.MarkupMu.RLock() // needed for HiTags access
	defer tv.Buf.MarkupMu.RUnlock()
	sty := &tv.Styles
	cspec := sty.BackgroundColor
	bg := cspec.Solid
	// STYTODO: fix textview colors
	// isDark := bg.IsDark()
	// nclrs := len(ViewDepthColors)
	lstdp := 0
	for ln := stln; ln <= edln; ln++ {
		lst := tv.CharStartPos(lex.Pos{Ln: ln}).Y // note: charstart pos includes descent
		led := lst + mat32.Max(tv.Renders[ln].Size.Y, tv.LineHeight)
		if int(mat32.Ceil(led)) < tv.ScBBox.Min.Y {
			continue
		}
		if int(mat32.Floor(lst)) > tv.ScBBox.Max.Y {
			continue
		}
		if ln >= len(tv.Buf.HiTags) { // may be out of sync
			continue
		}
		ht := tv.Buf.HiTags[ln]
		lsted := 0
		for ti := range ht {
			lx := &ht[ti]
			if lx.Tok.Depth > 0 {
				cspec.Solid = bg
				// if isDark {
				// 	// reverse order too
				// 	cspec.Color.Add(ViewDepthColors[nclrs-1-lx.Tok.Depth%nclrs])
				// } else {
				// 	cspec.Color.Sub(ViewDepthColors[lx.Tok.Depth%nclrs])
				// }
				st := min(lsted, lx.St)
				reg := textbuf.Region{Start: lex.Pos{Ln: ln, Ch: st}, End: lex.Pos{Ln: ln, Ch: lx.Ed}}
				lsted = lx.Ed
				lstdp = lx.Tok.Depth
				tv.RenderRegionBoxSty(reg, sty, &cspec)
			}
		}
		if lstdp > 0 {
			tv.RenderRegionToEnd(lex.Pos{Ln: ln, Ch: lsted}, sty, &cspec)
		}
	}
}

// RenderSelect renders the selection region as a selected background color
// -- always called within context of outer RenderLines or RenderAllLines
func (tv *View) RenderSelect() {
	if !tv.HasSelection() {
		return
	}
	tv.RenderRegionBox(tv.SelectReg, &tv.SelectColor)
}

// RenderHighlights renders the highlight regions as a highlighted background
// color -- always called within context of outer RenderLines or
// RenderAllLines
func (tv *View) RenderHighlights(stln, edln int) {
	for _, reg := range tv.Highlights {
		reg := tv.Buf.AdjustReg(reg)
		if reg.IsNil() || (stln >= 0 && (reg.Start.Ln > edln || reg.End.Ln < stln)) {
			continue
		}
		tv.RenderRegionBox(reg, &tv.HighlightColor)
	}
}

// RenderScopelights renders a highlight background color for regions
// in the Scopelights list
// -- always called within context of outer RenderLines or RenderAllLines
func (tv *View) RenderScopelights(stln, edln int) {
	for _, reg := range tv.Scopelights {
		reg := tv.Buf.AdjustReg(reg)
		if reg.IsNil() || (stln >= 0 && (reg.Start.Ln > edln || reg.End.Ln < stln)) {
			continue
		}
		tv.RenderRegionBox(reg, &tv.HighlightColor)
	}
}

// RenderRegionBox renders a region in background color according to given background color
func (tv *View) RenderRegionBox(reg textbuf.Region, bgclr *colors.Full) {
	tv.RenderRegionBoxSty(reg, &tv.Styles, bgclr)
}

// RenderRegionBoxSty renders a region in given style and background color
func (tv *View) RenderRegionBoxSty(reg textbuf.Region, sty *styles.Style, bgclr *colors.Full) {
	st := reg.Start
	ed := reg.End
	spos := tv.CharStartPos(st)
	epos := tv.CharStartPos(ed)
	epos.Y += tv.LineHeight
	if int(mat32.Ceil(epos.Y)) < tv.ScBBox.Min.Y || int(mat32.Floor(spos.Y)) > tv.ScBBox.Max.Y {
		return
	}

	rs := &tv.Sc.RenderState
	pc := &rs.Paint
	spc := sty.BoxSpace()

	rst := tv.RenderStartPos()
	// SidesTODO: this is sketchy
	ex := float32(tv.ScBBox.Max.X) - spc.Right
	sx := rst.X + tv.LineNoOff

	// fmt.Printf("select: %v -- %v\n", st, ed)

	stsi, _, _ := tv.WrappedLineNo(st)
	edsi, _, _ := tv.WrappedLineNo(ed)
	if st.Ln == ed.Ln && stsi == edsi {
		pc.FillBox(rs, spos, epos.Sub(spos), bgclr) // same line, done
		return
	}
	// on diff lines: fill to end of stln
	seb := spos
	seb.Y += tv.LineHeight
	seb.X = ex
	pc.FillBox(rs, spos, seb.Sub(spos), bgclr)
	sfb := seb
	sfb.X = sx
	if sfb.Y < epos.Y { // has some full box
		efb := epos
		efb.Y -= tv.LineHeight
		efb.X = ex
		pc.FillBox(rs, sfb, efb.Sub(sfb), bgclr)
	}
	sed := epos
	sed.Y -= tv.LineHeight
	sed.X = sx
	pc.FillBox(rs, sed, epos.Sub(sed), bgclr)
}

// RenderRegionToEnd renders a region in given style and background color, to end of line from start
func (tv *View) RenderRegionToEnd(st lex.Pos, sty *styles.Style, bgclr *colors.Full) {
	spos := tv.CharStartPos(st)
	epos := spos
	epos.Y += tv.LineHeight
	epos.X = float32(tv.ScBBox.Max.X)
	if int(mat32.Ceil(epos.Y)) < tv.ScBBox.Min.Y || int(mat32.Floor(spos.Y)) > tv.ScBBox.Max.Y {
		return
	}

	rs := &tv.Sc.RenderState
	pc := &rs.Paint

	pc.FillBox(rs, spos, epos.Sub(spos), bgclr) // same line, done
}

// RenderStartPos is absolute rendering start position from our allocpos
func (tv *View) RenderStartPos() mat32.Vec2 {
	st := &tv.Styles
	spc := st.BoxSpace()
	pos := tv.LayState.Alloc.Pos.Add(spc.Pos())
	delta := mat32.NewVec2FmPoint(tv.LayoutScrollDelta((image.Point{})))
	pos = pos.Add(delta)
	return pos
}

// RenderAllLinesInBounds displays all the visible lines on the screen --
// after PushBounds has already been called
func (tv *View) RenderAllLinesInBounds() {
	rs := &tv.Sc.RenderState
	rs.Lock()
	pc := &rs.Paint
	sty := &tv.Styles
	pos := mat32.NewVec2FmPoint(tv.ScBBox.Min)
	epos := mat32.NewVec2FmPoint(tv.ScBBox.Max)
	pc.FillBox(rs, pos, epos.Sub(pos), &sty.BackgroundColor)
	pos = tv.RenderStartPos()
	stln := -1
	edln := -1
	for ln := 0; ln < tv.NLines; ln++ {
		lst := pos.Y + tv.Offs[ln]
		led := lst + mat32.Max(tv.Renders[ln].Size.Y, tv.LineHeight)
		if int(mat32.Ceil(led)) < tv.ScBBox.Min.Y {
			continue
		}
		if int(mat32.Floor(lst)) > tv.ScBBox.Max.Y {
			continue
		}
		if stln < 0 {
			stln = ln
		}
		edln = ln
	}

	if stln < 0 || edln < 0 { // shouldn't happen.
		rs.Unlock()
		return
	}

	if tv.HasLineNos() {
		tv.RenderLineNosBoxAll()
		for ln := stln; ln <= edln; ln++ {
			tv.RenderLineNo(ln, false, false) // don't re-render std fill boxes, no separate vp upload
		}
	}

	tv.RenderDepthBg(stln, edln)
	tv.RenderHighlights(stln, edln)
	tv.RenderScopelights(stln, edln)
	tv.RenderSelect()
	if tv.HasLineNos() {
		tbb := tv.ScBBox
		tbb.Min.X += int(tv.LineNoOff)
		rs.Unlock()
		rs.PushBounds(tbb)
		rs.Lock()
	}
	for ln := stln; ln <= edln; ln++ {
		lst := pos.Y + tv.Offs[ln]
		lp := pos
		lp.Y = lst
		lp.X += tv.LineNoOff
		tv.Renders[ln].Render(rs, lp) // not top pos -- already has baseline offset
	}
	rs.Unlock()
	if tv.HasLineNos() {
		rs.PopBounds()
	}
}

// RenderLineNosBoxAll renders the background for the line numbers in the LineNumberColor
func (tv *View) RenderLineNosBoxAll() {
	if !tv.HasLineNos() {
		return
	}
	rs := &tv.Sc.RenderState
	pc := &rs.Paint
	sty := &tv.Styles
	spc := sty.BoxSpace()
	spos := mat32.NewVec2FmPoint(tv.ScBBox.Min)
	epos := mat32.NewVec2FmPoint(tv.ScBBox.Max)
	// SidesTODO: this is sketchy
	epos.X = spos.X + tv.LineNoOff - spc.Size().X/2
	pc.FillBoxColor(rs, spos, epos.Sub(spos), tv.LineNumberColor.Solid)
}

// RenderLineNosBox renders the background for the line numbers in given range, in the LineNumberColor
func (tv *View) RenderLineNosBox(st, ed int) {
	if !tv.HasLineNos() {
		return
	}
	rs := &tv.Sc.RenderState
	pc := &rs.Paint
	sty := &tv.Styles
	spc := sty.BoxSpace()
	spos := tv.CharStartPos(lex.Pos{Ln: st})
	spos.X = float32(tv.ScBBox.Min.X)
	epos := tv.CharEndPos(lex.Pos{Ln: ed + 1})
	epos.Y -= tv.LineHeight
	// SidesTODO: this is sketchy
	epos.X = spos.X + tv.LineNoOff - spc.Size().X/2
	// fmt.Printf("line box: st %v ed: %v spos %v  epos %v\n", st, ed, spos, epos)
	pc.FillBoxColor(rs, spos, epos.Sub(spos), tv.LineNumberColor.Solid)
}

// RenderLineNo renders given line number -- called within context of other render
// if defFill is true, it fills box color for default background color (use false for batch mode)
// and if vpUpload is true it uploads the rendered region to scene directly
// (only if totally separate from other updates)
func (tv *View) RenderLineNo(ln int, defFill bool, vpUpload bool) {
	if !tv.HasLineNos() || tv.Buf == nil {
		return
	}

	sc := tv.Sc
	sty := &tv.Styles
	spc := sty.BoxSpace()
	fst := sty.FontRender()
	rs := &sc.RenderState
	pc := &rs.Paint

	// render fillbox
	sbox := tv.CharStartPos(lex.Pos{Ln: ln})
	sbox.X = float32(tv.ScBBox.Min.X)
	ebox := tv.CharEndPos(lex.Pos{Ln: ln + 1})
	if ln < tv.NLines-1 {
		ebox.Y -= tv.LineHeight
	}
	// SidesTODO: this is sketchy
	ebox.X = sbox.X + tv.LineNoOff - spc.Size().X/2
	bsz := ebox.Sub(sbox)
	lclr, hasLClr := tv.Buf.LineColors[ln]
	if tv.CursorPos.Ln == ln {
		if hasLClr { // split the diff!
			bszhlf := bsz
			bszhlf.X /= 2
			pc.FillBoxColor(rs, sbox, bszhlf, lclr)
			nsp := sbox
			nsp.X += bszhlf.X
			pc.FillBoxColor(rs, nsp, bszhlf, tv.SelectColor.Solid)
		} else {
			pc.FillBoxColor(rs, sbox, bsz, tv.SelectColor.Solid)
		}
	} else if hasLClr {
		pc.FillBoxColor(rs, sbox, bsz, lclr)
	} else if defFill {
		pc.FillBoxColor(rs, sbox, bsz, tv.LineNumberColor.Solid)
	}

	fst.BackgroundColor.SetSolid(nil)
	lfmt := fmt.Sprintf("%d", tv.LineNoDigs)
	lfmt = "%" + lfmt + "d"
	lnstr := fmt.Sprintf(lfmt, ln+1)
	tv.LineNoRender.SetString(lnstr, fst, &sty.UnContext, &sty.Text, true, 0, 0)
	pos := mat32.Vec2{}
	lst := tv.CharStartPos(lex.Pos{Ln: ln}).Y // note: charstart pos includes descent
	pos.Y = lst + mat32.FromFixed(sty.Font.Face.Face.Metrics().Ascent) - mat32.FromFixed(sty.Font.Face.Face.Metrics().Descent)
	pos.X = float32(tv.ScBBox.Min.X) + spc.Pos().X

	tv.LineNoRender.Render(rs, pos)
	// todo: need an SvgRender interface that just takes an svg file or object
	// and renders it to a given bitmap, and then just keep that around.
	// if icnm, ok := tv.Buf.LineIcons[ln]; ok {
	// 	ic := tv.Buf.Icons[icnm]
	// 	ic.Par = tv
	// 	ic.Scene = tv.Sc
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
	// 	draw.Draw(tv.Sc.Pixels, r, sic.Pixels, image.Point{}, draw.Over)
	// }
	// if vpUpload {
	// 	tBBox := image.Rectangle{sbox.ToPointFloor(), ebox.ToPointCeil()}
	// 	winoff := tv.ScBBox.Min.Sub(tv.ScBBox.Min)
	// 	tScBBox := tBBox.Add(winoff)
	// 	sc.This().(gi.Scene).ScUploadRegion(tBBox, tScBBox)
	// }
}

// RenderLines displays a specific range of lines on the screen, also painting
// selection.  end is *inclusive* line.  returns false if nothing visible.
func (tv *View) RenderLines(st, ed int) bool {
	if tv == nil || tv.This() == nil || tv.Buf == nil {
		return false
	}
	if !tv.This().(gi.Widget).IsVisible() {
		return false
	}
	if st >= tv.NLines {
		st = tv.NLines - 1
	}
	if ed >= tv.NLines {
		ed = tv.NLines - 1
	}
	if st > ed {
		return false
	}
	sc := tv.Sc
	updt := tv.UpdateStart()
	sty := &tv.Styles
	rs := &sc.RenderState
	pc := &rs.Paint
	pos := tv.RenderStartPos()
	var boxMin, boxMax mat32.Vec2
	rs.PushBounds(tv.ScBBox)
	// first get the box to fill
	visSt := -1
	visEd := -1
	for ln := st; ln <= ed; ln++ {
		lst := tv.CharStartPos(lex.Pos{Ln: ln}).Y // note: charstart pos includes descent
		led := lst + mat32.Max(tv.Renders[ln].Size.Y, tv.LineHeight)
		if int(mat32.Ceil(led)) < tv.ScBBox.Min.Y {
			continue
		}
		if int(mat32.Floor(lst)) > tv.ScBBox.Max.Y {
			continue
		}
		lp := pos
		if visSt < 0 {
			visSt = ln
			lp.Y = lst
			boxMin = lp
		}
		visEd = ln // just keep updating
		lp.Y = led
		boxMax = lp
	}
	if !(visSt < 0 && visEd < 0) {
		rs.Lock()
		boxMin.X = float32(tv.ScBBox.Min.X) // go all the way
		boxMax.X = float32(tv.ScBBox.Max.X) // go all the way
		pc.FillBox(rs, boxMin, boxMax.Sub(boxMin), &sty.BackgroundColor)
		// fmt.Printf("lns: st: %v ed: %v vis st: %v ed %v box: min %v max: %v\n", st, ed, visSt, visEd, boxMin, boxMax)

		tv.RenderDepthBg(visSt, visEd)
		tv.RenderHighlights(visSt, visEd)
		tv.RenderScopelights(visSt, visEd)
		tv.RenderSelect()
		tv.RenderLineNosBox(visSt, visEd)

		if tv.HasLineNos() {
			for ln := visSt; ln <= visEd; ln++ {
				tv.RenderLineNo(ln, true, false)
			}
			tbb := tv.ScBBox
			tbb.Min.X += int(tv.LineNoOff)
			rs.Unlock()
			rs.PushBounds(tbb)
			rs.Lock()
		}
		for ln := visSt; ln <= visEd; ln++ {
			lst := pos.Y + tv.Offs[ln]
			lp := pos
			lp.Y = lst
			lp.X += tv.LineNoOff
			tv.Renders[ln].Render(rs, lp) // not top pos -- already has baseline offset
		}
		rs.Unlock()
		if tv.HasLineNos() {
			rs.PopBounds()
		}

		tBBox := image.Rectangle{boxMin.ToPointFloor(), boxMax.ToPointCeil()}
		winoff := tv.ScBBox.Min.Sub(tv.ScBBox.Min)
		tScBBox := tBBox.Add(winoff)
		_ = tScBBox
		// fmt.Printf("Render lines upload: tbbox: %v  twinbbox: %v\n", tBBox, tScBBox)
		// sc.This().(gi.Scene).ScUploadRegion(tBBox, tScBBox)
	}
	tv.PopBounds(sc)
	tv.UpdateEnd(updt)
	return true
}

// FirstVisibleLine finds the first visible line, starting at given line
// (typically cursor -- if zero, a visible line is first found) -- returns
// stln if nothing found above it.
func (tv *View) FirstVisibleLine(stln int) int {
	if stln == 0 {
		perln := float32(tv.LinesSize.Y) / float32(tv.NLines)
		stln = int(float32(tv.ScBBox.Min.Y-tv.ObjBBox.Min.Y)/perln) - 1
		if stln < 0 {
			stln = 0
		}
		for ln := stln; ln < tv.NLines; ln++ {
			cpos := tv.CharStartPos(lex.Pos{Ln: ln})
			if int(mat32.Floor(cpos.Y)) >= tv.ScBBox.Min.Y { // top definitely on screen
				stln = ln
				break
			}
		}
	}
	lastln := stln
	for ln := stln - 1; ln >= 0; ln-- {
		cpos := tv.CharStartPos(lex.Pos{Ln: ln})
		if int(mat32.Ceil(cpos.Y)) < tv.ScBBox.Min.Y { // top just offscreen
			break
		}
		lastln = ln
	}
	return lastln
}

// LastVisibleLine finds the last visible line, starting at given line
// (typically cursor) -- returns stln if nothing found beyond it.
func (tv *View) LastVisibleLine(stln int) int {
	lastln := stln
	for ln := stln + 1; ln < tv.NLines; ln++ {
		pos := lex.Pos{Ln: ln}
		cpos := tv.CharStartPos(pos)
		if int(mat32.Floor(cpos.Y)) > tv.ScBBox.Max.Y { // just offscreen
			break
		}
		lastln = ln
	}
	return lastln
}

// PixelToCursor finds the cursor position that corresponds to the given pixel
// location (e.g., from mouse click) which has had ScBBox.Min subtracted from
// it (i.e, relative to upper left of text area)
func (tv *View) PixelToCursor(pt image.Point) lex.Pos {
	if tv.NLines == 0 {
		return lex.PosZero
	}
	sty := &tv.Styles
	yoff := float32(tv.ScBBox.Min.Y)
	stln := tv.FirstVisibleLine(0)
	cln := stln
	fls := tv.CharStartPos(lex.Pos{Ln: stln}).Y - yoff
	if pt.Y < int(mat32.Floor(fls)) {
		cln = stln
	} else if pt.Y > tv.ScBBox.Max.Y {
		cln = tv.NLines - 1
	} else {
		got := false
		for ln := stln; ln < tv.NLines; ln++ {
			ls := tv.CharStartPos(lex.Pos{Ln: ln}).Y - yoff
			es := ls
			es += mat32.Max(tv.Renders[ln].Size.Y, tv.LineHeight)
			if pt.Y >= int(mat32.Floor(ls)) && pt.Y < int(mat32.Ceil(es)) {
				got = true
				cln = ln
				break
			}
		}
		if !got {
			cln = tv.NLines - 1
		}
	}
	// fmt.Printf("cln: %v  pt: %v\n", cln, pt)
	lnsz := tv.Buf.LineLen(cln)
	if lnsz == 0 {
		return lex.Pos{Ln: cln, Ch: 0}
	}
	xoff := float32(tv.ScBBox.Min.X)
	scrl := tv.ScBBox.Min.X - tv.ObjBBox.Min.X
	nolno := pt.X - int(tv.LineNoOff)
	sc := int(float32(nolno+scrl) / sty.Font.Face.Metrics.Ch)
	sc -= sc / 4
	sc = max(0, sc)
	cch := sc

	si := 0
	spoff := 0
	nspan := len(tv.Renders[cln].Spans)
	lstY := tv.CharStartPos(lex.Pos{Ln: cln}).Y - yoff
	if nspan > 1 {
		si = int((float32(pt.Y) - lstY) / tv.LineHeight)
		si = min(si, nspan-1)
		si = max(si, 0)
		for i := 0; i < si; i++ {
			spoff += len(tv.Renders[cln].Spans[i].Text)
		}
		// fmt.Printf("si: %v  spoff: %v\n", si, spoff)
	}

	ri := sc
	rsz := len(tv.Renders[cln].Spans[si].Text)
	if rsz == 0 {
		return lex.Pos{Ln: cln, Ch: spoff}
	}
	// fmt.Printf("sc: %v  rsz: %v\n", sc, rsz)

	c, _ := tv.Renders[cln].SpanPosToRuneIdx(si, rsz-1) // end
	rsp := mat32.Floor(tv.CharStartPos(lex.Pos{Ln: cln, Ch: c}).X - xoff)
	rep := mat32.Ceil(tv.CharEndPos(lex.Pos{Ln: cln, Ch: c}).X - xoff)
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
			c, _ := tv.Renders[cln].SpanPosToRuneIdx(si, rii)
			rsp = mat32.Floor(tv.CharStartPos(lex.Pos{Ln: cln, Ch: c}).X - xoff)
			rep = mat32.Ceil(tv.CharEndPos(lex.Pos{Ln: cln, Ch: c}).X - xoff)
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
			c, _ := tv.Renders[cln].SpanPosToRuneIdx(si, rii)
			rsp := mat32.Floor(tv.CharStartPos(lex.Pos{Ln: cln, Ch: c}).X - xoff)
			rep := mat32.Ceil(tv.CharEndPos(lex.Pos{Ln: cln, Ch: c}).X - xoff)
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
