// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package base provides base driver types that platform-specific drivers can extend
// to implement interfaces defined in package goosi.
package base

import (
	"sync"

	"goki.dev/goosi"
)

// App contains the data and logic common to all implementations of [goosi.App].
type App struct {
	Ths goosi.App

	// Mu is the main mutex protecting access to app operations, including [App.RunOnMain] calls.
	Mu sync.Mutex

	// MainQueue is the queue of functions to call on the main loop. To add to it, use [App.RunOnMain].
	MainQueue chan FuncRun

	// MainDone is a channel on which is a signal is sent when the main loop of the app should be terminated.
	MainDone chan struct{}

	// Nm is the name of the app.
	Nm string

	// Abt is the about information for the app.
	Abt string

	// Quitting is whether the app is quitting and thus closing all of the windows
	Quitting bool

	// QuitReqFunc is a function to call when a quit is requested
	QuitReqFunc func()

	// QuitCleanFunc is a function to call when the app is about to quit
	QuitCleanFunc func()

	// Dark is whether the system color theme is dark (as opposed to light)
	Dark bool
}

// FuncRun is a simple helper type that contains a function to call and a channel
// to send a signal on when the function is finished running.
type FuncRun struct {
	F    func()
	Done chan struct{}
}

// RunOnMain runs the given function on the main thread
func (a *App) RunOnMain(f func()) {
	if a.MainQueue == nil {
		f()
	} else {
		done := make(chan struct{})
		a.MainQueue <- FuncRun{F: f, Done: done}
		<-done
	}
}

// GoRunOnMain runs the given function on the main thread and returns immediately
func (a *App) GoRunOnMain(f func()) {
	go func() {
		a.MainQueue <- FuncRun{F: f, Done: nil}
	}()
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
	a.Ths.QuitClean()
	a.StopMain()
}
