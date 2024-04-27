// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package norm

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"cogentcore.org/core/tensor"
	"github.com/stretchr/testify/assert"
)

func TestNorm32(t *testing.T) {
	vals := []float32{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}

	zn := []float32{-1.5075567, -1.2060454, -0.90453404, -0.60302263, -0.30151132, 0, 0.3015114, 0.60302263, 0.90453404, 1.2060453, 1.5075567}
	nvals := make([]float32, len(vals))
	copy(nvals, vals)
	ZScore32(nvals)
	assert.Equal(t, zn, nvals)

	copy(nvals, vals)
	Unit32(nvals)
	assert.Equal(t, vals, nvals)

	tn := []float32{0.2, 0.2, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.8, 0.8}
	copy(nvals, vals)
	Thresh32(nvals, true, 0.8, true, 0.2)
	assert.Equal(t, tn, nvals)

	bn := []float32{0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1}
	copy(nvals, vals)
	Binarize32(nvals, 0.5, 1.0, 0.0)
	assert.Equal(t, bn, nvals)

	tsr := tensor.New[float32]([]int{11}).(*tensor.Float32)
	copy(tsr.Values, vals)
	TensorZScore(tsr, 0)
	tolassert.EqualTolSlice(t, zn, tsr.Values, 1.0e-6)

	copy(tsr.Values, vals)
	TensorUnit(tsr, 0)
	tolassert.EqualTolSlice(t, vals, tsr.Values, 1.0e-6)

}

func TestNorm64(t *testing.T) {
	vals := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}

	zn := []float64{-1.507556722888818, -1.2060453783110545, -0.9045340337332908, -0.6030226891555273, -0.3015113445777635, 0, 0.3015113445777635, 0.603022689155527, 0.904534033733291, 1.2060453783110545, 1.507556722888818}
	nvals := make([]float64, len(vals))
	copy(nvals, vals)
	ZScore64(nvals)
	assert.Equal(t, zn, nvals)

	copy(nvals, vals)
	Unit64(nvals)
	assert.Equal(t, vals, nvals)

	tn := []float64{0.2, 0.2, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.8, 0.8}
	copy(nvals, vals)
	Thresh64(nvals, true, 0.8, true, 0.2)
	assert.Equal(t, tn, nvals)

	bn := []float64{0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1}
	copy(nvals, vals)
	Binarize64(nvals, 0.5, 1.0, 0.0)
	assert.Equal(t, bn, nvals)

	tsr := tensor.New[float64]([]int{11}).(*tensor.Float64)
	copy(tsr.Values, vals)
	TensorZScore(tsr, 0)
	tolassert.EqualTolSlice(t, zn, tsr.Values, 1.0e-6)

	copy(tsr.Values, vals)
	TensorUnit(tsr, 0)
	tolassert.EqualTolSlice(t, vals, tsr.Values, 1.0e-6)

}
