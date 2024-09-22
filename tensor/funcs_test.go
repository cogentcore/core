// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"math"
	"testing"

	"cogentcore.org/core/base/errors"
	"github.com/stretchr/testify/assert"
)

func abs(in, out Tensor) {
	if err := SetShapeFromMustBeValues(out, in); errors.Log(err) != nil {
		return
	}
	VectorizeThreaded(1, NFirstLen, func(idx int, tsr ...Tensor) {
		tsr[1].SetFloat1D(math.Abs(tsr[0].Float1D(idx)), idx)
	}, in, out)
}

func TestFuncs(t *testing.T) {
	err := AddFunc("Abs", abs, 1)
	assert.NoError(t, err)

	err = AddFunc("Abs", abs, 1)
	assert.Error(t, err)

	err = AddFunc("Abs3", abs, 3)
	assert.Error(t, err)

	vals := []float64{-1.507556722888818, -1.2060453783110545, -0.9045340337332908, -0.6030226891555273, -0.3015113445777635, 0, 0.3015113445777635, 0.603022689155527, 0.904534033733291, 1.2060453783110545, 1.507556722888818, .3}

	oned := NewNumberFromValues(vals...)
	oneout := oned.Clone()

	err = Call("Abs", oned, oneout)
	assert.NoError(t, err)

	assert.Equal(t, 1.507556722888818, oneout.Float1D(0))

	err = Call("Abs", oned)
	assert.Error(t, err)

	out := CallOut("Abs", oned)
	// assert.NoError(t, err)
	assert.Equal(t, AsFloat64Scalar(oneout), AsFloat64Scalar(out))
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
