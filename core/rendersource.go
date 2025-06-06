// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/system/composer"
	"golang.org/x/image/draw"
)

//////// Scene

// SceneSource returns a [composer.Source] for the given scene
// using the given suggested draw operation.
func SceneSource(sc *Scene, op draw.Op) composer.Source {
	if sc.Painter.State == nil || sc.renderer == nil {
		return nil
	}
	render := sc.Painter.RenderDone()
	return &paintSource{render: render, renderer: sc.renderer, drawOp: op, drawPos: sc.SceneGeom.Pos}
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
func SpritesSource(stage *Stage, mainScene *Scene) composer.Source {
	stage.Sprites.Lock()
	defer stage.Sprites.Unlock()

	sz := math32.FromPoint(mainScene.SceneGeom.Size)
	if stage.spritePainter == nil || stage.spritePainter.State.Size != sz {
		stage.spritePainter = paint.NewPainter(sz)
		stage.spritePainter.Paint.UnitContext = mainScene.Styles.UnitContext
		stage.spriteRenderer = paint.NewSourceRenderer(sz)
	}
	pc := stage.spritePainter
	pc.Fill.Color = colors.Uniform(colors.Transparent)
	pc.Clear()
	stage.Sprites.Do(func(sl *SpriteList) {
		for _, sp := range sl.Values {
			if sp == nil || !sp.Active || sp.Draw == nil {
				continue
			}
			sp.Draw(stage.spritePainter)
		}
	})
	stage.Sprites.modified = false
	render := stage.spritePainter.RenderDone()
	return &paintSource{render: render, renderer: stage.spriteRenderer, drawOp: draw.Over, drawPos: image.Point{}}
}
