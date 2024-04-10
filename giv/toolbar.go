// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
)

// ConfigImageToolbar configures the given toolbar for the given image.
func ConfigImageToolbar(tb *core.Toolbar, im *core.Image) {
	NewFuncButton(tb, im.Open).SetIcon(icons.Open)
}

// ConfigSVGToolbar configures the given toolbar for the given SVG.
func ConfigSVGToolbar(tb *core.Toolbar, sv *core.SVG) {
	// TODO(kai): resolve svg panning and selection structure
	core.NewButton(tb).SetIcon(icons.PanTool).
		SetTooltip("toggle the ability to zoom and pan the view").OnClick(func(e events.Event) {
		sv.SetReadOnly(!sv.IsReadOnly())
		sv.ApplyStyleUpdate()
	})
	core.NewButton(tb).SetIcon(icons.ArrowForward).
		SetTooltip("turn on select mode for selecting SVG elements").
		OnClick(func(e events.Event) {
			fmt.Println("this will select select mode")
		})
	core.NewSeparator(tb)
	NewFuncButton(tb, sv.Open).SetIcon(icons.Open)
	NewFuncButton(tb, sv.SaveSVG).SetIcon(icons.Save)
	NewFuncButton(tb, sv.SavePNG).SetIcon(icons.Save)
}
