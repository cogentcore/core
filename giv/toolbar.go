// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
)

// ConfigImageToolbar configures the given toolbar for the given image.
func ConfigImageToolbar(tb *gi.Toolbar, im *gi.Image) {
	gi.NewButton(tb).SetText("Open Image").SetIcon(icons.Open).
		OnClick(func(e events.Event) {
			CallFunc(im, im.OpenImage)
		})
}

// ConfigSVGToolbar configures the given toolbar for the given SVG.
func ConfigSVGToolbar(tb *gi.Toolbar, sv *gi.SVG) {
	gi.NewButton(tb).SetIcon(icons.PanTool).
		SetTooltip("toggle the ability to zoom and pan the view").OnClick(func(e events.Event) {
		sv.SetReadOnly(!sv.IsReadOnly())
		sv.ApplyStyleUpdate()
	})
	gi.NewButton(tb).SetIcon(icons.ArrowForward).
		SetTooltip("turn on select mode for selecting SVG elements").
		OnClick(func(e events.Event) {
			fmt.Println("this will select select mode")
		})
	gi.NewSeparator(tb)
	gi.NewButton(tb).SetText("Open SVG").SetIcon(icons.Open).
		SetTooltip("Open from SVG file").OnClick(func(e events.Event) {
		CallFunc(sv, sv.OpenSVG)
	})
	gi.NewButton(tb).SetText("Save SVG").SetIcon(icons.Save).
		SetTooltip("Save to SVG file").OnClick(func(e events.Event) {
		CallFunc(sv, sv.SaveSVG)
	})
	gi.NewButton(tb).SetText("Save PNG").SetIcon(icons.Save).
		SetTooltip("Save to PNG file").OnClick(func(e events.Event) {
		CallFunc(sv, sv.SavePNG)
	})
}
