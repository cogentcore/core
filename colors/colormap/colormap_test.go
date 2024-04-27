// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colormap

import (
	"image"
	"image/color"
	"image/draw"
	"slices"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

func TestColorMap(t *testing.T) {
	cm := AvailableMaps["ColdHot"]
	assert.Equal(t, cm.Name, cm.String())

	assert.Equal(t, color.RGBA{54, 64, 212, 255}, cm.Map(0.3))
	assert.Equal(t, color.RGBA{200, 200, 200, 255}, cm.Map(math32.NaN()))
	assert.Equal(t, color.RGBA{0, 255, 255, 255}, cm.Map(-0.5))
	assert.Equal(t, color.RGBA{255, 255, 0, 255}, cm.Map(1.1))

	cm = &Map{}
	assert.Equal(t, color.RGBA{}, cm.Map(0.6))
	cm.Colors = append(cm.Colors, color.RGBA{50, 150, 65, 245})
	assert.Equal(t, color.RGBA{50, 150, 65, 245}, cm.Map(0.45))
}

func TestColorMaps(t *testing.T) {
	nmaps := len(StandardMaps)
	nblend := int(colors.BlendTypesN)
	// y axis is maps x blend mode
	nY := nmaps * (nblend + 1)
	nX := 512

	sqSz := 16
	sqX := 2
	sz := image.Point{sqX * nX, sqSz * nY}
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{127, 127, 127, 255}}, image.Point{}, draw.Src)

	yp := 0
	idx := 0
	keys := make([]string, nmaps)
	for k := range StandardMaps {
		keys[idx] = k
		idx++
	}
	slices.Sort(keys)
	for idx, k := range keys {
		cm := StandardMaps[k]
		for bi, bm := range colors.BlendTypesValues() {
			yp = idx*(nblend+1) + bi
			cm.Blend = bm
			for x := 0; x < nX; x++ {
				xv := float32(x) / float32(nX)
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

	imagex.Assert(t, img, "colormaps")
}

func TestColorMapIndexed(t *testing.T) {
	cm := AvailableMaps["ColdHot"]
	cm.Indexed = true
	n := len(cm.Colors)
	for i := 0; i < n+1; i++ {
		c := cm.MapIndex(i)
		if i == n {
			assert.Equal(t, cm.NoColor, c)
		} else {
			assert.Equal(t, cm.Colors[i], c)
		}
	}
}
