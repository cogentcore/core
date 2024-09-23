// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"
	"testing"

	"cogentcore.org/core/tensor"
	"github.com/stretchr/testify/assert"
)

func TestFuncs64(t *testing.T) {
	vals := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}
	ix := tensor.NewNumberFromValues(vals...)
	out := tensor.NewFloat64(1)

	results := []float64{11, 5.5, 5.5, 5.5, 0, 0, 1, 0, 1, 0.5, 0.11, math.Sqrt(0.11), math.Sqrt(0.11) / math.Sqrt(11), 3.85, math.Sqrt(3.85), 0.1, math.Sqrt(0.1), math.Sqrt(0.1) / math.Sqrt(11), 0.5, 0.25, 0.75}

	tol := 1.0e-8

	Count(ix, out)
	assert.Equal(t, results[StatCount], out.Values[0])

	Sum(ix, out)
	assert.Equal(t, results[StatSum], out.Values[0])

	SumAbs(ix, out)
	assert.Equal(t, results[StatSumAbs], out.Values[0])

	Prod(ix, out)
	assert.Equal(t, results[StatProd], out.Values[0])

	Min(ix, out)
	assert.Equal(t, results[StatMin], out.Values[0])

	Max(ix, out)
	assert.Equal(t, results[StatMax], out.Values[0])

	MinAbs(ix, out)
	assert.Equal(t, results[StatMinAbs], out.Values[0])

	MaxAbs(ix, out)
	assert.Equal(t, results[StatMaxAbs], out.Values[0])

	Mean(ix, out)
	assert.Equal(t, results[StatMean], out.Values[0])

	Var(ix, out)
	assert.InDelta(t, results[StatVar], out.Values[0], tol)

	Std(ix, out)
	assert.InDelta(t, results[StatStd], out.Values[0], tol)

	Sem(ix, out)
	assert.InDelta(t, results[StatSem], out.Values[0], tol)

	VarPop(ix, out)
	assert.InDelta(t, results[StatVarPop], out.Values[0], tol)

	StdPop(ix, out)
	assert.InDelta(t, results[StatStdPop], out.Values[0], tol)

	SemPop(ix, out)
	assert.InDelta(t, results[StatSemPop], out.Values[0], tol)

	SumSq(ix, out)
	assert.InDelta(t, results[StatSumSq], out.Values[0], tol)

	L2Norm(ix, out)
	assert.InDelta(t, results[StatL2Norm], out.Values[0], tol)

	Median(ix, out)
	assert.InDelta(t, results[StatMedian], out.Values[0], tol)

	Q1(ix, out)
	assert.InDelta(t, results[StatQ1], out.Values[0], tol)

	Q3(ix, out)
	assert.InDelta(t, results[StatQ3], out.Values[0], tol)

	for stat := StatCount; stat < StatsN; stat++ {
		err := stat.Call(ix, out)
		assert.NoError(t, err)
		assert.InDelta(t, results[stat], out.Values[0], tol)
	}
	err := tensor.Call("stats.Mean", ix, out) // ensure plain name is registered.
	assert.NoError(t, err)
	assert.InDelta(t, results[StatMean], out.Values[0], tol)
}

func TestFuncsInt(t *testing.T) {
	vals := []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	tsr := tensor.NewNumberFromValues(vals...)
	ix := tensor.NewRows(tsr)
	out := tensor.NewInt(1)

	results := []int{11, 550, 550, 550, 0, 0, 100, 0, 100, 50, 1100, int(math.Sqrt(1100)), int(math.Sqrt(1100) / math.Sqrt(11)), 38500, 196, 1000, int(math.Sqrt(1000)), int(math.Sqrt(1000) / math.Sqrt(11))}

	Count(ix, out)
	assert.Equal(t, results[StatCount], out.Values[0])

	Sum(ix, out)
	assert.Equal(t, results[StatSum], out.Values[0])

	SumAbs(ix, out)
	assert.Equal(t, results[StatSumAbs], out.Values[0])

	Prod(ix, out)
	assert.Equal(t, results[StatProd], out.Values[0])

	Min(ix, out)
	assert.Equal(t, results[StatMin], out.Values[0])

	Max(ix, out)
	assert.Equal(t, results[StatMax], out.Values[0])

	MinAbs(ix, out)
	assert.Equal(t, results[StatMinAbs], out.Values[0])

	MaxAbs(ix, out)
	assert.Equal(t, results[StatMaxAbs], out.Values[0])

	Mean(ix, out)
	assert.Equal(t, results[StatMean], out.Values[0])

	Var(ix, out)
	assert.Equal(t, results[StatVar], out.Values[0])

	Std(ix, out)
	assert.Equal(t, results[StatStd], out.Values[0])

	Sem(ix, out)
	assert.Equal(t, results[StatSem], out.Values[0])

	VarPop(ix, out)
	assert.Equal(t, results[StatVarPop], out.Values[0])

	StdPop(ix, out)
	assert.Equal(t, results[StatStdPop], out.Values[0])

	SemPop(ix, out)
	assert.Equal(t, results[StatSemPop], out.Values[0])

	SumSq(ix, out)
	assert.Equal(t, results[StatSumSq], out.Values[0])

	L2Norm(ix, out)
	assert.Equal(t, results[StatL2Norm], out.Values[0])

	for stat := StatCount; stat <= StatSemPop; stat++ {
		stat.Call(ix, out)
		assert.Equal(t, results[stat], out.Values[0])
	}
}

func TestFuncsCell(t *testing.T) {
	vals := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9}
	tsr := tensor.NewFloat32(20, 10)

	for i := range 20 {
		for j := range 10 {
			tsr.SetFloatRowCell(vals[j], i, j)
		}
	}

	ix := tensor.NewRows(tsr)
	out := tensor.NewFloat32(20, 10)

	Count(ix, out)
	nsub := out.Len()
	for i := range nsub {
		assert.Equal(t, 20.0, out.FloatRowCell(0, i))
	}
	Mean(ix, out)
	for i := range nsub {
		assert.InDelta(t, vals[i], out.FloatRowCell(0, i), 1.0e-7) // lower tol, using float32
	}
	Var(ix, out)
	for i := range nsub {
		assert.InDelta(t, 0.0, out.FloatRowCell(0, i), 1.0e-7)
	}
}

func TestNorm(t *testing.T) {
	vals := []float64{-1.507556722888818, -1.2060453783110545, -0.9045340337332908, -0.6030226891555273, -0.3015113445777635, 0.1, 0.3015113445777635, 0.603022689155527, 0.904534033733291, 1.2060453783110545, 1.507556722888818, .3}

	oned := tensor.NewNumberFromValues(vals...)
	oneout := oned.Clone()

	ZScore(oned, oneout)
	mout := tensor.NewFloat64()
	std, mean, _, err := StdOut64(oneout, mout)
	assert.NoError(t, err)
	assert.InDelta(t, 1.0, std.Float1D(0), 1.0e-6)
	assert.InDelta(t, 0.0, mean.Float1D(0), 1.0e-6)

	UnitNorm(oned, oneout)
	Min(oneout, mout)
	assert.InDelta(t, 0.0, mout.Float1D(0), 1.0e-6)
	Max(oneout, mout)
	assert.InDelta(t, 1.0, mout.Float1D(0), 1.0e-6)
	// fmt.Println(oneout)

	minv := tensor.NewFloat64Scalar(0)
	maxv := tensor.NewFloat64Scalar(1)
	Clamp(oned, minv, maxv, oneout)
	Min(oneout, mout)
	assert.InDelta(t, 0.0, mout.Float1D(0), 1.0e-6)
	Max(oneout, mout)
	assert.InDelta(t, 1.0, mout.Float1D(0), 1.0e-6)
	// fmt.Println(oneout)

	thr := tensor.NewFloat64Scalar(0.5)
	Binarize(oned, thr, oneout)
	Min(oneout, mout)
	assert.InDelta(t, 0.0, mout.Float1D(0), 1.0e-6)
	Max(oneout, mout)
	assert.InDelta(t, 1.0, mout.Float1D(0), 1.0e-6)
	// fmt.Println(oneout)
}
