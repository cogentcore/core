// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/paint"
	"goki.dev/mat32/v2"
)

// StyleSizes gets the size info based on Style settings.
func (ed *Editor) StyleSizes() {
	sty := &ed.Styles
	spc := sty.BoxSpace()
	sty.Font = paint.OpenFont(sty.FontRender(), &sty.UnContext)
	ed.FontHeight = sty.Font.Face.Metrics.Height
	ed.LineHeight = sty.Text.EffLineHeight(ed.FontHeight)
	ed.FontDescent = mat32.FromFixed(ed.Styles.Font.Face.Face.Metrics().Descent)
	ed.LineNoDigs = max(1+int(mat32.Log10(float32(ed.NLines))), 3)
	lno := true
	if ed.Buf != nil {
		lno = ed.Buf.Opts.LineNos
	}
	if lno {
		ed.SetFlag(true, EditorHasLineNos)
		// SidesTODO: this is sketchy
		ed.LineNoOff = float32(ed.LineNoDigs+3)*sty.Font.Face.Metrics.Ch + spc.Left // space for icon
	} else {
		ed.SetFlag(false, EditorHasLineNos)
		ed.LineNoOff = 0
	}
}

// UpdateFromAlloc updates size info based on allocated size:
// NLinesChars, LineNoOff, LineLayoutSize
func (ed *Editor) UpdateFromAlloc() {
	sty := &ed.Styles
	asz := ed.Geom.Size.Actual.Content
	ed.LineLayoutSize = asz
	nv := mat32.Vec2{}
	if asz == nv {
		ed.NLinesChars.Y = 40
		ed.NLinesChars.X = 80
	} else {
		ed.NLinesChars.Y = int(mat32.Floor(float32(asz.Y) / ed.LineHeight))
		ed.NLinesChars.X = int(mat32.Floor(float32(asz.X) / sty.Font.Face.Metrics.Ch))
	}
	ed.LineLayoutSize.X -= ed.LineNoOff
}

// note: Layout reverts to basic Widget behavior for layout if no kids, like us..

func (ed *Editor) SizeFinal(sc *gi.Scene) {
	sz := &ed.Geom.Size
	ed.ManageOverflow(sc, 0, false)
	ed.Layout.SizeFinal(sc)
	sbw := mat32.Ceil(ed.Styles.ScrollBarWidth.Dots)
	// sz.Actual.Content.X -= sbw // anticipate scroll
	ed.UpdateFromAlloc()
	ed.LayoutAllLines()
	// fmt.Println("final pre manage, actual:", sz.Actual, "space:", sz.Space)
	if ed.ManageOverflow(sc, 3, false) {
		if ed.HasScroll[mat32.X] {
			sz.Actual.Total.Y -= sbw
		}
		if ed.HasScroll[mat32.Y] {
			sz.Actual.Total.X -= sbw
		}
		sz.SetContentFromTotal(&sz.Actual) // reduce content
		// fmt.Println("adding scrolls, actual:", sz.Actual, "space:", sz.Space)
		ed.UpdateFromAlloc()
		ed.LayoutAllLines()
	}
}

func (ed *Editor) Position(sc *gi.Scene) {
	ed.Layout.Position(sc)
	ed.ConfigScrolls(sc)
}

func (ed *Editor) ScenePos(sc *gi.Scene) {
	ed.Layout.ScenePos(sc)
	ed.GetScrollPosition(sc)
	ed.PositionScrolls(sc)
}

// LayoutAllLines generates TextRenders of lines
// from the Markup version of the source in Buf.
// It computes the total LinesSize and TotalSize.
func (ed *Editor) LayoutAllLines() {
	if ed.LineLayoutSize == mat32.Vec2Zero || ed.Styles.Font.Size.Val == 0 {
		return
	}
	if ed.Buf == nil || ed.Buf.NumLines() == 0 {
		ed.NLines = 0
		return
	}
	ed.lastFilename = ed.Buf.Filename

	ed.Buf.Hi.TabSize = ed.Styles.Text.TabSize
	// fmt.Printf("layout all: %v\n", ed.Nm)

	ed.NLines = ed.Buf.NumLines()
	nln := ed.NLines
	if cap(ed.Renders) >= nln {
		ed.Renders = ed.Renders[:nln]
	} else {
		ed.Renders = make([]paint.Text, nln)
	}
	if cap(ed.Offs) >= nln {
		ed.Offs = ed.Offs[:nln]
	} else {
		ed.Offs = make([]float32, nln)
	}

	sz := ed.LineLayoutSize
	// fmt.Println("LineLayoutSize:", sz)

	sty := &ed.Styles
	fst := sty.FontRender()
	fst.BackgroundColor.SetSolid(nil)
	off := float32(0)
	mxwd := sz.X // always start with our render size

	ed.Buf.MarkupMu.RLock()
	ed.HasLinks = false
	for ln := 0; ln < nln; ln++ {
		ed.Renders[ln].SetHTMLPre(ed.Buf.Markup[ln], fst, &sty.Text, &sty.UnContext, ed.TextStyleProps())
		ed.Renders[ln].LayoutStdLR(&sty.Text, sty.FontRender(), &sty.UnContext, sz)
		if !ed.HasLinks && len(ed.Renders[ln].Links) > 0 {
			ed.HasLinks = true
		}
		ed.Offs[ln] = off
		lsz := mat32.Max(ed.Renders[ln].Size.Y, ed.LineHeight)
		off += lsz
		mxwd = mat32.Max(mxwd, ed.Renders[ln].Size.X)
	}
	ed.Buf.MarkupMu.RUnlock()

	ed.LinesSize = mat32.Vec2{mxwd, off}

	spc := sty.BoxSpace()
	ed.TotalSize = ed.LinesSize.Add(spc.Size())
	ed.TotalSize.X += ed.LineNoOff
	ed.Geom.Size.Internal = ed.TotalSize
	// fmt.Println(ed, "internal:", ed.TotalSize)
	// extraHalf := ed.LineHeight * 0.5 * float32(ed.NLinesChars.Y)
	// todo: add extra half to bottom of size?
}

// LayoutLine generates render of given line (including highlighting).
// If the line with exceeds the current maximum, or the number of effective
// lines (e.g., from word-wrap) is different, then SetNeedsLayout is called
// and it returns true.
func (ed *Editor) LayoutLine(ln int) bool {
	if ed.Buf == nil || ed.Buf.NumLines() == 0 {
		return false
	}
	sty := &ed.Styles
	fst := sty.FontRender()
	fst.BackgroundColor.SetSolid(nil)
	mxwd := float32(ed.LinesSize.X)
	needLay := false

	ed.Buf.MarkupMu.RLock()
	curspans := len(ed.Renders[ln].Spans)
	ed.Renders[ln].SetHTMLPre(ed.Buf.Markup[ln], fst, &sty.Text, &sty.UnContext, ed.TextStyleProps())
	ed.Renders[ln].LayoutStdLR(&sty.Text, sty.FontRender(), &sty.UnContext, ed.LineLayoutSize)
	if !ed.HasLinks && len(ed.Renders[ln].Links) > 0 {
		ed.HasLinks = true
	}
	nwspans := len(ed.Renders[ln].Spans)
	if nwspans != curspans && (nwspans > 1 || curspans > 1) {
		needLay = true
	}
	if ed.Renders[ln].Size.X > mxwd {
		needLay = true
	}
	ed.Buf.MarkupMu.RUnlock()

	if needLay {
		ed.SetNeedsLayout()
	} else {
		ed.SetNeedsRender()
	}
	return needLay
}
