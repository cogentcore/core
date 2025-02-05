// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
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

// Runes returns our rune range using textpos.Range
func (rn *Run) Runes() textpos.Range {
	return textpos.Range{rn.Output.Runes.Offset, rn.Output.Runes.Offset + rn.Output.Runes.Count}
}

// todo: spaces don't count here!

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
