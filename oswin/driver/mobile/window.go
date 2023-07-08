// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android || ios

package mobile

import (
	"image"
	"log"
	"sync"
	"time"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/event"
	"github.com/goki/gi/oswin/window"
	"github.com/goki/ki/bitflag"
	"github.com/goki/vgpu/vdraw"
)

// TODO: actually implement things for mobile window

type windowImpl struct {
	oswin.WindowBase
	event.Deque
	app                *appImpl
	scrnName           string // last known screen name
	runQueue           chan funcRun
	publish            chan struct{}
	publishDone        chan struct{}
	winClose           chan struct{}
	mu                 sync.Mutex
	mainMenu           oswin.MainMenu
	closeReqFunc       func(win oswin.Window)
	closeCleanFunc     func(win oswin.Window)
	mouseDisabled      bool
	resettingPos       bool
	lastMouseButtonPos image.Point
	lastMouseEventPos  image.Point
	RenderSize         image.Point
	isVisible          bool
}

var _ oswin.Window = &windowImpl{}

// Handle returns the driver-specific handle for this window.
// Currently, for all platforms, this is *glfw.Window, but that
// cannot always be assumed.  Only provided for unforeseen emergency use --
// please file an Issue for anything that should be added to Window
// interface.
func (w *windowImpl) Handle() any {
	return w.app.winptr
}

func (w *windowImpl) OSHandle() uintptr {
	return w.app.winptr
}

func (w *windowImpl) MainMenu() oswin.MainMenu {
	return w.mainMenu
}

func (w *windowImpl) Drawer() *vdraw.Drawer {
	return &w.app.Draw
}

func (w *windowImpl) IsClosed() bool {
	return w.app.gpu == nil
}

func (w *windowImpl) IsVisible() bool {
	return w.isVisible && w.app.Surface != nil
}

func (w *windowImpl) Activate() bool {
	return true
}

func (w *windowImpl) DeActivate() {}

// for sending window.Event's
func (w *windowImpl) sendWindowEvent(act window.Actions) {
	winEv := window.Event{
		Action: act,
	}
	winEv.Init()
	log.Printf("Sent window event %#v\n", winEv)
	w.Send(&winEv)
}

// NextEvent implements the oswin.EventDeque interface.
func (w *windowImpl) NextEvent() oswin.Event {
	e := w.Deque.NextEvent()
	return e
}

// winLoop is the window's own locked processing loop.
func (w *windowImpl) winLoop() {
	winShow := time.NewTimer(time.Second)
outer:
	for {
		log.Println("mobile window loop iteration")
		select {
		case <-w.winClose:
			break outer
		case f := <-w.runQueue:
			if w.app.gpu == nil {
				break outer
			}
			f.f()
			if f.done != nil {
				f.done <- true
			}
		case <-winShow.C:
			if w.app.gpu == nil {
				break outer
			}
			w.sendWindowEvent(window.Show)
		}
	}
}

// RunOnWin runs given function on the window's unique locked thread.
func (w *windowImpl) RunOnWin(f func()) {
	if w.IsClosed() {
		return
	}
	done := make(chan bool)
	w.runQueue <- funcRun{f: f, done: done}
	<-done
}

// GoRunOnWin runs given function on window's unique locked thread and returns immediately
func (w *windowImpl) GoRunOnWin(f func()) {
	if w.IsClosed() {
		return
	}
	go func() {
		w.runQueue <- funcRun{f: f, done: nil}
	}()
}

// SendEmptyEvent sends an empty, blank event to this window, which just has
// the effect of pushing the system along during cases when the window
// event loop needs to be "pinged" to get things moving along..
func (w *windowImpl) SendEmptyEvent() {
	if w.IsClosed() {
		return
	}
	oswin.SendCustomEvent(w, nil)
}

func (w *windowImpl) Screen() *oswin.Screen {
	sc := w.app.screens[0]
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

func (w *windowImpl) Insets() gist.SideFloats {
	return gist.NewSideFloats(
		float32(w.app.sizeEvent.InsetTopPx),
		float32(w.app.sizeEvent.InsetRightPx),
		float32(w.app.sizeEvent.InsetBottomPx),
		float32(w.app.sizeEvent.InsetLeftPx),
	)
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

func (w *windowImpl) SetTitle(title string) {
	w.Titl = title
}

func (w *windowImpl) SetWinSize(sz image.Point) {
	w.WnSize = sz
}

func (w *windowImpl) SetSize(sz image.Point) {
	w.PxSize = sz
}

func (w *windowImpl) SetPos(pos image.Point) {
	w.Pos = pos
}

func (w *windowImpl) SetGeom(pos image.Point, sz image.Point) {
	w.Pos = pos
	w.PxSize = sz
}

func (w *windowImpl) show() {
	w.isVisible = true
}

func (w *windowImpl) Raise() {
	w.isVisible = true
}

func (w *windowImpl) Minimize() {
	w.isVisible = false
}

func (w *windowImpl) SetCloseReqFunc(fun func(win oswin.Window)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closeReqFunc = fun
}

func (w *windowImpl) SetCloseCleanFunc(fun func(win oswin.Window)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closeCleanFunc = fun
}

func (w *windowImpl) CloseReq() {
	if theApp.quitting {
		w.Close()
	}
	if w.closeReqFunc != nil {
		w.closeReqFunc(w)
	} else {
		w.Close()
	}
}

func (w *windowImpl) CloseClean() {
	if w.closeCleanFunc != nil {
		w.closeCleanFunc(w)
	}
}

func (w *windowImpl) Close() {
	// this is actually the final common pathway for closing here
	w.mu.Lock()
	defer w.mu.Unlock()
	w.winClose <- struct{}{} // break out of draw loop
	w.CloseClean()
	// fmt.Printf("sending close event to window: %v\n", w.Nm)
	w.sendWindowEvent(window.Close)
	theApp.DeleteWin(w)
	if theApp.quitting {
		theApp.quitCloseCnt <- struct{}{}
	}
}

func (w *windowImpl) SetMousePos(x, y float64) {}

func (w *windowImpl) SetCursorEnabled(enabled, raw bool) {}

/////////////////////////////////////////////////////////
//  Window Callbacks

// getScreenOvlp gets the monitor for given window
// based on overlap of geometry, using limited glfw 3.3 api,
// which does not provide this functionality.
// See: https://github.com/glfw/glfw/issues/1699
// This is adapted from slawrence2302's code posted there.
func (w *windowImpl) getScreenOvlp() *oswin.Screen {
	return w.app.screens[0]
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

func (w *windowImpl) focus(focused bool) {
	log.Println("in focus")
	w.mu.Lock()
	log.Println("past mutex")
	defer w.mu.Unlock()
	if focused {
		bitflag.SetAtomic(&w.Flag, int(oswin.Focus))
		w.sendWindowEvent(window.Focus)
	} else {
		bitflag.ClearAtomic(&w.Flag, int(oswin.Focus))
		w.sendWindowEvent(window.DeFocus)
	}
}

// func (w *windowImpl) iconify(gw *glfw.Window, iconified bool) {}
