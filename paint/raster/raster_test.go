// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package raster

import (
	"image"
	"image/color"

	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/iox/imagex"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/scan"
	"golang.org/x/image/math/fixed"
)

func getOpenCubicPath() (p Path) {
	p.Start(ToFixedP(50, 50))
	p.Line(ToFixedP(100, 50)) // Yes I meant to do this
	p.CubeBezier(ToFixedP(120, 70), ToFixedP(80, 90), ToFixedP(100, 100))
	p.CubeBezier(ToFixedP(101, 95), ToFixedP(80, 90), ToFixedP(75, 100))
	p.Line(ToFixedP(95, 120))
	p.Line(ToFixedP(78, 100))
	return
}

func getOpenCubicPath2() (p Path) {
	//M87, 212 C 138, 90,  269, 75, 259, 147 C 254, 71, 104,176, 128, 282z
	p.Start(ToFixedP(87, 212))
	p.CubeBezier(ToFixedP(138, 90), ToFixedP(269, 75), ToFixedP(259, 147))
	p.CubeBezier(ToFixedP(254, 71), ToFixedP(104, 176), ToFixedP(128, 282))
	p.Stop(true)

	p.Start(ToFixedP(600-87, 212))
	p.CubeBezier(ToFixedP(600-138, 90), ToFixedP(600-269, 75), ToFixedP(600-259, 147))
	p.CubeBezier(ToFixedP(600-254, 71), ToFixedP(600-104, 176), ToFixedP(600-128, 282))
	p.Stop(true)
	return
}

func getPartPath() (testPath Path) {
	//M210.08,222.97
	testPath.Start(ToFixedP(210.08, 222.97))
	//L192.55,244.95
	testPath.Line(ToFixedP(192.55, 244.95))
	//Q146.53,229.95,115.55,209.55
	testPath.QuadBezier(ToFixedP(146.53, 229.95), ToFixedP(115.55, 209.55))
	//Q102.50,211.00,95.38,211.00
	testPath.QuadBezier(ToFixedP(102.50, 211.00), ToFixedP(95.38, 211.00))
	//Q56.09,211.00,31.17,182.33
	testPath.QuadBezier(ToFixedP(56.09, 211.00), ToFixedP(31.17, 182.33))
	//Q6.27,153.66,6.27,108.44
	testPath.QuadBezier(ToFixedP(6.27, 153.66), ToFixedP(6.27, 108.44))
	//Q6.27,61.89,31.44,33.94
	testPath.QuadBezier(ToFixedP(6.27, 61.89), ToFixedP(31.44, 33.94))
	//Q56.62,6.00,98.55,6.00
	testPath.QuadBezier(ToFixedP(56.62, 6.00), ToFixedP(98.55, 6.00))
	//Q141.27,6.00,166.64,33.88
	testPath.QuadBezier(ToFixedP(141.27, 6.00), ToFixedP(166.64, 33.88))
	//Q192.02,61.77,192.02,108.70
	testPath.QuadBezier(ToFixedP(192.02, 61.77), ToFixedP(192.02, 108.70))
	//Q192.02,175.67,140.86,202.05
	testPath.QuadBezier(ToFixedP(192.02, 175.67), ToFixedP(140.86, 202.05))
	//Q173.42,216.66,210.08,222.97
	testPath.QuadBezier(ToFixedP(173.42, 216.66), ToFixedP(210.08, 222.97))
	//z
	testPath.Stop(true)
	return
}

func GetTestPath() (testPath Path) {
	//Path for Q
	testPath = getPartPath()

	testPath.ToSVGPath()

	//M162.22,109.69 M162.22,109.69
	testPath.Start(ToFixedP(162.22, 109.69))
	//Q162.22,70.11,145.61,48.55
	testPath.QuadBezier(ToFixedP(162.22, 70.11), ToFixedP(145.61, 48.55))
	//Q129.00,27.00,98.42,27.00
	testPath.QuadBezier(ToFixedP(129.00, 27.00), ToFixedP(98.42, 27.00))
	//Q69.14,27.00,52.53,48.62
	testPath.QuadBezier(ToFixedP(69.14, 27.00), ToFixedP(52.53, 48.62))
	//Q35.92,70.25,35.92,108.50
	testPath.QuadBezier(ToFixedP(35.92, 70.25), ToFixedP(35.92, 108.50))
	//Q35.92,146.75,52.53,168.38
	testPath.QuadBezier(ToFixedP(35.92, 146.75), ToFixedP(52.53, 168.38))
	//Q69.14,190.00,98.42,190.00
	testPath.QuadBezier(ToFixedP(69.14, 190.00), ToFixedP(98.42, 190.00))
	//Q128.34,190.00,145.28,168.70
	testPath.QuadBezier(ToFixedP(128.34, 190.00), ToFixedP(145.28, 168.70))
	//Q162.22,147.41,162.22,109.69
	testPath.QuadBezier(ToFixedP(162.22, 147.41), ToFixedP(162.22, 109.69))
	//z
	testPath.Stop(true)

	return
}

func BenchmarkScan(b *testing.B) {
	var (
		p       = GetTestPath()
		wx, wy  = 512, 512
		img     = image.NewRGBA(image.Rect(0, 0, wx, wy))
		scanner = scan.NewScanner(scan.NewImgSpanner(img), wx, wy)
	)
	f := NewFiller(wx, wy, scanner)
	p.AddTo(f)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Draw()
	}
}

func BenchmarkFill(b *testing.B) {
	var (
		p       = GetTestPath()
		wx, wy  = 512, 512
		img     = image.NewRGBA(image.Rect(0, 0, wx, wy))
		scanner = scan.NewScanner(scan.NewImgSpanner(img), wx, wy)
	)
	f := NewFiller(wx, wy, scanner)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.AddTo(f)
		f.Draw()
		f.Clear()
	}
}

func BenchmarkDash(b *testing.B) {
	var (
		p       = GetTestPath()
		wx, wy  = 512, 512
		img     = image.NewRGBA(image.Rect(0, 0, wx, wy))
		scanner = scan.NewScanner(scan.NewImgSpanner(img), wx, wy)
	)
	b.ResetTimer()
	d := NewDasher(wx, wy, scanner)
	d.SetStroke(10*64, 4*64, RoundCap, nil, RoundGap, ArcClip, []float32{33, 12}, 0)
	for i := 0; i < b.N; i++ {
		p.AddTo(d)
		d.Draw()
		d.Clear()
	}
}

func TestRoundRect(t *testing.T) {
	var (
		wx, wy  = 512, 512
		img     = image.NewRGBA(image.Rect(0, 0, wx, wy))
		scanner = scan.NewScanner(scan.NewImgSpanner(img), wx, wy)
		f       = NewFiller(wx, wy, scanner)
	)

	scanner.SetColor(colors.C(colors.Cadetblue))
	AddRoundRect(30, 30, 130, 130, 40, 40, 0, RoundGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Burlywood))
	AddRoundRect(140, 30, 240, 130, 10, 40, 0, RoundGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Yellowgreen))
	AddRoundRect(250, 30, 350, 130, 40, 10, 0, RoundGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Lightgreen))
	AddRoundRect(370, 30, 470, 130, 20, 20, 45, RoundGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Cadetblue))
	AddRoundRect(30, 140, 130, 240, 40, 40, 0, QuadraticGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Burlywood))
	AddRoundRect(140, 140, 240, 240, 10, 40, 0, QuadraticGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Yellowgreen))
	AddRoundRect(250, 140, 350, 240, 40, 10, 0, QuadraticGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Blueviolet))
	AddRoundRect(370, 140, 470, 240, 20, 20, 45, QuadraticGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Cadetblue))
	AddRoundRect(30, 250, 130, 350, 40, 40, 0, CubicGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Burlywood))
	AddRoundRect(140, 250, 240, 350, 10, 40, 0, CubicGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Yellowgreen))
	AddRoundRect(250, 250, 350, 350, 40, 10, 0, CubicGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Lightgreen))
	AddRoundRect(370, 250, 470, 350, 20, 20, 45, CubicGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Cadetblue))
	AddRoundRect(30, 360, 130, 460, 40, 40, 0, FlatGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Burlywood))
	AddRoundRect(140, 360, 240, 460, 10, 40, 0, FlatGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Yellowgreen))
	AddRoundRect(250, 360, 350, 460, 40, 10, 0, FlatGap, f)
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Blueviolet))
	AddRoundRect(370, 360, 470, 460, 20, 20, 45, FlatGap, f)
	f.Draw()
	f.Clear()

	imagex.Assert(t, img, "roundRect")
}

func isClose(a, b math32.Matrix2, epsilon float32) bool {
	return !(math32.Abs(a.XX-b.XX) > epsilon ||
		math32.Abs(a.YX-b.YX) > epsilon ||
		math32.Abs(a.XY-b.XY) > epsilon ||
		math32.Abs(a.YY-b.YY) > epsilon ||
		math32.Abs(a.X0-b.X0) > epsilon ||
		math32.Abs(a.Y0-b.Y0) > epsilon)
}

func TestCircleLineIntersect(t *testing.T) {
	a := fixed.Point26_6{X: 30 * 64, Y: 55 * 64}
	b := fixed.Point26_6{X: 40 * 64, Y: 40 * 64}
	c := fixed.Point26_6{X: 40 * 64, Y: 40 * 64}
	r := fixed.Int26_6(10 * 64)
	_, touching := RayCircleIntersection(a, b, c, r)
	if touching == false {
		t.Error("Ray not intersecting circle ", touching)
	}
}

func TestToLength(t *testing.T) {
	p := fixed.Point26_6{X: 2, Y: -2}
	ln := fixed.I(40)

	q := ToLength(p, ln)
	expected := fixed.Point26_6{X: 1810, Y: -1810}
	if q != expected {
		t.Error("wrong point", q)
	}
}

func TestShapes(t *testing.T) {
	var (
		wx, wy = 512, 512

		img = image.NewRGBA(image.Rect(0, 0, wx, wy))

		scanner = scan.NewScanner(scan.NewImgSpanner(img), wx, wy)
		f       = NewFiller(wx, wy, scanner)
		s       = NewStroker(wx, wy, scanner)
		d       = NewDasher(wx, wy, scanner)
	)

	doShapes(t, f, f, "shapeF", img)

	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	s.SetStroke(10*64, 4*64, RoundCap, nil, RoundGap, ArcClip)
	doShapes(t, s, s, "shapeS1", img)

	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	s.SetStroke(10*64, 4*64, nil, RoundCap, RoundGap, ArcClip)
	doShapes(t, s, s, "shapeS2", img)

	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	s.SetStroke(10*64, 4*64, nil, nil, nil, Miter)
	doShapes(t, s, s, "shapeS3", img)

	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	d.SetStroke(10*64, 4*64, SquareCap, nil, RoundGap, ArcClip, []float32{33, 12}, 30)
	doShapes(t, d, d, "shapeD0", img)

	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	d.SetStroke(10*64, 4*64, RoundCap, nil, RoundGap, Miter, []float32{33, 12}, 250)
	doShapes(t, d, d, "shapeD1", img)

	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	d.SetStroke(10*64, 4*64, ButtCap, CubicCap, QuadraticGap, Arc, []float32{33, 12}, -30)
	doShapes(t, d, d, "shapeD2", img)

	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	d.SetStroke(10*64, 4*64, nil, QuadraticCap, RoundGap, MiterClip, []float32{12, 4}, 14)
	doShapes(t, d, d, "shapeD3", img)

	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	d.SetStroke(10*64, 4*64, RoundCap, nil, RoundGap, Bevel, []float32{0, 0}, 0)
	doShapes(t, d, d, "shapeD4", img)

	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	d.SetStroke(10*64, 4*64, SquareCap, nil, nil, Round, []float32{}, 0)
	doShapes(t, d, d, "shapeD5", img)

	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	d.SetStroke(10*64, 4*64, RoundCap, nil, RoundGap, MiterClip, nil, 0)
	doShapes(t, d, d, "shapeD6", img)

	getOpenCubicPath().AddTo(f)
	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	f.Draw()
	f.Clear()

	s.SetStroke(4*64, 1, SquareCap, nil, RoundGap, ArcClip)
	getOpenCubicPath().AddTo(s)
	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	s.Draw()
	s.Clear()

	imagex.Assert(t, img, "shapeT1")

	s.SetStroke(4<<6, 2<<6, SquareCap, nil, RoundGap, ArcClip)
	getOpenCubicPath2().AddTo(s)
	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	s.Draw()
	s.Clear()

	imagex.Assert(t, img, "shapeT2")

	s.SetStroke(25<<6, 200<<6, CubicCap, CubicCap, CubicGap, ArcClip)
	p := getOpenCubicPath2()
	p.AddTo(s)
	_ = p.String() // Just flexes to ToSVGString
	img = image.NewRGBA(image.Rect(0, 0, wx, wy))
	scanner.Spanner = scan.NewImgSpanner(img)
	s.Draw()
	s.Clear()
	p.Clear()

	imagex.Assert(t, img, "shapeT3")

	d.SetBounds(-20, -12) // Test min x and y value checking

}

func doShapes(t *testing.T, f Scanner, fa Adder, fname string, img image.Image) {
	f.SetColor(colors.C(colors.Blueviolet))
	AddEllipse(240, 200, 140, 180, 0, fa)
	f.Draw()
	f.Clear()

	f.SetColor(colors.C(colors.Darkseagreen))
	AddEllipse(240, 200, 40, 180, 45, fa)
	f.Draw()
	f.Clear()

	f.SetColor(colors.C(colors.Darkgoldenrod))
	AddCircle(300, 300, 80, fa)
	f.Draw()
	f.Clear()

	f.SetColor(colors.C(colors.Forestgreen))
	AddRoundRect(30, 30, 130, 130, 10, 20, 45, RoundGap, fa)
	f.Draw()
	f.Clear()

	f.SetColor(colors.C(colors.Blueviolet))
	AddRoundRect(30, 30, 130, 130, 150, 150, 0, nil, fa)
	f.Draw()
	f.Clear()

	f.SetColor(colors.C(colors.ApplyOpacity(colors.Lightgoldenrodyellow, 0.6)))
	AddCircle(80, 80, 50, fa)
	f.Draw()
	f.Clear()

	f.SetColor(colors.C(colors.Lemonchiffon))
	f.SetClip(image.Rect(65, 65, 95, 95))
	AddCircle(80, 80, 50, fa)
	f.Draw()
	f.Clear()

	f.SetClip(image.ZR)

	f.SetColor(colors.C(colors.Firebrick))
	AddRect(370, 370, 400, 500, 15, fa)
	f.Draw()
	f.Clear()

	imagex.Assert(t, img, fname)
}

func TestFindElipsecenter(t *testing.T) {
	ra, rb := float32(10), float32(5)
	cx, cy := FindEllipseCenter(&ra, &rb, 0.0, 0.0, 0.0, 20.0, 0.0, true, true)
	if cx != 10 || cy != 0 || ra != 10 || rb != 5 {
		t.Error("Find elipse center failed ", cx, cy, ra, rb)
	}
	cx, cy = FindEllipseCenter(&ra, &rb, 0.0, 0.0, 0.0, 35.0, 5.0, false, true)
	if ra == 10 || rb == 5 {
		t.Error("Find elipse center failed with resize of radiuses ", cx, cy, ra, rb)
	}
	ra, rb = 5.0, 5.0
	cx, cy = FindEllipseCenter(&ra, &rb, 0.0, 0.0, 0.0, 35.0, 5.0, true, true)
	if ra == 10 || rb == 5 {
		t.Error("Find elipse center failed with resize of radiuses ", cx, cy, ra, rb)
	}
}

// TestGradient tests a Dasher's ability to function
// as a filler, stroker, and dasher by invoking the corresponding anonymous structs
func TestGradient(t *testing.T) {
	var (
		wx, wy  = 512, 512
		img     = image.NewRGBA(image.Rect(0, 0, wx, wy))
		scanner = scan.NewScanner(scan.NewImgSpanner(img), wx, wy)
	)

	linear := gradient.NewLinear().
		SetBox(math32.B2(50, 50, 150, 150)).
		AddStop(colors.Aquamarine, 0.3).
		AddStop(colors.Skyblue, 0.6).
		AddStop(colors.ApplyOpacity(colors.Darksalmon, 0.75), 1)

	radial := gradient.NewRadial().
		SetBox(math32.B2(230, 230, 330, 330)).SetSpread(gradient.Reflect).
		AddStop(colors.Orchid, 0.3).
		AddStop(colors.Bisque, 0.6).
		AddStop(colors.ApplyOpacity(colors.Chartreuse, 0.4), 1)

	d := NewDasher(wx, wy, scanner)
	d.SetStroke(10*64, 4*64, RoundCap, nil, RoundGap, ArcClip, []float32{33, 12}, 0)
	// p is in the shape of a capital Q
	p := getPartPath()

	f := &d.Filler // This is the anon Filler in the Dasher. It also satisfies
	// the Rasterizer interface, and can only perform a fill on the path.

	offsetPath := &MatrixAdder{Adder: f, M: math32.Identity2().Translate(180, 180)}

	p.AddTo(offsetPath)

	scanner.SetColor(radial)
	f.Draw()
	f.Clear()

	scanner.SetClip(image.Rect(420, 350, 460, 400))
	offsetPath.M = math32.Identity2().Translate(340, 180)
	scanner.SetColor(radial)
	p.AddTo(offsetPath)
	f.Draw()
	f.Clear()
	scanner.SetClip(image.ZR)
	offsetPath.M = math32.Identity2().Translate(180, 340)
	p.AddTo(offsetPath)
	f.Draw()
	f.Clear()
	offsetPath.Reset()
	if isClose(offsetPath.M, math32.Identity2(), 1e-12) == false {
		t.Error("path reset failed", offsetPath)
	}

	scanner.SetColor(linear)
	p.AddTo(f)
	f.Draw()
	f.Clear()

	linear.SetSpread(gradient.Repeat)
	scanner.SetColor(linear)
	AddRect(20, 460, 150, 610, 45, f)
	f.Draw()
	f.Clear()

	radial.SetUnits(gradient.UserSpaceOnUse)
	scanner.SetColor(radial)
	AddRect(300, 20, 450, 170, 0, f)
	f.Draw()
	f.Clear()

	linear.SetUnits(gradient.UserSpaceOnUse)
	scanner.SetColor(linear)
	AddRect(300, 180, 450, 200, 0, f)
	f.Draw()
	f.Clear()

	radial.SetUnits(gradient.ObjectBoundingBox).
		SetFocal(math32.Vector2{}).SetRadius(math32.Vector2Scalar(0.2)) // move focus away from
	scanner.SetColor(radial)
	AddRect(300, 210, 450, 300, 0, f)
	f.Draw()
	f.Clear()

	radial.SetUnits(gradient.UserSpaceOnUse).SetSpread(gradient.Pad).
		SetFocal(math32.Vec2(0.1, 0.1)).SetRadius(math32.Vector2Scalar(0.5)) // move focus away from center
	scanner.SetColor(radial)
	AddRect(20, 160, 150, 310, 0, f)
	f.Draw()
	f.Clear()

	linear.Stops = linear.Stops[0:1]
	scanner.SetColor(linear)
	AddRect(300, 180, 450, 200, 0, f)
	f.Draw()
	f.Clear()

	linear.Stops = linear.Stops[0:0]
	scanner.SetColor(linear)
	AddRect(300, 180, 450, 200, 0, f)
	f.Draw()
	f.Clear()

	// Lets try a sinusoidal grid pattern.
	cf := func(x, y int) color.Color {
		sinx, siny, sinxy := math32.Sin(float32(x)*math32.Pi/20), math32.Sin(float32(y)*math32.Pi/10),
			math32.Sin(float32(y+x)*math32.Pi/30)
		r := (1 + sinx) * 120
		g := (1 + siny) * 120
		b := (1 + sinxy) * 120
		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	}

	scanner.SetColor(colors.Pattern(cf))
	AddRect(20, 300, 150, 450, 0, f)

	f.Draw()
	f.Clear()

	imagex.Assert(t, img, "grad")
}

// TestMultiFunction tests a Dasher's ability to function
// as a filler, stroker, and dasher by invoking the corresponding anonymous structs
func TestMultiFunction(t *testing.T) {

	var (
		wx, wy  = 512, 512
		img     = image.NewRGBA(image.Rect(0, 0, wx, wy))
		scanner = scan.NewScanner(scan.NewImgSpanner(img), wx, wy)
	)

	scanner.SetColor(colors.C(colors.Cornflowerblue))
	d := NewDasher(wx, wy, scanner)
	d.SetStroke(10*64, 4*64, RoundCap, nil, RoundGap, ArcClip, []float32{33, 12}, 0)
	// p is in the shape of a capital Q
	p := GetTestPath()

	f := &d.Filler // This is the anon Filler in the Dasher. It also satisfies
	// the Rasterizer interface, and will only perform a fill on the path.

	p.AddTo(f)

	extentR := scanner.GetPathExtent()
	x := int(extentR.Max.X)
	y := int(extentR.Max.Y)
	if x != 13445 && y != 15676 {
		t.Error("test extent Max value not as expected")
	}
	f.Draw()
	f.Clear()

	scanner.SetColor(colors.C(colors.Cornsilk))

	s := &d.Stroker // This is the anon Stroke in the Dasher. It also satisfies
	// the Rasterizer interface, but will perform a fill on the path.
	p.AddTo(s)
	s.Draw()
	s.Clear()

	scanner.SetColor(colors.C(colors.Darkolivegreen))

	// Now lets use the Dasher itself; it will perform a dashed stroke if dashes are set
	// in the SetStroke method.
	p.AddTo(d)
	d.Draw()
	d.Clear()

	imagex.Assert(t, img, "tmf")
}
