// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"image"
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
)

//go:generate core generate

// IMPORTANT: any changes here must be updated in props.go

// note: the go-text shaping framework does not support letter spacing
// or word spacing. These are uncommonly adjusted and not compatible with
// internationalized text in any case.

// todo: bidi override?

// Style is used for text layout styling.
// Most of these are inherited
type Style struct { //types:add

	// Align specifies how to align text along the default direction (inherited).
	// This *only* applies to the text within its containing element,
	// and is relevant only for multi-line text.
	Align Aligns

	// AlignV specifies "vertical" (orthogonal to default direction)
	// alignment of text (inherited).
	// This *only* applies to the text within its containing element:
	// if that element does not have a specified size
	// that is different from the text size, then this has *no effect*.
	AlignV Aligns

	// FontSize is the default font size. The rich text styling specifies
	// sizes relative to this value, with the normal text size factor = 1.
	// In the [styles.Style.Text] context, this is copied from [styles.Font.Size].
	FontSize units.Value

	// LineHeight is a multiplier on the default font size for spacing between lines.
	// If there are larger font elements within a line, they will be accommodated, with
	// the same amount of total spacing added above that maximum size as if it was all
	// the same height. The default of 1.3 represents standard "single spaced" text.
	LineHeight float32 `default:"1.3"`

	// ParaSpacing is the line spacing between paragraphs (inherited).
	// This will be copied from [Style.Margin] if that is non-zero,
	// or can be set directly. Like [LineHeight], this is a multiplier on
	// the default font size.
	ParaSpacing float32 `default:"1.5"`

	// WhiteSpace (not inherited) specifies how white space is processed,
	// and how lines are wrapped.  If set to WhiteSpaceNormal (default) lines are wrapped.
	// See info about interactions with Grow.X setting for this and the NoWrap case.
	WhiteSpace WhiteSpaces

	// Direction specifies the default text direction, which can be overridden if the
	// unicode text is typically written in a different direction.
	Direction rich.Directions

	// Indent specifies how much to indent the first line in a paragraph (inherited).
	Indent units.Value

	// TabSize specifies the tab size, in number of characters (inherited).
	TabSize int

	// Color is the default font fill color, used for inking fonts unless otherwise
	// specified in the [rich.Style].
	Color color.Color

	// SelectColor is the color to use for the background region of selected text (inherited).
	SelectColor image.Image

	// HighlightColor is the color to use for the background region of highlighted text (inherited).
	HighlightColor image.Image

	// CustomFont specifies the Custom font name for rich.Style.Family = Custom.
	CustomFont rich.FontName
}

func NewStyle() *Style {
	s := &Style{}
	s.Defaults()
	return s
}

func (ts *Style) Defaults() {
	ts.Align = Start
	ts.AlignV = Start
	ts.FontSize.Dp(16)
	ts.LineHeight = 1.3
	ts.ParaSpacing = 1.5
	ts.Direction = rich.LTR
	ts.TabSize = 4
	ts.Color = colors.ToUniform(colors.Scheme.OnSurface)
	ts.SelectColor = colors.Scheme.Select.Container
	ts.HighlightColor = colors.Scheme.Warn.Container
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (ts *Style) ToDots(uc *units.Context) {
	ts.FontSize.ToDots(uc)
	ts.FontSize.Dots = math32.Round(ts.FontSize.Dots)
	ts.Indent.ToDots(uc)
}

// InheritFields from parent
func (ts *Style) InheritFields(parent *Style) {
	ts.Align = parent.Align
	ts.AlignV = parent.AlignV
	ts.LineHeight = parent.LineHeight
	ts.ParaSpacing = parent.ParaSpacing
	// ts.WhiteSpace = par.WhiteSpace // todo: we can't inherit this b/c label base default then gets overwritten
	ts.Direction = parent.Direction
	ts.Indent = parent.Indent
	ts.TabSize = parent.TabSize
	ts.SelectColor = parent.SelectColor
	ts.HighlightColor = parent.HighlightColor
}

// FontHeight returns the effective font height based on
// FontSize * [rich.Style] Size multiplier.
func (ts *Style) FontHeight(sty *rich.Style) float32 {
	return math32.Round(ts.FontSize.Dots * sty.Size)
}

// FillColor returns the effective text fill color (main color)
// using any special setting in the given [rich.Style], falling back
// on the Color setting on this text style.
func (ts *Style) FillColor(sty *rich.Style) color.Color {
	fc := sty.FillColor()
	if fc != nil {
		return fc
	}
	return ts.Color
}

// LineHeightDots returns the effective line height in dots (actual pixels)
// as FontHeight * LineHeight
func (ts *Style) LineHeightDots(sty *rich.Style) float32 {
	return math32.Ceil(ts.FontHeight(sty) * ts.LineHeight)
}

// AlignFactors gets basic text alignment factors
func (ts *Style) AlignFactors() (ax, ay float32) {
	ax = ts.Align.Factor()
	val := ts.AlignV
	switch val {
	case Start:
		ay = 0.9 // todo: need to find out actual baseline
	case Center:
		ay = 0.45 // todo: determine if font is horiz or vert..
	case End:
		ay = -0.1 // todo: need actual baseline
	}
	return
}

// Aligns has the different types of alignment and justification for
// the text.
type Aligns int32 //enums:enum -transform kebab

const (
	// Start aligns to the start (top, left) of text region.
	Start Aligns = iota

	// End aligns to the end (bottom, right) of text region.
	End

	// Center aligns to the center of text region.
	Center

	// Justify spreads words to cover the entire text region.
	Justify
)

// Factor returns the alignment factor (0, .5, 1).
func (al Aligns) Factor() float32 {
	switch al {
	case Start:
		return 0
	case Center:
		return 0.5
	case End:
		return 1
	}
	return 0
}

// WhiteSpaces determine how white space is processed and line wrapping
// occurs, either only at whitespace or within words.
type WhiteSpaces int32 //enums:enum -trim-prefix WhiteSpace

const (
	// WrapAsNeeded means that all white space is collapsed to a single
	// space, and text wraps at white space except if there is a long word
	// that cannot fit on the next line, or would otherwise be truncated.
	// To get full word wrapping to expand to all available space, you also
	// need to set GrowWrap = true. Use the SetTextWrap convenience method
	// to set both.
	WrapAsNeeded WhiteSpaces = iota

	// WrapAlways is like [WrapAsNeeded] except that line wrap will always
	// occur within words if it allows more content to fit on a line.
	WrapAlways

	// WrapSpaceOnly means that line wrapping only occurs at white space,
	// and never within words. This means that long words may then exceed
	// the available space and will be truncated. White space is collapsed
	// to a single space.
	WrapSpaceOnly

	// WrapNever means that lines are never wrapped to fit. If there is an
	// explicit line or paragraph break, that will still result in
	// a new line. In general you also don't want simple non-wrapping
	// text labels to Grow (GrowWrap = false). Use the SetTextWrap method
	// to set both. White space is collapsed to a single space.
	WrapNever

	// WhiteSpacePre means that whitespace is preserved, including line
	// breaks. Text will only wrap on explicit line or paragraph breaks.
	// This acts like the <pre> tag in HTML.
	WhiteSpacePre

	// WhiteSpacePreWrap means that whitespace is preserved.
	// Text will wrap when necessary, and on line breaks
	WhiteSpacePreWrap
)

// HasWordWrap returns true if value supports word wrap.
func (ws WhiteSpaces) HasWordWrap() bool {
	switch ws {
	case WrapNever, WhiteSpacePre:
		return false
	default:
		return true
	}
}

// KeepWhiteSpace returns true if value preserves existing whitespace.
func (ws WhiteSpaces) KeepWhiteSpace() bool {
	switch ws {
	case WhiteSpacePre, WhiteSpacePreWrap:
		return true
	default:
		return false
	}
}

// SetUnitContext sets the font-specific information in the given
// units.Context, based on the given styles. Just uses standardized
// fractions of the font size for the other less common units such as ex, ch.
func (ts *Style) SetUnitContext(uc *units.Context) {
	fsz := math32.Round(ts.FontSize.Dots)
	if fsz == 0 {
		fsz = 16
	}
	uc.SetFont(fsz)
}

// TODO(text): ?
// UnicodeBidi determines the type of bidirectional text support.
// See https://pkg.go.dev/golang.org/x/text/unicode/bidi.
// type UnicodeBidi int32 //enums:enum -trim-prefix Bidi -transform kebab
//
// const (
// 	BidiNormal UnicodeBidi = iota
// 	BidiEmbed
// 	BidiBidiOverride
// )
