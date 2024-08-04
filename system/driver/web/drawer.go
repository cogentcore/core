// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"image"
	"image/draw"
	"syscall/js"
	"unsafe"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/gpudraw"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/system"
)

// Drawer implements [system.Drawer] with a WebGPU-based drawer if available
// and a backup 2D image drawer if not.
type Drawer struct {
	// wgpu is the WebGPU-based drawer if available.
	wgpu *gpudraw.Drawer

	// base is used for the backup 2D image drawer.
	base system.DrawerBase
}

// InitDrawer sets the [Drawer] to a WebGPU-based drawer
// if the browser supports WebGPU.
func (a *App) InitDrawer() {
	if !js.Global().Get("navigator").Get("gpu").Truthy() {
		return
	}
	gp := gpu.NewGPU()
	gp.Config(a.Name())
	surf := gp.Instance.CreateSurface(nil)
	sf := gpu.NewSurface(gp, surf, a.Scrn.PixSize.X, a.Scrn.PixSize.Y)
	a.Draw.wgpu = gpudraw.NewDrawerSurface(sf)
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
		ptr := uintptr(unsafe.Pointer(&dw.base.Image.Pix[0]))
		js.Global().Call("displayImage", ptr, len(dw.base.Image.Pix), sz.X, sz.Y)
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

func (dw *Drawer) Surface() any {
	if dw.wgpu != nil {
		return dw.wgpu.Surface()
	}
	return nil
}
