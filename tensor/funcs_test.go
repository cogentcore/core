// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// prototype of a simple compute function:
func absout(in Tensor, out Values) error {
	SetShapeFrom(out, in)
	VectorizeThreaded(1, NFirstLen, func(idx int, tsr ...Tensor) {
		tsr[1].SetFloat1D(math.Abs(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func TestFuncs(t *testing.T) {
	err := AddFunc("Abs", absout)
	assert.NoError(t, err)

	err = AddFunc("Abs", absout)
	assert.Error(t, err) // already

	vals := []float64{-1.507556722888818, -1.2060453783110545, -0.9045340337332908, -0.6030226891555273, -0.3015113445777635, 0, 0.3015113445777635, 0.603022689155527, 0.904534033733291, 1.2060453783110545, 1.507556722888818, .3}

	oned := NewNumberFromValues(vals...)
	oneout := oned.Clone()

	fn, err := FuncByName("Abs")
	assert.NoError(t, err)

	// fmt.Println(fn.Args[0], fn.Args[1])
	assert.Equal(t, true, fn.Args[0].IsTensor)
	assert.Equal(t, true, fn.Args[1].IsTensor)
	assert.Equal(t, false, fn.Args[0].IsInt)
	assert.Equal(t, false, fn.Args[1].IsInt)

	absout(oned, oneout)
	assert.Equal(t, 1.507556722888818, oneout.Float1D(0))
}

func TestAlign(t *testing.T) {
	a := NewFloat64(3, 4)
	b := NewFloat64(1, 3, 4)
	as, bs, os, err := AlignShapes(a, b)
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 3, 4}, os.Sizes)
	assert.Equal(t, []int{1, 3, 4}, as.Sizes)
	assert.Equal(t, []int{1, 3, 4}, bs.Sizes)

	ars := NewReshaped(a, 12)
	as, bs, os, err = AlignShapes(ars, b)
	assert.Error(t, err)

	brs := NewReshaped(b, 12)
	as, bs, os, err = AlignShapes(ars, brs)
	assert.NoError(t, err)

	ars = NewReshaped(a, 3, 1, 4)
	as, bs, os, err = AlignShapes(ars, b)
	assert.NoError(t, err)
	assert.Equal(t, []int{3, 3, 4}, os.Sizes)
	assert.Equal(t, []int{3, 1, 4}, as.Sizes)
	assert.Equal(t, []int{1, 3, 4}, bs.Sizes)
}

func TestCreate(t *testing.T) {
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
