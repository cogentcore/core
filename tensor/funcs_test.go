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

	_, err = FuncByName("Abs")
	assert.NoError(t, err)

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
