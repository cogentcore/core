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
	"goki.dev/icons"
	"goki.dev/mat32/v2"
)

func main() { gimain.Run(app) }

func app() {
	// turn on tracing in preferences, Debug
	gi.LayoutTrace = true
	gi.LayoutTraceDetail = true

	frsz := [5]mat32.Vec2{
		{20, 100},
		{80, 20},
		{60, 80},
		{40, 120},
		{150, 100},
	}
	// frsz := [4]mat32.Vec2{
	// 	{100, 100},
	// 	{100, 100},
	// 	{100, 100},
	// 	{100, 100},
	// }

	gi.SetAppName("layout")
	gi.SetAppAbout(`This is a demo of the layout functions in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	sc := gi.NewScene("gogi-layout-test").SetTitle("GoGi Layout Test")

	gi.DefaultTopAppBar = nil

	row1 := gi.NewLayout(sc, "row1")

	row1.Style(func(s *styles.Style) {
		s.MainAxis = mat32.X
		s.Grow.Set(1, 1)
		s.Gap.X.Em(2)
		// s.Margin.Set(units.Em(6))
		s.Align.X = styles.AlignCenter
	})

	for i, sz := range frsz {
		i := i
		sz := sz
		nm := fmt.Sprintf("fr%v", i)
		fr := gi.NewFrame(row1, nm)
		fr.Style(func(s *styles.Style) {
			s.MainAxis = mat32.X
			s.Align.X = styles.AlignEnd
			s.Align.Y = styles.AlignCenter
			s.Grow.Set(1, 1)
			s.Min.X.Px(sz.X)
			s.Min.Y.Px(sz.Y)
			// s.Padding.Set(units.Dp(6))
			s.MaxBorder.Color.Set(colors.Black)
			s.MaxBorder.Width.Set(units.Dp(2))
			s.Border = s.MaxBorder
			// s.Margin.Set(units.Dp(6))

			// if i == 2 {
			// 	fr.SetFixedWidth(units.Em(20))
			// 	spc := row1.NewChild(gi.SpaceType, "spc").(*gi.Space)
			// 	spc.SetFixedWidth(units.Em(4))
			// } else {
			// 	fr.SetProp("max-width", -1) // spacer
			// }
		})

		ic := gi.NewIcon(fr)
		ic.SetIcon(icons.Add)
		ic.Style(func(s *styles.Style) {
			s.Min.Set(units.Em(3))
			s.Align.X = styles.AlignStart
			s.Align.Y = styles.AlignCenter
		})
	}

	// row2 := gi.NewLayout(mfr, "row2", gi.LayoutHoriz)
	// row2.SetProp("text-align", "center")
	// row2.SetProp("max-width", -1) // always stretch width
	//
	// row2.SetProp("vertical-align", "center")
	// // row2.SetProp("horizontal-align", "justify")
	// row2.SetProp("horizontal-align", "left")
	// row2.SetProp("margin", 4.0)
	// row2.SetProp("spacing", 6.0)
	//
	// for i, sz := range frsz {
	// 	nm := fmt.Sprintf("fr%v", i)
	// 	fr := gi.NewFrame(row2, nm, gi.LayoutHoriz)
	// 	fr.SetProp("width", sz.X)
	// 	fr.SetProp("height", sz.Y)
	// 	fr.SetProp("vertical-align", "inherit")
	// 	// fr.SetProp("horizontal-align", "inherit")
	// 	fr.SetProp("margin", "inherit")
	// 	// if i == 2 {
	// 	// 	gi.NewStretch(row2, "str")
	// 	// }
	// }
	//
	// row3 := gi.NewLayout(mfr, "row3", gi.LayoutHorizFlow)
	// // row3.SetProp("text-align", "center")
	// row3.SetProp("max-width", -1) // always stretch width
	//
	// // row3.SetProp("vertical-align", "bottom")
	// // row3.SetProp("horizontal-align", "justify")
	// // row3.SetProp("horizontal-align", "left")
	// row3.SetProp("margin", 4.0)
	// row3.SetProp("spacing", 6.0)
	// row3.SetProp("width", units.Pt(200)) // needs default to set
	//
	// for i, sz := range frsz {
	// 	nm := fmt.Sprintf("fr%v", i)
	// 	fr := gi.NewFrame(row3, nm, gi.LayoutHoriz)
	// 	fr.SetProp("width", 5*sz.X)
	// 	fr.SetProp("height", sz.Y)
	// 	fr.SetProp("min-height", sz.Y)
	// 	fr.SetProp("min-width", 5*sz.X)
	// 	// fr.SetProp("vertical-align", "inherit")
	// 	// fr.SetProp("horizontal-align", "inherit")
	// 	fr.SetProp("margin", "inherit")
	// 	// fr.SetProp("max-width", -1) // spacer
	// }
	//
	// row4 := gi.NewLayout(mfr, "row4", gi.LayoutGrid)
	// row4.SetProp("columns", 2)
	// // row4.SetProp("max-width", -1)
	//
	// row4.SetProp("vertical-align", "top")
	// // row4.SetProp("horizontal-align", "justify")
	// row4.SetProp("horizontal-align", "left")
	// row4.SetProp("margin", 6.0)
	//
	// for i, sz := range frsz {
	// 	nm := fmt.Sprintf("fr%v", i)
	// 	fr := gi.NewFrame(row4, nm, gi.LayoutHoriz)
	// 	fr.SetProp("width", sz.X)
	// 	fr.SetProp("height", sz.Y)
	// 	// fr.SetProp("min-height", sz.Y)
	// 	fr.SetProp("vertical-align", "inherit")
	// 	fr.SetProp("horizontal-align", "inherit")
	// 	fr.SetProp("margin", 2.0)
	// 	// fr.SetProp("max-width", -1) // spacer
	// }
	//

	gi.NewWindow(sc).Run().Wait()
}
