// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"cogentcore.org/core/text/rich"
)

// Font is a compact encoding of font properties, which can be used
// to reconstruct the corresponding [rich.Style] from [text.Style].
type Font struct {
	// StyleRune is the rune-compressed version of the [rich.Style] parameters.
	StyleRune rune

	// Size is the Text.Style.FontSize.Dots value of the font size,
	// multiplied by font rich.Style.Size.
	Size float32

	// Family is a nonstandard family name: if standard, then empty,
	// and value is determined by [rich.DefaultSettings] and Style.Family.
	Family string
}

func NewFont(fsty *rich.Style, tsty *Style) *Font {
	fn := &Font{StyleRune: rich.RuneFromStyle(fsty), Size: tsty.FontHeight(fsty)}
	if fsty.Family == rich.Custom {
		fn.Family = string(tsty.CustomFont)
	}
	return fn
}

// Style returns the [rich.Style] version of this Font.
func (fn *Font) Style(tsty *Style) *rich.Style {
	sty := rich.NewStyle()
	rich.RuneToStyle(sty, fn.StyleRune)
	sty.Size = fn.Size / tsty.FontSize.Dots
	return sty
}

// FontFamily returns the string value of the font Family for given [rich.Style],
// using [text.Style] CustomFont or [rich.DefaultSettings] values.
func (ts *Style) FontFamily(sty *rich.Style) string {
	if sty.Family == rich.Custom {
		return string(ts.CustomFont)
	}
	return sty.FontFamily(&rich.DefaultSettings)
}

func (fn *Font) FamilyString(tsty *Style) string {
	if fn.Family != "" {
		return fn.Family
	}
	return tsty.FontFamily(fn.Style(tsty))
}
