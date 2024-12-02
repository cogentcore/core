// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"fmt"
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

func runBenchFuncs(b *testing.B, n int, fun Metrics) {
	av := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	bv := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	b.ResetTimer()
	for range b.N {
		fun.Call(av, bv)
	}
}

// 375 ns/op = fastest that DotProduct could be.
func BenchmarkFuncMulBaseline(b *testing.B) {
	n := 1000
	av := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	bv := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	b.ResetTimer()
	s := float64(0)
	for range b.N {
		for i := range n {
			s += av.Values[i] * bv.Values[i]
		}
	}
}

func runClosure(av, bv *tensor.Float64, fun func(a, b, agg float64) float64) float64 {
	// fun := func(a, b, agg float64) float64 { // note: it can inline closure if in same fun
	// 	return agg + a*b
	// }
	n := 1000
	s := float64(0)
	for i := range n {
		s = fun(av.Values[i], bv.Values[i], s) // note: Float1D here no extra cost
	}
	return s
}

// 1465 ns/op = ~4x penalty for cosure
func BenchmarkFuncMulClosure(b *testing.B) {
	n := 1000
	av := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	bv := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	b.ResetTimer()
	for range b.N {
		runClosure(av, bv, func(a, b, agg float64) float64 {
			return agg + a*b
		})
	}
}

func runClosureInterface(av, bv tensor.Tensor, fun func(a, b, agg float64) float64) float64 {
	n := 1000
	s := float64(0)
	for i := range n {
		s = fun(av.Float1D(i), bv.Float1D(i), s)
	}
	return s
}

// 3665 ns/op = going through the Tensor interface = another ~2x
func BenchmarkFuncMulClosureInterface(b *testing.B) {
	n := 1000
	av := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	bv := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	b.ResetTimer()
	for range b.N {
		runClosureInterface(av, bv, func(a, b, agg float64) float64 {
			return agg + a*b
		})
	}
}

// original pre-optimization was: 8027 ns/op = 21x slower than the MulBaseline!
func BenchmarkDotProductOut(b *testing.B) {
	n := 1000
	av := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	bv := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	ov := tensor.NewFloat64(1)
	b.ResetTimer()
	for range b.N {
		DotProductOut(av, bv, ov)
	}
}

// to run this benchmark, do:
// go test -bench BenchmarkFuncs -count 10 >bench.txt
// go install golang.org/x/perf/cmd/benchstat@latest
// benchstat -row /met -col .name bench.txt

var fns = []int{10, 20, 50, 100, 500, 1000, 10000}

// after 12/2/2024 optimizations
// goos: darwin
// goarch: arm64
// pkg: cogentcore.org/core/tensor/stats/metric
//                  │    Funcs    │
//                  │   sec/op    │
// L2Norm             1.853µ ± 1%
// SumSquares         1.878µ ± 1%
// L1Norm             1.686µ ± 1%
// Hamming            1.798µ ± 1%
// L2NormBinTol       1.906µ ± 0%
// SumSquaresBinTol   1.912µ ± 0%
// InvCosine          2.421µ ± 0%
// InvCorrelation     6.379µ ± 1%
// CrossEntropy       5.876µ ± 0%
// DotProduct         1.792µ ± 0%
// Covariance         5.914µ ± 0%
// Correlation        6.437µ ± 0%
// Cosine             2.451µ ± 0%
// geomean            2.777µ

// prior to optimization:
//                  │    Funcs    │
//                  │   sec/op    │
// L1Norm             8.283µ ± 0%
// DotProduct         8.299µ ± 0%
// L2Norm             8.457µ ± 1%
// SumSquares         8.483µ ± 1%
// L2NormBinTol       8.466µ ± 0%
// SumSquaresBinTol   8.470µ ± 0%
// Hamming            8.556µ ± 0%
// CrossEntropy       12.84µ ± 0%
// Cosine             13.91µ ± 0%
// InvCosine          14.43µ ± 0%
// Covariance         39.47µ ± 0%
// Correlation        47.15µ ± 0%
// InvCorrelation     45.48µ ± 0%
// geomean            13.80µ

// BenchmarkFuncMulBaseline: 				 376.7 ns/op
// BenchmarkFuncMulBaselineClosureArg: 	1464 ns/op

func BenchmarkFuncs(b *testing.B) {
	for met := MetricL2Norm; met < MetricsN; met++ {
		b.Run(fmt.Sprintf("met=%s", met.String()), func(b *testing.B) {
			runBenchFuncs(b, 1000, met)
		})
	}
}

func runBenchNs(b *testing.B, fun Metrics) {
	for _, n := range fns {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			runBenchFuncs(b, n, fun)
		})
	}
}

func BenchmarkNsL1Norm(b *testing.B) {
	runBenchNs(b, MetricL1Norm)
}

func BenchmarkNsCosine(b *testing.B) {
	runBenchNs(b, MetricCosine)
}

func BenchmarkNsCovariance(b *testing.B) {
	runBenchNs(b, MetricCovariance)
}

func BenchmarkNsCorrelation(b *testing.B) {
	runBenchNs(b, MetricCorrelation)
}
