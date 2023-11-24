// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log"

	"goki.dev/girl/abilities"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
)

// Frame is a Layout that renders a background according to the
// background-color style setting, and optional striping for grid layouts
type Frame struct {
	Layout

	// options for striped backgrounds -- rendered as darker bands relative to background color
	Stripes Stripes
}

func (fr *Frame) CopyFieldsFrom(frm any) {
	cp, ok := frm.(*Frame)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier Frame one\n", fr.KiType().Name)
		return
	}
	fr.Layout.CopyFieldsFrom(&cp.Layout)
	fr.Stripes = cp.Stripes
}

func (fr *Frame) OnInit() {
	fr.HandleLayoutEvents()
	fr.FrameStyles()
}

func (fr *Frame) FrameStyles() {
	fr.Style(func(s *styles.Style) {
		// note: using Pressable here so we get clicks, but don't change to Active state.
		// getting clicks allows us to clear focus on click.
		s.SetAbilities(true, abilities.Pressable, abilities.FocusWithinable)
		s.Border.Style.Set(styles.BorderNone)
		s.Border.Radius.Zero()
		s.Padding.Set(units.Dp(2))
		s.Grow.Set(1, 1)
		// we never want borders on frames
		s.MaxBorder = styles.Border{}
	})
}

// Stripes defines stripes options for elements that can render striped backgrounds
type Stripes int32 //enums:enum

const (
	NoStripes Stripes = iota
	RowStripes
	ColStripes
)

// FrameStdRender does the standard rendering of the frame itself
func (fr *Frame) FrameStdRender() {
	rs, _, st := fr.RenderLock()
	defer fr.RenderUnlock(rs)

	fr.RenderStdBox(st)
	//	if fr.Lay == LayoutGrid && fr.Stripes != NoStripes && Prefs.Params.ZebraStripeWeight != 0 {
	//		fr.RenderStripes(sc)
	//	}
}

func (fr *Frame) RenderStripes() {
	/*
		st := &fr.Styles
		rs := &sc.RenderState
		pc := &rs.Paint

		pos := fr.Geom.Pos
		sz := fr.Geom.Size.Actual.Content

		delta := fr.LayoutScrollDelta(image.Point{})

		// TODO: fix stripes
		// hic := st.BackgroundColor.Solid.Highlight(Prefs.Params.ZebraStripeWeight)
		hic := st.BackgroundColor.Solid
		if fr.Stripes == RowStripes {
			for r, gd := range fr.GridData[Row] {
				if r%2 == 0 {
					continue
				}
				pry := float32(delta.Y) + gd.AllocPosRel
				szy := gd.AllocSize
				if pry+szy < 0 || pry > sz.Y {
					continue
				}
				pr := pos
				pr.Y += pry
				sr := sz
				sr.Y = szy
				pc.FillBoxColor(rs, pr, sr, hic)
			}
		} else if fr.Stripes == ColStripes {
			for c, gd := range fr.GridData[Col] {
				if c%2 == 0 {
					continue
				}
				prx := float32(delta.X) + gd.AllocPosRel
				szx := gd.AllocSize
				if prx+szx < 0 || prx > sz.X {
					continue
				}
				pr := pos
				pr.X += prx
				sr := sz
				sr.X = szx
				pc.FillBoxColor(rs, pr, sr, hic)
			}
		}
	*/
}

func (fr *Frame) Render() {
	if fr.PushBounds() {
		fr.FrameStdRender()
		fr.RenderChildren()
		fr.RenderScrolls()
		fr.PopBounds()
	}
}
