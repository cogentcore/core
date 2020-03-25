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

	"github.com/chewxy/math32"
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
	Size  SizePrefs    `desc:"size constraints for this item -- from layout style and "`
	Alloc LayoutAllocs `desc:"allocated size and position -- set by parent Layout"`
}

// todo: not using yet:
// Margins Margins   `desc:"margins around this item"`
// GridPos      image.Point `desc:"position within a grid"`
// GridSpan     image.Point `desc:"number of grid elements that we take up in each direction"`

func (ld *LayoutState) Defaults() {
}

func (ld *LayoutState) SetFromStyle(ls *LayoutStyle) {
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

// Layout is the primary node type responsible for organizing the sizes and
// positions of child widgets -- all arbitrary collections of widgets should
// generally be contained within a layout -- otherwise the parent widget must
// take over responsibility for positioning.  The alignment is NOT inherited
// by default so must be specified per child, except that the parent alignment
// is used within the relevant dimension (e.g., horizontal-align for a LayoutHoriz
// layout, to determine left, right, center, justified).  Layouts
// can automatically add scrollbars depending on the Overflow layout style.
type Layout struct {
	WidgetBase
	Lay           Layouts             `xml:"lay" desc:"type of layout to use"`
	Spacing       units.Value         `xml:"spacing" desc:"extra space to add between elements in the layout"`
	StackTop      int                 `desc:"for Stacked layout, index of node to use as the top of the stack -- only node at this index is rendered -- if not a valid index, nothing is rendered"`
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

func (nb *Layout) CopyFieldsFrom(frm interface{}) {
	fr, ok := frm.(*Layout)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier Layout one\n", nb.Type().Name())
		ki.GenCopyFieldsFrom(nb.This(), frm)
		return
	}
	nb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	nb.Lay = fr.Lay
	nb.Spacing = fr.Spacing
	nb.StackTop = fr.StackTop
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

var KiT_Layouts = kit.Enums.AddEnumAltLower(LayoutsN, kit.NotBitFlag, StylePropProps, "Layout")

func (ev Layouts) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Layouts) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// row / col for grid data
type RowCol int32

const (
	Row RowCol = iota
	Col
	RowColN
)

var KiT_RowCol = kit.Enums.AddEnumAltLower(RowColN, kit.NotBitFlag, StylePropProps, "")

func (ev RowCol) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *RowCol) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

//go:generate stringer -type=RowCol

// LayoutDefault is default obj that can be used when property specifies "default"
var LayoutDefault Layout

// SumDim returns whether we sum up elements along given dimension?  else use
// max for shared dimension.
func (ly *Layout) SumDim(d mat32.Dims) bool {
	if (d == mat32.X && (ly.Lay == LayoutHoriz || ly.Lay == LayoutHorizFlow)) || (d == mat32.Y && (ly.Lay == LayoutVert || ly.Lay == LayoutVertFlow)) {
		return true
	}
	return false
}

// SummedDim returns the dimension along which layout is summing.
func (ly *Layout) SummedDim() mat32.Dims {
	if ly.Lay == LayoutHoriz || ly.Lay == LayoutHorizFlow {
		return mat32.X
	}
	return mat32.Y
}

////////////////////////////////////////////////////////////////////////////////////////
//     Gather Sizes

// first depth-first Size2D pass: terminal concrete items compute their AllocSize
// we focus on Need: Max(Min, AllocSize), and Want: Max(Pref, AllocSize) -- Max is
// only used if we need to fill space, during final allocation
//
// second me-first Layout2D pass: each layout allocates AllocSize for its
// children based on aggregated size data, and so on down the tree

// GatherSizesSumMax gets basic sum and max data across all kiddos
func (ly *Layout) GatherSizesSumMax() (sumPref, sumNeed, maxPref, maxNeed mat32.Vec2) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		ni.LayState.UpdateSizes()
		sumNeed = sumNeed.Add(ni.LayState.Size.Need)
		sumPref = sumPref.Add(ni.LayState.Size.Pref)
		maxNeed = maxNeed.Max(ni.LayState.Size.Need)
		maxPref = maxPref.Max(ni.LayState.Size.Pref)

		if Layout2DTrace {
			fmt.Printf("Size:   %v Child: %v, need: %v, pref: %v\n", ly.PathUnique(), ni.UniqueNm, ni.LayState.Size.Need.Dim(ly.SummedDim()), ni.LayState.Size.Pref.Dim(ly.SummedDim()))
		}
	}
	return
}

// GatherSizes is size first pass: gather the size information from the children
func (ly *Layout) GatherSizes() {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	sumPref, sumNeed, maxPref, maxNeed := ly.GatherSizesSumMax()

	prefSizing := false
	if ly.Viewport != nil && ly.Viewport.HasFlag(int(VpFlagPrefSizing)) {
		prefSizing = ly.Sty.Layout.Overflow == OverflowScroll // special case
	}

	for d := mat32.X; d <= mat32.Y; d++ {
		pref := ly.LayState.Size.Pref.Dim(d)
		if prefSizing || pref == 0 {
			if ly.SumDim(d) { // our layout now updated to sum
				ly.LayState.Size.Need.SetMaxDim(d, sumNeed.Dim(d))
				ly.LayState.Size.Pref.SetMaxDim(d, sumPref.Dim(d))
			} else { // use max for other dir
				ly.LayState.Size.Need.SetMaxDim(d, maxNeed.Dim(d))
				ly.LayState.Size.Pref.SetMaxDim(d, maxPref.Dim(d))
			}
		} else { // use target size from style
			if Layout2DTrace {
				fmt.Printf("Size:   %v pref nonzero, setting as need: %v\n", ly.PathUnique(), pref)
			}
			ly.LayState.Size.Need.SetDim(d, pref)
		}
	}

	spc := ly.Sty.BoxSpace()
	ly.LayState.Size.Need.SetAddScalar(2.0 * spc)
	ly.LayState.Size.Pref.SetAddScalar(2.0 * spc)

	elspc := float32(0.0)
	if sz >= 2 {
		elspc = float32(sz-1) * ly.Spacing.Dots
	}
	if ly.SumDim(mat32.X) {
		ly.LayState.Size.Need.X += elspc
		ly.LayState.Size.Pref.X += elspc
	}
	if ly.SumDim(mat32.Y) {
		ly.LayState.Size.Need.Y += elspc
		ly.LayState.Size.Pref.Y += elspc
	}

	ly.LayState.UpdateSizes() // enforce max and normal ordering, etc
	if Layout2DTrace {
		fmt.Printf("Size:   %v gather sizes need: %v, pref: %v, elspc: %v\n", ly.PathUnique(), ly.LayState.Size.Need, ly.LayState.Size.Pref, elspc)
	}
}

// ChildrenUpdateSizes calls UpdateSizes on all children -- layout must at least call this
func (ly *Layout) ChildrenUpdateSizes() {
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		ni.LayState.UpdateSizes()
	}
}

// GatherSizesFlow is size first pass: gather the size information from the children
func (ly *Layout) GatherSizesFlow(iter int) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	if iter > 0 {
		ly.ChildrenUpdateSizes() // essential to call this
		prv := ly.ChildSize
		ly.LayState.Size.Need = prv
		ly.LayState.Size.Pref = prv
		ly.LayState.Alloc.Size = prv
		ly.LayState.UpdateSizes() // enforce max and normal ordering, etc
		if Layout2DTrace {
			fmt.Printf("Size:   %v iter 1 fix size: %v\n", ly.PathUnique(), prv)
		}
		return
	}

	sumPref, sumNeed, maxPref, maxNeed := ly.GatherSizesSumMax()

	// for flow, the need is always *maxNeed* (i.e., a single item)
	// and the pref is based on styled pref estimate

	sdim := ly.SummedDim()
	odim := mat32.OtherDim(sdim)
	pref := ly.LayState.Size.Pref.Dim(sdim)
	// opref := ly.LayState.Size.Pref.Dim(odim) // not using

	if pref == 0 {
		pref = 6 * maxNeed.Dim(sdim) // 6 items preferred backup default
	}
	if pref == 0 {
		pref = 200 // final backstop
	}

	if Layout2DTrace {
		fmt.Printf("Size:   %v flow pref start: %v\n", ly.PathUnique(), pref)
	}

	sNeed := sumNeed.Dim(sdim)
	nNeed := float32(1)
	tNeed := sNeed
	oNeed := maxNeed.Dim(odim)
	if sNeed > pref {
		tNeed = pref
		nNeed = mat32.Ceil(sNeed / tNeed)
		oNeed *= nNeed
	}

	sPref := sumPref.Dim(sdim)
	nPref := float32(1)
	tPref := sPref
	oPref := maxPref.Dim(odim)
	if sPref > pref {
		if sNeed > pref {
			tPref = pref
			nPref = mat32.Ceil(sPref / tPref)
			oPref *= nPref
		} else {
			tPref = sNeed // go with need
		}
	}

	ly.LayState.Size.Need.SetMaxDim(sdim, maxNeed.Dim(sdim)) // Need min = max of single item
	ly.LayState.Size.Pref.SetMaxDim(sdim, tPref)
	ly.LayState.Size.Need.SetMaxDim(odim, oNeed)
	ly.LayState.Size.Pref.SetMaxDim(odim, oPref)

	spc := ly.Sty.BoxSpace()
	ly.LayState.Size.Need.SetAddScalar(2.0 * spc)
	ly.LayState.Size.Pref.SetAddScalar(2.0 * spc)

	elspc := float32(0.0)
	if sz >= 2 {
		elspc = float32(sz-1) * ly.Spacing.Dots
	}
	if ly.SumDim(mat32.X) {
		ly.LayState.Size.Need.X += elspc
		ly.LayState.Size.Pref.X += elspc
	}
	if ly.SumDim(mat32.Y) {
		ly.LayState.Size.Need.Y += elspc
		ly.LayState.Size.Pref.Y += elspc
	}

	ly.LayState.UpdateSizes() // enforce max and normal ordering, etc
	if Layout2DTrace {
		fmt.Printf("Size:   %v gather sizes need: %v, pref: %v, elspc: %v\n", ly.PathUnique(), ly.LayState.Size.Need, ly.LayState.Size.Pref, elspc)
	}
}

// todo: grid does not process spans at all yet -- assumes = 1

// GatherSizesGrid is size first pass: gather the size information from the
// children, grid version
func (ly *Layout) GatherSizesGrid() {
	if len(ly.Kids) == 0 {
		return
	}

	cols := ly.Sty.Layout.Columns
	rows := 0

	sz := len(ly.Kids)
	// collect overall size
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		lst := ni.Sty.Layout
		if lst.Col > 0 {
			cols = ints.MaxInt(cols, lst.Col+lst.ColSpan)
		}
		if lst.Row > 0 {
			rows = ints.MaxInt(rows, lst.Row+lst.RowSpan)
		}
	}

	if cols == 0 {
		cols = int(math32.Sqrt(float32(sz))) // whatever -- not well defined
	}
	if rows == 0 {
		rows = sz / cols
	}
	for rows*cols < sz { // not defined to have multiple items per cell -- make room for everyone
		rows++
	}

	ly.GridSize.X = cols
	ly.GridSize.Y = rows

	if len(ly.GridData[Row]) != rows {
		ly.GridData[Row] = make([]GridData, rows)
	}
	if len(ly.GridData[Col]) != cols {
		ly.GridData[Col] = make([]GridData, cols)
	}

	for i := range ly.GridData[Row] {
		gd := &ly.GridData[Row][i]
		gd.SizeNeed = 0
		gd.SizePref = 0
	}
	for i := range ly.GridData[Col] {
		gd := &ly.GridData[Col][i]
		gd.SizeNeed = 0
		gd.SizePref = 0
	}

	col := 0
	row := 0
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		ni.LayState.UpdateSizes()
		lst := ni.Sty.Layout
		if lst.Col > 0 {
			col = lst.Col
		}
		if lst.Row > 0 {
			row = lst.Row
		}
		// r   0   1   col X = max(ea in col) (Y = not used)
		//   +--+---+
		// 0 |  |   |  row Y = max(ea in row) (X = not used)
		//   +--+---+
		// 1 |  |   |
		//   +--+---+

		rgd := &(ly.GridData[Row][row])
		cgd := &(ly.GridData[Col][col])

		// todo: need to deal with span in sums..
		mat32.SetMax(&(rgd.SizeNeed), ni.LayState.Size.Need.Y)
		mat32.SetMax(&(rgd.SizePref), ni.LayState.Size.Pref.Y)
		mat32.SetMax(&(cgd.SizeNeed), ni.LayState.Size.Need.X)
		mat32.SetMax(&(cgd.SizePref), ni.LayState.Size.Pref.X)

		// for max: any -1 stretch dominates, else accumulate any max
		if rgd.SizeMax >= 0 {
			if ni.LayState.Size.Max.Y < 0 { // stretch
				rgd.SizeMax = -1
			} else {
				mat32.SetMax(&(rgd.SizeMax), ni.LayState.Size.Max.Y)
			}
		}
		if cgd.SizeMax >= 0 {
			if ni.LayState.Size.Max.Y < 0 { // stretch
				cgd.SizeMax = -1
			} else {
				mat32.SetMax(&(cgd.SizeMax), ni.LayState.Size.Max.X)
			}
		}

		col++
		if col >= cols { // todo: really only works if NO items specify row,col or ALL do..
			col = 0
			row++
			if row >= rows { // wrap-around.. no other good option
				row = 0
			}
		}
	}

	prefSizing := false
	if ly.Viewport != nil && ly.Viewport.HasFlag(int(VpFlagPrefSizing)) {
		prefSizing = ly.Sty.Layout.Overflow == OverflowScroll // special case
	}

	// if there aren't existing prefs, we need to compute size
	if prefSizing || ly.LayState.Size.Pref.X == 0 || ly.LayState.Size.Pref.Y == 0 {
		sbw := ly.Sty.Layout.ScrollBarWidth.Dots
		maxRow := len(ly.GridData[Row])
		maxCol := len(ly.GridData[Col])
		if prefSizing {
			maxRow = ints.MinInt(LayoutPrefMaxRows, maxRow)
			maxCol = ints.MinInt(LayoutPrefMaxCols, maxCol)
		}

		// Y = sum across rows which have max's
		var sumPref, sumNeed mat32.Vec2
		for i := 0; i < maxRow; i++ {
			gd := ly.GridData[Row][i]
			sumNeed.SetAddDim(mat32.Y, gd.SizeNeed)
			sumPref.SetAddDim(mat32.Y, gd.SizePref)
		}
		// X = sum across cols which have max's
		for i := 0; i < maxCol; i++ {
			gd := ly.GridData[Col][i]
			sumNeed.SetAddDim(mat32.X, gd.SizeNeed)
			sumPref.SetAddDim(mat32.X, gd.SizePref)
		}

		if prefSizing {
			sumNeed.X += sbw
			sumNeed.Y += sbw
			sumPref.X += sbw
			sumPref.Y += sbw
		}

		if ly.LayState.Size.Pref.X == 0 || prefSizing {
			ly.LayState.Size.Need.X = mat32.Max(ly.LayState.Size.Need.X, sumNeed.X)
			ly.LayState.Size.Pref.X = mat32.Max(ly.LayState.Size.Pref.X, sumPref.X)
		} else { // use target size from style otherwise
			ly.LayState.Size.Need.X = ly.LayState.Size.Pref.X
		}
		if ly.LayState.Size.Pref.Y == 0 || prefSizing {
			ly.LayState.Size.Need.Y = mat32.Max(ly.LayState.Size.Need.Y, sumNeed.Y)
			ly.LayState.Size.Pref.Y = mat32.Max(ly.LayState.Size.Pref.Y, sumPref.Y)
		} else { // use target size from style otherwise
			ly.LayState.Size.Need.Y = ly.LayState.Size.Pref.Y
		}
	} else { // neither are zero so we use those directly
		ly.LayState.Size.Need = ly.LayState.Size.Pref
	}

	spc := ly.Sty.BoxSpace()
	ly.LayState.Size.Need.SetAddScalar(2.0 * spc)
	ly.LayState.Size.Pref.SetAddScalar(2.0 * spc)

	ly.LayState.Size.Need.X += float32(cols-1) * ly.Spacing.Dots
	ly.LayState.Size.Pref.X += float32(cols-1) * ly.Spacing.Dots
	ly.LayState.Size.Need.Y += float32(rows-1) * ly.Spacing.Dots
	ly.LayState.Size.Pref.Y += float32(rows-1) * ly.Spacing.Dots

	ly.LayState.UpdateSizes() // enforce max and normal ordering, etc
	if Layout2DTrace {
		fmt.Printf("Size:   %v gather sizes grid need: %v, pref: %v\n", ly.PathUnique(), ly.LayState.Size.Need, ly.LayState.Size.Pref)
	}
}

// AllocFromParent: if we are not a child of a layout, then get allocation
// from a parent obj that has a layout size
func (ly *Layout) AllocFromParent() {
	if ly.Par == nil || ly.Viewport == nil || !ly.LayState.Alloc.Size.IsNil() {
		return
	}
	if ly.Par != ly.Viewport.This() {
		// note: zero alloc size happens all the time with non-visible tabs!
		// fmt.Printf("Layout: %v has zero allocation but is not a direct child of viewport -- this is an error -- every level must provide layout for the next! laydata:\n%+v\n", ly.PathUnique(), ly.LayState)
		return
	}
	pni, _ := KiToNode2D(ly.Par)
	lyp := pni.AsLayout2D()
	if lyp == nil {
		ly.FuncUpParent(0, ly.This(), func(k ki.Ki, level int, d interface{}) bool {
			pni, _ := KiToNode2D(k)
			if pni == nil {
				return false
			}
			pg := pni.AsWidget()
			if pg == nil {
				return false
			}
			if !pg.LayState.Alloc.Size.IsNil() {
				ly.LayState.Alloc.Size = pg.LayState.Alloc.Size
				if Layout2DTrace {
					fmt.Printf("Layout: %v got parent alloc: %v from %v\n", ly.PathUnique(), ly.LayState.Alloc.Size, pg.PathUnique())
				}
				return false
			}
			return true
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//     Layout children

// LayoutSharedDim implements calculations to layout for the shared dimension
// (i.e., Vertical for Horizontal layout). Returns pos and size.
func (ly *Layout) LayoutSharedDimImpl(avail, need, pref, max, spc float32, al Align) (pos, size float32) {
	usePref := true
	targ := pref
	extra := avail - targ
	if extra < -0.1 { // not fitting in pref, go with min
		usePref = false
		targ = need
		extra = avail - targ
	}
	extra = mat32.Max(extra, 0.0) // no negatives

	stretchNeed := false // stretch relative to need
	stretchMax := false  // only stretch Max = neg

	if usePref && extra > 0.0 { // have some stretch extra
		if max < 0.0 {
			stretchMax = true // only stretch those marked as infinitely stretchy
		}
	} else if extra > 0.0 { // extra relative to Need
		stretchNeed = true // stretch relative to need
	}

	pos = spc
	size = need
	if usePref {
		size = pref
	}
	if stretchMax || stretchNeed {
		size += extra
	} else {
		if IsAlignMiddle(al) {
			pos += 0.5 * extra
		} else if IsAlignEnd(al) {
			pos += extra
		} else if al == AlignJustify { // treat justify as stretch
			size += extra
		}
	}

	// if Layout2DTrace {
	// 	fmt.Printf("ly %v avail: %v targ: %v, extra %v, strMax: %v, strNeed: %v, pos: %v size: %v spc: %v\n", ly.Nm, avail, targ, extra, stretchMax, stretchNeed, pos, size, spc)
	// }

	return
}

// LayoutSharedDim lays out items along a shared dimension, where all elements
// share the same space, e.g., Horiz for a Vert layout, and vice-versa.
func (ly *Layout) LayoutSharedDim(dim mat32.Dims) {
	spc := ly.Sty.BoxSpace()
	avail := ly.LayState.Alloc.Size.Dim(dim) - 2.0*spc
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		al := ni.Sty.Layout.AlignDim(dim)
		pref := ni.LayState.Size.Pref.Dim(dim)
		need := ni.LayState.Size.Need.Dim(dim)
		max := ni.LayState.Size.Max.Dim(dim)
		pos, size := ly.LayoutSharedDimImpl(avail, need, pref, max, spc, al)
		ni.LayState.Alloc.Size.SetDim(dim, size)
		ni.LayState.Alloc.PosRel.SetDim(dim, pos)
	}
}

// LayoutAlongDim lays out all children along given dim -- only affects that dim --
// e.g., use LayoutSharedDim for other dim.
func (ly *Layout) LayoutAlongDim(dim mat32.Dims) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	elspc := float32(sz-1) * ly.Spacing.Dots
	al := ly.Sty.Layout.AlignDim(dim)
	spc := ly.Sty.BoxSpace()
	exspc := 2.0*spc + elspc
	avail := ly.LayState.Alloc.Size.Dim(dim) - exspc
	pref := ly.LayState.Size.Pref.Dim(dim) - exspc
	need := ly.LayState.Size.Need.Dim(dim) - exspc

	targ := pref
	usePref := true
	extra := avail - targ
	if extra < -0.1 { // not fitting in pref, go with need
		usePref = false
		targ = need
		extra = avail - targ
	}
	extra = mat32.Max(extra, 0.0) // no negatives

	nstretch := 0
	stretchTot := float32(0.0)
	stretchNeed := false        // stretch relative to need
	stretchMax := false         // only stretch Max = neg
	addSpace := false           // apply extra toward spacing -- for justify
	if usePref && extra > 0.0 { // have some stretch extra
		for _, c := range ly.Kids {
			if c == nil {
				continue
			}
			ni := c.(Node2D).AsWidget()
			if ni == nil {
				continue
			}
			if ni.LayState.Size.HasMaxStretch(dim) { // negative = stretch
				nstretch++
				stretchTot += ni.LayState.Size.Pref.Dim(dim)
			}
		}
		if nstretch > 0 {
			stretchMax = true // only stretch those marked as infinitely stretchy
		}
	} else if extra > 0.0 { // extra relative to Need
		for _, c := range ly.Kids {
			if c == nil {
				continue
			}
			ni := c.(Node2D).AsWidget()
			if ni == nil {
				continue
			}
			if ni.LayState.Size.HasMaxStretch(dim) || ni.LayState.Size.CanStretchNeed(dim) {
				nstretch++
				stretchTot += ni.LayState.Size.Pref.Dim(dim)
			}
		}
		if nstretch > 0 {
			stretchNeed = true // stretch relative to need
		}
	}

	extraSpace := float32(0.0)
	if sz > 1 && extra > 0.0 && al == AlignJustify && !stretchNeed && !stretchMax {
		addSpace = true
		// if neither, then just distribute as spacing for justify
		extraSpace = extra / float32(sz-1)
	}

	// now arrange everyone
	pos := spc

	// todo: need a direction setting too
	if IsAlignEnd(al) && !stretchNeed && !stretchMax {
		pos += extra
	}

	if Layout2DTrace {
		fmt.Printf("Layout: %v Along dim %v, avail: %v elspc: %v need: %v pref: %v targ: %v, extra %v, strMax: %v, strNeed: %v, nstr %v, strTot %v\n", ly.PathUnique(), dim, avail, elspc, need, pref, targ, extra, stretchMax, stretchNeed, nstretch, stretchTot)
	}

	for i, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		size := ni.LayState.Size.Need.Dim(dim)
		if usePref {
			size = ni.LayState.Size.Pref.Dim(dim)
		}
		if stretchMax { // negative = stretch
			if ni.LayState.Size.HasMaxStretch(dim) { // in proportion to pref
				size += extra * (ni.LayState.Size.Pref.Dim(dim) / stretchTot)
			}
		} else if stretchNeed {
			if ni.LayState.Size.HasMaxStretch(dim) || ni.LayState.Size.CanStretchNeed(dim) {
				size += extra * (ni.LayState.Size.Pref.Dim(dim) / stretchTot)
			}
		} else if addSpace { // implies align justify
			if i > 0 {
				pos += extraSpace
			}
		}

		ni.LayState.Alloc.Size.SetDim(dim, size)
		ni.LayState.Alloc.PosRel.SetDim(dim, pos)
		if Layout2DTrace {
			fmt.Printf("Layout: %v Child: %v, pos: %v, size: %v, need: %v, pref: %v\n", ly.PathUnique(), ni.UniqueNm, pos, size, ni.LayState.Size.Need.Dim(dim), ni.LayState.Size.Pref.Dim(dim))
		}
		pos += size + ly.Spacing.Dots
	}
}

// LayoutFlow manages the flow layout along given dimension
// returns true if needs another iteration (only if iter == 0)
func (ly *Layout) LayoutFlow(dim mat32.Dims, iter int) bool {
	ly.FlowBreaks = nil
	sz := len(ly.Kids)
	if sz == 0 {
		return false
	}

	elspc := float32(sz-1) * ly.Spacing.Dots
	spc := ly.Sty.BoxSpace()
	exspc := 2.0*spc + elspc

	avail := ly.LayState.Alloc.Size.Dim(dim) - exspc
	odim := mat32.OtherDim(dim)

	pos := spc
	for i, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		size := ni.LayState.Size.Need.Dim(dim)
		if pos+size > avail {
			ly.FlowBreaks = append(ly.FlowBreaks, i)
			pos = spc
		}
		ni.LayState.Alloc.Size.SetDim(dim, size)
		ni.LayState.Alloc.PosRel.SetDim(dim, pos)
		if Layout2DTrace {
			fmt.Printf("Layout: %v Child: %v, pos: %v, size: %v, need: %v, pref: %v\n", ly.PathUnique(), ni.UniqueNm, pos, size, ni.LayState.Size.Need.Dim(dim), ni.LayState.Size.Pref.Dim(dim))
		}
		pos += size + ly.Spacing.Dots
	}
	ly.FlowBreaks = append(ly.FlowBreaks, len(ly.Kids))

	nrows := len(ly.FlowBreaks)
	oavail := ly.LayState.Alloc.Size.Dim(odim) - exspc
	oavPerRow := oavail / float32(nrows)
	ci := 0
	rpos := float32(0)
	var nsz mat32.Vec2
	for _, bi := range ly.FlowBreaks {
		rmax := float32(0)
		for i := ci; i < bi; i++ {
			c := ly.Kids[i]
			if c == nil {
				continue
			}
			ni := c.(Node2D).AsWidget()
			if ni == nil {
				continue
			}
			al := ni.Sty.Layout.AlignDim(odim)
			pref := ni.LayState.Size.Pref.Dim(odim)
			need := ni.LayState.Size.Need.Dim(odim)
			max := ni.LayState.Size.Max.Dim(odim)
			pos, size := ly.LayoutSharedDimImpl(oavPerRow, need, pref, max, spc, al)
			ni.LayState.Alloc.Size.SetDim(odim, size)
			ni.LayState.Alloc.PosRel.SetDim(odim, rpos+pos)
			rmax = mat32.Max(rmax, size)
			nsz.X = mat32.Max(nsz.X, ni.LayState.Alloc.PosRel.X+ni.LayState.Alloc.Size.X)
			nsz.Y = mat32.Max(nsz.Y, ni.LayState.Alloc.PosRel.Y+ni.LayState.Alloc.Size.Y)
		}
		rpos += rmax + ly.Spacing.Dots
		ci = bi
	}
	ly.LayState.Size.Need = nsz
	ly.LayState.Size.Pref = nsz
	if Layout2DTrace {
		fmt.Printf("Layout: %v Flow final size: %v\n", ly.PathUnique(), nsz)
	}
	// if nrows == 1 {
	// 	return false
	// }
	return true
}

// LayoutGridDim lays out grid data along each dimension (row, Y; col, X),
// same as LayoutAlongDim.  For cols, X has width prefs of each -- turn that
// into an actual allocated width for each column, and likewise for rows.
func (ly *Layout) LayoutGridDim(rowcol RowCol, dim mat32.Dims) {
	gds := ly.GridData[rowcol]
	sz := len(gds)
	if sz == 0 {
		return
	}
	elspc := float32(sz-1) * ly.Spacing.Dots
	al := ly.Sty.Layout.AlignDim(dim)
	spc := ly.Sty.BoxSpace()
	exspc := 2.0*spc + elspc
	avail := ly.LayState.Alloc.Size.Dim(dim) - exspc
	pref := ly.LayState.Size.Pref.Dim(dim) - exspc
	need := ly.LayState.Size.Need.Dim(dim) - exspc

	targ := pref
	usePref := true
	extra := avail - targ
	if extra < -0.1 { // not fitting in pref, go with need
		usePref = false
		targ = need
		extra = avail - targ
	}
	extra = mat32.Max(extra, 0.0) // no negatives

	nstretch := 0
	stretchTot := float32(0.0)
	stretchNeed := false        // stretch relative to need
	stretchMax := false         // only stretch Max = neg
	addSpace := false           // apply extra toward spacing -- for justify
	if usePref && extra > 0.0 { // have some stretch extra
		for _, gd := range gds {
			if gd.SizeMax < 0 { // stretch
				nstretch++
				stretchTot += gd.SizePref
			}
		}
		if nstretch > 0 {
			stretchMax = true // only stretch those marked as infinitely stretchy
		}
	} else if extra > 0.0 { // extra relative to Need
		for _, gd := range gds {
			if gd.SizeMax < 0 || gd.SizePref > gd.SizeNeed {
				nstretch++
				stretchTot += gd.SizePref
			}
		}
		if nstretch > 0 {
			stretchNeed = true // stretch relative to need
		}
	}

	extraSpace := float32(0.0)
	if sz > 1 && extra > 0.0 && al == AlignJustify && !stretchNeed && !stretchMax {
		addSpace = true
		// if neither, then just distribute as spacing for justify
		extraSpace = extra / float32(sz-1)
	}

	// now arrange everyone
	pos := spc

	// todo: need a direction setting too
	if IsAlignEnd(al) && !stretchNeed && !stretchMax {
		pos += extra
	}

	if Layout2DTrace {
		fmt.Printf("Layout Grid Dim: %v All on dim %v, avail: %v need: %v pref: %v targ: %v, extra %v, strMax: %v, strNeed: %v, nstr %v, strTot %v\n", ly.PathUnique(), dim, avail, need, pref, targ, extra, stretchMax, stretchNeed, nstretch, stretchTot)
	}

	for i := range gds {
		gd := &gds[i]
		size := gd.SizeNeed
		if usePref {
			size = gd.SizePref
		}
		if stretchMax { // negative = stretch
			if gd.SizeMax < 0 { // in proportion to pref
				size += extra * (gd.SizePref / stretchTot)
			}
		} else if stretchNeed {
			if gd.SizeMax < 0 || gd.SizePref > gd.SizeNeed {
				size += extra * (gd.SizePref / stretchTot)
			}
		} else if addSpace { // implies align justify
			if i > 0 {
				pos += extraSpace
			}
		}

		gd.AllocSize = size
		gd.AllocPosRel = pos
		if Layout2DTrace {
			fmt.Printf("Grid %v pos: %v, size: %v\n", rowcol, pos, size)
		}
		pos += size + ly.Spacing.Dots
	}
}

// LayoutGrid manages overall grid layout of children
func (ly *Layout) LayoutGrid() {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	ly.LayoutGridDim(Row, mat32.Y)
	ly.LayoutGridDim(Col, mat32.X)

	col := 0
	row := 0
	cols := ly.GridSize.X
	rows := ly.GridSize.Y

	if cols*rows != ly.NumChildren() {
		ly.GatherSizesGrid()
	}

	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}

		lst := ni.Sty.Layout
		if lst.Col > 0 {
			col = lst.Col
		}
		if lst.Row > 0 {
			row = lst.Row
		}

		{ // col, X dim
			dim := mat32.X
			gd := ly.GridData[Col][col]
			avail := gd.AllocSize
			al := lst.AlignDim(dim)
			pref := ni.LayState.Size.Pref.Dim(dim)
			need := ni.LayState.Size.Need.Dim(dim)
			max := ni.LayState.Size.Max.Dim(dim)
			pos, size := ly.LayoutSharedDimImpl(avail, need, pref, max, 0, al)
			ni.LayState.Alloc.Size.SetDim(dim, size)
			ni.LayState.Alloc.PosRel.SetDim(dim, pos+gd.AllocPosRel)

		}
		{ // row, Y dim
			dim := mat32.Y
			gd := ly.GridData[Row][row]
			avail := gd.AllocSize
			al := lst.AlignDim(dim)
			pref := ni.LayState.Size.Pref.Dim(dim)
			need := ni.LayState.Size.Need.Dim(dim)
			max := ni.LayState.Size.Max.Dim(dim)
			pos, size := ly.LayoutSharedDimImpl(avail, need, pref, max, 0, al)
			ni.LayState.Alloc.Size.SetDim(dim, size)
			ni.LayState.Alloc.PosRel.SetDim(dim, pos+gd.AllocPosRel)
		}

		if Layout2DTrace {
			fmt.Printf("Layout: %v grid col: %v row: %v pos: %v size: %v\n", ly.PathUnique(), col, row, ni.LayState.Alloc.PosRel, ni.LayState.Alloc.Size)
		}

		col++
		if col >= cols { // todo: really only works if NO items specify row,col or ALL do..
			col = 0
			row++
			if row >= rows { // wrap-around.. no other good option
				row = 0
			}
		}
	}
}

// FinalizeLayout is final pass through children to finalize the layout,
// computing summary size stats
func (ly *Layout) FinalizeLayout() {
	ly.ChildSize = mat32.Vec2Zero
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		ly.ChildSize.SetMax(ni.LayState.Alloc.PosRel.Add(ni.LayState.Alloc.Size))
		ni.LayState.Alloc.SizeOrig = ni.LayState.Alloc.Size
	}
}

// AvailSize returns the total size avail to this layout -- typically
// AllocSize except for top-level layout which uses VpBBox in case less is
// avail
func (ly *Layout) AvailSize() mat32.Vec2 {
	spc := ly.Sty.BoxSpace()
	avail := ly.LayState.Alloc.Size.SubScalar(spc) // spc is for right size space
	parni, _ := KiToNode2D(ly.Par)
	if parni != nil {
		vp := parni.AsViewport2D()
		if vp != nil {
			if vp.Viewport == nil {
				avail = mat32.NewVec2FmPoint(ly.VpBBox.Size()).SubScalar(spc)
				// fmt.Printf("non-nil par ly: %v vp: %v %v\n", ly.PathUnique(), vp.PathUnique(), avail)
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

	if ly.Sty.Layout.Overflow != OverflowHidden {
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
		sc.SetParent(ly.This())
		sc.Dim = d
		sc.Init2D()
		sc.Defaults()
		sc.Tracking = true
		sc.Min = 0.0
	}
	spc := ly.Sty.BoxSpace()
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
	// fmt.Printf("set sc lay: %v  max: %v  val: %v\n", ly.PathUnique(), sc.Max, sc.Value)
	sc.SliderSig.ConnectOnly(ly.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig != int64(SliderValueChanged) {
			return
		}
		li, _ := KiToNode2D(recv)
		ls := li.AsLayout2D()
		if !ls.IsUpdating() {
			wupdt := ls.TopUpdateStart()
			ls.Move2DTree()
			ls.Viewport.ReRender2DNode(li)
			ls.TopUpdateEnd(wupdt)
			// } else {
			// 	fmt.Printf("not ready to update\n")
		}
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
	sc.LayState.Alloc.Pos = mat32.Vec2Zero
	sc.LayState.Alloc.Size = mat32.Vec2Zero
	sc.VpBBox = image.ZR
	sc.WinBBox = image.ZR
}

// LayoutScrolls arranges scrollbars
func (ly *Layout) LayoutScrolls() {
	sbw := ly.Sty.Layout.ScrollBarWidth.Dots

	spc := ly.Sty.BoxSpace()
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
			// fmt.Printf("turning scroll off for :%v dim: %v\n", ly.PathUnique(), d)
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
		sn := ly.Child(ly.StackTop)
		if sn == nil {
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
	did := false
	if ly.HasScroll[mat32.Y] && ly.HasScroll[mat32.X] {
		did = ly.AutoScrollDim(mat32.Y, ly.WinBBox.Min.Y, pos.Y)
		did = did || ly.AutoScrollDim(mat32.X, ly.WinBBox.Min.X, pos.X)
	} else if ly.HasScroll[mat32.Y] {
		did = ly.AutoScrollDim(mat32.Y, ly.WinBBox.Min.Y, pos.Y)
	} else if ly.HasScroll[mat32.X] {
		did = ly.AutoScrollDim(mat32.X, ly.WinBBox.Min.X, pos.X)
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
		fmt.Printf("Layout KeyInput: %v\n", ly.PathUnique())
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
		fmt.Printf("Layout FocusOnName: %v\n", ly.PathUnique())
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
	focel, found := ly.ChildByLabelStartsCanFocus(ly.FocusName, ly.FocusNameLast)
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
func (ly *Layout) ChildByLabelStartsCanFocus(name string, after ki.Ki) (ki.Ki, bool) {
	lcnm := strings.ToLower(name)
	var rki ki.Ki
	gotAfter := false
	ly.FuncDownBreadthFirst(0, nil, func(k ki.Ki, level int, data interface{}) bool {
		if k == ly.This() { // skip us
			return true
		}
		_, ni := KiToNode2D(k)
		if ni != nil && !ni.CanFocus() { // don't go any further
			return false
		}
		if after != nil && !gotAfter {
			if k == after {
				gotAfter = true
			}
			return true // skip to next
		}
		kn := strings.ToLower(ToLabel(k))
		if rki == nil && strings.HasPrefix(kn, lcnm) {
			rki = k
			return false
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
		if li.Viewport.IsMenu() {
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

func (ly *Layout) StyleLayout() {
	// pr := prof.Start("StyleLayout")
	// defer pr.End()
	hasTempl, saveTempl := ly.Sty.FromTemplate()
	if !hasTempl || saveTempl {
		ly.Style2DWidget()
	}
	ly.StyleFromProps(ly.Props, ly.Viewport)         // does "lay" and "spacing", in layoutstyles.go
	tprops := *kit.Types.Properties(ly.Type(), true) // true = makeNew
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
	ly.LayState.SetFromStyle(&ly.Sty.Layout) // also does reset
}

func (ly *Layout) Size2D(iter int) {
	ly.InitLayout2D()
	switch ly.Lay {
	case LayoutHorizFlow, LayoutVertFlow:
		ly.GatherSizesFlow(iter)
	case LayoutGrid:
		ly.GatherSizesGrid()
	default:
		ly.GatherSizes()
	}
}

func (ly *Layout) Layout2D(parBBox image.Rectangle, iter int) bool {
	//if iter > 0 {
	//	if Layout2DTrace {
	//		fmt.Printf("Layout: %v Iteration: %v  NeedsRedo: %v\n", ly.PathUnique(), iter, ly.NeedsRedo)
	//	}
	//}
	ly.AllocFromParent()                 // in case we didn't get anything
	ly.Layout2DBase(parBBox, true, iter) // init style
	redo := false
	switch ly.Lay {
	case LayoutHoriz:
		ly.LayoutAlongDim(mat32.X)
		ly.LayoutSharedDim(mat32.Y)
	case LayoutVert:
		ly.LayoutAlongDim(mat32.Y)
		ly.LayoutSharedDim(mat32.X)
	case LayoutGrid:
		ly.LayoutGrid()
	case LayoutStacked:
		ly.LayoutSharedDim(mat32.X)
		ly.LayoutSharedDim(mat32.Y)
	case LayoutHorizFlow:
		redo = ly.LayoutFlow(mat32.X, iter)
	case LayoutVertFlow:
		redo = ly.LayoutFlow(mat32.Y, iter)
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

func (nb *Stretch) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Stretch)
	nb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
}

var StretchProps = ki.Props{
	"EnumType:Flag": KiT_NodeFlags,
	"max-width":     -1.0,
	"max-height":    -1.0,
}

func (st *Stretch) Style2D() {
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

func (nb *Space) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Space)
	nb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
}

var SpaceProps = ki.Props{
	"EnumType:Flag": KiT_NodeFlags,
	"width":         units.NewCh(1),
	"height":        units.NewEm(1),
}

func (sp *Space) Style2D() {
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
