// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"time"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/mat32/v2"
)

func main() { gimain.Run(app) }

type Wide struct {
	Name  string
	Title string
	F2    string
	F3    string
}

type Test struct {
	Wide Wide `view:"inline"`
	Vec  mat32.Vec2
}

type TestTime struct {
	Date time.Time
}

var (
	ShortText = "This is a test of layout."

	LongText = "This is a test of the layout logic, which is pretty complex and requires some experimenting to understand how it all works.  The styling and behavior is the same as the CSS / HTML Flex model, except we only support Grow, not Shrink. "

	VeryLongText = LongText + LongText + LongText

	AlignText = "This is text to test for text align<br>This line is short<br>This is text to test for text align, this one is longer"

	FrameSizes = [5]mat32.Vec2{
		{20, 100},
		{80, 20},
		{60, 80},
		{40, 120},
		{150, 100},
	}
)

func app() {
	// turn on tracing in preferences, Debug
	gi.LayoutTrace = true
	gi.LayoutTraceDetail = true
	// gi.UpdateTrace = true

	gi.SetAppName("layout")
	gi.SetAppAbout(`This is a demo of the layout functions in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	sc := gi.NewScene("lay-test").SetTitle("GoGi Layout Test")
	gi.DefaultTopAppBar = nil

	doCase := "frames-vert"

	switch doCase {
	case "text-align":
		// 	row := HorizRow(sc)
		sc.Style(func(s *styles.Style) {
			s.Align.X = styles.AlignCenter
		})
		gi.NewLabel(sc, "lbl").SetText(AlignText).Style(func(s *styles.Style) {
			s.Align.X = styles.AlignCenter
			s.Text.Align = styles.AlignCenter
		})
	case "long-text-wrap": // just text
		WrapText(sc, VeryLongText)
	case "long-text-wrap-box": // text in box -- failing to adjust to full height
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
	case "long-text-wrap-max-box": // text in constrained box
		row := HorizRow(sc)
		lbl := WrapText(row, VeryLongText) // VeryLongText)
		row.Style(func(s *styles.Style) {
			// s.Align.X = styles.AlignEnd
			s.Max.X.Ch(100) // todo: this is *sometimes* failing to constrain..
			// s.Overflow.X = styles.OverflowAuto
		})
		lbl.Style(func(s *styles.Style) {
			s.Text.Align = styles.AlignCenter
		})
		// fr := BoxFrame(sc) // this takes up slack
		// sm := WrapText(fr, ShortText)
		// _ = sm
	case "frames-vert":
		PlainFrames(sc, mat32.Vec2{0, 0})
		sc.Style(func(s *styles.Style) {
			s.Wrap = true
			s.Align.X = styles.AlignCenter
		})
	case "frames-horiz":
		row := HorizRow(sc)
		PlainFrames(row, mat32.Vec2{1, 0})
	case "scroll-absorb": // Auto scroll should absorb extra size
		row := HorizRow(sc, "row")
		f1, sp := SpaceFrame(row)
		f1.Style(func(s *styles.Style) {
			s.Overflow.Y = styles.OverflowAuto // this should absorb the size
		})
		sp.Style(func(s *styles.Style) {
			s.Min.Y.Em(100)
		})
		BoxFrame(row).Style(func(s *styles.Style) {
			s.Min.Y.Em(20) // fix size
			s.Max.Y.Em(20) // fix size
		})
	case "scroll-absorb-splits": // Auto scroll should absorb extra size
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
	case "tabs-stack": // recreates the issue with demo tabs
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
	case "splits": // splits
		sp, f1, f2 := Splits2(sc)
		_ = f1
		_ = f2
		sp.SetSplits(.3, .7)
	case "textfield-parts": // textfield parts alignment
		gi.NewTextField(sc).AddClearButton()
	case "structview": // structview
		ts := &Test{}
		giv.NewStructView(sc).SetStruct(ts)
	case "timeview": // time view
		ts := &TestTime{}
		ts.Date = time.Now()
		giv.NewStructView(sc).SetStruct(ts)
	case "center-dialog":
		sc.Style(func(s *styles.Style) {
			s.Align.Set(styles.AlignCenter)
		})
		gi.NewLabel(sc).SetType(gi.LabelDisplayMedium).SetText("Event recorded!").
			Style(func(s *styles.Style) {
				// s.SetTextWrap(false)
				s.Align.Set(styles.AlignCenter)
				// s.Text.Align = styles.AlignCenter
			})
		gi.NewLabel(sc).SetType(gi.LabelBodyLarge).
			SetText("Thank you for reporting your issue!").
			Style(func(s *styles.Style) {
				s.Color = colors.Scheme.OnSurfaceVariant
				s.Align.Set(styles.AlignCenter)
			})
		gi.NewButton(sc).SetType(gi.ButtonTonal).SetText("Return home").
			Style(func(s *styles.Style) {
				// s.Grow.Set(1, 0)
				s.Align.Set(styles.AlignCenter)
			})
	default:
		fmt.Println("error: case didn't match:", doCase)
	}

	gi.NewWindow(sc).Run().Wait()
}

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

func HorizRow(par gi.Widget, nm ...string) *gi.Frame {
	row := BoxFrame(par, nm...)
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
			s.Align.X = styles.AlignCenter // Center
		})
		gi.NewSpace(fr) // if here, prevents frame from growing on its own
	}
}
