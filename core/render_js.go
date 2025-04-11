// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package core

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/paint/renderers/htmlcanvas"
	"cogentcore.org/core/system/composer"
)

func (ps *paintSource) Draw(c composer.Composer) {
	cw := c.(*composer.ComposerWeb)
	rd := ps.renderer.(*htmlcanvas.Renderer)

	elem := cw.Element(ps, "canvas")
	_, size := ps.renderer.Size()
	cw.SetElementGeom(elem, ps.drawPos, size.ToPoint())
	rd.SetCanvas(elem)
	ps.renderer.Render(ps.render)
}

func (ss *scrimSource) Draw(c composer.Composer) {
	cw := c.(*composer.ComposerWeb)
	clr := colors.ApplyOpacity(colors.ToUniform(colors.Scheme.Scrim), 0.5)
	elem := cw.Element(ss, "div")
	cw.SetElementGeom(elem, ss.bbox.Min, ss.bbox.Size())
	elem.Get("style").Set("backgroundColor", colors.AsHex(clr))
}

func (ss *spritesSource) Draw(c composer.Composer) {
	cw := c.(*composer.ComposerWeb)
	for _, sr := range ss.sprites {
		elem := cw.Element(ss, "div") // TODO: support full images
		if !sr.active {
			elem.Get("style").Set("display", "none")
			continue
		}
		elem.Get("style").Set("display", "initial")
		cw.SetElementGeom(elem, sr.drawPos, sr.pixels.Bounds().Size())
		elem.Get("style").Set("backgroundColor", colors.AsHex(colors.ToUniform(sr.pixels)))
	}
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


}
*/
