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

// HasAnyScroll returns true if layout has
func (ly *Layout) HasAnyScroll() bool {
	return ly.HasScroll[math32.X] || ly.HasScroll[math32.Y]
}

// ScrollGeom returns the target position and size for scrollbars
func (ly *Layout) ScrollGeom(d math32.Dims) (pos, sz math32.Vector2) {
	sbw := math32.Ceil(ly.Styles.ScrollBarWidth.Dots)
	od := d.Other()
	bbmin := math32.Vector2FromPoint(ly.Geom.ContentBBox.Min)
	bbmax := math32.Vector2FromPoint(ly.Geom.ContentBBox.Max)
	if ly.This() != ly.Scene.This() { // if not the scene, keep inside the scene
		bbmin.SetMax(math32.Vector2FromPoint(ly.Scene.Geom.ContentBBox.Min))
		bbmax.SetMin(math32.Vector2FromPoint(ly.Scene.Geom.ContentBBox.Max).SubScalar(sbw))
	}
	pos.SetDim(d, bbmin.Dim(d))
	pos.SetDim(od, bbmax.Dim(od))
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
func (ly *Layout) ConfigScrolls() {
	for d := math32.X; d <= math32.Y; d++ {
		if ly.HasScroll[d] {
			ly.ConfigScroll(d)
		}
	}
}

// ConfigScroll configures scroll for given dimension
func (ly *Layout) ConfigScroll(d math32.Dims) {
	if ly.Scrolls[d] != nil {
		return
	}
	ly.Scrolls[d] = NewSlider()
	sb := ly.Scrolls[d]
	tree.SetParent(sb, ly)
	// sr.SetFlag(true, tree.Field) // note: do not turn on -- breaks pos
	sb.SetType(SliderScrollbar)
	sb.InputThreshold = 1
	sb.Min = 0.0
	sb.Style(func(s *styles.Style) {
		s.Direction = styles.Directions(d)
		s.Padding.Zero()
		s.Margin.Zero()
		s.MaxBorder.Width.Zero()
		s.Border.Width.Zero()
		s.FillMargin = false
	})
	sb.StyleFinal(func(s *styles.Style) {
		od := d.Other()
		_, sz := ly.This().(Layouter).ScrollGeom(d)
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
		ly.This().(Layouter).ScrollChanged(d, sb)
	})
	sb.Update()
}

// ScrollChanged is called in the OnInput event handler for updating,
// when the scrollbar value has changed, for given dimension.
// This is part of the Layouter interface.
func (ly *Layout) ScrollChanged(d math32.Dims, sb *Slider) {
	ly.Geom.Scroll.SetDim(d, -sb.Value)
	ly.This().(Layouter).ScenePos() // computes updated positions
	ly.NeedsRender()
}

// ScrollValues returns the maximum size that could be scrolled,
// the visible size (which could be less than the max size, in which
// case no scrollbar is needed), and visSize / maxSize as the VisiblePct.
// This is used in updating the scrollbar and determining whether one is
// needed in the first place
func (ly *Layout) ScrollValues(d math32.Dims) (maxSize, visSize, visPct float32) {
	sz := &ly.Geom.Size
	maxSize = sz.Internal.Dim(d)
	visSize = sz.Alloc.Content.Dim(d)
	visPct = visSize / maxSize
	return
}

// SetScrollParams sets scrollbar parameters.  Must set Step and PageStep,
// but can also set others as needed.
// Max and VisiblePct are automatically set based on ScrollValues maxSize, visPct.
func (ly *Layout) SetScrollParams(d math32.Dims, sb *Slider) {
	sb.Step = ly.Styles.Font.Size.Dots // step by lines
	sb.PageStep = 10.0 * sb.Step       // todo: more dynamic
}

// PositionScrolls arranges scrollbars
func (ly *Layout) PositionScrolls() {
	for d := math32.X; d <= math32.Y; d++ {
		if ly.HasScroll[d] && ly.Scrolls[d] != nil {
			ly.PositionScroll(d)
		} else {
			ly.Geom.Scroll.SetDim(d, 0)
		}
	}
}

func (ly *Layout) PositionScroll(d math32.Dims) {
	sb := ly.Scrolls[d]
	pos, ssz := ly.This().(Layouter).ScrollGeom(d)
	maxSize, _, visPct := ly.This().(Layouter).ScrollValues(d)
	if sb.Geom.Pos.Total == pos && sb.Geom.Size.Actual.Content == ssz && sb.VisiblePct == visPct {
		return
	}
	if ssz.X <= 0 || ssz.Y <= 0 {
		sb.SetState(true, states.Invisible)
		return
	}
	sb.SetState(false, states.Invisible)
	sb.Max = maxSize
	sb.SetVisiblePct(visPct)
	// fmt.Println(ly, d, "vis pct:", asz/csz)
	sb.SetValue(sb.Value) // keep in range
	ly.This().(Layouter).SetScrollParams(d, sb)

	sb.Update() // applies style
	sb.SizeUp()
	sb.Geom.Size.Alloc = ly.Geom.Size.Actual
	sb.SizeDown(0)

	sb.Geom.Pos.Total = pos
	sb.SetContentPosFromPos()
	// note: usually these are intersected with parent *content* bbox,
	// but scrolls are specifically outside of that.
	sb.SetBBoxesFromAllocs()
}

// RenderScrolls draws the scrollbars
func (ly *Layout) RenderScrolls() {
	for d := math32.X; d <= math32.Y; d++ {
		if ly.HasScroll[d] && ly.Scrolls[d] != nil {
			ly.Scrolls[d].RenderWidget()
		}
	}
}

// SetScrollsOff turns off the scrollbars
func (ly *Layout) SetScrollsOff() {
	for d := math32.X; d <= math32.Y; d++ {
		ly.HasScroll[d] = false
	}
}

// ScrollActionDelta moves the scrollbar in given dimension by given delta
// and emits a ScrollSig signal.
func (ly *Layout) ScrollActionDelta(d math32.Dims, delta float32) {
	if ly.HasScroll[d] && ly.Scrolls[d] != nil {
		sb := ly.Scrolls[d]
		nval := sb.Value + sb.ScrollScale(delta)
		sb.SetValueAction(nval)
		ly.NeedsRender() // only render needed -- scroll updates pos
	}
}

// ScrollActionPos moves the scrollbar in given dimension to given
// position and emits a ScrollSig signal.
func (ly *Layout) ScrollActionPos(d math32.Dims, pos float32) {
	if ly.HasScroll[d] && ly.Scrolls[d] != nil {
		sb := ly.Scrolls[d]
		sb.SetValueAction(pos)
		ly.NeedsRender()
	}
}

// ScrollToPos moves the scrollbar in given dimension to given
// position and DOES NOT emit a ScrollSig signal.
func (ly *Layout) ScrollToPos(d math32.Dims, pos float32) {
	if ly.HasScroll[d] && ly.Scrolls[d] != nil {
		sb := ly.Scrolls[d]
		sb.SetValueAction(pos)
		ly.NeedsRender()
	}
}

// ScrollDelta processes a scroll event.  If only one dimension is processed,
// and there is a non-zero in other, then the consumed dimension is reset to 0
// and the event is left unprocessed, so a higher level can consume the
// remainder.
func (ly *Layout) ScrollDelta(e events.Event) {
	se := e.(*events.MouseScroll)
	fdel := se.Delta

	hasShift := e.HasAnyModifier(key.Shift, key.Alt) // shift or alt indicates to scroll horizontally
	if hasShift {
		if !ly.HasScroll[math32.X] { // if we have shift, we can only horizontal scroll
			return
		}
		ly.ScrollActionDelta(math32.X, fdel.Y)
		return
	}

	if ly.HasScroll[math32.Y] && ly.HasScroll[math32.X] {
		ly.ScrollActionDelta(math32.Y, fdel.Y)
		ly.ScrollActionDelta(math32.X, fdel.X)
	} else if ly.HasScroll[math32.Y] {
		ly.ScrollActionDelta(math32.Y, fdel.Y)
		if se.Delta.X != 0 {
			se.Delta.Y = 0
		}
	} else if ly.HasScroll[math32.X] {
		if se.Delta.X != 0 {
			ly.ScrollActionDelta(math32.X, fdel.X)
			if se.Delta.Y != 0 {
				se.Delta.X = 0
			}
		}
	}
}

// ParentLayout returns the parent layout
func (wb *WidgetBase) ParentLayout() *Layout {
	ly := wb.ParentByType(LayoutType, tree.Embeds)
	if ly == nil {
		return nil
	}
	return AsLayout(ly)
}

// ParentScrollLayout returns the parent layout that has active scrollbars
func (wb *WidgetBase) ParentScrollLayout() *Layout {
	lyk := wb.ParentByType(LayoutType, tree.Embeds)
	if lyk == nil {
		return nil
	}
	ly := AsLayout(lyk)
	if ly.HasAnyScroll() {
		return ly
	}
	return ly.ParentScrollLayout()
}

// ScrollToMe tells this widget's parent layout to scroll to keep
// this widget in view. It returns whether any scrolling was done.
func (wb *WidgetBase) ScrollToMe() bool {
	ly := wb.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollToItem(wb.This().(Widget))
}

// ScrollToItem scrolls the layout to ensure that given item is in view.
// Returns true if scrolling was needed
func (ly *Layout) ScrollToItem(wi Widget) bool {
	// note: critical to NOT use BBox b/c it is zero for invisible items!
	return ly.ScrollToBox(wi.AsWidget().Geom.TotalRect())
}

// AutoScrollDim auto-scrolls along one dimension, based on the current
// position value, which is in the current scroll value range.
func (ly *Layout) AutoScrollDim(d math32.Dims, pos float32) bool {
	if !ly.HasScroll[d] || ly.Scrolls[d] == nil {
		return false
	}
	sb := ly.Scrolls[d]
	smax := sb.EffectiveMax()
	ssz := sb.ScrollThumbValue()
	dst := sb.Step * AutoScrollRate

	mind := max(0, (pos - sb.Value))
	maxd := max(0, (sb.Value+ssz)-pos)

	if mind <= maxd {
		pct := mind / ssz
		if pct < .1 && sb.Value > 0 {
			dst = min(dst, sb.Value)
			sb.SetValueAction(sb.Value - dst)
			return true
		}
	} else {
		pct := maxd / ssz
		if pct < .1 && sb.Value < smax {
			dst = min(dst, (smax - sb.Value))
			sb.SetValueAction(sb.Value + dst)
			return true
		}
	}
	return false
}

var LayoutLastAutoScroll time.Time

// AutoScroll scrolls the layout based on given position in scroll
// coordinates (i.e., already subtracing the BBox Min for a mouse event).
func (ly *Layout) AutoScroll(pos math32.Vector2) bool {
	now := time.Now()
	lag := now.Sub(LayoutLastAutoScroll)
	if lag < SystemSettings.LayoutAutoScrollDelay {
		return false
	}
	did := false
	if ly.HasScroll[math32.Y] && ly.HasScroll[math32.X] {
		did = ly.AutoScrollDim(math32.Y, pos.Y)
		did = did || ly.AutoScrollDim(math32.X, pos.X)
	} else if ly.HasScroll[math32.Y] {
		did = ly.AutoScrollDim(math32.Y, pos.Y)
	} else if ly.HasScroll[math32.X] {
		did = ly.AutoScrollDim(math32.X, pos.X)
	}
	if did {
		LayoutLastAutoScroll = time.Now()
	}
	return did
}

// ScrollToBoxDim scrolls to ensure that given target [min..max] range
// along one dimension is in view. Returns true if scrolling was needed
func (ly *Layout) ScrollToBoxDim(d math32.Dims, tmini, tmaxi int) bool {
	if !ly.HasScroll[d] || ly.Scrolls[d] == nil {
		return false
	}
	sb := ly.Scrolls[d]
	if sb == nil || sb.This() == nil {
		return false
	}
	tmin, tmax := float32(tmini), float32(tmaxi)
	cmin, cmax := ly.Geom.ContentRangeDim(d)
	if tmin >= cmin && tmax <= cmax {
		return false
	}
	h := ly.Styles.Font.Size.Dots
	if tmin < cmin { // favors scrolling to start
		trg := sb.Value + tmin - cmin - h
		if trg < 0 {
			trg = 0
		}
		sb.SetValueAction(trg)
		return true
	} else {
		if (tmax - tmin) < sb.ScrollThumbValue() { // only if whole thing fits
			trg := sb.Value + float32(tmax-cmax) + h
			sb.SetValueAction(trg)
			return true
		}
	}
	return false
}

// ScrollToBox scrolls the layout to ensure that given rect box is in view.
// Returns true if scrolling was needed
func (ly *Layout) ScrollToBox(box image.Rectangle) bool {
	did := false
	if ly.HasScroll[math32.Y] && ly.HasScroll[math32.X] {
		did = ly.ScrollToBoxDim(math32.Y, box.Min.Y, box.Max.Y)
		did = did || ly.ScrollToBoxDim(math32.X, box.Min.X, box.Max.X)
	} else if ly.HasScroll[math32.Y] {
		did = ly.ScrollToBoxDim(math32.Y, box.Min.Y, box.Max.Y)
	} else if ly.HasScroll[math32.X] {
		did = ly.ScrollToBoxDim(math32.X, box.Min.X, box.Max.X)
	}
	if did {
		ly.NeedsRender()
	}
	return did
}

// ScrollDimToStart scrolls to put the given child coordinate position (eg.,
// top / left of a view box) at the start (top / left) of our scroll area, to
// the extent possible. Returns true if scrolling was needed.
func (ly *Layout) ScrollDimToStart(d math32.Dims, posi int) bool {
	if !ly.HasScroll[d] {
		return false
	}
	pos := float32(posi)
	cmin, _ := ly.Geom.ContentRangeDim(d)
	if pos == cmin {
		return false
	}
	sb := ly.Scrolls[d]
	trg := math32.Clamp(sb.Value+(pos-cmin), 0, sb.EffectiveMax())
	sb.SetValueAction(trg)
	return true
}

// ScrollDimToEnd scrolls to put the given child coordinate position (eg.,
// bottom / right of a view box) at the end (bottom / right) of our scroll
// area, to the extent possible. Returns true if scrolling was needed.
func (ly *Layout) ScrollDimToEnd(d math32.Dims, posi int) bool {
	if !ly.HasScroll[d] || ly.Scrolls[d] == nil {
		return false
	}
	pos := float32(posi)
	_, cmax := ly.Geom.ContentRangeDim(d)
	if pos == cmax {
		return false
	}
	sb := ly.Scrolls[d]
	trg := math32.Clamp(sb.Value+(pos-cmax), 0, sb.EffectiveMax())
	sb.SetValueAction(trg)
	return true
}

// ScrollDimToContentEnd is a helper function that scrolls the layout to the
// end of its content (ie: moves the scrollbar to the very end).
func (ly *Layout) ScrollDimToContentEnd(d math32.Dims) bool {
	end := ly.Geom.Pos.Content.Dim(d) + ly.Geom.Size.Internal.Dim(d)
	return ly.ScrollDimToEnd(d, int(end))
}

// ScrollDimToCenter scrolls to put the given child coordinate position (eg.,
// middle of a view box) at the center of our scroll area, to the extent
// possible. Returns true if scrolling was needed.
func (ly *Layout) ScrollDimToCenter(d math32.Dims, posi int) bool {
	if !ly.HasScroll[d] || ly.Scrolls[d] == nil {
		return false
	}
	pos := float32(posi)
	cmin, cmax := ly.Geom.ContentRangeDim(d)
	mid := 0.5 * (cmin + cmax)
	if pos == mid {
		return false
	}
	sb := ly.Scrolls[d]
	trg := math32.Clamp(sb.Value+(pos-mid), 0, sb.EffectiveMax())
	sb.SetValueAction(trg)
	return true
}
