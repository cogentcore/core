// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"github.com/stretchr/testify/assert"
)

func tolAssertEqualVector(t *testing.T, tol float32, vt, va Vector2) {
	tolassert.EqualTol(t, vt.X, va.X, tol)
	tolassert.EqualTol(t, vt.Y, va.Y, tol)
}

const standardTol = float32(1.0e-6)

func TestMatrix2(t *testing.T) {
	v0 := Vec2(0, 0)
	vx := Vec2(1, 0)
	vy := Vec2(0, 1)
	vxy := Vec2(1, 1)

	assert.Equal(t, vx, Identity3().MulVector2AsPoint(vx))
	assert.Equal(t, vy, Identity3().MulVector2AsPoint(vy))
	assert.Equal(t, vxy, Identity3().MulVector2AsPoint(vxy))

	assert.Equal(t, vxy, Translate2D(1, 1).MulVector2AsPoint(v0))

	assert.Equal(t, vxy.MulScalar(2), Scale2D(2, 2).MulVector2AsPoint(vxy))

	tolAssertEqualVector(t, standardTol, vy, Rotate2D(DegToRad(90)).MulVector2AsPoint(vx))  // left
	tolAssertEqualVector(t, standardTol, vx, Rotate2D(DegToRad(-90)).MulVector2AsPoint(vy)) // right
	tolAssertEqualVector(t, standardTol, vxy.Normal(), Rotate2D(DegToRad(45)).MulVector2AsPoint(vx))
	tolAssertEqualVector(t, standardTol, vxy.Normal(), Rotate2D(DegToRad(-45)).MulVector2AsPoint(vy))

	tolAssertEqualVector(t, standardTol, vy, Rotate2D(DegToRad(-90)).Inverse().MulVector2AsPoint(vx))
	tolAssertEqualVector(t, standardTol, vx, Rotate2D(DegToRad(90)).Inverse().MulVector2AsPoint(vy))

	tolAssertEqualVector(t, standardTol, vxy, Rotate2D(DegToRad(-45)).Mul(Rotate2D(DegToRad(45))).MulVector2AsPoint(vxy))
	tolAssertEqualVector(t, standardTol, vxy, Rotate2D(DegToRad(-45)).Mul(Rotate2D(DegToRad(-45)).Inverse()).MulVector2AsPoint(vxy))

	tolassert.EqualTol(t, DegToRad(-90), Rotate2D(DegToRad(-90)).ExtractRot(), standardTol)
	tolassert.EqualTol(t, DegToRad(-45), Rotate2D(DegToRad(-45)).ExtractRot(), standardTol)
	tolassert.EqualTol(t, DegToRad(45), Rotate2D(DegToRad(45)).ExtractRot(), standardTol)
	tolassert.EqualTol(t, DegToRad(90), Rotate2D(DegToRad(90)).ExtractRot(), standardTol)

	// 1,0 -> scale(2) = 2,0 -> rotate 90 = 0,2 -> trans 1,1 -> 1,3
	// multiplication order is *reverse* of "logical" order:
	tolAssertEqualVector(t, standardTol, Vec2(1, 3), Translate2D(1, 1).Mul(Rotate2D(DegToRad(90))).Mul(Scale2D(2, 2)).MulVector2AsPoint(vx))

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
