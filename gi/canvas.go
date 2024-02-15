// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
	"golang.org/x/image/draw"
)

// Canvas is a widget that can be arbitrarily drawn to by setting
// its Draw function using [Canvas.SetDraw].
type Canvas struct {
	Box

	// Draw is the function used to draw the content of the
	// canvas every time that it is rendered. It renders directly
	// to an image the size of the widget in real pixels (dots).
	// The image is 256dp by 256dp by default. You can access the
	// size of it in pixels by reading the bounds of pc.Image.
	Draw func(pc *paint.Context)

	// Context is the paint context that we use for drawing.
	Context *paint.Context `set:"-"`
}

func (c *Canvas) OnInit() {
	c.Box.OnInit()
	c.SetStyles()
}

func (c *Canvas) SetStyles() {
	c.Style(func(s *styles.Style) {
		s.Min.Set(units.Dp(256))
	})
}

func (c *Canvas) DrawIntoScene() {
	c.Context = paint.NewContext(c.Geom.ContentBBox.Dx(), c.Geom.ContentBBox.Dy())
	c.Context.Lock()
	c.Draw(c.Context)
	c.Context.Unlock()

	draw.Draw(c.Scene.Pixels, c.Geom.ContentBBox, c.Context.Image, image.Point{}, draw.Over)
}

func (c *Canvas) Render() {
	if c.PushBounds() {
		c.RenderBox()
		c.DrawIntoScene()
		c.RenderChildren()
		c.PopBounds()
	}
}
