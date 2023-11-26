// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"time"

	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// HasAnyScroll returns true if layout has
func (ly *Layout) HasAnyScroll() bool {
	return ly.HasScroll[mat32.X] || ly.HasScroll[mat32.Y]
}

// ScrollGeom returns the target position and size for scrollbars
func (ly *Layout) ScrollGeom(d mat32.Dims) (pos, sz mat32.Vec2) {
	sbw := mat32.Ceil(ly.Styles.ScrollBarWidth.Dots)
	od := d.Other()
	bbmin := mat32.NewVec2FmPoint(ly.Geom.ContentBBox.Min)
	bbmax := mat32.NewVec2FmPoint(ly.Geom.ContentBBox.Max)
	if ly.This() != ly.Sc.This() { // if not the scene, keep inside the scene
		bbmin.SetMax(mat32.NewVec2FmPoint(ly.Sc.Geom.ContentBBox.Min))
		bbmax.SetMin(mat32.NewVec2FmPoint(ly.Sc.Geom.ContentBBox.Max).SubScalar(sbw))
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
func (ly *Layout) ConfigScrolls() {
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.HasScroll[d] {
			ly.ConfigScroll(d)
		}
	}
}

// ConfigScroll configures scroll for given dimension
func (ly *Layout) ConfigScroll(d mat32.Dims) {
	if ly.Scrolls[d] != nil {
		return
	}
	ly.Scrolls[d] = &Slider{}
	sb := ly.Scrolls[d]
	sb.InitName(sb, fmt.Sprintf("scroll%v", d))
	ki.SetParent(sb, ly.This())
	// sr.SetFlag(true, ki.Field) // note: do not turn on -- breaks pos
	sb.SetType(SliderScrollbar)
	sb.Dim = d
	sb.InputThreshold = 1
	sb.Min = 0.0
	sb.Style(func(s *styles.Style) {
		s.Padding.Zero()
		s.Margin.Zero()
		s.MaxBorder.Width.Zero()
		s.Border.Width.Zero()
		od := d.Other()
		_, sz := ly.ScrollGeom(d)
		if sz.X > 0 && sz.Y > 0 {
			s.State.SetFlag(false, states.Invisible)
			s.Min.SetDim(d, units.Dot(sz.Dim(d)))
			s.Min.SetDim(od, units.Dot(sz.Dim(od)))
		} else {
			s.State.SetFlag(true, states.Invisible)
		}
		s.Max = s.Min
	})
	sb.OnInput(func(e events.Event) {
		e.SetHandled()
		// fmt.Println("change event")
		updt := ly.UpdateStart()
		ly.This().(Widget).ScenePos() // gets pos from scrolls, positions scrollbars
		ly.UpdateEndRender(updt)
	})
	sb.Update()
}

// GetScrollPosition sets our layout Scroll position from scrollbars
func (ly *Layout) GetScrollPosition() {
	for d := mat32.X; d <= mat32.Y; d++ {
		ly.Geom.Scroll.SetDim(d, 0)
		if ly.HasScroll[d] {
			sb := ly.Scrolls[d]
			if sb == nil {
				ly.HasScroll[d] = false
				continue
			}
			ly.Geom.Scroll.SetDim(d, -sb.Value)
		}
	}
}

// PositionScrolls arranges scrollbars
func (ly *Layout) PositionScrolls() {
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.HasScroll[d] {
			ly.PositionScroll(d)
		}
	}
}

func (ly *Layout) PositionScroll(d mat32.Dims) {
	sb := ly.Scrolls[d]
	sz := &ly.Geom.Size
	pos, ssz := ly.ScrollGeom(d)
	asz := sz.Alloc.Content.Dim(d)
	csz := sz.Internal.Dim(d)
	vis := asz / csz
	if sb.Geom.Pos.Total == pos && sb.Geom.Size.Actual.Content == ssz && sb.VisiblePct == vis {
		return
	}
	if ssz.X <= 0 || ssz.Y <= 0 {
		sb.SetState(true, states.Invisible)
		return
	}
	sb.SetState(false, states.Invisible)
	sb.Max = csz                       // only scrollbar
	sb.Step = ly.Styles.Font.Size.Dots // step by lines
	sb.PageStep = 10.0 * sb.Step       // todo: more dynamic
	sb.SetVisiblePct(vis)
	// fmt.Println(ly, d, "vis pct:", asz/csz)
	sb.SetValue(sb.Value) // keep in range

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
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.HasScroll[d] {
			ly.Scrolls[d].Render()
		}
	}
}

// SetScrollsOff turns off the scrollbars
func (ly *Layout) SetScrollsOff() {
	for d := mat32.X; d <= mat32.Y; d++ {
		ly.HasScroll[d] = false
	}
}

// ScrollActionDelta moves the scrollbar in given dimension by given delta
// and emits a ScrollSig signal.
func (ly *Layout) ScrollActionDelta(d mat32.Dims, delta float32) {
	if ly.HasScroll[d] {
		sb := ly.Scrolls[d]
		nval := sb.Value + delta
		sb.SetValueAction(nval)
		ly.SetNeedsRender(true) // only render needed -- scroll updates pos
	}
}

// ScrollActionPos moves the scrollbar in given dimension to given
// position and emits a ScrollSig signal.
func (ly *Layout) ScrollActionPos(d mat32.Dims, pos float32) {
	if ly.HasScroll[d] {
		sb := ly.Scrolls[d]
		sb.SetValueAction(pos)
		ly.SetNeedsRender(true)
	}
}

// ScrollToPos moves the scrollbar in given dimension to given
// position and DOES NOT emit a ScrollSig signal.
func (ly *Layout) ScrollToPos(d mat32.Dims, pos float32) {
	if ly.HasScroll[d] {
		sb := ly.Scrolls[d]
		sb.SetValueAction(pos)
		ly.SetNeedsRender(true)
	}
}

// ScrollDelta processes a scroll event.  If only one dimension is processed,
// and there is a non-zero in other, then the consumed dimension is reset to 0
// and the event is left unprocessed, so a higher level can consume the
// remainder.
func (ly *Layout) ScrollDelta(e events.Event) {
	se := e.(*events.MouseScroll)
	var del image.Point
	del.X = se.DimDelta(mat32.X)
	del.Y = se.DimDelta(mat32.Y)
	fdel := mat32.NewVec2FmPoint(del)

	hasShift := e.HasAnyModifier(key.Shift, key.Alt) // shift or alt indicates to scroll horizontally
	if hasShift {
		if !ly.HasScroll[mat32.X] { // if we have shift, we can only horizontal scroll
			e.SetHandled()
			return
		}
		ly.ScrollActionDelta(mat32.X, fdel.Y)
		e.SetHandled()
		return
	}

	if ly.HasScroll[mat32.Y] && ly.HasScroll[mat32.X] {
		// fmt.Printf("ly: %v both del: %v\n", ly.Nm, del)
		ly.ScrollActionDelta(mat32.Y, fdel.Y)
		ly.ScrollActionDelta(mat32.X, fdel.X)
		e.SetHandled()
	} else if ly.HasScroll[mat32.Y] {
		// fmt.Printf("ly: %v y del: %v\n", ly.Nm, del)
		ly.ScrollActionDelta(mat32.Y, fdel.Y)
		if del.X != 0 {
			se.Delta.Y = 0
		} else {
			e.SetHandled()
		}
	} else if ly.HasScroll[mat32.X] {
		// fmt.Printf("ly: %v x del: %v\n", ly.Nm, del)
		if del.X != 0 {
			ly.ScrollActionDelta(mat32.X, fdel.X)
			if del.Y != 0 {
				se.Delta.X = 0
			} else {
				e.SetHandled()
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
func (ly *Layout) ScrollToItem(wi Widget) bool {
	// note: critical to NOT use BBox b/c it is zero for invisible items!
	return ly.ScrollToBox(wi.AsWidget().Geom.TotalRect())
}

// AutoScrollDim auto-scrolls along one dimension
func (ly *Layout) AutoScrollDim(d mat32.Dims, sti, posi int) bool {
	if !ly.HasScroll[d] {
		return false
	}
	sb := ly.Scrolls[d]
	smax := sb.EffectiveMax()
	ssz := sb.ScrollThumbValue()
	dst := ly.Styles.Font.Size.Dots * AutoScrollRate

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
			ly.ScrollActionDelta(d, dst)
			return true
		}
	}
	return false
}

var LayoutLastAutoScroll time.Time

// AutoScroll scrolls the layout based on mouse position, when appropriate (DND, menus)
func (ly *Layout) AutoScroll(pos image.Point) bool {
	now := time.Now()
	lag := now.Sub(LayoutLastAutoScroll)
	if lag < LayoutAutoScrollDelay {
		return false
	}
	ly.BBoxMu.RLock()
	wbb := ly.Geom.ContentBBox
	ly.BBoxMu.RUnlock()
	did := false
	if ly.HasScroll[mat32.Y] && ly.HasScroll[mat32.X] {
		did = ly.AutoScrollDim(mat32.Y, wbb.Min.Y, pos.Y)
		did = did || ly.AutoScrollDim(mat32.X, wbb.Min.X, pos.X)
	} else if ly.HasScroll[mat32.Y] {
		did = ly.AutoScrollDim(mat32.Y, wbb.Min.Y, pos.Y)
	} else if ly.HasScroll[mat32.X] {
		did = ly.AutoScrollDim(mat32.X, wbb.Min.X, pos.X)
	}
	if did {
		LayoutLastAutoScroll = time.Now()
	}
	return did
}

// ScrollToBoxDim scrolls to ensure that given target [min..max] range
// along one dimension is in view. Returns true if scrolling was needed
func (ly *Layout) ScrollToBoxDim(d mat32.Dims, tmini, tmaxi int) bool {
	if !ly.HasScroll[d] {
		return false
	}
	sb := ly.Scrolls[d]
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
	if ly.HasScroll[mat32.Y] && ly.HasScroll[mat32.X] {
		did = ly.ScrollToBoxDim(mat32.Y, box.Min.Y, box.Max.Y)
		did = did || ly.ScrollToBoxDim(mat32.X, box.Min.X, box.Max.X)
	} else if ly.HasScroll[mat32.Y] {
		did = ly.ScrollToBoxDim(mat32.Y, box.Min.Y, box.Max.Y)
	} else if ly.HasScroll[mat32.X] {
		did = ly.ScrollToBoxDim(mat32.X, box.Min.X, box.Max.X)
	}
	if did {
		ly.SetNeedsRender(true)
	}
	return did
}

// ScrollDimToStart scrolls to put the given child coordinate position (eg.,
// top / left of a view box) at the start (top / left) of our scroll area, to
// the extent possible. Returns true if scrolling was needed.
func (ly *Layout) ScrollDimToStart(d mat32.Dims, posi int) bool {
	if !ly.HasScroll[d] {
		return false
	}
	pos := float32(posi)
	cmin, _ := ly.Geom.ContentRangeDim(d)
	if pos == cmin {
		return false
	}
	sb := ly.Scrolls[d]
	trg := mat32.Clamp(sb.Value+(pos-cmin), 0, sb.EffectiveMax())
	sb.SetValueAction(trg)
	return true
}

// ScrollDimToEnd scrolls to put the given child coordinate position (eg.,
// bottom / right of a view box) at the end (bottom / right) of our scroll
// area, to the extent possible. Returns true if scrolling was needed.
func (ly *Layout) ScrollDimToEnd(d mat32.Dims, posi int) bool {
	if !ly.HasScroll[d] {
		return false
	}
	pos := float32(posi)
	_, cmax := ly.Geom.ContentRangeDim(d)
	if pos == cmax {
		return false
	}
	sb := ly.Scrolls[d]
	trg := mat32.Clamp(sb.Value+(pos-cmax), 0, sb.EffectiveMax())
	sb.SetValueAction(trg)
	return true
}

// ScrollDimToCenter scrolls to put the given child coordinate position (eg.,
// middle of a view box) at the center of our scroll area, to the extent
// possible. Returns true if scrolling was needed.
func (ly *Layout) ScrollDimToCenter(d mat32.Dims, posi int) bool {
	if !ly.HasScroll[d] {
		return false
	}
	pos := float32(posi)
	cmin, cmax := ly.Geom.ContentRangeDim(d)
	mid := 0.5 * (cmin + cmax)
	if pos == mid {
		return false
	}
	sb := ly.Scrolls[d]
	trg := mat32.Clamp(sb.Value+(pos-mid), 0, sb.EffectiveMax())
	sb.SetValueAction(trg)
	return true
}
