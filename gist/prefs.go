// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"image/color"
	"log"
	"strings"

	"goki.dev/colors"
)

// Prefer defines the interface to preferences for style-relevant prefs
type Prefer interface {
	// PrefColor returns preference color of given name
	// std names are: font, background, shadow, border, control, icon, select, highlight, link
	// nil if not found
	PrefColor(name string) *color.RGBA

	// PrefFontFamily returns the default FontFamily
	PrefFontFamily() string
}

// ThePrefs is the prefs object to use to get preferences.
var ThePrefs Prefer

// Prefs provides a basic implementation of Prefer interface
type Prefs struct {

	// font family name
	FontFamily string `desc:"font family name"`

	// default font / pen color
	Font color.RGBA `desc:"default font / pen color"`

	// default background color
	Background color.RGBA `desc:"default background color"`

	// color for shadows -- should generally be a darker shade of the background color
	Shadow color.RGBA `desc:"color for shadows -- should generally be a darker shade of the background color"`

	// default border color, for button, frame borders, etc
	Border color.RGBA `desc:"default border color, for button, frame borders, etc"`

	// default main color for controls: buttons, etc
	Control color.RGBA `desc:"default main color for controls: buttons, etc"`

	// color for icons or other solidly-colored, small elements
	Icon color.RGBA `desc:"color for icons or other solidly-colored, small elements"`

	// color for selected elements
	Select color.RGBA `desc:"color for selected elements"`

	// color for highlight background
	Highlight color.RGBA `desc:"color for highlight background"`

	// color for links in text etc
	Link color.RGBA `desc:"color for links in text etc"`
}

func (pf *Prefs) Defaults() {
	pf.FontFamily = "Go"
	pf.Font = colors.Black
	pf.Border = colors.MustFromHex("#666")
	pf.Background = colors.White
	pf.Shadow = colors.MustFromString("darken-10", &pf.Background)
	pf.Control = colors.MustFromHex("#F8F8F8")
	pf.Icon = colors.MustFromString("highlight-30", pf.Control)
	pf.Select = colors.MustFromHex("#CFC")
	pf.Highlight = colors.MustFromHex("#FFA")
	pf.Link = colors.MustFromHex("#00F")
}

// PrefColor returns preference color of given name (case insensitive)
// std names are: font, background, shadow, border, control, icon, select, highlight, link
func (pf *Prefs) PrefColor(clrName string) *color.RGBA {
	lc := strings.Replace(strings.ToLower(clrName), "-", "", -1)
	switch lc {
	case "font":
		return &pf.Font
	case "background":
		return &pf.Background
	case "shadow":
		return &pf.Shadow
	case "border":
		return &pf.Border
	case "control":
		return &pf.Control
	case "icon":
		return &pf.Icon
	case "select":
		return &pf.Select
	case "highlight":
		return &pf.Highlight
	case "link":
		return &pf.Link
	}
	log.Printf("Preference color %v (simplified to: %v) not found\n", clrName, lc)
	return nil
}

// PrefFontFamily returns the default FontFamily
func (pf *Prefs) PrefFontFamily() string {
	return pf.FontFamily
}
