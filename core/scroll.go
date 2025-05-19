// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// autoScrollRate determines the rate of auto-scrolling of layouts
var autoScrollRate = float32(10)

// hasAnyScroll returns true if the frame has any scrollbars.
func (fr *Frame) hasAnyScroll() bool {
	return fr.HasScroll[math32.X] || fr.HasScroll[math32.Y]
}

// ScrollGeom returns the target position and size for scrollbars
func (fr *Frame) ScrollGeom(d math32.Dims) (pos, sz math32.Vector2) {
	sbw := math32.Ceil(fr.Styles.ScrollbarWidth.Dots)
	sbwb := sbw + fr.Styles.Border.Width.Right.Dots + fr.Styles.Margin.Right.Dots
	od := d.Other()
	bbmin := math32.FromPoint(fr.Geom.ContentBBox.Min)
	bbmax := math32.FromPoint(fr.Geom.ContentBBox.Max)
	bbtmax := math32.FromPoint(fr.Geom.TotalBBox.Max)
	if fr.This != fr.Scene.This { // if not the scene, keep inside the scene
		bbmin.SetMax(math32.FromPoint(fr.Scene.Geom.ContentBBox.Min))
		bbmax.SetMin(math32.FromPoint(fr.Scene.Geom.ContentBBox.Max))
		bbtmax.SetMin(math32.FromPoint(fr.Scene.Geom.TotalBBox.Max))
	}
	pos.SetDim(d, bbmin.Dim(d))
	pos.SetDim(od, bbtmax.Dim(od)-sbwb) // base from total
	bbsz := bbmax.Sub(bbmin)
	sz.SetDim(d, bbsz.Dim(d)-4)
	sz.SetDim(od, sbw)
	sz = sz.Ceil()
	return
}

// ConfigScrolls configures any scrollbars that have been enabled
// during the Layout process. This is called during Position, once
// the sizing and need for scrollbars has been established.
// The final position of the scrollbars is set during ScenePos in
// PositionScrolls.  Scrolls are kept around in general.
func (fr *Frame) ConfigScrolls() {
	for d := math32.X; d <= math32.Y; d++ {
		if fr.HasScroll[d] {
			fr.configScroll(d)
		}
	}
}

// configScroll configures scroll for given dimension
func (fr *Frame) configScroll(d math32.Dims) {
	if fr.Scrolls[d] != nil {
		return
	}
	fr.Scrolls[d] = NewSlider()
	sb := fr.Scrolls[d]
	tree.SetParent(sb, fr)
	// sr.SetFlag(true, tree.Field) // note: do not turn on -- breaks pos
	sb.SetType(SliderScrollbar)
	sb.InputThreshold = 1
	sb.Min = 0.0
	sb.Styler(func(s *styles.Style) {
		s.Direction = styles.Directions(d)
		s.Padding.Zero()
		s.Margin.Zero()
		s.MaxBorder.Width.Zero()
		s.Border.Width.Zero()
		s.FillMargin = false
	})
	sb.FinalStyler(func(s *styles.Style) {
		od := d.Other()
		_, sz := fr.This.(Layouter).ScrollGeom(d)
		if sz.X > 0 && sz.Y > 0 {
			s.SetState(false, states.Invisible)
			s.Min.SetDim(d, units.Dot(sz.Dim(d)))
			s.Min.SetDim(od, units.Dot(sz.Dim(od)))
		} else {
			s.SetState(true, states.Invisible)
		}
		s.Max = s.Min
	})
	sb.OnInput(func(e events.Event) {
		e.SetHandled()
		fr.This.(Layouter).ScrollChanged(d, sb)
	})
	sb.Update()
}

// ScrollChanged is called in the OnInput event handler for updating,
// when the scrollbar value has changed, for given dimension.
// This is part of the Layouter interface.
func (fr *Frame) ScrollChanged(d math32.Dims, sb *Slider) {
	fr.Geom.Scroll.SetDim(d, -sb.Value)
	fr.This.(Layouter).ApplyScenePos() // computes updated positions
	fr.NeedsRender()
}

// ScrollUpdateFromGeom updates the scrollbar for given dimension
// based on the current Geom.Scroll value for that dimension.
// This can be used to programatically update the scroll value.
func (fr *Frame) ScrollUpdateFromGeom(d math32.Dims) {
	if !fr.HasScroll[d] || fr.Scrolls[d] == nil {
		return
	}
	sb := fr.Scrolls[d]
	cv := fr.Geom.Scroll.Dim(d)
	sb.setValueEvent(-cv)
	fr.This.(Layouter).ApplyScenePos() // computes updated positions
	fr.NeedsRender()
}

// ScrollValues returns the maximum size that could be scrolled,
// the visible size (which could be less than the max size, in which
// case no scrollbar is needed), and visSize / maxSize as the VisiblePct.
// This is used in updating the scrollbar and determining whether one is
// needed in the first place
func (fr *Frame) ScrollValues(d math32.Dims) (maxSize, visSize, visPct float32) {
	sz := &fr.Geom.Size
	maxSize = sz.Internal.Dim(d)
	visSize = sz.Alloc.Content.Dim(d)
	visPct = visSize / maxSize
	return
}

// SetScrollParams sets scrollbar parameters.  Must set Step and PageStep,
// but can also set others as needed.
// Max and VisiblePct are automatically set based on ScrollValues maxSize, visPct.
func (fr *Frame) SetScrollParams(d math32.Dims, sb *Slider) {
	sb.Min = 0
	sb.Step = 1
	sb.PageStep = float32(fr.Geom.ContentBBox.Dy())
}

// PositionScrolls arranges scrollbars
func (fr *Frame) PositionScrolls() {
	for d := math32.X; d <= math32.Y; d++ {
		if fr.HasScroll[d] && fr.Scrolls[d] != nil {
			fr.positionScroll(d)
		} else {
			fr.Geom.Scroll.SetDim(d, 0)
		}
	}
}

func (fr *Frame) positionScroll(d math32.Dims) {
	sb := fr.Scrolls[d]
	pos, ssz := fr.This.(Layouter).ScrollGeom(d)
	maxSize, _, visPct := fr.This.(Layouter).ScrollValues(d)
	if sb.Geom.Pos.Total == pos && sb.Geom.Size.Actual.Content == ssz && sb.visiblePercent == visPct {
		return
	}
	if ssz.X <= 0 || ssz.Y <= 0 {
		sb.SetState(true, states.Invisible)
		return
	}
	sb.SetState(false, states.Invisible)
	sb.Max = maxSize
	sb.setVisiblePercent(visPct)
	// fmt.Println(ly, d, "vis pct:", asz/csz)
	sb.SetValue(sb.Value) // keep in range
	fr.This.(Layouter).SetScrollParams(d, sb)

	sb.Restyle() // applies style
	sb.SizeUp()
	sb.Geom.Size.Alloc = fr.Geom.Size.Actual
	sb.SizeDown(0)

	sb.Geom.Pos.Total = pos
	sb.setContentPosFromPos()
	// note: usually these are intersected with parent *content* bbox,
	// but scrolls are specifically outside of that.
	sb.setBBoxesFromAllocs()
}

// RenderScrolls renders the scrollbars.
func (fr *Frame) RenderScrolls() {
	for d := math32.X; d <= math32.Y; d++ {
		if fr.HasScroll[d] && fr.Scrolls[d] != nil {
			fr.Scrolls[d].RenderWidget()
		}
	}
}

// setScrollsOff turns off the scrollbars.
func (fr *Frame) setScrollsOff() {
	for d := math32.X; d <= math32.Y; d++ {
		fr.HasScroll[d] = false
	}
}

// scrollActionDelta moves the scrollbar in given dimension by given delta.
// returns whether actually scrolled.
func (fr *Frame) scrollActionDelta(d math32.Dims, delta float32) bool {
	if fr.HasScroll[d] && fr.Scrolls[d] != nil {
		sb := fr.Scrolls[d]
		nval := sb.Value + sb.scrollScale(delta)
		chg := sb.setValueEvent(nval)
		if chg {
			fr.NeedsRender() // only render needed -- scroll updates pos
		}
		return chg
	}
	return false
}

// scrollDelta processes a scroll event.  If only one dimension is processed,
// and there is a non-zero in other, then the consumed dimension is reset to 0
// and the event is left unprocessed, so a higher level can consume the
// remainder.
func (fr *Frame) scrollDelta(e events.Event) {
	se := e.(*events.MouseScroll)
	fdel := se.Delta

	hasShift := e.HasAnyModifier(key.Shift, key.Alt) // shift or alt indicates to scroll horizontally
	if hasShift {
		if !fr.HasScroll[math32.X] { // if we have shift, we can only horizontal scroll
			return
		}
		if fr.scrollActionDelta(math32.X, fdel.Y) {
			e.SetHandled()
		}
		return
	}

	if fr.HasScroll[math32.Y] && fr.HasScroll[math32.X] {
		ch1 := fr.scrollActionDelta(math32.Y, fdel.Y)
		ch2 := fr.scrollActionDelta(math32.X, fdel.X)
		if ch1 || ch2 {
			e.SetHandled()
		}
	} else if fr.HasScroll[math32.Y] {
		if fr.scrollActionDelta(math32.Y, fdel.Y) {
			e.SetHandled()
		}
	} else if fr.HasScroll[math32.X] {
		if se.Delta.X != 0 {
			if fr.scrollActionDelta(math32.X, fdel.X) {
				e.SetHandled()
			}
		} else if se.Delta.Y != 0 {
			if fr.scrollActionDelta(math32.X, fdel.Y) {
				e.SetHandled()
			}
		}
	}
}

// parentScrollFrame returns the first parent frame that has active scrollbars.
func (wb *WidgetBase) parentScrollFrame() *Frame {
	ly := tree.ParentByType[Layouter](wb)
	if ly == nil {
		return nil
	}
	fr := ly.AsFrame()
	if fr.hasAnyScroll() {
		return fr
	}
	return fr.parentScrollFrame()
}

// ScrollToThis tells this widget's parent frame to scroll to keep
// this widget in view. It returns whether any scrolling was done.
func (wb *WidgetBase) ScrollToThis() bool {
	if wb.This == nil {
		return false
	}
	fr := wb.parentScrollFrame()
	if fr == nil {
		return false
	}
	return fr.scrollToWidget(wb.This.(Widget))
}

// ScrollThisToTop tells this widget's parent frame to scroll so the top
// of this widget is at the top of the visible range.
// It returns whether any scrolling was done.
func (wb *WidgetBase) ScrollThisToTop() bool {
	if wb.This == nil {
		return false
	}
	fr := wb.parentScrollFrame()
	if fr == nil {
		return false
	}
	box := wb.AsWidget().Geom.totalRect()
	return fr.ScrollDimToStart(math32.Y, box.Min.Y)
}

// scrollToWidget scrolls the layout to ensure that the given widget is in view.
// It returns whether scrolling was needed.
func (fr *Frame) scrollToWidget(w Widget) bool {
	// note: critical to NOT use BBox b/c it is zero for invisible items!
	box := w.AsWidget().Geom.totalRect()
	if box.Size() == (image.Point{}) {
		return false
	}
	return fr.ScrollToBox(box)
}

// autoScrollDim auto-scrolls along one dimension, based on a position value
// relative to the visible dimensions of the frame
// (i.e., subtracting ed.Geom.Pos.Content).
func (fr *Frame) autoScrollDim(d math32.Dims, pos float32) bool {
	if !fr.HasScroll[d] || fr.Scrolls[d] == nil {
		return false
	}
	sb := fr.Scrolls[d]
	smax := sb.effectiveMax()
	ssz := sb.scrollThumbValue()
	dst := sb.Step * autoScrollRate

	fromMax := ssz - pos                      // distance from max in visible window
	if pos < 0 || pos < math32.Abs(fromMax) { // pushing toward min
		pct := pos / ssz
		if pct < .1 && sb.Value > 0 {
			dst = min(dst, sb.Value)
			sb.setValueEvent(sb.Value - dst)
			return true
		}
	} else {
		pct := fromMax / ssz
		if pct < .1 && sb.Value < smax {
			dst = min(dst, (smax - sb.Value))
			sb.setValueEvent(sb.Value + dst)
			return true
		}
	}
	return false
}

var lastAutoScroll time.Time

// AutoScroll scrolls the layout based on given position in scroll
// coordinates (i.e., already subtracing the BBox Min for a mouse event).
func (fr *Frame) AutoScroll(pos math32.Vector2) bool {
	now := time.Now()
	lag := now.Sub(lastAutoScroll)
	if lag < SystemSettings.LayoutAutoScrollDelay {
		return false
	}
	did := false
	if fr.HasScroll[math32.Y] && fr.HasScroll[math32.X] {
		did = fr.autoScrollDim(math32.Y, pos.Y)
		did = did || fr.autoScrollDim(math32.X, pos.X)
	} else if fr.HasScroll[math32.Y] {
		did = fr.autoScrollDim(math32.Y, pos.Y)
	} else if fr.HasScroll[math32.X] {
		did = fr.autoScrollDim(math32.X, pos.X)
	}
	if did {
		lastAutoScroll = time.Now()
	}
	return did
}

// scrollToBoxDim scrolls to ensure that given target [min..max] range
// along one dimension is in view. Returns true if scrolling was needed
func (fr *Frame) scrollToBoxDim(d math32.Dims, tmini, tmaxi int) bool {
	if !fr.HasScroll[d] || fr.Scrolls[d] == nil {
		return false
	}
	sb := fr.Scrolls[d]
	if sb == nil || sb.This == nil {
		return false
	}
	tmin, tmax := float32(tmini), float32(tmaxi)
	cmin, cmax := fr.Geom.contentRangeDim(d)
	if tmin >= cmin && tmax <= cmax {
		return false
	}
	h := fr.Styles.Font.Size.Dots
	if tmin < cmin { // favors scrolling to start
		trg := sb.Value + tmin - cmin - h
		if trg < 0 {
			trg = 0
		}
		sb.setValueEvent(trg)
		return true
	}
	if (tmax - tmin) < sb.scrollThumbValue() { // only if whole thing fits
		trg := sb.Value + float32(tmax-cmax) + h
		sb.setValueEvent(trg)
		return true
	}
	return false
}

// ScrollToBox scrolls the layout to ensure that given rect box is in view.
// Returns true if scrolling was needed
func (fr *Frame) ScrollToBox(box image.Rectangle) bool {
	did := false
	if fr.HasScroll[math32.Y] && fr.HasScroll[math32.X] {
		did = fr.scrollToBoxDim(math32.Y, box.Min.Y, box.Max.Y)
		did = did || fr.scrollToBoxDim(math32.X, box.Min.X, box.Max.X)
	} else if fr.HasScroll[math32.Y] {
		did = fr.scrollToBoxDim(math32.Y, box.Min.Y, box.Max.Y)
	} else if fr.HasScroll[math32.X] {
		did = fr.scrollToBoxDim(math32.X, box.Min.X, box.Max.X)
	}
	if did {
		fr.NeedsRender()
	}
	return did
}

// ScrollDimToStart scrolls to put the given child coordinate position (eg.,
// top / left of a view box) at the start (top / left) of our scroll area, to
// the extent possible. Returns true if scrolling was needed.
func (fr *Frame) ScrollDimToStart(d math32.Dims, posi int) bool {
	if !fr.HasScroll[d] {
		return false
	}
	pos := float32(posi)
	cmin, _ := fr.Geom.contentRangeDim(d)
	if pos == cmin {
		return false
	}
	sb := fr.Scrolls[d]
	trg := math32.Clamp(sb.Value+(pos-cmin), 0, sb.effectiveMax())
	sb.setValueEvent(trg)
	return true
}

// ScrollDimToContentStart is a helper function that scrolls the layout to the
// start of its content (ie: moves the scrollbar to the very start).
// See also [Frame.IsDimAtContentStart].
func (fr *Frame) ScrollDimToContentStart(d math32.Dims) bool {
	if !fr.HasScroll[d] || fr.Scrolls[d] == nil {
		return false
	}
	sb := fr.Scrolls[d]
	sb.setValueEvent(0)
	return true
}

// IsDimAtContentStart returns whether the given dimension is scrolled to the
// start of its content. See also [Frame.ScrollDimToContentStart].
func (fr *Frame) IsDimAtContentStart(d math32.Dims) bool {
	if !fr.HasScroll[d] || fr.Scrolls[d] == nil {
		return false
	}
	sb := fr.Scrolls[d]
	return sb.Value == 0
}

// ScrollDimToEnd scrolls to put the given child coordinate position (eg.,
// bottom / right of a view box) at the end (bottom / right) of our scroll
// area, to the extent possible. Returns true if scrolling was needed.
func (fr *Frame) ScrollDimToEnd(d math32.Dims, posi int) bool {
	if !fr.HasScroll[d] || fr.Scrolls[d] == nil {
		return false
	}
	pos := float32(posi)
	_, cmax := fr.Geom.contentRangeDim(d)
	if pos == cmax {
		return false
	}
	sb := fr.Scrolls[d]
	trg := math32.Clamp(sb.Value+(pos-cmax), 0, sb.effectiveMax())
	sb.setValueEvent(trg)
	return true
}

// ScrollDimToContentEnd is a helper function that scrolls the layout to the
// end of its content (ie: moves the scrollbar to the very end).
// See also [Frame.IsDimAtContentEnd].
func (fr *Frame) ScrollDimToContentEnd(d math32.Dims) bool {
	if !fr.HasScroll[d] || fr.Scrolls[d] == nil {
		return false
	}
	sb := fr.Scrolls[d]
	sb.setValueEvent(sb.effectiveMax())
	return true
}

// IsDimAtContentEnd returns whether the given dimension is scrolled to the
// end of its content. See also [Frame.ScrollDimToContentEnd].
func (fr *Frame) IsDimAtContentEnd(d math32.Dims) bool {
	if !fr.HasScroll[d] || fr.Scrolls[d] == nil {
		return false
	}
	sb := fr.Scrolls[d]
	return sb.Value == sb.effectiveMax()
}

// ScrollDimToCenter scrolls to put the given child coordinate position (eg.,
// middle of a view box) at the center of our scroll area, to the extent
// possible. Returns true if scrolling was needed.
func (fr *Frame) ScrollDimToCenter(d math32.Dims, posi int) bool {
	if !fr.HasScroll[d] || fr.Scrolls[d] == nil {
		return false
	}
	pos := float32(posi)
	cmin, cmax := fr.Geom.contentRangeDim(d)
	mid := 0.5 * (cmin + cmax)
	if pos == mid {
		return false
	}
	sb := fr.Scrolls[d]
	trg := math32.Clamp(sb.Value+(pos-mid), 0, sb.effectiveMax())
	sb.setValueEvent(trg)
	return true
}
