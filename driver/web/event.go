// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"image"
	"syscall/js"

	"goki.dev/goosi/events"
)

func (app *appImpl) addEventListeners() {
	g := js.Global()
	g.Call("addEventListener", "mousedown", js.FuncOf(app.onMouseDown))
	g.Call("addEventListener", "mouseup", js.FuncOf(app.onMouseUp))
	g.Call("addEventListener", "mousemove", js.FuncOf(app.onMouseMove))
}

func (app *appImpl) onMouseDown(this js.Value, args []js.Value) any {
	e := args[0]
	x, y := e.Get("clientX").Int(), args[0].Get("clientY").Int()
	but := e.Get("button").Int()
	var ebut events.Buttons
	switch but {
	case 0:
		ebut = events.Left
	case 1:
		ebut = events.Middle
	case 2:
		ebut = events.Right
	}
	app.window.EvMgr.MouseButton(events.MouseDown, ebut, image.Pt(x, y), 0) // TODO(kai/web): modifiers
	e.Call("preventDefault")
	return nil
}

func (app *appImpl) onMouseUp(this js.Value, args []js.Value) any {
	e := args[0]
	x, y := e.Get("clientX").Int(), args[0].Get("clientY").Int()
	but := e.Get("button").Int()
	var ebut events.Buttons
	switch but {
	case 0:
		ebut = events.Left
	case 1:
		ebut = events.Middle
	case 2:
		ebut = events.Right
	}
	app.window.EvMgr.MouseButton(events.MouseUp, ebut, image.Pt(x, y), 0) // TODO(kai/web): modifiers
	e.Call("preventDefault")
	return nil
}

func (app *appImpl) onMouseMove(this js.Value, args []js.Value) any {
	e := args[0]
	x, y := e.Get("clientX").Int(), args[0].Get("clientY").Int()
	app.window.EvMgr.MouseMove(image.Pt(x, y))
	e.Call("preventDefault")
	return nil
}
