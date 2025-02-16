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

// styleSizes gets the charSize based on Style settings,
// and updates lineNumberOffset.
func (ed *Base) styleSizes() {
	ed.lineNumberDigits = max(1+int(math32.Log10(float32(ed.NumLines()))), 3)
	lno := true
	if ed.Lines != nil {
		lno = ed.Lines.Settings.LineNumbers
		ed.Lines.SetFontStyle(&ed.Styles.Font)
	}
	if lno {
		ed.hasLineNumbers = true
		ed.lineNumberOffset = ed.lineNumberDigits + 3
	} else {
		ed.hasLineNumbers = false
		ed.lineNumberOffset = 0
	}

	if ed.Scene == nil {
		ed.charSize.Set(16, 22)
		return
	}
	sty := &ed.Styles
	sh := ed.Scene.TextShaper
	r := sh.FontSize('M', &sty.Font, &sty.Text, &core.AppearanceSettings.Text)
	ed.charSize.X = r.Advance()
	ed.charSize.Y = sh.LineHeight(&sty.Font, &sty.Text, &core.AppearanceSettings.Text)
}

// visSizeFromAlloc updates visSize based on allocated size.
func (ed *Base) visSizeFromAlloc() {
	asz := ed.Geom.Size.Alloc.Content
	sbw := math32.Ceil(ed.Styles.ScrollbarWidth.Dots)
	if ed.HasScroll[math32.Y] {
		asz.X -= sbw
	}
	if ed.HasScroll[math32.X] {
		asz.Y -= sbw
	}
	ed.visSizeAlloc = asz
	ed.visSize.Y = int(math32.Floor(float32(asz.Y) / ed.charSize.Y))
	ed.visSize.X = int(math32.Floor(float32(asz.X) / ed.charSize.X))
	// fmt.Println("vis size:", ed.visSize, "alloc:", asz, "charSize:", ed.charSize, "grow:", sty.Grow)
}

// layoutAllLines uses the visSize width to update the line wrapping
// of the Lines text, getting the total height.
func (ed *Base) layoutAllLines() {
	ed.visSizeFromAlloc()
	if ed.visSize.Y == 0 || ed.Lines == nil || ed.Lines.NumLines() == 0 {
		return
	}
	ed.lastFilename = ed.Lines.Filename()
	sty := &ed.Styles
	buf := ed.Lines
	// todo: self-lock method for this, and general better api
	buf.Highlighter.TabSize = sty.Text.TabSize

	// todo: may want to support horizontal scroll and min width
	ed.linesSize.X = ed.visSize.X - ed.lineNumberOffset // width
	buf.SetWidth(ed.viewId, ed.linesSize.X)             // inexpensive if same, does update
	ed.linesSize.Y = buf.ViewLines(ed.viewId)
	ed.totalSize.X = ed.charSize.X * float32(ed.visSize.X)
	ed.totalSize.Y = ed.charSize.Y * float32(ed.linesSize.Y)

	// ed.hasLinks = false // todo: put on lines
	ed.lastVisSizeAlloc = ed.visSizeAlloc
}

// reLayoutAllLines updates the Renders Layout given current size, if changed
func (ed *Base) reLayoutAllLines() {
	ed.visSizeFromAlloc()
	if ed.visSize.Y == 0 || ed.Lines == nil || ed.Lines.NumLines() == 0 {
		return
	}
	if ed.lastVisSizeAlloc == ed.visSizeAlloc {
		return
	}
	ed.layoutAllLines()
}

// note: Layout reverts to basic Widget behavior for layout if no kids, like us..

// sizeToLines sets the Actual.Content size based on number of lines of text,
// subject to maxGrowLines, for the non-grow case.
func (ed *Base) sizeToLines() {
	if ed.Styles.Grow.Y > 0 {
		return
	}
	nln := ed.Lines.NumLines()
	if ed.linesSize.Y > 0 { // we have already been through layout
		nln = ed.linesSize.Y
	}
	nln = min(maxGrowLines, nln)
	maxh := float32(nln) * ed.charSize.Y
	sz := &ed.Geom.Size
	ty := styles.ClampMin(styles.ClampMax(maxh, sz.Max.Y), sz.Min.Y)
	sz.Actual.Content.Y = ty
	sz.Actual.Total.Y = sz.Actual.Content.Y + sz.Space.Y
	if core.DebugSettings.LayoutTrace {
		fmt.Println(ed, "textcore.Base sizeToLines targ:", ty, "nln:", nln, "Actual:", sz.Actual.Content)
	}
}

func (ed *Base) SizeUp() {
	ed.Frame.SizeUp() // sets Actual size based on styles
	if ed.Lines == nil || ed.Lines.NumLines() == 0 {
		return
	}
	ed.sizeToLines()
}

func (ed *Base) SizeDown(iter int) bool {
	if iter == 0 {
		ed.layoutAllLines()
	} else {
		ed.reLayoutAllLines()
	}
	ed.sizeToLines()
	redo := ed.Frame.SizeDown(iter)
	// todo: redo sizeToLines again?
	// chg := ed.ManageOverflow(iter, true)
	return redo
}

func (ed *Base) SizeFinal() {
	ed.Frame.SizeFinal()
	ed.reLayoutAllLines()
}

func (ed *Base) Position() {
	ed.Frame.Position()
	ed.ConfigScrolls()
}

func (ed *Base) ApplyScenePos() {
	ed.Frame.ApplyScenePos()
	ed.PositionScrolls()
}

func (ed *Base) ScrollValues(d math32.Dims) (maxSize, visSize, visPct float32) {
	if d == math32.X {
		return ed.Frame.ScrollValues(d)
	}
	maxSize = float32(max(ed.linesSize.Y, 1))
	visSize = float32(ed.visSize.Y)
	visPct = visSize / maxSize
	return
}

func (ed *Base) ScrollChanged(d math32.Dims, sb *core.Slider) {
	if d == math32.X {
		ed.Frame.ScrollChanged(d, sb)
		return
	}
	ed.scrollPos = sb.Value
	ed.NeedsRender()
}

// updateScroll sets the scroll position to given value, in lines.
func (ed *Base) updateScroll(idx int) {
	if !ed.HasScroll[math32.Y] || ed.Scrolls[math32.Y] == nil {
		return
	}
	sb := ed.Scrolls[math32.Y]
	sb.SetValue(float32(idx))
}
