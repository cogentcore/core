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
func (app *App) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	defer func() { base.HandleRecover(recover()) }()

	app.Win = &Window{base.NewWindowSingle(app, opts)}
	app.Win.EvMgr.Deque = &app.Win.Deque
	app.SetSysWindow()

	go app.Win.WinLoop()

	return app.Win, nil
}

// SetSysWindow sets the underlying system window information.
func (app *App) SetSysWindow() {
	defer func() { base.HandleRecover(recover()) }()

	ua := js.Global().Get("navigator").Get("userAgent").String()
	lua := strings.ToLower(ua)
	if strings.Contains(lua, "android") {
		app.platform = goosi.Android
	} else if strings.Contains(lua, "ipad") || strings.Contains(lua, "iphone") || strings.Contains(lua, "ipod") {
		app.platform = goosi.IOS
	} else {
		// TODO(kai/web): more specific desktop platform
		app.platform = goosi.Windows
	}

	app.Resize()
	app.Win.EvMgr.Window(events.WinShow)
	app.Win.EvMgr.Window(events.ScreenUpdate)
	app.Win.EvMgr.Window(events.WinFocus)
}

// Resize updates the app sizing information and sends a Resize event.
func (app *App) Resize() {
	app.Scrn.DevicePixelRatio = float32(js.Global().Get("devicePixelRatio").Float())
	dpi := 160 * app.Scrn.DevicePixelRatio
	app.Scrn.PhysicalDPI = dpi
	app.Scrn.LogicalDPI = dpi

	w, h := js.Global().Get("screen").Get("innerWidth").Int(), js.Global().Get("screen").Get("innerHeight").Int()
	sz := image.Pt(w, h)
	app.Scrn.Geometry.Max = sz
	app.Scrn.PixSize = image.Pt(int(float32(sz.X)*app.Scrn.DevicePixelRatio), int(float32(sz.Y)*app.Scrn.DevicePixelRatio))
	physX := 25.4 * float32(w) / dpi
	physY := 25.4 * float32(h) / dpi
	app.Scrn.PhysicalSize = image.Pt(int(physX), int(physY))

	// ww, wh := js.Global().Get("innerWidth").Int(), js.Global().Get("innerHeight").Int()
	// wsz := image.Pt(ww, wh)
	// app.window.WnSize = wsz
	// app.window.PxSize = image.Pt(int(float32(wsz.X)*app.screen.DevicePixelRatio), int(float32(wsz.Y)*app.screen.DevicePixelRatio))
	// app.window.RenderSize = app.window.PxSize

	canvas := js.Global().Get("document").Call("getElementById", "app")
	canvas.Set("width", app.Scrn.PixSize.X)
	canvas.Set("height", app.Scrn.PixSize.Y)

	app.Win.EvMgr.WindowResize()
}

func (app *App) PrefsDir() string {
	// TODO(kai): implement web filesystem
	return "/data/data"
}

func (app *App) Platform() goosi.Platforms {
	return goosi.Web
}

func (app *App) OpenURL(url string) {
	js.Global().Call("open", url)
}

func (app *App) ClipBoard(win goosi.Window) clip.Board {
	return &TheClip
}

func (app *App) Cursor(win goosi.Window) cursor.Cursor {
	return &TheCursor
}

func (app *App) IsDark() bool {
	return js.Global().Get("matchMedia").Truthy() &&
		js.Global().Call("matchMedia", "(prefers-color-scheme: dark)").Get("matches").Truthy()
}

func (app *App) ShowVirtualKeyboard(typ goosi.VirtualKeyboardTypes) {
	js.Global().Get("document").Call("getElementById", "text-field").Call("focus")
}

func (app *App) HideVirtualKeyboard() {
	js.Global().Get("document").Call("getElementById", "text-field").Call("blur")
}
