// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// IMPORTANT: any changes here must be updated in style_props.go StyleTextFuncs

// Text is used for layout-level (widget, html-style) text styling --
// FontStyle contains all the lower-level text rendering info used in SVG --
// most of these are inherited
type Text struct {

	// prop: text-align (inherited) = how to align text, horizontally. This *only* applies to the text within its containing element, and is typically relevant only for multi-line text: for single-line text, if element does not have a specified size that is different from the text size, then this has *no effect*.
	Align Align `xml:"text-align" inherit:"true" desc:"prop: text-align (inherited) = how to align text, horizontally. This *only* applies to the text within its containing element, and is typically relevant only for multi-line text: for single-line text, if element does not have a specified size that is different from the text size, then this has *no effect*."`

	// prop: text-vertical-align (inherited) = vertical alignment of text. This *only* applies to the text within its containing element -- if that element does not have a specified size that is different from the text size, then this has *no effect*.
	AlignV Align `xml:"text-vertical-align" inherit:"true" desc:"prop: text-vertical-align (inherited) = vertical alignment of text. This *only* applies to the text within its containing element -- if that element does not have a specified size that is different from the text size, then this has *no effect*."`

	// prop: text-anchor (inherited) = for svg rendering only: determines the alignment relative to text position coordinate: for RTL start is right, not left, and start is top for TB
	Anchor TextAnchors `xml:"text-anchor" inherit:"true" desc:"prop: text-anchor (inherited) = for svg rendering only: determines the alignment relative to text position coordinate: for RTL start is right, not left, and start is top for TB"`

	// prop: letter-spacing = spacing between characters and lines
	LetterSpacing units.Value `xml:"letter-spacing" desc:"prop: letter-spacing = spacing between characters and lines"`

	// prop: word-spacing (inherited) = extra space to add between words
	WordSpacing units.Value `xml:"word-spacing" inherit:"true" desc:"prop: word-spacing (inherited) = extra space to add between words"`

	// prop: line-height (inherited) = specified height of a line of text; text is centered within the overall lineheight; the standard way to specify line height is in terms of em
	LineHeight units.Value `xml:"line-height" inherit:"true" desc:"prop: line-height (inherited) = specified height of a line of text; text is centered within the overall lineheight; the standard way to specify line height is in terms of em"`

	// prop: white-space (*not* inherited) = specifies how white space is processed, and how lines are wrapped
	WhiteSpace WhiteSpaces `xml:"white-space" desc:"prop: white-space (*not* inherited) = specifies how white space is processed, and how lines are wrapped"`

	// prop: unicode-bidi (inherited) = determines how to treat unicode bidirectional information
	UnicodeBidi UnicodeBidi `xml:"unicode-bidi" inherit:"true" desc:"prop: unicode-bidi (inherited) = determines how to treat unicode bidirectional information"`

	// prop: direction (inherited) = direction of text -- only applicable for unicode-bidi = bidi-override or embed -- applies to all text elements
	Direction TextDirections `xml:"direction" inherit:"true" desc:"prop: direction (inherited) = direction of text -- only applicable for unicode-bidi = bidi-override or embed -- applies to all text elements"`

	// prop: writing-mode (inherited) = overall writing mode -- only for text elements, not tspan
	WritingMode TextDirections `xml:"writing-mode" inherit:"true" desc:"prop: writing-mode (inherited) = overall writing mode -- only for text elements, not tspan"`

	// prop: glyph-orientation-vertical (inherited) = for TBRL writing mode (only), determines orientation of alphabetic characters -- 90 is default (rotated) -- 0 means keep upright
	OrientationVert float32 `xml:"glyph-orientation-vertical" inherit:"true" desc:"prop: glyph-orientation-vertical (inherited) = for TBRL writing mode (only), determines orientation of alphabetic characters -- 90 is default (rotated) -- 0 means keep upright"`

	// prop: glyph-orientation-horizontal (inherited) = for horizontal LR/RL writing mode (only), determines orientation of all characters -- 0 is default (upright)
	OrientationHoriz float32 `xml:"glyph-orientation-horizontal" inherit:"true" desc:"prop: glyph-orientation-horizontal (inherited) = for horizontal LR/RL writing mode (only), determines orientation of all characters -- 0 is default (upright)"`

	// prop: text-indent (inherited) = how much to indent the first line in a paragraph
	Indent units.Value `xml:"text-indent" inherit:"true" desc:"prop: text-indent (inherited) = how much to indent the first line in a paragraph"`

	// prop: para-spacing (inherited) = extra spacing between paragraphs -- copied from Style.Margin per CSS spec if that is non-zero, else can be set directly with para-spacing
	ParaSpacing units.Value `xml:"para-spacing" inherit:"true" desc:"prop: para-spacing (inherited) = extra spacing between paragraphs -- copied from Style.Margin per CSS spec if that is non-zero, else can be set directly with para-spacing"`

	// prop: tab-size (inherited) = tab size, in number of characters
	TabSize int `xml:"tab-size" inherit:"true" desc:"prop: tab-size (inherited) = tab size, in number of characters"`
}

// LineHeightNormal represents a normal line height,
// equal to the default height of the font being used.
var LineHeightNormal = units.Px(-1)

func (ts *Text) Defaults() {
	ts.LineHeight = LineHeightNormal
	ts.Align = AlignLeft
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

// SetStylePost applies any updates after generic xml-tag property setting
func (ts *Text) SetStylePost(props ki.Props) {
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
	switch {
	case IsAlignMiddle(hal):
		ax = 0.5 // todo: determine if font is horiz or vert..
	case IsAlignEnd(hal):
		ax = 1.0
	}
	val := ts.AlignV
	switch {
	case val == AlignSub:
		ay = -0.15 // todo: fixme -- need actual font metrics
	case val == AlignSuper:
		ay = 0.65 // todo: fixme
	case IsAlignStart(val):
		ay = 0.9 // todo: need to find out actual baseline
	case IsAlignMiddle(val):
		ay = 0.45 // todo: determine if font is horiz or vert..
	case IsAlignEnd(val):
		ay = -0.1 // todo: need actual baseline
	}
	return
}

// https://godoc.org/golang.org/x/text/unicode/bidi
// UnicodeBidi determines how
type UnicodeBidi int32

const (
	BidiNormal UnicodeBidi = iota
	BidiEmbed
	BidiBidiOverride
	UnicodeBidiN
)

var TypeUnicodeBidi = kit.Enums.AddEnumAltLower(UnicodeBidiN, kit.NotBitFlag, StylePropProps, "Bidi")

func (ev UnicodeBidi) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *UnicodeBidi) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// TextDirections are for direction of text writing, used in direction and writing-mode styles
type TextDirections int32

const (
	LRTB TextDirections = iota
	RLTB
	TBRL
	LR
	RL
	TB
	LTR
	RTL
	TextDirectionsN
)

var TypeTextDirections = kit.Enums.AddEnumAltLower(TextDirectionsN, kit.NotBitFlag, StylePropProps, "")

func (ev TextDirections) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *TextDirections) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// TextAnchors are for direction of text writing, used in direction and writing-mode styles
type TextAnchors int32

const (
	AnchorStart TextAnchors = iota
	AnchorMiddle
	AnchorEnd
	TextAnchorsN
)

var TypeTextAnchors = kit.Enums.AddEnumAltLower(TextAnchorsN, kit.NotBitFlag, StylePropProps, "Anchor")

func (ev TextAnchors) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *TextAnchors) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// WhiteSpaces determine how white space is processed
type WhiteSpaces int32

const (
	// WhiteSpaceNormal means that all white space is collapsed to a single
	// space, and text wraps when necessary
	WhiteSpaceNormal WhiteSpaces = iota

	// WhiteSpaceNowrap means that sequences of whitespace will collapse into
	// a single whitespace. Text will never wrap to the next line. The text
	// continues on the same line until a <br> tag is encountered
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

	WhiteSpacesN
)

var TypeWhiteSpaces = kit.Enums.AddEnumAltLower(WhiteSpacesN, kit.NotBitFlag, StylePropProps, "WhiteSpace")

func (ev WhiteSpaces) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *WhiteSpaces) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// HasWordWrap returns true if current white space option supports word wrap
func (ts *Text) HasWordWrap() bool {
	switch ts.WhiteSpace {
	case WhiteSpaceNormal:
		fallthrough
	case WhiteSpacePreLine:
		fallthrough
	case WhiteSpacePreWrap:
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
	case WhiteSpaceNormal:
		fallthrough
	case WhiteSpaceNowrap:
		return false
	default:
		return true
	}
}
