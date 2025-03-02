// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"

	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/system/composer"
	"golang.org/x/image/draw"
)

//////// Scene

// SceneSource returns a [composer.Source] for the given scene
// using the given suggested draw operation.
func SceneSource(sc *Scene, op draw.Op) composer.Source {
	if sc.Painter.State == nil || len(sc.Painter.State.Renderers) == 0 {
		return nil
	}
	rd := sc.Painter.State.Renderers[0]
	render := sc.Painter.RenderDone()
	return &paintSource{render: render, renderer: rd, drawOp: op, drawPos: sc.SceneGeom.Pos}
}

// paintSource is the [composer.Source] for [paint.Painter] content, such as for a [Scene].
type paintSource struct {

	// render is the render content.
	render render.Render

	// renderer is the renderer for drawing the painter content.
	renderer render.Renderer

	// drawOp is the [draw.Op] operation: [draw.Src] to copy source,
	// [draw.Over] to alpha blend.
	drawOp draw.Op

	// drawPos is the position offset for the [Image] renderer to
	// use in its Draw to a [composer.Drawer] (i.e., the [Scene] position).
	drawPos image.Point
}

//////// Scrim

// ScrimSource returns a [composer.Source] for a scrim with the given bounding box.
func ScrimSource(bbox image.Rectangle) composer.Source {
	return &scrimSource{bbox: bbox}
}

// scrimSource is a [composer.Source] implementation for a scrim.
type scrimSource struct {
	bbox image.Rectangle
}

//////// Sprites

// SpritesSource returns a [composer.Source] for rendering [Sprites].
func SpritesSource(sprites *Sprites, scpos image.Point) composer.Source {
	ss := &spritesSource{}
	ss.sprites = make([]spriteRender, 0, len(sprites.Order))
	for _, kv := range sprites.Order {
		sp := kv.Value
		if !sp.Active {
			continue
		}
		// note: may need to copy pixels but hoping not..
		sd := spriteRender{drawPos: sp.Geom.Pos.Add(scpos), pixels: sp.Pixels}
		ss.sprites = append(ss.sprites, sd)
	}
	sprites.Modified = false
	return ss
}

// spritesSource is a [composer.Source] implementation for [Sprites].
type spritesSource struct {
	sprites []spriteRender
}
