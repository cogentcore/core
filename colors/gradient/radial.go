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

// Radial represents a radial gradient. It implements the [image.Image] interface.
type Radial struct { //types:add -setters
	Base

	// the center point of the gradient (cx and cy in SVG)
	Center math32.Vector2

	// the focal point of the gradient (fx and fy in SVG)
	Focal math32.Vector2

	// the radius of the gradient (rx and ry in SVG)
	Radius math32.Vector2

	// computed current render versions transformed by object matrix
	rCenter math32.Vector2
	rFocal  math32.Vector2
	rRadius math32.Vector2

	rotTrans math32.Matrix2
}

var _ Gradient = &Radial{}

// NewRadial returns a new centered [Radial] gradient.
func NewRadial() *Radial {
	return &Radial{
		Base: NewBase(),
		// default is fully centered
		Center: math32.Vector2Scalar(0.5),
		Focal:  math32.Vector2Scalar(0.5),
		Radius: math32.Vector2Scalar(0.5),
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
func (r *Radial) Update(opacity float32, box math32.Box2, objTransform math32.Matrix2) {
	r.Box = box
	r.Opacity = opacity
	r.updateBase()
	r.rotTrans = math32.Identity2()

	c, f, rs := r.Center, r.Focal, r.Radius
	sz := r.Box.Size()
	if r.Units == ObjectBoundingBox {
		c = r.Box.Min.Add(sz.Mul(c))
		f = r.Box.Min.Add(sz.Mul(f))
		rs.SetMul(sz)
	} else {
		ct := objTransform.Mul(r.Transform)
		c = ct.MulVector2AsPoint(c)
		f = ct.MulVector2AsPoint(f)
		_, _, phi, sx, sy, _ := ct.Decompose()
		r.rotTrans = math32.Rotate2D(phi)
		rs.SetMul(math32.Vec2(sx, sy))
	}
	if c != f {
		f.SetDiv(rs)
		c.SetDiv(rs)
		df := f.Sub(c)
		if df.X*df.X+df.Y*df.Y > 1 { // Focus outside of circle; use intersection
			// point of line from center to focus and circle as per SVG specs.
			nf, intersects := rayCircleIntersectionF(f, c, c, 1-epsilonF)
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
		pt := math32.Vec2(float32(x)+0.5, float32(y)+0.5)
		if r.Units == ObjectBoundingBox {
			pt = r.boxTransform.MulVector2AsPoint(pt)
		}
		d := r.rotTrans.MulVector2AsVector(pt.Sub(r.rCenter))
		pos := math32.Sqrt(d.X*d.X/(r.rRadius.X*r.rRadius.X) + (d.Y*d.Y)/(r.rRadius.Y*r.rRadius.Y))
		return r.getColor(pos)
	}
	if r.rFocal == math32.Vec2(0, 0) {
		return color.RGBA{} // should not happen
	}

	pt := math32.Vec2(float32(x)+0.5, float32(y)+0.5)
	if r.Units == ObjectBoundingBox {
		pt = r.boxTransform.MulVector2AsPoint(pt)
	}
	pt = r.rotTrans.MulVector2AsVector(pt)
	e := pt.Div(r.rRadius)

	t1, intersects := rayCircleIntersectionF(e, r.rFocal, r.rCenter, 1)
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

	pos := math32.Sqrt(d.X*d.X+d.Y*d.Y) / math32.Sqrt(td.X*td.X+td.Y*td.Y)
	return r.getColor(pos)
}

// rayCircleIntersectionF calculates in floating point the points of intersection of
// a ray starting at s2 passing through s1 and a circle in fixed point.
// Returns intersects == false if no solution is possible. If two
// solutions are possible, the point closest to s2 is returned.
func rayCircleIntersectionF(s1, s2, c math32.Vector2, r float32) (pt math32.Vector2, intersects bool) {
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

	D = math32.Sqrt(D)
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
	return math32.Vec2((n-e*t1)+c.X, (m-d*t1)+c.Y), true
}
