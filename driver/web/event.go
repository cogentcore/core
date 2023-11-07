// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"image"
	"syscall/js"

	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
)

func (app *appImpl) addEventListeners() {
	g := js.Global()
	g.Call("addEventListener", "mousedown", js.FuncOf(app.onMouseDown))
	g.Call("addEventListener", "touchstart", js.FuncOf(app.onTouchStart))
	g.Call("addEventListener", "mouseup", js.FuncOf(app.onMouseUp))
	g.Call("addEventListener", "touchend", js.FuncOf(app.onMouseUp))
	g.Call("addEventListener", "mousemove", js.FuncOf(app.onMouseMove))
	g.Call("addEventListener", "touchmove", js.FuncOf(app.onMouseMove))
	g.Call("addEventListener", "contextmenu", js.FuncOf(app.onContextMenu))
	g.Call("addEventListener", "keydown", js.FuncOf(app.onKeyDown))
	g.Call("addEventListener", "keyup", js.FuncOf(app.onKeyUp))
	g.Call("addEventListener", "resize", js.FuncOf(app.onResize))
}

func (app *appImpl) onMouseDown(this js.Value, args []js.Value) any {
	e := args[0]
	x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
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
	app.window.EvMgr.MouseButton(events.MouseDown, ebut, image.Pt(x, y), app.keyMods)
	e.Call("preventDefault")
	return nil
}

func (app *appImpl) onTouchStart(this js.Value, args []js.Value) any {
	e := args[0]
	touches := e.Get("changedTouches")
	for i := 0; i < touches.Length(); i++ {
		touch := touches.Index(i)
		x, y := touch.Get("clientX").Int(), touch.Get("clientY").Int()
		app.window.EvMgr.MouseButton(events.MouseDown, events.Left, image.Pt(x, y), 0)
	}
	e.Call("preventDefault")
	return nil
}

func (app *appImpl) onMouseUp(this js.Value, args []js.Value) any {
	e := args[0]
	x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
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
	app.window.EvMgr.MouseButton(events.MouseUp, ebut, image.Pt(x, y), app.keyMods)
	e.Call("preventDefault")
	return nil
}

func (app *appImpl) onMouseMove(this js.Value, args []js.Value) any {
	e := args[0]
	x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
	app.window.EvMgr.MouseMove(image.Pt(x, y))
	e.Call("preventDefault")
	return nil
}

func (app *appImpl) onContextMenu(this js.Value, args []js.Value) any {
	// no-op (we handle elsewhere), but needed to prevent browser
	// from making its own context menus on right clicks
	e := args[0]
	e.Call("preventDefault")
	return nil
}

// down is whether this is a keyDown event (instead of a keyUp one)
func (app *appImpl) runeAndCodeFromKey(k string, down bool) (rune, key.Codes) {
	switch k {
	case "Shift":
		app.keyMods.SetFlag(down, key.Shift)
		return 0, key.CodeLeftShift
	case "Control":
		app.keyMods.SetFlag(down, key.Control)
		return 0, key.CodeLeftControl
	case "Alt":
		app.keyMods.SetFlag(down, key.Alt)
		return 0, key.CodeLeftAlt
	case "Meta":
		app.keyMods.SetFlag(down, key.Meta)
		return 0, key.CodeLeftMeta
	case "Backspace":
		return 0, key.CodeDeleteBackspace
	case "Delete":
		return 0, key.CodeDeleteForward
	case "Enter":
		return 0, key.CodeReturnEnter
	case "Tab":
		return 0, key.CodeTab
	case "ArrowDown":
		return 0, key.CodeDownArrow
	case "ArrowLeft":
		return 0, key.CodeLeftArrow
	case "ArrowRight":
		return 0, key.CodeRightArrow
	case "ArrowUp":
		return 0, key.CodeUpArrow
	case "Spacebar":
		return ' ', 0
	default:
		return []rune(k)[0], 0
	}
}

func (app *appImpl) onKeyDown(this js.Value, args []js.Value) any {
	e := args[0]
	key := e.Get("key")
	r, c := app.runeAndCodeFromKey(key.String(), true)
	app.window.EvMgr.Key(events.KeyDown, r, c, app.keyMods)
	e.Call("preventDefault")
	return nil
}

func (app *appImpl) onKeyUp(this js.Value, args []js.Value) any {
	e := args[0]
	key := e.Get("key")
	r, c := app.runeAndCodeFromKey(key.String(), false)
	app.window.EvMgr.Key(events.KeyUp, r, c, app.keyMods)
	e.Call("preventDefault")
	return nil
}

func (app *appImpl) onResize(this js.Value, args []js.Value) any {
	app.resize()
	return nil
}
