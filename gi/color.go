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
		Primary:    gist.ColorFromHSL(217, 0.9, 0.43),
		Secondary:  colors.White.Darker(10),
		Accent:     gist.ColorFromHSL(217, 0.92, 0.86),
		Success:    colors.Green,
		Failure:    colors.Red,
	},
	Dark: ColorScheme{
		Text:       colors.White,
		Background: colors.Black.Lighter(10),
		Primary:    gist.ColorFromHSL(217, 0.89, 0.76),
		Secondary:  colors.Black.Lighter(10),
		Accent:     gist.ColorFromHSL(217, 0.89, 0.35),
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

var TypeColorSchemeTypes = kit.Enums.AddEnumAltLower(ColorSchemesN, kit.NotBitFlag, gist.StylePropProps, "ColorScheme")

//go:generate stringer -type=ColorSchemeTypes

// ColorScheme contains the colors for
// one color scheme (ex: light or dark).
type ColorScheme struct {
	Primary            gist.Color `desc:"Primary is the base primary color applied to important elements"`
	OnPrimary          gist.Color `desc:"OnPrimary is the color applied to content on top of Primary. It defaults to the contrast color of Primary."`
	PrimaryContainer   gist.Color `desc:"PrimaryContainer is the color applied to elements with less emphasis than Primary"`
	OnPrimaryContainer gist.Color `desc:"OnPrimaryContainer is the color applied to content on top of PrimaryContainer. It defaults to the contrast color of PrimaryContainer."`

	Secondary            gist.Color `desc:"Secondary is the base secondary color applied to less important elements"`
	OnSecondary          gist.Color `desc:"OnSecondary is the color applied to content on top of Secondary. It defaults to the contrast color of Secondary."`
	SecondaryContainer   gist.Color `desc:"SecondaryContainer is the color applied to elements with less emphasis than Secondary"`
	OnSecondaryContainer gist.Color `desc:"OnSecondaryContainer is the color applied to content on top of SecondaryContainer. It defaults to the contrast color of SecondaryContainer."`

	Tertiary            gist.Color `desc:"Tertiary is the base tertiary color applied as an accent to highlight elements and screate contrast between other colors"`
	OnTertiary          gist.Color `desc:"OnTertiary is the color applied to content on top of Tertiary. It defaults to the contrast color of Tertiary."`
	TertiaryContainer   gist.Color `desc:"TertiaryContainer is the color applied to elements with less emphasis than Tertiary"`
	OnTertiaryContainer gist.Color `desc:"OnTertiaryContainer is the color applied to content on top of TertiaryContainer. It defaults to the contrast color of TertiaryContainer."`

	Error            gist.Color `desc:"Error is the base error color applied to elements that indicate an error or danger"`
	OnError          gist.Color `desc:"OnError is the color applied to content on top of Error. It defaults to the contrast color of Error."`
	ErrorContainer   gist.Color `desc:"ErrorContainer is the color applied to elements with less emphasis than Error"`
	OnErrorContainer gist.Color `desc:"OnErrorContainer is the color applied to content on top of ErrorContainer. It defaults to the contrast color of ErrorContainer."`
}
