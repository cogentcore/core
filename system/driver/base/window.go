// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"fmt"
	"image"
	"sync"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/system"
)

// Window contains the data and logic common to all implementations of [system.Window].
// A Window is associated with a corresponding [system.App] type.
type Window[A system.App] struct {

	// This is the Window as a [system.Window] interface, which preserves the actual identity
	// of the window when calling interface methods in the base Window.
	This system.Window `display:"-"`

	// App is the [system.App] associated with the window.
	App A

	// Mu is the main mutex protecting access to window operations, including [Window.RunOnWin] functions.
	Mu sync.Mutex `display:"-"`

	// WinClose is a channel on which a single is sent to indicate that the
	// window should close.
	WinClose chan struct{} `display:"-"`

	// CloseReqFunc is the function to call on a close request
	CloseReqFunc func(win system.Window)

	// CloseCleanFunc is the function to call to close the window
	CloseCleanFunc func(win system.Window)

	// Nm is the name of the window
	Nm string `label:"Name"`

	// Titl is the title of the window
	Titl string `label:"Title"`

	// Flgs contains the flags associated with the window
	Flgs system.WindowFlags `label:"Flags" table:"-"`

	// DestroyGPUFunc should be set to a function that will destroy GPU resources
	// in the main thread prior to destroying the drawer
	// and the surface; otherwise it is difficult to
	// ensure that the proper ordering of destruction applies.
	DestroyGPUFunc func()

	// CursorEnabled is whether the cursor is currently enabled
	CursorEnabled bool
}

// NewWindow makes a new [Window] for the given app with the given options.
func NewWindow[A system.App](a A, opts *system.NewWindowOptions) Window[A] {
	return Window[A]{
		WinClose:      make(chan struct{}),
		App:           a,
		Titl:          opts.GetTitle(),
		Flgs:          opts.Flags,
		CursorEnabled: true,
	}
}

// WinLoop runs the window's own locked processing loop.
func (w *Window[A]) WinLoop() {
	defer func() { system.HandleRecover(recover()) }()

	fps := w.This.Screen().RefreshRate
	if fps <= 0 {
		fps = 60
	}
	dur := time.Second / time.Duration(fps)
	winPaint := time.NewTicker(dur)
outer:
	for {
		select {
		case <-w.WinClose:
			winPaint.Stop()
			break outer
		case <-winPaint.C:
			if w.This.IsClosed() {
				winPaint.Stop()
				fmt.Println("win IsClosed in paint:", w.Name())
				break outer
			}
			w.This.SendPaintEvent()
		}
	}
}

func (w *Window[A]) SendPaintEvent() {
	w.This.Events().WindowPaint()
}

func (w *Window[A]) Lock() bool {
	if w.This.IsClosed() {
		return false
	}
	w.Mu.Lock()
	return true
}

func (w *Window[A]) Unlock() {
	w.Mu.Unlock()
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

func (w *Window[A]) SetTitle(title string) {
	if w.This.IsClosed() {
		return
	}
	w.Titl = title
}

func (w *Window[A]) SetIcon(images []image.Image) {
	// no-op by default
}

func (w *Window[A]) Raise() {
	// no-op by default
}

func (w *Window[A]) Minimize() {
	// no-op by default
}

func (w *Window[A]) Flags() system.WindowFlags {
	return w.Flgs
}

func (w *Window[A]) Is(flag system.WindowFlags) bool {
	return w.Flgs.HasFlag(flag)
}

func (w *Window[A]) IsClosed() bool {
	return w == nil || w.This == nil || w.This.Composer() == nil
}

func (w *Window[A]) IsVisible() bool {
	return !w.This.IsClosed() && !w.Is(system.Minimized)
}

func (w *Window[A]) SetDestroyGPUResourcesFunc(f func()) {
	w.DestroyGPUFunc = f
}

func (w *Window[A]) RenderGeom() math32.Geom2DInt {
	// {0, Size} by default
	return math32.Geom2DInt{Size: w.This.Size()}
}

func (w *Window[A]) SetCloseReqFunc(fun func(win system.Window)) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	w.CloseReqFunc = fun
}

func (w *Window[A]) SetCloseCleanFunc(fun func(win system.Window)) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	w.CloseCleanFunc = fun
}

func (w *Window[A]) CloseReq() {
	if w.CloseReqFunc != nil {
		w.CloseReqFunc(w.This)
	} else {
		w.This.Close()
	}
}

func (w *Window[A]) CloseClean() {
	if w.CloseCleanFunc != nil {
		w.CloseCleanFunc(w.This)
	}
}

func (w *Window[A]) Close() {
	// base implementation doesn't actually close any system windows,
	// but platform-specific implementations can
	w.This.Events().Window(events.WinClose)

	w.WinClose <- struct{}{}

	w.Mu.Lock()
	defer w.Mu.Unlock()

	w.CloseClean()
	w.App.RemoveWindow(w.This)
}

func (w *Window[A]) SetCursorEnabled(enabled, raw bool) {
	w.CursorEnabled = enabled
}

func (w *Window[A]) IsCursorEnabled() bool {
	return w.CursorEnabled
}

func (w *Window[A]) SetMousePos(x, y float64) {
	// no-op by default
}

func (w *Window[A]) SetTitleBarIsDark(isDark bool) {
	// no-op by default
}
