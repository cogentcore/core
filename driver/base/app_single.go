// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import "goki.dev/goosi"

// AppSingle contains the data and logic common to all implementations of [goosi.App]
// on single-window platforms (mobile, web, and offscreen), as opposed to multi-window
// platforms (desktop), for which you should use [AppMulti]. An AppSingle is associated
// with a corresponding type of [goosi.Drawer] and [goosi.Window]. The [goosi.Window]
// type should embed [WindowSingle].
type AppSingle[D goosi.Drawer, W goosi.Window] struct {
	App

	// Drawer is the single [goosi.Drawer] used for the app.
	Drawer D

	// Win is the single [goosi.Window] associated with the app.
	Win W

	// Scrn is the single [goosi.Screen] associated with the app.
	Scrn *goosi.Screen
}

// AppSingler describes the common functionality implemented by [AppSingle]
// apps that [WindowSingle] windows need to access.
type AppSingler interface {
	goosi.App

	// SingleDrawer returns the single [goosi.Drawer] associated with this app.
	SingleDrawer() goosi.Drawer
}

func (a *AppSingle[D, W]) SingleDrawer() goosi.Drawer {
	return a.Drawer
}

func (app *AppSingle[D, W]) NScreens() int {
	if app.Scrn != nil {
		return 1
	}
	return 0
}

func (app *AppSingle[D, W]) Screen(n int) *goosi.Screen {
	if n == 0 {
		return app.Scrn
	}
	return nil
}

func (app *AppSingle[D, W]) ScreenByName(name string) *goosi.Screen {
	if app.Scrn.Name == name {
		return app.Scrn
	}
	return nil
}

func (app *AppSingle[D, W]) NWindows() int {
	app.Mu.Lock()
	defer app.Mu.Unlock()
	if goosi.Window(app.Win) != nil {
		return 1
	}
	return 0
}

func (app *AppSingle[D, W]) Window(win int) goosi.Window {
	app.Mu.Lock()
	defer app.Mu.Unlock()
	if win == 0 {
		return app.Win
	}
	return nil
}

func (app *AppSingle[D, W]) WindowByName(name string) goosi.Window {
	app.Mu.Lock()
	defer app.Mu.Unlock()
	if app.Win.Name() == name {
		return app.Win
	}
	return nil
}

func (app *AppSingle[D, W]) WindowInFocus() goosi.Window {
	app.Mu.Lock()
	defer app.Mu.Unlock()
	if app.Win.IsFocus() {
		return app.Win
	}
	return nil
}

func (app *AppSingle[D, W]) ContextWindow() goosi.Window {
	app.Mu.Lock()
	defer app.Mu.Unlock()
	return app.Win
}
