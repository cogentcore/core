// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package gradient

import (
	"cmp"
	"image"
	"image/color"
	"math"
	"slices"

	"goki.dev/colors"
	"goki.dev/mat32/v2"
)

// Linear represents a linear gradient. It implements the [image.Image] interface.
type Linear struct { //gti:add -setters

	// the starting point of the gradient (x1 and y1 in SVG)
	Start mat32.Vec2

	// the ending point of the gradient (x2 and y2 in SVG)
	End mat32.Vec2

	// the stops for the gradient
	Stops []Stop `set:"-"`

	// the spread method used for the gradient if it stops before the end
	Spread SpreadMethods

	// the colorspace algorithm to use for blending colors
	Blend BlendTypes

	// the matrix for the gradient; this should typically not be set by end-users
	Matrix mat32.Mat2 `set:"-"`
}

var _ image.Image = &Linear{}

// NewLinear returns a new [Linear] gradient.
func NewLinear() *Linear {
	return &Linear{
		End:    mat32.Vec2{0, 1},
		Matrix: mat32.Identity2D(),
	}
}

// AddStop adds a new stop with the given color and position to the linear gradient.
func (l *Linear) AddStop(color color.RGBA, pos float32) *Linear {
	l.Stops = append(l.Stops, Stop{color, pos})
	return l
}

// ColorModel returns the color model used by the gradient, which is [color.RGBAModel]
func (l *Linear) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds returns the bounds of the gradient
func (l *Linear) Bounds() image.Rectangle {
	return image.Rect(math.MinInt, math.MinInt, math.MaxInt, math.MaxInt)
}

// At returns the color of the gradient at the given point
func (l *Linear) At(x, y int) color.Color {
	switch len(l.Stops) {
	case 0:
		return colors.Black // default error color for gradient w/o stops.
	case 1:
		return l.Stops[0].Color // Illegal, I think, should really should not happen.
	}

	// sort by pos in ascending order
	slices.SortFunc(l.Stops, func(a, b Stop) int {
		return cmp.Compare(a.Pos, b.Pos)
	})

	s, e := l.Start, l.End
	s = l.Matrix.MulVec2AsPt(s)
	e = l.Matrix.MulVec2AsPt(e)

	d := e.Sub(s)
	dd := d.X*d.X + d.Y*d.Y // self inner prod

	pt := mat32.V2(float32(x)+0.5, float32(y)+0.5)
	df := pt.Sub(s)
	return l.ColorAt((d.X*df.X + d.Y*df.Y) / dd)
}
