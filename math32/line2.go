// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

// Line2 represents a 2D line segment defined by a start and an end point.
type Line2 struct {
	Start Vector2
	End   Vector2
}

// NewLine2 creates and returns a new Line2 with the
// specified start and end points.
func NewLine2(start, end Vector2) Line2 {
	return Line2{start, end}
}

// Set sets this line segment start and end points.
func (l *Line2) Set(start, end Vector2) {
	l.Start = start
	l.End = end
}

// Center calculates this line segment center point.
func (l *Line2) Center() Vector2 {
	return l.Start.Add(l.End).MulScalar(0.5)
}

// Delta calculates the vector from the start to end point of this line segment.
func (l *Line2) Delta() Vector2 {
	return l.End.Sub(l.Start)
}

// LengthSquared returns the square of the distance from the start point to the end point.
func (l *Line2) LengthSquared() float32 {
	return l.Start.DistanceToSquared(l.End)
}

// Length returns the length from the start point to the end point.
func (l *Line2) Length() float32 {
	return l.Start.DistanceTo(l.End)
}

// note: ClosestPointToPoint is adapted from https://math.stackexchange.com/questions/2193720/find-a-point-on-a-line-segment-which-is-the-closest-to-other-point-not-on-the-li

// ClosestPointToPoint returns the point along the line that is
// closest to the given point.
func (l *Line2) ClosestPointToPoint(point Vector2) Vector2 {
	v := l.Delta()
	u := point.Sub(l.Start)
	vu := v.Dot(u)
	ds := v.LengthSquared()
	t := vu / ds
	switch {
	case t <= 0:
		return l.Start
	case t >= 1:
		return l.End
	default:
		return l.Start.Add(v.MulScalar(t))
	}
}
