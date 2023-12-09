// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"slices"

	"goki.dev/goosi"
)

// AppMulti contains the data and logic common to all implementations of [goosi.App]
// on multi-window platforms (desktop), as opposed to single-window
// platforms (mobile, web, and offscreen), for which you should use [AppSingle]. An AppMulti is associated
// with a corresponding type of [goosi.Window]. The [goosi.Window]
// type should embed [WindowMulti].
type AppMulti[W goosi.Window] struct {
	App

	// Windows are the windows associated with the app
	Windows []W

	// Screens are the screens associated with the app
	Screens []*goosi.Screen

	// AllScreens is a unique list of all screens ever seen, from which
	// information can be got if something is missing in [AppMulti.Screens]
	AllScreens []*goosi.Screen

	// CtxWindow is a dynamically set context window used for some operations
	CtxWindow W
}

// NewAppMulti makes a new [AppMulti].
func NewAppMulti[W goosi.Window]() AppMulti[W] {
	return AppMulti[W]{}
}

func (a *AppMulti[W]) NScreens() int {
	return len(a.Screens)
}

func (a *AppMulti[W]) Screen(n int) *goosi.Screen {
	if n < len(a.Screens) {
		return a.Screens[n]
	}
	return nil
}

func (a *AppMulti[W]) ScreenByName(name string) *goosi.Screen {
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

func (a *AppMulti[W]) Window(win int) goosi.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if win < len(a.Windows) {
		return a.Windows[win]
	}
	return nil
}

func (a *AppMulti[W]) WindowByName(name string) goosi.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	for _, win := range a.Windows {
		if win.Name() == name {
			return win
		}
	}
	return nil
}

func (a *AppMulti[W]) WindowInFocus() goosi.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	for _, win := range a.Windows {
		if win.Is(goosi.Focused) {
			return win
		}
	}
	return nil
}

func (a *AppMulti[W]) ContextWindow() goosi.Window {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	return a.CtxWindow
}

// RemoveWindow removes the given Window from the app's list of windows.
// It does not actually close it; see [Window.Close] for that.
func (a *AppMulti[W]) RemoveWindow(w goosi.Window) {
	slices.DeleteFunc(a.Windows, func(ew W) bool {
		return goosi.Window(ew) == w
	})
}

func (a *AppMulti[W]) QuitClean() {
	a.Quitting = true
	if a.QuitCleanFunc != nil {
		a.QuitCleanFunc()
	}
	a.Mu.Lock()
	nwin := len(a.Windows)
	for i := nwin - 1; i >= 0; i-- {
		win := a.Windows[i]
		go win.Close()
	}
	a.Mu.Unlock()
	// for i := 0; i < nwin; i++ {
	// 	<-app.QuitCloseCnt
	// }
}
