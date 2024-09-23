// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"testing"

	"cogentcore.org/core/tensor"
	"github.com/stretchr/testify/assert"
)

func TestOpsCall(t *testing.T) {
	x := tensor.NewIntScalar(1)
	y := tensor.NewIntScalar(4)

	a := tensor.CallOut("Mul", x, tensor.NewIntScalar(2))
	b := tensor.CallOut("Add", x, y)
	c := tensor.CallOut("Add", tensor.CallOut("Mul", x, y), tensor.CallOut("Mul", a, b))

	assert.Equal(t, 14.0, c.Float1D(0))
}

func TestOps(t *testing.T) {
	scalar := tensor.NewFloat64Scalar(-5.5)
	scb := scalar.Clone()
	scb.SetFloat1D(-4.0, 0)
	scout := scalar.Clone()

	vals := []float64{-1.507556722888818, -1.2060453783110545, -0.9045340337332908, -0.6030226891555273, -0.3015113445777635, 0.1, 0.3015113445777635, 0.603022689155527, 0.904534033733291, 1.2060453783110545, 1.507556722888818, .3}

	oned := tensor.NewNumberFromValues(vals...)
	oneout := oned.Clone()

	cell2d := tensor.NewFloat32(5, 12)
	_, cells := cell2d.Shape().RowCellSize()
	assert.Equal(t, cells, 12)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		ci := idx % cells
		cell2d.SetFloat1D(oned.Float1D(ci), idx)
	}, cell2d)
	// cell2d.DeleteRows(3, 1)
	cellout := cell2d.Clone()
	_ = cellout

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
			assert.InDelta(t, v+v, cellout.FloatRowCell(ri, i), 1.0e-6)
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
			assert.InDelta(t, v-v, cellout.FloatRowCell(ri, i), 1.0e-6)
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
			assert.InDelta(t, v*v, cellout.FloatRowCell(ri, i), 1.0e-6)
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
			assert.InDelta(t, v/v, cellout.FloatRowCell(ri, i), 1.0e-6)
		}
	}

}
