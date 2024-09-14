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

	atsr := tensor.NewIndexed(tensor.NewNumberFromSlice(a64))
	btsr := tensor.NewIndexed(tensor.NewNumberFromSlice(b64))
	out := tensor.NewFloat64(1)
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
		Metric(met, atsr, btsr, oix)
		assert.InDelta(t, results[met], out.Values[0], tol)
	}
}

func TestMatrix(t *testing.T) {
	var simres = `Tensor: [12, 12]
[0]:       0 3.4641016151377544 8.831760866327848 9.273618495495704 8.717797887081348 9.38083151964686 4.69041575982343 5.830951894845301 8.12403840463596 8.54400374531753 5.291502622129181 6.324555320336759 
[1]: 3.4641016151377544       0 9.38083151964686 8.717797887081348 9.273618495495704 8.831760866327848 5.830951894845301 4.69041575982343 8.717797887081348 7.937253933193772 6.324555320336759 5.291502622129181 
[2]: 8.831760866327848 9.38083151964686       0 3.4641016151377544 4.242640687119285 5.0990195135927845 9.38083151964686 9.899494936611665 4.47213595499958 5.744562646538029 9.38083151964686 9.899494936611665 
[3]: 9.273618495495704 8.717797887081348 3.4641016151377544       0 5.477225575051661 3.7416573867739413 9.797958971132712 9.273618495495704 5.656854249492381 4.58257569495584 9.797958971132712 9.273618495495704 
[4]: 8.717797887081348 9.273618495495704 4.242640687119285 5.477225575051661       0       4 8.831760866327848 9.38083151964686 4.242640687119285 5.5677643628300215 8.831760866327848 9.38083151964686 
[5]: 9.38083151964686 8.831760866327848 5.0990195135927845 3.7416573867739413       4       0 9.486832980505138 8.94427190999916 5.830951894845301 4.795831523312719 9.486832980505138 8.94427190999916 
[6]: 4.69041575982343 5.830951894845301 9.38083151964686 9.797958971132712 8.831760866327848 9.486832980505138       0 3.4641016151377544 9.16515138991168 9.539392014169456 4.242640687119285 5.477225575051661 
[7]: 5.830951894845301 4.69041575982343 9.899494936611665 9.273618495495704 9.38083151964686 8.94427190999916 3.4641016151377544       0 9.695359714832659       9 5.477225575051661 4.242640687119285 
[8]: 8.12403840463596 8.717797887081348 4.47213595499958 5.656854249492381 4.242640687119285 5.830951894845301 9.16515138991168 9.695359714832659       0 3.605551275463989 9.16515138991168 9.695359714832659 
[9]: 8.54400374531753 7.937253933193772 5.744562646538029 4.58257569495584 5.5677643628300215 4.795831523312719 9.539392014169456       9 3.605551275463989       0 9.539392014169456       9 
[10]: 5.291502622129181 6.324555320336759 9.38083151964686 9.797958971132712 8.831760866327848 9.486832980505138 4.242640687119285 5.477225575051661 9.16515138991168 9.539392014169456       0 3.4641016151377544 
[11]: 6.324555320336759 5.291502622129181 9.899494936611665 9.273618495495704 9.38083151964686 8.94427190999916 5.477225575051661 4.242640687119285 9.695359714832659       9 3.4641016151377544       0 
`
	dt := table.NewTable()
	err := dt.OpenCSV("../cluster/testdata/faces.dat", tensor.Tab)
	assert.NoError(t, err)
	in := dt.Column("Input")
	out := tensor.NewFloat64Indexed()
	Matrix(Euclidean.FuncName(), in, out)
	// fmt.Println(out.Tensor)
	assert.Equal(t, simres, out.Tensor.String())
}

func TestPCAIris(t *testing.T) {
	// note: these results are verified against this example:
	// https://plot.ly/ipython-notebooks/principal-component-analysis/

	dt := table.NewTable()
	dt.AddFloat64TensorColumn("data", 4)
	dt.AddStringColumn("class")
	err := dt.OpenCSV("testdata/iris.data", tensor.Comma)
	if err != nil {
		t.Error(err)
	}
	data := dt.Column("data")
	covar := tensor.NewFloat64Indexed()
	CovarMatrix(Correlation.FuncName(), data, covar)
	// fmt.Printf("correl: %s\n", covar.Tensor.String())

	vecs := tensor.NewFloat64Indexed()
	vals := tensor.NewFloat64Indexed()
	PCA(covar, vecs, vals)

	// fmt.Printf("correl vec: %v\n", vecs)
	// fmt.Printf("correl val: %v\n", vals)
	errtol := 1.0e-9
	corvals := []float64{0.020607707235624825, 0.14735327830509573, 0.9212209307072254, 2.910818083752054}
	for i, v := range vals.Tensor.(*tensor.Float64).Values {
		assert.InDelta(t, corvals[i], v, errtol)
	}

	colidx := tensor.NewFloat64Scalar(3) // strongest at end
	prjns := tensor.NewFloat64Indexed()
	ProjectOnMatrixColumn(vecs, data, colidx, prjns)
	// tensor.SaveCSV(prjns.Tensor, "testdata/pca_projection.csv", tensor.Comma)
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
	assert.Equal(t, trgprjns, tensor.AsFloat64(prjns.Tensor).Values[:10])

	////////////////////////////////////////////////////////////
	//  	SVD

	SVD(covar, vecs, vals)
	// fmt.Printf("correl vec: %v\n", vecs)
	// fmt.Printf("correl val: %v\n", vals)
	for i, v := range vals.Tensor.(*tensor.Float64).Values {
		assert.InDelta(t, corvals[3-i], v, errtol) // opposite order
	}

	colidx.SetFloat1D(0, 0) // strongest at start
	ProjectOnMatrixColumn(vecs, data, colidx, prjns)
	// tensor.SaveCSV(prjns.Tensor, "testdata/svd_projection.csv", tensor.Comma)
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
	assert.Equal(t, trgprjns, tensor.AsFloat64(prjns.Tensor).Values[:10])
}
