// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedjs

import (
	"math"
	"unicode"

	"github.com/cogentcore/typesetting/di"
	"github.com/cogentcore/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

// fitWords scales the advances of the glyphs of each word of the output,
// whose text is the given runes starting at rune index start, to add up to
// the width of the word, which width returns. The glyphs are measured by
// themselves, without the kerning (and ligatures) that the browser applies
// to the text it draws, so they add up to more than the words: the line
// wrapper, which wraps runs by the advances of their glyphs, would wrap
// lines that fit, and the positions of the runes would differ from those
// that the browser draws. Only left to right outputs are fit.
func fitWords(out *shaping.Output, rs []rune, start int, width func(word []rune) fixed.Int26_6) {
	if out.Direction != di.DirectionLTR {
		return
	}
	gs := out.Glyphs
	space := func(g *shaping.Glyph) bool {
		i := g.TextIndex() - start
		return i < 0 || i >= len(rs) || unicode.IsSpace(rs[i])
	}
	for i := 0; i < len(gs); {
		// A word is the glyphs up to the next space.
		if space(&gs[i]) {
			i++
			continue
		}
		j := i + 1
		for j < len(gs) && !space(&gs[j]) {
			j++
		}

		// A word of more than one glyph is fit to its width.
		if j-i > 1 {
			first, last := &gs[i], &gs[j-1]
			end := min(last.TextIndex()-start+last.RunesCount(), len(rs))
			fitAdvances(gs[i:j], width(rs[first.TextIndex()-start:end]))
		}
		i = j
	}
}

// fitAdvances scales the advances of the glyphs to add up to the given
// advance, unless they differ from it by more than kerning does.
func fitAdvances(gs []shaping.Glyph, adv fixed.Int26_6) {
	var sum fixed.Int26_6
	for i := range gs {
		sum += gs[i].Advance
	}
	if sum <= 0 || sum == adv {
		return
	}
	f := float64(adv) / float64(sum)
	if f < 0.8 || f > 1.2 {
		return
	}

	// The last glyph has the rest of the advance, so that they add up exactly.
	var total fixed.Int26_6
	last := len(gs) - 1
	for i := range last {
		g := &gs[i]
		g.Advance = fixed.Int26_6(math.Round(float64(g.Advance) * f))
		total += g.Advance
	}
	gs[last].Advance = adv - total
}
