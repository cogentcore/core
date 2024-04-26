// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tsragg

import (
	"math"
	"testing"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/agg"
	"github.com/stretchr/testify/assert"
)

func TestTsrAgg(t *testing.T) {
	tsr := tensor.New[float64]([]int{5}).(*tensor.Float64)
	tsr.Values = []float64{1, 2, 3, 4, 5}

	results := []float64{5, 15, 120, 1, 5, 3, 2.5, math.Sqrt(2.5), math.Sqrt(2.5) / math.Sqrt(5),
		2, math.Sqrt(2), math.Sqrt(2) / math.Sqrt(5), 3, 2, 4, 55}

	assert.Equal(t, results[agg.AggCount], Count(tsr))
	assert.Equal(t, results[agg.AggSum], Sum(tsr))
	assert.Equal(t, results[agg.AggProd], Prod(tsr))
	assert.Equal(t, results[agg.AggMin], Min(tsr))
	assert.Equal(t, results[agg.AggMax], Max(tsr))
	assert.Equal(t, results[agg.AggMean], Mean(tsr))
	assert.Equal(t, results[agg.AggVar], Var(tsr))
	assert.Equal(t, results[agg.AggStd], Std(tsr))
	assert.Equal(t, results[agg.AggSem], Sem(tsr))
	assert.Equal(t, results[agg.AggVarPop], VarPop(tsr))
	assert.Equal(t, results[agg.AggStdPop], StdPop(tsr))
	assert.Equal(t, results[agg.AggSemPop], SemPop(tsr))
	assert.Equal(t, results[agg.AggSumSq], SumSq(tsr))
}
