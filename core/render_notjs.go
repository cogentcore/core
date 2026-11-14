// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package core

import (
	"cogentcore.org/core/base/iox/imagex"
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
	scimg := wb.Scene.Renderer.Image()
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
	cd.Drawer.Copy(ss.bbox.Min, clr, ss.bbox, draw.Over, composer.Unchanged) // (at its place)
}

////////  fillInsets

// fillInsetsSource is a [composer.Source] implementation for fillInsets.
type fillInsetsSource struct {
	rbb, wbb image.Rectangle

	// clr is the color to fill them with.
	clr image.Image
}

func (ss *fillInsetsSource) Draw(c composer.Composer) {
	cd := c.(*composer.ComposerDrawer)
	fill := func(x0, y0, x1, y1 int) {
		r := image.Rect(x0, y0, x1, y1)
		if r.Dx() == 0 || r.Dy() == 0 {
			return
		}
		cd.Drawer.Copy(r.Min, ss.clr, r, draw.Src, composer.Unchanged)
	}
	rb := ss.rbb
	wb := ss.wbb
	fill(0, 0, wb.Max.X, rb.Min.Y)        // top
	fill(0, rb.Max.Y, wb.Max.X, wb.Max.Y) // bottom
	fill(rb.Max.X, 0, wb.Max.X, wb.Max.Y) // right
	fill(0, 0, rb.Min.X, wb.Max.Y)        // left
}

// fillInsets fills the window insets, if any (as behind the system bars
// on mobile), with the background color of the body of the top window
// scene, so that they match it, or else [colors.Scheme.Surface].
func (w *renderWindow) fillInsets(cp composer.Composer) {
	// render geom and window geom
	rg := w.SystemWindow.RenderGeom()
	wg := math32.Geom2DInt{Size: w.SystemWindow.Size()}

	// if our window geom is the same as our render geom, we have no
	// window insets to fill
	if wg == rg {
		return
	}
	var clr image.Image = colors.Scheme.Surface
	for _, st := range w.mains.stack.Values() {
		if st.Type != WindowStage || st.Scene == nil {
			continue
		}
		// that of its body, which is what is seen, or else that of the
		// scene, if it is opaque (a transparent one would leave the
		// background of the system window)
		opaque := func(bg image.Image) bool {
			u, ok := bg.(*image.Uniform)
			if !ok {
				return false
			}
			_, _, _, a := u.RGBA()
			return a == 0xffff
		}
		if bg := st.Scene.Styles.ActualBackground; opaque(bg) {
			clr = bg
		}
		if bd := st.Scene.Body; bd != nil && opaque(bd.Styles.ActualBackground) {
			clr = bd.Styles.ActualBackground
		}
	}
	cp.Add(&fillInsetsSource{rbb: rg.Bounds(), wbb: wg.Bounds(), clr: clr}, w)
}

// getImager is implemented by offscreen.Drawer for [Body.AssertRender].
type getImager interface {
	GetImage() *image.RGBA
}

// windowImage returns a copy of the rendered image of the given window,
// for the tests (see [Body.RenderImage]), or nil if it is closed.
func windowImage(rw *renderWindow) *image.RGBA {
	// The image is copied under the window lock, which is held while it is
	// composed, as paint events can still make the event loop render.
	dw := rw.SystemWindow.Composer().(*composer.ComposerDrawer).Drawer
	if !rw.SystemWindow.Lock() {
		return nil
	}
	img := imagex.CloneAsRGBA(dw.(getImager).GetImage())
	rw.SystemWindow.Unlock()
	return img
}

// testImageName returns the name of the test image of the given name in
// testdata (see [Body.AssertRender]): the same, other than on the web.
func testImageName(name string) string {
	return name
}

// clearTestWindow is for the tests on the web (see the web version), and
// does nothing elsewhere, where the windows of tests are closed with all
// of their stages.
func clearTestWindow() {}
