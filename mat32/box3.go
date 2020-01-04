// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

// Box3 represents a 3D bounding box defined by two points:
// the point with minimum coordinates and the point with maximum coordinates.
type Box3 struct {
	Min Vec3
	Max Vec3
}

// NewBox3 creates and returns a new Box3 defined
// by its minimum and maximum coordinates.
func NewBox3(min, max Vec3) Box3 {
	return Box3{min, max}
}

// NewEmptyBox3 creates and returns a new Box3 with empty min / max
func NewEmptyBox3() Box3 {
	bx := Box3{}
	bx.SetEmpty()
	return bx
}

// SetEmpty set this bounding box to empty (min / max +/- Infinity)
func (b *Box3) SetEmpty() {
	b.Min.SetScalar(Infinity)
	b.Max.SetScalar(-Infinity)
}

// IsEmpty returns true if this bounding box is empty (max < min on any coord).
func (b *Box3) IsEmpty() bool {
	return (b.Max.X < b.Min.X) || (b.Max.Y < b.Min.Y) || (b.Max.Z < b.Min.Z)
}

// Set sets this bounding box minimum and maximum coordinates.
// If either min or max are nil, then corresponding values are set to +/- Infinity.
func (b *Box3) Set(min, max *Vec3) {
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

// SetFromPoints sets this bounding box from the specified array of points.
func (b *Box3) SetFromPoints(points []Vec3) {
	b.SetEmpty()
	b.ExpandByPoints(points)
}

// ExpandByPoints may expand this bounding box from the specified array of points.
func (b *Box3) ExpandByPoints(points []Vec3) {
	for i := 0; i < len(points); i++ {
		b.ExpandByPoint(points[i])
	}
}

// ExpandByPoint may expand this bounding box to include the specified point.
func (b *Box3) ExpandByPoint(point Vec3) {
	b.Min.SetMin(point)
	b.Max.SetMax(point)
}

// ExpandByVector expands this bounding box by the specified vector
// subtracting from min and adding to max.
func (b *Box3) ExpandByVector(vector Vec3) {
	b.Min.SetSub(vector)
	b.Max.SetAdd(vector)
}

// ExpandByScalar expands this bounding box by the specified scalar
// subtracting from min and adding to max.
func (b *Box3) ExpandByScalar(scalar float32) {
	b.Min.SetSubScalar(scalar)
	b.Max.SetAddScalar(scalar)
}

// SetFromCenterAndSize sets this bounding box from a center point and size.
// Size is a vector from the minimum point to the maximum point.
func (b *Box3) SetFromCenterAndSize(center, size Vec3) {
	halfSize := size.MulScalar(0.5)
	b.Min = center.Sub(halfSize)
	b.Max = center.Add(halfSize)
}

// Center returns the center of the bounding box.
func (b *Box3) Center() Vec3 {
	return b.Min.Add(b.Max).MulScalar(0.5)
}

// Size calculates the size of this bounding box: the vector from
// its minimum point to its maximum point.
func (b *Box3) Size() Vec3 {
	return b.Max.Sub(b.Min)
}

// ContainsPoint returns if this bounding box contains the specified point.
func (b *Box3) ContainsPoint(point Vec3) bool {
	if point.X < b.Min.X || point.X > b.Max.X ||
		point.Y < b.Min.Y || point.Y > b.Max.Y ||
		point.Z < b.Min.Z || point.Z > b.Max.Z {
		return false
	}
	return true
}

// ContainsBox returns if this bounding box contains other box.
func (b *Box3) ContainsBox(box Box3) bool {
	if (b.Min.X <= box.Max.X) && (box.Max.X <= b.Max.X) &&
		(b.Min.Y <= box.Min.Y) && (box.Max.Y <= b.Max.Y) &&
		(b.Min.Z <= box.Min.Z) && (box.Max.Z <= b.Max.Z) {
		return true
	}
	return false
}

// IsIntersectionBox returns if other box intersects this one.
func (b *Box3) IsIntersectionBox(other Box3) bool {
	// using 6 splitting planes to rule out intersections.
	if other.Max.X < b.Min.X || other.Min.X > b.Max.X ||
		other.Max.Y < b.Min.Y || other.Min.Y > b.Max.Y ||
		other.Max.Z < b.Min.Z || other.Min.Z > b.Max.Z {
		return false
	}
	return true
}

// ClampPoint returns a new point which is the specified point clamped inside this box.
func (b *Box3) ClampPoint(point Vec3) Vec3 {
	point.Clamp(b.Min, b.Max)
	return point
}

// DistToPoint returns the distance from this box to the specified point.
func (b *Box3) DistToPoint(point Vec3) float32 {
	clamp := b.ClampPoint(point)
	return clamp.Sub(point).Length()
}

// GetBoundingSphere returns a bounding sphere to this bounding box.
func (b *Box3) GetBoundingSphere() Sphere {
	return Sphere{b.Center(), b.Size().Length() * 0.5}
}

// Intersect returns the intersection with other box.
func (b *Box3) Intersect(other Box3) Box3 {
	other.Min.Max(b.Min)
	other.Max.Min(b.Max)
	return other
}

// Union returns the union with other box.
func (b *Box3) Union(other Box3) Box3 {
	other.Min.Min(b.Min)
	other.Max.Max(b.Max)
	return other
}

// MulMat4 multiplies the specified matrix to the vertices of this bounding box
// and computes the resulting spanning Box3 of the transformed points
func (b *Box3) MulMat4(m *Mat4) Box3 {
	xax := m[0] * b.Min.X
	xay := m[1] * b.Min.X
	xaz := m[2] * b.Min.X
	xbx := m[0] * b.Max.X
	xby := m[1] * b.Max.X
	xbz := m[2] * b.Max.X
	yax := m[4] * b.Min.Y
	yay := m[5] * b.Min.Y
	yaz := m[6] * b.Min.Y
	ybx := m[4] * b.Max.Y
	yby := m[5] * b.Max.Y
	ybz := m[6] * b.Max.Y
	zax := m[8] * b.Min.Z
	zay := m[9] * b.Min.Z
	zaz := m[10] * b.Min.Z
	zbx := m[8] * b.Max.Z
	zby := m[9] * b.Max.Z
	zbz := m[10] * b.Max.Z

	nb := Box3{}
	nb.Min.X = Min(xax, xbx) + Min(yax, ybx) + Min(zax, zbx) + m[12]
	nb.Min.Y = Min(xay, xby) + Min(yay, yby) + Min(zay, zby) + m[13]
	nb.Min.Z = Min(xaz, xbz) + Min(yaz, ybz) + Min(zaz, zbz) + m[14]
	nb.Max.X = Max(xax, xbx) + Max(yax, ybx) + Max(zax, zbx) + m[12]
	nb.Max.Y = Max(xay, xby) + Max(yay, yby) + Max(zay, zby) + m[13]
	nb.Max.Z = Max(xaz, xbz) + Max(yaz, ybz) + Max(zaz, zbz) + m[14]
	return nb
}

// MulQuat multiplies the specified quaternion to the vertices of this bounding box
// and computes the resulting spanning Box3 of the transformed points
func (b *Box3) MulQuat(q Quat) Box3 {
	var cs [8]Vec3
	cs[0] = Vec3{b.Min.X, b.Min.Y, b.Min.Z}.MulQuat(q)
	cs[1] = Vec3{b.Min.X, b.Min.Y, b.Max.Z}.MulQuat(q)
	cs[2] = Vec3{b.Min.X, b.Max.Y, b.Min.Z}.MulQuat(q)
	cs[3] = Vec3{b.Max.X, b.Min.Y, b.Min.Z}.MulQuat(q)

	cs[4] = Vec3{b.Max.X, b.Max.Y, b.Max.Z}.MulQuat(q)
	cs[5] = Vec3{b.Max.X, b.Max.Y, b.Min.Z}.MulQuat(q)
	cs[6] = Vec3{b.Max.X, b.Min.Y, b.Max.Z}.MulQuat(q)
	cs[7] = Vec3{b.Min.X, b.Max.Y, b.Max.Z}.MulQuat(q)

	nb := NewEmptyBox3()
	for i := 0; i < 8; i++ {
		nb.ExpandByPoint(cs[i])
	}
	return nb
}

// Translate returns translated position of this box by offset.
func (b *Box3) Translate(offset Vec3) Box3 {
	nb := Box3{}
	nb.Min = b.Min.Add(offset)
	nb.Max = b.Max.Add(offset)
	return nb
}

// IsEqual returns if this box is equal to other
func (b *Box3) IsEqual(other Box3) bool {
	return other.Min.IsEqual(b.Min) && other.Max.IsEqual(b.Max)
}

// MVProjToNDC projects bounding box through given MVP model-view-projection Mat4
// with perspective divide to return normalized display coordinates (NDC).
func (b *Box3) MVProjToNDC(m *Mat4) Box3 {
	// all corners: i = min, a = max
	var cs [8]Vec3
	cs[0] = Vec4{b.Min.X, b.Min.Y, b.Min.Z, 1}.MulMat4(m).PerspDiv()
	cs[1] = Vec4{b.Min.X, b.Min.Y, b.Max.Z, 1}.MulMat4(m).PerspDiv()
	cs[2] = Vec4{b.Min.X, b.Max.Y, b.Min.Z, 1}.MulMat4(m).PerspDiv()
	cs[3] = Vec4{b.Max.X, b.Min.Y, b.Min.Z, 1}.MulMat4(m).PerspDiv()

	cs[4] = Vec4{b.Max.X, b.Max.Y, b.Max.Z, 1}.MulMat4(m).PerspDiv()
	cs[5] = Vec4{b.Max.X, b.Max.Y, b.Min.Z, 1}.MulMat4(m).PerspDiv()
	cs[6] = Vec4{b.Max.X, b.Min.Y, b.Max.Z, 1}.MulMat4(m).PerspDiv()
	cs[7] = Vec4{b.Min.X, b.Max.Y, b.Max.Z, 1}.MulMat4(m).PerspDiv()

	nb := NewEmptyBox3()
	for i := 0; i < 8; i++ {
		nb.ExpandByPoint(cs[i])
	}
	return nb
}
