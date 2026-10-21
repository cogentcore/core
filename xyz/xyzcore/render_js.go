// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package xyzcore

import (
	"image"
	"image/draw"
	"syscall/js"

	"cogentcore.org/core/core"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/phong"
	"cogentcore.org/core/system/composer"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// canvases are the canvas elements with the GPU surfaces of the scenes.
// A scene keeps its canvas for as long as it has its surface, because
// [composer.ComposerWeb.Compose] removes the element of a source that is
// not drawn, as when the scene is hidden in a tab that is not selected or
// scrolled offscreen, and a new canvas would have no surface.
var canvases = map[*Scene]js.Value{}

// xyzSource implements [composer.Source] for core direct rendering.
type xyzSource struct {
	sw *Scene
}

func (xs *xyzSource) Draw(c composer.Composer) {
	cw := c.(*composer.ComposerWeb)
	elem := xs.sw.canvas(cw)
	cw.SetElementGeom(elem, xs.sw.Geom.ContentBBox.Min, xs.sw.Geom.ContentBBox.Size())
}

// RenderSource returns the [composer.Source] for direct rendering.
func (sw *Scene) RenderSource(op draw.Op) composer.Source {
	if sw.XYZ.Frame == nil || !sw.IsVisible() {
		return nil
	}
	return &xyzSource{sw: sw}
}

// canvas returns the canvas element of the scene for the given composer,
// making it the first time. It is the element of sw.This, the widget
// registered in [Scene.OnAdd], whose render source draws to the element of
// that context, so that we have the same element as in [xyzSource.Draw].
func (sw *Scene) canvas(cw *composer.ComposerWeb) js.Value {
	if elem, ok := canvases[sw]; ok {
		cw.SetElementContext(sw.This, elem)
		return elem
	}
	elem := cw.ElementContext(sw.This, "canvas")
	canvases[sw] = elem
	return elem
}

// configFrame configures the render frame in a platform-specific manner.
func (sw *Scene) configFrame(sz image.Point) {
	// Even though we are not in [xyzSource.Draw], we can get the Composer
	// and use it to get the canvas element, which we may need before we
	// get to [xyzSource.Draw].
	cw := core.TheApp.Window(0).Composer().(*composer.ComposerWeb)
	elem := sw.canvas(cw)
	cw.SetElementGeom(elem, sw.Geom.ContentBBox.Min, sz)

	wsurf := gpu.Instance().CreateSurface(&wgpu.SurfaceDescriptor{Canvas: elem})
	gp := gpu.NewGPU(nil)
	sc := sw.XYZ
	sc.Frame = gpu.NewSurface(gp, wsurf, sz, 4, gpu.Depth32)
	sc.Phong = phong.NewPhong(gp, sc.Frame)
	sc.ConfigNewPhong()
}

// destroyFrame forgets the canvas of the scene, which the composer removes
// as its widget is no longer drawn.
func (sw *Scene) destroyFrame() {
	delete(canvases, sw)
}
