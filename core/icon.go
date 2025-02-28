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

	// svg drawing of the icon
	svg svg.SVG
}

func (ic *Icon) WidgetValue() any { return &ic.Icon }

func (ic *Icon) Init() {
	ic.WidgetBase.Init()
	ic.svg.Scale = 1

	ic.Updater(ic.readIcon)
	ic.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(1))
	})
	ic.FinalStyler(func(s *styles.Style) {
		if ic.svg.Root != nil {
			ic.svg.Root.ViewBox.PreserveAspectRatio.SetFromStyle(s)
		}
	})
}

// readIcon reads the [Icon.Icon] if necessary.
func (ic *Icon) readIcon() {
	if ic.Icon == ic.prevIcon {
		// if nothing has changed, we don't need to read it
		return
	}
	if !ic.Icon.IsSet() {
		ic.svg.DeleteAll()
		ic.prevIcon = ic.Icon
		return
	}

	ic.svg.Config(2, 2)
	err := ic.svg.ReadXML(strings.NewReader(string(ic.Icon)))
	if errors.Log(err) != nil {
		return
	}
	icons.Used[ic.Icon] = struct{}{}
	ic.prevIcon = ic.Icon
}

// renderSVG renders the [Icon.svg] if necessary.
func (ic *Icon) renderSVG() {
	if ic.svg.Root == nil || !ic.svg.Root.HasChildren() {
		return
	}

	sv := &ic.svg
	sv.TextShaper = ic.Scene.TextShaper
	sz := ic.Geom.Size.Actual.Content.ToPoint()
	clr := gradient.ApplyOpacity(ic.Styles.Color, ic.Styles.Opacity)
	if !ic.NeedsRebuild() { // if rebuilding then rebuild
		isz := sv.Geom.Size
		// if nothing has changed, we don't need to re-render
		if isz == sz && sv.Name == string(ic.Icon) && sv.Color == clr {
			return
		}
	}

	if sz == (image.Point{}) {
		return
	}
	sv.Geom.Size = sz // make sure
	sv.Resize(sz)     // does Config if needed
	sv.Color = clr
	sv.Scale = 1
	sv.Render()
	sv.Name = string(ic.Icon)
}

func (ic *Icon) Render() {
	ic.renderSVG()

	img := ic.svg.RenderImage()
	if img == nil {
		return
	}
	r := ic.Geom.ContentBBox
	sp := ic.Geom.ScrollOffset()
	ic.Scene.Painter.DrawImage(img, r, sp, draw.Over)
}
