// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"image"

	"goki.dev/goosi"
	"goki.dev/goosi/events"
)

// Window contains the data and logic common to all implementations of [goosi.Window].
type Window struct {
	events.Deque

	// Nm is the name of the window
	Nm string

	// Titl is the title of the window
	Titl string

	// Pos is the position of the window
	Pos image.Point

	// WnSize is the size of the window in window-manager coords
	WnSize image.Point

	// PixSize is the pixel size of the window in raw display dots
	PixSize image.Point

	// DevicePixelRatio is a factor that scales the screen's
	// "natural" pixel coordinates into actual device pixels.
	// On OS-X, it is backingScaleFactor = 2.0 on "retina"
	DevicePixelRatio float32

	// PhysicalDPI is the physical dots per inch of the screen,
	// for generating true-to-physical-size output.
	// It is computed as 25.4 * (PixSize.X / PhysicalSize.X)
	// where 25.4 is the number of mm per inch.
	PhysDPI float32

	// LogicalDPI is the logical dots per inch of the screen,
	// which is used for all rendering.
	// It is: transient zoom factor * screen-specific multiplier * PhysicalDPI
	LogDPI float32

	// Flag contains the flags associated with the window
	Flag goosi.WindowFlags

	// FPS is the FPS (frames per second) for rendering the window
	FPS int

	// EvMgr is the event manager for the window
	EvMgr events.Mgr

	// DestroyGPUFunc should be set to a function that will destroy GPU resources
	// in the main thread prior to destroying the drawer
	// and the surface; otherwise it is difficult to
	// ensure that the proper ordering of destruction applies.
	DestroyGPUFunc func()
}

func (w *Window) Name() string {
	return w.Nm
}

func (w *Window) SetName(name string) {
	w.Nm = name
}

func (w *Window) Title() string {
	return w.Titl
}

func (w *Window) Flags() goosi.WindowFlags {
	return w.Flag
}

func (w *Window) Is(flag goosi.WindowFlags) bool {
	return w.Flag.HasFlag(flag)
}

func (w *Window) SetFPS(fps int) {
	w.FPS = fps
}

func (w *Window) EventMgr() *events.Mgr {
	return &w.EvMgr
}

func (w *Window) SetDestroyGPUResourcesFunc(f func()) {
	w.DestroyGPUFunc = f
}
