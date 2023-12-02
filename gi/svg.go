// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/girl/styles"
	"goki.dev/girl/units"
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
	sv.SVG = svg.NewSVG(0, 0)
	sv.SVG.Norm = true
	sv.HandleWidgetEvents()
	sv.SVGStyles()
}

func (sv *SVG) SVGStyles() {
	sv.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Min.Set(units.Dp(sv.SVG.Root.ViewBox.Size.X), units.Dp(sv.SVG.Root.ViewBox.Size.Y))
	})
}

func (sv *SVG) SizeFinal() {
	sv.WidgetBase.SizeFinal()
	sv.SVG.Resize(sv.Geom.Size.Actual.Content.ToPoint())
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
