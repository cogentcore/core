// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"testing"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/stats"
	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	scalar := tensor.NewFloat64Scalar(-5.5)
	scb := scalar.Clone()
	scb.SetFloat1D(-4.0, 0)
	scout := scalar.Clone()

	vals := []float64{-1.507556722888818, -1.2060453783110545, -0.9045340337332908, -0.6030226891555273, -0.3015113445777635, 0.1, 0.3015113445777635, 0.603022689155527, 0.904534033733291, 1.2060453783110545, 1.507556722888818, .3}

	oned := tensor.NewIndexed(tensor.NewNumberFromSlice(vals...))
	oneout := oned.Clone()

	cell2d := tensor.NewIndexed(tensor.NewFloat32(5, 2, 6))
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, ci := cell2d.RowCellIndex(idx)
		cell2d.SetFloat1D(oned.Float1D(ci), i)
	}, cell2d)
	// cell2d.DeleteRows(3, 1)
	cellout := cell2d.Clone()

	Add(scalar, scb, scout)
	assert.Equal(t, -5.5+-4, scout.Float1D(0))

	Add(scalar, oned, oneout)
	for i, v := range vals {
		assert.Equal(t, v+-5.5, oneout.Float1D(i))
	}

	Add(oned, oned, oneout)
	for i, v := range vals {
		assert.Equal(t, v+v, oneout.Float1D(i))
	}

	Add(cell2d, oned, cellout)
	for ri := range 5 {
		for i, v := range vals {
			assert.InDelta(t, v+v, cellout.Tensor.FloatRowCell(ri, i), 1.0e-6)
		}
	}

	Sub(scalar, scb, scout)
	assert.Equal(t, -5.5 - -4, scout.Float1D(0))

	Sub(scb, scalar, scout)
	assert.Equal(t, -4 - -5.5, scout.Float1D(0))

	Sub(scalar, oned, oneout)
	for i, v := range vals {
		assert.Equal(t, -5.5-v, oneout.Float1D(i))
	}

	Sub(oned, scalar, oneout)
	for i, v := range vals {
		assert.Equal(t, v - -5.5, oneout.Float1D(i))
	}

	Sub(oned, oned, oneout)
	for i, v := range vals {
		assert.Equal(t, v-v, oneout.Float1D(i))
	}

	Sub(cell2d, oned, cellout)
	for ri := range 5 {
		for i, v := range vals {
			assert.InDelta(t, v-v, cellout.Tensor.FloatRowCell(ri, i), 1.0e-6)
		}
	}

	Mul(scalar, scb, scout)
	assert.Equal(t, -5.5*-4, scout.Float1D(0))

	Mul(scalar, oned, oneout)
	for i, v := range vals {
		assert.Equal(t, v*-5.5, oneout.Float1D(i))
	}

	Mul(oned, oned, oneout)
	for i, v := range vals {
		assert.Equal(t, v*v, oneout.Float1D(i))
	}

	Mul(cell2d, oned, cellout)
	for ri := range 5 {
		for i, v := range vals {
			assert.InDelta(t, v*v, cellout.Tensor.FloatRowCell(ri, i), 1.0e-6)
		}
	}

	Div(scalar, scb, scout)
	assert.Equal(t, -5.5/-4, scout.Float1D(0))

	Div(scb, scalar, scout)
	assert.Equal(t, -4/-5.5, scout.Float1D(0))

	Div(scalar, oned, oneout)
	for i, v := range vals {
		assert.Equal(t, -5.5/v, oneout.Float1D(i))
	}

	Div(oned, scalar, oneout)
	for i, v := range vals {
		assert.Equal(t, v/-5.5, oneout.Float1D(i))
	}

	Div(oned, oned, oneout)
	for i, v := range vals {
		assert.Equal(t, v/v, oneout.Float1D(i))
	}

	Div(cell2d, oned, cellout)
	for ri := range 5 {
		for i, v := range vals {
			assert.InDelta(t, v/v, cellout.Tensor.FloatRowCell(ri, i), 1.0e-6)
		}
	}

	ZScore(oned, oneout)
	mout := tensor.NewIndexed(tensor.NewFloat64())
	std, mean, _ := stats.StdFuncOut64(oneout, mout)
	assert.InDelta(t, 1.0, std.Float1D(0), 1.0e-6)
	assert.InDelta(t, 0.0, mean.Float1D(0), 1.0e-6)

	UnitNorm(oned, oneout)
	stats.MinFunc(oneout, mout)
	assert.InDelta(t, 0.0, mout.Float1D(0), 1.0e-6)
	stats.MaxFunc(oneout, mout)
	assert.InDelta(t, 1.0, mout.Float1D(0), 1.0e-6)
	// fmt.Println(oneout.Tensor)

	minv := tensor.NewFloat64Scalar(0)
	maxv := tensor.NewFloat64Scalar(1)
	Clamp(oned, minv, maxv, oneout)
	stats.MinFunc(oneout, mout)
	assert.InDelta(t, 0.0, mout.Float1D(0), 1.0e-6)
	stats.MaxFunc(oneout, mout)
	assert.InDelta(t, 1.0, mout.Float1D(0), 1.0e-6)
	// fmt.Println(oneout.Tensor)

	thr := tensor.NewFloat64Scalar(0.5)
	Binarize(oned, thr, oneout)
	stats.MinFunc(oneout, mout)
	assert.InDelta(t, 0.0, mout.Float1D(0), 1.0e-6)
	stats.MaxFunc(oneout, mout)
	assert.InDelta(t, 1.0, mout.Float1D(0), 1.0e-6)
	// fmt.Println(oneout.Tensor)
}
