// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"github.com/stretchr/testify/assert"
)

func TestVector3(t *testing.T) {
	assert.Equal(t, Vector3{5, 10, 7}, Vec3(5, 10, 7))
	assert.Equal(t, Vec3(20, 20, 20), Vector3Scalar(20))
	assert.Equal(t, Vec3(15, -5, 3), Vector3FromVector4(Vec4(15, -5, 3, 8)))

	v := Vector3{}
	v.Set(-1, 7, 12)
	assert.Equal(t, Vec3(-1, 7, 12), v)

	v.SetScalar(8.12)
	assert.Equal(t, Vec3(8.12, 8.12, 8.12), v)

	v.SetFromVector3i(Vec3i(8, 9, -7))
	assert.Equal(t, Vec3(8, 9, -7), v)

	v.SetDim(X, -4)
	assert.Equal(t, Vec3(-4, 9, -7), v)

	v.SetDim(Y, 14.3)
	assert.Equal(t, Vec3(-4, 14.3, -7), v)

	v.SetDim(Z, 3.14)
	assert.Equal(t, Vec3(-4, 14.3, 3.14), v)

	assert.Equal(t, float32(-4), v.Dim(X))
	assert.Equal(t, float32(14.3), v.Dim(Y))
	assert.Equal(t, float32(3.14), v.Dim(Z))

	v.SetZero()
	assert.Equal(t, Vec3(0, 0, 0), v)

	v.FromSlice([]float32{3, 2, 1, 4}, 1)
	assert.Equal(t, Vec3(2, 1, 4), v)

	slice := []float32{0, 0, 0, 0, 0, 0}
	v.ToSlice(slice, 2)
	assert.Equal(t, []float32{0, 0, 2, 1, 4, 0}, slice)

	v = Vec3(-2, 4, 5)

	assert.Equal(t, Vec3(3, 1, 7), v.Add(Vec3(5, -3, 2)))
	assert.Equal(t, Vec3(4, 10, 11), v.AddScalar(6))

	v.SetAdd(Vec3(2, 1, 4))
	assert.Equal(t, Vec3(0, 5, 9), v)

	v.SetAddScalar(-3)
	assert.Equal(t, Vec3(-3, 2, 6), v)

	assert.Equal(t, Vec3(-8, -1, 2), v.Sub(Vec3(5, 3, 4)))
	assert.Equal(t, Vec3(2, 7, 11), v.SubScalar(-5))

	v.SetSub(Vec3(2, 1, 5))
	assert.Equal(t, Vec3(-5, 1, 1), v)

	v.SetSubScalar(3)
	assert.Equal(t, Vec3(-8, -2, -2), v)

	assert.Equal(t, Vec3(-40, -6, -8), v.Mul(Vec3(5, 3, 4)))
	assert.Equal(t, Vec3(40, 10, 10), v.MulScalar(-5))

	v.SetMul(Vec3(2, 1, -4))
	assert.Equal(t, Vec3(-16, -2, 8), v)

	v.SetMulScalar(-3)
	assert.Equal(t, Vec3(48, 6, -24), v)

	assert.Equal(t, Vec3(16, 3, 12), v.Div(Vec3(3, 2, -2)))
	assert.Equal(t, Vec3(-12, -1.5, 6), v.DivScalar(-4))

	v.SetDiv(Vec3(2, 1, 3))
	assert.Equal(t, Vec3(24, 6, -8), v)

	v.SetDivScalar(-3)
	assert.Equal(t, Vec3(-8, -2, float32(8)/3), v)

	assert.Equal(t, Vec3(8, 2, float32(8)/3), v.Abs())

	assert.Equal(t, Vec3(-11, -2, 1), v.Min(Vec3(-11, 3, 1)))

	v.SetMin(Vec3(-11, 3, 1))
	assert.Equal(t, Vec3(-11, -2, 1), v)

	assert.Equal(t, Vec3(11, -2, 4), v.Max(Vec3(11, -3, 4)))

	v.SetMax(Vec3(11, -3, 4))
	assert.Equal(t, Vec3(11, -2, 4), v)

	v.Clamp(Vec3(1, 2, 3), Vec3(9, 5, 7))
	assert.Equal(t, Vec3(9, 2, 4), v)

	v = Vec3(3.5, 19.2, 4.8)

	assert.Equal(t, Vec3(3, 19, 4), v.Floor())
	assert.Equal(t, Vec3(4, 20, 5), v.Ceil())
	assert.Equal(t, Vec3(4, 19, 5), v.Round())

	assert.Equal(t, Vec3(-3.5, -19.2, -4.8), v.Negate())

	v = Vec3(2, 3, 4)

	assert.Equal(t, float32(1), v.Dot(Vec3(4, -5, 2)))

	assert.Equal(t, Sqrt(29), v.Length())
	assert.Equal(t, float32(29), v.LengthSquared())

	assert.Equal(t, Vec3(0.37139067, 0.557086, 0.74278134), v.Normal())

	assert.Equal(t, Sqrt(72), v.DistanceTo(Vec3(4, -5, 2)))
	assert.Equal(t, float32(72), v.DistanceToSquared(Vec3(4, -5, 2)))

	assert.Equal(t, Vec3(26, 12, -22), v.Cross(Vec3(4, -5, 2)))

	tolassert.Equal(t, 0.027681828, v.CosTo(Vec3(4, -5, 2)))

	assert.Equal(t, Vec3(14, -45, -8), v.Lerp(Vec3(4, -5, 2), 6))
}
