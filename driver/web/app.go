// Copyright 2023 The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

// Package web implements goosi interfaces on the web through WASM
package web

//go:generate goki generate

import (
	"image"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"syscall/js"

	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/driver/base"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/jsfs"
)

func Init() {
	TheApp.Drawer = &Drawer{}

	err := os.Setenv("HOME", "/home/me")
	if err != nil {
		slog.Error("error setting home directory", "err", err)
	}

	fs, err := jsfs.Config(js.Global().Get("fs"))
	if err != nil {
		slog.Error("error configuring basic web filesystem", "err", err)
	} else {
		err := fs.ConfigUnix()
		if err != nil {
			slog.Error("error setting up standard unix filesystem structure", "err", err)
		}
	}

	base.Init(TheApp, &TheApp.App)
}

// TheApp is the single [goosi.App] for the web platform
var TheApp = &App{AppSingle: base.NewAppSingle[*Drawer, *Window]()}

// App is the [goosi.App] implementation for the web platform
type App struct { //gti:add
	base.AppSingle[*Drawer, *Window]

	// SystemPlatform is the underlying system SystemPlatform (Android, iOS, etc)
	SystemPlatform goosi.Platforms

	// KeyMods are the current key mods
	KeyMods key.Modifiers
}

// NewWindow creates a new window with the given options.
// It waits for the underlying system window to be created first.
// Also, it hides all other windows and shows the new one.
func (a *App) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	defer func() { goosi.HandleRecover(recover()) }()

	a.Win = &Window{base.NewWindowSingle(a, opts)}
	a.Win.This = a.Win
	a.SetSystemWindow()

	go a.Win.WinLoop()

	return a.Win, nil
}

// SetSystemWindow sets the underlying system window information.
func (a *App) SetSystemWindow() {
	defer func() { goosi.HandleRecover(recover()) }()

	a.AddEventListeners()

	ua := js.Global().Get("navigator").Get("userAgent").String()
	lua := strings.ToLower(ua)
	if strings.Contains(lua, "android") {
		a.SystemPlatform = goosi.Android
	} else if strings.Contains(lua, "ipad") || strings.Contains(lua, "iphone") || strings.Contains(lua, "ipod") {
		a.SystemPlatform = goosi.IOS
	} else {
		// TODO(kai/web): more specific desktop platform
		a.SystemPlatform = goosi.Windows
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

	if goosi.InitScreenLogicalDPIFunc != nil {
		goosi.InitScreenLogicalDPIFunc()
	}

	w, h := js.Global().Get("innerWidth").Int(), js.Global().Get("innerHeight").Int()
	sz := image.Pt(w, h)
	a.Scrn.Geometry.Max = sz
	a.Scrn.PixSize = image.Pt(int(float32(sz.X)*a.Scrn.DevicePixelRatio), int(float32(sz.Y)*a.Scrn.DevicePixelRatio))
	physX := 25.4 * float32(w) / dpi
	physY := 25.4 * float32(h) / dpi
	a.Scrn.PhysicalSize = image.Pt(int(physX), int(physY))

	canvas := js.Global().Get("document").Call("getElementById", "app")
	canvas.Set("width", a.Scrn.PixSize.X)
	canvas.Set("height", a.Scrn.PixSize.Y)

	// we need to manually set the style width and height of the canvas to innerWidth and innerHeight
	// instead of using 100vw and 100vh because vw and vh are incorrect on mobile browsers
	// due to the address bar but innerWidth and innerHeight are correct
	// (see https://stackoverflow.com/questions/43575363/css-100vh-is-too-tall-on-mobile-due-to-browser-ui)
	cstyle := canvas.Get("style")
	cstyle.Set("width", strconv.Itoa(w)+"px")
	cstyle.Set("height", strconv.Itoa(h)+"px")

	a.Drawer.Image = image.NewRGBA(image.Rectangle{Max: a.Scrn.PixSize})

	a.Win.EvMgr.WindowResize()
}

func (a *App) DataDir() string {
	return "/home/me/.data"
}

func (a *App) Platform() goosi.Platforms {
	return goosi.Web
}

func (a *App) OpenURL(url string) {
	js.Global().Call("open", url)
}

func (a *App) ClipBoard(win goosi.Window) clip.Board {
	return TheClip
}

func (a *App) Cursor(win goosi.Window) cursor.Cursor {
	return TheCursor
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
