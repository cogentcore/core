// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/image/math/fixed"
)

func TestVector2(t *testing.T) {
	assert.Equal(t, Vector2{5, 10}, Vec2(5, 10))
	assert.Equal(t, Vec2(20, 20), Vector2Scalar(20))
	assert.Equal(t, Vec2(15, -5), Vector2FromPoint(image.Pt(15, -5)))
	assert.Equal(t, Vec2(8, 3), Vector2FromFixed(fixed.P(8, 3)))

	v := Vector2{}
	v.Set(-1, 7)
	assert.Equal(t, Vec2(-1, 7), v)

	v.SetScalar(8.12)
	assert.Equal(t, Vec2(8.12, 8.12), v)

	v.SetFromVector2i(Vec2i(8, 9))
	assert.Equal(t, Vec2(8, 9), v)

	v.SetDim(X, -4)
	assert.Equal(t, Vec2(-4, 9), v)

	v.SetDim(Y, 14.3)
	assert.Equal(t, Vec2(-4, 14.3), v)

	assert.Equal(t, float32(-4), v.Dim(X))
	assert.Equal(t, float32(14.3), v.Dim(Y))

	pt := image.Point{}

	SetPointDim(&pt, X, 2)
	assert.Equal(t, image.Pt(2, 0), pt)

	SetPointDim(&pt, Y, 43)
	assert.Equal(t, image.Pt(2, 43), pt)

	assert.Equal(t, 2, PointDim(pt, X))
	assert.Equal(t, 43, PointDim(pt, Y))

	v = Vec2(3.5, 19.2)

	assert.Equal(t, image.Pt(3, 19), v.ToPoint())
	assert.Equal(t, image.Pt(3, 19), v.ToPointFloor())
	assert.Equal(t, image.Pt(4, 20), v.ToPointCeil())
	assert.Equal(t, image.Pt(4, 19), v.ToPointRound())

	assert.Equal(t, fixed.Point26_6{224, 1228}, v.ToFixed())

	size := Vec2(4.7, 9.3)

	assert.Equal(t, image.Rect(3, 19, 8, 29), RectFromPosSizeMax(v, size))
	assert.Equal(t, image.Rect(4, 20, 8, 29), RectFromPosSizeMin(v, size))

	v.SetZero()
	assert.Equal(t, Vec2(0, 0), v)

	v.FromSlice([]float32{3, 2, 1}, 1)
	assert.Equal(t, Vec2(2, 1), v)

	slice := []float32{0, 0, 0, 0, 0}
	v.ToSlice(slice, 2)
	assert.Equal(t, []float32{0, 0, 2, 1, 0}, slice)

	v = Vec2(-2, 4)

	assert.Equal(t, Vec2(3, 1), v.Add(Vec2(5, -3)))
	assert.Equal(t, Vec2(4, 10), v.AddScalar(6))

	v.SetAdd(Vec2(2, 1))
	assert.Equal(t, Vec2(0, 5), v)

	v.SetAddScalar(-3)
	assert.Equal(t, Vec2(-3, 2), v)

	assert.Equal(t, Vec2(-8, -1), v.Sub(Vec2(5, 3)))
	assert.Equal(t, Vec2(2, 7), v.SubScalar(-5))

	v.SetSub(Vec2(2, 1))
	assert.Equal(t, Vec2(-5, 1), v)

	v.SetSubScalar(3)
	assert.Equal(t, Vec2(-8, -2), v)

	assert.Equal(t, Vec2(-40, -6), v.Mul(Vec2(5, 3)))
	assert.Equal(t, Vec2(40, 10), v.MulScalar(-5))

	v.SetMul(Vec2(2, 1))
	assert.Equal(t, Vec2(-16, -2), v)

	v.SetMulScalar(-3)
	assert.Equal(t, Vec2(48, 6), v)

	assert.Equal(t, Vec2(16, 3), v.Div(Vec2(3, 2)))
	assert.Equal(t, Vec2(-12, -1.5), v.DivScalar(-4))

	v.SetDiv(Vec2(2, 1))
	assert.Equal(t, Vec2(24, 6), v)

	v.SetDivScalar(-3)
	assert.Equal(t, Vec2(-8, -2), v)

	assert.Equal(t, Vec2(8, 2), v.Abs())

	assert.Equal(t, Vec2(-11, -2), v.Min(Vec2(-11, 3)))

	v.SetMin(Vec2(-11, 3))
	assert.Equal(t, Vec2(-11, -2), v)

	assert.Equal(t, Vec2(11, -2), v.Max(Vec2(11, -3)))

	v.SetMax(Vec2(11, -3))
	assert.Equal(t, Vec2(11, -2), v)

	v.Clamp(Vec2(1, 2), Vec2(9, 5))
	assert.Equal(t, Vec2(9, 2), v)

	v = Vec2(3.5, 19.2)

	assert.Equal(t, Vec2(3, 19), v.Floor())
	assert.Equal(t, Vec2(4, 20), v.Ceil())
	assert.Equal(t, Vec2(4, 19), v.Round())

	assert.Equal(t, Vec2(-3.5, -19.2), v.Negate())

	v = Vec2(3.5, 19)

	assert.Equal(t, Vec2(7.5, 19), v.AddDim(X, 4))
	assert.Equal(t, Vec2(3.5, 20), v.AddDim(Y, 1))

	assert.Equal(t, Vec2(-2, 19), v.SubDim(X, 5.5))
	assert.Equal(t, Vec2(3.5, 2), v.SubDim(Y, 17))

	assert.Equal(t, Vec2(7, 19), v.MulDim(X, 2))
	assert.Equal(t, Vec2(3.5, 57), v.MulDim(Y, 3))

	assert.Equal(t, Vec2(0.5, 19), v.DivDim(X, 7))
	assert.Equal(t, Vec2(3.5, 2.375), v.DivDim(Y, 8))
}
