// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textview

import (
	"image"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/paint"
	"goki.dev/mat32/v2"
)

// StyleSizes gets the size info based on Style settings.
func (tv *View) StyleSizes() {
	sty := &tv.Style
	spc := sty.BoxSpace()
	sty.Font = paint.OpenFont(sty.FontRender(), &sty.UnContext)
	tv.FontHeight = sty.Font.Face.Metrics.Height
	tv.LineHeight = sty.Text.EffLineHeight(tv.FontHeight)
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
}

// UpdateFromAlloc updates size info based on allocated size:
// NLinesChars, LineNoOff, LineLayoutSize
func (tv *View) UpdateFromAlloc() {
	sty := &tv.Style
	spc := sty.BoxSpace()
	asz := tv.LayState.Alloc.SizeOrig
	nv := mat32.Vec2{}
	if asz == nv {
		tv.NLinesChars.Y = 40
		tv.NLinesChars.X = 80
	} else {
		tv.NLinesChars.Y = int(mat32.Floor(float32(asz.Y) / tv.LineHeight))
		tv.NLinesChars.X = int(mat32.Floor(float32(asz.X) / sty.Font.Face.Metrics.Ch))
	}
	tv.LineLayoutSize = asz.Sub(tv.ExtraSize).Sub(spc.Size())
	tv.LineLayoutSize.X -= tv.LineNoOff
	// SidesTODO: this is sketchy
	// tv.LineLayoutSize.X -= spc.Size().X / 2 // extra space for word wrap
}

func (tv *View) GetSize(sc *gi.Scene, iter int) {
	tv.InitLayout(sc)
	tv.LayoutAllLines()
	tv.LayState.Size.Need = tv.TotalSize
	tv.LayState.Size.Pref = tv.LayState.Size.Need
	// fmt.Println("GetSize: need:", tv.LayState.Size.Need)
}

func (tv *View) DoLayout(sc *gi.Scene, parBBox image.Rectangle, iter int) bool {
	tv.NeedsRedo = false
	tv.DoLayoutBase(sc, parBBox, iter)
	spc := tv.BoxSpace()
	tv.ChildSize = tv.LayState.Size.Need.Sub(spc.Size()) // we are what we need

	tv.ManageOverflow(sc)
	tv.NeedsRedo = tv.DoLayoutChildren(sc, iter)
	// generally no kids here..
	if !tv.NeedsRedo || iter == 1 {
		delta := tv.LayoutScrollDelta((image.Point{}))
		if delta != (image.Point{}) {
			tv.LayoutScrollChildren(sc, delta) // move is a separate step
		}
	}
	tv.UpdateFromAlloc()
	return tv.NeedsRedo
}

// LayoutAllLines generates TextRenders of lines
// from the Markup version of the source in Buf.
// It computes the total LinesSize and TotalSize.
func (tv *View) LayoutAllLines() {
	if tv.LineLayoutSize == mat32.Vec2Zero || tv.Style.Font.Size.Val == 0 {
		return
	}
	if tv.Buf == nil || tv.Buf.NumLines() == 0 {
		tv.NLines = 0
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

	sz := tv.LineLayoutSize
	// fmt.Println("LineLayoutSize:", sz)

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

	tv.LinesSize = mat32.Vec2{mxwd, off}

	spc := sty.BoxSpace()
	tv.TotalSize = tv.LinesSize.Add(spc.Size())
	tv.TotalSize.X += tv.LineNoOff
	// extraHalf := tv.LineHeight * 0.5 * float32(tv.NLinesChars.Y)
	// todo: add extra half to bottom of size?
}

// LayoutLine generates render of given line (including highlighting).
// If the line with exceeds the current maximum, or the number of effective
// lines (e.g., from word-wrap) is different, then SetNeedsLayout is called
// and it returns true.
func (tv *View) LayoutLine(ln int) bool {
	if tv.Buf == nil || tv.Buf.NumLines() == 0 {
		return false
	}
	sty := &tv.Style
	fst := sty.FontRender()
	fst.BackgroundColor.SetSolid(nil)
	mxwd := float32(tv.LinesSize.X)
	needLay := false

	tv.Buf.MarkupMu.RLock()
	curspans := len(tv.Renders[ln].Spans)
	tv.Renders[ln].SetHTMLPre(tv.Buf.Markup[ln], fst, &sty.Text, &sty.UnContext, tv.CSS)
	tv.Renders[ln].LayoutStdLR(&sty.Text, sty.FontRender(), &sty.UnContext, tv.LineLayoutSize)
	if !tv.HasLinks && len(tv.Renders[ln].Links) > 0 {
		tv.HasLinks = true
	}
	nwspans := len(tv.Renders[ln].Spans)
	if nwspans != curspans && (nwspans > 1 || curspans > 1) {
		needLay = true
	}
	if tv.Renders[ln].Size.X > mxwd {
		needLay = true
	}
	tv.Buf.MarkupMu.RUnlock()

	if needLay {
		tv.SetNeedsLayout()
	} else {
		tv.SetNeedsRender()
	}
	return needLay
}
