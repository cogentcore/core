// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

import "log"

// Plane represents a plane in 3D space by its normal vector and a constant offset.
// When the the normal vector is the unit vector the offset is the distance from the origin.
type Plane struct {
	Norm Vec3
	Off  float32
}

// NewPlane creates and returns a new plane from a normal vector and a offset.
func NewPlane(normal Vec3, offset float32) *Plane {
	p := &Plane{normal, offset}
	return p
}

// Set sets this plane normal vector and offset.
func (p *Plane) Set(normal Vec3, offset float32) {
	p.Norm = normal
	p.Off = offset
}

// SetDims sets this plane normal vector dimensions and offset.
func (p *Plane) SetDims(x, y, z, w float32) {
	p.Norm.Set(x, y, z)
	p.Off = w
}

// SetFromNormalAndCoplanarPoint sets this plane from a normal vector and a point on the plane.
func (p *Plane) SetFromNormalAndCoplanarPoint(normal Vec3, point Vec3) {
	p.Norm = normal
	p.Off = -point.Dot(p.Norm)
}

// SetFromCoplanarPoints sets this plane from three coplanar points.
func (p *Plane) SetFromCoplanarPoints(a, b, c Vec3) {
	norm := c.Sub(b).Cross(a.Sub(b))
	norm.SetNormal()
	if norm.IsNil() {
		log.Printf("mat32.SetFromCoplanarPonts: points not actually coplanar: %v %v %v\n", a, b, c)
	}
	p.SetFromNormalAndCoplanarPoint(norm, a)
}

// Normalize normalizes this plane normal vector and adjusts the offset.
// Note: will lead to a divide by zero if the plane is invalid.
func (p *Plane) Normalize() {
	invLen := 1.0 / p.Norm.Length()
	p.Norm.MulScalar(invLen)
	p.Off *= invLen
}

// Negate negates this plane normal.
func (p *Plane) Negate() {
	p.Off *= -1
	p.Norm.SetNegate()
}

// DistToPoint returns the distance of this plane from point.
func (p *Plane) DistToPoint(point Vec3) float32 {
	return p.Norm.Dot(point) + p.Off
}

// DistToSphere returns the distance of this place from the sphere.
func (p *Plane) DistToSphere(sphere Sphere) float32 {
	return p.DistToPoint(sphere.Center) - sphere.Radius
}

// IsIntersectionLine returns the line intersects this plane.
func (p *Plane) IsIntersectionLine(line Line3) bool {
	startSign := p.DistToPoint(line.Start)
	endSign := p.DistToPoint(line.End)
	return (startSign < 0 && endSign > 0) || (endSign < 0 && startSign > 0)
}

// IntersectLine calculates the point in the plane which intersets the specified line.
// Returns false if the line does not intersects the plane.
func (p *Plane) IntersectLine(line Line3) (Vec3, bool) {
	dir := line.Delta()
	denom := p.Norm.Dot(dir)
	if denom == 0 {
		// line is coplanar, return origin
		if p.DistToPoint(line.Start) == 0 {
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
func (p *Plane) CoplanarPoint() Vec3 {
	return p.Norm.MulScalar(-p.Off)
}

// SetTranslate translates this plane in the direction of its normal by offset.
func (p *Plane) SetTranslate(offset Vec3) {
	p.Off -= offset.Dot(p.Norm)
}

// IsEqual returns if this plane is equal to other
func (p *Plane) IsEqual(other *Plane) bool {
	return other.Norm.IsEqual(p.Norm) && (other.Off == p.Off)
}
