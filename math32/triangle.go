// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit Cogent Core functionality.

package math32

// Triangle represents a triangle made of three vertices.
type Triangle struct {
	A Vector3
	B Vector3
	C Vector3
}

// NewTriangle returns a new Triangle object.
func NewTriangle(a, b, c Vector3) Triangle {
	return Triangle{a, b, c}
}

// Normal returns the triangle's normal.
func Normal(a, b, c Vector3) Vector3 {
	nv := c.Sub(b).Cross(a.Sub(b))
	lenSq := nv.LengthSquared()
	if lenSq > 0 {
		return nv.MulScalar(1 / Sqrt(lenSq))
	}
	return Vector3{}
}

// BarycoordFromPoint returns the barycentric coordinates for the specified point.
func BarycoordFromPoint(point, a, b, c Vector3) Vector3 {
	v0 := c.Sub(a)
	v1 := b.Sub(a)
	v2 := point.Sub(a)

	dot00 := v0.Dot(v0)
	dot01 := v0.Dot(v1)
	dot02 := v0.Dot(v2)
	dot11 := v1.Dot(v1)
	dot12 := v1.Dot(v2)

	denom := dot00*dot11 - dot01*dot01

	// colinear or singular triangle
	if denom == 0 {
		// arbitrary location outside of triangle?
		// not sure if this is the best idea, maybe should be returning undefined
		return Vec3(-2, -1, -1)
	}

	invDenom := 1 / denom
	u := (dot11*dot02 - dot01*dot12) * invDenom
	v := (dot00*dot12 - dot01*dot02) * invDenom

	// barycoordinates must always sum to 1
	return Vec3(1-u-v, v, u)
}

// ContainsPoint returns whether a triangle contains a point.
func ContainsPoint(point, a, b, c Vector3) bool {
	rv := BarycoordFromPoint(point, a, b, c)
	return (rv.X >= 0) && (rv.Y >= 0) && ((rv.X + rv.Y) <= 1)
}

// Set sets the triangle's three vertices.
func (t *Triangle) Set(a, b, c Vector3) {
	t.A = a
	t.B = b
	t.C = c
}

// SetFromPointsAndIndices sets the triangle's vertices based on the specified points and indices.
func (t *Triangle) SetFromPointsAndIndices(points []Vector3, i0, i1, i2 int) {
	t.A = points[i0]
	t.B = points[i1]
	t.C = points[i2]
}

// Area returns the triangle's area.
func (t *Triangle) Area() float32 {
	v0 := t.C.Sub(t.B)
	v1 := t.A.Sub(t.B)
	return v0.Cross(v1).Length() * 0.5
}

// Midpoint returns the triangle's midpoint.
func (t *Triangle) Midpoint() Vector3 {
	return t.A.Add(t.B).Add(t.C).MulScalar(float32(1) / 3)
}

// Normal returns the triangle's normal.
func (t *Triangle) Normal() Vector3 {
	return Normal(t.A, t.B, t.C)
}

// Plane returns a Plane object aligned with the triangle.
func (t *Triangle) Plane() Plane {
	pv := Plane{}
	pv.SetFromCoplanarPoints(t.A, t.B, t.C)
	return pv
}

// BarycoordFromPoint returns the barycentric coordinates for the specified point.
func (t *Triangle) BarycoordFromPoint(point Vector3) Vector3 {
	return BarycoordFromPoint(point, t.A, t.B, t.C)
}

// ContainsPoint returns whether the triangle contains a point.
func (t *Triangle) ContainsPoint(point Vector3) bool {
	return ContainsPoint(point, t.A, t.B, t.C)
}
