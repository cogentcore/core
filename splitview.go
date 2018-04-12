// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

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

var KiT_SplitView = kit.Types.AddType(&SplitView{}, nil)

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
}

// auto-max-stretch
var SplitViewProps = map[string]interface{}{
	"max-width":  -1.0,
	"max-height": -1.0,
}

func (g *SplitView) Style2D() {
	g.Style2DWidget(SplitViewProps)
	g.UpdateSplits()
}

func (g *SplitView) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	g.UpdateSplits()

	sz := len(g.Kids)
	g.Parts.SetNChildren(sz-1, KiT_Splitter, "Splitter")

	handsz := 10.0

	odim := OtherDim(g.Dim)
	avail := g.LayData.AllocSize.Dim(g.Dim) - handsz*float64(sz-1)
	osz := g.LayData.AllocSize.Dim(odim)
	pos := 0.0

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
		}
	}

	g.Layout2DChildren()
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

var KiT_Splitter = kit.Types.AddType(&Splitter{}, nil)

func (g *Splitter) Defaults() { // todo: should just get these from props
	g.ValThumb = false
	g.ThumbSize = 10.0
	g.Step = 0.1
	g.PageStep = 0.2
	g.Max = 1.0
}

func (g *Splitter) Init2D() {
	g.Init2DSlider()
}

var SplitterProps = []map[string]interface{}{
	{
		"width":            "16px", // assumes vertical -- user needs to set!
		"min-width":        "16px",
		"border-width":     "1px",
		"border-radius":    "4px",
		"border-color":     "black",
		"border-style":     "solid",
		"padding":          "0px",
		"margin":           "2px",
		"background-color": "#EEF",
	}, { // disabled
		"border-color":     "#BBB",
		"background-color": "#DDD",
	}, { // hover
		"background-color": "#CCF", // todo "darker"
	}, { // focus
		"border-color":     "#008",
		"background.color": "#CCF",
	}, { // press
		"border-color":     "#000",
		"background-color": "#DDF",
	}, { // value fill
		"border-color":     "#00F",
		"background-color": "#00F",
	}, { // overall box -- just white
		"border-color":     "#FFF",
		"background-color": "#FFF",
	},
}

func (g *Splitter) Style2D() {
	bitflag.Set(&g.NodeFlags, int(CanFocus))
	g.Style2DWidget(SplitterProps[SliderNormal])
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, SplitterProps[i])
		}
		g.StateStyles[i].SetUnitContext(g.Viewport, Vec2DZero)
	}
	// todo: how to get state-specific user prefs?  need an extra prefix..
}

func (g *Splitter) Size2D() {
	g.InitLayout2D()
}

func (g *Splitter) Layout2D(parBBox image.Rectangle) {
	g.Layout2DWidget(parBBox)
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.SizeFromAlloc()
	g.Layout2DChildren()
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

	// overall fill box
	g.RenderStdBox(&g.StateStyles[SliderBox])

	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	pc.FillStyle.SetColor(&st.Background.Color)

	// scrollbar is basic box in content size
	spc := st.BoxSpace()
	pos := g.LayData.AllocPos.AddVal(spc)
	sz := g.LayData.AllocSize.SubVal(2.0 * spc)

	g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots) // surround box
	pos.SetAddDim(g.Dim, g.Pos)                     // start of thumb
	sz.SetDim(g.Dim, g.ThumbSize)
	pc.FillStyle.SetColor(&g.StateStyles[SliderValueFill].Background.Color)
	g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
}

func (g *Splitter) FocusChanged2D(gotFocus bool) {
	// fmt.Printf("focus changed %v\n", gotFocus)
	g.UpdateStart()
	if gotFocus {
		g.SetSliderState(SliderFocus)
	} else {
		g.SetSliderState(SliderNormal) // lose any hover state but whatever..
	}
	g.UpdateEnd()
}

// check for interface implementation
var _ Node2D = &Splitter{}
