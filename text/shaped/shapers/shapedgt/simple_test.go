// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"reflect"
	"testing"

	"cogentcore.org/core/text/rich"
	"github.com/go-text/typesetting/shaping"
)

// TestSimpleSpans tests that spans of simple text are shaped with
// [Shaper.SimpleSpans] into the glyphs of the go-text shaper, apart from
// the glyph metrics that the browser measures (with no kerning or ligatures
// in these texts), and that other spans are shaped by it.
func TestSimpleSpans(t *testing.T) {
	plain := rich.NewStyle()
	mono := *plain
	mono.Family = rich.Monospace
	italic := *plain
	italic.Slant = rich.Italic
	simple := []string{"Hello, World! 0123456789", "Control+P, Meta+UpArrow ", " ", "αβγ Ωμέγα кириллица", "Größe Ĳ ŝ"}
	// mixed has runes of other faces, so it is shaped by the go-text shaper.
	mixed := "a — b “c” → d ≤ e"
	other := []string{mixed, "line one\nline two", "tab\there", "שלום world", "e\u0301", "a\u200db", "emoji 😀"}
	sh := NewShaper().(*Shaper)
	tsty := testStyle()
	shape := func(tx rich.Text, simple bool) []shaping.Output {
		sh.SimpleSpans = simple
		return copyOutputs(sh.ShapeTextOutput(tx, tsty, tx.Join()))
	}
	for _, sty := range []*rich.Style{plain, &mono, &italic} {
		for _, s := range simple {
			tx := rich.NewText(sty, []rune(s))
			tx.AddSpan(&mono, []rune(" and "+s))
			want, got := shape(tx, false), shape(tx, true)
			if len(got) != len(tx) {
				t.Errorf("%q: %d outputs for %d spans", s, len(got), len(tx))
				continue
			}

			// The go-text shaper may split a span by script, in more outputs.
			var glyphs []shaping.Glyph
			for i := range want {
				glyphs = append(glyphs, want[i].Glyphs...)
			}
			gi := 0
			for _, out := range got {
				if out.Face != want[0].Face && out.Face != want[len(want)-1].Face {
					t.Errorf("%q: face %v", s, out.Face.Describe().Family)
				}
				for _, g := range out.Glyphs {
					w := glyphs[gi]
					gi++
					if g.ClusterIndex != w.ClusterIndex || g.RuneCount != w.RuneCount || g.GlyphCount != w.GlyphCount || g.GlyphID != w.GlyphID {
						t.Errorf("%q: glyph %d is %+v, want %+v", s, gi, g, w)
					}
				}
			}
			if gi != len(glyphs) {
				t.Errorf("%q: %d glyphs, want %d", s, gi, len(glyphs))
			}
		}
		for _, s := range other {
			tx := rich.NewText(sty, []rune(s))
			if want, got := shape(tx, false), shape(tx, true); !reflect.DeepEqual(got, want) {
				t.Errorf("%q: not shaped by the go-text shaper", s)
			}
		}
	}
}
