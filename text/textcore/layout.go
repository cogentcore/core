// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"fmt"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
)

// maxGrowLines is the maximum number of lines to grow to
// (subject to other styling constraints).
const maxGrowLines = 25

// styleSizes gets the size info based on Style settings.
func (ed *Editor) styleSizes() {
	if ed.Scene == nil {
		ed.charSize.Set(16, 22)
		return
	}
	pc := &ed.Scene.Painter
	sty := &ed.Styles
	r := ed.Scene.TextShaper.FontSize('M', &sty.Font, &sty.Text, &core.AppearanceSettings.Text)
	ed.charSize.X = r.Advance()
	ed.charSize.Y = ed.Scene.TextShaper.LineHeight(&sty.Font, &sty.Text, &core.AppearanceSettings.Text)

	ed.lineNumberDigits = max(1+int(math32.Log10(float32(ed.NumLines))), 3)
	lno := true
	if ed.Lines != nil {
		lno = ed.Lines.Settings.ineNumbers
	}
	if lno {
		ed.hasLineNumbers = true
		ed.lineNumberOffset = ed.lineNumberDigits + 3
	} else {
		ed.hasLineNumbers = false
		ed.lineNumberOffset = 0
	}
}

// updateFromAlloc updates size info based on allocated size:
// visSize, linesSize
func (ed *Editor) updateFromAlloc() {
	sty := &ed.Styles
	asz := ed.Geom.Size.Alloc.Content
	spsz := sty.BoxSpace().Size()
	asz.SetSub(spsz)
	sbw := math32.Ceil(ed.Styles.ScrollbarWidth.Dots)
	asz.X -= sbw
	if ed.HasScroll[math32.X] {
		asz.Y -= sbw
	}
	ed.lineLayoutSize = asz

	if asz == (math32.Vector2{}) {
		ed.visSize.Y = 20
		ed.visSize.X = 80
	} else {
		ed.visSize.Y = int(math32.Floor(float32(asz.Y) / ed.charSize.Y))
		ed.visSize.X = int(math32.Floor(float32(asz.X) / ed.charSize.X))
	}
	ed.linesSize.X = ed.visSize.X - ed.lineNumberOffset
}

func (ed *Editor) internalSizeFromLines() {
	ed.Geom.Size.Internal = ed.totalSize
	ed.Geom.Size.Internal.Y += ed.lineHeight
}

// layoutAllLines gets the width, then from that the total height.
func (ed *Editor) layoutAllLines() {
	ed.updateFromAlloc()
	if ed.visSize.Y == 0 {
		return
	}
	if ed.Lines == nil || ed.Lines.NumLines() == 0 {
		return
	}
	ed.lastFilename = ed.Lines.Filename()
	sty := &ed.Styles

	buf := ed.Lines
	// buf.Lock()
	// todo: self-lock method for this:
	buf.Highlighter.TabSize = sty.Text.TabSize

	width := ed.linesSize.X
	buf.SetWidth(ed.viewId, width) // inexpensive if same
	ed.linesSize.Y = buf.TotalLines(ed.viewId)
	et.totalSize.X = float32(ed.charSize.X) * ed.visSize.X
	et.totalSize.Y = float32(ed.charSize.Y) * ed.linesSize.Y

	// todo: don't bother with rendering now -- just do JIT in render
	// buf.Unlock()
	ed.hasLinks = false // todo: put on lines
	ed.lastlineLayoutSize = ed.lineLayoutSize
	ed.internalSizeFromLines()
}

// reLayoutAllLines updates the Renders Layout given current size, if changed
func (ed *Editor) reLayoutAllLines() {
	ed.updateFromAlloc()
	if ed.lineLayoutSize.Y == 0 || ed.Styles.Font.Size.Value == 0 {
		return
	}
	if ed.Lines == nil || ed.Lines.NumLines() == 0 {
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
	if ed.Lines == nil {
		return
	}
	nln := ed.Lines.NumLines()
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
	if ed.Lines == nil || ed.Lines.NumLines() == 0 || ln >= len(ed.renders) {
		return false
	}
	sty := &ed.Styles
	fst := sty.FontRender()
	fst.Background = nil
	mxwd := float32(ed.linesSize.X)
	needLay := false

	rn := &ed.renders[ln]
	curspans := len(rn.Spans)
	ed.Lines.Lock()
	rn.SetHTMLPre(ed.Lines.Markup[ln], fst, &sty.Text, &sty.UnitContext, ed.textStyleProperties())
	ed.Lines.Unlock()
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
