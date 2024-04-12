// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colormap

import (
	"image/color"
	"maps"
	"sort"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
)

// Map maps a value onto a color by interpolating between a list of colors
// defining a spectrum, or optionally as an indexed list of colors.
type Map struct {
	// Name is the name of the color map
	Name string

	// if true, this map should be used as an indexed list instead of interpolating a normalized floating point value: requires caller to check this flag and pass int indexes instead of normalized values to MapIndex
	Indexed bool

	// the colorspace algorithm to use for blending colors
	Blend colors.BlendTypes

	// color to display for invalid numbers (e.g., NaN)
	NoColor color.RGBA

	// list of colors to interpolate between
	Colors []color.RGBA
}

func (cm *Map) String() string {
	return cm.Name
}

// Map returns color for normalized value in range 0-1.  NaN returns NoColor
// which can be used to indicate missing values.
func (cm *Map) Map(val float32) color.RGBA {
	nc := len(cm.Colors)
	if nc == 0 {
		return color.RGBA{}
	}
	if nc == 1 {
		return cm.Colors[0]
	}
	if math32.IsNaN(val) {
		return cm.NoColor
	}
	if val <= 0 {
		return cm.Colors[0]
	} else if val >= 1 {
		return cm.Colors[nc-1]
	}
	ival := val * float32(nc-1)
	lidx := math32.Floor(ival)
	uidx := math32.Ceil(ival)
	if lidx == uidx {
		return cm.Colors[int(lidx)]
	}
	cmix := 100 * (1 - (ival - lidx))
	lclr := cm.Colors[int(lidx)]
	uclr := cm.Colors[int(uidx)]
	return colors.Blend(cm.Blend, cmix, lclr, uclr)
}

// MapIndex returns color for given index, for scale in Indexed mode.
// NoColor is returned for values out of range of available colors.
// It is responsibility of the caller to use this method instead of Map
// based on the Indexed flag.
func (cm *Map) MapIndex(val int) color.RGBA {
	nc := len(cm.Colors)
	if val < 0 || val >= nc {
		return cm.NoColor
	}
	return cm.Colors[val]
}

// see https://matplotlib.org/tutorials/colors/colormap-manipulation.html
// for how to read out matplotlib scales -- still don't understand segmented ones!

// StandardMaps is a list of standard color maps
var StandardMaps = map[string]*Map{
	"ColdHot": {
		Name:    "ColdHot",
		NoColor: colors.FromRGB(200, 200, 200),
		Colors: []color.RGBA{
			{0, 255, 255, 255},
			{0, 0, 255, 255},
			{127, 127, 127, 255},
			{255, 0, 0, 255},
			{255, 255, 0, 255},
		},
	},
	"Jet": {
		Name:    "Jet",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{0, 0, 127, 255},
			{0, 0, 255, 255},
			{0, 127, 255, 255},
			{0, 255, 255, 255},
			{127, 255, 127, 255},
			{255, 255, 0, 255},
			{255, 127, 0, 255},
			{255, 0, 0, 255},
			{127, 0, 0, 255},
		},
	},
	"JetMuted": {
		Name:    "JetMuted",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{25, 25, 153, 255},
			{25, 102, 230, 255},
			{0, 230, 230, 255},
			{0, 179, 0, 255},
			{230, 230, 0, 255},
			{230, 102, 25, 255},
			{153, 25, 25, 255},
		},
	},
	"Viridis": {
		Name:    "Viridis",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{72, 33, 114, 255},
			{67, 62, 133, 255},
			{56, 87, 140, 255},
			{45, 111, 142, 255},
			{36, 133, 142, 255},
			{30, 155, 138, 255},
			{42, 176, 127, 255},
			{81, 197, 105, 255},
			{134, 212, 73, 255},
			{194, 223, 35, 255},
			{253, 231, 37, 255},
		},
	},
	"Plasma": {
		Name:    "Plasma",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{61, 4, 155, 255},
			{99, 0, 167, 255},
			{133, 6, 166, 255},
			{166, 32, 152, 255},
			{192, 58, 131, 255},
			{213, 84, 110, 255},
			{231, 111, 90, 255},
			{246, 141, 69, 255},
			{253, 174, 50, 255},
			{252, 210, 36, 255},
			{240, 248, 33, 255},
		},
	},
	"Inferno": {
		Name:    "Inferno",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{37, 12, 3, 255},
			{19, 11, 52, 255},
			{57, 9, 99, 255},
			{95, 19, 110, 255},
			{133, 33, 107, 255},
			{169, 46, 94, 255},
			{203, 65, 73, 255},
			{230, 93, 47, 255},
			{247, 131, 17, 255},
			{252, 174, 19, 255},
			{245, 219, 76, 255},
			{252, 254, 164, 255},
		},
	},
	"BlueRed": {
		Name:    "BlueRed",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{0, 0, 255, 255},
			{255, 0, 0, 255},
		},
	},
	"BlueBlackRed": {
		Name:    "BlueBlackRed",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{0, 0, 255, 255},
			{76, 76, 76, 255},
			{255, 0, 0, 255},
		},
	},
	"BlueGreyRed": {
		Name:    "BlueGreyRed",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{0, 0, 255, 255},
			{127, 127, 127, 255},
			{255, 0, 0, 255},
		},
	},
	"BlueWhiteRed": {
		Name:    "BlueWhiteRed",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{0, 0, 255, 255},
			{230, 230, 230, 255},
			{255, 0, 0, 255},
		},
	},
	"BlueGreenRed": {
		Name:    "BlueGreenRed",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{0, 0, 255, 255},
			{0, 230, 0, 255},
			{255, 0, 0, 255},
		},
	},
	"Rainbow": {
		Name:    "Rainbow",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{255, 0, 255, 255},
			{0, 0, 255, 255},
			{0, 255, 0, 255},
			{255, 255, 0, 255},
			{255, 0, 0, 255},
		},
	},
	"ROYGBIV": {
		Name:    "ROYGBIV",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{255, 0, 255, 255},
			{0, 0, 127, 255},
			{0, 0, 255, 255},
			{0, 255, 0, 255},
			{255, 255, 0, 255},
			{255, 0, 0, 255},
		},
	},
	"DarkLight": {
		Name:    "DarkLight",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{0, 0, 0, 255},
			{250, 250, 250, 255},
		},
	},
	"DarkLightDark": {
		Name:    "DarkLightDark",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{0, 0, 0, 255},
			{250, 250, 250, 255},
			{0, 0, 0, 255},
		},
	},
	"LightDarkLight": {
		Name:    "DarkLightDark",
		NoColor: color.RGBA{200, 200, 200, 255},
		Colors: []color.RGBA{
			{250, 250, 250, 255},
			{0, 0, 0, 255},
			{250, 250, 250, 255},
		},
	},
}

// AvailableMaps is the list of all available color maps
var AvailableMaps = map[string]*Map{}

func init() {
	maps.Copy(AvailableMaps, StandardMaps)
}

// AvailableMapsList returns a sorted list of color map names, e.g., for choosers
func AvailableMapsList() []string {
	sl := make([]string, len(AvailableMaps))
	ctr := 0
	for k := range AvailableMaps {
		sl[ctr] = k
		ctr++
	}
	sort.Strings(sl)
	return sl
}
