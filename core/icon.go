// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"image/color"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"golang.org/x/image/draw"
)

// Icon renders an [icons.Icon].
// The rendered version is cached for the current size.
// Icons do not render a background or border independent of their SVG object.
// The size of an Icon is determined by the [styles.Font.Size] property.
type Icon struct {
	WidgetBase

	// Icon is the [icons.Icon] used to render the [Icon].
	Icon icons.Icon

	// prevIcon is the previously rendered icon.
	prevIcon icons.Icon

	// prevColor is the previously rendered color, as uniform.
	prevColor color.RGBA

	// prevOpacity is the previously rendered opacity.
	prevOpacity float32

	// image representation of the icon, cached for faster drawing.
	pixels image.Image
}

func (ic *Icon) WidgetValue() any { return &ic.Icon }

func (ic *Icon) Init() {
	ic.WidgetBase.Init()
	ic.FinalStyler(func(s *styles.Style) {
		s.Min = s.IconSize
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
	cc := colors.ToUniform(ic.Styles.Color)
	if ic.Icon == ic.prevIcon && sz == isz && ic.prevColor == cc && ic.prevOpacity == ic.Styles.Opacity && !ic.NeedsRebuild() {
		return nil
	}
	ic.pixels = nil
	if !ic.Icon.IsSet() {
		ic.prevIcon = ic.Icon
		return nil
	}
	sv := svg.NewSVG(ic.Geom.Size.Actual.Content)
	err := sv.ReadXML(strings.NewReader(string(ic.Icon)))
	if errors.Log(err) != nil || sv.Root == nil || !sv.Root.HasChildren() {
		return nil
	}
	icons.AddUsed(ic.Icon)
	ic.prevIcon = ic.Icon
	sv.Root.ViewBox.PreserveAspectRatio.SetFromStyle(&ic.Styles)
	sv.TextShaper = ic.Scene.TextShaper()
	clr := gradient.ApplyOpacity(ic.Styles.Color, ic.Styles.Opacity)
	sv.DefaultFill = clr
	sv.Scale = 1
	ic.pixels = sv.RenderImage()
	ic.prevColor = cc
	ic.prevOpacity = ic.Styles.Opacity
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
