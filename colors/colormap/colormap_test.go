// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colormap

import (
	"image"
	"image/color"
	"image/draw"
	"maps"
	"slices"
	"sync"
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

func TestMapTable(t *testing.T) {
	n := 20000
	if testing.Short() {
		n = 2000
	}
	keys := slices.Sorted(maps.Keys(StandardMaps))
	for _, bt := range []colors.BlendTypes{colors.HCT, colors.CAM16} {
		for _, k := range keys {
			cm := &Map{Name: k, Blend: bt, NoColor: StandardMaps[k].NoColor, Colors: StandardMaps[k].Colors}
			cm.exactCalls.Store(lutMinCalls) // build the table on first use

			// exact at the map colors, NaN, and out of range values
			nc := len(cm.Colors)
			for i, c := range cm.Colors {
				assert.Equal(t, c, cm.Map(float32(i)/float32(nc-1)))
			}
			assert.Equal(t, cm.NoColor, cm.Map(math32.NaN()))
			assert.Equal(t, cm.Colors[0], cm.Map(-0.1))
			assert.Equal(t, cm.Colors[nc-1], cm.Map(1.1))

			// within one level of the exact blend, except where the exact
			// blend itself jumps by more than that over a tiny distance
			const near = 1e-5
			for i := range n {
				v := (float32(i) + 0.5) / float32(n)
				c := cm.Map(v)
				ex := cm.MapExact(v)
				if maxChannelDiff(c, ex) <= 1 {
					continue
				}
				if maxChannelDiff(c, cm.MapExact(v-near)) <= 1 || maxChannelDiff(c, cm.MapExact(v+near)) <= 1 {
					continue
				}
				t.Errorf("%s %s: Map(%g) = %v, exact %v", k, bt, v, c, ex)
			}
		}
	}
}

func TestMapTableChanges(t *testing.T) {
	cm := &Map{Colors: []color.RGBA{{0, 0, 255, 255}, {255, 0, 0, 255}}}
	v := float32(0.3)
	for range lutMinCalls {
		assert.Equal(t, cm.MapExact(v), cm.Map(v))
	}
	assert.Nil(t, cm.table.Load())
	assert.Equal(t, cm.MapExact(v), cm.Map(v))
	assert.NotNil(t, cm.table.Load())

	// changes to Blend and Colors, including in place, rebuild the table
	cm.Blend = colors.CAM16
	assert.Equal(t, cm.MapExact(v), cm.Map(v))
	cm.Colors[1] = color.RGBA{0, 255, 0, 255}
	assert.Equal(t, cm.MapExact(v), cm.Map(v))
	cm.Colors = append(cm.Colors, color.RGBA{255, 255, 0, 255})
	assert.Equal(t, cm.MapExact(v), cm.Map(v))
}

func TestMapTableConcurrent(t *testing.T) {
	cm := &Map{Colors: StandardMaps["Viridis"].Colors}
	cm.exactCalls.Store(lutMinCalls)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for i := range 100 {
				v := float32(i) / 100
				assert.Equal(t, cm.MapExact(v), cm.Map(v))
			}
		})
	}
	wg.Wait()
}

func BenchmarkMap(b *testing.B) {
	for _, bt := range []colors.BlendTypes{colors.HCT, colors.CAM16, colors.RGB} {
		cm := &Map{Blend: bt, Colors: StandardMaps["ColdHot"].Colors}
		cm.exactCalls.Store(lutMinCalls)
		b.Run(bt.String(), func(b *testing.B) {
			i := 0
			for b.Loop() {
				cm.Map(float32(i%1000) / 1000)
				i++
			}
		})
		b.Run(bt.String()+"Exact", func(b *testing.B) {
			i := 0
			for b.Loop() {
				cm.MapExact(float32(i%1000) / 1000)
				i++
			}
		})
	}
}

func BenchmarkMapTableBuild(b *testing.B) {
	for _, k := range []string{"Viridis", "ColdHot", "Jet"} {
		b.Run(k, func(b *testing.B) {
			for b.Loop() {
				newLUT(&Map{Colors: StandardMaps[k].Colors})
			}
		})
	}
}
