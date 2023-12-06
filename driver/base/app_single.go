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

	// Window is the single [goosi.Window] associated with the app.
	Window W

	// Screen is the single [goosi.Screen] associated with the app.
	Screen *goosi.Screen
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

func (app *appImpl) NScreens() int {
	if app.screen != nil {
		return 1
	}
	return 0
}

func (app *appImpl) Screen(scrN int) *goosi.Screen {
	if scrN == 0 {
		return app.screen
	}
	return nil
}

func (app *appImpl) ScreenByName(name string) *goosi.Screen {
	if app.screen.Name == name {
		return app.screen
	}
	return nil
}

func (app *appImpl) NoScreens() bool {
	return app.screen == nil
}

func (app *appImpl) NWindows() int {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.window != nil {
		return 1
	}
	return 0
}

func (app *appImpl) Window(win int) goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	if win == 0 {
		return app.window
	}
	return nil
}

func (app *appImpl) WindowByName(name string) goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.window.Name() == name {
		return app.window
	}
	return nil
}

func (app *appImpl) WindowInFocus() goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.window.IsFocus() {
		return app.window
	}
	return nil
}

func (app *appImpl) ContextWindow() goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.window
}
