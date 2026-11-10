// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"strings"
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
	"github.com/stretchr/testify/assert"
)

// TestTruncation tests that text is clamped to its LineClamp lines, and
// that text that does not wrap and has an Ellipsis is cut to its width,
// both ending with an ellipsis (…), and that the cached wrappings keep it.
func TestTruncation(t *testing.T) {
	sh := NewShaper().(*Shaper)
	defSty := rich.NewStyle()
	long := rich.NewText(defSty, []rune(strings.Repeat("words that wrap in a narrow width ", 6)))
	// endsWithEllipsis returns whether the given line ends with an ellipsis
	endsWithEllipsis := func(lines *shaped.Lines, li int, tsty *text.Style) bool {
		runs := lines.Lines[li].Runs
		if len(runs) == 0 {
			return false
		}
		r, ok := runs[len(runs)-1].(*Run)
		ell := sh.truncator(defSty, tsty)
		return ok && len(r.Output.Glyphs) == 1 && r.Output.Glyphs[0].GlyphID == ell.Glyphs[0].GlyphID
	}
	for range 2 { // (the second from the caches)
		tsty := testStyle()
		lines := sh.WrapLines(long, defSty, tsty, math32.Vec2(200, 1e6))
		assert.Greater(t, len(lines.Lines), 2, "it wraps")
		assert.False(t, lines.Truncated)

		tsty.LineClamp = 2
		lines = sh.WrapLines(long, defSty, tsty, math32.Vec2(200, 1e6))
		assert.Len(t, lines.Lines, 2, "clamped")
		assert.True(t, lines.Truncated)
		assert.LessOrEqual(t, lines.Bounds.Size().X, float32(201))
		assert.True(t, endsWithEllipsis(lines, 1, tsty), "ends with an ellipsis")

		tsty = testStyle()
		tsty.WhiteSpace = text.WrapNever
		lines = sh.WrapLines(long, defSty, tsty, math32.Vec2(200, 1e6))
		assert.Len(t, lines.Lines, 1)
		assert.Greater(t, lines.Bounds.Size().X, float32(400), "on one line")

		tsty.Ellipsis = true
		lines = sh.WrapLines(long, defSty, tsty, math32.Vec2(200, 1e6))
		assert.Len(t, lines.Lines, 1)
		assert.True(t, lines.Truncated)
		assert.LessOrEqual(t, lines.Bounds.Size().X, float32(201), "cut to its width")
		assert.True(t, endsWithEllipsis(lines, 0, tsty), "ends with an ellipsis")
	}
}
