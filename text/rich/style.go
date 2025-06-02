// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"fmt"
	"image/color"
	"strings"

	"cogentcore.org/core/colors"
	"github.com/go-text/typesetting/di"
)

//go:generate core generate

// IMPORTANT: enums must remain in sync with
// "github.com/go-text/typesetting/font"
// and props.go must be updated as needed.

// Style contains all of the rich text styling properties, that apply to one
// span of text. These are encoded into a uint32 rune value in [rich.Text].
// See [text.Style] and [Settings] for additional context needed for full specification.
type Style struct { //types:add -setters

	// Size is the font size multiplier relative to the standard font size
	// specified in the [text.Style].
	Size float32

	// Family indicates the generic family of typeface to use, where the
	// specific named values to use for each are provided in the [Settings],
	// or [text.Style] for [Custom].
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
	// See [Specials] for usage information: use [Text.StartSpecial]
	// and [Text.EndSpecial] to set.
	Special Specials

	// Decorations are underline, line-through, etc, as bit flags
	// that must be set using [Decorations.SetFlag].
	Decoration Decorations `set:"-"`

	// Direction is the direction to render the text.
	Direction Directions

	// fillColor is the color to use for glyph fill (i.e., the standard "ink" color).
	// Must use SetFillColor to set Decoration fillColor flag.
	// This will be encoded in a uint32 following the style rune, in rich.Text spans.
	fillColor color.Color

	// strokeColor is the color to use for glyph outline stroking.
	// Must use SetStrokeColor to set Decoration strokeColor flag.
	// This will be encoded in a uint32 following the style rune, in rich.Text spans.
	strokeColor color.Color

	// background is the color to use for the background region.
	// Must use SetBackground to set the Decoration background flag.
	// This will be encoded in a uint32 following the style rune, in rich.Text spans.
	background color.Color `set:"-"`

	// URL is the URL for a link element. It is encoded in runes after the style runes.
	URL string
}

func NewStyle() *Style {
	s := &Style{}
	s.Defaults()
	return s
}

// Clone returns a copy of this style.
func (s *Style) Clone() *Style {
	ns := &Style{}
	*ns = *s
	return ns
}

// NewStyleFromRunes returns a new style initialized with data from given runes,
// returning the remaining actual rune string content after style data.
func NewStyleFromRunes(rs []rune) (*Style, []rune) {
	s := NewStyle()
	c := s.FromRunes(rs)
	return s, c
}

func (s *Style) Defaults() {
	s.Size = 1
	s.Weight = Normal
	s.Stretch = StretchNormal
	s.Direction = Default
}

// InheritFields from parent
func (s *Style) InheritFields(parent *Style) {
	// fs.Color = par.Color
	s.Family = parent.Family
	s.Slant = parent.Slant
	if parent.Size != 0 {
		s.Size = parent.Size
	}
	s.Weight = parent.Weight
	s.Stretch = parent.Stretch
}

// FontFamily returns the font family name(s) based on [Style.Family] and the
// values specified in the given [Settings].
func (s *Style) FontFamily(ctx *Settings) string {
	return ctx.Family(s.Family)
}

// Family specifies the generic family of typeface to use, where the
// specific named values to use for each are provided in the Settings.
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

	//	Math fonts are for displaying mathematical expressions, for example
	// superscript and subscript, brackets that cross several lines, nesting
	// expressions, and double-struck glyphs with distinct meanings.
	Math

	// Emoji fonts are specifically designed to render emoji.
	Emoji

	// Fangsong are a particular style of Chinese characters that are between
	// serif-style Song and cursive-style Kai forms. This style is often used
	// for government documents.
	Fangsong

	// Custom is a custom font name that is specified in the [text.Style]
	// CustomFont name.
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
// The corresponding value ranges from 100.0 to 900.0, with 400.0 as normal.
type Weights int32 //enums:enum -transform kebab

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

// ToFloat32 converts the weight to its numerical 100x value
func (w Weights) ToFloat32() float32 {
	return float32((w + 1) * 100)
}

func (w Weights) HTMLTag() string {
	switch w {
	case Bold:
		return "b"
	}
	return ""
}

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

var stretchFloatValues = []float32{0.5, 0.625, 0.75, 0.875, 1, 1.125, 1.25, 1.5, 2.0}

// ToFloat32 converts the stretch to its numerical multiplier value
func (s Stretch) ToFloat32() float32 {
	return stretchFloatValues[s]
}

// note: 11 bits reserved, 8 used

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

	// ParagraphStart indicates that this text is the start of a paragraph,
	// and therefore may be indented according to [text.Style] settings.
	ParagraphStart

	// fillColor means that the fill color of the glyph is set.
	// The standard font rendering uses this fill color (compare to StrokeColor).
	fillColor

	// strokeColor means that the stroke color of the glyph is set.
	// This is normally not rendered: it looks like an outline of the glyph at
	// larger font sizes, and will make smaller font sizes look significantly thicker.
	strokeColor

	// background means that the background region behind the text is colored.
	// The background is not normally colored so it renders over any background.
	background

	// colorFlagsMask is a mask for the color flags, to exclude them as needed.
	colorFlagsMask = 1<<fillColor | 1<<strokeColor | 1<<background
)

// NumColors returns the number of colors used by this decoration setting.
func (d Decorations) NumColors() int {
	nc := 0
	if d.HasFlag(fillColor) {
		nc++
	}
	if d.HasFlag(strokeColor) {
		nc++
	}
	if d.HasFlag(background) {
		nc++
	}
	return nc
}

// Specials are special additional mutually exclusive formatting factors that are not
// otherwise captured by changes in font rendering properties or decorations.
// Each special must be terminated by an End span element, on its own, which
// pops the stack on the last special that was started.
// Use [Text.StartSpecial] and [Text.EndSpecial] to manage the specials,
// avoiding the potential for repeating the start of a given special.
type Specials int32 //enums:enum -transform kebab

const (
	// Nothing special.
	Nothing Specials = iota

	// Super starts super-scripted text.
	Super

	// Sub starts sub-scripted text.
	Sub

	// Link starts a hyperlink, which is in the URL field of the
	// style, and encoded in the runes after the style runes.
	// It also identifies this span for functional interactions
	// such as hovering and clicking. It does not specify the styling,
	// which therefore must be set in addition.
	Link

	// MathInline starts a TeX formatted math sequence, styled for
	// inclusion inline with other text.
	MathInline

	// MathDisplay starts a TeX formatted math sequence, styled as
	// a larger standalone display.
	MathDisplay

	// Quote starts an indented paragraph-level quote.
	Quote

	// todo: could add SmallCaps here?

	// End must be added to terminate the last Special started: use [Text.AddEnd].
	// The renderer maintains a stack of special elements.
	End
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

	// Default uses the [text.Style] default direction.
	Default
)

// ToGoText returns the go-text version of direction.
func (d Directions) ToGoText() di.Direction {
	return di.Direction(d)
}

// IsVertical returns true if given text is vertical.
func (d Directions) IsVertical() bool {
	return d >= TTB && d <= BTT
}

// SetDecoration sets given decoration flag(s) on.
func (s *Style) SetDecoration(deco ...Decorations) *Style {
	for _, d := range deco {
		s.Decoration.SetFlag(true, d)
	}
	return s
}

func (s *Style) FillColor() color.Color {
	if s.Decoration.HasFlag(fillColor) {
		return s.fillColor
	}
	return nil
}

func (s *Style) StrokeColor() color.Color {
	if s.Decoration.HasFlag(strokeColor) {
		return s.strokeColor
	}
	return nil
}

func (s *Style) Background() color.Color {
	if s.Decoration.HasFlag(background) {
		return s.background
	}
	return nil
}

// SetFillColor sets the fill color to given color, setting the Decoration
// flag and the color value.
func (s *Style) SetFillColor(clr color.Color) *Style {
	s.fillColor = clr
	s.Decoration.SetFlag(clr != nil, fillColor)
	return s
}

// SetStrokeColor sets the stroke color to given color, setting the Decoration
// flag and the color value.
// This is normally not set: it looks like an outline of the glyph at
// larger font sizes, and will make smaller font sizes look significantly thicker.
func (s *Style) SetStrokeColor(clr color.Color) *Style {
	s.strokeColor = clr
	s.Decoration.SetFlag(clr != nil, strokeColor)
	return s
}

// SetBackground sets the background color to given color, setting the Decoration
// flag and the color value.
// The background is not normally colored so it renders over any background.
func (s *Style) SetBackground(clr color.Color) *Style {
	s.background = clr
	s.Decoration.SetFlag(clr != nil, background)
	return s
}

// SetLinkStyle sets the default hyperlink styling: primary.Base color (e.g., blue)
// and Underline.
func (s *Style) SetLinkStyle() *Style {
	s.SetFillColor(colors.ToUniform(colors.Scheme.Primary.Base))
	s.Decoration.SetFlag(true, Underline)
	return s
}

// SetLink sets the given style as a hyperlink, with given URL, and
// default link styling.
func (s *Style) SetLink(url string) *Style {
	s.URL = url
	s.Special = Link
	return s.SetLinkStyle()
}

// IsMath returns true if is a Special MathInline or MathDisplay.
func (s *Style) IsMath() bool {
	return s.Special == MathInline || s.Special == MathDisplay
}

func (s *Style) String() string {
	str := ""
	if s.Special == End {
		return "{End Special}"
	}
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
		if s.Special == Link {
			str += "[" + s.URL + "] "
		}
	}
	for d := Underline; d <= ParagraphStart; d++ {
		if s.Decoration.HasFlag(d) {
			str += d.BitIndexString() + " "
		}
	}
	if s.Decoration.HasFlag(fillColor) {
		str += "fill-color "
	}
	if s.Decoration.HasFlag(strokeColor) {
		str += "stroke-color "
	}
	if s.Decoration.HasFlag(background) {
		str += "background "
	}
	return strings.TrimSpace(str)
}
