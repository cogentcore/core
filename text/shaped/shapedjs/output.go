// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"cogentcore.org/core/text/textpos"
	"github.com/go-text/typesetting/di"
	"golang.org/x/image/math/fixed"
)

type Output struct {
	// Advance is the distance the Dot has advanced.
	// It is typically positive for horizontal text, negative for vertical.
	Advance fixed.Int26_6
	// Size is copied from the shaping.Input.Size that produced this Output.
	Size fixed.Int26_6
	// Glyphs are the shaped output text.
	Glyphs []Glyph
	// LineBounds describes the font's suggested line bounding dimensions. The
	// dimensions described should contain any glyphs from the given font.
	LineBounds Bounds
	// GlyphBounds describes a tight bounding box on the specific glyphs contained
	// within this output. The dimensions may not be sufficient to contain all
	// glyphs within the chosen font.
	//
	// Its [Gap] field is always zero.
	GlyphBounds Bounds

	// Direction is the direction used to shape the text,
	// as provided in the Input.
	Direction di.Direction

	// Runes describes the runes this output represents from the input text.
	Runes textpos.Range

	// Face is the font face that this output is rendered in. This is needed in
	// the output in order to render each run in a multi-font sequence in the
	// correct font.
	// Face *font.Face

	// VisualIndex is the visual position of this run within its containing line where
	// 0 indicates the leftmost run and increasing values move to the right. This is
	// useful for sorting the runs for drawing purposes.
	VisualIndex int32
}

// Glyph describes the attributes of a single glyph from a single
// font face in a shaped output.
type Glyph struct {
	// Width is the width of the glyph content,
	// expressed as a distance from the [XBearing],
	// typically positive
	Width float32
	// Height is the height of the glyph content,
	// expressed as a distance from the [YBearing],
	// typically negative
	Height float32
	// XBearing is the distance between the dot (with offset applied) and
	// the glyph content, typically positive for horizontal text;
	// often negative for vertical text.
	XBearing float32
	// YBearing is the distance between the dot (with offset applied) and
	// the top of the glyph content, typically positive
	YBearing float32
	// XAdvance is the distance between the current dot (without offset applied) and the next dot.
	// It is typically positive for horizontal text, and always zero for vertical text.
	XAdvance float32
	// YAdvance is the distance between the current dot (without offset applied) and the next dot.
	// It is typically negative for vertical text, and always zero for horizontal text.
	YAdvance float32

	// Offsets to be applied to the dot before actually drawing
	// the glyph.
	// For vertical text, YOffset is typically used to position the glyph
	// below the horizontal line at the dot
	XOffset, YOffset float32

	// ClusterIndex is the lowest rune index of all runes shaped into
	// this glyph cluster. All glyphs sharing the same cluster value
	// are part of the same cluster and will have identical RuneCount
	// and GlyphCount fields.
	ClusterIndex int
	// RuneCount is the number of input runes shaped into this output
	// glyph cluster.
	RuneCount int
	// GlyphCount is the number of glyphs in this output glyph cluster.
	GlyphCount int
	GlyphID    uint32
	Mask       uint32

	// startLetterSpacing and endLetterSpacing are set when letter spacing is applied,
	// measuring the whitespace added on one side (half of the user provided letter spacing)
	// The line wrapper will ignore [endLetterSpacing] when deciding where to break,
	// and will trim [startLetterSpacing] at the start of the lines
	startLetterSpacing, endLetterSpacing float32
}

type Bounds struct {
	// Ascent is the maximum ascent away from the baseline. This value is typically
	// positive in coordiate systems that grow up.
	Ascent float32
	// Descent is the maximum descent away from the baseline. This value is typically
	// negative in coordinate systems that grow up.
	Descent float32
	// Gap is the height of empty pixels between lines. This value is typically positive
	// in coordinate systems that grow up.
	Gap float32
}
