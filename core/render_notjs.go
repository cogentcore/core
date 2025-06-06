// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package core

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/renderers/rasterx"
	"cogentcore.org/core/system/composer"
	"golang.org/x/image/draw"
)

// grabRenderFrom grabs the rendered image from the given widget.
// If it returns nil, then the image could not be fetched.
func grabRenderFrom(w Widget) *image.RGBA {
	wb := w.AsWidget()
	scimg := wb.Scene.renderer.Image()
	if scimg == nil {
		return nil
	}
	if wb.Geom.TotalBBox.Empty() { // the widget is offscreen
		return nil
	}
	sz := wb.Geom.TotalBBox.Size()
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), scimg, wb.Geom.TotalBBox.Min, draw.Src)
	return img
}

func (ps *paintSource) Draw(c composer.Composer) {
	cd := c.(*composer.ComposerDrawer)
	rd := ps.renderer.(*rasterx.Renderer)

	unchanged := len(ps.render) == 0
	if !unchanged {
		rd.Render(ps.render)
	}
	img := rd.Image()
	cd.Drawer.Copy(ps.drawPos, img, img.Bounds(), ps.drawOp, unchanged)
}

func (ss *scrimSource) Draw(c composer.Composer) {
	cd := c.(*composer.ComposerDrawer)
	clr := colors.Uniform(colors.ApplyOpacity(colors.ToUniform(colors.Scheme.Scrim), 0.5))
	cd.Drawer.Copy(image.Point{}, clr, ss.bbox, draw.Over, composer.Unchanged)
}

////////  fillInsets

// fillInsetsSource is a [composer.Source] implementation for fillInsets.
type fillInsetsSource struct {
	rbb, wbb image.Rectangle
}

func (ss *fillInsetsSource) Draw(c composer.Composer) {
	cd := c.(*composer.ComposerDrawer)
	clr := colors.Scheme.Background

	fill := func(x0, y0, x1, y1 int) {
		r := image.Rect(x0, y0, x1, y1)
		if r.Dx() == 0 || r.Dy() == 0 {
			return
		}
		cd.Drawer.Copy(image.Point{}, clr, r, draw.Src, composer.Unchanged)
	}
	rb := ss.rbb
	wb := ss.wbb
	fill(0, 0, wb.Max.X, rb.Min.Y)        // top
	fill(0, rb.Max.Y, wb.Max.X, wb.Max.Y) // bottom
	fill(rb.Max.X, 0, wb.Max.X, wb.Max.Y) // right
	fill(0, 0, rb.Min.X, wb.Max.Y)        // left
}

// fillInsets fills the window insets, if any, with [colors.Scheme.Background].
func (w *renderWindow) fillInsets(cp composer.Composer) {
	// render geom and window geom
	rg := w.SystemWindow.RenderGeom()
	wg := math32.Geom2DInt{Size: w.SystemWindow.Size()}

	// if our window geom is the same as our render geom, we have no
	// window insets to fill
	if wg == rg {
		return
	}
	cp.Add(&fillInsetsSource{rbb: rg.Bounds(), wbb: wg.Bounds()}, w)
}
