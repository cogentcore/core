// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"image"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
)

// AppSingle contains the data and logic common to all implementations of [system.App]
// on single-window platforms (mobile, web, and offscreen), as opposed to multi-window
// platforms (desktop), for which you should use [AppMulti]. An AppSingle is associated
// with a corresponding type of [system.Drawer] and [system.Window]. The [system.Window]
// type should embed [WindowSingle].
type AppSingle[D system.Drawer, W system.Window] struct {
	App

	// Event is the event manager for the app.
	Event events.Source `label:"Events"`

	// Draw is the single [system.Drawer] used for the app.
	Draw D `label:"Drawer"`

	// Win is the single [system.Window] associated with the app.
	Win W `label:"Window"`

	// Scrn is the single [system.Screen] associated with the app.
	Scrn *system.Screen `label:"Screen"`

	// Insets are the size of any insets on the sides of the screen.
	Insets styles.Sides[int]
}

// AppSingler describes the common functionality implemented by [AppSingle]
// apps that [WindowSingle] windows need to access.
type AppSingler interface {
	system.App

	// Events returns the single [events.Source] associated with this app.
	Events() *events.Source

	// Drawer returns the single [system.Drawer] associated with this app.
	Drawer() system.Drawer

	// RenderGeom returns the actual effective geometry of the window used
	// for rendering content, which may be different from {0, [system.Screen.PixSize]}
	// due to insets caused by things like status bars and button overlays.
	RenderGeom() math32.Geom2DInt
}

// NewAppSingle makes a new [AppSingle].
func NewAppSingle[D system.Drawer, W system.Window]() AppSingle[D, W] {
	return AppSingle[D, W]{
		Scrn: &system.Screen{Name: "main"},
	}
}

func (a *AppSingle[D, W]) Events() *events.Source {
	return &a.Event
}

func (a *AppSingle[D, W]) Drawer() system.Drawer {
	return a.Draw
}

func (a *AppSingle[D, W]) RenderGeom() math32.Geom2DInt {
	pos := image.Pt(a.Insets.Left, a.Insets.Top)
	return math32.Geom2DInt{
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

func (a *AppSingle[D, W]) Screen(n int) *system.Screen {
	if n == 0 {
		return a.Scrn
	}
	return nil
}

func (a *AppSingle[D, W]) ScreenByName(name string) *system.Screen {
	if a.Scrn.Name == name {
		return a.Scrn
	}
	return nil
}

func (a *AppSingle[D, W]) NWindows() int {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if system.Window(a.Win) != nil {
		return 1
	}
	return 0
}

func (a *AppSingle[D, W]) Window(win int) system.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if win == 0 {
		return a.Win
	}
	return nil
}

func (a *AppSingle[D, W]) WindowByName(name string) system.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if a.Win.Name() == name {
		return a.Win
	}
	return nil
}

func (a *AppSingle[D, W]) WindowInFocus() system.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if a.Win.Is(system.Focused) {
		return a.Win
	}
	return nil
}

func (a *AppSingle[D, W]) ContextWindow() system.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	return a.Win
}

func (a *AppSingle[D, W]) RemoveWindow(w system.Window) {
	// no-op
}

func (a *AppSingle[D, W]) QuitClean() bool {
	a.Quitting = true
	for _, qf := range a.QuitCleanFuncs {
		qf()
	}
	a.Mu.Lock()
	defer a.Mu.Unlock()
	a.Win.Close()
	return a.Win.IsClosed()
}
