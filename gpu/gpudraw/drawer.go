// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpudraw

//go:generate core generate

import (
	"image"
	"image/draw"
	"sync"

	"cogentcore.org/core/gpu"
)

// These draw.Op constants are provided so that users of this package don't
// have to explicitly import "image/draw".  We also add the fill operations.
const (
	// Over = alpha blend with existing content.
	Over = draw.Over

	// Src = copy source to destination with no blending ("blit").
	Src = draw.Src

	// fillOver is used internally for the fill version of Over.
	fillOver = draw.Src + 1

	// fillSrc is used internally for the fill version of Src.
	fillSrc = draw.Src + 2
)

// AllocChunk is number of images / matrix elements to allocate
// at a time, to reduce number of reallocations.
// Should be set to the rough scale of number of items expected.
var AllocChunk = 16

// Drawer is the overall GPUDraw implementation, which draws Textures
// or Fills solid colors to a render target.
// A sequence of drawing operations is programmed for each render pass,
// between Start and End calls, which is then uploaded and performed
// in one GPU render pass, according to the recorded order of operations.
type Drawer struct {
	// drawing system
	Sys *gpu.System

	// surface if render target
	surface *gpu.Surface

	// render frame if render target
	Frame *gpu.RenderFrame

	// opList is the list of drawing operations made on the current pass.
	// This is recorded after Start and executed at End.
	opList []draw.Op

	// images manages the list of images and their allocation
	// to Value indexes.
	images images

	// size of current image, set via Use* methods
	curImageSize image.Point

	// use Lock, Unlock on Drawer for all impl routines
	sync.Mutex
}

// NewDrawerSurface returns a new Drawer configured for rendering
// to given Surface.
func NewDrawerSurface(sf *gpu.Surface) *Drawer {
	dw := &Drawer{}
	dw.ConfigSurface(sf)
	return dw
}

// NewDrawerFrame returns a new Drawer configured for rendering
// to a RenderFrame of given size.
// Uses given Device if non-nil; otherwise a new Device is made.
func NewDrawerFrame(dev *gpu.Device, size image.Point) *Drawer {
	dw := &Drawer{}
	dw.ConfigFrame(dev, size)
	return dw
}

// ConfigSurface configures the Drawer to use given surface as a render target.
func (dw *Drawer) ConfigSurface(sf *gpu.Surface) {
	dw.surface = sf
	dw.configSystem(sf.GPU, sf.Device, &sf.Format)
}

// ConfigFrame configures the Drawer to use a RenderFrame as a render target,
// of given size.  Use dw.Frame.SetSize to resize later.
// Frame is owned and managed by the Drawer.
// Uses given Device if non-nil; otherwise a new Device is made.
func (dw *Drawer) ConfigFrame(dev *gpu.Device, size image.Point) {
	// dw.Frame = gpu.NewRenderFrame(dw.Sys.GPU, dev, size)
	// dw.configSystem(sf.GPU, sf.Device, &sf.Format)
}

func (dw *Drawer) Release() {
	if dw.Sys == nil {
		return
	}
	dw.Sys.Release()
	dw.Sys = nil
	if dw.Frame != nil {
		dw.Frame.Release()
		dw.Frame = nil
	}
}

// DestSize returns the size of the render destination
func (dw *Drawer) DestSize() image.Point {
	if dw.surface != nil {
		return dw.surface.Format.Size
	} else {
		return dw.Frame.Format.Size
	}
}

// DestBounds returns the bounds of the render destination
func (dw *Drawer) DestBounds() image.Rectangle {
	if dw.surface != nil {
		return dw.surface.Format.Bounds()
	} else {
		return dw.Frame.Format.Bounds()
	}
}

func (dw *Drawer) Surface() any {
	return dw.surface
}

// Start starts recording a sequence of draw / fill actions,
// which will be performed on the GPU at End().
// This must be called prior to any Drawer operations.
func (dw *Drawer) Start() {
	dw.Lock()
	defer dw.Unlock()

	dw.opList = dw.opList[:0]
	dw.images.start()
}

// End ends image drawing rendering process on render target.
func (dw *Drawer) End() {
	if len(dw.opList) == 0 {
		return
	}
	dw.Lock()
	defer dw.Unlock()
	//	write up to GPU
	dw.drawAll()

	dw.opList = dw.opList[:0]
}
