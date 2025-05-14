// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package ppath

import "cogentcore.org/core/math32"

func QuadraticToCubicBezier(p0, p1, p2 math32.Vector2) (math32.Vector2, math32.Vector2) {
	c1 := p0.Lerp(p1, 2.0/3.0)
	c2 := p2.Lerp(p1, 2.0/3.0)
	return c1, c2
}

func QuadraticBezierDeriv(p0, p1, p2 math32.Vector2, t float32) math32.Vector2 {
	p0 = p0.MulScalar(-2.0 + 2.0*t)
	p1 = p1.MulScalar(2.0 - 4.0*t)
	p2 = p2.MulScalar(2.0 * t)
	return p0.Add(p1).Add(p2)
}

func CubicBezierDeriv(p0, p1, p2, p3 math32.Vector2, t float32) math32.Vector2 {
	p0 = p0.MulScalar(-3.0 + 6.0*t - 3.0*t*t)
	p1 = p1.MulScalar(3.0 - 12.0*t + 9.0*t*t)
	p2 = p2.MulScalar(6.0*t - 9.0*t*t)
	p3 = p3.MulScalar(3.0 * t * t)
	return p0.Add(p1).Add(p2).Add(p3)
}
