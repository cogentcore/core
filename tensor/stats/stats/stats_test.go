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
	// ix := tensor.NewIndexed(tensor.NewNumberFromSlice(vals...))
	ix := tensor.NewNumberFromSlice(vals...)
	out := tensor.NewFloat64(1)

	results := []float64{11, 5.5, 5.5, 5.5, 0, 0, 1, 0, 1, 0.5, 0.11, math.Sqrt(0.11), math.Sqrt(0.11) / math.Sqrt(11), 3.85, math.Sqrt(3.85), 0.1, math.Sqrt(0.1), math.Sqrt(0.1) / math.Sqrt(11), 0.5, 0.25, 0.75}

	tol := 1.0e-8

	CountFunc(ix, out)
	assert.Equal(t, results[Count], out.Values[0])

	SumFunc(ix, out)
	assert.Equal(t, results[Sum], out.Values[0])

	SumAbsFunc(ix, out)
	assert.Equal(t, results[SumAbs], out.Values[0])

	ProdFunc(ix, out)
	assert.Equal(t, results[Prod], out.Values[0])

	MinFunc(ix, out)
	assert.Equal(t, results[Min], out.Values[0])

	MaxFunc(ix, out)
	assert.Equal(t, results[Max], out.Values[0])

	MinAbsFunc(ix, out)
	assert.Equal(t, results[MinAbs], out.Values[0])

	MaxAbsFunc(ix, out)
	assert.Equal(t, results[MaxAbs], out.Values[0])

	MeanFunc(ix, out)
	assert.Equal(t, results[Mean], out.Values[0])

	VarFunc(ix, out)
	assert.InDelta(t, results[Var], out.Values[0], tol)

	StdFunc(ix, out)
	assert.InDelta(t, results[Std], out.Values[0], tol)

	SemFunc(ix, out)
	assert.InDelta(t, results[Sem], out.Values[0], tol)

	VarPopFunc(ix, out)
	assert.InDelta(t, results[VarPop], out.Values[0], tol)

	StdPopFunc(ix, out)
	assert.InDelta(t, results[StdPop], out.Values[0], tol)

	SemPopFunc(ix, out)
	assert.InDelta(t, results[SemPop], out.Values[0], tol)

	SumSqFunc(ix, out)
	assert.InDelta(t, results[SumSq], out.Values[0], tol)

	L2NormFunc(ix, out)
	assert.InDelta(t, results[L2Norm], out.Values[0], tol)

	MedianFunc(ix, out)
	assert.InDelta(t, results[Median], out.Values[0], tol)

	Q1Func(ix, out)
	assert.InDelta(t, results[Q1], out.Values[0], tol)

	Q3Func(ix, out)
	assert.InDelta(t, results[Q3], out.Values[0], tol)

	for stat := Count; stat < StatsN; stat++ {
		Stat(stat, ix, out)
		assert.InDelta(t, results[stat], out.Values[0], tol)
	}
}

func TestFuncsInt(t *testing.T) {
	vals := []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	tsr := tensor.NewNumberFromSlice(vals...)
	ix := tensor.NewIndexed(tsr)
	out := tensor.NewInt(1)

	results := []int{11, 550, 550, 550, 0, 0, 100, 0, 100, 50, 1100, int(math.Sqrt(1100)), int(math.Sqrt(1100) / math.Sqrt(11)), 38500, 196, 1000, int(math.Sqrt(1000)), int(math.Sqrt(1000) / math.Sqrt(11))}

	CountFunc(ix, out)
	assert.Equal(t, results[Count], out.Values[0])

	SumFunc(ix, out)
	assert.Equal(t, results[Sum], out.Values[0])

	SumAbsFunc(ix, out)
	assert.Equal(t, results[SumAbs], out.Values[0])

	ProdFunc(ix, out)
	assert.Equal(t, results[Prod], out.Values[0])

	MinFunc(ix, out)
	assert.Equal(t, results[Min], out.Values[0])

	MaxFunc(ix, out)
	assert.Equal(t, results[Max], out.Values[0])

	MinAbsFunc(ix, out)
	assert.Equal(t, results[MinAbs], out.Values[0])

	MaxAbsFunc(ix, out)
	assert.Equal(t, results[MaxAbs], out.Values[0])

	MeanFunc(ix, out)
	assert.Equal(t, results[Mean], out.Values[0])

	VarFunc(ix, out)
	assert.Equal(t, results[Var], out.Values[0])

	StdFunc(ix, out)
	assert.Equal(t, results[Std], out.Values[0])

	SemFunc(ix, out)
	assert.Equal(t, results[Sem], out.Values[0])

	VarPopFunc(ix, out)
	assert.Equal(t, results[VarPop], out.Values[0])

	StdPopFunc(ix, out)
	assert.Equal(t, results[StdPop], out.Values[0])

	SemPopFunc(ix, out)
	assert.Equal(t, results[SemPop], out.Values[0])

	SumSqFunc(ix, out)
	assert.Equal(t, results[SumSq], out.Values[0])

	L2NormFunc(ix, out)
	assert.Equal(t, results[L2Norm], out.Values[0])

	for stat := Count; stat <= SemPop; stat++ {
		Stat(stat, ix, out)
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

	ix := tensor.NewIndexed(tsr)
	out := tensor.NewFloat32(20, 10)

	CountFunc(ix, out)
	nsub := out.Len()
	for i := range nsub {
		assert.Equal(t, 20.0, out.FloatRowCell(0, i))
	}
	MeanFunc(ix, out)
	for i := range nsub {
		assert.InDelta(t, vals[i], out.FloatRowCell(0, i), 1.0e-7) // lower tol, using float32
	}
	VarFunc(ix, out)
	for i := range nsub {
		assert.InDelta(t, 0.0, out.FloatRowCell(0, i), 1.0e-7)
	}
}
