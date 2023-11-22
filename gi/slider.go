// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/mat32/v2"
)

// Slider is a slideable widget that provides slider functionality for two Types:
// Slider type provides a movable thumb that represents Value as the center of thumb
// Pos position, with room reserved at ends for 1/2 of the thumb size.
// Scrollbar has a VisiblePct factor that specifies the percent of the content
// currently visible, which determines the size of the thumb, and thus the range of motion
// remaining for the thumb Value (VisiblePct = 1 means thumb is full size, and no remaining
// range of motion).
// The Content size (inside the margin and padding) determines the outer bounds of
// the rendered area.
type Slider struct { //goki:embedder
	WidgetBase

	// the type of the slider, which determines the visual and functional properties
	Type SliderTypes `set:"-"`

	// Current value, represented by the position of the thumb.
	Value float32 `set:"-"`

	// dimension along which the slider slides
	Dim mat32.Dims

	// minimum value in range
	Min float32

	// maximum value in range
	Max float32

	// smallest step size to increment
	Step float32

	// larger PageUp / Dn step size
	PageStep float32

	// For Scrollbar type only: proportion (1 max) of the full range of scrolled data
	// that is currently visible.  This determines the thumb size and range of motion:
	// if 1, full slider is the thumb and no motion is possible.
	VisiblePct float32 `set:"-"`

	// Size of the thumb as a proportion of the slider thickness, which is
	// Content size (inside the padding).  This is for actual X,Y dimensions,
	// so must be sensitive to Dim dimension alignment.
	ThumbSize mat32.Vec2

	// TrackSize is the proportion of slider thickness for the visible track
	// for the Slider type.  It is often thinner than the thumb, achieved by
	// values < 1 (.5 default)
	TrackSize float32

	// optional icon for the dragging knob
	Icon icons.Icon `view:"show-name"`

	// threshold for amount of change in scroll value before emitting an input event
	InputThreshold float32

	// whether to snap the values to Step size increments
	Snap bool

	// specifies the precision of decimal places (total, not after the decimal point)
	// to use in representing the number. This helps to truncate small weird floating
	// point values in the nether regions.
	Prec int

	// The background color that is used for styling the selected value section of the slider.
	// It should be set in the StyleFuncs, just like the main style object is.
	// If it is set to transparent, no value is rendered, so the value section of the slider
	// just looks like the rest of the slider.
	ValueColor colors.Full

	// The background color that is used for styling the thumb (handle) of the slider.
	// It should be set in the StyleFuncs, just like the main style object is.
	// If it is set to transparent, no thumb is rendered, so the thumb section of the slider
	// just looks like the rest of the slider.
	ThumbColor colors.Full

	// If true, keep the slider (typically a Scrollbar) within the parent Scene
	// bounding box, if the parent is in view.  This is the default behavior
	// for Layout scrollbars, and setting this flag replicates that behavior
	// in other scrollbars.
	StayInView bool

	//////////////////////////////////////////////////////////////////
	// 	Computed values below

	// logical position of the slider relative to Size
	Pos float32 `edit:"-" set:"-"`

	// previous emitted value - don't re-emit if it is the same
	LastValue float32 `edit:"-" copy:"-" xml:"-" json:"-" set:"-"`

	// Computed size of the slide box in the relevant dimension
	// range of motion, exclusive of spacing, based on layout allocation.
	Size float32 `edit:"-" set:"-"`

	// underlying drag position of slider -- not subject to snapping
	SlideStartPos float32 `edit:"-" set:"-"`
}

// SliderTypes are the different types of sliders
type SliderTypes int32 //enums:enum -trimprefix Slider

const (
	// SliderSlider indicates a standard, user-controllable slider
	// for setting a numeric value
	SliderSlider SliderTypes = iota

	// SliderScrollbar indicates a slider acting as a scrollbar for content
	// This sets the
	SliderScrollbar
)

func (sr *Slider) CopyFieldsFrom(frm any) {
	fr := frm.(*Slider)
	sr.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	sr.Value = fr.Value
	sr.Min = fr.Min
	sr.Max = fr.Max
	sr.Step = fr.Step
	sr.PageStep = fr.PageStep
	sr.VisiblePct = fr.VisiblePct
	sr.ThumbSize = fr.ThumbSize
	sr.Icon = fr.Icon
	sr.InputThreshold = fr.InputThreshold
	sr.Snap = fr.Snap
	sr.Prec = fr.Prec
	sr.ValueColor = fr.ValueColor
	sr.ThumbColor = fr.ThumbColor
}

func (sr *Slider) OnInit() {
	sr.Max = 1.0
	sr.VisiblePct = 1
	sr.Step = 0.1
	sr.PageStep = 0.2
	sr.Prec = 9
	sr.ThumbSize.Set(1, 1)
	sr.TrackSize = 0.5
	sr.HandleSliderEvents()
	sr.SliderStyles()
}

func (sr *Slider) SliderStyles() {
	sr.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable)

		// we use a different color for the thumb and value color
		// (compared to the background color) so that they get the
		// correct state layer
		s.Color = colors.Scheme.Primary.On

		if sr.Dim == mat32.X {
			s.Min.X.Em(20)
			s.Min.Y.Em(1)
		} else {
			s.Min.Y.Em(20)
			s.Min.X.Em(1)
		}
		if sr.Type == SliderSlider {
			sr.ValueColor.SetSolid(colors.Scheme.Primary.Base)
			sr.ThumbColor.SetSolid(colors.Scheme.Primary.Base)
			s.Padding.Set(units.Dp(8))
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceVariant)
		} else {
			if sr.Dim == mat32.X {
				s.Min.Y = s.ScrollBarWidth
			} else {
				s.Min.X = s.ScrollBarWidth
			}
			sr.ValueColor.SetSolid(colors.Scheme.OutlineVariant)
			sr.ThumbColor.SetSolid(colors.Scheme.OutlineVariant)
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
		}

		sr.ValueColor = s.StateBackgroundColor(sr.ValueColor)
		sr.ThumbColor = s.StateBackgroundColor(sr.ThumbColor)
		s.Color = colors.Scheme.OnSurface

		s.Border.Style.Set(styles.BorderNone)
		s.Border.Radius = styles.BorderRadiusFull
		if !sr.IsReadOnly() {
			s.Cursor = cursors.Grab
			switch {
			case s.Is(states.Sliding):
				s.Cursor = cursors.Grabbing
			case s.Is(states.Active):
				s.Cursor = cursors.Grabbing
			}
		}
	})
	sr.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(sr) {
		case "parts/icon":
			w.Style(func(s *styles.Style) {
				s.Min.X.Em(1.5)
				s.Min.Y.Em(1.5)
				s.Margin.Zero()
				s.Padding.Zero()
			})
		}
	})
}

// SetType sets the type of the slider
func (sr *Slider) SetType(typ SliderTypes) *Slider {
	updt := sr.UpdateStart()
	sr.Type = typ
	sr.UpdateEndLayout(updt)
	return sr
}

// SnapValue snaps the value to step sizes if snap option is set
func (sr *Slider) SnapValue() {
	if !sr.Snap {
		return
	}
	sr.Value = mat32.IntMultiple(sr.Value, sr.Step)
	sr.Value = mat32.Truncate(sr.Value, sr.Prec)
}

// SendChanged sends a Changed message if given new value is
// different from the existing Value.
func (sr *Slider) SendChanged(e ...events.Event) bool {
	if sr.Value == sr.LastValue {
		return false
	}
	sr.LastValue = sr.Value
	sr.Send(events.Change, e...)
	return true
}

// SliderSize returns the size available for sliding, based on allocation
func (sr *Slider) SliderSize() float32 {
	sz := sr.Geom.Size.Actual.Content.Dim(sr.Dim)
	if sr.Type != SliderScrollbar {
		thsz := sr.ThumbSizeDots()
		sz -= thsz.Dim(sr.Dim) // half on each size
	}
	return sz
}

// SliderThickness returns the thickness of the slider: Content size in other dim.
func (sr *Slider) SliderThickness() float32 {
	return sr.Geom.Size.Actual.Content.Dim(sr.Dim.Other())
}

// ThumbSizeDots returns the thumb size in dots, based on ThumbSize
// and the content thickness
func (sr *Slider) ThumbSizeDots() mat32.Vec2 {
	return sr.ThumbSize.MulScalar(sr.SliderThickness())
}

// SlideThumbSize returns thumb size, based on type
func (sr *Slider) SlideThumbSize() float32 {
	if sr.Type == SliderScrollbar {
		minsz := sr.SliderThickness()
		return max(mat32.Clamp(sr.VisiblePct, 0, 1)*sr.SliderSize(), minsz)
	}
	return sr.ThumbSizeDots().Dim(sr.Dim)
}

// EffectiveMax returns the effective maximum value represented.
// For the Slider type, it it is just Max.
// for the Scrollbar type, it is Max - Value of thumb size
func (sr *Slider) EffectiveMax() float32 {
	if sr.Type == SliderScrollbar {
		return sr.Max - mat32.Clamp(sr.VisiblePct, 0, 1)*(sr.Max-sr.Min)
	}
	return sr.Max
}

// ScrollThumbValue returns the current scroll VisiblePct
// in terms of the Min - Max range of values.
func (sr *Slider) ScrollThumbValue() float32 {
	return mat32.Clamp(sr.VisiblePct, 0, 1) * (sr.Max - sr.Min)
}

// SetSliderPos sets the position of the slider at the given
// relative position within the usable Content sliding range,
// in pixels, and updates the corresponding Value based on that position.
func (sr *Slider) SetSliderPos(pos float32) {
	sz := sr.Geom.Size.Actual.Content.Dim(sr.Dim)
	if sz <= 0 {
		return
	}
	updt := sr.UpdateStart()
	defer sr.UpdateEndRender(updt)

	thsz := sr.SlideThumbSize()
	thszh := .5 * thsz
	sr.Pos = mat32.Clamp(pos, thszh, sz-thszh)
	prel := (sr.Pos - thszh) / (sz - thsz)
	effmax := sr.EffectiveMax()
	val := mat32.Truncate(sr.Min+prel*(effmax-sr.Min), sr.Prec)
	val = mat32.Clamp(val, sr.Min, effmax)
	// fmt.Println(pos, thsz, prel, val)
	sr.Value = val
	if sr.Snap {
		sr.SnapValue()
	}
	sr.SetPosFromValue(sr.Value) // go back the other way to be fully consistent
}

// SetSliderPosAction sets the position of the slider at the given position in pixels,
// and updates the corresponding Value based on that position.
// This version sends tracking changes
func (sr *Slider) SetSliderPosAction(pos float32) {
	sr.SetSliderPos(pos)
	if mat32.Abs(sr.LastValue-sr.Value) > sr.InputThreshold {
		// TODO(kai/input): we need this for InputThreshold to work, but it breaks Change events
		// sr.LastValue = sr.Value
		sr.Send(events.Input)
	}
}

// SetPosFromValue sets the slider position based on the given value
// (typically rs.Value)
func (sr *Slider) SetPosFromValue(val float32) {
	sz := sr.Geom.Size.Actual.Content.Dim(sr.Dim)
	if sz <= 0 {
		return
	}
	updt := sr.UpdateStart()
	defer sr.UpdateEndRender(updt)

	effmax := sr.EffectiveMax()
	val = mat32.Clamp(val, sr.Min, effmax)
	prel := (val - sr.Min) / (effmax - sr.Min) // relative position 0-1
	thsz := sr.SlideThumbSize()
	thszh := .5 * thsz
	sr.Pos = 0.5*thsz + prel*(sz-thsz)
	sr.Pos = mat32.Clamp(sr.Pos, thszh, sz-thszh)
}

// SetVisiblePct sets the visible pct value for Scrollbar type.
func (sr *Slider) SetVisiblePct(val float32) *Slider {
	sr.VisiblePct = mat32.Clamp(val, 0, 1)
	return sr
}

// SetValue sets the value and updates the slider position,
// but does not send a Change event (see Action version)
func (sr *Slider) SetValue(val float32) *Slider {
	updt := sr.UpdateStart()
	defer sr.UpdateEndRender(updt)

	effmax := sr.EffectiveMax()
	val = mat32.Clamp(val, sr.Min, effmax)
	if sr.Value != val {
		sr.Value = val
		sr.SetPosFromValue(val)
	}
	return sr
}

// SetValueAction sets the value and updates the slider representation, and
// emits an input and change event
func (sr *Slider) SetValueAction(val float32) {
	if sr.Value == val {
		return
	}
	sr.SetValue(val)
	sr.Send(events.Input)
	sr.SendChange()
}

///////////////////////////////////////////////////////////
// 	Events

// PointToRelPos translates a point in scene local pixel coords into relative
// position within the slider content range
func (sr *Slider) PointToRelPos(pt image.Point) float32 {
	sr.BBoxMu.RLock()
	defer sr.BBoxMu.RUnlock()
	ptf := mat32.NewVec2FmPoint(pt).Dim(sr.Dim)
	return ptf - sr.Geom.Pos.Content.Dim(sr.Dim)
}

func (sr *Slider) HandleSliderMouse() {
	sr.On(events.MouseDown, func(e events.Event) {
		pos := sr.PointToRelPos(e.LocalPos())
		sr.SetSliderPosAction(pos)
		sr.SlideStartPos = sr.Pos
	})
	// note: not doing anything in particular on SlideStart
	sr.On(events.SlideMove, func(e events.Event) {
		del := e.StartDelta()
		if sr.Dim == mat32.X {
			sr.SetSliderPosAction(sr.SlideStartPos + float32(del.X))
		} else {
			sr.SetSliderPosAction(sr.SlideStartPos + float32(del.Y))
		}
	})
	sr.On(events.SlideStop, func(e events.Event) {
		pos := sr.PointToRelPos(e.LocalPos())
		sr.SetSliderPosAction(pos)
		sr.SendChanged()
	})
	sr.On(events.Scroll, func(e events.Event) {
		se := e.(*events.MouseScroll)
		se.SetHandled()
		del := float32(se.DimDelta(sr.Dim))
		if sr.Type == SliderScrollbar {
			del = -del // invert for "natural" scroll
		}
		sr.SetSliderPosAction(sr.Pos - del)
		sr.SendChanged()
	})
}

func (sr *Slider) HandleSliderKeys() {
	sr.OnKeyChord(func(e events.Event) {
		if KeyEventTrace {
			fmt.Printf("SliderBase KeyInput: %v\n", sr.Path())
		}
		kf := keyfun.Of(e.KeyChord())
		switch kf {
		case keyfun.MoveUp:
			sr.SetValueAction(sr.Value - sr.Step)
			e.SetHandled()
		case keyfun.MoveLeft:
			sr.SetValueAction(sr.Value - sr.Step)
			e.SetHandled()
		case keyfun.MoveDown:
			sr.SetValueAction(sr.Value + sr.Step)
			e.SetHandled()
		case keyfun.MoveRight:
			sr.SetValueAction(sr.Value + sr.Step)
			e.SetHandled()
		case keyfun.PageUp:
			sr.SetValueAction(sr.Value - sr.PageStep)
			e.SetHandled()
		// case keyfun.PageLeft:
		// 	sr.SetValueAction(sr.Value - sr.PageStep)
		// 	kt.SetHandled()
		case keyfun.PageDown:
			sr.SetValueAction(sr.Value + sr.PageStep)
			e.SetHandled()
		// case keyfun.PageRight:
		// 	sr.SetValueAction(sr.Value + sr.PageStep)
		// 	kt.SetHandled()
		case keyfun.Home:
			sr.SetValueAction(sr.Min)
			e.SetHandled()
		case keyfun.End:
			sr.SetValueAction(sr.Max)
			e.SetHandled()
		}
	})
}

func (sr *Slider) HandleSliderEvents() {
	sr.HandleWidgetEvents()
	sr.HandleSliderMouse()
	sr.HandleSliderKeys()
}

///////////////////////////////////////////////////////////
// 	Config

func (sr *Slider) ConfigWidget(sc *Scene) {
	sr.ConfigSlider(sc)
}

func (sr *Slider) ConfigSlider(sc *Scene) {
	sr.ConfigParts(sc)
}

func (sr *Slider) ConfigParts(sc *Scene) {
	if !sr.Icon.IsValid() {
		if sr.Parts != nil {
			sr.DeleteParts()
		}
		return
	}
	parts := sr.NewParts()
	if !parts.HasChildren() {
		NewIcon(parts, "icon")
	}
	ic := sr.Parts.Child(0).(*Icon)
	ic.SetIcon(sr.Icon)
	ic.Update()
}

func (sr *Slider) Render(sc *Scene) {
	if sr.PushBounds(sc) {
		sr.RenderSlider(sc)
		sr.PopBounds(sc)
	}
}

func (sr *Slider) RenderSlider(sc *Scene) {
	rs, pc, st := sr.RenderLock(sc)

	sr.SetPosFromValue(sr.Value)

	od := sr.Dim.Other()
	if sr.Type == SliderScrollbar {
		// pc.StrokeStyle.SetColor(&st.Border.Color)
		// pc.StrokeStyle.Width = st.Border.Width

		sr.RenderStdBox(sc, st)

		bg := st.StateBackgroundColor(st.BackgroundColor)
		if bg.IsNil() {
			bg, _ = sr.ParentBackgroundColor()
		}
		sz := sr.Geom.Size.Actual.Content
		pos := sr.Geom.Pos.Content
		pc.FillStyle.SetFullColor(&bg)
		sr.RenderBoxImpl(sc, pos, sz, st.Border) // surround box
		if !sr.ValueColor.IsNil() {
			thsz := sr.SlideThumbSize()
			osz := sr.ThumbSizeDots().Dim(od)
			tpos := pos
			tpos.SetAddDim(sr.Dim, sr.Pos)
			tpos.SetSubDim(sr.Dim, thsz*.5)
			tsz := sz
			tsz.SetDim(sr.Dim, thsz)
			origsz := sz.Dim(od)
			tsz.SetDim(od, osz)
			tpos.SetAddDim(od, 0.5*(osz-origsz))
			pc.FillStyle.SetFullColor(&sr.ValueColor)
			sr.RenderBoxImpl(sc, tpos, tsz, st.Border)
		}
		sr.RenderUnlock(rs)
	} else {
		// pc.StrokeStyle.SetColor(&st.Border.Color)
		// pc.StrokeStyle.Width = st.Border.Width

		sz := sr.Geom.Size.Actual.Content
		pos := sr.Geom.Pos.Content
		bg, _ := sr.ParentBackgroundColor()
		pc.FillStyle.SetFullColor(&bg)
		sr.RenderBoxImpl(sc, pos, sz, st.Border)

		// need to apply state layer
		ebg := st.StateBackgroundColor(st.BackgroundColor)
		pc.FillStyle.SetFullColor(&ebg)
		if ebg.IsNil() {
			ebg, _ = sr.ParentBackgroundColor()
		}

		trsz := sz.Dim(od) * sr.TrackSize
		bsz := sz
		bsz.SetDim(od, trsz)
		bpos := pos
		bpos.SetAddDim(od, .5*(sz.Dim(od)-trsz))
		sr.RenderBoxImpl(sc, bpos, bsz, st.Border) // track

		if !sr.ValueColor.IsNil() {
			bsz.SetDim(sr.Dim, sr.Pos)
			pc.FillStyle.SetFullColor(&sr.ValueColor)
			sr.RenderBoxImpl(sc, bpos, bsz, st.Border)
		}

		thsz := sr.ThumbSizeDots()
		tpos := pos
		tpos.SetDim(sr.Dim, pos.Dim(sr.Dim)+sr.Pos)
		tpos.SetAddDim(od, 0.5*sz.Dim(od)) // ctr

		// render thumb as icon or box
		if sr.Icon.IsValid() && sr.Parts.HasChildren() {
			sr.RenderUnlock(rs)
			ic := sr.Parts.Child(0).(*Icon)
			icsz := ic.Geom.Size.Actual.Content
			tpos.SetSub(icsz.MulScalar(.5))
			ic.Geom.Pos.Total = tpos
			ic.SetContentPosFromPos()
			ic.SetBBoxes(sc)
			sr.Parts.Render(sc)
		} else {
			pc.FillStyle.SetFullColor(&sr.ThumbColor)
			tpos.SetSub(thsz.MulScalar(0.5))
			sr.RenderBoxImpl(sc, tpos, thsz, st.Border)
			sr.RenderUnlock(rs)
		}
	}
}

func (sr *Slider) ScenePos(sc *Scene) {
	sr.WidgetBase.ScenePos(sc)
	if !sr.StayInView {
		return
	}
	_, pwb := sr.ParentWidget()
	zr := image.Rectangle{}
	if !pwb.IsVisible() || pwb.Geom.TotalBBox == zr {
		return
	}
	sbw := mat32.Ceil(sr.Styles.ScrollBarWidth.Dots)
	scmax := mat32.NewVec2FmPoint(sc.Geom.ContentBBox.Max).SubScalar(sbw)
	sr.Geom.Pos.Total.SetMin(scmax)
	sr.SetContentPosFromPos()
	sr.SetBBoxesFromAllocs()
}
