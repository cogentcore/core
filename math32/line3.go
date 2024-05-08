// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit Cogent Core functionality.

package math32

// Line3 represents a 3D line segment defined by a start and an end point.
type Line3 struct {
	Start Vector3
	End   Vector3
}

// NewLine3 creates and returns a new Line3 with the
// specified start and end points.
func NewLine3(start, end Vector3) Line3 {
	return Line3{start, end}
}

// Set sets this line segment start and end points.
func (l *Line3) Set(start, end Vector3) {
	l.Start = start
	l.End = end
}

// Center calculates this line segment center point.
func (l *Line3) Center() Vector3 {
	return l.Start.Add(l.End).MulScalar(0.5)
}

// Delta calculates the vector from the start to end point of this line segment.
func (l *Line3) Delta() Vector3 {
	return l.End.Sub(l.Start)
}

// DistanceSquared returns the square of the distance from the start point to the end point.
func (l *Line3) DistanceSquared() float32 {
	return l.Start.DistanceToSquared(l.End)
}

// Dist returns the distance from the start point to the end point.
func (l *Line3) Dist() float32 {
	return l.Start.DistanceTo(l.End)
}

// MulMatrix4 returns specified matrix multiplied to this line segment start and end points.
func (l *Line3) MulMatrix4(mat *Matrix4) Line3 {
	return Line3{l.Start.MulMatrix4(mat), l.End.MulMatrix4(mat)}
}
