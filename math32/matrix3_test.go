// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatrix3(t *testing.T) {
	v0 := Vec2(0, 0)
	vx := Vec2(1, 0)
	vy := Vec2(0, 1)
	vxy := Vec2(1, 1)

	assert.Equal(t, vx, Identity3().MulVector2AsPoint(vx))
	assert.Equal(t, vy, Identity3().MulVector2AsPoint(vy))
	assert.Equal(t, vxy, Identity3().MulVector2AsPoint(vxy))

	assert.Equal(t, vxy, Matrix3FromMatrix2(Translate2D(1, 1)).MulVector2AsPoint(v0))

	assert.Equal(t, vxy.MulScalar(2), Matrix3FromMatrix2(Scale2D(2, 2)).MulVector2AsPoint(vxy))

	tolAssertEqualVector(t, standardTol, vy, Matrix3FromMatrix2(Rotate2D(DegToRad(90))).MulVector2AsPoint(vx))  // left
	tolAssertEqualVector(t, standardTol, vx, Matrix3FromMatrix2(Rotate2D(DegToRad(-90))).MulVector2AsPoint(vy)) // right
	tolAssertEqualVector(t, standardTol, vxy.Normal(), Matrix3FromMatrix2(Rotate2D(DegToRad(45))).MulVector2AsPoint(vx))
	tolAssertEqualVector(t, standardTol, vxy.Normal(), Matrix3FromMatrix2(Rotate2D(DegToRad(-45))).MulVector2AsPoint(vy))

	tolAssertEqualVector(t, standardTol, vy, Matrix3FromMatrix2(Rotate2D(DegToRad(-90))).Inverse().MulVector2AsPoint(vx)) // left
	tolAssertEqualVector(t, standardTol, vx, Matrix3FromMatrix2(Rotate2D(DegToRad(90))).Inverse().MulVector2AsPoint(vy))  // right

	// 1,0 -> scale(2) = 2,0 -> rotate 90 = 0,2 -> trans 1,1 -> 1,3
	// multiplication order is *reverse* of "logical" order:
	tolAssertEqualVector(t, standardTol, Vec2(1, 3), Matrix3Translate2D(1, 1).Mul(Matrix3Rotate2D(DegToRad(90))).Mul(Matrix3Scale2D(2, 2)).MulVector2AsPoint(vx))

	// xmat := Matrix3Translate2D(1, 1).Mul(Matrix3Rotate2D(DegToRad(90))).Mul(Matrix3Scale2D(2, 2)).MulVector2AsPoint(vx))
}

func TestMatrix3SetFromMatrix4(t *testing.T) {
	m := &Matrix3{}
	src := &Matrix4{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}

	m.SetFromMatrix4(src)

	expected := &Matrix3{
		1, 2, 3,
		5, 6, 7,
		9, 10, 11,
	}

	assert.Equal(t, expected, m)
}

func TestMatrix3SetFromMatrix2(t *testing.T) {
	m := &Matrix3{}
	src := Matrix2{
		XX: 1, XY: 2,
		YX: 3, YY: 4,
	}

	m.SetFromMatrix2(src)

	expected := &Matrix3{
		1, 2, 0,
		3, 4, 0,
		0, 0, 1,
	}

	assert.Equal(t, expected, m)
}

func TestMatrix3MulScalar(t *testing.T) {
	m := Matrix3{
		1, 2, 3,
		4, 5, 6,
		7, 8, 9,
	}
	original := m
	s := float32(2)

	expected := Matrix3{
		2, 4, 6,
		8, 10, 12,
		14, 16, 18,
	}

	result := m.MulScalar(s)

	assert.Equal(t, expected, result)
	assert.Equal(t, original, m)
}
