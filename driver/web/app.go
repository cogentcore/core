// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

// Package web implements goosi interfaces on the web through WASM
package web

import (
	"image"
	"strings"
	"syscall/js"

	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/driver/base"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
)

// TheApp is the single [goosi.App] for the web platform
var TheApp = &App{AppSingle: base.NewAppSingle[*Drawer, *Window]()}

var _ goosi.App = TheApp

// App is the [goosi.App] implementation on the web platform
type App struct {
	base.AppSingle[*Drawer, *Window]

	// Platform is the underlying system platform (Android, iOS, etc)
	platform goosi.Platforms

	// KeyMods are the current key mods
	keyMods key.Modifiers
}

// Main is called from main thread when it is time to start running the
// main loop.  When function f returns, the app ends automatically.
func Main(f func(goosi.App)) {
	defer func() { base.HandleRecover(recover()) }()
	TheApp.AddEventListeners()
	goosi.TheApp = TheApp
	go func() {
		f(TheApp)
		TheApp.StopMain()
	}()
	TheApp.MainLoop()
}

// NewWindow creates a new window with the given options.
// It waits for the underlying system window to be created first.
// Also, it hides all other windows and shows the new one.
func (a *App) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	defer func() { base.HandleRecover(recover()) }()

	a.Win = &Window{base.NewWindowSingle(a, opts)}
	a.Win.EvMgr.Deque = &a.Win.Deque
	a.SetSysWindow()

	go a.Win.WinLoop()

	return a.Win, nil
}

// SetSysWindow sets the underlying system window information.
func (a *App) SetSysWindow() {
	defer func() { base.HandleRecover(recover()) }()

	ua := js.Global().Get("navigator").Get("userAgent").String()
	lua := strings.ToLower(ua)
	if strings.Contains(lua, "android") {
		a.platform = goosi.Android
	} else if strings.Contains(lua, "ipad") || strings.Contains(lua, "iphone") || strings.Contains(lua, "ipod") {
		a.platform = goosi.IOS
	} else {
		// TODO(kai/web): more specific desktop platform
		a.platform = goosi.Windows
	}

	a.Resize()
	a.Win.EvMgr.Window(events.WinShow)
	a.Win.EvMgr.Window(events.ScreenUpdate)
	a.Win.EvMgr.Window(events.WinFocus)
}

// Resize updates the app sizing information and sends a Resize event.
func (a *App) Resize() {
	a.Scrn.DevicePixelRatio = float32(js.Global().Get("devicePixelRatio").Float())
	dpi := 160 * a.Scrn.DevicePixelRatio
	a.Scrn.PhysicalDPI = dpi
	a.Scrn.LogicalDPI = dpi

	w, h := js.Global().Get("screen").Get("innerWidth").Int(), js.Global().Get("screen").Get("innerHeight").Int()
	sz := image.Pt(w, h)
	a.Scrn.Geometry.Max = sz
	a.Scrn.PixSize = image.Pt(int(float32(sz.X)*a.Scrn.DevicePixelRatio), int(float32(sz.Y)*a.Scrn.DevicePixelRatio))
	physX := 25.4 * float32(w) / dpi
	physY := 25.4 * float32(h) / dpi
	a.Scrn.PhysicalSize = image.Pt(int(physX), int(physY))

	// ww, wh := js.Global().Get("innerWidth").Int(), js.Global().Get("innerHeight").Int()
	// wsz := image.Pt(ww, wh)
	// app.window.WnSize = wsz
	// app.window.PxSize = image.Pt(int(float32(wsz.X)*app.screen.DevicePixelRatio), int(float32(wsz.Y)*app.screen.DevicePixelRatio))
	// app.window.RenderSize = app.window.PxSize

	canvas := js.Global().Get("document").Call("getElementById", "app")
	canvas.Set("width", a.Scrn.PixSize.X)
	canvas.Set("height", a.Scrn.PixSize.Y)

	a.Win.EvMgr.WindowResize()
}

func (a *App) PrefsDir() string {
	// TODO(kai): implement web filesystem
	return "/data/data"
}

func (a *App) Platform() goosi.Platforms {
	return goosi.Web
}

func (a *App) OpenURL(url string) {
	js.Global().Call("open", url)
}

func (a *App) ClipBoard(win goosi.Window) clip.Board {
	return &TheClip
}

func (a *App) Cursor(win goosi.Window) cursor.Cursor {
	return &TheCursor
}

func (a *App) IsDark() bool {
	return js.Global().Get("matchMedia").Truthy() &&
		js.Global().Call("matchMedia", "(prefers-color-scheme: dark)").Get("matches").Truthy()
}

func (a *App) ShowVirtualKeyboard(typ goosi.VirtualKeyboardTypes) {
	js.Global().Get("document").Call("getElementById", "text-field").Call("focus")
}

func (a *App) HideVirtualKeyboard() {
	js.Global().Get("document").Call("getElementById", "text-field").Call("blur")
}
