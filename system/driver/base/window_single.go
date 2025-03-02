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

// WindowSingle contains the data and logic common to all implementations of [system.Window]
// on single-window platforms (mobile and web), as opposed to multi-window
// platforms (desktop and offscreen), for which you should use [WindowSingle].
// A WindowSingle is associated with a corresponding [AppSingler] type.
type WindowSingle[A AppSingler] struct {
	Window[A]
}

// NewWindowSingle makes a new [WindowSingle] for the given app with the given options.
func NewWindowSingle[A AppSingler](a A, opts *system.NewWindowOptions) WindowSingle[A] {
	return WindowSingle[A]{
		NewWindow(a, opts),
	}
}

func (w *WindowSingle[A]) Events() *events.Source {
	return w.App.Events()
}

func (w *WindowSingle[A]) Composer() composer.Composer {
	return w.App.Composer()
}

func (w *WindowSingle[A]) Screen() *system.Screen {
	return w.App.Screen(0)
}

func (w *WindowSingle[A]) Size() image.Point {
	// w.Mu.Lock() // this prevents race conditions but also locks up
	// defer w.Mu.Unlock()
	return w.Screen().PixelSize
}

func (w *WindowSingle[A]) WinSize() image.Point {
	// w.Mu.Lock() // this prevents race conditions but also locks up
	// defer w.Mu.Unlock()
	return w.Screen().PixelSize
}

func (w *WindowSingle[A]) Position(screen *system.Screen) image.Point {
	// w.Mu.Lock()
	// defer w.Mu.Unlock()
	return image.Point{}
}

func (w *WindowSingle[A]) PhysicalDPI() float32 {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	return w.Screen().PhysicalDPI
}

func (w *WindowSingle[A]) LogicalDPI() float32 {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	return w.Screen().LogicalDPI
}

func (w *WindowSingle[A]) SetLogicalDPI(dpi float32) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	w.Screen().LogicalDPI = dpi
}

func (w *WindowSingle[A]) SetWinSize(sz image.Point) {
	if w.This.IsClosed() {
		return
	}
	w.Screen().PixelSize = sz
}

func (w *WindowSingle[A]) SetSize(sz image.Point) {
	if w.This.IsClosed() || w.App.Platform().IsMobile() {
		return
	}
	w.Screen().PixelSize = sz
}

func (w *WindowSingle[A]) SetPos(pos image.Point, screen *system.Screen) {
	// no-op
}

func (w *WindowSingle[A]) SetGeometry(fullscreen bool, pos image.Point, size image.Point, screen *system.Screen) {
	// mostly no-op
	w.Flgs.SetFlag(fullscreen, system.Fullscreen)
}

func (w *WindowSingle[A]) ConstrainFrame(topOnly bool) sides.Sides[int] {
	// no-op
	return sides.Sides[int]{}
}

func (w *WindowSingle[A]) RenderGeom() math32.Geom2DInt {
	return w.App.RenderGeom()
}
