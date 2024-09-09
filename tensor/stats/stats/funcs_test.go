// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"
	"testing"

	"cogentcore.org/core/base/tolassert"
	"cogentcore.org/core/tensor"
	"github.com/stretchr/testify/assert"
)

// todo: add int, tensor cell tests!

func TestFuncs64(t *testing.T) {
	vals := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}
	tsr := tensor.NewNumberFromSlice(vals)
	ix := tensor.NewIndexed(tsr)
	out := tensor.NewFloat64([]int{1})
	oix := tensor.NewIndexed(out)

	results := []float64{11, 5.5, 5.5, 5.5, 0, 0, 1, 0, 1, 0.5, 0.11, math.Sqrt(0.11), math.Sqrt(0.11) / math.Sqrt(11), 3.85, math.Sqrt(3.85), 0.1, math.Sqrt(0.1), math.Sqrt(0.1) / math.Sqrt(11)}

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
	assert.InDelta(t, results[Var], out.Values[0], 1.0e-8)

	StdFunc(ix, oix)
	assert.InDelta(t, results[Std], out.Values[0], 1.0e-8)

	SemFunc(ix, oix)
	assert.InDelta(t, results[Sem], out.Values[0], 1.0e-8)

	VarPopFunc(ix, oix)
	assert.InDelta(t, results[VarPop], out.Values[0], 1.0e-8)

	StdPopFunc(ix, oix)
	assert.InDelta(t, results[StdPop], out.Values[0], 1.0e-8)

	SemPopFunc(ix, oix)
	assert.InDelta(t, results[SemPop], out.Values[0], 1.0e-8)

	SumSqFunc(ix, oix)
	assert.InDelta(t, results[SumSq], out.Values[0], 1.0e-8)

	L2NormFunc(ix, oix)
	assert.InDelta(t, results[L2Norm], out.Values[0], 1.0e-8)

	for stat := Count; stat <= SemPop; stat++ {
		Standard(stat, ix, oix)
		tolassert.EqualTol(t, results[stat], out.Values[0], 1.0e-8)
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
