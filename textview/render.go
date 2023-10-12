// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textview

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"sync"
	"time"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/textview/textbuf"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/lex"
)

// todo: should we try to optimize rendering parts when they change
// e.g., RenderSelectLines or just do the full update whenever anything
// changes?  The problem is that we'd have to tell the renderwin
// to upload but not call our render function..
// just probably not worth it.

// Render does some preliminary work and then calls render on children
func (tv *View) Render(sc *gi.Scene) {
	// fmt.Printf("tv render: %v\n", tv.Nm)
	// if tv.NeedsFullReRender() {
	// 	tv.SetNeedsRefresh()
	// }
	// if tv.FullReRenderIfNeeded() {
	// 	return
	// }
	//
	// if tv.Buf != nil && (tv.NLines != tv.Buf.NumLines() || tv.NeedsRefresh()) {
	// 	tv.LayoutAllLines(false)
	// 	if tv.NeedsRefresh() {
	// 		tv.ClearNeedsRefresh()
	// 	}
	// }

	tv.VisSizes()
	if tv.NLines == 0 {
		ply := tv.ParentLayout()
		if ply != nil {
			tv.ScBBox = ply.ScBBox
		}
	}

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
			// fmt.Printf("tv render: %v  stop cursor\n", tv.Nm)
			tv.StopCursor()
		}
		tv.RenderChildren(sc)
		tv.PopBounds(sc)
	} else {
		// fmt.Printf("tv render: %v  not vis stop cursor\n", tv.Nm)
		tv.StopCursor()
	}
}

// ParentLayout returns our parent layout.
// we ensure this is our immediate parent which is necessary for textview
func (tv *View) ParentLayout() *gi.Layout {
	if tv.Par == nil {
		return nil
	}
	return gi.AsLayout(tv.Par)
}

// RenderSize is the size we should pass to text rendering, based on alloc
func (tv *View) RenderSize() mat32.Vec2 {
	spc := tv.Style.BoxSpace()
	if tv.Par == nil {
		return mat32.Vec2Zero
	}
	parw := tv.ParentLayout()
	if parw == nil {
		log.Printf("giv.View Programmer Error: A View MUST be located within a parent Layout object -- instead parent is %v at: %v\n", tv.Par.KiType(), tv.Path())
		return mat32.Vec2Zero
	}
	paloc := parw.LayState.Alloc.SizeOrig
	if !paloc.IsNil() {
		// fmt.Printf("paloc: %v, psc: %v  lineonoff: %v\n", paloc, parw.ScBBox, tv.LineNoOff)
		tv.RenderSz = paloc.Sub(parw.ExtraSize).Sub(spc.Size())
		// SidesTODO: this is sketchy
		tv.RenderSz.X -= spc.Size().X / 2 // extra space
		// fmt.Printf("alloc rendersz: %v\n", tv.RenderSz)
	} else {
		sz := tv.LayState.Alloc.SizeOrig
		if sz.IsNil() {
			sz = tv.LayState.SizePrefOrMax()
		}
		if !sz.IsNil() {
			sz.SetSub(spc.Size())
		}
		tv.RenderSz = sz
		// fmt.Printf("fallback rendersz: %v\n", tv.RenderSz)
	}
	tv.RenderSz.X -= tv.LineNoOff
	// fmt.Printf("rendersz: %v\n", tv.RenderSz)
	return tv.RenderSz
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

// LayoutAllLines generates TextRenders of lines from our Buf, from the
// Markup version of the source, and returns whether the current rendered size
// is different from what it was previously
func (tv *View) LayoutAllLines(inLayout bool) bool {
	if inLayout && tv.Is(ViewInReLayout) {
		return false
	}
	if tv.Buf == nil || tv.Buf.NumLines() == 0 {
		tv.NLines = 0
		return tv.ResizeIfNeeded(image.Point{})
	}
	tv.StyMu.RLock()
	needSty := tv.Style.Font.Size.Val == 0
	tv.StyMu.RUnlock()
	if needSty {
		// fmt.Print("textview: no style\n")
		return false
		// tv.StyleView() // this fails on mac
	}
	tv.lastFilename = tv.Buf.Filename

	tv.Buf.Hi.TabSize = tv.Style.Text.TabSize
	tv.HiStyle()
	// fmt.Printf("layout all: %v\n", tv.Nm)

	tv.NLines = tv.Buf.NumLines()
	nln := tv.NLines
	if cap(tv.Renders) >= nln {
		tv.Renders = tv.Renders[:nln]
	} else {
		tv.Renders = make([]paint.Text, nln)
	}
	if cap(tv.Offs) >= nln {
		tv.Offs = tv.Offs[:nln]
	} else {
		tv.Offs = make([]float32, nln)
	}

	tv.VisSizes()
	sz := tv.RenderSz

	// fmt.Printf("rendersize: %v\n", sz)
	sty := &tv.Style
	fst := sty.FontRender()
	fst.BackgroundColor.SetSolid(nil)
	off := float32(0)
	mxwd := sz.X // always start with our render size

	tv.Buf.MarkupMu.RLock()
	tv.HasLinks = false
	for ln := 0; ln < nln; ln++ {
		tv.Renders[ln].SetHTMLPre(tv.Buf.Markup[ln], fst, &sty.Text, &sty.UnContext, tv.CSS)
		tv.Renders[ln].LayoutStdLR(&sty.Text, sty.FontRender(), &sty.UnContext, sz)
		if !tv.HasLinks && len(tv.Renders[ln].Links) > 0 {
			tv.HasLinks = true
		}
		tv.Offs[ln] = off
		lsz := mat32.Max(tv.Renders[ln].Size.Y, tv.LineHeight)
		off += lsz
		mxwd = mat32.Max(mxwd, tv.Renders[ln].Size.X)
	}
	tv.Buf.MarkupMu.RUnlock()

	extraHalf := tv.LineHeight * 0.5 * float32(tv.VisSize.Y)
	nwSz := mat32.Vec2{mxwd, off + extraHalf}.ToPointCeil()
	// fmt.Printf("lay lines: diff: %v  old: %v  new: %v\n", diff, tv.LinesSize, nwSz)
	if inLayout {
		tv.LinesSize = nwSz
		return tv.SetSize()
	}
	return tv.ResizeIfNeeded(nwSz)
}

// SetSize updates our size only if larger than our allocation
func (tv *View) SetSize() bool {
	sty := &tv.Style
	spc := sty.BoxSpace()
	rndsz := tv.RenderSz
	rndsz.X += tv.LineNoOff
	netsz := mat32.Vec2{float32(tv.LinesSize.X) + tv.LineNoOff, float32(tv.LinesSize.Y)}
	cursz := tv.LayState.Alloc.Size.Sub(spc.Size())
	if cursz.X < 10 || cursz.Y < 10 {
		nwsz := netsz.Max(rndsz)
		tv.GetSizeFromWH(nwsz.X, nwsz.Y)
		tv.LayState.Size.Need = tv.LayState.Alloc.Size
		tv.LayState.Size.Pref = tv.LayState.Alloc.Size
		return true
	}
	nwsz := netsz.Max(rndsz)
	alloc := tv.LayState.Alloc.Size
	tv.GetSizeFromWH(nwsz.X, nwsz.Y)
	if alloc != tv.LayState.Alloc.Size {
		tv.LayState.Size.Need = tv.LayState.Alloc.Size
		tv.LayState.Size.Pref = tv.LayState.Alloc.Size
		return true
	}
	// fmt.Printf("NO resize: netsz: %v  cursz: %v rndsz: %v\n", netsz, cursz, rndsz)
	return false
}

// ResizeIfNeeded resizes the edit area if different from current setting --
// returns true if resizing was performed
func (tv *View) ResizeIfNeeded(nwSz image.Point) bool {
	if nwSz == tv.LinesSize {
		return false
	}
	// fmt.Printf("%v needs resize: %v\n", tv.Nm, nwSz)
	tv.LinesSize = nwSz
	diff := tv.SetSize()
	if !diff {
		// fmt.Printf("%v resize no setsize: %v\n", tv.Nm, nwSz)
		return false
	}
	ly := tv.ParentLayout()
	if ly != nil {
		tv.SetFlag(true, ViewInReLayout)
		gi.GatherSizes(ly) // can't call GetSize b/c that resets layout
		ly.DoLayoutTree(tv.Sc)
		tv.SetFlag(true, ViewRenderScrolls)
		tv.SetFlag(false, ViewInReLayout)
		// fmt.Printf("resized: %v\n", tv.LayState.Alloc.Size)
	}
	return true
}

// LayoutLines generates render of given range of lines (including
// highlighting). end is *inclusive* line.  isDel means this is a delete and
// thus offsets for all higher lines need to be recomputed.  returns true if
// overall number of effective lines (e.g., from word-wrap) is now different
// than before, and thus a full re-render is needed.
func (tv *View) LayoutLines(st, ed int, isDel bool) bool {
	if tv.Buf == nil || tv.Buf.NumLines() == 0 {
		return false
	}
	sty := &tv.Style
	fst := sty.FontRender()
	fst.BackgroundColor.SetSolid(nil)
	mxwd := float32(tv.LinesSize.X)
	rerend := false

	tv.Buf.MarkupMu.RLock()
	for ln := st; ln <= ed; ln++ {
		curspans := len(tv.Renders[ln].Spans)
		tv.Renders[ln].SetHTMLPre(tv.Buf.Markup[ln], fst, &sty.Text, &sty.UnContext, tv.CSS)
		tv.Renders[ln].LayoutStdLR(&sty.Text, sty.FontRender(), &sty.UnContext, tv.RenderSz)
		if !tv.HasLinks && len(tv.Renders[ln].Links) > 0 {
			tv.HasLinks = true
		}
		nwspans := len(tv.Renders[ln].Spans)
		if nwspans != curspans && (nwspans > 1 || curspans > 1) {
			rerend = true
		}
		mxwd = mat32.Max(mxwd, tv.Renders[ln].Size.X)
	}
	tv.Buf.MarkupMu.RUnlock()

	// update all offsets to end of text
	if rerend || isDel || st != ed {
		ofst := st - 1
		if ofst < 0 {
			ofst = 0
		}
		off := tv.Offs[ofst]
		for ln := ofst; ln < tv.NLines; ln++ {
			tv.Offs[ln] = off
			lsz := mat32.Max(tv.Renders[ln].Size.Y, tv.LineHeight)
			off += lsz
		}
		extraHalf := tv.LineHeight * 0.5 * float32(tv.VisSize.Y)
		nwSz := mat32.Vec2{mxwd, off + extraHalf}.ToPointCeil()
		tv.ResizeIfNeeded(nwSz)
	} else {
		nwSz := mat32.Vec2{mxwd, 0}.ToPointCeil()
		nwSz.Y = tv.LinesSize.Y
		tv.ResizeIfNeeded(nwSz)
	}
	return rerend
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
		spos.Y += tv.Offs[pos.Ln] + mat32.FromFixed(tv.Style.Font.Face.Face.Metrics().Descent)
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
	spos.Y += tv.Offs[pos.Ln] + mat32.FromFixed(tv.Style.Font.Face.Face.Metrics().Descent)
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

// ViewBlinkMu is mutex protecting ViewBlink updating and access
var ViewBlinkMu sync.Mutex

// ViewBlinker is the time.Ticker for blinking cursors for text fields,
// only one of which can be active at at a time
var ViewBlinker *time.Ticker

// BlinkingView is the text field that is blinking
var BlinkingView *View

// ViewSpriteName is the name of the window sprite used for the cursor
var ViewSpriteName = "textview.View.Cursor"

// ViewBlink is function that blinks text field cursor
func ViewBlink() {
	for {
		ViewBlinkMu.Lock()
		if ViewBlinker == nil {
			ViewBlinkMu.Unlock()
			return // shutdown..
		}
		ViewBlinkMu.Unlock()
		<-ViewBlinker.C
		ViewBlinkMu.Lock()
		if BlinkingView == nil || BlinkingView.This() == nil {
			ViewBlinkMu.Unlock()
			continue
		}
		if BlinkingView.Is(ki.Destroyed) || BlinkingView.Is(ki.Deleted) {
			BlinkingView = nil
			ViewBlinkMu.Unlock()
			continue
		}
		tv := BlinkingView
		if tv.Sc == nil || !tv.StateIs(states.Focused) || !tv.This().(gi.Widget).IsVisible() {
			tv.RenderCursor(false)
			BlinkingView = nil
			ViewBlinkMu.Unlock()
			continue
		}
		tv.BlinkOn = !tv.BlinkOn
		tv.RenderCursor(tv.BlinkOn)
		ViewBlinkMu.Unlock()
	}
}

// StartCursor starts the cursor blinking and renders it
func (tv *View) StartCursor() {
	if tv == nil || tv.This() == nil {
		return
	}
	if !tv.This().(gi.Widget).IsVisible() {
		return
	}
	tv.BlinkOn = true
	if gi.CursorBlinkTime == 0 {
		tv.RenderCursor(true)
		return
	}
	ViewBlinkMu.Lock()
	if ViewBlinker == nil {
		ViewBlinker = time.NewTicker(gi.CursorBlinkTime)
		go ViewBlink()
	}
	tv.BlinkOn = true
	tv.RenderCursor(true)
	BlinkingView = tv
	ViewBlinkMu.Unlock()
}

// StopCursor stops the cursor from blinking
func (tv *View) StopCursor() {
	if tv == nil || tv.This() == nil {
		return
	}
	if !tv.This().(gi.Widget).IsVisible() {
		return
	}
	tv.RenderCursor(false)
	ViewBlinkMu.Lock()
	if BlinkingView == tv {
		BlinkingView = nil
	}
	ViewBlinkMu.Unlock()
}

// CursorBBox returns a bounding-box for a cursor at given position
func (tv *View) CursorBBox(pos lex.Pos) image.Rectangle {
	cpos := tv.CharStartPos(pos)
	cbmin := cpos.SubScalar(tv.CursorWidth.Dots)
	cbmax := cpos.AddScalar(tv.CursorWidth.Dots)
	cbmax.Y += tv.FontHeight
	curBBox := image.Rectangle{cbmin.ToPointFloor(), cbmax.ToPointCeil()}
	return curBBox
}

// RenderCursor renders the cursor on or off, as a sprite that is either on or off
func (tv *View) RenderCursor(on bool) {
	if tv == nil || tv.This() == nil {
		return
	}
	if !tv.This().(gi.Widget).IsVisible() {
		return
	}
	if tv.Renders == nil {
		return
	}
	tv.CursorMu.Lock()
	defer tv.CursorMu.Unlock()

	sp := tv.CursorSprite(on)
	if sp == nil {
		return
	}
	sp.Geom.Pos = tv.CharStartPos(tv.CursorPos).ToPointFloor()
}

// CursorSpriteName returns the name of the cursor sprite
func (tv *View) CursorSpriteName() string {
	spnm := fmt.Sprintf("%v-%v", ViewSpriteName, tv.FontHeight)
	return spnm
}

// CursorSprite returns the sprite for the cursor, which is
// only rendered once with a vertical bar, and just activated and inactivated
// depending on render status.
func (tv *View) CursorSprite(on bool) *gi.Sprite {
	sc := tv.Sc
	if sc == nil {
		return nil
	}
	ms := sc.MainStage()
	if ms == nil {
		return nil // only MainStage has sprites
	}
	spnm := tv.CursorSpriteName()
	sp, ok := ms.Sprites.SpriteByName(spnm)
	if !ok {
		bbsz := image.Point{int(mat32.Ceil(tv.CursorWidth.Dots)), int(mat32.Ceil(tv.FontHeight))}
		if bbsz.X < 2 { // at least 2
			bbsz.X = 2
		}
		sp = gi.NewSprite(spnm, bbsz, image.Point{})
		ibox := sp.Pixels.Bounds()
		draw.Draw(sp.Pixels, ibox, &image.Uniform{tv.CursorColor.Solid}, image.Point{}, draw.Src)
		ms.Sprites.Add(sp)
	}
	if on {
		ms.Sprites.ActivateSprite(sp.Name)
	} else {
		ms.Sprites.InactivateSprite(sp.Name)
	}
	return sp
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
	sty := &tv.Style
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

// UpdateHighlights re-renders lines from previous highlights and current
// highlights -- assumed to be within a window update block
func (tv *View) UpdateHighlights(prev []textbuf.Region) {
	for _, ph := range prev {
		ph := tv.Buf.AdjustReg(ph)
		tv.RenderLines(ph.Start.Ln, ph.End.Ln)
	}
	for _, ch := range tv.Highlights {
		ch := tv.Buf.AdjustReg(ch)
		tv.RenderLines(ch.Start.Ln, ch.End.Ln)
	}
}

// ClearHighlights clears the Highlights slice of all regions
func (tv *View) ClearHighlights() {
	if len(tv.Highlights) == 0 {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.Highlights = tv.Highlights[:0]
	// tv.RenderAllLines()
}

// ClearScopelights clears the Highlights slice of all regions
func (tv *View) ClearScopelights() {
	if len(tv.Scopelights) == 0 {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	sl := make([]textbuf.Region, len(tv.Scopelights))
	copy(sl, tv.Scopelights)
	tv.Scopelights = tv.Scopelights[:0]
	for _, si := range sl {
		ln := si.Start.Ln
		tv.RenderLines(ln, ln)
	}
}

// RenderRegionBox renders a region in background color according to given background color
func (tv *View) RenderRegionBox(reg textbuf.Region, bgclr *colors.Full) {
	tv.RenderRegionBoxSty(reg, &tv.Style, bgclr)
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
	st := &tv.Style
	spc := st.BoxSpace()
	pos := tv.LayState.Alloc.Pos.Add(spc.Pos())
	return pos
}

// VisSizes computes the visible size of view given current parameters
func (tv *View) VisSizes() {
	if tv.Style.Font.Size.Val == 0 { // called under lock
		fmt.Println("style in size - shouldn't happen")
		tv.StyleView(tv.Sc)
	}
	sty := &tv.Style
	spc := sty.BoxSpace()
	sty.Font = paint.OpenFont(sty.FontRender(), &sty.UnContext)
	tv.FontHeight = sty.Font.Face.Metrics.Height
	tv.LineHeight = sty.Text.EffLineHeight(tv.FontHeight)
	sz := tv.ScBBox.Size()
	if sz == (image.Point{}) {
		tv.VisSize.Y = 40
		tv.VisSize.X = 80
	} else {
		tv.VisSize.Y = int(mat32.Floor(float32(sz.Y) / tv.LineHeight))
		tv.VisSize.X = int(mat32.Floor(float32(sz.X) / sty.Font.Face.Metrics.Ch))
	}
	tv.LineNoDigs = max(1+int(mat32.Log10(float32(tv.NLines))), 3)
	lno := true
	if tv.Buf != nil {
		lno = tv.Buf.Opts.LineNos
	}
	if lno {
		tv.SetFlag(true, ViewHasLineNos)
		// SidesTODO: this is sketchy
		tv.LineNoOff = float32(tv.LineNoDigs+3)*sty.Font.Face.Metrics.Ch + spc.Left // space for icon
	} else {
		tv.SetFlag(false, ViewHasLineNos)
		tv.LineNoOff = 0
	}
	tv.RenderSize()
}

// todo: don't do this:

// RenderAllLines displays all the visible lines on the screen -- this is
// called outside of update process and has its own bounds check and updating
func (tv *View) RenderAllLines() {
	if tv == nil || tv.This() == nil {
		return
	}
	if !tv.This().(gi.Widget).IsVisible() {
		return
	}
	rs := &tv.Sc.RenderState
	rs.PushBounds(tv.ScBBox)
	updt := tv.UpdateStart()
	tv.RenderAllLinesInBounds()
	tv.PopBounds(tv.Sc)
	// tv.Sc.This().(gi.Scene).ScUploadRegion(tv.ScBBox, tv.ScBBox)
	tv.RenderScrolls()
	tv.UpdateEnd(updt)
}

// RenderAllLinesInBounds displays all the visible lines on the screen --
// after PushBounds has already been called
func (tv *View) RenderAllLinesInBounds() {
	// fmt.Printf("render all: %v\n", tv.Nm)
	rs := &tv.Sc.RenderState
	rs.Lock()
	pc := &rs.Paint
	sty := &tv.Style
	tv.VisSizes()
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
	sty := &tv.Style
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
	sty := &tv.Style
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
	sty := &tv.Style
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

// RenderScrolls renders scrollbars if needed
func (tv *View) RenderScrolls() {
	if tv.Is(ViewRenderScrolls) {
		ly := tv.ParentLayout()
		if ly != nil {
			ly.ReRenderScrolls(tv.Sc)
		}
		tv.SetFlag(false, ViewRenderScrolls)
	}
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
	sty := &tv.Style
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
	tv.RenderScrolls()
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
	sty := &tv.Style
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
