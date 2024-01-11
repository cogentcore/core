// Copyright 2023 The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"image"
	"syscall/js"
	"unsafe"

	"goki.dev/goosi"
)

// Drawer is a TEMPORARY, low-performance implementation of [goosi.Drawer].
// It will be replaced with a full WebGPU based drawer at some point.
// TODO: replace Drawer with WebGPU
type Drawer struct {
	goosi.DrawerBase
}

// DestBounds returns the bounds of the render destination
func (dw *Drawer) DestBounds() image.Rectangle {
	return TheApp.Scrn.Geometry
}

// EndDraw ends image drawing rendering process on render target.
// This is the thing that actually does the drawing on web.
func (dw *Drawer) EndDraw() {
	sz := dw.Image.Bounds().Size()
	ptr := uintptr(unsafe.Pointer(&dw.Image.Pix[0]))
	js.Global().Call("displayImage", ptr, len(dw.Image.Pix), sz.X, sz.Y)
}
