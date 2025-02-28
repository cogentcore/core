// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
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
	c.painter = &c.Scene.Painter
	sty := styles.NewPaint()
	sty.Transform = math32.Translate2D(c.Geom.Pos.Content.X, c.Geom.Pos.Content.Y).Scale(sz.X, sz.Y)
	c.painter.PushContext(sty, nil)
	c.painter.VectorEffect = ppath.VectorEffectNonScalingStroke
	c.Draw(c.painter)
	c.painter.PopContext()
}
