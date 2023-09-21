// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"

	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
)

// Frame is a Layout that renders a background according to the
// background-color style setting, and optional striping for grid layouts
type Frame struct {
	Layout

	// options for striped backgrounds -- rendered as darker bands relative to background color
	Stripes Stripes `desc:"options for striped backgrounds -- rendered as darker bands relative to background color"`
}

func (fr *Frame) OnInit() {
	fr.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.Border.Style.Set(gist.BorderNone)
		s.Border.Radius.Set()
		s.Padding.Set(units.Px(2 * Prefs.DensityMul()))
	})
}

func (fr *Frame) CopyFieldsFrom(frm any) {
	cp, ok := frm.(*Frame)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier Frame one\n", ki.Type(fr).Name())
		ki.GenCopyFieldsFrom(fr.This(), frm)
		return
	}
	fr.Layout.CopyFieldsFrom(&cp.Layout)
	fr.Stripes = cp.Stripes
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

	if fr.Lay == LayoutGrid && fr.Stripes != NoStripes && Prefs.Params.ZebraStripeWeight != 0 {
		fr.RenderStripes()
	}
}

func (fr *Frame) RenderStripes() {
	st := &fr.Style
	rs := &fr.Viewport.Render
	pc := &rs.Paint

	pos := fr.LayState.Alloc.Pos
	sz := fr.LayState.Alloc.Size

	delta := fr.Move2DDelta(image.Point{})

	// TODO: fix stripes
	// hic := st.BackgroundColor.Color.Highlight(Prefs.Params.ZebraStripeWeight)
	hic := st.BackgroundColor.Color
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
}

func (fr *Frame) Render(vp *Viewport) {
	if fr.FullReRenderIfNeeded() {
		return
	}
	if fr.PushBounds() {
		fr.FrameStdRender()
		fr.This().(Node2D).ConnectEvents()
		fr.RenderScrolls()
		fr.RenderChildren()
		fr.PopBounds()
	} else {
		fr.SetScrollsOff()
		fr.DisconnectAllEvents(AllPris) // uses both Low and Hi
	}
}
