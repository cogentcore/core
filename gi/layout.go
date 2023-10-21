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
)

// LayoutAllocs contains all the the layout allocations: size, position.
// These are set by the parent Layout during the Layout process.
type LayoutAllocs struct {

	// allocated size of this item, by the parent layout -- also used temporarily during size process to hold computed size constraints based on content in terminal nodes
	Size mat32.Vec2

	// position of this item, computed by adding in the PosRel to parent position
	Pos mat32.Vec2

	// allocated relative position of this item, computed by the parent layout
	PosRel mat32.Vec2

	// original copy of allocated size of this item, by the parent layout -- some widgets will resize themselves within a given layout (e.g., a TextView), but still need access to their original allocated size
	SizeOrig mat32.Vec2

	// original copy of allocated relative position of this item, by the parent layout -- need for scrolling which can update AllocPos
	PosOrig mat32.Vec2
}

// Reset is called at start of layout process -- resets all values back to 0
func (la *LayoutAllocs) Reset() {
	la.Size = mat32.Vec2Zero
	la.Pos = mat32.Vec2Zero
	la.PosRel = mat32.Vec2Zero
}

func (la LayoutAllocs) String() string {
	return fmt.Sprintf("Alloc: Size=%s; Pos=%s; PosRel=%s; SizeOrig=%s; PosOrig=%s", la.Size.String(), la.Pos.String(), la.PosRel.String(), la.SizeOrig.String(), la.PosOrig.String())
}

// LayoutState contains all the state needed to specify the layout of an item
// within a Layout.  Is initialized with computed values of style prefs.
type LayoutState struct {

	// size constraints for this item -- set from layout style at start of layout process and then updated for Layout nodes to fit everything within it
	Size styles.SizePrefs

	// allocated size and position -- set by parent Layout
	Alloc LayoutAllocs
}

// todo: not using yet:
// Margins Margins   `desc:"margins around this item"`
// GridPos      image.Point `desc:"position within a grid"`
// GridSpan     image.Point `desc:"number of grid elements that we take up in each direction"`

func (ld *LayoutState) Defaults() {
}

func (ld *LayoutState) String() string {
	return ld.Size.String() + "\n" + ld.Alloc.String() + "\n"
}

func (ld *LayoutState) SetFromStyle(ls *styles.Style) {
	ld.Reset()
	// these are layout hints:
	ld.Size.Need = ls.MinSizeDots()
	ld.Size.Pref = ls.SizeDots()
	ld.Size.Max = ls.MaxSizeDots()

	// this is an actual initial desired setting
	ld.Alloc.Pos = ls.PosDots()
	// not setting size, so we can keep that as a separate constraint
}

// SizePrefOrMax returns the pref size if non-zero, else the max-size -- use
// for style-based constraints during initial sizing (e.g., word wrapping)
func (ld *LayoutState) SizePrefOrMax() mat32.Vec2 {
	return ld.Size.Pref.MinPos(ld.Size.Max)
}

// Reset is called at start of layout process -- resets all values back to 0
func (ld *LayoutState) Reset() {
	ld.Alloc.Reset()
}

// UpdateSizes updates our sizes based on AllocSize and Max constraints, etc
func (ld *LayoutState) UpdateSizes() {
	ld.Size.Need.SetMax(ld.Alloc.Size)  // min cannot be < alloc -- bare min
	ld.Size.Pref.SetMax(ld.Size.Need)   // pref cannot be < min
	ld.Size.Need.SetMinPos(ld.Size.Max) // min cannot be > max
	ld.Size.Pref.SetMinPos(ld.Size.Max) // pref cannot be > max
}

// GridData contains data for grid layout -- only one value needed for relevant dim
type GridData struct {
	SizeNeed    float32
	SizePref    float32
	SizeMax     float32
	AllocSize   float32
	AllocPosRel float32
}

////////////////////////////////////////////////////////////////////////////////////////
// Layout

// LayoutFocusNameTimeoutMSec is the number of milliseconds between keypresses
// to combine characters into name to search for within layout -- starts over
// after this delay.
var LayoutFocusNameTimeoutMSec = 500

// LayoutFocusNameTabMSec is the number of milliseconds since last focus name
// event to allow tab to focus on next element with same name.
var LayoutFocusNameTabMSec = 2000

// Layout is the primary node type responsible for organizing the sizes
// and positions of child widgets. It does not render, only organize,
// so properties like background color will have no effect.
// All arbitrary collections of widgets should generally be contained
// within a layout -- otherwise the parent widget must take over
// responsibility for positioning.
// The alignment is NOT inherited by default so must be specified per
// child, except that the parent alignment is used within the relevant
// dimension (e.g., horizontal-align for a LayoutHoriz layout,
// to determine left, right, center, justified).
// Layouts can automatically add scrollbars depending on the Overflow
// layout style.
// For a Grid layout, the 'columns' property should generally be set
// to the desired number of columns, from which the number of rows
// is computed -- otherwise it uses the square root of number of
// elements.
//
//goki:embedder
type Layout struct {
	WidgetBase

	// type of layout to use
	Lay Layouts `xml:"lay" setter:"-"`

	// extra space to add between elements in the layout
	Spacing units.Value `xml:"spacing"`

	// for Stacked layout, index of node to use as the top of the stack -- only node at this index is rendered -- if not a valid index, nothing is rendered
	StackTop int

	// for stacked layout, only layout the top widget -- this is appropriate for e.g., tab layout, which does a full redraw on stack changes, but not for e.g., check boxes which don't
	StackTopOnly bool

	// total max size of children as laid out
	ChildSize mat32.Vec2 `copy:"-" json:"-" xml:"-"`

	// extra size in each dim due to scrollbars we add
	ExtraSize mat32.Vec2 `copy:"-" json:"-" xml:"-"`

	// whether scrollbar is used for given dim
	HasScroll [2]bool `copy:"-" json:"-" xml:"-"`

	// scroll bars -- we fully manage them as needed
	Scrolls [2]*Slider `copy:"-" json:"-" xml:"-"`

	// computed size of a grid layout based on all the constraints -- computed during GetSize pass
	GridSize image.Point `copy:"-" json:"-" xml:"-"`

	// grid data for rows in and cols in
	GridData [RowColN][]GridData `copy:"-" json:"-" xml:"-"`

	// line breaks for flow layout
	FlowBreaks []int `copy:"-" json:"-" xml:"-"`

	// true if this layout got a redo = true on previous iteration -- otherwise it just skips any re-layout on subsequent iteration
	NeedsRedo bool `copy:"-" json:"-" xml:"-"`

	// accumulated name to search for when keys are typed
	FocusName string `copy:"-" json:"-" xml:"-"`

	// time of last focus name event -- for timeout
	FocusNameTime time.Time `copy:"-" json:"-" xml:"-"`

	// last element focused on -- used as a starting point if name is the same
	FocusNameLast ki.Ki `copy:"-" json:"-" xml:"-"`

	// scrollbars have been manually turned off due to layout being invisible -- must be reactivated when re-visible
	ScrollsOff bool `copy:"-" json:"-" xml:"-" setter:"-"`
}

func (ly *Layout) CopyFieldsFrom(frm any) {
	fr, ok := frm.(*Layout)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier Layout one\n", ly.KiType().Name)
		return
	}
	ly.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	ly.Lay = fr.Lay
	ly.Spacing = fr.Spacing
	ly.StackTop = fr.StackTop
}

func (ly *Layout) OnInit() {
	ly.LayoutStyles()
	ly.HandleLayoutEvents()
}

func (ly *Layout) LayoutStyles() {
	ly.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.FocusWithinable)
		// we never want state layers and borders on layouts
		s.StateLayer = 0
		s.MaxBorder = styles.Border{}
	})
}

func (ly *Layout) HandleLayoutEvents() {
	ly.HandleWidgetEvents()
	ly.HandleLayoutKeys()
	ly.HandleLayoutScrollEvents()
}

// Layouts are the different types of layouts
type Layouts int32 //enums:enum -trim-prefix Layout

const (
	// LayoutHoriz arranges items horizontally across a row
	LayoutHoriz Layouts = iota

	// LayoutVert arranges items vertically in a column
	LayoutVert

	// LayoutGrid arranges items according to a regular grid
	LayoutGrid

	// todo: add LayoutGridIrreg that deals with irregular grids with spans etc -- keep
	// the basic grid for fully regular cases -- need high performance for large grids

	// LayoutHorizFlow arranges items horizontally across a row, overflowing
	// vertically as needed.  Ballpark target width or height props should be set
	// to generate initial first-pass sizing estimates.
	LayoutHorizFlow

	// LayoutVertFlow arranges items vertically within a column, overflowing
	// horizontally as needed.  Ballpark target width or height props should be set
	// to generate initial first-pass sizing estimates.
	LayoutVertFlow

	// LayoutStacked arranges items stacked on top of each other -- Top index
	// indicates which to show -- overall size accommodates largest in each
	// dimension
	LayoutStacked

	// LayoutNil is a nil layout -- doesn't do anything -- for cases when a
	// parent wants to take over the job of the layout
	LayoutNil
)

// row / col for grid data
type RowCol int32 //enums:enum

const (
	Row RowCol = iota
	Col
)

// LayoutDefault is default obj that can be used when property specifies "default"
var LayoutDefault Layout

// SetLayout sets the layout type of the Layout.
func (ly *Layout) SetLayout(lay Layouts) *Layout {
	ly.Lay = lay
	return ly
}

////////////////////////////////////////////////////////////////////////////////////////
//     Overflow: Scrolling mainly

// AvailSize returns the total size avail to this layout -- typically
// AllocSize except for top-level layout which uses ScBBox in case less is
// avail
func (ly *Layout) AvailSize() mat32.Vec2 {
	spc := ly.BoxSpace()
	// we only want to subtract pos, not size here
	// because this is for right and bottom side space
	avail := ly.LayState.Alloc.Size.Sub(spc.Pos())
	parni, _ := AsWidget(ly.Par)
	// SidesTODO: what is the story with this?
	if parni != nil {
		// if vp.Sc == nil {
		// 	// SidesTODO: might not be right
		// 	avail = mat32.NewVec2FmPoint(ly.ScBBox.Size()).SubScalar(spc.Right)
		// 	// fmt.Printf("non-nil par ly: %v vp: %v %v\n", ly.Path(), vp.Path(), avail)
		// }
	}
	return avail
}

// ManageOverflow processes any overflow according to overflow settings.
func (ly *Layout) ManageOverflow(sc *Scene) {
	// wasscof := ly.ScrollsOff
	ly.ScrollsOff = false
	if ly.Lay == LayoutNil {
		return
	}
	avail := ly.AvailSize()

	ly.ExtraSize.SetScalar(0)
	for d := mat32.X; d <= mat32.Y; d++ {
		ly.HasScroll[d] = false
	}

	if ly.Styles.Overflow != styles.OverflowHidden {
		sbw := ly.Styles.ScrollBarWidth.Dots
		for d := mat32.X; d <= mat32.Y; d++ {
			odim := mat32.OtherDim(d)
			if ly.ChildSize.Dim(d) > (avail.Dim(d) + 2.0) { // overflowing -- allow some margin
				// if wasscof {
				// 	fmt.Printf("overflow, setting scb: %v\n", d)
				// }
				ly.HasScroll[d] = true
				ly.ExtraSize.SetAddDim(odim, sbw)
			}
		}
		for d := mat32.X; d <= mat32.Y; d++ {
			if ly.HasScroll[d] {
				ly.SetScroll(sc, d)
			}
		}
		ly.LayoutScrolls(sc)
	}
}

// HasAnyScroll returns true if layout has
func (ly *Layout) HasAnyScroll() bool {
	return ly.HasScroll[mat32.X] || ly.HasScroll[mat32.Y]
}

// SetScroll sets a scrollbar along given dimension
func (ly *Layout) SetScroll(sc *Scene, d mat32.Dims) {
	if ly.Scrolls[d] == nil {
		ly.Scrolls[d] = &Slider{}
		sr := ly.Scrolls[d]
		sr.InitName(sr, fmt.Sprintf("Scroll%v", d))
		ki.SetParent(sr, ly.This())
		// sr.SetFlag(true, ki.Field) // note: do not turn on -- breaks pos
		sr.SetType(SliderScrollbar)
		sr.Sc = sc
		sr.Dim = d
		sr.Config(sc)
		sr.Tracking = true
		sr.Min = 0.0
		sr.Style(func(s *styles.Style) {
			s.Padding.Set(units.Dp(0))
			s.Margin.Set(units.Dp(0))
		})
		sr.OnChange(func(e events.Event) {
			e.SetHandled()
			// fmt.Println("change event")
			ly.SetNeedsLayout()
			ly.LayoutScrollTree(sc)
		})
	}
	spc := ly.BoxSpace()
	avail := ly.AvailSize().Sub(spc.Size())
	sb := ly.Scrolls[d]
	if d == mat32.X {
		sb.SetFixedHeight(ly.Styles.ScrollBarWidth)
		sb.SetFixedWidth(units.Dot(avail.Dim(d)))
	} else {
		sb.SetFixedWidth(ly.Styles.ScrollBarWidth)
		sb.SetFixedHeight(units.Dot(avail.Dim(d)))
	}
	sb.ApplyStyle(sc)
	sb.Max = ly.ChildSize.Dim(d) + ly.ExtraSize.Dim(d) // only scrollbar
	sb.Step = ly.Styles.Font.Size.Dots                 // step by lines
	sb.PageStep = 10.0 * sb.Step                       // todo: more dynamic
	sb.ThumbVal = avail.Dim(d) - spc.Size().Dim(d)/2
	sb.TrackThr = 1
	sb.Value = mat32.Min(sb.Value, sb.Max-sb.ThumbVal) // keep in range
	// fmt.Printf("set sc lay: %v  max: %v  val: %v\n", ly.Path(), sc.Max, sc.Value)
}

// DeleteScroll deletes scrollbar along given dimesion.  todo: we are leaking
// the scrollbars -- move into a container Field
func (ly *Layout) DeleteScroll(d mat32.Dims) {
	if ly.Scrolls[d] == nil {
		return
	}
	sb := ly.Scrolls[d]
	sb.This().Destroy()
	ly.Scrolls[d] = nil
}

// DeactivateScroll turns off given scrollbar, without deleting, so it can be easily re-used
func (ly *Layout) DeactivateScroll(sb *Slider) {
	sb.BBoxMu.Lock()
	defer sb.BBoxMu.Unlock()
	sb.LayState.Alloc.Pos = mat32.Vec2Zero
	sb.LayState.Alloc.Size = mat32.Vec2Zero
	sb.ScBBox = image.Rectangle{}
}

// LayoutScrolls arranges scrollbars
func (ly *Layout) LayoutScrolls(sc *Scene) {
	sbw := ly.Styles.ScrollBarWidth.Dots

	spc := ly.BoxSpace()
	pad := ly.Styles.Padding.Dots()
	marg := ly.Styles.Margin.Dots()
	avail := ly.AvailSize()
	for d := mat32.X; d <= mat32.Y; d++ {
		odim := mat32.OtherDim(d)
		var opad float32
		if odim == mat32.X {
			opad = pad.Right + marg.Right
		} else {
			opad = pad.Bottom + marg.Bottom
		}
		// opad = 0// todo: temporary override until we get this fixed.
		// if opad > 0 {
		// 	fmt.Println(ly, "opad: ", odim, opad)
		// }
		if ly.HasScroll[d] {
			sb := ly.Scrolls[d]
			sb.GetSize(sc, 0)
			sb.LayState.Alloc.PosRel.SetDim(d, spc.Pos().Dim(d))

			sb.LayState.Alloc.PosRel.SetDim(odim, avail.Dim(odim)-sbw+2+opad)
			// SidesTODO: not sure about this
			sb.LayState.Alloc.Size.SetDim(d, avail.Dim(d)-spc.Size().Dim(d)/2)
			if ly.HasScroll[odim] { // make room for other
				sb.LayState.Alloc.Size.SetSubDim(d, sbw)
			}
			sb.LayState.Alloc.Size.SetDim(odim, sbw)
			sb.DoLayout(sc, ly.ScBBox, 0) // this will add parent position to above rel pos
		} else {
			if ly.Scrolls[d] != nil {
				ly.DeactivateScroll(ly.Scrolls[d])
			}
		}
	}
}

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
func (ly *Layout) ReRenderScrolls(sc *Scene) {
	if ly.PushBounds(sc) {
		ly.RenderScrolls(sc)
		ly.PopBounds(sc)
	}
}

// SetScrollsOff turns off the scrolls -- e.g., when layout is not visible
func (ly *Layout) SetScrollsOff() {
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.HasScroll[d] {
			// fmt.Printf("turning scroll off for :%v dim: %v\n", ly.Path(), d)
			ly.ScrollsOff = true
			ly.HasScroll[d] = false
			if ly.Scrolls[d] != nil {
				ly.DeactivateScroll(ly.Scrolls[d])
			}
		}
	}
}

// LayoutScrollScrolls moves scrollbars based on scrolling taking place in parent
// layouts -- critical to call this BEFORE we add our own delta, which is
// generated from these very same scrollbars.
func (ly *Layout) LayoutScrollScrolls(sc *Scene, delta image.Point, parBBox image.Rectangle) {
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.HasScroll[d] {
			ly.Scrolls[d].LayoutScroll(sc, delta, parBBox)
		}
	}
}

// ScrollActionDelta moves the scrollbar in given dimension by given delta
// and emits a ScrollSig signal.
func (ly *Layout) ScrollActionDelta(dim mat32.Dims, delta float32) {
	if ly.HasScroll[dim] {
		sb := ly.Scrolls[dim]
		nval := sb.Value + delta
		sb.SetValue(nval)
		ly.SetNeedsLayoutUpdate(ly.Sc, true)
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

func (ly *Layout) DoLayoutChildren(sc *Scene, iter int) bool {
	cbb := ly.ChildrenBBoxes(sc)
	// todo: we need a flag for this for tab frame to not do this -- thought this was there already!
	// if ly.Lay == LayoutStacked {
	// 	sn, err := ly.ChildTry(ly.StackTop)
	// 	if err != nil {
	// 		return false
	// 	}
	// 	nii, _ := AsWidget(sn)
	// 	return nii.DoLayout(sc, cbb, iter)
	// } else {
	redo := false
	for _, kid := range ly.Kids {
		wi, _ := AsWidget(kid)
		if wi == nil || wi.This() == nil {
			continue
		}
		if wi.DoLayout(sc, cbb, iter) {
			redo = true
		}
	}
	return redo
	// }
}

// render the children
func (ly *Layout) RenderChildren(sc *Scene) {
	if ly.Lay == LayoutStacked {
		for i, kid := range ly.Kids {
			if _, wi := AsWidget(kid); wi != nil {
				wi.SetState(i != ly.StackTop, states.Invisible)
			}
		}
	}
	for _, kid := range ly.Kids {
		if kid == nil {
			continue
		}
		wi, _ := AsWidget(kid)
		if wi != nil {
			wi.Render(sc)
		}
	}
}

func (ly *Layout) LayoutScrollChildren(sc *Scene, delta image.Point) {
	wi := ly.This().(Widget)
	cbb := wi.ChildrenBBoxes(sc)
	if ly.Lay == LayoutStacked {
		sn, err := ly.ChildTry(ly.StackTop)
		if err != nil {
			return
		}
		ci, _ := AsWidget(sn)
		ci.LayoutScroll(sc, delta, cbb)
	} else {
		for _, kid := range ly.Kids {
			ci, _ := AsWidget(kid)
			if ci != nil {
				ci.LayoutScroll(sc, delta, cbb)
			}
		}
	}
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

// LayoutAutoScrollDelayMSec is amount of time to wait (in Milliseconds) before
// trying to autoscroll again
var LayoutAutoScrollDelayMSec = 25

// AutoScroll scrolls the layout based on mouse position, when appropriate (DND, menus)
func (ly *Layout) AutoScroll(pos image.Point) bool {
	now := time.Now()
	lagMs := int(now.Sub(LayoutLastAutoScroll) / time.Millisecond)
	if lagMs < LayoutAutoScrollDelayMSec {
		return false
	}
	ly.BBoxMu.RLock()
	wbb := ly.ScBBox
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
	vpMin := ly.ScBBox.Min.X
	if dim == mat32.Y {
		vpMin = ly.ScBBox.Min.Y
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
	return ly.ScrollToBox(wi.AsWidget().ObjBBox)
}

// ScrollDimToStart scrolls to put the given child coordinate position (eg.,
// top / left of a view box) at the start (top / left) of our scroll area, to
// the extent possible -- returns true if scrolling was needed.
func (ly *Layout) ScrollDimToStart(dim mat32.Dims, pos int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
	vpMin := ly.ScBBox.Min.X
	if dim == mat32.Y {
		vpMin = ly.ScBBox.Min.Y
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
	vpMin := ly.ScBBox.Min.X
	if dim == mat32.Y {
		vpMin = ly.ScBBox.Min.Y
	}
	sc := ly.Scrolls[dim]
	scrange := sc.Max - sc.ThumbVal                // amount that can be scrolled
	vissz := (sc.ThumbVal - ly.ExtraSize.Dim(dim)) // amount visible
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
	vpMin := ly.ScBBox.Min.X
	if dim == mat32.Y {
		vpMin = ly.ScBBox.Min.Y
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
	if ly.Lay == LayoutGrid && updn {
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
	if ly.Lay == LayoutGrid && updn {
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
	if KeyEventTrace {
		fmt.Println("Layout KeyInput:", ly)
	}
	kf := KeyFun(e.KeyChord())
	if kf == KeyFunAbort {
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
	case KeyFunFocusNext: // tab
		if em.FocusNext() {
			// fmt.Println("foc next", ly, ly.EventMgr().Focus)
			e.SetHandled()
		}
		return
	case KeyFunFocusPrev: // shift-tab
		if em.FocusPrev() {
			// fmt.Println("foc prev", ly, ly.EventMgr().Focus)
			e.SetHandled()
		}
		return
	}
	if ly.Lay == LayoutHoriz || ly.Lay == LayoutGrid || ly.Lay == LayoutHorizFlow {
		switch kf {
		case KeyFunMoveRight:
			if ly.FocusNextChild(false) {
				e.SetHandled()
			}
			return
		case KeyFunMoveLeft:
			if ly.FocusPrevChild(false) {
				e.SetHandled()
			}
			return
		}
	}
	if ly.Lay == LayoutVert || ly.Lay == LayoutGrid || ly.Lay == LayoutVertFlow {
		switch kf {
		case KeyFunMoveDown:
			if ly.FocusNextChild(true) {
				e.SetHandled()
			}
			return
		case KeyFunMoveUp:
			if ly.FocusPrevChild(true) {
				e.SetHandled()
			}
			return
		case KeyFunPageDown:
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
		case KeyFunPageUp:
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
	kf := KeyFun(e.KeyChord())
	delayMs := int(e.Time().Sub(ly.FocusNameTime) / time.Millisecond)
	ly.FocusNameTime = e.Time()
	if kf == KeyFunFocusNext { // tab means go to next match -- don't worry about time
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

///////////////////////////////////////////////////
//   Standard Widget interface

func (ly *Layout) BBoxes() image.Rectangle {
	bb := ly.BBoxFromAlloc()
	return bb
}

func (ly *Layout) ComputeBBoxes(sc *Scene, parBBox image.Rectangle, delta image.Point) {
	ly.ComputeBBoxesBase(sc, parBBox, delta)
}

func (ly *Layout) ChildrenBBoxes(sc *Scene) image.Rectangle {
	nb := ly.ChildrenBBoxesWidget(sc)
	nb.Max.X -= int(ly.ExtraSize.X)
	nb.Max.Y -= int(ly.ExtraSize.Y)
	return nb
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (ly *Layout) StyleToDots(uc *units.Context) {
	ly.Spacing.ToDots(uc)
}

// StyleLayout does layout styling -- it sets the StyMu Lock
func (ly *Layout) StyleLayout(sc *Scene) {
	ly.ApplyStyleWidget(sc)
	ly.StyleToDots(&ly.Styles.UnContext)
}

func (ly *Layout) ApplyStyle(sc *Scene) {
	ly.StyleLayout(sc)
}

func (ly *Layout) GetSize(sc *Scene, iter int) {
	ly.InitLayout(sc)
	switch ly.Lay {
	case LayoutHorizFlow, LayoutVertFlow:
		GatherSizesFlow(ly, iter)
	case LayoutGrid:
		GatherSizesGrid(ly)
	default:
		GatherSizes(ly)
	}
}

func (ly *Layout) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	if iter > 0 && LayoutTrace {
		fmt.Printf("Layout: %v Iteration: %v  NeedsRedo: %v\n", ly.Path(), iter, ly.NeedsRedo)
	}
	ly.DoLayoutBase(sc, parBBox, iter)
	redo := false
	switch ly.Lay {
	case LayoutHoriz:
		LayoutAlongDim(ly, mat32.X)
		LayoutSharedDim(ly, mat32.Y)
	case LayoutVert:
		LayoutAlongDim(ly, mat32.Y)
		LayoutSharedDim(ly, mat32.X)
	case LayoutGrid:
		LayoutGridLay(ly)
	case LayoutStacked:
		LayoutSharedDim(ly, mat32.X)
		LayoutSharedDim(ly, mat32.Y)
	case LayoutHorizFlow:
		redo = LayoutFlow(ly, mat32.X, iter)
	case LayoutVertFlow:
		redo = LayoutFlow(ly, mat32.Y, iter)
	case LayoutNil:
		// nothing
	}
	ly.FinalizeLayout()
	if redo && iter == 0 {
		ly.NeedsRedo = true
		ly.LayState.Alloc.Size = ly.ChildSize // this is what we actually need.
		return true
	}
	ly.ManageOverflow(sc)
	ly.NeedsRedo = ly.DoLayoutChildren(sc, iter) // layout done with canonical positions

	if !ly.NeedsRedo || iter == 1 {
		delta := ly.LayoutScrollDelta((image.Point{}))
		if delta != (image.Point{}) {
			ly.LayoutScrollChildren(sc, delta) // move is a separate step
		}
	}
	return ly.NeedsRedo
}

// we add our own offset here
func (ly *Layout) LayoutScrollDelta(delta image.Point) image.Point {
	if ly.HasScroll[mat32.X] {
		off := ly.Scrolls[mat32.X].Value
		delta.X -= int(off)
	}
	if ly.HasScroll[mat32.Y] {
		off := ly.Scrolls[mat32.Y].Value
		delta.Y -= int(off)
	}
	return delta
}

func (ly *Layout) LayoutScroll(sc *Scene, delta image.Point, parBBox image.Rectangle) {
	ly.LayoutScrollBase(sc, delta, parBBox)
	ly.LayoutScrollScrolls(sc, delta, parBBox) // move scrolls BEFORE adding our own!
	preDelta := delta
	_ = preDelta
	delta = ly.LayoutScrollDelta(delta) // add our offset
	if ly.HasScroll[mat32.X] || ly.HasScroll[mat32.Y] {
		// todo: diagnose direct manip
		// fmt.Println("layout scroll", preDelta, delta)
	}
	ly.LayoutScrollChildren(sc, delta)
	ly.RenderScrolls(sc)
}

func (ly *Layout) Render(sc *Scene) {
	if ly.PushBounds(sc) {
		ly.RenderChildren(sc)
		ly.RenderScrolls(sc)
		ly.PopBounds(sc)
	} else {
		ly.SetScrollsOff()
	}
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
		s.SetMinPrefHeight(units.Ch(1))
		s.SetMinPrefWidth(units.Em(1))
		s.MaxWidth.SetDp(-1)
		s.MaxHeight.SetDp(-1)
	})
}

func (st *Stretch) CopyFieldsFrom(frm any) {
	fr := frm.(*Stretch)
	st.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
}

func (st *Stretch) ApplyStyle(sc *Scene) {
	st.StyMu.Lock()
	defer st.StyMu.Unlock()

	st.ApplyStyleWidget(sc)
}

// Space adds a fixed sized (1 ch x 1 em by default) blank space to a layout -- set
// width / height property to change
type Space struct {
	WidgetBase
}

// check for interface impl
var _ Widget = (*Space)(nil)

func (sp *Space) OnInit() {
	sp.Style(func(s *styles.Style) {
		s.Width.SetCh(1)
		s.Height.SetEm(1)
	})
}

func (sp *Space) CopyFieldsFrom(frm any) {
	fr := frm.(*Space)
	sp.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
}

func (sp *Space) ApplyStyle(sc *Scene) {
	sp.StyMu.Lock()
	defer sp.StyMu.Unlock()

	sp.ApplyStyleWidget(sc)
}
