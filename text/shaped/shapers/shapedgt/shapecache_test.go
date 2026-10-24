// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"fmt"
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
)

// codeLines returns n distinct lines of highlighted code, as an editor
// shapes them.
func codeLines(n int) []rich.Text {
	plain := rich.NewStyle()
	plain.Family = rich.Monospace
	kw := *plain
	kw.Weight = rich.Bold
	str := *plain
	str.Slant = rich.Italic
	lines := make([]rich.Text, n)
	for i := range lines {
		tx := rich.NewText(&kw, []rune("func "))
		tx.AddSpan(plain, []rune(fmt.Sprintf("line%d(x int) string { return fmt.Sprint(x * %d, ", i, i%97)))
		tx.AddSpan(&str, []rune(fmt.Sprintf(`"value %d"`, i)))
		tx.AddSpan(plain, []rune(") }"))
		lines[i] = tx
	}
	return lines
}

// scrollFrames calls frame for each frame of scrolling a view of the
// given number of lines through n lines, down and back up, by step lines.
func scrollFrames(n, view, step int, frame func(start, end int)) {
	for st := 0; st+view <= n; st += step {
		frame(st, st+view)
	}
	for st := n - view; st >= 0; st -= step {
		frame(st, st+view)
	}
}

// checkList checks that the list of entries in the order of their use has
// the entries of the cache.
func checkList(t *testing.T, sc *shapeCache) {
	t.Helper()
	n := 0
	var newer *shapeEntry
	for se := sc.newest; se != nil; newer, se = se, se.older {
		if se.newer != newer {
			t.Fatalf("entry %d of the list does not link to the newer one", n)
		}
		n++
	}
	if newer != sc.oldest || n != sc.n {
		t.Fatalf("list has %d entries, cache has %d", n, sc.n)
	}
	for _, se := range sc.entries {
		for ; se != nil; se = se.next {
			n--
		}
	}
	if n != 0 {
		t.Fatalf("list and hash chains differ by %d entries", n)
	}
}

// TestShapeCacheEviction tests that a full cache drops its least recently
// used texts, keeping a text that is used while many others are added.
func TestShapeCacheEviction(t *testing.T) {
	sh := NewShaper().(*Shaper)
	tsty := testStyle()
	defSty := rich.NewStyle()
	size := math32.Vec2(800, 20)
	hot := testTexts()
	lines := codeLines(maxShapeCache + 1000)
	for i, tx := range lines {
		sh.WrapLines(tx, defSty, tsty, size)
		if i%100 == 0 {
			for _, htx := range hot {
				sh.WrapLines(htx, defSty, tsty, size)
			}
		}
	}
	checkList(t, &sh.shapes)
	if sh.shapes.n != maxShapeCache {
		t.Errorf("cached %d texts, want %d", sh.shapes.n, maxShapeCache)
	}
	for ti, tx := range hot {
		if se, _ := sh.shapes.get(tx, tsty); se == nil {
			t.Errorf("text %d that is in use was dropped", ti)
		}
	}
	if se, _ := sh.shapes.get(lines[0], tsty); se != nil {
		t.Error("least recently used text was kept")
	}
	if se, _ := sh.shapes.get(lines[len(lines)-1], tsty); se == nil {
		t.Error("most recent text was dropped")
	}

	// Forgetting texts keeps the list.
	removed := sh.ForgetTexts(func(tx rich.Text, customFont rich.FontName) bool {
		return len(tx) == 4
	})
	if removed == 0 {
		t.Error("no texts forgotten")
	}
	checkList(t, &sh.shapes)
}

// wrapCounting returns a function that wraps a text as an editor does, and
// one that reports the number of texts that were not cached per op.
func wrapCounting(b *testing.B, sh *Shaper) (wrap func(tx rich.Text, size math32.Vector2), report func()) {
	tsty := testStyle()
	defSty := rich.NewStyle()
	misses := 0
	wrap = func(tx rich.Text, size math32.Vector2) {
		if se, _ := sh.shapes.get(tx, tsty); se == nil {
			misses++
		}
		sh.WrapLines(tx, defSty, tsty, size)
	}
	report = func() {
		b.ReportMetric(float64(misses)/float64(b.N), "shapes/op")
	}
	return
}

// BenchmarkShapeCacheScroll scrolls an editor view down and back up
// through a text with more lines than the shape cache holds.
func BenchmarkShapeCacheScroll(b *testing.B) {
	sh := NewShaper().(*Shaper)
	wrap, report := wrapCounting(b, sh)
	defer report()
	lines := codeLines(3 * maxShapeCache)
	size := math32.Vec2(800, 20)
	for b.Loop() {
		scrollFrames(len(lines), 60, 20, func(start, end int) {
			for _, tx := range lines[start:end] {
				wrap(tx, size)
			}
		})
	}
}

// BenchmarkShapeCacheScrollHot scrolls an editor view as
// [BenchmarkShapeCacheScroll] while other texts are laid out again in each
// frame, as are those of the rest of a window.
func BenchmarkShapeCacheScrollHot(b *testing.B) {
	sh := NewShaper().(*Shaper)
	wrap, report := wrapCounting(b, sh)
	defer report()
	lines := codeLines(3 * maxShapeCache)
	hot := testTexts()
	for i := range 200 {
		hot = append(hot, rich.NewText(rich.NewStyle(), []rune(fmt.Sprintf("Label %d of a widget", i))))
	}
	size := math32.Vec2(800, 20)
	for b.Loop() {
		scrollFrames(len(lines), 60, 20, func(start, end int) {
			for _, tx := range hot {
				wrap(tx, size)
			}
			for _, tx := range lines[start:end] {
				wrap(tx, size)
			}
		})
	}
}

// BenchmarkShapeCacheRewrap wraps the lines of a view at two widths in
// each frame, as while resizing an editor, while it scrolls down and back
// up as [BenchmarkShapeCacheScroll], with the other texts of
// [BenchmarkShapeCacheScrollHot].
func BenchmarkShapeCacheRewrap(b *testing.B) {
	sh := NewShaper().(*Shaper)
	wrap, report := wrapCounting(b, sh)
	defer report()
	lines := codeLines(3 * maxShapeCache)
	hot := testTexts()
	for i := range 200 {
		hot = append(hot, rich.NewText(rich.NewStyle(), []rune(fmt.Sprintf("Label %d of a widget", i))))
	}
	for b.Loop() {
		w := float32(300)
		scrollFrames(len(lines), 60, 20, func(start, end int) {
			w = 600 - w
			for _, size := range []math32.Vector2{math32.Vec2(w, 20), math32.Vec2(w+150, 20)} {
				for _, tx := range hot {
					wrap(tx, size)
				}
				for _, tx := range lines[start:end] {
					wrap(tx, size)
				}
			}
		})
	}
}
