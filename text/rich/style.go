// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/go-text/typesetting/di"
)

//go:generate core generate -add-types -setters

// IMPORTANT: enums must remain in sync with
// "github.com/go-text/typesetting/font"
// and props.go must be updated as needed.

// Style contains all of the rich text styling properties, that apply to one
// span of text. These are encoded into a uint32 rune value in [rich.Text].
// See [Context] for additional context needed for full specification.
type Style struct { //types:add

	// Size is the font size multiplier relative to the standard font size
	// specified in the Context.
	Size float32

	// Family indicates the generic family of typeface to use, where the
	// specific named values to use for each are provided in the Context.
	Family Family

	// Slant allows italic or oblique faces to be selected.
	Slant Slants

	// Weights are the degree of blackness or stroke thickness of a font.
	// This value ranges from 100.0 to 900.0, with 400.0 as normal.
	Weight Weights

	// Stretch is the width of a font as an approximate fraction of the normal width.
	// Widths range from 0.5 to 2.0 inclusive, with 1.0 as the normal width.
	Stretch Stretch

	// Special additional formatting factors that are not otherwise
	// captured by changes in font rendering properties or decorations.
	Special Specials

	// Decorations are underline, line-through, etc, as bit flags
	// that must be set using [Decorations.SetFlag].
	Decoration Decorations

	// Direction is the direction to render the text.
	Direction Directions

	//	FillColor is the color to use for glyph fill (i.e., the standard "ink" color)
	// if the Decoration FillColor flag is set. This will be encoded in a uint32 following
	// the style rune, in rich.Text spans.
	FillColor color.Color `set:"-"`

	//	StrokeColor is the color to use for glyph stroking if the Decoration StrokeColor
	// flag is set. This will be encoded in a uint32 following the style rune,
	// in rich.Text spans.
	StrokeColor color.Color `set:"-"`

	//	Background is the color to use for the background region if the Decoration
	// Background flag is set. This will be encoded in a uint32 following the style rune,
	// in rich.Text spans.
	Background color.Color `set:"-"`

	// URL is the URL for a link element. It is encoded in runes after the style runes.
	URL string
}

func NewStyle() *Style {
	s := &Style{}
	s.Defaults()
	return s
}

func (s *Style) Defaults() {
	s.Size = 1
	s.Weight = Normal
	s.Stretch = StretchNormal
}

// FontFamily returns the font family name(s) based on [Style.Family] and the
// values specified in the given [Context].
func (s *Style) FontFamily(ctx *Context) string {
	return ctx.Family(s.Family)
}

// FontSize returns the font size in dot pixels based on [Style.Size] and the
// Standard size specified in the given [Context].
func (s *Style) FontSize(ctx *Context) float32 {
	return ctx.SizeDots(s.Size)
}

// Color returns the FillColor for inking the font based on
// [Style.Size] and the default color in [Context]
func (s *Style) Color(ctx *Context) color.Color {
	if s.Decoration.HasFlag(FillColor) {
		return s.FillColor
	}
	return ctx.Color
}

// Family specifies the generic family of typeface to use, where the
// specific named values to use for each are provided in the Context.
type Family int32 //enums:enum -trim-prefix Family -transform kebab

const (
	// SansSerif is a font without serifs, where glyphs have plain stroke endings,
	// without ornamentation. Example sans-serif fonts include Arial, Helvetica,
	// Open Sans, Fira Sans, Lucida Sans, Lucida Sans Unicode, Trebuchet MS,
	// Liberation Sans, and Nimbus Sans L.
	SansSerif Family = iota

	// Serif is a small line or stroke attached to the end of a larger stroke
	// in a letter. In serif fonts, glyphs have finishing strokes, flared or
	// tapering ends. Examples include Times New Roman, Lucida Bright,
	// Lucida Fax, Palatino, Palatino Linotype, Palladio, and URW Palladio.
	Serif

	// Monospace fonts have all glyphs with he same fixed width.
	// Example monospace fonts include Fira Mono, DejaVu Sans Mono,
	// Menlo, Consolas, Liberation Mono, Monaco, and Lucida Console.
	Monospace

	// Cursive glyphs generally have either joining strokes or other cursive
	// characteristics beyond those of italic typefaces. The glyphs are partially
	// or completely connected, and the result looks more like handwritten pen or
	// brush writing than printed letter work. Example cursive fonts include
	// Brush Script MT, Brush Script Std, Lucida Calligraphy, Lucida Handwriting,
	// and Apple Chancery.
	Cursive

	// Fantasy fonts are primarily decorative fonts that contain playful
	// representations of characters. Example fantasy fonts include Papyrus,
	// Herculanum, Party LET, Curlz MT, and Harrington.
	Fantasy

	//	Maths fonts are for displaying mathematical expressions, for example
	// superscript and subscript, brackets that cross several lines, nesting
	// expressions, and double-struck glyphs with distinct meanings.
	Maths

	// Emoji fonts are specifically designed to render emoji.
	Emoji

	// Fangsong are a particular style of Chinese characters that are between
	// serif-style Song and cursive-style Kai forms. This style is often used
	// for government documents.
	Fangsong

	// Custom is a custom font name that can be set in Context.
	Custom
)

// Slants (also called style) allows italic or oblique faces to be selected.
type Slants int32 //enums:enum -trim-prefix Slant -transform kebab

const (

	// A face that is neither italic not obliqued.
	SlantNormal Slants = iota

	// A form that is generally cursive in nature or slanted.
	// This groups what is usually called Italic or Oblique.
	Italic
)

// Weights are the degree of blackness or stroke thickness of a font.
// This value ranges from 100.0 to 900.0, with 400.0 as normal.
type Weights int32 //enums:enum Weight -transform kebab

const (
	// Thin weight (100), the thinnest value.
	Thin Weights = iota

	// Extra light weight (200).
	ExtraLight

	// Light weight (300).
	Light

	// Normal (400).
	Normal

	// Medium weight (500, higher than normal).
	Medium

	// Semibold weight (600).
	Semibold

	// Bold weight (700).
	Bold

	// Extra-bold weight (800).
	ExtraBold

	// Black weight (900), the thickest value.
	Black
)

// Stretch is the width of a font as an approximate fraction of the normal width.
// Widths range from 0.5 to 2.0 inclusive, with 1.0 as the normal width.
type Stretch int32 //enums:enum -trim-prefix Stretch -transform kebab

const (

	// Ultra-condensed width (50%), the narrowest possible.
	UltraCondensed Stretch = iota

	// Extra-condensed width (62.5%).
	ExtraCondensed

	// Condensed width (75%).
	Condensed

	// Semi-condensed width (87.5%).
	SemiCondensed

	// Normal width (100%).
	StretchNormal

	// Semi-expanded width (112.5%).
	SemiExpanded

	// Expanded width (125%).
	Expanded

	// Extra-expanded width (150%).
	ExtraExpanded

	// Ultra-expanded width (200%), the widest possible.
	UltraExpanded
)

// Decorations are underline, line-through, etc, as bit flags
// that must be set using [Font.SetDecoration].
type Decorations int64 //enums:bitflag -transform kebab

const (
	// Underline indicates to place a line below text.
	Underline Decorations = iota

	// Overline indicates to place a line above text.
	Overline

	// LineThrough indicates to place a line through text.
	LineThrough

	// DottedUnderline is used for abbr tag.
	DottedUnderline

	// Link indicates a hyperlink, which is in the URL field of the
	// style, and encoded in the runes after the style runes.
	// It also identifies this span for functional interactions
	// such as hovering and clicking. It does not specify the styling.
	Link

	// FillColor means that the fill color of the glyph is set to FillColor,
	// which encoded in the rune following the style rune, rather than the default.
	// The standard font rendering uses this fill color (compare to StrokeColor).
	FillColor

	// StrokeColor means that the stroke color of the glyph is set to StrokeColor,
	// which is encoded in the rune following the style rune. This is normally not rendered:
	// it looks like an outline of the glyph at larger font sizes, it will
	// make smaller font sizes look significantly thicker.
	StrokeColor

	// Background means that the background region behind the text is colored to
	// Background, which is encoded in the rune following the style rune.
	// The background is not normally colored.
	Background
)

// NumColors returns the number of colors used by this decoration setting.
func (d Decorations) NumColors() int {
	nc := 0
	if d.HasFlag(FillColor) {
		nc++
	}
	if d.HasFlag(StrokeColor) {
		nc++
	}
	if d.HasFlag(Background) {
		nc++
	}
	return nc
}

// Specials are special additional formatting factors that are not
// otherwise captured by changes in font rendering properties or decorations.
type Specials int32 //enums:enum -transform kebab

const (
	// Nothing special.
	Nothing Specials = iota

	// Super indicates super-scripted text.
	Super

	// Sub indicates sub-scripted text.
	Sub

	// Math indicates a LaTeX formatted math sequence.
	Math
)

// Directions specifies the text layout direction.
type Directions int32 //enums:enum -transform kebab

const (
	// LTR is Left-to-Right text.
	LTR Directions = iota

	// RTL is Right-to-Left text.
	RTL

	// TTB is Top-to-Bottom text.
	TTB

	// BTT is Bottom-to-Top text.
	BTT
)

// ToGoText returns the go-text version of direction.
func (d Directions) ToGoText() di.Direction {
	return di.Direction(d)
}

// SetFillColor sets the fill color to given color, setting the Decoration
// flag and the color value.
func (s *Style) SetFillColor(clr color.Color) *Style {
	s.FillColor = clr
	s.Decoration.SetFlag(true, FillColor)
	return s
}

// SetStrokeColor sets the stroke color to given color, setting the Decoration
// flag and the color value.
func (s *Style) SetStrokeColor(clr color.Color) *Style {
	s.StrokeColor = clr
	s.Decoration.SetFlag(true, StrokeColor)
	return s
}

// SetBackground sets the background color to given color, setting the Decoration
// flag and the color value.
func (s *Style) SetBackground(clr color.Color) *Style {
	s.Background = clr
	s.Decoration.SetFlag(true, Background)
	return s
}

// SetLink sets this span style as a Link, setting the Decoration
// flag for Link and the URL field to given link.
func (s *Style) SetLink(url string) *Style {
	s.URL = url
	s.Decoration.SetFlag(true, Link)
	return s
}

func (s *Style) String() string {
	str := ""
	if s.Size != 1 {
		str += fmt.Sprintf("%5.2fx ", s.Size)
	}
	if s.Family != SansSerif {
		str += s.Family.String() + " "
	}
	if s.Slant != SlantNormal {
		str += s.Slant.String() + " "
	}
	if s.Weight != Normal {
		str += s.Weight.String() + " "
	}
	if s.Stretch != StretchNormal {
		str += s.Stretch.String() + " "
	}
	if s.Special != Nothing {
		str += s.Special.String() + " "
	}
	for d := Underline; d <= Background; d++ {
		if s.Decoration.HasFlag(d) {
			str += d.BitIndexString() + " "
		}
	}
	return strings.TrimSpace(str)
}
