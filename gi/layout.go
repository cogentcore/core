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

	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// LayoutPrefMaxRows is maximum number of rows to use in a grid layout
// when computing the preferred size (VpFlagPrefSizing)
var LayoutPrefMaxRows = 20

// LayoutPrefMaxCols is maximum number of columns to use in a grid layout
// when computing the preferred size (VpFlagPrefSizing)
var LayoutPrefMaxCols = 20

// LayoutAllocs contains all the the layout allocations: size, position.
// These are set by the parent Layout during the Layout process.
type LayoutAllocs struct {
	Size     mat32.Vec2 `desc:"allocated size of this item, by the parent layout -- also used temporarily during size process to hold computed size constraints based on content in terminal nodes"`
	Pos      mat32.Vec2 `desc:"position of this item, computed by adding in the PosRel to parent position"`
	PosRel   mat32.Vec2 `desc:"allocated relative position of this item, computed by the parent layout"`
	SizeOrig mat32.Vec2 `desc:"original copy of allocated size of this item, by the parent layout -- some widgets will resize themselves within a given layout (e.g., a TextView), but still need access to their original allocated size"`
	PosOrig  mat32.Vec2 `desc:"original copy of allocated relative position of this item, by the parent layout -- need for scrolling which can update AllocPos"`
}

// Reset is called at start of layout process -- resets all values back to 0
func (la *LayoutAllocs) Reset() {
	la.Size = mat32.Vec2Zero
	la.Pos = mat32.Vec2Zero
	la.PosRel = mat32.Vec2Zero
}

// LayoutState contains all the state needed to specify the layout of an item
// within a Layout.  Is initialized with computed values of style prefs.
type LayoutState struct {
	Size  gist.SizePrefs `desc:"size constraints for this item -- set from layout style at start of layout process and then updated for Layout nodes to fit everything within it"`
	Alloc LayoutAllocs   `desc:"allocated size and position -- set by parent Layout"`
}

// todo: not using yet:
// Margins Margins   `desc:"margins around this item"`
// GridPos      image.Point `desc:"position within a grid"`
// GridSpan     image.Point `desc:"number of grid elements that we take up in each direction"`

func (ld *LayoutState) Defaults() {
}

func (ld *LayoutState) SetFromStyle(ls *gist.Layout) {
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
// and positions of child widgets.
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
type Layout struct {
	WidgetBase
	Lay           Layouts             `xml:"lay" desc:"type of layout to use"`
	Spacing       units.Value         `xml:"spacing" desc:"extra space to add between elements in the layout"`
	StackTop      int                 `desc:"for Stacked layout, index of node to use as the top of the stack -- only node at this index is rendered -- if not a valid index, nothing is rendered"`
	StackTopOnly  bool                `desc:"for stacked layout, only layout the top widget -- this is appropriate for e.g., tab layout, which does a full redraw on stack changes, but not for e.g., check boxes which don't"`
	ChildSize     mat32.Vec2          `copy:"-" json:"-" xml:"-" desc:"total max size of children as laid out"`
	ExtraSize     mat32.Vec2          `copy:"-" json:"-" xml:"-" desc:"extra size in each dim due to scrollbars we add"`
	HasScroll     [2]bool             `copy:"-" json:"-" xml:"-" desc:"whether scrollbar is used for given dim"`
	Scrolls       [2]*ScrollBar       `copy:"-" json:"-" xml:"-" desc:"scroll bars -- we fully manage them as needed"`
	GridSize      image.Point         `copy:"-" json:"-" xml:"-" desc:"computed size of a grid layout based on all the constraints -- computed during Size2D pass"`
	GridData      [RowColN][]GridData `copy:"-" json:"-" xml:"-" desc:"grid data for rows in [0] and cols in [1]"`
	FlowBreaks    []int               `copy:"-" json:"-" xml:"-" desc:"line breaks for flow layout"`
	NeedsRedo     bool                `copy:"-" json:"-" xml:"-" desc:"true if this layout got a redo = true on previous iteration -- otherwise it just skips any re-layout on subsequent iteration"`
	FocusName     string              `copy:"-" json:"-" xml:"-" desc:"accumulated name to search for when keys are typed"`
	FocusNameTime time.Time           `copy:"-" json:"-" xml:"-" desc:"time of last focus name event -- for timeout"`
	FocusNameLast ki.Ki               `copy:"-" json:"-" xml:"-" desc:"last element focused on -- used as a starting point if name is the same"`
	ScrollsOff    bool                `copy:"-" json:"-" xml:"-" desc:"scrollbars have been manually turned off due to layout being invisible -- must be reactivated when re-visible"`
	ScrollSig     ki.Signal           `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for layout scrolling -- sends signal whenever layout is scrolled due to user input -- signal type is dimension (mat32.X or Y) and data is new position (not delta)"`
}

var KiT_Layout = kit.Types.AddType(&Layout{}, LayoutProps)

var LayoutProps = ki.Props{
	"EnumType:Flag": KiT_NodeFlags,
}

// AddNewLayout adds a new layout to given parent node, with given name and layout
func AddNewLayout(parent ki.Ki, name string, layout Layouts) *Layout {
	ly := parent.AddNewChild(KiT_Layout, name).(*Layout)
	ly.Lay = layout
	return ly
}

func (ly *Layout) CopyFieldsFrom(frm interface{}) {
	fr, ok := frm.(*Layout)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier Layout one\n", ki.Type(ly).Name())
		ki.GenCopyFieldsFrom(ly.This(), frm)
		return
	}
	ly.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	ly.Lay = fr.Lay
	ly.Spacing = fr.Spacing
	ly.StackTop = fr.StackTop
}

// Layouts are the different types of layouts
type Layouts int32

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

	LayoutsN
)

//go:generate stringer -type=Layouts

var KiT_Layouts = kit.Enums.AddEnumAltLower(LayoutsN, kit.NotBitFlag, gist.StylePropProps, "Layout")

func (ev Layouts) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Layouts) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// row / col for grid data
type RowCol int32

const (
	Row RowCol = iota
	Col
	RowColN
)

var KiT_RowCol = kit.Enums.AddEnumAltLower(RowColN, kit.NotBitFlag, gist.StylePropProps, "")

func (ev RowCol) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *RowCol) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

//go:generate stringer -type=RowCol

// LayoutDefault is default obj that can be used when property specifies "default"
var LayoutDefault Layout

// AvailSize returns the total size avail to this layout -- typically
// AllocSize except for top-level layout which uses VpBBox in case less is
// avail
func (ly *Layout) AvailSize() mat32.Vec2 {
	spc := ly.BoxSpace()
	avail := ly.LayState.Alloc.Size.SubScalar(spc) // spc is for right size space
	parni, _ := KiToNode2D(ly.Par)
	if parni != nil {
		vp := parni.AsViewport2D()
		if vp != nil {
			if vp.ViewportSafe() == nil {
				avail = mat32.NewVec2FmPoint(ly.VpBBox.Size()).SubScalar(spc)
				// fmt.Printf("non-nil par ly: %v vp: %v %v\n", ly.Path(), vp.Path(), avail)
			}
		}
	}
	return avail
}

////////////////////////////////////////////////////////////////////////////////////////
//     Overflow: Scrolling mainly

// ManageOverflow processes any overflow according to overflow settings.
func (ly *Layout) ManageOverflow() {
	// wasscof := ly.ScrollsOff
	ly.ScrollsOff = false
	if len(ly.Kids) == 0 || ly.Lay == LayoutNil {
		return
	}
	avail := ly.AvailSize()

	ly.ExtraSize.SetScalar(0)
	for d := mat32.X; d <= mat32.Y; d++ {
		ly.HasScroll[d] = false
	}

	if ly.Sty.Layout.Overflow != gist.OverflowHidden {
		sbw := ly.Sty.Layout.ScrollBarWidth.Dots
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
				ly.SetScroll(d)
			}
		}
		ly.LayoutScrolls()
	}
}

// HasAnyScroll returns true if layout has
func (ly *Layout) HasAnyScroll() bool {
	return ly.HasScroll[mat32.X] || ly.HasScroll[mat32.Y]
}

// SetScroll sets a scrollbar along given dimension
func (ly *Layout) SetScroll(d mat32.Dims) {
	if ly.Scrolls[d] == nil {
		ly.Scrolls[d] = &ScrollBar{}
		sc := ly.Scrolls[d]
		sc.InitName(sc, fmt.Sprintf("Scroll%v", d))
		ki.SetParent(sc, ly.This())
		sc.Dim = d
		sc.Init2D()
		sc.Defaults()
		sc.Tracking = true
		sc.Min = 0.0
	}
	spc := ly.BoxSpace()
	avail := ly.AvailSize().SubScalar(spc * 2.0)
	sc := ly.Scrolls[d]
	if d == mat32.X {
		sc.SetFixedHeight(ly.Sty.Layout.ScrollBarWidth)
		sc.SetFixedWidth(units.NewValue(avail.Dim(d), units.Dot))
	} else {
		sc.SetFixedWidth(ly.Sty.Layout.ScrollBarWidth)
		sc.SetFixedHeight(units.NewValue(avail.Dim(d), units.Dot))
	}
	sc.Style2D()
	sc.Max = ly.ChildSize.Dim(d) + ly.ExtraSize.Dim(d) // only scrollbar
	sc.Step = ly.Sty.Font.Size.Dots                    // step by lines
	sc.PageStep = 10.0 * sc.Step                       // todo: more dynamic
	sc.ThumbVal = avail.Dim(d) - spc
	sc.TrackThr = sc.Step
	sc.Value = mat32.Min(sc.Value, sc.Max-sc.ThumbVal) // keep in range
	// fmt.Printf("set sc lay: %v  max: %v  val: %v\n", ly.Path(), sc.Max, sc.Value)
	sc.SliderSig.ConnectOnly(ly.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig != int64(SliderValueChanged) {
			return
		}
		li, _ := KiToNode2D(recv)
		ls := li.AsLayout2D()
		// if ls.IsUpdating() {
		// 	fmt.Printf("Layout: %v scroll signal while still in update\n", ly.Path())
		// }
		wupdt := ls.TopUpdateStart()
		ls.Move2DTree()
		ls.ViewportSafe().ReRender2DNode(li)
		ls.TopUpdateEnd(wupdt)
	})
}

// DeleteScroll deletes scrollbar along given dimesion.  todo: we are leaking
// the scrollbars -- move into a container Field
func (ly *Layout) DeleteScroll(d mat32.Dims) {
	if ly.Scrolls[d] == nil {
		return
	}
	sc := ly.Scrolls[d]
	sc.DisconnectAllEvents(AllPris)
	sc.This().Destroy()
	ly.Scrolls[d] = nil
}

// DeactivateScroll turns off given scrollbar, without deleting, so it can be easily re-used
func (ly *Layout) DeactivateScroll(sc *ScrollBar) {
	sc.BBoxMu.Lock()
	defer sc.BBoxMu.Unlock()
	sc.LayState.Alloc.Pos = mat32.Vec2Zero
	sc.LayState.Alloc.Size = mat32.Vec2Zero
	sc.VpBBox = image.ZR
	sc.WinBBox = image.ZR
}

// LayoutScrolls arranges scrollbars
func (ly *Layout) LayoutScrolls() {
	sbw := ly.Sty.Layout.ScrollBarWidth.Dots

	spc := ly.BoxSpace()
	avail := ly.AvailSize()
	for d := mat32.X; d <= mat32.Y; d++ {
		odim := mat32.OtherDim(d)
		if ly.HasScroll[d] {
			sc := ly.Scrolls[d]
			sc.Size2D(0)
			sc.LayState.Alloc.PosRel.SetDim(d, spc)
			sc.LayState.Alloc.PosRel.SetDim(odim, avail.Dim(odim)-sbw-2.0)
			sc.LayState.Alloc.Size.SetDim(d, avail.Dim(d)-spc)
			if ly.HasScroll[odim] { // make room for other
				sc.LayState.Alloc.Size.SetSubDim(d, sbw)
			}
			sc.LayState.Alloc.Size.SetDim(odim, sbw)
			sc.Layout2D(ly.VpBBox, 0) // this will add parent position to above rel pos
		} else {
			if ly.Scrolls[d] != nil {
				ly.DeactivateScroll(ly.Scrolls[d])
			}
		}
	}
}

// RenderScrolls draws the scrollbars
func (ly *Layout) RenderScrolls() {
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.HasScroll[d] {
			ly.Scrolls[d].Render2D()
		}
	}
}

// ReRenderScrolls re-draws the scrollbars de-novo -- can be called ad-hoc by others
func (ly *Layout) ReRenderScrolls() {
	if ly.PushBounds() {
		ly.RenderScrolls()
		ly.PopBounds()
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

// Move2DScrolls moves scrollbars based on scrolling taking place in parent
// layouts -- critical to call this BEFORE we add our own delta, which is
// generated from these very same scrollbars.
func (ly *Layout) Move2DScrolls(delta image.Point, parBBox image.Rectangle) {
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.HasScroll[d] {
			ly.Scrolls[d].Move2D(delta, parBBox)
		}
	}
}

// ScrollActionDelta moves the scrollbar in given dimension by given delta
// and emits a ScrollSig signal.
func (ly *Layout) ScrollActionDelta(dim mat32.Dims, delta float32) {
	if ly.HasScroll[dim] {
		nval := ly.Scrolls[dim].Value + delta
		ly.Scrolls[dim].SetValueAction(nval)
		ly.ScrollSig.Emit(ly.This(), int64(dim), nval)
	}
}

// ScrollActionPos moves the scrollbar in given dimension to given
// position and emits a ScrollSig signal.
func (ly *Layout) ScrollActionPos(dim mat32.Dims, pos float32) {
	if ly.HasScroll[dim] {
		ly.Scrolls[dim].SetValueAction(pos)
		ly.ScrollSig.Emit(ly.This(), int64(dim), pos)
	}
}

// ScrollToPos moves the scrollbar in given dimension to given
// position and DOES NOT emit a ScrollSig signal.
func (ly *Layout) ScrollToPos(dim mat32.Dims, pos float32) {
	if ly.HasScroll[dim] {
		ly.Scrolls[dim].SetValueAction(pos)
	}
}

// ScrollDelta processes a scroll event.  If only one dimension is processed,
// and there is a non-zero in other, then the consumed dimension is reset to 0
// and the event is left unprocessed, so a higher level can consume the
// remainder.
func (ly *Layout) ScrollDelta(me *mouse.ScrollEvent) {
	del := me.Delta
	if ly.HasScroll[mat32.Y] && ly.HasScroll[mat32.X] {
		// fmt.Printf("ly: %v both del: %v\n", ly.Nm, del)
		ly.ScrollActionDelta(mat32.Y, float32(del.Y))
		ly.ScrollActionDelta(mat32.X, float32(del.X))
		me.SetProcessed()
	} else if ly.HasScroll[mat32.Y] {
		// fmt.Printf("ly: %v y del: %v\n", ly.Nm, del)
		ly.ScrollActionDelta(mat32.Y, float32(del.Y))
		if del.X != 0 {
			me.Delta.Y = 0
		} else {
			me.SetProcessed()
		}
	} else if ly.HasScroll[mat32.X] {
		// fmt.Printf("ly: %v x del: %v\n", ly.Nm, del)
		if del.X != 0 {
			ly.ScrollActionDelta(mat32.X, float32(del.X))
			if del.Y != 0 {
				me.Delta.X = 0
			} else {
				me.SetProcessed()
			}
		} else { // use Y instead as mouse wheels typically only have this
			hasShift := me.HasAnyModifier(key.Shift, key.Alt) // shift or alt says: use vert for other dimension
			if hasShift {
				ly.ScrollActionDelta(mat32.X, float32(del.Y))
				me.SetProcessed()
			}
		}
	}
}

// render the children
func (ly *Layout) Render2DChildren() {
	if ly.Lay == LayoutStacked {
		for i, kid := range ly.Kids {
			if _, ni := KiToNode2D(kid); ni != nil {
				if i == ly.StackTop {
					ni.ClearInvisible()
				} else {
					ni.SetInvisible()
				}
			}
		}
		// note: all nodes need to render to disconnect b/c of invisible
	}
	for _, kid := range ly.Kids {
		if kid == nil {
			continue
		}
		nii, _ := KiToNode2D(kid)
		if nii != nil {
			nii.Render2D()
		}
	}
}

func (ly *Layout) Move2DChildren(delta image.Point) {
	cbb := ly.This().(Node2D).ChildrenBBox2D()
	if ly.Lay == LayoutStacked {
		sn, err := ly.ChildTry(ly.StackTop)
		if err != nil {
			return
		}
		nii, _ := KiToNode2D(sn)
		nii.Move2D(delta, cbb)
	} else {
		for _, kid := range ly.Kids {
			nii, _ := KiToNode2D(kid)
			if nii != nil {
				nii.Move2D(delta, cbb)
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

	h := ly.Sty.Font.Size.Dots
	dst := h * AutoScrollRate

	mind := ints.MaxInt(0, pos-st)
	maxd := ints.MaxInt(0, (st+int(vissz))-pos)

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
	wbb := ly.WinBBox
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
	vpMin := ly.VpBBox.Min.X
	if dim == mat32.Y {
		vpMin = ly.VpBBox.Min.Y
	}
	sc := ly.Scrolls[dim]
	scrange := sc.Max - sc.ThumbVal // amount that can be scrolled
	vissz := sc.ThumbVal            // amount visible
	vpMax := vpMin + int(vissz)

	if minBox >= vpMin && maxBox <= vpMax {
		return false
	}

	h := ly.Sty.Font.Size.Dots

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
func (ly *Layout) ScrollToItem(ni Node2D) bool {
	return ly.ScrollToBox(ni.AsNode2D().ObjBBox)
}

// ScrollDimToStart scrolls to put the given child coordinate position (eg.,
// top / left of a view box) at the start (top / left) of our scroll area, to
// the extent possible -- returns true if scrolling was needed.
func (ly *Layout) ScrollDimToStart(dim mat32.Dims, pos int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
	vpMin := ly.VpBBox.Min.X
	if dim == mat32.Y {
		vpMin = ly.VpBBox.Min.Y
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
	vpMin := ly.VpBBox.Min.X
	if dim == mat32.Y {
		vpMin = ly.VpBBox.Min.Y
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
	vpMin := ly.VpBBox.Min.X
	if dim == mat32.Y {
		vpMin = ly.VpBBox.Min.Y
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
	em := ly.EventMgr2D()
	if em == nil {
		return nil, -1
	}
	for i, k := range ly.Kids {
		if k == nil {
			continue
		}
		_, ni := KiToNode2D(k)
		if ni == nil {
			continue
		}
		if ni.ContainsFocus() {
			return k, i
		}
	}
	return nil, -1
}

// FocusNextChild attempts to move the focus into the next layout child (with
// wraparound to start) -- returns true if successful
func (ly *Layout) FocusNextChild(updn bool) bool {
	sz := len(ly.Kids)
	if sz <= 1 {
		return false
	}
	foc, idx := ly.ChildWithFocus()
	if foc == nil {
		return false
	}
	em := ly.EventMgr2D()
	cur := em.CurFocus()
	nxti := idx + 1
	if ly.Lay == LayoutGrid && updn {
		nxti = idx + ly.Sty.Layout.Columns
	}
	did := false
	if nxti < sz {
		did = em.FocusOnOrNext(ly.Child(nxti))
	} else {
		did = em.FocusOnOrNext(ly.Child(0))
	}
	if !did || em.CurFocus() == cur {
		return false
	}
	return true
}

// FocusPrevChild attempts to move the focus into the previous layout child
// (with wraparound to end) -- returns true if successful
func (ly *Layout) FocusPrevChild(updn bool) bool {
	sz := len(ly.Kids)
	if sz <= 1 {
		return false
	}
	foc, idx := ly.ChildWithFocus()
	if foc == nil {
		return false
	}
	em := ly.EventMgr2D()
	cur := em.CurFocus()
	nxti := idx - 1
	if ly.Lay == LayoutGrid && updn {
		nxti = idx - ly.Sty.Layout.Columns
	}
	did := false
	if nxti >= 0 {
		did = em.FocusOnOrPrev(ly.Child(nxti))
	} else {
		did = em.FocusOnOrPrev(ly.Child(sz - 1))
	}
	if !did || em.CurFocus() == cur {
		return false
	}
	return true
}

// LayoutPageSteps is the number of steps to take in PageUp / Down events
// in terms of number of items.
var LayoutPageSteps = 10

// LayoutKeys is key processing for layouts -- focus name and arrow keys
func (ly *Layout) LayoutKeys(kt *key.ChordEvent) {
	if KeyEventTrace {
		fmt.Printf("Layout KeyInput: %v\n", ly.Path())
	}
	kf := KeyFun(kt.Chord())
	if ly.Lay == LayoutHoriz || ly.Lay == LayoutGrid || ly.Lay == LayoutHorizFlow {
		switch kf {
		case KeyFunMoveRight:
			if ly.FocusNextChild(false) { // allow higher layers to try..
				kt.SetProcessed()
			}
			return
		case KeyFunMoveLeft:
			if ly.FocusPrevChild(false) {
				kt.SetProcessed()
			}
			return
		}
	}
	if ly.Lay == LayoutVert || ly.Lay == LayoutGrid || ly.Lay == LayoutVertFlow {
		switch kf {
		case KeyFunMoveDown:
			if ly.FocusNextChild(true) {
				kt.SetProcessed()
			}
			return
		case KeyFunMoveUp:
			if ly.FocusPrevChild(true) {
				kt.SetProcessed()
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
				kt.SetProcessed()
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
				kt.SetProcessed()
			}
			return
		}
	}
	if nf, err := ly.PropTry("no-focus-name"); err == nil {
		if nf.(bool) {
			return
		}
	}
	ly.FocusOnName(kt)
}

// FocusOnName processes key events to look for an element starting with given name
func (ly *Layout) FocusOnName(kt *key.ChordEvent) bool {
	if KeyEventTrace {
		fmt.Printf("Layout FocusOnName: %v\n", ly.Path())
	}
	kf := KeyFun(kt.Chord())
	delayMs := int(kt.Time().Sub(ly.FocusNameTime) / time.Millisecond)
	ly.FocusNameTime = kt.Time()
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
		if !unicode.IsPrint(kt.Rune) || kt.Modifiers != 0 {
			return false
		}
		sr := string(kt.Rune)
		if ly.FocusName == sr {
			// re-search same letter
		} else {
			ly.FocusName += sr
			ly.FocusNameLast = nil // only use last if tabbing
		}
	}
	kt.SetProcessed()
	// fmt.Printf("searching for: %v  last: %v\n", ly.FocusName, ly.FocusNameLast)
	focel, found := ChildByLabelStartsCanFocus(ly, ly.FocusName, ly.FocusNameLast)
	if found {
		em := ly.EventMgr2D()
		if em != nil {
			em.SetFocus(focel) // this will also scroll by default!
		}
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
	ly.FuncDownBreadthFirst(0, nil, func(k ki.Ki, level int, data interface{}) bool {
		if k == ly.This() { // skip us
			return ki.Continue
		}
		_, ni := KiToNode2D(k)
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

// LayoutScrollEvents registers scrolling-related mouse events processed by
// Layout -- most subclasses of Layout will want these..
func (ly *Layout) LayoutScrollEvents() {
	// LowPri to allow other focal widgets to capture
	ly.ConnectEvent(oswin.MouseScrollEvent, LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.ScrollEvent)
		li := recv.Embed(KiT_Layout).(*Layout)
		li.ScrollDelta(me)
	})
	// HiPri to do it first so others can be in view etc -- does NOT consume event!
	ly.ConnectEvent(oswin.DNDMoveEvent, HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*dnd.MoveEvent)
		li := recv.Embed(KiT_Layout).(*Layout)
		li.AutoScroll(me.Pos())
	})
	ly.ConnectEvent(oswin.MouseMoveEvent, HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.MoveEvent)
		li := recv.Embed(KiT_Layout).(*Layout)
		if li.ViewportSafe().IsMenu() {
			li.AutoScroll(me.Pos())
		}
	})
}

// KeyChordEvent processes (lowpri) layout key events
func (ly *Layout) KeyChordEvent() {
	// LowPri to allow other focal widgets to capture
	ly.ConnectEvent(oswin.KeyChordEvent, LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		li := recv.Embed(KiT_Layout).(*Layout)
		kt := d.(*key.ChordEvent)
		li.LayoutKeys(kt)
	})
}

///////////////////////////////////////////////////
//   Standard Node2D interface

func (ly *Layout) AsLayout2D() *Layout {
	return ly
}

func (ly *Layout) Init2D() {
	ly.Init2DWidget()
}

func (ly *Layout) BBox2D() image.Rectangle {
	return ly.BBoxFromAlloc()
}

func (ly *Layout) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	ly.ComputeBBox2DBase(parBBox, delta)
}

func (ly *Layout) ChildrenBBox2D() image.Rectangle {
	nb := ly.ChildrenBBox2DWidget()
	nb.Max.X -= int(ly.ExtraSize.X)
	nb.Max.Y -= int(ly.ExtraSize.Y)
	return nb
}

// StyleFromProps styles Layout-specific fields from ki.Prop properties
// doesn't support inherit or default
func (ly *Layout) StyleFromProps(props ki.Props, vp *Viewport2D) {
	keys := []string{"lay", "spacing"}
	for _, key := range keys {
		val, has := props[key]
		if !has {
			continue
		}
		switch key {
		case "lay":
			switch vt := val.(type) {
			case string:
				ly.Lay.FromString(vt)
			case Layouts:
				ly.Lay = vt
			default:
				if iv, ok := kit.ToInt(val); ok {
					ly.Lay = Layouts(iv)
				} else {
					gist.StyleSetError(key, val)
				}
			}
		case "spacing":
			ly.Spacing.SetIFace(val, key)
		}
	}
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (ly *Layout) StyleToDots(uc *units.Context) {
	ly.Spacing.ToDots(uc)
}

// StyleLayout does layout styling -- it sets the StyMu Lock
func (ly *Layout) StyleLayout() {
	ly.StyMu.Lock()
	defer ly.StyMu.Unlock()

	// pr := prof.Start("StyleLayout")
	// defer pr.End()

	hasTempl, saveTempl := ly.Sty.FromTemplate()
	if !hasTempl || saveTempl {
		ly.Style2DWidget()
	}
	ly.StyleFromProps(ly.Props, ly.Viewport)           // does "lay" and "spacing", in layoutstyles.go
	tprops := *kit.Types.Properties(ki.Type(ly), true) // true = makeNew
	if len(tprops) > 0 {
		kit.TypesMu.RLock()
		ly.StyleFromProps(tprops, ly.Viewport)
		kit.TypesMu.RUnlock()
	}
	ly.StyleToDots(&ly.Sty.UnContext)
	if hasTempl && saveTempl {
		ly.Sty.SaveTemplate()
	}
}

func (ly *Layout) Style2D() {
	ly.StyleLayout()
	ly.StyMu.Lock()
	ly.LayState.SetFromStyle(&ly.Sty.Layout) // also does reset
	ly.StyMu.Unlock()
}

func (ly *Layout) Size2D(iter int) {
	ly.InitLayout2D()
	switch ly.Lay {
	case LayoutHorizFlow, LayoutVertFlow:
		GatherSizesFlow(ly, iter)
	case LayoutGrid:
		GatherSizesGrid(ly)
	default:
		GatherSizes(ly)
	}
}

func (ly *Layout) Layout2D(parBBox image.Rectangle, iter int) bool {
	//if iter > 0 {
	//	if Layout2DTrace {
	//		fmt.Printf("Layout: %v Iteration: %v  NeedsRedo: %v\n", ly.Path(), iter, ly.NeedsRedo)
	//	}
	//}
	LayAllocFromParent(ly)               // in case we didn't get anything
	ly.Layout2DBase(parBBox, true, iter) // init style
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
	ly.ManageOverflow()
	ly.NeedsRedo = ly.Layout2DChildren(iter) // layout done with canonical positions

	if !ly.NeedsRedo || iter == 1 {
		delta := ly.Move2DDelta(image.ZP)
		if delta != image.ZP {
			ly.Move2DChildren(delta) // move is a separate step
		}
	}
	return ly.NeedsRedo
}

// we add our own offset here
func (ly *Layout) Move2DDelta(delta image.Point) image.Point {
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

func (ly *Layout) Move2D(delta image.Point, parBBox image.Rectangle) {
	ly.Move2DBase(delta, parBBox)
	ly.Move2DScrolls(delta, parBBox) // move scrolls BEFORE adding our own!
	delta = ly.Move2DDelta(delta)    // add our offset
	ly.Move2DChildren(delta)
	ly.RenderScrolls()
}

func (ly *Layout) Render2D() {
	if ly.FullReRenderIfNeeded() {
		return
	}
	if ly.PushBounds() {
		ly.This().(Node2D).ConnectEvents2D()
		if ly.ScrollsOff {
			ly.ManageOverflow()
		}
		ly.RenderScrolls()
		ly.Render2DChildren()
		ly.PopBounds()
	} else {
		ly.SetScrollsOff()
		ly.DisconnectAllEvents(AllPris) // uses both Low and Hi
	}
}

func (ly *Layout) ConnectEvents2D() {
	if ly.HasAnyScroll() {
		ly.LayoutScrollEvents()
	}
	ly.KeyChordEvent()
}

func (ly *Layout) HasFocus2D() bool {
	if ly.IsInactive() {
		return false
	}
	return ly.ContainsFocus() // needed for getting key events
}

///////////////////////////////////////////////////////////
//    Stretch and Space -- dummy elements for layouts

// Stretch adds an infinitely stretchy element for spacing out layouts
// (max-size = -1) set the width / height property to determine how much it
// takes relative to other stretchy elements
type Stretch struct {
	WidgetBase
}

var KiT_Stretch = kit.Types.AddType(&Stretch{}, StretchProps)

// AddNewStretch adds a new stretch to given parent node, with given name.
func AddNewStretch(parent ki.Ki, name string) *Stretch {
	return parent.AddNewChild(KiT_Stretch, name).(*Stretch)
}

func (st *Stretch) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Stretch)
	st.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
}

var StretchProps = ki.Props{
	"EnumType:Flag": KiT_NodeFlags,
	"max-width":     -1.0,
	"max-height":    -1.0,
}

func (st *Stretch) Style2D() {
	st.StyMu.Lock()
	defer st.StyMu.Unlock()

	hasTempl, saveTempl := st.Sty.FromTemplate()
	if !hasTempl || saveTempl {
		st.Style2DWidget()
	}
	if hasTempl && saveTempl {
		st.Sty.SaveTemplate()
	}
	st.LayState.SetFromStyle(&st.Sty.Layout) // also does reset
}

func (st *Stretch) Layout2D(parBBox image.Rectangle, iter int) bool {
	st.Layout2DBase(parBBox, true, iter) // init style
	return st.Layout2DChildren(iter)
}

// Space adds a fixed sized (1 ch x 1 em by default) blank space to a layout -- set
// width / height property to change
type Space struct {
	WidgetBase
}

var KiT_Space = kit.Types.AddType(&Space{}, SpaceProps)

// AddNewSpace adds a new space to given parent node, with given name.
func AddNewSpace(parent ki.Ki, name string) *Space {
	return parent.AddNewChild(KiT_Space, name).(*Space)
}

func (sp *Space) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Space)
	sp.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
}

var SpaceProps = ki.Props{
	"EnumType:Flag": KiT_NodeFlags,
	"width":         units.NewCh(1),
	"height":        units.NewEm(1),
}

func (sp *Space) Style2D() {
	sp.StyMu.Lock()
	defer sp.StyMu.Unlock()

	hasTempl, saveTempl := sp.Sty.FromTemplate()
	if !hasTempl || saveTempl {
		sp.Style2DWidget()
	}
	if hasTempl && saveTempl {
		sp.Sty.SaveTemplate()
	}
	sp.LayState.SetFromStyle(&sp.Sty.Layout) // also does reset
}

func (sp *Space) Layout2D(parBBox image.Rectangle, iter int) bool {
	sp.Layout2DBase(parBBox, true, iter) // init style
	return sp.Layout2DChildren(iter)
}
