// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/goki/gi/gist"
	"github.com/goki/gi/gist/colors"
	"github.com/goki/ki/kit"
)

// ColorSchemes contains the color schemes for an app.
// It contains a light and a dark color scheme.
type ColorSchemes struct {
	Light ColorScheme
	Dark  ColorScheme
}

// TheColorSchemes are the color schemes used to style
// the app. You should set them in your mainrun funtion
// if you want to change the color schemes.
// [Colors] is set based on TheColorSchemes and
// the user preferences.
var TheColorSchemes = ColorSchemes{
	Light: ColorScheme{
		Text:       colors.Black,
		Background: colors.White,
		Primary:    gist.ColorFromColor(gist.HSLA{H: 217, S: 0.9, L: 0.43, A: 1}),
		Secondary:  colors.White.Darker(10),
		Accent:     gist.ColorFromColor(gist.HSLA{H: 217, S: 0.92, L: 0.86, A: 1}),
		Success:    colors.Green,
		Failure:    colors.Red,
	},
	Dark: ColorScheme{
		Text:       colors.White,
		Background: colors.Black.Lighter(10),
		Primary:    gist.ColorFromColor(gist.HSLA{H: 217, S: 0.89, L: 0.76, A: 1}),
		Secondary:  colors.Black.Lighter(10),
		Accent:     gist.ColorFromColor(gist.HSLA{H: 217, S: 0.89, L: 0.35, A: 1}),
		Success:    colors.Green,
		Failure:    colors.Red,
	},
}

// Colors is the current color scheme
// that is used to style the app. It should
// not be set by end-user code, as it is set
// automatically from the user's preferences and
// [TheColorSchemes]. You should set [TheColorSchemes]
// to customize the color scheme of your app.
var Colors ColorScheme

// ColorSchemeTypes is an enum that contains
// the supported color scheme types
type ColorSchemeTypes int

const (
	// ColorSchemeLight is a light color scheme
	ColorSchemeLight ColorSchemeTypes = iota
	// ColorSchemeDark is a dark color scheme
	ColorSchemeDark

	ColorSchemesN
)

var KiT_ColorSchemeTypes = kit.Enums.AddEnumAltLower(ColorSchemesN, kit.NotBitFlag, gist.StylePropProps, "ColorScheme")

//go:generate stringer -type=ColorSchemeTypes

// ColorScheme contains the colors for
// one color scheme (ex: light or dark).
type ColorScheme struct {
	Text       gist.Color `desc:"default text color"`
	Background gist.Color `desc:"default background color"`
	Primary    gist.Color `desc:"the primary button color"`
	Secondary  gist.Color `desc:"the secondary button color"`
	Accent     gist.Color `desc:"the accent color, typically used for selected elements"`
	Success    gist.Color `desc:"a semantic color that indicates success"`
	Failure    gist.Color `desc:"a semantic color that indicates failure or danger"`
}
