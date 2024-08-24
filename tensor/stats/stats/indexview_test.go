// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"
	"testing"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/tolassert"
	"cogentcore.org/core/tensor/table"

	"github.com/stretchr/testify/assert"
)

func TestIndexView(t *testing.T) {
	dt := table.NewTable().SetNumRows(5)
	dt.AddFloat64Column("data")
	dt.SetFloat("data", 0, 1)
	dt.SetFloat("data", 1, 2)
	dt.SetFloat("data", 2, 3)
	dt.SetFloat("data", 3, 4)
	dt.SetFloat("data", 4, 5)

	ix := table.NewIndexView(dt)

	results := []float64{5, 15, 120, 1, 5, 1, 5, 3, 2.5, math.Sqrt(2.5), math.Sqrt(2.5) / math.Sqrt(5),
		15, 55, math.Sqrt(55), 2, math.Sqrt(2), math.Sqrt(2) / math.Sqrt(5), 3, 2, 4}

	assert.Equal(t, results[Count:Count+1], CountColumn(ix, "data"))
	assert.Equal(t, results[Sum:Sum+1], SumColumn(ix, "data"))
	assert.Equal(t, results[Prod:Prod+1], ProdColumn(ix, "data"))
	assert.Equal(t, results[Min:Min+1], MinColumn(ix, "data"))
	assert.Equal(t, results[Max:Max+1], MaxColumn(ix, "data"))
	assert.Equal(t, results[MinAbs:MinAbs+1], MinAbsColumn(ix, "data"))
	assert.Equal(t, results[MaxAbs:MaxAbs+1], MaxAbsColumn(ix, "data"))
	assert.Equal(t, results[Mean:Mean+1], MeanColumn(ix, "data"))
	assert.Equal(t, results[Var:Var+1], VarColumn(ix, "data"))
	assert.Equal(t, results[Std:Std+1], StdColumn(ix, "data"))
	assert.Equal(t, results[Sem:Sem+1], SemColumn(ix, "data"))
	assert.Equal(t, results[L1Norm:L1Norm+1], L1NormColumn(ix, "data"))
	tolassert.EqualTol(t, results[SumSq], SumSqColumn(ix, "data")[0], 1.0e-8)
	tolassert.EqualTol(t, results[L2Norm], L2NormColumn(ix, "data")[0], 1.0e-8)
	assert.Equal(t, results[VarPop:VarPop+1], VarPopColumn(ix, "data"))
	assert.Equal(t, results[StdPop:StdPop+1], StdPopColumn(ix, "data"))
	assert.Equal(t, results[SemPop:SemPop+1], SemPopColumn(ix, "data"))
	assert.Equal(t, results[Median:Median+1], MedianColumn(ix, "data"))
	assert.Equal(t, results[Q1:Q1+1], Q1Column(ix, "data"))
	assert.Equal(t, results[Q3:Q3+1], Q3Column(ix, "data"))

	for _, stat := range StatsValues() {
		tolassert.EqualTol(t, results[stat], errors.Log1(StatColumn(ix, "data", stat))[0], 1.0e-8)
	}

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
