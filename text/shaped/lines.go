// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"fmt"
	"image"
	"image/color"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

// Lines is a list of Lines of shaped text, with an overall bounding
// box and position for the entire collection. This is the renderable
// unit of text, although it is not a [render.Item] because it lacks
// a position, and it can potentially be re-used in different positions.
type Lines struct {

	// Source is the original input source that generated this set of lines.
	// Each Line has its own set of spans that describes the Line contents.
	Source rich.Spans

	// Lines are the shaped lines.
	Lines []Line

	// Offset is an optional offset to add to the position given when rendering.
	Offset math32.Vector2

	// Bounds is the bounding box for the entire set of rendered text,
	// relative to a rendering Position (and excluding any contribution
	// of Offset). This is centered at the baseline and the upper left
	// typically has a negative Y. Use Size() method to get the size
	// and ToRect() to get an [image.Rectangle].
	Bounds math32.Box2

	// FontSize is the [rich.Context] StandardSize from the Context used
	// at the time of shaping. Actual lines can be larger depending on font
	// styling parameters.
	FontSize float32

	// LineHeight is the line height used at the time of shaping.
	LineHeight float32

	// Truncated indicates whether any lines were truncated.
	Truncated bool

	// Direction is the default text rendering direction from the Context.
	Direction rich.Directions

	// Links holds any hyperlinks within shaped text.
	Links []Link

	// Color is the default fill color to use for inking text.
	Color color.Color

	// SelectionColor is the color to use for rendering selected regions.
	SelectionColor image.Image
}

// Line is one line of shaped text, containing multiple Runs.
// This is not an independent render target: see [Lines] (can always
// use one Line per Lines as needed).
type Line struct {

	// Source is the input source corresponding to the line contents,
	// derived from the original Lines Source. The style information for
	// each Run is embedded here.
	Source rich.Spans

	// SourceRange is the range of runes in the original [Lines.Source] that
	// are represented in this line.
	SourceRange textpos.Range

	// Runs are the shaped [Run] elements, in one-to-one correspondance with
	// the Source spans.
	Runs []Run

	// Offset specifies the relative offset from the Lines Position
	// determining where to render the line in a target render image.
	// This is the baseline position (not the upper left: see Bounds for that).
	Offset math32.Vector2

	// Bounds is the bounding box for the Line of rendered text,
	// relative to the baseline rendering position (excluding any contribution
	// of Offset). This is centered at the baseline and the upper left
	// typically has a negative Y. Use Size() method to get the size
	// and ToRect() to get an [image.Rectangle]. This is based on the output
	// LineBounds, not the actual GlyphBounds.
	Bounds math32.Box2

	// Selections specifies region(s) of runes within this line that are selected,
	// and will be rendered with the [Lines.SelectionColor] background,
	// replacing any other background color that might have been specified.
	Selections []textpos.Range
}

// Run is a span of text with the same font properties, with full rendering information.
type Run struct {
	shaping.Output

	// MaxBounds are the maximal line-level bounds for this run, suitable for region
	// rendering and mouse interaction detection.
	MaxBounds math32.Box2

	// Decoration are the decorations from the style to apply to this run.
	Decoration rich.Decorations

	//	FillColor is the color to use for glyph fill (i.e., the standard "ink" color).
	// Will only be non-nil if set for this run; Otherwise use default.
	FillColor image.Image

	//	StrokeColor is the color to use for glyph outline stroking, if non-nil.
	StrokeColor image.Image

	//	Background is the color to use for the background region, if non-nil.
	Background image.Image
}

func (ln *Line) String() string {
	return ln.Source.String() + fmt.Sprintf(" runs: %d\n", len(ln.Runs))
}

func (ls *Lines) String() string {
	str := ""
	for li := range ls.Lines {
		ln := &ls.Lines[li]
		str += fmt.Sprintf("#### Line: %d\n", li)
		str += ln.String()
	}
	return str
}

// GlyphBounds returns the tight bounding box for given glyph within this run.
func (rn *Run) GlyphBounds(g *shaping.Glyph) fixed.Rectangle26_6 {
	if rn.Direction.IsVertical() {
		if rn.Direction.IsSideways() {
			fmt.Println("sideways")
			return fixed.Rectangle26_6{Min: fixed.Point26_6{X: g.XBearing, Y: -g.YBearing}, Max: fixed.Point26_6{X: g.XBearing + g.Width, Y: -g.YBearing - g.Height}}
		}
		return fixed.Rectangle26_6{Min: fixed.Point26_6{X: -g.XBearing - g.Width/2, Y: g.Height - g.YOffset}, Max: fixed.Point26_6{X: g.XBearing + g.Width/2, Y: -(g.YBearing + g.Height) - g.YOffset}}
	}
	return fixed.Rectangle26_6{Min: fixed.Point26_6{X: g.XBearing, Y: -g.YBearing}, Max: fixed.Point26_6{X: g.XBearing + g.Width, Y: -g.YBearing - g.Height}}
}

// GlyphSelectBounds returns the maximal line-bounds level bounding box for given
// glyph, suitable for selection.
func (rn *Run) GlyphSelectBounds(g *shaping.Glyph) fixed.Rectangle26_6 {
	return fixed.Rectangle26_6{}
}

// Bounds returns the LineBounds for given Run as rect bounding box,
// which can easily be converted to math32.Box2.
func (rn *Run) Bounds() fixed.Rectangle26_6 {
	gapdec := rn.LineBounds.Descent
	if gapdec < 0 && rn.LineBounds.Gap < 0 || gapdec > 0 && rn.LineBounds.Gap > 0 {
		gapdec += rn.LineBounds.Gap
	} else {
		gapdec -= rn.LineBounds.Gap
	}
	if rn.Direction.IsVertical() {
		// ascent, descent describe horizontal, advance is vertical
		return fixed.Rectangle26_6{Min: fixed.Point26_6{X: -rn.LineBounds.Ascent, Y: 0},
			Max: fixed.Point26_6{X: -gapdec, Y: -rn.Advance}}
	}
	return fixed.Rectangle26_6{Min: fixed.Point26_6{X: 0, Y: -rn.LineBounds.Ascent},
		Max: fixed.Point26_6{X: rn.Advance, Y: -gapdec}}
}
