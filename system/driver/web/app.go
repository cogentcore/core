// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

// Package web implements system interfaces on the web through WASM
package web

import (
	"fmt"
	"image"
	"log/slog"
	"os"
	"strings"
	"syscall/js"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/jsfs"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/driver/base"
)

func Init() {
	TheApp.Draw = &Drawer{}

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

	TheApp.SetSystemWindow()

	base.Init(TheApp, &TheApp.App)
}

// TheApp is the single [system.App] for the web platform
var TheApp = &App{AppSingle: base.NewAppSingle[*Drawer, *Window]()}

// App is the [system.App] implementation for the web platform
type App struct {
	base.AppSingle[*Drawer, *Window]

	// UnderlyingPlatform is the underlying system platform (Android, iOS, etc)
	UnderlyingPlatform system.Platforms

	// KeyMods are the current key mods
	KeyMods key.Modifiers
}

// NewWindow creates a new window with the given options.
// It waits for the underlying system window to be created first.
// Also, it hides all other windows and shows the new one.
func (a *App) NewWindow(opts *system.NewWindowOptions) (system.Window, error) {
	defer func() { system.HandleRecover(recover()) }()

	a.Win = &Window{base.NewWindowSingle(a, opts)}
	a.Win.This = a.Win

	go a.Win.WinLoop()

	return a.Win, nil
}

// SetSystemWindow sets the underlying system window information.
func (a *App) SetSystemWindow() {
	defer func() { system.HandleRecover(recover()) }()

	a.AddEventListeners()

	ua := js.Global().Get("navigator").Get("userAgent").String()
	a.UnderlyingPlatform = UserAgentToOS(ua)

	a.Resize()
	a.EvMgr.Window(events.WinShow)
	a.EvMgr.Window(events.ScreenUpdate)
	a.EvMgr.Window(events.WinFocus)
}

// UserAgentToOS converts the given user agent string to a [system.Platforms] value.
func UserAgentToOS(ua string) system.Platforms {
	lua := strings.ToLower(ua)
	switch {
	case strings.Contains(lua, "android"):
		return system.Android
	case strings.Contains(lua, "ipad"),
		strings.Contains(lua, "iphone"),
		strings.Contains(lua, "ipod"):
		return system.IOS
	case strings.Contains(lua, "mac"):
		return system.MacOS
	case strings.Contains(lua, "win"):
		return system.Windows
	default:
		return system.Linux
	}
}

// Resize updates the app sizing information and sends a Resize event.
func (a *App) Resize() {
	a.Scrn.DevicePixelRatio = float32(js.Global().Get("devicePixelRatio").Float())
	dpi := 160 * a.Scrn.DevicePixelRatio
	a.Scrn.PhysicalDPI = dpi
	a.Scrn.LogicalDPI = dpi

	if system.InitScreenLogicalDPIFunc != nil {
		system.InitScreenLogicalDPIFunc()
	}

	w, h := js.Global().Get("innerWidth").Int(), js.Global().Get("innerHeight").Int()
	sz := image.Pt(w, h)
	a.Scrn.Geometry.Max = sz
	a.Scrn.PixSize = image.Pt(int(math32.Ceil(float32(sz.X)*a.Scrn.DevicePixelRatio)), int(math32.Ceil(float32(sz.Y)*a.Scrn.DevicePixelRatio)))
	physX := 25.4 * float32(w) / dpi
	physY := 25.4 * float32(h) / dpi
	a.Scrn.PhysicalSize = image.Pt(int(physX), int(physY))

	canvas := js.Global().Get("document").Call("getElementById", "app")
	canvas.Set("width", a.Scrn.PixSize.X)
	canvas.Set("height", a.Scrn.PixSize.Y)

	// we need to manually set the style width and height of the canvas
	// instead of using 100vw and 100vh because vw and vh are incorrect on mobile browsers
	// due to the address bar but innerWidth and innerHeight are correct
	// (see https://stackoverflow.com/questions/43575363/css-100vh-is-too-tall-on-mobile-due-to-browser-ui)
	//
	// we also need to divide by the pixel size by the pixel ratio again instead of just using innerWidth
	// and innerHeight so that there are no rounding errors (CSS supports fractional pixels but HTML doesn't);
	// these rounding errors lead to blurriness on devices with fractional device pixel ratios
	// (see https://github.com/cogentcore/core/issues/779 and
	// https://stackoverflow.com/questions/15661339/how-do-i-fix-blurry-text-in-my-html5-canvas/54027313#54027313)
	cstyle := canvas.Get("style")
	cstyle.Set("width", fmt.Sprintf("%gpx", float32(a.Scrn.PixSize.X)/a.Scrn.DevicePixelRatio))
	cstyle.Set("height", fmt.Sprintf("%gpx", float32(a.Scrn.PixSize.Y)/a.Scrn.DevicePixelRatio))

	a.Draw.Image = image.NewRGBA(image.Rectangle{Max: a.Scrn.PixSize})

	a.EvMgr.WindowResize()
}

func (a *App) DataDir() string {
	return "/home/me/.data"
}

func (a *App) Platform() system.Platforms {
	return system.Web
}

func (a *App) SystemPlatform() system.Platforms {
	return a.UnderlyingPlatform
}

func (a *App) OpenURL(url string) {
	js.Global().Call("open", url)
}

func (a *App) Clipboard(win system.Window) system.Clipboard {
	return TheClipboard
}

func (a *App) Cursor(win system.Window) system.Cursor {
	return TheCursor
}

func (a *App) IsDark() bool {
	return js.Global().Get("matchMedia").Truthy() &&
		js.Global().Call("matchMedia", "(prefers-color-scheme: dark)").Get("matches").Truthy()
}

func (a *App) ShowVirtualKeyboard(typ styles.VirtualKeyboards) {
	tf := js.Global().Get("document").Call("getElementById", "text-field")
	switch typ {
	case styles.KeyboardNumber, styles.KeyboardPassword, styles.KeyboardEmail, styles.KeyboardURL:
		tf.Set("type", typ.String())
	case styles.KeyboardPhone:
		tf.Set("type", "tel")
	default:
		tf.Set("type", "text")
	}
	tf.Call("focus")
}

func (a *App) HideVirtualKeyboard() {
	js.Global().Get("document").Call("getElementById", "text-field").Call("blur")
}
