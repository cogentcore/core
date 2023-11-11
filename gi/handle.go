// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/girl/abilities"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/mat32/v2"
)

// Handle represents a draggable handle that can be
// used to control the size of an element.
type Handle struct {
	Frame

	// dimension along which the handle slides (opposite of the dimension it is longest on)
	Dim mat32.Dims

	// Min is the minimum value that the handle can go to
	// (typically the lower bound of the dialog/splits)
	Min float32
	// Max is the maximum value that the handle can go to
	// (typically the upper bound of the dialog/splits)
	Max float32
	// Pos is the current position of the handle on the
	// scale of [Handle.Min] to [Handle.Max]
	Pos float32
}

func (hl *Handle) OnInit() {
	hl.HandleEvents()
	hl.HandleStyles()
}

func (hl *Handle) HandleStyles() {
	hl.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Focusable, abilities.Hoverable, abilities.Slideable)

		s.Border.Radius = styles.BorderRadiusFull
		s.BackgroundColor.SetSolid(colors.Scheme.OutlineVariant)

		// TODO(kai/margin): this is intentionally designed to work with the
		// current way margin is handled, but will break after it is fixed.
		// Width/height should actually be 6dp, not 14dp.
		if hl.Dim == mat32.X {
			s.Min.X.Dp(14)
			s.Min.Y.Em(3)
			s.Margin.SetHoriz(units.Dp(4))
		} else {
			s.Min.X.Em(3)
			s.Min.Y.Dp(14)
			s.Margin.SetVert(units.Dp(4))
		}

		if !hl.IsReadOnly() {
			if hl.Dim == mat32.X {
				s.Cursor = cursors.ResizeEW
			} else {
				s.Cursor = cursors.ResizeNS
			}
		}
	})
}

func (hl *Handle) HandleEvents() {
	hl.On(events.SlideMove, func(e events.Event) {
		hl.Pos = mat32.NewVec2FmPoint(e.Pos()).Dim(hl.Dim)
		hl.Send(events.Change, e)
	})
}

// Value returns the value on a normalized scale of 0-1,
// based on [Handle.Pos], [Handle.Min], and [Handle.Max].
func (hl *Handle) Value() float32 {
	return hl.Pos / (hl.Max - hl.Min)
}
