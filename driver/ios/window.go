// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ios

package ios

import (
	"fmt"
	"image"
	"time"

	"goki.dev/girl/styles"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
)

type windowImpl struct {
	goosi.WindowBase
	app                *appImpl
	scrnName           string // last known screen name
	runQueue           chan funcRun
	publish            chan struct{}
	publishDone        chan struct{}
	winClose           chan struct{}
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

func (w *windowImpl) Lock() bool {
	// we re-use app mu for window because the app actually controls the system window
	w.app.mu.Lock()
	if w.app.gpu == nil || w.app.Surface == nil {
		w.app.mu.Unlock()
		return false
	}
	return true
}

func (w *windowImpl) Unlock() {
	w.app.mu.Unlock()
}

func (w *windowImpl) Drawer() goosi.Drawer {
	return &w.app.Draw
}

func (w *windowImpl) IsClosed() bool {
	return w.app.gpu == nil || w.app.Surface == nil
}

func (w *windowImpl) IsVisible() bool {
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
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
	defer func() { handleRecover(recover()) }()
	fmt.Println("starting window loop")
	var winPaint *time.Ticker
	if w.FPS > 0 {
		winPaint = time.NewTicker(time.Second / time.Duration(w.FPS))
	} else {
		winPaint = &time.Ticker{C: make(chan time.Time)} // nop
	}
outer:
	for {
		select {
		case <-w.winClose:
			winPaint.Stop() // todo: close channel too??
			break outer
		case f := <-w.runQueue:
			if w.app.gpu == nil {
				break outer
			}
			f.f()
			if f.done != nil {
				f.done <- true
			}
		case <-winPaint.C:
			// the app is closed, so we are done
			if w.app.gpu == nil {
				break outer
			}
			// we don't have a surface, so we skip for
			// now, but we don't break the outer loop,
			// as we could come back later
			if w.app.Surface == nil {
				break
			}
			w.app.mu.Lock()
			w.EvMgr.WindowPaint()
			w.app.mu.Unlock()
			// NOTE: this is incredibly important; do not remove it
			select {
			case w.publishDone <- struct{}{}:
			default:
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
	w.EvMgr.Custom(nil)
}

func (w *windowImpl) Screen() *goosi.Screen {
	return w.app.screen
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
	return w.app.insets
}

func (w *windowImpl) PhysicalDPI() float32 {
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
	return w.PhysDPI
}

func (w *windowImpl) LogicalDPI() float32 {
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
	return w.LogDPI
}

func (w *windowImpl) SetLogicalDPI(dpi float32) {
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
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
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
	w.closeReqFunc = fun
}

func (w *windowImpl) SetCloseCleanFunc(fun func(win goosi.Window)) {
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
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
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
	w.winClose <- struct{}{} // break out of draw loop
	w.CloseClean()
	// fmt.Printf("sending close event to window: %v\n", w.Nm)
	w.EvMgr.Window(events.WinClose)
	theApp.DeleteWin(w)
	if theApp.quitting {
		theApp.quitCloseCnt <- struct{}{}
	}
}

func (w *windowImpl) SetMousePos(x, y float64) {
	// no-op
}

func (w *windowImpl) SetCursorEnabled(enabled, raw bool) {
	// no-op
}

func (w *windowImpl) IsCursorEnabled() bool {
	// no-op
	return false
}
