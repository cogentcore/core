// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"log/slog"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/svg"
	"golang.org/x/image/draw"
)

// Icon renders an [svg.SVG] icon.
// The rendered version is cached for a given size.
// Icons do not render a background or border independent of their SVG object.
// The size of on Icon is determined by the [styles.Font.Size] property.
type Icon struct {
	WidgetBase

	// icon name that has been set.
	Icon icons.Icon `set:"-"`

	// file name for the loaded icon, if loaded
	Filename string `set:"-"`

	// SVG drawing of the icon
	SVG svg.SVG `set:"-" copier:"-"`
}

func (ic *Icon) WidgetValue() any { return &ic.Icon }

func (ic *Icon) Init() {
	ic.WidgetBase.Init()
	ic.SVG.Scale = 1
	ic.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(1))
	})
	ic.FinalStyler(func(s *styles.Style) {
		if ic.SVG.Root != nil {
			ic.SVG.Root.ViewBox.PreserveAspectRatio.SetFromStyle(s)
		}
	})
}

// SetIcon sets the icon, logging error if not found.
// Does nothing if Icon is already equal to the given icon.
func (ic *Icon) SetIcon(icon icons.Icon) *Icon {
	_, err := ic.SetIconTry(icon)
	if err != nil {
		slog.Error("error opening icon named", "name", icon, "err", err)
	}
	return ic
}

// SetIconTry sets the icon, returning error message if not found etc,
// and returning true if a new icon was actually set.
// Does nothing and returns false if Icon is already equal to the given icon.
func (ic *Icon) SetIconTry(icon icons.Icon) (bool, error) {
	if icon.IsNil() {
		ic.SVG.DeleteAll()
		return false, nil
	}
	if ic.SVG.Root != nil && ic.SVG.Root.HasChildren() && ic.Icon == icon {
		// fmt.Println("icon already set:", icon)
		return false, nil
	}
	fnm := icon.Filename()
	ic.SVG.Config(2, 2)
	err := ic.SVG.OpenFS(icons.Icons, fnm)
	if err != nil {
		ic.UpdateWidget()
		return false, err
	}
	ic.Icon = icon
	// fmt.Println("icon set:", icon)
	return true, nil

}

// RenderSVG renders the [Icon.SVG] to the [Icon.Pixels] if they need to be updated.
func (ic *Icon) RenderSVG() {
	rc := ic.Scene.RenderContext()
	sv := &ic.SVG
	sz := ic.Geom.Size.Actual.Content.ToPoint()
	clr := gradient.ApplyOpacityImage(ic.Styles.Color, ic.Styles.Opacity)
	if !rc.Rebuild && sv.Pixels != nil { // if rebuilding rebuild..
		isz := sv.Pixels.Bounds().Size()
		// if nothing has changed, we don't need to re-render
		if isz == sz && sv.Name == string(ic.Icon) && sv.Color == clr {
			return
		}
	}
	// todo: units context from us to SVG??

	if sz == (image.Point{}) {
		return
	}
	// ensure that we have new pixels to render to in order to prevent
	// us from rendering over ourself
	sv.Pixels = image.NewRGBA(image.Rectangle{Max: sz})
	sv.RenderState.Init(sz.X, sz.Y, sv.Pixels)
	sv.Geom.Size = sz // make sure

	sv.Resize(sz) // does Config if needed

	sv.Color = clr

	sv.Scale = 1
	sv.Render()
	sv.Name = string(ic.Icon)
}

func (ic *Icon) Render() {
	ic.RenderSVG()

	if ic.SVG.Pixels == nil {
		return
	}
	r := ic.Geom.ContentBBox
	sp := ic.Geom.ScrollOffset()
	draw.Draw(ic.Scene.Pixels, r, ic.SVG.Pixels, sp, draw.Over)
}
