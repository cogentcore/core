// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"slices"
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
)

// TestForgetTexts tests that the texts that ForgetTexts forgets are shaped
// again, and that the others stay cached.
func TestForgetTexts(t *testing.T) {
	sh := NewShaper().(*Shaper)
	tsty := testStyle()
	defSty := rich.NewStyle()
	txs := testTexts()
	size := math32.Vec2(250, 200)
	for _, tx := range txs {
		sh.WrapLines(tx, defSty, tsty, size)
	}
	hasLine := func(tx rich.Text, customFont rich.FontName) bool {
		return slices.Contains(tx.Join(), '\n')
	}
	forgotten := 0
	for _, tx := range txs {
		if hasLine(tx, "") {
			forgotten++
		}
	}
	if n := sh.ForgetTexts(hasLine); n != forgotten || n == 0 {
		t.Errorf("forgot %d texts, want %d", n, forgotten)
	}
	if sh.shapes.n != len(txs)-forgotten {
		t.Errorf("%d cached texts, want %d", sh.shapes.n, len(txs)-forgotten)
	}
	for _, tx := range txs {
		se, _ := sh.shapes.get(tx, tsty)
		if (se == nil) != hasLine(tx, "") {
			t.Errorf("%q: cached %v", string(tx.Join()), se != nil)
		}
	}

	// The forgotten texts are shaped again as before.
	for _, tx := range txs {
		sh.WrapLines(tx, defSty, tsty, size)
	}
	if sh.shapes.n != len(txs) {
		t.Errorf("%d cached texts, want %d", sh.shapes.n, len(txs))
	}
}
