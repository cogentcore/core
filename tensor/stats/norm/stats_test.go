// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package norm

import (
	"math"
	"testing"

	"cogentcore.org/core/base/tolassert"
	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

func TestStats32(t *testing.T) {
	vals := []float32{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}

	assert.Equal(t, float32(11), N32(vals))
	assert.Equal(t, float32(5.5), Sum32(vals))
	assert.Equal(t, float32(0.5), Mean32(vals))
	assert.Equal(t, float32(0.11), Var32(vals))
	assert.Equal(t, math32.Sqrt(0.11), Std32(vals))
	assert.Equal(t, float32(1), Max32(vals))
	assert.Equal(t, float32(0), Min32(vals))
	assert.Equal(t, float32(1), MaxAbs32(vals))
	assert.Equal(t, float32(0), MinAbs32(vals))
	assert.Equal(t, float32(5.5), L132(vals))
	tolassert.EqualTol(t, float32(3.85), SumSquares32(vals), 1.0e-6)
	tolassert.EqualTol(t, math32.Sqrt(3.85), L232(vals), 1.0e-6)
}

func TestStats64(t *testing.T) {
	vals := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}

	assert.Equal(t, float64(11), N64(vals))
	assert.Equal(t, float64(5.5), Sum64(vals))
	assert.Equal(t, float64(0.5), Mean64(vals))
	tolassert.EqualTol(t, float64(0.11), Var64(vals), 1.0e-8)
	tolassert.EqualTol(t, math.Sqrt(0.11), Std64(vals), 1.0e-8)
	assert.Equal(t, float64(1), Max64(vals))
	assert.Equal(t, float64(0), Min64(vals))
	assert.Equal(t, float64(1), MaxAbs64(vals))
	assert.Equal(t, float64(0), MinAbs64(vals))
	assert.Equal(t, float64(5.5), L164(vals))
	tolassert.EqualTol(t, float64(3.85), SumSquares64(vals), 1.0e-6)
	tolassert.EqualTol(t, math.Sqrt(3.85), L264(vals), 1.0e-6)
}
