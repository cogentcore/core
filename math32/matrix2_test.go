// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"github.com/stretchr/testify/assert"
)

const standardTol = float32(1.0e-6)

func tolAssertEqualVector(t *testing.T, vt, va Vector2, tols ...float32) {
	tol := standardTol
	if len(tols) == 1 {
		tol = tols[0]
	}
	assert.InDelta(t, vt.X, va.X, float64(tol))
	assert.InDelta(t, vt.Y, va.Y, float64(tol))
}

func tolAssertEqualMatrix2(t *testing.T, vt, va Matrix2, tols ...float32) {
	tol := standardTol
	if len(tols) == 1 {
		tol = tols[0]
	}
	assert.InDelta(t, vt.XX, va.XX, float64(tol))
	assert.InDelta(t, vt.YX, va.YX, float64(tol))
	assert.InDelta(t, vt.XY, va.XY, float64(tol))
	assert.InDelta(t, vt.YY, va.YY, float64(tol))
	assert.InDelta(t, vt.X0, va.X0, float64(tol))
	assert.InDelta(t, vt.Y0, va.Y0, float64(tol))
}

func TestMatrix2(t *testing.T) {
	v0 := Vec2(0, 0)
	vx := Vec2(1, 0)
	vy := Vec2(0, 1)
	vxy := Vec2(1, 1)

	rot90 := DegToRad(90)
	rot45 := DegToRad(45)

	assert.Equal(t, vx, Identity3().MulPoint(vx))
	assert.Equal(t, vy, Identity3().MulPoint(vy))
	assert.Equal(t, vxy, Identity3().MulPoint(vxy))

	assert.Equal(t, vxy, Translate2D(1, 1).MulPoint(v0))

	assert.Equal(t, vxy.MulScalar(2), Scale2D(2, 2).MulPoint(vxy))

	tolAssertEqualVector(t, vy, Rotate2D(rot90).MulPoint(vx))  // left, CCW
	tolAssertEqualVector(t, vx, Rotate2D(-rot90).MulPoint(vy)) // right, CW
	tolAssertEqualVector(t, vxy.Normal(), Rotate2D(rot45).MulPoint(vx))
	tolAssertEqualVector(t, vxy.Normal(), Rotate2D(-rot45).MulPoint(vy))

	tolAssertEqualVector(t, vy, Rotate2D(-rot90).Inverse().MulPoint(vx))
	tolAssertEqualVector(t, vx, Rotate2D(rot90).Inverse().MulPoint(vy))

	tolAssertEqualVector(t, vxy, Rotate2D(-rot45).Mul(Rotate2D(rot45)).MulPoint(vxy))
	tolAssertEqualVector(t, vxy, Rotate2D(-rot45).Mul(Rotate2D(-rot45).Inverse()).MulPoint(vxy))

	tolassert.EqualTol(t, -rot90, Rotate2D(-rot90).ExtractRot(), standardTol)
	tolassert.EqualTol(t, -rot45, Rotate2D(-rot45).ExtractRot(), standardTol)
	tolassert.EqualTol(t, rot45, Rotate2D(rot45).ExtractRot(), standardTol)
	tolassert.EqualTol(t, rot90, Rotate2D(rot90).ExtractRot(), standardTol)

	// 1,0 -> scale(2) = 2,0 -> rotate 90 = 0,2 -> trans 1,1 -> 1,3
	// multiplication order is *reverse* of "logical" order:
	tolAssertEqualVector(t, Vec2(1, 3), Translate2D(1, 1).Mul(Rotate2D(rot90)).Mul(Scale2D(2, 2)).MulPoint(vx))

}

func TestMatrix2SetString(t *testing.T) {
	tests := []struct {
		str     string
		wantErr bool
		want    Matrix2
	}{
		{
			str:     "none",
			wantErr: false,
			want:    Identity2(),
		},
		{
			str:     "matrix(1, 2, 3, 4, 5, 6)",
			wantErr: false,
			want:    Matrix2{1, 2, 3, 4, 5, 6},
		},
		{
			str:     "translate(1, 2)",
			wantErr: false,
			want:    Matrix2{XX: 1, YX: 0, XY: 0, YY: 1, X0: 1, Y0: 2},
		},
		{
			str:     "invalid(1, 2)",
			wantErr: true,
			want:    Identity2(),
		},
	}

	for _, tt := range tests {
		a := &Matrix2{}
		err := a.SetString(tt.str)
		if tt.wantErr {
			assert.Error(t, err, tt.str)
		} else {
			assert.NoError(t, err, tt.str)
		}
		assert.Equal(t, tt.want, *a, tt.str)
	}
}

func TestMatrix2String(t *testing.T) {
	tests := []struct {
		matrix Matrix2
		want   string
	}{
		{
			matrix: Identity2(),
			want:   "none",
		},
		{
			matrix: Matrix2{XX: 1, YX: 2, XY: 3, YY: 4, X0: 5, Y0: 6},
			want:   "matrix(1,2,3,4,5,6)",
		},
		{
			matrix: Matrix2{XX: 2, XY: 0, YX: 0, YY: 2, X0: 0, Y0: 0},
			want:   "scale(2,2)",
		},
		{
			matrix: Matrix2{XX: 1, XY: 0, YX: 0, YY: 1, X0: 1, Y0: 2},
			want:   "translate(1,2)",
		},
		{
			matrix: Matrix2{XX: 2, XY: 0, YX: 0, YY: 2, X0: 1, Y0: 2},
			want:   "translate(1,2) scale(2,2)",
		},
	}

	for _, tt := range tests {
		got := tt.matrix.String()
		assert.Equal(t, tt.want, got)
	}
}

// tests from tdewolff/canvas package:
func TestMatrix2Canvas(t *testing.T) {
	p := Vector2{3, 4}
	rot90 := DegToRad(90)
	rot45 := DegToRad(45)
	tolAssertEqualVector(t, Identity2().Translate(2.0, 2.0).MulPoint(p), Vector2{5.0, 6.0})
	tolAssertEqualVector(t, Identity2().Scale(2.0, 2.0).MulPoint(p), Vector2{6.0, 8.0})
	tolAssertEqualVector(t, Identity2().Scale(1.0, -1.0).MulPoint(p), Vector2{3.0, -4.0})
	tolAssertEqualVector(t, Identity2().ScaleAbout(2.0, -1.0, 2.0, 2.0).MulPoint(p), Vector2{4.0, 0.0})
	tolAssertEqualVector(t, Identity2().Shear(1.0, 0.0).MulPoint(p), Vector2{7.0, 4.0})
	tolAssertEqualVector(t, Identity2().Rotate(rot90).MulPoint(p), p.Rot90CCW())
	tolAssertEqualVector(t, Identity2().RotateAbout(rot90, 5.0, 5.0).MulPoint(p), p.Rot(90.0*Pi/180.0, Vector2{5.0, 5.0}))
	tolAssertEqualVector(t, Identity2().Rotate(rot90).Transpose().MulPoint(p), p.Rot90CW())
	tolAssertEqualMatrix2(t, Identity2().Scale(2.0, 4.0).Inverse(), Identity2().Scale(0.5, 0.25))
	tolAssertEqualMatrix2(t, Identity2().Rotate(rot90).Inverse(), Identity2().Rotate(-rot90))
	tolAssertEqualMatrix2(t, Identity2().Rotate(rot90).Scale(2.0, 1.0), Identity2().Scale(1.0, 2.0).Rotate(rot90))

	lambda1, lambda2, v1, v2 := Identity2().Rotate(rot90).Scale(2.0, 1.0).Rotate(-rot90).Eigen()
	assert.Equal(t, lambda1, float32(1.0))
	assert.Equal(t, lambda2, float32(2.0))
	// fmt.Println(v1, v2)
	tolAssertEqualVector(t, v1, Vector2{1.0, 0.0})
	tolAssertEqualVector(t, v2, Vector2{0.0, 1.0})

	halfSqrt2 := 1.0 / Sqrt(2.0)
	lambda1, lambda2, v1, v2 = Identity2().Shear(1.0, 1.0).Eigen()
	assert.Equal(t, lambda1, float32(0.0))
	assert.Equal(t, lambda2, float32(2.0))
	tolAssertEqualVector(t, v1, Vector2{-halfSqrt2, halfSqrt2})
	tolAssertEqualVector(t, v2, Vector2{halfSqrt2, halfSqrt2})

	lambda1, lambda2, v1, v2 = Identity2().Shear(1.0, 0.0).Eigen()
	assert.Equal(t, lambda1, float32(1.0))
	assert.Equal(t, lambda2, float32(1.0))
	tolAssertEqualVector(t, v1, Vector2{1.0, 0.0})
	tolAssertEqualVector(t, v2, Vector2{1.0, 0.0})

	lambda1, lambda2, v1, v2 = Identity2().Scale(NaN(), NaN()).Eigen()
	assert.True(t, IsNaN(lambda1))
	assert.True(t, IsNaN(lambda2))
	tolAssertEqualVector(t, v1, Vector2{0.0, 0.0})
	tolAssertEqualVector(t, v2, Vector2{0.0, 0.0})

	tx, ty, phi, sx, sy, theta := Identity2().Rotate(rot90).Scale(2.0, 1.0).Rotate(-rot90).Translate(0.0, 10.0).Decompose()
	assert.InDelta(t, tx, float32(0.0), 1.0e-6)
	assert.Equal(t, ty, float32(20.0))
	assert.Equal(t, phi, rot90)
	assert.Equal(t, sx, float32(2.0))
	assert.Equal(t, sy, float32(1.0))
	assert.Equal(t, theta, -rot90)

	x, y := Identity2().Translate(p.X, p.Y).Pos()
	assert.Equal(t, x, p.X)
	assert.Equal(t, y, p.Y)

	tolAssertEqualMatrix2(t, Identity2().Shear(1.0, 1.0), Identity2().Rotate(rot45).Scale(2.0, 0.0).Rotate(-rot45))
}

func TestSolveQuadraticFormula(t *testing.T) {
	x1, x2 := solveQuadraticFormula(0.0, 0.0, 0.0)
	assert.Equal(t, x1, float32(0.0))
	assert.True(t, IsNaN(x2))

	x1, x2 = solveQuadraticFormula(0.0, 0.0, 1.0)
	assert.True(t, IsNaN(x1))
	assert.True(t, IsNaN(x2))

	x1, x2 = solveQuadraticFormula(0.0, 1.0, 1.0)
	assert.Equal(t, x1, float32(-1.0))
	assert.True(t, IsNaN(x2))

	x1, x2 = solveQuadraticFormula(1.0, 1.0, 0.0)
	assert.Equal(t, x1, float32(0.0))
	assert.Equal(t, x2, float32(-1.0))

	x1, x2 = solveQuadraticFormula(1.0, 1.0, 1.0) // discriminant negative
	assert.True(t, IsNaN(x1))
	assert.True(t, IsNaN(x2))

	x1, x2 = solveQuadraticFormula(1.0, 1.0, 0.25) // discriminant zero
	assert.Equal(t, x1, float32(-0.5))
	assert.True(t, IsNaN(x2))

	x1, x2 = solveQuadraticFormula(2.0, -5.0, 2.0) // negative b, flip x1 and x2
	assert.Equal(t, x1, float32(0.5))
	assert.Equal(t, x2, float32(2.0))

	x1, x2 = solveQuadraticFormula(-4.0, 0.0, 0.0)
	assert.Equal(t, x1, float32(0.0))
	assert.True(t, IsNaN(x2))
}
