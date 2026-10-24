// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
	"github.com/cogentcore/typesetting/language"
	"github.com/cogentcore/typesetting/shaping"
	"golang.org/x/image/math/fixed"
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

// TestBoundsCache tests that the lines with bounds of cached wrappings are
// those of the uncached path as the values that the bounds depend on vary
// between calls with the same wrapping, and that changing the returned lines
// does not change those of later calls.
func TestBoundsCache(t *testing.T) {
	sh := NewShaper().(*Shaper)
	defSty := rich.NewStyle()
	bigSty := *defSty
	bigSty.Size = 1.5
	// variant is one set of values that the bounds depend on.
	type variant struct {
		width  float32
		height float32 // 0 for no truncation, so that the wrapping does not depend on the line height
		defSty *rich.Style
		style  func(ts *text.Style)
	}
	variants := []variant{
		{90, 200, defSty, func(ts *text.Style) {}},
		{5, 0, defSty, func(ts *text.Style) {}},
		{5.6, 0, defSty, func(ts *text.Style) {}},
		{90, 0, defSty, func(ts *text.Style) {}},
		{90, 0, &bigSty, func(ts *text.Style) {}},
		{90.6, 200, defSty, func(ts *text.Style) {}},
		{90, 200, &bigSty, func(ts *text.Style) {}},
		{90, 200, defSty, func(ts *text.Style) { ts.Align = text.Center }},
		{90, 200, defSty, func(ts *text.Style) { ts.Align = text.End }},
		{90, 0, defSty, func(ts *text.Style) { ts.LineHeight = 2 }},
		{90, 200, defSty, func(ts *text.Style) { ts.WhiteSpace = text.WrapAlways }},
		{90, 200, defSty, func(ts *text.Style) { ts.Color = colors.Red; ts.SelectColor = colors.Uniform(colors.Blue) }},
		{300, 200, defSty, func(ts *text.Style) { ts.Align = text.Center }},
	}
	for round := range 3 {
		for ti, tx := range testTexts() {
			for vi, v := range variants {
				tsty := testStyle()
				v.style(tsty)
				size := math32.Vec2(v.width, v.height)
				txt := tx.Join()
				outs := copyOutputs(sh.ShapeTextOutput(tx, tsty, txt))
				lines, truncated := sh.WrapLinesOutput(outs, txt, tx, v.defSty, tsty, size)
				want := sh.LinesBounds(lines, truncated, tx, v.defSty, tsty, size)
				// A copy of the text is the source of the lines.
				src := slices.Clone(tx)
				got := sh.WrapLines(src, v.defSty, tsty, size)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("round %d text %d variant %d: cached lines differ:\n%v\n%v", round, ti, vi, got, want)
				}
				if len(got.Source) > 0 && &got.Source[0] != &src[0] {
					t.Errorf("round %d text %d variant %d: the source is not the given text", round, ti, vi)
				}
				// Changes to the lines that callers make are their own.
				got.SetGlyphXAdvance(fixed.I(3))
				for li := range got.Lines {
					ln := &got.Lines[li]
					ln.Offset.X += 7
					ln.Selections = append(ln.Selections, textpos.Range{Start: 0, End: 1})
					for _, sp := range ln.Source {
						for i := range sp {
							sp[i] = 'x'
						}
					}
				}
			}
		}
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

// TestSegmenterCache tests that wrapping long texts with the break
// opportunities that the shape cache keeps, at more sizes than it keeps
// wrappings for, gives the same lines as finding them for each wrapping.
func TestSegmenterCache(t *testing.T) {
	sh := NewShaper().(*Shaper)
	tsty := testStyle()
	defSty := rich.NewStyle()
	plain := rich.NewStyle()
	mono := *plain
	mono.Family = rich.Monospace
	para := "The quick brown fox jumps over the lazy dog, again and again, until it is tired. "
	tx := rich.NewText(plain, []rune(para+para))
	tx.AddSpan(&mono, []rune("Control+Shift+P\n"))
	tx.AddSpan(plain, []rune(para+"\r\n"+para+" "+para))
	txs := append(testTexts(), tx)
	for range 2 {
		for ti, tx := range txs {
			for w := 15; w < 900; w += 29 {
				size := math32.Vec2(float32(w), 2000)
				txt := tx.Join()
				outs := copyOutputs(sh.ShapeTextOutput(tx, tsty, txt))
				lines, truncated := sh.WrapLinesOutput(outs, txt, tx, defSty, tsty, size)
				want := sh.LinesBounds(lines, truncated, tx, defSty, tsty, size)
				got := sh.WrapLines(tx, defSty, tsty, size)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("text %d width %d: lines wrapped with the cached segmenter differ:\n%v\n%v", ti, w, got, want)
				}
			}
		}
	}
	se, _ := sh.shapes.get(tx, tsty)
	if se == nil || se.seg == nil || se.seg.Len() != len(tx.Join()) {
		t.Fatal("the long text has no segmenter of its runes")
	}
}

// BenchmarkSegmenterMemory reports the memory that the segmenter of a
// cached text takes, per rune of the text.
func BenchmarkSegmenterMemory(b *testing.B) {
	txt := []rune(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 20))
	var se *shapeEntry
	for b.Loop() {
		se = &shapeEntry{}
		se.segmenter(txt)
	}
	var ms0, ms1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&ms0)
	for range 1000 {
		se = &shapeEntry{}
		se.segmenter(txt)
	}
	runtime.ReadMemStats(&ms1)
	b.ReportMetric(float64(ms1.TotalAlloc-ms0.TotalAlloc)/1000/float64(len(txt)), "B/rune")
}
