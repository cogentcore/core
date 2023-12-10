// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"goki.dev/grows/images"
)

func TestColorSpace(t *testing.T) {
	tones := []float32{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 98, 100}
	nTones := len(tones)

	hueInc := 15
	hueMax := 345
	nHue := (hueMax / hueInc) + 1
	huePages := 4
	huePerPage := nHue / huePages
	chromaInc := 10
	chromaMax := 150
	nChroma := (chromaMax / chromaInc) + 1

	// y axis is hue then chroma within that
	nY := huePages * nChroma
	nX := huePerPage * nTones

	sqSz := 16
	sz := image.Point{sqSz * nX, sqSz * nY}
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 255, 255}}, image.Point{}, draw.Src)

	xp := 0
	yp := 0
	for hue := 0; hue <= hueMax; hue += hueInc {
		hi := hue / hueInc
		if hi%huePerPage == 0 {
			xp = 0
			yp = (hi / huePerPage) * nChroma
		}
		for chroma := 0; chroma <= chromaMax; chroma += chromaInc {
			ci := chroma / chromaInc
			for ti, tone := range tones {
				h := New(float32(hue), float32(chroma), tone)
				c := h.AsRGBA()
				ys := (yp + ci) * sqSz
				xs := (xp + ti) * sqSz
				for y := 0; y < sqSz; y++ {
					for x := 0; x < sqSz; x++ {
						img.SetRGBA(xs+x, ys+y, c)
					}
				}
			}
		}
		xp += nTones
	}

	images.Assert(t, img, "hctspace")
}
