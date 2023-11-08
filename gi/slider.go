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
	"goki.dev/ki/v2"
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

// Slider is a slideable widget that provides slider functionality for two major Types
// Slider type provides a movable thumb that represents a value.
// Scrollbar type uses the VisiblePct factor that specifies what percent
// of the content is currently visible, which determines the size of the thumb, and the
// range of motion remaining for the thumb Value.
// The Content size (inside the margin and padding) determines the outer bounds of
// the rendered area.
type Slider struct { //goki:embedder
	WidgetBase

	// the type of the slider, which determines the visual and functional properties
	Type SliderTypes `set:"-"`

	// Current value, represented by the position of the thumb.
	// For Slider type, this is the center of the thumb.
	// For Scrollbar, this is the top of the thumb.
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

	// For Scrollbar type only: percent (100 max) of the full range
	// that is visible, which determines the thumb size and range of motion.
	VisiblePct float32

	// size of the thumb as a proportion of the slider thickness, which is
	// Content size (inside the padding).
	ThumbSize mat32.Vec2

	// optional icon for the dragging knob
	Icon icons.Icon `view:"show-name"`

	// if true, will send continuous updates of value changes as user moves the slider.
	// otherwise only at the end. See TrackThr for a threshold on amount of change
	Tracking bool

	// threshold for amount of change in scroll value before emitting a signal in Tracking mode
	TrackThr float32

	// snap the values to Step size increments
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
	sr.Tracking = fr.Tracking
	sr.TrackThr = fr.TrackThr
	sr.Snap = fr.Snap
	sr.Prec = fr.Prec
	sr.ValueColor = fr.ValueColor
	sr.ThumbColor = fr.ThumbColor
}

func (sr *Slider) OnInit() {
	sr.Max = 1.0
	sr.Step = 0.1
	sr.PageStep = 0.2
	sr.Prec = 9
	sr.ThumbSize.Set(1, 1)
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

		if sr.Type == SliderSlider {
			sr.ValueColor.SetSolid(colors.Scheme.Primary.Base)
			sr.ThumbColor.SetSolid(colors.Scheme.Primary.Base)
			s.Padding.Set(units.Dp(8))
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceVariant)
		} else {
			sr.ValueColor.SetSolid(colors.Scheme.OutlineVariant)
			sr.ThumbColor.SetSolid(colors.Scheme.OutlineVariant)
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
		}

		sr.ValueColor = s.StateBackgroundColor(sr.ValueColor)
		sr.ThumbColor = s.StateBackgroundColor(sr.ThumbColor)

		s.Color = colors.Scheme.OnSurface

		s.Display = styles.DisplayNone
		sr.StyleBox.Defaults()
		sr.StyleBox.Border.Style.Set(styles.BorderNone)

		if sr.Dim == mat32.X {
			s.Min.X.Em(20)
			s.Min.Y.Dp(4)
		} else {
			s.Min.Y.Em(20)
			s.Min.X.Dp(4)
		}

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
				s.Min.X.Em(1)
				s.Min.Y.Em(1)
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
	if typ == SliderScrollbar {
		sr.ValThumb = true
		sr.ThumbSize = units.Ex(1)
	}
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

// SizeFromAlloc gets size from allocation
func (sr *Slider) SizeFromAlloc() {
	if sr.Alloc.Size.Total.IsNil() {
		return
	}
	spc := sr.BoxSpace()
	sr.Size = sr.Alloc.Size.Total.Dim(sr.Dim) - spc.Size().Dim(sr.Dim)
	if sr.Size <= 0 {
		return
	}
	if !sr.ValThumb {
		sr.Size -= sr.ThSize // half on each side
	}
	sr.UpdatePosFromValue(sr.Value)
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

// SetSliderPos sets the position of the slider at the given position in pixels,
// and updates the corresponding Value based on that position.
func (sr *Slider) SetSliderPos(pos float32) {
	updt := sr.UpdateStart()
	sr.Pos = pos
	sr.Pos = mat32.Min(sr.Size, sr.Pos)
	effSz := sr.Size
	if sr.ValThumb {
		sr.UpdateThumbValSize()
		sr.Pos = mat32.Min(sr.Size-sr.ThSize, sr.Pos)
		if sr.ThSize != sr.ThSizeReal {
			effSz -= sr.ThSize - sr.ThSizeReal
			effSz -= .5 // rounding errors
		}
	}
	sr.Pos = mat32.Max(0, sr.Pos)
	sr.Value = mat32.Truncate(sr.Min+(sr.Max-sr.Min)*(sr.Pos/effSz), sr.Prec)
	sr.Value = mat32.Clamp(sr.Value, sr.Min, sr.Max)
	if sr.ValThumb {
		sr.Value = mat32.Min(sr.Value, sr.Max-sr.ThumbVal)
	}
	if sr.Snap {
		sr.SnapValue()
	}
	sr.UpdatePosFromValue(sr.Value)
	sr.UpdateEndRender(updt)
}

// SetSliderPosAction sets the position of the slider at the given position in pixels,
// and updates the corresponding Value based on that position.
// This version sends tracking changes
func (sr *Slider) SetSliderPosAction(pos float32) {
	sr.SetSliderPos(pos)
	if sr.Tracking && mat32.Abs(sr.LastValue-sr.Value) > sr.TrackThr {
		sr.SendChanged()
	}
}

// UpdatePosFromValue updates the slider position based on the current Value
func (sr *Slider) UpdatePosFromValue(val float32) {
	if sr.Size == 0.0 {
		return
	}
	effSz := sr.Size
	if sr.ValThumb {
		sr.UpdateThumbValSize()
		if sr.ThSize != sr.ThSizeReal {
			effSz -= sr.ThSize - sr.ThSizeReal
			effSz -= 0.5 // rounding errors
		}
	}
	sr.Pos = effSz * (val - sr.Min) / (sr.Max - sr.Min)
}

// SetValue sets the value and updates the slider position,
// but does not send a Change event (see Action version)
func (sr *Slider) SetValue(val float32) *Slider {
	updt := sr.UpdateStart()
	val = mat32.Min(val, sr.Max)
	if sr.ValThumb {
		val = mat32.Min(val, sr.Max-sr.ThumbVal)
	}
	val = mat32.Max(val, sr.Min)
	if sr.Value != val {
		sr.Value = val
		sr.UpdatePosFromValue(val)
	}
	sr.UpdateEndRender(updt)
	return sr
}

// SetValueAction sets the value and updates the slider representation, and
// emits a changed signal
func (sr *Slider) SetValueAction(val float32) {
	if sr.Value == val {
		return
	}
	sr.SetValue(val)
}

// SetThumbValue sets the thumb value to given value and updates the thumb size.
// For scrollbar-style sliders where the thumb size represents visible range.
func (sr *Slider) SetThumbValue(val float32) *Slider {
	updt := sr.UpdateStart()
	sr.ThumbVal = mat32.Min(val, sr.Max)
	sr.ThumbVal = mat32.Max(sr.ThumbVal, sr.Min)
	sr.UpdateThumbValSize()
	sr.UpdateEndRender(updt)
	return sr
}

// UpdateThumbValSize sets thumb size as proportion of min / max (e.sr., amount
// visible in scrollbar) -- max's out to full size
func (sr *Slider) UpdateThumbValSize() {
	sr.ThSizeReal = ((sr.ThumbVal - sr.Min) / (sr.Max - sr.Min))
	sr.ThSizeReal = mat32.Min(sr.ThSizeReal, 1.0)
	sr.ThSizeReal = mat32.Max(sr.ThSizeReal, 0.0)
	sr.ThSizeReal *= sr.Size
	sr.ThSize = mat32.Max(sr.ThSizeReal, SliderMinThumbSize)
}

///////////////////////////////////////////////////////////
// 	Events

// PointToRelPos translates a point in global pixel coords into relative
// position within node.  This satisfies the SliderPositioner interface.
func (sr *Slider) PointToRelPos(pt image.Point) image.Point {
	sr.BBoxMu.RLock()
	defer sr.BBoxMu.RUnlock()
	return pt.Sub(sr.Alloc.BBox.Min)
}

func (sr *Slider) HandleSliderMouse() {
	sr.On(events.MouseDown, func(e events.Event) {
		ed := sr.This().(SliderPositioner).PointToRelPos(e.LocalPos())
		st := &sr.Styles
		spc := st.TotalMargin().Pos().Dim(sr.Dim) + 0.5*sr.ThSizeReal
		if sr.Dim == mat32.X {
			sr.SetSliderPosAction(float32(ed.X) - spc)
		} else {
			sr.SetSliderPosAction(float32(ed.Y) - spc)
		}
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
		ed := sr.This().(SliderPositioner).PointToRelPos(e.LocalPos())
		st := &sr.Styles
		spc := st.TotalMargin().Pos().Dim(sr.Dim) + 0.5*sr.ThSizeReal
		if sr.Dim == mat32.X {
			sr.SetSliderPosAction(float32(ed.X) - spc)
		} else {
			sr.SetSliderPosAction(float32(ed.Y) - spc)
		}
	})
	sr.On(events.Scroll, func(e events.Event) {
		se := e.(*events.MouseScroll)
		se.SetHandled()
		del := float32(se.DimDelta(sr.Dim))
		if sr.Type == SliderScrollbar {
			del = -del // invert for "natural" scroll
		}
		sr.SetSliderPosAction(sr.Pos - del)
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
	parts := sr.NewParts()
	config := ki.Config{}
	icIdx := -1
	if sr.Icon.IsValid() {
		icIdx = len(config)
		config.Add(IconType, "icon")
	}
	mods, updt := parts.ConfigChildren(config)
	if icIdx >= 0 {
		ic := sr.Parts.Child(icIdx).(*Icon)
		ic.SetIcon(sr.Icon)
	}
	if mods {
		parts.UpdateEndLayout(updt)
		sr.SetNeedsLayoutUpdate(sc, updt)
	}
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (sr *Slider) StyleToDots(uc *units.Context) {
	sr.ThumbSize.ToDots(uc)
}

func (sr *Slider) StyleSlider(sc *Scene) {
	sr.StyMu.Lock()
	defer sr.StyMu.Unlock()

	sr.ApplyStyleWidget(sc)
	sr.StyleToDots(&sr.Styles.UnContext)
	if !sr.ValThumb {
		sr.ThSize = sr.ThumbSize.Dots
	}
}

func (sr *Slider) ApplyStyle(sc *Scene) {
	sr.SetCanFocusIfActive()
	sr.StyleSlider(sc)
}

// todo:
// func (sr *Slider) GetSize(sc *Scene, iter int) {
// 	sr.InitLayout(sc)
// 	st := &sr.Styles
// 	odim := mat32.OtherDim(sr.Dim)
// 	// get at least thumbsize + margin + border.size
// 	sz := sr.ThSize + st.TotalMargin().Size().Dim(odim) + (st.Border.Width.Dots().Size().Dim(odim))
// 	sr.Alloc.Size.Total.SetDim(odim, sz)
// }
//
// func (sr *Slider) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
// 	sr.DoLayoutBase(sc, parBBox, iter)
// 	sr.DoLayoutParts(sc, parBBox, iter)
// 	sr.SizeFromAlloc()
// 	return sr.DoLayoutChildren(sc, iter)
// }

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

	if sr.Type == SliderScrollbar {
		// render entire slider inside the ContentBBox!
		// Size of thumb is implicit in size of BBox = s.Min.Y for a horiz slider
		// s.Min.X is length of track
		// TrackThickness = proportion of content thickness (height for horiz) = 1 default
		// ThumbSize mat32.Vec2 1,1 default.  only orthogonal one is used here

		// track = content + padding
		// thumb = content

		// pc.StrokeStyle.SetColor(&st.Border.Color)
		// pc.StrokeStyle.Width = st.Border.Width
		bg := st.BackgroundColor
		if bg.IsNil() {
			// STYTODO: should we handle parent state layer here?
			bg, _ = sr.ParentBackgroundColor()
		}
		pc.FillStyle.SetFullColor(&bg)

		// scrollbar is basic box in content size
		// spc := st.BoxSpace()
		// pos := sr.Alloc.Pos.Add(spc.Pos())
		sz := sr.Alloc.Size.Content
		pos := sr.Alloc.ContentPos

		if !sr.ValueColor.IsNil() {
			sr.RenderBoxImpl(sc, sr.Alloc.Pos, sr.Alloc.Size.Total, st.Border) // surround box
			pos.SetAddDim(sr.Dim, sr.Pos)                                      // start of thumb
			sz.SetDim(sr.Dim, sr.ThSize)
			pc.FillStyle.SetFullColor(&sr.ValueColor)
			sr.RenderBoxImpl(sc, pos, sz, st.Border)
		}

		sr.RenderUnlock(rs)
	} else {
		// pc.StrokeStyle.SetColor(&st.Border.Color)
		// pc.StrokeStyle.Width = st.Border.Width

		// render entire slider inside the ContentBBox!
		// Size of thumb is implicit in size of BBox = s.Min.Y for a horiz slider
		// s.Min.X is length of track
		// TrackThickness = proportion of content thickness (height for horiz)
		// ThumbSize mat32.Vec2 1,1 default.

		// need to apply state layer
		ebg := st.StateBackgroundColor(st.BackgroundColor)
		pc.FillStyle.SetFullColor(&ebg)

		// layout is as follows, for width dimension
		// |      bw             bw     |
		// |      | pad |  | pad |      |
		// |  |        thumb         |  |
		// |    spc    | | <- ctr
		//
		// for length: | spc | ht | <-start of slider

		spc := st.BoxSpace()
		pos := sr.Alloc.ContentPos
		sz := sr.Alloc.Size.Total
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

		if !sr.ValueColor.IsNil() {
			bsz.SetDim(sr.Dim, sr.Pos)
			pc.FillStyle.SetFullColor(&sr.ValueColor)
			sr.RenderBoxImpl(sc, bpos, bsz, st.Border)
		}

		if !sr.ThumbColor.IsNil() {
			tpos.SetDim(sr.Dim, bpos.Dim(sr.Dim)+sr.Pos)
			tpos.SetAddDim(odim, 0.5*sz.Dim(odim)) // ctr
			pc.FillStyle.SetFullColor(&sr.ThumbColor)
		}

		if sr.Icon.IsValid() && sr.Parts.HasChildren() {
			sr.RenderUnlock(rs)
			sr.Parts.Render(sc)
		} else {
			pc.DrawCircle(rs, tpos.X, tpos.Y, ht)
			pc.FillStrokeClear(rs)
			sr.RenderUnlock(rs)
		}
	}
}
