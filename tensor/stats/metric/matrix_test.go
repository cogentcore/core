// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"fmt"
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

func runBenchCovar(b *testing.B, n int, thread bool) {
	if thread {
		tensor.ThreadingThreshold = 1
	} else {
		tensor.ThreadingThreshold = 100_000_000
	}
	nrows := 10
	av := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, nrows*n*n+1), nrows, n, n))
	ov := tensor.NewFloat64(nrows, n, n)
	b.ResetTimer()
	for range b.N {
		CovarianceMatrixOut(Correlation, av, ov)
	}
}

// to run this benchmark, do:
// go test -bench BenchmarkCovar -count 10 >bench.txt
// go install golang.org/x/perf/cmd/benchstat@latest
// benchstat -row /n -col .name bench.txt

var ns = []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 20, 40}

func BenchmarkCovarThreaded(b *testing.B) {
	for _, n := range ns {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			runBenchCovar(b, n, true)
		})
	}
}

func BenchmarkCovarSingle(b *testing.B) {
	for _, n := range ns {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			runBenchCovar(b, n, false)
		})
	}
}
