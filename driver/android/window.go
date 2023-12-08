// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

package android

import (
	"image"

	"goki.dev/girl/styles"
	"goki.dev/goosi"
	"goki.dev/goosi/driver/base"
	"goki.dev/goosi/events"
)

// Window is the implementation of [goosi.Window] for the Android platform.
type Window struct {
	base.WindowSingle[*App]
}

func (w *Window) Handle() any {
	return w.App.Winptr
}

// RunOnWin runs given function on the window's unique locked thread.
func (w *Window) RunOnWin(f func()) {
	if w.IsClosed() {
		return
	}
	done := make(chan bool)
	w.runQueue <- funcRun{f: f, done: done}
	<-done
}

// GoRunOnWin runs given function on window's unique locked thread and returns immediately
func (w *Window) GoRunOnWin(f func()) {
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
func (w *Window) SendEmptyEvent() {
	if w.IsClosed() {
		return
	}
	w.EvMgr.Custom(nil)
}

func (w *Window) Screen() *goosi.Screen {
	return w.app.screen
}

func (w *Window) Size() image.Point {
	return w.PxSize
}

func (w *Window) WinSize() image.Point {
	return w.WnSize
}

func (w *Window) Position() image.Point {
	return image.Point{} // always true
}

func (w *Window) Insets() styles.SideFloats {
	return w.app.insets
}

func (w *Window) PhysicalDPI() float32 {
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
	return w.PhysDPI
}

func (w *Window) LogicalDPI() float32 {
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
	return w.LogDPI
}

func (w *Window) SetLogicalDPI(dpi float32) {
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
	w.LogDPI = dpi
}

func (w *Window) SetTitle(title string) {
	w.Titl = title
}

func (w *Window) SetWinSize(sz image.Point) {
	w.WnSize = sz
}

func (w *Window) SetSize(sz image.Point) {
	w.PxSize = sz
}

func (w *Window) SetPos(pos image.Point) {
	w.Pos = pos
}

func (w *Window) SetGeom(pos image.Point, sz image.Point) {
	w.Pos = pos
	w.PxSize = sz
}

func (w *Window) show() {
	// TODO: implement?
	w.isVisible = true
}

func (w *Window) Raise() {
	// TODO: implement?
	w.isVisible = true
}

func (w *Window) Minimize() {
	// TODO: implement?
	w.isVisible = false
}

func (w *Window) SetCloseReqFunc(fun func(win goosi.Window)) {
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
	w.closeReqFunc = fun
}

func (w *Window) SetCloseCleanFunc(fun func(win goosi.Window)) {
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
	w.closeCleanFunc = fun
}

func (w *Window) CloseReq() {
	if TheApp.quitting {
		w.Close()
	}
	if w.closeReqFunc != nil {
		w.closeReqFunc(w)
	} else {
		w.Close()
	}
}

func (w *Window) CloseClean() {
	if w.closeCleanFunc != nil {
		w.closeCleanFunc(w)
	}
}

func (w *Window) Close() {
	// this is actually the final common pathway for closing here
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
	w.winClose <- struct{}{} // break out of draw loop
	w.CloseClean()
	// fmt.Printf("sending close event to window: %v\n", w.Nm)
	w.EvMgr.Window(events.WinClose)
	TheApp.DeleteWin(w)
	if TheApp.quitting {
		TheApp.quitCloseCnt <- struct{}{}
	}
}

func (w *Window) SetMousePos(x, y float64) {
	// no-op
}

func (w *Window) SetCursorEnabled(enabled, raw bool) {
	// no-op
}

func (w *Window) IsCursorEnabled() bool {
	// no-op
	return false
}

func (w *Window) SetTitleBarIsDark(isDark bool) {
	// no-op
}
