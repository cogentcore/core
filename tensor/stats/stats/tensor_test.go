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

func TestTsrAgg(t *testing.T) {
	tsr := tensor.New[float64]([]int{5}).(*tensor.Float64)
	tsr.Values = []float64{1, 2, 3, 4, 5}

	results := []float64{5, 15, 120, 1, 5, 1, 5, 3, 2.5, math.Sqrt(2.5), math.Sqrt(2.5) / math.Sqrt(5),
		15, 55, math.Sqrt(55), 2, math.Sqrt(2), math.Sqrt(2) / math.Sqrt(5), 3, 2, 4}

	assert.Equal(t, results[Count], CountTensor(tsr))
	assert.Equal(t, results[Sum], SumTensor(tsr))
	assert.Equal(t, results[Prod], ProdTensor(tsr))
	assert.Equal(t, results[Min], MinTensor(tsr))
	assert.Equal(t, results[Max], MaxTensor(tsr))
	assert.Equal(t, results[MinAbs], MinAbsTensor(tsr))
	assert.Equal(t, results[MaxAbs], MaxAbsTensor(tsr))
	assert.Equal(t, results[Mean], MeanTensor(tsr))
	assert.Equal(t, results[Var], VarTensor(tsr))
	assert.Equal(t, results[Std], StdTensor(tsr))
	assert.Equal(t, results[Sem], SemTensor(tsr))
	assert.Equal(t, results[L1Norm], L1NormTensor(tsr))
	tolassert.EqualTol(t, results[SumSq], SumSqTensor(tsr), 1.0e-8)
	tolassert.EqualTol(t, results[L2Norm], L2NormTensor(tsr), 1.0e-8)
	assert.Equal(t, results[VarPop], VarPopTensor(tsr))
	assert.Equal(t, results[StdPop], StdPopTensor(tsr))
	assert.Equal(t, results[SemPop], SemPopTensor(tsr))

	for stat := Count; stat <= SemPop; stat++ {
		tolassert.EqualTol(t, results[stat], StatTensor(tsr, stat), 1.0e-8)
	}
}
