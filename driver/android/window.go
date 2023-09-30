// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

package android

import (
	"image"
	"sync"
	"time"

	"goki.dev/girl/styles"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/vgpu/v2/vdraw"
)

// TODO: actually implement things for mobile window

type windowImpl struct {
	goosi.WindowBase
	events.Deque
	app                *appImpl
	EventMgr           events.Mgr
	scrnName           string // last known screen name
	runQueue           chan funcRun
	publish            chan struct{}
	publishDone        chan struct{}
	winClose           chan struct{}
	mu                 sync.Mutex
	mainMenu           goosi.MainMenu
	closeReqFunc       func(win goosi.Window)
	closeCleanFunc     func(win goosi.Window)
	mouseDisabled      bool
	resettingPos       bool
	lastMouseButtonPos image.Point
	lastMouseEventPos  image.Point
	RenderSize         image.Point
	isVisible          bool
}

var _ goosi.Window = &windowImpl{}

func (w *windowImpl) Handle() any {
	return w.app.winptr
}

func (w *windowImpl) OSHandle() uintptr {
	return w.app.winptr
}

func (w *windowImpl) MainMenu() goosi.MainMenu {
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
	// TODO: implement?
	return true
}

func (w *windowImpl) DeActivate() {
	// TODO: implement?
}

// NextEvent implements the events.EventDeque interface.
func (w *windowImpl) NextEvent() events.Event {
	e := w.Deque.NextEvent()
	return e
}

// winLoop is the window's own locked processing loop.
func (w *windowImpl) winLoop() {
	winShow := time.NewTimer(time.Second / 2) // this is a backup to ensure shown eventually..
	var winPaint *time.Ticker
	if w.FPS > 0 {
		winPaint = time.NewTicker(time.Second / time.Duration(w.FPS))
	} else {
		winPaint = &time.Ticker{C: make(chan time.Time)} // nop
	}
	hasShown := false
outer:
	for {
		select {
		case <-w.winClose:
			winShow.Stop() // todo: close channel too??
			winPaint.Stop()
			hasShown = false
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
			w.EventMgr.Window(events.Show)
			hasShown = true
		case <-winPaint.C:
			if w.app.gpu == nil {
				break outer
			}
			if hasShown {
				w.EventMgr.WindowPaint()
			}
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
	w.EventMgr.Custom(nil)
}

func (w *windowImpl) Screen() *goosi.Screen {
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
	return image.Point{} // always true
}

func (w *windowImpl) Insets() styles.SideFloats {
	return styles.NewSideFloats(
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
	// TODO: implement?
	w.isVisible = true
}

func (w *windowImpl) Raise() {
	// TODO: implement?
	w.isVisible = true
}

func (w *windowImpl) Minimize() {
	// TODO: implement?
	w.isVisible = false
}

func (w *windowImpl) SetCloseReqFunc(fun func(win goosi.Window)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closeReqFunc = fun
}

func (w *windowImpl) SetCloseCleanFunc(fun func(win goosi.Window)) {
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
	w.EventMgr.Window(events.Close)
	theApp.DeleteWin(w)
	if theApp.quitting {
		theApp.quitCloseCnt <- struct{}{}
	}
}

func (w *windowImpl) SetMousePos(x, y float64) {}

func (w *windowImpl) SetCursorEnabled(enabled, raw bool) {}
