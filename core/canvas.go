// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/path"
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
	Draw func(pc *paint.Painter)

	// painter is the paint painter used for drawing.
	painter *paint.Painter
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
	c.painter = paint.NewPainter(szp.X, szp.Y)
	c.painter.Paint.Transform = math32.Scale2D(sz.X, sz.Y)
	c.painter.Context().Transform = math32.Scale2D(sz.X, sz.Y)
	c.painter.UnitContext = c.Styles.UnitContext
	c.painter.ToDots()
	c.painter.VectorEffect = path.VectorEffectNonScalingStroke
	c.Draw(c.painter)

	draw.Draw(c.Scene.Pixels, c.Geom.ContentBBox, c.painter.Image, c.Geom.ScrollOffset(), draw.Over)
}
