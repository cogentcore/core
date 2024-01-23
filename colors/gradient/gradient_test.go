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
	"cogentcore.org/core/grows/images"
	"cogentcore.org/core/mat32"
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
		// ensure same results with ObjectBoundingBox and UserSpaceOnUse
		{NewLinear().
			AddStop(colors.White, 0).
			AddStop(colors.Black, 1),
			[]value{
				{33, 71, color.RGBA{68, 67, 67, 255}},
				{78, 71, color.RGBA{68, 67, 67, 255}},
				{78, 17, color.RGBA{205, 205, 205, 255}},
				{33, 50, color.RGBA{118, 118, 117, 255}},
			}},
		{CopyOf(linearTransformTest),
			[]value{
				{50, 50, color.RGBA{255, 141, 52, 255}},
				{7, 50, color.RGBA{255, 141, 52, 255}},
				{81, 23, color.RGBA{255, 185, 76, 255}},
				{81, 94, color.RGBA{254, 12, 0, 255}},
			}},
		{NewRadial().
			SetCenter(mat32.V2(0.9, 0.5)).SetFocal(mat32.V2(0.9, 0.5)).
			AddStop(colors.Blue, 0.1).
			AddStop(colors.Yellow, 0.85),
			[]value{
				{90, 50, colors.Blue},
				{70, 60, color.RGBA{0, 165, 183, 255}},
				{35, 40, colors.Yellow},
			}},
		{CopyOf(radialTransformTest),
			[]value{
				{41, 62, color.RGBA{166, 54, 212, 255}},
				{26, 54, color.RGBA{221, 0, 106, 255}},
				{53, 75, color.RGBA{255, 165, 0, 255}},
				{38, 61, color.RGBA{51, 12, 252, 255}},
			}},
	}
	for i, test := range tests {
		test.gr.Update()
		for j, v := range test.want {
			have := test.gr.At(v.x, v.y)
			if have != v.want {
				t.Errorf("%d.%d: expected %v at %v but got %v", i, j, v.want, image.Pt(v.x, v.y), have)
			}
		}

		// ensure same results with UserSpaceOnUse as ObjectBoundingBox
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
		ugr.AsBase().SetUnits(UserSpaceOnUse)
		ugr.Update()

		for j, v := range test.want {
			have := ugr.At(v.x, v.y)
			if have != v.want {
				t.Errorf("%d.%d: UserSpaceOnUse: expected %v at %v but got %v", i, j, v.want, image.Pt(v.x, v.y), have)
			}
		}
	}
}

func TestRenderLinear(t *testing.T) {
	sz := image.Point{512, 512}
	img := image.NewRGBA(image.Rectangle{Max: sz})
	g := CopyOf(linearTransformTest)
	g.AsBase().Box.Max = mat32.V2FromPoint(sz)
	g.Update()
	draw.Draw(img, img.Bounds(), g, image.Point{}, draw.Src)
	images.Assert(t, img, "linear")
}

func TestRenderRadial(t *testing.T) {
	sz := image.Point{512, 512}
	img := image.NewRGBA(image.Rectangle{Max: sz})
	g := CopyOf(radialTransformTest)
	g.AsBase().Box.Max = mat32.V2FromPoint(sz)
	g.Update()
	draw.Draw(img, img.Bounds(), g, image.Point{}, draw.Src)
	images.Assert(t, img, "radial")
}
