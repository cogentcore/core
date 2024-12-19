// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"image"
	"image/draw"
	"syscall/js"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/gpudraw"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/system"
	"github.com/cogentcore/webgpu/wgpu"
)

// Drawer implements [system.Drawer] with a WebGPU-based drawer if available
// and a backup 2D image drawer if not.
type Drawer struct {
	// wgpu is the WebGPU-based drawer if available.
	wgpu *gpudraw.Drawer

	// base is used for the backup 2D image drawer.
	base system.DrawerBase

	// context2D is the 2D rendering context of the canvas
	// for the backup 2D image drawer.
	context2D js.Value
}

// AsGPUDrawer implements [gpudraw.AsGPUDrawer].
func (dw *Drawer) AsGPUDrawer() *gpudraw.Drawer {
	return dw.wgpu
}

// InitDrawer sets the [Drawer] to a WebGPU-based drawer if the browser
// supports WebGPU and a backup 2D image drawer otherwise.
func (a *App) InitDrawer() {
	// TODO(wgpu): various mobile and Linux browsers do not fully support WebGPU yet.
	isMobile := a.SystemPlatform().IsMobile() || a.SystemPlatform() == system.Linux
	// TODO(wgpu): Firefox currently does not support WebGPU in release builds.
	// isFirefox := strings.Contains(js.Global().Get("navigator").Get("userAgent").String(), "Firefox")
	isFirefox := false
	if isMobile || isFirefox || !js.Global().Get("navigator").Get("gpu").Truthy() {
		a.Draw.context2D = js.Global().Get("document").Call("querySelector", "canvas").Call("getContext", "2d")
		return
	}
	gp := gpu.NewGPU(nil)
	surf := gpu.Instance().CreateSurface(nil)
	sf := gpu.NewSurface(gp, surf, a.Scrn.PixelSize, 1, gpu.UndefinedType)
	a.Draw.wgpu = gpudraw.NewDrawer(gp, sf)
}

var loader = js.Global().Get("document").Call("getElementById", "app-wasm-loader")

// End ends image drawing rendering process on render target.
// This is the thing that actually does the drawing for the web
// backup 2D image drawer.
func (dw *Drawer) End() {
	if dw.wgpu != nil {
		dw.wgpu.End()
	} else {
		sz := dw.base.Image.Bounds().Size()
		bytes := wgpu.BytesToJS(dw.base.Image.Pix)
		data := js.Global().Get("ImageData").New(bytes, sz.X, sz.Y)
		dw.context2D.Call("putImageData", data, 0, 0)
	}
	// Only remove the loader after we have successfully rendered.
	if loader.Truthy() {
		loader.Call("remove")
		loader = js.Value{}
	}
}

func (dw *Drawer) DestBounds() image.Rectangle {
	if dw.wgpu != nil {
		return dw.wgpu.DestBounds()
	}
	return TheApp.Scrn.Geometry
}

func (dw *Drawer) Start() {
	if dw.wgpu != nil {
		dw.wgpu.Start()
		return
	}
	dw.base.Start()
}

func (dw *Drawer) Copy(dp image.Point, src image.Image, sr image.Rectangle, op draw.Op, unchanged bool) {
	if dw.wgpu != nil {
		dw.wgpu.Copy(dp, src, sr, op, unchanged)
		return
	}
	dw.base.Copy(dp, src, sr, op, unchanged)
}

func (dw *Drawer) Scale(dr image.Rectangle, src image.Image, sr image.Rectangle, rotateDeg float32, op draw.Op, unchanged bool) {
	if dw.wgpu != nil {
		dw.wgpu.Scale(dr, src, sr, rotateDeg, op, unchanged)
		return
	}
	dw.base.Scale(dr, src, sr, rotateDeg, op, unchanged)
}

func (dw *Drawer) Transform(xform math32.Matrix3, src image.Image, sr image.Rectangle, op draw.Op, unchanged bool) {
	if dw.wgpu != nil {
		dw.wgpu.Transform(xform, src, sr, op, unchanged)
		return
	}
	dw.base.Transform(xform, src, sr, op, unchanged)
}

func (dw *Drawer) Renderer() any {
	if dw.wgpu != nil {
		return dw.wgpu.Renderer()
	}
	return nil
}
