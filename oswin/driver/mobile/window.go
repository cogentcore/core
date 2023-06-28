// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package mobile

import (
	"image"
	"sync"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/event"
	"github.com/goki/mobile/gl"
	"github.com/goki/vgpu/vdraw"
	"github.com/goki/vgpu/vgpu"
)

// TODO: actually implement things for mobile window

type windowImpl struct {
	oswin.WindowBase
	event.Deque
	// app            *appImpl
	ctx      gl.Context
	Surface  *vgpu.Surface
	Draw     vdraw.Drawer
	scrnName string // last known screen name
	// runQueue       chan funcRun
	publish        chan struct{}
	publishDone    chan struct{}
	winClose       chan struct{}
	mu             sync.Mutex
	mainMenu       oswin.MainMenu
	closeReqFunc   func(win oswin.Window)
	closeCleanFunc func(win oswin.Window)
	mouseDisabled  bool
	resettingPos   bool
}

var _ oswin.Window = &windowImpl{}

// Handle returns the driver-specific handle for this window.
// Currently, for all platforms, this is *glfw.Window, but that
// cannot always be assumed.  Only provided for unforeseen emergency use --
// please file an Issue for anything that should be added to Window
// interface.
func (w *windowImpl) Handle() any {
	return nil
}

func (w *windowImpl) OSHandle() uintptr {
	return 0
}

func (w *windowImpl) MainMenu() oswin.MainMenu {
	return nil
}

func (w *windowImpl) Drawer() *vdraw.Drawer {
	return nil
}

func (w *windowImpl) IsClosed() bool {
	if w == nil {
		return true
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ctx == nil
}

func (w *windowImpl) IsVisible() bool {
	return w.IsClosed()
}

func (w *windowImpl) Activate() bool {
	return true
}

func (w *windowImpl) DeActivate() {}

// NextEvent implements the oswin.EventDeque interface.
func (w *windowImpl) NextEvent() oswin.Event {
	e := w.Deque.NextEvent()
	return e
}

// RunOnWin runs given function on the window's unique locked thread.
func (w *windowImpl) RunOnWin(f func()) {}

// GoRunOnWin runs given function on window's unique locked thread and returns immediately
func (w *windowImpl) GoRunOnWin(f func()) {}

// SendEmptyEvent sends an empty, blank event to this window, which just has
// the effect of pushing the system along during cases when the window
// event loop needs to be "pinged" to get things moving along..
func (w *windowImpl) SendEmptyEvent() {}

func (w *windowImpl) Screen() *oswin.Screen {
	sc := w.getScreen()
	return sc
}

func (w *windowImpl) Size() image.Point {
	return w.PxSize
}

func (w *windowImpl) WinSize() image.Point {
	return w.WnSize
}

func (w *windowImpl) Position() image.Point {
	return image.Point{}
}
func (w *windowImpl) PhysicalDPI() float32 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.PhysDPI
}

func (w *windowImpl) LogicalDPI() float32 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.LogDPI
}

func (w *windowImpl) SetLogicalDPI(dpi float32) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.LogDPI = dpi
}

func (w *windowImpl) SetTitle(title string) {}

func (w *windowImpl) SetWinSize(sz image.Point) {}

func (w *windowImpl) SetSize(sz image.Point) {}

func (w *windowImpl) SetPos(pos image.Point) {}

func (w *windowImpl) SetGeom(pos image.Point, sz image.Point) {}

func (w *windowImpl) show() {}

func (w *windowImpl) Raise() {}

func (w *windowImpl) Minimize() {}

func (w *windowImpl) SetCloseReqFunc(fun func(win oswin.Window)) {}

func (w *windowImpl) SetCloseCleanFunc(fun func(win oswin.Window)) {}

func (w *windowImpl) CloseReq() {}

func (w *windowImpl) CloseClean() {}

func (w *windowImpl) Close() {}

func (w *windowImpl) SetMousePos(x, y float64) {}

func (w *windowImpl) SetCursorEnabled(enabled, raw bool) {}

/////////////////////////////////////////////////////////
//  Window Callbacks

func (w *windowImpl) getScreen() *oswin.Screen {
	return nil
}

// getScreenOvlp gets the monitor for given window
// based on overlap of geometry, using limited glfw 3.3 api,
// which does not provide this functionality.
// See: https://github.com/glfw/glfw/issues/1699
// This is adapted from slawrence2302's code posted there.
func (w *windowImpl) getScreenOvlp() *oswin.Screen {
	return nil
}

// func (w *windowImpl) moved(gw *glfw.Window, x, y int) {}

// func (w *windowImpl) winResized(gw *glfw.Window, width, height int) {}

// func (w *windowImpl) updtGeom() {}

// func (w *windowImpl) fbResized(gw *glfw.Window, width, height int) {
// 	fbsz := image.Point{width, height}
// 	if w.PxSize != fbsz {
// 		w.updtGeom()
// 	}
// }

// func (w *windowImpl) closeReq(gw *glfw.Window) {
// 	go w.CloseReq()
// }

// func (w *windowImpl) refresh(gw *glfw.Window) {}

// func (w *windowImpl) focus(gw *glfw.Window, focused bool) {}

// func (w *windowImpl) iconify(gw *glfw.Window, iconified bool) {}
