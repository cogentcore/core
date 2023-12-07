// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package offscreen provides placeholder implementations of goosi interfaces
// to allow for offscreen testing and capturing of apps.
package offscreen

import (
	"image"
	"log"
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

// handleRecover takes the given value of recover, and, if it is not nil,
// prints a panic message and a stack trace, using a string-based log
// method that guarantees that the stack trace will be printed before
// the program exits. This is needed because, without this, the program
// will exit before it can print the stack trace, which makes debugging
// nearly impossible. The correct usage of handleRecover is:
//
//	func myFunc() {
//		defer func() { handleRecover(recover()) }()
//		...
//	}
func handleRecover(r any) {
	if r == nil {
		return
	}
	log.Println("panic:", r)
	log.Println("")
	log.Println("----- START OF STACK TRACE: -----")
	log.Println(string(debug.Stack()))
	log.Fatalln("----- END OF STACK TRACE -----")
}

// Main is called from main thread when it is time to start running the
// main loop.  When function f returns, the app ends automatically.
func Main(f func(goosi.App)) {
	debug.SetPanicOnFault(true)
	defer func() { handleRecover(recover()) }()
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
func (app *App) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	defer func() { handleRecover(recover()) }()
	app.Win = &Window{
		App:         app,
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
	app.Win.EvMgr.Deque = &app.Win.Deque
	app.setSysWindow(opts.Size)

	go app.Win.WinLoop()

	return app.Win, nil
}

// setSysWindow sets the underlying system window information.
func (app *App) setSysWindow(sz image.Point) error {
	debug.SetPanicOnFault(true)
	defer func() { handleRecover(recover()) }()

	if sz.X == 0 {
		sz.X = 800
	}
	if sz.Y == 0 {
		sz.Y = 600
	}

	app.Win.EvMgr.WindowResize()
	app.Win.EvMgr.Window(events.WinShow)
	app.Win.EvMgr.Window(events.ScreenUpdate)
	app.Win.EvMgr.Window(events.WinFocus)
	return nil
}

func (app *App) PrefsDir() string {
	// TODO(kai): figure out a better solution to offscreen prefs dir
	return filepath.Join(".", "tmpPrefsDir")
}

func (app *App) GetScreens() {
	sz := image.Point{1920, 1080}
	app.Scrn.DevicePixelRatio = 1
	app.Scrn.PixSize = sz
	app.Scrn.Geometry.Max = app.Scrn.PixSize
	dpi := float32(160)
	app.Scrn.PhysicalDPI = dpi
	app.Scrn.LogicalDPI = dpi

	physX := 25.4 * float32(sz.X) / dpi
	physY := 25.4 * float32(sz.Y) / dpi
	app.Scrn.PhysicalSize = image.Pt(int(physX), int(physY))
}

func (app *App) Platform() goosi.Platforms {
	return goosi.Offscreen
}

func (app *App) OpenURL(url string) {
	// no-op
}

func (app *App) ClipBoard(win goosi.Window) clip.Board {
	// TODO: implement clipboard
	return nil
}

func (app *App) Cursor(win goosi.Window) cursor.Cursor {
	return &cursor.CursorBase{} // no-op
}
