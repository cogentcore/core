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

func (w *renderWindow) fillInsets(c composer.Composer) {
	// no-op
}
