// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit Cogent Core functionality.

package math32

import "log"

// Plane represents a plane in 3D space by its normal vector and a constant offset.
// When the the normal vector is the unit vector the offset is the distance from the origin.
type Plane struct {
	Norm Vector3
	Off  float32
}

// NewPlane creates and returns a new plane from a normal vector and a offset.
func NewPlane(normal Vector3, offset float32) *Plane {
	p := &Plane{normal, offset}
	return p
}

// Set sets this plane normal vector and offset.
func (p *Plane) Set(normal Vector3, offset float32) {
	p.Norm = normal
	p.Off = offset
}

// SetDims sets this plane normal vector dimensions and offset.
func (p *Plane) SetDims(x, y, z, w float32) {
	p.Norm.Set(x, y, z)
	p.Off = w
}

// SetFromNormalAndCoplanarPoint sets this plane from a normal vector and a point on the plane.
func (p *Plane) SetFromNormalAndCoplanarPoint(normal Vector3, point Vector3) {
	p.Norm = normal
	p.Off = -point.Dot(p.Norm)
}

// SetFromCoplanarPoints sets this plane from three coplanar points.
func (p *Plane) SetFromCoplanarPoints(a, b, c Vector3) {
	norm := c.Sub(b).Cross(a.Sub(b))
	norm.SetNormal()
	if norm == (Vector3{}) {
		log.Printf("math32.SetFromCoplanarPonts: points not actually coplanar: %v %v %v\n", a, b, c)
	}
	p.SetFromNormalAndCoplanarPoint(norm, a)
}

// Normalize normalizes this plane normal vector and adjusts the offset.
// Note: will lead to a divide by zero if the plane is invalid.
func (p *Plane) Normalize() {
	invLen := 1.0 / p.Norm.Length()
	p.Norm.SetMulScalar(invLen)
	p.Off *= invLen
}

// Negate negates this plane normal.
func (p *Plane) Negate() {
	p.Off *= -1
	p.Norm = p.Norm.Negate()
}

// DistanceToPoint returns the distance of this plane from point.
func (p *Plane) DistanceToPoint(point Vector3) float32 {
	return p.Norm.Dot(point) + p.Off
}

// DistanceToSphere returns the distance of this place from the sphere.
func (p *Plane) DistanceToSphere(sphere Sphere) float32 {
	return p.DistanceToPoint(sphere.Center) - sphere.Radius
}

// IsIntersectionLine returns the line intersects this plane.
func (p *Plane) IsIntersectionLine(line Line3) bool {
	startSign := p.DistanceToPoint(line.Start)
	endSign := p.DistanceToPoint(line.End)
	return (startSign < 0 && endSign > 0) || (endSign < 0 && startSign > 0)
}

// IntersectLine calculates the point in the plane which intersets the specified line.
// Returns false if the line does not intersects the plane.
func (p *Plane) IntersectLine(line Line3) (Vector3, bool) {
	dir := line.Delta()
	denom := p.Norm.Dot(dir)
	if denom == 0 {
		// line is coplanar, return origin
		if p.DistanceToPoint(line.Start) == 0 {
			return line.Start, true
		}
		// Unsure if this is the correct method to handle this case.
		return dir, false
	}
	var t = -(line.Start.Dot(p.Norm) + p.Off) / denom
	if t < 0 || t > 1 {
		return dir, false
	}
	return dir.MulScalar(t).Add(line.Start), true
}

// CoplanarPoint returns a point in the plane that is the closest point from the origin.
func (p *Plane) CoplanarPoint() Vector3 {
	return p.Norm.MulScalar(-p.Off)
}

// SetTranslate translates this plane in the direction of its normal by offset.
func (p *Plane) SetTranslate(offset Vector3) {
	p.Off -= offset.Dot(p.Norm)
}
