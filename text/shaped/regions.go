// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/textpos"
	"github.com/go-text/typesetting/shaping"
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

// GlyphAtPoint returns the glyph at given rendered location, based
// on given starting location for rendering. The Glyph.ClusterIndex is the
// index of the rune in the original source that it corresponds to.
// Can return nil if not within lines.
func (ls *Lines) GlyphAtPoint(pt math32.Vector2, start math32.Vector2) *shaping.Glyph {
	start.SetAdd(ls.Offset)
	lbb := ls.Bounds.Translate(start)
	if !lbb.ContainsPoint(pt) {
		return nil
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
				continue
			}
			// in this run:
			gi := run.FirstGlyphContainsPoint(pt, off)
			if gi >= 0 { // someone should, given the run does
				return &run.Glyphs[gi]
			}
		}
	}
	return nil
}

// Runes returns our rune range using textpos.Range
func (rn *Run) Runes() textpos.Range {
	return textpos.Range{rn.Output.Runes.Offset, rn.Output.Runes.Offset + rn.Output.Runes.Count}
}

// FirstGlyphAt returns the index of the first glyph at given original source rune index.
// returns -1 if none found.
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

// FirstGlyphContainsPoint returns the index of the first glyph that contains given point,
// using the maximal glyph bounding box. Off is the offset for this run within overall
// image rendering context of point coordinates. Assumes point is already identified
// as being within the [Run.MaxBounds].
func (rn *Run) FirstGlyphContainsPoint(pt, off math32.Vector2) int {
	// todo: vertical case!
	adv := float32(0)
	for gi := range rn.Glyphs {
		g := &rn.Glyphs[gi]
		gb := rn.GlyphBoundsBox(g)
		if pt.X < adv+gb.Min.X { // it is before us, in space
			// todo: fabricate a space??
			return gi
		}
		if pt.X >= adv+gb.Min.X && pt.X < adv+gb.Max.X {
			return gi // for real
		}
		adv += math32.FromFixed(g.XAdvance)
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
