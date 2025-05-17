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
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
)

// WindowMulti contains the data and logic common to all implementations of [system.Window]
// on multi-window platforms (desktop and offscreen), as opposed to single-window
// platforms (mobile and web), for which you should use [WindowSingle].
// A WindowMulti is associated with a corresponding [system.App] type.
// The [system.App] type should embed [AppMulti].
type WindowMulti[A system.App, C composer.Composer] struct {
	Window[A]

	// Event is the event manager for the window
	Event events.Source `label:"Event manger"`

	// Compose is the [composer.Composer] for this window.
	Compose C `label:"Composer"`

	// Pos is the position of the window
	Pos image.Point `label:"Position"`

	// WnSize is the size of the window in window manager coordinates
	WnSize image.Point `label:"Window manager size"`

	// PixelSize is the pixel size of the window in raw display dots
	PixelSize image.Point `label:"Pixel size"`

	// FrameSize of the window frame: Min = left, top; Max = right, bottom.
	FrameSize sides.Sides[int]

	// DevicePixelRatio is a factor that scales the screen's
	// "natural" pixel coordinates into actual device pixels.
	// On OS-X, it is backingScaleFactor = 2.0 on "retina"
	DevicePixelRatio float32

	// PhysicalDPI is the physical dots per inch of the screen,
	// for generating true-to-physical-size output.
	// It is computed as 25.4 * (PixelSize.X / PhysicalSize.X)
	// where 25.4 is the number of mm per inch.
	PhysDPI float32 `label:"Physical DPI"`

	// LogicalDPI is the logical dots per inch of the screen,
	// which is used for all rendering.
	// It is: transient zoom factor * screen-specific multiplier * PhysicalDPI
	LogDPI float32 `label:"Logical DPI"`
}

// NewWindowMulti makes a new [WindowMulti] for the given app with the given options.
func NewWindowMulti[A system.App, C composer.Composer](a A, opts *system.NewWindowOptions) WindowMulti[A, C] {
	return WindowMulti[A, C]{
		Window: NewWindow(a, opts),
	}
}

func (w *WindowMulti[A, D]) Composer() composer.Composer {
	return w.Compose
}

func (w *WindowMulti[A, D]) Events() *events.Source {
	return &w.Event
}

func (w *WindowMulti[A, D]) Size() image.Point {
	// w.Mu.Lock() // this prevents race conditions but also locks up
	// defer w.Mu.Unlock()
	return w.PixelSize
}

func (w *WindowMulti[A, D]) WinSize() image.Point {
	// w.Mu.Lock() // this prevents race conditions but also locks up
	// defer w.Mu.Unlock()
	return w.WnSize
}

func (w *WindowMulti[A, D]) Position(screen *system.Screen) image.Point {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	return w.Pos
}

func (w *WindowMulti[A, D]) PhysicalDPI() float32 {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	return w.PhysDPI
}

func (w *WindowMulti[A, D]) LogicalDPI() float32 {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	return w.LogDPI
}

func (w *WindowMulti[A, D]) SetLogicalDPI(dpi float32) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	w.LogDPI = dpi
}

func (w *WindowMulti[A, D]) SetWinSize(sz image.Point) {
	if w.This.IsClosed() {
		return
	}
	w.WnSize = sz
}

func (w *WindowMulti[A, D]) SetSize(sz image.Point) {
	if w.This.IsClosed() {
		return
	}
	sc := w.This.Screen()
	sz = sc.WindowSizeFromPixels(sz)
	w.SetWinSize(sz)
}

func (w *WindowMulti[A, D]) SetPos(pos image.Point, screen *system.Screen) {
	if w.This.IsClosed() {
		return
	}
	w.Pos = pos
}

func (w *WindowMulti[A, D]) SetGeometry(fullscreen bool, pos image.Point, size image.Point, screen *system.Screen) {
	if w.This.IsClosed() {
		return
	}
	sc := w.This.Screen()
	size = sc.WindowSizeFromPixels(size)
	w.SetWinSize(size)
	w.Pos = pos
}

func (w *WindowMulti[A, D]) ConstrainFrame(topOnly bool) sides.Sides[int] {
	// no-op
	return sides.Sides[int]{}
}

func (w *WindowMulti[A, D]) IsVisible() bool {
	return w.Window.IsVisible() && w.App.NScreens() != 0
}
