// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Frame is a Layout that renders a background according to the
// background-color style setting, and optional striping for grid layouts
type Frame struct {
	Layout
	Stripes Stripes `desc:"options for striped backgrounds -- rendered as darker bands relative to background color"`
}

var KiT_Frame = kit.Types.AddType(&Frame{}, FrameProps)

// AddNewFrame adds a new frame to given parent node, with given name and layout
func AddNewFrame(parent ki.Ki, name string, layout Layouts) *Frame {
	fr := parent.AddNewChild(KiT_Frame, name).(*Frame)
	fr.Lay = layout
	return fr
}

func (fr *Frame) CopyFieldsFrom(frm interface{}) {
	cp, ok := frm.(*Frame)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier Frame one\n", ki.Type(fr).Name())
		ki.GenCopyFieldsFrom(fr.This(), frm)
		return
	}
	fr.Layout.CopyFieldsFrom(&cp.Layout)
	fr.Stripes = cp.Stripes
}

var FrameProps = ki.Props{
	"EnumType:Flag":    KiT_NodeFlags,
	"border-width":     units.NewPx(2),
	"border-radius":    units.NewPx(0),
	"border-color":     &Prefs.Colors.Border,
	"padding":          units.NewPx(2),
	"margin":           units.NewPx(2),
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

var KiT_Stripes = kit.Enums.AddEnumAltLower(StripesN, kit.NotBitFlag, gist.StylePropProps, "Stripes")

func (ev Stripes) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Stripes) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// FrameStdRender does the standard rendering of the frame itself
func (fr *Frame) FrameStdRender() {
	rs, pc, st := fr.RenderLock()
	defer fr.RenderUnlock(rs)

	pos := fr.LayState.Alloc.Pos
	sz := fr.LayState.Alloc.Size
	pc.FillBox(rs, pos, sz, &st.Font.BgColor)

	rad := st.Border.Radius.Dots
	pos = pos.AddScalar(st.Layout.Margin.Dots).SubScalar(0.5 * st.Border.Width.Dots)
	sz = sz.SubScalar(2.0 * st.Layout.Margin.Dots).AddScalar(st.Border.Width.Dots)

	// then any shadow -- todo: optimize!
	if st.BoxShadow.HasShadow() {
		spos := pos.Add(mat32.Vec2{st.BoxShadow.HOffset.Dots, st.BoxShadow.VOffset.Dots})
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
}

func (fr *Frame) RenderStripes() {
	st := &fr.Sty
	rs := &fr.Viewport.Render
	pc := &rs.Paint

	pos := fr.LayState.Alloc.Pos
	sz := fr.LayState.Alloc.Size

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
		fr.SetScrollsOff()
		fr.DisconnectAllEvents(AllPris) // uses both Low and Hi
	}
}
