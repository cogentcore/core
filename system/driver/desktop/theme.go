// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"cogentcore.org/core/events"
)

// IsDark returns whether the system color theme is dark (as opposed to light).
// It is kept up to date by the platform-specific theme watcher started by
// [App.watchTheme].
func (a *App) IsDark() bool {
	return a.dark.Load()
}

// setDark sets whether the system color theme is dark. If it changed, it sends
// an [events.ScreenUpdate] event to the first window, which updates the color
// scheme when the Auto theme is used. It can be called on any thread.
func (a *App) setDark(dark bool) {
	if a.dark.Swap(dark) == dark {
		return
	}
	// not on this thread, which may be the main thread, as Window locks App.Mu
	go func() {
		if w := a.Window(0); w != nil {
			w.Events().Window(events.ScreenUpdate)
		}
	}()
}
