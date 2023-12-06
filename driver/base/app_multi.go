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

func (app *AppMulti[W]) NScreens() int {
	return len(app.Screens)
}

func (app *AppMulti[W]) Screen(n int) *goosi.Screen {
	if n < len(app.Screens) {
		return app.Screens[n]
	}
	return nil
}

func (app *AppMulti[W]) ScreenByName(name string) *goosi.Screen {
	for _, sc := range app.Screens {
		if sc.Name == name {
			return sc
		}
	}
	return nil
}

func (app *AppMulti[W]) NWindows() int {
	app.Mu.Lock()
	defer app.Mu.Unlock()
	return len(app.Windows)
}

func (app *AppMulti[W]) Window(win int) goosi.Window {
	app.Mu.Lock()
	defer app.Mu.Unlock()
	if win < len(app.Windows) {
		return app.Windows[win]
	}
	return nil
}

func (app *AppMulti[W]) WindowByName(name string) goosi.Window {
	app.Mu.Lock()
	defer app.Mu.Unlock()
	for _, win := range app.Windows {
		if win.Name() == name {
			return win
		}
	}
	return nil
}

func (app *AppMulti[W]) WindowInFocus() goosi.Window {
	app.Mu.Lock()
	defer app.Mu.Unlock()
	for _, win := range app.Windows {
		if win.IsFocus() {
			return win
		}
	}
	return nil
}

func (app *AppMulti[W]) ContextWindow() goosi.Window {
	app.Mu.Lock()
	defer app.Mu.Unlock()
	return app.CtxWindow
}

// DeleteWin removes the given window from the list of windows.
func (a *AppMulti[W]) DeleteWin(w W) {
	slices.DeleteFunc(a.Windows, func(ew W) bool {
		return goosi.Window(ew) == goosi.Window(w)
	})
}
