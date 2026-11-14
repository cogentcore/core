// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package core

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
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
	// rendered by the browser, with its text, as in the canvas of the
	// scene, rather than by rasterx, which has no fonts on the web (#1552)
	imgRend := htmlcanvas.NewImage(math32.FromPoint(wb.Scene.SceneGeom.Size))
	wb.RenderWidget()
	rend := wb.Scene.Painter.RenderDone()
	imgRend.Render(rend)
	scimg := imgRend.Image()
	if scimg == nil {
		return nil
	}
	sz := wb.Geom.TotalBBox.Size()
	img := image.NewRGBA(image.Rectangle{Max: sz})
	// over the background of the parent, as the widget is shown, where it
	// does not render its own background
	if pbg := wb.parentActualBackground(); pbg != nil {
		draw.Draw(img, img.Bounds(), pbg, wb.Geom.TotalBBox.Min, draw.Src)
	}
	draw.Draw(img, img.Bounds(), scimg, wb.Geom.TotalBBox.Min, draw.Over)
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

// windowImage returns an image of the given window, for the tests (see
// [Body.RenderImage]), composited from the elements of its composer.
func windowImage(rw *renderWindow) *image.RGBA {
	cw, ok := rw.SystemWindow.Composer().(*composer.ComposerWeb)
	if !ok {
		return nil
	}
	return cw.Image(rw.SystemWindow.Size())
}

// testImageName returns the name of the test image of the given name in
// testdata (see [Body.AssertRender]): in testdata/js on the web, as the
// browser renders text and paths differently than on other platforms.
func testImageName(name string) string {
	return "js/" + name
}

// clearTestWindow closes the stages left in the window of the tests by
// earlier tests (as dialogs, or windows that were not closed), so that a
// test starts with an empty window, as elsewhere, where the window of a
// test is closed with all of its stages, whereas on the web there is one
// window, which stays.
func clearTestWindow() {
	cw := getCurrentRenderWindow()
	if cw == nil {
		return
	}
	rc := cw.renderContext()
	rc.Lock()
	defer rc.Unlock()
	for range 100 { // (in case a stage does not close)
		st := cw.mains.top()
		if st == nil {
			return
		}
		cw.mains.deleteStage(st) // (with its popups)
	}
}
