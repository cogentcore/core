// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

// Line3 represents a 3D line segment defined by a start and an end point.
type Line3 struct {
	start Vec3
	end   Vec3
}

// NewLine3 creates and returns a pointer to a new Line3 with the
// specified start and end points.
func NewLine3(start, end *Vec3) *Line3 {

	l := new(Line3)
	l.Set(start, end)
	return l
}

// Set sets this line segment start and end points.
// Returns pointer to this updated line segment.
func (l *Line3) Set(start, end *Vec3) *Line3 {

	if start != nil {
		l.start = *start
	}
	if end != nil {
		l.end = *end
	}
	return l
}

// Copy copy other line segment to this one.
// Returns pointer to this updated line segment.
func (l *Line3) Copy(other *Line3) *Line3 {

	*l = *other
	return l
}

// Center calculates this line segment center point.
// Store its pointer into optionalTarget, if not nil, and also returns it.
func (l *Line3) Center(optionalTarget *Vec3) *Vec3 {

	var result *Vec3
	if optionalTarget == nil {
		result = NewVec3(0, 0, 0)
	} else {
		result = optionalTarget
	}
	return result.AddVectors(&l.start, &l.end).MultiplyScalar(0.5)
}

// Delta calculates the vector from the start to end point of this line segment.
// Store its pointer in optionalTarget, if not nil, and also returns it.
func (l *Line3) Delta(optionalTarget *Vec3) *Vec3 {

	var result *Vec3
	if optionalTarget == nil {
		result = NewVec3(0, 0, 0)
	} else {
		result = optionalTarget
	}
	return result.SubVectors(&l.end, &l.start)
}

// DistanceSq returns the square of the distance from the start point to the end point.
func (l *Line3) DistanceSq() float32 {

	return l.start.DistanceToSquared(&l.end)
}

// Distance returns the distance from the start point to the end point.
func (l *Line3) Distance() float32 {

	return l.start.DistanceTo(&l.end)
}

// ApplyMat4 applies the specified matrix to this line segment start and end points.
// Returns pointer to this updated line segment.
func (l *Line3) ApplyMat4(matrix *Mat4) *Line3 {

	l.start.ApplyMat4(matrix)
	l.end.ApplyMat4(matrix)
	return l
}

// Equals returns if this line segement is equal to other.
func (l *Line3) Equals(other *Line3) bool {

	return other.start.Equals(&l.start) && other.end.Equals(&l.end)
}

// Clone creates and returns a pointer to a copy of this line segment.
func (l *Line3) Clone() *Line3 {

	return NewLine3(&l.start, &l.end)
}
