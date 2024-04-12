// Copyright 2021 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"cogentcore.org/core/gox/tolassert"
	"github.com/stretchr/testify/assert"
)

func TolAssertEqualVector(t *testing.T, tol float32, vt, va Vector2) {
	tolassert.EqualTol(t, vt.X, va.X, tol)
	tolassert.EqualTol(t, vt.Y, va.Y, tol)
}

const StandardTol = float32(1.0e-6)

func TestMat2(t *testing.T) {
	v0 := Vec2(0, 0)
	vx := Vec2(1, 0)
	vy := Vec2(0, 1)
	vxy := Vec2(1, 1)

	assert.Equal(t, vx, Identity3().MulVector2AsPt(vx))
	assert.Equal(t, vy, Identity3().MulVector2AsPt(vy))
	assert.Equal(t, vxy, Identity3().MulVector2AsPt(vxy))

	assert.Equal(t, vxy, Translate2D(1, 1).MulVector2AsPoint(v0))

	assert.Equal(t, vxy.MulScalar(2), Scale2D(2, 2).MulVector2AsPoint(vxy))

	TolAssertEqualVector(t, StandardTol, vy, Rotate2D(DegToRad(90)).MulVector2AsPoint(vx))  // left
	TolAssertEqualVector(t, StandardTol, vx, Rotate2D(DegToRad(-90)).MulVector2AsPoint(vy)) // right
	TolAssertEqualVector(t, StandardTol, vxy.Normal(), Rotate2D(DegToRad(45)).MulVector2AsPoint(vx))
	TolAssertEqualVector(t, StandardTol, vxy.Normal(), Rotate2D(DegToRad(-45)).MulVector2AsPoint(vy))

	TolAssertEqualVector(t, StandardTol, vy, Rotate2D(DegToRad(-90)).Inverse().MulVector2AsPoint(vx))
	TolAssertEqualVector(t, StandardTol, vx, Rotate2D(DegToRad(90)).Inverse().MulVector2AsPoint(vy))

	TolAssertEqualVector(t, StandardTol, vxy, Rotate2D(DegToRad(-45)).Mul(Rotate2D(DegToRad(45))).MulVector2AsPoint(vxy))
	TolAssertEqualVector(t, StandardTol, vxy, Rotate2D(DegToRad(-45)).Mul(Rotate2D(DegToRad(-45)).Inverse()).MulVector2AsPoint(vxy))

	tolassert.EqualTol(t, DegToRad(-90), Rotate2D(DegToRad(-90)).ExtractRot(), StandardTol)
	tolassert.EqualTol(t, DegToRad(-45), Rotate2D(DegToRad(-45)).ExtractRot(), StandardTol)
	tolassert.EqualTol(t, DegToRad(45), Rotate2D(DegToRad(45)).ExtractRot(), StandardTol)
	tolassert.EqualTol(t, DegToRad(90), Rotate2D(DegToRad(90)).ExtractRot(), StandardTol)

	// 1,0 -> scale(2) = 2,0 -> rotate 90 = 0,2 -> trans 1,1 -> 1,3
	// multiplication order is *reverse* of "logical" order:
	TolAssertEqualVector(t, StandardTol, Vec2(1, 3), Translate2D(1, 1).Mul(Rotate2D(DegToRad(90))).Mul(Scale2D(2, 2)).MulVector2AsPoint(vx))

}

func TestMat3(t *testing.T) {
	v0 := Vec2(0, 0)
	vx := Vec2(1, 0)
	vy := Vec2(0, 1)
	vxy := Vec2(1, 1)

	assert.Equal(t, vx, Identity3().MulVector2AsPt(vx))
	assert.Equal(t, vy, Identity3().MulVector2AsPt(vy))
	assert.Equal(t, vxy, Identity3().MulVector2AsPt(vxy))

	assert.Equal(t, vxy, Mat3FromMat2(Translate2D(1, 1)).MulVector2AsPt(v0))

	assert.Equal(t, vxy.MulScalar(2), Mat3FromMat2(Scale2D(2, 2)).MulVector2AsPt(vxy))

	TolAssertEqualVector(t, StandardTol, vy, Mat3FromMat2(Rotate2D(DegToRad(90))).MulVector2AsPt(vx))  // left
	TolAssertEqualVector(t, StandardTol, vx, Mat3FromMat2(Rotate2D(DegToRad(-90))).MulVector2AsPt(vy)) // right
	TolAssertEqualVector(t, StandardTol, vxy.Normal(), Mat3FromMat2(Rotate2D(DegToRad(45))).MulVector2AsPt(vx))
	TolAssertEqualVector(t, StandardTol, vxy.Normal(), Mat3FromMat2(Rotate2D(DegToRad(-45))).MulVector2AsPt(vy))

	TolAssertEqualVector(t, StandardTol, vy, Mat3FromMat2(Rotate2D(DegToRad(-90))).Inverse().MulVector2AsPt(vx)) // left
	TolAssertEqualVector(t, StandardTol, vx, Mat3FromMat2(Rotate2D(DegToRad(90))).Inverse().MulVector2AsPt(vy))  // right

	// 1,0 -> scale(2) = 2,0 -> rotate 90 = 0,2 -> trans 1,1 -> 1,3
	// multiplication order is *reverse* of "logical" order:
	TolAssertEqualVector(t, StandardTol, Vec2(1, 3), Mat3Translate2D(1, 1).Mul(Mat3Rotate2D(DegToRad(90))).Mul(Mat3Scale2D(2, 2)).MulVector2AsPt(vx))

	// xmat := Mat3Translate2D(1, 1).Mul(Mat3Rotate2D(DegToRad(90))).Mul(Mat3Scale2D(2, 2)).MulVector2AsPt(vx))
}

func TestMat4Prjn(t *testing.T) {
	pts := []Vector3{{0.0, 0.0, 0.0}, {1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {0.5, 0.5, 0.5}, {-0.5, -0.5, -0.5}, {1, 1, 1}, {-1, -1, -1}}

	campos := V3(0, 0, 10)
	target := V3(0, 0, 0)
	var lookq Quat
	lookq.SetFromRotationMatrix(NewLookAt(campos, target, V3(0, 1, 0)))
	scale := V3(1, 1, 1)
	var cview Mat4
	cview.SetTransform(campos, lookq, scale)
	view, _ := cview.Inverse()

	var glprjn Mat4
	glprjn.SetPerspective(90, 1.5, 0.01, 100)

	var proj Mat4
	proj.MulMatrices(&glprjn, view)

	for _, pt := range pts {
		pjpt := pt.MulMat4(&proj)
		_ = pjpt
		// fmt.Printf("pt: %v\t   pj: %v\n", pt, pjpt)
	}
}
