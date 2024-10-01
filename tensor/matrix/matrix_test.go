// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"testing"

	"cogentcore.org/core/tensor"
	"github.com/stretchr/testify/assert"
)

func TestMatrix(t *testing.T) {
	a := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, 5), 2, 2))
	// fmt.Println(a)

	v := tensor.NewFloat64FromValues(2, 3)
	_ = v

	o := Mul(a, a).(*tensor.Float64)
	// fmt.Println(o)

	assert.Equal(t, []float64{7, 10, 15, 22}, o.Values)

	o = Mul(a, v).(*tensor.Float64)
	// fmt.Println(o)

	assert.Equal(t, []float64{8, 18}, o.Values)
	assert.Equal(t, []int{2, 1}, o.Shape().Sizes)

	o = Mul(v, a).(*tensor.Float64)
	// fmt.Println(o)

	assert.Equal(t, []float64{11, 16}, o.Values)
	assert.Equal(t, []int{1, 2}, o.Shape().Sizes)

	nr := 3
	b := tensor.NewFloat64(nr, 2, 2)
	for r := range nr {
		b.SetRowTensor(a, r)
	}
	// fmt.Println(b)

	o = Mul(b, a).(*tensor.Float64)
	// fmt.Println(o)
	assert.Equal(t, []float64{7, 10, 15, 22, 7, 10, 15, 22, 7, 10, 15, 22}, o.Values)
	assert.Equal(t, []int{3, 2, 2}, o.Shape().Sizes)

	o = Mul(a, b).(*tensor.Float64)
	// fmt.Println(o)
	assert.Equal(t, []float64{7, 10, 15, 22, 7, 10, 15, 22, 7, 10, 15, 22}, o.Values)
	assert.Equal(t, []int{3, 2, 2}, o.Shape().Sizes)

	o = Mul(b, b).(*tensor.Float64)
	// fmt.Println(o)
	assert.Equal(t, []float64{7, 10, 15, 22, 7, 10, 15, 22, 7, 10, 15, 22}, o.Values)
	assert.Equal(t, []int{3, 2, 2}, o.Shape().Sizes)

	o = Mul(v, b).(*tensor.Float64)
	// fmt.Println(o)
	assert.Equal(t, []float64{11, 16, 11, 16, 11, 16}, o.Values)
	assert.Equal(t, []int{3, 1, 2}, o.Shape().Sizes)

	o = Mul(b, v).(*tensor.Float64)
	// fmt.Println(o)
	assert.Equal(t, []float64{8, 18, 8, 18, 8, 18}, o.Values)
	assert.Equal(t, []int{3, 2, 1}, o.Shape().Sizes)

}
