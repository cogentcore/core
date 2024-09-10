// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"math"
	"testing"

	"cogentcore.org/core/tensor"
	"github.com/stretchr/testify/assert"
)

func TestAll(t *testing.T) {
	a64 := []float64{.5, .2, .1, .7, math.NaN(), .5}
	b64 := []float64{.2, .9, .1, .7, 0, .2}

	results := []float64{math.Sqrt(0.67), 0.67, 0.9, 3, 0.7, 0.49, 1 - 0.7319115529256469, 1 - 0.11189084777289171, 0, 0.88, 0.008, 0.11189084777289171, 0.7319115529256469}

	atsr := tensor.NewIndexed(tensor.NewNumberFromSlice(a64))
	btsr := tensor.NewIndexed(tensor.NewNumberFromSlice(b64))
	out := tensor.NewFloat64([]int{1})
	oix := tensor.NewIndexed(out)

	EuclideanFunc(atsr, btsr, oix)
	assert.InDelta(t, results[Euclidean], out.Values[0], 1.0e-8)

	SumSquaresFunc(atsr, btsr, oix)
	assert.InDelta(t, results[SumSquares], out.Values[0], 1.0e-8)

	EuclideanBinTolFunc(atsr, btsr, oix)
	assert.Equal(t, results[EuclideanBinTol], out.Values[0])

	SumSquaresBinTolFunc(atsr, btsr, oix)
	assert.InDelta(t, results[SumSquaresBinTol], out.Values[0], 1.0e-8)

	CovarianceFunc(atsr, btsr, oix)
	assert.InDelta(t, results[Covariance], out.Values[0], 1.0e-8)

	CorrelationFunc(atsr, btsr, oix)
	assert.Equal(t, results[Correlation], out.Values[0])

	InvCorrelationFunc(atsr, btsr, oix)
	assert.Equal(t, results[InvCorrelation], out.Values[0])

	CosineFunc(atsr, btsr, oix)
	assert.Equal(t, results[Cosine], out.Values[0])

	InvCosineFunc(atsr, btsr, oix)
	assert.Equal(t, results[InvCosine], out.Values[0])

	InnerProductFunc(atsr, btsr, oix)
	assert.Equal(t, results[InnerProduct], out.Values[0])

	/*
		ab := Abs64(a64, b64)
		if ab != 0.8999999999999999 {
			t.Errorf("Abs64: %g\n", ab)
		}

		hm := Hamming64(a64, b64)
		if hm != 3 {
			t.Errorf("Hamming64: %g\n", hm)
		}
	*/
}
