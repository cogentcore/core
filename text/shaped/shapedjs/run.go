// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/textpos"
	"golang.org/x/image/math/fixed"
)

// Run is a span of text with the same font properties, with full rendering information.
type Run struct {
	shaped.RunBase
	Output
}

func (run *Run) AsBase() *shaped.RunBase {
	return &run.RunBase
}

func (run *Run) Advance() float32 {
	return run.Output.Advance
}

// Runes returns our rune range using textpos.Range
func (run *Run) Runes() textpos.Range {
	return run.Output.Runes
}

// GlyphBoundsBox returns the math32.Box2 version of [Run.GlyphBounds],
// providing a tight bounding box for given glyph within this run.
func (run *Run) GlyphBoundsBox(g *Glyph) math32.Box2 {
	// if run.Direction.IsVertical() {
	// 	if run.Direction.IsSideways() {
	// 		fmt.Println("sideways")
	// 		return fixed.Rectangle26_6{Min: fixed.Point26_6{X: g.XBearing, Y: -g.YBearing}, Max: fixed.Point26_6{X: g.XBearing + g.Width, Y: -g.YBearing - g.Height}}
	// 	}
	// 	return fixed.Rectangle26_6{Min: fixed.Point26_6{X: -g.XBearing - g.Width/2, Y: g.Height - g.YOffset}, Max: fixed.Point26_6{X: g.XBearing + g.Width/2, Y: -(g.YBearing + g.Height) - g.YOffset}}
	// }
	return math32.B2(g.XBearing, -g.YBearing, g.XBearing+g.Width, -g.YBearing-g.Height)
}

// LineBounds returns the LineBounds for given Run as a math32.Box2
// bounding box
func (run *Run) LineBounds() math32.Box2 {
	lb := &run.Output.LineBounds
	gapdec := lb.Descent
	if gapdec < 0 && lb.Gap < 0 || gapdec > 0 && lb.Gap > 0 {
		gapdec += lb.Gap
	} else {
		gapdec -= lb.Gap
	}
	// if run.Direction.IsVertical() {
	// 	// ascent, descent describe horizontal, advance is vertical
	// 	return fixed.Rectangle26_6{Min: fixed.Point26_6{X: -lb.Ascent, Y: 0},
	// 		Max: fixed.Point26_6{X: -gapdec, Y: -run.Output.Advance}}
	// }
	return math32.B2(0, -lb.Ascent, run.Output.Advance, -gapdec)
}

// GlyphsAt returns the indexs of the glyph(s) at given original source rune index.
// Empty if none found.
func (run *Run) GlyphsAt(i int) []int {
	var gis []int
	for gi := range run.Glyphs {
		g := &run.Glyphs[gi]
		if g.ClusterIndex > i {
			break
		}
		if g.ClusterIndex == i {
			gis = append(gis, gi)
		}
	}
	return gis
}

// FirstGlyphAt returns the index of the first glyph at or above given original
// source rune index, returns -1 if none found.
func (run *Run) FirstGlyphAt(i int) int {
	for gi := range run.Glyphs {
		g := &run.Glyphs[gi]
		if g.ClusterIndex >= i {
			return gi
		}
	}
	return -1
}

// LastGlyphAt returns the index of the last glyph at given original source rune index,
// returns -1 if none found.
func (run *Run) LastGlyphAt(i int) int {
	ng := len(run.Glyphs)
	for gi := ng - 1; gi >= 0; gi-- {
		g := &run.Glyphs[gi]
		if g.ClusterIndex <= i {
			return gi
		}
	}
	return -1
}

// SetGlyphXAdvance sets the x advance on all glyphs to given value:
// for monospaced case.
func (run *Run) SetGlyphXAdvance(adv fixed.Int26_6) {
	fadv := math32.FromFixed(adv)
	for gi := range run.Glyphs {
		g := &run.Glyphs[gi]
		g.XAdvance = fadv
	}
	run.Output.Advance = fadv * float32(len(run.Glyphs))
}

// RuneAtPoint returns the index of the rune in the source, which contains given point,
// using the maximal glyph bounding box. Off is the offset for this run within overall
// image rendering context of point coordinates. Assumes point is already identified
// as being within the [Run.MaxBounds].
func (run *Run) RuneAtPoint(src rich.Text, pt, off math32.Vector2) int {
	// todo: vertical case!
	adv := off.X
	rr := run.Runes()
	for gi := range run.Glyphs {
		g := &run.Glyphs[gi]
		cri := g.ClusterIndex
		gadv := g.XAdvance
		mx := adv + gadv
		// fmt.Println(gi, cri, adv, mx, pt.X)
		if pt.X >= adv && pt.X < mx {
			// fmt.Println("fits!")
			return cri
		}
		adv += gadv
	}
	return rr.End
}

// RuneBounds returns the maximal line-bounds level bounding box for given rune index.
func (run *Run) RuneBounds(ri int) math32.Box2 {
	gis := run.GlyphsAt(ri)
	if len(gis) == 0 {
		// fmt.Println("no glyphs")
		return (math32.Box2{})
	}
	return run.GlyphRegionBounds(gis[0], gis[len(gis)-1])
}

// GlyphRegionBounds returns the maximal line-bounds level bounding box
// between two glyphs in this run, where the st comes before the ed.
func (run *Run) GlyphRegionBounds(st, ed int) math32.Box2 {
	if run.Direction.IsVertical() {
		// todo: write me!
		return math32.Box2{}
	}
	sg := &run.Glyphs[st]
	stb := run.GlyphBoundsBox(sg)
	mb := run.MaxBounds
	off := float32(0)
	for gi := 0; gi < st; gi++ {
		g := &run.Glyphs[gi]
		off += g.XAdvance
	}
	mb.Min.X = off + stb.Min.X - 2
	for gi := st; gi <= ed; gi++ {
		g := &run.Glyphs[gi]
		gb := run.GlyphBoundsBox(g)
		mb.Max.X = off + gb.Max.X + 2
		off += g.XAdvance
	}
	return mb
}
