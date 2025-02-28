// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package core

import "cogentcore.org/core/paint/render"

// doRender is the implementation of the main render pass on non-web platforms.
// This is called in a separate goroutine.
func (w *renderWindow) doRender(rs render.Scene) {
	w.renderMu.Lock()
	w.flags.SetFlag(true, winIsRendering)
	defer func() {
		w.flags.SetFlag(false, winIsRendering)
		w.renderMu.Unlock()
	}()

	drw := w.SystemWindow.Drawer()
	drw.Start()
	w.fillInsets()

	for _, r := range rs {
		r.Render()
		r.Draw(drw)
	}

	drw.End()
}
