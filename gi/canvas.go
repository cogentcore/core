// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"cogentcore.org/core/paint"
	"golang.org/x/image/draw"
)

// Canvas is a widget that can be arbitrarily drawn to.
type Canvas struct {
	WidgetBase

	// Context is the paint context that we use for drawing.
	Context paint.Context `set:"-"`
}

// Draw draws to the canvas by calling the given function with its paint context.
func (c *Canvas) Draw(f func(pc *paint.Context)) {
	f(&c.Context)
	c.SetNeedsRender(true)
}

func (c *Canvas) DrawIntoScene() {
	draw.Draw(c.Scene.Pixels, c.Geom.ContentBBox, c.Context.Image, image.Point{}, draw.Over)
}

func (c *Canvas) Render() {
	if c.PushBounds() {
		c.DrawIntoScene()
		c.RenderChildren()
		c.PopBounds()
	}
}
