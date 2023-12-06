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
	"sync"

	"goki.dev/girl/styles"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
)

// Window contains the data and logic common to all implementations of [goosi.Window].
// A Window is associated with a corresponding [goosi.App] type.
type Window[A goosi.App] struct {
	events.Deque

	// This is the Window as a [goosi.Window] interface, which preserves the actual identity
	// of the window when calling interface methods in the base Window.
	This goosi.Window `view:"-"`

	// App is the [goosi.App] associated with the window.
	App A

	// Mu is the main mutex protecting access to window operations, including [Window.RunOnWin] functions.
	Mu sync.Mutex `view:"-"`

	// RunQueue is the queue of functions to call on the window loop. To add to it, use [Window.RunOnWin].
	RunQueue chan FuncRun

	publish     chan struct{}
	publishDone chan struct{}
	winClose    chan struct{}

	mainMenu goosi.MainMenu

	closeReqFunc   func(win goosi.Window)
	closeCleanFunc func(win goosi.Window)

	// Nm is the name of the window
	Nm string `label:"Name"`

	// Titl is the title of the window
	Titl string `label:"Title"`

	// Pos is the position of the window
	Pos image.Point

	// WnSize is the size of the window in window-manager coords
	WnSize image.Point

	// PixSize is the pixel size of the window in raw display dots
	PixSize image.Point

	// Insts are the cached insets of the window
	Insts styles.SideFloats `label:"Insets"`

	// DevicePixelRatio is a factor that scales the screen's
	// "natural" pixel coordinates into actual device pixels.
	// On OS-X, it is backingScaleFactor = 2.0 on "retina"
	DevicePixelRatio float32

	// PhysicalDPI is the physical dots per inch of the screen,
	// for generating true-to-physical-size output.
	// It is computed as 25.4 * (PixSize.X / PhysicalSize.X)
	// where 25.4 is the number of mm per inch.
	PhysDPI float32 `label:"Physical DPI"`

	// LogicalDPI is the logical dots per inch of the screen,
	// which is used for all rendering.
	// It is: transient zoom factor * screen-specific multiplier * PhysicalDPI
	LogDPI float32 `label:"Logical DPI"`

	// Flag contains the flags associated with the window
	Flag goosi.WindowFlags

	// FPS is the FPS (frames per second) for rendering the window
	FPS int

	// EvMgr is the event manager for the window
	EvMgr events.Mgr `label:"Event manger"`

	// DestroyGPUFunc should be set to a function that will destroy GPU resources
	// in the main thread prior to destroying the drawer
	// and the surface; otherwise it is difficult to
	// ensure that the proper ordering of destruction applies.
	DestroyGPUFunc func()
}

// RunOnWin runs given function on the window's unique locked thread.
func (w *Window[A]) RunOnWin(f func()) {
	if w.This.IsClosed() {
		return
	}
	done := make(chan struct{})
	w.RunQueue <- FuncRun{F: f, Done: done}
	<-done
}

// GoRunOnWin runs given function on window's unique locked thread and returns immediately
func (w *Window[A]) GoRunOnWin(f func()) {
	if w.This.IsClosed() {
		return
	}
	go func() {
		w.RunQueue <- FuncRun{F: f, Done: nil}
	}()
}

func (w *Window[A]) Name() string {
	return w.Nm
}

func (w *Window[A]) SetName(name string) {
	w.Nm = name
}

func (w *Window[A]) Title() string {
	return w.Titl
}

func (w *Window[A]) Flags() goosi.WindowFlags {
	return w.Flag
}

func (w *Window[A]) Is(flag goosi.WindowFlags) bool {
	return w.Flag.HasFlag(flag)
}

func (w *Window[A]) SetFPS(fps int) {
	w.FPS = fps
}

func (w *Window[A]) EventMgr() *events.Mgr {
	return &w.EvMgr
}

func (w *Window[A]) SetDestroyGPUResourcesFunc(f func()) {
	w.DestroyGPUFunc = f
}

func (w *Window[A]) Size() image.Point {
	// w.Mu.Lock() // this prevents race conditions but also locks up
	// defer w.Mu.Unlock()
	return w.PixSize
}

func (w *Window[A]) WinSize() image.Point {
	// w.Mu.Lock() // this prevents race conditions but also locks up
	// defer w.Mu.Unlock()
	return w.WnSize
}

func (w *Window[A]) Position() image.Point {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	return w.Pos
}

func (w *Window[A]) Insets() styles.SideFloats {
	return w.Insts
}

func (w *Window[A]) PhysicalDPI() float32 {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	return w.PhysDPI
}

func (w *Window[A]) LogicalDPI() float32 {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	return w.LogDPI
}

func (w *Window[A]) SetLogicalDPI(dpi float32) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	w.LogDPI = dpi
}
