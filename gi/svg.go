// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/girl/styles"
	"goki.dev/svg"
	"golang.org/x/image/draw"
)

// SVG is a Widget that renders an [svg.SVG] object. It expects to be a terminal
// node and does NOT call rendering etc on its children.
type SVG struct {
	WidgetBase

	// SVG is the SVG object associated with the element.
	SVG *svg.SVG
}

func (sv *SVG) CopyFieldsFrom(frm any) {
	fr := frm.(*SVG)
	sv.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	sv.SVG = fr.SVG
}

func (sv *SVG) OnInit() {
	sv.HandleWidgetEvents()
	sv.SVGStyles()
}

func (sv *SVG) SVGStyles() {
	sv.Style(func(s *styles.Style) {
		if sv.SVG.Pixels != nil {
			sz := sv.SVG.Pixels.Bounds().Size()
			s.Min.X.Dp(float32(sz.X))
			s.Min.Y.Dp(float32(sz.Y))
		}
	})
}

func (sv *SVG) DrawIntoScene() {
	if sv.SVG == nil {
		return
	}
	sv.SVG.Render()
	r := sv.Geom.TotalBBox
	sp := image.Point{}
	draw.Draw(sv.Sc.Pixels, r, sv.SVG.Pixels, sp, draw.Over)
}

func (sv *SVG) Render() {
	if sv.PushBounds() {
		sv.RenderChildren()
		sv.DrawIntoScene()
		sv.PopBounds()
	}
}
