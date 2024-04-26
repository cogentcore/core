// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package agg

import (
	"math"
	"testing"

	"cogentcore.org/core/tensor/table"

	"github.com/stretchr/testify/assert"
)

func TestAgg(t *testing.T) {
	dt := table.NewTable(5)
	dt.AddFloat64Column("data")
	dt.SetFloat("data", 0, 1)
	dt.SetFloat("data", 1, 2)
	dt.SetFloat("data", 2, 3)
	dt.SetFloat("data", 3, 4)
	dt.SetFloat("data", 4, 5)

	ix := table.NewIndexView(dt)

	results := []float64{5, 15, 120, 1, 5, 3, 2.5, math.Sqrt(2.5), math.Sqrt(2.5) / math.Sqrt(5),
		2, math.Sqrt(2), math.Sqrt(2) / math.Sqrt(5), 3, 2, 4, 55}

	assert.Equal(t, results[AggCount:AggCount+1], Count(ix, "data"))
	assert.Equal(t, results[AggSum:AggSum+1], Sum(ix, "data"))
	assert.Equal(t, results[AggProd:AggProd+1], Prod(ix, "data"))
	assert.Equal(t, results[AggMin:AggMin+1], Min(ix, "data"))
	assert.Equal(t, results[AggMax:AggMax+1], Max(ix, "data"))
	assert.Equal(t, results[AggMean:AggMean+1], Mean(ix, "data"))
	assert.Equal(t, results[AggVar:AggVar+1], Var(ix, "data"))
	assert.Equal(t, results[AggStd:AggStd+1], Std(ix, "data"))
	assert.Equal(t, results[AggSem:AggSem+1], Sem(ix, "data"))
	assert.Equal(t, results[AggVarPop:AggVarPop+1], VarPop(ix, "data"))
	assert.Equal(t, results[AggStdPop:AggStdPop+1], StdPop(ix, "data"))
	assert.Equal(t, results[AggSemPop:AggSemPop+1], SemPop(ix, "data"))
	assert.Equal(t, results[AggMedian:AggMedian+1], Median(ix, "data"))
	assert.Equal(t, results[AggQ1:AggQ1+1], Q1(ix, "data"))
	assert.Equal(t, results[AggQ3:AggQ3+1], Q3(ix, "data"))
	assert.Equal(t, results[AggSumSq:AggSumSq+1], SumSq(ix, "data"))

	for ag := AggCount; ag < AggsN; ag++ {
		assert.Equal(t, results[ag:ag+1], Agg(ix, "data", ag))
	}

	desc := DescAll(ix)
	assert.Equal(t, len(DescAggs), desc.Rows)
	assert.Equal(t, 2, desc.NumColumns())

	for ri, ag := range DescAggs {
		dv := desc.Float("data", ri)
		// fmt.Println(ri, ag.String(), dv, results[ag])
		assert.Equal(t, results[ag], dv)
	}

	desc = Desc(ix, "data")
	assert.Equal(t, len(DescAggs), desc.Rows)
	assert.Equal(t, 2, desc.NumColumns())
	for ri, ag := range DescAggs {
		dv := desc.Float("data", ri)
		// fmt.Println(ri, ag.String(), dv, results[ag])
		assert.Equal(t, results[ag], dv)
	}

	pcts := PctIf(ix, "data", func(idx int, val float64) bool {
		return val > 2
	})
	assert.Equal(t, []float64{60}, pcts)

	props := PropIf(ix, "data", func(idx int, val float64) bool {
		return val > 2
	})
	assert.Equal(t, []float64{0.6}, props)
}
