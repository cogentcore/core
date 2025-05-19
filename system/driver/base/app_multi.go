// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"slices"

	"cogentcore.org/core/system"
)

// AppMulti contains the data and logic common to all implementations of [system.App]
// on multi-window platforms (desktop and offscreen), as opposed to single-window
// platforms (mobile and web), for which you should use [AppSingle]. An AppMulti is associated
// with a corresponding type of [system.Window]. The [system.Window]
// type should embed [WindowMulti].
type AppMulti[W system.Window] struct {
	App

	// Windows are the windows associated with the app
	Windows []W

	// Screens are the screens associated with the app
	Screens []*system.Screen

	// AllScreens is a unique list of all screens ever seen, from which
	// information can be got if something is missing in [AppMulti.Screens]
	AllScreens []*system.Screen

	// CtxWindow is a dynamically set context window used for some operations
	CtxWindow W `label:"Context window"`
}

// NewAppMulti makes a new [AppMulti].
func NewAppMulti[W system.Window]() AppMulti[W] {
	return AppMulti[W]{}
}

func (a *AppMulti[W]) NScreens() int {
	return len(a.Screens)
}

func (a *AppMulti[W]) Screen(n int) *system.Screen {
	if n >= 0 && n < len(a.Screens) {
		return a.Screens[n]
	}
	return a.Screens[0]
}

func (a *AppMulti[W]) ScreenByName(name string) *system.Screen {
	for _, sc := range a.Screens {
		if sc.Name == name {
			return sc
		}
	}
	return nil
}

func (a *AppMulti[W]) NWindows() int {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	return len(a.Windows)
}

func (a *AppMulti[W]) Window(win int) system.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if win < len(a.Windows) {
		return a.Windows[win]
	}
	return nil
}

func (a *AppMulti[W]) WindowByName(name string) system.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	for _, win := range a.Windows {
		if win.Name() == name {
			return win
		}
	}
	return nil
}

func (a *AppMulti[W]) WindowInFocus() system.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	for _, win := range a.Windows {
		if win.Is(system.Focused) {
			return win
		}
	}
	return nil
}

func (a *AppMulti[W]) ContextWindow() system.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	return a.CtxWindow
}

// RemoveWindow removes the given Window from the app's list of windows.
// It does not actually close it; see [Window.Close] for that.
func (a *AppMulti[W]) RemoveWindow(w system.Window) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	a.Windows = slices.DeleteFunc(a.Windows, func(ew W) bool {
		return system.Window(ew) == w
	})
}

func (a *AppMulti[W]) QuitClean() bool {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	for _, qf := range a.QuitCleanFuncs {
		qf()
	}
	nwin := len(a.Windows)
	for i := nwin - 1; i >= 0; i-- {
		win := a.Windows[i]
		// CloseReq calls RemoveWindow, which also Locks, so we must Unlock
		a.Mu.Unlock()
		win.CloseReq()
		a.Mu.Lock()
	}
	return len(a.Windows) == 0
}
