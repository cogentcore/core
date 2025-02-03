// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"image"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
)

// Lines is a list of Lines of shaped text, with an overall bounding
// box and position for the entire collection. This is the renderable
// unit of text, satisfying the [render.Item] interface.
type Lines struct {
	// Source is the original input source that generated this set of lines.
	// Each Line has its own set of spans that describes the Line contents.
	Source rich.Spans

	// Lines are the shaped lines.
	Lines []Line

	// Position specifies the absolute position within a target render image
	// where the lines are to be rendered, specifying the
	// baseline position (not the upper left: see Bounds for that).
	Position math32.Vector2

	// Bounds is the bounding box for the entire set of rendered text,
	// starting at Position. Use Size() method to get the size and ToRect()
	// to get an [image.Rectangle].
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

	// SelectionColor is the color to use for rendering selected regions.
	SelectionColor image.Image

	// Context is our rendering context
	Context render.Context
}

// render.Item interface assertion.
func (ls *Lines) IsRenderItem() {
}

// Line is one line of shaped text, containing multiple Runs.
// This is not an independent render target: see [Lines] (can always
// use one Line per Lines as needed).
type Line struct {
	// Source is the input source corresponding to the line contents,
	// derived from the original Lines Source. The style information for
	// each Run is embedded here.
	Source rich.Spans

	// Runs are the shaped [Run] elements, in one-to-one correspondance with
	// the Source spans.
	Runs []Run

	// Offset specifies the relative offset from the Lines Position
	// determining where to render the line in a target render image.
	// This is the baseline position (not the upper left: see Bounds for that).
	Offset math32.Vector2

	// Bounds is the bounding box for the entire set of rendered text,
	// starting at the effective render position based on Offset relative to
	// [Lines.Position]. Use Size() method to get the size and ToRect()
	// to get an [image.Rectangle].
	Bounds math32.Box2

	// Selections specifies region(s) within this line that are selected,
	// and will be rendered with the [Lines.SelectionColor] background,
	// replacing any other background color that might have been specified.
	Selections []textpos.Range
}
