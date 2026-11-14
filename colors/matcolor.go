// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"

	"cogentcore.org/core/colors/matcolor"
)

// Palette contains the main, global [MatPalette]. It can
// be used by end-user code for accessing tonal palette values,
// although [Scheme] is a more typical way to access the color
// scheme values. It defaults to a palette based around a
// primary color of Google Blue (#4285f4)
var Palette = matcolor.NewPalette(matcolor.KeyFromPrimary(color.RGBA{66, 133, 244, 255}))

// Schemes are the main global Material Design 3 color schemes.
// They should not be used for accessing the current color scheme;
// see [Scheme] for that. Instead, they should be set if you want
// to define your own custom color schemes for your app. The recommended
// way to set the Schemes is through the [SetSchemes] function.
var Schemes = matcolor.NewSchemes(Palette)

// Scheme is the main currently active global Material Design 3
// color scheme. It is the main way that end-user code should
// access the color scheme; ideally, almost all color values should
// be set to something in here. For more specific tones of colors,
// see [Palette]. For setting the color schemes of your app, see
// [Schemes] and [SetSchemes]. For setting this scheme to
// be light or dark, see [SetScheme].
var Scheme = &Schemes.Light

// SetSchemes sets [Schemes], [Scheme], and [Palette] based on the
// given primary color. It is the main way that end-user code should
// set the color schemes to something custom. For more specific control,
// see [SetSchemesFromKey].
func SetSchemes(primary color.RGBA) {
	SetSchemesFromKey(matcolor.KeyFromPrimary(primary))
}

// SetSchemes sets [Schemes], [Scheme], and [Palette] based on the
// given [matcolor.Key]. It should be used instead of [SetSchemes]
// if you want more specific control over the color scheme.
// The schemes are kept if [Palette] is already from the same key
// colors, so that setting them again (as when settings are applied)
// does not change them while they are used in other goroutines.
func SetSchemesFromKey(key *matcolor.Key) {
	if !paletteFromKey(Palette, key) {
		Palette = matcolor.NewPalette(key)
		Schemes = matcolor.NewSchemes(Palette)
	}
	SetScheme(matcolor.SchemeIsDark)
}

// paletteFromKey returns whether the given palette is from the
// given key colors.
func paletteFromKey(p *matcolor.Palette, key *matcolor.Key) bool {
	if p == nil || len(p.Custom) != len(key.Custom) {
		return false
	}
	same := p.Primary.Key == key.Primary && p.Secondary.Key == key.Secondary &&
		p.Tertiary.Key == key.Tertiary && p.Select.Key == key.Select &&
		p.Error.Key == key.Error && p.Success.Key == key.Success &&
		p.Warn.Key == key.Warn && p.Neutral.Key == key.Neutral &&
		p.NeutralVariant.Key == key.NeutralVariant
	for name, c := range key.Custom {
		if t, ok := p.Custom[name]; !ok || t.Key != c {
			return false
		}
	}
	return same
}

// SetScheme sets the value of [Scheme] to either [Schemes.Dark]
// or [Schemes.Light], based on the given value of whether the
// color scheme should be dark. It also sets the value of
// [matcolor.SchemeIsDark]. Values that are already set are not
// written again, as they may be read in other goroutines.
func SetScheme(isDark bool) {
	if matcolor.SchemeIsDark != isDark {
		matcolor.SchemeIsDark = isDark
	}
	sc := &Schemes.Light
	if isDark {
		sc = &Schemes.Dark
	}
	if Scheme != sc {
		Scheme = sc
	}
}
