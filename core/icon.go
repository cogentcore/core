// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/svg"
	"golang.org/x/image/draw"
)

// Icon renders an [icons.Icon].
// The rendered version is cached for the current size.
// Icons do not render a background or border independent of their SVG object.
// The size of an Icon is determined by the [styles.Text.FontSize] property.
type Icon struct {
	WidgetBase

	// Icon is the [icons.Icon] used to render the [Icon].
	Icon icons.Icon

	// prevIcon is the previously rendered icon.
	prevIcon icons.Icon

	// image representation of the icon, cached for faster drawing.
	pixels *image.RGBA
}

func (ic *Icon) WidgetValue() any { return &ic.Icon }

func (ic *Icon) Init() {
	ic.WidgetBase.Init()
	ic.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(1))
	})
}

// RerenderSVG forcibly renders the icon, returning the [svg.SVG]
// used to render.
func (ic *Icon) RerenderSVG() *svg.SVG {
	ic.pixels = nil
	ic.prevIcon = ""
	return ic.renderSVG()
}

// renderSVG renders the icon if necessary, returning the [svg.SVG]
// used to render if it was rendered, otherwise nil.
func (ic *Icon) renderSVG() *svg.SVG {
	sz := ic.Geom.Size.Actual.Content.ToPoint()
	if sz == (image.Point{}) {
		return nil
	}
	var isz image.Point
	if ic.pixels != nil {
		isz = ic.pixels.Bounds().Size()
	}
	if ic.Icon == ic.prevIcon && sz == isz && !ic.NeedsRebuild() {
		return nil
	}
	ic.pixels = nil
	if !ic.Icon.IsSet() {
		ic.prevIcon = ic.Icon
		return nil
	}
	sv := svg.NewSVG(sz.X, sz.Y)
	err := sv.ReadXML(strings.NewReader(string(ic.Icon)))
	if errors.Log(err) != nil || sv.Root == nil || !sv.Root.HasChildren() {
		return nil
	}
	icons.Used[ic.Icon] = struct{}{}
	ic.prevIcon = ic.Icon
	sv.Root.ViewBox.PreserveAspectRatio.SetFromStyle(&ic.Styles)
	sv.TextShaper = ic.Scene.TextShaper()
	// todo: we aren't rebuilding on color change
	clr := gradient.ApplyOpacity(ic.Styles.Color, ic.Styles.Opacity)
	sv.Color = clr
	sv.Scale = 1
	sv.Render()
	ic.pixels = sv.RenderImage()
	return sv
}

func (ic *Icon) Render() {
	ic.renderSVG()
	if ic.pixels == nil {
		return
	}
	r := ic.Geom.ContentBBox
	sp := ic.Geom.ScrollOffset()
	ic.Scene.Painter.DrawImage(ic.pixels, r, sp, draw.Over)
}
