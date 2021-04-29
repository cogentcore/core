// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"sync"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// SliderMinThumbSize is the minimum thumb size, even if computed value would make it smaller
var SliderMinThumbSize = float32(8)

// SliderPositioner is a minor interface for functions related to
// computing slider positions.  Needed for more complex sliders
// such as Splitters that do this computation in a different way.
type SliderPositioner interface {
	// PointToRelPos translates a point in global pixel coords into relative
	// position within node
	PointToRelPos(pt image.Point) image.Point
}

// SliderBase has common slider functionality -- two major modes: ValThumb =
// false is a slider with a fixed-size thumb knob, while = true has a thumb
// that represents a value, as in a scrollbar, and the scrolling range is size
// - thumbsize
type SliderBase struct {
	PartsWidgetBase
	Value       float32                   `xml:"value" desc:"current value"`
	EmitValue   float32                   `copy:"-" xml:"-" json:"-" desc:"previous emitted value - don't re-emit if it is the same"`
	Min         float32                   `xml:"min" desc:"minimum value in range"`
	Max         float32                   `xml:"max" desc:"maximum value in range"`
	Step        float32                   `xml:"step" desc:"smallest step size to increment"`
	PageStep    float32                   `xml:"pagestep" desc:"larger PageUp / Dn step size"`
	Size        float32                   `xml:"size" desc:"size of the slide box in the relevant dimension -- range of motion -- exclusive of spacing"`
	ThSize      float32                   `xml:"-" desc:"computed size of the thumb -- if ValThumb then this is auto-sized based on ThumbVal and is subtracted from Size in computing Value -- this is the display size version subject to SliderMinThumbSize"`
	ThSizeReal  float32                   `xml:"-" desc:"computed size of the thumb, without any SliderMinThumbSize limitation -- use this for more accurate calculations of true value"`
	ThumbSize   units.Value               `xml:"thumb-size" desc:"styled fixed size of the thumb"`
	Prec        int                       `xml:"prec" desc:"specifies the precision of decimal places (total, not after the decimal point) to use in representing the number -- this helps to truncate small weird floating point values in the nether regions"`
	Icon        IconName                  `view:"show-name" desc:"optional icon for the dragging knob"`
	ValThumb    bool                      `xml:"val-thumb" alt:"prop-thumb" desc:"if true, has a proportionally-sized thumb knob reflecting another value -- e.g., the amount visible in a scrollbar, and thumb is completely inside Size -- otherwise ThumbSize affects Size so that full Size range can be traversed"`
	ThumbVal    float32                   `xml:"thumb-val" desc:"value that the thumb represents, in the same units"`
	Pos         float32                   `xml:"-" desc:"logical position of the slider relative to Size"`
	DragPos     float32                   `xml:"-" desc:"underlying drag position of slider -- not subject to snapping"`
	Dim         mat32.Dims                `desc:"dimension along which the slider slides"`
	Tracking    bool                      `xml:"tracking" desc:"if true, will send continuous updates of value changes as user moves the slider -- otherwise only at the end -- see TrackThr for a threshold on amount of change"`
	TrackThr    float32                   `xml:"track-thr" desc:"threshold for amount of change in scroll value before emitting a signal in Tracking mode"`
	Snap        bool                      `xml:"snap" desc:"snap the values to Step size increments"`
	Off         bool                      `desc:"can turn off e.g., scrollbar rendering with this flag -- just prevents rendering"`
	State       SliderStates              `json:"-" xml:"-" desc:"state of slider"`
	StateStyles [SliderStatesN]gist.Style `copy:"-" json:"-" xml:"-" desc:"styles for different states of the slider, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	SliderSig   ki.Signal                 `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for slider -- see SliderSignals for the types"`
}

var KiT_SliderBase = kit.Types.AddType(&SliderBase{}, SliderBaseProps)

var SliderBaseProps = ki.Props{
	"base-type":     true,
	"EnumType:Flag": KiT_NodeFlags,
}

func (sb *SliderBase) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*SliderBase)
	sb.PartsWidgetBase.CopyFieldsFrom(&fr.PartsWidgetBase)
	sb.Value = fr.Value
	sb.Min = fr.Min
	sb.Max = fr.Max
	sb.Step = fr.Step
	sb.PageStep = fr.PageStep
	sb.Size = fr.Size
	sb.ThSize = fr.ThSize
	sb.ThSizeReal = fr.ThSizeReal
	sb.ThumbSize = fr.ThumbSize
	sb.Prec = fr.Prec
	sb.Icon = fr.Icon
	sb.ValThumb = fr.ValThumb
	sb.Pos = fr.Pos
	sb.DragPos = fr.DragPos
	sb.Tracking = fr.Tracking
	sb.TrackThr = fr.TrackThr
	sb.Snap = fr.Snap
	sb.Off = fr.Off
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
	// end.  The data on the signal is the float32 Value.
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
		sb.Value = mat32.IntMultiple(sb.Value, sb.Step)
		sb.Value = mat32.Truncate(sb.Value, sb.Prec)
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

// SliderPress sets the slider in the down state -- mouse clicked down but
// not yet up -- emits SliderPress signal
func (sb *SliderBase) SliderPress(pos float32) {
	sb.EmitValue = sb.Min - 1.0 // invalid value
	updt := sb.UpdateStart()
	sb.SetSliderState(SliderDown)
	sb.SetSliderPos(pos)
	sb.SliderSig.Emit(sb.This(), int64(SliderPressed), sb.Value)
	// bitflasb.Set(&sb.Flag, int(SliderFlagDragging))
	sb.UpdateEnd(updt)
}

// SliderRelease called when the slider has just been released -- sends a
// released signal and returns state to normal, and emits clicked signal if if
// it was previously in pressed state
func (sb *SliderBase) SliderRelease() {
	wasPressed := (sb.State == SliderDown)
	updt := sb.UpdateStart()
	sb.SetSliderState(SliderActive)
	sb.SliderSig.Emit(sb.This(), int64(SliderReleased), sb.Value)
	if wasPressed {
		sb.EmitNewValue()
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
	if sb.LayState.Alloc.Size.IsNil() {
		return
	}
	if sb.Min == 0 && sb.Max == 0 { // uninit
		sb.Defaults()
	}
	spc := sb.BoxSpace()
	sb.Size = sb.LayState.Alloc.Size.Dim(sb.Dim) - 2.0*spc
	if sb.Size <= 0 {
		return
	}
	if !sb.ValThumb {
		sb.Size -= sb.ThSize // half on each side
	}
	sb.UpdatePosFromValue()
	sb.DragPos = sb.Pos
}

// EmitNewValue emits new Value, if it has not already been emitted.
// Compares Value to EmitValue and only emits if different, sets EmitValue.
// Returns true if value emitted, false otherwise.
func (sb *SliderBase) EmitNewValue() bool {
	if sb.Value == sb.EmitValue {
		return false
	}
	sb.SliderSig.Emit(sb.This(), int64(SliderValueChanged), sb.Value)
	sb.EmitValue = sb.Value
	return true
}

// SetSliderPos sets the position of the slider at the given position in pixels,
// and updates the corresponding Value based on that position.
func (sb *SliderBase) SetSliderPos(pos float32) {
	updt := sb.UpdateStart()
	sb.Pos = pos
	sb.Pos = mat32.Min(sb.Size, sb.Pos)
	effSz := sb.Size
	if sb.ValThumb {
		sb.UpdateThumbValSize()
		sb.Pos = mat32.Min(sb.Size-sb.ThSize, sb.Pos)
		if sb.ThSize != sb.ThSizeReal {
			effSz -= sb.ThSize - sb.ThSizeReal
			effSz -= .5 // rounding errors
		}
	}
	sb.Pos = mat32.Max(0, sb.Pos)
	sb.Value = mat32.Truncate(sb.Min+(sb.Max-sb.Min)*(sb.Pos/effSz), sb.Prec)
	sb.Value = mat32.Clamp(sb.Value, sb.Min, sb.Max)
	if sb.ValThumb {
		sb.Value = mat32.Min(sb.Value, sb.Max-sb.ThumbVal)
	}
	sb.DragPos = sb.Pos
	if sb.Snap {
		sb.SnapValue()
		sb.UpdatePosFromValue()
	}
	if sb.Tracking && mat32.Abs(sb.Value-sb.EmitValue) > sb.TrackThr {
		sb.EmitNewValue()
	}
	sb.UpdateEnd(updt)
}

// SliderMove called when slider moved along relevant axis
func (sb *SliderBase) SliderMove(start, end float32) {
	del := end - start
	sb.SetSliderPos(sb.DragPos + del)
	sb.SliderSig.Emit(sb.This(), int64(SliderMoved), sb.Value)
}

// UpdatePosFromValue updates the slider position based on the current Value
func (sb *SliderBase) UpdatePosFromValue() {
	if sb.Size == 0.0 {
		return
	}
	effSz := sb.Size
	if sb.ValThumb {
		sb.UpdateThumbValSize()
		if sb.ThSize != sb.ThSizeReal {
			effSz -= sb.ThSize - sb.ThSizeReal
			effSz -= 0.5 // rounding errors
		}
	}
	sb.Pos = effSz * (sb.Value - sb.Min) / (sb.Max - sb.Min)
}

// SetValue sets the value and updates the slider position, but does not
// emit an updated signal (see SetValueAction)
func (sb *SliderBase) SetValue(val float32) {
	updt := sb.UpdateStart()
	val = mat32.Min(val, sb.Max)
	if sb.ValThumb {
		val = mat32.Min(val, sb.Max-sb.ThumbVal)
	}
	val = mat32.Max(val, sb.Min)
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
	sb.EmitNewValue()
}

// SetThumbValue sets the thumb value to given value and updates the thumb size
// -- for scrollbar-style sliders where the thumb size represents visible range
func (sb *SliderBase) SetThumbValue(val float32) {
	updt := sb.UpdateStart()
	sb.ThumbVal = mat32.Min(val, sb.Max)
	sb.ThumbVal = mat32.Max(sb.ThumbVal, sb.Min)
	sb.UpdateThumbValSize()
	sb.UpdateEnd(updt)
}

// UpdateThumbValSize sets thumb size as proportion of min / max (e.sb., amount
// visible in scrollbar) -- max's out to full size
func (sb *SliderBase) UpdateThumbValSize() {
	sb.ThSizeReal = ((sb.ThumbVal - sb.Min) / (sb.Max - sb.Min))
	sb.ThSizeReal = mat32.Min(sb.ThSizeReal, 1.0)
	sb.ThSizeReal = mat32.Max(sb.ThSizeReal, 0.0)
	sb.ThSizeReal *= sb.Size
	sb.ThSize = mat32.Max(sb.ThSizeReal, SliderMinThumbSize)
}

func (sb *SliderBase) KeyInput(kt *key.ChordEvent) {
	if KeyEventTrace {
		fmt.Printf("SliderBase KeyInput: %v\n", sb.Path())
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
// position within node.  This satisfies the SliderPositioner interface.
func (sb *SliderBase) PointToRelPos(pt image.Point) image.Point {
	sb.BBoxMu.RLock()
	defer sb.BBoxMu.RUnlock()
	return pt.Sub(sb.WinBBox.Min)
}

func (sb *SliderBase) MouseDragEvent() {
	sb.ConnectEvent(oswin.MouseDragEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		sbb := recv.Embed(KiT_SliderBase).(*SliderBase)
		if sbb.IsInactive() {
			return
		}
		me.SetProcessed()
		st := sbb.This().(SliderPositioner).PointToRelPos(me.From)
		ed := sbb.This().(SliderPositioner).PointToRelPos(me.Where)
		if sbb.Dim == mat32.X {
			sbb.SliderMove(float32(st.X), float32(ed.X))
		} else {
			sbb.SliderMove(float32(st.Y), float32(ed.Y))
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
					ed := sbb.This().(SliderPositioner).PointToRelPos(me.Where)
					st := &sbb.Sty
					spc := st.Layout.Margin.Dots + 0.5*sbb.ThSizeReal
					if sbb.Dim == mat32.X {
						sbb.SliderPress(float32(ed.X) - spc)
					} else {
						sbb.SliderPress(float32(ed.Y) - spc)
					}
				} else {
					sbb.SliderRelease()
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
		if sbb.Dim == mat32.X {
			sbb.SliderMove(cur, cur+float32(me.NonZeroDelta(true))) // preferX
		} else {
			sbb.SliderMove(cur, cur-float32(me.NonZeroDelta(false))) // preferY
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
	mods, updt := sb.Parts.ConfigChildren(config)
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
		ick := sb.Parts.ChildByType(KiT_Icon, ki.Embeds, 0)
		if ick != nil {
			ic := ick.(*Icon)
			mrg := sb.Sty.Layout.Margin.Dots
			pad := sb.Sty.Layout.Padding.Dots
			spc := mrg + pad
			odim := mat32.OtherDim(sb.Dim)
			ic.LayState.Alloc.PosRel.SetDim(sb.Dim, sb.Pos+spc-0.5*sb.ThSize)
			ic.LayState.Alloc.PosRel.SetDim(odim, -pad)
			ic.LayState.Alloc.Size.X = sb.ThSize
			ic.LayState.Alloc.Size.Y = sb.ThSize
			if render {
				ic.Layout2DTree()
			}
		}
	}
}

// StyleFromProps styles Slider-specific fields from ki.Prop properties
// doesn't support inherit or default
func (sr *SliderBase) StyleFromProps(props ki.Props, vp *Viewport2D) {
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		switch key {
		case "value":
			if iv, ok := kit.ToFloat32(val); ok {
				sr.Value = iv
			}
		case "min":
			if iv, ok := kit.ToFloat32(val); ok {
				sr.Min = iv
			}
		case "max":
			if iv, ok := kit.ToFloat32(val); ok {
				sr.Max = iv
			}
		case "step":
			if iv, ok := kit.ToFloat32(val); ok {
				sr.Step = iv
			}
		case "pagestep":
			if iv, ok := kit.ToFloat32(val); ok {
				sr.PageStep = iv
			}
		case "size":
			if iv, ok := kit.ToFloat32(val); ok {
				sr.Size = iv
			}
		case "thumb-size":
			sr.ThumbSize.SetIFace(val, key)
		case "thumb-val":
			if iv, ok := kit.ToFloat32(val); ok {
				sr.ThumbVal = iv
			}
		case "track-thr":
			if iv, ok := kit.ToFloat32(val); ok {
				sr.TrackThr = iv
			}
		case "prec":
			if iv, ok := kit.ToInt(val); ok {
				sr.Prec = int(iv)
			}
		case "val-thumb":
			if bv, ok := kit.ToBool(val); ok {
				sr.ValThumb = bv
			}
		case "tracking":
			if bv, ok := kit.ToBool(val); ok {
				sr.Tracking = bv
			}
		case "snap":
			if bv, ok := kit.ToBool(val); ok {
				sr.Snap = bv
			}
		}
	}
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (sr *SliderBase) StyleToDots(uc *units.Context) {
	sr.ThumbSize.ToDots(uc)
}

func (sr *SliderBase) StyleSlider() {
	sr.StyMu.Lock()
	defer sr.StyMu.Unlock()

	sr.Style2DWidget()
	pst := &(sr.Par.(Node2D).AsWidget().Sty)
	for i := 0; i < int(SliderStatesN); i++ {
		sr.StateStyles[i].CopyFrom(&sr.Sty)
		sr.StateStyles[i].SetStyleProps(pst, sr.StyleProps(SliderSelectors[i]), sr.Viewport)
		sr.StateStyles[i].CopyUnitContext(&sr.Sty.UnContext)
	}
	sr.StyleFromProps(sr.Props, sr.Viewport)           // does all the min / max / step etc
	tprops := *kit.Types.Properties(ki.Type(sr), true) // true = makeNew
	if len(tprops) > 0 {
		kit.TypesMu.RLock()
		sr.StyleFromProps(tprops, sr.Viewport)
		kit.TypesMu.RUnlock()
	}
	sr.StyleToDots(&sr.Sty.UnContext)
	sr.ThSize = sr.ThumbSize.Dots
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

func (sr *Slider) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Slider)
	sr.SliderBase.CopyFieldsFrom(&fr.SliderBase)
}

var SliderProps = ki.Props{
	"EnumType:Flag":    KiT_NodeFlags,
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
	sr.ThSizeReal = sr.ThSize
	sr.Step = 0.1
	sr.PageStep = 0.2
	sr.Max = 1.0
	sr.Prec = 9
}

func (sr *Slider) Init2D() {
	sr.Init2DSlider()
	sr.ConfigParts()
}

func (sr *Slider) Style2D() {
	sr.SetCanFocusIfActive()
	sr.StyleSlider()
	sr.StyMu.Lock()
	sr.LayState.SetFromStyle(&sr.Sty.Layout) // also does reset
	sr.StyMu.Unlock()
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
	sr.LayState.Alloc.Size.SetDim(mat32.OtherDim(sr.Dim), sz)
}

func (sr *Slider) Layout2D(parBBox image.Rectangle, iter int) bool {
	sr.ConfigPartsIfNeeded(false)
	sr.Layout2DBase(parBBox, true, iter) // init style
	sr.Layout2DParts(parBBox, iter)
	for i := 0; i < int(SliderStatesN); i++ {
		sr.StateStyles[i].CopyUnitContext(&sr.Sty.UnContext)
	}
	sr.SizeFromAlloc()
	return sr.Layout2DChildren(iter)
}

func (sr *Slider) Move2D(delta image.Point, parBBox image.Rectangle) {
	sr.SliderBase.Move2D(delta, parBBox)
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
	rs, pc, st := sr.RenderLock()

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
	pos := sr.LayState.Alloc.Pos
	sz := sr.LayState.Alloc.Size
	bpos := pos // box pos
	bsz := sz
	tpos := pos // thumb pos

	ht := 0.5 * sr.ThSize

	odim := mat32.OtherDim(sr.Dim)
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
		sr.RenderUnlock(rs)
		sr.Parts.Render2DTree()
	} else {
		pc.DrawCircle(rs, tpos.X, tpos.Y, ht)
		pc.FillStrokeClear(rs)
		sr.RenderUnlock(rs)
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

func (sb *ScrollBar) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*ScrollBar)
	sb.SliderBase.CopyFieldsFrom(&fr.SliderBase)
}

var ScrollBarProps = ki.Props{
	"EnumType:Flag":    KiT_NodeFlags,
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

func (sb *ScrollBar) Style2D() {
	sb.SetCanFocusIfActive()
	sb.StyleSlider()
	sb.StyMu.Lock()
	sb.LayState.SetFromStyle(&sb.Sty.Layout) // also does reset
	sb.StyMu.Unlock()
	sb.ConfigParts()
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
	return sb.Layout2DChildren(iter)
}

func (sb *ScrollBar) Move2D(delta image.Point, parBBox image.Rectangle) {
	sb.SliderBase.Move2D(delta, parBBox)
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
	rs, pc, st := sb.RenderLock()
	defer sb.RenderUnlock(rs)

	// overall fill box
	sb.RenderStdBox(&sb.StateStyles[SliderBox])

	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	pc.FillStyle.SetColorSpec(&st.Font.BgColor)

	// scrollbar is basic box in content size
	spc := st.BoxSpace()
	pos := sb.LayState.Alloc.Pos.AddScalar(spc)
	sz := sb.LayState.Alloc.Size.SubScalar(2.0 * spc)

	sb.RenderBoxImpl(pos, sz, st.Border.Radius.Dots) // surround box
	pos.SetAddDim(sb.Dim, sb.Pos)                    // start of thumb
	sz.SetDim(sb.Dim, sb.ThSize)
	pc.FillStyle.SetColorSpec(&sb.StateStyles[SliderValue].Font.BgColor)
	sb.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
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

////////////////////////////////////////////////////////////////////////////////////////
//  ProgressBar

// ProgressBar is a progress bar that fills up bar as progress continues.
// Call Start with a maximum value to work toward, and ProgStep each time
// a progress step has been accomplished -- increments the ProgCur by one
// and display is updated every ProgInc such steps.
type ProgressBar struct {
	ScrollBar
	ProgMax int        `desc:"maximum amount of progress to be achieved"`
	ProgInc int        `desc:"progress increment when display is updated -- automatically computed from ProgMax at Start but can be overwritten"`
	ProgCur int        `desc:"current progress level"`
	ProgMu  sync.Mutex `desc:"mutex for updating progress"`
}

var KiT_ProgressBar = kit.Types.AddType(&ProgressBar{}, ProgressBarProps)

// AddNewProgressBar adds a new progress bar to given parent node, with given name.
func AddNewProgressBar(parent ki.Ki, name string) *ProgressBar {
	pb := parent.AddNewChild(KiT_ProgressBar, name).(*ProgressBar)
	pb.Defaults()
	return pb
}

func (pb *ProgressBar) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*ProgressBar)
	pb.SliderBase.CopyFieldsFrom(&fr.SliderBase)
}

func ProgressDefaultInc(max int) int {
	switch {
	case max > 50000:
		return 1000
	case max > 5000:
		return 100
	case max > 500:
		return 10
	}
	return 1
}

func (pb *ProgressBar) Start(max int) {
	pb.ProgMax = max - 1
	pb.ProgMax = ints.MaxInt(1, pb.ProgMax)
	pb.ProgInc = ProgressDefaultInc(max)
	pb.ProgCur = 0
	pb.UpdtBar()
}

func (pb *ProgressBar) UpdtBar() {
	updt := pb.UpdateStart()
	pb.SetThumbValue(float32(pb.ProgCur) / float32(pb.ProgMax))
	pb.UpdateEnd(updt)
}

// ProgStep is called every time there is an increment of progress.
// This is threadsafe to call from different routines.
func (pb *ProgressBar) ProgStep() {
	pb.ProgMu.Lock()
	pb.ProgCur++
	if pb.ProgCur%pb.ProgInc == 0 {
		pb.UpdtBar()
	}
	pb.ProgMu.Unlock()
}

var ProgressBarProps = ki.Props{
	"EnumType:Flag":    KiT_NodeFlags,
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

func (pb *ProgressBar) Defaults() {
	pb.Dim = mat32.X
	pb.ValThumb = true
	pb.ThumbVal = 1
	pb.Value = 0
	pb.ThumbSize = units.NewEx(1)
	pb.Step = 0.1
	pb.PageStep = 0.2
	pb.Max = 1.0
	pb.Prec = 9
	pb.SetInactive()
	pb.SetMinPrefWidth(units.NewEm(20))
	pb.SetMinPrefHeight(units.NewEm(1))
}
