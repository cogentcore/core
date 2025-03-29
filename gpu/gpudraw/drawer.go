// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpudraw

//go:generate core generate

import (
	"image"
	"image/draw"
	"sync"

	"cogentcore.org/core/colors"
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
	System *gpu.GraphicsSystem

	// opList is the list of drawing operations made on the current pass.
	// This is recorded after Start and executed at End.
	opList []draw.Op

	// lastOpN is the number of items in the last opList, which is
	// reset to 0, but can be recovered for Redraw.
	lastOpN int

	// images manages the list of images and their allocation
	// to Value indexes.
	images images

	// size of current image, set via Use* methods
	curImageSize image.Point

	// use Lock, Unlock on Drawer for all impl routines
	sync.Mutex
}

// AsGPUDrawer represents a type that can be used as a [Drawer].
type AsGPUDrawer interface {

	// AsGPUDrawer returns the drawer as a [Drawer].
	// It may return nil if it cannot be used as a [Drawer].
	AsGPUDrawer() *Drawer
}

// NewDrawer returns a new [Drawer] configured for rendering
// to given Renderer.
func NewDrawer(gp *gpu.GPU, rd gpu.Renderer) *Drawer {
	dw := &Drawer{}
	dw.configSystem(gp, rd)
	return dw
}

// AsGPUDrawer implements [AsGPUDrawer].
func (dw *Drawer) AsGPUDrawer() *Drawer {
	return dw
}

func (dw *Drawer) Release() {
	if dw.System == nil {
		return
	}
	dw.System.Release()
	dw.System = nil
}

func (dw *Drawer) Renderer() any {
	if dw.System == nil {
		return nil
	}
	return dw.System.Renderer
}

// DestSize returns the size of the render destination
func (dw *Drawer) DestSize() image.Point {
	return dw.System.Renderer.Render().Format.Size
}

// DestBounds returns the bounds of the render destination
func (dw *Drawer) DestBounds() image.Rectangle {
	return dw.System.Renderer.Render().Format.Bounds()
}

// Start starts recording a sequence of draw / fill actions,
// which will be performed on the GPU at End().
// This must be called prior to any Drawer operations.
func (dw *Drawer) Start() {
	dw.Lock()
	defer dw.Unlock()

	// always use the default background color for clearing in general
	dw.System.SetClearColor(colors.ToUniform(colors.Scheme.Background))

	dw.opList = dw.opList[:0]
	dw.images.start()
}

// End ends image drawing rendering process on render target.
func (dw *Drawer) End() {
	dw.Lock()
	defer dw.Unlock()
	dw.lastOpN = len(dw.opList)
	if dw.lastOpN == 0 {
		return
	}
	//	write up to GPU
	dw.drawAll()
	dw.opList = dw.opList[:0]
}

// Redraw re-renders the last draw
func (dw *Drawer) Redraw() {
	dw.Lock()
	defer dw.Unlock()
	if dw.lastOpN == 0 {
		return
	}
	dw.opList = dw.opList[:dw.lastOpN]
	dw.drawAll()
	dw.opList = dw.opList[:0]
}
