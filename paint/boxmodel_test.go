// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"testing"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/mat32/v2"
)

func TestBoxModel(t *testing.T) {
	RunTest(t, "boxmodel", 300, 300, func(pc *Context) {
		st := styles.NewStyle()
		st.Color = colors.Black
		st.BackgroundColor.SetSolid(colors.Lightblue)
		st.Border.Style.Set(styles.BorderSolid)
		st.Border.Width.Set(units.Dp(5))
		st.Border.Color.Set(colors.Red)
		st.Border.Radius = styles.BorderRadiusFull
		st.ToDots()

		sz := st.BoxSpace().Size().Add(mat32.Vec2{200, 100})
		pc.DrawStdBox(st, mat32.Vec2{50, 100}, sz, colors.SolidFull(colors.White))
	})
}

func TestBoxShadow(t *testing.T) {
	RunTest(t, "boxshadow", 300, 300, func(pc *Context) {
		st := styles.NewStyle()
		st.Color = colors.Black
		st.BackgroundColor.SetSolid(colors.Lightblue)
		st.Border.Style.Set(styles.BorderSolid)
		st.Border.Width.Set(units.Dp(0))
		st.Border.Color.Set(colors.Red)
		st.Border.Radius = styles.BorderRadiusFull
		st.BoxShadow = styles.BoxShadow1()
		st.ToDots()

		spc := st.BoxSpace().Size()
		sz := spc.Add(mat32.Vec2{200, 100})

		pc.DrawStdBox(st, mat32.Vec2{50, 100}, sz, colors.SolidFull(colors.White))
	})
}

func TestActualBackgroundColor(t *testing.T) {
	RunTest(t, "actual_background_color", 300, 300, func(pc *Context) {
		a := styles.NewStyle()
		a.BackgroundColor.SetSolid(colors.Lightgray)
		pc.DrawStdBox(a, mat32.Vec2{}, mat32.Vec2{300, 300}, colors.SolidFull(colors.White))

		b := styles.NewStyle()
		b.BackgroundColor.SetSolid(colors.Red)
		b.Opacity = 0.5
		pc.DrawStdBox(b, mat32.Vec2{50, 50}, mat32.Vec2{200, 200}, &a.ActualBackgroundColor)

		c := styles.NewStyle()
		c.BackgroundColor.SetSolid(colors.Blue)
		c.Opacity = 0.5
		pc.DrawStdBox(c, mat32.Vec2{75, 75}, mat32.Vec2{150, 150}, &b.ActualBackgroundColor)

		// d is transparent and thus should not be any different than c
		d := styles.NewStyle()
		d.BackgroundColor.SetSolid(colors.Transparent)
		d.Opacity = 0.5
		pc.DrawStdBox(d, mat32.Vec2{100, 100}, mat32.Vec2{100, 100}, &c.ActualBackgroundColor)
	})
}
