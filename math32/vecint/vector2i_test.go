// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vecint

import (
	"testing"

	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

// TestVector2i tests making, converting, and doing math with vectors.
func TestVector2i(t *testing.T) {
	v := Vec2i(6, 8)
	assert.Equal(t, Vector2i{2, 2}, Vector2iScalar(2))
	f := math32.Vec2(1.5, -1.5)
	assert.Equal(t, Vec2i(2, -2), Vector2iFromVector2Round(f))
	assert.Equal(t, Vec2i(1, -2), Vector2iFromVector2Floor(f))
	assert.Equal(t, Vec2i(2, -1), Vector2iFromVector2Ceil(f))
	assert.Equal(t, math32.Vec2(6, 8), v.ToVector2())

	w := v
	w.Set(1, 2)
	assert.Equal(t, Vec2i(1, 2), w)
	w.SetScalar(3)
	assert.Equal(t, Vec2i(3, 3), w)
	w.SetDim(math32.Y, 5)
	assert.Equal(t, 5, w.Dim(math32.Y))
	w.SetDim(math32.X, 4)
	assert.Equal(t, 4, w.Dim(math32.X))
	w.SetZero()
	assert.Equal(t, Vector2i{}, w)
	a := []int{0, 7, 9}
	w.FromSlice(a, 1)
	assert.Equal(t, Vec2i(7, 9), w)
	b := make([]int, 3)
	w.ToSlice(b, 1)
	assert.Equal(t, []int{0, 7, 9}, b)

	o := Vec2i(2, 4)
	assert.Equal(t, Vec2i(8, 12), v.Add(o))
	assert.Equal(t, Vec2i(7, 9), v.AddScalar(1))
	assert.Equal(t, Vec2i(4, 4), v.Sub(o))
	assert.Equal(t, Vec2i(5, 7), v.SubScalar(1))
	assert.Equal(t, Vec2i(12, 32), v.Mul(o))
	assert.Equal(t, Vec2i(12, 16), v.MulScalar(2))
	assert.Equal(t, Vec2i(3, 2), v.Div(o))
	assert.Equal(t, Vec2i(3, 4), v.DivScalar(2))
	assert.Equal(t, Vec2i(0, 0), v.DivScalar(0), "division by zero is zero")
	assert.Equal(t, Vec2i(2, 4), v.Min(o))
	assert.Equal(t, Vec2i(6, 8), v.Max(o))
	assert.Equal(t, Vec2i(-6, -8), v.Negate())

	w = v
	w.SetAdd(o)
	assert.Equal(t, v.Add(o), w)
	w = v
	w.SetAddScalar(1)
	assert.Equal(t, v.AddScalar(1), w)
	w = v
	w.SetSub(o)
	assert.Equal(t, v.Sub(o), w)
	w = v
	w.SetSubScalar(1)
	assert.Equal(t, v.SubScalar(1), w)
	w = v
	w.SetMul(o)
	assert.Equal(t, v.Mul(o), w)
	w = v
	w.SetMulScalar(2)
	assert.Equal(t, v.MulScalar(2), w)
	w = v
	w.SetDiv(o)
	assert.Equal(t, v.Div(o), w)
	w = v
	w.SetDivScalar(2)
	assert.Equal(t, v.DivScalar(2), w)
	w = v
	w.SetDivScalar(0)
	assert.Equal(t, Vector2i{}, w)
	w = v
	w.SetMin(o)
	assert.Equal(t, v.Min(o), w)
	w = v
	w.SetMax(o)
	assert.Equal(t, v.Max(o), w)

	w = Vec2i(-1, 20)
	w.Clamp(Vec2i(0, 0), Vec2i(10, 10))
	assert.Equal(t, Vec2i(0, 10), w)
	w = Vec2i(20, -1)
	w.Clamp(Vec2i(0, 0), Vec2i(10, 10))
	assert.Equal(t, Vec2i(10, 0), w)
}
