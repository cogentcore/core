// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"goki.dev/girl/styles"
	"goki.dev/goosi"
)

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

	// Insts are the size of any insets on the sides of the screen.
	Insts styles.SideFloats
}

// AppSingler describes the common functionality implemented by [AppSingle]
// apps that [WindowSingle] windows need to access.
type AppSingler interface {
	goosi.App

	// SingleDrawer returns the single [goosi.Drawer] associated with this app.
	SingleDrawer() goosi.Drawer

	// Insets returns the size of any insets on the sides of the screen.
	Insets() styles.SideFloats
}

// NewAppSingle makes a new [AppSingle].
func NewAppSingle[D goosi.Drawer, W goosi.Window]() AppSingle[D, W] {
	return AppSingle[D, W]{
		Scrn: &goosi.Screen{},
	}
}

func (a *AppSingle[D, W]) SingleDrawer() goosi.Drawer {
	return a.Drawer
}

func (a *AppSingle[D, W]) Insets() styles.SideFloats {
	return a.Insts
}

func (a *AppSingle[D, W]) NScreens() int {
	if a.Scrn != nil {
		return 1
	}
	return 0
}

func (a *AppSingle[D, W]) Screen(n int) *goosi.Screen {
	if n == 0 {
		return a.Scrn
	}
	return nil
}

func (a *AppSingle[D, W]) ScreenByName(name string) *goosi.Screen {
	if a.Scrn.Name == name {
		return a.Scrn
	}
	return nil
}

func (a *AppSingle[D, W]) NWindows() int {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if goosi.Window(a.Win) != nil {
		return 1
	}
	return 0
}

func (a *AppSingle[D, W]) Window(win int) goosi.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if win == 0 {
		return a.Win
	}
	return nil
}

func (a *AppSingle[D, W]) WindowByName(name string) goosi.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if a.Win.Name() == name {
		return a.Win
	}
	return nil
}

func (a *AppSingle[D, W]) WindowInFocus() goosi.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if a.Win.Is(goosi.Focused) {
		return a.Win
	}
	return nil
}

func (a *AppSingle[D, W]) ContextWindow() goosi.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	return a.Win
}

func (a *AppSingle[D, W]) RemoveWindow(w goosi.Window) {
	// no-op
}

func (a *AppSingle[D, W]) QuitClean() {
	a.Quitting = true
	if a.QuitCleanFunc != nil {
		a.QuitCleanFunc()
	}
	a.Mu.Lock()
	a.Win.Close()
	a.Mu.Unlock()
}
