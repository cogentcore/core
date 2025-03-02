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

// painterSource is the [composer.Source] for [paint.Painter] content.
type painterSource[R render.Renderer] struct {

	// render is the render content.
	render render.Render

	// renderer is the renderer for drawing the painter content.
	renderer R

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

// scrimSource is a [composer.Source] implementation for scrim.
type scrimSource struct {
	bbox image.Rectangle
}
