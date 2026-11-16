// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
	"cogentcore.org/core/system/driver/base"
)

// Window is the implementation of [system.Window] for the offscreen platform.
type Window struct {
	base.WindowMulti[*App, *composer.ComposerDrawer]
}

func (w *Window) Screen() *system.Screen {
	return TheApp.Screen(0)
}

// Close closes the window, once, as on desktop platforms: it is closed
// (see [base.Window.IsClosed]) right after the close event is sent, so
// that an event loop that closes its own window gets the close event
// once the window is closed, and can no longer lock it.
func (w *Window) Close() {
	if !w.MarkClosing() {
		return
	}
	w.CloseBase()
	w.Mu.Lock()
	w.Compose = nil
	w.Mu.Unlock()
}
