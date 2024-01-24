// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package gradient

import (
	"image/color"

	"cogentcore.org/core/mat32"
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

	// current render version -- transformed by object matrix
	rCenter mat32.Vec2

	// current render version -- transformed by object matrix
	rFocal mat32.Vec2

	// current render version -- transformed by object matrix
	rRadius mat32.Vec2
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

// AddStop adds a new stop with the given color, position, and
// optional opacity to the gradient.
func (r *Radial) AddStop(color color.RGBA, pos float32, opacity ...float32) *Radial {
	r.Base.AddStop(color, pos, opacity...)
	return r
}

// Update updates the computed fields of the gradient, using
// the given current bounding box, and additional
// object-level transform (i.e., the current painting transform),
// which is applied in addition to the gradient's own Transform.
// This must be called before rendering the gradient, and it should only be called then.
func (r *Radial) Update(opacity float32, box mat32.Box2, objTransform mat32.Mat2) {
	r.Box = box
	r.Opacity = opacity
	r.UpdateBase()

	c, f, rs := r.Center, r.Focal, r.Radius
	sz := r.Box.Size()
	if r.Units == ObjectBoundingBox {
		c = r.Box.Min.Add(sz.Mul(c))
		f = r.Box.Min.Add(sz.Mul(f))
		rs.SetMul(sz)
	} else {
		c = r.Transform.MulVec2AsPt(c)
		f = r.Transform.MulVec2AsPt(f)
		rs = r.Transform.MulVec2AsVec(rs)
		c = objTransform.MulVec2AsPt(c)
		f = objTransform.MulVec2AsPt(f)
		rs = objTransform.MulVec2AsVec(rs)
	}
	if c != f {
		f.SetDiv(rs)
		c.SetDiv(rs)
		df := f.Sub(c)
		if df.X*df.X+df.Y*df.Y > 1 { // Focus outside of circle; use intersection
			// point of line from center to focus and circle as per SVG specs.
			nf, intersects := RayCircleIntersectionF(f, c, c, 1-epsilonF)
			f = nf
			if !intersects {
				f.Set(0, 0)
			}
		}
	}
	r.rCenter, r.rFocal, r.rRadius = c, f, rs
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

	if r.rCenter == r.rFocal {
		// When the center and focal are the same things are much simpler;
		// pos is just distance from center scaled by radius
		pt := mat32.V2(float32(x)+0.5, float32(y)+0.5)
		if r.Units == ObjectBoundingBox {
			pt = r.boxTransform.MulVec2AsPt(pt)
		}
		d := pt.Sub(r.rCenter)
		pos := mat32.Sqrt(d.X*d.X/(r.rRadius.X*r.rRadius.X) + (d.Y*d.Y)/(r.rRadius.Y*r.rRadius.Y))
		return r.GetColor(pos)
	}
	if r.rFocal == mat32.V2(0, 0) {
		return color.RGBA{} // should not happen
	}

	pt := mat32.V2(float32(x)+0.5, float32(y)+0.5)
	if r.Units == ObjectBoundingBox {
		pt = r.boxTransform.MulVec2AsPt(pt)
	}
	e := pt.Div(r.rRadius)

	t1, intersects := RayCircleIntersectionF(e, r.rFocal, r.rCenter, 1)
	if !intersects { // In this case, use the last stop color
		s := r.Stops[len(r.Stops)-1]
		return s.Color
	}

	td := t1.Sub(r.rFocal)
	d := e.Sub(r.rFocal)
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
