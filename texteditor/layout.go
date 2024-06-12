// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
)

// StyleSizes gets the size info based on Style settings.
func (ed *Editor) StyleSizes() {
	sty := &ed.Styles
	spc := sty.BoxSpace()
	sty.Font = paint.OpenFont(sty.FontRender(), &sty.UnitContext)
	ed.fontHeight = sty.Font.Face.Metrics.Height
	ed.lineHeight = sty.Text.EffLineHeight(ed.fontHeight)
	ed.fontDescent = math32.FromFixed(ed.Styles.Font.Face.Face.Metrics().Descent)
	ed.fontAscent = math32.FromFixed(ed.Styles.Font.Face.Face.Metrics().Ascent)
	ed.LineNumberDigits = max(1+int(math32.Log10(float32(ed.NLines))), 3)
	lno := true
	if ed.Buffer != nil {
		lno = ed.Buffer.Options.LineNumbers
	}
	if lno {
		ed.SetFlag(true, EditorHasLineNumbers)
		ed.LineNumberOffset = float32(ed.LineNumberDigits+3)*sty.Font.Face.Metrics.Ch + spc.Left // space for icon
	} else {
		ed.SetFlag(false, EditorHasLineNumbers)
		ed.LineNumberOffset = 0
	}
}

// UpdateFromAlloc updates size info based on allocated size:
// NLinesChars, LineNumberOff, LineLayoutSize
func (ed *Editor) UpdateFromAlloc() {
	sty := &ed.Styles
	asz := ed.Geom.Size.Alloc.Content
	spsz := sty.BoxSpace().Size()
	asz.SetSub(spsz)
	sbw := math32.Ceil(ed.Styles.ScrollBarWidth.Dots)
	// if ed.HasScroll[math32.Y] {
	asz.X -= sbw
	// }
	if ed.HasScroll[math32.X] {
		asz.Y -= sbw
	}
	ed.lineLayoutSize = asz

	if asz == (math32.Vector2{}) {
		ed.nLinesChars.Y = 20
		ed.nLinesChars.X = 80
	} else {
		ed.nLinesChars.Y = int(math32.Floor(float32(asz.Y) / ed.lineHeight))
		if sty.Font.Face != nil {
			ed.nLinesChars.X = int(math32.Floor(float32(asz.X) / sty.Font.Face.Metrics.Ch))
		}
	}
	ed.lineLayoutSize.X -= ed.LineNumberOffset
}

func (ed *Editor) InternalSizeFromLines() {
	// sty := &ed.Styles
	// spc := sty.BoxSpace()
	ed.totalSize = ed.linesSize // .Add(spc.Size())
	ed.totalSize.X += ed.LineNumberOffset
	ed.Geom.Size.Internal = ed.totalSize
	ed.Geom.Size.Internal.Y += ed.lineHeight
	// fmt.Println(ed, "total:", ed.TotalSize)
}

// LayoutAllLines generates paint.Text Renders of lines
// from the Markup version of the source in Buf.
// It computes the total LinesSize and TotalSize.
func (ed *Editor) LayoutAllLines() {
	ed.UpdateFromAlloc()
	if ed.lineLayoutSize.Y == 0 || ed.Styles.Font.Size.Value == 0 {
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
	if cap(ed.Offsets) >= nln {
		ed.Offsets = ed.Offsets[:nln]
	} else {
		ed.Offsets = make([]float32, nln)
	}

	sz := ed.lineLayoutSize

	sty := &ed.Styles
	fst := sty.FontRender()
	fst.Background = nil
	off := float32(0)
	mxwd := sz.X // always start with our render size

	ed.hasLinks = false
	for ln := 0; ln < nln; ln++ {
		if ln >= len(ed.Renders) || ln >= len(buf.Markup) {
			break
		}
		rn := &ed.Renders[ln]
		rn.SetHTMLPre(buf.Markup[ln], fst, &sty.Text, &sty.UnitContext, ed.TextStyleProperties())
		rn.Layout(&sty.Text, sty.FontRender(), &sty.UnitContext, sz)
		if !ed.hasLinks && len(rn.Links) > 0 {
			ed.hasLinks = true
		}
		ed.Offsets[ln] = off
		lsz := math32.Ceil(math32.Max(rn.BBox.Size().Y, ed.lineHeight))
		off += lsz
		mxwd = math32.Max(mxwd, rn.BBox.Size().X)
	}
	buf.MarkupMu.RUnlock()
	ed.linesSize = math32.Vec2(mxwd, off)
	ed.lastlineLayoutSize = ed.lineLayoutSize
	ed.InternalSizeFromLines()
}

// ReLayoutAllLines updates the Renders Layout given current size, if changed
func (ed *Editor) ReLayoutAllLines() {
	ed.UpdateFromAlloc()
	if ed.lineLayoutSize.Y == 0 || ed.Styles.Font.Size.Value == 0 {
		return
	}
	if ed.Buffer == nil || ed.Buffer.NumLines() == 0 {
		return
	}
	if ed.lastlineLayoutSize == ed.lineLayoutSize {
		ed.InternalSizeFromLines()
		return
	}
	ed.LayoutAllLines()
}

// note: Layout reverts to basic Widget behavior for layout if no kids, like us..

func (ed *Editor) SizeDown(iter int) bool {
	if iter == 0 {
		ed.LayoutAllLines()
	} else {
		ed.ReLayoutAllLines()
	}
	redo := ed.Frame.SizeDown(iter)
	chg := ed.ManageOverflow(iter, true) // this must go first.
	return redo || chg
}

func (ed *Editor) SizeFinal() {
	ed.Frame.SizeFinal()
	ed.ReLayoutAllLines()
}

func (ed *Editor) Position() {
	ed.Frame.Position()
	ed.ConfigScrolls()
}

func (ed *Editor) ScenePos() {
	ed.Frame.ScenePos()
	ed.PositionScrolls()
}

// LayoutLine generates render of given line (including highlighting).
// If the line with exceeds the current maximum, or the number of effective
// lines (e.g., from word-wrap) is different, then NeedsLayout is called
// and it returns true.
func (ed *Editor) LayoutLine(ln int) bool {
	if ed.Buffer == nil || ed.Buffer.NumLines() == 0 || ln >= len(ed.Renders) {
		return false
	}
	sty := &ed.Styles
	fst := sty.FontRender()
	fst.Background = nil
	mxwd := float32(ed.linesSize.X)
	needLay := false

	ed.Buffer.MarkupMu.RLock()
	rn := &ed.Renders[ln]
	curspans := len(rn.Spans)
	rn.SetHTMLPre(ed.Buffer.Markup[ln], fst, &sty.Text, &sty.UnitContext, ed.TextStyleProperties())
	rn.Layout(&sty.Text, sty.FontRender(), &sty.UnitContext, ed.lineLayoutSize)
	if !ed.hasLinks && len(rn.Links) > 0 {
		ed.hasLinks = true
	}
	nwspans := len(rn.Spans)
	if nwspans != curspans && (nwspans > 1 || curspans > 1) {
		needLay = true
	}
	if rn.BBox.Size().X > mxwd {
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
