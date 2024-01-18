// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package base provides base driver types that platform-specific drivers can extend
// to implement interfaces defined in package goosi.
package base

//go:generate core generate

import (
	"os"
	"path/filepath"
	"sync"

	"cogentcore.org/core/cursor"
	"cogentcore.org/core/goosi"
)

// App contains the data and logic common to all implementations of [goosi.App].
type App struct { //gti:add
	// This is the App as a [goosi.App] interface, which preserves the actual identity
	// of the app when calling interface methods in the base App.
	This goosi.App `view:"-"`

	// Mu is the main mutex protecting access to app operations, including [App.RunOnMain] functions.
	Mu sync.Mutex `view:"-"`

	// MainQueue is the queue of functions to call on the main loop.
	// To add to it, use [App.RunOnMain].
	MainQueue chan FuncRun `view:"-"`

	// MainDone is a channel on which is a signal is sent when the main
	// loop of the app should be terminated.
	MainDone chan struct{} `view:"-"`

	// Nm is the name of the app.
	Nm string `label:"Name"`

	// Abt is the about information for the app.
	Abt string `label:"About"`

	// OpenFls are files that have been set by the operating system to open at startup.
	OpenFls []string `label:"Open files"`

	// Quitting is whether the app is quitting and thus closing all of the windows
	Quitting bool

	// QuitReqFunc is a function to call when a quit is requested
	QuitReqFunc func()

	// QuitCleanFuncs are functions to call when the app is about to quit
	QuitCleanFuncs []func()

	// Dark is whether the system color theme is dark (as opposed to light)
	Dark bool
}

// Init does basic initialization steps of the given App. It should be called by
// platform-specific implementations of Init with their platform-specific app
// instance and its base App field. Other platform-specific initial configuration
// steps can be called before this.
func Init(a goosi.App, ab *App) {
	ab.This = a
	goosi.TheApp = a
}

func (a *App) MainLoop() {
	a.MainQueue = make(chan FuncRun)
	a.MainDone = make(chan struct{})
	for {
		select {
		case <-a.MainDone:
			return
		case f := <-a.MainQueue:
			f.F()
			if f.Done != nil {
				f.Done <- struct{}{}
			}
		}
	}
}

// RunOnMain runs the given function on the main thread
func (a *App) RunOnMain(f func()) {
	if a.MainQueue == nil {
		f()
		return
	}
	a.This.SendEmptyEvent()
	done := make(chan struct{})
	a.MainQueue <- FuncRun{F: f, Done: done}
	<-done
	a.This.SendEmptyEvent()
}

// SendEmptyEvent sends an empty, blank event to global event processing
// system, which has the effect of pushing the system along during cases when
// the event loop needs to be "pinged" to get things moving along..
func (a *App) SendEmptyEvent() {
	// no-op by default
}

// StopMain stops the main loop and thus terminates the app
func (a *App) StopMain() {
	a.MainDone <- struct{}{}
}

func (a *App) Name() string {
	return a.Nm
}

func (a *App) SetName(name string) {
	a.Nm = name
}

func (a *App) About() string {
	return a.Abt
}

func (a *App) SetAbout(about string) {
	a.Abt = about
}

func (a *App) OpenFiles() []string {
	return a.OpenFls
}

func (a *App) CogentCoreDataDir() string {
	pdir := filepath.Join(a.This.DataDir(), "CogentCore")
	os.MkdirAll(pdir, 0755)
	return pdir
}

func (a *App) SetQuitReqFunc(fun func()) {
	a.QuitReqFunc = fun
}

func (a *App) AddQuitCleanFunc(fun func()) {
	a.QuitCleanFuncs = append(a.QuitCleanFuncs, fun)
}

func (a *App) QuitReq() {
	if a.Quitting {
		return
	}
	if a.QuitReqFunc != nil {
		a.QuitReqFunc()
	} else {
		a.Quit()
	}
}

func (a *App) IsQuitting() bool {
	return a.Quitting
}

func (a *App) Quit() {
	if a.Quitting {
		return
	}
	a.This.QuitClean()
	a.StopMain()
}

func (a *App) IsDark() bool {
	return a.Dark
}

func (a *App) GetScreens() {
	// no-op by default
}

func (a *App) OpenURL(url string) {
	// no-op by default
}

func (a *App) Clipboard(win goosi.Window) goosi.Clipboard {
	// no-op by default
	return &goosi.ClipboardBase{}
}

func (a *App) Cursor(win goosi.Window) cursor.Cursor {
	// no-op by default
	return &cursor.CursorBase{}
}

func (a *App) ShowVirtualKeyboard(typ goosi.VirtualKeyboardTypes) {
	// no-op by default
}

func (a *App) HideVirtualKeyboard() {
	// no-op by default
}
