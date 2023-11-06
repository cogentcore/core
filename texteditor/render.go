// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"
	"image"
	"image/color"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/lex"
)

// Rendering Notes: all rendering is done in Render call.
// Layout must be called whenever content changes across
// lines.

func (ed *Editor) Render(sc *gi.Scene) {
	if ed.PushBounds(sc) {
		ed.RenderAllLinesInBounds()
		if ed.ScrollToCursorOnRender {
			ed.ScrollToCursorOnRender = false
			ed.CursorPos = ed.ScrollToCursorPos
			ed.ScrollCursorToTop()
		}
		if ed.StateIs(states.Focused) {
			ed.StartCursor()
		} else {
			ed.StopCursor()
		}
		ed.RenderChildren(sc)
		ed.RenderScrolls(sc)
		ed.PopBounds(sc)
	} else {
		ed.StopCursor()
	}
}

// HiStyle applies the highlighting styles from buffer markup style
func (ed *Editor) HiStyle() {
	// STYTODO: need to figure out what to do with histyle
	if !ed.Buf.Hi.HasHi() {
		return
	}
	ed.CSS = ed.Buf.Hi.CSSProps
	if chp, ok := ki.SubProps(ed.CSS, ".chroma"); ok {
		for ky, vl := range chp { // apply to top level
			ed.SetProp(ky, vl)
		}
	}
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
		spos.Y += ed.Offs[pos.Ln] + mat32.FromFixed(ed.Styles.Font.Face.Face.Metrics().Descent)
	}
	if len(ed.Renders[pos.Ln].Spans) > 0 {
		// note: Y from rune pos is baseline
		rrp, _, _, _ := ed.Renders[pos.Ln].RuneRelPos(pos.Ch)
		spos.X += rrp.X
		spos.Y += rrp.Y - ed.Renders[pos.Ln].Spans[0].RelPos.Y // relative
	}
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
	// if pos.Ln >= ed.NLines {
	// 	spos.Y += float32(ed.LinesSize.Y)
	// 	spos.X += ed.LineNoOff
	// 	return spos
	// }
	spos.Y += ed.Offs[pos.Ln] + mat32.FromFixed(ed.Styles.Font.Face.Face.Metrics().Descent)
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

// RenderDepthBg renders the depth background color
func (ed *Editor) RenderDepthBg(stln, edln int) {
	if ed.Buf == nil {
		return
	}
	if !ed.Buf.Opts.DepthColor || ed.IsDisabled() || !ed.StateIs(states.Focused) {
		return
	}
	ed.Buf.MarkupMu.RLock() // needed for HiTags access
	defer ed.Buf.MarkupMu.RUnlock()
	sty := &ed.Styles
	cspec := sty.BackgroundColor
	bg := cspec.Solid
	// STYTODO: fix text editor colors
	// isDark := bg.IsDark()
	// nclrs := len(ViewDepthColors)
	lstdp := 0
	for ln := stln; ln <= edln; ln++ {
		lst := ed.CharStartPos(lex.Pos{Ln: ln}).Y // note: charstart pos includes descent
		led := lst + mat32.Max(ed.Renders[ln].Size.Y, ed.LineHeight)
		if int(mat32.Ceil(led)) < ed.ScBBox.Min.Y {
			continue
		}
		if int(mat32.Floor(lst)) > ed.ScBBox.Max.Y {
			continue
		}
		if ln >= len(ed.Buf.HiTags) { // may be out of sync
			continue
		}
		ht := ed.Buf.HiTags[ln]
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
				ed.RenderRegionBoxSty(reg, sty, &cspec)
			}
		}
		if lstdp > 0 {
			ed.RenderRegionToEnd(lex.Pos{Ln: ln, Ch: lsted}, sty, &cspec)
		}
	}
}

// RenderSelect renders the selection region as a selected background color
// -- always called within context of outer RenderLines or RenderAllLines
func (ed *Editor) RenderSelect() {
	if !ed.HasSelection() {
		return
	}
	ed.RenderRegionBox(ed.SelectReg, &ed.SelectColor)
}

// RenderHighlights renders the highlight regions as a highlighted background
// color -- always called within context of outer RenderLines or
// RenderAllLines
func (ed *Editor) RenderHighlights(stln, edln int) {
	for _, reg := range ed.Highlights {
		reg := ed.Buf.AdjustReg(reg)
		if reg.IsNil() || (stln >= 0 && (reg.Start.Ln > edln || reg.End.Ln < stln)) {
			continue
		}
		ed.RenderRegionBox(reg, &ed.HighlightColor)
	}
}

// RenderScopelights renders a highlight background color for regions
// in the Scopelights list
// -- always called within context of outer RenderLines or RenderAllLines
func (ed *Editor) RenderScopelights(stln, edln int) {
	for _, reg := range ed.Scopelights {
		reg := ed.Buf.AdjustReg(reg)
		if reg.IsNil() || (stln >= 0 && (reg.Start.Ln > edln || reg.End.Ln < stln)) {
			continue
		}
		ed.RenderRegionBox(reg, &ed.HighlightColor)
	}
}

// RenderRegionBox renders a region in background color according to given background color
func (ed *Editor) RenderRegionBox(reg textbuf.Region, bgclr *colors.Full) {
	ed.RenderRegionBoxSty(reg, &ed.Styles, bgclr)
}

// RenderRegionBoxSty renders a region in given style and background color
func (ed *Editor) RenderRegionBoxSty(reg textbuf.Region, sty *styles.Style, bgclr *colors.Full) {
	st := reg.Start
	end := reg.End
	spos := ed.CharStartPos(st)
	epos := ed.CharStartPos(end)
	epos.Y += ed.LineHeight
	if int(mat32.Ceil(epos.Y)) < ed.ScBBox.Min.Y || int(mat32.Floor(spos.Y)) > ed.ScBBox.Max.Y {
		return
	}

	rs := &ed.Sc.RenderState
	pc := &rs.Paint
	spc := sty.BoxSpace()

	rst := ed.RenderStartPos()
	// SidesTODO: this is sketchy
	ex := float32(ed.ScBBox.Max.X) - spc.Right
	sx := rst.X + ed.LineNoOff

	// fmt.Printf("select: %v -- %v\n", st, ed)

	stsi, _, _ := ed.WrappedLineNo(st)
	edsi, _, _ := ed.WrappedLineNo(end)
	if st.Ln == end.Ln && stsi == edsi {
		pc.FillBox(rs, spos, epos.Sub(spos), bgclr) // same line, done
		return
	}
	// on diff lines: fill to end of stln
	seb := spos
	seb.Y += ed.LineHeight
	seb.X = ex
	pc.FillBox(rs, spos, seb.Sub(spos), bgclr)
	sfb := seb
	sfb.X = sx
	if sfb.Y < epos.Y { // has some full box
		efb := epos
		efb.Y -= ed.LineHeight
		efb.X = ex
		pc.FillBox(rs, sfb, efb.Sub(sfb), bgclr)
	}
	sed := epos
	sed.Y -= ed.LineHeight
	sed.X = sx
	pc.FillBox(rs, sed, epos.Sub(sed), bgclr)
}

// RenderRegionToEnd renders a region in given style and background color, to end of line from start
func (ed *Editor) RenderRegionToEnd(st lex.Pos, sty *styles.Style, bgclr *colors.Full) {
	spos := ed.CharStartPos(st)
	epos := spos
	epos.Y += ed.LineHeight
	epos.X = float32(ed.ScBBox.Max.X)
	if int(mat32.Ceil(epos.Y)) < ed.ScBBox.Min.Y || int(mat32.Floor(spos.Y)) > ed.ScBBox.Max.Y {
		return
	}

	rs := &ed.Sc.RenderState
	pc := &rs.Paint

	pc.FillBox(rs, spos, epos.Sub(spos), bgclr) // same line, done
}

// RenderStartPos is absolute rendering start position from our allocpos
func (ed *Editor) RenderStartPos() mat32.Vec2 {
	st := &ed.Styles
	spc := st.BoxSpace()
	pos := ed.Alloc.Pos.Add(spc.Pos())
	delta := mat32.NewVec2FmPoint(ed.LayoutScrollDelta((image.Point{})))
	pos = pos.Add(delta)
	return pos
}

// RenderAllLinesInBounds displays all the visible lines on the screen --
// after PushBounds has already been called
func (ed *Editor) RenderAllLinesInBounds() {
	rs := &ed.Sc.RenderState
	rs.Lock()
	pc := &rs.Paint
	sty := &ed.Styles
	pos := mat32.NewVec2FmPoint(ed.ScBBox.Min)
	epos := mat32.NewVec2FmPoint(ed.ScBBox.Max)
	pc.FillBox(rs, pos, epos.Sub(pos), &sty.BackgroundColor)
	pos = ed.RenderStartPos()
	stln := -1
	edln := -1
	for ln := 0; ln < ed.NLines; ln++ {
		lst := pos.Y + ed.Offs[ln]
		led := lst + mat32.Max(ed.Renders[ln].Size.Y, ed.LineHeight)
		if int(mat32.Ceil(led)) < ed.ScBBox.Min.Y {
			continue
		}
		if int(mat32.Floor(lst)) > ed.ScBBox.Max.Y {
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
		tbb := ed.ScBBox
		tbb.Min.X += int(ed.LineNoOff)
		rs.Unlock()
		rs.PushBounds(tbb)
		rs.Lock()
	}
	for ln := stln; ln <= edln; ln++ {
		lst := pos.Y + ed.Offs[ln]
		lp := pos
		lp.Y = lst
		lp.X += ed.LineNoOff
		ed.Renders[ln].Render(rs, lp) // not top pos -- already has baseline offset
	}
	rs.Unlock()
	if ed.HasLineNos() {
		rs.PopBounds()
	}
}

// RenderLineNosBoxAll renders the background for the line numbers in the LineNumberColor
func (ed *Editor) RenderLineNosBoxAll() {
	if !ed.HasLineNos() {
		return
	}
	rs := &ed.Sc.RenderState
	pc := &rs.Paint
	sty := &ed.Styles
	spc := sty.BoxSpace()
	spos := mat32.NewVec2FmPoint(ed.ScBBox.Min)
	epos := mat32.NewVec2FmPoint(ed.ScBBox.Max)
	// SidesTODO: this is sketchy
	epos.X = spos.X + ed.LineNoOff - spc.Size().X/2
	pc.FillBoxColor(rs, spos, epos.Sub(spos), ed.LineNumberColor.Solid)
}

// RenderLineNosBox renders the background for the line numbers in given range, in the LineNumberColor
func (ed *Editor) RenderLineNosBox(st, end int) {
	if !ed.HasLineNos() {
		return
	}
	rs := &ed.Sc.RenderState
	pc := &rs.Paint
	sty := &ed.Styles
	spc := sty.BoxSpace()
	spos := ed.CharStartPos(lex.Pos{Ln: st})
	spos.X = float32(ed.ScBBox.Min.X)
	epos := ed.CharEndPos(lex.Pos{Ln: end + 1})
	epos.Y -= ed.LineHeight
	// SidesTODO: this is sketchy
	epos.X = spos.X + ed.LineNoOff - spc.Size().X/2
	// fmt.Printf("line box: st %v ed: %v spos %v  epos %v\n", st, ed, spos, epos)
	pc.FillBoxColor(rs, spos, epos.Sub(spos), ed.LineNumberColor.Solid)
}

// RenderLineNo renders given line number -- called within context of other render
// if defFill is true, it fills box color for default background color (use false for batch mode)
// and if vpUpload is true it uploads the rendered region to scene directly
// (only if totally separate from other updates)
func (ed *Editor) RenderLineNo(ln int, defFill bool, vpUpload bool) {
	if !ed.HasLineNos() || ed.Buf == nil {
		return
	}

	sc := ed.Sc
	sty := &ed.Styles
	spc := sty.BoxSpace()
	fst := sty.FontRender()
	rs := &sc.RenderState
	pc := &rs.Paint

	// render fillbox
	sbox := ed.CharStartPos(lex.Pos{Ln: ln})
	sbox.X = float32(ed.ScBBox.Min.X)
	ebox := ed.CharEndPos(lex.Pos{Ln: ln + 1})
	if ln < ed.NLines-1 {
		ebox.Y -= ed.LineHeight
	}
	// SidesTODO: this is sketchy
	ebox.X = sbox.X + ed.LineNoOff - spc.Size().X/2
	bsz := ebox.Sub(sbox)
	lclr, hasLClr := ed.Buf.LineColors[ln]
	if ed.CursorPos.Ln == ln {
		if hasLClr { // split the diff!
			bszhlf := bsz
			bszhlf.X /= 2
			pc.FillBoxColor(rs, sbox, bszhlf, lclr)
			nsp := sbox
			nsp.X += bszhlf.X
			pc.FillBoxColor(rs, nsp, bszhlf, ed.SelectColor.Solid)
		} else {
			pc.FillBoxColor(rs, sbox, bsz, ed.SelectColor.Solid)
		}
	} else if hasLClr {
		pc.FillBoxColor(rs, sbox, bsz, lclr)
	} else if defFill {
		pc.FillBoxColor(rs, sbox, bsz, ed.LineNumberColor.Solid)
	}

	fst.BackgroundColor.SetSolid(nil)
	lfmt := fmt.Sprintf("%d", ed.LineNoDigs)
	lfmt = "%" + lfmt + "d"
	lnstr := fmt.Sprintf(lfmt, ln+1)
	ed.LineNoRender.SetString(lnstr, fst, &sty.UnContext, &sty.Text, true, 0, 0)
	pos := mat32.Vec2{}
	lst := ed.CharStartPos(lex.Pos{Ln: ln}).Y // note: charstart pos includes descent
	pos.Y = lst + mat32.FromFixed(sty.Font.Face.Face.Metrics().Ascent) - mat32.FromFixed(sty.Font.Face.Face.Metrics().Descent)
	pos.X = float32(ed.ScBBox.Min.X) + spc.Pos().X

	ed.LineNoRender.Render(rs, pos)
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
	// 	winoff := ed.ScBBox.Min.Sub(ed.ScBBox.Min)
	// 	tScBBox := tBBox.Add(winoff)
	// 	sc.This().(gi.Scene).ScUploadRegion(tBBox, tScBBox)
	// }
}

// RenderLines displays a specific range of lines on the screen, also painting
// selection.  end is *inclusive* line.  returns false if nothing visible.
func (ed *Editor) RenderLines(st, end int) bool {
	if ed == nil || ed.This() == nil || ed.Buf == nil {
		return false
	}
	if !ed.This().(gi.Widget).IsVisible() {
		return false
	}
	if st >= ed.NLines {
		st = ed.NLines - 1
	}
	if end >= ed.NLines {
		end = ed.NLines - 1
	}
	if st > end {
		return false
	}
	sc := ed.Sc
	updt := ed.UpdateStart()
	sty := &ed.Styles
	rs := &sc.RenderState
	pc := &rs.Paint
	pos := ed.RenderStartPos()
	var boxMin, boxMax mat32.Vec2
	rs.PushBounds(ed.ScBBox)
	// first get the box to fill
	visSt := -1
	visEd := -1
	for ln := st; ln <= end; ln++ {
		lst := ed.CharStartPos(lex.Pos{Ln: ln}).Y // note: charstart pos includes descent
		led := lst + mat32.Max(ed.Renders[ln].Size.Y, ed.LineHeight)
		if int(mat32.Ceil(led)) < ed.ScBBox.Min.Y {
			continue
		}
		if int(mat32.Floor(lst)) > ed.ScBBox.Max.Y {
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
		boxMin.X = float32(ed.ScBBox.Min.X) // go all the way
		boxMax.X = float32(ed.ScBBox.Max.X) // go all the way
		pc.FillBox(rs, boxMin, boxMax.Sub(boxMin), &sty.BackgroundColor)
		// fmt.Printf("lns: st: %v ed: %v vis st: %v ed %v box: min %v max: %v\n", st, ed, visSt, visEd, boxMin, boxMax)

		ed.RenderDepthBg(visSt, visEd)
		ed.RenderHighlights(visSt, visEd)
		ed.RenderScopelights(visSt, visEd)
		ed.RenderSelect()
		ed.RenderLineNosBox(visSt, visEd)

		if ed.HasLineNos() {
			for ln := visSt; ln <= visEd; ln++ {
				ed.RenderLineNo(ln, true, false)
			}
			tbb := ed.ScBBox
			tbb.Min.X += int(ed.LineNoOff)
			rs.Unlock()
			rs.PushBounds(tbb)
			rs.Lock()
		}
		for ln := visSt; ln <= visEd; ln++ {
			lst := pos.Y + ed.Offs[ln]
			lp := pos
			lp.Y = lst
			lp.X += ed.LineNoOff
			ed.Renders[ln].Render(rs, lp) // not top pos -- already has baseline offset
		}
		rs.Unlock()
		if ed.HasLineNos() {
			rs.PopBounds()
		}

		tBBox := image.Rectangle{boxMin.ToPointFloor(), boxMax.ToPointCeil()}
		winoff := ed.ScBBox.Min.Sub(ed.ScBBox.Min)
		tScBBox := tBBox.Add(winoff)
		_ = tScBBox
		// fmt.Printf("Render lines upload: tbbox: %v  twinbbox: %v\n", tBBox, tScBBox)
		// sc.This().(gi.Scene).ScUploadRegion(tBBox, tScBBox)
	}
	ed.PopBounds(sc)
	ed.UpdateEnd(updt)
	return true
}

// FirstVisibleLine finds the first visible line, starting at given line
// (typically cursor -- if zero, a visible line is first found) -- returns
// stln if nothing found above it.
func (ed *Editor) FirstVisibleLine(stln int) int {
	if stln == 0 {
		perln := float32(ed.LinesSize.Y) / float32(ed.NLines)
		stln = int(float32(ed.ScBBox.Min.Y-ed.ObjBBox.Min.Y)/perln) - 1
		if stln < 0 {
			stln = 0
		}
		for ln := stln; ln < ed.NLines; ln++ {
			cpos := ed.CharStartPos(lex.Pos{Ln: ln})
			if int(mat32.Floor(cpos.Y)) >= ed.ScBBox.Min.Y { // top definitely on screen
				stln = ln
				break
			}
		}
	}
	lastln := stln
	for ln := stln - 1; ln >= 0; ln-- {
		cpos := ed.CharStartPos(lex.Pos{Ln: ln})
		if int(mat32.Ceil(cpos.Y)) < ed.ScBBox.Min.Y { // top just offscreen
			break
		}
		lastln = ln
	}
	return lastln
}

// LastVisibleLine finds the last visible line, starting at given line
// (typically cursor) -- returns stln if nothing found beyond it.
func (ed *Editor) LastVisibleLine(stln int) int {
	lastln := stln
	for ln := stln + 1; ln < ed.NLines; ln++ {
		pos := lex.Pos{Ln: ln}
		cpos := ed.CharStartPos(pos)
		if int(mat32.Floor(cpos.Y)) > ed.ScBBox.Max.Y { // just offscreen
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
	sty := &ed.Styles
	yoff := float32(ed.ScBBox.Min.Y)
	stln := ed.FirstVisibleLine(0)
	cln := stln
	fls := ed.CharStartPos(lex.Pos{Ln: stln}).Y - yoff
	if pt.Y < int(mat32.Floor(fls)) {
		cln = stln
	} else if pt.Y > ed.ScBBox.Max.Y {
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
	xoff := float32(ed.ScBBox.Min.X)
	scrl := ed.ScBBox.Min.X - ed.ObjBBox.Min.X
	nolno := pt.X - int(ed.LineNoOff)
	sc := int(float32(nolno+scrl) / sty.Font.Face.Metrics.Ch)
	sc -= sc / 4
	sc = max(0, sc)
	cch := sc

	si := 0
	spoff := 0
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
