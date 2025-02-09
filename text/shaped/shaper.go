// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// NewShaper returns the correct type of shaper.
var NewShaper func() Shaper

// Shaper is a text shaping system that can shape the layout of [rich.Text],
// including line wrapping.
type Shaper interface {

	// Shape turns given input spans into [Runs] of rendered text,
	// using given context needed for complete styling.
	// The results are only valid until the next call to Shape or WrapParagraph:
	// use slices.Clone if needed longer than that.
	Shape(tx rich.Text, tsty *text.Style, rts *rich.Settings) []Run

	// WrapLines performs line wrapping and shaping on the given rich text source,
	// using the given style information, where the [rich.Style] provides the default
	// style information reflecting the contents of the source (e.g., the default family,
	// weight, etc), for use in computing the default line height. Paragraphs are extracted
	// first using standard newline markers, assumed to coincide with separate spans in the
	// source text, and wrapped separately. For horizontal text, the Lines will render with
	// a position offset at the upper left corner of the overall bounding box of the text.
	WrapLines(tx rich.Text, defSty *rich.Style, tsty *text.Style, rts *rich.Settings, size math32.Vector2) *Lines
}

// WrapSizeEstimate is the size to use for layout during the SizeUp pass,
// for word wrap case, where the sizing actually matters,
// based on trying to fit the given number of characters into the given content size
// with given font height, and ratio of width to height.
// Ratio is used when csz is 0: 1.618 is golden, and smaller numbers to allow
// for narrower, taller text columns.
func WrapSizeEstimate(csz math32.Vector2, nChars int, ratio float32, sty *rich.Style, tsty *text.Style) math32.Vector2 {
	chars := float32(nChars)
	fht := tsty.FontHeight(sty)
	if fht == 0 {
		fht = 16
	}
	area := chars * fht * fht
	if csz.X > 0 && csz.Y > 0 {
		ratio = csz.X / csz.Y
		// fmt.Println(lb, "content size ratio:", ratio)
	}
	// w = ratio * h
	// w^2 + h^2 = a
	// (ratio*h)^2 + h^2 = a
	h := math32.Sqrt(area) / math32.Sqrt(ratio+1)
	w := ratio * h
	if w < csz.X { // must be at least this
		w = csz.X
		h = area / w
		h = max(h, csz.Y)
	}
	sz := math32.Vec2(w, h)
	return sz
}
