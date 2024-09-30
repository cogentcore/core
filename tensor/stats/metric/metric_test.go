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

func TestPCAIris(t *testing.T) {
	// note: these results are verified against this example:
	// https://plot.ly/ipython-notebooks/principal-component-analysis/

	dt := table.New()
	dt.AddFloat64Column("data", 4)
	dt.AddStringColumn("class")
	err := dt.OpenCSV("testdata/iris.data", tensor.Comma)
	if err != nil {
		t.Error(err)
	}
	data := dt.Column("data")
	covar := tensor.NewFloat64()
	err = CovarianceMatrixOut(Correlation, data, covar)
	assert.NoError(t, err)
	// fmt.Printf("correl: %s\n", covar.String())

	vecs := tensor.NewFloat64()
	vals := tensor.NewFloat64()
	PCA(covar, vecs, vals)

	// fmt.Printf("correl vec: %v\n", vecs)
	// fmt.Printf("correl val: %v\n", vals)
	errtol := 1.0e-9
	corvals := []float64{0.020607707235624825, 0.14735327830509573, 0.9212209307072254, 2.910818083752054}
	for i, v := range vals.Values {
		assert.InDelta(t, corvals[i], v, errtol)
	}

	colidx := tensor.NewFloat64Scalar(3) // strongest at end
	prjns := tensor.NewFloat64()
	err = ProjectOnMatrixColumnOut(vecs, data, colidx, prjns)
	assert.NoError(t, err)
	// tensor.SaveCSV(prjns, "testdata/pca_projection.csv", tensor.Comma)
	trgprjns := []float64{
		2.6692308782935146,
		2.696434011868953,
		2.4811633041648684,
		2.5715124347750256,
		2.5906582247213543,
		3.0080988099460613,
		2.490941664609344,
		2.7014546083439073,
		2.4615836931965167,
		2.6716628159090594,
	}
	for i, v := range prjns.Values[:10] {
		assert.InDelta(t, trgprjns[i], v, errtol)
	}

	////////////////////////////////////////////////////////////
	//  	SVD

	err = SVD(covar, vecs, vals)
	assert.NoError(t, err)
	// fmt.Printf("correl vec: %v\n", vecs)
	// fmt.Printf("correl val: %v\n", vals)
	for i, v := range vals.Values {
		assert.InDelta(t, corvals[3-i], v, errtol) // opposite order
	}

	colidx.SetFloat1D(0, 0) // strongest at start
	err = ProjectOnMatrixColumnOut(vecs, data, colidx, prjns)
	assert.NoError(t, err)
	// tensor.SaveCSV(prjns, "testdata/svd_projection.csv", tensor.Comma)
	trgprjns = []float64{
		-2.6692308782935172,
		-2.696434011868955,
		-2.48116330416487,
		-2.5715124347750273,
		-2.590658224721357,
		-3.008098809946064,
		-2.4909416646093456,
		-2.70145460834391,
		-2.4615836931965185,
		-2.671662815909061,
	}
	for i, v := range prjns.Values[:10] {
		assert.InDelta(t, trgprjns[i], v, errtol)
	}
}
