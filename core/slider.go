// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log/slog"
	"reflect"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// Slider is a slideable widget that provides slider functionality with a draggable
// thumb and a clickable track. The [styles.Style.Direction] determines the direction
// in which the slider slides.
type Slider struct {
	Frame

	// Type is the type of the slider, which determines its visual
	// and functional properties. The default type, [SliderSlider],
	// should work for most end-user use cases.
	Type SliderTypes

	// Value is the current value, represented by the position of the thumb.
	// It defaults to 0.5.
	Value float32 `set:"-"`

	// Min is the minimum possible value.
	// It defaults to 0.
	Min float32

	// Max is the maximum value supported.
	// It defaults to 1.
	Max float32

	// Step is the amount that the arrow keys increment/decrement the value by.
	// It defaults to 0.1.
	Step float32

	// EnforceStep is whether to ensure that the value is always
	// a multiple of [Slider.Step].
	EnforceStep bool

	// PageStep is the amount that the PageUp and PageDown keys
	// increment/decrement the value by.
	// It defaults to 0.2, and will be at least as big as [Slider.Step].
	PageStep float32

	// Icon is an optional icon to use for the dragging thumb.
	Icon icons.Icon

	// For Scrollbar type only: proportion (1 max) of the full range of scrolled data
	// that is currently visible.  This determines the thumb size and range of motion:
	// if 1, full slider is the thumb and no motion is possible.
	visiblePercent float32 `set:"-"`

	// ThumbSize is the size of the thumb as a proportion of the slider thickness,
	// which is the content size (inside the padding).
	ThumbSize math32.Vector2

	// TrackSize is the proportion of slider thickness for the visible track
	// for the [SliderSlider] type. It is often thinner than the thumb, achieved
	// by values less than 1 (0.5 default).
	TrackSize float32 `default:"0.5"`

	// InputThreshold is the threshold for the amount of change in scroll
	// value before emitting an input event.
	InputThreshold float32

	// Precision specifies the precision of decimal places (total, not after the decimal
	// point) to use in representing the number. This helps to truncate small weird
	// floating point values.
	Precision int

	// ValueColor is the background color that is used for styling the selected value
	// section of the slider. It should be set in a Styler, just like the main style
	// object is. If it is set to transparent, no value is rendered, so the value
	// section of the slider just looks like the rest of the slider.
	ValueColor image.Image

	// ThumbColor is the background color that is used for styling the thumb (handle)
	// of the slider. It should be set in a Styler, just like the main style object is.
	// If it is set to transparent, no thumb is rendered, so the thumb section of the
	// slider just looks like the rest of the slider.
	ThumbColor image.Image

	// StayInView is whether to keep the slider (typically a [SliderScrollbar]) within
	// the parent [Scene] bounding box, if the parent is in view. This is the default
	// behavior for [Frame] scrollbars, and setting this flag replicates that behavior
	// in other scrollbars.
	StayInView bool

	// Computed values below:

	// logical position of the slider relative to Size
	pos float32

	// previous Change event emitted value; don't re-emit Change if it is the same
	lastValue float32

	// previous sliding value (for computing the Input change)
	prevSlide float32

	// underlying drag position of slider; not subject to snapping
	slideStartPos float32
}

// SliderTypes are the different types of sliders.
type SliderTypes int32 //enums:enum -trim-prefix Slider

const (
	// SliderSlider indicates a standard, user-controllable slider
	// for setting a numeric value.
	SliderSlider SliderTypes = iota

	// SliderScrollbar indicates a slider acting as a scrollbar for content.
	// It has a [Slider.visiblePercent] factor that specifies the percent of the content
	// currently visible, which determines the size of the thumb, and thus the range
	// of motion remaining for the thumb Value ([Slider.visiblePercent] = 1 means thumb
	// is full size, and no remaining range of motion). The content size (inside the
	// margin and padding) determines the outer bounds of the rendered area.
	SliderScrollbar
)

func (sr *Slider) WidgetValue() any { return &sr.Value }

func (sr *Slider) OnBind(value any, tags reflect.StructTag) {
	kind := reflectx.NonPointerType(reflect.TypeOf(value)).Kind()
	if kind >= reflect.Int && kind <= reflect.Uintptr {
		sr.SetStep(1).SetEnforceStep(true).SetMax(100)
	}
	setFromTag(tags, "min", func(v float32) { sr.SetMin(v) })
	setFromTag(tags, "max", func(v float32) { sr.SetMax(v) })
	setFromTag(tags, "step", func(v float32) { sr.SetStep(v) })
}

func (sr *Slider) Init() {
	sr.Frame.Init()
	sr.Value = 0.5
	sr.Max = 1
	sr.visiblePercent = 1
	sr.Step = 0.1
	sr.PageStep = 0.2
	sr.Precision = 9
	sr.ThumbSize.Set(1, 1)
	sr.TrackSize = 0.5
	sr.lastValue = -math32.MaxFloat32
	sr.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable)

		// we use a different color for the thumb and value color
		// (compared to the background color) so that they get the
		// correct state layer
		s.Color = colors.Scheme.Primary.On

		if sr.Type == SliderSlider {
			sr.ValueColor = colors.Scheme.Primary.Base
			sr.ThumbColor = colors.Scheme.Primary.Base
			s.Padding.Set(units.Dp(8))
			s.Background = colors.Scheme.SurfaceVariant
		} else {
			sr.ValueColor = colors.Scheme.OutlineVariant
			sr.ThumbColor = colors.Scheme.OutlineVariant
			s.Background = colors.Scheme.SurfaceContainerLow
		}

		// sr.ValueColor = s.StateBackgroundColor(sr.ValueColor)
		// sr.ThumbColor = s.StateBackgroundColor(sr.ThumbColor)
		s.Color = colors.Scheme.OnSurface

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
	sr.FinalStyler(func(s *styles.Style) {
		if s.Direction == styles.Row {
			s.Min.X.Em(20)
			s.Min.Y.Em(1)
		} else {
			s.Min.Y.Em(20)
			s.Min.X.Em(1)
		}
		if sr.Type == SliderScrollbar {
			if s.Direction == styles.Row {
				s.Min.Y = s.ScrollbarWidth
			} else {
				s.Min.X = s.ScrollbarWidth
			}
		}
	})
	sr.On(events.SlideStart, func(e events.Event) {
		pos := sr.pointToRelPos(e.Pos())
		sr.setSliderPosEvent(pos)
		sr.lastValue = -math32.MaxFloat32
		sr.slideStartPos = sr.pos
	})
	sr.On(events.SlideMove, func(e events.Event) {
		del := e.StartDelta()
		if sr.Styles.Direction == styles.Row {
			sr.setSliderPosEvent(sr.slideStartPos + float32(del.X))
		} else {
			sr.setSliderPosEvent(sr.slideStartPos + float32(del.Y))
		}
	})
	sr.On(events.SlideStop, func(e events.Event) {
		del := e.StartDelta()
		if sr.Styles.Direction == styles.Row {
			sr.setSliderPosEvent(sr.slideStartPos + float32(del.X))
		} else {
			sr.setSliderPosEvent(sr.slideStartPos + float32(del.Y))
		}
		sr.sendChange()
	})
	sr.On(events.Click, func(e events.Event) {
		pos := sr.pointToRelPos(e.Pos())
		if !sr.setSliderPosEvent(pos) {
			sr.Send(events.Input) // always send on click even if same.
		}
		sr.sendChange()
	})
	sr.On(events.Scroll, func(e events.Event) {
		se := e.(*events.MouseScroll)
		se.SetHandled()
		var del float32
		// if we are scrolling in the y direction on an x slider,
		// we still count it
		if sr.Styles.Direction == styles.Row && se.Delta.X != 0 {
			del = se.Delta.X
		} else {
			del = se.Delta.Y
		}
		if sr.Type == SliderScrollbar {
			del = -del // invert for "natural" scroll
		}
		edel := sr.scrollScale(del)
		sr.setValueEvent(sr.Value + edel)
		sr.sendChange()
	})
	sr.OnKeyChord(func(e events.Event) {
		kf := keymap.Of(e.KeyChord())
		if DebugSettings.KeyEventTrace {
			slog.Info("SliderBase KeyInput", "widget", sr, "keyFunction", kf)
		}
		switch kf {
		case keymap.MoveUp:
			sr.setValueEvent(sr.Value - sr.Step)
			e.SetHandled()
		case keymap.MoveLeft:
			sr.setValueEvent(sr.Value - sr.Step)
			e.SetHandled()
		case keymap.MoveDown:
			sr.setValueEvent(sr.Value + sr.Step)
			e.SetHandled()
		case keymap.MoveRight:
			sr.setValueEvent(sr.Value + sr.Step)
			e.SetHandled()
		case keymap.PageUp:
			if sr.PageStep < sr.Step {
				sr.PageStep = 2 * sr.Step
			}
			sr.setValueEvent(sr.Value - sr.PageStep)
			e.SetHandled()
		case keymap.PageDown:
			if sr.PageStep < sr.Step {
				sr.PageStep = 2 * sr.Step
			}
			sr.setValueEvent(sr.Value + sr.PageStep)
			e.SetHandled()
		case keymap.Home:
			sr.setValueEvent(sr.Min)
			e.SetHandled()
		case keymap.End:
			sr.setValueEvent(sr.Max)
			e.SetHandled()
		}
	})

	sr.Maker(func(p *tree.Plan) {
		if !sr.Icon.IsSet() {
			return
		}
		tree.AddAt(p, "icon", func(w *Icon) {
			w.Styler(func(s *styles.Style) {
				s.Font.Size.Dp(24)
				s.Color = sr.ThumbColor
			})
			w.Updater(func() {
				w.SetIcon(sr.Icon)
			})
		})
	})
}

// snapValue snaps the value to [Slider.Step] if [Slider.EnforceStep] is on.
func (sr *Slider) snapValue() {
	if !sr.EnforceStep {
		return
	}
	// round to the nearest step
	sr.Value = sr.Step * math32.Round(sr.Value/sr.Step)
}

// sendChange calls [WidgetBase.SendChange] if the current value
// is different from the last value.
func (sr *Slider) sendChange(e ...events.Event) bool {
	if sr.Value == sr.lastValue {
		return false
	}
	sr.lastValue = sr.Value
	sr.SendChange(e...)
	return true
}

// sliderSize returns the size available for sliding, based on allocation
func (sr *Slider) sliderSize() float32 {
	sz := sr.Geom.Size.Actual.Content.Dim(sr.Styles.Direction.Dim())
	if sr.Type != SliderScrollbar {
		thsz := sr.thumbSizeDots()
		sz -= thsz.Dim(sr.Styles.Direction.Dim()) // half on each size
	}
	return sz
}

// sliderThickness returns the thickness of the slider: Content size in other dim.
func (sr *Slider) sliderThickness() float32 {
	return sr.Geom.Size.Actual.Content.Dim(sr.Styles.Direction.Dim().Other())
}

// thumbSizeDots returns the thumb size in dots, based on ThumbSize
// and the content thickness
func (sr *Slider) thumbSizeDots() math32.Vector2 {
	return sr.ThumbSize.MulScalar(sr.sliderThickness())
}

// slideThumbSize returns thumb size, based on type
func (sr *Slider) slideThumbSize() float32 {
	if sr.Type == SliderScrollbar {
		minsz := sr.sliderThickness()
		return max(math32.Clamp(sr.visiblePercent, 0, 1)*sr.sliderSize(), minsz)
	}
	return sr.thumbSizeDots().Dim(sr.Styles.Direction.Dim())
}

// effectiveMax returns the effective maximum value represented.
// For the Slider type, it it is just Max.
// for the Scrollbar type, it is Max - Value of thumb size
func (sr *Slider) effectiveMax() float32 {
	if sr.Type == SliderScrollbar {
		return sr.Max - math32.Clamp(sr.visiblePercent, 0, 1)*(sr.Max-sr.Min)
	}
	return sr.Max
}

// scrollThumbValue returns the current scroll VisiblePct
// in terms of the Min - Max range of values.
func (sr *Slider) scrollThumbValue() float32 {
	return math32.Clamp(sr.visiblePercent, 0, 1) * (sr.Max - sr.Min)
}

// setSliderPos sets the position of the slider at the given
// relative position within the usable Content sliding range,
// in pixels, and updates the corresponding Value based on that position.
func (sr *Slider) setSliderPos(pos float32) {
	sz := sr.Geom.Size.Actual.Content.Dim(sr.Styles.Direction.Dim())
	if sz <= 0 {
		return
	}

	thsz := sr.slideThumbSize()
	thszh := .5 * thsz
	sr.pos = math32.Clamp(pos, thszh, sz-thszh)
	prel := (sr.pos - thszh) / (sz - thsz)
	effmax := sr.effectiveMax()
	val := math32.Truncate(sr.Min+prel*(effmax-sr.Min), sr.Precision)
	val = math32.Clamp(val, sr.Min, effmax)
	// fmt.Println(pos, thsz, prel, val)
	sr.Value = val
	sr.snapValue()
	sr.setPosFromValue(sr.Value) // go back the other way to be fully consistent
	sr.NeedsRender()
}

// setSliderPosEvent sets the position of the slider at the given position in pixels,
// and updates the corresponding Value based on that position.
// This version sends input events. Returns true if event sent.
func (sr *Slider) setSliderPosEvent(pos float32) bool {
	sr.setSliderPos(pos)
	if math32.Abs(sr.prevSlide-sr.Value) > sr.InputThreshold {
		sr.prevSlide = sr.Value
		sr.Send(events.Input)
		return true
	}
	return false
}

// setPosFromValue sets the slider position based on the given value
// (typically rs.Value)
func (sr *Slider) setPosFromValue(val float32) {
	sz := sr.Geom.Size.Actual.Content.Dim(sr.Styles.Direction.Dim())
	if sz <= 0 {
		return
	}

	effmax := sr.effectiveMax()
	val = math32.Clamp(val, sr.Min, effmax)
	prel := (val - sr.Min) / (effmax - sr.Min) // relative position 0-1
	thsz := sr.slideThumbSize()
	thszh := .5 * thsz
	sr.pos = 0.5*thsz + prel*(sz-thsz)
	sr.pos = math32.Clamp(sr.pos, thszh, sz-thszh)
	sr.NeedsRender()
}

// setVisiblePercent sets the [Slider.visiblePercent] value for a [SliderScrollbar].
func (sr *Slider) setVisiblePercent(val float32) *Slider {
	sr.visiblePercent = math32.Clamp(val, 0, 1)
	return sr
}

// SetValue sets the value and updates the slider position,
// but does not send an [events.Change] event.
func (sr *Slider) SetValue(value float32) *Slider {
	effmax := sr.effectiveMax()
	value = math32.Clamp(value, sr.Min, effmax)
	if sr.Value != value {
		sr.Value = value
		sr.snapValue()
		sr.setPosFromValue(value)
	}
	sr.NeedsRender()
	return sr
}

// setValueEvent sets the value and updates the slider representation, and
// emits an input and change event.  Returns true if value actually changed.
func (sr *Slider) setValueEvent(val float32) bool {
	if sr.Value == val {
		return false
	}
	curVal := sr.Value
	sr.SetValue(val)
	sr.Send(events.Input)
	sr.SendChange()
	return curVal != sr.Value
}

func (sr *Slider) WidgetTooltip(pos image.Point) (string, image.Point) {
	res := sr.Tooltip
	if sr.Type == SliderScrollbar {
		return res, sr.DefaultTooltipPos()
	}
	if res != "" {
		res += " "
	}
	res += fmt.Sprintf("(value: %.4g, minimum: %.4g, maximum: %.4g)", sr.Value, sr.Min, sr.Max)
	return res, sr.DefaultTooltipPos()
}

// pointToRelPos translates a point in scene local pixel coords into relative
// position within the slider content range
func (sr *Slider) pointToRelPos(pt image.Point) float32 {
	ptf := math32.FromPoint(pt).Dim(sr.Styles.Direction.Dim())
	return ptf - sr.Geom.Pos.Content.Dim(sr.Styles.Direction.Dim())
}

// scrollScale returns scaled value of scroll delta
// as a function of the step size.
func (sr *Slider) scrollScale(del float32) float32 {
	return del * sr.Step
}

func (sr *Slider) Render() {
	sr.setPosFromValue(sr.Value)

	pc := &sr.Scene.Painter
	st := &sr.Styles

	dim := sr.Styles.Direction.Dim()
	od := dim.Other()
	sz := sr.Geom.Size.Actual.Content
	pos := sr.Geom.Pos.Content

	pabg := sr.parentActualBackground()

	if sr.Type == SliderScrollbar {
		pc.StandardBox(st, pos, sz, pabg) // track
		if sr.ValueColor != nil {
			thsz := sr.slideThumbSize()
			osz := sr.thumbSizeDots().Dim(od)
			tpos := pos
			tpos = tpos.AddDim(dim, sr.pos)
			tpos = tpos.SubDim(dim, thsz*.5)
			tsz := sz
			tsz.SetDim(dim, thsz)
			origsz := sz.Dim(od)
			tsz.SetDim(od, osz)
			tpos = tpos.AddDim(od, 0.5*(osz-origsz))
			vabg := sr.Styles.ComputeActualBackgroundFor(sr.ValueColor, pabg)
			pc.Fill.Color = vabg
			sr.RenderBoxGeom(tpos, tsz, styles.Border{Radius: st.Border.Radius}) // thumb
		}
	} else {
		prevbg := st.Background
		prevsl := st.StateLayer
		// use surrounding background with no state layer for surrounding box
		st.Background = pabg
		st.StateLayer = 0
		st.ComputeActualBackground(pabg)
		// surrounding box (needed to prevent it from rendering over itself)
		sr.RenderStandardBox()
		st.Background = prevbg
		st.StateLayer = prevsl
		st.ComputeActualBackground(pabg)

		trsz := sz.Dim(od) * sr.TrackSize
		bsz := sz
		bsz.SetDim(od, trsz)
		bpos := pos
		bpos = bpos.AddDim(od, .5*(sz.Dim(od)-trsz))
		pc.Fill.Color = st.ActualBackground
		sr.RenderBoxGeom(bpos, bsz, styles.Border{Radius: st.Border.Radius}) // track

		if sr.ValueColor != nil {
			bsz.SetDim(dim, sr.pos)
			vabg := sr.Styles.ComputeActualBackgroundFor(sr.ValueColor, pabg)
			pc.Fill.Color = vabg
			sr.RenderBoxGeom(bpos, bsz, styles.Border{Radius: st.Border.Radius})
		}

		thsz := sr.thumbSizeDots()
		tpos := pos
		tpos.SetDim(dim, pos.Dim(dim)+sr.pos)
		tpos = tpos.AddDim(od, 0.5*sz.Dim(od)) // ctr

		// render thumb as icon or box
		if sr.Icon.IsSet() && sr.HasChildren() {
			ic := sr.Child(0).(*Icon)
			tpos.SetSub(thsz.MulScalar(.5))
			ic.Geom.Pos.Total = tpos
			ic.setContentPosFromPos()
			ic.setBBoxes()
		} else {
			tabg := sr.Styles.ComputeActualBackgroundFor(sr.ThumbColor, pabg)
			pc.Fill.Color = tabg
			tpos.SetSub(thsz.MulScalar(0.5))
			sr.RenderBoxGeom(tpos, thsz, styles.Border{Radius: st.Border.Radius})
		}
	}
}

func (sr *Slider) ApplyScenePos() {
	sr.WidgetBase.ApplyScenePos()
	if !sr.StayInView {
		return
	}
	pwb := sr.parentWidget()
	if !pwb.IsVisible() {
		return
	}
	sbw := math32.Ceil(sr.Styles.ScrollbarWidth.Dots)
	scmax := math32.FromPoint(sr.Scene.Geom.ContentBBox.Max).SubScalar(sbw)
	sr.Geom.Pos.Total.SetMin(scmax)
	sr.setContentPosFromPos()
	sr.setBBoxesFromAllocs()
}
