// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"fmt"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
)

// SelectRegion adds the selection to given region of runes from
// the original source runes. Use SelectReset to clear first if desired.
func (ls *Lines) SelectRegion(r textpos.Range) {
	nr := ls.Source.Len()
	r = r.Intersect(textpos.Range{0, nr})
	for li := range ls.Lines {
		ln := &ls.Lines[li]
		lr := r.Intersect(ln.SourceRange)
		if lr.Len() > 0 {
			ln.Selections = append(ln.Selections, lr)
		}
	}
}

// SelectReset removes all existing selected regions.
func (ls *Lines) SelectReset() {
	for li := range ls.Lines {
		ln := &ls.Lines[li]
		ln.Selections = nil
	}
}

// RuneBounds returns the glyph bounds for given rune index in Lines source,
// relative to the upper-left corner of the lines bounding box.
func (ls *Lines) RuneBounds(ti int) math32.Box2 {
	n := ls.Source.Len()
	zb := math32.Box2{}
	if ti >= n {
		return zb
	}
	start := ls.Offset
	for li := range ls.Lines {
		ln := &ls.Lines[li]
		if !ln.SourceRange.Contains(ti) {
			continue
		}
		off := start.Add(ln.Offset)
		for ri := range ln.Runs {
			run := &ln.Runs[ri]
			rr := run.Runes()
			if ti < rr.Start { // space?
				fmt.Println("early:", ti, rr.Start)
				off.X += math32.FromFixed(run.Advance)
				continue
			}
			if ti >= rr.End {
				off.X += math32.FromFixed(run.Advance)
				continue
			}
			gis := run.GlyphsAt(ti)
			if len(gis) == 0 {
				fmt.Println("no glyphs")
				return zb // nope
			}
			bb := run.GlyphRegionBounds(gis[0], gis[len(gis)-1])
			return bb.Translate(off)
		}
	}
	return zb
}

// RuneAtPoint returns the rune index in Lines source, at given rendered location,
// based on given starting location for rendering. Returns -1 if not within lines.
func (ls *Lines) RuneAtPoint(pt math32.Vector2, start math32.Vector2) int {
	start.SetAdd(ls.Offset)
	lbb := ls.Bounds.Translate(start)
	if !lbb.ContainsPoint(pt) {
		return -1
	}
	for li := range ls.Lines {
		ln := &ls.Lines[li]
		off := start.Add(ln.Offset)
		lbb := ln.Bounds.Translate(off)
		if !lbb.ContainsPoint(pt) {
			continue
		}
		for ri := range ln.Runs {
			run := &ln.Runs[ri]
			rbb := run.MaxBounds.Translate(off)
			if !rbb.ContainsPoint(pt) {
				off.X += math32.FromFixed(run.Advance)
				continue
			}
			return run.RuneAtPoint(ls.Source, pt, off)
		}
	}
	return -1
}

// Runes returns our rune range using textpos.Range
func (rn *Run) Runes() textpos.Range {
	return textpos.Range{rn.Output.Runes.Offset, rn.Output.Runes.Offset + rn.Output.Runes.Count}
}

// GlyphsAt returns the indexs of the glyph(s) at given original source rune index.
// Only works for non-space rendering runes. Empty if none found.
func (rn *Run) GlyphsAt(i int) []int {
	var gis []int
	for gi := range rn.Glyphs {
		g := &rn.Glyphs[gi]
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
func (rn *Run) FirstGlyphAt(i int) int {
	for gi := range rn.Glyphs {
		g := &rn.Glyphs[gi]
		if g.ClusterIndex >= i {
			return gi
		}
	}
	return -1
}

// LastGlyphAt returns the index of the last glyph at given original source rune index,
// returns -1 if none found.
func (rn *Run) LastGlyphAt(i int) int {
	ng := len(rn.Glyphs)
	for gi := ng - 1; gi >= 0; gi-- {
		g := &rn.Glyphs[gi]
		if g.ClusterIndex <= i {
			return gi
		}
	}
	return -1
}

// RuneAtPoint returns the index of the rune in the source, which contains given point,
// using the maximal glyph bounding box. Off is the offset for this run within overall
// image rendering context of point coordinates. Assumes point is already identified
// as being within the [Run.MaxBounds].
func (rn *Run) RuneAtPoint(src rich.Text, pt, off math32.Vector2) int {
	// todo: vertical case!
	adv := off.X
	pri := 0 // todo: need starting rune of run
	for gi := range rn.Glyphs {
		g := &rn.Glyphs[gi]
		cri := g.ClusterIndex
		gb := rn.GlyphBoundsBox(g)
		gadv := math32.FromFixed(g.XAdvance)
		cx := adv + gb.Min.X
		if pt.X < cx { // it is before us, in space
			nri := cri - pri
			ri := pri + int(math32.Round(float32(nri)*((pt.X-adv)/(cx-adv)))) // linear interpolation
			return ri
		}
		if pt.X >= adv+gb.Min.X && pt.X < adv+gb.Max.X {
			return cri
		}
		pri = cri
		adv += gadv
	}
	return -1
}

// GlyphRegionBounds returns the maximal line-bounds level bounding box
// between two glyphs in this run, where the st comes before the ed.
func (rn *Run) GlyphRegionBounds(st, ed int) math32.Box2 {
	if rn.Direction.IsVertical() {
		// todo: write me!
		return math32.Box2{}
	}
	sg := &rn.Glyphs[st]
	stb := rn.GlyphBoundsBox(sg)
	mb := rn.MaxBounds
	off := float32(0)
	for gi := 0; gi < st; gi++ {
		g := &rn.Glyphs[gi]
		off += math32.FromFixed(g.XAdvance)
	}
	mb.Min.X = off + stb.Min.X - 2
	for gi := st; gi <= ed; gi++ {
		g := &rn.Glyphs[gi]
		gb := rn.GlyphBoundsBox(g)
		mb.Max.X = off + gb.Max.X + 2
		off += math32.FromFixed(g.XAdvance)
	}
	return mb
}
