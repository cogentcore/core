// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colormap

import (
	"image/color"
	"slices"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/cam/hct"
)

// lut is a lookup table of the blended colors of a [Map], which makes
// [Map.Map] much faster than blending (in HCT space by default) on every
// call. Each segment between two adjacent map colors is divided into
// equal cells, which are bisected where the blend bends more sharply,
// and the exact blended colors at the cell boundaries are stored unrounded.
// A value within a cell is linearly interpolated between the colors at its
// boundaries, which is within one level per channel of the exact blend,
// and usually equal to it (HCT blends also have rare jumps over tiny
// distances, which the table smooths over). The smallest cells where
// interpolation is still not that accurate (e.g., at the sharp bends that
// HCT blending has at the ends of some segments) are marked as exact,
// and their values are blended exactly.
type lut struct {

	// blend is the blend type the table was built with.
	blend colors.BlendTypes

	// colors is a copy of the map colors the table was built with.
	colors []color.RGBA

	// keys are the colors packed by [colorKey], to check that the
	// map colors have not changed.
	keys []uint32

	// entries are the exact blended colors at the cell boundaries,
	// as four unrounded alpha-premultiplied 0-255 channel values each.
	entries []float32

	// The cells of all segments are the first lutCells for each segment,
	// followed by the halves of bisected cells. They are stored as
	// separate slices of their fields, which is faster in JavaScript.

	// half is the index of the first half of a bisected cell,
	// followed by its second half, or 0 if the cell is not bisected.
	half []int32

	// lo and hi are the indexes in entries of the colors at the
	// boundaries of a cell that is not bisected.
	lo, hi []int32

	// exact is whether colors in a cell that is not bisected must be
	// blended exactly, because interpolation is not accurate enough there.
	exact []bool
}

// lutColor is an unrounded alpha-premultiplied color with 0-255 channels.
type lutColor [4]float32

const (
	// lutCells is the number of cells each segment is divided into
	// before bisecting.
	lutCells = 16

	// lutDepth is the maximum number of times a cell is bisected,
	// so the smallest cells are 1/4096 of a segment.
	lutDepth = 8

	// lutTolerance is the largest difference between an interpolated and
	// an exact unrounded channel value, at the middle and quarters of
	// a cell, for the cell to be interpolated. It is above the noise of
	// about half a level in HCT blends, which are not quite smooth.
	lutTolerance = 0.5

	// lutEndCheck is the distance from the ends of a segment, as a
	// proportion of it, at which cells at the ends are also checked:
	// a quarter of the smallest cell.
	lutEndCheck = 1.0 / (4 * lutCells << lutDepth)
)

// matches returns whether the table was built for the current
// blend type and colors of the given map.
func (lt *lut) matches(cm *Map) bool {
	if lt.blend != cm.Blend || len(lt.colors) != len(cm.Colors) {
		return false
	}
	for i, k := range lt.keys {
		if k != colorKey(cm.Colors, i) {
			return false
		}
	}
	return true
}

// colorKey returns color i of the given colors packed into a uint32,
// which is faster to compare than the color in JavaScript.
func colorKey(cs []color.RGBA, i int) uint32 {
	return uint32(cs[i].R)<<24 | uint32(cs[i].G)<<16 | uint32(cs[i].B)<<8 | uint32(cs[i].A)
}

// lookup returns the color at proportion t (0 < t < 1) along
// segment si, interpolated from the table or blended exactly.
func (lt *lut) lookup(si int, t float32) color.RGBA {
	p := t * lutCells
	ci := int(p)
	if ci >= lutCells {
		ci = lutCells - 1
	}
	f := p - float32(ci)
	ci += si * lutCells
	for h := int(lt.half[ci]); h != 0; h = int(lt.half[ci]) {
		f *= 2
		if f < 1 {
			ci = h
		} else {
			f -= 1
			ci = h + 1
		}
	}
	if lt.exact[ci] {
		return colors.Blend(lt.blend, 100*(1-t), lt.colors[si], lt.colors[si+1])
	}
	e := lt.entries
	a, b := int(lt.lo[ci]), int(lt.hi[ci])
	r, g, bl, al := lerp8(e[a], e[b], f), lerp8(e[a+1], e[b+1], f), lerp8(e[a+2], e[b+2], f), lerp8(e[a+3], e[b+3], f)
	return color.RGBA{r, g, bl, al}
}

// lerp8 returns the value at proportion f between a and b,
// rounded to 8 bits.
func lerp8(a, b, f float32) uint8 {
	return uint8(a + (b-a)*f + 0.5)
}

// newLUT builds the lookup table for the given map, which has at least
// two colors.
func newLUT(cm *Map) *lut {
	lt := &lut{blend: cm.Blend, colors: slices.Clone(cm.Colors)}
	ns := len(lt.colors) - 1
	lt.keys = make([]uint32, ns+1)
	for i := range lt.keys {
		lt.keys[i] = colorKey(lt.colors, i)
	}
	for range ns * lutCells {
		lt.addCell()
	}
	for si := range ns {
		lo := lt.addEntry(si, 0)
		for ci := range lutCells {
			hi := lt.addEntry(si, float32(ci+1)/lutCells)
			lt.buildCell(si*lutCells+ci, si, float32(ci)/lutCells, 1.0/lutCells, lo, hi, 0)
			lo = hi
		}
	}
	return lt
}

// addCell adds a cell, returning its index.
func (lt *lut) addCell() int32 {
	lt.half = append(lt.half, 0)
	lt.lo = append(lt.lo, 0)
	lt.hi = append(lt.hi, 0)
	lt.exact = append(lt.exact, false)
	return int32(len(lt.half) - 1)
}

// addEntry adds the exact blend at proportion t along segment si
// to the entries, returning its index.
func (lt *lut) addEntry(si int, t float32) int32 {
	c := lt.blendAt(si, t)
	lt.entries = append(lt.entries, c[:]...)
	return int32(len(lt.entries) - 4)
}

// entry returns the entry at index i.
func (lt *lut) entry(i int32) lutColor {
	e := lt.entries
	return lutColor{e[i], e[i+1], e[i+2], e[i+3]}
}

// buildCell sets cell ci of segment si, starting at proportion t0 along it
// with width w and boundary entries lo and hi, bisecting it until
// interpolating is accurate or it has been bisected depth times.
// The exact color at its middle, used to check it, becomes the
// boundary between its halves.
func (lt *lut) buildCell(ci, si int, t0, w float32, lo, hi int32, depth int) {
	mid := lt.addEntry(si, t0+0.5*w)
	lt.lo[ci], lt.hi[ci] = lo, hi
	if lt.accurate(si, t0, w, lo, hi, mid) {
		lt.entries = lt.entries[:mid] // middle not needed
		return
	}
	if depth == lutDepth {
		lt.exact[ci] = true
		return
	}
	half := lt.addCell()
	lt.addCell()
	lt.half[ci] = half
	lt.buildCell(int(half), si, t0, 0.5*w, lo, mid, depth+1)
	lt.buildCell(int(half)+1, si, t0+0.5*w, 0.5*w, mid, hi, depth+1)
}

// accurate returns whether interpolating between the entries lo and hi
// of the cell starting at proportion t0 along segment si with width w
// is within [lutTolerance] of the exact colors at its middle (entry mid)
// and quarters, in all channels. Cells at the ends of the segment are
// also checked very close to the end, because blending can change sharply
// there (in HCT, the colors just inside a segment can differ by a few
// levels from its end colors).
func (lt *lut) accurate(si int, t0, w float32, lo, hi, mid int32) bool {
	a, b := lt.entry(lo), lt.entry(hi)
	if !near(a, b, 0.5, lt.entry(mid)) {
		return false
	}
	if t0 == 0 && !near(a, b, lutEndCheck/w, lt.blendAt(si, lutEndCheck)) {
		return false
	}
	if t0+w == 1 && !near(a, b, 1-lutEndCheck/w, lt.blendAt(si, 1-lutEndCheck)) {
		return false
	}
	return near(a, b, 0.25, lt.blendAt(si, t0+0.25*w)) && near(a, b, 0.75, lt.blendAt(si, t0+0.75*w))
}

// near returns whether interpolating at proportion f between a and b
// is within [lutTolerance] of c in all channels.
func near(a, b lutColor, f float32, c lutColor) bool {
	for i := range 4 {
		d := a[i] + (b[i]-a[i])*f - c[i]
		if d > lutTolerance || d < -lutTolerance {
			return false
		}
	}
	return true
}

// blendAt returns the unrounded exact blend at proportion t along
// segment si, which rounds to the color that [Map.MapExact] returns.
func (lt *lut) blendAt(si int, t float32) lutColor {
	x, y := lt.colors[si], lt.colors[si+1]
	pct := 100 * (1 - t)
	if lt.blend == colors.HCT {
		h := hct.BlendHCT(pct, x, y)
		return lutColor{h.R * h.A * 255, h.G * h.A * 255, h.B * h.A * 255, h.A * 255}
	}
	c := colors.Blend(lt.blend, pct, x, y)
	return lutColor{float32(c.R), float32(c.G), float32(c.B), float32(c.A)}
}

// maxChannelDiff returns the largest difference between
// the channels of the two colors.
func maxChannelDiff(a, b color.RGBA) int {
	return max(absDiff(a.R, b.R), absDiff(a.G, b.G), absDiff(a.B, b.B), absDiff(a.A, b.A))
}

// absDiff returns the absolute difference between a and b.
func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
