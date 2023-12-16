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

// Radial represents a radial gradient. It implements the [image.Image] interface.
type Radial struct { //gti:add -setters
	Base

	// the center point of the gradient (cx and cy in SVG)
	Center mat32.Vec2

	// the focal point of the gradient (fx and fy in SVG)
	Focal mat32.Vec2

	// the radius of the gradient (rx and ry in SVG)
	Radius mat32.Vec2
}

// NewRadial returns a new centered [Radial] gradient.
func NewRadial() *Radial {
	return &Radial{
		// default is fully centered
		Center: mat32.V2Scalar(0.5),
		Focal:  mat32.V2Scalar(0.5),
		Radius: mat32.V2Scalar(0.5),
	}
}

// AddStop adds a new stop with the given color and position to the radial gradient.
func (r *Radial) AddStop(color color.RGBA, pos float32) *Radial {
	r.Base.AddStop(color, pos)
	return r
}

// At returns the color of the radial gradient at the given point
func (r *Radial) At(x, y int) color.Color {
	switch len(r.Stops) {
	case 0:
		return color.RGBA{}
	case 1:
		return r.Stops[0].Color
	}

	if r.Center == r.Focal {
		// When the center and focal are the same things are much simpler;
		// pos is just distance from center scaled by r
		v := mat32.V2(float32(x)+0.5, float32(y)+0.5)
		d := v.Sub(r.Center)
		pos := mat32.Sqrt(d.X*d.X/(r.Radius.X*r.Radius.X) + (d.Y*d.Y)/(r.Radius.Y*r.Radius.Y))
		return r.GetColor(pos)
	}

	c, f := r.Center.Div(r.Radius), r.Focal.Div(r.Radius)

	df := f.Sub(c)

	if df.X*df.X+df.Y*df.Y > 1 { // Focus outside of circle; use intersection
		// point of line from center to focus and circle as per SVG specs.
		nf, intersects := RayCircleIntersectionF(f, c, c, 1.0-epsilonF)
		f = nf
		if !intersects {
			return color.RGBA{} // should not happen
		}
	}

	v := mat32.V2(float32(x)+0.5, float32(y)+0.5)
	e := v.Div(r.Radius)

	t1, intersects := RayCircleIntersectionF(e, f, c, 1)
	if !intersects { // In this case, use the last stop color
		s := r.Stops[len(r.Stops)-1]
		return s.Color
	}

	td := t1.Sub(f)
	d := e.Sub(f)
	if td.X*td.X+td.Y*td.Y < epsilonF {
		s := r.Stops[len(r.Stops)-1]
		return s.Color
	}

	pos := mat32.Sqrt(d.X*d.X+d.Y*d.Y) / mat32.Sqrt(td.X*td.X+td.Y*td.Y)
	return r.GetColor(pos)
}
