// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package core

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/renderers/htmlcanvas"
	"cogentcore.org/core/system/composer"
	"golang.org/x/image/draw"
)

// grabRenderFrom grabs the rendered image from the given widget.
// If it returns nil, then the image could not be fetched.
func grabRenderFrom(w Widget) *image.RGBA {
	wb := w.AsWidget()
	if wb.Geom.TotalBBox.Empty() { // the widget is offscreen
		return nil
	}
	// todo: grab region from canvas!
	imgRend := paint.NewImageRenderer(math32.FromPoint(wb.Scene.SceneGeom.Size))
	wb.RenderWidget()
	rend := wb.Scene.Painter.RenderDone()
	imgRend.Render(rend)
	scimg := imgRend.Image()
	if scimg == nil {
		return nil
	}
	sz := wb.Geom.TotalBBox.Size()
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), scimg, wb.Geom.TotalBBox.Min, draw.Src)
	return img
}

func (ps *paintSource) Draw(c composer.Composer) {
	cw := c.(*composer.ComposerWeb)
	rd := ps.renderer.(*htmlcanvas.Renderer)

	elem := cw.Element(ps, "canvas")
	_, size := ps.renderer.Size()
	cw.SetElementGeom(elem, ps.drawPos, size.ToPoint())
	rd.SetCanvas(elem)
	ps.renderer.Render(ps.render)
}

func (ss *scrimSource) Draw(c composer.Composer) {
	cw := c.(*composer.ComposerWeb)
	clr := colors.ApplyOpacity(colors.ToUniform(colors.Scheme.Scrim), 0.5)
	elem := cw.Element(ss, "div")
	cw.SetElementGeom(elem, ss.bbox.Min, ss.bbox.Size())
	elem.Get("style").Set("backgroundColor", colors.AsHex(clr))
}

func (w *renderWindow) fillInsets(c composer.Composer) {
	// no-op
}
