// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image"
	"image/draw"
	"testing"

	"goki.dev/cam/hct"
	"goki.dev/grows/images"
)

func TestSpacedLight(t *testing.T) {
	hues := []float32{25, 255, 150, 105, 340, 210, 60, 300}
	ncats := 8
	nX := ncats
	nY := 5
	mx := nY * nX

	ysp := 8
	xsp := 8
	lnY := 8
	spY := lnY + ysp
	lnX := 40
	spX := lnX + xsp
	sz := image.Point{spX*nX + 2*xsp, spY*nY + 3*ysp}
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), &image.Uniform{White}, image.Point{}, draw.Src)

	for idx := 0; idx < mx; idx++ {
		c := Spaced(idx)
		yp := idx / nX
		xp := idx % nX
		ys := yp*spY + ysp
		xs := xp*spX + xsp
		for y := 0; y < lnY; y++ {
			for x := 0; x < lnX; x++ {
				img.SetRGBA(xs+x, ys+y, c)
			}
		}
	}

	chroma := float32(90)
	tone := float32(65)
	yp := nY
	for hue := 0; hue < 360; hue++ {
		c := hct.New(float32(hue), chroma, tone).AsRGBA()

		ys := yp*spY + ysp
		xs := hue + xsp
		for y := 0; y < spY; y++ {
			img.SetRGBA(xs, ys+y, c)
		}
		for _, h := range hues {
			if int(h) == hue {
				img.SetRGBA(xs, ys-1, Black)
			}
		}
	}

	images.Assert(t, img, "spacedlight")
}

func TestSpacedDark(t *testing.T) {
	hues := []float32{25, 255, 150, 105, 340, 210, 60, 300}
	ncats := 8
	nX := ncats
	nY := 5
	mx := nY * nX

	ysp := 8
	xsp := 8
	lnY := 8
	spY := lnY + ysp
	lnX := 40
	spX := lnX + xsp
	sz := image.Point{spX*nX + 2*xsp, spY*nY + 3*ysp}
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), &image.Uniform{Black}, image.Point{}, draw.Src)

	for idx := 0; idx < mx; idx++ {
		c := Spaced(idx)
		yp := idx / nX
		xp := idx % nX
		ys := yp*spY + ysp
		xs := xp*spX + xsp
		for y := 0; y < lnY; y++ {
			for x := 0; x < lnX; x++ {
				img.SetRGBA(xs+x, ys+y, c)
			}
		}
	}

	chroma := float32(90)
	tone := float32(65)
	yp := nY
	for hue := 0; hue < 360; hue++ {
		c := hct.New(float32(hue), chroma, tone).AsRGBA()

		ys := yp*spY + ysp
		xs := hue + xsp
		for y := 0; y < spY; y++ {
			img.SetRGBA(xs, ys+y, c)
		}
		for _, h := range hues {
			if int(h) == hue {
				img.SetRGBA(xs, ys-1, White)
			}
		}
	}

	images.Assert(t, img, "spaceddark")
}
