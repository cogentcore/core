// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"
	"testing"

	"cogentcore.org/core/base/tolassert"
	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

func TestStats32(t *testing.T) {
	vals := []float32{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}

	results := []float32{11, 5.5, 0, 0, 1, 0, 1, 0.5, 0.11, math32.Sqrt(0.11), math32.Sqrt(0.11) / math32.Sqrt(11), 5.5, 3.85, math32.Sqrt(3.85), 0.1, math32.Sqrt(0.1), math32.Sqrt(0.1) / math32.Sqrt(11)}

	assert.Equal(t, results[Count], Count32(vals))
	assert.Equal(t, results[Sum], Sum32(vals))
	assert.Equal(t, results[Prod], Prod32(vals))
	assert.Equal(t, results[Min], Min32(vals))
	assert.Equal(t, results[Max], Max32(vals))
	assert.Equal(t, results[MinAbs], MinAbs32(vals))
	assert.Equal(t, results[MaxAbs], MaxAbs32(vals))
	assert.Equal(t, results[Mean], Mean32(vals))
	assert.Equal(t, results[Var], Var32(vals))
	assert.Equal(t, results[Std], Std32(vals))
	assert.Equal(t, results[Sem], Sem32(vals))
	assert.Equal(t, results[L1Norm], L1Norm32(vals))
	tolassert.EqualTol(t, results[SumSq], SumSq32(vals), 1.0e-6)
	tolassert.EqualTol(t, results[L2Norm], L2Norm32(vals), 1.0e-6)
	assert.Equal(t, results[VarPop], VarPop32(vals))
	assert.Equal(t, results[StdPop], StdPop32(vals))
	assert.Equal(t, results[SemPop], SemPop32(vals))

	for stat := Count; stat <= SemPop; stat++ {
		tolassert.EqualTol(t, results[stat], Stat32(vals, stat), 1.0e-6)
	}
}

func TestStats64(t *testing.T) {
	vals := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}

	results := []float64{11, 5.5, 0, 0, 1, 0, 1, 0.5, 0.11, math.Sqrt(0.11), math.Sqrt(0.11) / math.Sqrt(11), 5.5, 3.85, math.Sqrt(3.85), 0.1, math.Sqrt(0.1), math.Sqrt(0.1) / math.Sqrt(11)}

	assert.Equal(t, results[Count], Count64(vals))
	assert.Equal(t, results[Sum], Sum64(vals))
	assert.Equal(t, results[Prod], Prod64(vals))
	assert.Equal(t, results[Min], Min64(vals))
	assert.Equal(t, results[Max], Max64(vals))
	assert.Equal(t, results[MinAbs], MinAbs64(vals))
	assert.Equal(t, results[MaxAbs], MaxAbs64(vals))
	assert.Equal(t, results[Mean], Mean64(vals))
	tolassert.EqualTol(t, results[Var], Var64(vals), 1.0e-8)
	tolassert.EqualTol(t, results[Std], Std64(vals), 1.0e-8)
	tolassert.EqualTol(t, results[Sem], Sem64(vals), 1.0e-8)
	assert.Equal(t, results[L1Norm], L1Norm64(vals))
	tolassert.EqualTol(t, results[SumSq], SumSq64(vals), 1.0e-8)
	tolassert.EqualTol(t, results[L2Norm], L2Norm64(vals), 1.0e-8)
	assert.Equal(t, results[VarPop], VarPop64(vals))
	assert.Equal(t, results[StdPop], StdPop64(vals))
	assert.Equal(t, results[SemPop], SemPop64(vals))

	for stat := Count; stat <= SemPop; stat++ {
		tolassert.EqualTol(t, results[stat], Stat64(vals, stat), 1.0e-8)
	}
}
