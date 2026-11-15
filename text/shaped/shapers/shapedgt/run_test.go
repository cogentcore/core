// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/text/fonts"
	"cogentcore.org/core/text/rich"
	"github.com/cogentcore/typesetting/di"
	"github.com/cogentcore/typesetting/shaping"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/math/fixed"
)

// shapeRuns shapes the given string with a shaper of the embedded fonts.
func shapeRuns(t *testing.T, s string) (*Shaper, rich.Text, []*Run) {
	t.Helper()
	sh := NewShaperWithFonts(fonts.Embedded).(*Shaper)
	tx := rich.NewText(rich.NewStyle(), []rune(s))
	runs := sh.Shape(tx, testStyle())
	require.NotEmpty(t, runs)
	rs := make([]*Run, len(runs))
	for i, r := range runs {
		rs[i] = r.(*Run)
	}
	return sh, tx, rs
}

// TestRunGlyphs tests finding the glyphs of runes, the bounds of glyphs
// and runes, and the rune at a point, in a run of horizontal text.
func TestRunGlyphs(t *testing.T) {
	_, tx, runs := shapeRuns(t, "office ok")
	run := runs[0]
	rr := run.Runes()
	assert.Equal(t, 0, rr.Start)
	assert.Equal(t, 9, rr.End)
	assert.Positive(t, run.Advance())
	assert.Same(t, &run.RunBase, run.AsBase())

	nr := rr.End
	assert.LessOrEqual(t, len(run.Glyphs), nr, "a ligature has one glyph for more runes")
	assert.Equal(t, 0, run.FirstGlyphAt(0))
	assert.Equal(t, -1, run.FirstGlyphAt(nr+1))
	assert.Equal(t, len(run.Glyphs)-1, run.LastGlyphAt(nr))
	assert.Equal(t, -1, run.LastGlyphAt(-1))
	assert.Equal(t, []int{0}, run.GlyphsAt(0))

	lb := run.LineBounds()
	prevX := float32(-100)
	for ri := range nr {
		rb := run.RuneBounds(ri)
		if tx.Join()[ri] == ' ' {
			continue
		}
		assert.Greater(t, rb.Size().X, float32(0), "rune %d", ri)
		assert.Greater(t, rb.Min.X, prevX, "rune %d is after the last", ri)
		prevX = rb.Min.X
		assert.Equal(t, lb.Min.Y, rb.Min.Y, "line bounds")
		// the rune at the center of its bounds is itself
		c := rb.Center()
		got := run.RuneAtPoint(tx, c, math32.Vector2{})
		gis := run.GlyphsAt(ri)
		if len(gis) > 0 {
			assert.Equal(t, ri, got, "rune %d", ri)
		}
	}
	assert.Equal(t, rr.End, run.RuneAtPoint(tx, math32.Vec2(1e6, 0), math32.Vector2{}), "past the end")

	g := &run.Glyphs[0]
	gb := run.GlyphBoundsBox(g)
	assert.Greater(t, gb.Size().X, float32(0))
	glb := run.GlyphLineBoundsBox(g)
	assert.Equal(t, gb.Min.X, glb.Min.X)
	assert.Equal(t, lb.Min.Y, glb.Min.Y)
	assert.Equal(t, math32.B2FromFixed(run.RunBounds()).Max.X, run.Advance())
	assert.Equal(t, run.Advance(), lb.Max.X)
}

// TestRunClusters tests the bounds of runes that are shaped into the glyph
// of a cluster that they do not start, as the i of a ligature, and in
// right-to-left text.
func TestRunClusters(t *testing.T) {
	_, _, runs := shapeRuns(t, "fi")
	run := runs[0]
	if len(run.Glyphs) != 1 {
		t.Skip("the font has no fi ligature")
	}
	f := run.RuneBounds(0)
	i := run.RuneBounds(1)
	// the f has the bounds of the whole glyph, and the i its share of it
	adv := run.Advance()
	assert.InDelta(t, adv/2, i.Size().X, 0.01, "an even share")
	assert.InDelta(t, adv/2, i.Min.X, 0.01, "after the f")
	assert.Less(t, f.Min.X, i.Min.X)
	assert.GreaterOrEqual(t, f.Max.X, i.Max.X)
	assert.Equal(t, math32.Box2{}, run.clusterRuneBounds(5), "not in a cluster")

	run.Direction = di.DirectionRTL
	assert.InDelta(t, 0, run.clusterRuneBounds(1).Min.X, 0.01, "the i comes first from the left")
	run.Direction = di.DirectionTTB
	assert.Equal(t, math32.Box2{}, run.clusterRuneBounds(1))
	assert.Equal(t, math32.Box2{}, run.GlyphRegionBounds(0, 0), "not for vertical text")
}

// TestRunVerticalBounds tests the bounds of glyphs and runs of vertical
// and sideways text, and of math, which has the bounds of its path.
func TestRunVerticalBounds(t *testing.T) {
	_, _, runs := shapeRuns(t, "ab")
	run := runs[0]
	g := &run.Glyphs[0]
	horiz := run.GlyphBounds(g)
	run.Direction = di.DirectionTTB
	vert := run.GlyphBounds(g)
	assert.NotEqual(t, horiz, vert)
	assert.NotEqual(t, run.GlyphLineBounds(g), fixed.Rectangle26_6{})
	rb := run.RunBounds()
	assert.Equal(t, -run.Output.Advance, rb.Max.Y, "the advance is vertical")
	run.Direction.SetSideways(true)
	assert.Equal(t, horiz, run.GlyphBounds(g))
	assert.Equal(t, horiz, run.GlyphLineBounds(g))

	run.Direction = di.DirectionLTR
	mb := math32.B2(1, 2, 3, 4)
	run.MaxBounds = mb
	run.Math.Path = *ppath.New().Rectangle(1, 2, 2, 2)
	assert.Equal(t, mb, run.GlyphBoundsBox(g))
	assert.Equal(t, mb, run.GlyphLineBoundsBox(g))
	assert.Equal(t, mb, run.LineBounds())
	assert.Equal(t, mb.ToFixed(), run.Bounds())
	assert.Equal(t, mb.ToFixed(), run.RunBounds())
	assert.Equal(t, mb.ToFixed(), run.GlyphBounds(g))
	assert.Equal(t, mb.ToFixed(), run.GlyphLineBounds(g))
}

// TestShaperAPI tests the shaper with given fonts, its font map, which
// resets its caches, the cached wrapping, and the list of fonts.
func TestShaperAPI(t *testing.T) {
	sh, tx, _ := shapeRuns(t, "hello world")
	tsty := testStyle()
	defSty := rich.NewStyle()
	sh.Lock()
	lines, trunc := sh.WrapLinesCached(tx, defSty, tsty, math32.Vec2(500, 100), nil)
	again, _ := sh.WrapLinesCached(tx, defSty, tsty, math32.Vec2(500, 100), nil)
	sh.Unlock()
	assert.Len(t, lines, 1)
	assert.Equal(t, 0, trunc)
	assert.Equal(t, lines, again, "from the cache")
	adjusted := 0
	sh.Lock()
	sh.ResetCache()
	sh.WrapLinesCached(tx, defSty, tsty, math32.Vec2(30, 100), func(outs []shaping.Output) { adjusted++ })
	sh.Unlock()
	assert.Positive(t, adjusted)

	assert.NotNil(t, sh.FontMap())
	assert.False(t, sh.fontMap.hasQuery, "forgotten")
	(&nilLogger{}).Printf("nothing %d", 1)

	if testing.Short() {
		return
	}
	fl := sh.FontList()
	assert.NotEmpty(t, fl, "the system fonts")
}

// TestIsTruncator tests recognizing the truncator output by its glyphs.
func TestIsTruncator(t *testing.T) {
	tr := shaping.Output{Glyphs: make([]shaping.Glyph, 1)}
	assert.True(t, isTruncator(tr, tr))
	assert.False(t, isTruncator(shaping.Output{Glyphs: make([]shaping.Glyph, 1)}, tr))
	assert.False(t, isTruncator(shaping.Output{}, tr))
}
