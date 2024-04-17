// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"testing"

	"cogentcore.org/core/cam/hct"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
)

func TestCanvas(t *testing.T) {
	b := NewBody()
	NewCanvas(b).SetDraw(func(pc *paint.Context) {
		pc.MoveTo(0.15, 0.3)
		pc.LineTo(0.3, 0.15)
		pc.StrokeStyle.Color = colors.C(colors.Blue)
		pc.Stroke()

		pc.FillBox(math32.Vec2(0.7, 0.3), math32.Vec2(0.2, 0.5), colors.C(colors.Scheme.Success.Container))

		pc.FillStyle.Color = colors.C(colors.Orange)
		pc.DrawCircle(0.4, 0.5, 0.15)
		pc.Fill()
	})
	b.AssertRender(t, "canvas/basic")
}

func TestCanvasLogo(t *testing.T) {
	b := NewBody()
	inner := hct.Lighten(colors.Scheme.Primary.Base, 10)
	outer := hct.Darken(inner, 30)
	core := hct.Saturate(hct.Spin(hct.Lighten(inner, 20), 160), 10)

	fmt.Println("Outer:", colors.AsHex(outer))
	fmt.Println("Inner:", colors.AsHex(inner))
	fmt.Println("Core:", colors.AsHex(core))

	NewCanvas(b).SetDraw(func(pc *paint.Context) {
		pc.VectorEffect = styles.VectorEffectNone
		pc.StrokeStyle.Width.Dots = 0.2

		pc.DrawArc(0.55, 0.5, 0.4, math32.DegToRad(30), math32.DegToRad(30+300))
		pc.StrokeStyle.Color = colors.C(outer)
		pc.Stroke()

		pc.DrawArc(0.55, 0.5, 0.22, math32.DegToRad(30), math32.DegToRad(30+300))
		pc.StrokeStyle.Color = colors.C(inner)
		pc.Stroke()

		pc.FillStyle.Color = colors.C(core)
		pc.DrawCircle(0.55, 0.5, 0.15)
		pc.Fill()
	})
	b.AssertRender(t, "canvas/logo")
}
