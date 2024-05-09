// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"github.com/stretchr/testify/assert"
)

func TestQuatSetFromEuler(t *testing.T) {
	euler := Vector3{X: 0.1, Y: 0.2, Z: 0.3}

	q := Quat{}
	q.SetFromEuler(euler)

	expected := Quat{X: 0.034270797, Y: 0.10602052, Z: 0.14357218, W: 0.9833474}

	tolassert.Equal(t, expected.X, q.X)
	tolassert.Equal(t, expected.Y, q.Y)
	tolassert.Equal(t, expected.Z, q.Z)
	tolassert.Equal(t, expected.W, q.W)
}

func TestQuatSetFromAxisAngle(t *testing.T) {
	axis := Vector3{X: 1, Y: 0, Z: 0}

	q := Quat{}
	q.SetFromAxisAngle(axis, Pi/2)

	expected := Quat{X: 0.7071068, Y: 0, Z: 0, W: 0.70710677}

	assert.Equal(t, expected, q)
}

func TestQuatSetFromRotationMatrix(t *testing.T) {
	m := &Matrix4{
		0, 1, 2, 3,
		4, 5, 6, 7,
		8, 9, 10, 11,
		12, 13, 14, 15,
	}

	q := Quat{}
	q.SetFromRotationMatrix(m)

	expected := NewQuat(-0.375, 0.75, -0.375, 2)

	assert.Equal(t, expected, q)
}

func TestQuatSetFromUnitVectors(t *testing.T) {
	vFrom := Vector3{X: 1, Y: 2, Z: 3}
	vTo := Vector3{X: 4, Y: 5, Z: 6}

	q := Quat{}
	q.SetFromUnitVectors(vFrom, vTo)

	expected := NewQuat(-0.08873564, 0.17747128, -0.08873564, 0.9760921)

	assert.Equal(t, expected, q)
}

func TestQuatNormalize(t *testing.T) {
	q := Quat{X: 1, Y: 2, Z: 3, W: 4}
	q.Normalize()

	expected := Quat{X: 0.18257419, Y: 0.36514837, Z: 0.5477226, W: 0.73029674}

	assert.Equal(t, expected, q)
}

func TestQuatNormalizeZeroLength(t *testing.T) {
	q := Quat{X: 0, Y: 0, Z: 0, W: 0}
	q.Normalize()

	expected := Quat{X: 0, Y: 0, Z: 0, W: 1}

	assert.Equal(t, expected, q)
}

func TestQuatNormalizeFast(t *testing.T) {
	q := Quat{X: 1, Y: 2, Z: 3, W: 4}
	q.NormalizeFast()

	expected := Quat{X: -13.5, Y: -27, Z: -40.5, W: -54}

	assert.Equal(t, expected, q)
}

func TestQuatNormalizeFastZeroLength(t *testing.T) {
	q := Quat{X: 0, Y: 0, Z: 0, W: 0}
	q.NormalizeFast()

	expected := Quat{X: 0, Y: 0, Z: 0, W: 0}

	assert.Equal(t, expected, q)
}

func TestQuatMulQuats(t *testing.T) {
	q1 := Quat{X: 1, Y: 2, Z: 3, W: 4}
	q2 := Quat{X: 5, Y: 6, Z: 7, W: 8}

	q := Quat{}
	q.MulQuats(q1, q2)

	expected := Quat{X: 24, Y: 48, Z: 48, W: -6}

	assert.Equal(t, expected, q)
}

func TestQuatSlerp(t *testing.T) {
	q1 := Quat{X: 1, Y: 2, Z: 3, W: 4}
	q2 := Quat{X: 5, Y: 6, Z: 7, W: 8}

	q := q1
	q.Slerp(q2, 0)
	assert.Equal(t, q1, q)

	q = q1
	q.Slerp(q2, 1)
	assert.Equal(t, q2, q)

	q = q1
	q.Slerp(q2, 0.5)
	assert.Equal(t, Quat{X: 1, Y: 2, Z: 3, W: 4}, q)
}
