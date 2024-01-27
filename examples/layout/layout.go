// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

type Control struct {
	// Display controls how items are displayed, in terms of layout
	Display styles.Displays

	// Direction specifies the order elements are organized:
	// Row is horizontal, Col is vertical.
	// See also [Wrap]
	Direction styles.Directions

	// Wrap causes elements to wrap around in the CrossAxis dimension
	// to fit within sizing constraints (on by default).
	Wrap bool

	// Justify specifies the distribution of elements along the main axis,
	// i.e., the same as Direction, for Flex Display.  For Grid, the main axis is
	// given by the writing direction (e.g., Row-wise for latin based languages).
	Justify styles.AlignSet `view:"inline"`

	// Align specifies the cross-axis alignment of elements, orthogonal to the
	// main Direction axis. For Grid, the cross-axis is orthogonal to the
	// writing direction (e.g., Column-wise for latin based languages).
	Align styles.AlignSet `view:"inline"`

	// Min is the minimum size of the actual content, exclusive of additional space
	// from padding, border, margin; 0 = default is sum of Min for all content
	// (which _includes_ space for all sub-elements).
	// This is equivalent to the Basis for the CSS flex styling model.
	Min units.XY `view:"inline"`

	// Max is the maximum size of the actual content, exclusive of additional space
	// from padding, border, margin; 0 = default provides no Max size constraint
	Max units.XY `view:"inline"`

	// Grow is the proportional amount that the element can grow (stretch)
	// if there is more space available.  0 = default = no growth.
	// Extra available space is allocated as: Grow / sum (all Grow).
	// Important: grow elements absorb available space and thus are not
	// subject to alignment (Center, End).
	Grow mat32.Vec2
}

var FrameSizes = [5]mat32.Vec2{
	{20, 100},
	{80, 20},
	{60, 80},
	{40, 120},
	{150, 100},
}

func PlainFrames(par gi.Widget, grow mat32.Vec2) {
	for i, sz := range FrameSizes {
		i := i
		sz := sz
		nm := fmt.Sprintf("fr%v", i)
		fr := BoxFrame(par, nm)
		fr.Style(func(s *styles.Style) {
			s.Min.X.Px(sz.X)
			s.Min.Y.Px(sz.Y)
			s.Grow = grow
			// s.Align.X = styles.Center
		})
	}
}

func BoxFrame(par gi.Widget, nm ...string) *gi.Frame {
	fr := gi.NewFrame(par, nm...)
	fr.Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Scheme.SurfaceContainerHighest)
	})
	return fr
}

func main() {
	b := gi.NewBody("Cogent Core Layout Demo")

	ctrl := &Control{}
	ctrl.Grow.Set(1, 1)
	splt := gi.NewSplits(b)

	svfr := gi.NewFrame(splt).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	sv := giv.NewStructView(svfr).SetStruct(ctrl)

	fr := gi.NewFrame(splt)
	PlainFrames(fr, mat32.V2(0, 0))
	fr.Style(func(s *styles.Style) {
		s.Display = ctrl.Display
		s.Direction = ctrl.Direction
		s.Wrap = ctrl.Wrap
		s.Justify = ctrl.Justify
		s.Align = ctrl.Align
		s.Min = ctrl.Min
		s.Max = ctrl.Max
		s.Grow = ctrl.Grow
	})

	sv.OnChange(func(e events.Event) {
		if ctrl.Display != styles.Stacked { // stacked will make all but top invisible
			fr.WidgetKidsIter(func(i int, kwi gi.Widget, kwb *gi.WidgetBase) bool {
				kwb.SetState(false, states.Invisible)
				return ki.Continue
			})
		}
		fr.Update()
	})

	b.RunMainWindow()
}
