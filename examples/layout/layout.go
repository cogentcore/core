// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/mat32/v2"
)

func main() { gimain.Run(app) }

var (
	ShortText = "This is a test of layout."

	LongText = "This is a test of the layout logic, which is pretty complex and requires some experimenting to understand how it all works.  The styling and behavior is the same as the CSS / HTML Flex model, except we only support Grow, not Shrink. "

	VeryLongText = LongText + LongText + LongText

	FrameSizes = [5]mat32.Vec2{
		{20, 100},
		{80, 20},
		{60, 80},
		{40, 120},
		{150, 100},
	}
)

func BoxFrame(par gi.Widget, nm ...string) *gi.Frame {
	fr := gi.NewFrame(par, nm...)
	fr.Style(func(s *styles.Style) {
		s.Border.Color.Set(colors.Black)
		s.Border.Width.Set(units.Dp(2))
	})
	return fr
}

func SpaceFrame(par gi.Widget, nm ...string) (*gi.Frame, *gi.Space) {
	fr := gi.NewFrame(par, nm...)
	fr.Style(func(s *styles.Style) {
		s.Border.Color.Set(colors.Black)
		s.Border.Width.Set(units.Dp(2))
	})
	sp := gi.NewSpace(fr)
	return fr, sp
}

func HorizRow(par gi.Widget) *gi.Frame {
	row := BoxFrame(par)
	row.Style(func(s *styles.Style) {
		s.MainAxis = mat32.X
		s.Grow.Set(1, 0)
	})
	return row
}

func Splits2(par gi.Widget) (*gi.Splits, *gi.Frame, *gi.Frame) {
	sp := gi.NewSplits(par)
	f1 := BoxFrame(sp)
	f2 := BoxFrame(sp)
	return sp, f1, f2
}

func TabFrame(par gi.Widget) (*gi.Frame, *gi.Frame) {
	tab := BoxFrame(par)
	tab.Style(func(s *styles.Style) {
		s.Display = styles.DisplayStacked
		tab.StackTop = 0
	})
	tfr := BoxFrame(tab)
	tfr.Style(func(s *styles.Style) {
		s.MainAxis = mat32.Y
	})
	return tab, tfr
}

func WrapText(par gi.Widget, txt string) *gi.Label {
	lbl := gi.NewLabel(par, "wrap-text").SetText(txt)
	return lbl
}

func app() {
	// turn on tracing in preferences, Debug
	gi.LayoutTrace = true
	// gi.LayoutTraceDetail = true
	// gi.UpdateTrace = true

	gi.SetAppName("layout")
	gi.SetAppAbout(`This is a demo of the layout functions in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	sc := gi.NewScene("lay-test").SetTitle("GoGi Layout Test")
	gi.DefaultTopAppBar = nil

	doCase := 5

	switch doCase {
	case 0: // just text
		WrapText(sc, VeryLongText)
	case 1: // text in box -- failing to adjust to full height
		row := HorizRow(sc)
		lbl := WrapText(row, VeryLongText)
		row.Style(func(s *styles.Style) {
			// s.Align.X = styles.AlignEnd
		})
		lbl.Style(func(s *styles.Style) {
			s.Align.X = styles.AlignCenter
		})
		fr := BoxFrame(sc) // this takes up slack
		sm := WrapText(fr, ShortText)
		_ = sm
	case 2: // text in constrained box
		row := HorizRow(sc)
		lbl := WrapText(row, VeryLongText)
		row.Style(func(s *styles.Style) {
			// s.Align.X = styles.AlignEnd
			s.Max.X.Ch(100) // todo: this is *sometimes* failing to constrain..
			s.Overflow.X = styles.OverflowAuto
		})
		lbl.Style(func(s *styles.Style) {
			// s.Align.X = styles.AlignCenter
		})
		fr := BoxFrame(sc) // this takes up slack
		sm := WrapText(fr, ShortText)
		_ = sm
	case 3:
		PlainFrames(sc, mat32.Vec2{0, 1})
	case 4:
		row := HorizRow(sc)
		PlainFrames(row, mat32.Vec2{1, 0})
	case 5: // Auto scroll should absorb extra size
		sp, f1, f2 := Splits2(sc)
		_ = sp
		f1.Style(func(s *styles.Style) {
			s.Overflow.Y = styles.OverflowAuto // this should absorb the size
		})
		gi.NewSpace(f1).Style(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Min.Y.Em(100)
		})
		f2.Style(func(s *styles.Style) {
			s.Grow.Set(0, 0)
		})
		gi.NewSpace(f2).Style(func(s *styles.Style) {
			s.Min.X.Ch(20)
			s.Min.Y.Em(20)
		})
	case 6: // recreates the issue with demo tabs
		// does not grow -- stacked not doing the right thing
		tab, tfr := TabFrame(sc)
		_ = tab
		par := tfr // or sc
		row := HorizRow(par)

		sp := gi.NewSpace(row)
		_ = sp
		WrapText(par, LongText)
		fr, sp2 := SpaceFrame(par)
		_ = fr
		_ = sp2
		fr.Style(func(s *styles.Style) {
			s.Grow.Set(0, 1)
			s.Min.X.Em(20)
			s.Min.Y.Em(10)
		})
	}

	gi.NewWindow(sc).Run().Wait()
}

func PlainFrames(par gi.Widget, grow mat32.Vec2) {
	for i, sz := range FrameSizes {
		i := i
		sz := sz
		nm := fmt.Sprintf("fr%v", i)
		fr := BoxFrame(par, nm)
		fr.Style(func(s *styles.Style) {
			s.Min.X.Px(sz.X)
			s.Min.Y.Px(sz.Y)
			s.Grow = grow
		})
		gi.NewSpace(fr) // if here, prevents frame from growing on its own
	}
}
