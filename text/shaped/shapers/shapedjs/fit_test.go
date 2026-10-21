// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedjs

import (
	"testing"

	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

func TestFitAdvances(t *testing.T) {
	gs := []shaping.Glyph{{Advance: fixed.I(10)}, {Advance: fixed.I(10)}, {Advance: fixed.I(10)}}
	fitAdvances(gs, fixed.I(27))
	var sum fixed.Int26_6
	for _, g := range gs {
		sum += g.Advance
		if g.Advance < fixed.I(8) || g.Advance > fixed.I(10) {
			t.Errorf("advance %v", g.Advance)
		}
	}
	if sum != fixed.I(27) {
		t.Errorf("sum %v, want 27", sum)
	}

	// Advances far from the given one are kept.
	gs = []shaping.Glyph{{Advance: fixed.I(20)}}
	fitAdvances(gs, fixed.I(10))
	if gs[0].Advance != fixed.I(20) {
		t.Errorf("advance %v, want 20", gs[0].Advance)
	}
}

// TestFitWords tests that the words of an output are fit to their widths,
// starting at a rune index other than 0, and that spaces are kept.
func TestFitWords(t *testing.T) {
	rs := []rune("xx ab  cde f")
	start := 3 // the output is "ab  cde f"
	out := shaping.Output{}
	for i := start; i < len(rs); i++ {
		out.Glyphs = append(out.Glyphs, shaping.Glyph{ClusterIndex: i, RuneCount: 1, GlyphCount: 1, Advance: fixed.I(10)})
	}
	widths := map[string]fixed.Int26_6{}
	fitWords(&out, rs[start:], start, func(word []rune) fixed.Int26_6 {
		w := fixed.I(9 * len(word))
		widths[string(word)] = w
		return w
	})
	if len(widths) != 2 || widths["ab"] == 0 || widths["cde"] == 0 {
		t.Errorf("measured words %v, want ab and cde", widths)
	}
	want := []int{9, 9, 10, 10, 9, 9, 9, 10, 10}
	for i, g := range out.Glyphs {
		if g.Advance != fixed.I(want[i]) {
			t.Errorf("glyph %d (%q): advance %v, want %d", i, rs[g.ClusterIndex], g.Advance, want[i])
		}
	}
}
