// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

// Frustum represents a frustum
type Frustum struct {
	Planes [6]Plane
}

// NewFrustumFromMatrix creates and returns a Frustum based on the provided matrix
func NewFrustumFromMatrix(m *Mat4) *Frustum {
	f := new(Frustum)
	f.SetFromMatrix(m)
	return f
}

// NewFrustum returns a pointer to a new Frustum object made of 6 explicit planes
func NewFrustum(p0, p1, p2, p3, p4, p5 *Plane) *Frustum {
	f := new(Frustum)
	f.Set(p0, p1, p2, p3, p4, p5)
	return f
}

// Set sets the frustum's planes
func (f *Frustum) Set(p0, p1, p2, p3, p4, p5 *Plane) {
	if p0 != nil {
		f.Planes[0] = *p0
	}
	if p1 != nil {
		f.Planes[1] = *p1
	}
	if p2 != nil {
		f.Planes[2] = *p2
	}
	if p3 != nil {
		f.Planes[3] = *p3
	}
	if p4 != nil {
		f.Planes[4] = *p4
	}
	if p5 != nil {
		f.Planes[5] = *p5
	}
}

// SetFromMatrix sets the frustum's planes based on the specified Mat4
func (f *Frustum) SetFromMatrix(m *Mat4) {
	me0 := m[0]
	me1 := m[1]
	me2 := m[2]
	me3 := m[3]
	me4 := m[4]
	me5 := m[5]
	me6 := m[6]
	me7 := m[7]
	me8 := m[8]
	me9 := m[9]
	me10 := m[10]
	me11 := m[11]
	me12 := m[12]
	me13 := m[13]
	me14 := m[14]
	me15 := m[15]

	f.Planes[0].SetComponents(me3-me0, me7-me4, me11-me8, me15-me12)
	f.Planes[1].SetComponents(me3+me0, me7+me4, me11+me8, me15+me12)
	f.Planes[2].SetComponents(me3+me1, me7+me5, me11+me9, me15+me13)
	f.Planes[3].SetComponents(me3-me1, me7-me5, me11-me9, me15-me13)
	f.Planes[4].SetComponents(me3-me2, me7-me6, me11-me10, me15-me14)
	f.Planes[5].SetComponents(me3+me2, me7+me6, me11+me10, me15+me14)

	for i := 0; i < 6; i++ {
		f.Planes[i].Normalize()
	}
}

// IntersectsSphere determines whether the specified sphere is intersecting the frustum
func (f *Frustum) IntersectsSphere(sphere Sphere) bool {
	negRadius := -sphere.Radius
	for i := 0; i < 6; i++ {
		dist := f.Planes[i].DistToPoint(sphere.Center)
		if dist < negRadius {
			return false
		}
	}
	return true
}

// IntersectsBox determines whether the specified box is intersecting the frustum
func (f *Frustum) IntersectsBox(box Box3) bool {
	var p1 Vec3
	var p2 Vec3

	for i := 0; i < 6; i++ {
		plane := &f.Planes[i]
		if plane.Norm.X > 0 {
			p1.X = box.Min.X
		} else {
			p1.X = box.Max.X
		}
		if plane.Norm.X > 0 {
			p2.X = box.Max.X
		} else {
			p2.X = box.Min.X
		}
		if plane.Norm.Y > 0 {
			p1.Y = box.Min.Y
		} else {
			p1.Y = box.Max.Y
		}
		if plane.Norm.Y > 0 {
			p2.Y = box.Max.Y
		} else {
			p2.Y = box.Min.Y
		}
		if plane.Norm.Z > 0 {
			p1.Z = box.Min.Z
		} else {
			p1.Z = box.Max.Z
		}
		if plane.Norm.Z > 0 {
			p2.Z = box.Max.Z
		} else {
			p2.Z = box.Min.Z
		}

		d1 := plane.DistToPoint(p1)
		d2 := plane.DistToPoint(p2)

		// if both outside plane, no intersection
		if d1 < 0 && d2 < 0 {
			return false
		}
	}
	return true
}

// ContainsPoint determines whether the frustum contains the specified point
func (f *Frustum) ContainsPoint(point Vec3) bool {
	for i := 0; i < 6; i++ {
		if f.Planes[i].DistToPoint(point) < 0 {
			return false
		}
	}
	return true
}
