// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/mat32/v2"
)

func main() { gimain.Run(app) }

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

var (
	ShortText = "This is a test of layout."

	LongText = "This is a test of the layout logic, which is pretty complex and requires some experimenting to understand how it all works.  The styling and behavior is the same as the CSS / HTML Flex model, except we only support Grow, not Shrink. "

	VeryLongText = LongText + LongText + LongText

	AlignText = "This is text to test for text align<br>This line is short<br>This is text to test for text align, this one is longer"

	FrameSizes = [5]mat32.Vec2{
		{20, 100},
		{80, 20},
		{60, 80},
		{40, 120},
		{150, 100},
	}
)

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
		s.Border.Color.Set(colors.Black)
		s.Border.Width.Set(units.Dp(2))
	})
	return fr
}

func app() {
	// turn on tracing in preferences, Debug
	// gi.LayoutTrace = true
	// gi.LayoutTraceDetail = true
	// gi.UpdateTrace = true
	// gi.RenderTrace = true

	gi.SetAppName("layout")
	gi.SetAppAbout(`This is a demo of the layout functions in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	b := gi.NewBody("lay-test").SetTitle("GoGi Layout Test")
	// gi.DefaultTopAppBar = nil // note: comment out for dialog tests..

	ctrl := &Control{}
	splt := gi.NewSplits(b)

	svfr := gi.NewFrame(splt).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	giv.NewStructView(svfr).SetStruct(ctrl)

	fr := gi.NewFrame(splt)
	PlainFrames(fr, mat32.Vec2{0, 0})
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

	gi.NewButton(svfr).SetText("Redraw").OnClick(func(e events.Event) {
		fr.Update()
	})

	b.NewWindow().Run().Wait()
}
