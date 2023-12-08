// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package offscreen provides placeholder implementations of goosi interfaces
// to allow for offscreen testing and capturing of apps.
package offscreen

import (
	"image"
	"path/filepath"
	"runtime/debug"

	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/driver/base"
	"goki.dev/goosi/events"
)

// TheApp is the single [goosi.App] for the offscreen platform
var TheApp = &App{}

var _ goosi.App = TheApp

// App is the [goosi.App] implementation on the offscreen platform
type App struct {
	base.AppSingle[*Drawer, *Window]
}

// Main is called from main thread when it is time to start running the
// main loop.  When function f returns, the app ends automatically.
func Main(f func(goosi.App)) {
	debug.SetPanicOnFault(true)
	defer func() { base.HandleRecover(recover()) }()
	TheApp.This = TheApp
	TheApp.GetScreens()
	goosi.TheApp = TheApp
	go func() {
		f(TheApp)
		TheApp.StopMain()
	}()
	TheApp.MainLoop()
}

////////////////////////////////////////////////////////
//  Window

// NewWindow creates a new window with the given options.
// It waits for the underlying system window to be created first.
// Also, it hides all other windows and shows the new one.
func (a *App) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	defer func() { base.HandleRecover(recover()) }()
	bw := &base.WindowSingle[*App]{
		App:         a,
		isVisible:   true,
		publish:     make(chan struct{}),
		winClose:    make(chan struct{}),
		publishDone: make(chan struct{}),
		WindowBase: goosi.WindowBase{
			Titl: opts.GetTitle(),
			Flag: opts.Flags,
			FPS:  60,
		},
	}
	a.Win = &Window{base.WindowSingle[*App]{}}
	a.App = a

	a.Win.EvMgr.Deque = &a.Win.Deque
	a.setSysWindow(opts.Size)

	go a.Win.WinLoop()

	return a.Win, nil
}

// setSysWindow sets the underlying system window information.
func (a *App) setSysWindow(sz image.Point) error {
	debug.SetPanicOnFault(true)
	defer func() { handleRecover(recover()) }()

	if sz.X == 0 {
		sz.X = 800
	}
	if sz.Y == 0 {
		sz.Y = 600
	}
	a.Scrn.PixSize = sz
	a.GetScreens()

	a.Win.EvMgr.WindowResize()
	a.Win.EvMgr.Window(events.WinShow)
	a.Win.EvMgr.Window(events.ScreenUpdate)
	a.Win.EvMgr.Window(events.WinFocus)
	return nil
}

func (a *App) PrefsDir() string {
	// TODO(kai): figure out a better solution to offscreen prefs dir
	return filepath.Join(".", "tmpPrefsDir")
}

func (a *App) GetScreens() {
	if a.Scrn.PixSize == (image.Point{}) {
		a.Scrn.PixSize = image.Point{800, 600}
	}
	a.Scrn.DevicePixelRatio = 1
	a.Scrn.Geometry.Max = a.Scrn.PixSize
	dpi := float32(160)
	a.Scrn.PhysicalDPI = dpi
	a.Scrn.LogicalDPI = dpi

	physX := 25.4 * float32(a.Scrn.PixSize.X) / dpi
	physY := 25.4 * float32(a.Scrn.PixSize.Y) / dpi
	a.Scrn.PhysicalSize = image.Pt(int(physX), int(physY))
}

func (a *App) Platform() goosi.Platforms {
	return goosi.Offscreen
}

func (a *App) OpenURL(url string) {
	// no-op
}

func (a *App) ClipBoard(win goosi.Window) clip.Board {
	// TODO: implement clipboard
	return nil
}

func (a *App) Cursor(win goosi.Window) cursor.Cursor {
	return &cursor.CursorBase{} // no-op
}
