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

// RuneToLinePos returns the [textpos.Pos] line and character position for given rune
// index in Lines source. If ti >= source Len(), returns a position just after
// the last actual rune.
func (ls *Lines) RuneToLinePos(ti int) textpos.Pos {
	if len(ls.Lines) == 0 {
		return textpos.Pos{}
	}
	n := ls.Source.Len()
	el := len(ls.Lines) - 1
	ep := textpos.Pos{el, ls.Lines[el].SourceRange.End}
	if ti >= n {
		return ep
	}
	for li := range ls.Lines {
		ln := &ls.Lines[li]
		if !ln.SourceRange.Contains(ti) {
			continue
		}
		return textpos.Pos{li, ti - ln.SourceRange.Start}
	}
	return ep // shouldn't happen
}

// RuneFromLinePos returns the rune index in Lines source for given
// [textpos.Pos] line and character position. Returns Len() of source
// if it goes past that.
func (ls *Lines) RuneFromLinePos(tp textpos.Pos) int {
	if len(ls.Lines) == 0 {
		return 0
	}
	n := ls.Source.Len()
	nl := len(ls.Lines)
	if tp.Line >= nl {
		return n
	}
	ln := &ls.Lines[tp.Line]
	return ln.SourceRange.Start + tp.Char
}

// RuneAtLineDelta returns the rune index in Lines source at given
// relative vertical offset in lines from the current line for given rune.
// It uses pixel locations of glyphs and the LineHeight to find the
// rune at given vertical offset with the same horizontal position.
// If the delta goes out of range, it will return the appropriate in-range
// rune index at the closest horizontal position.
func (ls *Lines) RuneAtLineDelta(ti, lineDelta int) int {
	rp := ls.RuneBounds(ti).Center()
	tp := rp
	ld := float32(lineDelta) * ls.LineHeight // todo: should iterate over lines for different sizes..
	tp.Y = math32.Clamp(tp.Y+ld, ls.Bounds.Min.Y+2, ls.Bounds.Max.Y-2)
	return ls.RuneAtPoint(tp, math32.Vector2{})
}

// RuneBounds returns the glyph bounds for given rune index in Lines source,
// relative to the upper-left corner of the lines bounding box.
// If the index is >= the source length, it returns a box at the end of the
// rendered text (i.e., where a cursor should be to add more text).
func (ls *Lines) RuneBounds(ti int) math32.Box2 {
	n := ls.Source.Len()
	zb := math32.Box2{}
	if len(ls.Lines) == 0 {
		return zb
	}
	start := ls.Offset
	if ti >= n { // goto end
		ln := ls.Lines[len(ls.Lines)-1]
		off := start.Add(ln.Offset)
		run := &ln.Runs[len(ln.Runs)-1]
		ep := run.MaxBounds.Max.Add(off)
		ep.Y = run.MaxBounds.Min.Y + off.Y
		return math32.Box2{ep, ep}
	}
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
// based on given starting location for rendering. If the point is out of the
// line bounds, the nearest point is returned (e.g., start of line based on Y coordinate).
func (ls *Lines) RuneAtPoint(pt math32.Vector2, start math32.Vector2) int {
	start.SetAdd(ls.Offset)
	lbb := ls.Bounds.Translate(start)
	if !lbb.ContainsPoint(pt) {
		// smaller bb so point will be inside stuff
		sbb := math32.Box2{lbb.Min.Add(math32.Vec2(0, 2)), lbb.Max.Sub(math32.Vec2(0, 2))}
		pt = sbb.ClampPoint(pt)
	}
	nl := len(ls.Lines)
	for li := range ls.Lines {
		ln := &ls.Lines[li]
		off := start.Add(ln.Offset)
		lbb := ln.Bounds.Translate(off)
		if !lbb.ContainsPoint(pt) {
			if pt.Y >= lbb.Min.Y && pt.Y < lbb.Max.Y { // this is our line
				if pt.X <= lbb.Min.X+2 {
					return ln.SourceRange.Start
				}
				return ln.SourceRange.End
			}
			continue
		}
		for ri := range ln.Runs {
			run := &ln.Runs[ri]
			rbb := run.MaxBounds.Translate(off)
			if !rbb.ContainsPoint(pt) {
				off.X += math32.FromFixed(run.Advance)
				continue
			}
			rp := run.RuneAtPoint(ls.Source, pt, off)
			if rp == run.Runes().End && li < nl-1 { // if not at full end, don't go past
				rp--
			}
			return rp
		}
		return ln.SourceRange.End
	}
	return 0
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
	rr := rn.Runes()
	pri := rr.Start
	for gi := range rn.Glyphs {
		g := &rn.Glyphs[gi]
		cri := g.ClusterIndex
		gb := rn.GlyphBoundsBox(g)
		gadv := math32.FromFixed(g.XAdvance)
		cx := adv + gb.Min.X
		dx := pt.X - cx
		if dx >= -2 && pt.X < adv+gb.Max.X+2 {
			return cri
		}
		if pt.X < cx { // it is before us, in space
			nri := cri - pri
			ri := pri + int(math32.Round(float32(nri)*((adv-pt.X)/(cx-adv)))) // linear interpolation
			// fmt.Println("before:", gi, ri, pri, cri, adv, cx, adv-pt.X, cx-adv)
			return ri
		}
		pri = cri
		adv += gadv
	}
	return rr.End
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
