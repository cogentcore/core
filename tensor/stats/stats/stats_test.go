// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"fmt"
	"math"
	"testing"

	"cogentcore.org/core/tensor"
	"github.com/stretchr/testify/assert"
)

func TestFuncs64(t *testing.T) {
	vals := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}
	ix := tensor.NewNumberFromValues(vals...)
	out := tensor.NewFloat64(1)

	results := []float64{11, 5.5, 5.5, 0, 0, 1, 0, 1, 0.5, 0.11, math.Sqrt(0.11), math.Sqrt(0.11) / math.Sqrt(11), 3.85, math.Sqrt(3.85), 0.1, math.Sqrt(0.1), math.Sqrt(0.1) / math.Sqrt(11), 0.5, 0.25, 0.75}

	tol := 1.0e-8

	CountOut(ix, out)
	assert.Equal(t, results[StatCount], out.Values[0])

	SumOut(ix, out)
	assert.Equal(t, results[StatSum], out.Values[0])

	L1NormOut(ix, out)
	assert.Equal(t, results[StatL1Norm], out.Values[0])

	ProdOut(ix, out)
	assert.Equal(t, results[StatProd], out.Values[0])

	MinOut(ix, out)
	assert.Equal(t, results[StatMin], out.Values[0])

	MaxOut(ix, out)
	assert.Equal(t, results[StatMax], out.Values[0])

	MinAbsOut(ix, out)
	assert.Equal(t, results[StatMinAbs], out.Values[0])

	MaxAbsOut(ix, out)
	assert.Equal(t, results[StatMaxAbs], out.Values[0])

	MeanOut(ix, out)
	assert.Equal(t, results[StatMean], out.Values[0])

	VarOut(ix, out)
	assert.InDelta(t, results[StatVar], out.Values[0], tol)

	StdOut(ix, out)
	assert.InDelta(t, results[StatStd], out.Values[0], tol)

	SemOut(ix, out)
	assert.InDelta(t, results[StatSem], out.Values[0], tol)

	VarPopOut(ix, out)
	assert.InDelta(t, results[StatVarPop], out.Values[0], tol)

	StdPopOut(ix, out)
	assert.InDelta(t, results[StatStdPop], out.Values[0], tol)

	SemPopOut(ix, out)
	assert.InDelta(t, results[StatSemPop], out.Values[0], tol)

	SumSqOut(ix, out)
	assert.InDelta(t, results[StatSumSq], out.Values[0], tol)

	L2NormOut(ix, out)
	assert.InDelta(t, results[StatL2Norm], out.Values[0], tol)

	MedianOut(ix, out)
	assert.InDelta(t, results[StatMedian], out.Values[0], tol)

	Q1Out(ix, out)
	assert.InDelta(t, results[StatQ1], out.Values[0], tol)

	Q3Out(ix, out)
	assert.InDelta(t, results[StatQ3], out.Values[0], tol)

	for stat := StatCount; stat < StatsN; stat++ {
		out := stat.Call(ix)
		assert.InDelta(t, results[stat], out.Float1D(0), tol)
	}
}

func TestFuncsInt(t *testing.T) {
	vals := []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	tsr := tensor.NewNumberFromValues(vals...)
	ix := tensor.NewRows(tsr)
	out := tensor.NewInt(1)

	results := []int{11, 550, 550, 0, 0, 100, 0, 100, 50, 1100, int(math.Sqrt(1100)), int(math.Sqrt(1100) / math.Sqrt(11)), 38500, 196, 1000, int(math.Sqrt(1000)), int(math.Sqrt(1000) / math.Sqrt(11))}

	CountOut(ix, out)
	assert.Equal(t, results[StatCount], out.Values[0])

	SumOut(ix, out)
	assert.Equal(t, results[StatSum], out.Values[0])

	L1NormOut(ix, out)
	assert.Equal(t, results[StatL1Norm], out.Values[0])

	ProdOut(ix, out)
	assert.Equal(t, results[StatProd], out.Values[0])

	MinOut(ix, out)
	assert.Equal(t, results[StatMin], out.Values[0])

	MaxOut(ix, out)
	assert.Equal(t, results[StatMax], out.Values[0])

	MinAbsOut(ix, out)
	assert.Equal(t, results[StatMinAbs], out.Values[0])

	MaxAbsOut(ix, out)
	assert.Equal(t, results[StatMaxAbs], out.Values[0])

	MeanOut(ix, out)
	assert.Equal(t, results[StatMean], out.Values[0])

	VarOut(ix, out)
	assert.Equal(t, results[StatVar], out.Values[0])

	StdOut(ix, out)
	assert.Equal(t, results[StatStd], out.Values[0])

	SemOut(ix, out)
	assert.Equal(t, results[StatSem], out.Values[0])

	VarPopOut(ix, out)
	assert.Equal(t, results[StatVarPop], out.Values[0])

	StdPopOut(ix, out)
	assert.Equal(t, results[StatStdPop], out.Values[0])

	SemPopOut(ix, out)
	assert.Equal(t, results[StatSemPop], out.Values[0])

	SumSqOut(ix, out)
	assert.Equal(t, results[StatSumSq], out.Values[0])

	L2NormOut(ix, out)
	assert.Equal(t, results[StatL2Norm], out.Values[0])

	for stat := StatCount; stat <= StatSemPop; stat++ {
		out := stat.Call(ix)
		assert.Equal(t, results[stat], out.Int1D(0))
	}
}

func TestFuncsCell(t *testing.T) {
	vals := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9}
	tsr := tensor.NewFloat32(20, 10)

	for i := range 20 {
		for j := range 10 {
			tsr.SetFloatRow(vals[j], i, j)
		}
	}

	ix := tensor.NewRows(tsr)
	out := tensor.NewFloat32(20, 10)

	CountOut(ix, out)
	nsub := out.Len()
	for i := range nsub {
		assert.Equal(t, 20.0, out.FloatRow(0, i))
	}
	MeanOut(ix, out)
	for i := range nsub {
		assert.InDelta(t, vals[i], out.FloatRow(0, i), 1.0e-7) // lower tol, using float32
	}
	VarOut(ix, out)
	for i := range nsub {
		assert.InDelta(t, 0.0, out.FloatRow(0, i), 1.0e-7)
	}
}

func TestNorm(t *testing.T) {
	vals := []float64{-1.507556722888818, -1.2060453783110545, -0.9045340337332908, -0.6030226891555273, -0.3015113445777635, 0.1, 0.3015113445777635, 0.603022689155527, 0.904534033733291, 1.2060453783110545, 1.507556722888818, .3}

	oned := tensor.NewNumberFromValues(vals...)
	oneout := oned.Clone()

	ZScoreOut(oned, oneout)
	mout := tensor.NewFloat64()
	std, mean, _ := StdOut64(oneout, mout)
	assert.InDelta(t, 1.0, std.Float1D(0), 1.0e-6)
	assert.InDelta(t, 0.0, mean.Float1D(0), 1.0e-6)

	UnitNormOut(oned, oneout)
	MinOut(oneout, mout)
	assert.InDelta(t, 0.0, mout.Float1D(0), 1.0e-6)
	MaxOut(oneout, mout)
	assert.InDelta(t, 1.0, mout.Float1D(0), 1.0e-6)
	// fmt.Println(oneout)

	minv := tensor.NewFloat64Scalar(0)
	maxv := tensor.NewFloat64Scalar(1)
	ClampOut(oned, minv, maxv, oneout)
	MinOut(oneout, mout)
	assert.InDelta(t, 0.0, mout.Float1D(0), 1.0e-6)
	MaxOut(oneout, mout)
	assert.InDelta(t, 1.0, mout.Float1D(0), 1.0e-6)
	// fmt.Println(oneout)

	thr := tensor.NewFloat64Scalar(0.5)
	BinarizeOut(oned, thr, oneout)
	MinOut(oneout, mout)
	assert.InDelta(t, 0.0, mout.Float1D(0), 1.0e-6)
	MaxOut(oneout, mout)
	assert.InDelta(t, 1.0, mout.Float1D(0), 1.0e-6)
	// fmt.Println(oneout)
}

// after optimizing: 12/1/2024:  also, GOEXPERIMENT=newinliner didn't make any diff
// go test -bench BenchmarkFuncs -count=1
// goos: darwin
// goarch: arm64
// pkg: cogentcore.org/core/tensor/stats/stats
// stat=Count-16         	  677764	      1789 ns/op
// stat=Sum-16           	  668791	      1809 ns/op
// stat=L1Norm-16        	  821071	      1484 ns/op
// stat=Prod-16          	  703598	      1706 ns/op
// stat=Min-16           	  182587	      6564 ns/op
// stat=Max-16           	  181981	      6577 ns/op
// stat=MinAbs-16        	  176342	      6787 ns/op
// stat=MaxAbs-16        	  175491	      6784 ns/op
// stat=Mean-16          	  592713	      2014 ns/op
// stat=Var-16           	  330260	      3620 ns/op
// stat=Std-16           	  329876	      3625 ns/op
// stat=Sem-16           	  330141	      3629 ns/op
// stat=SumSq-16         	  366603	      3267 ns/op
// stat=L2Norm-16        	  366862	      3264 ns/op
// stat=VarPop-16        	  330362	      3617 ns/op
// stat=StdPop-16        	  329172	      3626 ns/op
// stat=SemPop-16        	  331568	      3631 ns/op
// stat=Median-16        	  116071	     10316 ns/op
// stat=Q1-16            	  116175	     10334 ns/op
// stat=Q3-16            	  115149	     10331 ns/op

// old: prior to optimizing 12/1/2024:
// stat=Count-16         	  166908	      7189 ns/op
// stat=Sum-16           	  166287	      7198 ns/op
// stat=L1Norm-16        	  166587	      7195 ns/op
// stat=Prod-16          	  166029	      7185 ns/op
// stat=Min-16           	  125803	      9523 ns/op
// stat=Max-16           	  125067	      9561 ns/op
// stat=MinAbs-16        	  126109	      9524 ns/op
// stat=MaxAbs-16        	  126346	      9500 ns/op
// stat=Mean-16          	   83302	     14365 ns/op
// stat=Var-16           	   53138	     22707 ns/op
// stat=Std-16           	   53073	     22611 ns/op
// stat=Sem-16           	   52928	     22611 ns/op
// stat=SumSq-16         	  125698	      9486 ns/op
// stat=L2Norm-16        	  126196	      9483 ns/op
// stat=VarPop-16        	   53010	     22659 ns/op
// stat=StdPop-16        	   52994	     22573 ns/op
// stat=SemPop-16        	   52897	     22726 ns/op
// stat=Median-16        	  116223	     10334 ns/op
// stat=Q1-16            	  115728	     10431 ns/op
// stat=Q3-16            	  111325	     10307 ns/op

func runBenchFuncs(b *testing.B, n int, fun Stats) {
	av := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	b.ResetTimer()
	for range b.N {
		fun.Call(av)
	}
}

func BenchmarkFuncs(b *testing.B) {
	for stf := StatCount; stf < StatsN; stf++ {
		b.Run(fmt.Sprintf("stat=%s", stf.String()), func(b *testing.B) {
			runBenchFuncs(b, 1000, stf)
		})
	}
}

// 258.6 ns/op, vs 1809 actual
func BenchmarkSumBaseline(b *testing.B) {
	n := 1000
	av := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	b.ResetTimer()
	for range b.N {
		sum := float64(0)
		for i := range n {
			val := av.Float1D(i)
			if math.IsNaN(val) {
				continue
			}
			sum += val
		}
		_ = sum
	}
}

func runClosure(av *tensor.Float64, fun func(a, agg float64) float64) float64 {
	// fun := func(a, agg float64) float64 { // note: it can inline closure if in same fun
	// 	return agg + a
	// }
	n := 1000
	s := float64(0)
	for i := range n {
		s = fun(av.Float1D(i), s) // note: Float1D here no extra cost
	}
	return s
}

// 1242 ns/op, vs 1809 actual -- mostly it is the closure
func BenchmarkSumClosure(b *testing.B) {
	n := 1000
	av := tensor.AsFloat64(tensor.Reshape(tensor.NewIntRange(1, n+1), n))
	b.ResetTimer()
	for range b.N {
		runClosure(av, func(val, agg float64) float64 {
			if math.IsNaN(val) {
				return agg
			}
			return agg + val
		})
	}
}
