// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"goki.dev/goosi"
	"goki.dev/goosi/driver/base"
	"goki.dev/goosi/events"
)

// Window is the implementation of [goosi.Window] on the offscreen platform.
type Window struct {
	base.WindowSingle[*App]
}

var _ goosi.Window = &Window{}

func (w *Window) Handle() any {
	return nil
}

func (w *Window) OSHandle() uintptr {
	return 0
}

func (w *Window) MainMenu() goosi.MainMenu {
	return w.mainMenu
}

func (w *Window) Lock() bool {
	// we re-use app mu for window because the app actually controls the system window
	w.app.mu.Lock()
	// if w.app.gpu == nil || w.app.Surface == nil {
	// 	w.app.mu.Unlock()
	// 	return false
	// }
	return true
}

func (w *Window) Unlock() {
	w.app.mu.Unlock()
}

func (w *Window) IsClosed() bool {
	return false
	// return w.app.gpu == nil || w.app.Surface == nil
}

func (w *Window) IsVisible() bool {
	w.app.mu.Lock()
	defer w.app.mu.Unlock()
	return true
	// return w.isVisible && w.app.Surface != nil
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

func (w *Window) Close() {
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
