// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/kit"
)

// all different types of alignment -- only some are applicable to different
// contexts, but there is also so much overlap that it makes sense to have
// them all in one list -- some are not standard CSS and used by layout
type Align int32

const (
	AlignLeft Align = iota
	AlignTop
	AlignCenter
	// middle = vertical version of center
	AlignMiddle
	AlignRight
	AlignBottom
	AlignBaseline
	// same as CSS space-between
	AlignJustify
	AlignSpaceAround
	AlignFlexStart
	AlignFlexEnd
	AlignTextTop
	AlignTextBottom
	// align to subscript
	AlignSub
	// align to superscript
	AlignSuper
	AlignN
)

//go:generate stringer -type=Align

var KiT_Align = kit.Enums.AddEnumAltLower(AlignN, false, StylePropProps, "Align")

// is this a generalized alignment to start of container?
func IsAlignStart(a Align) bool {
	return (a == AlignLeft || a == AlignTop || a == AlignFlexStart || a == AlignTextTop)
}

// is this a generalized alignment to middle of container?
func IsAlignMiddle(a Align) bool {
	return (a == AlignCenter || a == AlignMiddle)
}

// is this a generalized alignment to end of container?
func IsAlignEnd(a Align) bool {
	return (a == AlignRight || a == AlignBottom || a == AlignFlexEnd || a == AlignTextBottom)
}

// overflow type -- determines what happens when there is too much stuff in a layout
type Overflow int32

const (
	// automatically add scrollbars as needed -- this is pretty much the only sensible option, and is the default here, but Visible is default in html
	OverflowAuto Overflow = iota
	// pretty much the same as auto -- we treat it as such
	OverflowScroll
	// make the overflow visible -- this is generally unsafe and not very feasible and will be ignored as long as possible -- currently falls back on auto, but could go to Hidden if that works better overall
	OverflowVisible
	// hide the overflow and don't present scrollbars (supported)
	OverflowHidden
	OverflowN
)

var KiT_Overflow = kit.Enums.AddEnumAltLower(OverflowN, false, StylePropProps, "Overflow")

//go:generate stringer -type=Overflow

// todo: for style
// Align = layouts
// Flex -- flexbox -- https://www.w3schools.com/css/css3_flexbox.asp -- key to look at further for layout ideas
// as is Position -- absolute, sticky, etc
// Resize: user-resizability
// z-index

// CSS vs. Layout alignment
//
// CSS has align-self, align-items (for a container, provides a default for
// items) and align-content which only applies to lines in a flex layout (akin
// to a flow layout) -- there is a presumed horizontal aspect to these, except
// align-content, so they are subsumed in the AlignH parameter in this style.
// Vertical-align works as expected, and Text.Align uses left/center/right
//
// LayoutRow, Col both allow explicit Top/Left Center/Middle, Right/Bottom alignment
// along with Justify and SpaceAround -- they use IsAlign functions

// style preferences on the layout of the element
type LayoutStyle struct {
	z_index   int           `xml:"z-index" desc:"ordering factor for rendering depth -- lower numbers rendered first -- sort children according to this factor"`
	AlignH    Align         `xml:"align-self" alt:"horiz-align,align-horiz" desc:"horizontal alignment -- for widget layouts -- not a standard css property"`
	AlignV    Align         `xml:"vertical-align" alt:"vert-align,align-vert" desc:"vertical alignment -- for widget layouts -- not a standard css property"`
	PosX      units.Value   `xml:"x" desc:"horizontal position -- often superceded by layout but otherwise used"`
	PosY      units.Value   `xml:"y" desc:"vertical position -- often superceded by layout but otherwise used"`
	Width     units.Value   `xml:"width" desc:"specified size of element -- 0 if not specified"`
	Height    units.Value   `xml:"height" desc:"specified size of element -- 0 if not specified"`
	MaxWidth  units.Value   `xml:"max-width" desc:"specified maximum size of element -- 0  means just use other values, negative means stretch"`
	MaxHeight units.Value   `xml:"max-height" desc:"specified maximum size of element -- 0 means just use other values, negative means stretch"`
	MinWidth  units.Value   `xml:"min-width" desc:"specified mimimum size of element -- 0 if not specified"`
	MinHeight units.Value   `xml:"min-height" desc:"specified mimimum size of element -- 0 if not specified"`
	Offsets   []units.Value `xml:"{top,right,bottom,left}" desc:"specified offsets for each side"`
	Margin    units.Value   `xml:"margin" desc:"outer-most transparent space around box element -- todo: can be specified per side"`
	Padding   units.Value   `xml:"padding" desc:"transparent space around central content of box -- todo: if 4 values it is top, right, bottom, left; 3 is top, right&left, bottom; 2 is top & bottom, right and left"`
	Overflow  Overflow      `xml:"overflow" desc:"what to do with content that overflows -- default is Auto add of scrollbars as needed -- todo: can have separate -x -y values"`
	Columns   int           `xml:"columns" alt:"grid-cols" desc:"number of columns to use in a grid layout -- used as a constraint in layout if individual elements do not specify their row, column positions"`
	Row       int           `xml:"row" desc:"specifies the row that this element should appear within a grid layout"`
	Col       int           `xml:"col" desc:"specifies the column that this element should appear within a grid layout"`
	RowSpan   int           `xml:"row-span" desc:"specifies the number of sequential rows that this element should occupy within a grid layout (todo: not currently supported)"`
	ColSpan   int           `xml:"col-span" desc:"specifies the number of sequential columns that this element should occupy within a grid layout"`

	ScrollBarWidth units.Value `xml:"scrollbar-width" desc:"width of a layout scrollbar"`
}

func (ls *LayoutStyle) Defaults() {
	ls.MinWidth.Set(2.0, units.Px)
	ls.MinHeight.Set(2.0, units.Px)
	ls.ScrollBarWidth.Set(16.0, units.Px)
}

func (ls *LayoutStyle) SetStylePost() {
}

// return the alignment for given dimension
func (ls *LayoutStyle) AlignDim(d Dims2D) Align {
	switch d {
	case X:
		return ls.AlignH
	default:
		return ls.AlignV
	}
}

// position settings, in dots
func (ls *LayoutStyle) PosDots() Vec2D {
	return NewVec2D(ls.PosX.Dots, ls.PosY.Dots)
}

// size settings, in dots
func (ls *LayoutStyle) SizeDots() Vec2D {
	return NewVec2D(ls.Width.Dots, ls.Height.Dots)
}

// size max settings, in dots
func (ls *LayoutStyle) MaxSizeDots() Vec2D {
	return NewVec2D(ls.MaxWidth.Dots, ls.MaxHeight.Dots)
}

// size min settings, in dots
func (ls *LayoutStyle) MinSizeDots() Vec2D {
	return NewVec2D(ls.MinWidth.Dots, ls.MinHeight.Dots)
}

////////////////////////////////////////////////////////////////////////////////////////
// Layout Data for actually computing the layout

// size preferences
type SizePrefs struct {
	Need Vec2D `desc:"minimum size needed -- set to at least computed allocsize"`
	Pref Vec2D `desc:"preferred size -- start here for layout"`
	Max  Vec2D `desc:"maximum size -- will not be greater than this -- 0 = no constraint, neg = stretch"`
}

// return true if Max < 0 meaning can stretch infinitely along given dimension
func (sp SizePrefs) HasMaxStretch(d Dims2D) bool {
	return (sp.Max.Dim(d) < 0.0)
}

// return true if Pref > Need meaning can stretch more along given dimension
func (sp SizePrefs) CanStretchNeed(d Dims2D) bool {
	return (sp.Pref.Dim(d) > sp.Need.Dim(d))
}

// 2D margins
type Margins struct {
	left, right, top, bottom float64
}

// set a single margin for all items
func (m *Margins) SetMargin(marg float64) {
	m.left = marg
	m.right = marg
	m.top = marg
	m.bottom = marg
}

// LayoutData contains all the data needed to specify the layout of an item within a layout -- includes computed values of style prefs -- everything is concrete and specified here, whereas style may not be fully resolved
type LayoutData struct {
	Size         SizePrefs   `desc:"size constraints for this item -- from layout style"`
	Margins      Margins     `desc:"margins around this item"`
	GridPos      image.Point `desc:"position within a grid"`
	GridSpan     image.Point `desc:"number of grid elements that we take up in each direction"`
	AllocSize    Vec2D       `desc:"allocated size of this item, by the parent layout"`
	AllocPos     Vec2D       `desc:"position of this item, computed by adding in the AllocPosRel to parent position"`
	AllocPosRel  Vec2D       `desc:"allocated relative position of this item, computed by the parent layout"`
	AllocPosOrig Vec2D       `desc:"original copy of allocated relative position of this item, by the parent layout -- need for scrolling which can update AllocPos"`
}

func (ld *LayoutData) Defaults() {
	if ld.GridSpan.X < 1 {
		ld.GridSpan.X = 1
	}
	if ld.GridSpan.Y < 1 {
		ld.GridSpan.Y = 1
	}
}

func (ld *LayoutData) SetFromStyle(ls *LayoutStyle) {
	ld.Reset()
	// these are layout hints:
	ld.Size.Need = ls.MinSizeDots()
	ld.Size.Pref = ls.SizeDots()
	ld.Size.Max = ls.MaxSizeDots()

	// this is an actual initial desired setting
	ld.AllocPos = ls.PosDots()
	// not setting size, so we can keep that as a separate constraint
}

// called at start of layout process -- resets all values back to 0
func (ld *LayoutData) Reset() {
	ld.AllocSize = Vec2DZero
	ld.AllocPos = Vec2DZero
	ld.AllocPosRel = Vec2DZero
	ld.AllocPosOrig = Vec2DZero
}

// update our sizes based on AllocSize and Max constraints, etc
func (ld *LayoutData) UpdateSizes() {
	ld.Size.Need.SetMax(ld.AllocSize)   // min cannot be < alloc -- bare min
	ld.Size.Pref.SetMax(ld.Size.Need)   // pref cannot be < min
	ld.Size.Need.SetMinPos(ld.Size.Max) // min cannot be > max
	ld.Size.Pref.SetMinPos(ld.Size.Max) // pref cannot be > max
}

////////////////////////////////////////////////////////////////////////////////////////
//    Layout handles all major types of layout

// different types of layouts
type Layouts int32

const (
	// arrange items horizontally across a row
	LayoutRow Layouts = iota
	// arrange items vertically in a column
	LayoutCol
	// arrange items according to a grid
	LayoutGrid
	// arrange items horizontally across a row, overflowing vertically as needed
	LayoutRowFlow
	// arrange items vertically within a column, overflowing horizontally as needed
	LayoutColFlow
	// arrange items stacked on top of each other -- Top index indicates which to show -- overall size accommodates largest in each dimension
	LayoutStacked
	// nil layout -- don't do anything -- for cases when a parent wants to take over the job of the layout
	LayoutNil
	LayoutsN
)

//go:generate stringer -type=Layouts

// row / col for grid data
type RowCol int32

const (
	Row RowCol = iota
	Col
	RowColN
)

var KiT_RowCol = kit.Enums.AddEnumAltLower(RowColN, false, StylePropProps, "")

//go:generate stringer -type=RowCol

// note: Layout cannot be a Widget type because Controls in Widget is a Layout..

// Layout is the primary node type responsible for organizing the sizes and
// positions of child widgets -- all arbitrary collections of widgets should
// generally be contained within a layout -- otherwise the parent widget must
// take over responsibility for positioning.  The alignment is NOT inherited
// by default so must be specified per child, except that the parent alignment
// is used within the relevant dimension (e.g., align-horiz for a LayoutRow
// layout, to determine left, right, center, justified).  Layouts
// can automatically add scrollbars depending on the Overflow layout style
type Layout struct {
	Node2DBase
	Lay       Layouts               `xml:"lay" desc:"type of layout to use"`
	StackTop  ki.Ptr                `desc:"pointer to node to use as the top of the stack -- only node matching this pointer is rendered, even if this is nil"`
	ChildSize Vec2D                 `xml:"-" desc:"total max size of children as laid out"`
	ExtraSize Vec2D                 `xml:"-" desc:"extra size in each dim due to scrollbars we add"`
	HasScroll [Dims2DN]bool         `desc:"whether scrollbar is used for given dim"`
	Scrolls   [Dims2DN]*ScrollBar   `xml:"-" desc:"scroll bars -- we fully manage them as needed"`
	GridSize  image.Point           `desc:"computed size of a grid layout based on all the constraints -- computed during Size2D pass"`
	GridData  [RowColN][]LayoutData `json:"-" xml:"-" desc:"grid data for rows in [0] and cols in [1]"`
}

var KiT_Layout = kit.Types.AddType(&Layout{}, nil)

// do we sum up elements along given dimension?  else max
func (ly *Layout) SumDim(d Dims2D) bool {
	if (d == X && ly.Lay == LayoutRow) || (d == Y && ly.Lay == LayoutCol) {
		return true
	}
	return false
}

// first depth-first Size2D pass: terminal concrete items compute their AllocSize
// we focus on Need: Max(Min, AllocSize), and Want: Max(Pref, AllocSize) -- Max is
// only used if we need to fill space, during final allocation
//
// second me-first Layout2D pass: each layout allocates AllocSize for its
// children based on aggregated size data, and so on down the tree

// first pass: gather the size information from the children
func (ly *Layout) GatherSizes() {
	if len(ly.Kids) == 0 {
		return
	}

	var sumPref, sumNeed, maxPref, maxNeed Vec2D
	for _, c := range ly.Kids {
		_, gi := KiToNode2D(c)
		if gi == nil {
			continue
		}
		gi.LayData.UpdateSizes()
		sumNeed = sumNeed.Add(gi.LayData.Size.Need)
		sumPref = sumPref.Add(gi.LayData.Size.Pref)
		maxNeed = maxNeed.Max(gi.LayData.Size.Need)
		maxPref = maxPref.Max(gi.LayData.Size.Pref)
	}

	for d := X; d <= Y; d++ {
		if ly.SumDim(d) { // our layout now updated to sum
			ly.LayData.Size.Need.SetMaxDim(d, sumNeed.Dim(d))
			ly.LayData.Size.Pref.SetMaxDim(d, sumPref.Dim(d))
		} else { // use max for other dir
			ly.LayData.Size.Need.SetMaxDim(d, maxNeed.Dim(d))
			ly.LayData.Size.Pref.SetMaxDim(d, maxPref.Dim(d))
		}
	}

	spc := ly.Style.BoxSpace()
	ly.LayData.Size.Need.SetAddVal(2.0 * spc)
	ly.LayData.Size.Pref.SetAddVal(2.0 * spc)

	// todo: something entirely different needed for grids..

	ly.LayData.UpdateSizes() // enforce max and normal ordering, etc
	if Layout2DTrace {
		fmt.Printf("Size:   %v gather sizes need: %v, pref: %v\n", ly.PathUnique(), ly.LayData.Size.Need, ly.LayData.Size.Pref)
	}
}

// todo: grid does not process spans at all yet -- assumes = 1

// first pass: gather the size information from the children, grid version
func (ly *Layout) GatherSizesGrid() {
	if len(ly.Kids) == 0 {
		return
	}

	cols := ly.Style.Layout.Columns
	rows := 0

	sz := len(ly.Kids)
	// collect overal size
	for _, c := range ly.Kids {
		_, gi := KiToNode2D(c)
		if gi == nil {
			continue
		}
		lst := gi.Style.Layout
		if lst.Col > 0 {
			cols = kit.MaxInt(cols, lst.Col+lst.ColSpan)
		}
		if lst.Row > 0 {
			rows = kit.MaxInt(rows, lst.Row+lst.RowSpan)
		}
	}

	if cols == 0 {
		cols = int(math.Sqrt(float64(sz))) // whatever -- not well defined
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
		ly.GridData[Row] = make([]LayoutData, rows)
	}
	if len(ly.GridData[Col]) != cols {
		ly.GridData[Col] = make([]LayoutData, cols)
	}

	for i := range ly.GridData[Row] {
		ld := &ly.GridData[Row][i]
		ld.Size.Need.Set(0, 0)
		ld.Size.Pref.Set(0, 0)
	}
	for i := range ly.GridData[Col] {
		ld := &ly.GridData[Col][i]
		ld.Size.Need.Set(0, 0)
		ld.Size.Pref.Set(0, 0)
	}

	col := 0
	row := 0
	for _, c := range ly.Kids {
		_, gi := KiToNode2D(c)
		if gi == nil {
			continue
		}
		gi.LayData.UpdateSizes()
		lst := gi.Style.Layout
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

		// todo: need to deal with span in sums..
		ly.GridData[Row][row].Size.Need.SetMaxDim(Y, gi.LayData.Size.Need.Y)
		ly.GridData[Row][row].Size.Pref.SetMaxDim(Y, gi.LayData.Size.Pref.Y)
		ly.GridData[Col][col].Size.Need.SetMaxDim(X, gi.LayData.Size.Need.X)
		ly.GridData[Col][col].Size.Pref.SetMaxDim(X, gi.LayData.Size.Pref.X)

		// for max: any -1 stretch dominates, else accumulate any max
		if ly.GridData[Row][row].Size.Max.Y >= 0 {
			if gi.LayData.Size.Max.Y < 0 { // stretch
				ly.GridData[Row][row].Size.Max.Y = -1
			} else {
				ly.GridData[Row][row].Size.Max.SetMaxDim(Y, gi.LayData.Size.Max.Y)
			}
		}
		if ly.GridData[Col][col].Size.Max.X >= 0 {
			if gi.LayData.Size.Max.Y < 0 { // stretch
				ly.GridData[Col][col].Size.Max.X = -1
			} else {
				ly.GridData[Col][col].Size.Max.SetMaxDim(X, gi.LayData.Size.Max.X)
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

	// Y = sum across rows which have max's
	var sumPref, sumNeed Vec2D
	for _, ld := range ly.GridData[Row] {
		sumNeed.SetAddDim(Y, ld.Size.Need.Y)
		sumPref.SetAddDim(Y, ld.Size.Pref.Y)
	}
	// X = sum across cols which have max's
	for _, ld := range ly.GridData[Col] {
		sumNeed.SetAddDim(X, ld.Size.Need.X)
		sumPref.SetAddDim(X, ld.Size.Pref.X)
	}

	ly.LayData.Size.Need.SetMax(sumNeed)
	ly.LayData.Size.Pref.SetMax(sumPref)

	spc := ly.Style.BoxSpace()
	ly.LayData.Size.Need.SetAddVal(2.0 * spc)
	ly.LayData.Size.Pref.SetAddVal(2.0 * spc)

	ly.LayData.UpdateSizes() // enforce max and normal ordering, etc
	if Layout2DTrace {
		fmt.Printf("Size:   %v gather sizes grid need: %v, pref: %v\n", ly.PathUnique(), ly.LayData.Size.Need, ly.LayData.Size.Pref)
	}
}

// if we are not a child of a layout, then get allocation from a parent obj that
// has a layout size
func (ly *Layout) AllocFromParent() {
	if ly.Par == nil || !ly.LayData.AllocSize.IsZero() {
		return
	}
	pgi, _ := KiToNode2D(ly.Par)
	lyp := pgi.AsLayout2D()
	if lyp == nil {
		ly.FuncUpParent(0, ly.This, func(k ki.Ki, level int, d interface{}) bool {
			_, pg := KiToNode2D(k)
			if pg == nil {
				return false
			}
			if !pg.LayData.AllocSize.IsZero() {
				ly.LayData.AllocSize = pg.LayData.AllocSize
				if Layout2DTrace {
					fmt.Printf("Layout: %v got parent alloc: %v from %v\n", ly.PathUnique(), ly.LayData.AllocSize, pg.PathUnique())
				}
				return false
			}
			return true
		})
	}
}

// calculations to layout a single-element dimension, returns pos and size
func (ly *Layout) LayoutSingleImpl(avail, need, pref, max, spc float64, al Align) (pos, size float64) {
	usePref := true
	targ := pref
	extra := avail - targ
	if extra < -0.1 { // not fitting in pref, go with min
		usePref = false
		targ = need
		extra = avail - targ
	}
	extra = math.Max(extra, 0.0) // no negatives

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

// layout item in single-dimensional case -- e.g., orthogonal dimension from LayoutRow / Col
func (ly *Layout) LayoutSingle(dim Dims2D) {
	spc := ly.Style.BoxSpace()
	avail := ly.LayData.AllocSize.Dim(dim) - 2.0*spc
	for _, c := range ly.Kids {
		_, gi := KiToNode2D(c)
		if gi == nil {
			continue
		}
		al := gi.Style.Layout.AlignDim(dim)
		pref := gi.LayData.Size.Pref.Dim(dim)
		need := gi.LayData.Size.Need.Dim(dim)
		max := gi.LayData.Size.Max.Dim(dim)
		pos, size := ly.LayoutSingleImpl(avail, need, pref, max, spc, al)
		gi.LayData.AllocSize.SetDim(dim, size)
		gi.LayData.AllocPosRel.SetDim(dim, pos)
	}
}

// layout all children along given dim -- only affects that dim -- e.g., use
// LayoutSingle for other dim
func (ly *Layout) LayoutAll(dim Dims2D) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	al := ly.Style.Layout.AlignDim(dim)
	spc := ly.Style.BoxSpace()
	avail := ly.LayData.AllocSize.Dim(dim) - 2.0*spc
	pref := ly.LayData.Size.Pref.Dim(dim) - 2.0*spc
	need := ly.LayData.Size.Need.Dim(dim) - 2.0*spc

	targ := pref
	usePref := true
	extra := avail - targ
	if extra < -0.1 { // not fitting in pref, go with need
		usePref = false
		targ = need
		extra = avail - targ
	}
	extra = math.Max(extra, 0.0) // no negatives

	nstretch := 0
	stretchTot := 0.0
	stretchNeed := false        // stretch relative to need
	stretchMax := false         // only stretch Max = neg
	addSpace := false           // apply extra toward spacing -- for justify
	if usePref && extra > 0.0 { // have some stretch extra
		for _, c := range ly.Kids {
			_, gi := KiToNode2D(c)
			if gi == nil {
				continue
			}
			if gi.LayData.Size.HasMaxStretch(dim) { // negative = stretch
				nstretch++
				stretchTot += gi.LayData.Size.Pref.Dim(dim)
			}
		}
		if nstretch > 0 {
			stretchMax = true // only stretch those marked as infinitely stretchy
		}
	} else if extra > 0.0 { // extra relative to Need
		for _, c := range ly.Kids {
			_, gi := KiToNode2D(c)
			if gi == nil {
				continue
			}
			if gi.LayData.Size.HasMaxStretch(dim) || gi.LayData.Size.CanStretchNeed(dim) {
				nstretch++
				stretchTot += gi.LayData.Size.Pref.Dim(dim)
			}
		}
		if nstretch > 0 {
			stretchNeed = true // stretch relative to need
		}
	}

	extraSpace := 0.0
	if sz > 1 && extra > 0.0 && al == AlignJustify && !stretchNeed && !stretchMax {
		addSpace = true
		// if neither, then just distribute as spacing for justify
		extraSpace = extra / float64(sz-1)
	}

	// now arrange everyone
	pos := spc

	// todo: need a direction setting too
	if IsAlignEnd(al) && !stretchNeed && !stretchMax {
		pos += extra
	}

	if Layout2DTrace {
		fmt.Printf("Layout: %v All on dim %v, avail: %v need: %v pref: %v targ: %v, extra %v, strMax: %v, strNeed: %v, nstr %v, strTot %v\n", ly.PathUnique(), dim, avail, need, pref, targ, extra, stretchMax, stretchNeed, nstretch, stretchTot)
	}

	for i, c := range ly.Kids {
		_, gi := KiToNode2D(c)
		if gi == nil {
			continue
		}
		size := gi.LayData.Size.Need.Dim(dim)
		if usePref {
			size = gi.LayData.Size.Pref.Dim(dim)
		}
		if stretchMax { // negative = stretch
			if gi.LayData.Size.HasMaxStretch(dim) { // in proportion to pref
				size += extra * (gi.LayData.Size.Pref.Dim(dim) / stretchTot)
			}
		} else if stretchNeed {
			if gi.LayData.Size.HasMaxStretch(dim) || gi.LayData.Size.CanStretchNeed(dim) {
				size += extra * (gi.LayData.Size.Pref.Dim(dim) / stretchTot)
			}
		} else if addSpace { // implies align justify
			if i > 0 {
				pos += extraSpace
			}
		}

		gi.LayData.AllocSize.SetDim(dim, size)
		gi.LayData.AllocPosRel.SetDim(dim, pos)
		if Layout2DTrace {
			fmt.Printf("Layout: %v Child: %v, pos: %v, size: %v\n", ly.PathUnique(), gi.UniqueNm, pos, size)
		}
		pos += size
	}
}

// layout grid data along each dimension (row, Y; col, X), same as LayoutAll.
// For cols, X has width prefs of each -- turn that into an actual allocated
// width for each column, and likewise for rows.
func (ly *Layout) LayoutGridDim(rowcol RowCol, dim Dims2D) {
	gd := ly.GridData[rowcol]
	sz := len(gd)
	if sz == 0 {
		return
	}
	al := ly.Style.Layout.AlignDim(dim)
	spc := ly.Style.BoxSpace()
	avail := ly.LayData.AllocSize.Dim(dim) - 2.0*spc
	pref := ly.LayData.Size.Pref.Dim(dim) - 2.0*spc
	need := ly.LayData.Size.Need.Dim(dim) - 2.0*spc

	targ := pref
	usePref := true
	extra := avail - targ
	if extra < -0.1 { // not fitting in pref, go with need
		usePref = false
		targ = need
		extra = avail - targ
	}
	extra = math.Max(extra, 0.0) // no negatives

	nstretch := 0
	stretchTot := 0.0
	stretchNeed := false        // stretch relative to need
	stretchMax := false         // only stretch Max = neg
	addSpace := false           // apply extra toward spacing -- for justify
	if usePref && extra > 0.0 { // have some stretch extra
		for i := range gd {
			ld := &gd[i]
			if ld.Size.HasMaxStretch(dim) {
				nstretch++
				stretchTot += ld.Size.Pref.Dim(dim)
			}
		}
		if nstretch > 0 {
			stretchMax = true // only stretch those marked as infinitely stretchy
		}
	} else if extra > 0.0 { // extra relative to Need
		for i := range gd {
			ld := &gd[i]
			if ld.Size.HasMaxStretch(dim) || ld.Size.CanStretchNeed(dim) {
				nstretch++
				stretchTot += ld.Size.Pref.Dim(dim)
			}
		}
		if nstretch > 0 {
			stretchNeed = true // stretch relative to need
		}
	}

	extraSpace := 0.0
	if sz > 1 && extra > 0.0 && al == AlignJustify && !stretchNeed && !stretchMax {
		addSpace = true
		// if neither, then just distribute as spacing for justify
		extraSpace = extra / float64(sz-1)
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

	for i := range gd {
		ld := &gd[i]
		size := ld.Size.Need.Dim(dim)
		if usePref {
			size = ld.Size.Pref.Dim(dim)
		}
		if stretchMax { // negative = stretch
			if ld.Size.HasMaxStretch(dim) { // in proportion to pref
				size += extra * (ld.Size.Pref.Dim(dim) / stretchTot)
			}
		} else if stretchNeed {
			if ld.Size.HasMaxStretch(dim) || ld.Size.CanStretchNeed(dim) {
				size += extra * (ld.Size.Pref.Dim(dim) / stretchTot)
			}
		} else if addSpace { // implies align justify
			if i > 0 {
				pos += extraSpace
			}
		}

		ld.AllocSize.SetDim(dim, size)
		ld.AllocPosRel.SetDim(dim, pos)
		if Layout2DTrace {
			fmt.Printf("Grid %v Dim: %v, pos: %v, size: %v\n", rowcol, dim, pos, size)
		}
		pos += size
	}
}

func (ly *Layout) LayoutGrid() {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	ly.LayoutGridDim(Row, Y)
	ly.LayoutGridDim(Col, X)

	col := 0
	row := 0
	cols := ly.GridSize.X
	rows := ly.GridSize.Y
	for _, c := range ly.Kids {
		_, gi := KiToNode2D(c)
		if gi == nil {
			continue
		}

		lst := gi.Style.Layout
		if lst.Col > 0 {
			col = lst.Col
		}
		if lst.Row > 0 {
			row = lst.Row
		}

		{ // col, X dim
			dim := X
			ld := &(ly.GridData[Col][col])
			avail := ld.AllocSize.Dim(dim)
			al := lst.AlignDim(dim)
			pref := gi.LayData.Size.Pref.Dim(dim)
			need := gi.LayData.Size.Need.Dim(dim)
			max := gi.LayData.Size.Max.Dim(dim)
			pos, size := ly.LayoutSingleImpl(avail, need, pref, max, 0, al)
			gi.LayData.AllocSize.SetDim(dim, size)
			gi.LayData.AllocPosRel.SetDim(dim, pos+ld.AllocPosRel.X)

		}
		{ // row, Y dim
			dim := Y
			ld := &(ly.GridData[Row][row])
			avail := ld.AllocSize.Dim(dim)
			al := lst.AlignDim(dim)
			pref := gi.LayData.Size.Pref.Dim(dim)
			need := gi.LayData.Size.Need.Dim(dim)
			max := gi.LayData.Size.Max.Dim(dim)
			pos, size := ly.LayoutSingleImpl(avail, need, pref, max, 0, al)
			gi.LayData.AllocSize.SetDim(dim, size)
			gi.LayData.AllocPosRel.SetDim(dim, pos+ld.AllocPosRel.Y)
		}

		if Layout2DTrace {
			fmt.Printf("Layout: %v grid col: %v row: %v pos: %v size: %v\n", ly.PathUnique(), col, row, gi.LayData.AllocPosRel, gi.LayData.AllocSize)
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

// final pass through children to finalize the layout, computing summary size stats
func (ly *Layout) FinalizeLayout() {
	ly.ChildSize = Vec2DZero
	for _, c := range ly.Kids {
		_, gi := KiToNode2D(c)
		if gi == nil {
			continue
		}
		ly.ChildSize.SetMax(gi.LayData.AllocPosRel.Add(gi.LayData.AllocSize))
	}
}

// AvailSize returns the total size avail to this layout -- typically
// AllocSize except for top-level layout which uses VpBBox in case less is
// avail
func (ly *Layout) AvailSize() Vec2D {
	spc := ly.Style.BoxSpace()
	avail := ly.LayData.AllocSize.SubVal(spc)
	pargi, _ := KiToNode2D(ly.Par)
	if pargi != nil {
		vp := pargi.AsViewport2D()
		if vp != nil {
			if vp.Viewport == nil {
				avail = NewVec2DFmPoint(ly.VpBBox.Size()).SubVal(spc)
				// fmt.Printf("non-nil par ly: %v vp: %v %v\n", ly.PathUnique(), vp.PathUnique(), avail)
			}
		}
	}
	return avail
}

// process any overflow according to overflow settings
func (ly *Layout) ManageOverflow() {
	if len(ly.Kids) == 0 {
		return
	}
	avail := ly.AvailSize()

	ly.ExtraSize.SetVal(0.0)
	for d := X; d < Dims2DN; d++ {
		ly.HasScroll[d] = false
	}

	if ly.Style.Layout.Overflow != OverflowHidden {
		sbw := ly.Style.Layout.ScrollBarWidth.Dots
		for d := X; d < Dims2DN; d++ {
			odim := OtherDim(d)
			if ly.ChildSize.Dim(d) > (avail.Dim(d) + 2.0) { // overflowing -- allow some margin
				ly.HasScroll[d] = true
				ly.ExtraSize.SetAddDim(odim, sbw)
			}
		}
		for d := X; d < Dims2DN; d++ {
			if ly.HasScroll[d] {
				ly.SetScroll(d)
			}
		}
		ly.LayoutScrolls()
	}
}

func (ly *Layout) SetScroll(d Dims2D) {
	if ly.Scrolls[d] == nil {
		ly.Scrolls[d] = &ScrollBar{}
		sc := ly.Scrolls[d]
		sc.InitName(sc, fmt.Sprintf("Scroll%v", d))
		sc.SetParent(ly.This)
		sc.Dim = d
		sc.Init2D()
		sc.Defaults()
	}
	spc := ly.Style.BoxSpace()
	avail := ly.AvailSize().SubVal(spc * 2.0)
	sc := ly.Scrolls[d]
	if d == X {
		sc.SetFixedHeight(ly.Style.Layout.ScrollBarWidth)
		sc.SetFixedWidth(units.NewValue(avail.Dim(d), units.Dot))
	} else {
		sc.SetFixedWidth(ly.Style.Layout.ScrollBarWidth)
		sc.SetFixedHeight(units.NewValue(avail.Dim(d), units.Dot))
	}
	sc.Style2D()
	sc.Min = 0.0
	sc.Max = ly.ChildSize.Dim(d) + ly.ExtraSize.Dim(d) // only scrollbar
	sc.Step = ly.Style.Font.Size.Dots                  // step by lines
	sc.PageStep = 10.0 * sc.Step                       // todo: more dynamic
	sc.ThumbVal = avail.Dim(d) - spc
	sc.Tracking = true
	sc.TrackThr = sc.Step
	sc.SliderSig.Connect(ly.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig != int64(SliderValueChanged) {
			return
		}
		li, _ := KiToNode2D(recv) // note: avoid using closures
		ls := li.AsLayout2D()
		if ls.Updating.Value() == 0 {
			ls.Move2DTree()
			ls.Viewport.ReRender2DNode(li)
		} else {
			fmt.Printf("not ready to update\n")
		}
	})
}

// todo: we are leaking the scrollbars..
func (ly *Layout) DeleteScroll(d Dims2D) {
	if ly.Scrolls[d] == nil {
		return
	}
	sc := ly.Scrolls[d]
	win := ly.ParentWindow()
	if win != nil {
		sc.DisconnectAllEvents(win)
	}
	sc.Destroy()
	ly.Scrolls[d] = nil
}

func (ly *Layout) DeactivateScroll(sc *ScrollBar) {
	sc.LayData.AllocPos = Vec2DZero
	sc.LayData.AllocSize = Vec2DZero
	sc.VpBBox = image.ZR
	sc.WinBBox = image.ZR
}

func (ly *Layout) LayoutScrolls() {
	sbw := ly.Style.Layout.ScrollBarWidth.Dots

	spc := ly.Style.BoxSpace()
	avail := ly.AvailSize()
	for d := X; d < Dims2DN; d++ {
		odim := OtherDim(d)
		if ly.HasScroll[d] {
			sc := ly.Scrolls[d]
			sc.Size2D()
			sc.LayData.AllocPosRel.SetDim(d, spc)
			sc.LayData.AllocPosRel.SetDim(odim, avail.Dim(odim)-sbw-2.0)
			sc.LayData.AllocSize.SetDim(d, avail.Dim(d)-spc)
			if ly.HasScroll[odim] { // make room for other
				sc.LayData.AllocSize.SetSubDim(d, sbw)
			}
			sc.LayData.AllocSize.SetDim(odim, sbw)
			sc.Layout2D(ly.VpBBox) // this will add parent position to above rel pos
		} else {
			if ly.Scrolls[d] != nil {
				ly.DeactivateScroll(ly.Scrolls[d])
			}
		}
	}
}

func (ly *Layout) RenderScrolls() {
	for d := X; d < Dims2DN; d++ {
		if ly.HasScroll[d] {
			ly.Scrolls[d].Render2D()
		}
	}
}

// render the children
func (ly *Layout) Render2DChildren() {
	if ly.Lay == LayoutStacked {
		if ly.StackTop.Ptr == nil {
			return
		}
		gii, _ := KiToNode2D(ly.StackTop.Ptr)
		gii.Render2D()
		return
	}
	for _, kid := range ly.Kids {
		gii, _ := KiToNode2D(kid)
		if gii != nil {
			gii.Render2D()
		}
	}
}

// convenience for LayoutStacked to show child node at a given index
func (ly *Layout) ShowChildAtIndex(idx int) error {
	idx, err := ly.Kids.ValidIndex(idx)
	if err != nil {
		return err
	}
	ly.StackTop.Ptr = ly.Child(idx)
	return nil
}

///////////////////////////////////////////////////
//   Standard Node2D interface

func (ly *Layout) AsNode2D() *Node2DBase {
	return &ly.Node2DBase
}

func (ly *Layout) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Layout) AsLayout2D() *Layout {
	return g
}

func (ly *Layout) Init2D() {
	ly.Init2DBase()
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

func (ly *Layout) Style2D() {
	ly.Style2DWidget(nil)
}

func (ly *Layout) Size2D() {
	ly.InitLayout2D()
	if ly.Lay == LayoutGrid {
		ly.GatherSizesGrid()
	} else {
		ly.GatherSizes()
	}
}

func (ly *Layout) Layout2D(parBBox image.Rectangle) {
	ly.AllocFromParent()           // in case we didn't get anything
	ly.Layout2DBase(parBBox, true) // init style
	switch ly.Lay {
	case LayoutRow:
		ly.LayoutAll(X)
		ly.LayoutSingle(Y)
	case LayoutCol:
		ly.LayoutAll(Y)
		ly.LayoutSingle(X)
	case LayoutGrid:
		ly.LayoutGrid()
	case LayoutStacked:
		ly.LayoutSingle(X)
		ly.LayoutSingle(Y)
	case LayoutNil:
		// nothing
	}
	ly.FinalizeLayout()
	ly.ManageOverflow()
	ly.Layout2DChildren() // layout done with canonical positions

	delta := ly.Move2DDelta(image.ZP)
	if delta != image.ZP {
		ly.Move2DChildren(delta) // move is a separate step
	}
}

// we add our own offset here
func (ly *Layout) Move2DDelta(delta image.Point) image.Point {
	if ly.HasScroll[X] {
		off := ly.Scrolls[X].Value
		delta.X -= int(off)
	}
	if ly.HasScroll[Y] {
		off := ly.Scrolls[Y].Value
		delta.Y -= int(off)
	}
	return delta
}

func (ly *Layout) Move2D(delta image.Point, parBBox image.Rectangle) {
	ly.Move2DBase(delta, parBBox)
	delta = ly.Move2DDelta(delta) // add our offset
	ly.Move2DChildren(delta)
}

func (ly *Layout) Render2D() {
	if ly.PushBounds() {
		ly.RenderScrolls()
		ly.Render2DChildren()
		ly.PopBounds()
	}
}

func (ly *Layout) ReRender2D() (node Node2D, layout bool) {
	node = ly.This.(Node2D)
	layout = true
	return
}

func (ly *Layout) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Layout{}

///////////////////////////////////////////////////////////
//    Frame -- generic container that is also a Layout

// Frame is a basic container for widgets -- a layout that renders the
// standard box model
type Frame struct {
	Layout
}

var KiT_Frame = kit.Types.AddType(&Frame{}, FrameProps)

var FrameProps = map[string]interface{}{
	"border-width":     units.NewValue(2, units.Px),
	"border-radius":    units.NewValue(0, units.Px),
	"border-color":     color.Black,
	"border-style":     BorderSolid,
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"color":            color.Black,
	"background-color": color.White,
}

func (g *Frame) Style2D() {
	g.Style2DWidget(FrameProps)
}

func (g *Frame) Render2D() {
	if g.PushBounds() {
		pc := &g.Paint
		st := &g.Style
		rs := &g.Viewport.Render
		// first draw a background rectangle in our full area
		pc.StrokeStyle.SetColor(nil)
		pc.FillStyle.SetColor(&st.Background.Color)
		pos := g.LayData.AllocPos
		sz := g.LayData.AllocSize
		pc.DrawRectangle(rs, pos.X, pos.Y, sz.X, sz.Y)
		pc.FillStrokeClear(rs)

		rad := st.Border.Radius.Dots
		pos = pos.AddVal(st.Layout.Margin.Dots).SubVal(0.5 * st.Border.Width.Dots)
		sz = sz.SubVal(2.0 * st.Layout.Margin.Dots).AddVal(st.Border.Width.Dots)

		// then any shadow
		if st.BoxShadow.HasShadow() {
			spos := pos.Add(Vec2D{st.BoxShadow.HOffset.Dots, st.BoxShadow.VOffset.Dots})
			pc.StrokeStyle.SetColor(nil)
			pc.FillStyle.SetColor(&st.BoxShadow.Color)
			if rad == 0.0 {
				pc.DrawRectangle(rs, spos.X, spos.Y, sz.X, sz.Y)
			} else {
				pc.DrawRoundedRectangle(rs, spos.X, spos.Y, sz.X, sz.Y, rad)
			}
			pc.FillStrokeClear(rs)
		}

		pc.FillStyle.SetColor(&st.Background.Color)
		pc.StrokeStyle.SetColor(&st.Border.Color)
		pc.StrokeStyle.Width = st.Border.Width
		if rad == 0.0 {
			pc.DrawRectangle(rs, pos.X, pos.Y, sz.X, sz.Y)
		} else {
			pc.DrawRoundedRectangle(rs, pos.X, pos.Y, sz.X, sz.Y, rad)
		}
		pc.FillStrokeClear(rs)

		g.Layout.Render2D()
		g.PopBounds()
	}
}

// check for interface implementation
var _ Node2D = &Frame{}

///////////////////////////////////////////////////////////
//    Stretch and Space -- dummy elements for layouts

// Stretch adds an infinitely stretchy element for spacing out layouts
// (max-size = -1) set the width / height property to determine how much it
// takes relative to other stretchy elements
type Stretch struct {
	Node2DBase
}

var KiT_Stretch = kit.Types.AddType(&Stretch{}, StretchProps)

var StretchProps = map[string]interface{}{
	"max-width":  -1.0,
	"max-height": -1.0,
}

func (g *Stretch) Style2D() {
	g.Style2DWidget(StretchProps)
}

func (g *Stretch) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	g.Layout2DChildren()
}

// check for interface implementation
var _ Node2D = &Stretch{}

// Space adds a fixed sized (1 em by default) blank space to a layout -- set width / height property to change
type Space struct {
	Node2DBase
}

var KiT_Space = kit.Types.AddType(&Space{}, SpaceProps)

var SpaceProps = map[string]interface{}{
	"width":  units.NewValue(1, units.Em),
	"height": units.NewValue(1, units.Em),
}

func (g *Space) Style2D() {
	g.Style2DWidget(SpaceProps)
}

func (g *Space) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	g.Layout2DChildren()
}

// check for interface implementation
var _ Node2D = &Space{}
