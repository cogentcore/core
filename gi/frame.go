// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"cogentcore.org/core/abilities"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// Frame is a Layout that renders a background according to the
// background-color style setting, and optional striping for grid layouts
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

// RenderFrame does the standard rendering of the frame itself
func (fr *Frame) RenderFrame() {
	_, st := fr.RenderLock()
	fr.RenderStdBox(st)
	fr.RenderUnlock()
}

func (fr *Frame) Render() {
	if fr.PushBounds() {
		fr.RenderFrame()
		fr.RenderChildren()
		fr.RenderParts()
		fr.RenderScrolls()
		fr.PopBounds()
	}
}
