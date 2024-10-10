// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"testing"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"github.com/stretchr/testify/assert"
)

func TestPCAIris(t *testing.T) {
	dt := table.New()
	dt.AddFloat64Column("data", 4)
	dt.AddStringColumn("class")
	err := dt.OpenCSV("testdata/iris.data", tensor.Comma)
	if err != nil {
		t.Error(err)
	}
	data := dt.Column("data")

	covar := tensor.NewFloat64(4, 4)
	tensor.OpenCSV(covar, "testdata/iris-covar.tsv", tensor.Tab)

	vecs, vals := EigSym(covar)

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

	err = SVDOut(covar, vecs, vals)
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
