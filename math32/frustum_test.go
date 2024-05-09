// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrustumSet(t *testing.T) {
	p0 := &Plane{Norm: Vector3{1, 0, 0}, Off: 1}
	p1 := &Plane{Norm: Vector3{-1, 0, 0}, Off: 2}
	p2 := &Plane{Norm: Vector3{0, 1, 0}, Off: 3}
	p3 := &Plane{Norm: Vector3{0, -1, 0}, Off: 4}
	p4 := &Plane{Norm: Vector3{0, 0, 1}, Off: 0}
	p5 := &Plane{Norm: Vector3{0, 0, -1}, Off: -3}

	f := &Frustum{}

	f.Set(p0, p1, p2, p3, p4, p5)

	assert.Equal(t, *p0, f.Planes[0])
	assert.Equal(t, *p1, f.Planes[1])
	assert.Equal(t, *p2, f.Planes[2])
	assert.Equal(t, *p3, f.Planes[3])
	assert.Equal(t, *p4, f.Planes[4])
	assert.Equal(t, *p5, f.Planes[5])
}

func TestFrustumSetFromMatrix(t *testing.T) {
	m := &Matrix4{
		0, 1, 2, 3,
		4, 5, 6, 7,
		8, 9, 10, 11,
		12, 13, 14, 15,
	}

	f := &Frustum{}
	f.SetFromMatrix(m)

	assert.Equal(t, Plane{Norm: Vector3{0.57735026, 0.57735026, 0.57735026}, Off: 0.57735026}, f.Planes[0])
	assert.Equal(t, Plane{Norm: Vector3{0.1353881, 0.49642307, 0.85745806}, Off: 1.218493}, f.Planes[1])
	assert.Equal(t, Plane{Norm: Vector3{0.16903085, 0.50709254, 0.8451542}, Off: 1.1832159}, f.Planes[2])
	assert.Equal(t, Plane{Norm: Vector3{0.57735026, 0.57735026, 0.57735026}, Off: 0.57735026}, f.Planes[3])
	assert.Equal(t, Plane{Norm: Vector3{0.57735026, 0.57735026, 0.57735026}, Off: 0.57735026}, f.Planes[4])
	assert.Equal(t, Plane{Norm: Vector3{0.19841896, 0.5158893, 0.83335966}, Off: 1.15083}, f.Planes[5])
}

func TestFrustumIntersectsBox(t *testing.T) {
	f := &Frustum{
		Planes: [6]Plane{
			{Norm: Vector3{1, 0, 0}, Off: 1},
			{Norm: Vector3{-1, 0, 0}, Off: 2},
			{Norm: Vector3{0, 1, 0}, Off: 3},
			{Norm: Vector3{0, -1, 0}, Off: 4},
			{Norm: Vector3{0, 0, 1}, Off: 0},
			{Norm: Vector3{0, 0, -1}, Off: -3},
		},
	}

	box := Box3{
		Min: Vector3{-1, -1, -1},
		Max: Vector3{1, 1, 1},
	}

	result := f.IntersectsBox(box)
	assert.False(t, result)

	box = Box3{
		Min: Vector3{2, 2, 2},
		Max: Vector3{3, 3, 3},
	}

	result = f.IntersectsBox(box)
	assert.False(t, result)
}
