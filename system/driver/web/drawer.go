// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"image"
	"syscall/js"
	"unsafe"

	"cogentcore.org/core/system"
)

// Drawer is a TEMPORARY, low-performance implementation of [system.Drawer].
// It will be replaced with a full WebGPU based drawer at some point.
// TODO: replace Drawer with WebGPU
type Drawer struct {
	system.DrawerBase
}

// DestBounds returns the bounds of the render destination
func (dw *Drawer) DestBounds() image.Rectangle {
	return TheApp.Scrn.Geometry
}

var loader = js.Global().Get("document").Call("getElementById", "app-wasm-loader")

// EndDraw ends image drawing rendering process on render target.
// This is the thing that actually does the drawing on web.
func (dw *Drawer) EndDraw() {
	sz := dw.Image.Bounds().Size()
	ptr := uintptr(unsafe.Pointer(&dw.Image.Pix[0]))
	js.Global().Call("displayImage", ptr, len(dw.Image.Pix), sz.X, sz.Y)
	if loader.Truthy() {
		loader.Call("remove")
		loader = js.Value{}
	}
}
