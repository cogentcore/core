// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"github.com/chewxy/math32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// SliderMinThumbSize is the minimum thumb size, even if computed value would make it smaller
var SliderMinThumbSize = float32(8)

// SliderBase has common slider functionality -- two major modes: ValThumb =
// false is a slider with a fixed-size thumb knob, while = true has a thumb
// that represents a value, as in a scrollbar, and the scrolling range is size
// - thumbsize
type SliderBase struct {
	PartsWidgetBase
	Value       float32              `xml:"value" desc:"current value"`
	EmitValue   float32              `copy:"-" xml:"-" json:"-" desc:"previous emitted value - don't re-emit if it is the same"`
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
	Tracking    bool                 `xml:"tracking" desc:"if true, will send continuous updates of value changes as user moves the slider -- otherwise only at the end -- see TrackThr for a threshold on amount of change"`
	TrackThr    float32              `xml:"track-thr" desc:"threshold for amount of change in scroll value before emitting a signal in Tracking mode"`
	Snap        bool                 `xml:"snap" desc:"snap the values to Step size increments"`
	Off         bool                 `desc:"can turn off e.g., scrollbar rendering with this flag -- just prevents rendering"`
	State       SliderStates         `json:"-" xml:"-" desc:"state of slider"`
	StateStyles [SliderStatesN]Style `copy:"-" json:"-" xml:"-" desc:"styles for different states of the slider, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	SliderSig   ki.Signal            `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for slider -- see SliderSignals for the types"`
	// todo: icon -- should be an xml
	OrigWinBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"copy of the win bbox, used for translating mouse events for cases like splitter where the bbox is restricted to the slider itself"`
}

var KiT_SliderBase = kit.Types.AddType(&SliderBase{}, SliderBaseProps)

var SliderBaseProps = ki.Props{
	"base-type": true,
}

func (nb *SliderBase) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*SliderBase)
	nb.PartsWidgetBase.CopyFieldsFrom(&fr.PartsWidgetBase)
	nb.Value = fr.Value
	nb.Min = fr.Min
	nb.Max = fr.Max
	nb.Step = fr.Step
	nb.PageStep = fr.PageStep
	nb.Size = fr.Size
	nb.ThSize = fr.ThSize
	nb.ThumbSize = fr.ThumbSize
	nb.Prec = fr.Prec
	nb.Icon = fr.Icon
	nb.ValThumb = fr.ValThumb
	nb.Pos = fr.Pos
	nb.DragPos = fr.DragPos
	nb.VisPos = fr.VisPos
	nb.Tracking = fr.Tracking
	nb.TrackThr = fr.TrackThr
	nb.Snap = fr.Snap
	nb.Off = fr.Off
}

func (sb *SliderBase) Disconnect() {
	sb.WidgetBase.Disconnect()
	sb.SliderSig.DisconnectAll()
}

// SliderSignals are signals that sliders can send
type SliderSignals int64

const (
	// SliderValueChanged indicates that the value has changed -- if tracking
	// is enabled, then this tracks online changes -- otherwise only at the
	// end.
	SliderValueChanged SliderSignals = iota

	// SliderPressed means slider was pushed down but not yet up.
	SliderPressed

	// SliderReleased means the slider has been released after being pressed.
	SliderReleased

	// SliderMoved means the slider position has moved (low level move event).
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

func (sb *SliderBase) Defaults() { // todo: should just get these from props
	sb.ThumbSize = units.NewEm(1)
	sb.Step = 0.1
	sb.PageStep = 0.2
	sb.Max = 1.0
	sb.Prec = 9
}

// SnapValue snaps the value to step sizes if snap option is set
func (sb *SliderBase) SnapValue() {
	if sb.Snap {
		sb.Value = FloatMod32(sb.Value, sb.Step)
		sb.Value = Truncate32(sb.Value, sb.Prec)
	}
}

// SetSliderState sets the slider state to given state, updates style
func (sb *SliderBase) SetSliderState(state SliderStates) {
	if state == SliderActive && sb.HasFocus() {
		state = SliderFocus
	}
	sb.State = state
	sb.Sty = sb.StateStyles[state] // get relevant styles
}

// SliderPressed sets the slider in the down state -- mouse clicked down but
// not yet up -- emits SliderPressed signal
func (sb *SliderBase) SliderPressed(pos float32) {
	sb.EmitValue = sb.Min - 1.0 // invalid value
	updt := sb.UpdateStart()
	sb.SetSliderState(SliderDown)
	sb.SetSliderPos(pos)
	sb.SliderSig.Emit(sb.This(), int64(SliderPressed), sb.Value)
	// bitflasb.Set(&sb.Flag, int(SliderFlagDragging))
	sb.UpdateEnd(updt)
}

// SliderReleased called when the slider has just been released -- sends a
// released signal and returns state to normal, and emits clicked signal if if
// it was previously in pressed state
func (sb *SliderBase) SliderReleased() {
	wasPressed := (sb.State == SliderDown)
	updt := sb.UpdateStart()
	sb.SetSliderState(SliderActive)
	sb.SliderSig.Emit(sb.This(), int64(SliderReleased), sb.Value)
	if wasPressed && sb.Value != sb.EmitValue {
		sb.SliderSig.Emit(sb.This(), int64(SliderValueChanged), sb.Value)
		sb.EmitValue = sb.Value
	}
	sb.UpdateEnd(updt)
}

// SliderEnterHover slider starting hover
func (sb *SliderBase) SliderEnterHover() {
	if sb.State != SliderHover {
		updt := sb.UpdateStart()
		sb.SetSliderState(SliderHover)
		sb.UpdateEnd(updt)
	}
}

// SliderExitHover called when slider exiting hover
func (sb *SliderBase) SliderExitHover() {
	if sb.State == SliderHover {
		updt := sb.UpdateStart()
		sb.SetSliderState(SliderActive)
		sb.UpdateEnd(updt)
	}
}

// SizeFromAlloc gets size from allocation
func (sb *SliderBase) SizeFromAlloc() {
	if sb.LayData.AllocSize.IsZero() {
		return
	}
	if sb.Min == 0 && sb.Max == 0 { // uninit
		sb.Defaults()
	}
	spc := sb.Sty.BoxSpace()
	sb.Size = sb.LayData.AllocSize.Dim(sb.Dim) - 2.0*spc
	if sb.Size <= 0 {
		return
	}
	if !sb.ValThumb {
		sb.Size -= sb.ThSize // half on each side
	}
	sb.UpdatePosFromValue()
	sb.DragPos = sb.Pos
}

// SetSliderPos sets the position of the slider at the given position in
// pixels -- updates the corresponding Value
func (sb *SliderBase) SetSliderPos(pos float32) {
	updt := sb.UpdateStart()
	sb.Pos = pos
	sb.Pos = Min32(sb.Size, sb.Pos)
	if sb.ValThumb {
		sb.UpdateThumbValSize()
		sb.Pos = Min32(sb.Size-sb.ThSize, sb.Pos)
	}
	sb.Pos = Max32(0, sb.Pos)
	sb.Value = Truncate32(sb.Min+(sb.Max-sb.Min)*(sb.Pos/sb.Size), sb.Prec)
	sb.DragPos = sb.Pos
	if sb.Snap {
		sb.SnapValue()
		sb.UpdatePosFromValue()
	}
	if sb.Tracking && sb.Value != sb.EmitValue {
		if math32.Abs(sb.Value-sb.EmitValue) > sb.TrackThr {
			sb.SliderSig.Emit(sb.This(), int64(SliderValueChanged), sb.Value)
			sb.EmitValue = sb.Value
		}
	}
	sb.UpdateEnd(updt)
}

// SliderMoved called when slider moved along relevant axis
func (sb *SliderBase) SliderMoved(start, end float32) {
	del := end - start
	sb.SetSliderPos(sb.DragPos + del)
}

// UpdatePosFromValue updates the slider position based on the current Value
func (sb *SliderBase) UpdatePosFromValue() {
	if sb.Size == 0.0 {
		return
	}
	if sb.ValThumb {
		sb.UpdateThumbValSize()
	}
	sb.Pos = sb.Size * (sb.Value - sb.Min) / (sb.Max - sb.Min)
}

// SetValue sets the value and updates the slider position, but does not
// emit an updated signal (see SetValueAction)
func (sb *SliderBase) SetValue(val float32) {
	updt := sb.UpdateStart()
	val = Min32(val, sb.Max)
	if sb.ValThumb {
		val = Min32(val, sb.Max-sb.ThumbVal)
	}
	val = Max32(val, sb.Min)
	if sb.Value != val {
		sb.Value = val
		sb.UpdatePosFromValue()
		sb.DragPos = sb.Pos
	}
	sb.UpdateEnd(updt)
}

// SetValueAction sets the value and updates the slider representation, and
// emits a changed signal
func (sb *SliderBase) SetValueAction(val float32) {
	if sb.Value == val {
		return
	}
	sb.SetValue(val)
	sb.SliderSig.Emit(sb.This(), int64(SliderValueChanged), sb.Value)
}

// SetThumbValue sets the thumb value to given value and updates the thumb size
// -- for scrollbar-style sliders where the thumb size represents visible range
func (sb *SliderBase) SetThumbValue(val float32) {
	updt := sb.UpdateStart()
	sb.ThumbVal = Min32(val, sb.Max)
	sb.ThumbVal = Max32(sb.ThumbVal, sb.Min)
	sb.UpdateThumbValSize()
	sb.UpdateEnd(updt)
}

// UpdateThumbValSize sets thumb size as proportion of min / max (e.sb., amount
// visible in scrollbar) -- max's out to full size
func (sb *SliderBase) UpdateThumbValSize() {
	sb.ThSize = ((sb.ThumbVal - sb.Min) / (sb.Max - sb.Min))
	sb.ThSize = Min32(sb.ThSize, 1.0)
	sb.ThSize = Max32(sb.ThSize, 0.0)
	sb.ThSize *= sb.Size
	sb.ThSize = Max32(sb.ThSize, SliderMinThumbSize)
}

func (sb *SliderBase) KeyInput(kt *key.ChordEvent) {
	if KeyEventTrace {
		fmt.Printf("SliderBase KeyInput: %v\n", sb.PathUnique())
	}
	kf := KeyFun(kt.Chord())
	switch kf {
	case KeyFunMoveUp:
		sb.SetValueAction(sb.Value - sb.Step)
		kt.SetProcessed()
	case KeyFunMoveLeft:
		sb.SetValueAction(sb.Value - sb.Step)
		kt.SetProcessed()
	case KeyFunMoveDown:
		sb.SetValueAction(sb.Value + sb.Step)
		kt.SetProcessed()
	case KeyFunMoveRight:
		sb.SetValueAction(sb.Value + sb.Step)
		kt.SetProcessed()
	case KeyFunPageUp:
		sb.SetValueAction(sb.Value - sb.PageStep)
		kt.SetProcessed()
	// case KeyFunPageLeft:
	// 	sb.SetValueAction(sb.Value - sb.PageStep)
	// 	kt.SetProcessed()
	case KeyFunPageDown:
		sb.SetValueAction(sb.Value + sb.PageStep)
		kt.SetProcessed()
	// case KeyFunPageRight:
	// 	sb.SetValueAction(sb.Value + sb.PageStep)
	// 	kt.SetProcessed()
	case KeyFunHome:
		sb.SetValueAction(sb.Min)
		kt.SetProcessed()
	case KeyFunEnd:
		sb.SetValueAction(sb.Max)
		kt.SetProcessed()
	}
}

// PointToRelPos translates a point in global pixel coords into relative
// position within node
func (sb *SliderBase) PointToRelPos(pt image.Point) image.Point {
	return pt.Sub(sb.OrigWinBBox.Min)
}

func (sb *SliderBase) MouseDragEvent() {
	sb.ConnectEvent(oswin.MouseDragEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		sbb := recv.Embed(KiT_SliderBase).(*SliderBase)
		if sbb.IsInactive() {
			return
		}
		me.SetProcessed()
		st := sbb.PointToRelPos(me.From)
		ed := sbb.PointToRelPos(me.Where)
		if sbb.Dim == X {
			sbb.SliderMoved(float32(st.X), float32(ed.X))
		} else {
			sbb.SliderMoved(float32(st.Y), float32(ed.Y))
		}
	})
}

func (sb *SliderBase) MouseEvent() {
	sb.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		sbb := recv.Embed(KiT_SliderBase).(*SliderBase)
		if sbb.IsInactive() {
			me.SetProcessed()
			sbb.SetSelectedState(!sbb.IsSelected())
			sbb.EmitSelectedSignal()
			sbb.UpdateSig()
		} else {
			if me.Button == mouse.Left {
				me.SetProcessed()
				if me.Action == mouse.Press {
					ed := sbb.PointToRelPos(me.Where)
					st := &sbb.Sty
					spc := st.Layout.Margin.Dots + 0.5*sbb.ThSize
					if sbb.Dim == X {
						sbb.SliderPressed(float32(ed.X) - spc)
					} else {
						sbb.SliderPressed(float32(ed.Y) - spc)
					}
				} else {
					sbb.SliderReleased()
				}
			}
		}
	})
}

func (sb *SliderBase) MouseFocusEvent() {
	sb.ConnectEvent(oswin.MouseFocusEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		sbb := recv.Embed(KiT_SliderBase).(*SliderBase)
		if sbb.IsInactive() {
			return
		}
		me := d.(*mouse.FocusEvent)
		me.SetProcessed()
		if me.Action == mouse.Enter {
			sbb.SliderEnterHover()
		} else {
			sbb.SliderExitHover()
		}
	})
}

func (sb *SliderBase) MouseScrollEvent() {
	sb.ConnectEvent(oswin.MouseScrollEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		sbb := recv.Embed(KiT_SliderBase).(*SliderBase)
		if sbb.IsInactive() {
			return
		}
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		cur := float32(sbb.Pos)
		if sbb.Dim == X {
			sbb.SliderMoved(cur, cur+float32(me.NonZeroDelta(true))) // preferX
		} else {
			sbb.SliderMoved(cur, cur-float32(me.NonZeroDelta(false))) // preferY
		}
	})
}

func (sb *SliderBase) KeyChordEvent() {
	sb.ConnectEvent(oswin.KeyChordEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		sbb := recv.Embed(KiT_SliderBase).(*SliderBase)
		if sbb.IsInactive() {
			return
		}
		sbb.KeyInput(d.(*key.ChordEvent))
	})
}

func (sb *SliderBase) SliderEvents() {
	sb.MouseDragEvent()
	sb.MouseEvent()
	sb.MouseFocusEvent()
	sb.MouseScrollEvent()
	sb.KeyChordEvent()
}

func (sb *SliderBase) Init2DSlider() {
	sb.Init2DWidget()
}

func (sb *SliderBase) ConfigParts() {
	sb.Parts.Lay = LayoutNil
	config := kit.TypeAndNameList{}
	icIdx, lbIdx := sb.ConfigPartsIconLabel(&config, string(sb.Icon), "")
	mods, updt := sb.Parts.ConfigChildren(config, false)
	sb.ConfigPartsSetIconLabel(string(sb.Icon), "", icIdx, lbIdx)
	if mods {
		sb.UpdateEnd(updt)
	}
}

func (sb *SliderBase) ConfigPartsIfNeeded(render bool) {
	if sb.PartsNeedUpdateIconLabel(string(sb.Icon), "") {
		sb.ConfigParts()
	}
	if sb.Icon.IsValid() && sb.Parts.HasChildren() {
		ick := sb.Parts.ChildByType(KiT_Icon, true, 0)
		if ick != nil {
			ic := ick.(*Icon)
			mrg := sb.Sty.Layout.Margin.Dots
			pad := sb.Sty.Layout.Padding.Dots
			spc := mrg + pad
			odim := OtherDim(sb.Dim)
			ic.LayData.AllocPosRel.SetDim(sb.Dim, sb.Pos+spc-0.5*sb.ThSize)
			ic.LayData.AllocPosRel.SetDim(odim, -pad)
			ic.LayData.AllocSize.X = sb.ThSize
			ic.LayData.AllocSize.Y = sb.ThSize
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

// AddNewSlider adds a new slider to given parent node, with given name.
func AddNewSlider(parent ki.Ki, name string) *Slider {
	return parent.AddNewChild(KiT_Slider, name).(*Slider)
}

func (nb *Slider) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Slider)
	nb.SliderBase.CopyFieldsFrom(&fr.SliderBase)
}

var SliderProps = ki.Props{
	"border-width":     units.NewPx(1),
	"border-radius":    units.NewPx(4),
	"border-color":     &Prefs.Colors.Border,
	"padding":          units.NewPx(6),
	"margin":           units.NewPx(4),
	"background-color": &Prefs.Colors.Control,
	"color":            &Prefs.Colors.Font,
	"#icon": ki.Props{
		"width":   units.NewEm(1),
		"height":  units.NewEm(1),
		"margin":  units.NewPx(0),
		"padding": units.NewPx(0),
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
		"border-width":     units.NewPx(2),
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

func (sr *Slider) Defaults() {
	sr.ThumbSize = units.NewEm(1.5)
	sr.ThSize = 25.0
	sr.Step = 0.1
	sr.PageStep = 0.2
	sr.Max = 1.0
	sr.Prec = 9
}

func (sr *Slider) Init2D() {
	sr.Init2DSlider()
	sr.ConfigParts()
}

func (sr *Slider) StyleSlider() {
	sr.Style2DWidget()
	pst := &(sr.Par.(Node2D).AsWidget().Sty)
	for i := 0; i < int(SliderStatesN); i++ {
		sr.StateStyles[i].CopyFrom(&sr.Sty)
		sr.StateStyles[i].SetStyleProps(pst, sr.StyleProps(SliderSelectors[i]), sr.Viewport)
		sr.StateStyles[i].CopyUnitContext(&sr.Sty.UnContext)
	}
	SliderFields.Style(sr, nil, sr.Props, sr.Viewport)
	SliderFields.ToDots(sr, &sr.Sty.UnContext)
	sr.ThSize = sr.ThumbSize.Dots
}

func (sr *Slider) Style2D() {
	sr.SetCanFocusIfActive()
	sr.StyleSlider()
	sr.LayData.SetFromStyle(&sr.Sty.Layout) // also does reset
	sr.ConfigParts()
}

func (sr *Slider) Size2D(iter int) {
	sr.InitLayout2D()
	if sr.ThSize == 0.0 {
		sr.Defaults()
	}
	st := &sr.Sty
	// get at least thumbsize + margin + border.size
	sz := sr.ThSize + 2.0*(st.Layout.Margin.Dots+st.Border.Width.Dots)
	sr.LayData.AllocSize.SetDim(OtherDim(sr.Dim), sz)
}

func (sr *Slider) Layout2D(parBBox image.Rectangle, iter int) bool {
	sr.ConfigPartsIfNeeded(false)
	sr.Layout2DBase(parBBox, true, iter) // init style
	sr.Layout2DParts(parBBox, iter)
	for i := 0; i < int(SliderStatesN); i++ {
		sr.StateStyles[i].CopyUnitContext(&sr.Sty.UnContext)
	}
	sr.SizeFromAlloc()
	sr.OrigWinBBox = sr.WinBBox
	return sr.Layout2DChildren(iter)
}

func (sr *Slider) Move2D(delta image.Point, parBBox image.Rectangle) {
	sr.SliderBase.Move2D(delta, parBBox)
	sr.OrigWinBBox = sr.WinBBox
}

func (sr *Slider) Render2D() {
	if sr.FullReRenderIfNeeded() {
		return
	}
	if !sr.Off && sr.PushBounds() {
		sr.This().(Node2D).ConnectEvents2D()
		sr.Render2DDefaultStyle()
		sr.Render2DChildren()
		sr.PopBounds()
	} else {
		sr.DisconnectAllEvents(RegPri)
	}
}

// render using a default style if not otherwise styled
func (sr *Slider) Render2DDefaultStyle() {
	st := &sr.Sty
	rs := &sr.Viewport.Render
	rs.Lock()
	pc := &rs.Paint

	sr.ConfigPartsIfNeeded(true)

	// overall fill box
	sr.RenderStdBox(&sr.StateStyles[SliderBox])

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
	pos := sr.LayData.AllocPos
	sz := sr.LayData.AllocSize
	bpos := pos // box pos
	bsz := sz
	tpos := pos // thumb pos

	ht := 0.5 * sr.ThSize

	odim := OtherDim(sr.Dim)
	bpos.SetAddDim(odim, spc)
	bsz.SetSubDim(odim, 2.0*spc)
	bpos.SetAddDim(sr.Dim, spc+ht)
	bsz.SetSubDim(sr.Dim, 2.0*(spc+ht))
	sr.RenderBoxImpl(bpos, bsz, st.Border.Radius.Dots)

	bsz.SetDim(sr.Dim, sr.Pos)
	pc.FillStyle.SetColorSpec(&sr.StateStyles[SliderValue].Font.BgColor)
	sr.RenderBoxImpl(bpos, bsz, st.Border.Radius.Dots)

	tpos.SetDim(sr.Dim, bpos.Dim(sr.Dim)+sr.Pos)
	tpos.SetAddDim(odim, 0.5*sz.Dim(odim)) // ctr
	pc.FillStyle.SetColorSpec(&st.Font.BgColor)

	if sr.Icon.IsValid() && sr.Parts.HasChildren() {
		rs.Unlock()
		sr.Parts.Render2DTree()
	} else {
		pc.DrawCircle(rs, tpos.X, tpos.Y, ht)
		pc.FillStrokeClear(rs)
		rs.Unlock()
	}
}

func (sr *Slider) ConnectEvents2D() {
	sr.SliderEvents()
}

func (sr *Slider) FocusChanged2D(change FocusChanges) {
	switch change {
	case FocusLost:
		sr.SetSliderState(SliderActive) // lose any hover state but whatever..
		sr.UpdateSig()
	case FocusGot:
		sr.ScrollToMe()
		sr.SetSliderState(SliderFocus)
		sr.EmitFocusedSignal()
		sr.UpdateSig()
	case FocusInactive: // don't care..
	case FocusActive:
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  ScrollBar

// ScrollBar has a proportional thumb size reflecting amount of content visible
type ScrollBar struct {
	SliderBase
}

var KiT_ScrollBar = kit.Types.AddType(&ScrollBar{}, ScrollBarProps)

// AddNewScrollBar adds a new scrollbar to given parent node, with given name.
func AddNewScrollBar(parent ki.Ki, name string) *ScrollBar {
	return parent.AddNewChild(KiT_ScrollBar, name).(*ScrollBar)
}

func (nb *ScrollBar) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*ScrollBar)
	nb.SliderBase.CopyFieldsFrom(&fr.SliderBase)
}

var ScrollBarProps = ki.Props{
	"border-width":     units.NewPx(1),
	"border-radius":    units.NewPx(4),
	"border-color":     &Prefs.Colors.Border,
	"padding":          units.NewPx(0),
	"margin":           units.NewPx(2),
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
		"border-width":     units.NewPx(2),
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

func (sb *ScrollBar) Defaults() { // todo: should just get these from props
	sb.ValThumb = true
	sb.ThumbSize = units.NewEx(1)
	sb.Step = 0.1
	sb.PageStep = 0.2
	sb.Max = 1.0
	sb.Prec = 9
}

func (sb *ScrollBar) Init2D() {
	sb.Init2DSlider()
}

func (sb *ScrollBar) StyleScrollBar() {
	sb.Style2DWidget()
	pst := &(sb.Par.(Node2D).AsWidget().Sty)
	for i := 0; i < int(SliderStatesN); i++ {
		sb.StateStyles[i].CopyFrom(&sb.Sty)
		sb.StateStyles[i].SetStyleProps(pst, sb.StyleProps(SliderSelectors[i]), sb.Viewport)
		sb.StateStyles[i].CopyUnitContext(&sb.Sty.UnContext)
	}
	SliderFields.Style(sb, nil, sb.Props, sb.Viewport)
	SliderFields.ToDots(sb, &sb.Sty.UnContext)
}

func (sb *ScrollBar) Style2D() {
	sb.SetCanFocusIfActive()
	sb.StyleScrollBar()
	sb.LayData.SetFromStyle(&sb.Sty.Layout) // also does reset
}

func (sb *ScrollBar) Size2D(iter int) {
	sb.InitLayout2D()
}

func (sb *ScrollBar) Layout2D(parBBox image.Rectangle, iter int) bool {
	sb.Layout2DBase(parBBox, true, iter) // init style
	sb.Layout2DParts(parBBox, iter)
	for i := 0; i < int(SliderStatesN); i++ {
		sb.StateStyles[i].CopyUnitContext(&sb.Sty.UnContext)
	}
	sb.SizeFromAlloc()
	sb.OrigWinBBox = sb.WinBBox
	return sb.Layout2DChildren(iter)
}

func (sb *ScrollBar) Move2D(delta image.Point, parBBox image.Rectangle) {
	sb.SliderBase.Move2D(delta, parBBox)
	sb.OrigWinBBox = sb.WinBBox
}

func (sb *ScrollBar) Render2D() {
	if sb.FullReRenderIfNeeded() {
		return
	}
	if !sb.Off && sb.PushBounds() {
		sb.This().(Node2D).ConnectEvents2D()
		sb.Render2DDefaultStyle()
		sb.Render2DChildren()
		sb.PopBounds()
	} else {
		sb.DisconnectAllEvents(RegPri)
	}
}

// render using a default style if not otherwise styled
func (sb *ScrollBar) Render2DDefaultStyle() {
	st := &sb.Sty
	rs := &sb.Viewport.Render
	rs.Lock()
	pc := &rs.Paint

	// overall fill box
	sb.RenderStdBox(&sb.StateStyles[SliderBox])

	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	pc.FillStyle.SetColorSpec(&st.Font.BgColor)

	// scrollbar is basic box in content size
	spc := st.BoxSpace()
	pos := sb.LayData.AllocPos.AddVal(spc)
	sz := sb.LayData.AllocSize.SubVal(2.0 * spc)

	sb.RenderBoxImpl(pos, sz, st.Border.Radius.Dots) // surround box
	pos.SetAddDim(sb.Dim, sb.Pos)                    // start of thumb
	sz.SetDim(sb.Dim, sb.ThSize)
	pc.FillStyle.SetColorSpec(&sb.StateStyles[SliderValue].Font.BgColor)
	sb.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
	rs.Unlock()
}

func (sb *ScrollBar) ConnectEvents2D() {
	sb.SliderEvents()
}

func (sb *ScrollBar) FocusChanged2D(change FocusChanges) {
	switch change {
	case FocusLost:
		sb.SetSliderState(SliderActive) // lose any hover state but whatever..
		sb.UpdateSig()
	case FocusGot:
		sb.SetSliderState(SliderFocus)
		sb.EmitFocusedSignal()
		sb.UpdateSig()
	case FocusInactive: // don't care..
	case FocusActive:
	}
}
