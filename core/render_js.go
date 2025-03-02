// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package core

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/system/composer"
)

func (ps *paintSource) Draw(c composer.Composer) {
	cw := c.(*composer.ComposerWeb)
	cw.Element(ps, "canvas")
	ps.renderer.Render(ps.render)
}

func (ss *scrimSource) Draw(c composer.Composer) {
	cw := c.(*composer.ComposerWeb)
	clr := colors.ApplyOpacity(colors.ToUniform(colors.Scheme.Scrim), 0.5)
	elem := cw.Element(ss, "div")
	elem.Set("style", "background-color: "+colors.AsHex(clr))
}

/*

// doRender is the implementation of the main render pass on web.
// It ensures that all canvases are properly configured.
func (w *renderWindow) doRender(top *Stage) {
	w.updateCanvases(&w.mains, active)

}

// updateCanvases updates all of the canvases corresponding to the given stages
// and their popups.
func (w *renderWindow) updateCanvases(sm *stages, active map[*htmlcanvas.Renderer]bool) {
	for _, kv := range sm.stack.Order {
		st := kv.Value
		for _, rd := range st.Scene.Painter.Renderers {
			if hc, ok := rd.(*htmlcanvas.Renderer); ok {
				active[hc] = true
				w.updateCanvas(hc, st)
			}
		}
		// If we own popups, update them too.
		if st.Main == st && st.popups != nil {
			w.updateCanvases(st.popups, active)
		}
	}
}

// updateCanvas ensures that the given [htmlcanvas.Renderer] is properly configured.
func (w *renderWindow) updateCanvas(hc *htmlcanvas.Renderer, st *Stage) {
	screen := w.SystemWindow.Screen()

	hc.SetSize(units.UnitDot, math32.FromPoint(st.Scene.SceneGeom.Size))

	style := hc.Canvas.Get("style")

	// Dividing by the DevicePixelRatio in this way avoids rounding errors (CSS
	// supports fractional pixels but HTML doesn't). These rounding errors lead to blurriness on devices
	// with fractional device pixel ratios
	// (see https://github.com/cogentcore/core/issues/779 and
	// https://stackoverflow.com/questions/15661339/how-do-i-fix-blurry-text-in-my-html5-canvas/54027313#54027313)
	style.Set("left", fmt.Sprintf("%gpx", float32(st.Scene.SceneGeom.Pos.X)/screen.DevicePixelRatio))
	style.Set("top", fmt.Sprintf("%gpx", float32(st.Scene.SceneGeom.Pos.Y)/screen.DevicePixelRatio))

	style.Set("width", fmt.Sprintf("%gpx", float32(st.Scene.SceneGeom.Size.X)/screen.DevicePixelRatio))
	style.Set("height", fmt.Sprintf("%gpx", float32(st.Scene.SceneGeom.Size.Y)/screen.DevicePixelRatio))
}
*/
