// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"goki.dev/girl/units"
)

// IMPORTANT: any changes here must be updated in style_props.go StyleTextFuncs

// Text is used for layout-level (widget, html-style) text styling --
// FontStyle contains all the lower-level text rendering info used in SVG --
// most of these are inherited
type Text struct { //gti:add

	// prop: text-align (inherited) = how to align text, horizontally.
	// This *only* applies to the text within its containing element,
	// and is typically relevant only for multi-line text:
	// for single-line text, if element does not have a specified size
	// that is different from the text size, then this has *no effect*.
	Align Align `xml:"text-align" inherit:"true"`

	// prop: text-vertical-align (inherited) = vertical alignment of text.
	// This is only applicable for SVG styling, not regular CSS / GoGi,
	// which uses the global Align.Y.  It *only* applies to the text within
	// its containing element: if that element does not have a specified size
	// that is different from the text size, then this has *no effect*.
	AlignV Align `xml:"text-vertical-align" inherit:"true"`

	// prop: text-anchor (inherited) = for svg rendering only:
	// determines the alignment relative to text position coordinate.
	// For RTL start is right, not left, and start is top for TB
	Anchor TextAnchors `xml:"text-anchor" inherit:"true"`

	// prop: letter-spacing = spacing between characters and lines
	LetterSpacing units.Value `xml:"letter-spacing"`

	// prop: word-spacing (inherited) = extra space to add between words
	WordSpacing units.Value `xml:"word-spacing" inherit:"true"`

	// prop: line-height (inherited) = specified height of a line of text; text is centered within the overall lineheight; the standard way to specify line height is in terms of em
	LineHeight units.Value `xml:"line-height" inherit:"true"`

	// prop: white-space (*not* inherited) specifies how white space is processed,
	// and how lines are wrapped.  If set to WhiteSpaceNormal (default) lines are wrapped.
	// See info about interactions with Grow.X setting for this and the NoWrap case.
	WhiteSpace WhiteSpaces `xml:"white-space"`

	// prop: unicode-bidi (inherited) = determines how to treat unicode bidirectional information
	UnicodeBidi UnicodeBidi `xml:"unicode-bidi" inherit:"true"`

	// prop: direction (inherited) = direction of text -- only applicable for unicode-bidi = bidi-override or embed -- applies to all text elements
	Direction TextDirections `xml:"direction" inherit:"true"`

	// prop: writing-mode (inherited) = overall writing mode -- only for text elements, not span
	WritingMode TextDirections `xml:"writing-mode" inherit:"true"`

	// prop: glyph-orientation-vertical (inherited) = for TBRL writing mode (only), determines orientation of alphabetic characters -- 90 is default (rotated) -- 0 means keep upright
	OrientationVert float32 `xml:"glyph-orientation-vertical" inherit:"true"`

	// prop: glyph-orientation-horizontal (inherited) = for horizontal LR/RL writing mode (only), determines orientation of all characters -- 0 is default (upright)
	OrientationHoriz float32 `xml:"glyph-orientation-horizontal" inherit:"true"`

	// prop: text-indent (inherited) = how much to indent the first line in a paragraph
	Indent units.Value `xml:"text-indent" inherit:"true"`

	// prop: para-spacing (inherited) = extra spacing between paragraphs -- copied from Style.Margin per CSS spec if that is non-zero, else can be set directly with para-spacing
	ParaSpacing units.Value `xml:"para-spacing" inherit:"true"`

	// prop: tab-size (inherited) = tab size, in number of characters
	TabSize int `xml:"tab-size" inherit:"true"`
}

// LineHeightNormal represents a normal line height,
// equal to the default height of the font being used.
var LineHeightNormal = units.Dp(-1)

func (ts *Text) Defaults() {
	ts.LineHeight = LineHeightNormal
	ts.Align = AlignStart
	ts.AlignV = AlignBaseline
	ts.Direction = LTR
	ts.OrientationVert = 90
	ts.TabSize = 4
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (ts *Text) ToDots(uc *units.Context) {
	ts.LetterSpacing.ToDots(uc)
	ts.WordSpacing.ToDots(uc)
	ts.LineHeight.ToDots(uc)
	ts.Indent.ToDots(uc)
	ts.ParaSpacing.ToDots(uc)
}

// InheritFields from parent: Manual inheriting of values is much faster than
// automatic version!
func (ts *Text) InheritFields(par *Text) {
	ts.Align = par.Align
	ts.AlignV = par.AlignV
	ts.Anchor = par.Anchor
	ts.WordSpacing = par.WordSpacing
	ts.LineHeight = par.LineHeight
	// ts.WhiteSpace = par.WhiteSpace // todo: we can't inherit this b/c label base default then gets overwritten
	ts.UnicodeBidi = par.UnicodeBidi
	ts.Direction = par.Direction
	ts.WritingMode = par.WritingMode
	ts.OrientationVert = par.OrientationVert
	ts.OrientationHoriz = par.OrientationHoriz
	ts.Indent = par.Indent
	ts.ParaSpacing = par.ParaSpacing
	ts.TabSize = par.TabSize
}

func (ts *Text) SetStylePost(props map[string]any) {

}

// EffLineHeight returns the effective line height for the given
// font height, handling the [LineHeightNormal] special case.
func (ts *Text) EffLineHeight(fontHeight float32) float32 {
	if ts.LineHeight.Val < 0 {
		return fontHeight
	}
	return ts.LineHeight.Dots
}

// AlignFactors gets basic text alignment factors
func (ts *Text) AlignFactors() (ax, ay float32) {
	ax = 0.0
	ay = 0.0
	hal := ts.Align
	switch hal {
	case AlignCenter:
		ax = 0.5 // todo: determine if font is horiz or vert..
	case AlignEnd:
		ax = 1.0
	}
	val := ts.AlignV
	switch val {
	case AlignStart:
		ay = 0.9 // todo: need to find out actual baseline
	case AlignCenter:
		ay = 0.45 // todo: determine if font is horiz or vert..
	case AlignEnd:
		ay = -0.1 // todo: need actual baseline
	}
	return
}

// https://godoc.org/golang.org/x/text/unicode/bidi
// UnicodeBidi determines how
type UnicodeBidi int32 //enums:enum -trim-prefix Bidi

const (
	BidiNormal UnicodeBidi = iota
	BidiEmbed
	BidiBidiOverride
)

// TextDirections are for direction of text writing, used in direction and writing-mode styles
type TextDirections int32 //enums:enum

const (
	LRTB TextDirections = iota
	RLTB
	TBRL
	LR
	RL
	TB
	LTR
	RTL
)

// TextAnchors are for direction of text writing, used in direction and writing-mode styles
type TextAnchors int32 //enums:enum -trim-prefix Anchor

const (
	AnchorStart TextAnchors = iota
	AnchorMiddle
	AnchorEnd
)

// WhiteSpaces determine how white space is processed
type WhiteSpaces int32 //enums:enum -trim-prefix WhiteSpace

const (
	// WhiteSpaceNormal means that all white space is collapsed to a single
	// space, and text wraps when necessary.  To get full word wrapping to
	// expand to all available space, you also need to set Grow.X = 1.
	// Use the SetTextWrap convenience method to set both.
	WhiteSpaceNormal WhiteSpaces = iota

	// WhiteSpaceNowrap means that sequences of whitespace will collapse into
	// a single whitespace. Text will never wrap to the next line except
	// if there is an explicit line break via a <br> tag.  In general you
	// also don't want simple non-wrapping text labels to Grow (Grow.X = 0).
	// Use the SetTextWrap method to set both.
	WhiteSpaceNowrap

	// WhiteSpacePre means that whitespace is preserved by the browser. Text
	// will only wrap on line breaks. Acts like the <pre> tag in HTML.  This
	// invokes a different hand-written parser because the default golang
	// parser automatically throws away whitespace
	WhiteSpacePre

	// WhiteSpacePreLine means that sequences of whitespace will collapse
	// into a single whitespace. Text will wrap when necessary, and on line
	// breaks
	WhiteSpacePreLine

	// WhiteSpacePreWrap means that whitespace is preserved by the
	// browser. Text will wrap when necessary, and on line breaks
	WhiteSpacePreWrap
)

// HasWordWrap returns true if current white space option supports word wrap
func (ts *Text) HasWordWrap() bool {
	switch ts.WhiteSpace {
	case WhiteSpaceNormal, WhiteSpacePreLine, WhiteSpacePreWrap:
		return true
	default:
		return false
	}
}

// HasPre returns true if current white space option preserves existing
// whitespace (or at least requires that parser in case of PreLine, which is
// intermediate)
func (ts *Text) HasPre() bool {
	switch ts.WhiteSpace {
	case WhiteSpaceNormal, WhiteSpaceNowrap:
		return false
	default:
		return true
	}
}
