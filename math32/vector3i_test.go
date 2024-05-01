// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVector3i(t *testing.T) {
	assert.Equal(t, Vector3i{5, 10, 7}, Vec3i(5, 10, 7))
	assert.Equal(t, Vec3i(20, 20, 20), Vector3iScalar(20))

	v := Vector3i{}
	v.Set(-1, 7, 12)
	assert.Equal(t, Vec3i(-1, 7, 12), v)

	v.SetScalar(8)
	assert.Equal(t, Vec3i(8, 8, 8), v)

	v.SetFromVector3(Vec3(8, 9, -7))
	assert.Equal(t, Vec3i(8, 9, -7), v)

	v.SetDim(X, -4)
	assert.Equal(t, Vec3i(-4, 9, -7), v)

	v.SetDim(Y, 14)
	assert.Equal(t, Vec3i(-4, 14, -7), v)

	v.SetDim(Z, 8)
	assert.Equal(t, Vec3i(-4, 14, 8), v)

	assert.Equal(t, int32(-4), v.Dim(X))
	assert.Equal(t, int32(14), v.Dim(Y))
	assert.Equal(t, int32(8), v.Dim(Z))

	v.SetZero()
	assert.Equal(t, Vec3i(0, 0, 0), v)

	v.FromSlice([]int32{3, 2, 1, 4}, 1)
	assert.Equal(t, Vec3i(2, 1, 4), v)

	slice := []int32{0, 0, 0, 0, 0, 0}
	v.ToSlice(slice, 2)
	assert.Equal(t, []int32{0, 0, 2, 1, 4, 0}, slice)

	v = Vec3i(-2, 4, 5)

	assert.Equal(t, Vec3i(3, 1, 7), v.Add(Vec3i(5, -3, 2)))
	assert.Equal(t, Vec3i(4, 10, 11), v.AddScalar(6))

	v.SetAdd(Vec3i(2, 1, 4))
	assert.Equal(t, Vec3i(0, 5, 9), v)

	v.SetAddScalar(-3)
	assert.Equal(t, Vec3i(-3, 2, 6), v)

	assert.Equal(t, Vec3i(-8, -1, 2), v.Sub(Vec3i(5, 3, 4)))
	assert.Equal(t, Vec3i(2, 7, 11), v.SubScalar(-5))

	v.SetSub(Vec3i(2, 1, 5))
	assert.Equal(t, Vec3i(-5, 1, 1), v)

	v.SetSubScalar(3)
	assert.Equal(t, Vec3i(-8, -2, -2), v)

	assert.Equal(t, Vec3i(-40, -6, -8), v.Mul(Vec3i(5, 3, 4)))
	assert.Equal(t, Vec3i(40, 10, 10), v.MulScalar(-5))

	v.SetMul(Vec3i(2, 1, -4))
	assert.Equal(t, Vec3i(-16, -2, 8), v)

	v.SetMulScalar(-3)
	assert.Equal(t, Vec3i(48, 6, -24), v)

	assert.Equal(t, Vec3i(16, 3, 12), v.Div(Vec3i(3, 2, -2)))
	assert.Equal(t, Vec3i(-12, -1, 6), v.DivScalar(-4))

	v.SetDiv(Vec3i(2, 1, 3))
	assert.Equal(t, Vec3i(24, 6, -8), v)

	v.SetDivScalar(-3)
	assert.Equal(t, Vec3i(-8, -2, 8/3), v)

	assert.Equal(t, Vec3i(-11, -2, 1), v.Min(Vec3i(-11, 3, 1)))

	v.SetMin(Vec3i(-11, 3, 1))
	assert.Equal(t, Vec3i(-11, -2, 1), v)

	assert.Equal(t, Vec3i(11, -2, 4), v.Max(Vec3i(11, -3, 4)))

	v.SetMax(Vec3i(11, -3, 4))
	assert.Equal(t, Vec3i(11, -2, 4), v)

	v.Clamp(Vec3i(1, 2, 3), Vec3i(9, 5, 7))
	assert.Equal(t, Vec3i(9, 2, 4), v)

	assert.Equal(t, Vec3i(-9, -2, -4), v.Negate())
}
