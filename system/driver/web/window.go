// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"image"
	"syscall/js"

	"cogentcore.org/core/system"
	"cogentcore.org/core/system/driver/base"
)

// Window is the implementation of [system.Window] for the web platform.
type Window struct {
	base.WindowSingle[*App]
}

// WinLoop is a web-specific implementation using requestAnimationFrame.
func (w *Window) WinLoop() {
	defer func() { system.HandleRecover(recover()) }()

	var f js.Func
	f = js.FuncOf(func(this js.Value, args []js.Value) any {
		w.SendPaintEvent()
		js.Global().Call("requestAnimationFrame", f)
		return nil
	})
	js.Global().Call("requestAnimationFrame", f)
}

func (w *Window) SetTitle(title string) {
	w.WindowSingle.SetTitle(title)
	js.Global().Get("document").Set("title", title)
}

func (w *Window) SetGeometry(fullscreen bool, pos image.Point, size image.Point, screen *system.Screen) {
	w.WindowSingle.SetGeometry(fullscreen, pos, size, screen)
	// We only support fullscreen, not pos, size, or screen.
	doc := js.Global().Get("document")
	if fullscreen {
		doc.Get("documentElement").Call("requestFullscreen")
		return
	}
	doc.Call("exitFullscreen")
}
