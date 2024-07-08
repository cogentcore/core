// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package gradient

import (
	"image/color"

	"cogentcore.org/core/math32"
)

// Linear represents a linear gradient. It implements the [image.Image] interface.
type Linear struct { //types:add -setters
	Base

	// the starting point of the gradient (x1 and y1 in SVG)
	Start math32.Vector2

	// the ending point of the gradient (x2 and y2 in SVG)
	End math32.Vector2

	// computed current render versions transformed by object matrix
	rStart                math32.Vector2
	rEnd                  math32.Vector2
	distance              math32.Vector2
	distanceLengthSquared float32
}

var _ Gradient = &Linear{}

// NewLinear returns a new left-to-right [Linear] gradient.
func NewLinear() *Linear {
	return &Linear{
		Base: NewBase(),
		// default in SVG is LTR
		End: math32.Vec2(1, 0),
	}
}

// AddStop adds a new stop with the given color, position, and
// optional opacity to the gradient.
func (l *Linear) AddStop(color color.RGBA, pos float32, opacity ...float32) *Linear {
	l.Base.AddStop(color, pos, opacity...)
	return l
}

// Update updates the computed fields of the gradient, using
// the given current bounding box, and additional
// object-level transform (i.e., the current painting transform),
// which is applied in addition to the gradient's own Transform.
// This must be called before rendering the gradient, and it should only be called then.
func (l *Linear) Update(opacity float32, box math32.Box2, objTransform math32.Matrix2) {
	l.Box = box
	l.Opacity = opacity
	l.updateBase()

	if l.Units == ObjectBoundingBox {
		sz := l.Box.Size()
		l.rStart = l.Box.Min.Add(sz.Mul(l.Start))
		l.rEnd = l.Box.Min.Add(sz.Mul(l.End))
	} else {
		l.rStart = l.Transform.MulVector2AsPoint(l.Start)
		l.rEnd = l.Transform.MulVector2AsPoint(l.End)
		l.rStart = objTransform.MulVector2AsPoint(l.rStart)
		l.rEnd = objTransform.MulVector2AsPoint(l.rEnd)
	}

	l.distance = l.rEnd.Sub(l.rStart)
	l.distanceLengthSquared = l.distance.LengthSquared()
}

// At returns the color of the linear gradient at the given point
func (l *Linear) At(x, y int) color.Color {
	switch len(l.Stops) {
	case 0:
		return color.RGBA{}
	case 1:
		return l.Stops[0].OpacityColor(l.Opacity, l.ApplyFuncs)
	}

	pt := math32.Vec2(float32(x)+0.5, float32(y)+0.5)
	if l.Units == ObjectBoundingBox {
		pt = l.boxTransform.MulVector2AsPoint(pt)
	}
	df := pt.Sub(l.rStart)
	pos := (l.distance.X*df.X + l.distance.Y*df.Y) / l.distanceLengthSquared
	return l.getColor(pos)
}
