// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

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
