// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/chewxy/math32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// SliderBase -- basis for sliders

// SliderBase has common slider functionality -- two major modes: ValThumb =
// false is a slider with a fixed-size thumb knob, while = true has a thumb
// that represents a value, as in a scrollbar, and the scrolling range is size
// - thumbsize
type SliderBase struct {
	PartsWidgetBase
	Value       float32              `xml:"value" desc:"current value"`
	EmitValue   float32              `xml:"-" desc:"previous emitted value - don't re-emit if it is the same"`
	Min         float32              `xml:"min" desc:"minimum value in range"`
	Max         float32              `xml:"max" desc:"maximum value in range"`
	Step        float32              `xml:"step" desc:"smallest step size to increment"`
	PageStep    float32              `xml:"pagestep" desc:"larger PageUp / Dn step size"`
	Size        float32              `xml:"size" desc:"size of the slide box in the relevant dimension -- range of motion -- exclusive of spacing"`
	ThSize      float32              `xml:"-" desc:"computed size of the thumb -- if ValThumb then this is auto-sized based on ThumbVal and is subtracted from Size in computing Value"`
	ThumbSize   units.Value          `xml:"thumb-size" desc:"styled fixed size of the thumb"`
	Prec        int                  `xml:"prec" desc:"specifies the precision of decimal places (total, not after the decimal point) to use in representing the number -- this helps to truncate small weird floating point values in the nether regions"`
	Icon        IconName             `view:"show-name" desc:"optional icon for the dragging knob"`
	ValThumb    bool                 `xml:"val-thumb" alt:"prop-thumb" desc:"if true, has a proportionally-sized thumb knob reflecting another value -- e.g., the amount visible in a scrollbar, and thumb is completely inside Size -- otherwise ThumbSize affects Size so that full Size range can be traversed"`
	ThumbVal    float32              `xml:"thumb-val" desc:"value that the thumb represents, in the same units"`
	Pos         float32              `xml:"pos" desc:"logical position of the slider relative to Size"`
	DragPos     float32              `xml:"-" desc:"underlying drag position of slider -- not subject to snapping"`
	VisPos      float32              `xml:"vispos" desc:"visual position of the slider -- can be different from pos in a RTL environment"`
	Dim         Dims2D               `desc:"dimension along which the slider slides"`
	Tracking    bool                 `xml:"tracking" desc:"if true, will send continuous updates of value changes as user moves the slider -- otherwise only at the end -- see ThrackThr for a threshold on amount of change"`
	TrackThr    float32              `xml:"track-thr" desc:"threshold for amount of change in scroll value before emitting a signal in Tracking mode"`
	Snap        bool                 `xml:"snap" desc:"snap the values to Step size increments"`
	State       SliderStates         `json:"-" xml:"-" desc:"state of slider"`
	StateStyles [SliderStatesN]Style `json:"-" xml:"-" desc:"styles for different states of the slider, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	SliderSig   ki.Signal            `json:"-" xml:"-" view:"-" desc:"signal for slider -- see SliderSignals for the types"`
	// todo: icon -- should be an xml
	OrigWinBBox image.Rectangle `desc:"copy of the win bbox, used for translating mouse events for cases like splitter where the bbox is restricted to the slider itself"`
}

var KiT_SliderBase = kit.Types.AddType(&SliderBase{}, SliderBaseProps)

var SliderBaseProps = ki.Props{
	"base-type": true,
}

// SliderSignals are signals that sliders can send
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

// SliderStates are mutually-exclusive slider states -- determines appearance
type SliderStates int32

const (
	// normal state -- there but not being interacted with
	SliderActive SliderStates = iota

	// inactive -- not responsive
	SliderInactive

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

// SliderSelectors are Style selector names for the different states
var SliderSelectors = []string{":active", ":inactive", ":hover", ":focus", ":down", ":value", ":box"}

func (g *SliderBase) Defaults() { // todo: should just get these from props
	g.ThumbSize = units.NewValue(1, units.Em)
	g.Step = 0.1
	g.PageStep = 0.2
	g.Max = 1.0
	g.Prec = 9
}

// SnapValue snaps the value to step sizes if snap option is set
func (g *SliderBase) SnapValue() {
	if g.Snap {
		g.Value = FloatMod32(g.Value, g.Step)
		g.Value = Truncate32(g.Value, g.Prec)
	}
}

// SetSliderState sets the slider state to given state, updates style
func (g *SliderBase) SetSliderState(state SliderStates) {
	if state == SliderActive && g.HasFocus() {
		state = SliderFocus
	}
	g.State = state
	g.Sty = g.StateStyles[state] // get relevant styles
}

// SliderPressed sets the slider in the down state -- mouse clicked down but
// not yet up -- emits SliderPressed signal
func (g *SliderBase) SliderPressed(pos float32) {
	g.EmitValue = g.Min - 1.0 // invalid value
	updt := g.UpdateStart()
	g.SetSliderState(SliderDown)
	g.SetSliderPos(pos)
	g.SliderSig.Emit(g.This, int64(SliderPressed), g.Value)
	// bitflag.Set(&g.Flag, int(SliderFlagDragging))
	g.UpdateEnd(updt)
}

// SliderReleased called when the slider has just been released -- sends a
// released signal and returns state to normal, and emits clicked signal if if
// it was previously in pressed state
func (g *SliderBase) SliderReleased() {
	wasPressed := (g.State == SliderDown)
	updt := g.UpdateStart()
	g.SetSliderState(SliderActive)
	g.SliderSig.Emit(g.This, int64(SliderReleased), g.Value)
	if wasPressed && g.Value != g.EmitValue {
		g.SliderSig.Emit(g.This, int64(SliderValueChanged), g.Value)
		g.EmitValue = g.Value
	}
	g.UpdateEnd(updt)
}

// SliderEnterHover slider starting hover
func (g *SliderBase) SliderEnterHover() {
	if g.State != SliderHover {
		updt := g.UpdateStart()
		g.SetSliderState(SliderHover)
		g.UpdateEnd(updt)
	}
}

// SliderExitHover called when slider exiting hover
func (g *SliderBase) SliderExitHover() {
	if g.State == SliderHover {
		updt := g.UpdateStart()
		g.SetSliderState(SliderActive)
		g.UpdateEnd(updt)
	}
}

// SizeFromAlloc gets size from allocation
func (g *SliderBase) SizeFromAlloc() {
	if g.LayData.AllocSize.IsZero() {
		return
	}
	spc := g.Sty.BoxSpace()
	g.Size = g.LayData.AllocSize.Dim(g.Dim) - 2.0*spc
	if !g.ValThumb {
		g.Size -= g.ThSize // half on each side
	}
	g.UpdatePosFromValue()
	g.DragPos = g.Pos
}

// SetSliderPos sets the position of the slider at the given position in
// pixels -- updates the corresponding Value
func (g *SliderBase) SetSliderPos(pos float32) {
	updt := g.UpdateStart()
	g.Pos = pos
	g.Pos = Min32(g.Size, g.Pos)
	if g.ValThumb {
		g.UpdateThumbValSize()
		g.Pos = Min32(g.Size-g.ThSize, g.Pos)
	}
	g.Pos = Max32(0, g.Pos)
	g.Value = Truncate32(g.Min+(g.Max-g.Min)*(g.Pos/g.Size), g.Prec)
	g.DragPos = g.Pos
	if g.Snap {
		g.SnapValue()
		g.UpdatePosFromValue()
	}
	if g.Tracking && g.Value != g.EmitValue {
		if math32.Abs(g.Value-g.EmitValue) > g.TrackThr {
			g.SliderSig.Emit(g.This, int64(SliderValueChanged), g.Value)
			g.EmitValue = g.Value
		}
	}
	g.UpdateEnd(updt)
}

// SliderMoved called when slider moved along relevant axis
func (g *SliderBase) SliderMoved(start, end float32) {
	del := end - start
	g.SetSliderPos(g.DragPos + del)
}

// UpdatePosFromValue updates the slider position based on the current Value
func (g *SliderBase) UpdatePosFromValue() {
	if g.Size == 0.0 {
		return
	}
	if g.ValThumb {
		g.UpdateThumbValSize()
	}
	g.Pos = g.Size * (g.Value - g.Min) / (g.Max - g.Min)
}

// SetValue sets the value and updates the slider position, but does not
// emit an updated signal (see SetValueAction)
func (g *SliderBase) SetValue(val float32) {
	updt := g.UpdateStart()
	val = Min32(val, g.Max)
	if g.ValThumb {
		val = Min32(val, g.Max-g.ThumbVal)
	}
	val = Max32(val, g.Min)
	if g.Value != val {
		g.Value = val
		g.UpdatePosFromValue()
		g.DragPos = g.Pos
	}
	g.UpdateEnd(updt)
}

// SetValueAction sets the value and updates the slider representation, and
// emits a changed signal
func (g *SliderBase) SetValueAction(val float32) {
	if g.Value == val {
		return
	}
	g.SetValue(val)
	g.SliderSig.Emit(g.This, int64(SliderValueChanged), g.Value)
}

// SetThumValue sets the thumb value to given value and updates the thumb size
// -- for scrollbar-style sliders where the thumb size represents visible range
func (g *SliderBase) SetThumbValue(val float32) {
	updt := g.UpdateStart()
	g.ThumbVal = Min32(val, g.Max)
	g.ThumbVal = Max32(g.ThumbVal, g.Min)
	g.UpdateThumbValSize()
	g.UpdateEnd(updt)
}

// UpdateThumbValSize sets thumb size as proportion of min / max (e.g., amount
// visible in scrollbar) -- max's out to full size
func (g *SliderBase) UpdateThumbValSize() {
	g.ThSize = ((g.ThumbVal - g.Min) / (g.Max - g.Min))
	g.ThSize = Min32(g.ThSize, 1.0)
	g.ThSize = Max32(g.ThSize, 0.0)
	g.ThSize *= g.Size
}

func (g *SliderBase) KeyInput(kt *key.ChordEvent) {
	kf := KeyFun(kt.ChordString())
	switch kf {
	case KeyFunMoveUp:
		g.SetValueAction(g.Value - g.Step)
		kt.SetProcessed()
	case KeyFunMoveLeft:
		g.SetValueAction(g.Value - g.Step)
		kt.SetProcessed()
	case KeyFunMoveDown:
		g.SetValueAction(g.Value + g.Step)
		kt.SetProcessed()
	case KeyFunMoveRight:
		g.SetValueAction(g.Value + g.Step)
		kt.SetProcessed()
	case KeyFunPageUp:
		g.SetValueAction(g.Value - g.PageStep)
		kt.SetProcessed()
	case KeyFunPageLeft:
		g.SetValueAction(g.Value - g.PageStep)
		kt.SetProcessed()
	case KeyFunPageDown:
		g.SetValueAction(g.Value + g.PageStep)
		kt.SetProcessed()
	case KeyFunPageRight:
		g.SetValueAction(g.Value + g.PageStep)
		kt.SetProcessed()
	case KeyFunHome:
		g.SetValueAction(g.Min)
		kt.SetProcessed()
	case KeyFunEnd:
		g.SetValueAction(g.Max)
		kt.SetProcessed()
	}
}

// PointToRelPos translates a point in global pixel coords into relative
// position within node
func (g *SliderBase) PointToRelPos(pt image.Point) image.Point {
	return pt.Sub(g.OrigWinBBox.Min)
}

func (g *SliderBase) SliderEvents() {
	g.ConnectEventType(oswin.MouseDragEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		sl := recv.Embed(KiT_SliderBase).(*SliderBase)
		if sl.IsInactive() {
			return
		}
		if sl.IsDragging() {
			me.SetProcessed()
			st := sl.PointToRelPos(me.From)
			ed := sl.PointToRelPos(me.Where)
			if sl.Dim == X {
				sl.SliderMoved(float32(st.X), float32(ed.X))
			} else {
				sl.SliderMoved(float32(st.Y), float32(ed.Y))
			}
		}
	})
	g.ConnectEventType(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		sl := recv.Embed(KiT_SliderBase).(*SliderBase)
		if sl.IsInactive() {
			me.SetProcessed()
			sl.SetSelectedState(!sl.IsSelected())
			sl.EmitSelectedSignal()
			sl.UpdateSig()
		} else {
			if me.Button == mouse.Left {
				me.SetProcessed()
				if me.Action == mouse.Press {
					ed := sl.PointToRelPos(me.Where)
					st := &sl.Sty
					spc := st.Layout.Margin.Dots + 0.5*g.ThSize
					if sl.Dim == X {
						sl.SliderPressed(float32(ed.X) - spc)
					} else {
						sl.SliderPressed(float32(ed.Y) - spc)
					}
				} else {
					sl.SliderReleased()
				}
			}
		}
	})
	g.ConnectEventType(oswin.MouseFocusEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl := recv.Embed(KiT_SliderBase).(*SliderBase)
		if sl.IsInactive() {
			return
		}
		me := d.(*mouse.FocusEvent)
		me.SetProcessed()
		if me.Action == mouse.Enter {
			sl.SliderEnterHover()
		} else {
			sl.SliderExitHover()
		}
	})
	g.ConnectEventType(oswin.MouseScrollEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl := recv.Embed(KiT_SliderBase).(*SliderBase)
		if sl.IsInactive() {
			return
		}
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		cur := float32(sl.Pos)
		if sl.Dim == X {
			sl.SliderMoved(cur, cur+float32(me.NonZeroDelta(true))) // preferX
		} else {
			sl.SliderMoved(cur, cur-float32(me.NonZeroDelta(false))) // preferY
		}
	})
	g.ConnectEventType(oswin.KeyChordEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl := recv.Embed(KiT_SliderBase).(*SliderBase)
		if sl.IsInactive() {
			return
		}
		sl.KeyInput(d.(*key.ChordEvent))
	})
}

func (g *SliderBase) Init2DSlider() {
	g.Init2DWidget()
}

func (g *SliderBase) ConfigParts() {
	g.Parts.Lay = LayoutNil
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(string(g.Icon), "")
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(string(g.Icon), "", icIdx, lbIdx)
	if mods {
		g.UpdateEnd(updt)
	}
}

func (g *SliderBase) ConfigPartsIfNeeded(render bool) {
	if g.PartsNeedUpdateIconLabel(string(g.Icon), "") {
		g.ConfigParts()
	}
	if g.Icon.IsValid() && g.Parts.HasChildren() {
		ick, ok := g.Parts.Children().ElemByType(KiT_Icon, true, 0)
		if ok {
			ic := ick.(*Icon)
			mrg := g.Sty.Layout.Margin.Dots
			pad := g.Sty.Layout.Padding.Dots
			spc := mrg + pad
			odim := OtherDim(g.Dim)
			ic.LayData.AllocPosRel.SetDim(g.Dim, g.Pos+spc-0.5*g.ThSize)
			ic.LayData.AllocPosRel.SetDim(odim, -pad)
			ic.LayData.AllocSize.X = g.ThSize
			ic.LayData.AllocSize.Y = g.ThSize
			if render {
				ic.Layout2DTree()
			}
		}
	}
}

// SliderDefault is default obj that can be used when property specifies "default"
var SliderDefault SliderBase

// SliderFields contain the StyledFields for Slider type
var SliderFields = initSlider()

func initSlider() *StyledFields {
	SliderDefault = SliderBase{}
	SliderDefault.Defaults()
	sf := &StyledFields{}
	sf.Init(&SliderDefault)
	return sf
}

////////////////////////////////////////////////////////////////////////////////////////
//  Slider

// Slider is a standard value slider with a fixed-sized thumb knob -- if an
// Icon is set, it is used for the knob of the slider
type Slider struct {
	SliderBase
}

var KiT_Slider = kit.Types.AddType(&Slider{}, SliderProps)

var SliderProps = ki.Props{
	"border-width":     units.NewValue(1, units.Px),
	"border-radius":    units.NewValue(4, units.Px),
	"border-color":     &Prefs.Colors.Border,
	"border-style":     BorderSolid,
	"padding":          units.NewValue(6, units.Px),
	"margin":           units.NewValue(4, units.Px),
	"background-color": &Prefs.Colors.Control,
	"color":            &Prefs.Colors.Font,
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &Prefs.Colors.Icon,
		"stroke":  &Prefs.Colors.Font,
	},
	SliderSelectors[SliderActive]: ki.Props{
		"background-color": "lighter-0",
	},
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
	SliderSelectors[SliderDown]: ki.Props{
		"background-color": "highlight-20",
	},
	SliderSelectors[SliderValue]: ki.Props{
		"border-color":     &Prefs.Colors.Icon,
		"background-color": &Prefs.Colors.Icon,
	},
	SliderSelectors[SliderBox]: ki.Props{
		"border-color":     &Prefs.Colors.Background,
		"background-color": &Prefs.Colors.Background,
	},
}

func (g *Slider) Defaults() {
	g.ThumbSize = units.NewValue(1.5, units.Em)
	g.ThSize = 25.0
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
	g.SetCanFocusIfActive()
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

func (g *Slider) Size2D() {
	g.InitLayout2D()
	if g.ThSize == 0.0 {
		g.Defaults()
	}
	st := &g.Sty
	// get at least thumbsize + margin + border.size
	sz := g.ThSize + 2.0*(st.Layout.Margin.Dots+st.Border.Width.Dots)
	g.LayData.AllocSize.SetDim(OtherDim(g.Dim), sz)
}

func (g *Slider) Layout2D(parBBox image.Rectangle, iter int) bool {
	g.ConfigPartsIfNeeded(false)
	g.Layout2DBase(parBBox, true, iter) // init style
	g.Layout2DParts(parBBox, iter)
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Sty.UnContext)
	}
	g.SizeFromAlloc()
	g.OrigWinBBox = g.WinBBox
	return g.Layout2DChildren(iter)
}

func (g *Slider) Move2D(delta image.Point, parBBox image.Rectangle) {
	g.SliderBase.Move2D(delta, parBBox)
	g.OrigWinBBox = g.WinBBox
}

func (g *Slider) Render2D() {
	if g.FullReRenderIfNeeded() {
		return
	}
	if g.PushBounds() {
		g.SliderEvents()
		g.Render2DDefaultStyle()
		g.Render2DChildren()
		g.PopBounds()
	} else {
		g.DisconnectAllEvents(RegPri)
	}
}

// render using a default style if not otherwise styled
func (g *Slider) Render2DDefaultStyle() {
	st := &g.Sty
	rs := &g.Viewport.Render
	pc := &rs.Paint

	g.ConfigPartsIfNeeded(true)

	// overall fill box
	g.RenderStdBox(&g.StateStyles[SliderBox])

	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	pc.FillStyle.SetColorSpec(&st.Font.BgColor)

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

	ht := 0.5 * g.ThSize

	odim := OtherDim(g.Dim)
	bpos.SetAddDim(odim, spc)
	bsz.SetSubDim(odim, 2.0*spc)
	bpos.SetAddDim(g.Dim, spc+ht)
	bsz.SetSubDim(g.Dim, 2.0*(spc+ht))
	g.RenderBoxImpl(bpos, bsz, st.Border.Radius.Dots)

	bsz.SetDim(g.Dim, g.Pos)
	pc.FillStyle.SetColorSpec(&g.StateStyles[SliderValue].Font.BgColor)
	g.RenderBoxImpl(bpos, bsz, st.Border.Radius.Dots)

	tpos.SetDim(g.Dim, bpos.Dim(g.Dim)+g.Pos)
	tpos.SetAddDim(odim, 0.5*sz.Dim(odim)) // ctr
	pc.FillStyle.SetColorSpec(&st.Font.BgColor)

	if g.Icon.IsValid() && g.Parts.HasChildren() {
		g.Parts.Render2DTree()
	} else {
		pc.DrawCircle(rs, tpos.X, tpos.Y, ht)
		pc.FillStrokeClear(rs)
	}
}

func (g *Slider) FocusChanged2D(gotFocus bool) {
	if gotFocus {
		g.EmitFocusedSignal()
		g.SetSliderState(SliderFocus)
	} else {
		g.SetSliderState(SliderActive) // lose any hover state but whatever..
	}
	g.UpdateSig()
}

////////////////////////////////////////////////////////////////////////////////////////
//  ScrollBar

// ScrollBar has a proportional thumb size reflecting amount of content visible
type ScrollBar struct {
	SliderBase
}

var KiT_ScrollBar = kit.Types.AddType(&ScrollBar{}, ScrollBarProps)

var ScrollBarProps = ki.Props{
	"border-width":     units.NewValue(1, units.Px),
	"border-radius":    units.NewValue(4, units.Px),
	"border-color":     &Prefs.Colors.Border,
	"border-style":     BorderSolid,
	"padding":          units.NewValue(0, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"background-color": &Prefs.Colors.Control,
	"color":            &Prefs.Colors.Font,
	SliderSelectors[SliderActive]: ki.Props{
		"background-color": "lighter-0",
	},
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
	SliderSelectors[SliderDown]: ki.Props{
		"background-color": "highlight-20",
	},
	SliderSelectors[SliderValue]: ki.Props{
		"border-color":     &Prefs.Colors.Icon,
		"background-color": &Prefs.Colors.Icon,
	},
	SliderSelectors[SliderBox]: ki.Props{
		"border-color":     &Prefs.Colors.Background,
		"background-color": &Prefs.Colors.Background,
	},
}

func (g *ScrollBar) Defaults() { // todo: should just get these from props
	g.ValThumb = true
	g.ThumbSize = units.NewValue(1, units.Ex)
	g.Step = 0.1
	g.PageStep = 0.2
	g.Max = 1.0
	g.Prec = 9
}

func (g *ScrollBar) Init2D() {
	g.Init2DSlider()
}

func (g *ScrollBar) Style2D() {
	g.SetCanFocusIfActive()
	g.Style2DWidget()
	pst := &(g.Par.(Node2D).AsWidget().Sty)
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i].CopyFrom(&g.Sty)
		g.StateStyles[i].SetStyleProps(pst, g.StyleProps(SliderSelectors[i]))
		g.StateStyles[i].CopyUnitContext(&g.Sty.UnContext)
	}
	SliderFields.Style(g, nil, g.Props)
	SliderFields.ToDots(g, &g.Sty.UnContext)
}

func (g *ScrollBar) Size2D() {
	g.InitLayout2D()
}

func (g *ScrollBar) Layout2D(parBBox image.Rectangle, iter int) bool {
	g.Layout2DBase(parBBox, true, iter) // init style
	g.Layout2DParts(parBBox, iter)
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Sty.UnContext)
	}
	g.SizeFromAlloc()
	g.OrigWinBBox = g.WinBBox
	return g.Layout2DChildren(iter)
}

func (g *ScrollBar) Move2D(delta image.Point, parBBox image.Rectangle) {
	g.SliderBase.Move2D(delta, parBBox)
	g.OrigWinBBox = g.WinBBox
}

func (g *ScrollBar) Render2D() {
	if g.FullReRenderIfNeeded() {
		return
	}
	if g.PushBounds() {
		g.SliderEvents()
		g.Render2DDefaultStyle()
		g.Render2DChildren()
		g.PopBounds()
	} else {
		g.DisconnectAllEvents(RegPri)
	}
}

// render using a default style if not otherwise styled
func (g *ScrollBar) Render2DDefaultStyle() {
	st := &g.Sty
	rs := &g.Viewport.Render
	pc := &rs.Paint

	// overall fill box
	g.RenderStdBox(&g.StateStyles[SliderBox])

	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	pc.FillStyle.SetColorSpec(&st.Font.BgColor)

	// scrollbar is basic box in content size
	spc := st.BoxSpace()
	pos := g.LayData.AllocPos.AddVal(spc)
	sz := g.LayData.AllocSize.SubVal(2.0 * spc)

	g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots) // surround box
	pos.SetAddDim(g.Dim, g.Pos)                     // start of thumb
	sz.SetDim(g.Dim, g.ThSize)
	pc.FillStyle.SetColorSpec(&g.StateStyles[SliderValue].Font.BgColor)
	g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
}

func (g *ScrollBar) FocusChanged2D(gotFocus bool) {
	if gotFocus {
		g.EmitFocusedSignal()
		g.SetSliderState(SliderFocus)
	} else {
		g.SetSliderState(SliderActive) // lose any hover state but whatever..
	}
	g.UpdateSig()
}
