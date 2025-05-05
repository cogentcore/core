// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"log/slog"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// IMPORTANT: any changes here must be updated in style_properties.go StyleFontFuncs

// Font contains all font styling information.
// Most of font information is inherited.
type Font struct { //types:add

	// Size of font to render (inherited).
	// Converted to points when getting font to use.
	Size units.Value

	// Family indicates the generic family of typeface to use, where the
	// specific named values to use for each are provided in the [Settings],
	// or [CustomFont] for [Custom].
	Family rich.Family

	// CustomFont specifies the Custom font name for Family = Custom.
	CustomFont rich.FontName

	// Slant allows italic or oblique faces to be selected.
	Slant rich.Slants

	// Weights are the degree of blackness or stroke thickness of a font.
	// This value ranges from 100.0 to 900.0, with 400.0 as normal.
	Weight rich.Weights

	// Stretch is the width of a font as an approximate fraction of the normal width.
	// Widths range from 0.5 to 2.0 inclusive, with 1.0 as the normal width.
	Stretch rich.Stretch

	// Decorations are underline, line-through, etc, as bit flags
	// that must be set using [Decorations.SetFlag].
	Decoration rich.Decorations
}

func (fs *Font) Defaults() {
	fs.Size.Dp(16)
	fs.Weight = rich.Normal
	fs.Stretch = rich.StretchNormal
}

// InheritFields from parent
func (fs *Font) InheritFields(parent *Font) {
	if parent.Size.Value != 0 {
		fs.Size = parent.Size
	}
	fs.Family = parent.Family
	fs.CustomFont = parent.CustomFont
	fs.Slant = parent.Slant
	fs.Weight = parent.Weight
	fs.Stretch = parent.Stretch
	fs.Decoration = parent.Decoration
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (fs *Font) ToDots(uc *units.Context) {
	if fs.Size.Unit == units.UnitEm || fs.Size.Unit == units.UnitEx || fs.Size.Unit == units.UnitCh {
		slog.Error("girl/styles.Font.Size was set to Em, Ex, or Ch; that is recursive and unstable!", "unit", fs.Size.Unit)
		fs.Size.Dp(16)
	}
	fs.Size.ToDots(uc)
}

// SetUnitContext sets the font-specific information in the given
// units.Context, based on the given styles. Just uses standardized
// fractions of the font size for the other less common units such as ex, ch.
func (fs *Font) SetUnitContext(uc *units.Context) {
	fsz := math32.Round(fs.Size.Dots)
	if fsz == 0 {
		fsz = 16
	}
	uc.SetFont(fsz)
}

// SetDecoration sets text decoration (underline, etc),
// which uses bitflags to allow multiple combinations.
func (fs *Font) SetDecoration(deco ...rich.Decorations) *Font {
	for _, d := range deco {
		fs.Decoration.SetFlag(true, d)
	}
	return fs
}

// FontHeight returns the font height in dots (actual pixels).
// Only valid after ToDots has been called, as final step of styling.
func (fs *Font) FontHeight() float32 {
	return math32.Round(fs.Size.Dots)
}

// SetRich sets the rich.Style from font style.
func (fs *Font) SetRich(sty *rich.Style) {
	sty.Family = fs.Family
	sty.Slant = fs.Slant
	sty.Weight = fs.Weight
	sty.Stretch = fs.Stretch
	sty.Decoration = fs.Decoration
}

// SetRichText sets the rich.Style and text.Style properties from the style props.
func (s *Style) SetRichText(sty *rich.Style, tsty *text.Style) {
	s.Font.SetRich(sty)
	s.Text.SetText(tsty)
	tsty.FontSize = s.Font.Size
	tsty.CustomFont = s.Font.CustomFont
	if s.Color != nil {
		clr := colors.ApplyOpacity(colors.ToUniform(s.Color), s.Opacity)
		tsty.Color = clr
	}
	// not doing:
	// if s.Background != nil {
	// 	clr := colors.ApplyOpacity(colors.ToUniform(s.Background), s.Opacity)
	// 	s.Font.SetBackground(clr)
	// }
}

// NewRichText sets the rich.Style and text.Style properties from the style props.
func (s *Style) NewRichText() (sty *rich.Style, tsty *text.Style) {
	sty = rich.NewStyle()
	tsty = text.NewStyle()
	s.SetRichText(sty, tsty)
	return
}
