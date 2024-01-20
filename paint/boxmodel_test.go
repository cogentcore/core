// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"path/filepath"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
	"github.com/iancoleman/strcase"
)

func TestBoxModel(t *testing.T) {
	RunTest(t, "boxmodel", 300, 300, func(pc *Context) {
		pabg := colors.C(colors.White)
		s := styles.NewStyle()
		s.Color = colors.Black
		s.Background = colors.C(colors.Lightblue)
		s.Border.Style.Set(styles.BorderSolid)
		s.Border.Width.Set(units.Dp(5))
		s.Border.Color.Set(colors.Red)
		s.Border.Radius = styles.BorderRadiusFull
		s.ComputeActualBackground(pabg)
		s.ToDots()

		sz := s.BoxSpace().Size().Add(mat32.V2(200, 100))
		pc.DrawStdBox(s, mat32.V2(50, 100), sz, pabg)
	})
}

func TestBoxShadow(t *testing.T) {
	RunTest(t, "boxshadow", 300, 300, func(pc *Context) {
		pabg := colors.C(colors.White)
		s := styles.NewStyle()
		s.Color = colors.Black
		s.Background = colors.C(colors.Lightblue)
		s.Border.Style.Set(styles.BorderSolid)
		s.Border.Width.Set(units.Dp(0))
		s.Border.Color.Set(colors.Red)
		s.Border.Radius = styles.BorderRadiusFull
		s.BoxShadow = styles.BoxShadow1()
		s.ComputeActualBackground(pabg)
		s.ToDots()

		sz := s.BoxSpace().Size().Add(mat32.V2(200, 100))

		pc.DrawStdBox(s, mat32.V2(50, 100), sz, pabg)
	})
}

func TestActualBackgroundColor(t *testing.T) {
	RunTest(t, "actual-background-color", 300, 300, func(pc *Context) {
		pabg := colors.C(colors.White)
		a := styles.NewStyle()
		a.Background = colors.C(colors.Lightgray)
		a.ComputeActualBackground(pabg)
		pc.DrawStdBox(a, mat32.Vec2{}, mat32.V2(300, 300), pabg)

		b := styles.NewStyle()
		b.Background = colors.C(colors.Red)
		b.Opacity = 0.5
		b.ComputeActualBackground(a.ActualBackground)
		pc.DrawStdBox(b, mat32.V2(50, 50), mat32.V2(200, 200), a.ActualBackground)

		c := styles.NewStyle()
		c.Background = colors.C(colors.Blue)
		c.Opacity = 0.5
		c.StateLayer = 0.1
		c.ComputeActualBackground(b.ActualBackground)
		pc.DrawStdBox(c, mat32.V2(75, 75), mat32.V2(150, 150), b.ActualBackground)

		// d is transparent and thus should not be any different than c
		d := styles.NewStyle()
		d.Opacity = 0.5
		d.ComputeActualBackground(c.ActualBackground)
		pc.DrawStdBox(d, mat32.V2(100, 100), mat32.V2(100, 100), c.ActualBackground)
	})
}

func TestBorderStyle(t *testing.T) {
	for _, typ := range styles.BorderStylesValues() {
		RunTest(t, filepath.Join("border-styles", strcase.ToKebab(typ.String())), 300, 300, func(pc *Context) {
			s := styles.NewStyle()
			s.Background = colors.C(colors.Lightgray)
			s.Border.Style.Set(typ)
			s.Border.Width.Set(units.Dp(10))
			s.Border.Color.Set(colors.Blue)
			s.Border.Radius.Set(units.Dp(50))
			s.ToDots()

			sz := s.BoxSpace().Size().Add(mat32.V2(200, 100))
			pc.DrawStdBox(s, mat32.V2(50, 100), sz, colors.C(colors.White))
		})
	}
}
