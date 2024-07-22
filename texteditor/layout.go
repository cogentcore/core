// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"

	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
)

// maxGrowLines is the maximum number of lines to grow to
// (subject to other styling constraints).
const maxGrowLines = 25

// styleSizes gets the size info based on Style settings.
func (ed *Editor) styleSizes() {
	sty := &ed.Styles
	spc := sty.BoxSpace()
	sty.Font = paint.OpenFont(sty.FontRender(), &sty.UnitContext)
	ed.fontHeight = sty.Font.Face.Metrics.Height
	ed.lineHeight = sty.Text.EffLineHeight(ed.fontHeight)
	ed.fontDescent = math32.FromFixed(ed.Styles.Font.Face.Face.Metrics().Descent)
	ed.fontAscent = math32.FromFixed(ed.Styles.Font.Face.Face.Metrics().Ascent)
	ed.lineNumberDigits = max(1+int(math32.Log10(float32(ed.NumLines))), 3)
	lno := true
	if ed.Buffer != nil {
		lno = ed.Buffer.Options.LineNumbers
	}
	if lno {
		ed.hasLineNumbers = true
		ed.LineNumberOffset = float32(ed.lineNumberDigits+3)*sty.Font.Face.Metrics.Ch + spc.Left // space for icon
	} else {
		ed.hasLineNumbers = false
		ed.LineNumberOffset = 0
	}
}

// updateFromAlloc updates size info based on allocated size:
// NLinesChars, LineNumberOff, LineLayoutSize
func (ed *Editor) updateFromAlloc() {
	sty := &ed.Styles
	asz := ed.Geom.Size.Alloc.Content
	spsz := sty.BoxSpace().Size()
	asz.SetSub(spsz)
	sbw := math32.Ceil(ed.Styles.ScrollBarWidth.Dots)
	asz.X -= sbw
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

func (ed *Editor) internalSizeFromLines() {
	ed.totalSize = ed.linesSize
	ed.totalSize.X += ed.LineNumberOffset
	ed.Geom.Size.Internal = ed.totalSize
	ed.Geom.Size.Internal.Y += ed.lineHeight
}

// layoutAllLines generates paint.Text Renders of lines
// from the Markup version of the source in Buf.
// It computes the total LinesSize and TotalSize.
func (ed *Editor) layoutAllLines() {
	ed.updateFromAlloc()
	if ed.lineLayoutSize.Y == 0 || ed.Styles.Font.Size.Value == 0 {
		return
	}
	if ed.Buffer == nil || ed.Buffer.NumLines() == 0 {
		ed.NumLines = 0
		return
	}
	ed.lastFilename = ed.Buffer.Filename

	ed.NumLines = ed.Buffer.NumLines()
	ed.Buffer.Lock()
	ed.Buffer.Highlighter.TabSize = ed.Styles.Text.TabSize
	buf := ed.Buffer

	nln := ed.NumLines
	if nln >= len(buf.Markup) {
		nln = len(buf.Markup)
	}
	ed.renders = slicesx.SetLength(ed.renders, nln)
	ed.offsets = slicesx.SetLength(ed.offsets, nln)

	sz := ed.lineLayoutSize

	sty := &ed.Styles
	fst := sty.FontRender()
	fst.Background = nil
	off := float32(0)
	mxwd := sz.X // always start with our render size

	ed.hasLinks = false
	for ln := 0; ln < nln; ln++ {
		if ln >= len(ed.renders) || ln >= len(buf.Markup) {
			break
		}
		rn := &ed.renders[ln]
		rn.SetHTMLPre(buf.Markup[ln], fst, &sty.Text, &sty.UnitContext, ed.textStyleProperties())
		rn.Layout(&sty.Text, sty.FontRender(), &sty.UnitContext, sz)
		if !ed.hasLinks && len(rn.Links) > 0 {
			ed.hasLinks = true
		}
		ed.offsets[ln] = off
		lsz := math32.Ceil(math32.Max(rn.BBox.Size().Y, ed.lineHeight))
		off += lsz
		mxwd = math32.Max(mxwd, rn.BBox.Size().X)
	}
	ed.Buffer.Unlock()
	ed.linesSize = math32.Vec2(mxwd, off)
	ed.lastlineLayoutSize = ed.lineLayoutSize
	ed.internalSizeFromLines()
}

// reLayoutAllLines updates the Renders Layout given current size, if changed
func (ed *Editor) reLayoutAllLines() {
	ed.updateFromAlloc()
	if ed.lineLayoutSize.Y == 0 || ed.Styles.Font.Size.Value == 0 {
		return
	}
	if ed.Buffer == nil || ed.Buffer.NumLines() == 0 {
		return
	}
	if ed.lastlineLayoutSize == ed.lineLayoutSize {
		ed.internalSizeFromLines()
		return
	}
	ed.layoutAllLines()
}

// note: Layout reverts to basic Widget behavior for layout if no kids, like us..

func (ed *Editor) SizeUp() {
	ed.Frame.SizeUp() // sets Actual size based on styles
	sz := &ed.Geom.Size
	if ed.Buffer == nil {
		return
	}
	nln := ed.Buffer.NumLines()
	if nln == 0 {
		return
	}
	if ed.Styles.Grow.Y == 0 {
		maxh := maxGrowLines * ed.lineHeight
		ty := styles.ClampMin(styles.ClampMax(min(float32(nln+1)*ed.lineHeight, maxh), sz.Max.Y), sz.Min.Y)
		sz.Actual.Content.Y = ty
		sz.Actual.Total.Y = sz.Actual.Content.Y + sz.Space.Y
		if core.DebugSettings.LayoutTrace {
			fmt.Println(ed, "texteditor SizeUp targ:", ty, "nln:", nln, "Actual:", sz.Actual.Content)
		}
	}
}

func (ed *Editor) SizeDown(iter int) bool {
	if iter == 0 {
		ed.layoutAllLines()
	} else {
		ed.reLayoutAllLines()
	}
	// use actual lineSize from layout to ensure fit
	sz := &ed.Geom.Size
	maxh := maxGrowLines * ed.lineHeight
	ty := ed.linesSize.Y + 1*ed.lineHeight
	ty = styles.ClampMin(styles.ClampMax(min(ty, maxh), sz.Max.Y), sz.Min.Y)
	if ed.Styles.Grow.Y == 0 {
		sz.Actual.Content.Y = ty
		sz.Actual.Total.Y = sz.Actual.Content.Y + sz.Space.Y
	}
	if core.DebugSettings.LayoutTrace {
		fmt.Println(ed, "texteditor SizeDown targ:", ty, "linesSize:", ed.linesSize.Y, "Actual:", sz.Actual.Content)
	}

	redo := ed.Frame.SizeDown(iter)
	if ed.Styles.Grow.Y == 0 {
		sz.Actual.Content.Y = ty
		sz.Actual.Content.Y = min(ty, sz.Alloc.Content.Y)
	}
	sz.Actual.Total.Y = sz.Actual.Content.Y + sz.Space.Y
	chg := ed.ManageOverflow(iter, true) // this must go first.
	return redo || chg
}

func (ed *Editor) SizeFinal() {
	ed.Frame.SizeFinal()
	ed.reLayoutAllLines()
}

func (ed *Editor) Position() {
	ed.Frame.Position()
	ed.ConfigScrolls()
}

func (ed *Editor) ApplyScenePos() {
	ed.Frame.ApplyScenePos()
	ed.PositionScrolls()
}

// layoutLine generates render of given line (including highlighting).
// If the line with exceeds the current maximum, or the number of effective
// lines (e.g., from word-wrap) is different, then NeedsLayout is called
// and it returns true.
func (ed *Editor) layoutLine(ln int) bool {
	if ed.Buffer == nil || ed.Buffer.NumLines() == 0 || ln >= len(ed.renders) {
		return false
	}
	sty := &ed.Styles
	fst := sty.FontRender()
	fst.Background = nil
	mxwd := float32(ed.linesSize.X)
	needLay := false

	rn := &ed.renders[ln]
	curspans := len(rn.Spans)
	ed.Buffer.Lock()
	rn.SetHTMLPre(ed.Buffer.Markup[ln], fst, &sty.Text, &sty.UnitContext, ed.textStyleProperties())
	ed.Buffer.Unlock()
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

	if needLay {
		ed.NeedsLayout()
	} else {
		ed.NeedsRender()
	}
	return needLay
}
