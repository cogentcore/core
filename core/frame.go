// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/units"
)

// Frame is a [Layout] that also renders the standard box model.
// By default, [Frame]s grow in both the x and y directions; this
// can be changed by setting [styles.Style.Grow].
type Frame struct {
	Layout
}

func (fr *Frame) OnInit() {
	fr.Layout.HandleEvents()
	fr.SetStyles()
}

func (fr *Frame) SetStyles() {
	fr.Style(func(s *styles.Style) {
		// note: using Clickable here so we get clicks, but don't change to Active state.
		// getting clicks allows us to clear focus on click.
		s.SetAbilities(true, abilities.Clickable)
		s.Padding.Set(units.Dp(2))
		s.Grow.Set(1, 1)
		// we never want borders on frames
		s.MaxBorder = styles.Border{}
	})
	fr.StyleFinal(func(s *styles.Style) {
		s.SetAbilities(s.Overflow.X == styles.OverflowAuto || s.Overflow.Y == styles.OverflowAuto, abilities.Scrollable, abilities.Slideable)
	})
}

func (fr *Frame) Render() {
	fr.RenderStandardBox()
}

func (fr *Frame) RenderWidget() {
	if fr.PushBounds() {
		fr.Render()
		fr.RenderChildren()
		fr.RenderParts()
		fr.RenderScrolls()
		fr.PopBounds()
	}
}
