// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/goki/gi/gist"
	"github.com/goki/gi/gist/colors"
)

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
