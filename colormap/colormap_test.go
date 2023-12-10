// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colormap

import (
	"image"
	"image/color"
	"image/draw"
	"slices"
	"testing"

	"goki.dev/grows/images"
)

func TestColorMaps(t *testing.T) {
	nmaps := len(StdMaps)
	nblend := 2
	// y axis is maps x blend mode
	nY := nmaps * (nblend + 1)
	nX := 256

	sqSz := 16
	sqX := 2
	sz := image.Point{sqX * nX, sqSz * nY}
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{127, 127, 127, 255}}, image.Point{}, draw.Src)

	yp := 0
	idx := 0
	keys := make([]string, nmaps)
	for k := range StdMaps {
		keys[idx] = k
		idx++
	}
	slices.Sort(keys)
	for idx, k := range keys {
		cm := StdMaps[k]
		for bm := 0; bm < nblend; bm++ {
			yp = idx*3 + bm
			cm.RGBBlend = bm == 1
			for x := 0; x < nX; x++ {
				xv := float64(x) / float64(nX)
				c := cm.Map(xv)
				ys := yp * sqSz
				xs := x * sqX
				for y := 0; y < sqSz; y++ {
					for x := 0; x < sqX; x++ {
						img.SetRGBA(xs+x, ys+y, c)
					}
				}
			}
		}
		idx++
	}

	images.Assert(t, img, "colormaps")
}
