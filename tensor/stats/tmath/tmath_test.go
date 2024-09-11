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
	scalar := tensor.NewIndexed(tensor.NewFloat64([]int{1}))
	scalar.Tensor.SetFloat1D(0, -5.5)
	scout := scalar.Clone()

	vals := []float64{-1.507556722888818, -1.2060453783110545, -0.9045340337332908, -0.6030226891555273, -0.3015113445777635, 0, 0.3015113445777635, 0.603022689155527, 0.904534033733291, 1.2060453783110545, 1.507556722888818, .3}

	oned := tensor.NewIndexed(tensor.NewNumberFromSlice(vals))
	oneout := oned.Clone()

	cell2d := tensor.NewIndexed(tensor.NewFloat32([]int{5, 2, 6}))
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, ci := cell2d.RowCellIndex(idx)
		cell2d.Tensor.SetFloat1D(i1d, oned.Tensor.Float1D(ci))
	}, cell2d)
	// cell2d.DeleteRows(3, 1)
	cellout := cell2d.Clone()

	mfuncs := []onef{math.Abs, math.Acos, math.Acosh, math.Asin, math.Asinh, math.Atan, math.Atanh, math.Cbrt, math.Ceil, math.Cos, math.Cosh, math.Erf, math.Erfc, math.Erfcinv, math.Erfcinv, math.Erfinv, math.Exp, math.Exp2, math.Expm1, math.Floor, math.Gamma, math.J0, math.J1, math.Log, math.Log10, math.Log1p, math.Log2, math.Logb, math.Round, math.RoundToEven, math.Sin, math.Sinh, math.Sqrt, math.Tan, math.Tanh, math.Trunc, math.Y0, math.Y1}
	tfuncs := []Func1in1out{Abs, Acos, Acosh, Asin, Asinh, Atan, Atanh, Cbrt, Ceil, Cos, Cosh, Erf, Erfc, Erfcinv, Erfcinv, Erfinv, Exp, Exp2, Expm1, Floor, Gamma, J0, J1, Log, Log10, Log1p, Log2, Logb, Round, RoundToEven, Sin, Sinh, Sqrt, Tan, Tanh, Trunc, Y0, Y1}

	for i, fun := range mfuncs {
		tf := tfuncs[i]
		tf(scalar, scout)
		tf(oned, oneout)
		tf(cell2d, cellout)

		Equal(t, fun(scalar.Tensor.Float1D(0)), scout.Tensor.Float1D(0))
		for i, v := range vals {
			Equal(t, fun(v), oneout.Tensor.Float1D(i))
		}
		lv := len(vals)
		for r := range 5 {
			// fmt.Println(r)
			si := lv * r
			for c, v := range vals {
				ov := cellout.Tensor.(*tensor.Float32).Values[si+c]
				Equal(t, fun(v), float64(ov))
			}
		}
	}
}

/*
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
*/
