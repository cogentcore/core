// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"math"
	"testing"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"github.com/stretchr/testify/assert"
)

func TestFuncs(t *testing.T) {
	a64 := []float64{.5, .2, .1, .7, math.NaN(), .5}
	b64 := []float64{.2, .9, .1, .7, 0, .2}

	results := []float64{math.Sqrt(0.67), 0.67, 1.3, 3, 0.7, 0.49, 1 - 0.7319115529256469, 1 - 0.11189084777289171, 1.8090248566170337, 0.88, 0.008, 0.11189084777289171, 0.7319115529256469}

	tol := 1.0e-8

	atsr := tensor.NewNumberFromValues(a64...)
	btsr := tensor.NewNumberFromValues(b64...)
	out := tensor.NewFloat64(1)

	L2NormOut(atsr, btsr, out)
	assert.InDelta(t, results[MetricL2Norm], out.Values[0], tol)

	SumSquaresOut(atsr, btsr, out)
	assert.InDelta(t, results[MetricSumSquares], out.Values[0], tol)

	L2NormBinTolOut(atsr, btsr, out)
	assert.InDelta(t, results[MetricL2NormBinTol], out.Values[0], tol)

	L1NormOut(atsr, btsr, out)
	assert.InDelta(t, results[MetricL1Norm], out.Values[0], tol)

	HammingOut(atsr, btsr, out)
	assert.Equal(t, results[MetricHamming], out.Values[0])

	SumSquaresBinTolOut(atsr, btsr, out)
	assert.InDelta(t, results[MetricSumSquaresBinTol], out.Values[0], tol)

	CovarianceOut(atsr, btsr, out)
	assert.InDelta(t, results[MetricCovariance], out.Values[0], tol)

	CorrelationOut(atsr, btsr, out)
	assert.InDelta(t, results[MetricCorrelation], out.Values[0], tol)

	InvCorrelationOut(atsr, btsr, out)
	assert.InDelta(t, results[MetricInvCorrelation], out.Values[0], tol)

	CrossEntropyOut(atsr, btsr, out)
	assert.InDelta(t, results[MetricCrossEntropy], out.Values[0], tol)

	DotProductOut(atsr, btsr, out)
	assert.InDelta(t, results[MetricDotProduct], out.Values[0], tol)

	CosineOut(atsr, btsr, out)
	assert.InDelta(t, results[MetricCosine], out.Values[0], tol)

	InvCosineOut(atsr, btsr, out)
	assert.InDelta(t, results[MetricInvCosine], out.Values[0], tol)

	for met := MetricL2Norm; met < MetricsN; met++ {
		out := met.Call(atsr, btsr)
		assert.InDelta(t, results[met], out.Float1D(0), tol)
	}
}

func TestMatrix(t *testing.T) {

	simres := []float64{0, 3.464101552963257, 8.83176040649414, 9.273618698120117, 8.717798233032227, 9.380831718444824, 4.690415859222412, 5.830951690673828, 8.124038696289062, 8.5440034866333, 5.291502475738525, 6.324555397033691}

	dt := table.New()
	err := dt.OpenCSV("../cluster/testdata/faces.dat", tensor.Tab)
	assert.NoError(t, err)
	in := dt.Column("Input")
	out := tensor.NewFloat64()
	err = MatrixOut(L2Norm, in, out)
	assert.NoError(t, err)
	// fmt.Println(out.Tensor)
	for i, v := range simres {
		assert.InDelta(t, v, out.Float1D(i), 1.0e-8)
	}
}
