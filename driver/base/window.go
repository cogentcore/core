// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"sync"
	"time"

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

	// WinClose is a channel on which a single is sent to indicate that the
	// window should close.
	WinClose chan struct{}

	// CloseReqFunc is the function to call on a close request
	CloseReqFunc func(win goosi.Window)

	// CloseCleanFunc is the function to call to close the window
	CloseCleanFunc func(win goosi.Window)

	// Nm is the name of the window
	Nm string `label:"Name"`

	// Titl is the title of the window
	Titl string `label:"Title"`

	// Insts are the cached insets of the window
	Insts styles.SideFloats `label:"Insets"`

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

	// CursorEnabled is whether the cursor is currently disabled
	CursorEnabled bool
}

// WinLoop runs the window's own locked processing loop.
func (w *Window[A]) WinLoop() {
	var winPaint *time.Ticker
	if w.FPS > 0 {
		winPaint = time.NewTicker(time.Second / time.Duration(w.FPS))
	} else {
		winPaint = &time.Ticker{C: make(chan time.Time)} // nop
	}
	winShow := time.NewTimer(200 * time.Millisecond)
outer:
	for {
		select {
		case <-w.WinClose:
			winPaint.Stop()
			break outer
		case <-winShow.C:
			if !w.This.IsVisible() {
				break outer
			}
			w.EvMgr.Window(events.WinShow)
		case f := <-w.RunQueue:
			if !w.This.IsVisible() {
				break outer
			}
			f.F()
			if f.Done != nil {
				f.Done <- struct{}{}
			}
		case <-winPaint.C:
			if !w.This.IsVisible() {
				break outer
			}
			w.EvMgr.WindowPaint()
		}
	}
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

func (w *Window[A]) SetTitle(title string) {
	if w.This.IsClosed() {
		return
	}
	w.Titl = title
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

func (w *Window[A]) Insets() styles.SideFloats {
	return w.Insts
}

func (w *Window[A]) SetCloseReqFunc(fun func(win goosi.Window)) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	w.CloseReqFunc = fun
}

func (w *Window[A]) SetCloseCleanFunc(fun func(win goosi.Window)) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	w.CloseCleanFunc = fun
}

func (w *Window[A]) CloseReq() {
	if w.App.IsQuitting() {
		w.This.Close()
	}
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
