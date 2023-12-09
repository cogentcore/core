// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

package android

import (
	"goki.dev/goosi/driver/base"
)

// Window is the implementation of [goosi.Window] for the Android platform.
type Window struct {
	base.WindowSingle[*App]
}

func (w *Window) Handle() any {
	defer func() { base.HandleRecover(recover()) }()

	return w.App.Winptr
}

// // WinLoop runs the window's own locked processing loop.
// func (w *Window) WinLoop() {
// 	var winPaint *time.Ticker
// 	if w.FPS > 0 {
// 		winPaint = time.NewTicker(time.Second / time.Duration(w.FPS))
// 	} else {
// 		winPaint = &time.Ticker{C: make(chan time.Time)} // nop
// 	}
// 	winShow := time.NewTimer(200 * time.Millisecond)
// outer:
// 	for {
// 		select {
// 		case <-w.WinClose:
// 			winPaint.Stop()
// 			break outer
// 		case <-winShow.C:
// 			if !w.This.IsVisible() {
// 				break outer
// 			}
// 			w.EvMgr.Window(events.WinShow)
// 		case f := <-w.RunQueue:
// 			if !w.This.IsVisible() {
// 				break outer
// 			}
// 			f.F()
// 			if f.Done != nil {
// 				f.Done <- struct{}{}
// 			}
// 		case <-winPaint.C:
// 			if !w.This.IsVisible() {
// 				break outer
// 			}
// 			w.EvMgr.WindowPaint()
// 			// NOTE: this is incredibly important; do not remove it (see [onNativeWindowRedrawNeeded] for why)
// 			select {
// 			case windowRedrawDone <- struct{}{}:
// 			default:
// 			}
// 		}
// 	}
// }
