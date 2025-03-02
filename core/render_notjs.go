// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package core

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/paint/renderers/rasterx"
	"cogentcore.org/core/system/composer"
	"golang.org/x/image/draw"
)

func (ps *paintSource) Draw(c composer.Composer) {
	cd := c.(*composer.ComposerDrawer)
	rd := ps.renderer.(*rasterx.Renderer)

	unchanged := len(ps.render) == 0
	if !unchanged {
		rd.Render(ps.render)
	}
	img := rd.Image()
	cd.Drawer.Copy(ps.drawPos, img, img.Bounds(), ps.drawOp, unchanged)
}

func (ss *scrimSource) Draw(c composer.Composer) {
	cd := c.(*composer.ComposerDrawer)
	clr := colors.Uniform(colors.ApplyOpacity(colors.ToUniform(colors.Scheme.Scrim), 0.5))
	cd.Drawer.Copy(image.Point{}, clr, ss.bbox, draw.Over, composer.Unchanged)
}

func (ss *spritesSource) Draw(c composer.Composer) {
	cd := c.(*composer.ComposerDrawer)
	for _, sd := range ss.sprites {
		cd.Drawer.Copy(sd.drawPos, sd.pixels, sd.pixels.Bounds(), draw.Over, composer.Unchanged)
	}
}
