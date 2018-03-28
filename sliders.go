// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	// "github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
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
	Min         float64              `xml:"min" desc:"minimum value in range"`
	Max         float64              `xml:"max" desc:"maximum value in range"`
	Step        float64              `xml:"step" desc:"smallest step size to increment"`
	PageStep    float64              `xml:"step" desc:"larger PageUp / Dn step size"`
	Value       float64              `xml:"value" desc:"current value"`
	EmitValue   float64              `xml:"value" desc:"previous emitted value - don't re-emit if it is the same"`
	Size        float64              `xml:"size" desc:"size of the slide box in the relevant dimension -- range of motion -- exclusive of spacing"`
	ThumbSize   float64              `xml:"thumb-size" desc:"size of the thumb -- if ValThumb then this is auto-sized based on ThumbVal and is subtracted from Size in computing Value"`
	ValThumb    bool                 `xml:"prop-thumb","desc:"if true, has a proportionally-sized thumb knob reflecting another value -- e.g., the amount visible in a scrollbar, and thumb is completely inside Size -- otherwise ThumbSize affects Size so that full Size range can be traversed"`
	ThumbVal    float64              `xml:thumb-val" desc:"value that the thumb represents, in the same units"`
	Pos         float64              `xml:"pos" desc:"logical position of the slider relative to Size"`
	DragPos     float64              `xml:"-" desc:"underlying drag position of slider -- not subject to snapping"`
	VisPos      float64              `xml:"vispos" desc:"visual position of the slider -- can be different from pos in a RTL environment"`
	Horiz       bool                 `xml:"horiz" desc:"true if horizontal, else vertical"`
	Tracking    bool                 `xml:"tracking" desc:"if true, will send continuous updates of value changes as user moves the slider -- otherwise only at the end"`
	Snap        bool                 `xml:"snap" desc:"snap the values to Step size increments"`
	StateStyles [SliderStatesN]Style `desc:"styles for different states of the slider, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State       SliderStates
	SliderSig   ki.Signal `json:"-" desc:"signal for slider -- see SliderSignals for the types"`
	// todo: icon -- should be an xml
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_SliderBase = kit.Types.AddType(&SliderBase{}, nil)

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
	// bitflag.Set(&g.NodeFlags, int(SliderFlagDragging))
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
	spc := g.Style.BoxSpace()
	if g.Horiz {
		g.Size = g.LayData.AllocSize.X - 2.0*spc
	} else {
		g.Size = g.LayData.AllocSize.Y - 2.0*spc
	}
	if !g.ValThumb {
		g.Size -= g.ThumbSize // half on each side
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
var KiT_Slider = kit.Types.AddType(&Slider{}, nil)

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

func (g *Slider) Init2D() {
	g.Init2DBase()
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
		"padding":          "6px",
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
	bitflag.Set(&g.NodeFlags, int(CanFocus))
	g.Style.SetStyle(nil, &StyleDefault, SliderProps[SliderNormal])
	g.Style2DWidget()
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, SliderProps[i])
		}
		g.StateStyles[i].SetUnitContext(g.Viewport, Vec2DZero)
	}
	// todo: how to get state-specific user prefs?  need an extra prefix..
}

func (g *Slider) Size2D() {
	g.InitLayout2D()
	if g.ThumbSize == 0.0 {
		g.Defaults()
	}
	st := &g.Style
	// get at least thumbsize + margin + border.size
	sz := g.ThumbSize + 2.0*(st.Layout.Margin.Dots+st.Border.Width.Dots)
	if g.Horiz {
		g.LayData.AllocSize.Y = sz
	} else {
		g.LayData.AllocSize.X = sz
	}
}

func (g *Slider) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.SizeFromAlloc()
	g.Layout2DChildren()
}

func (g *Slider) BBox2D() image.Rectangle {
	return g.BBoxFromAlloc()
}

func (g *Slider) ChildrenBBox2D() image.Rectangle {
	return g.ChildrenBBox2DWidget()
}

func (g *Slider) Render2D() {
	if g.PushBounds() {
		if g.IsLeaf() {
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
func (g *Slider) Render2DDefaultStyle() {
	pc := &g.Paint
	st := &g.Style
	rs := &g.Viewport.Render

	// overall fill box
	g.RenderStdBox(&g.StateStyles[SliderBox])

	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	pc.FillStyle.SetColor(&st.Background.Color)

	// layout is as follows, for width dimension
	// |      bw             bw     |
	// |      | pad |  | pad |      |
	// |  |        thumb         |  |
	// |    spc    | | <- ctr
	//
	// for length: | spc | ht | <-start of slider

	spc := st.BoxSpace()
	pos := g.LayData.AllocPos
	sz := g.LayData.AllocSize
	bpos := pos // box pos
	bsz := sz
	tpos := pos // thumb pos

	ht := 0.5 * g.ThumbSize

	if g.Horiz {
		bpos.Y += spc
		bsz.Y -= 2.0 * spc
		bpos.X += spc + ht
		bsz.X -= 2.0 * (spc + ht)
		g.RenderBoxImpl(bpos, bsz, st.Border.Radius.Dots)

		bsz.X = g.Pos
		pc.FillStyle.SetColor(&g.StateStyles[SliderValueFill].Background.Color)
		g.RenderBoxImpl(bpos, bsz, st.Border.Radius.Dots)

		tpos.X = bpos.X + g.Pos
		tpos.Y += 0.5 * sz.Y // ctr
		pc.FillStyle.SetColor(&st.Background.Color)
		pc.DrawCircle(rs, tpos.X, tpos.Y, ht)
		pc.FillStrokeClear(rs)
	} else {
		bpos.X += spc
		bsz.X -= 2.0 * spc
		bpos.Y += spc + ht
		bsz.Y -= 2.0 * (spc + ht)
		g.RenderBoxImpl(bpos, bsz, st.Border.Radius.Dots)

		bsz.Y = g.Pos
		pc.FillStyle.SetColor(&g.StateStyles[SliderValueFill].Background.Color)
		g.RenderBoxImpl(bpos, bsz, st.Border.Radius.Dots)

		tpos.Y = bpos.Y + g.Pos
		tpos.X += 0.5 * sz.X // ctr
		pc.FillStyle.SetColor(&st.Background.Color)
		pc.DrawCircle(rs, tpos.X, tpos.Y, ht)
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
var KiT_ScrollBar = kit.Types.AddType(&ScrollBar{}, nil)

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

func (g *ScrollBar) Init2D() {
	g.Init2DBase()
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

func (g *ScrollBar) Style2D() {
	// we can focus by default
	bitflag.Set(&g.NodeFlags, int(CanFocus))
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
		g.StateStyles[i].SetUnitContext(g.Viewport, Vec2DZero)
	}
	// todo: how to get state-specific user prefs?  need an extra prefix..
}

func (g *ScrollBar) Size2D() {
	g.InitLayout2D()
	// st := &g.Style
	// if we have the default fixed width vals, then update based on orientation
	// if st.Layout.Width.Val == 12.0 && st.Layout.Width.Un == units.Px {
	// 	if g.Horiz {
	// 		st.Layout.Height = st.Layout.Width
	// 		st.Layout.MaxHeight = st.Layout.Width
	// 		g.LayData.AllocSize.Y = st.Layout.Width.Dots
	// 		st.Layout.Width.Val = 0     // reset
	// 		st.Layout.MaxWidth.Val = -1 // infinite stretch
	// 	} else {
	// 		st.Layout.MaxWidth = st.Layout.Width
	// 		g.LayData.AllocSize.X = st.Layout.Width.Dots
	// 		st.Layout.MaxHeight.Val = -1 // infinite stretch
	// 	}
	// }
}

func (g *ScrollBar) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.SizeFromAlloc()
	g.Layout2DChildren()
}

func (g *ScrollBar) BBox2D() image.Rectangle {
	return g.BBoxFromAlloc()
}

func (g *ScrollBar) ChildrenBBox2D() image.Rectangle {
	return g.ChildrenBBox2DWidget()
}

func (g *ScrollBar) Render2D() {
	if g.PushBounds() {
		if g.IsLeaf() {
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
func (g *ScrollBar) Render2DDefaultStyle() {
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

	if g.Horiz {
		g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots) // surround box
		pos.X += g.Pos                                  // start of thumb
		sz.X = g.ThumbSize
		pc.FillStyle.SetColor(&g.StateStyles[SliderValueFill].Background.Color)
		g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
	} else {
		g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
		pos.Y += g.Pos
		sz.Y = g.ThumbSize
		pc.FillStyle.SetColor(&g.StateStyles[SliderValueFill].Background.Color)
		g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
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
