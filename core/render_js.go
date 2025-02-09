// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package core

import (
	"fmt"

	"cogentcore.org/core/paint/renderers/htmlcanvas"
)

// doRender is the implementation of the main render pass on web.
// It ensures that all canvases are properly configured.
func (w *renderWindow) doRender(top *Stage) {
	w.updateCanvases(&w.mains)
}

// updateCanvases updates all of the canvases corresponding to the given stages
// and their popups.
func (w *renderWindow) updateCanvases(sm *stages) {
	for _, kv := range sm.stack.Order {
		st := kv.Value
		for _, rd := range st.Scene.Painter.Renderers {
			if hc, ok := rd.(*htmlcanvas.Renderer); ok {
				w.updateCanvas(hc, st)
			}
		}
		// If we own popups, update them too.
		if st.Main == st && st.popups != nil {
			w.updateCanvases(st.popups)
		}
	}
}

// updateCanvas ensures that the given [htmlcanvas.Renderer] is properly configured.
func (w *renderWindow) updateCanvas(hc *htmlcanvas.Renderer, st *Stage) {
	// screen := w.SystemWindow.Screen()
	style := hc.Canvas.Get("style")
	style.Set("left", fmt.Sprintf("%dpx", st.Scene.SceneGeom.Pos.X))
	style.Set("top", fmt.Sprintf("%dpx", st.Scene.SceneGeom.Pos.Y))
}
