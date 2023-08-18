// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colormap

import (
	"image/color"
	"math"
	"sort"

	"github.com/goki/colors"
)

// Map maps a value onto a color by interpolating between a list of colors
// defining a spectrum, or optionally as an indexed list of colors.
type Map struct {
	Name string

	// color to display for invalid numbers (e.g., NaN)
	NoColor color.RGBA `desc:"color to display for invalid numbers (e.g., NaN)"`

	// list of colors to interpolate between
	Colors []color.RGBA `desc:"list of colors to interpolate between"`

	// if true, this map should be used as an indexed list instead of interpolating a normalized floating point value: requires caller to check this flag and pass int indexes instead of normalized values to MapIndex
	Indexed bool `desc:"if true, this map should be used as an indexed list instead of interpolating a normalized floating point value: requires caller to check this flag and pass int indexes instead of normalized values to MapIndex"`
}

// Map returns color for normalized value in range 0-1.  NaN returns NoColor
// which can be used to indicate missing values.
func (cm *Map) Map(val float64) color.RGBA {
	nc := len(cm.Colors)
	if nc < 2 {
		return color.RGBA{}
	}
	if math.IsNaN(val) {
		return cm.NoColor
	}
	if val <= 0 {
		return cm.Colors[0]
	} else if val >= 1 {
		return cm.Colors[nc-1]
	}
	ival := val * float64(nc-1)
	lidx := math.Floor(ival)
	uidx := math.Ceil(ival)
	if lidx == uidx {
		return cm.Colors[int(lidx)]
	}
	cmix := ival - lidx
	lclr := cm.Colors[int(lidx)]
	uclr := cm.Colors[int(uidx)]
	return colors.Blend(float32(cmix)*100, lclr, uclr)
}

// MapIndex returns color for given index, for scale in Indexed mode.
// NoColor is returned for values out of range of available colors.
// It is responsibility of the caller to use this method instead of Map
// based on the Indexed flag.
func (cm *Map) MapIndex(val int) color.RGBA {
	nc := len(cm.Colors)
	if val < 0 || val > nc {
		return cm.NoColor
	}
	return cm.Colors[val]
}

// see https://matplotlib.org/tutorials/colors/colormap-manipulation.html
// for how to read out matplotlib scales -- still don't understand segmented ones!

// StdMaps is a list of standard color maps
var StdMaps = map[string]*Map{
	"ColdHot":        {"ColdHot", color.RGBA{200, 200, 200, 255}, []color.RGBA{{0, 255, 255, 255}, {0, 0, 255, 255}, {127, 127, 127, 255}, {255, 0, 0, 255}, {255, 255, 0, 255}}, false},
	"Jet":            {"Jet", color.RGBA{200, 200, 200, 255}, []color.RGBA{{0, 0, 127, 255}, {0, 0, 255, 255}, {0, 127, 255, 255}, {0, 255, 255, 255}, {127, 255, 127, 255}, {255, 255, 0, 255}, {255, 127, 0, 255}, {255, 0, 0, 255}, {127, 0, 0, 255}}, false},
	"JetMuted":       {"JetMuted", color.RGBA{200, 200, 200, 255}, []color.RGBA{{25, 25, 153, 255}, {25, 102, 230, 255}, {0, 230, 230, 255}, {0, 179, 0, 255}, {230, 230, 0, 255}, {230, 102, 25, 255}, {153, 25, 25, 255}}, false},
	"Viridis":        {"Viridis", color.RGBA{200, 200, 200, 255}, []color.RGBA{{72, 33, 114, 255}, {67, 62, 133, 255}, {56, 87, 140, 255}, {45, 111, 142, 255}, {36, 133, 142, 255}, {30, 155, 138, 255}, {42, 176, 127, 255}, {81, 197, 105, 255}, {134, 212, 73, 255}, {194, 223, 35, 255}, {253, 231, 37, 255}}, false},
	"Plasma":         {"Plasma", color.RGBA{200, 200, 200, 255}, []color.RGBA{{61, 4, 155, 255}, {99, 0, 167, 255}, {133, 6, 166, 255}, {166, 32, 152, 255}, {192, 58, 131, 255}, {213, 84, 110, 255}, {231, 111, 90, 255}, {246, 141, 69, 255}, {253, 174, 50, 255}, {252, 210, 36, 255}, {240, 248, 33, 255}}, false},
	"Inferno":        {"Inferno", color.RGBA{200, 200, 200, 255}, []color.RGBA{{37, 12, 3, 255}, {19, 11, 52, 255}, {57, 9, 99, 255}, {95, 19, 110, 255}, {133, 33, 107, 255}, {169, 46, 94, 255}, {203, 65, 73, 255}, {230, 93, 47, 255}, {247, 131, 17, 255}, {252, 174, 19, 255}, {245, 219, 76, 255}, {252, 254, 164, 255}}, false},
	"BlueBlackRed":   {"BlueBlackRed", color.RGBA{200, 200, 200, 255}, []color.RGBA{{0, 0, 255, 255}, {76, 76, 76, 255}, {255, 0, 0, 255}}, false},
	"BlueGreyRed":    {"BlueGreyRed", color.RGBA{200, 200, 200, 255}, []color.RGBA{{0, 0, 255, 255}, {127, 127, 127, 255}, {255, 0, 0, 255}}, false},
	"BlueWhiteRed":   {"BlueWhiteRed", color.RGBA{200, 200, 200, 255}, []color.RGBA{{0, 0, 255, 255}, {230, 230, 230, 255}, {255, 0, 0, 255}}, false},
	"BlueGreenRed":   {"BlueGreenRed", color.RGBA{200, 200, 200, 255}, []color.RGBA{{0, 0, 255, 255}, {0, 230, 0, 255}, {255, 0, 0, 255}}, false},
	"Rainbow":        {"Rainbow", color.RGBA{200, 200, 200, 255}, []color.RGBA{{255, 0, 255, 255}, {0, 0, 255, 255}, {0, 255, 0, 255}, {255, 255, 0, 255}, {255, 0, 0, 255}}, false},
	"ROYGBIV":        {"ROYGBIV", color.RGBA{200, 200, 200, 255}, []color.RGBA{{255, 0, 255, 255}, {0, 0, 127, 255}, {0, 0, 255, 255}, {0, 255, 0, 255}, {255, 255, 0, 255}, {255, 0, 0, 255}}, false},
	"DarkLight":      {"DarkLight", color.RGBA{200, 200, 200, 255}, []color.RGBA{{0, 0, 0, 255}, {250, 250, 250, 255}}, false},
	"DarkLightDark":  {"DarkLightDark", color.RGBA{200, 200, 200, 255}, []color.RGBA{{0, 0, 0, 255}, {250, 250, 250, 255}, {0, 0, 0, 255}}, false},
	"LightDarkLight": {"DarkLightDark", color.RGBA{200, 200, 200, 255}, []color.RGBA{{250, 250, 250, 255}, {0, 0, 0, 255}, {250, 250, 250, 255}}, false},
}

// AvailMaps is the list of all available color maps
var AvailMaps = map[string]*Map{}

func init() {
	for k, v := range StdMaps {
		AvailMaps[k] = v
	}
}

// AvailMapsList returns a sorted list of color map names, e.g., for choosers
func AvailMapsList() []string {
	sl := make([]string, len(AvailMaps))
	ctr := 0
	for k := range AvailMaps {
		sl[ctr] = k
		ctr++
	}
	sort.Strings(sl)
	return sl
}
