// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"math"
)

/*
This is verbatium from: https://github.com/fogleman/gg

Copyright (C) 2016 Michael Fogleman

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// from "golang.org/x/image/vector"
// devSquared returns a measure of how curvy the sequence (ax, ay) to (bx, by)
// to (cx, cy) is. It determines how many line segments will approximate a
// Bezier curve segment.
//
// http://lists.nongnu.org/archive/html/freetype-devel/2016-08/msg00080.html
// gives the rationale for this evenly spaced heuristic instead of a recursive
// de Casteljau approach:
//
// The reason for the subdivision by n is that I expect the "flatness"
// computation to be semi-expensive (it's done once rather than on each
// potential subdivision) and also because you'll often get fewer subdivisions.
// Taking a circular arc as a simplifying assumption (ie a spherical cow),
// where I get n, a recursive approach would get 2^[lg n], which, if I haven't
// made any horrible mistakes, is expected to be 33% more in the limit.
func devSquared(ax, ay, bx, by, cx, cy float32) float32 {
	devx := ax - 2*bx + cx
	devy := ay - 2*by + cy
	return devx*devx + devy*devy
}

func quadratic(x0, y0, x1, y1, x2, y2, t float32) (x, y float32) {
	u := 1 - t
	a := u * u
	b := 2 * u * t
	c := t * t
	x = a*x0 + b*x1 + c*x2
	y = a*y0 + b*y1 + c*y2
	return
}

func QuadraticBezier(x0, y0, x1, y1, x2, y2 float32) []Vec2D {
	// l := (math.Hypot(float64(x1-x0), float64(y1-y0)) + math.Hypot(float64(x2-x1), float64(y2-y1)))
	l := math.Sqrt(float64(devSquared(x0, y0, x1, y1, x2, y2)))
	n := int(l + 0.5)
	if n < 4 {
		n = 4
	}
	d := float32(n) - 1
	result := make([]Vec2D, n)
	for i := 0; i < n; i++ {
		t := float32(i) / d
		x, y := quadratic(x0, y0, x1, y1, x2, y2, t)
		result[i] = Vec2D{x, y}
	}
	return result
}

func cubic(x0, y0, x1, y1, x2, y2, x3, y3, t float32) (x, y float32) {
	u := 1 - t
	a := u * u * u
	b := 3 * u * u * t
	c := 3 * u * t * t
	d := t * t * t
	x = a*x0 + b*x1 + c*x2 + d*x3
	y = a*y0 + b*y1 + c*y2 + d*y3
	return
}

func CubicBezier(x0, y0, x1, y1, x2, y2, x3, y3 float32) []Vec2D {
	// l := (math.Hypot(float64(x1-x0), float64(y1-y0)) + math.Hypot(float64(x2-x1), float64(y2-y1)) + math.Hypot(float64(3-x2), float64(y3-y2)))
	l := math.Sqrt(float64(devSquared(x0, y0, x1, y1, x2, y2)))
	n := int(l + 0.5)
	if n < 4 {
		n = 4
	}
	d := float32(n) - 1
	result := make([]Vec2D, n)
	for i := 0; i < n; i++ {
		t := float32(i) / d
		x, y := cubic(x0, y0, x1, y1, x2, y2, x3, y3, t)
		result[i] = Vec2D{x, y}
	}
	return result
}
