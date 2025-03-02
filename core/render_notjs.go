// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package core

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/paint/renderers/rasterx"
	"cogentcore.org/core/system/composer"
	"golang.org/x/image/draw"
)

// SceneSource returns the [composer.Source] for given scene
// using given suggested draw operation.
func SceneSource(sc *Scene, op draw.Op) composer.Source {
	if sc.Painter.State == nil || len(sc.Painter.State.Renderers) == 0 {
		return nil
	}
	rd, ok := sc.Painter.State.Renderers[0].(*rasterx.Renderer)
	if !ok {
		return nil
	}
	render := sc.Painter.RenderDone()
	rs := &painterSource{render: render, renderer: rd, drawOp: op, drawPos: sc.SceneGeom.Pos}
	return rs
}

// painterSource is the [composer.Source] for [paint.Painter] content.
type painterSource struct {

	// render is the render content.
	render render.Render

	// renderer is the renderer for drawing the painter content
	renderer *rasterx.Renderer

	// DrawOp is the [draw.Op] operation: [draw.Src] to copy source,
	// [draw.Over] to alpha blend.
	drawOp draw.Op

	// DrawPos is the position offset for the [Image] renderer to
	// use in its Draw to a [system.Drawer] (i.e., the [core.Scene] position).
	drawPos image.Point
}

func (ps *painterSource) Draw(c composer.Composer) {
	cd, ok := c.(*composer.ComposerDrawer)
	if !ok {
		return
	}
	unchanged := len(ps.render) == 0
	if !unchanged {
		ps.renderer.Render(ps.render)
	}
	img := ps.renderer.Image()
	cd.Drawer.Copy(ps.drawPos, img, img.Bounds(), ps.drawOp, unchanged)
}

//////// Scrim

func ScrimSource(bbox image.Rectangle) composer.Source {
	return &scrimSource{bbox: bbox}
}

// scrimSource is a [composer.Source] implementation for scrim.
type scrimSource struct {
	bbox image.Rectangle
}

func (sr *scrimSource) Draw(c composer.Composer) {
	cd, ok := c.(*composer.ComposerDrawer)
	if !ok {
		return
	}
	clr := colors.Uniform(colors.ApplyOpacity(colors.ToUniform(colors.Scheme.Scrim), 0.5))
	cd.Drawer.Copy(image.Point{}, clr, sr.bbox, draw.Over, composer.Unchanged)
}

//////// Sprites

// SpritesSource returns a [composer.Source] for rendering Sprites
func SpritesSource(ss *Sprites, scpos image.Point) composer.Source {
	sr := &spriteSource{}
	sr.sprites = make([]spriteRender, 0, len(ss.Order))
	for _, kv := range ss.Order {
		sp := kv.Value
		if !sp.Active {
			continue
		}
		// note: may need to copy pixels but hoping not..
		sd := spriteRender{drawPos: sp.Geom.Pos.Add(scpos), pixels: sp.Pixels}
		sr.sprites = append(sr.sprites, sd)
	}
	ss.Modified = false
	return sr
}

// spriteSource is a [composer.Source] implementation for sprites.
type spriteSource struct {
	sprites []spriteRender
}

func (sr *spriteSource) Draw(c composer.Composer) {
	cd, ok := c.(*composer.ComposerDrawer)
	if !ok {
		return
	}
	for _, sd := range sr.sprites {
		cd.Drawer.Copy(sd.drawPos, sd.pixels, sd.pixels.Bounds(), draw.Over, composer.Unchanged)
	}
}
