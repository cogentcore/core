// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

// Plane represents a plane in 3D space by its normal vector and a constant.
// When the the normal vector is the unit vector the constant is the distance from the origin.
type Plane struct {
	normal   Vec3
	constant float32
}

// NewPlane creates and returns a new plane from a normal vector and a constant.
func NewPlane(normal *Vec3, constant float32) *Plane {

	p := new(Plane)
	if normal != nil {
		p.normal = *normal
	}
	p.constant = constant
	return p
}

// Set sets this plane normal vector and constant.
// Returns pointer to this updated plane.
func (p *Plane) Set(normal *Vec3, constant float32) *Plane {

	p.normal = *normal
	p.constant = constant
	return p
}

// SetComponents sets this plane normal vector components and constant.
// Returns pointer to this updated plane.
func (p *Plane) SetComponents(x, y, z, w float32) *Plane {

	p.normal.Set(x, y, z)
	p.constant = w
	return p
}

// SetFromNormalAndCoplanarPoint sets this plane from a normal vector and a point on the plane.
// Returns pointer to this updated plane.
func (p *Plane) SetFromNormalAndCoplanarPoint(normal *Vec3, point *Vec3) *Plane {

	p.normal = *normal
	p.constant = -point.Dot(&p.normal)
	return p
}

// SetFromCoplanarPoints sets this plane from three coplanar points.
// Returns pointer to this updated plane.
func (p *Plane) SetFromCoplanarPoints(a, b, c *Vec3) *Plane {

	var v1 Vec3
	var v2 Vec3

	normal := v1.SubVectors(c, b).Cross(v2.SubVectors(a, b)).Normalize()
	// Q: should an error be thrown if normal is zero (e.g. degenerate plane)?
	p.SetFromNormalAndCoplanarPoint(normal, a)
	return p
}

// Copy sets this plane to a copy of other.
// Returns pointer to this updated plane.
func (p *Plane) Copy(other *Plane) *Plane {

	p.normal.Copy(&other.normal)
	p.constant = other.constant
	return p
}

// Normalize normalizes this plane normal vector and adjusts the constant.
// Note: will lead to a divide by zero if the plane is invalid.
// Returns pointer to this updated plane.
func (p *Plane) Normalize() *Plane {

	inverseNormalLength := 1.0 / p.normal.Length()
	p.normal.MultiplyScalar(inverseNormalLength)
	p.constant *= inverseNormalLength
	return p
}

// Negate negates this plane normal.
// Returns pointer to this updated plane.
func (p *Plane) Negate() *Plane {

	p.constant *= -1
	p.normal.Negate()
	return p
}

// DistanceToPoint returns the distance of this plane from point.
func (p *Plane) DistanceToPoint(point *Vec3) float32 {

	return p.normal.Dot(point) + p.constant
}

// DistanceToSphere returns the distance of this place from the sphere.
func (p *Plane) DistanceToSphere(sphere *Sphere) float32 {

	return p.DistanceToPoint(&sphere.Center) - sphere.Radius
}

// IsIntersectionLine returns the line intersects this plane.
func (p *Plane) IsIntersectionLine(line *Line3) bool {

	startSign := p.DistanceToPoint(&line.start)
	endSign := p.DistanceToPoint(&line.end)
	return (startSign < 0 && endSign > 0) || (endSign < 0 && startSign > 0)
}

// IntersectLine calculates the point in the plane which intersets the specified line.
// Sets the optionalTarget, if not nil to this point, and also returns it.
// Returns nil if the line does not intersects the plane.
func (p *Plane) IntersectLine(line *Line3, optionalTarget *Vec3) *Vec3 {

	var v1 Vec3
	var result *Vec3
	if optionalTarget == nil {
		result = NewVec3(0, 0, 0)
	} else {
		result = optionalTarget
	}

	direction := line.Delta(&v1)
	denominator := p.normal.Dot(direction)
	if denominator == 0 {
		// line is coplanar, return origin
		if p.DistanceToPoint(&line.start) == 0 {
			return result.Copy(&line.start)
		}
		// Unsure if this is the correct method to handle this case.
		return nil
	}

	var t = -(line.start.Dot(&p.normal) + p.constant) / denominator
	if t < 0 || t > 1 {
		return nil
	}
	return result.Copy(direction).MultiplyScalar(t).Add(&line.start)
}

// CoplanarPoint sets the optionalTarget to a point in the plane and also returns it.
// The point set and returned is the closest point from the origin.
func (p *Plane) CoplanarPoint(optionalTarget *Vec3) *Vec3 {

	var result *Vec3
	if optionalTarget == nil {
		result = NewVec3(0, 0, 0)
	} else {
		result = optionalTarget
	}
	return result.Copy(&p.normal).MultiplyScalar(-p.constant)
}

// Translate translates this plane in the direction of its normal by offset.
// Returns pointer to this updated plane.
func (p *Plane) Translate(offset *Vec3) *Plane {

	p.constant = p.constant - offset.Dot(&p.normal)
	return p
}

// Equals returns if this plane is equal to other
func (p *Plane) Equals(other *Plane) bool {

	return other.normal.Equals(&p.normal) && (other.constant == p.constant)
}

// Clone creates and returns a pointer to a copy of this plane.
func (p *Plane) Clone(plane *Plane) *Plane {

	return NewPlane(&plane.normal, plane.constant)
}
