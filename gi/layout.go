// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"strings"
	"time"
	"unicode"

	"goki.dev/enums"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

var (
	// LayoutPrefMaxRows is maximum number of rows to use in a grid layout
	// when computing the preferred size (ScPrefSizing)
	LayoutPrefMaxRows = 20

	// LayoutPrefMaxCols is maximum number of columns to use in a grid layout
	// when computing the preferred size (ScPrefSizing)
	LayoutPrefMaxCols = 20

	// LayoutFocusNameTimeoutMSec is the number of milliseconds between keypresses
	// to combine characters into name to search for within layout -- starts over
	// after this delay.
	LayoutFocusNameTimeoutMSec = 500

	// LayoutFocusNameTabMSec is the number of milliseconds since last focus name
	// event to allow tab to focus on next element with same name.
	LayoutFocusNameTabMSec = 2000

	// LayoutAutoScrollDelayMSec is amount of time to wait (in Milliseconds) before
	// trying to autoscroll again
	LayoutAutoScrollDelayMSec = 25
)

// Layoutlags has bool flags for Layout
type LayoutFlags WidgetFlags //enums:bitflag -trim-prefix Layout

const (
	// for stacked layout, only layout the top widget.
	// this is appropriate for e.g., tab layout, which does a full
	// redraw on stack changes, but not for e.g., check boxes which don't
	LayoutStackTopOnly LayoutFlags = LayoutFlags(WidgetFlagsN) + iota

	// true if this layout got a redo = true on previous iteration -- otherwise it just skips any re-layout on subsequent iteration
	LayoutNeedsRedo

	// LayoutNoKeys prevents processing of keyboard events for this layout.
	// By default, Layout handles focus navigation events, but if an
	// outer Widget handles these instead, then this should be set.
	LayoutNoKeys
)

///////////////////////////////////////////////////////////////////
// Layouter

// Layouter is the interface for layout functions, called by Layout
// widget type.
type Layouter interface {
	Widget

	// AsLayout returns the base Layout type
	AsLayout() *Layout
}

// AsLayout returns the given value as a value of type Layout if the type
// of the given value embeds Layout, or nil otherwise
func AsLayout(k ki.Ki) *Layout {
	if k == nil || k.This() == nil {
		return nil
	}
	if t, ok := k.(Layouter); ok {
		return t.AsLayout()
	}
	return nil
}

// AsLayout satisfies the [LayoutEmbedder] interface
func (t *Layout) AsLayout() *Layout {
	return t
}

///////////////////////////////////////////////////////////////////
// Layout

// Layout is the primary node type responsible for organizing the sizes
// and positions of child widgets. It does not render, only organize,
// so properties like background color will have no effect.
// All arbitrary collections of widgets should generally be contained
// within a layout -- otherwise the parent widget must take over
// responsibility for positioning.
// Layouts can automatically add scrollbars depending on the Overflow
// layout style.
// For a Grid layout, the 'columns' property should generally be set
// to the desired number of columns, from which the number of rows
// is computed -- otherwise it uses the square root of number of
// elements.
type Layout struct {
	WidgetBase

	// for Stacked layout, index of node to use as the top of the stack.
	// Only the node at this index is rendered -- if not a valid index, nothing is rendered.
	StackTop int

	// LayImpl contains implementational state info for doing layout
	LayImpl LayImplState `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// whether scrollbar is used for given dim
	HasScroll [2]bool `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// scroll bars -- we fully manage them as needed
	Scrolls [2]*Slider `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// accumulated name to search for when keys are typed
	FocusName string `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// time of last focus name event -- for timeout
	FocusNameTime time.Time `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// last element focused on -- used as a starting point if name is the same
	FocusNameLast ki.Ki `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`
}

func (ly *Layout) FlagType() enums.BitFlag {
	return LayoutFlags(ly.Flags)
}

func (ly *Layout) CopyFieldsFrom(frm any) {
	fr, ok := frm.(*Layout)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier Layout one\n", ly.KiType().Name)
		return
	}
	ly.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	ly.StackTop = fr.StackTop
}

func (ly *Layout) OnInit() {
	ly.LayoutStyles()
	ly.HandleLayoutEvents()
}

func (ly *Layout) LayoutStyles() {
	ly.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.FocusWithinable)
		// we never want borders on layouts
		s.MaxBorder = styles.Border{}

		switch {
		case s.Display == styles.DisplayFlex:
			// if s.Wrap {
			// 	s.Grow.Set(1, 1)
			// } else {
			s.Grow.SetDim(s.MainAxis, 1)
			s.Grow.SetDim(s.MainAxis.OtherDim(), 0)
			// }
		case s.Display == styles.DisplayStacked:
			s.Grow.Set(1, 1)
		case s.Display == styles.DisplayGrid:
			s.Grow.Set(1, 1)
		}
	})
}

func (ly *Layout) HandleLayoutEvents() {
	ly.HandleWidgetEvents()
	ly.HandleLayoutKeys()
	ly.HandleLayoutScrollEvents()
}

func (ly *Layout) Destroy() {
	for d := mat32.X; d <= mat32.Y; d++ {
		ly.DeleteScroll(d)
	}
	ly.WidgetBase.Destroy()
}

// DeleteScroll deletes scrollbar along given dimesion.
func (ly *Layout) DeleteScroll(d mat32.Dims) {
	if ly.Scrolls[d] == nil {
		return
	}
	sb := ly.Scrolls[d]
	sb.This().Destroy()
	ly.Scrolls[d] = nil
}

func (ly *Layout) Render(sc *Scene) {
	if ly.PushBounds(sc) {
		ly.RenderChildren(sc)
		ly.RenderScrolls(sc)
		ly.PopBounds(sc)
		// } else {
		// 	ly.SetScrollsOff()
	}
}

// HasAnyScroll returns true if layout has
func (ly *Layout) HasAnyScroll() bool {
	return ly.HasScroll[mat32.X] || ly.HasScroll[mat32.Y]
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
	sr := ly.Scrolls[d]
	sr.InitName(sr, fmt.Sprintf("scroll%v", d))
	ki.SetParent(sr, ly.This())
	// sr.SetFlag(true, ki.Field) // note: do not turn on -- breaks pos
	sr.SetType(SliderScrollbar)
	sr.Sc = sc
	sr.Dim = d
	sr.Tracking = true
	sr.Min = 0.0
	sr.Style(func(s *styles.Style) {
		s.Padding.Zero()
		s.Margin.Zero()
		bbsz := mat32.NewVec2FmPoint(ly.Alloc.ContentBBox.Size())
		if d == mat32.X {
			s.Min.Y = ly.Styles.ScrollBarWidth
			s.Min.X = units.Dot(bbsz.X)
		} else {
			s.Min.X = ly.Styles.ScrollBarWidth
			s.Min.Y = units.Dot(bbsz.Y)
		}
	})
	sr.OnChange(func(e events.Event) {
		e.SetHandled()
		// fmt.Println("change event")
		ly.SetNeedsLayout()
		ly.ScenePos(ly.Sc) // gets pos from scrolls, positions scrollbars
	})
	sr.Update()
	fmt.Println(ly, "configed scroll:", d)
}

// SetPosFromScrolls sets the Scroll position from scrollbars
func (ly *Layout) SetPosFromScrolls(sc *Scene) {
	for d := mat32.X; d <= mat32.Y; d++ {
		ly.Alloc.Scroll.SetDim(d, 0)
		if ly.HasScroll[d] {
			sb := ly.Scrolls[d]
			ly.Alloc.Scroll.SetDim(d, sb.Value)
			fmt.Println(ly, "set scroll val:", d, sb.Value)
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
	sb.ApplyStyle(sc)
	sb.Max = ly.LayImpl.KidsSize.Dim(d) // only scrollbar
	sb.Step = ly.Styles.Font.Size.Dots  // step by lines
	sb.PageStep = 10.0 * sb.Step        // todo: more dynamic
	sb.ThumbVal = ly.LayImpl.ContentSubGap.Dim(d)
	sb.TrackThr = 1
	sb.Value = mat32.Min(sb.Value, sb.Max-sb.ThumbVal) // keep in range
	od := d.OtherDim()
	bbmax := mat32.NewVec2FmPoint(ly.Alloc.ContentBBox.Max)
	sb.Alloc.Pos.SetDim(d, ly.Alloc.ContentPos.Dim(d))
	sb.Alloc.Pos.SetDim(od, bbmax.Dim(od)-sb.Alloc.Size.Content.Dim(od))
	sb.SetBBoxes(sc)
	fmt.Println(ly, "position scroll:", d, sb.Alloc.Pos, sb.Alloc.BBox)

	// sbw := ly.Styles.ScrollBarWidth.Dots
	//
	//	spc := ly.BoxSpace()
	//	pad := ly.Styles.Padding.Dots()
	//	marg := ly.Styles.Margin.Dots()
	//	avail := ly.AvailSize()
	//		odim := mat32.OtherDim(d)
	//		var opad float32
	//		if odim == mat32.X {
	//			opad = pad.Right + marg.Right
	//		} else {
	//			opad = pad.Bottom + marg.Bottom
	//		}
	//		// opad = 0// todo: temporary override until we get this fixed.
	//		// if opad > 0 {
	//		// 	fmt.Println(ly, "opad: ", odim, opad)
	//		// }
	//		if ly.HasScroll[d] {
	//			sb := ly.Scrolls[d]
	//			sb.GetSize(sc, 0)
	//			sb.Alloc.PosRel.SetDim(d, spc.Pos().Dim(d))
	//
	//			sb.Alloc.PosRel.SetDim(odim, avail.Dim(odim)-sbw+2+opad)
	//			// SidesTODO: not sure about this
	//			sb.Alloc.Size.Total.SetDim(d, avail.Dim(d)-spc.Size().Dim(d)/2)
	//			if ly.HasScroll[odim] { // make room for other
	//				sb.Alloc.Size.Total.SetSubDim(d, sbw)
	//			}
	//			sb.Alloc.Size.Total.SetDim(odim, sbw)
	//			sb.DoLayout(sc, ly.ScBBox, 0) // this will add parent position to above rel pos
	//		} else {
	//			if ly.Scrolls[d] != nil {
	//				ly.DeactivateScroll(ly.Scrolls[d])
	//			}
	//		}
	//	}
}

// func (ly *Layout) LayoutScroll(sc *Scene, delta image.Point, parBBox image.Rectangle) {
// 	ly.LayoutScrollBase(sc, delta, parBBox)
// 	ly.LayoutScrollScrolls(sc, delta, parBBox) // move scrolls BEFORE adding our own!
// 	preDelta := delta
// 	_ = preDelta
// 	delta = ly.LayoutScrollDelta(delta) // add our offset
// 	if ly.HasScroll[mat32.X] || ly.HasScroll[mat32.Y] {
// 		// todo: diagnose direct manip
// 		// fmt.Println("layout scroll", preDelta, delta)
// 	}
// 	ly.LayoutScrollChildren(sc, delta)
// 	ly.RenderScrolls(sc)
// }
//

// RenderScrolls draws the scrollbars
func (ly *Layout) RenderScrolls(sc *Scene) {
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.HasScroll[d] {
			// fmt.Println("render scroll", d)
			ly.Scrolls[d].Render(sc)
		}
	}
}

// ReRenderScrolls re-draws the scrollbars de-novo -- can be called ad-hoc by others
// func (ly *Layout) ReRenderScrolls(sc *Scene) {
// 	if ly.PushBounds(sc) {
// 		ly.RenderScrolls(sc)
// 		ly.PopBounds(sc)
// 	}
// }

// SetScrollsOff turns off the scrollbars
func (ly *Layout) SetScrollsOff() {
	for d := mat32.X; d <= mat32.Y; d++ {
		ly.HasScroll[d] = false
	}
}

// LayoutScrollScrolls moves scrollbars based on scrolling taking place in parent
// layouts -- critical to call this BEFORE we add our own delta, which is
// generated from these very same scrollbars.
// func (ly *Layout) LayoutScrollScrolls(sc *Scene, delta image.Point, parBBox image.Rectangle) {
// 	for d := mat32.X; d <= mat32.Y; d++ {
// 		// if ly.HasScroll[d] {
// 		// 	ly.Scrolls[d].LayoutScroll(sc, delta, parBBox)
// 		// }
// 	}
// }

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

func (ly *Layout) RenderChildren(sc *Scene) {
	if ly.Styles.Display == styles.DisplayStacked {
		ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			kwi.SetState(i != ly.StackTop, states.Invisible)
			return ki.Continue
		})
	}
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.Render(sc)
		return ki.Continue
	})
}

// AutoScrollRate determines the rate of auto-scrolling of layouts
var AutoScrollRate = float32(1.0)

// AutoScrollDim auto-scrolls along one dimension
func (ly *Layout) AutoScrollDim(dim mat32.Dims, st, pos int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
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
	return true
}

// ScrollDimToEnd scrolls to put the given child coordinate position (eg.,
// bottom / right of a view box) at the end (bottom / right) of our scroll
// area, to the extent possible -- returns true if scrolling was needed.
func (ly *Layout) ScrollDimToEnd(dim mat32.Dims, pos int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
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
	return true
}

// ScrollDimToCenter scrolls to put the given child coordinate position (eg.,
// middle of a view box) at the center of our scroll area, to the extent
// possible -- returns true if scrolling was needed.
func (ly *Layout) ScrollDimToCenter(dim mat32.Dims, pos int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
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
	return true
}

// ChildWithFocus returns a direct child of this layout that either is the
// current window focus item, or contains that focus item (along with its
// index) -- nil, -1 if none.
func (ly *Layout) ChildWithFocus() (ki.Ki, int) {
	em := ly.EventMgr()
	if em == nil {
		return nil, -1
	}
	for i, k := range ly.Kids {
		if k == nil {
			continue
		}
		_, ni := AsWidget(k)
		if ni == nil {
			continue
		}
		if ni.ContainsFocus() {
			return k, i
		}
	}
	return nil, -1
}

// FocusNextChild attempts to move the focus into the next layout child
// (with wraparound to start) -- returns true if successful.
// if updn is true, then for Grid layouts, it moves down to next row
// instead of just the sequentially next item.
func (ly *Layout) FocusNextChild(updn bool) bool {
	sz := len(ly.Kids)
	if sz <= 1 {
		return false
	}
	foc, idx := ly.ChildWithFocus()
	if foc == nil {
		return false
	}
	em := ly.EventMgr()
	if em == nil {
		return false
	}
	cur := em.Focus
	nxti := idx + 1
	if ly.Styles.Display == styles.DisplayGrid && updn {
		nxti = idx + ly.Styles.Columns
	}
	did := false
	if nxti < sz {
		nx := ly.Child(nxti).(Widget)
		did = em.FocusOnOrNext(nx)
	} else {
		nx := ly.Child(0).(Widget)
		did = em.FocusOnOrNext(nx)
	}
	if !did || em.Focus == cur {
		return false
	}
	return true
}

// FocusPrevChild attempts to move the focus into the previous layout child
// (with wraparound to end) -- returns true if successful.
// If updn is true, then for Grid layouts, it moves up to next row
// instead of just the sequentially next item.
func (ly *Layout) FocusPrevChild(updn bool) bool {
	sz := len(ly.Kids)
	if sz <= 1 {
		return false
	}
	foc, idx := ly.ChildWithFocus()
	if foc == nil {
		return false
	}
	em := ly.EventMgr()
	if em == nil {
		return false
	}
	cur := em.Focus
	nxti := idx - 1
	if ly.Styles.Display == styles.DisplayGrid && updn {
		nxti = idx - ly.Styles.Columns
	}
	did := false
	if nxti >= 0 {
		did = em.FocusOnOrPrev(ly.Child(nxti).(Widget))
	} else {
		did = em.FocusOnOrPrev(ly.Child(sz - 1).(Widget))
	}
	if !did || em.Focus == cur {
		return false
	}
	return true
}

// ClosePopup closes the parent Stage as a PopupStage.
// Returns false if not a popup.
func (ly *Layout) ClosePopup() bool {
	ps := ly.Sc.PopupStage()
	if ps == nil {
		return false
	}
	ps.Close()
	return true
}

// LayoutPageSteps is the number of steps to take in PageUp / Down events
// in terms of number of items.
var LayoutPageSteps = 10

// HandleLayoutKeys handles all key events for navigating focus within a Layout
// Typically this is done by the parent Scene level layout, but can be
// done by default if FocusWithinable Ability is set.
func (ly *Layout) HandleLayoutKeys() {
	ly.OnKeyChord(func(e events.Event) {
		ly.LayoutKeysImpl(e)
	})
}

// LayoutKeys is key processing for layouts -- focus name and arrow keys
func (ly *Layout) LayoutKeysImpl(e events.Event) {
	if ly.Is(LayoutNoKeys) {
		return
	}
	if KeyEventTrace {
		fmt.Println("Layout KeyInput:", ly)
	}
	kf := keyfun.Of(e.KeyChord())
	if kf == keyfun.Abort {
		if ly.ClosePopup() {
			e.SetHandled()
		}
		return
	}
	em := ly.EventMgr()
	if em == nil {
		return
	}
	switch kf {
	case keyfun.FocusNext: // tab
		if em.FocusNext() {
			// fmt.Println("foc next", ly, ly.EventMgr().Focus)
			e.SetHandled()
		}
		return
	case keyfun.FocusPrev: // shift-tab
		if em.FocusPrev() {
			// fmt.Println("foc prev", ly, ly.EventMgr().Focus)
			e.SetHandled()
		}
		return
	}
	grid := ly.Styles.Display == styles.DisplayGrid
	if ly.Styles.MainAxis == mat32.X || grid {
		switch kf {
		case keyfun.MoveRight:
			if ly.FocusNextChild(false) {
				e.SetHandled()
			}
			return
		case keyfun.MoveLeft:
			if ly.FocusPrevChild(false) {
				e.SetHandled()
			}
			return
		}
	}
	if ly.Styles.MainAxis == mat32.Y || grid {
		switch kf {
		case keyfun.MoveDown:
			if ly.FocusNextChild(true) {
				e.SetHandled()
			}
			return
		case keyfun.MoveUp:
			if ly.FocusPrevChild(true) {
				e.SetHandled()
			}
			return
		case keyfun.PageDown:
			proc := false
			for st := 0; st < LayoutPageSteps; st++ {
				if !ly.FocusNextChild(true) {
					break
				}
				proc = true
			}
			if proc {
				e.SetHandled()
			}
			return
		case keyfun.PageUp:
			proc := false
			for st := 0; st < LayoutPageSteps; st++ {
				if !ly.FocusPrevChild(true) {
					break
				}
				proc = true
			}
			if proc {
				e.SetHandled()
			}
			return
		}
	}
	ly.FocusOnName(e)
}

// FocusOnName processes key events to look for an element starting with given name
func (ly *Layout) FocusOnName(e events.Event) bool {
	if KeyEventTrace {
		fmt.Printf("Layout FocusOnName: %v\n", ly.Path())
	}
	kf := keyfun.Of(e.KeyChord())
	delayMs := int(e.Time().Sub(ly.FocusNameTime) / time.Millisecond)
	ly.FocusNameTime = e.Time()
	if kf == keyfun.FocusNext { // tab means go to next match -- don't worry about time
		if ly.FocusName == "" || delayMs > LayoutFocusNameTabMSec {
			ly.FocusName = ""
			ly.FocusNameLast = nil
			return false
		}
	} else {
		if delayMs > LayoutFocusNameTimeoutMSec {
			ly.FocusName = ""
		}
		if !unicode.IsPrint(e.KeyRune()) || e.Modifiers() != 0 {
			return false
		}
		sr := string(e.KeyRune())
		if ly.FocusName == sr {
			// re-search same letter
		} else {
			ly.FocusName += sr
			ly.FocusNameLast = nil // only use last if tabbing
		}
	}
	e.SetHandled()
	// fmt.Printf("searching for: %v  last: %v\n", ly.FocusName, ly.FocusNameLast)
	focel, found := ChildByLabelStartsCanFocus(ly, ly.FocusName, ly.FocusNameLast)
	if found {
		// todo:
		// em := ly.EventMgr()
		// if em != nil {
		// 	em.SetFocus(focel.(Widget)) // this will also scroll by default!
		// }
		ly.FocusNameLast = focel
		return true
	} else {
		if ly.FocusNameLast == nil {
			ly.FocusName = "" // nothing being found
		}
		ly.FocusNameLast = nil // start over
	}
	return false
}

// ChildByLabelStartsCanFocus uses breadth-first search to find first element
// within layout whose Label (from Labeler interface) starts with given string
// (case insensitive) and can focus.  If after is non-nil, only finds after
// given element.
func ChildByLabelStartsCanFocus(ly *Layout, name string, after ki.Ki) (ki.Ki, bool) {
	lcnm := strings.ToLower(name)
	var rki ki.Ki
	gotAfter := false
	ly.WalkBreadth(func(k ki.Ki) bool {
		if k == ly.This() { // skip us
			return ki.Continue
		}
		_, ni := AsWidget(k)
		if ni != nil && !ni.CanFocus() { // don't go any further
			return ki.Break
		}
		if after != nil && !gotAfter {
			if k == after {
				gotAfter = true
			}
			return ki.Continue // skip to next
		}
		kn := strings.ToLower(ToLabel(k))
		if rki == nil && strings.HasPrefix(kn, lcnm) {
			rki = k
			return ki.Break
		}
		return rki == nil // only continue if haven't found yet
	})
	if rki != nil {
		return rki, true
	}
	return nil, false
}

// HandleLayoutScrollEvents registers scrolling-related mouse events processed by
// Layout -- most subclasses of Layout will want these..
func (ly *Layout) HandleLayoutScrollEvents() {
	ly.On(events.Scroll, func(e events.Event) {
		// fmt.Println("event")
		ly.ScrollDelta(e)
	})
	// HiPri to do it first so others can be in view etc -- does NOT consume event!
	// we.AddFunc(events.DNDMoveEvent, HiPri, func(recv, send ki.Ki, sig int64, d any) {
	// 	me := d.(*dnd.Event)
	// 	li := AsLayout(recv)
	// 	li.AutoScroll(me.Pos())
	// })
	// we.AddFunc(events.MouseMoveEvent, HiPri, func(recv, send ki.Ki, sig int64, d any) {
	// 	me := d.(events.Event)
	// 	li := AsLayout(recv)
	// 	if li.Sc.Type == ScMenu {
	// 		li.AutoScroll(me.Pos())
	// 	}
	// })
}

///////////////////////////////////////////////////////////
//    Stretch and Space -- dummy elements for layouts

// Stretch adds an infinitely stretchy element for spacing out layouts
// (max-size = -1) set the width / height property to determine how much it
// takes relative to other stretchy elements
type Stretch struct {
	WidgetBase
}

func (st *Stretch) OnInit() {
	st.Style(func(s *styles.Style) {
		s.Min.X.Ch(1)
		s.Min.Y.Em(1)
		s.Grow.Set(1, 1)
	})
}

func (st *Stretch) CopyFieldsFrom(frm any) {
	fr := frm.(*Stretch)
	st.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
}

// Space adds a fixed sized (1 ch x 1 em by default) blank space to a layout.
// Set width / height property to change.
type Space struct {
	WidgetBase
}

// check for interface impl
var _ Widget = (*Space)(nil)

func (sp *Space) OnInit() {
	sp.Style(func(s *styles.Style) {
		s.Min.X.Ch(1)
		s.Min.Y.Em(1)
		s.Padding.Zero()
		s.Border.Width.Zero()
		s.Margin.Zero()
		s.MaxBorder.Width.Zero()
	})
}

func (sp *Space) CopyFieldsFrom(frm any) {
	fr := frm.(*Space)
	sp.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
}
