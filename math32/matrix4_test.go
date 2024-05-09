// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatrix4Projection(t *testing.T) {
	pts := []Vector3{{0.0, 0.0, 0.0}, {1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {0.5, 0.5, 0.5}, {-0.5, -0.5, -0.5}, {1, 1, 1}, {-1, -1, -1}}

	campos := Vec3(0, 0, 10)
	target := Vec3(0, 0, 0)
	var lookq Quat
	lookq.SetFromRotationMatrix(NewLookAt(campos, target, Vec3(0, 1, 0)))
	scale := Vec3(1, 1, 1)
	var cview Matrix4
	cview.SetTransform(campos, lookq, scale)
	view, _ := cview.Inverse()

	var glprojection Matrix4
	glprojection.SetPerspective(90, 1.5, 0.01, 100)

	var proj Matrix4
	proj.MulMatrices(&glprojection, view)

	for _, pt := range pts {
		pjpt := pt.MulMatrix4(&proj)
		_ = pjpt
		// fmt.Printf("pt: %v\t   pj: %v\n", pt, pjpt)
	}
}

func TestMatrix4Set(t *testing.T) {
	m := Matrix4{}
	m.Set(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16)

	expected := Matrix4{
		1, 5, 9, 13,
		2, 6, 10, 14,
		3, 7, 11, 15,
		4, 8, 12, 16,
	}

	assert.Equal(t, expected, m)
}

func TestMatrix4MulScalar(t *testing.T) {
	m := Matrix4{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}

	s := float32(2)

	m.MulScalar(s)

	expected := Matrix4{
		2, 4, 6, 8,
		10, 12, 14, 16,
		18, 20, 22, 24,
		26, 28, 30, 32,
	}

	assert.Equal(t, expected, m)
}

func TestMatrix4Determinant(t *testing.T) {
	m := Matrix4{
		10, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 17,
	}

	expected := float32(-36)

	result := m.Determinant()

	assert.Equal(t, expected, result)
}

func TestMatrix4Decompose(t *testing.T) {
	m := Matrix4{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}

	expectedPos := Vector3{13, 14, 15}
	expectedQuat := Quat{0.029251702, -0.09027569, 0.018377202, 0.78618026}
	expectedScale := Vector3{3.7416575, 10.488089, 17.378147}

	pos, quat, scale := m.Decompose()

	assert.Equal(t, expectedPos, pos)
	assert.Equal(t, expectedQuat, quat)
	assert.Equal(t, expectedScale, scale)
}

func TestMatrix4ExtractRotation(t *testing.T) {
	src := Matrix4{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}

	expected := Matrix4{
		0.26726124, 0.5345225, 0.8017837, 0,
		0.4767313, 0.5720775, 0.6674238, 0,
		0.5178918, 0.57543534, 0.63297886, 0,
		0, 0, 0, 0,
	}

	m := Matrix4{}
	m.ExtractRotation(&src)

	assert.Equal(t, expected, m)
}

func TestMatrix4SetRotationFromEuler(t *testing.T) {
	euler := Vector3{0.1, 0.2, 0.3}
	m := Matrix4{}
	m.SetRotationFromEuler(euler)

	expected := Matrix4{
		0.93629336, 0.31299183, -0.15934508, 0,
		-0.2896295, 0.9447025, 0.153792, 0,
		0.19866933, -0.0978434, 0.9751704, 0,
		0, 0, 0, 1,
	}

	assert.Equal(t, expected, m)
}

func TestMatrix4SetOrthographic(t *testing.T) {
	m := Matrix4{}
	width := float32(800)
	height := float32(600)
	near := float32(0.1)
	far := float32(100)

	m.SetOrthographic(width, height, near, far)

	expected := Matrix4{
		0.0025, 0, 0, 0,
		0, 0.0033333334, 0, 0,
		0, 0, -0.02002002, 0,
		0, 0, -1.002002, 1,
	}

	assert.Equal(t, expected, m)
}

func TestMatrix4SetVkFrustum(t *testing.T) {
	m := Matrix4{}
	left := float32(-1)
	right := float32(1)
	bottom := float32(-1)
	top := float32(1)
	near := float32(0.1)
	far := float32(100)

	m.SetVkFrustum(left, right, bottom, top, near, far)

	expected := Matrix4{
		0.1, 0, 0, 0,
		0, -0.1, 0, 0,
		0, 0, -1.001001, -1,
		0, 0, -0.1001001, 0,
	}

	assert.Equal(t, expected, m)
}

func TestMatrix4SetVkPerspective(t *testing.T) {
	m := Matrix4{}
	fov := float32(60)
	aspect := float32(16.0 / 9.0)
	near := float32(0.1)
	far := float32(100)

	m.SetVkPerspective(fov, aspect, near, far)

	expected := Matrix4{
		0.97427857, 0, 0, 0,
		0, -1.7320509, 0, 0,
		0, 0, -1.001001, -1,
		0, 0, -0.1001001, 0,
	}

	assert.Equal(t, expected, m)
}
