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

func TestFuncs(t *testing.T) {
	vals := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}
	tsr := tensor.NewNumberFromSlice(vals)
	ix := tensor.NewIndexed(tsr)
	out := tensor.NewFloat64([]int{1})
	oix := tensor.NewIndexed(out)

	results := []float64{11, 5.5, 0, 0, 1, 0, 1, 0.5, 0.11, math.Sqrt(0.11), math.Sqrt(0.11) / math.Sqrt(11), 5.5, 3.85, math.Sqrt(3.85), 0.1, math.Sqrt(0.1), math.Sqrt(0.1) / math.Sqrt(11)}

	// assert.Equal(t, results[Count], Count64(vals))
	tensor.Vectorize(NFunc, SumFunc, ix, oix)
	assert.Equal(t, results[Sum], out.Values[0])

	tensor.Vectorize(NFunc, ProdFunc, ix, oix)
	assert.Equal(t, results[Prod], out.Values[0])

	tensor.Vectorize(NFunc, MinFunc, ix, oix)
	assert.Equal(t, results[Min], out.Values[0])

	tensor.Vectorize(NFunc, MaxFunc, ix, oix)
	assert.Equal(t, results[Max], out.Values[0])

	tensor.Vectorize(NFunc, MinAbsFunc, ix, oix)
	assert.Equal(t, results[MinAbs], out.Values[0])

	tensor.Vectorize(NFunc, MaxAbsFunc, ix, oix)
	assert.Equal(t, results[MaxAbs], out.Values[0])

	// assert.Equal(t, results[Mean], Mean64(vals))
	// tolassert.EqualTol(t, results[Var], Var64(vals), 1.0e-8)
	// tolassert.EqualTol(t, results[Std], Std64(vals), 1.0e-8)
	// tolassert.EqualTol(t, results[Sem], Sem64(vals), 1.0e-8)
	// assert.Equal(t, results[L1Norm], L1Norm64(vals))
	// tolassert.EqualTol(t, results[SumSq], SumSq64(vals), 1.0e-8)
	// tolassert.EqualTol(t, results[L2Norm], L2Norm64(vals), 1.0e-8)
	// assert.Equal(t, results[VarPop], VarPop64(vals))
	// assert.Equal(t, results[StdPop], StdPop64(vals))
	// assert.Equal(t, results[SemPop], SemPop64(vals))
	//
	// for stat := Count; stat <= SemPop; stat++ {
	// 	tolassert.EqualTol(t, results[stat], Stat64(vals, stat), 1.0e-8)
	// }
}
