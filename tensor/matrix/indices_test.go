// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"testing"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/vector"
	"github.com/stretchr/testify/assert"
)

func TestIndices(t *testing.T) {
	assert.Equal(t, []float64{1, 0, 0, 1}, Identity(2).Values)
	assert.Equal(t, []float64{1, 0, 0, 0, 1, 0, 0, 0, 1}, Identity(3).Values)
	assert.Equal(t, []float64{0, 0, 0, 1, 0, 0, 0, 1, 0}, Identity(3, -1).Values)

	assert.Equal(t, int(vector.Sum(Identity(3)).Float1D(0)), DiagonalN(3))
	assert.Equal(t, int(vector.Sum(Identity(3, 0, 4)).Float1D(0)), DiagonalN(3, 0, 4))
	assert.Equal(t, int(vector.Sum(Identity(3, 0, 2)).Float1D(0)), DiagonalN(3, 0, 2))
	assert.Equal(t, int(vector.Sum(Identity(3, 1)).Float1D(0)), DiagonalN(3, 1))
	assert.Equal(t, int(vector.Sum(Identity(10, 4, 7)).Float1D(0)), DiagonalN(10, 4, 7))
	assert.Equal(t, int(vector.Sum(Identity(10, 4, 12)).Float1D(0)), DiagonalN(10, 4, 12))
	assert.Equal(t, int(vector.Sum(Identity(3, -1)).Float1D(0)), DiagonalN(3, -1))
	assert.Equal(t, int(vector.Sum(Identity(10, -4, 7)).Float1D(0)), DiagonalN(10, -4, 7))
	assert.Equal(t, int(vector.Sum(Identity(10, -4, 12)).Float1D(0)), DiagonalN(10, -4, 12))

	assert.Equal(t, []int{0, 0, 1, 1, 2, 2}, DiagonalIndices(3).Values)
	assert.Equal(t, []int{0, 1, 1, 2}, DiagonalIndices(3, 1).Values)
	assert.Equal(t, []int{1, 0, 2, 1}, DiagonalIndices(3, -1).Values)
	assert.Equal(t, []int{1, 0, 2, 1}, DiagonalIndices(3, -1, 4).Values)
	assert.Equal(t, []int{0, 1, 1, 2, 2, 3}, DiagonalIndices(3, 1, 4).Values)

	a := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, 10), 3, 3))
	assert.Equal(t, []float64{1, 5, 9}, tensor.Flatten(Diagonal(a)).(*tensor.Float64).Values)
	assert.Equal(t, []float64{4, 8}, tensor.Flatten(Diagonal(a, -1)).(*tensor.Float64).Values)
	assert.Equal(t, []float64{2, 6}, tensor.Flatten(Diagonal(a, 1)).(*tensor.Float64).Values)

	assert.Equal(t, 15.0, Trace(a).Float1D(0))
	assert.Equal(t, 12.0, Trace(a, -1).Float1D(0))
	assert.Equal(t, 8.0, Trace(a, 1).Float1D(0))

	assert.Equal(t, []float64{1, 0, 0, 1, 1, 0, 1, 1, 1}, Tri(3).Values)
	assert.Equal(t, []float64{1, 1, 0, 1, 1, 1, 1, 1, 1}, Tri(3, 1).Values)
	assert.Equal(t, []float64{0, 0, 0, 1, 0, 0, 1, 1, 0}, Tri(3, -1).Values)

	assert.Equal(t, []float64{0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 0, 0}, Tri(3, -1, 4).Values)
	assert.Equal(t, []float64{0, 0, 1, 0, 1, 1}, Tri(3, -1, 2).Values)

	assert.Equal(t, []float64{1, 1, 0, 0, 1, 1, 1, 0, 1, 1, 1, 1}, Tri(3, 1, 4).Values)
	assert.Equal(t, []float64{1, 1, 1, 1, 1, 1}, Tri(3, 1, 2).Values)

	assert.Equal(t, int(vector.Sum(TriUpper(3)).Float1D(0)), TriUNum(3))
	assert.Equal(t, int(vector.Sum(TriUpper(3, 0, 4)).Float1D(0)), TriUNum(3, 0, 4))
	assert.Equal(t, int(vector.Sum(TriUpper(3, 0, 2)).Float1D(0)), TriUNum(3, 0, 2))
	assert.Equal(t, int(vector.Sum(TriUpper(3, 1)).Float1D(0)), TriUNum(3, 1))
	assert.Equal(t, int(vector.Sum(TriUpper(3, 1, 4)).Float1D(0)), TriUNum(3, 1, 4))
	assert.Equal(t, int(vector.Sum(TriUpper(10, 4, 7)).Float1D(0)), TriUNum(10, 4, 7))
	assert.Equal(t, int(vector.Sum(TriUpper(10, 4, 12)).Float1D(0)), TriUNum(10, 4, 12))
	assert.Equal(t, int(vector.Sum(TriUpper(3, -1)).Float1D(0)), TriUNum(3, -1))
	assert.Equal(t, int(vector.Sum(TriUpper(3, -1, 4)).Float1D(0)), TriUNum(3, -1, 4))
	assert.Equal(t, int(vector.Sum(TriUpper(3, -1, 2)).Float1D(0)), TriUNum(3, -1, 2))
	assert.Equal(t, int(vector.Sum(TriUpper(10, -4, 7)).Float1D(0)), TriUNum(10, -4, 7))
	assert.Equal(t, int(vector.Sum(TriUpper(10, -4, 12)).Float1D(0)), TriUNum(10, -4, 12))

	assert.Equal(t, int(vector.Sum(Tri(3)).Float1D(0)), TriLNum(3))
	assert.Equal(t, int(vector.Sum(Tri(3, 0, 4)).Float1D(0)), TriLNum(3, 0, 4))
	assert.Equal(t, int(vector.Sum(Tri(3, 0, 2)).Float1D(0)), TriLNum(3, 0, 2))
	assert.Equal(t, int(vector.Sum(Tri(3, 1)).Float1D(0)), TriLNum(3, 1))
	assert.Equal(t, int(vector.Sum(Tri(3, 1, 4)).Float1D(0)), TriLNum(3, 1, 4))
	assert.Equal(t, int(vector.Sum(Tri(10, 4, 7)).Float1D(0)), TriLNum(10, 4, 7))
	assert.Equal(t, int(vector.Sum(Tri(10, 4, 12)).Float1D(0)), TriLNum(10, 4, 12))
	assert.Equal(t, int(vector.Sum(Tri(3, -1)).Float1D(0)), TriLNum(3, -1))
	assert.Equal(t, int(vector.Sum(Tri(3, -1, 4)).Float1D(0)), TriLNum(3, -1, 4))
	assert.Equal(t, int(vector.Sum(Tri(3, -1, 2)).Float1D(0)), TriLNum(3, -1, 2))
	assert.Equal(t, int(vector.Sum(Tri(10, -4, 7)).Float1D(0)), TriLNum(10, -4, 7))
	assert.Equal(t, int(vector.Sum(Tri(10, -4, 12)).Float1D(0)), TriLNum(10, -4, 12))

	tli := TriLIndicies(3)
	assert.Equal(t, []int{0, 0, 1, 0, 1, 1, 2, 0, 2, 1, 2, 2}, tli.Values)

	tli = TriLIndicies(3, -1)
	assert.Equal(t, []int{1, 0, 2, 0, 2, 1}, tli.Values)

	tli = TriLIndicies(3, 1)
	assert.Equal(t, []int{0, 0, 0, 1, 1, 0, 1, 1, 1, 2, 2, 0, 2, 1, 2, 2}, tli.Values)

	tli = TriUIndicies(3, 1)
	assert.Equal(t, []int{0, 1, 0, 2, 1, 2}, tli.Values)

	tli = TriUIndicies(3, -1)
	assert.Equal(t, []int{0, 0, 0, 1, 0, 2, 1, 0, 1, 1, 1, 2, 2, 1, 2, 2}, tli.Values)

	tf := tensor.NewFloat64Ones(3, 4)

	assert.Equal(t, Tri(3, -1, 4).Values, TriL(tf, -1).(*tensor.Float64).Values)
	assert.Equal(t, Tri(3, 0, 4).Values, TriL(tf).(*tensor.Float64).Values)
	assert.Equal(t, Tri(3, 1, 4).Values, TriL(tf, 1).(*tensor.Float64).Values)

	assert.Equal(t, TriUpper(3, -1, 4).Values, TriU(tf, -1).(*tensor.Float64).Values)
	assert.Equal(t, TriUpper(3, 0, 4).Values, TriU(tf).(*tensor.Float64).Values)
	assert.Equal(t, TriUpper(3, 1, 4).Values, TriU(tf, 1).(*tensor.Float64).Values)
}
