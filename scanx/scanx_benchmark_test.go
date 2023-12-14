// Copyright 2019 The oksvg Authors. All rights reserved.
// created: 2018 by S.R.Wiley
package scanx_test

import (
	"image"

	"testing"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"github.com/srwiley/scanFT"
	"github.com/srwiley/scanx"
)

func ReadIconSet(paths []string) (icons []*oksvg.SvgIcon) {
	for _, p := range paths {
		icon, errSvg := oksvg.ReadIcon(p, oksvg.IgnoreErrorMode)
		if errSvg == nil {
			icons = append(icons, icon)
		}
	}
	return
}

func BenchmarkLinkListSpanner5(b *testing.B) {
	RunLinkListSpanner(b, 5)
}

func BenchmarkImgSpanner5(b *testing.B) {
	RunImgSpanner(b, 5)
}

func BenchmarkFTScanner5(b *testing.B) {
	RunFTScanner(b, 5)
}

func BenchmarkGVScanner5(b *testing.B) {
	RunGVScanner(b, 5)
}

func BenchmarkLinkListSpanner10(b *testing.B) {
	RunLinkListSpanner(b, 10)
}

func BenchmarkImgSpanner10(b *testing.B) {
	RunImgSpanner(b, 10)
}

func BenchmarkFTScanner10(b *testing.B) {
	RunFTScanner(b, 10)
}
func BenchmarkGVScanner10(b *testing.B) {
	RunGVScanner(b, 10)
}

func BenchmarkLinkListSpanner50(b *testing.B) {
	RunLinkListSpanner(b, 50)
}

func BenchmarkImgSpanner50(b *testing.B) {
	RunImgSpanner(b, 50)
}
func BenchmarkFTScanner50(b *testing.B) {
	RunFTScanner(b, 50)
}
func BenchmarkGVScanner50(b *testing.B) {
	RunGVScanner(b, 50)
}

func BenchmarkLinkListSpanner150(b *testing.B) {
	RunLinkListSpanner(b, 150)
}

func BenchmarkImgSpanner150(b *testing.B) {
	RunImgSpanner(b, 150)
}
func BenchmarkFTScanner150(b *testing.B) {
	RunFTScanner(b, 150)
}
func BenchmarkGVScanner150(b *testing.B) {
	RunGVScanner(b, 150)
}

func RunLinkListSpanner(b *testing.B, mult int) {
	beachIconNames, err := FilePathWalkDir("testdata/svg/landscapeIcons")
	if err != nil {
		b.Log("cannot walk file path testdata/svg")
		b.FailNow()
	}
	var (
		beachIcons  = ReadIconSet(beachIconNames)
		wi, hi      = int(beachIcons[0].ViewBox.W), int(beachIcons[0].ViewBox.H)
		w, h        = wi * mult / 10, hi * mult / 10
		bounds      = image.Rect(0, 0, w, h)
		img         = image.NewRGBA(bounds)
		spannerC    = &scanx.LinkListSpanner{}
		scannerC    = scanx.NewScanner(spannerC, w, h)
		rasterScanC = rasterx.NewDasher(w, h, scannerC)
	)
	spannerC.SetBounds(bounds)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ic := range beachIcons {
			ic.SetTarget(0.0, 0.0, float64(bounds.Max.X), float64(bounds.Max.Y))
			ic.Draw(rasterScanC, 1.0)
			spannerC.DrawToImage(img)
			rasterScanC.Clear()
			spannerC.Clear()
		}
	}
}

func RunFTScanner(b *testing.B, mult int) {
	beachIconNames, err := FilePathWalkDir("testdata/svg/landscapeIcons")
	if err != nil {
		b.Log("cannot walk file path testdata/svg")
		b.FailNow()
	}
	var (
		beachIcons = ReadIconSet(beachIconNames)
		wi, hi     = int(beachIcons[0].ViewBox.W), int(beachIcons[0].ViewBox.H)
		w, h       = wi * mult / 10, hi * mult / 10
		bounds     = image.Rect(0, 0, w, h)
		img        = image.NewRGBA(bounds)
		painter    = scanFT.NewRGBAPainter(img)
		scanner    = scanFT.NewScannerFT(w, h, painter)
		rasterFT   = rasterx.NewDasher(w, h, scanner)
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ic := range beachIcons {
			ic.SetTarget(0.0, 0.0, float64(bounds.Max.X), float64(bounds.Max.Y))
			ic.Draw(rasterFT, 1.0)
			rasterFT.Clear()
		}
	}
}

func RunImgSpanner(b *testing.B, mult int) {
	beachIconNames, err := FilePathWalkDir("testdata/svg/landscapeIcons")
	if err != nil {
		b.Log("cannot walk file path testdata/svg")
		b.FailNow()
	}
	var (
		beachIcons  = ReadIconSet(beachIconNames)
		wi, hi      = int(beachIcons[0].ViewBox.W), int(beachIcons[0].ViewBox.H)
		w, h        = wi * mult / 10, hi * mult / 10
		bounds      = image.Rect(0, 0, w, h)
		img         = image.NewRGBA(bounds)
		spanner     = scanx.NewImgSpanner(img)
		scannerX    = scanx.NewScanner(spanner, w, h)
		rasterScanX = rasterx.NewDasher(w, h, scannerX)
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ic := range beachIcons {
			ic.SetTarget(0.0, 0.0, float64(bounds.Max.X), float64(bounds.Max.Y))
			ic.Draw(rasterScanX, 1.0)
			rasterScanX.Clear()
		}
	}
}

func RunGVScanner(b *testing.B, mult int) {
	beachIconNames, err := FilePathWalkDir("testdata/svg/landscapeIcons")
	if err != nil {
		b.Log("cannot walk file path testdata/svg")
		b.FailNow()
	}
	var (
		beachIcons = ReadIconSet(beachIconNames)
		wi, hi     = int(beachIcons[0].ViewBox.W), int(beachIcons[0].ViewBox.H)
		w, h       = wi * mult / 10, hi * mult / 10
		bounds     = image.Rect(0, 0, w, h)
		img        = image.NewRGBA(bounds)
		scannerGV  = rasterx.NewScannerGV(w, h, img, img.Bounds())
		raster     = rasterx.NewDasher(w, h, scannerGV)
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ic := range beachIcons {
			ic.SetTarget(0.0, 0.0, float64(bounds.Max.X), float64(bounds.Max.Y))
			ic.Draw(raster, 1.0)
			scannerGV.Clear()
		}
	}
}
