// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/paint"
)

// StyleSizes gets the size info based on Style settings.
func (ed *Editor) StyleSizes() {
	sty := &ed.Styles
	spc := sty.BoxSpace()
	sty.Font = paint.OpenFont(sty.FontRender(), &sty.UnitContext)
	ed.FontHeight = sty.Font.Face.Metrics.Height
	ed.LineHeight = sty.Text.EffLineHeight(ed.FontHeight)
	ed.FontDescent = mat32.FromFixed(ed.Styles.Font.Face.Face.Metrics().Descent)
	ed.FontAscent = mat32.FromFixed(ed.Styles.Font.Face.Face.Metrics().Ascent)
	ed.LineNoDigs = max(1+int(mat32.Log10(float32(ed.NLines))), 3)
	lno := true
	if ed.Buffer != nil {
		lno = ed.Buffer.Opts.LineNos
	}
	if lno {
		ed.SetFlag(true, EditorHasLineNos)
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
	asz := ed.Geom.Size.Alloc.Content
	sbw := mat32.Ceil(ed.Styles.ScrollBarWidth.Dots)
	// if ed.HasScroll[mat32.Y] {
	asz.X -= sbw
	// }
	if ed.HasScroll[mat32.X] {
		asz.Y -= sbw
	}
	ed.LineLayoutSize = asz
	nv := mat32.Vec2{}
	if asz == nv {
		ed.NLinesChars.Y = 20
		ed.NLinesChars.X = 80
	} else {
		ed.NLinesChars.Y = int(mat32.Floor(float32(asz.Y) / ed.LineHeight))
		if sty.Font.Face != nil {
			ed.NLinesChars.X = int(mat32.Floor(float32(asz.X) / sty.Font.Face.Metrics.Ch))
		}
	}
	ed.LineLayoutSize.X -= ed.LineNoOff
}

func (ed *Editor) InternalSizeFromLines() {
	sty := &ed.Styles
	spc := sty.BoxSpace()
	ed.TotalSize = ed.LinesSize.Add(spc.Size())
	ed.TotalSize.X += ed.LineNoOff
	ed.Geom.Size.Internal = ed.TotalSize
	ed.Geom.Size.Internal.Y += ed.LineHeight
}

// LayoutAllLines generates TextRenders of lines
// from the Markup version of the source in Buf.
// It computes the total LinesSize and TotalSize.
func (ed *Editor) LayoutAllLines() {
	ed.UpdateFromAlloc()
	if ed.LineLayoutSize.Y == 0 || ed.Styles.Font.Size.Value == 0 {
		return
	}
	if ed.Buffer == nil || ed.Buffer.NumLines() == 0 {
		ed.NLines = 0
		return
	}
	ed.lastFilename = ed.Buffer.Filename

	ed.Buffer.Hi.TabSize = ed.Styles.Text.TabSize
	ed.NLines = ed.Buffer.NumLines()
	buf := ed.Buffer
	buf.MarkupMu.RLock()

	nln := ed.NLines
	if nln >= len(buf.Markup) {
		nln = len(buf.Markup)
	}
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

	sty := &ed.Styles
	fst := sty.FontRender()
	fst.Background = nil
	off := float32(0)
	mxwd := sz.X // always start with our render size

	ed.HasLinks = false
	for ln := 0; ln < nln; ln++ {
		if ln >= len(ed.Renders) || ln >= len(buf.Markup) {
			break
		}
		ed.Renders[ln].SetHTMLPre(buf.Markup[ln], fst, &sty.Text, &sty.UnitContext, ed.TextStyleProps())
		ed.Renders[ln].Layout(&sty.Text, sty.FontRender(), &sty.UnitContext, sz)
		if !ed.HasLinks && len(ed.Renders[ln].Links) > 0 {
			ed.HasLinks = true
		}
		ed.Offs[ln] = off
		lsz := mat32.Max(ed.Renders[ln].Size.Y, ed.LineHeight)
		off += lsz
		mxwd = mat32.Max(mxwd, ed.Renders[ln].Size.X)
	}
	buf.MarkupMu.RUnlock()
	ed.LinesSize = mat32.V2(mxwd, off)
	ed.lastlineLayoutSize = ed.LineLayoutSize
	ed.InternalSizeFromLines()
}

// ReLayoutAllLines updates the Renders Layout given current size, if changed
func (ed *Editor) ReLayoutAllLines() {
	ed.UpdateFromAlloc()
	if ed.LineLayoutSize.Y == 0 || ed.Styles.Font.Size.Value == 0 {
		return
	}
	if ed.Buffer == nil || ed.Buffer.NumLines() == 0 {
		return
	}
	if ed.lastlineLayoutSize == ed.LineLayoutSize {
		ed.InternalSizeFromLines()
		return
	}
	buf := ed.Buffer
	buf.MarkupMu.RLock()

	nln := ed.NLines
	if nln >= len(buf.Markup) {
		nln = len(buf.Markup)
	}
	sz := ed.LineLayoutSize

	sty := &ed.Styles
	fst := sty.FontRender()
	fst.Background = nil
	off := float32(0)
	mxwd := sz.X // always start with our render size

	for ln := 0; ln < nln; ln++ {
		if ln >= len(ed.Renders) || ln >= len(buf.Markup) {
			break
		}
		ed.Renders[ln].Layout(&sty.Text, sty.FontRender(), &sty.UnitContext, sz)
		ed.Offs[ln] = off
		lsz := mat32.Max(ed.Renders[ln].Size.Y, ed.LineHeight)
		off += lsz
		mxwd = mat32.Max(mxwd, ed.Renders[ln].Size.X)
	}
	buf.MarkupMu.RUnlock()

	ed.LinesSize = mat32.V2(mxwd, off)
	ed.lastlineLayoutSize = ed.LineLayoutSize
	ed.InternalSizeFromLines()
}

// note: Layout reverts to basic Widget behavior for layout if no kids, like us..

func (ed *Editor) SizeUp() {
	ed.Layout.SizeUp()
	ed.LayoutAllLines() // initial
}

func (ed *Editor) SizeDown(iter int) bool {
	redo := ed.Layout.SizeDown(iter)
	chg := ed.ManageOverflow(iter, true) // this must go first.
	ed.ReLayoutAllLines()
	return redo || chg
}

func (ed *Editor) SizeFinal() {
	ed.Layout.SizeFinal()
	ed.ReLayoutAllLines()
}

func (ed *Editor) Position() {
	ed.Layout.Position()
	ed.ConfigScrolls()
}

func (ed *Editor) ScenePos() {
	ed.Layout.ScenePos()
	ed.PositionScrolls()
}

// LayoutLine generates render of given line (including highlighting).
// If the line with exceeds the current maximum, or the number of effective
// lines (e.g., from word-wrap) is different, then NeedsLayout is called
// and it returns true.
func (ed *Editor) LayoutLine(ln int) bool {
	if ed.Buffer == nil || ed.Buffer.NumLines() == 0 {
		return false
	}
	sty := &ed.Styles
	fst := sty.FontRender()
	fst.Background = nil
	mxwd := float32(ed.LinesSize.X)
	needLay := false

	ed.Buffer.MarkupMu.RLock()
	curspans := len(ed.Renders[ln].Spans)
	ed.Renders[ln].SetHTMLPre(ed.Buffer.Markup[ln], fst, &sty.Text, &sty.UnitContext, ed.TextStyleProps())
	ed.Renders[ln].Layout(&sty.Text, sty.FontRender(), &sty.UnitContext, ed.LineLayoutSize)
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
	ed.Buffer.MarkupMu.RUnlock()

	if needLay {
		ed.NeedsLayout()
	} else {
		ed.NeedsRender()
	}
	return needLay
}
