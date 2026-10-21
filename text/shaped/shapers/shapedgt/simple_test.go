// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"reflect"
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
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

// TestMathCache tests that the lines of cached texts with math spans are
// those of the uncached path, with their math.
func TestMathCache(t *testing.T) {
	old := shaped.ShapeMath
	defer func() { shaped.ShapeMath = old }()
	shaped.ShapeMath = func(expr string, fontHeight float32) (ppath.Path, error) {
		p := ppath.Path{}
		w := float32(len(expr)) * fontHeight / 2
		p.MoveTo(0, -fontHeight)
		p.LineTo(w, -fontHeight)
		p.LineTo(w, fontHeight/3)
		p.Close()
		return p, nil
	}
	sh := NewShaper().(*Shaper)
	tsty := testStyle()
	defSty := rich.NewStyle()
	var txs []rich.Text
	for _, m := range []string{"x", `y = f(x^2)`, `\frac{1}{N}`} {
		tx := rich.NewText(defSty, []rune("The value "))
		tx.AddMathInline(defSty, m)
		tx.AddSpan(defSty, []rune(" is what we compute here, and it wraps"))
		txs = append(txs, tx)
	}
	for round := range 2 {
		for ti, tx := range txs {
			for _, w := range []float32{60, 150, 400, 150} {
				size := math32.Vec2(w, 200)
				txt := tx.Join()
				outs := copyOutputs(sh.ShapeTextOutput(tx, tsty, txt))
				lines, truncated := sh.WrapLinesOutput(outs, txt, tx, defSty, tsty, size)
				want := sh.LinesBounds(lines, truncated, tx, defSty, tsty, size)
				got := sh.WrapLines(tx, defSty, tsty, size)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("round %d text %d width %v: cached lines differ:\n%v\n%v", round, ti, w, got, want)
				}
			}
		}
	}
	if sh.shapes.n != len(txs) {
		t.Errorf("cached %d texts, want %d", sh.shapes.n, len(txs))
	}
}
