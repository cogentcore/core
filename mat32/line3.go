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
	Start Vec3
	End   Vec3
}

// NewLine3 creates and returns a new Line3 with the
// specified start and end points.
func NewLine3(start, end Vec3) Line3 {
	return Line3{start, end}
}

// Set sets this line segment start and end points.
func (l *Line3) Set(start, end Vec3) {
	l.Start = start
	l.End = end
}

// Center calculates this line segment center point.
func (l *Line3) Center() Vec3 {
	return l.Start.Add(l.End).MulScalar(0.5)
}

// Delta calculates the vector from the start to end point of this line segment.
func (l *Line3) Delta() Vec3 {
	return l.End.Sub(l.Start)
}

// DistSq returns the square of the distance from the start point to the end point.
func (l *Line3) DistSq() float32 {
	return l.Start.DistToSquared(l.End)
}

// Dist returns the distance from the start point to the end point.
func (l *Line3) Dist() float32 {
	return l.Start.DistTo(l.End)
}

// MulMat4 returns specified matrix multiplied to this line segment start and end points.
func (l *Line3) MulMat4(mat *Mat4) Line3 {
	return Line3{l.Start.MulMat4(mat), l.End.MulMat4(mat)}
}

// IsEqual returns if this line segement is equal to other.
func (l *Line3) IsEqual(other Line3) bool {
	return other.Start.IsEqual(l.Start) && other.End.IsEqual(l.End)
}
