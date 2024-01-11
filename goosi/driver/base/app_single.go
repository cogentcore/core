// Copyright 2023 The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"image"

	"goki.dev/events"
	"goki.dev/goosi"
	"goki.dev/mat32"
	"goki.dev/styles"
)

// AppSingle contains the data and logic common to all implementations of [goosi.App]
// on single-window platforms (mobile, web, and offscreen), as opposed to multi-window
// platforms (desktop), for which you should use [AppMulti]. An AppSingle is associated
// with a corresponding type of [goosi.Drawer] and [goosi.Window]. The [goosi.Window]
// type should embed [WindowSingle].
type AppSingle[D goosi.Drawer, W goosi.Window] struct { //gti:add
	App

	// EvMgr is the event manager for the app
	EvMgr events.Mgr `label:"Event manger"`

	// Draw is the single [goosi.Drawer] used for the app.
	Draw D

	// Win is the single [goosi.Window] associated with the app.
	Win W `label:"Window"`

	// Scrn is the single [goosi.Screen] associated with the app.
	Scrn *goosi.Screen `label:"Screen"`

	// Insets are the size of any insets on the sides of the screen.
	Insets styles.Sides[int]
}

// AppSingler describes the common functionality implemented by [AppSingle]
// apps that [WindowSingle] windows need to access.
type AppSingler interface {
	goosi.App

	// EventMgr returns the single [events.Mgr] associated with this app.
	EventMgr() *events.Mgr

	// Drawer returns the single [goosi.Drawer] associated with this app.
	Drawer() goosi.Drawer

	// RenderGeom returns the actual effective geometry of the window used
	// for rendering content, which may be different from {0, [goosi.Screen.PixSize]}
	// due to insets caused by things like status bars and button overlays.
	RenderGeom() mat32.Geom2DInt
}

// NewAppSingle makes a new [AppSingle].
func NewAppSingle[D goosi.Drawer, W goosi.Window]() AppSingle[D, W] {
	return AppSingle[D, W]{
		Scrn: &goosi.Screen{},
	}
}

func (a *AppSingle[D, W]) EventMgr() *events.Mgr {
	return &a.EvMgr
}

func (a *AppSingle[D, W]) Drawer() goosi.Drawer {
	return a.Draw
}

func (a *AppSingle[D, W]) RenderGeom() mat32.Geom2DInt {
	pos := image.Pt(a.Insets.Left, a.Insets.Top)
	return mat32.Geom2DInt{
		Pos:  pos,
		Size: a.Scrn.PixSize.Sub(pos).Sub(image.Pt(a.Insets.Right, a.Insets.Bottom)),
	}
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
