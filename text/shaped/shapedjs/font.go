// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"fmt"
	"strings"
	"syscall/js"

	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
)

// todo: equivalent of *Font with just the font family, etc params. per Describe
// for more efficiency it would be good to just use Family enum, with optional string
// that would typically be empty.

// Font is a compact encoding of font properties, suitable as a
// map key in the glyphcache.
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

func NewFont(fsty *rich.Style, tsty *text.Style, ctx *rich.Settings) *Font {
	fn := &Font{StyleRune: rich.RuneFromStyle(fsty), Size: tsty.FontHeight(fsty)}
	if fsty.Family == rich.Custom {
		fn.Family = string(tsty.CustomFont)
	}
	return fn
}

// Style returns the [rich.Style] version of this Font.
func (fn *Font) Style(tsty *text.Style) *rich.Style {
	sty := rich.NewStyle()
	rich.RuneToStyle(sty, fn.StyleRune)
	sty.Size = fn.Size / tsty.FontSize.Dots
	return sty
}

// SetFontStyle sets the html canvas font style from [Font] and
// [text.Style], with optional lineHeight
func SetFontStyle(ctx js.Value, fn *Font, tsty *text.Style, lineHeight float32) {
	// See https://developer.mozilla.org/en-US/docs/Web/CSS/font
	fsty := fn.Style(tsty)
	fsz := ""
	if lineHeight > 0 {
		fsz = fmt.Sprintf("%gpx/%gpx", fn.Size, lineHeight)
	} else {
		fsz = fmt.Sprintf("%gpx", fn.Size)
	}
	fam := tsty.FontFamily(fsty)
	// note: no fsty.Stretch.String(), here:
	// font: font-style font-variant font-weight font-size/line-height font-family
	parts := []string{fsty.Slant.String(), "normal", fmt.Sprintf("%g", fsty.Weight.ToFloat32()), fsz, fam}
	fspec := strings.Join(parts, " ")
	fmt.Println("fspec:", fspec)
	ctx.Set("font", fspec)
}

// FontList returns the list of fonts that have been loaded.
func (sh *Shaper) FontList() []shaped.FontInfo {
	return nil
}
