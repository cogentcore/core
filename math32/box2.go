// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit Cogent Core functionality.

package math32

import (
	"image"

	"golang.org/x/image/math/fixed"
)

// Box2 represents a 2D bounding box defined by two points:
// the point with minimum coordinates and the point with maximum coordinates.
type Box2 struct {
	Min Vector2
	Max Vector2
}

// B2 returns a new [Box2] from the given minimum and maximum x and y coordinates.
func B2(x0, y0, x1, y1 float32) Box2 {
	return Box2{Vec2(x0, y0), Vec2(x1, y1)}
}

// B2Empty returns a new [Box2] with empty minimum and maximum values
func B2Empty() Box2 {
	bx := Box2{}
	bx.SetEmpty()
	return bx
}

// B2FromRect returns a new [Box2] from the given [image.Rectangle].
func B2FromRect(rect image.Rectangle) Box2 {
	b := Box2{}
	b.SetFromRect(rect)
	return b
}

// B2FromFixed returns a new [Box2] from the given [fixed.Rectangle26_6].
func B2FromFixed(rect fixed.Rectangle26_6) Box2 {
	b := Box2{}
	b.Min.SetFixed(rect.Min)
	b.Max.SetFixed(rect.Max)
	return b
}

// SetEmpty set this bounding box to empty (min / max +/- Infinity)
func (b *Box2) SetEmpty() {
	b.Min.SetScalar(Infinity)
	b.Max.SetScalar(-Infinity)
}

// IsEmpty returns if this bounding box is empty (max < min on any coord).
func (b *Box2) IsEmpty() bool {
	return (b.Max.X < b.Min.X) || (b.Max.Y < b.Min.Y)
}

// Set sets this bounding box minimum and maximum coordinates.
// If either min or max are nil, then corresponding values are set to +/- Infinity.
func (b *Box2) Set(min, max *Vector2) {
	if min != nil {
		b.Min = *min
	} else {
		b.Min.SetScalar(Infinity)
	}
	if max != nil {
		b.Max = *max
	} else {
		b.Max.SetScalar(-Infinity)
	}
}

// SetFromPoints set this bounding box from the specified array of points.
func (b *Box2) SetFromPoints(points []Vector2) {
	b.SetEmpty()
	for i := 0; i < len(points); i++ {
		b.ExpandByPoint(points[i])
	}
}

// SetFromRect set this bounding box from an image.Rectangle
func (b *Box2) SetFromRect(rect image.Rectangle) {
	b.Min = FromPoint(rect.Min)
	b.Max = FromPoint(rect.Max)
}

// ToRect returns image.Rectangle version of this bbox, using floor for min
// and Ceil for max.
func (b Box2) ToRect() image.Rectangle {
	rect := image.Rectangle{}
	rect.Min = b.Min.ToPointFloor()
	rect.Max = b.Max.ToPointCeil()
	return rect
}

// ToFixed returns fixed.Rectangle26_6 version of this bbox.
func (b Box2) ToFixed() fixed.Rectangle26_6 {
	rect := fixed.Rectangle26_6{Min: b.Min.ToFixed(), Max: b.Max.ToFixed()}
	return rect
}

// RectInNotEmpty returns true if rect r is contained within b box
// and r is not empty.
// The existing image.Rectangle.In method returns true if r is empty,
// but we typically expect that case to be false (out of range box)
func RectInNotEmpty(r, b image.Rectangle) bool {
	if r.Empty() {
		return false
	}
	return r.In(b)
}

// Canon returns the canonical version of the box.
// The returned rectangle has minimum and maximum coordinates swapped
// if necessary so that it is well-formed.
func (b Box2) Canon() Box2 {
	if b.Max.X < b.Min.X {
		b.Min.X, b.Max.X = b.Max.X, b.Min.X
	}
	if b.Max.Y < b.Min.Y {
		b.Min.Y, b.Max.Y = b.Max.Y, b.Min.Y
	}
	return b
}

// ExpandByPoint may expand this bounding box to include the specified point.
func (b *Box2) ExpandByPoint(point Vector2) {
	b.Min.SetMin(point)
	b.Max.SetMax(point)
}

// ExpandByVector expands this bounding box by the specified vector.
func (b *Box2) ExpandByVector(vector Vector2) {
	b.Min.SetSub(vector)
	b.Max.SetAdd(vector)
}

// ExpandByScalar expands this bounding box by the specified scalar.
func (b *Box2) ExpandByScalar(scalar float32) {
	b.Min.SetSubScalar(scalar)
	b.Max.SetAddScalar(scalar)
}

// ExpandByBox may expand this bounding box to include the specified box
func (b *Box2) ExpandByBox(box Box2) {
	b.ExpandByPoint(box.Min)
	b.ExpandByPoint(box.Max)
}

// MulMatrix2 multiplies the specified matrix to the vertices of this bounding box
// and computes the resulting spanning Box2 of the transformed points
func (b Box2) MulMatrix2(m Matrix2) Box2 {
	var cs [4]Vector2
	cs[0] = m.MulVector2AsPoint(Vec2(b.Min.X, b.Min.Y))
	cs[1] = m.MulVector2AsPoint(Vec2(b.Min.X, b.Max.Y))
	cs[2] = m.MulVector2AsPoint(Vec2(b.Max.X, b.Min.Y))
	cs[3] = m.MulVector2AsPoint(Vec2(b.Max.X, b.Max.Y))

	nb := B2Empty()
	for i := 0; i < 4; i++ {
		nb.ExpandByPoint(cs[i])
	}
	return nb
}

// SetFromCenterAndSize set this bounding box from a center point and size.
// Size is a vector from the minimum point to the maximum point.
func (b *Box2) SetFromCenterAndSize(center, size Vector2) {
	halfSize := size.MulScalar(0.5)
	b.Min = center.Sub(halfSize)
	b.Max = center.Add(halfSize)
}

// Center calculates the center point of this bounding box.
func (b Box2) Center() Vector2 {
	return b.Min.Add(b.Max).MulScalar(0.5)
}

// Size calculates the size of this bounding box: the vector from
// its minimum point to its maximum point.
func (b Box2) Size() Vector2 {
	return b.Max.Sub(b.Min)
}

// ContainsPoint returns if this bounding box contains the specified point.
func (b Box2) ContainsPoint(point Vector2) bool {
	if point.X < b.Min.X || point.X > b.Max.X ||
		point.Y < b.Min.Y || point.Y > b.Max.Y {
		return false
	}
	return true
}

// ContainsBox returns if this bounding box contains other box.
func (b Box2) ContainsBox(box Box2) bool {
	return (b.Min.X <= box.Min.X) && (box.Max.X <= b.Max.X) && (b.Min.Y <= box.Min.Y) && (box.Max.Y <= b.Max.Y)
}

// IntersectsBox returns if other box intersects this one.
func (b Box2) IntersectsBox(other Box2) bool {
	if other.Max.X < b.Min.X || other.Min.X > b.Max.X ||
		other.Max.Y < b.Min.Y || other.Min.Y > b.Max.Y {
		return false
	}
	return true
}

// ClampPoint calculates a new point which is the specified point clamped inside this box.
func (b Box2) ClampPoint(point Vector2) Vector2 {
	point.Clamp(b.Min, b.Max)
	return point
}

// DistanceToPoint returns the distance from this box to the specified point.
func (b Box2) DistanceToPoint(point Vector2) float32 {
	clamp := b.ClampPoint(point)
	return clamp.Sub(point).Length()
}

// Intersect returns the intersection with other box.
func (b Box2) Intersect(other Box2) Box2 {
	other.Min.SetMax(b.Min)
	other.Max.SetMin(b.Max)
	return other
}

// Union returns the union with other box.
func (b Box2) Union(other Box2) Box2 {
	other.Min.SetMin(b.Min)
	other.Max.SetMax(b.Max)
	return other
}

// Translate returns translated position of this box by offset.
func (b Box2) Translate(offset Vector2) Box2 {
	nb := Box2{}
	nb.Min = b.Min.Add(offset)
	nb.Max = b.Max.Add(offset)
	return nb
}

// ProjectX projects normalized value along the X dimension of this box
func (b Box2) ProjectX(v float32) float32 {
	return b.Min.X + v*(b.Max.X-b.Min.X)
}

// ProjectY projects normalized value along the Y dimension of this box
func (b Box2) ProjectY(v float32) float32 {
	return b.Min.Y + v*(b.Max.Y-b.Min.Y)
}
