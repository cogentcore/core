// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVector4(t *testing.T) {
	assert.Equal(t, Vector4{5, 10, 7, 3}, Vec4(5, 10, 7, 3))
	assert.Equal(t, Vec4(20, 20, 20, 20), Vector4Scalar(20))
	assert.Equal(t, Vec4(15, -5, 3, 5), Vector4FromVector3(Vec3(15, -5, 3), 5))

	v := Vector4{}
	v.Set(-1, 7, 12, 2.1)
	assert.Equal(t, Vec4(-1, 7, 12, 2.1), v)

	v.SetScalar(8.12)
	assert.Equal(t, Vec4(8.12, 8.12, 8.12, 8.12), v)

	v.SetDim(X, -4)
	assert.Equal(t, Vec4(-4, 8.12, 8.12, 8.12), v)

	v.SetDim(Y, 14.3)
	assert.Equal(t, Vec4(-4, 14.3, 8.12, 8.12), v)

	v.SetDim(Z, 3.14)
	assert.Equal(t, Vec4(-4, 14.3, 3.14, 8.12), v)

	v.SetDim(W, -21)
	assert.Equal(t, Vec4(-4, 14.3, 3.14, -21), v)

	assert.Equal(t, float32(-4), v.Dim(X))
	assert.Equal(t, float32(14.3), v.Dim(Y))
	assert.Equal(t, float32(3.14), v.Dim(Z))
	assert.Equal(t, float32(-21), v.Dim(W))

	v.SetZero()
	assert.Equal(t, Vec4(0, 0, 0, 1), v)

	v.FromSlice([]float32{3, 2, 1, 4, 5}, 1)
	assert.Equal(t, Vec4(2, 1, 4, 5), v)

	slice := []float32{0, 0, 0, 0, 0, 0, 0}
	v.ToSlice(slice, 2)
	assert.Equal(t, []float32{0, 0, 2, 1, 4, 5, 0}, slice)

	v = Vec4(-2, 4, 5, 1)

	assert.Equal(t, Vec4(3, 1, 7, 4.5), v.Add(Vec4(5, -3, 2, 3.5)))
	assert.Equal(t, Vec4(4, 10, 11, 7), v.AddScalar(6))

	v.SetAdd(Vec4(2, 1, 4, 3))
	assert.Equal(t, Vec4(0, 5, 9, 4), v)

	v.SetAddScalar(-3)
	assert.Equal(t, Vec4(-3, 2, 6, 1), v)

	assert.Equal(t, Vec4(-8, -1, 2, 3), v.Sub(Vec4(5, 3, 4, -2)))
	assert.Equal(t, Vec4(2, 7, 11, 6), v.SubScalar(-5))

	v.SetSub(Vec4(2, 1, 5, -4))
	assert.Equal(t, Vec4(-5, 1, 1, 5), v)

	v.SetSubScalar(3)
	assert.Equal(t, Vec4(-8, -2, -2, 2), v)

	assert.Equal(t, Vec4(-40, -6, -8, 4), v.Mul(Vec4(5, 3, 4, 2)))
	assert.Equal(t, Vec4(40, 10, 10, -10), v.MulScalar(-5))

	v.SetMul(Vec4(2, 1, -4, 3))
	assert.Equal(t, Vec4(-16, -2, 8, 6), v)

	v.SetMulScalar(-3)
	assert.Equal(t, Vec4(48, 6, -24, -18), v)

	assert.Equal(t, Vec4(16, 3, 12, 6), v.Div(Vec4(3, 2, -2, -3)))
	assert.Equal(t, Vec4(-12, -1.5, 6, 4.5), v.DivScalar(-4))

	v.SetDiv(Vec4(2, 1, 3, 2))
	assert.Equal(t, Vec4(24, 6, -8, -9), v)

	v.SetDivScalar(-3)
	assert.Equal(t, Vec4(-8, -2, float32(8)/3, 3), v)

	assert.Equal(t, Vec4(-11, -2, 1, 2), v.Min(Vec4(-11, 3, 1, 2)))

	v.SetMin(Vec4(-11, 3, 1, 2))
	assert.Equal(t, Vec4(-11, -2, 1, 2), v)

	assert.Equal(t, Vec4(11, -2, 4, 7), v.Max(Vec4(11, -3, 4, 7)))

	v.SetMax(Vec4(11, -3, 4, 7))
	assert.Equal(t, Vec4(11, -2, 4, 7), v)

	v.Clamp(Vec4(1, 2, 3, 5), Vec4(9, 5, 7, 6))
	assert.Equal(t, Vec4(9, 2, 4, 6), v)

	v = Vec4(3.5, 19.2, 4.8, -3.1)

	assert.Equal(t, Vec4(3, 19, 4, -4), v.Floor())
	assert.Equal(t, Vec4(4, 20, 5, -3), v.Ceil())
	assert.Equal(t, Vec4(4, 19, 5, -3), v.Round())

	assert.Equal(t, Vec4(-3.5, -19.2, -4.8, 3.1), v.Negate())

	v = Vec4(2, 3, 4, 1)

	assert.Equal(t, float32(4), v.Dot(Vec4(4, -5, 2, 3)))

	assert.Equal(t, Sqrt(30), v.Length())
	assert.Equal(t, float32(30), v.LengthSquared())

	assert.Equal(t, Vec4(0.36514837, 0.5477226, 0.73029673, 0.18257418), v.Normal())

	assert.Equal(t, Vec4(14, -45, -8, 13), v.Lerp(Vec4(4, -5, 2, 3), 6))
}
