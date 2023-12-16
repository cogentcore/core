// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package gradient

import "goki.dev/mat32/v2"

// Linear represents a linear gradient.
type Linear struct {

	// the starting point of the gradient (x1 and y1 in SVG)
	Start mat32.Vec2

	// the ending point of the gradient (x2 and y2 in SVG)
	End mat32.Vec2

	// the stops for the gradient
	Stops []GradientStop

	// the spread method used for the gradient
	Spread SpreadMethods

	// the units used for the gradient
	Units GradientUnits

	// the colorspace algorithm to use for blending colors
	Blend BlendTypes

	// the bounds of the gradient; this should typically not be set by end-users
	Bounds mat32.Box2

	// the matrix for the gradient; this should typically not be set by end-users
	Matrix mat32.Mat2
}

// NewLinear returns a new [Linear] gradient
func NewLinear() *Linear {
	return &Linear{
		Spread: PadSpread,
		End:    mat32.Vec2{0, 1},
		Matrix: mat32.Identity2D(),
		Bounds: mat32.NewBox2(mat32.Vec2{}, mat32.Vec2{1, 1}),
	}
}
