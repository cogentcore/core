// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
	"golang.org/x/image/math/fixed"
)

// Run is a span of shaped text with the same font properties,
// with layout information to enable GUI interaction with shaped text.
type Run interface {

	// AsBase returns the base type with relevant shaped text information.
	AsBase() *RunBase

	// LineBounds returns the Line-level Bounds for given Run as rect bounding box.
	LineBounds() math32.Box2

	// Runes returns our rune range in original source using textpos.Range.
	Runes() textpos.Range

	// Advance returns the total distance to advance in going from one run to the next.
	Advance() float32

	// RuneBounds returns the maximal line-bounds level bounding box for given rune index.
	RuneBounds(ri int) math32.Box2

	// RuneAtPoint returns the rune index in Lines source, at given rendered location,
	// based on given starting location for rendering. If the point is out of the
	// line bounds, the nearest point is returned (e.g., start of line based on Y coordinate).
	RuneAtPoint(src rich.Text, pt math32.Vector2, start math32.Vector2) int

	// SetGlyphXAdvance sets the x advance on all glyphs to given value:
	// for monospaced case.
	SetGlyphXAdvance(adv fixed.Int26_6)
}

// Run is a span of text with the same font properties, with full rendering information.
type RunBase struct {

	// Font is the [text.Font] compact encoding of the font to use for rendering.
	Font text.Font

	// MaxBounds are the maximal line-level bounds for this run, suitable for region
	// rendering and mouse interaction detection.
	MaxBounds math32.Box2

	// Decoration are the decorations from the style to apply to this run.
	Decoration rich.Decorations

	// Path is an optional rendering of the text directly in glyph paths,
	// which is used for Math formatting via the tex package.
	Path *ppath.Path

	// FillColor is the color to use for glyph fill (i.e., the standard "ink" color).
	// Will only be non-nil if set for this run; Otherwise use default.
	FillColor image.Image

	// StrokeColor is the color to use for glyph outline stroking, if non-nil.
	StrokeColor image.Image

	// Background is the color to use for the background region, if non-nil.
	Background image.Image
}

func (run *RunBase) SetFromStyle(sty *rich.Style, tsty *text.Style) {
	run.Decoration = sty.Decoration
	run.Font = *text.NewFont(sty, tsty)
	if sty.Decoration.HasFlag(rich.FillColor) {
		run.FillColor = colors.Uniform(sty.FillColor)
	}
	if sty.Decoration.HasFlag(rich.StrokeColor) {
		run.StrokeColor = colors.Uniform(sty.StrokeColor)
	}
	if sty.Decoration.HasFlag(rich.Background) {
		run.Background = colors.Uniform(sty.Background)
	}
}
