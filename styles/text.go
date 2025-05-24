// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// Text has styles for text layout styling.
// Most of these are inherited
type Text struct { //types:add

	// Align specifies how to align text along the default direction (inherited).
	// This *only* applies to the text within its containing element,
	// and is relevant only for multi-line text.
	Align text.Aligns

	// AlignV specifies "vertical" (orthogonal to default direction)
	// alignment of text (inherited).
	// This *only* applies to the text within its containing element:
	// if that element does not have a specified size
	// that is different from the text size, then this has *no effect*.
	AlignV text.Aligns

	// LineHeight is a multiplier on the default font size for spacing between lines.
	// If there are larger font elements within a line, they will be accommodated, with
	// the same amount of total spacing added above that maximum size as if it was all
	// the same height. The default of 1.3 represents standard "single spaced" text.
	LineHeight float32 `default:"1.3"`

	// WhiteSpace (not inherited) specifies how white space is processed,
	// and how lines are wrapped.  If set to WhiteSpaceNormal (default) lines are wrapped.
	// See info about interactions with Grow.X setting for this and the NoWrap case.
	WhiteSpace text.WhiteSpaces

	// Direction specifies the default text direction, which can be overridden if the
	// unicode text is typically written in a different direction.
	Direction rich.Directions

	// TabSize specifies the tab size, in number of characters (inherited).
	TabSize int

	// SelectColor is the color to use for the background region of selected text (inherited).
	SelectColor image.Image

	// HighlightColor is the color to use for the background region of highlighted text (inherited).
	HighlightColor image.Image
}

func (ts *Text) Defaults() {
	ts.Align = text.Start
	ts.AlignV = text.Start
	ts.LineHeight = 1.3
	ts.Direction = rich.LTR
	ts.TabSize = 4
	ts.SelectColor = colors.Scheme.Select.Container
	ts.HighlightColor = colors.Scheme.Warn.Container
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (ts *Text) ToDots(uc *units.Context) {
}

// InheritFields from parent
func (ts *Text) InheritFields(parent *Text) {
	ts.Align = parent.Align
	ts.AlignV = parent.AlignV
	ts.LineHeight = parent.LineHeight
	// ts.WhiteSpace = par.WhiteSpace // note: we can't inherit this b/c label base default then gets overwritten
	ts.Direction = parent.Direction
	ts.TabSize = parent.TabSize
	ts.SelectColor = parent.SelectColor
	ts.HighlightColor = parent.HighlightColor
}

// SetText sets the text.Style from this style.
func (ts *Text) SetText(tsty *text.Style) {
	tsty.Align = ts.Align
	tsty.AlignV = ts.AlignV
	tsty.LineHeight = ts.LineHeight
	tsty.WhiteSpace = ts.WhiteSpace
	tsty.Direction = ts.Direction
	tsty.TabSize = ts.TabSize
	tsty.SelectColor = ts.SelectColor
	tsty.HighlightColor = ts.HighlightColor
}

// SetFromText sets from the given [text.Style].
func (ts *Text) SetFromText(tsty *text.Style) {
	ts.Align = tsty.Align
	ts.AlignV = tsty.AlignV
	ts.LineHeight = tsty.LineHeight
	ts.WhiteSpace = tsty.WhiteSpace
	ts.Direction = tsty.Direction
	ts.TabSize = tsty.TabSize
	ts.SelectColor = tsty.SelectColor
	ts.HighlightColor = tsty.HighlightColor
}

// LineHeightDots returns the effective line height in dots (actual pixels)
// as FontHeight * LineHeight
func (s *Style) LineHeightDots() float32 {
	return math32.Ceil(s.Font.FontHeight() * s.Text.LineHeight)
}
