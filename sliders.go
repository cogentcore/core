// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/ki"
	"image"
	"math"
	// "time"
)

////////////////////////////////////////////////////////////////////////////////////////
// SliderBase -- basis for sliders

// signals that sliders can send
type SliderSignals int64

const (
	// value has changed -- if tracking is enabled, then this tracks online changes -- otherwise only at the end
	SliderValueChanged SliderSignals = iota
	// slider pushed down but not yet up
	SliderPressed
	SliderReleased
	SliderMoved
	SliderSignalsN
)

//go:generate stringer -type=SliderSignals

// mutually-exclusive slider states -- determines appearance
type SliderStates int32

const (
	// normal state -- there but not being interacted with
	SliderNormal SliderStates = iota
	// disabled -- not pressable
	SliderDisabled
	// mouse is hovering over the slider
	SliderHover
	// slider is the focus -- will respond to keyboard input
	SliderFocus
	// slider is currently being pressed down
	SliderDown
	// use background-color here to fill in selected value of slider
	SliderValueFill
	// these styles define the overall box around slider -- typically no border and a white background -- needs a background to allow local re-rendering
	SliderBox
	// total number of slider states
	SliderStatesN
)

//go:generate stringer -type=SliderStates

// SliderBase has common slider functionality -- two major modes: ValThumb =
// false is a slider with a fixed-size thumb knob, while = true has a
// thumb that represents a value, as in a scrollbar, and the scrolling range is size - thumbsize
type SliderBase struct {
	WidgetBase
	Min         float64              `xml:"min",desc:"minimum value in range"`
	Max         float64              `xml:"max",desc:"maximum value in range"`
	Step        float64              `xml:"step",desc:"smallest step size to increment"`
	PageStep    float64              `xml:"step",desc:"larger PageUp / Dn step size"`
	Value       float64              `xml:"value",desc:"current value"`
	EmitValue   float64              `xml:"value",desc:"previous emitted value - don't re-emit if it is the same"`
	Size        float64              `xml:"size",desc:"size of the slide box in the relevant dimension -- range of motion -- exclusive of spacing"`
	ThumbSize   float64              `xml:"thumb-size",desc:"size of the thumb -- if ValThumb then this is auto-sized based on ThumbVal and is subtracted from Size in computing Value"`
	ValThumb    bool                 `xml:"prop-thumb","desc:"if true, has a proportionally-sized thumb knob reflecting another value -- e.g., the amount visible in a scrollbar, and thumb is completely inside Size -- otherwise ThumbSize affects Size so that full Size range can be traversed"`
	ThumbVal    float64              `xml:thumb-val",desc:"value that the thumb represents, in the same units"`
	Pos         float64              `xml:"pos",desc:"logical position of the slider relative to Size"`
	DragPos     float64              `xml:"-",desc:"underlying drag position of slider -- not subject to snapping"`
	VisPos      float64              `xml:"vispos",desc:"visual position of the slider -- can be different from pos in a RTL environment"`
	Horiz       bool                 `xml:"horiz",desc:"true if horizontal, else vertical"`
	Tracking    bool                 `xml:"tracking",desc:"if true, will send continuous updates of value changes as user moves the slider -- otherwise only at the end"`
	Snap        bool                 `xml:"snap",desc:"snap the values to Step size increments"`
	StateStyles [SliderStatesN]Style `desc:"styles for different states of the slider, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State       SliderStates
	SliderSig   ki.Signal `json:"-",desc:"signal for slider -- see SliderSignals for the types"`
	// todo: icon -- should be an xml
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_SliderBase = ki.Types.AddType(&SliderBase{}, nil)

// if snap is set, then snap the value to step sizes
func (g *SliderBase) SnapValue() {
	if g.Snap {
		g.Value = float64(int(math.Round(g.Value/g.Step))) * g.Step
	}
}

// set the slider state to target
func (g *SliderBase) SetSliderState(state SliderStates) {
	// todo: process disabled state -- probably just deal with the property directly?
	// it overrides any choice here and just sets state to disabled..
	if state == SliderNormal && g.HasFocus() {
		state = SliderFocus
	}
	g.State = state
	g.Style = g.StateStyles[state] // get relevant styles
}

// set the slider in the down state -- mouse clicked down but not yet up --
// emits SliderPressed signal
func (g *SliderBase) SliderPressed(pos float64) {
	g.EmitValue = g.Min - 1.0 // invalid value
	g.UpdateStart()
	g.SetSliderState(SliderDown)
	g.SetSliderPos(pos)
	g.SliderSig.Emit(g.This, int64(SliderPressed), g.Value)
	// ki.SetBitFlag(&g.NodeFlags, int(SliderFlagDragging))
	g.UpdateEnd()
}

// the slider has just been released -- sends a released signal and returns
// state to normal, and emits clicked signal if if it was previously in pressed state
func (g *SliderBase) SliderReleased() {
	wasPressed := (g.State == SliderDown)
	g.UpdateStart()
	g.SetSliderState(SliderNormal)
	g.SliderSig.Emit(g.This, int64(SliderReleased), g.Value)
	if wasPressed && g.Value != g.EmitValue {
		g.SliderSig.Emit(g.This, int64(SliderValueChanged), g.Value)
		g.EmitValue = g.Value
	}
	g.UpdateEnd()
}

// slider starting hover-- todo: keep track of time and popup a tooltip -- signal?
func (g *SliderBase) SliderEnterHover() {
	if g.State != SliderHover {
		g.UpdateStart()
		g.SetSliderState(SliderHover)
		g.UpdateEnd()
	}
}

// slider exiting hover
func (g *SliderBase) SliderExitHover() {
	if g.State == SliderHover {
		g.UpdateStart()
		g.SetSliderState(SliderNormal)
		g.UpdateEnd()
	}
}

// get size from allocation
func (g *SliderBase) SizeFromAlloc() {
	if g.LayData.AllocSize.IsZero() {
		return
	}
	st := &g.Style
	spc := st.Layout.Margin.Dots
	if g.Horiz {
		g.Size = g.LayData.AllocSize.X - 2.0*spc
	} else {
		g.Size = g.LayData.AllocSize.Y - 2.0*spc
	}
	if !g.ValThumb {
		g.Size -= g.ThumbSize + 2.0*st.Border.Width.Dots + 2.0
	}
	g.UpdatePosFromValue()
	g.DragPos = g.Pos
}

// set the position of the slider at the given position in pixels -- updates the corresponding Value
func (g *SliderBase) SetSliderPos(pos float64) {
	g.UpdateStart()
	g.Pos = pos
	g.Pos = math.Min(g.Size, g.Pos)
	if g.ValThumb {
		g.UpdateThumbValSize()
		g.Pos = math.Min(g.Size-g.ThumbSize, g.Pos)
	}
	g.Pos = math.Max(0, g.Pos)
	g.Value = g.Min + (g.Max-g.Min)*(g.Pos/g.Size)
	g.DragPos = g.Pos
	if g.Snap {
		g.SnapValue()
		g.UpdatePosFromValue()
	}
	if g.Tracking && g.Value != g.EmitValue {
		g.SliderSig.Emit(g.This, int64(SliderValueChanged), g.Value)
		g.EmitValue = g.Value
	}
	g.UpdateEnd()
}

// slider moved along relevant axis
func (g *SliderBase) SliderMoved(start, end float64) {
	del := end - start
	g.SetSliderPos(g.DragPos + del)
}

func (g *SliderBase) UpdatePosFromValue() {
	if g.Size == 0.0 {
		return
	}
	if g.ValThumb {
		g.UpdateThumbValSize()
	}
	g.Pos = g.Size * (g.Value - g.Min) / (g.Max - g.Min)
}

// set a value
func (g *SliderBase) SetValue(val float64) {
	g.UpdateStart()
	g.Value = math.Min(val, g.Max)
	if g.ValThumb {
		g.Value = math.Min(g.Value, g.Max-g.ThumbVal)
	}
	g.Value = math.Max(g.Value, g.Min)
	g.UpdatePosFromValue()
	g.DragPos = g.Pos
	g.SliderSig.Emit(g.This, int64(SliderValueChanged), g.Value)
	g.UpdateEnd()
}

func (g *SliderBase) SetThumbValue(val float64) {
	g.UpdateStart()
	g.ThumbVal = math.Min(val, g.Max)
	g.ThumbVal = math.Max(g.ThumbVal, g.Min)
	g.UpdateThumbValSize()
	g.UpdateEnd()
}

// set thumb size as proportion of min / max (e.g., amount visible in
// scrollbar) -- max's out to full size
func (g *SliderBase) UpdateThumbValSize() {
	g.ThumbSize = ((g.ThumbVal - g.Min) / (g.Max - g.Min))
	g.ThumbSize = math.Min(g.ThumbSize, 1.0)
	g.ThumbSize = math.Max(g.ThumbSize, 0.0)
	g.ThumbSize *= g.Size
}

func (g *SliderBase) KeyInput(kt KeyTypedEvent) {
	kf := KeyFun(kt.Key, kt.Chord)
	switch kf {
	case KeyFunMoveUp:
		g.SetValue(g.Value - g.Step)
	case KeyFunMoveLeft:
		g.SetValue(g.Value - g.Step)
	case KeyFunMoveDown:
		g.SetValue(g.Value + g.Step)
	case KeyFunMoveRight:
		g.SetValue(g.Value + g.Step)
	case KeyFunPageUp:
		g.SetValue(g.Value - g.PageStep)
	case KeyFunPageLeft:
		g.SetValue(g.Value - g.PageStep)
	case KeyFunPageDown:
		g.SetValue(g.Value + g.PageStep)
	case KeyFunPageRight:
		g.SetValue(g.Value + g.PageStep)
	case KeyFunHome:
		g.SetValue(g.Min)
	case KeyFunEnd:
		g.SetValue(g.Max)
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  Slider

// Slider is a standard value slider with a fixed-sized thumb knob
type Slider struct {
	SliderBase
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Slider = ki.Types.AddType(&Slider{}, nil)

func (g *Slider) Defaults() { // todo: should just get these from props
	g.ThumbSize = 25.0
	g.Step = 0.1
	g.PageStep = 0.2
	g.Max = 1.0
}

func (g *Slider) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Slider) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Slider) AsLayout2D() *Layout {
	return nil
}

func (g *Slider) InitNode2D() {
	g.InitNode2DBase()
	g.ReceiveEventType(MouseDraggedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*Slider)
		if ok {
			if sl.IsDragging() {
				me := d.(MouseDraggedEvent)
				st := sl.PointToRelPos(me.From)
				ed := sl.PointToRelPos(me.Where)
				if sl.Horiz {
					sl.SliderMoved(float64(st.X), float64(ed.X))
				} else {
					sl.SliderMoved(float64(st.Y), float64(ed.Y))
				}
			}
		}
	})
	g.ReceiveEventType(MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*Slider)
		if ok {
			me := d.(MouseDownEvent)
			ed := sl.PointToRelPos(me.Where)
			st := &sl.Style
			spc := st.Layout.Margin.Dots + 0.5*g.ThumbSize
			if sl.Horiz {
				sl.SliderPressed(float64(ed.X) - spc)
			} else {
				sl.SliderPressed(float64(ed.Y) - spc)
			}
		}
	})
	g.ReceiveEventType(MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*Slider)
		if ok {
			sl.SliderReleased()
		}
	})
	g.ReceiveEventType(MouseEnteredEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*Slider)
		if ok {
			sl.SliderEnterHover()
		}
	})
	g.ReceiveEventType(MouseExitedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*Slider)
		if ok {
			sl.SliderExitHover()
		}
	})
	g.ReceiveEventType(KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*Slider)
		if ok {
			kt, ok := d.(KeyTypedEvent)
			if ok {
				sl.KeyInput(kt)
			}
		}
	})
}

var SliderProps = []map[string]interface{}{
	{
		"border-width":     "1px",
		"border-radius":    "4px",
		"border-color":     "black",
		"border-style":     "solid",
		"padding":          "8px",
		"margin":           "4px",
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

func (g *Slider) Style2D() {
	ki.SetBitFlag(&g.NodeFlags, int(CanFocus))
	g.Style.SetStyle(nil, &StyleDefault, SliderProps[SliderNormal])
	g.Style2DWidget()
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, SliderProps[i])
		}
		g.StateStyles[i].SetUnitContext(&g.Viewport.Render, 0)
	}
	// todo: how to get state-specific user prefs?  need an extra prefix..
}

func (g *Slider) Layout2D(iter int) {
	if iter == 0 {
		g.InitLayout2D()
		if g.ThumbSize == 0.0 {
			g.Defaults()
		}
		st := &g.Style
		// get at least thumbsize
		sz := g.ThumbSize + 2.0*(st.Layout.Margin.Dots+st.Padding.Dots)
		if g.Horiz {
			g.LayData.AllocSize.Y = sz
		} else {
			g.LayData.AllocSize.X = sz
		}
	} else {
		g.GeomFromLayout() // get our geom from layout -- always do this for widgets  iter > 0
		g.SizeFromAlloc()
	}

	// todo: test for use of parent-el relative units -- indicates whether multiple loops
	// are required
	g.Style.SetUnitContext(&g.Viewport.Render, 0)
	// now get styles for the different states
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i].SetUnitContext(&g.Viewport.Render, 0)
	}
}

func (g *Slider) Node2DBBox() image.Rectangle {
	return g.WinBBoxFromAlloc()
}

func (g *Slider) Render2D() {
	if g.IsLeaf() {
		g.Render2DDefaultStyle()
	} else {
		// todo: manage stacked layout to select appropriate image based on state
		// return
	}
	g.Render2DChildren()
}

// render using a default style if not otherwise styled
func (g *Slider) Render2DDefaultStyle() {
	pc := &g.Paint
	st := &g.Style
	rs := &g.Viewport.Render

	// overall fill box
	g.DrawStdBox(&g.StateStyles[SliderBox])

	// draw a 1/2 thumbsize box with a circular thumb
	spc := st.Layout.Margin.Dots
	pos := g.LayData.AllocPos.AddVal(spc)
	sz := g.LayData.AllocSize.AddVal(-2.0 * spc)
	fullsz := sz

	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	pc.FillStyle.SetColor(&st.Background.Color)

	ht := 0.5 * g.ThumbSize

	if g.Horiz {
		pos.X += ht
		sz.X -= g.ThumbSize
		sz.Y = g.ThumbSize - 2.0*st.Padding.Dots
		ctr := pos.Y + 0.5*fullsz.Y
		pos.Y = ctr - ht + st.Padding.Dots
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)
		sz.X = spc + g.Pos
		pc.FillStyle.SetColor(&g.StateStyles[SliderValueFill].Background.Color)
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)
		pc.FillStyle.SetColor(&st.Background.Color)
		pc.DrawCircle(rs, pos.X+sz.X, ctr, ht)
		pc.FillStrokeClear(rs)
	} else {
		pos.Y += ht
		sz.Y -= g.ThumbSize
		sz.X = g.ThumbSize - 2.0*st.Padding.Dots
		ctr := pos.X + 0.5*fullsz.X
		pos.X = ctr - ht + st.Padding.Dots
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)
		sz.Y = spc + g.Pos
		pc.FillStyle.SetColor(&g.StateStyles[SliderValueFill].Background.Color)
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)
		pc.FillStyle.SetColor(&st.Background.Color)
		pc.DrawCircle(rs, ctr, pos.Y+sz.Y, ht)
		pc.FillStrokeClear(rs)
	}
}

func (g *Slider) CanReRender2D() bool {
	return true
}

func (g *Slider) FocusChanged2D(gotFocus bool) {
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
var _ Node2D = &Slider{}

////////////////////////////////////////////////////////////////////////////////////////
//  ScrollBar

// ScrollBar has a proportional thumb size reflecting amount of content visible
type ScrollBar struct {
	SliderBase
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_ScrollBar = ki.Types.AddType(&ScrollBar{}, nil)

func (g *ScrollBar) Defaults() { // todo: should just get these from props
	g.ValThumb = true
	g.ThumbSize = 20.0
	g.Step = 0.1
	g.PageStep = 0.2
	g.Max = 1.0
}

func (g *ScrollBar) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *ScrollBar) AsViewport2D() *Viewport2D {
	return nil
}

func (g *ScrollBar) AsLayout2D() *Layout {
	return nil
}

func (g *ScrollBar) InitNode2D() {
	g.InitNode2DBase()
	g.ReceiveEventType(MouseDraggedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*ScrollBar)
		if ok {
			if sl.IsDragging() {
				me := d.(MouseDraggedEvent)
				st := sl.PointToRelPos(me.From)
				ed := sl.PointToRelPos(me.Where)
				if sl.Horiz {
					sl.SliderMoved(float64(st.X), float64(ed.X))
				} else {
					sl.SliderMoved(float64(st.Y), float64(ed.Y))
				}
			}
		}
	})
	g.ReceiveEventType(MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*ScrollBar)
		if ok {
			me := d.(MouseDownEvent)
			ed := sl.PointToRelPos(me.Where)
			st := &sl.Style
			spc := st.Layout.Margin.Dots + 0.5*g.ThumbSize
			if sl.Horiz {
				sl.SliderPressed(float64(ed.X) - spc)
			} else {
				sl.SliderPressed(float64(ed.Y) - spc)
			}
		}
	})
	g.ReceiveEventType(MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*ScrollBar)
		if ok {
			sl.SliderReleased()
		}
	})
	g.ReceiveEventType(MouseEnteredEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*ScrollBar)
		if ok {
			sl.SliderEnterHover()
		}
	})
	g.ReceiveEventType(MouseExitedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*ScrollBar)
		if ok {
			sl.SliderExitHover()
		}
	})
	g.ReceiveEventType(KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*ScrollBar)
		if ok {
			kt, ok := d.(KeyTypedEvent)
			if ok {
				sl.KeyInput(kt)
			}
		}
	})
}

var ScrollBarProps = []map[string]interface{}{
	{
		"width":            "20px", // this is automatically applied depending on orientation
		"border-width":     "1px",
		"border-radius":    "4px",
		"border-color":     "black",
		"border-style":     "solid",
		"padding":          "8px",
		"margin":           "4px",
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

func (g *ScrollBar) Style2D() {
	// we can focus by default
	ki.SetBitFlag(&g.NodeFlags, int(CanFocus))
	// first do our normal default styles
	g.Style.SetStyle(nil, &StyleDefault, ScrollBarProps[SliderNormal])
	// then style with user props
	g.Style2DWidget()
	// now get styles for the different states
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, ScrollBarProps[i])
		}
		g.StateStyles[i].SetUnitContext(&g.Viewport.Render, 0)
	}
	// todo: how to get state-specific user prefs?  need an extra prefix..
}

func (g *ScrollBar) Layout2D(iter int) {
	if iter == 0 {
		g.InitLayout2D()
		st := &g.Style
		wd := st.Layout.Width.Dots // this is the width pref
		if wd == 20.0 {            // if we have the default fixed vals
			sz := wd + 2.0*(st.Layout.Margin.Dots+st.Padding.Dots)
			if g.Horiz {
				st.Layout.Height = st.Layout.Width
				st.Layout.MaxHeight = st.Layout.Width
				g.LayData.AllocSize.Y = sz
				st.Layout.Width.Val = 0     // reset
				st.Layout.MaxWidth.Val = -1 // infinite stretch
			} else {
				st.Layout.MaxWidth = st.Layout.Width
				g.LayData.AllocSize.X = sz
				st.Layout.MaxHeight.Val = -1 // infinite stretch
			}
		}
	} else {
		g.GeomFromLayout()
		g.SizeFromAlloc()
	}

	// todo: test for use of parent-el relative units -- indicates whether multiple loops
	// are required
	g.Style.SetUnitContext(&g.Viewport.Render, 0)
	// now get styles for the different states
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i].SetUnitContext(&g.Viewport.Render, 0)
	}
}

func (g *ScrollBar) Node2DBBox() image.Rectangle {
	return g.WinBBoxFromAlloc()
}

func (g *ScrollBar) Render2D() {
	if g.IsLeaf() {
		g.Render2DDefaultStyle()
	} else {
		// todo: manage stacked layout to select appropriate image based on state
		// return
	}
	g.Render2DChildren()
}

// render using a default style if not otherwise styled
func (g *ScrollBar) Render2DDefaultStyle() {
	pc := &g.Paint
	st := &g.Style
	// rs := &g.Viewport.Render

	// overall fill box
	g.DrawStdBox(&g.StateStyles[SliderBox])

	spc := st.Layout.Margin.Dots
	pos := g.LayData.AllocPos.AddVal(spc)
	sz := g.LayData.AllocSize.AddVal(-2.0 * spc)

	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	pc.FillStyle.SetColor(&st.Background.Color)

	if g.Horiz {
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots) // surround box
		pos.X += g.Pos                                // start of thumb
		sz.X = g.ThumbSize
		pc.FillStyle.SetColor(&g.StateStyles[SliderValueFill].Background.Color)
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)
	} else {
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)
		pos.Y += g.Pos
		sz.Y = g.ThumbSize
		pc.FillStyle.SetColor(&g.StateStyles[SliderValueFill].Background.Color)
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)
	}
}

func (g *ScrollBar) CanReRender2D() bool {
	return true
}

func (g *ScrollBar) FocusChanged2D(gotFocus bool) {
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
var _ Node2D = &ScrollBar{}
