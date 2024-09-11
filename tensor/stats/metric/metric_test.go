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

func TestFuncs(t *testing.T) {
	a64 := []float64{.5, .2, .1, .7, math.NaN(), .5}
	b64 := []float64{.2, .9, .1, .7, 0, .2}

	results := []float64{math.Sqrt(0.67), 0.67, 1.3, 3, 0.7, 0.49, 1 - 0.7319115529256469, 1 - 0.11189084777289171, 1.8090248566170337, 0.88, 0.008, 0.11189084777289171, 0.7319115529256469}

	tol := 1.0e-8

	atsr := tensor.NewIndexed(tensor.NewNumberFromSlice(a64))
	btsr := tensor.NewIndexed(tensor.NewNumberFromSlice(b64))
	out := tensor.NewFloat64([]int{1})
	oix := tensor.NewIndexed(out)

	EuclideanFunc(atsr, btsr, oix)
	assert.InDelta(t, results[Euclidean], out.Values[0], tol)

	SumSquaresFunc(atsr, btsr, oix)
	assert.InDelta(t, results[SumSquares], out.Values[0], tol)

	EuclideanBinTolFunc(atsr, btsr, oix)
	assert.InDelta(t, results[EuclideanBinTol], out.Values[0], tol)

	AbsFunc(atsr, btsr, oix)
	assert.InDelta(t, results[Abs], out.Values[0], tol)

	HammingFunc(atsr, btsr, oix)
	assert.Equal(t, results[Hamming], out.Values[0])

	SumSquaresBinTolFunc(atsr, btsr, oix)
	assert.InDelta(t, results[SumSquaresBinTol], out.Values[0], tol)

	CovarianceFunc(atsr, btsr, oix)
	assert.InDelta(t, results[Covariance], out.Values[0], tol)

	CorrelationFunc(atsr, btsr, oix)
	assert.InDelta(t, results[Correlation], out.Values[0], tol)

	InvCorrelationFunc(atsr, btsr, oix)
	assert.InDelta(t, results[InvCorrelation], out.Values[0], tol)

	CrossEntropyFunc(atsr, btsr, oix)
	assert.InDelta(t, results[CrossEntropy], out.Values[0], tol)

	InnerProductFunc(atsr, btsr, oix)
	assert.InDelta(t, results[InnerProduct], out.Values[0], tol)

	CosineFunc(atsr, btsr, oix)
	assert.InDelta(t, results[Cosine], out.Values[0], tol)

	InvCosineFunc(atsr, btsr, oix)
	assert.InDelta(t, results[InvCosine], out.Values[0], tol)

	for met := Euclidean; met < MetricsN; met++ {
		Standard(met, atsr, btsr, oix)
		assert.InDelta(t, results[met], out.Values[0], tol)
	}
}
