// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pca

import (
	"fmt"
	"math"
	"testing"

	"cogentcore.org/core/tensor/stats/metric"
	"cogentcore.org/core/tensor/table"
	"gonum.org/v1/gonum/mat"
)

func TestSVDIris(t *testing.T) {
	// note: these results are verified against this example:
	// https://plot.ly/ipython-notebooks/principal-component-analysis/

	dt := table.NewTable()
	dt.AddFloat64TensorColumn("data", []int{4})
	dt.AddStringColumn("class")
	err := dt.OpenCSV("testdata/iris.data", table.Comma)
	if err != nil {
		t.Error(err)
	}
	ix := table.NewIndexView(dt)
	pc := &SVD{}
	pc.Init()
	pc.Kind = mat.SVDFull
	// pc.TableColumn(ix, "data", metric.Covariance64)
	// fmt.Printf("covar: %v\n", pc.Covar)
	err = pc.TableColumn(ix, "data", metric.Correlation64)
	if err != nil {
		t.Error(err)
	}
	// fmt.Printf("correl: %v\n", pc.Covar)
	// fmt.Printf("correl vec: %v\n", pc.Vectors)
	// fmt.Printf("correl val: %v\n", pc.Values)

	errtol := 1.0e-9
	corvals := []float64{2.910818083752054, 0.9212209307072254, 0.14735327830509573, 0.020607707235624825}
	for i, v := range pc.Values {
		dif := math.Abs(corvals[i] - v)
		if dif > errtol {
			err = fmt.Errorf("eigenvalue: %v  differs from correct: %v  was:  %v", i, corvals[i], v)
			t.Error(err)
		}
	}

	prjt := &table.Table{}
	err = pc.ProjectColumnToTable(prjt, ix, "data", "class", []int{0, 1})
	if err != nil {
		t.Error(err)
	}
	// prjt.SaveCSV("test_data/svd_projection01.csv", table.Comma, true)
}
