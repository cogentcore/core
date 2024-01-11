// Copyright 2023 The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package offscreen provides placeholder implementations of goosi interfaces
// to allow for offscreen testing and capturing of apps.
package offscreen

//go:generate goki generate

import (
	"image"
	"os"

	"goki.dev/events"
	"goki.dev/goosi"
	"goki.dev/goosi/driver/base"
	"goki.dev/grr"
)

func Init() {
	TheApp.Draw = &Drawer{}
	TheApp.GetScreens()

	TheApp.TempDataDir = grr.Log1(os.MkdirTemp("", "goki-goosi-offscreen-data-dir-"))

	base.Init(TheApp, &TheApp.App)
}

// TheApp is the single [goosi.App] for the offscreen platform
var TheApp = &App{AppSingle: base.NewAppSingle[*Drawer, *Window]()}

// App is the [goosi.App] implementation for the offscreen platform
type App struct { //gti:add
	base.AppSingle[*Drawer, *Window]

	// TempDataDir is the path of the app data directory, used as the
	// return value of [App.DataDir]. It is set to a temporary directory,
	// as offscreen tests should not be dependent on user preferences and
	// other data.
	TempDataDir string
}

// NewWindow creates a new window with the given options.
// It waits for the underlying system window to be created first.
// Also, it hides all other windows and shows the new one.
func (a *App) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	defer func() { goosi.HandleRecover(recover()) }()

	a.Win = &Window{base.NewWindowSingle(a, opts)}
	a.Win.This = a.Win
	a.Scrn.PixSize = opts.Size
	a.GetScreens()

	a.EvMgr.WindowResize()
	a.EvMgr.Window(events.WinShow)
	a.EvMgr.Window(events.ScreenUpdate)
	a.EvMgr.Window(events.WinFocus)

	go a.Win.WinLoop()

	return a.Win, nil
}

func (a *App) GetScreens() {
	if a.Scrn.PixSize.X == 0 {
		a.Scrn.PixSize.X = 800
	}
	if a.Scrn.PixSize.Y == 0 {
		a.Scrn.PixSize.Y = 600
	}

	a.Scrn.DevicePixelRatio = 1
	a.Scrn.Geometry.Max = a.Scrn.PixSize
	dpi := float32(160)
	a.Scrn.PhysicalDPI = dpi
	a.Scrn.LogicalDPI = dpi

	if goosi.InitScreenLogicalDPIFunc != nil {
		goosi.InitScreenLogicalDPIFunc()
	}

	physX := 25.4 * float32(a.Scrn.PixSize.X) / dpi
	physY := 25.4 * float32(a.Scrn.PixSize.Y) / dpi
	a.Scrn.PhysicalSize = image.Pt(int(physX), int(physY))

	a.Draw.Image = image.NewRGBA(image.Rectangle{Max: a.Scrn.PixSize})
}

func (a *App) QuitClean() {
	if a.TempDataDir != "" {
		grr.Log(os.RemoveAll(a.TempDataDir))
	}
	a.AppSingle.QuitClean()
}

func (a *App) DataDir() string {
	return a.TempDataDir
}

func (a *App) Platform() goosi.Platforms {
	return goosi.Offscreen
}
