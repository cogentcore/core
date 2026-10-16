// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"reflect"
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
)

// testTexts returns texts with one or more spans, spaces at the ends,
// other fonts and line breaks.
func testTexts() []rich.Text {
	plain := rich.NewStyle()
	bold := *plain
	bold.Weight = rich.Bold
	mono := *plain
	mono.Family = rich.Monospace
	var txs []rich.Text
	for _, s := range []string{"UpArrow", "Control+P, Meta+UpArrow ", "The quick brown fox jumps over the lazy dog. ", "a b", "line one\nline two", " ", "ends in a newline\n", "ends in\r\n", "cr\ronly", "two\n\n", "\n"} {
		txs = append(txs, rich.NewText(plain, []rune(s)))
		tx := rich.NewText(plain, []rune("See "))
		tx.AddSpan(&mono, []rune(s))
		tx.AddSpan(&bold, []rune(" and "+s))
		txs = append(txs, tx)
	}
	return txs
}

func testStyle() *text.Style {
	uc := units.Context{}
	uc.Defaults()
	tsty := text.NewStyle()
	tsty.ToDots(&uc)
	return tsty
}

// TestSingleLine tests that texts are wrapped as the line wrapper wraps
// them, also when they fit on one line.
func TestSingleLine(t *testing.T) {
	sh := NewShaper().(*Shaper)
	tsty := testStyle()
	defSty := rich.NewStyle()
	fast := 0
	for ti, tx := range testTexts() {
		txt := tx.Join()
		for w := 10; w < 700; w += 7 {
			size := math32.Vec2(float32(w), 100)
			cfg, maxSize := wrapConfig(defSty, tsty, size)
			outs := copyOutputs(sh.ShapeTextOutput(tx, tsty, txt))
			want, wantTrunc := sh.wrapper.WrapParagraph(cfg, maxSize, txt, shaping.NewSliceIterator(copyOutputs(outs)))
			want = copyLines(want)
			lines, ok := singleLine(copyOutputs(outs), txt, cfg, maxSize, true)
			if ok {
				fast++
				if !reflect.DeepEqual(lines, want) {
					t.Errorf("text %d width %d: single line differs from the wrapper:\n%v\n%v", ti, w, lines, want)
				}
			}
			lines, trunc := sh.WrapLinesOutput(copyOutputs(outs), txt, tx, defSty, tsty, size)
			if !reflect.DeepEqual(lines, want) || trunc != wantTrunc {
				t.Errorf("text %d width %d: wrapped lines differ from the wrapper:\n%v\n%v", ti, w, lines, want)
			}
		}
	}
	if fast == 0 {
		t.Error("no text fit on one line")
	}
}

// TestShapeCache tests that wrapping texts with the shape cache, at more
// sizes than it keeps wrappings for, gives the same lines as without it.
func TestShapeCache(t *testing.T) {
	sh := NewShaper().(*Shaper)
	tsty := testStyle()
	defSty := rich.NewStyle()
	widths := []int{20, 90, 250, 800, 33, 90, 20}
	for range 2 {
		for ti, tx := range testTexts() {
			for _, w := range widths {
				size := math32.Vec2(float32(w), 200)
				txt := tx.Join()
				outs := copyOutputs(sh.ShapeTextOutput(tx, tsty, txt))
				lines, truncated := sh.WrapLinesOutput(outs, txt, tx, defSty, tsty, size)
				want := sh.LinesBounds(lines, truncated, tx, defSty, tsty, size)
				got := sh.WrapLines(tx, defSty, tsty, size)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("text %d width %d: cached lines differ:\n%v\n%v", ti, w, got, want)
				}
			}
		}
	}
	if sh.shapes.n != len(testTexts()) {
		t.Errorf("cached %d texts, want %d", sh.shapes.n, len(testTexts()))
	}
}

// TestFontMapFaces tests that the faces of runes that the font map keeps
// are those that its FontMap resolves, as queries and scripts alternate.
func TestFontMapFaces(t *testing.T) {
	sh := NewShaper().(*Shaper)
	tsty := testStyle()
	plain := rich.NewStyle()
	mono := *plain
	mono.Family = rich.Monospace
	mono.Weight = rich.Bold
	fm := &sh.fontMap
	for range 3 {
		for _, sty := range []*rich.Style{plain, &mono} {
			for _, script := range []language.Script{language.Latin, language.Han} {
				fm.SetQuery(StyleToQuery(sty, tsty))
				fm.SetScript(script)
				for _, r := range "Aa1 ,.日本😀→" {
					got := fm.ResolveFace(r)
					if want := fm.FontMap.ResolveFace(r); got != want {
						t.Errorf("rune %c script %v: face %p, want %p", r, script, got, want)
					}
				}
			}
		}
	}
	if len(fm.faceSets) != 4 {
		t.Errorf("%d face sets, want 4", len(fm.faceSets))
	}
}
