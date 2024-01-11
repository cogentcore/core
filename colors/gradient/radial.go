// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package gradient

import (
	"image/color"

	"goki.dev/goki/mat32"
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

var _ Gradient = &Radial{}

// NewRadial returns a new centered [Radial] gradient.
func NewRadial() *Radial {
	return &Radial{
		Base: NewBase(),
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

// Update updates the computed fields of the gradient. It must be
// called before rendering the gradient, and it should only be called then.
func (r *Radial) Update() {
	r.UpdateBase()
}

const epsilonF = 1e-5

// At returns the color of the radial gradient at the given point
func (r *Radial) At(x, y int) color.Color {
	switch len(r.Stops) {
	case 0:
		return color.RGBA{}
	case 1:
		return r.Stops[0].Color
	}

	c, f, rs := r.Center, r.Focal, r.Radius
	if r.Units == ObjectBoundingBox {
		c = r.Box.Min.Add(r.Box.Size().Mul(c))
		f = r.Box.Min.Add(r.Box.Size().Mul(f))
		rs.SetMul(r.Box.Size())
	} else {
		c = r.Transform.MulVec2AsPt(c)
		f = r.Transform.MulVec2AsPt(f)
		rs = r.Transform.MulVec2AsVec(rs)
	}

	if r.Center == r.Focal {
		// When the center and focal are the same things are much simpler;
		// pos is just distance from center scaled by radius
		pt := mat32.V2(float32(x)+0.5, float32(y)+0.5)
		if r.Units == ObjectBoundingBox {
			pt = r.ObjectMatrix.MulVec2AsPt(pt)
		}
		d := pt.Sub(c)
		pos := mat32.Sqrt(d.X*d.X/(rs.X*rs.X) + (d.Y*d.Y)/(rs.Y*rs.Y))
		return r.GetColor(pos)
	}

	f.SetDiv(rs)
	c.SetDiv(rs)

	df := f.Sub(c)

	if df.X*df.X+df.Y*df.Y > 1 { // Focus outside of circle; use intersection
		// point of line from center to focus and circle as per SVG specs.
		nf, intersects := RayCircleIntersectionF(f, c, c, 1-epsilonF)
		f = nf
		if !intersects {
			return color.RGBA{} // should not happen
		}
	}

	pt := mat32.V2(float32(x)+0.5, float32(y)+0.5)
	if r.Units == ObjectBoundingBox {
		pt = r.ObjectMatrix.MulVec2AsPt(pt)
	}
	e := pt.Div(rs)

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

// RayCircleIntersectionF calculates in floating point the points of intersection of
// a ray starting at s2 passing through s1 and a circle in fixed point.
// Returns intersects == false if no solution is possible. If two
// solutions are possible, the point closest to s2 is returned
func RayCircleIntersectionF(s1, s2, c mat32.Vec2, r float32) (pt mat32.Vec2, intersects bool) {
	n := s2.X - c.X // Calculating using 64* rather than divide
	m := s2.Y - c.Y

	e := s2.X - s1.X
	d := s2.Y - s1.Y

	// Quadratic normal form coefficients
	A, B, C := e*e+d*d, -2*(e*n+m*d), n*n+m*m-r*r

	D := B*B - 4*A*C

	if D <= 0 {
		return // No intersection or is tangent
	}

	D = mat32.Sqrt(D)
	t1, t2 := (-B+D)/(2*A), (-B-D)/(2*A)
	p1OnSide := t1 > 0
	p2OnSide := t2 > 0

	switch {
	case p1OnSide && p2OnSide:
		if t2 < t1 { // both on ray, use closest to s2
			t1 = t2
		}
	case p2OnSide: // Only p2 on ray
		t1 = t2
	case p1OnSide: // only p1 on ray
	default: // Neither solution is on the ray
		return
	}
	return mat32.V2((n-e*t1)+c.X, (m-d*t1)+c.Y), true
}
