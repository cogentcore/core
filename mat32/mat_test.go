// Copyright 2021 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mat32

import (
	"testing"

	"cogentcore.org/core/glop/tolassert"
	"github.com/stretchr/testify/assert"
)

func TolAssertEqualVec(t *testing.T, tol float32, vt, va Vec2) {
	tolassert.EqualTol(t, vt.X, va.X, tol)
	tolassert.EqualTol(t, vt.Y, va.Y, tol)
}

const StandardTol = float32(1.0e-6)

func TestMat2(t *testing.T) {
	v0 := V2(0, 0)
	vx := V2(1, 0)
	vy := V2(0, 1)
	vxy := V2(1, 1)

	assert.Equal(t, vx, Identity3().MulVec2AsPt(vx))
	assert.Equal(t, vy, Identity3().MulVec2AsPt(vy))
	assert.Equal(t, vxy, Identity3().MulVec2AsPt(vxy))

	assert.Equal(t, vxy, Translate2D(1, 1).MulVec2AsPoint(v0))

	assert.Equal(t, vxy.MulScalar(2), Scale2D(2, 2).MulVec2AsPoint(vxy))

	TolAssertEqualVec(t, StandardTol, vy, Rotate2D(DegToRad(90)).MulVec2AsPoint(vx))  // left
	TolAssertEqualVec(t, StandardTol, vx, Rotate2D(DegToRad(-90)).MulVec2AsPoint(vy)) // right
	TolAssertEqualVec(t, StandardTol, vxy.Normal(), Rotate2D(DegToRad(45)).MulVec2AsPoint(vx))
	TolAssertEqualVec(t, StandardTol, vxy.Normal(), Rotate2D(DegToRad(-45)).MulVec2AsPoint(vy))

	TolAssertEqualVec(t, StandardTol, vy, Rotate2D(DegToRad(-90)).Inverse().MulVec2AsPoint(vx))
	TolAssertEqualVec(t, StandardTol, vx, Rotate2D(DegToRad(90)).Inverse().MulVec2AsPoint(vy))

	TolAssertEqualVec(t, StandardTol, vxy, Rotate2D(DegToRad(-45)).Mul(Rotate2D(DegToRad(45))).MulVec2AsPoint(vxy))
	TolAssertEqualVec(t, StandardTol, vxy, Rotate2D(DegToRad(-45)).Mul(Rotate2D(DegToRad(-45)).Inverse()).MulVec2AsPoint(vxy))

	tolassert.EqualTol(t, DegToRad(-90), Rotate2D(DegToRad(-90)).ExtractRot(), StandardTol)
	tolassert.EqualTol(t, DegToRad(-45), Rotate2D(DegToRad(-45)).ExtractRot(), StandardTol)
	tolassert.EqualTol(t, DegToRad(45), Rotate2D(DegToRad(45)).ExtractRot(), StandardTol)
	tolassert.EqualTol(t, DegToRad(90), Rotate2D(DegToRad(90)).ExtractRot(), StandardTol)

	// 1,0 -> scale(2) = 2,0 -> rotate 90 = 0,2 -> trans 1,1 -> 1,3
	// multiplication order is *reverse* of "logical" order:
	TolAssertEqualVec(t, StandardTol, V2(1, 3), Translate2D(1, 1).Mul(Rotate2D(DegToRad(90))).Mul(Scale2D(2, 2)).MulVec2AsPoint(vx))

}

func TestMat3(t *testing.T) {
	v0 := V2(0, 0)
	vx := V2(1, 0)
	vy := V2(0, 1)
	vxy := V2(1, 1)

	assert.Equal(t, vx, Identity3().MulVec2AsPt(vx))
	assert.Equal(t, vy, Identity3().MulVec2AsPt(vy))
	assert.Equal(t, vxy, Identity3().MulVec2AsPt(vxy))

	assert.Equal(t, vxy, Mat3FromMat2(Translate2D(1, 1)).MulVec2AsPt(v0))

	assert.Equal(t, vxy.MulScalar(2), Mat3FromMat2(Scale2D(2, 2)).MulVec2AsPt(vxy))

	TolAssertEqualVec(t, StandardTol, vy, Mat3FromMat2(Rotate2D(DegToRad(90))).MulVec2AsPt(vx))  // left
	TolAssertEqualVec(t, StandardTol, vx, Mat3FromMat2(Rotate2D(DegToRad(-90))).MulVec2AsPt(vy)) // right
	TolAssertEqualVec(t, StandardTol, vxy.Normal(), Mat3FromMat2(Rotate2D(DegToRad(45))).MulVec2AsPt(vx))
	TolAssertEqualVec(t, StandardTol, vxy.Normal(), Mat3FromMat2(Rotate2D(DegToRad(-45))).MulVec2AsPt(vy))

	TolAssertEqualVec(t, StandardTol, vy, Mat3FromMat2(Rotate2D(DegToRad(-90))).Inverse().MulVec2AsPt(vx)) // left
	TolAssertEqualVec(t, StandardTol, vx, Mat3FromMat2(Rotate2D(DegToRad(90))).Inverse().MulVec2AsPt(vy))  // right

	// 1,0 -> scale(2) = 2,0 -> rotate 90 = 0,2 -> trans 1,1 -> 1,3
	// multiplication order is *reverse* of "logical" order:
	TolAssertEqualVec(t, StandardTol, V2(1, 3), Mat3Translate2D(1, 1).Mul(Mat3Rotate2D(DegToRad(90))).Mul(Mat3Scale2D(2, 2)).MulVec2AsPt(vx))

	// xmat := Mat3Translate2D(1, 1).Mul(Mat3Rotate2D(DegToRad(90))).Mul(Mat3Scale2D(2, 2)).MulVec2AsPt(vx))
}

func TestMat4Prjn(t *testing.T) {
	pts := []Vec3{{0.0, 0.0, 0.0}, {1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {0.5, 0.5, 0.5}, {-0.5, -0.5, -0.5}, {1, 1, 1}, {-1, -1, -1}}

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
