// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"cogentcore.org/core/tensor"
	"github.com/stretchr/testify/assert"
)

func TestMatrix(t *testing.T) {
	a := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, 5), 2, 2))
	// fmt.Println(a)

	v := tensor.NewFloat64FromValues(2, 3)
	_ = v

	o := Mul(a, a)
	assert.Equal(t, []float64{7, 10, 15, 22}, o.Values)

	o = Mul(a, v)
	assert.Equal(t, []float64{8, 18}, o.Values)
	assert.Equal(t, []int{2}, o.Shape().Sizes)

	o = Mul(v, a)
	assert.Equal(t, []float64{11, 16}, o.Values)
	assert.Equal(t, []int{2}, o.Shape().Sizes)

	nr := 3
	b := tensor.NewFloat64(nr, 1, 2, 2)
	for r := range nr {
		b.SetRowTensor(a, r)
	}
	// fmt.Println(b)

	o = Mul(b, a)
	assert.Equal(t, []float64{7, 10, 15, 22, 7, 10, 15, 22, 7, 10, 15, 22}, o.Values)
	assert.Equal(t, []int{3, 2, 2}, o.Shape().Sizes)

	o = Mul(a, b)
	assert.Equal(t, []float64{7, 10, 15, 22, 7, 10, 15, 22, 7, 10, 15, 22}, o.Values)
	assert.Equal(t, []int{3, 2, 2}, o.Shape().Sizes)

	o = Mul(b, b)
	assert.Equal(t, []float64{7, 10, 15, 22, 7, 10, 15, 22, 7, 10, 15, 22}, o.Values)
	assert.Equal(t, []int{3, 2, 2}, o.Shape().Sizes)

	o = Mul(v, b)
	assert.Equal(t, []float64{11, 16, 11, 16, 11, 16}, o.Values)
	assert.Equal(t, []int{3, 2}, o.Shape().Sizes)

	o = Mul(b, v)
	assert.Equal(t, []float64{8, 18, 8, 18, 8, 18}, o.Values)
	assert.Equal(t, []int{3, 2}, o.Shape().Sizes)

	o = Mul(a, tensor.Transpose(a))
	assert.Equal(t, []float64{5, 11, 11, 25}, o.Values)

	d := Det(a)
	assert.Equal(t, -2.0, d.Float1D(0))

	inv := Inverse(a)
	tolassert.EqualTolSlice(t, []float64{-2, 1, 1.5, -0.5}, inv.Values, 1.0e-8)

	inv = Inverse(b)
	tolassert.EqualTolSlice(t, []float64{-2, 1, 1.5, -0.5, -2, 1, 1.5, -0.5, -2, 1, 1.5, -0.5}, inv.Values, 1.0e-8)
}
