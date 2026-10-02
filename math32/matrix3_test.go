// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"github.com/stretchr/testify/assert"
)

func TestMatrix3(t *testing.T) {
	v0 := Vec2(0, 0)
	vx := Vec2(1, 0)
	vy := Vec2(0, 1)
	vxy := Vec2(1, 1)

	assert.Equal(t, vx, Identity3().MulPoint(vx))
	assert.Equal(t, vy, Identity3().MulPoint(vy))
	assert.Equal(t, vxy, Identity3().MulPoint(vxy))

	assert.Equal(t, vxy, Matrix3FromMatrix2(Translate2D(1, 1)).MulPoint(v0))

	assert.Equal(t, vxy.MulScalar(2), Matrix3FromMatrix2(Scale2D(2, 2)).MulPoint(vxy))

	tolAssertEqualVector(t, vy, Matrix3FromMatrix2(Rotate2D(DegToRad(90))).MulPoint(vx))  // left
	tolAssertEqualVector(t, vx, Matrix3FromMatrix2(Rotate2D(DegToRad(-90))).MulPoint(vy)) // right
	tolAssertEqualVector(t, vxy.Normal(), Matrix3FromMatrix2(Rotate2D(DegToRad(45))).MulPoint(vx))
	tolAssertEqualVector(t, vxy.Normal(), Matrix3FromMatrix2(Rotate2D(DegToRad(-45))).MulPoint(vy))

	tolAssertEqualVector(t, vy, Matrix3FromMatrix2(Rotate2D(DegToRad(-90))).Inverse().MulPoint(vx)) // left
	tolAssertEqualVector(t, vx, Matrix3FromMatrix2(Rotate2D(DegToRad(90))).Inverse().MulPoint(vy))  // right

	// 1,0 -> scale(2) = 2,0 -> rotate 90 = 0,2 -> trans 1,1 -> 1,3
	// multiplication order is *reverse* of "logical" order, as in Matrix2:
	tolAssertEqualVector(t, Vec2(1, 3), Matrix3Translate2D(1, 1).Mul(Matrix3Rotate2D(DegToRad(90))).Mul(Matrix3Scale2D(2, 2)).MulPoint(vx))
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

	// stored column-wise, so this is the standard affine matrix
	// [XX XY X0 / YX YY Y0 / 0 0 1] = [1 2 0 / 3 4 0 / 0 0 1]
	expected := &Matrix3{
		1, 3, 0,
		2, 4, 0,
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

func TestMatrix3Determinant(t *testing.T) {
	m := Matrix3{
		1, 2, 3,
		4, 15, 6,
		7, 8, 9,
	}

	expected := float32(-120)

	result := m.Determinant()

	assert.Equal(t, expected, result)
}

func TestMatrix3ScaleCols(t *testing.T) {
	m := &Matrix3{
		1, 2, 3,
		4, 5, 6,
		7, 8, 9,
	}
	v := Vector3{2, 3, 4}

	expected := &Matrix3{
		2, 4, 6,
		12, 15, 18,
		28, 32, 36,
	}

	result := m.ScaleCols(v)

	assert.Equal(t, expected, result)
	assert.NotEqual(t, m, result)
}

func TestMatrix3SetNormalMatrix(t *testing.T) {
	src := &Matrix4{
		12, 2, 3, 4,
		5, 60, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}

	m := &Matrix3{}
	err := m.SetNormalMatrix(src)

	expected := Matrix3{
		0.104870245, 0.0014219694, -0.087095626,
		0.0014219694, 0.018663349, -0.018130109,
		-0.029505864, -0.012264486, 0.12619978,
	}

	assert.Equal(t, expected, *m)
	assert.NoError(t, err)
}

func TestMatrix3SetNormalMatrixError(t *testing.T) {
	src := &Matrix4{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}

	m := &Matrix3{}
	err := m.SetNormalMatrix(src)

	expected := Identity3()

	assert.Equal(t, expected, *m)
	assert.Error(t, err)
}

func TestMatrix3SetRotationFromQuat(t *testing.T) {
	q := Quat{X: 0.5, Y: 0.5, Z: 0.5, W: 0.5}

	m := &Matrix3{}
	m.SetRotationFromQuat(q)

	expected := &Matrix3{
		0, 1, 0,
		0, 0, 1,
		1, 0, 0,
	}

	assert.Equal(t, expected, m)
}

// TestMatrix3MulStandard checks that Mul is the standard matrix product
// a * b: element (row i, column j) is row i of a dotted with column j of b.
func TestMatrix3MulStandard(t *testing.T) {
	a := Mat3(1, 2, 3, 4, 5, 6, 7, 8, 10)
	b := Mat3(2, 0, 1, 3, 1, 0, 0, 4, 2)
	got := a.Mul(b)
	for c := range 3 {
		for r := range 3 {
			var want float32
			for k := range 3 {
				want += a[k*3+r] * b[c*3+k]
			}
			assert.Equal(t, want, got[c*3+r], "element row %d col %d", r, c)
		}
	}
	assert.Equal(t, a, a.Mul(Identity3()))
	assert.Equal(t, a, Identity3().Mul(a))

	// SetMul matches Mul
	sm := a
	sm.SetMul(b)
	assert.Equal(t, got, sm)
}

// TestMatrix3MulMatchesMatrix4 checks that Matrix3 and Matrix4 compose
// 3D rotations in the same order, which they did not before Matrix3.Mul
// was changed to the standard a * b product.
func TestMatrix3MulMatchesMatrix4(t *testing.T) {
	qa := NewQuatAxisAngle(Vec3(0, 0, 1), DegToRad(90))
	qb := NewQuatAxisAngle(Vec3(1, 0, 0), DegToRad(90))

	var a3, b3 Matrix3
	a3.SetRotationFromQuat(qa)
	b3.SetRotationFromQuat(qb)

	a4, b4 := Identity4(), Identity4()
	a4.SetRotationFromQuat(qa)
	b4.SetRotationFromQuat(qb)

	m3 := a3.Mul(b3)
	m4 := a4.Mul(b4)
	for _, v := range []Vector3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 2, 3}} {
		tolAssertEqualVector3(t, v.MulMatrix4(m4), m3.MulVector3(v))
		// and both apply b before a
		tolAssertEqualVector3(t, a3.MulVector3(b3.MulVector3(v)), m3.MulVector3(v))
	}
}

func tolAssertEqualVector3(t *testing.T, vt, va Vector3) {
	t.Helper()
	tolassert.EqualTol(t, vt.X, va.X, 1.0e-6)
	tolassert.EqualTol(t, vt.Y, va.Y, 1.0e-6)
	tolassert.EqualTol(t, vt.Z, va.Z, 1.0e-6)
}

// TestMatrix3Matches2D checks that the 2D affine side of Matrix3 agrees
// with Matrix2, which is the convention the rest of the codebase uses:
// the point multiplies on the right, so a chain of Mul calls applies its
// transforms right to left.
func TestMatrix3Matches2D(t *testing.T) {
	m2s := []Matrix2{
		Identity2(),
		Translate2D(3, -4),
		Scale2D(2, 0.5),
		Rotate2D(DegToRad(30)),
		Shear2D(0.3, -0.7),
		Identity2().Translate(1, 2).Rotate(DegToRad(45)).Scale(2, 3),
	}
	pts := []Vector2{{0, 0}, {1, 0}, {0, 1}, {1, 1}, {-2.5, 7.25}}

	for _, a2 := range m2s {
		a3 := Matrix3FromMatrix2(a2)
		for _, v := range pts {
			tolAssertEqualVector(t, a2.MulPoint(v), a3.MulPoint(v))
			tolAssertEqualVector(t, a2.MulVector(v), a3.MulVector(v))
		}
		for _, b2 := range m2s {
			b3 := Matrix3FromMatrix2(b2)
			// composition agrees in the same operand order as Matrix2
			m2 := a2.Mul(b2)
			m3 := a3.Mul(b3)
			tolAssertEqualMatrix3(t, Matrix3FromMatrix2(m2), m3)
			for _, v := range pts {
				tolAssertEqualVector(t, m2.MulPoint(v), m3.MulPoint(v))
			}
		}
	}

	// the 2D constructors agree with their Matrix2 counterparts
	tolAssertEqualMatrix3(t, Matrix3FromMatrix2(Translate2D(3, -4)), Matrix3Translate2D(3, -4))
	tolAssertEqualMatrix3(t, Matrix3FromMatrix2(Scale2D(3, -4)), Matrix3Scale2D(3, -4))
	tolAssertEqualMatrix3(t, Matrix3FromMatrix2(Rotate2D(0.7)), Matrix3Rotate2D(0.7))
}

func tolAssertEqualMatrix3(t *testing.T, mt, ma Matrix3) {
	t.Helper()
	for i := range mt {
		tolassert.EqualTol(t, mt[i], ma[i], 1.0e-6)
	}
}
