// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/anthonynsimon/bild/blur"
)

// This mostly replicates the first test from this reference:
// https://docs.scipy.org/doc/scipy/reference/generated/scipy.ndimage.gaussian_filter.html
func TestGaussianBlur(t *testing.T) {
	t.Skip("mostly informational; TODO: maybe make this a real test at some point")
	sigma := 1.0
	k := GaussianBlurKernel1D(sigma)
	fmt.Println(k.Matrix)

	testIn := []uint8{}
	for n := uint8(0); n < 50; n += 2 {
		testIn = append(testIn, n)
	}
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	for i, v := range testIn {
		img.Set(i%5, i/5, color.RGBA{v, v, v, v})
	}
	blr := GaussianBlur(img, sigma)
	for i := range testIn {
		fmt.Print(blr.At(i%5, i/5).(color.RGBA).R, " ")
		if i%5 == 4 {
			fmt.Println("")
		}
	}
	fmt.Println("bild:")

	bildRad := sigma // 0.5 * sigma * sigma
	blrBild := blur.Gaussian(img, bildRad)
	for i := range testIn {
		fmt.Print(blrBild.At(i%5, i/5).(color.RGBA).R, " ")
		if i%5 == 4 {
			fmt.Println("")
		}
	}

	// our results -- these could be rounding errors
	// 3 5 7 8 10
	// 10 12 14 15 17  <- correct
	// 20 22 24 25 27  <- correct
	// 29 31 33 34 36  <- correct
	// 36 38 40 41 43

	// scipy says:
	// 4  6  8  9 11
	// 10 12 14 15 17
	// 20 22 24 25 27
	// 29 31 33 34 36
	// 35 37 39 40 42
}

func TestEdgeBlurFactors(t *testing.T) {
	t.Skip("mostly informational; TODO: maybe make this a real test at some point")
	fmt.Println(EdgeBlurFactors(2, 4))
}

func RunShadowBlur(t *testing.T, imgName string, shadow styles.Shadow) {
	RunTest(t, imgName, 300, 300, func(pc *Context) {
		st := styles.NewStyle()
		st.Color = colors.C(colors.Black)
		st.Border.Width.Set(units.Dp(0))
		st.Border.Radius = styles.BorderRadiusFull
		st.BoxShadow = []styles.Shadow{shadow}
		st.ToDots()

		spc := st.BoxSpace().Size()
		sz := spc.Add(math32.Vec2(200, 100))
		pc.DrawStandardBox(st, math32.Vec2(50, 100), sz, colors.C(colors.White))
	})
}

func TestShadowBlur(t *testing.T) {

	// fmt.Println("0.12", cie.SRGBToLinearComp(0.12)) // 0.013 -- too low

	RunShadowBlur(t, "shadow5big-op1", styles.Shadow{
		OffsetX: units.Zero(),
		OffsetY: units.Dp(6),
		Blur:    units.Dp(30),
		Spread:  units.Dp(5),
		Color:   colors.C(colors.WithAF32(colors.Scheme.Shadow, 1)), // opacity 1 to see clearly
	})
	RunShadowBlur(t, "shadow5big-op12", styles.Shadow{
		OffsetX: units.Zero(),
		OffsetY: units.Dp(6),
		Blur:    units.Dp(30),
		Spread:  units.Dp(5),
		Color:   colors.C(colors.WithAF32(colors.Scheme.Shadow, 0.12)), // actual
	})
	RunShadowBlur(t, "shadow5big-op1off36", styles.Shadow{
		OffsetX: units.Zero(),
		OffsetY: units.Dp(36),
		Blur:    units.Dp(30),
		Spread:  units.Dp(5),
		Color:   colors.C(colors.WithAF32(colors.Scheme.Shadow, 1)), // opacity 1 to see clearly
	})

	RunShadowBlur(t, "shadow1sm-op1", styles.Shadow{
		OffsetX: units.Zero(),
		OffsetY: units.Dp(3),
		Blur:    units.Dp(1),
		Spread:  units.Dp(-2),
		Color:   colors.C(colors.WithAF32(colors.Scheme.Shadow, 1)),
	})
	RunShadowBlur(t, "shadow1sm-op12", styles.Shadow{
		OffsetX: units.Zero(),
		OffsetY: units.Dp(3),
		Blur:    units.Dp(1),
		Spread:  units.Dp(-2),
		Color:   colors.C(colors.WithAF32(colors.Scheme.Shadow, 0.12)),
	})
}
