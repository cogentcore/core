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
	ix := tensor.NewIndexed(tensor.NewNumberFromSlice(vals))
	out := tensor.NewFloat64([]int{1})
	oix := tensor.NewIndexed(out)

	results := []float64{11, 5.5, 5.5, 5.5, 0, 0, 1, 0, 1, 0.5, 0.11, math.Sqrt(0.11), math.Sqrt(0.11) / math.Sqrt(11), 3.85, math.Sqrt(3.85), 0.1, math.Sqrt(0.1), math.Sqrt(0.1) / math.Sqrt(11), 0.5, 0.25, 0.75}

	tol := 1.0e-8

	CountFunc(ix, oix)
	assert.Equal(t, results[Count], out.Values[0])

	SumFunc(ix, oix)
	assert.Equal(t, results[Sum], out.Values[0])

	SumAbsFunc(ix, oix)
	assert.Equal(t, results[SumAbs], out.Values[0])

	ProdFunc(ix, oix)
	assert.Equal(t, results[Prod], out.Values[0])

	MinFunc(ix, oix)
	assert.Equal(t, results[Min], out.Values[0])

	MaxFunc(ix, oix)
	assert.Equal(t, results[Max], out.Values[0])

	MinAbsFunc(ix, oix)
	assert.Equal(t, results[MinAbs], out.Values[0])

	MaxAbsFunc(ix, oix)
	assert.Equal(t, results[MaxAbs], out.Values[0])

	MeanFunc(ix, oix)
	assert.Equal(t, results[Mean], out.Values[0])

	VarFunc(ix, oix)
	assert.InDelta(t, results[Var], out.Values[0], tol)

	StdFunc(ix, oix)
	assert.InDelta(t, results[Std], out.Values[0], tol)

	SemFunc(ix, oix)
	assert.InDelta(t, results[Sem], out.Values[0], tol)

	VarPopFunc(ix, oix)
	assert.InDelta(t, results[VarPop], out.Values[0], tol)

	StdPopFunc(ix, oix)
	assert.InDelta(t, results[StdPop], out.Values[0], tol)

	SemPopFunc(ix, oix)
	assert.InDelta(t, results[SemPop], out.Values[0], tol)

	SumSqFunc(ix, oix)
	assert.InDelta(t, results[SumSq], out.Values[0], tol)

	L2NormFunc(ix, oix)
	assert.InDelta(t, results[L2Norm], out.Values[0], tol)

	MedianFunc(ix, oix)
	assert.InDelta(t, results[Median], out.Values[0], tol)

	Q1Func(ix, oix)
	assert.InDelta(t, results[Q1], out.Values[0], tol)

	Q3Func(ix, oix)
	assert.InDelta(t, results[Q3], out.Values[0], tol)

	for stat := Count; stat < StatsN; stat++ {
		Standard(stat, ix, oix)
		assert.InDelta(t, results[stat], out.Values[0], tol)
	}
}

func TestFuncsInt(t *testing.T) {
	vals := []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	tsr := tensor.NewNumberFromSlice(vals)
	ix := tensor.NewIndexed(tsr)
	out := tensor.NewInt([]int{1})
	oix := tensor.NewIndexed(out)

	results := []int{11, 550, 550, 550, 0, 0, 100, 0, 100, 50, 1100, int(math.Sqrt(1100)), int(math.Sqrt(1100) / math.Sqrt(11)), 38500, 196, 1000, int(math.Sqrt(1000)), int(math.Sqrt(1000) / math.Sqrt(11))}

	CountFunc(ix, oix)
	assert.Equal(t, results[Count], out.Values[0])

	SumFunc(ix, oix)
	assert.Equal(t, results[Sum], out.Values[0])

	SumAbsFunc(ix, oix)
	assert.Equal(t, results[SumAbs], out.Values[0])

	ProdFunc(ix, oix)
	assert.Equal(t, results[Prod], out.Values[0])

	MinFunc(ix, oix)
	assert.Equal(t, results[Min], out.Values[0])

	MaxFunc(ix, oix)
	assert.Equal(t, results[Max], out.Values[0])

	MinAbsFunc(ix, oix)
	assert.Equal(t, results[MinAbs], out.Values[0])

	MaxAbsFunc(ix, oix)
	assert.Equal(t, results[MaxAbs], out.Values[0])

	MeanFunc(ix, oix)
	assert.Equal(t, results[Mean], out.Values[0])

	VarFunc(ix, oix)
	assert.Equal(t, results[Var], out.Values[0])

	StdFunc(ix, oix)
	assert.Equal(t, results[Std], out.Values[0])

	SemFunc(ix, oix)
	assert.Equal(t, results[Sem], out.Values[0])

	VarPopFunc(ix, oix)
	assert.Equal(t, results[VarPop], out.Values[0])

	StdPopFunc(ix, oix)
	assert.Equal(t, results[StdPop], out.Values[0])

	SemPopFunc(ix, oix)
	assert.Equal(t, results[SemPop], out.Values[0])

	SumSqFunc(ix, oix)
	assert.Equal(t, results[SumSq], out.Values[0])

	L2NormFunc(ix, oix)
	assert.Equal(t, results[L2Norm], out.Values[0])

	for stat := Count; stat <= SemPop; stat++ {
		Standard(stat, ix, oix)
		assert.Equal(t, results[stat], out.Values[0])
	}
}

func TestFuncsCell(t *testing.T) {
	vals := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9}
	tsr := tensor.NewFloat32([]int{20, 10})

	for i := range 20 {
		for j := range 10 {
			tsr.SetFloatRowCell(i, j, vals[j])
		}
	}

	ix := tensor.NewIndexed(tsr)
	out := tensor.NewFloat32([]int{20, 10})
	oix := tensor.NewIndexed(out)

	CountFunc(ix, oix)
	nsub := out.Len()
	for i := range nsub {
		assert.Equal(t, 20.0, out.FloatRowCell(0, i))
	}
	MeanFunc(ix, oix)
	for i := range nsub {
		assert.InDelta(t, vals[i], out.FloatRowCell(0, i), 1.0e-7) // lower tol, using float32
	}
	VarFunc(ix, oix)
	for i := range nsub {
		assert.InDelta(t, 0.0, out.FloatRowCell(0, i), 1.0e-7)
	}
}

/*
func TestIndexed(t *testing.T) {
	desc := DescAll(ix)
	assert.Equal(t, len(DescStats), desc.Rows)
	assert.Equal(t, 2, desc.NumColumns())

	for ri, stat := range DescStats {
		dv := desc.Float("data", ri)
		// fmt.Println(ri, ag.String(), dv, results[ag])
		assert.Equal(t, results[stat], dv)
	}

	desc, err := DescColumn(ix, "data")
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, len(DescStats), desc.Rows)
	assert.Equal(t, 2, desc.NumColumns())
	for ri, stat := range DescStats {
		dv := desc.Float("data", ri)
		// fmt.Println(ri, ag.String(), dv, results[ag])
		assert.Equal(t, results[stat], dv)
	}

	pcts := PctIfColumn(ix, "data", func(idx int, val float64) bool {
		return val > 2
	})
	assert.Equal(t, []float64{60}, pcts)

	props := PropIfColumn(ix, "data", func(idx int, val float64) bool {
		return val > 2
	})
	assert.Equal(t, []float64{0.6}, props)
}
*/
