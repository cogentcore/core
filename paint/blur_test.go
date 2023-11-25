// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/anthonynsimon/bild/blur"
	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/grows/images"
	"goki.dev/mat32/v2"
)

// This mostly replicates the first test from this reference:
// https://docs.scipy.org/doc/scipy/reference/generated/scipy.ndimage.gaussian_filter.html
func TestGaussianBlur(t *testing.T) {
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
	fmt.Println(EdgeBlurFactors(2, 4))
}

func RunShadowBlur(t *testing.T, imgName string, shadow styles.Shadow) {
	imgsz := image.Point{320, 240}
	szrec := image.Rectangle{Max: imgsz}

	rs := &State{}
	pc := &Paint{}

	pc.Defaults() // zeros are not good defaults for paint

	st := &styles.Style{}
	st.Defaults()
	st.Color = colors.Black
	st.BackgroundColor.SetSolid(colors.Transparent) // Lightblue)
	st.Border.Width.Set(units.Dp(0))
	st.Border.Radius = styles.BorderRadiusFull
	st.BoxShadow = []styles.Shadow{shadow}

	pc.SetUnitContextExt(imgsz) // initialize units
	img := image.NewRGBA(szrec)
	rs.Init(imgsz.X, imgsz.Y, img)
	rs.PushBounds(szrec)
	rs.Lock()

	st.ToDots()
	sbg := &colors.Full{Solid: colors.White}
	spc := st.BoxSpace().Size()
	sz := spc.Add(mat32.Vec2{200, 100})
	pc.DrawStdBox(rs, st, mat32.Vec2{50, 75}, sz, sbg, 0)
	rs.Unlock()
	images.Assert(t, img, imgName)
}

func TestShadowBlur(t *testing.T) {

	// fmt.Println("0.12", cie.SRGBToLinearComp(0.12)) // 0.013 -- too low

	RunShadowBlur(t, "shadow5big_op1", styles.Shadow{
		HOffset: units.Zero(),
		VOffset: units.Dp(6),
		Blur:    units.Dp(30),
		Spread:  units.Dp(5),
		Color:   colors.SetAF32(colors.Scheme.Shadow, 1), // opacity 1 to see clearly
	})
	RunShadowBlur(t, "shadow5big_op12", styles.Shadow{
		HOffset: units.Zero(),
		VOffset: units.Dp(6),
		Blur:    units.Dp(30),
		Spread:  units.Dp(5),
		Color:   colors.SetAF32(colors.Scheme.Shadow, 0.12), // actual
	})
	RunShadowBlur(t, "shadow5big_op1off36", styles.Shadow{
		HOffset: units.Zero(),
		VOffset: units.Dp(36),
		Blur:    units.Dp(30),
		Spread:  units.Dp(5),
		Color:   colors.SetAF32(colors.Scheme.Shadow, 1), // opacity 1 to see clearly
	})

	RunShadowBlur(t, "shadow1sm_op1", styles.Shadow{
		HOffset: units.Zero(),
		VOffset: units.Dp(3),
		Blur:    units.Dp(1),
		Spread:  units.Dp(-2),
		Color:   colors.SetAF32(colors.Scheme.Shadow, 1),
	})
	RunShadowBlur(t, "shadow1sm_op12", styles.Shadow{
		HOffset: units.Zero(),
		VOffset: units.Dp(3),
		Blur:    units.Dp(1),
		Spread:  units.Dp(-2),
		Color:   colors.SetAF32(colors.Scheme.Shadow, 0.12),
	})
}
