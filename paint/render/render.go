// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package render

import (
	"image"
	"image/draw"
	"reflect"
	"slices"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/system"
)

// Render is an interface for render data, that should contain
// everything necessary to render to a [Renderer], including the
// list of such Renderers. It is essential that the Render state
// is fully self-contained and does not depend on any other elements,
// so that a full [Scene] of such render state can be drawn in a
// separate goroutine.
type Render interface {
	// Render renders this render data to its list of [Renderer]s.
	// For non-[PaintRender] types, this should populate the relevant
	// rendering data.
	Render()

	// Draw draws the resulting render to given sytem drawer,
	// for renderers that support this functionality.
	Draw(drw system.Drawer)
}

// PaintRender is a standard painting [Render] that
// has a collection of render [Item]s to be rendered.
type PaintRender struct {
	// Items are the list of render [Item]s to be rendered.
	Items []Item

	// Renderers is the list of [Renderer]s that we render to.
	Renderers []Renderer

	// DrawOp is the [draw.Op] operation: [draw.Src] to copy source,
	// [draw.Over] to alpha blend.
	DrawOp draw.Op

	// DrawPos is the position offset for the [Image] renderer to
	// use in its Draw to a [system.Drawer] (i.e., the [core.Scene] position).
	DrawPos image.Point
}

// Render renders this render data to its list of [Renderer]s.
func (pr *PaintRender) Render() {
	for _, rd := range pr.Renderers {
		rd.Render(pr)
	}
}

// Draw draws the resulting render to given system drawer,
// for renderers that support this functionality.
func (pr *PaintRender) Draw(drw system.Drawer) {
	for _, rd := range pr.Renderers {
		rd.Draw(pr, drw, pr.DrawOp)
	}
}

// Clone returns a copy of this PaintRender,
// with shallow clones of the Items and Renderers lists.
func (pr *PaintRender) Clone() *PaintRender {
	npr := &PaintRender{}
	npr.Items = slices.Clone(pr.Items)
	npr.Renderers = slices.Clone(pr.Renderers)
	npr.DrawOp = pr.DrawOp
	npr.DrawPos = pr.DrawPos
	return npr
}

// Add adds item(s) to render. Filters any nil items.
func (pr *PaintRender) Add(item ...Item) *PaintRender {
	for _, it := range item {
		if reflectx.IsNil(reflect.ValueOf(it)) {
			continue
		}
		pr.Items = append(pr.Items, it)
	}
	return pr
}

// Reset resets back to an empty Render state.
// It preserves the existing slice memory for re-use.
func (pr *PaintRender) Reset() {
	pr.Items = pr.Items[:0]
}

// ImageRenderer returns the first ImageRenderer present, or nil if none.
func (pr *PaintRender) ImageRenderer() Renderer {
	for _, rd := range pr.Renderers {
		if rd.Type() == Image {
			return rd
		}
	}
	return nil
}

// Image returns the Go [image.RGBA] from our first [Image] renderer
// if present, else nil.
func (pr *PaintRender) Image() *image.RGBA {
	rd := pr.ImageRenderer()
	if rd == nil {
		return nil
	}
	return rd.Image()
}

// DrawImage draws the first [Image] renderer image to the given
// [system.Drawer]. It uses the presence of Items to determine if the
// image is changed or not. This is an implementation for [Renderer.Draw].
func (pr *PaintRender) DrawImage(drw system.Drawer, op draw.Op) {
	rd := pr.ImageRenderer()
	if rd == nil {
		return
	}
	img := rd.Image()
	if img == nil {
		return
	}
	unchanged := len(pr.Items) == 0
	drw.Copy(pr.DrawPos, img, img.Bounds(), op, unchanged)
}
