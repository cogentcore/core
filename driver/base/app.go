// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package base provides base driver types that platform-specific drivers can extend
// to implement interfaces defined in package goosi.
package base

import (
	"os"
	"path/filepath"
	"sync"

	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/cursor"
)

// App contains the data and logic common to all implementations of [goosi.App].
type App struct { //gti:add
	// This is the App as a [goosi.App] interface, which preserves the actual identity
	// of the app when calling interface methods in the base App.
	This goosi.App `view:"-"`

	// Mu is the main mutex protecting access to app operations, including [App.RunOnMain] functions.
	Mu sync.Mutex `view:"-"`

	// MainQueue is the queue of functions to call on the main loop. To add to it, use [App.RunOnMain].
	MainQueue chan FuncRun `view:"-"`

	// MainDone is a channel on which is a signal is sent when the main loop of the app should be terminated.
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

	// QuitCleanFunc is a function to call when the app is about to quit
	QuitCleanFunc func()

	// Dark is whether the system color theme is dark (as opposed to light)
	Dark bool
}

// Main is called from main thread when it is time to start running the
// main loop. When function f returns, the app ends automatically.
//
// This version of Main should be called by platform-specific implementations
// of Main with their platform-specific app instance and its base App field.
// Other platform-specific initial configuration steps can be called before this.
func Main(f func(a goosi.App), a goosi.App, ab *App) {
	defer func() { HandleRecover(recover()) }()
	ab.This = a
	goosi.TheApp = a
	go func() {
		f(a)
		ab.StopMain()
	}()
	a.MainLoop()
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
	} else {
		a.This.SendEmptyEvent()
		done := make(chan struct{})
		a.MainQueue <- FuncRun{F: f, Done: done}
		<-done
		a.This.SendEmptyEvent()
	}
}

// GoRunOnMain runs the given function on the main thread and returns immediately
func (a *App) GoRunOnMain(f func()) {
	go func() {
		a.This.SendEmptyEvent()
		a.MainQueue <- FuncRun{F: f, Done: nil}
		a.This.SendEmptyEvent()
	}()
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

func (a *App) GoGiDataDir() string {
	pdir := filepath.Join(a.This.DataDir(), "GoGi")
	os.MkdirAll(pdir, 0755)
	return pdir
}

func (a *App) AppDataDir() string {
	pdir := filepath.Join(a.This.DataDir(), a.Name())
	os.MkdirAll(pdir, 0755)
	return pdir
}

func (a *App) SetQuitReqFunc(fun func()) {
	a.QuitReqFunc = fun
}

func (a *App) SetQuitCleanFunc(fun func()) {
	a.QuitCleanFunc = fun
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

func (a *App) ClipBoard(win goosi.Window) clip.Board {
	// no-op by default
	return &clip.BoardBase{}
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
