// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVector2i(t *testing.T) {
	assert.Equal(t, Vector2i{5, 10}, Vec2i(5, 10))
	assert.Equal(t, Vec2i(20, 20), Vector2iScalar(20))

	v := Vector2i{}
	v.Set(-1, 7)
	assert.Equal(t, Vec2i(-1, 7), v)

	v.SetScalar(8)
	assert.Equal(t, Vec2i(8, 8), v)

	v.SetFromVector2(Vec2(8.3, 9.7))
	assert.Equal(t, Vec2i(8, 9), v)

	v.SetDim(X, -4)
	assert.Equal(t, Vec2i(-4, 9), v)

	v.SetDim(Y, 14)
	assert.Equal(t, Vec2i(-4, 14), v)

	assert.Equal(t, int32(-4), v.Dim(X))
	assert.Equal(t, int32(14), v.Dim(Y))

	v.SetZero()
	assert.Equal(t, Vec2i(0, 0), v)

	v.FromSlice([]int32{3, 2, 1}, 1)
	assert.Equal(t, Vec2i(2, 1), v)

	slice := []int32{0, 0, 0, 0, 0}
	v.ToSlice(slice, 2)
	assert.Equal(t, []int32{0, 0, 2, 1, 0}, slice)

	v = Vec2i(-2, 4)

	assert.Equal(t, Vec2i(3, 1), v.Add(Vec2i(5, -3)))
	assert.Equal(t, Vec2i(4, 10), v.AddScalar(6))

	v.SetAdd(Vec2i(2, 1))
	assert.Equal(t, Vec2i(0, 5), v)

	v.SetAddScalar(-3)
	assert.Equal(t, Vec2i(-3, 2), v)

	assert.Equal(t, Vec2i(-8, -1), v.Sub(Vec2i(5, 3)))
	assert.Equal(t, Vec2i(2, 7), v.SubScalar(-5))

	v.SetSub(Vec2i(2, 1))
	assert.Equal(t, Vec2i(-5, 1), v)

	v.SetSubScalar(3)
	assert.Equal(t, Vec2i(-8, -2), v)

	assert.Equal(t, Vec2i(-40, -6), v.Mul(Vec2i(5, 3)))
	assert.Equal(t, Vec2i(40, 10), v.MulScalar(-5))

	v.SetMul(Vec2i(2, 1))
	assert.Equal(t, Vec2i(-16, -2), v)

	v.SetMulScalar(-3)
	assert.Equal(t, Vec2i(48, 6), v)

	assert.Equal(t, Vec2i(16, 3), v.Div(Vec2i(3, 2)))
	assert.Equal(t, Vec2i(-12, -1), v.DivScalar(-4))

	v.SetDiv(Vec2i(2, 1))
	assert.Equal(t, Vec2i(24, 6), v)

	v.SetDivScalar(-3)
	assert.Equal(t, Vec2i(-8, -2), v)

	assert.Equal(t, Vec2i(-11, -2), v.Min(Vec2i(-11, 3)))

	v.SetMin(Vec2i(-11, 3))
	assert.Equal(t, Vec2i(-11, -2), v)

	assert.Equal(t, Vec2i(11, -2), v.Max(Vec2i(11, -3)))

	v.SetMax(Vec2i(11, -3))
	assert.Equal(t, Vec2i(11, -2), v)

	v.Clamp(Vec2i(1, 2), Vec2i(9, 5))
	assert.Equal(t, Vec2i(9, 2), v)

	assert.Equal(t, Vec2i(-9, -2), v.Negate())
}
