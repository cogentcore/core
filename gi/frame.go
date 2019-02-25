// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Frame is a Layout that renders a background according to the
// background-color style setting, and optional striping for grid layouts
type Frame struct {
	Layout
	Stripes Stripes `desc:"options for striped backgrounds -- rendered as darker bands relative to background color"`
}

var KiT_Frame = kit.Types.AddType(&Frame{}, FrameProps)

var FrameProps = ki.Props{
	"border-width":     units.NewValue(2, units.Px),
	"border-radius":    units.NewValue(0, units.Px),
	"border-color":     &Prefs.Colors.Border,
	"border-style":     BorderSolid,
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"color":            &Prefs.Colors.Font,
	"background-color": &Prefs.Colors.Background,
}

// Stripes defines stripes options for elements that can render striped backgrounds
type Stripes int32

const (
	NoStripes Stripes = iota
	RowStripes
	ColStripes
	StripesN
)

//go:generate stringer -type=Stripes

var KiT_Stripes = kit.Enums.AddEnumAltLower(StripesN, false, StylePropProps, "Stripes")

func (ev Stripes) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Stripes) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// FrameStdRender does the standard rendering of the frame itself
func (fr *Frame) FrameStdRender() {
	st := &fr.Sty
	rs := &fr.Viewport.Render
	rs.Lock()
	pc := &rs.Paint
	// first draw a background rectangle in our full area

	pos := fr.LayData.AllocPos
	sz := fr.LayData.AllocSize
	pc.FillBox(rs, pos, sz, &st.Font.BgColor)

	rad := st.Border.Radius.Dots
	pos = pos.AddVal(st.Layout.Margin.Dots).SubVal(0.5 * st.Border.Width.Dots)
	sz = sz.SubVal(2.0 * st.Layout.Margin.Dots).AddVal(st.Border.Width.Dots)

	// then any shadow -- todo: optimize!
	if st.BoxShadow.HasShadow() {
		spos := pos.Add(Vec2D{st.BoxShadow.HOffset.Dots, st.BoxShadow.VOffset.Dots})
		pc.StrokeStyle.SetColor(nil)
		pc.FillStyle.SetColor(&st.BoxShadow.Color)
		if rad == 0.0 {
			pc.DrawRectangle(rs, spos.X, spos.Y, sz.X, sz.Y)
		} else {
			pc.DrawRoundedRectangle(rs, spos.X, spos.Y, sz.X, sz.Y, rad)
		}
		pc.FillStrokeClear(rs)
	}

	if fr.Lay == LayoutGrid && fr.Stripes != NoStripes {
		fr.RenderStripes()
	}

	pc.FillStyle.SetColor(nil)
	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	if rad == 0.0 {
		pc.DrawRectangle(rs, pos.X, pos.Y, sz.X, sz.Y)
	} else {
		pc.DrawRoundedRectangle(rs, pos.X, pos.Y, sz.X, sz.Y, rad)
	}
	pc.FillStrokeClear(rs)
	rs.Unlock()
}

func (fr *Frame) RenderStripes() {
	st := &fr.Sty
	rs := &fr.Viewport.Render
	pc := &rs.Paint

	pos := fr.LayData.AllocPos
	sz := fr.LayData.AllocSize

	delta := fr.Move2DDelta(image.ZP)

	hic := st.Font.BgColor.Color.Highlight(10)
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

func (fr *Frame) Render2D() {
	if fr.FullReRenderIfNeeded() {
		return
	}
	if fr.PushBounds() {
		fr.FrameStdRender()
		fr.This().(Node2D).ConnectEvents2D()
		fr.RenderScrolls()
		fr.Render2DChildren()
		fr.PopBounds()
	} else {
		fr.DisconnectAllEvents(AllPris) // uses both Low and Hi
	}
}
