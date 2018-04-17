// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//    SplitView

// SplitView allocates a fixed proportion of space to each child, along given dimension, always using only the available space given to it by its parent (i.e., it will force its children, which should be layouts (typically Frame's), to have their own scroll bars as necesssary).  It should generally be used as a main outer-level structure within a window, providing a framework for inner elements -- it allows individual child elements to update indpendently and thus is important for speeding update performance.  It uses the Widget Parts to hold the splitter widgets separately from the children that contain the rest of the scenegraph to be displayed within each region.
type SplitView struct {
	WidgetBase
	Splits      []float64 `desc:"proportion (0-1 normalized, enforced) of space allocated to each element -- can enter 0 to collapse a given element"`
	SavedSplits []float64 `desc:"A saved version of the splits which can be restored -- for dynamic collapse / expand operations"`
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
		g.Splits = make([]float64, sz)
	}
	sum := 0.0
	for _, sp := range g.Splits {
		sum += sp
	}
	if sum == 0 { // set default even splits
		even := 1.0 / float64(sz)
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
func (g *SplitView) SetSplits(splits ...float64) {
	g.UpdateStart()
	g.UpdateSplits()
	sz := len(g.Kids)
	mx := kit.MinInt(sz, len(splits))
	for i := 0; i < mx; i++ {
		g.Splits[i] = splits[i]
	}
	g.UpdateSplits()
	g.UpdateEnd()
}

// SaveSplits saves the current set of splits in SavedSplits, for a later RestoreSplits
func (g *SplitView) SaveSplits() {
	sz := len(g.Splits)
	if sz == 0 {
		return
	}
	if g.SavedSplits == nil || len(g.SavedSplits) != sz {
		g.SavedSplits = make([]float64, sz)
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
	g.UpdateStart()
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
	g.UpdateEnd()
}

func (g *SplitView) Init2D() {
	g.Parts.Lay = LayoutNil
	g.Init2DWidget()
	g.UpdateSplits()
	g.ConfigSplitters()
}

func (g *SplitView) ConfigSplitters() {
	sz := len(g.Kids)
	updt := g.Parts.SetNChildren(sz-1, KiT_Splitter, "Splitter")
	odim := OtherDim(g.Dim)
	spc := g.Style.BoxSpace()
	size := g.LayData.AllocSize.Dim(g.Dim) - 2.0*spc
	osz := 50.0
	mid := 0.5 * size
	spicon := IconByName("widget-handle-circles")
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
		if updt {
			sp.SliderSig.Connect(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(SliderValueChanged) {
					spr, _ := recv.EmbeddedStruct(KiT_SplitView).(*SplitView)
					spr.UpdateStart()
					spl := send.(*Splitter)
					spr.Splits[i+1] = 1.0 - spl.Value
					spr.Splits[i] = spl.Value
					// fmt.Printf("splits: %v value: %v\n", i, spl.Value)
					spr.UpdateSplits()
					// fmt.Printf("splits: %v\n", spr.Splits)
					spr.UpdateEnd()
				}
			})
		}
	}
}

func (g *SplitView) Style2D() {
	g.Style2DWidget(SplitViewProps)
	g.UpdateSplits()
	g.ConfigSplitters()
}

func (g *SplitView) Layout2D(parBBox image.Rectangle) {
	g.ConfigSplitters()
	g.Layout2DWidget(parBBox)
	g.UpdateSplits()

	handsz := 10.0

	sz := len(g.Kids)
	odim := OtherDim(g.Dim)
	size := g.LayData.AllocSize.Dim(g.Dim)
	avail := size - handsz*float64(sz-1)
	osz := g.LayData.AllocSize.Dim(odim)
	pos := 0.0
	handval := 0.6 * handsz / size

	for i, sp := range g.Splits {
		_, gi := KiToNode2D(g.Kids[i])
		if gi != nil {
			if gi.TypeEmbeds(KiT_Frame) {
				gi.SetReRenderAnchor()
			}
			size := sp * avail
			gi.LayData.AllocSize.SetDim(g.Dim, size)
			gi.LayData.AllocSize.SetDim(odim, osz)
			gi.LayData.AllocPosRel.SetDim(g.Dim, pos)
			gi.LayData.AllocPosRel.SetDim(odim, 0)
			pos += size + handsz

			if i < sz-1 {
				spl := g.Parts.Child(i).(*Splitter)
				spl.Value = sp + handval
				spl.UpdatePosFromValue()
			}
		}
	}

	g.Layout2DChildren()
}

func (g *SplitView) Render2D() {
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

func (g *SplitView) ReRender2D() (node Node2D, layout bool) {
	node = g.This.(Node2D)
	layout = true
	return
}

// check for interface implementation
var _ Node2D = &SplitView{}

////////////////////////////////////////////////////////////////////////////////////////
//    Splitter

// Splitter provides the splitter handle and line separating two elements in a SplitView, with draggable resizing of the splitter -- parent is Parts layout of the SplitView -- based on SliderBase
type Splitter struct {
	SliderBase
}

var KiT_Splitter = kit.Types.AddType(&Splitter{}, SplitterProps)

var SplitterProps = ki.Props{
	SliderSelectors[SliderActive]: ki.Props{
		"padding":          "0px",
		"margin":           "0px",
		"background-color": "#EEF",
		"#icon": ki.Props{
			"max-width":  units.NewValue(1, units.Em),
			"max-height": units.NewValue(5, units.Em),
			"min-width":  units.NewValue(1, units.Em),
			"min-height": units.NewValue(5, units.Em),
			"margin":     units.NewValue(0, units.Px),
			"padding":    units.NewValue(0, units.Px),
			"vert-align": AlignMiddle,
		},
	},
	SliderSelectors[SliderDisabled]: ki.Props{
		"border-color":     "#BBB",
		"background-color": "#DDD",
	},
	SliderSelectors[SliderHover]: ki.Props{
		"background-color": "#EEF",
	},
	SliderSelectors[SliderFocus]: ki.Props{
		"border-color":     "#008",
		"background.color": "#CCF",
	},
	SliderSelectors[SliderDown]: ki.Props{
		"border-color":     "#000",
		"background-color": "#EEF",
	},
	SliderSelectors[SliderValue]: ki.Props{
		"border-color":     "#00F",
		"background-color": "#00F",
	},
	SliderSelectors[SliderBox]: ki.Props{
		"border-color":     "#888",
		"background-color": "#FFF",
	},
}

func (g *Splitter) Defaults() { // todo: should just get these from props
	g.ValThumb = false
	g.ThumbSize = 10.0
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
	if g.PartsNeedUpdateIconLabel(g.Icon, "") {
		g.ConfigParts()
	}
	if g.Icon != nil && g.Parts.HasChildren() {
		ic := g.Parts.ChildByType(KiT_Icon, true, 0).(*Icon)
		if ic != nil {
			mrg := g.Style.Layout.Margin.Dots
			pad := g.Style.Layout.Padding.Dots
			spc := mrg + pad
			odim := OtherDim(g.Dim)
			if g.IsDragging() {
				bitflag.Set(&ic.Flag, int(VpFlagDrawIntoWin))
				// ic.DrawMainVpOverMe()
			} else {
				bitflag.Clear(&ic.Flag, int(VpFlagDrawIntoWin))
			}
			ic.LayData.AllocPosRel.SetDim(g.Dim, g.Pos+spc-0.5*g.ThumbSize)
			ic.LayData.AllocPosRel.SetDim(odim, -pad)
			ic.LayData.AllocSize.SetDim(odim, 2.0*g.ThumbSize)
			ic.LayData.AllocSize.SetDim(g.Dim, g.ThumbSize)
			if render {
				ic.Layout2DTree()
			}
		}
	}
}

func (g *Splitter) Style2D() {
	g.Style2DWidget(g.StyleProps(SliderSelectors[SliderActive]))
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, g.StyleProps(SliderSelectors[i]))
		}
		g.StateStyles[i].SetUnitContext(g.Viewport, Vec2DZero)
	}
	g.ConfigParts()
}

func (g *Splitter) Size2D() {
	g.InitLayout2D()
	if g.ThumbSize == 0.0 {
		g.Defaults()
	}
}

func (g *Splitter) Layout2D(parBBox image.Rectangle) {
	g.ConfigPartsIfNeeded(false)
	g.Layout2DWidget(parBBox)
	g.SizeFromAlloc()
	g.Layout2DChildren()
	g.origWinBBox = g.WinBBox
}

func (g *Splitter) Render2D() {
	if g.PushBounds() {
		if !g.HasChildren() {
			g.Render2DDefaultStyle()
		} else {
			// todo: manage stacked layout to select appropriate image based on state
			// return
		}
		g.Render2DChildren()
		g.PopBounds()
	}
}

// render using a default style if not otherwise styled
func (g *Splitter) Render2DDefaultStyle() {
	pc := &g.Paint
	st := &g.Style
	// rs := &g.Viewport.Render

	{ // update bboxes to
		pos := int(g.Pos)
		if g.Dim == X {
			g.VpBBox = image.Rect(pos, g.VpBBox.Min.Y, pos+10, g.VpBBox.Max.Y)
			g.WinBBox = image.Rect(pos, g.WinBBox.Min.Y, pos+10, g.WinBBox.Max.Y)
		} else {
			g.VpBBox = image.Rect(g.VpBBox.Min.X, pos, g.VpBBox.Max.X, pos+10)
			g.WinBBox = image.Rect(g.WinBBox.Min.X, pos, g.WinBBox.Max.X, pos+10)
		}
		for i := 0; i < int(SliderStatesN); i++ {
			g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
		}
	}

	g.ConfigPartsIfNeeded(true)

	if g.Icon != nil && g.Parts.HasChildren() {
		g.Parts.Render2DTree()
	} else {
		pc.StrokeStyle.SetColor(nil)
		pc.FillStyle.SetColor(&st.Background.Color)

		pos := NewVec2DFmPoint(g.VpBBox.Min)
		pos.SetSubDim(OtherDim(g.Dim), 10.0)
		sz := NewVec2DFmPoint(g.VpBBox.Size())
		g.RenderBoxImpl(pos, sz, 0)
	}
}

func (g *Splitter) ReRender2D() (node Node2D, layout bool) {
	// if g.IsDragging() {
	// 	if g.Icon != nil && g.Parts.HasChildren() {
	// 		ic := g.Parts.ChildByType(KiT_Icon, true, 0).(*Icon)
	// 		if ic != nil {
	// 			g.ConfigPartsIfNeeded(true)
	// 			node = ic.This.(Node2D)
	// 			layout = false
	// 			return
	// 		}
	// 	}
	// }
	node = g.This.(Node2D)
	layout = false
	return
}

func (g *Splitter) FocusChanged2D(gotFocus bool) {
	// fmt.Printf("focus changed %v\n", gotFocus)
	g.UpdateStart()
	if gotFocus {
		g.SetSliderState(SliderFocus)
	} else {
		g.SetSliderState(SliderActive) // lose any hover state but whatever..
	}
	g.UpdateEnd()
}

// check for interface implementation
var _ Node2D = &Splitter{}
