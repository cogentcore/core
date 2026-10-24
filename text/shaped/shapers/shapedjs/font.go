// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"strconv"
	"sync"
	"syscall/js"

	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// SetFontStyle sets the html canvas font style from [text.Font] and
// [text.Style], with optional lineHeight
func SetFontStyle(ctx js.Value, fnt *text.Font, tsty *text.Style, lineHeight float32) {
	ctx.Set("font", FontSpec(fnt, tsty, lineHeight))
}

// fontSpecKey is the cache key of a CSS font specification.
type fontSpecKey struct {
	styleRune        rune
	size, lineHeight float32
	family           string
}

var (
	// fontSpecs caches the CSS font specifications made by [FontSpec],
	// which are made for every text run that is measured or rendered.
	fontSpecs = map[fontSpecKey]string{}

	// fontSpecsMu protects fontSpecs.
	fontSpecsMu sync.Mutex
)

// FontSpec returns the CSS font specification for the html canvas
// from [text.Font] and [text.Style], with optional lineHeight.
// See https://developer.mozilla.org/en-US/docs/Web/CSS/font
func FontSpec(fnt *text.Font, tsty *text.Style, lineHeight float32) string {
	var fsty rich.Style
	rich.RuneToStyle(&fsty, fnt.StyleRune)
	fam := tsty.FontFamily(&fsty)
	key := fontSpecKey{styleRune: fnt.StyleRune, size: fnt.Size, lineHeight: lineHeight, family: fam}
	fontSpecsMu.Lock()
	defer fontSpecsMu.Unlock()
	if spec, ok := fontSpecs[key]; ok {
		return spec
	}

	// font: font-style font-variant font-weight font-size/line-height font-family
	// note: no fsty.Stretch.String(), here:
	spec := fsty.Slant.String() + " normal " + formatFloat32(fsty.Weight.ToFloat32()) + " " + formatFloat32(fnt.Size) + "px"
	if lineHeight > 0 {
		spec += "/" + formatFloat32(lineHeight) + "px"
	}
	spec += " " + fam
	if len(fontSpecs) >= 1024 {
		clear(fontSpecs)
	}
	fontSpecs[key] = spec
	return spec
}

// formatFloat32 formats v as %g does.
func formatFloat32(v float32) string {
	return strconv.FormatFloat(float64(v), 'g', -1, 32)
}
