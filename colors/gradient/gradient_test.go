// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gradient

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/iox/imagex"
	"cogentcore.org/core/math32"
)

func ExampleLinear() {
	NewLinear().AddStop(colors.White, 0).AddStop(colors.Black, 1)
}

func ExampleRadial() {
	NewRadial().AddStop(colors.Green, 0).AddStop(colors.Yellow, 0.5).AddStop(colors.Red, 1)
}

func TestColorAt(t *testing.T) {
	type value struct {
		x    int
		y    int
		want color.RGBA
	}
	type test struct {
		gr   Gradient
		want []value
	}
	tests := []test{
		{NewLinear().
			AddStop(colors.White, 0).
			AddStop(colors.Black, 1),
			[]value{
				{71, 33, color.RGBA{72, 72, 72, 255}},
				{71, 78, color.RGBA{72, 72, 72, 255}},
				{17, 78, color.RGBA{211, 211, 211, 255}},
				{50, 33, color.RGBA{126, 126, 126, 255}},
			}},
		{CopyOf(linearTransformTest),
			[]value{
				{50, 50, color.RGBA{255, 106, 0, 255}},
				{7, 50, color.RGBA{255, 106, 0, 255}},
				{81, 23, color.RGBA{255, 171, 0, 255}},
				{81, 94, color.RGBA{255, 1, 0, 255}},
			}},
		{NewRadial().
			SetCenter(math32.V2(0.9, 0.5)).SetFocal(math32.V2(0.9, 0.5)).
			AddStop(colors.Blue, 0.1).
			AddStop(colors.Yellow, 0.85),
			[]value{
				{90, 50, colors.Blue},
				{70, 60, color.RGBA{117, 117, 138, 255}},
				{35, 40, colors.Yellow},
			}},
		{CopyOf(radialTransformTest),
			[]value{
				{41, 62, color.RGBA{104, 0, 151, 255}},
				{26, 54, color.RGBA{2, 0, 253, 255}},
				{53, 75, color.RGBA{132, 85, 123, 255}},
				{38, 61, color.RGBA{141, 0, 114, 255}},
			}},
	}
	for i, test := range tests {
		gb := test.gr.AsBase()
		test.gr.Update(1, gb.Box, math32.Identity2())
		for j, v := range test.want {
			have := test.gr.At(v.x, v.y)
			if have != v.want {
				t.Errorf("%d.%d: expected %v at %v but got %v", i, j, v.want, image.Pt(v.x, v.y), have)
			}
		}

		// ensure same results with UserSpaceOnUse as ObjectBoundingBox
		// (except for case 3, for which that is not true)
		if i == 3 {
			continue
		}
		ugr := CopyOf(test.gr)
		switch ugr := ugr.(type) {
		case *Linear:
			ugr.Start.SetMul(ugr.Box.Size())
			ugr.End.SetMul(ugr.Box.Size())
		case *Radial:
			ugr.Center.SetMul(ugr.Box.Size())
			ugr.Focal.SetMul(ugr.Box.Size())
			ugr.Radius.SetMul(ugr.Box.Size())
		}
		ugb := ugr.AsBase()
		ugb.SetUnits(UserSpaceOnUse)
		ugr.Update(1, ugb.Box, math32.Identity2())

		for j, v := range test.want {
			have := ugr.At(v.x, v.y)
			if have != v.want {
				t.Errorf("%d.%d: UserSpaceOnUse: expected %v at %v but got %v", i, j, v.want, image.Pt(v.x, v.y), have)
			}
		}
	}
}

func TestRenderLinear(t *testing.T) {
	r := image.Rectangle{Max: image.Point{128, 128}}
	b := math32.B2FromRect(r)
	img := image.NewRGBA(r)
	g := CopyOf(linearTransformTest)
	// g.AsBase().Blend = colors.HCT
	g.Update(1, b, math32.Rotate2D(math32.DegToRad(45)))
	draw.Draw(img, img.Bounds(), g, image.Point{}, draw.Src)
	imagex.Assert(t, img, "linear")

	ug := CopyOf(g).(*Linear)
	ug.SetUnits(UserSpaceOnUse)
	ug.Start.SetMul(ug.Box.Size())
	ug.End.SetMul(ug.Box.Size())
	ug.Update(1, b, math32.Rotate2D(math32.DegToRad(45)))
	draw.Draw(img, img.Bounds(), ug, image.Point{}, draw.Src)
	imagex.Assert(t, img, "linear-user-space")
}

func TestRenderRadial(t *testing.T) {
	r := image.Rectangle{Max: image.Point{128, 128}}
	b := math32.B2FromRect(r)
	img := image.NewRGBA(r)
	g := CopyOf(radialTransformTest)
	// g.AsBase().Blend = colors.HCT
	g.Update(1, b, math32.Identity2())
	draw.Draw(img, img.Bounds(), g, image.Point{}, draw.Src)
	imagex.Assert(t, img, "radial")

	ug := CopyOf(g).(*Radial)
	ug.SetUnits(UserSpaceOnUse)
	ug.Center.SetMul(ug.Box.Size())
	ug.Focal.SetMul(ug.Box.Size())
	ug.Radius.SetMul(ug.Box.Size())
	ug.Update(1, b, math32.Identity2())
	draw.Draw(img, img.Bounds(), ug, image.Point{}, draw.Src)
	imagex.Assert(t, img, "radial-user-space")
}

// func matToRasterx(mat *math32.Mat2) rasterx.Matrix2D {
// 	// A = XX
// 	// B = YX
// 	// C = XY
// 	// D = YY
// 	// E = X0
// 	// F = Y0
// 	return rasterx.Matrix2D{float64(mat.XX), float64(mat.YX), float64(mat.XY), float64(mat.YY), float64(mat.X0), float64(mat.Y0)}
// }

func compareTol(t *testing.T, a, c float32) {
	if math32.Abs(a-c) > 1.0e-5 {
		t.Errorf("value not in tolerance. actual: %g  correct: %g\n", a, c)
	}
}

func TestTransform(t *testing.T) {
	r := image.Rect(20, 20, 140, 140)
	b := math32.B2FromRect(r)
	g := CopyOf(linearTransformTest)
	gb := g.AsBase()
	gb.Transform = math32.Rotate2D(math32.DegToRad(25))
	// fmt.Println(gb.Transform)
	g.Update(1, b, math32.Identity2())
	// fmt.Println(gb.boxTransform)
	btcorrect := math32.Mat2{XX: 0.9063079, YX: -0.42261833, XY: 0.42261833, YY: 0.9063079, X0: -6.5785227, Y0: 10.326212}
	compareTol(t, gb.boxTransform.XX, btcorrect.XX)
	compareTol(t, gb.boxTransform.YX, btcorrect.YX)
	compareTol(t, gb.boxTransform.XY, btcorrect.XY)
	compareTol(t, gb.boxTransform.YY, btcorrect.YY)
	compareTol(t, gb.boxTransform.X0, btcorrect.X0)
	compareTol(t, gb.boxTransform.Y0, btcorrect.Y0)

	// szf := math32.V2FromPoint(r.Size())
	// w := float64(szf.X)
	// h := float64(szf.Y)
	// oriX := float64(r.Min.X)
	// oriY := float64(r.Min.Y)
	// mtx := matToRasterx(&gb.Transform)
	// gradT := rasterx.Identity.Translate(oriX, oriY).Scale(w, h).Mult(mtx).
	// 	Scale(1/w, 1/h).Translate(-oriX, -oriY).Invert()
	// fmt.Println(gradT)
}

func TestApply(t *testing.T) {
	r := image.Rect(0, 0, 2, 2)
	img := image.NewRGBA(r)
	img.Set(0, 0, colors.Red)
	img.Set(1, 0, colors.Blue)
	img.Set(0, 1, colors.Green)
	img.Set(1, 1, colors.Yellow)

	var ocs []uint8
	ap := Apply(img, func(c color.Color) color.Color {
		oc := colors.ApplyOpacity(c, .5)
		ocs = append(ocs, oc.R, oc.G, oc.B, oc.A)
		return oc
	})
	nim := image.NewRGBA(r)
	draw.Draw(nim, r, ap, image.Point{}, draw.Src)
	for i, c := range nim.Pix {
		if c != ocs[i] {
			t.Errorf("output not the same: %v != %v\n", c, ocs[i])
		}
	}
}

func TestHCT(t *testing.T) {
	t.Skip("for reference")
	orig := []color.RGBA{
		{255, 255, 255, 255},
		{226, 226, 226, 255},
		{199, 198, 198, 255},
		{171, 171, 171, 255},
		{145, 144, 144, 255},
		{119, 119, 119, 255},
		{95, 94, 94, 255},
		{71, 71, 70, 255},
		{49, 48, 48, 255},
		{28, 27, 27, 255},
		{0, 0, 0, 255},
	}

	hct := NewLinear().AddStop(colors.White, 0).AddStop(colors.Black, 1)
	hct.Blend = colors.HCT
	idx := 0
	for pct := float32(0); pct <= 1.01; pct += 0.1 {
		// lc := lin.GetColor(pct)
		hc := hct.GetColor(pct)
		oc := orig[idx]
		if oc != hc {
			t.Errorf("original: %#v != %#v\n", oc, hc)
		}
		// fmt.Println(pct, hc)
		idx++
	}
	// fmt.Printf("\n%#v\n", hct)
}
