// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/mat32/v2"
)

// Handle represents a draggable handle that can be
// used to control the size of an element.
type Handle struct {
	Frame

	// dimension along which the handle slides
	Dim mat32.Dims
}

func (hl *Handle) OnInit() {
	hl.HandleStyles()
}

func (hl *Handle) HandleStyles() {
	hl.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Focusable, abilities.Hoverable, abilities.Slideable)

		s.Border.Radius = styles.BorderRadiusFull
		s.BackgroundColor.SetSolid(colors.Scheme.OutlineVariant)

		if hl.Dim == mat32.X {
			s.SetFixedWidth(units.Em(3))
			s.SetFixedHeight(units.Dp(4))
		} else {
			s.SetFixedWidth(units.Dp(4))
			s.SetFixedHeight(units.Em(3))
		}

		if !hl.IsReadOnly() {
			s.Cursor = cursors.Grab
			switch {
			case s.Is(states.Sliding):
				s.Cursor = cursors.Grabbing
			case s.Is(states.Active):
				s.Cursor = cursors.Grabbing
			}
		}
	})
}
