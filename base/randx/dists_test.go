// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randx

import (
	"math"
	"testing"

	"cogentcore.org/core/tensor/datafs"
	"cogentcore.org/core/tensor/stats/stats"
	"cogentcore.org/core/tensor/table"
)

func TestGaussianGen(t *testing.T) {
	nsamp := int(1e6)
	dt := table.NewTable()
	dt.AddFloat32Column("Val")
	dt.SetNumRows(nsamp)

	mean := 0.5
	sig := 0.25
	tol := 1e-2

	for i := 0; i < nsamp; i++ {
		vl := GaussianGen(mean, sig)
		dt.Column("Val").SetFloatRow(vl, i)
	}
	dir, _ := datafs.NewDir("Desc")
	stats.DescribeTableAll(dir, dt)
	desc := dir.GetDirTable(nil)
	// fmt.Println(desc.Columns.Keys)

	actMean := desc.Column("Val/Mean").FloatRow(0)
	actStd := desc.Column("Val/Std").FloatRow(0)

	if math.Abs(actMean-mean) > tol {
		t.Errorf("Gaussian: mean %g\t out of tolerance vs target: %g\n", actMean, mean)
	}
	if math.Abs(actStd-sig) > tol {
		t.Errorf("Gaussian: stdev %g\t out of tolerance vs target: %g\n", actStd, sig)
	}
}

func TestBinomialGen(t *testing.T) {
	nsamp := int(1e6)
	dt := table.NewTable()
	dt.AddFloat32Column("Val")
	dt.SetNumRows(nsamp)

	n := 1.0
	p := 0.5
	tol := 1e-2

	for i := 0; i < nsamp; i++ {
		vl := BinomialGen(n, p)
		dt.Column("Val").SetFloat(vl, i)
	}
	dir, _ := datafs.NewDir("Desc")
	stats.DescribeTableAll(dir, dt)
	desc := dir.GetDirTable(nil)
	actMean := desc.Column("Val/Mean").FloatRow(0)
	actStd := desc.Column("Val/Std").FloatRow(0)
	actMin := desc.Column("Val/Min").FloatRow(0)
	actMax := desc.Column("Val/Max").FloatRow(0)
	mean := n * p
	if math.Abs(actMean-mean) > tol {
		t.Errorf("Binomial: mean %g\t out of tolerance vs target: %g\n", actMean, mean)
	}
	sig := math.Sqrt(n * p * (1.0 - p))
	if math.Abs(actStd-sig) > tol {
		t.Errorf("Binomial: stdev %g\t out of tolerance vs target: %g\n", actStd, sig)
	}
	if actMin < 0 {
		t.Errorf("Binomial: min %g\t should not be < 0\n", actMin)
	}
	if actMax < 0 {
		t.Errorf("Binomial: max %g\t should not be > 1\n", actMax)
	}
}

func TestPoissonGen(t *testing.T) {
	nsamp := int(1e6)
	dt := table.NewTable()
	dt.AddFloat32Column("Val")
	dt.SetNumRows(nsamp)

	lambda := 10.0
	tol := 1e-2

	for i := 0; i < nsamp; i++ {
		vl := PoissonGen(lambda)
		dt.Column("Val").SetFloatRow(vl, i)
	}
	dir, _ := datafs.NewDir("Desc")
	stats.DescribeTableAll(dir, dt)
	desc := dir.GetDirTable(nil)
	actMean := desc.Column("Val/Mean").FloatRow(0)
	actStd := desc.Column("Val/Std").FloatRow(0)
	actMin := desc.Column("Val/Min").FloatRow(0)
	// actMax := desc.Column("Val/Max").FloatRow(0)

	mean := lambda
	if math.Abs(actMean-mean) > tol {
		t.Errorf("Poisson: mean %g\t out of tolerance vs target: %g\n", actMean, mean)
	}
	sig := math.Sqrt(lambda)
	if math.Abs(actStd-sig) > tol {
		t.Errorf("Poisson: stdev %g\t out of tolerance vs target: %g\n", actStd, sig)
	}
	if actMin < 0 {
		t.Errorf("Poisson: min %g\t should not be < 0\n", actMin)
	}
	// if actMax < 0 {
	// 	t.Errorf("Poisson: max %g\t should not be > 1\n", actMax)
	// }
}

func TestGammaGen(t *testing.T) {
	nsamp := int(1e6)
	dt := table.NewTable()
	dt.AddFloat32Column("Val")
	dt.SetNumRows(nsamp)

	alpha := 0.5
	beta := 0.8
	tol := 1e-2

	for i := 0; i < nsamp; i++ {
		vl := GammaGen(alpha, beta)
		dt.Column("Val").SetFloatRow(vl, i)
	}
	dir, _ := datafs.NewDir("Desc")
	stats.DescribeTableAll(dir, dt)
	desc := dir.GetDirTable(nil)
	actMean := desc.Column("Val/Mean").FloatRow(0)
	actStd := desc.Column("Val/Std").FloatRow(0)
	// actMin := desc.Column("Val/Min").FloatRow(0)
	// actMax := desc.Column("Val/Max").FloatRow(0)
	mean := alpha / beta
	if math.Abs(actMean-mean) > tol {
		t.Errorf("Gamma: mean %g\t out of tolerance vs target: %g\n", actMean, mean)
	}
	sig := math.Sqrt(alpha / beta / beta)
	if math.Abs(actStd-sig) > tol {
		t.Errorf("Gamma: stdev %g\t out of tolerance vs target: %g\n", actStd, sig)
	}
}

func TestBetaGen(t *testing.T) {
	nsamp := int(1e6)
	dt := table.NewTable()
	dt.AddFloat32Column("Val")
	dt.SetNumRows(nsamp)

	alpha := 0.5
	beta := 0.8
	tol := 1e-2

	for i := 0; i < nsamp; i++ {
		vl := BetaGen(alpha, beta)
		dt.Column("Val").SetFloatRow(vl, i)
	}
	dir, _ := datafs.NewDir("Desc")
	stats.DescribeTableAll(dir, dt)
	desc := dir.GetDirTable(nil)
	actMean := desc.Column("Val/Mean").FloatRow(0)
	actStd := desc.Column("Val/Std").FloatRow(0)
	// actMin := desc.Column("Val/Min").FloatRow(0)
	// actMax := desc.Column("Val/Max").FloatRow(0)
	mean := alpha / (alpha + beta)
	if math.Abs(actMean-mean) > tol {
		t.Errorf("Beta: mean %g\t out of tolerance vs target: %g\n", actMean, mean)
	}
	vr := alpha * beta / ((alpha + beta) * (alpha + beta) * (alpha + beta + 1))
	sig := math.Sqrt(vr)
	if math.Abs(actStd-sig) > tol {
		t.Errorf("Beta: stdev %g\t out of tolerance vs target: %g\n", actStd, sig)
	}
}
