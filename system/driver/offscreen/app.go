// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package offscreen provides placeholder implementations of system interfaces
// to allow for offscreen testing and capturing of apps.
package offscreen

import (
	"image"
	"os"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
	"cogentcore.org/core/system/driver/base"
)

func Init() {
	TheApp.GetScreens()

	TheApp.TempDataDir = errors.Log1(os.MkdirTemp("", "cogent-core-offscreen-data-dir-"))

	base.Init(TheApp, &TheApp.App)
}

// TheApp is the single [system.App] for the offscreen platform
var TheApp = &App{AppMulti: base.NewAppMulti[*Window]()}

// App is the [system.App] implementation for the offscreen platform.
// It is multi-window but only has one screen.
type App struct {
	base.AppMulti[*Window]

	// TempDataDir is the path of the app data directory, used as the
	// return value of [App.DataDir]. It is set to a temporary directory,
	// as offscreen tests should not be dependent on user preferences and
	// other data.
	TempDataDir string
}

// NewWindow creates a new window with the given options.
// It waits for the underlying system window to be created first.
// Also, it hides all other windows and shows the new one.
func (a *App) NewWindow(opts *system.NewWindowOptions) (system.Window, error) {
	defer func() { system.HandleRecover(recover()) }()

	w := &Window{base.NewWindowMulti[*App, *composer.ComposerDrawer](a, opts)}
	w.This = w
	w.Compose = &composer.ComposerDrawer{Drawer: &Drawer{Window: w}}
	w.PixelSize = opts.Size

	a.Mu.Lock()
	a.Windows = append(a.Windows, w)
	a.GetScreens()
	a.Mu.Unlock()

	w.Event.WindowResize()
	w.Event.Window(events.WinShow)
	w.Event.Window(events.ScreenUpdate)
	w.Event.Window(events.WinFocus)

	go w.WinLoop()

	return w, nil
}

func (a *App) GetScreens() {
	if len(a.Screens) != 1 {
		a.Screens = []*system.Screen{{}}
	}
	sc := a.Screen(0)
	if sc.PixelSize.X == 0 {
		sc.PixelSize.X = 800
	}
	if sc.PixelSize.Y == 0 {
		sc.PixelSize.Y = 600
	}

	sc.DevicePixelRatio = 1
	sc.Geometry.Max = sc.PixelSize
	dpi := float32(160)
	sc.PhysicalDPI = dpi
	sc.LogicalDPI = dpi

	if system.InitScreenLogicalDPIFunc != nil {
		system.InitScreenLogicalDPIFunc()
	}

	physX := 25.4 * float32(sc.PixelSize.X) / dpi
	physY := 25.4 * float32(sc.PixelSize.Y) / dpi
	sc.PhysicalSize = image.Pt(int(physX), int(physY))
}

func (a *App) QuitClean() bool {
	if a.TempDataDir != "" {
		errors.Log(os.RemoveAll(a.TempDataDir))
	}
	return a.AppMulti.QuitClean()
}

func (a *App) DataDir() string {
	return a.TempDataDir
}

func (a *App) Platform() system.Platforms {
	return system.Offscreen
}
