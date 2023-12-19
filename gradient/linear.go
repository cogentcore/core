// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package gradient

import (
	"image/color"

	"goki.dev/mat32/v2"
)

// Linear represents a linear gradient. It implements the [image.Image] interface.
type Linear struct { //gti:add -setters
	Base

	// the starting point of the gradient (x1 and y1 in SVG)
	Start mat32.Vec2

	// the ending point of the gradient (x2 and y2 in SVG)
	End mat32.Vec2

	// EffStart is the computed effective transformed starting point of the gradient.
	// It should not be set by end users.
	EffStart mat32.Vec2 `set:"-"`

	// EffEnd is the computed effective transformed ending point of the gradient.
	// It should not be set by end users.
	EffEnd mat32.Vec2 `set:"-"`
}

var _ Gradient = &Linear{}

// NewLinear returns a new left-to-right [Linear] gradient.
func NewLinear() *Linear {
	return &Linear{
		Base: NewBase(),
		// default in SVG is LTR
		End: mat32.V2(1, 0),
	}
}

// AddStop adds a new stop with the given color and position to the linear gradient.
func (l *Linear) AddStop(color color.RGBA, pos float32) *Linear {
	l.Base.AddStop(color, pos)
	return l
}

// Update updates the computed fields of the gradient. It must be
// called before rendering the gradient, and it should only be called then.
func (l *Linear) Update() {
	l.UpdateBase()

	if l.Units == ObjectBoundingBox {
		l.EffStart = l.Box.Min.Add(l.Box.Size().Mul(l.Start))
		l.EffEnd = l.Box.Min.Add(l.Box.Size().Mul(l.End))
	} else {
		l.EffStart = l.Transform.MulVec2AsPt(l.Start)
		l.EffEnd = l.Transform.MulVec2AsPt(l.End)
	}
}

// At returns the color of the linear gradient at the given point
func (l *Linear) At(x, y int) color.Color {
	switch len(l.Stops) {
	case 0:
		return color.RGBA{}
	case 1:
		return l.Stops[0].Color
	}

	d := l.EffEnd.Sub(l.EffStart)
	dd := d.X*d.X + d.Y*d.Y // self inner prod

	pt := mat32.V2(float32(x)+0.5, float32(y)+0.5)
	if l.Units == ObjectBoundingBox {
		pt = l.ObjectMatrix.MulVec2AsPt(pt)
	}
	df := pt.Sub(l.EffStart)
	pos := (d.X*df.X + d.Y*df.Y) / dd
	return l.GetColor(pos)
}
