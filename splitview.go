// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//    SplitView

// SplitView allocates a fixed proportion of space to each child, along given
// dimension, always using only the available space given to it by its parent
// (i.e., it will force its children, which should be layouts (typically
// Frame's), to have their own scroll bars as necesssary).  It should
// generally be used as a main outer-level structure within a window,
// providing a framework for inner elements -- it allows individual child
// elements to update indpendently and thus is important for speeding update
// performance.  It uses the Widget Parts to hold the splitter widgets
// separately from the children that contain the rest of the scenegraph to be
// displayed within each region.
type SplitView struct {
	PartsWidgetBase
	Splits      []float32 `desc:"proportion (0-1 normalized, enforced) of space allocated to each element -- can enter 0 to collapse a given element"`
	SavedSplits []float32 `desc:"A saved version of the splits which can be restored -- for dynamic collapse / expand operations"`
	Dim         Dims2D    `desc:"dimension along which to split the space"`
}

var KiT_SplitView = kit.Types.AddType(&SplitView{}, SplitViewProps)

// auto-max-stretch
var SplitViewProps = ki.Props{
	"max-width":  -1.0,
	"max-height": -1.0,
	"margin":     0,
	"padding":    0,
}

// UpdateSplits updates the splits to be same length as number of children, and normalized
func (g *SplitView) UpdateSplits() {
	sz := len(g.Kids)
	if sz == 0 {
		return
	}
	if g.Splits == nil || len(g.Splits) != sz {
		g.Splits = make([]float32, sz)
	}
	sum := float32(0.0)
	for _, sp := range g.Splits {
		sum += sp
	}
	if sum == 0 { // set default even splits
		even := 1.0 / float32(sz)
		for i := range g.Splits {
			g.Splits[i] = even
		}
		sum = 1.0
	}
	norm := 1.0 / sum
	for i := range g.Splits {
		g.Splits[i] *= norm
	}
}

// SetSplits sets the split proportions -- can use 0 to hide / collapse a child entirely -- does an Update
func (g *SplitView) SetSplits(splits ...float32) {
	updt := g.UpdateStart()
	g.UpdateSplits()
	sz := len(g.Kids)
	mx := kit.MinInt(sz, len(splits))
	for i := 0; i < mx; i++ {
		g.Splits[i] = splits[i]
	}
	g.UpdateSplits()
	g.UpdateEnd(updt)
}

// SaveSplits saves the current set of splits in SavedSplits, for a later RestoreSplits
func (g *SplitView) SaveSplits() {
	sz := len(g.Splits)
	if sz == 0 {
		return
	}
	if g.SavedSplits == nil || len(g.SavedSplits) != sz {
		g.SavedSplits = make([]float32, sz)
	}
	for i, sp := range g.Splits {
		g.SavedSplits[i] = sp
	}
}

// RestoreSplits restores a previously-saved set of splits (if it exists), does an update
func (g *SplitView) RestoreSplits() {
	if g.SavedSplits == nil {
		return
	}
	g.SetSplits(g.SavedSplits...)
}

// CollapseChild collapses given child(ren) (sets split proportion to 0), optionally saving the prior splits for later Restore function -- does an Update -- triggered by double-click of splitter
func (g *SplitView) CollapseChild(save bool, idxs ...int) {
	updt := g.UpdateStart()
	if save {
		g.SaveSplits()
	}
	sz := len(g.Kids)
	for _, idx := range idxs {
		if idx >= 0 && idx < sz {
			g.Splits[idx] = 0
		}
	}
	g.UpdateSplits()
	g.UpdateEnd(updt)
}

func (g *SplitView) SetSplitsAction(idx int, nwval float32) {
	updt := g.UpdateStart()
	g.SetFullReRender()
	g.Splits[idx+1] = 1.0 - nwval
	g.Splits[idx] = nwval
	// fmt.Printf("splits: %v value: %v\n", idx, spl.Value)
	g.UpdateSplits()
	// fmt.Printf("splits: %v\n", g.Splits)
	g.UpdateEnd(updt)
}

func (g *SplitView) Init2D() {
	g.Parts.Lay = LayoutNil
	g.Init2DWidget()
	g.UpdateSplits()
	g.ConfigSplitters()
}

func (g *SplitView) ConfigSplitters() {
	sz := len(g.Kids)
	mods, updt := g.Parts.SetNChildren(sz-1, KiT_Splitter, "Splitter")
	odim := OtherDim(g.Dim)
	spc := g.Sty.BoxSpace()
	size := g.LayData.AllocSize.Dim(g.Dim) - 2.0*spc
	osz := float32(50.0)
	mid := 0.5 * (g.LayData.AllocSize.Dim(odim) - 2.0*spc)
	spicon := IconName("widget-handle-circles")
	for i, spk := range g.Parts.Children() {
		sp := spk.(*Splitter)
		sp.Defaults()
		sp.Icon = spicon
		sp.Dim = g.Dim
		sp.LayData.AllocSize.SetDim(g.Dim, size)
		sp.LayData.AllocSize.SetDim(odim, osz)
		sp.LayData.AllocPosRel.SetDim(g.Dim, 0)
		sp.LayData.AllocPosRel.SetDim(odim, mid-0.5*osz)
		sp.Min = 0.0
		sp.Max = 1.0
		sp.Snap = false
		if mods {
			sp.SliderSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(SliderValueChanged) {
					spr, _ := recv.EmbeddedStruct(KiT_SplitView).(*SplitView)
					spl := send.(*Splitter)
					spr.SetSplitsAction(i, spl.Value)
				}
			})
		}
	}
	if mods {
		g.Parts.UpdateEnd(updt)
	}
}

func (g *SplitView) Style2D() {
	g.Style2DWidget()
	g.UpdateSplits()
	g.ConfigSplitters()
}

func (g *SplitView) Layout2D(parBBox image.Rectangle) {
	g.ConfigSplitters()
	g.Layout2DBase(parBBox, true) // init style
	g.Layout2DParts(parBBox)
	g.UpdateSplits()

	handsz := float32(10.0)

	sz := len(g.Kids)
	odim := OtherDim(g.Dim)
	size := g.LayData.AllocSize.Dim(g.Dim)
	avail := size - handsz*float32(sz-1)
	osz := g.LayData.AllocSize.Dim(odim)
	pos := float32(0.0)
	handval := 0.6 * handsz / size

	for i, sp := range g.Splits {
		gis := g.Kids[i].(Node2D).AsWidget()
		if gis == nil {
			continue
		}
		if gis.TypeEmbeds(KiT_Frame) {
			gis.SetReRenderAnchor()
		}
		size := sp * avail
		gis.LayData.AllocSize.SetDim(g.Dim, size)
		gis.LayData.AllocSize.SetDim(odim, osz)
		gis.LayData.AllocPosRel.SetDim(g.Dim, pos)
		gis.LayData.AllocPosRel.SetDim(odim, 0)
		pos += size + handsz

		if i < sz-1 {
			spl := g.Parts.Child(i).(*Splitter)
			spl.Value = sp + handval
			spl.UpdatePosFromValue()
		}
	}

	g.Layout2DChildren()
}

func (g *SplitView) Render2D() {
	if g.FullReRenderIfNeeded() {
		return
	}
	if g.PushBounds() {
		for i, kid := range g.Kids {
			gii, _ := KiToNode2D(kid)
			if gii != nil {
				sp := g.Splits[i]
				if sp <= 0 {
					continue
				}
				gii.Render2D()
			}
		}
		g.Parts.Render2DTree()
		g.PopBounds()
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//    Splitter

// Splitter provides the splitter handle and line separating two elements in a
// SplitView, with draggable resizing of the splitter -- parent is Parts
// layout of the SplitView -- based on SliderBase
type Splitter struct {
	SliderBase
}

var KiT_Splitter = kit.Types.AddType(&Splitter{}, SplitterProps)

var SplitterProps = ki.Props{
	"padding":          units.NewValue(0, units.Px),
	"margin":           units.NewValue(0, units.Px),
	"background-color": &Prefs.BackgroundColor,
	"color":            &Prefs.FontColor,
	"#icon": ki.Props{
		"max-width":      units.NewValue(1, units.Em),
		"max-height":     units.NewValue(5, units.Em),
		"min-width":      units.NewValue(1, units.Em),
		"min-height":     units.NewValue(5, units.Em),
		"margin":         units.NewValue(0, units.Px),
		"padding":        units.NewValue(0, units.Px),
		"vertical-align": AlignMiddle,
		"fill":           &Prefs.IconColor,
		"stroke":         &Prefs.FontColor,
	},
	SliderSelectors[SliderActive]: ki.Props{},
	SliderSelectors[SliderInactive]: ki.Props{
		"border-color": "highlight-50",
		"color":        "highlight-50",
	},
	SliderSelectors[SliderHover]: ki.Props{
		"background-color": "highlight-10",
	},
	SliderSelectors[SliderFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "samelight-50",
	},
	SliderSelectors[SliderDown]: ki.Props{},
	SliderSelectors[SliderValue]: ki.Props{
		"border-color":     &Prefs.IconColor,
		"background-color": &Prefs.IconColor,
	},
	SliderSelectors[SliderBox]: ki.Props{
		"border-color":     &Prefs.BackgroundColor,
		"background-color": &Prefs.BackgroundColor,
	},
}

func (g *Splitter) Defaults() { // todo: should just get these from props
	g.ValThumb = false
	g.ThumbSize = units.NewValue(1, units.Em)
	g.Step = 0.01
	g.PageStep = 0.1
	g.Max = 1.0
	g.Snap = false
	g.Prec = 4
}

func (g *Splitter) Init2D() {
	g.Init2DSlider()
	g.Defaults()
	g.ConfigParts()
}

func (g *Splitter) ConfigPartsIfNeeded(render bool) {
	if g.PartsNeedUpdateIconLabel(string(g.Icon), "") {
		g.ConfigParts()
	}
	if g.Icon.IsValid() && g.Parts.HasChildren() {
		ic := g.Parts.ChildByType(KiT_Icon, true, 0).(*Icon)
		if ic != nil {
			mrg := g.Sty.Layout.Margin.Dots
			pad := g.Sty.Layout.Padding.Dots
			spc := mrg + pad
			odim := OtherDim(g.Dim)
			ic.LayData.AllocPosRel.SetDim(g.Dim, g.Pos+spc-0.45*g.ThSize)
			ic.LayData.AllocPosRel.SetDim(odim, -pad)
			ic.LayData.AllocSize.SetDim(odim, 2.0*g.ThSize)
			ic.LayData.AllocSize.SetDim(g.Dim, g.ThSize)
			if render {
				ic.Layout2DTree()
			}
		}
	}
}

func (g *Splitter) Style2D() {
	bitflag.Clear(&g.Flag, int(CanFocus))
	g.Style2DWidget()
	pst := &(g.Par.(Node2D).AsWidget().Sty)
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i].CopyFrom(&g.Sty)
		g.StateStyles[i].SetStyleProps(pst, g.StyleProps(SliderSelectors[i]))
		g.StateStyles[i].CopyUnitContext(&g.Sty.UnContext)
	}
	SliderFields.Style(g, nil, g.Props)
	SliderFields.ToDots(g, &g.Sty.UnContext)
	g.ThSize = g.ThumbSize.Dots
	g.ConfigParts()
}

func (g *Splitter) Size2D() {
	g.InitLayout2D()
	if g.ThSize == 0.0 {
		g.Defaults()
	}
}

func (g *Splitter) Layout2D(parBBox image.Rectangle) {
	g.ConfigPartsIfNeeded(false)
	g.Layout2DBase(parBBox, true) // init style
	g.Layout2DParts(parBBox)
	g.SizeFromAlloc()
	g.Layout2DChildren()
	g.OrigWinBBox = g.WinBBox
}

func (g *Splitter) UpdateSplitterPos() {
	pos := int(g.Pos)
	if g.Dim == X {
		g.VpBBox = image.Rect(pos, g.VpBBox.Min.Y, pos+10, g.VpBBox.Max.Y)
		g.WinBBox = image.Rect(pos, g.WinBBox.Min.Y, pos+10, g.WinBBox.Max.Y)
	} else {
		g.VpBBox = image.Rect(g.VpBBox.Min.X, pos, g.VpBBox.Max.X, pos+10)
		g.WinBBox = image.Rect(g.WinBBox.Min.X, pos, g.WinBBox.Max.X, pos+10)
	}
}

func (g *Splitter) Render2D() {
	vp := g.Viewport
	win := vp.Win
	g.SliderEvents()
	if g.IsDragging() {
		ic := g.Parts.ChildByType(KiT_Icon, true, 0).(*Icon)
		if ic == nil {
			return
		}
		ovk := win.OverlayVp.ChildByName(g.UniqueName(), 0)
		var ovb *Bitmap
		if ovk == nil {
			ovb = &Bitmap{}
			ovb.SetName(g.UniqueName())
			win.OverlayVp.AddChild(ovb)
			ovk = ovb.This
		}
		ovb = ovk.(*Bitmap)
		ovb.GrabRenderFrom(ic)
		ovb.LayData = ic.LayData // copy
		g.UpdateSplitterPos()
		ovb.LayData.AllocPos.SetPoint(g.VpBBox.Min)
		win.RenderOverlays()
	} else {
		ovk := win.OverlayVp.ChildIndexByName(g.UniqueName(), 0)
		if ovk >= 0 {
			win.OverlayVp.DeleteChildAtIndex(ovk, true)
			win.RenderOverlays()
		} else {
			if g.PushBounds() {
				g.Render2DDefaultStyle()
				g.Render2DChildren()
				g.PopBounds()
			}
		}
	}
}

// render using a default style if not otherwise styled
func (g *Splitter) Render2DDefaultStyle() {
	st := &g.Sty
	rs := &g.Viewport.Render
	pc := &rs.Paint

	g.UpdateSplitterPos()
	g.ConfigPartsIfNeeded(true)

	if g.Icon.IsValid() && g.Parts.HasChildren() {
		g.Parts.Render2DTree()
	} else {
		pc.StrokeStyle.SetColor(nil)
		pc.FillStyle.SetColorSpec(&st.Font.BgColor)

		pos := NewVec2DFmPoint(g.VpBBox.Min)
		pos.SetSubDim(OtherDim(g.Dim), 10.0)
		sz := NewVec2DFmPoint(g.VpBBox.Size())
		g.RenderBoxImpl(pos, sz, 0)
	}
}

func (g *Splitter) FocusChanged2D(gotFocus bool) {
	if gotFocus {
		g.EmitFocusedSignal()
		g.SetSliderState(SliderFocus)
	} else {
		g.SetSliderState(SliderActive) // lose any hover state but whatever..
	}
	g.UpdateSig()
}
