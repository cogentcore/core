// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

// Sphere represents a 3D sphere defined by its center point and a radius
type Sphere struct {
	Center Vec3    // center of the sphere
	Radius float32 // radius of the sphere
}

// NewSphere creates and returns a pointer to a new sphere with
// the specified center and radius.
func NewSphere(center Vec3, radius float32) *Sphere {
	return &Sphere{center, radius}
}

// Set sets the center and radius of this sphere.
func (s *Sphere) Set(center Vec3, radius float32) {
	s.Center = center
	s.Radius = radius
}

// SetFromBox sets the center and radius of this sphere to surround given box
func (s *Sphere) SetFromBox(box Box3) {
	s.Center = box.Center()
	s.Radius = 0.5 * box.Size().Length()
}

// SetFromPoints sets this sphere from the specified points array and optional center.
func (s *Sphere) SetFromPoints(points []Vec3, optCenter *Vec3) {
	box := NewEmptyBox3()
	if optCenter != nil {
		s.Center = *optCenter
	} else {
		box.SetFromPoints(points)
		s.Center = box.Center()
	}
	var maxRadiusSq float32
	for i := 0; i < len(points); i++ {
		maxRadiusSq = Max(maxRadiusSq, s.Center.DistToSquared(points[i]))
	}
	s.Radius = Sqrt(maxRadiusSq)
}

// IsEmpty checks if this sphere is empty (radius <= 0)
func (s *Sphere) IsEmpty(sphere *Sphere) bool {
	return s.Radius <= 0
}

// ContainsPoint returns if this sphere contains the specified point.
func (s *Sphere) ContainsPoint(point Vec3) bool {
	if point.DistToSquared(s.Center) <= (s.Radius * s.Radius) {
		return true
	}
	return false
}

// DistToPoint returns the distance from the sphere surface to the specified point.
func (s *Sphere) DistToPoint(point Vec3) float32 {
	return point.DistTo(s.Center) - s.Radius
}

// IntersectSphere returns if other sphere intersects this one.
func (s *Sphere) IntersectSphere(other Sphere) bool {
	radiusSum := s.Radius + other.Radius
	if other.Center.DistToSquared(s.Center) <= (radiusSum * radiusSum) {
		return true
	}
	return false
}

// ClampPoint clamps the specified point inside the sphere.
// If the specified point is inside the sphere, it is the clamped point.
// Otherwise the clamped point is the the point in the sphere surface in the
// nearest of the specified point.
func (s *Sphere) ClampPoint(point Vec3) Vec3 {
	deltaLengthSq := s.Center.DistToSquared(point)
	rv := point
	if deltaLengthSq > (s.Radius * s.Radius) {
		rv = point.Sub(s.Center).Normal().MulScalar(s.Radius).Add(s.Center)
	}
	return rv
}

// GetBoundingBox calculates a Box3 which bounds this sphere.
func (s *Sphere) GetBoundingBox() Box3 {
	box := NewBox3(s.Center, s.Center)
	box.ExpandByScalar(s.Radius)
	return box
}

// MulMat4 applies the specified matrix transform to this sphere.
func (s *Sphere) MulMat4(mat *Mat4) {
	s.Center = s.Center.MulMat4(mat)
	s.Radius = s.Radius * mat.GetMaxScaleOnAxis()
}

// Translate translates this sphere by the specified offset.
func (s *Sphere) Translate(offset Vec3) {
	s.Center.SetAdd(offset)
}
