// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image/color"
	"log"
	"strings"

	"github.com/goki/gi/gist"
)

// Prefs are needed for setting gist.ThePrefs, for any text-based
// rendering, as it relies on these prefs
type Prefs struct {
	FontFamily string     `desc:"font family name"`
	Font       gist.Color `desc:"default font / pen color"`
	Background gist.Color `desc:"default background color"`
	Shadow     gist.Color `desc:"color for shadows -- should generally be a darker shade of the background color"`
	Border     gist.Color `desc:"default border color, for button, frame borders, etc"`
	Control    gist.Color `desc:"default main color for controls: buttons, etc"`
	Icon       gist.Color `desc:"color for icons or other solidly-colored, small elements"`
	Select     gist.Color `desc:"color for selected elements"`
	Highlight  gist.Color `desc:"color for highlight background"`
	Link       gist.Color `desc:"color for links in text etc"`
}

func (pf *Prefs) Defaults() {
	pf.FontFamily = "Go"
	pf.Font.SetColor(color.Black)
	pf.Border.SetString("#666", nil)
	pf.Background.SetColor(color.White)
	pf.Shadow.SetString("darker-10", &pf.Background)
	pf.Control.SetString("#F8F8F8", nil)
	pf.Icon.SetString("highlight-30", pf.Control)
	pf.Select.SetString("#CFC", nil)
	pf.Highlight.SetString("#FFA", nil)
	pf.Link.SetString("#00F", nil)
}

// PrefColor returns preference color of given name (case insensitive)
// std names are: font, background, shadow, border, control, icon, select, highlight, link
func (pf *Prefs) PrefColor(clrName string) *gist.Color {
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
