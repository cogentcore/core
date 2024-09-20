// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"math"
	"testing"

	"cogentcore.org/core/tensor"
	"github.com/stretchr/testify/assert"
)

type onef func(x float64) float64
type tonef func(in, out tensor.Tensor)

// Equal does equal testing taking into account NaN
func Equal(t *testing.T, trg, val float64) {
	if math.IsNaN(trg) {
		if !math.IsNaN(val) {
			t.Error("target is NaN but actual is not")
		}
		return
	}
	assert.InDelta(t, trg, val, 1.0e-4)
}

func TestMath(t *testing.T) {
	scalar := tensor.NewFloat64Scalar(-5.5)
	scout := scalar.Clone()

	vals := []float64{-1.507556722888818, -1.2060453783110545, -0.9045340337332908, -0.6030226891555273, -0.3015113445777635, 0, 0.3015113445777635, 0.603022689155527, 0.904534033733291, 1.2060453783110545, 1.507556722888818, .3}

	oned := tensor.NewNumberFromSlice(vals...)
	oneout := oned.Clone()

	cell2d := tensor.NewFloat32(5, 2, 6)
	_, cells := cell2d.RowCellSize()
	assert.Equal(t, cells, 12)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		ci := idx % cells
		cell2d.SetFloat1D(oned.Float1D(ci), idx)
	}, cell2d)
	cellout := cell2d.Clone()

	mfuncs := []onef{math.Abs, math.Acos, math.Acosh, math.Asin, math.Asinh, math.Atan, math.Atanh, math.Cbrt, math.Ceil, math.Cos, math.Cosh, math.Erf, math.Erfc, math.Erfcinv, math.Erfinv, math.Exp, math.Exp2, math.Expm1, math.Floor, math.Gamma, math.J0, math.J1, math.Log, math.Log10, math.Log1p, math.Log2, math.Logb, math.Round, math.RoundToEven, math.Sin, math.Sinh, math.Sqrt, math.Tan, math.Tanh, math.Trunc, math.Y0, math.Y1}
	tfuncs := []tonef{Abs, Acos, Acosh, Asin, Asinh, Atan, Atanh, Cbrt, Ceil, Cos, Cosh, Erf, Erfc, Erfcinv, Erfinv, Exp, Exp2, Expm1, Floor, Gamma, J0, J1, Log, Log10, Log1p, Log2, Logb, Round, RoundToEven, Sin, Sinh, Sqrt, Tan, Tanh, Trunc, Y0, Y1}

	for i, fun := range mfuncs {
		tf := tfuncs[i]
		tf(scalar, scout)
		tf(oned, oneout)
		tf(cell2d, cellout)

		Equal(t, fun(scalar.Float1D(0)), scout.Float1D(0))
		for i, v := range vals {
			Equal(t, fun(v), oneout.Float1D(i))
		}
		lv := len(vals)
		for r := range 5 {
			// fmt.Println(r)
			si := lv * r
			for c, v := range vals {
				ov := tensor.AsFloat32Tensor(cellout).Values[si+c]
				Equal(t, fun(v), float64(ov))
			}
		}
	}
}

func TestOps(t *testing.T) {
	x := tensor.NewIntScalar(1)
	y := tensor.NewIntScalar(4)
	a := tensor.CallOut("Mul", x, tensor.NewIntScalar(2))
	b := tensor.CallOut("Add", x, y)
	c := tensor.CallOut("Add", tensor.CallOut("Mul", x, y), tensor.CallOut("Mul", a, b))

	assert.Equal(t, 14, c.IntRow(0))
}
