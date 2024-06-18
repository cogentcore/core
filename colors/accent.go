// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"

	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/colors/matcolor"
)

// Based on matcolor/accent.go

// ToBase returns the base accent color for the given color
// based on the current scheme (light or dark), which is
// typically used for high emphasis objects or text.
func ToBase(c color.Color) color.RGBA {
	if matcolor.SchemeIsDark {
		return hct.FromColor(c).WithTone(80).AsRGBA()
	}
	return hct.FromColor(c).WithTone(40).AsRGBA()
}

// ToOn returns the accent color for the given color
// that should be placed on top of [ToBase] based on
// the current scheme (light or dark).
func ToOn(c color.Color) color.RGBA {
	if matcolor.SchemeIsDark {
		return hct.FromColor(c).WithTone(20).AsRGBA()
	}
	return hct.FromColor(c).WithTone(100).AsRGBA()
}

// ToContainer returns the container accent color for the given color
// based on the current scheme (light or dark), which is
// typically used for lower emphasis content.
func ToContainer(c color.Color) color.RGBA {
	if matcolor.SchemeIsDark {
		return hct.FromColor(c).WithTone(30).AsRGBA()
	}
	return hct.FromColor(c).WithTone(90).AsRGBA()
}

// ToOnContainer returns the accent color for the given color
// that should be placed on top of [ToContainer] based on
// the current scheme (light or dark).
func ToOnContainer(c color.Color) color.RGBA {
	if matcolor.SchemeIsDark {
		return hct.FromColor(c).WithTone(90).AsRGBA()
	}
	return hct.FromColor(c).WithTone(10).AsRGBA()
}
