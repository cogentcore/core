// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// HasAnyScroll returns true if layout has
func (l *Layout) HasAnyScroll() bool {
	return l.HasScroll[mat32.X] || l.HasScroll[mat32.Y]
}

// ScrollGeom returns the target position and size for scrollbars
func (l *Layout) ScrollGeom(d mat32.Dims) (pos, sz mat32.Vec2) {
	sbw := mat32.Ceil(l.Styles.ScrollBarWidth.Dots)
	od := d.Other()
	bbmin := mat32.V2FromPoint(l.Geom.ContentBBox.Min)
	bbmax := mat32.V2FromPoint(l.Geom.ContentBBox.Max)
	if l.This() != l.Scene.This() { // if not the scene, keep inside the scene
		bbmin.SetMax(mat32.V2FromPoint(l.Scene.Geom.ContentBBox.Min))
		bbmax.SetMin(mat32.V2FromPoint(l.Scene.Geom.ContentBBox.Max).SubScalar(sbw))
	}
	pos.SetDim(d, bbmin.Dim(d))
	pos.SetDim(od, bbmax.Dim(od))
	bbsz := bbmax.Sub(bbmin)
	sz.SetDim(d, bbsz.Dim(d)-4)
	sz.SetDim(od, sbw)
	sz.SetCeil()
	return
}

// ConfigScrolls configures any scrollbars that have been enabled
// during the Layout process. This is called during Position, once
// the sizing and need for scrollbars has been established.
// The final position of the scrollbars is set during ScenePos in
// PositionScrolls.  Scrolls are kept around in general.
func (l *Layout) ConfigScrolls() {
	for d := mat32.X; d <= mat32.Y; d++ {
		if l.HasScroll[d] {
			l.ConfigScroll(d)
		}
	}
}

// ConfigScroll configures scroll for given dimension
func (l *Layout) ConfigScroll(d mat32.Dims) {
	if l.Scrolls[d] != nil {
		return
	}
	l.Scrolls[d] = &Slider{}
	sb := l.Scrolls[d]
	sb.InitName(sb, fmt.Sprintf("scroll%v", d))
	ki.SetParent(sb, l.This())
	// sr.SetFlag(true, ki.Field) // note: do not turn on -- breaks pos
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
		_, sz := l.This().(Layouter).ScrollGeom(d)
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
		l.This().(Layouter).ScrollChanged(d, sb)
	})
	sb.Update()
}

// ScrollChanged is called in the OnInput event handler for updating,
// when the scrollbar value has changed, for given dimension.
// This is part of the Layouter interface.
func (l *Layout) ScrollChanged(d mat32.Dims, sb *Slider) {
	updt := l.UpdateStart()
	l.Geom.Scroll.SetDim(d, -sb.Value)
	l.This().(Layouter).ScenePos() // computes updated positions
	l.UpdateEndRender(updt)
}

// ScrollValues returns the maximum size that could be scrolled,
// the visible size (which could be less than the max size, in which
// case no scrollbar is needed), and visSize / maxSize as the VisiblePct.
// This is used in updating the scrollbar and determining whether one is
// needed in the first place
func (l *Layout) ScrollValues(d mat32.Dims) (maxSize, visSize, visPct float32) {
	sz := &l.Geom.Size
	maxSize = sz.Internal.Dim(d)
	visSize = sz.Alloc.Content.Dim(d)
	visPct = visSize / maxSize
	return
}

// SetScrollParams sets scrollbar parameters.  Must set Step and PageStep,
// but can also set others as needed.
// Max and VisiblePct are automatically set based on ScrollValues maxSize, visPct.
func (l *Layout) SetScrollParams(d mat32.Dims, sb *Slider) {
	sb.Step = l.Styles.Font.Size.Dots // step by lines
	sb.PageStep = 10.0 * sb.Step      // todo: more dynamic
}

// PositionScrolls arranges scrollbars
func (l *Layout) PositionScrolls() {
	for d := mat32.X; d <= mat32.Y; d++ {
		if l.HasScroll[d] && l.Scrolls[d] != nil {
			l.PositionScroll(d)
		}
	}
}

func (l *Layout) PositionScroll(d mat32.Dims) {
	sb := l.Scrolls[d]
	pos, ssz := l.This().(Layouter).ScrollGeom(d)
	maxSize, _, visPct := l.This().(Layouter).ScrollValues(d)
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
	// fmt.Println(l, d, "vis pct:", asz/csz)
	sb.SetValue(sb.Value) // keep in range
	l.This().(Layouter).SetScrollParams(d, sb)

	sb.Update() // applies style
	sb.SizeUp()
	sb.Geom.Size.Alloc = l.Geom.Size.Actual
	sb.SizeDown(0)

	sb.Geom.Pos.Total = pos
	sb.SetContentPosFromPos()
	// note: usually these are intersected with parent *content* bbox,
	// but scrolls are specifically outside of that.
	sb.SetBBoxesFromAllocs()
}

// RenderScrolls draws the scrollbars
func (l *Layout) RenderScrolls() {
	for d := mat32.X; d <= mat32.Y; d++ {
		if l.HasScroll[d] && l.Scrolls[d] != nil {
			l.Scrolls[d].Render()
		}
	}
}

// SetScrollsOff turns off the scrollbars
func (l *Layout) SetScrollsOff() {
	for d := mat32.X; d <= mat32.Y; d++ {
		l.HasScroll[d] = false
	}
}

// ScrollActionDelta moves the scrollbar in given dimension by given delta
// and emits a ScrollSig signal.
func (l *Layout) ScrollActionDelta(d mat32.Dims, delta float32) {
	if l.HasScroll[d] && l.Scrolls[d] != nil {
		sb := l.Scrolls[d]
		nval := sb.Value + sb.ScrollScale(delta)
		sb.SetValueAction(nval)
		l.SetNeedsRender(true) // only render needed -- scroll updates pos
	}
}

// ScrollActionPos moves the scrollbar in given dimension to given
// position and emits a ScrollSig signal.
func (l *Layout) ScrollActionPos(d mat32.Dims, pos float32) {
	if l.HasScroll[d] && l.Scrolls[d] != nil {
		sb := l.Scrolls[d]
		sb.SetValueAction(pos)
		l.SetNeedsRender(true)
	}
}

// ScrollToPos moves the scrollbar in given dimension to given
// position and DOES NOT emit a ScrollSig signal.
func (l *Layout) ScrollToPos(d mat32.Dims, pos float32) {
	if l.HasScroll[d] && l.Scrolls[d] != nil {
		sb := l.Scrolls[d]
		sb.SetValueAction(pos)
		l.SetNeedsRender(true)
	}
}

// ScrollDelta processes a scroll event.  If only one dimension is processed,
// and there is a non-zero in other, then the consumed dimension is reset to 0
// and the event is left unprocessed, so a higher level can consume the
// remainder.
func (l *Layout) ScrollDelta(e events.Event) {
	se := e.(*events.MouseScroll)
	fdel := se.Delta

	hasShift := e.HasAnyModifier(key.Shift, key.Alt) // shift or alt indicates to scroll horizontally
	if hasShift {
		if !l.HasScroll[mat32.X] { // if we have shift, we can only horizontal scroll
			return
		}
		l.ScrollActionDelta(mat32.X, fdel.Y)
		return
	}

	if l.HasScroll[mat32.Y] && l.HasScroll[mat32.X] {
		l.ScrollActionDelta(mat32.Y, fdel.Y)
		l.ScrollActionDelta(mat32.X, fdel.X)
	} else if l.HasScroll[mat32.Y] {
		l.ScrollActionDelta(mat32.Y, fdel.Y)
		if se.Delta.X != 0 {
			se.Delta.Y = 0
		}
	} else if l.HasScroll[mat32.X] {
		if se.Delta.X != 0 {
			l.ScrollActionDelta(mat32.X, fdel.X)
			if se.Delta.Y != 0 {
				se.Delta.X = 0
			}
		}
	}
}

// ParentLayout returns the parent layout
func (wb *WidgetBase) ParentLayout() *Layout {
	ly := wb.ParentByType(LayoutType, ki.Embeds)
	if ly == nil {
		return nil
	}
	return AsLayout(ly)
}

// ParentScrollLayout returns the parent layout that has active scrollbars
func (wb *WidgetBase) ParentScrollLayout() *Layout {
	lyk := wb.ParentByType(LayoutType, ki.Embeds)
	if lyk == nil {
		return nil
	}
	ly := AsLayout(lyk)
	if ly.HasAnyScroll() {
		return ly
	}
	return ly.ParentScrollLayout()
}

// ScrollToMe tells my parent layout (that has scroll bars) to scroll to keep
// this widget in view -- returns true if scrolled
func (wb *WidgetBase) ScrollToMe() bool {
	ly := wb.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollToItem(wb.This().(Widget))
}

// ScrollToItem scrolls the layout to ensure that given item is in view.
// Returns true if scrolling was needed
func (l *Layout) ScrollToItem(wi Widget) bool {
	// note: critical to NOT use BBox b/c it is zero for invisible items!
	return l.ScrollToBox(wi.AsWidget().Geom.TotalRect())
}

// AutoScrollDim auto-scrolls along one dimension
func (l *Layout) AutoScrollDim(d mat32.Dims, sti, posi int) bool {
	if !l.HasScroll[d] || l.Scrolls[d] == nil {
		return false
	}
	sb := l.Scrolls[d]
	smax := sb.EffectiveMax()
	ssz := sb.ScrollThumbValue()
	dst := l.Styles.Font.Size.Dots * AutoScrollRate

	st, pos := float32(sti), float32(posi)
	mind := max(0, pos-st)
	maxd := max(0, (st+ssz)-pos)

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
			l.ScrollActionDelta(d, dst)
			return true
		}
	}
	return false
}

var LayoutLastAutoScroll time.Time

// AutoScroll scrolls the layout based on mouse position, when appropriate (DND, menus)
func (l *Layout) AutoScroll(pos image.Point) bool {
	now := time.Now()
	lag := now.Sub(LayoutLastAutoScroll)
	if lag < SystemSettings.LayoutAutoScrollDelay {
		return false
	}
	l.BBoxMu.RLock()
	wbb := l.Geom.ContentBBox
	l.BBoxMu.RUnlock()
	did := false
	if l.HasScroll[mat32.Y] && l.HasScroll[mat32.X] {
		did = l.AutoScrollDim(mat32.Y, wbb.Min.Y, pos.Y)
		did = did || l.AutoScrollDim(mat32.X, wbb.Min.X, pos.X)
	} else if l.HasScroll[mat32.Y] {
		did = l.AutoScrollDim(mat32.Y, wbb.Min.Y, pos.Y)
	} else if l.HasScroll[mat32.X] {
		did = l.AutoScrollDim(mat32.X, wbb.Min.X, pos.X)
	}
	if did {
		LayoutLastAutoScroll = time.Now()
	}
	return did
}

// ScrollToBoxDim scrolls to ensure that given target [min..max] range
// along one dimension is in view. Returns true if scrolling was needed
func (l *Layout) ScrollToBoxDim(d mat32.Dims, tmini, tmaxi int) bool {
	if !l.HasScroll[d] || l.Scrolls[d] == nil {
		return false
	}
	sb := l.Scrolls[d]
	if sb == nil || sb.This() == nil {
		return false
	}
	tmin, tmax := float32(tmini), float32(tmaxi)
	cmin, cmax := l.Geom.ContentRangeDim(d)
	if tmin >= cmin && tmax <= cmax {
		return false
	}
	h := l.Styles.Font.Size.Dots
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
func (l *Layout) ScrollToBox(box image.Rectangle) bool {
	did := false
	if l.HasScroll[mat32.Y] && l.HasScroll[mat32.X] {
		did = l.ScrollToBoxDim(mat32.Y, box.Min.Y, box.Max.Y)
		did = did || l.ScrollToBoxDim(mat32.X, box.Min.X, box.Max.X)
	} else if l.HasScroll[mat32.Y] {
		did = l.ScrollToBoxDim(mat32.Y, box.Min.Y, box.Max.Y)
	} else if l.HasScroll[mat32.X] {
		did = l.ScrollToBoxDim(mat32.X, box.Min.X, box.Max.X)
	}
	if did {
		l.SetNeedsRender(true)
	}
	return did
}

// ScrollDimToStart scrolls to put the given child coordinate position (eg.,
// top / left of a view box) at the start (top / left) of our scroll area, to
// the extent possible. Returns true if scrolling was needed.
func (l *Layout) ScrollDimToStart(d mat32.Dims, posi int) bool {
	if !l.HasScroll[d] {
		return false
	}
	pos := float32(posi)
	cmin, _ := l.Geom.ContentRangeDim(d)
	if pos == cmin {
		return false
	}
	sb := l.Scrolls[d]
	trg := mat32.Clamp(sb.Value+(pos-cmin), 0, sb.EffectiveMax())
	sb.SetValueAction(trg)
	return true
}

// ScrollDimToEnd scrolls to put the given child coordinate position (eg.,
// bottom / right of a view box) at the end (bottom / right) of our scroll
// area, to the extent possible. Returns true if scrolling was needed.
func (l *Layout) ScrollDimToEnd(d mat32.Dims, posi int) bool {
	if !l.HasScroll[d] || l.Scrolls[d] == nil {
		return false
	}
	pos := float32(posi)
	_, cmax := l.Geom.ContentRangeDim(d)
	if pos == cmax {
		return false
	}
	sb := l.Scrolls[d]
	trg := mat32.Clamp(sb.Value+(pos-cmax), 0, sb.EffectiveMax())
	sb.SetValueAction(trg)
	return true
}

// ScrollDimToCenter scrolls to put the given child coordinate position (eg.,
// middle of a view box) at the center of our scroll area, to the extent
// possible. Returns true if scrolling was needed.
func (l *Layout) ScrollDimToCenter(d mat32.Dims, posi int) bool {
	if !l.HasScroll[d] || l.Scrolls[d] == nil {
		return false
	}
	pos := float32(posi)
	cmin, cmax := l.Geom.ContentRangeDim(d)
	mid := 0.5 * (cmin + cmax)
	if pos == mid {
		return false
	}
	sb := l.Scrolls[d]
	trg := mat32.Clamp(sb.Value+(pos-mid), 0, sb.EffectiveMax())
	sb.SetValueAction(trg)
	return true
}
