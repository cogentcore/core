// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"golang.org/x/image/draw"
)

// Canvas is a widget that can be arbitrarily drawn to by setting
// its Draw function using [Canvas.SetDraw].
type Canvas struct {
	WidgetBase

	// Draw is the function used to draw the content of the
	// canvas every time that it is rendered. The paint context
	// is automatically normalized to the size of the canvas,
	// so you should specify points on a 0-1 scale.
	Draw func(pc *paint.Context)

	// context is the paint context used for drawing.
	context *paint.Context
}

func (c *Canvas) Init() {
	c.WidgetBase.Init()
	c.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(256))
	})
}

func (c *Canvas) Render() {
	c.WidgetBase.Render()

	sz := c.Geom.Size.Actual.Content
	szp := c.Geom.Size.Actual.Content.ToPoint()
	c.context = paint.NewContext(szp.X, szp.Y)
	c.context.UnitContext = c.Styles.UnitContext
	c.context.ToDots()
	c.context.PushTransform(math32.Scale2D(sz.X, sz.Y))
	c.context.VectorEffect = styles.VectorEffectNonScalingStroke
	c.Draw(c.context)

	draw.Draw(c.Scene.Pixels, c.Geom.ContentBBox, c.context.Image, c.Geom.ScrollOffset(), draw.Over)
}
