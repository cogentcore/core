// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"math"
	"strconv"

	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
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
	SliderActive SliderStates = iota
	// disabled -- not pressable
	SliderDisabled
	// mouse is hovering over the slider
	SliderHover
	// slider is the focus -- will respond to keyboard input
	SliderFocus
	// slider is currently being pressed down
	SliderDown
	// use background-color here to fill in selected value of slider
	SliderValue
	// these styles define the overall box around slider -- typically no border and a white background -- needs a background to allow local re-rendering
	SliderBox
	// total number of slider states
	SliderStatesN
)

//go:generate stringer -type=SliderStates

// Style selector names for the different states
var SliderSelectors = []string{":active", ":disabled", ":hover", ":focus", ":down", ":value", ":box"}

// SliderBase has common slider functionality -- two major modes: ValThumb =
// false is a slider with a fixed-size thumb knob, while = true has a thumb
// that represents a value, as in a scrollbar, and the scrolling range is size
// - thumbsize
type SliderBase struct {
	WidgetBase
	Value       float64              `xml:"value" desc:"current value"`
	EmitValue   float64              `xml:"-" desc:"previous emitted value - don't re-emit if it is the same"`
	Min         float64              `xml:"min" desc:"minimum value in range"`
	Max         float64              `xml:"max" desc:"maximum value in range"`
	Step        float64              `xml:"step" desc:"smallest step size to increment"`
	PageStep    float64              `xml:"pagestep" desc:"larger PageUp / Dn step size"`
	Size        float64              `xml:"size" desc:"size of the slide box in the relevant dimension -- range of motion -- exclusive of spacing"`
	ThumbSize   float64              `xml:"thumb-size" desc:"size of the thumb -- if ValThumb then this is auto-sized based on ThumbVal and is subtracted from Size in computing Value"`
	Prec        int                  `desc:"specifies the precision of decimal places (total, not after the decimal point) to use in representing the number -- this helps to truncate small weird floating point values in the nether regions"`
	Icon        *Icon                `desc:"optional icon for the dragging knob"`
	ValThumb    bool                 `xml:"prop-thumb","desc:"if true, has a proportionally-sized thumb knob reflecting another value -- e.g., the amount visible in a scrollbar, and thumb is completely inside Size -- otherwise ThumbSize affects Size so that full Size range can be traversed"`
	ThumbVal    float64              `xml:thumb-val" desc:"value that the thumb represents, in the same units"`
	Pos         float64              `xml:"pos" desc:"logical position of the slider relative to Size"`
	DragPos     float64              `xml:"-" desc:"underlying drag position of slider -- not subject to snapping"`
	VisPos      float64              `xml:"vispos" desc:"visual position of the slider -- can be different from pos in a RTL environment"`
	Dim         Dims2D               `desc:"dimension along which the slider slides"`
	Tracking    bool                 `xml:"tracking" desc:"if true, will send continuous updates of value changes as user moves the slider -- otherwise only at the end -- see ThrackThr for a threshold on amount of change"`
	TrackThr    float64              `xml:"threshold for amount of change in scroll value before emitting a signal in Tracking mode"`
	Snap        bool                 `xml:"snap" desc:"snap the values to Step size increments"`
	StateStyles [SliderStatesN]Style `desc:"styles for different states of the slider, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State       SliderStates
	SliderSig   ki.Signal `json:"-" desc:"signal for slider -- see SliderSignals for the types"`
	// todo: icon -- should be an xml
	origWinBBox image.Rectangle `desc:"copy of the win bbox, used for translating mouse events for cases like splitter where the bbox is restricted to the slider itself"`
}

var KiT_SliderBase = kit.Types.AddType(&SliderBase{}, nil)

// if snap is set, then snap the value to step sizes
func (g *SliderBase) SnapValue() {
	if g.Snap {
		g.Value = float64(int(math.Round(g.Value/g.Step))) * g.Step
		frep := strconv.FormatFloat(g.Value, 'g', g.Prec, 64)
		g.Value, _ = strconv.ParseFloat(frep, 64)
	}
}

// set the slider state to target
func (g *SliderBase) SetSliderState(state SliderStates) {
	// todo: process disabled state -- probably just deal with the property directly?
	// it overrides any choice here and just sets state to disabled..
	if state == SliderActive && g.HasFocus() {
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
	// bitflag.Set(&g.Flag, int(SliderFlagDragging))
	g.UpdateEnd()
}

// the slider has just been released -- sends a released signal and returns
// state to normal, and emits clicked signal if if it was previously in pressed state
func (g *SliderBase) SliderReleased() {
	wasPressed := (g.State == SliderDown)
	g.UpdateStart()
	g.SetSliderState(SliderActive)
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
		g.SetSliderState(SliderActive)
		g.UpdateEnd()
	}
}

// get size from allocation
func (g *SliderBase) SizeFromAlloc() {
	if g.LayData.AllocSize.IsZero() {
		return
	}
	spc := g.Style.BoxSpace()
	g.Size = g.LayData.AllocSize.Dim(g.Dim) - 2.0*spc
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
	frep := strconv.FormatFloat(g.Value, 'g', g.Prec, 64)
	g.Value, _ = strconv.ParseFloat(frep, 64)
	g.DragPos = g.Pos
	if g.Snap {
		g.SnapValue()
		g.UpdatePosFromValue()
	}
	if g.Tracking && g.Value != g.EmitValue {
		if math.Abs(g.Value-g.EmitValue) > g.TrackThr {
			g.SliderSig.Emit(g.This, int64(SliderValueChanged), g.Value)
			g.EmitValue = g.Value
		}
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

func (g *SliderBase) KeyInput(kt *oswin.KeyTypedEvent) {
	kf := KeyFun(kt.Key, kt.Chord)
	switch kf {
	case KeyFunMoveUp:
		g.SetValue(g.Value - g.Step)
		kt.SetProcessed()
	case KeyFunMoveLeft:
		g.SetValue(g.Value - g.Step)
		kt.SetProcessed()
	case KeyFunMoveDown:
		g.SetValue(g.Value + g.Step)
		kt.SetProcessed()
	case KeyFunMoveRight:
		g.SetValue(g.Value + g.Step)
		kt.SetProcessed()
	case KeyFunPageUp:
		g.SetValue(g.Value - g.PageStep)
		kt.SetProcessed()
	case KeyFunPageLeft:
		g.SetValue(g.Value - g.PageStep)
		kt.SetProcessed()
	case KeyFunPageDown:
		g.SetValue(g.Value + g.PageStep)
		kt.SetProcessed()
	case KeyFunPageRight:
		g.SetValue(g.Value + g.PageStep)
		kt.SetProcessed()
	case KeyFunHome:
		g.SetValue(g.Min)
		kt.SetProcessed()
	case KeyFunEnd:
		g.SetValue(g.Max)
		kt.SetProcessed()
	}
}

// translate a point in global pixel coords into relative position within node
func (g *SliderBase) PointToRelPos(pt image.Point) image.Point {
	return pt.Sub(g.origWinBBox.Min)
}

func (g *SliderBase) Init2DSlider() {
	g.Init2DWidget()
	g.ReceiveEventType(oswin.MouseDraggedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		(d.(oswin.Event)).SetProcessed()
		sl := recv.EmbeddedStruct(KiT_SliderBase).(*SliderBase)
		if sl.IsDragging() {
			me := d.(*oswin.MouseDraggedEvent)
			st := sl.PointToRelPos(me.From)
			ed := sl.PointToRelPos(me.Where)
			if sl.Dim == X {
				sl.SliderMoved(float64(st.X), float64(ed.X))
			} else {
				sl.SliderMoved(float64(st.Y), float64(ed.Y))
			}
		}
	})
	g.ReceiveEventType(oswin.MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		(d.(oswin.Event)).SetProcessed()
		sl := recv.EmbeddedStruct(KiT_SliderBase).(*SliderBase)
		me := d.(*oswin.MouseDownEvent)
		ed := sl.PointToRelPos(me.Where)
		st := &sl.Style
		spc := st.Layout.Margin.Dots + 0.5*g.ThumbSize
		if sl.Dim == X {
			sl.SliderPressed(float64(ed.X) - spc)
		} else {
			sl.SliderPressed(float64(ed.Y) - spc)
		}
	})
	g.ReceiveEventType(oswin.MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		(d.(oswin.Event)).SetProcessed()
		sl := recv.EmbeddedStruct(KiT_SliderBase).(*SliderBase)
		sl.SliderReleased()
	})
	g.ReceiveEventType(oswin.MouseEnteredEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		(d.(oswin.Event)).SetProcessed()
		sl := recv.EmbeddedStruct(KiT_SliderBase).(*SliderBase)
		sl.SliderEnterHover()
	})
	g.ReceiveEventType(oswin.MouseExitedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		(d.(oswin.Event)).SetProcessed()
		sl := recv.EmbeddedStruct(KiT_SliderBase).(*SliderBase)
		sl.SliderExitHover()
	})
	g.ReceiveEventType(oswin.KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl := recv.EmbeddedStruct(KiT_SliderBase).(*SliderBase)
		sl.KeyInput(d.(*oswin.KeyTypedEvent))
	})
}

func (g *SliderBase) ConfigParts() {
	g.Parts.Lay = LayoutNil
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(g.Icon, "")
	g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(g.Icon, "", icIdx, lbIdx, g.StyleProps(SliderSelectors[SliderActive]))
}

func (g *SliderBase) ConfigPartsIfNeeded(render bool) {
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
			ic.LayData.AllocPosRel.SetDim(g.Dim, g.Pos+spc-0.5*g.ThumbSize)
			ic.LayData.AllocPosRel.SetDim(odim, -pad)
			ic.LayData.AllocSize.X = g.ThumbSize
			ic.LayData.AllocSize.Y = g.ThumbSize
			if render {
				ic.Layout2DTree()
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  Slider

// Slider is a standard value slider with a fixed-sized thumb knob -- if an Icon is set, it is used for the knob of the slider
type Slider struct {
	SliderBase
}

var KiT_Slider = kit.Types.AddType(&Slider{}, SliderProps)

var SliderProps = map[string]interface{}{
	SliderSelectors[SliderActive]: map[string]interface{}{
		"border-width":     "1px",
		"border-radius":    "4px",
		"border-color":     "black",
		"border-style":     "solid",
		"padding":          "6px",
		"margin":           "4px",
		"background-color": "#EEF",
		"#icon": map[string]interface{}{
			"width":   units.NewValue(1, units.Em),
			"height":  units.NewValue(1, units.Em),
			"margin":  units.NewValue(0, units.Px),
			"padding": units.NewValue(0, units.Px),
		},
	},
	SliderSelectors[SliderDisabled]: map[string]interface{}{
		"border-color":     "#BBB",
		"background-color": "#DDD",
	},
	SliderSelectors[SliderHover]: map[string]interface{}{
		"background-color": "#CCF", // todo "darker"
	},
	SliderSelectors[SliderFocus]: map[string]interface{}{
		"border-color":     "#008",
		"background.color": "#CCF",
	},
	SliderSelectors[SliderDown]: map[string]interface{}{
		"border-color":     "#000",
		"background-color": "#DDF",
	},
	SliderSelectors[SliderValue]: map[string]interface{}{
		"border-color":     "#00F",
		"background-color": "#00F",
	},
	SliderSelectors[SliderBox]: map[string]interface{}{
		"border-color":     "#FFF",
		"background-color": "#FFF",
	},
}

func (g *Slider) Defaults() { // todo: should just get these from props
	g.ThumbSize = 25.0
	g.Step = 0.1
	g.PageStep = 0.2
	g.Max = 1.0
	g.Prec = 9
}

func (g *Slider) Init2D() {
	g.Init2DSlider()
	g.ConfigParts()
}

func (g *Slider) Style2D() {
	bitflag.Set(&g.Flag, int(CanFocus))
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

func (g *Slider) Size2D() {
	g.InitLayout2D()
	if g.ThumbSize == 0.0 {
		g.Defaults()
	}
	st := &g.Style
	// get at least thumbsize + margin + border.size
	sz := g.ThumbSize + 2.0*(st.Layout.Margin.Dots+st.Border.Width.Dots)
	g.LayData.AllocSize.SetDim(OtherDim(g.Dim), sz)
}

func (g *Slider) Layout2D(parBBox image.Rectangle) {
	g.ConfigPartsIfNeeded(false)
	g.Layout2DWidget(parBBox)
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.SizeFromAlloc()
	g.Layout2DChildren()
	g.origWinBBox = g.WinBBox
}

func (g *Slider) Render2D() {
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
func (g *Slider) Render2DDefaultStyle() {
	pc := &g.Paint
	st := &g.Style
	rs := &g.Viewport.Render

	g.ConfigPartsIfNeeded(true)

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

	odim := OtherDim(g.Dim)
	bpos.SetAddDim(odim, spc)
	bsz.SetSubDim(odim, 2.0*spc)
	bpos.SetAddDim(g.Dim, spc+ht)
	bsz.SetSubDim(g.Dim, 2.0*(spc+ht))
	g.RenderBoxImpl(bpos, bsz, st.Border.Radius.Dots)

	bsz.SetDim(g.Dim, g.Pos)
	pc.FillStyle.SetColor(&g.StateStyles[SliderValue].Background.Color)
	g.RenderBoxImpl(bpos, bsz, st.Border.Radius.Dots)

	tpos.SetDim(g.Dim, bpos.Dim(g.Dim)+g.Pos)
	tpos.SetAddDim(odim, 0.5*sz.Dim(odim)) // ctr
	pc.FillStyle.SetColor(&st.Background.Color)

	if g.Icon != nil && g.Parts.HasChildren() {
		g.Parts.Render2DTree()
	} else {
		pc.DrawCircle(rs, tpos.X, tpos.Y, ht)
		pc.FillStrokeClear(rs)
	}
}

func (g *Slider) FocusChanged2D(gotFocus bool) {
	g.UpdateStart()
	if gotFocus {
		g.SetSliderState(SliderFocus)
	} else {
		g.SetSliderState(SliderActive) // lose any hover state but whatever..
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

var KiT_ScrollBar = kit.Types.AddType(&ScrollBar{}, ScrollBarProps)

var ScrollBarProps = map[string]interface{}{
	SliderSelectors[SliderActive]: map[string]interface{}{
		"width":            "16px", // assumes vertical -- user needs to set!
		"min-width":        "16px",
		"border-width":     "1px",
		"border-radius":    "4px",
		"border-color":     "black",
		"border-style":     "solid",
		"padding":          "0px",
		"margin":           "2px",
		"background-color": "#EEF",
	},
	SliderSelectors[SliderDisabled]: map[string]interface{}{
		"border-color":     "#BBB",
		"background-color": "#DDD",
	},
	SliderSelectors[SliderHover]: map[string]interface{}{
		"background-color": "#CCF", // todo "darker"
	},
	SliderSelectors[SliderFocus]: map[string]interface{}{
		"border-color":     "#008",
		"background.color": "#CCF",
	},
	SliderSelectors[SliderDown]: map[string]interface{}{
		"border-color":     "#000",
		"background-color": "#DDF",
	},
	SliderSelectors[SliderValue]: map[string]interface{}{
		"border-color":     "#00F",
		"background-color": "#00F",
	},
	SliderSelectors[SliderBox]: map[string]interface{}{
		"border-color":     "#FFF",
		"background-color": "#FFF",
	},
}

func (g *ScrollBar) Defaults() { // todo: should just get these from props
	g.ValThumb = true
	g.ThumbSize = 20.0
	g.Step = 0.1
	g.PageStep = 0.2
	g.Max = 1.0
	g.Prec = 9
}

func (g *ScrollBar) Init2D() {
	g.Init2DSlider()
}

func (g *ScrollBar) Style2D() {
	bitflag.Set(&g.Flag, int(CanFocus))
	g.Style2DWidget(g.StyleProps(SliderSelectors[SliderActive]))
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, g.StyleProps(SliderSelectors[i]))
		}
		g.StateStyles[i].SetUnitContext(g.Viewport, Vec2DZero)
	}
	// todo: how to get state-specific user prefs?  need an extra prefix..
}

func (g *ScrollBar) Size2D() {
	g.InitLayout2D()
}

func (g *ScrollBar) Layout2D(parBBox image.Rectangle) {
	g.Layout2DWidget(parBBox)
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.SizeFromAlloc()
	g.Layout2DChildren()
	g.origWinBBox = g.WinBBox
}

func (g *ScrollBar) Render2D() {
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

	g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots) // surround box
	pos.SetAddDim(g.Dim, g.Pos)                     // start of thumb
	sz.SetDim(g.Dim, g.ThumbSize)
	pc.FillStyle.SetColor(&g.StateStyles[SliderValue].Background.Color)
	g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
}

func (g *ScrollBar) FocusChanged2D(gotFocus bool) {
	g.UpdateStart()
	if gotFocus {
		g.SetSliderState(SliderFocus)
	} else {
		g.SetSliderState(SliderActive) // lose any hover state but whatever..
	}
	g.UpdateEnd()
}

// check for interface implementation
var _ Node2D = &ScrollBar{}
