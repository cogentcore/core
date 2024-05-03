// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randx

import (
	"math"
	"testing"

	"cogentcore.org/core/tensor/stats/stats"
	"cogentcore.org/core/tensor/table"
)

func TestGaussianGen(t *testing.T) {
	nsamp := int(1e6)
	dt := &table.Table{}
	dt.AddFloat32Column("Val")
	dt.SetNumRows(nsamp)

	mean := 0.5
	sig := 0.25
	tol := 1e-2

	for i := 0; i < nsamp; i++ {
		vl := GaussianGen(mean, sig)
		dt.SetFloat("Val", i, vl)
	}
	ix := table.NewIndexView(dt)
	desc := stats.DescAll(ix)

	meanRow := desc.RowsByString("Stat", "Mean", table.Equals, table.UseCase)[0]
	stdRow := desc.RowsByString("Stat", "Std", table.Equals, table.UseCase)[0]
	// minRow := desc.RowsByString("Stat", "Min", table.Equals, table.UseCase)[0]
	// maxRow := desc.RowsByString("Stat", "Max", table.Equals, table.UseCase)[0]

	actMean := desc.Float("Val", meanRow)
	actStd := desc.Float("Val", stdRow)

	if math.Abs(actMean-mean) > tol {
		t.Errorf("Gaussian: mean %g\t out of tolerance vs target: %g\n", actMean, mean)
	}
	if math.Abs(actStd-sig) > tol {
		t.Errorf("Gaussian: stdev %g\t out of tolerance vs target: %g\n", actStd, sig)
	}
	// b := bytes.NewBuffer(nil)
	// desc.WriteCSV(b, table.Tab, table.Headers)
	// fmt.Printf("%s\n", string(b.Bytes()))
}

func TestBinomialGen(t *testing.T) {
	nsamp := int(1e6)
	dt := &table.Table{}
	dt.AddFloat32Column("Val")
	dt.SetNumRows(nsamp)

	n := 1.0
	p := 0.5
	tol := 1e-2

	for i := 0; i < nsamp; i++ {
		vl := BinomialGen(n, p)
		dt.SetFloat("Val", i, vl)
	}
	ix := table.NewIndexView(dt)
	desc := stats.DescAll(ix)

	meanRow := desc.RowsByString("Stat", "Mean", table.Equals, table.UseCase)[0]
	stdRow := desc.RowsByString("Stat", "Std", table.Equals, table.UseCase)[0]
	minRow := desc.RowsByString("Stat", "Min", table.Equals, table.UseCase)[0]
	maxRow := desc.RowsByString("Stat", "Max", table.Equals, table.UseCase)[0]

	actMean := desc.Float("Val", meanRow)
	actStd := desc.Float("Val", stdRow)
	actMin := desc.Float("Val", minRow)
	actMax := desc.Float("Val", maxRow)

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
	// b := bytes.NewBuffer(nil)
	// desc.WriteCSV(b, table.Tab, table.Headers)
	// fmt.Printf("%s\n", string(b.Bytes()))
}

func TestPoissonGen(t *testing.T) {
	nsamp := int(1e6)
	dt := &table.Table{}
	dt.AddFloat32Column("Val")
	dt.SetNumRows(nsamp)

	lambda := 10.0
	tol := 1e-2

	for i := 0; i < nsamp; i++ {
		vl := PoissonGen(lambda)
		dt.SetFloat("Val", i, vl)
	}
	ix := table.NewIndexView(dt)
	desc := stats.DescAll(ix)

	meanRow := desc.RowsByString("Stat", "Mean", table.Equals, table.UseCase)[0]
	stdRow := desc.RowsByString("Stat", "Std", table.Equals, table.UseCase)[0]
	minRow := desc.RowsByString("Stat", "Min", table.Equals, table.UseCase)[0]
	// maxRow := desc.RowsByString("Stat", "Max", table.Equals, table.UseCase)[0]

	actMean := desc.Float("Val", meanRow)
	actStd := desc.Float("Val", stdRow)
	actMin := desc.Float("Val", minRow)
	// actMax := desc.Float("Val", maxRow)

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
	// b := bytes.NewBuffer(nil)
	// desc.WriteCSV(b, table.Tab, table.Headers)
	// fmt.Printf("%s\n", string(b.Bytes()))
}

func TestGammaGen(t *testing.T) {
	nsamp := int(1e6)
	dt := &table.Table{}
	dt.AddFloat32Column("Val")
	dt.SetNumRows(nsamp)

	alpha := 0.5
	beta := 0.8
	tol := 1e-2

	for i := 0; i < nsamp; i++ {
		vl := GammaGen(alpha, beta)
		dt.SetFloat("Val", i, vl)
	}
	ix := table.NewIndexView(dt)
	desc := stats.DescAll(ix)

	meanRow := desc.RowsByString("Stat", "Mean", table.Equals, table.UseCase)[0]
	stdRow := desc.RowsByString("Stat", "Std", table.Equals, table.UseCase)[0]

	actMean := desc.Float("Val", meanRow)
	actStd := desc.Float("Val", stdRow)

	mean := alpha / beta
	if math.Abs(actMean-mean) > tol {
		t.Errorf("Gamma: mean %g\t out of tolerance vs target: %g\n", actMean, mean)
	}
	sig := math.Sqrt(alpha / beta / beta)
	if math.Abs(actStd-sig) > tol {
		t.Errorf("Gamma: stdev %g\t out of tolerance vs target: %g\n", actStd, sig)
	}
	// b := bytes.NewBuffer(nil)
	// desc.WriteCSV(b, table.Tab, table.Headers)
	// fmt.Printf("%s\n", string(b.Bytes()))
}

func TestBetaGen(t *testing.T) {
	nsamp := int(1e6)
	dt := &table.Table{}
	dt.AddFloat32Column("Val")
	dt.SetNumRows(nsamp)

	alpha := 0.5
	beta := 0.8
	tol := 1e-2

	for i := 0; i < nsamp; i++ {
		vl := BetaGen(alpha, beta)
		dt.SetFloat("Val", i, vl)
	}
	ix := table.NewIndexView(dt)
	desc := stats.DescAll(ix)

	meanRow := desc.RowsByString("Stat", "Mean", table.Equals, table.UseCase)[0]
	stdRow := desc.RowsByString("Stat", "Std", table.Equals, table.UseCase)[0]

	actMean := desc.Float("Val", meanRow)
	actStd := desc.Float("Val", stdRow)

	mean := alpha / (alpha + beta)
	if math.Abs(actMean-mean) > tol {
		t.Errorf("Beta: mean %g\t out of tolerance vs target: %g\n", actMean, mean)
	}
	vr := alpha * beta / ((alpha + beta) * (alpha + beta) * (alpha + beta + 1))
	sig := math.Sqrt(vr)
	if math.Abs(actStd-sig) > tol {
		t.Errorf("Beta: stdev %g\t out of tolerance vs target: %g\n", actStd, sig)
	}
	// b := bytes.NewBuffer(nil)
	// desc.WriteCSV(b, table.Tab, table.Headers)
	// fmt.Printf("%s\n", string(b.Bytes()))
}
