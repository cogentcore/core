// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped_test

import (
	"math/rand"
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	. "cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
	"github.com/stretchr/testify/assert"
)

// runeBoundsChars are the characters of the random texts of
// [TestRuneBoundsFuzz], with those that make ligatures (fi, fl, ff).
const runeBoundsChars = "abcdefghijklmnopqrstuvwxyz ABCXYZ fffiflffi 0123 .,:;'\"-_()[]/\\!?@#$%&*+=<>"

// checkRuneBounds checks that the bounds of each rune of the given text,
// shaped on one line, are on that line, and go to the right.
func checkRuneBounds(t *testing.T, sh Shaper, tsty *text.Style, s string) bool {
	t.Helper()
	sty := rich.NewStyle()
	tx := rich.NewText(sty, []rune(s))
	lns := sh.WrapLines(tx, sty, tsty, math32.Vec2(10000, 1000))
	if !assert.Len(t, lns.Lines, 1, "%q", s) {
		return false
	}
	ln := lns.Lines[0]
	top := lns.Offset.Y + ln.Offset.Y + ln.Bounds.Min.Y
	ok := true
	lastX := float32(-100)
	for i := range len([]rune(s)) {
		bb := lns.RuneBounds(i)
		ok = assert.InDelta(t, top, bb.Min.Y, 0.01, "%q: top of rune %d %q", s, i, []rune(s)[i]) && ok
		ok = assert.Greater(t, bb.Size().Y, float32(0), "%q: height of rune %d", s, i) && ok
		ok = assert.GreaterOrEqual(t, bb.Min.X, lastX, "%q: left of rune %d %q", s, i, []rune(s)[i]) && ok
		lastX = bb.Min.X
		if !ok {
			return false
		}
	}
	return true
}

// TestRuneBoundsFuzz tests the rune bounds of random short texts,
// on which the cursor of a text field is drawn.
func TestRuneBoundsFuzz(t *testing.T) {
	if testing.Short() {
		t.Skip("random texts")
	}
	uc := units.Context{}
	uc.Defaults()
	tsty := text.NewStyle()
	tsty.ToDots(&uc)
	sh := NewShaper()
	for _, s := range []string{"fix the bug", "file", "off", "affine", "a  b", " leading", "trailing "} {
		checkRuneBounds(t, sh, tsty, s)
	}
	rnd := rand.New(rand.NewSource(1))
	chars := []rune(runeBoundsChars)
	for range 2000 {
		n := 1 + rnd.Intn(20)
		rs := make([]rune, n)
		for i := range rs {
			rs[i] = chars[rnd.Intn(len(chars))]
		}
		if !checkRuneBounds(t, sh, tsty, string(rs)) {
			return
		}
	}
}
