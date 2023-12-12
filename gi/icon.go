// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log/slog"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/icons"
	"goki.dev/svg"
	"golang.org/x/image/draw"
)

// Icon contains a svg.SVG element.
// The rendered version is cached for a given size.
type Icon struct {
	WidgetBase

	// icon name that has been set.
	IconName icons.Icon `set:"-"`

	// file name for the loaded icon, if loaded
	Filename string `set:"-"`

	// SVG drawing
	SVG svg.SVG `set:"-"`

	// RendSize is the last rendered size of the Icon SVG.
	// if the SVG.Name == IconName and this size is the same
	// then the current SVG image is used.
	RendSize image.Point `set:"-"`
}

func (ic *Icon) CopyFieldsFrom(frm any) {
	fr := frm.(*Icon)
	ic.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	ic.IconName = fr.IconName
	ic.Filename = fr.Filename
}

func (ic *Icon) OnInit() {
	ic.HandleWidgetEvents()
	ic.IconStyles()
}

func (ic *Icon) IconStyles() {
	ic.SVG.Norm = true
	ic.Style(func(s *styles.Style) {
		s.Min.X.Dp(16)
		s.Min.Y.Dp(16)
	})
}

// SetIcon sets the icon, logging error if not found.
// Does nothing if IconName is already == icon name.
func (ic *Icon) SetIcon(icon icons.Icon) *Icon {
	_, err := ic.SetIconTry(icon)
	if err != nil {
		slog.Error("error opening icon named", "name", icon, "err", err)
	}
	return ic
}

// SetIconUpdate sets the icon and sets flag to render.
// Does nothing if IconName is already == icon name.
func (ic *Icon) SetIconUpdate(icon icons.Icon) *Icon {
	ic.SetIcon(icon)
	ic.SetNeedsRender(true)
	return ic
}

// SetIconTry sets the icon, returning error
// message if not found etc, and returning true if a new icon was actually set.
// Does nothing and returns false if IconName is already == icon name.
func (ic *Icon) SetIconTry(icon icons.Icon) (bool, error) {
	if icon.IsNil() {
		ic.SVG.DeleteAll()
		ic.Config()
		return false, nil
	}
	if ic.SVG.Root.HasChildren() && ic.IconName == icon {
		return false, nil
	}
	fnm := icon.Filename()
	ic.SVG.Config(2, 2)
	err := ic.SVG.OpenFS(icons.Icons, fnm)
	if err != nil {
		ic.Config()
		return false, err
	}
	ic.IconName = icon
	return true, nil

}

func (ic *Icon) DrawIntoScene() {
	_, st := ic.RenderLock()
	defer ic.RenderUnlock()
	ic.RenderStdBox(st)

	if ic.SVG.Pixels == nil {
		return
	}
	r := ic.Geom.ContentBBox
	sp := image.Point{}
	if ic.Par != nil { // use parents children bbox to determine where we can draw
		_, pwb := AsWidget(ic.Par)
		pbb := pwb.Geom.ContentBBox
		nr := r.Intersect(pbb)
		sp = nr.Min.Sub(r.Min)
		if sp.X < 0 || sp.Y < 0 || sp.X > 10000 || sp.Y > 10000 {
			slog.Error("gi.Icon bad bounding box", "path", ic, "startPos", sp, "bbox", r, "parBBox", pbb)
			return
		}
		r = nr
	}
	draw.Draw(ic.Sc.Pixels, r, ic.SVG.Pixels, sp, draw.Over)
}

// RenderSVG renders the SVG to Pixels if needs update
func (ic *Icon) RenderSVG() {
	rc := ic.Sc.RenderCtx()
	sv := &ic.SVG
	clr := colors.ApplyOpacity(ic.Styles.Color, ic.Styles.Opacity)
	if !rc.HasFlag(RenderRebuild) && sv.Pixels != nil { // if rebuilding rebuild..
		isz := sv.Pixels.Bounds().Size()
		// if nothing has changed, we don't need to re-render
		if isz == ic.RendSize && sv.Name == string(ic.IconName) && sv.Color.Solid == clr {
			return
		}
	}
	// todo: units context from us to SVG??
	zp := image.Point{}
	sz := ic.Geom.Size.Actual.Content.ToPoint()
	if sz == zp {
		ic.RendSize = zp
		return
	}
	sv.Resize(sz) // does Config if needed

	// TODO(kai): what about gradient icons?
	ic.SVG.Color.SetSolid(clr)

	sv.Render()
	ic.RendSize = sz
	sv.Name = string(ic.IconName)
	// fmt.Println("re-rendered icon:", sv.Name, "size:", sz)
}

func (ic *Icon) Render() {
	ic.RenderSVG()
	if ic.PushBounds() {
		ic.RenderChildren()
		ic.DrawIntoScene()
		ic.PopBounds()
	}
}
