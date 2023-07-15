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
var TheColorSchemes = ColorSchemes{
	Light: ColorScheme{
		Font:       colors.Black,
		Background: colors.White,
		Primary:    colors.Blue,
		Secondary:  colors.White.Darker(10),
		Border:     colors.Black.Lighter(20),
		Select:     colors.Lightblue,
		Highlight:  colors.Lightblue,
	},
	Dark: ColorScheme{
		Font:       colors.White,
		Background: colors.Black.Lighter(10),
		Primary:    colors.Lightblue,
		Secondary:  colors.Black.Lighter(10),
		Border:     colors.White.Darker(20),
		Select:     colors.Lightblue,
		Highlight:  colors.Lightblue,
	},
}

// CurrentColorScheme returns the current color scheme
// to be used to style things.
func CurrentColorScheme() ColorScheme {
	if Prefs.ColorSchemeType == ColorSchemeLight {
		return TheColorSchemes.Light
	}
	return TheColorSchemes.Dark
}

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
	Font       gist.Color `desc:"default font / pen color"`
	Background gist.Color `desc:"default background color"`
	Primary    gist.Color `desc:"the primary button color"`
	Secondary  gist.Color `desc:"the secondary button color"`
	Border     gist.Color `desc:"default border color, for button, frame borders, etc"`
	Select     gist.Color `desc:"color for selected elements"`
	Highlight  gist.Color `desc:"color for highlight background"`
}
