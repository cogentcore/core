// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"sync"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
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

// todo: need a Slider interface with all the Set* methods
// returning Slider

// SliderBase has common slider functionality -- two major modes: ValThumb =
// false is a slider with a fixed-size thumb knob, while = true has a thumb
// that represents a value, as in a scrollbar, and the scrolling range is size
// - thumbsize
//
//goki:embedder
type SliderBase struct {
	WidgetBase

	// current value
	Value float32 `xml:"value" desc:"current value"`

	// previous emitted value - don't re-emit if it is the same
	LastValue float32 `copy:"-" xml:"-" json:"-" desc:"previous emitted value - don't re-emit if it is the same"`

	// minimum value in range
	Min float32 `xml:"min" desc:"minimum value in range"`

	// maximum value in range
	Max float32 `xml:"max" desc:"maximum value in range"`

	// smallest step size to increment
	Step float32 `xml:"step" desc:"smallest step size to increment"`

	// larger PageUp / Dn step size
	PageStep float32 `xml:"pagestep" desc:"larger PageUp / Dn step size"`

	// size of the slide box in the relevant dimension -- range of motion -- exclusive of spacing
	Size float32 `xml:"size" desc:"size of the slide box in the relevant dimension -- range of motion -- exclusive of spacing"`

	// computed size of the thumb -- if ValThumb then this is auto-sized based on ThumbVal and is subtracted from Size in computing Value -- this is the display size version subject to SliderMinThumbSize
	ThSize float32 `xml:"-" desc:"computed size of the thumb -- if ValThumb then this is auto-sized based on ThumbVal and is subtracted from Size in computing Value -- this is the display size version subject to SliderMinThumbSize"`

	// computed size of the thumb, without any SliderMinThumbSize limitation -- use this for more accurate calculations of true value
	ThSizeReal float32 `xml:"-" desc:"computed size of the thumb, without any SliderMinThumbSize limitation -- use this for more accurate calculations of true value"`

	// styled fixed size of the thumb
	ThumbSize units.Value `xml:"thumb-size" desc:"styled fixed size of the thumb"`

	// specifies the precision of decimal places (total, not after the decimal point) to use in representing the number -- this helps to truncate small weird floating point values in the nether regions
	Prec int `xml:"prec" desc:"specifies the precision of decimal places (total, not after the decimal point) to use in representing the number -- this helps to truncate small weird floating point values in the nether regions"`

	// [view: show-name] optional icon for the dragging knob
	Icon icons.Icon `view:"show-name" desc:"optional icon for the dragging knob"`

	// if true, has a proportionally-sized thumb knob reflecting another value -- e.g., the amount visible in a scrollbar, and thumb is completely inside Size -- otherwise ThumbSize affects Size so that full Size range can be traversed
	ValThumb bool `xml:"val-thumb" alt:"prop-thumb" desc:"if true, has a proportionally-sized thumb knob reflecting another value -- e.g., the amount visible in a scrollbar, and thumb is completely inside Size -- otherwise ThumbSize affects Size so that full Size range can be traversed"`

	// value that the thumb represents, in the same units
	ThumbVal float32 `xml:"thumb-val" desc:"value that the thumb represents, in the same units"`

	// logical position of the slider relative to Size
	Pos float32 `xml:"-" desc:"logical position of the slider relative to Size"`

	// underlying drag position of slider -- not subject to snapping
	SlideStartPos float32 `xml:"-" desc:"underlying drag position of slider -- not subject to snapping"`

	// dimension along which the slider slides
	Dim mat32.Dims `desc:"dimension along which the slider slides"`

	// if true, will send continuous updates of value changes as user moves the slider -- otherwise only at the end -- see TrackThr for a threshold on amount of change
	Tracking bool `xml:"tracking" desc:"if true, will send continuous updates of value changes as user moves the slider -- otherwise only at the end -- see TrackThr for a threshold on amount of change"`

	// threshold for amount of change in scroll value before emitting a signal in Tracking mode
	TrackThr float32 `xml:"track-thr" desc:"threshold for amount of change in scroll value before emitting a signal in Tracking mode"`

	// snap the values to Step size increments
	Snap bool `xml:"snap" desc:"snap the values to Step size increments"`

	// can turn off e.g., scrollbar rendering with this flag -- just prevents rendering
	Off bool `desc:"can turn off e.g., scrollbar rendering with this flag -- just prevents rendering"`

	// an additional style object that is used for styling the overall box around the slider; it should be set in the StyleFuncs, just the like the main style object is; it typically has no border and a white/black background; it needs a background to allow local re-rendering
	StyleBox styles.Style `desc:"an additional style object that is used for styling the overall box around the slider; it should be set in the StyleFuncs, just the like the main style object is; it typically has no border and a white/black background; it needs a background to allow local re-rendering"`

	// TODO: make value and thumb full style objects

	// the background color that is used for styling the selected value section of the slider; it should be set in the StyleFuncs, just like the main style object is
	ValueColor colors.Full `desc:"the background color that is used for styling the selected value section of the slider; it should be set in the StyleFuncs, just like the main style object is"`

	// the background color that is used for styling the thumb (handle) of the slider; it should be set in the StyleFuncs, just like the main style object is
	ThumbColor colors.Full `desc:"the background color that is used for styling the thumb (handle) of the slider; it should be set in the StyleFuncs, just like the main style object is"`

	// state of slider
	State SliderStates `json:"-" xml:"-" desc:"state of slider"`

	// styles for different states of the slider, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that
	StateStyles [SliderStatesN]styles.Style `copy:"-" json:"-" xml:"-" desc:"styles for different states of the slider, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`

	// [view: -] signal for slider -- see SliderSignals for the types
	//	SliderSig ki.Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for slider -- see SliderSignals for the types"`
}

func (sb *SliderBase) CopyFieldsFrom(frm any) {
	fr := frm.(*SliderBase)
	sb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
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
	sb.SlideStartPos = fr.SlideStartPos
	sb.Tracking = fr.Tracking
	sb.TrackThr = fr.TrackThr
	sb.Snap = fr.Snap
	sb.Off = fr.Off
}

// func (sb *SliderBase) Disconnect() {
// 	sb.WidgetBase.Disconnect()
// 	// sb.SliderSig.DisconnectAll()
// }

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

	// slider has been selected
	SliderSelected

	// TODO: remove these hacky states

	// use background-color here to fill in selected value of slider
	SliderValue

	// these styles define the overall box around slider -- typically no border and a white background -- needs a background to allow local re-rendering
	SliderBox

	// total number of slider states
	SliderStatesN
)

// SliderSelectors are Style selector names for the different states
var SliderSelectors = []string{":active", ":inactive", ":hover", ":focus", ":down", ":selected", ":value", ":box"}

func (sb *SliderBase) OnInit() {
	sb.Step = 0.1
	sb.PageStep = 0.2
	sb.Max = 1.0
	sb.Prec = 9
	sb.ThumbSize = units.Em(1.5)
	sb.ThSize = 25.0
	sb.ThSizeReal = sb.ThSize
}

// SnapValue snaps the value to step sizes if snap option is set
func (sb *SliderBase) SnapValue() {
	if !sb.Snap {
		return
	}
	sb.Value = mat32.IntMultiple(sb.Value, sb.Step)
	sb.Value = mat32.Truncate(sb.Value, sb.Prec)
}

// SetSliderState sets the slider state to given state, updates style
func (sb *SliderBase) SetSliderState(state SliderStates) {
	prev := sb.State
	if sb.IsDisabled() {
		if sb.StateIs(states.Selected) {
			state = SliderSelected
		} else {
			state = SliderInactive
		}
	} else {
		if state == SliderActive && sb.StateIs(states.Selected) {
			state = SliderSelected
		} else if state == SliderActive && sb.StateIs(states.Focused) {
			state = SliderFocus
		}
	}
	sb.State = state
	sb.Style = sb.StateStyles[state] // get relevant styles
	if prev != state {
		sb.StyMu.Lock()
		sb.ApplyStyleWidget(sb.Sc)
		sb.StyMu.Unlock()
	}
}

// SizeFromAlloc gets size from allocation
func (sb *SliderBase) SizeFromAlloc() {
	if sb.LayState.Alloc.Size.IsNil() {
		return
	}
	spc := sb.BoxSpace()
	sb.Size = sb.LayState.Alloc.Size.Dim(sb.Dim) - spc.Size().Dim(sb.Dim)
	if sb.Size <= 0 {
		return
	}
	if !sb.ValThumb {
		sb.Size -= sb.ThSize // half on each side
	}
	sb.UpdatePosFromValue(sb.Value)
	sb.SlideStartPos = sb.Pos
}

// SendChanged sends a Changed message if given new value is
// different from the existing Value.
func (sb *SliderBase) SendChanged(e events.Event) bool {
	if sb.Value == sb.LastValue {
		return false
	}
	sb.LastValue = sb.Value
	sb.Send(events.Change, e)
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
	if sb.Snap {
		sb.SnapValue()
	}
	sb.UpdatePosFromValue(sb.Value)
	if sb.Tracking && mat32.Abs(sb.LastValue-sb.Value) > sb.TrackThr {
		sb.SendChanged(nil)
	}
	sb.UpdateEndRender(updt)
}

// UpdatePosFromValue updates the slider position based on the current Value
func (sb *SliderBase) UpdatePosFromValue(val float32) {
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
	sb.Pos = effSz * (val - sb.Min) / (sb.Max - sb.Min)
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
		sb.UpdatePosFromValue(val)
		sb.SlideStartPos = sb.Pos
	}
	sb.UpdateEndRender(updt)
}

// SetValueAction sets the value and updates the slider representation, and
// emits a changed signal
func (sb *SliderBase) SetValueAction(val float32) {
	if sb.Value == val {
		return
	}
	sb.SetValue(val)
	sb.Send(events.Change, nil)
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

// PointToRelPos translates a point in global pixel coords into relative
// position within node.  This satisfies the SliderPositioner interface.
func (sb *SliderBase) PointToRelPos(pt image.Point) image.Point {
	sb.BBoxMu.RLock()
	defer sb.BBoxMu.RUnlock()
	return pt.Sub(sb.ScBBox.Min)
}

///////////////////////////////////////////////////////////
// 	Events

func (sb *SliderBase) SliderMouse() {
	sb.On(events.MouseDown, func(e events.Event) {
		if sb.StateIs(states.Disabled) {
			return
		}
		ed := sb.This().(SliderPositioner).PointToRelPos(e.Pos())
		st := &sb.Style
		spc := st.EffMargin().Pos().Dim(sb.Dim) + 0.5*sb.ThSizeReal
		if sb.Dim == mat32.X {
			sb.SetSliderPos(float32(ed.X) - spc)
		} else {
			sb.SetSliderPos(float32(ed.Y) - spc)
		}
		sb.SlideStartPos = sb.Pos
	})
	// note: not doing anything in particular on SlideStart
	sb.On(events.SlideMove, func(e events.Event) {
		if sb.StateIs(states.Disabled) {
			return
		}
		del := e.StartDelta()
		if sb.Dim == mat32.X {
			sb.SetSliderPos(sb.SlideStartPos + float32(del.X))
		} else {
			sb.SetSliderPos(sb.SlideStartPos + float32(del.Y))
		}
	})
	sb.On(events.SlideStop, func(e events.Event) {
		if sb.StateIs(states.Disabled) {
			return
		}
		ed := sb.This().(SliderPositioner).PointToRelPos(e.Pos())
		st := &sb.Style
		spc := st.EffMargin().Pos().Dim(sb.Dim) + 0.5*sb.ThSizeReal
		if sb.Dim == mat32.X {
			sb.SetSliderPos(float32(ed.X) - spc)
		} else {
			sb.SetSliderPos(float32(ed.Y) - spc)
		}
		sb.SlideStartPos = sb.Pos
	})
	sb.On(events.Scroll, func(e events.Event) {
		if sb.StateIs(states.Disabled) {
			return
		}
		se := e.(*events.MouseScroll)
		se.SetHandled()
		if sb.Dim == mat32.X {
			sb.SetSliderPos(sb.SlideStartPos - float32(se.NonZeroDelta(true))) // preferX
		} else {
			sb.SetSliderPos(sb.SlideStartPos - float32(se.NonZeroDelta(false))) // preferY
		}
		sb.SlideStartPos = sb.Pos
	})
}

func (sb *SliderBase) SliderKeys() {
	sb.On(events.KeyChord, func(e events.Event) {
		if sb.StateIs(states.Disabled) {
			return
		}
		if KeyEventTrace {
			fmt.Printf("SliderBase KeyInput: %v\n", sb.Path())
		}
		kf := KeyFun(e.KeyChord())
		switch kf {
		case KeyFunMoveUp:
			sb.SetValueAction(sb.Value - sb.Step)
			e.SetHandled()
		case KeyFunMoveLeft:
			sb.SetValueAction(sb.Value - sb.Step)
			e.SetHandled()
		case KeyFunMoveDown:
			sb.SetValueAction(sb.Value + sb.Step)
			e.SetHandled()
		case KeyFunMoveRight:
			sb.SetValueAction(sb.Value + sb.Step)
			e.SetHandled()
		case KeyFunPageUp:
			sb.SetValueAction(sb.Value - sb.PageStep)
			e.SetHandled()
		// case KeyFunPageLeft:
		// 	sb.SetValueAction(sb.Value - sb.PageStep)
		// 	kt.SetHandled()
		case KeyFunPageDown:
			sb.SetValueAction(sb.Value + sb.PageStep)
			e.SetHandled()
		// case KeyFunPageRight:
		// 	sb.SetValueAction(sb.Value + sb.PageStep)
		// 	kt.SetHandled()
		case KeyFunHome:
			sb.SetValueAction(sb.Min)
			e.SetHandled()
		case KeyFunEnd:
			sb.SetValueAction(sb.Max)
			e.SetHandled()
		}
	})
}

func (sb *SliderBase) SliderBaseHandlers() {
	sb.WidgetHandlers()
	sb.SliderMouse()
	sb.SliderKeys()
}

///////////////////////////////////////////////////////////
// 	Config

func (sb *SliderBase) ConfigWidget(sc *Scene) {
	sb.ConfigSlider(sc)
}

func (sb *SliderBase) ConfigSlider(sc *Scene) {
	sb.State = SliderActive
	if sb.IsDisabled() {
		sb.State = SliderInactive
	}
	sb.ConfigParts(sc)
}

func (sb *SliderBase) ConfigParts(sc *Scene) {
	parts := sb.NewParts(LayoutNil)
	config := ki.Config{}
	icIdx, lbIdx := sb.ConfigPartsIconLabel(&config, sb.Icon, "")
	mods, updt := parts.ConfigChildren(config)
	sb.ConfigPartsSetIconLabel(sb.Icon, "", icIdx, lbIdx)
	if mods {
		sb.UpdateEnd(updt)
	}
}

// StyleFromProps styles Slider-specific fields from ki.Prop properties
// doesn't support inherit or default
func (sr *SliderBase) StyleFromProps(props ki.Props, sc *Scene) {
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		switch key {
		case "value":
			if iv, ok := laser.ToFloat32(val); ok {
				sr.Value = iv
			}
		case "min":
			if iv, ok := laser.ToFloat32(val); ok {
				sr.Min = iv
			}
		case "max":
			if iv, ok := laser.ToFloat32(val); ok {
				sr.Max = iv
			}
		case "step":
			if iv, ok := laser.ToFloat32(val); ok {
				sr.Step = iv
			}
		case "pagestep":
			if iv, ok := laser.ToFloat32(val); ok {
				sr.PageStep = iv
			}
		case "size":
			if iv, ok := laser.ToFloat32(val); ok {
				sr.Size = iv
			}
		case "thumb-size":
			sr.ThumbSize.SetIFace(val, key)
		case "thumb-val":
			if iv, ok := laser.ToFloat32(val); ok {
				sr.ThumbVal = iv
			}
		case "track-thr":
			if iv, ok := laser.ToFloat32(val); ok {
				sr.TrackThr = iv
			}
		case "prec":
			if iv, ok := laser.ToInt(val); ok {
				sr.Prec = int(iv)
			}
		case "val-thumb":
			if bv, ok := laser.ToBool(val); ok {
				sr.ValThumb = bv
			}
		case "tracking":
			if bv, ok := laser.ToBool(val); ok {
				sr.Tracking = bv
			}
		case "snap":
			if bv, ok := laser.ToBool(val); ok {
				sr.Snap = bv
			}
		}
	}
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (sr *SliderBase) StyleToDots(uc *units.Context) {
	sr.ThumbSize.ToDots(uc)
}

func (sr *SliderBase) StyleSlider(sc *Scene) {
	sr.StyMu.Lock()
	defer sr.StyMu.Unlock()

	sr.ApplyStyleWidget(sc)
	sr.StyleToDots(&sr.Style.UnContext)
	if !sr.ValThumb {
		sr.ThSize = sr.ThumbSize.Dots
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  Slider

// Slider is a standard value slider with a fixed-sized thumb knob -- if an
// Icon is set, it is used for the knob of the slider
type Slider struct {
	SliderBase
}

func (sr *Slider) CopyFieldsFrom(frm any) {
	fr := frm.(*Slider)
	sr.SliderBase.CopyFieldsFrom(&fr.SliderBase)
}

func (sr *Slider) OnInit() {
	sr.SliderBase.OnInit() // defaults
	sr.SliderBaseHandlers()
	sr.SliderStyles()
}

func (sr *Slider) SliderStyles() {
	sr.ThumbSize = units.Em(1.5)
	sr.ThSize = 25.0
	sr.ThSizeReal = sr.ThSize

	sr.AddStyles(func(s *styles.Style) {
		s.SetAbilities(true, states.Activatable, states.Focusable, states.Hoverable, states.LongHoverable, states.Slideable)
		sr.ThumbSize = units.Dp(20)
		sr.ValueColor.SetColor(colors.Scheme.Primary.Base)
		sr.ThumbColor.SetColor(colors.Scheme.Primary.Base)

		sr.StyleBox.Border.Style.Set(styles.BorderNone)

		s.Cursor = cursors.Grab
		s.Border.Style.Set(styles.BorderNone)
		s.Border.Radius = styles.BorderRadiusFull
		s.Padding.Set(units.Dp(8))
		if sr.Dim == mat32.X {
			s.Width.SetEm(20)
			s.Height.SetDp(4)
		} else {
			s.Height.SetEm(20)
			s.Width.SetDp(4)
		}
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)
		s.Color = colors.Scheme.Primary.On
		// STYTODO: state styles
		switch {
		case s.Is(states.Sliding):
			sr.ThumbColor.SetSolid(colors.Palette.Primary.Tone(60))
			sr.ValueColor.SetSolid(colors.Palette.Primary.Tone(60))
			s.BackgroundColor.SetSolid(colors.Scheme.OutlineVariant)
			s.Cursor = cursors.Grabbing
		case s.Is(states.Active):
			// todo: just picking something at random to make it visible:
			s.BackgroundColor.SetSolid(colors.Scheme.OutlineVariant)
			s.Color = colors.Scheme.Primary.On
			sr.ThumbColor.SetSolid(colors.Palette.Primary.Tone(50))
			sr.ValueColor.SetSolid(colors.Palette.Primary.Tone(50))
			s.Cursor = cursors.Grabbing
		case s.Is(states.Hovered):
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceVariant)
		}
		if s.Is(states.Focused) {
			s.Border.Style.Set(styles.BorderSolid)
			s.Border.Color.Set(colors.Scheme.Outline)
			s.Border.Width.Set(units.Dp(1))
		}
	})
}

func (sr *Slider) OnChildAdded(child ki.Ki) {
	if _, wb := AsWidget(child); wb != nil {
		switch wb.Name() {
		case "icon":
			wb.AddStyles(func(s *styles.Style) {
				s.Width.SetEm(1)
				s.Height.SetEm(1)
				s.Margin.Set()
				s.Padding.Set()
			})
		}
	}

}

func (sr *Slider) ConfigWidget(sc *Scene) {
	sr.ConfigSlider(sc)
	sr.ConfigParts(sc)
}

func (sr *Slider) ApplyStyle(sc *Scene) {
	sr.SetCanFocusIfActive()
	sr.StyleSlider(sc)
}

func (sr *Slider) GetSize(sc *Scene, iter int) {
	sr.InitLayout(sc)
	st := &sr.Style
	odim := mat32.OtherDim(sr.Dim)
	// get at least thumbsize + margin + border.size
	sz := sr.ThSize + st.EffMargin().Size().Dim(odim) + (st.Border.Width.Dots().Size().Dim(odim))
	sr.LayState.Alloc.Size.SetDim(odim, sz)
}

func (sr *Slider) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	sr.DoLayoutBase(sc, parBBox, iter)
	sr.DoLayoutParts(sc, parBBox, iter)
	sr.SizeFromAlloc()
	return sr.DoLayoutChildren(sc, iter)
}

func (sr *Slider) Render(sc *Scene) {
	if !sr.Off && sr.PushBounds(sc) {
		sr.RenderDefaultStyle(sc)
		sr.RenderChildren(sc)
		sr.PopBounds(sc)
	}
}

// render using a default style if not otherwise styled
func (sr *Slider) RenderDefaultStyle(sc *Scene) {
	rs, pc, st := sr.RenderLock(sc)

	// overall fill box
	sr.RenderStdBox(sc, &sr.StyleBox)

	// SidesTODO: look here if slider borders break

	// pc.StrokeStyle.SetColor(&st.Border.Color)
	// pc.StrokeStyle.Width = st.Border.Width
	pc.FillStyle.SetFullColor(&st.BackgroundColor)

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
	bpos.SetAddDim(odim, spc.Pos().Dim(odim))
	bsz.SetSubDim(odim, spc.Size().Dim(odim))
	bpos.SetAddDim(sr.Dim, spc.Pos().Dim(odim)+ht)
	bsz.SetSubDim(sr.Dim, spc.Size().Dim(odim)+2*ht)
	sr.RenderBoxImpl(sc, bpos, bsz, st.Border)

	bsz.SetDim(sr.Dim, sr.Pos)
	pc.FillStyle.SetFullColor(&sr.ValueColor)
	sr.RenderBoxImpl(sc, bpos, bsz, st.Border)

	tpos.SetDim(sr.Dim, bpos.Dim(sr.Dim)+sr.Pos)
	tpos.SetAddDim(odim, 0.5*sz.Dim(odim)) // ctr
	pc.FillStyle.SetFullColor(&sr.ThumbColor)

	if sr.Icon.IsValid() && sr.Parts.HasChildren() {
		sr.RenderUnlock(rs)
		sr.Parts.Render(sc)
	} else {
		pc.DrawCircle(rs, tpos.X, tpos.Y, ht)
		pc.FillStrokeClear(rs)
		sr.RenderUnlock(rs)
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  ScrollBar

// ScrollBar has a proportional thumb size reflecting amount of content visible
type ScrollBar struct {
	SliderBase
}

func (sb *ScrollBar) CopyFieldsFrom(frm any) {
	fr := frm.(*ScrollBar)
	sb.SliderBase.CopyFieldsFrom(&fr.SliderBase)
}

func (sb *ScrollBar) OnInit() {
	sb.SliderBase.OnInit()
	sb.SliderBaseHandlers()
	sb.ScrollBarStyles()
}

func (sb *ScrollBar) ScrollBarStyles() {
	sb.ValThumb = true
	sb.ThumbSize = units.Ex(1)

	sb.AddStyles(func(s *styles.Style) {
		s.SetAbilities(true, states.Activatable, states.Focusable, states.Hoverable, states.LongHoverable, states.Slideable)
		sb.StyleBox.Border.Style.Set(styles.BorderNone)

		sb.ValueColor.SetSolid(colors.Scheme.OutlineVariant)
		sb.ThumbColor.SetSolid(colors.Transparent)
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)

		s.Border.Style.Set(styles.BorderNone)
		s.Border.Radius = styles.BorderRadiusFull
		// STYTODO: state styles
		switch {
		case s.Is(states.Sliding):
			sb.ThumbColor.SetSolid(colors.Palette.Secondary.Tone(40))
			sb.ValueColor.SetSolid(colors.Palette.Secondary.Tone(40))
			s.BackgroundColor.SetSolid(colors.Scheme.OutlineVariant)
		case s.Is(states.Active):
			// todo: just picking something at random to make it visible:
			s.BackgroundColor.SetSolid(colors.Scheme.OutlineVariant)
			s.Color = colors.Scheme.Primary.On
			sb.ThumbColor.SetSolid(colors.Palette.Secondary.Tone(60))
			sb.ValueColor.SetSolid(colors.Palette.Secondary.Tone(60))
		case s.Is(states.Hovered):
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceVariant)
		}
		if s.Is(states.Focused) {
			s.Border.Style.Set(styles.BorderSolid)
			s.Border.Color.Set(colors.Scheme.Outline)
			s.Border.Width.Set(units.Dp(1))
		}
	})
}

func (sb *ScrollBar) ConfigWidget(sc *Scene) {
	sb.ConfigSlider(sc)
}

func (sb *ScrollBar) ApplyStyle(sc *Scene) {
	sb.SetCanFocusIfActive()
	sb.StyleSlider(sc)
	sb.ConfigParts(sc)
}

func (sb *ScrollBar) GetSize(sc *Scene, iter int) {
	sb.InitLayout(sc)
}

func (sb *ScrollBar) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	sb.DoLayoutBase(sc, parBBox, iter)
	sb.DoLayoutParts(sc, parBBox, iter)
	for i := 0; i < int(SliderStatesN); i++ {
		sb.StateStyles[i].CopyUnitContext(&sb.Style.UnContext)
	}
	sb.SizeFromAlloc()
	return sb.DoLayoutChildren(sc, iter)
}

func (sb *ScrollBar) Render(sc *Scene) {
	if !sb.Off && sb.PushBounds(sc) {
		sb.RenderDefaultStyle(sc)
		sb.RenderChildren(sc)
		sb.PopBounds(sc)
	}
}

// render using a default style if not otherwise styled
func (sb *ScrollBar) RenderDefaultStyle(sc *Scene) {
	rs, pc, st := sb.RenderLock(sc)
	defer sb.RenderUnlock(rs)

	// overall fill box
	sb.RenderStdBox(sc, &sb.StyleBox)

	// pc.StrokeStyle.SetColor(&st.Border.Color)
	// pc.StrokeStyle.Width = st.Border.Width
	bg := st.BackgroundColor
	if bg.IsNil() {
		bg = sb.ParentBackgroundColor()
	}
	pc.FillStyle.SetFullColor(&bg)

	// scrollbar is basic box in content size
	spc := st.BoxSpace()
	pos := sb.LayState.Alloc.Pos.Add(spc.Pos())
	sz := sb.LayState.Alloc.Size.Sub(spc.Size())

	sb.RenderBoxImpl(sc, pos, sz, st.Border) // surround box
	pos.SetAddDim(sb.Dim, sb.Pos)            // start of thumb
	sz.SetDim(sb.Dim, sb.ThSize)
	pc.FillStyle.SetFullColor(&sb.ValueColor)
	sb.RenderBoxImpl(sc, pos, sz, st.Border)
}

////////////////////////////////////////////////////////////////////////////////////////
//  ProgressBar

// ProgressBar is a progress bar that fills up bar as progress continues.
// Call Start with a maximum value to work toward, and ProgStep each time
// a progress step has been accomplished -- increments the ProgCur by one
// and display is updated every ProgInc such steps.
type ProgressBar struct {
	ScrollBar

	// maximum amount of progress to be achieved
	ProgMax int `desc:"maximum amount of progress to be achieved"`

	// progress increment when display is updated -- automatically computed from ProgMax at Start but can be overwritten
	ProgInc int `desc:"progress increment when display is updated -- automatically computed from ProgMax at Start but can be overwritten"`

	// current progress level
	ProgCur int `desc:"current progress level"`

	// mutex for updating progress
	ProgMu sync.Mutex `desc:"mutex for updating progress"`
}

func (pb *ProgressBar) CopyFieldsFrom(frm any) {
	fr := frm.(*ProgressBar)
	pb.SliderBase.CopyFieldsFrom(&fr.SliderBase)
}

func (pb *ProgressBar) OnInit() {
	pb.ScrollBar.OnInit() // use same handlers etc

	pb.Dim = mat32.X
	pb.ValThumb = true
	pb.ThumbVal = 1
	pb.Value = 0
	pb.ThumbSize = units.Ex(1)
	pb.SetState(true, states.ReadOnly) // TODO: this shouldn't be disabled, just read only
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

func (pb *ProgressBar) Start(mx int) {
	pb.ProgMax = mx - 1
	pb.ProgMax = max(1, pb.ProgMax)
	pb.ProgInc = ProgressDefaultInc(mx)
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
