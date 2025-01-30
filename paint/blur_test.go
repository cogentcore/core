// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint_test

import (
	"fmt"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	. "cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

func TestEdgeBlurFactors(t *testing.T) {
	t.Skip("mostly informational; TODO: maybe make this a real test at some point")
	fmt.Println(EdgeBlurFactors(2, 4))
}

func RunShadowBlur(t *testing.T, imgName string, shadow styles.Shadow) {
	RunTest(t, imgName, 300, 300, func(pc *Painter) {
		st := styles.NewStyle()
		st.Color = colors.Uniform(colors.Black)
		st.Border.Width.Set(units.Dp(0))
		st.Border.Radius = styles.BorderRadiusFull
		st.BoxShadow = []styles.Shadow{shadow}
		st.ToDots()

		spc := st.BoxSpace().Size()
		sz := spc.Add(math32.Vec2(200, 100))
		pc.StandardBox(st, math32.Vec2(50, 100), sz, colors.Uniform(colors.White))
	})
}

func TestShadowBlur(t *testing.T) {

	// fmt.Println("0.12", cie.SRGBToLinearComp(0.12)) // 0.013 -- too low

	RunShadowBlur(t, "shadow5big-op1", styles.Shadow{
		OffsetX: units.Zero(),
		OffsetY: units.Dp(6),
		Blur:    units.Dp(30),
		Spread:  units.Dp(5),
		Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 1), // opacity 1 to see clearly
	})
	RunShadowBlur(t, "shadow5big-op12", styles.Shadow{
		OffsetX: units.Zero(),
		OffsetY: units.Dp(6),
		Blur:    units.Dp(30),
		Spread:  units.Dp(5),
		Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.12), // actual
	})
	RunShadowBlur(t, "shadow5big-op1off36", styles.Shadow{
		OffsetX: units.Zero(),
		OffsetY: units.Dp(36),
		Blur:    units.Dp(30),
		Spread:  units.Dp(5),
		Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 1), // opacity 1 to see clearly
	})

	RunShadowBlur(t, "shadow1sm-op1", styles.Shadow{
		OffsetX: units.Zero(),
		OffsetY: units.Dp(3),
		Blur:    units.Dp(1),
		Spread:  units.Dp(-2),
		Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 1),
	})
	RunShadowBlur(t, "shadow1sm-op12", styles.Shadow{
		OffsetX: units.Zero(),
		OffsetY: units.Dp(3),
		Blur:    units.Dp(1),
		Spread:  units.Dp(-2),
		Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.12),
	})
}
