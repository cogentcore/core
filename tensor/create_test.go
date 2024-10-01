// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	assert.Equal(t, 5.5, NewFloat64Scalar(5.5).Float1D(0))
	assert.Equal(t, 5, NewIntScalar(5).Int1D(0))
	assert.Equal(t, "test", NewStringScalar("test").String1D(0))

	assert.Equal(t, []float64{5.5, 1.5}, NewFloat64FromValues(5.5, 1.5).Values)
	assert.Equal(t, []int{5, 1}, NewIntFromValues(5, 1).Values)
	assert.Equal(t, []string{"test1", "test2"}, NewStringFromValues("test1", "test2").Values)

	assert.Equal(t, []float64{5.5, 5.5, 5.5, 5.5}, NewFloat64Full(5.5, 2, 2).Values)
	assert.Equal(t, []float64{1, 1, 1, 1}, NewFloat64Ones(2, 2).Values)

	ar := NewIntRange(5)
	assert.Equal(t, []int{0, 1, 2, 3, 4}, AsIntSlice(ar))

	ar = NewIntRange(2, 5)
	assert.Equal(t, []int{2, 3, 4}, AsIntSlice(ar))

	ar = NewIntRange(0, 5, 2)
	assert.Equal(t, []int{0, 2, 4}, AsIntSlice(ar))

	lr := NewFloat64SpacedLinear(NewFloat64Scalar(0), NewFloat64Scalar(5), 6, true)
	assert.Equal(t, []float64{0, 1, 2, 3, 4, 5}, AsFloat64Slice(lr))

	lr = NewFloat64SpacedLinear(NewFloat64Scalar(0), NewFloat64Scalar(5), 5, false)
	assert.Equal(t, []float64{0, 1, 2, 3, 4}, AsFloat64Slice(lr))

	lr2 := NewFloat64SpacedLinear(NewFloat64FromValues(0, 2), NewFloat64FromValues(5, 7), 5, false)
	// fmt.Println(lr2)
	assert.Equal(t, []float64{0, 2, 1, 3, 2, 4, 3, 5, 4, 6}, AsFloat64Slice(lr2))
}
