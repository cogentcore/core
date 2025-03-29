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
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
)

// AppSingle contains the data and logic common to all implementations of [system.App]
// on single-window platforms (mobile and web), as opposed to multi-window
// platforms (desktop and offscreen), for which you should use [AppMulti]. An AppSingle is associated
// with a corresponding type of [composer.Composer] and [system.Window]. The [system.Window]
// type should embed [WindowSingle].
type AppSingle[C composer.Composer, W system.Window] struct {
	App

	// Event is the event manager for the app.
	Event events.Source `label:"Events"`

	// Compose is the single [composer.Composer] used for the app.
	Compose C `label:"Composer"`

	// Win is the single [system.Window] associated with the app.
	Win W `label:"Window"`

	// Scrn is the single [system.Screen] associated with the app.
	Scrn *system.Screen `label:"Screen"`

	// Insets are the size of any insets on the sides of the screen.
	Insets sides.Sides[int]
}

// AppSingler describes the common functionality implemented by [AppSingle]
// apps that [WindowSingle] windows need to access.
type AppSingler interface {
	system.App

	// Events returns the single [events.Source] associated with this app.
	Events() *events.Source

	// Composer returns the single [composer.Composer] associated with this app.
	Composer() composer.Composer

	// RenderGeom returns the actual effective geometry of the window used
	// for rendering content, which may be different from {0, [system.Screen.PixelSize]}
	// due to insets caused by things like status bars and button overlays.
	RenderGeom() math32.Geom2DInt
}

// NewAppSingle makes a new [AppSingle].
func NewAppSingle[C composer.Composer, W system.Window]() AppSingle[C, W] {
	return AppSingle[C, W]{
		Scrn: &system.Screen{Name: "main"},
	}
}

func (a *AppSingle[C, W]) Events() *events.Source {
	return &a.Event
}

func (a *AppSingle[C, W]) Composer() composer.Composer {
	return a.Compose
}

func (a *AppSingle[C, W]) RenderGeom() math32.Geom2DInt {
	pos := image.Pt(a.Insets.Left, a.Insets.Top)
	return math32.Geom2DInt{
		Pos:  pos,
		Size: a.Scrn.PixelSize.Sub(pos).Sub(image.Pt(a.Insets.Right, a.Insets.Bottom)),
	}
}

func (a *AppSingle[C, W]) NScreens() int {
	if a.Scrn != nil {
		return 1
	}
	return 0
}

func (a *AppSingle[C, W]) Screen(n int) *system.Screen {
	return a.Scrn
}

func (a *AppSingle[C, W]) ScreenByName(name string) *system.Screen {
	if a.Scrn.Name == name {
		return a.Scrn
	}
	return nil
}

func (a *AppSingle[C, W]) NWindows() int {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if system.Window(a.Win) != nil {
		return 1
	}
	return 0
}

func (a *AppSingle[C, W]) Window(win int) system.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if win == 0 {
		return a.Win
	}
	return nil
}

func (a *AppSingle[C, W]) WindowByName(name string) system.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if a.Win.Name() == name {
		return a.Win
	}
	return nil
}

func (a *AppSingle[C, W]) WindowInFocus() system.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if a.Win.Is(system.Focused) {
		return a.Win
	}
	return nil
}

func (a *AppSingle[C, W]) ContextWindow() system.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	return a.Win
}

func (a *AppSingle[C, W]) RemoveWindow(w system.Window) {
	// no-op
}

func (a *AppSingle[C, W]) QuitClean() bool {
	a.Quitting = true
	for _, qf := range a.QuitCleanFuncs {
		qf()
	}
	a.Mu.Lock()
	defer a.Mu.Unlock()
	a.Win.Close()
	return a.Win.IsClosed()
}
