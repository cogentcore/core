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
	od := d.OtherDim()
	bbmin := mat32.NewVec2FmPoint(ly.Alloc.ContentBBox.Min)
	bbmax := mat32.NewVec2FmPoint(ly.Alloc.ContentBBox.Max)
	pos.SetDim(d, bbmin.Dim(d))
	pos.SetDim(od, bbmax.Dim(od))
	bbsz := bbmax.Sub(bbmin)
	sz.SetDim(d, bbsz.Dim(d)-4)
	sz.SetDim(od, ly.Styles.ScrollBarWidth.Dots)
	sz.SetCeil()
	return
}

// ConfigScrolls configures any scrollbars that have been enabled
// during the Layout process. This is called during Position, once
// the sizing and need for scrollbars has been established.
// The final position of the scrollbars is set during ScenePos in
// PositionScrolls.  Scrolls are kept around
func (ly *Layout) ConfigScrolls(sc *Scene) {
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.HasScroll[d] {
			ly.ConfigScroll(sc, d)
		}
	}
}

// ConfigScroll configures scroll for given dimension
func (ly *Layout) ConfigScroll(sc *Scene, d mat32.Dims) {
	if ly.Scrolls[d] != nil {
		return
	}
	ly.Scrolls[d] = &Slider{}
	sb := ly.Scrolls[d]
	sb.InitName(sb, fmt.Sprintf("scroll%v", d))
	ki.SetParent(sb, ly.This())
	// sr.SetFlag(true, ki.Field) // note: do not turn on -- breaks pos
	sb.SetType(SliderScrollbar)
	sb.Sc = sc
	sb.Dim = d
	sb.Tracking = true
	sb.TrackThr = 1
	sb.Min = 0.0
	sb.Style(func(s *styles.Style) {
		s.Padding.Zero()
		s.Margin.Zero()
		s.MaxBorder.Width.Zero()
		s.Border.Width.Zero()
		od := d.OtherDim()
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
	sb.OnChange(func(e events.Event) {
		e.SetHandled()
		// fmt.Println("change event")
		updt := ly.UpdateStart()
		ly.ScenePos(ly.Sc) // gets pos from scrolls, positions scrollbars
		ly.UpdateEndLayout(updt)
	})
	sb.Update()
}

// GetScrollPosition sets our layout Scroll position from scrollbars
func (ly *Layout) GetScrollPosition(sc *Scene) {
	for d := mat32.X; d <= mat32.Y; d++ {
		ly.Alloc.Scroll.SetDim(d, 0)
		if ly.HasScroll[d] {
			sb := ly.Scrolls[d]
			ly.Alloc.Scroll.SetDim(d, -sb.Value)
		}
	}
}

// PositionScrolls arranges scrollbars
func (ly *Layout) PositionScrolls(sc *Scene) {
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.HasScroll[d] {
			ly.PositionScroll(sc, d)
		}
	}
}

func (ly *Layout) PositionScroll(sc *Scene, d mat32.Dims) {
	sb := ly.Scrolls[d]
	pos, sz := ly.ScrollGeom(d)
	if sb.Alloc.Pos == pos && sb.Alloc.Size.Content == sz {
		return
	}
	if sz.X <= 0 || sz.Y <= 0 {
		sb.SetState(true, states.Invisible)
		return
	}
	sb.SetState(false, states.Invisible)
	csz := ly.LayImpl.ContentSubGap.Dim(d)
	ksz := ly.LayImpl.KidsSize.Dim(d)
	sb.Max = ksz                       // only scrollbar
	sb.Step = ly.Styles.Font.Size.Dots // step by lines
	sb.PageStep = 10.0 * sb.Step       // todo: more dynamic
	sb.SetVisiblePct(csz / ksz)
	sb.SetValue(sb.Value) // keep in range

	sb.Update() // applies style
	sb.SizeUp(sc)
	sb.Alloc.Size.Alloc = ly.Alloc.Size.Content
	sb.SizeDown(sc, 0)

	sb.Alloc.Pos = pos
	sb.SetContentPosFromPos()
	// note: usually these are intersected with parent *content* bbox,
	// but scrolls are specifically outside of that.
	sb.SetBBoxesFromAllocs()
}

// RenderScrolls draws the scrollbars
func (ly *Layout) RenderScrolls(sc *Scene) {
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.HasScroll[d] {
			ly.Scrolls[d].Render(sc)
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
func (ly *Layout) ScrollActionDelta(dim mat32.Dims, delta float32) {
	if ly.HasScroll[dim] {
		sb := ly.Scrolls[dim]
		nval := sb.Value + delta
		sb.SetValue(nval)
		ly.SetNeedsLayout()
	}
}

// ScrollActionPos moves the scrollbar in given dimension to given
// position and emits a ScrollSig signal.
func (ly *Layout) ScrollActionPos(dim mat32.Dims, pos float32) {
	if ly.HasScroll[dim] {
		sb := ly.Scrolls[dim]
		sb.SetValue(pos)
		ly.SetNeedsLayout()
	}
}

// ScrollToPos moves the scrollbar in given dimension to given
// position and DOES NOT emit a ScrollSig signal.
func (ly *Layout) ScrollToPos(dim mat32.Dims, pos float32) {
	if ly.HasScroll[dim] {
		sb := ly.Scrolls[dim]
		sb.SetValue(pos)
		ly.SetNeedsLayout()
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

// AutoScrollDim auto-scrolls along one dimension
func (ly *Layout) AutoScrollDim(dim mat32.Dims, st, pos int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
	/*
		sc := ly.Scrolls[dim]
		scrange := sc.Max - sc.ThumbVal // amount that can be scrolled
		vissz := sc.ThumbVal            // amount visible

		h := ly.Styles.Font.Size.Dots
		dst := h * AutoScrollRate

		mind := max(0, pos-st)
		maxd := max(0, (st+int(vissz))-pos)

		if mind <= maxd {
			pct := float32(mind) / float32(vissz)
			if pct < .1 && sc.Value > 0 {
				dst = mat32.Min(dst, sc.Value)
				sc.SetValueAction(sc.Value - dst)
				return true
			}
		} else {
			pct := float32(maxd) / float32(vissz)
			if pct < .1 && sc.Value < scrange {
				dst = mat32.Min(dst, (scrange - sc.Value))
				ly.ScrollActionDelta(dim, dst)
				return true
			}
		}
	*/
	return false
}

var LayoutLastAutoScroll time.Time

// AutoScroll scrolls the layout based on mouse position, when appropriate (DND, menus)
func (ly *Layout) AutoScroll(pos image.Point) bool {
	now := time.Now()
	lagMs := int(now.Sub(LayoutLastAutoScroll) / time.Millisecond)
	if lagMs < LayoutAutoScrollDelayMSec {
		return false
	}
	ly.BBoxMu.RLock()
	wbb := ly.Alloc.BBox
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

// ScrollToBoxDim scrolls to ensure that given rect box along one dimension is
// in view -- returns true if scrolling was needed
func (ly *Layout) ScrollToBoxDim(dim mat32.Dims, minBox, maxBox int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
	/*
		vpMin := ly.Alloc.BBox.Min.X
		if dim == mat32.Y {
			vpMin = ly.Alloc.BBox.Min.Y
		}
		sc := ly.Scrolls[dim]
		scrange := sc.Max - sc.ThumbVal // amount that can be scrolled
		vissz := sc.ThumbVal            // amount visible
		vpMax := vpMin + int(vissz)

		if minBox >= vpMin && maxBox <= vpMax {
			return false
		}

		h := ly.Styles.Font.Size.Dots

		if minBox < vpMin { // favors scrolling to start
			trg := sc.Value + float32(minBox-vpMin) - h
			if trg < 0 {
				trg = 0
			}
			sc.SetValueAction(trg)
			return true
		} else {
			if (maxBox - minBox) < int(vissz) {
				trg := sc.Value + float32(maxBox-vpMax) + h
				if trg > scrange {
					trg = scrange
				}
				sc.SetValueAction(trg)
				return true
			}
		}
	*/
	return false
}

// ScrollToBox scrolls the layout to ensure that given rect box is in view --
// returns true if scrolling was needed
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
	return did
}

// ScrollToItem scrolls the layout to ensure that given item is in view --
// returns true if scrolling was needed
func (ly *Layout) ScrollToItem(wi Widget) bool {
	// return ly.ScrollToBox(wi.AsWidget().ObjBBox)
	return false
}

// ScrollDimToStart scrolls to put the given child coordinate position (eg.,
// top / left of a view box) at the start (top / left) of our scroll area, to
// the extent possible -- returns true if scrolling was needed.
func (ly *Layout) ScrollDimToStart(dim mat32.Dims, pos int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
	/*
		vpMin := ly.Alloc.BBox.Min.X
		if dim == mat32.Y {
			vpMin = ly.Alloc.BBox.Min.Y
		}
		sc := ly.Scrolls[dim]
		if pos == vpMin { // already at min
			return false
		}
		scrange := sc.Max - sc.ThumbVal // amount that can be scrolled

		trg := sc.Value + float32(pos-vpMin)
		if trg < 0 {
			trg = 0
		} else if trg > scrange {
			trg = scrange
		}
		if sc.Value == trg {
			return false
		}
		sc.SetValueAction(trg)
	*/
	return true
}

// ScrollDimToEnd scrolls to put the given child coordinate position (eg.,
// bottom / right of a view box) at the end (bottom / right) of our scroll
// area, to the extent possible -- returns true if scrolling was needed.
func (ly *Layout) ScrollDimToEnd(dim mat32.Dims, pos int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
	/*
		vpMin := ly.Alloc.BBox.Min.X
		if dim == mat32.Y {
			vpMin = ly.Alloc.BBox.Min.Y
		}
		sc := ly.Scrolls[dim]
		scrange := sc.Max - sc.ThumbVal // amount that can be scrolled
		vissz := (sc.ThumbVal)          // todo: - ly.ExtraSize.Dim(dim)) // amount visible
		vpMax := vpMin + int(vissz)
		if pos == vpMax { // already at max
			return false
		}
		trg := sc.Value + float32(pos-vpMax)
		if trg < 0 {
			trg = 0
		} else if trg > scrange {
			trg = scrange
		}
		if sc.Value == trg {
			return false
		}
		sc.SetValueAction(trg)
	*/
	return true
}

// ScrollDimToCenter scrolls to put the given child coordinate position (eg.,
// middle of a view box) at the center of our scroll area, to the extent
// possible -- returns true if scrolling was needed.
func (ly *Layout) ScrollDimToCenter(dim mat32.Dims, pos int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
	/*
		vpMin := ly.Alloc.BBox.Min.X
		if dim == mat32.Y {
			vpMin = ly.Alloc.BBox.Min.Y
		}
		sc := ly.Scrolls[dim]
		scrange := sc.Max - sc.ThumbVal // amount that can be scrolled
		vissz := sc.ThumbVal            // amount visible
		vpMid := vpMin + int(0.5*vissz)
		if pos == vpMid { // already at mid
			return false
		}
		trg := sc.Value + float32(pos-vpMid)
		if trg < 0 {
			trg = 0
		} else if trg > scrange {
			trg = scrange
		}
		if sc.Value == trg {
			return false
		}
		sc.SetValueAction(trg)
	*/
	return true
}
