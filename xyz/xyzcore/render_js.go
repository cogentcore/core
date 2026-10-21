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
// The GPU device, surface and Phong renderer are made once, the first time.
// After that, a new size only reconfigures the surface and remakes its
// depth and multisample textures, as [xyz.Scene.ConfigOffscreen] does for
// the render texture in native builds.
func (sw *Scene) configFrame(sz image.Point) {
	// Even though we are not in [xyzSource.Draw], we can get the Composer
	// and use it to get the canvas element, which we may need before we
	// get to [xyzSource.Draw]. The size of the canvas is that of the
	// textures that the surface gives, so it is set first.
	cw := core.TheApp.Window(0).Composer().(*composer.ComposerWeb)
	elem := sw.canvas(cw)
	cw.SetElementGeom(elem, sw.Geom.ContentBBox.Min, sz)

	sc := sw.XYZ
	if sc.Frame != nil {
		sc.Lock()
		sc.SetSize(sz)
		sc.Unlock()
		return
	}
	wsurf := gpu.Instance().CreateSurface(&wgpu.SurfaceDescriptor{Canvas: elem})
	gp := gpu.NewGPU(nil)
	sc.Frame = gpu.NewSurface(gp, wsurf, sz, 4, gpu.Depth32)
	sc.Phong = phong.NewPhong(gp, sc.Frame)
	sc.ConfigNewPhong()
}

// destroyFrame releases the GPU resources of the canvas of the scene, after
// [xyz.Scene.Destroy]: the Release methods of WebGPU objects do nothing on
// the web, so it unconfigures the canvas context and destroys the device of
// the surface, which frees all of its buffers and textures at once. The
// composer removes the canvas as the widget is no longer drawn.
func (sw *Scene) destroyFrame() {
	elem, ok := canvases[sw]
	if !ok {
		return
	}
	delete(canvases, sw)
	ctx := elem.Call("getContext", "webgpu")
	if !ctx.Truthy() {
		return
	}
	var device js.Value
	if ctx.Get("getConfiguration").Truthy() {
		if conf := ctx.Call("getConfiguration"); conf.Truthy() {
			device = conf.Get("device")
		}
	}
	ctx.Call("unconfigure")
	if device.Truthy() {
		device.Call("destroy")
	}
}
