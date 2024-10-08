// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"github.com/stretchr/testify/assert"
)

func TestCovarIris(t *testing.T) {
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
	// fmt.Printf("covar: %s\n", covar.String())
	// tensor.SaveCSV(covar, "testdata/iris-covar.tsv", tensor.Tab)

	cv := []float64{1, -0.10936924995064935, 0.8717541573048719, 0.8179536333691635,
		-0.10936924995064935, 1, -0.4205160964011548, -0.3565440896138057,
		0.8717541573048719, -0.4205160964011548, 1, 0.9627570970509667,
		0.8179536333691635, -0.3565440896138057, 0.9627570970509667, 1}

	tolassert.EqualTolSlice(t, cv, covar.Values, 1.0e-8)
}
