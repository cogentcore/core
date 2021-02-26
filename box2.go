// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

import "image"

// Box2 represents a 2D bounding box defined by two points:
// the point with minimum coordinates and the point with maximum coordinates.
type Box2 struct {
	Min Vec2
	Max Vec2
}

// NewBox2 creates and returns a new Box2 defined
// by its minimum and maximum coordinates.
func NewBox2(min, max Vec2) Box2 {
	return Box2{min, max}
}

// NewEmptyBox2 creates and returns a new Box2 with empty min / max
func NewEmptyBox2() Box2 {
	bx := Box2{}
	bx.SetEmpty()
	return bx
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
func (b *Box2) Set(min, max *Vec2) {
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
func (b *Box2) SetFromPoints(points []Vec2) {
	b.SetEmpty()
	for i := 0; i < len(points); i++ {
		b.ExpandByPoint(points[i])
	}
}

// SetFromRect set this bounding box from an image.Rectangle
func (b *Box2) SetFromRect(rect image.Rectangle) {
	b.Min = NewVec2FmPoint(rect.Min)
	b.Max = NewVec2FmPoint(rect.Max)
}

// ToRect returns image.Rectangle version of this bbox, using floor for min
// and Ceil for max.
func (b *Box2) ToRect() image.Rectangle {
	rect := image.Rectangle{}
	rect.Min = b.Min.ToPointFloor()
	rect.Max = b.Max.ToPointCeil()
	return rect
}

// ExpandByPoint may expand this bounding box to include the specified point.
func (b *Box2) ExpandByPoint(point Vec2) {
	b.Min.SetMin(point)
	b.Max.SetMax(point)
}

// ExpandByVector expands this bounding box by the specified vector.
func (b *Box2) ExpandByVector(vector Vec2) {
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

// SetFromCenterAndSize set this bounding box from a center point and size.
// Size is a vector from the minimum point to the maximum point.
func (b *Box2) SetFromCenterAndSize(center, size Vec2) {
	halfSize := size.MulScalar(0.5)
	b.Min = center.Sub(halfSize)
	b.Max = center.Add(halfSize)
}

// Center calculates the center point of this bounding box.
func (b Box2) Center() Vec2 {
	return b.Min.Add(b.Max).MulScalar(0.5)
}

// Size calculates the size of this bounding box: the vector from
// its minimum point to its maximum point.
func (b Box2) Size() Vec2 {
	return b.Max.Sub(b.Min)
}

// ContainsPoint returns if this bounding box contains the specified point.
func (b Box2) ContainsPoint(point Vec2) bool {
	if point.X < b.Min.X || point.X > b.Max.X ||
		point.Y < b.Min.Y || point.Y > b.Max.Y {
		return false
	}
	return true
}

// ContainsBox returns if this bounding box contains other box.
func (b Box2) ContainsBox(box Box2) bool {
	if (b.Min.X <= box.Min.X) && (box.Max.X <= b.Max.X) &&
		(b.Min.Y <= box.Min.Y) && (box.Max.Y <= b.Max.Y) {
		return true
	}
	return false
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
func (b Box2) ClampPoint(point Vec2) Vec2 {
	point.Clamp(b.Min, b.Max)
	return point
}

// DistToPoint returns the distance from this box to the specified point.
func (b Box2) DistToPoint(point Vec2) float32 {
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
func (b Box2) Translate(offset Vec2) Box2 {
	nb := Box2{}
	nb.Min = b.Min.Add(offset)
	nb.Max = b.Max.Add(offset)
	return nb
}

// IsEqual returns if this box is equal to other.
func (b Box2) IsEqual(other Box2) bool {
	return other.Min.IsEqual(b.Min) && other.Max.IsEqual(b.Max)
}
