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
	Start math32.Vec2

	// the ending point of the gradient (x2 and y2 in SVG)
	End math32.Vec2

	// current render version -- transformed by object matrix
	rStart math32.Vec2 `set:"-"`

	// current render version -- transformed by object matrix
	rEnd math32.Vec2 `set:"-"`
}

var _ Gradient = &Linear{}

// NewLinear returns a new left-to-right [Linear] gradient.
func NewLinear() *Linear {
	return &Linear{
		Base: NewBase(),
		// default in SVG is LTR
		End: math32.V2(1, 0),
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
func (l *Linear) Update(opacity float32, box math32.Box2, objTransform math32.Mat2) {
	l.Box = box
	l.Opacity = opacity
	l.UpdateBase()

	if l.Units == ObjectBoundingBox {
		sz := l.Box.Size()
		l.rStart = l.Box.Min.Add(sz.Mul(l.Start))
		l.rEnd = l.Box.Min.Add(sz.Mul(l.End))
	} else {
		l.rStart = l.Transform.MulVec2AsPoint(l.Start)
		l.rEnd = l.Transform.MulVec2AsPoint(l.End)
		l.rStart = objTransform.MulVec2AsPoint(l.rStart)
		l.rEnd = objTransform.MulVec2AsPoint(l.rEnd)
	}
}

// At returns the color of the linear gradient at the given point
func (l *Linear) At(x, y int) color.Color {
	switch len(l.Stops) {
	case 0:
		return color.RGBA{}
	case 1:
		return l.Stops[0].OpacityColor(l.Opacity, l.ApplyFuncs)
	}

	d := l.rEnd.Sub(l.rStart)
	dd := d.X*d.X + d.Y*d.Y // self inner prod

	pt := math32.V2(float32(x)+0.5, float32(y)+0.5)
	if l.Units == ObjectBoundingBox {
		pt = l.boxTransform.MulVec2AsPoint(pt)
	}
	df := pt.Sub(l.rStart)
	pos := (d.X*df.X + d.Y*df.Y) / dd
	return l.GetColor(pos)
}
