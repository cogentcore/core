// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"strings"
	"time"
	"unicode"

	"github.com/chewxy/math32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/kit"
)

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
// LayoutHoriz, Vert both allow explicit Top/Left Center/Middle, Right/Bottom
// alignment along with Justify and SpaceAround -- they use IsAlign functions

// LayoutStyle contains style preferences on the layout of the element.
type LayoutStyle struct {
	ZIndex         int         `xml:"z-index" desc:"prop: z-index = ordering factor for rendering depth -- lower numbers rendered first -- sort children according to this factor"`
	AlignH         Align       `xml:"horizontal-align" desc:"prop: horizontal-align = horizontal alignment -- for widget layouts -- not a standard css property"`
	AlignV         Align       `xml:"vertical-align" desc:"prop: vertical-align = vertical alignment -- for widget layouts -- not a standard css property"`
	PosX           units.Value `xml:"x" desc:"prop: x = horizontal position -- often superceded by layout but otherwise used"`
	PosY           units.Value `xml:"y" desc:"prop: y = vertical position -- often superceded by layout but otherwise used"`
	Width          units.Value `xml:"width" desc:"prop: width = specified size of element -- 0 if not specified"`
	Height         units.Value `xml:"height" desc:"prop: height = specified size of element -- 0 if not specified"`
	MaxWidth       units.Value `xml:"max-width" desc:"prop: max-width = specified maximum size of element -- 0  means just use other values, negative means stretch"`
	MaxHeight      units.Value `xml:"max-height" desc:"prop: max-height = specified maximum size of element -- 0 means just use other values, negative means stretch"`
	MinWidth       units.Value `xml:"min-width" desc:"prop: min-width = specified mimimum size of element -- 0 if not specified"`
	MinHeight      units.Value `xml:"min-height" desc:"prop: min-height = specified mimimum size of element -- 0 if not specified"`
	Margin         units.Value `xml:"margin" desc:"prop: margin = outer-most transparent space around box element -- todo: can be specified per side"`
	Padding        units.Value `xml:"padding" desc:"prop: padding = transparent space around central content of box -- todo: if 4 values it is top, right, bottom, left; 3 is top, right&left, bottom; 2 is top & bottom, right and left"`
	Overflow       Overflow    `xml:"overflow" desc:"prop: overflow = what to do with content that overflows -- default is Auto add of scrollbars as needed -- todo: can have separate -x -y values"`
	Columns        int         `xml:"columns" alt:"grid-cols" desc:"prop: columns = number of columns to use in a grid layout -- used as a constraint in layout if individual elements do not specify their row, column positions"`
	Row            int         `xml:"row" desc:"prop: row = specifies the row that this element should appear within a grid layout"`
	Col            int         `xml:"col" desc:"prop: col = specifies the column that this element should appear within a grid layout"`
	RowSpan        int         `xml:"row-span" desc:"prop: row-span = specifies the number of sequential rows that this element should occupy within a grid layout (todo: not currently supported)"`
	ColSpan        int         `xml:"col-span" desc:"prop: col-span = specifies the number of sequential columns that this element should occupy within a grid layout"`
	ScrollBarWidth units.Value `xml:"scrollbar-width" desc:"prop: scrollbar-width = width of a layout scrollbar"`
}

func (ls *LayoutStyle) Defaults() {
	ls.AlignV = AlignMiddle
	ls.MinWidth.Set(2.0, units.Px)
	ls.MinHeight.Set(2.0, units.Px)
	ls.ScrollBarWidth.Set(16.0, units.Px)
}

func (ls *LayoutStyle) SetStylePost(props ki.Props) {
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

// Align has all different types of alignment -- only some are applicable to
// different contexts, but there is also so much overlap that it makes sense
// to have them all in one list -- some are not standard CSS and used by
// layout
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

func (ev Align) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Align) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

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

func (ev Overflow) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Overflow) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

//go:generate stringer -type=Overflow

////////////////////////////////////////////////////////////////////////////////////////
// Layout Data for actually computing the layout

// SizePrefs represents size preferences
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

// // 2D margins
// type Margins struct {
// 	left, right, top, bottom float32
// }

// // set a single margin for all items
// func (m *Margins) SetMargin(marg float32) {
// 	m.left = marg
// 	m.right = marg
// 	m.top = marg
// 	m.bottom = marg
// }

// LayoutData contains all the data needed to specify the layout of an item
// within a layout -- includes computed values of style prefs -- everything is
// concrete and specified here, whereas style may not be fully resolved
type LayoutData struct {
	Size          SizePrefs `desc:"size constraints for this item -- from layout style"`
	AllocSize     Vec2D     `desc:"allocated size of this item, by the parent layout"`
	AllocPos      Vec2D     `desc:"position of this item, computed by adding in the AllocPosRel to parent position"`
	AllocPosRel   Vec2D     `desc:"allocated relative position of this item, computed by the parent layout"`
	AllocSizeOrig Vec2D     `desc:"original copy of allocated size of this item, by the parent layout -- some widgets will resize themselves within a given layout (e.g., a TextView), but still need access to their original allocated size"`
	AllocPosOrig  Vec2D     `desc:"original copy of allocated relative position of this item, by the parent layout -- need for scrolling which can update AllocPos"`
}

// todo: not using yet:
// Margins Margins   `desc:"margins around this item"`
// GridPos      image.Point `desc:"position within a grid"`
// GridSpan     image.Point `desc:"number of grid elements that we take up in each direction"`

func (ld *LayoutData) Defaults() {
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

// SizePrefOrMax returns the pref size if non-zero, else the max-size -- use
// for style-based constraints during initial sizing (e.g., word wrapping)
func (ld *LayoutData) SizePrefOrMax() Vec2D {
	return ld.Size.Pref.MinPos(ld.Size.Max)
}

// Reset is called at start of layout process -- resets all values back to 0
func (ld *LayoutData) Reset() {
	ld.AllocSize = Vec2DZero
	ld.AllocPos = Vec2DZero
	ld.AllocPosRel = Vec2DZero
}

// UpdateSizes updates our sizes based on AllocSize and Max constraints, etc
func (ld *LayoutData) UpdateSizes() {
	ld.Size.Need.SetMax(ld.AllocSize)   // min cannot be < alloc -- bare min
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
// is used within the relevant dimension (e.g., align-horiz for a LayoutHoriz
// layout, to determine left, right, center, justified).  Layouts
// can automatically add scrollbars depending on the Overflow layout style.
type Layout struct {
	WidgetBase
	Lay           Layouts             `xml:"lay" desc:"type of layout to use"`
	Spacing       units.Value         `xml:"spacing" desc:"extra space to add between elements in the layout"`
	StackTop      int                 `desc:"for Stacked layout, index of node to use as the top of the stack -- only node at this index is rendered -- if not a valid index, nothing is rendered"`
	ChildSize     Vec2D               `json:"-" xml:"-" desc:"total max size of children as laid out"`
	ExtraSize     Vec2D               `json:"-" xml:"-" desc:"extra size in each dim due to scrollbars we add"`
	HasScroll     [Dims2DN]bool       `json:"-" xml:"-" desc:"whether scrollbar is used for given dim"`
	Scrolls       [Dims2DN]*ScrollBar `json:"-" xml:"-" desc:"scroll bars -- we fully manage them as needed"`
	GridSize      image.Point         `json:"-" xml:"-" desc:"computed size of a grid layout based on all the constraints -- computed during Size2D pass"`
	GridData      [RowColN][]GridData `json:"-" xml:"-" desc:"grid data for rows in [0] and cols in [1]"`
	NeedsRedo     bool                `json:"-" xml:"-" desc:"true if this layout got a redo = true on previous iteration -- otherwise it just skips any re-layout on subsequent iteration"`
	FocusName     string              `json:"-" xml:"-" desc:"accumulated name to search for when keys are typed"`
	FocusNameTime time.Time           `json:"-" xml:"-" desc:"time of last focus name event -- for timeout"`
	FocusNameLast ki.Ki               `json:"-" xml:"-" desc:"last element focused on -- used as a starting point if name is the same"`
	ScrollsOff    bool                `json:"-" xml:"-" desc:"scrollbars have been manually turned off due to layout being invisible -- must be reactivated when re-visible"`
}

var KiT_Layout = kit.Types.AddType(&Layout{}, nil)

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
	// vertically as needed
	LayoutHorizFlow

	// LayoutVertFlow arranges items vertically within a column, overflowing
	// horizontally as needed
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

var KiT_Layouts = kit.Enums.AddEnumAltLower(LayoutsN, false, StylePropProps, "Layout")

func (ev Layouts) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Layouts) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// row / col for grid data
type RowCol int32

const (
	Row RowCol = iota
	Col
	RowColN
)

var KiT_RowCol = kit.Enums.AddEnumAltLower(RowColN, false, StylePropProps, "")

func (ev RowCol) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *RowCol) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

//go:generate stringer -type=RowCol

// LayoutDefault is default obj that can be used when property specifies "default"
var LayoutDefault Layout

// LayoutFields contain the StyledFields for Layout type
var LayoutFields = initLayout()

func initLayout() *StyledFields {
	LayoutDefault = Layout{}
	sf := &StyledFields{}
	sf.Default = &LayoutDefault
	sf.AddField(&LayoutDefault, "Lay")
	sf.AddField(&LayoutDefault, "Spacing")
	return sf
}

// SumDim returns whether we sum up elements along given dimension?  else use
// max for shared dimension.
func (ly *Layout) SumDim(d Dims2D) bool {
	if (d == X && ly.Lay == LayoutHoriz) || (d == Y && ly.Lay == LayoutVert) {
		return true
	}
	return false
}

// SummedDim returns the dimension along which layout is summing.
func (ly *Layout) SummedDim() Dims2D {
	if ly.Lay == LayoutHoriz {
		return X
	}
	return Y
}

////////////////////////////////////////////////////////////////////////////////////////
//     Gather Sizes

// first depth-first Size2D pass: terminal concrete items compute their AllocSize
// we focus on Need: Max(Min, AllocSize), and Want: Max(Pref, AllocSize) -- Max is
// only used if we need to fill space, during final allocation
//
// second me-first Layout2D pass: each layout allocates AllocSize for its
// children based on aggregated size data, and so on down the tree

// GatherSizes is size first pass: gather the size information from the children
func (ly *Layout) GatherSizes() {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	var sumPref, sumNeed, maxPref, maxNeed Vec2D
	for _, c := range ly.Kids {
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		ni.LayData.UpdateSizes()
		sumNeed = sumNeed.Add(ni.LayData.Size.Need)
		sumPref = sumPref.Add(ni.LayData.Size.Pref)
		maxNeed = maxNeed.Max(ni.LayData.Size.Need)
		maxPref = maxPref.Max(ni.LayData.Size.Pref)

		if Layout2DTrace {
			fmt.Printf("Size:   %v Child: %v, need: %v, pref: %v\n", ly.PathUnique(), ni.UniqueNm, ni.LayData.Size.Need.Dim(ly.SummedDim()), ni.LayData.Size.Pref.Dim(ly.SummedDim()))
		}
	}

	for d := X; d <= Y; d++ {
		if ly.LayData.Size.Pref.Dim(d) == 0 {
			if ly.SumDim(d) { // our layout now updated to sum
				ly.LayData.Size.Need.SetMaxDim(d, sumNeed.Dim(d))
				ly.LayData.Size.Pref.SetMaxDim(d, sumPref.Dim(d))
			} else { // use max for other dir
				ly.LayData.Size.Need.SetMaxDim(d, maxNeed.Dim(d))
				ly.LayData.Size.Pref.SetMaxDim(d, maxPref.Dim(d))
			}
		} else { // use target size from style
			ly.LayData.Size.Need.SetDim(d, ly.LayData.Size.Pref.Dim(d))
		}
	}

	spc := ly.Sty.BoxSpace()
	ly.LayData.Size.Need.SetAddVal(2.0 * spc)
	ly.LayData.Size.Pref.SetAddVal(2.0 * spc)

	elspc := float32(0.0)
	if sz >= 2 {
		elspc = float32(sz-1) * ly.Spacing.Dots
	}
	if ly.SumDim(X) {
		ly.LayData.Size.Need.X += elspc
		ly.LayData.Size.Pref.X += elspc
	}
	if ly.SumDim(Y) {
		ly.LayData.Size.Need.Y += elspc
		ly.LayData.Size.Pref.Y += elspc
	}

	ly.LayData.UpdateSizes() // enforce max and normal ordering, etc
	if Layout2DTrace {
		fmt.Printf("Size:   %v gather sizes need: %v, pref: %v, elspc: %v\n", ly.PathUnique(), ly.LayData.Size.Need, ly.LayData.Size.Pref, elspc)
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
	// collect overal size
	for _, c := range ly.Kids {
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
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		ni.LayData.UpdateSizes()
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
		SetMax32(&(rgd.SizeNeed), ni.LayData.Size.Need.Y)
		SetMax32(&(rgd.SizePref), ni.LayData.Size.Pref.Y)
		SetMax32(&(cgd.SizeNeed), ni.LayData.Size.Need.X)
		SetMax32(&(cgd.SizePref), ni.LayData.Size.Pref.X)

		// for max: any -1 stretch dominates, else accumulate any max
		if rgd.SizeMax >= 0 {
			if ni.LayData.Size.Max.Y < 0 { // stretch
				rgd.SizeMax = -1
			} else {
				SetMax32(&(rgd.SizeMax), ni.LayData.Size.Max.Y)
			}
		}
		if cgd.SizeMax >= 0 {
			if ni.LayData.Size.Max.Y < 0 { // stretch
				cgd.SizeMax = -1
			} else {
				SetMax32(&(cgd.SizeMax), ni.LayData.Size.Max.X)
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
	for _, gd := range ly.GridData[Row] {
		sumNeed.SetAddDim(Y, gd.SizeNeed)
		sumPref.SetAddDim(Y, gd.SizePref)
	}
	// X = sum across cols which have max's
	for _, gd := range ly.GridData[Col] {
		sumNeed.SetAddDim(X, gd.SizeNeed)
		sumPref.SetAddDim(X, gd.SizePref)
	}

	if ly.LayData.Size.Pref.X == 0 {
		ly.LayData.Size.Need.X = Max32(ly.LayData.Size.Need.X, sumNeed.X)
		ly.LayData.Size.Pref.X = Max32(ly.LayData.Size.Pref.X, sumPref.X)
	} else { // use target size from style otherwise
		ly.LayData.Size.Need.X = ly.LayData.Size.Pref.X
	}
	if ly.LayData.Size.Pref.Y == 0 {
		ly.LayData.Size.Need.Y = Max32(ly.LayData.Size.Need.Y, sumNeed.Y)
		ly.LayData.Size.Pref.Y = Max32(ly.LayData.Size.Pref.Y, sumPref.Y)
	} else { // use target size from style otherwise
		ly.LayData.Size.Need.Y = ly.LayData.Size.Pref.Y
	}

	spc := ly.Sty.BoxSpace()
	ly.LayData.Size.Need.SetAddVal(2.0 * spc)
	ly.LayData.Size.Pref.SetAddVal(2.0 * spc)

	ly.LayData.Size.Need.X += float32(cols-1) * ly.Spacing.Dots
	ly.LayData.Size.Pref.X += float32(cols-1) * ly.Spacing.Dots
	ly.LayData.Size.Need.Y += float32(rows-1) * ly.Spacing.Dots
	ly.LayData.Size.Pref.Y += float32(rows-1) * ly.Spacing.Dots

	ly.LayData.UpdateSizes() // enforce max and normal ordering, etc
	if Layout2DTrace {
		fmt.Printf("Size:   %v gather sizes grid need: %v, pref: %v\n", ly.PathUnique(), ly.LayData.Size.Need, ly.LayData.Size.Pref)
	}
}

// AllocFromParent: if we are not a child of a layout, then get allocation
// from a parent obj that has a layout size
func (ly *Layout) AllocFromParent() {
	if ly.Par == nil || ly.Viewport == nil || !ly.LayData.AllocSize.IsZero() {
		return
	}
	if ly.Par != ly.Viewport.This() {
		// note: zero alloc size happens all the time with non-visible tabs!
		// fmt.Printf("Layout: %v has zero allocation but is not a direct child of viewport -- this is an error -- every level must provide layout for the next! laydata:\n%+v\n", ly.PathUnique(), ly.LayData)
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
	extra = Max32(extra, 0.0) // no negatives

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
func (ly *Layout) LayoutSharedDim(dim Dims2D) {
	spc := ly.Sty.BoxSpace()
	avail := ly.LayData.AllocSize.Dim(dim) - 2.0*spc
	for _, c := range ly.Kids {
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		al := ni.Sty.Layout.AlignDim(dim)
		pref := ni.LayData.Size.Pref.Dim(dim)
		need := ni.LayData.Size.Need.Dim(dim)
		max := ni.LayData.Size.Max.Dim(dim)
		pos, size := ly.LayoutSharedDimImpl(avail, need, pref, max, spc, al)
		ni.LayData.AllocSize.SetDim(dim, size)
		ni.LayData.AllocPosRel.SetDim(dim, pos)
	}
}

// LayoutAlongDim lays out all children along given dim -- only affects that dim --
// e.g., use LayoutSharedDim for other dim.
func (ly *Layout) LayoutAlongDim(dim Dims2D) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	elspc := float32(sz-1) * ly.Spacing.Dots
	al := ly.Sty.Layout.AlignDim(dim)
	spc := ly.Sty.BoxSpace()
	exspc := 2.0*spc + elspc
	avail := ly.LayData.AllocSize.Dim(dim) - exspc
	pref := ly.LayData.Size.Pref.Dim(dim) - exspc
	need := ly.LayData.Size.Need.Dim(dim) - exspc

	targ := pref
	usePref := true
	extra := avail - targ
	if extra < -0.1 { // not fitting in pref, go with need
		usePref = false
		targ = need
		extra = avail - targ
	}
	extra = Max32(extra, 0.0) // no negatives

	nstretch := 0
	stretchTot := float32(0.0)
	stretchNeed := false        // stretch relative to need
	stretchMax := false         // only stretch Max = neg
	addSpace := false           // apply extra toward spacing -- for justify
	if usePref && extra > 0.0 { // have some stretch extra
		for _, c := range ly.Kids {
			ni := c.(Node2D).AsWidget()
			if ni == nil {
				continue
			}
			if ni.LayData.Size.HasMaxStretch(dim) { // negative = stretch
				nstretch++
				stretchTot += ni.LayData.Size.Pref.Dim(dim)
			}
		}
		if nstretch > 0 {
			stretchMax = true // only stretch those marked as infinitely stretchy
		}
	} else if extra > 0.0 { // extra relative to Need
		for _, c := range ly.Kids {
			ni := c.(Node2D).AsWidget()
			if ni == nil {
				continue
			}
			if ni.LayData.Size.HasMaxStretch(dim) || ni.LayData.Size.CanStretchNeed(dim) {
				nstretch++
				stretchTot += ni.LayData.Size.Pref.Dim(dim)
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
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		size := ni.LayData.Size.Need.Dim(dim)
		if usePref {
			size = ni.LayData.Size.Pref.Dim(dim)
		}
		if stretchMax { // negative = stretch
			if ni.LayData.Size.HasMaxStretch(dim) { // in proportion to pref
				size += extra * (ni.LayData.Size.Pref.Dim(dim) / stretchTot)
			}
		} else if stretchNeed {
			if ni.LayData.Size.HasMaxStretch(dim) || ni.LayData.Size.CanStretchNeed(dim) {
				size += extra * (ni.LayData.Size.Pref.Dim(dim) / stretchTot)
			}
		} else if addSpace { // implies align justify
			if i > 0 {
				pos += extraSpace
			}
		}

		ni.LayData.AllocSize.SetDim(dim, size)
		ni.LayData.AllocPosRel.SetDim(dim, pos)
		if Layout2DTrace {
			fmt.Printf("Layout: %v Child: %v, pos: %v, size: %v, need: %v, pref: %v\n", ly.PathUnique(), ni.UniqueNm, pos, size, ni.LayData.Size.Need.Dim(dim), ni.LayData.Size.Pref.Dim(dim))
		}
		pos += size + ly.Spacing.Dots
	}
}

// LayoutGridDim lays out grid data along each dimension (row, Y; col, X),
// same as LayoutAlongDim.  For cols, X has width prefs of each -- turn that
// into an actual allocated width for each column, and likewise for rows.
func (ly *Layout) LayoutGridDim(rowcol RowCol, dim Dims2D) {
	gds := ly.GridData[rowcol]
	sz := len(gds)
	if sz == 0 {
		return
	}
	elspc := float32(sz-1) * ly.Spacing.Dots
	al := ly.Sty.Layout.AlignDim(dim)
	spc := ly.Sty.BoxSpace()
	exspc := 2.0*spc + elspc
	avail := ly.LayData.AllocSize.Dim(dim) - exspc
	pref := ly.LayData.Size.Pref.Dim(dim) - exspc
	need := ly.LayData.Size.Need.Dim(dim) - exspc

	targ := pref
	usePref := true
	extra := avail - targ
	if extra < -0.1 { // not fitting in pref, go with need
		usePref = false
		targ = need
		extra = avail - targ
	}
	extra = Max32(extra, 0.0) // no negatives

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

	ly.LayoutGridDim(Row, Y)
	ly.LayoutGridDim(Col, X)

	col := 0
	row := 0
	cols := ly.GridSize.X
	rows := ly.GridSize.Y
	for _, c := range ly.Kids {
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
			dim := X
			gd := ly.GridData[Col][col]
			avail := gd.AllocSize
			al := lst.AlignDim(dim)
			pref := ni.LayData.Size.Pref.Dim(dim)
			need := ni.LayData.Size.Need.Dim(dim)
			max := ni.LayData.Size.Max.Dim(dim)
			pos, size := ly.LayoutSharedDimImpl(avail, need, pref, max, 0, al)
			ni.LayData.AllocSize.SetDim(dim, size)
			ni.LayData.AllocPosRel.SetDim(dim, pos+gd.AllocPosRel)

		}
		{ // row, Y dim
			dim := Y
			gd := ly.GridData[Row][row]
			avail := gd.AllocSize
			al := lst.AlignDim(dim)
			pref := ni.LayData.Size.Pref.Dim(dim)
			need := ni.LayData.Size.Need.Dim(dim)
			max := ni.LayData.Size.Max.Dim(dim)
			pos, size := ly.LayoutSharedDimImpl(avail, need, pref, max, 0, al)
			ni.LayData.AllocSize.SetDim(dim, size)
			ni.LayData.AllocPosRel.SetDim(dim, pos+gd.AllocPosRel)
		}

		if Layout2DTrace {
			fmt.Printf("Layout: %v grid col: %v row: %v pos: %v size: %v\n", ly.PathUnique(), col, row, ni.LayData.AllocPosRel, ni.LayData.AllocSize)
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
	ly.ChildSize = Vec2DZero
	for _, c := range ly.Kids {
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		ly.ChildSize.SetMax(ni.LayData.AllocPosRel.Add(ni.LayData.AllocSize))
		ni.LayData.AllocSizeOrig = ni.LayData.AllocSize
	}
}

// AvailSize returns the total size avail to this layout -- typically
// AllocSize except for top-level layout which uses VpBBox in case less is
// avail
func (ly *Layout) AvailSize() Vec2D {
	spc := ly.Sty.BoxSpace()
	avail := ly.LayData.AllocSize.SubVal(spc) // spc is for right size space
	parni, _ := KiToNode2D(ly.Par)
	if parni != nil {
		vp := parni.AsViewport2D()
		if vp != nil {
			if vp.Viewport == nil {
				avail = NewVec2DFmPoint(ly.VpBBox.Size()).SubVal(spc)
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

	ly.ExtraSize.SetVal(0.0)
	for d := X; d < Dims2DN; d++ {
		ly.HasScroll[d] = false
	}

	if ly.Sty.Layout.Overflow != OverflowHidden {
		sbw := ly.Sty.Layout.ScrollBarWidth.Dots
		for d := X; d < Dims2DN; d++ {
			odim := OtherDim(d)
			if ly.ChildSize.Dim(d) > (avail.Dim(d) + 2.0) { // overflowing -- allow some margin
				// if wasscof {
				// 	fmt.Printf("overflow, setting scb: %v\n", d)
				// }
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

// HasAnyScroll returns true if layout has
func (ly *Layout) HasAnyScroll() bool {
	return ly.HasScroll[X] || ly.HasScroll[Y]
}

// SetScroll sets a scrollbar along given dimension
func (ly *Layout) SetScroll(d Dims2D) {
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
	avail := ly.AvailSize().SubVal(spc * 2.0)
	sc := ly.Scrolls[d]
	if d == X {
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
	sc.Value = Min32(sc.Value, sc.Max-sc.ThumbVal) // keep in range
	// fmt.Printf("set sc lay: %v  max: %v  val: %v\n", ly.PathUnique(), sc.Max, sc.Value)
	sc.SliderSig.ConnectOnly(ly.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig != int64(SliderValueChanged) {
			return
		}
		li, _ := KiToNode2D(recv)
		ls := li.AsLayout2D()
		if !ls.IsUpdating() {
			wupdt := ls.Viewport.Win.UpdateStart()
			ls.Move2DTree()
			ls.Viewport.ReRender2DNode(li)
			ls.Viewport.Win.UpdateEnd(wupdt)
			// } else {
			// 	fmt.Printf("not ready to update\n")
		}
	})
}

// DeleteScroll deletes scrollbar along given dimesion.  todo: we are leaking
// the scrollbars -- move into a container Field
func (ly *Layout) DeleteScroll(d Dims2D) {
	if ly.Scrolls[d] == nil {
		return
	}
	sc := ly.Scrolls[d]
	sc.DisconnectAllEvents(AllPris)
	sc.Destroy()
	ly.Scrolls[d] = nil
}

// DeactivateScroll turns off given scrollbar, without deleting, so it can be easily re-used
func (ly *Layout) DeactivateScroll(sc *ScrollBar) {
	sc.LayData.AllocPos = Vec2DZero
	sc.LayData.AllocSize = Vec2DZero
	sc.VpBBox = image.ZR
	sc.WinBBox = image.ZR
}

// LayoutScrolls arranges scrollbars
func (ly *Layout) LayoutScrolls() {
	sbw := ly.Sty.Layout.ScrollBarWidth.Dots

	spc := ly.Sty.BoxSpace()
	avail := ly.AvailSize()
	for d := X; d < Dims2DN; d++ {
		odim := OtherDim(d)
		if ly.HasScroll[d] {
			sc := ly.Scrolls[d]
			sc.Size2D(0)
			sc.LayData.AllocPosRel.SetDim(d, spc)
			sc.LayData.AllocPosRel.SetDim(odim, avail.Dim(odim)-sbw-2.0)
			sc.LayData.AllocSize.SetDim(d, avail.Dim(d)-spc)
			if ly.HasScroll[odim] { // make room for other
				sc.LayData.AllocSize.SetSubDim(d, sbw)
			}
			sc.LayData.AllocSize.SetDim(odim, sbw)
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
	for d := X; d < Dims2DN; d++ {
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
	for d := X; d < Dims2DN; d++ {
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
	for d := X; d < Dims2DN; d++ {
		if ly.HasScroll[d] {
			ly.Scrolls[d].Move2D(delta, parBBox)
		}
	}
}

// ScrollDelta processes a scroll event.  If only one dimension is processed,
// and there is a non-zero in other, then the consumed dimension is reset to 0
// and the event is left unprocessed, so a higher level can consume the
// remainder.
func (ly *Layout) ScrollDelta(me *mouse.ScrollEvent) {
	del := me.Delta
	if ly.HasScroll[Y] && ly.HasScroll[X] {
		// fmt.Printf("ly: %v both del: %v\n", ly.Nm, del)
		ly.Scrolls[Y].SetValueAction(ly.Scrolls[Y].Value + float32(del.Y))
		ly.Scrolls[X].SetValueAction(ly.Scrolls[X].Value + float32(del.X))
		me.SetProcessed()
	} else if ly.HasScroll[Y] {
		// fmt.Printf("ly: %v y del: %v\n", ly.Nm, del)
		ly.Scrolls[Y].SetValueAction(ly.Scrolls[Y].Value + float32(del.Y))
		if del.X != 0 {
			me.Delta.Y = 0
		} else {
			me.SetProcessed()
		}
	} else if ly.HasScroll[X] {
		// fmt.Printf("ly: %v x del: %v\n", ly.Nm, del)
		if del.X != 0 {
			ly.Scrolls[X].SetValueAction(ly.Scrolls[X].Value + float32(del.X))
			if del.Y != 0 {
				me.Delta.X = 0
			} else {
				me.SetProcessed()
			}
		} else { // use Y instead as mouse wheels typically on have this
			ly.Scrolls[X].SetValueAction(ly.Scrolls[X].Value + float32(del.Y))
			me.SetProcessed()
		}
	}
}

// render the children
func (ly *Layout) Render2DChildren() {
	if ly.Lay == LayoutStacked {
		for i, kid := range ly.Kids {
			if _, ni := KiToNode2D(kid); ni != nil {
				if i == ly.StackTop {
					ni.ClearInvisibleTree()
				} else {
					ni.SetInvisibleTree()
				}
			}
		}
		// note: all nodes need to render to disconnect b/c of invisible
	}
	for _, kid := range ly.Kids {
		nii, _ := KiToNode2D(kid)
		if nii != nil {
			nii.Render2D()
		}
	}
}

func (ly *Layout) Move2DChildren(delta image.Point) {
	cbb := ly.This().(Node2D).ChildrenBBox2D()
	if ly.Lay == LayoutStacked {
		sn, ok := ly.Child(ly.StackTop)
		if !ok {
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
func (ly *Layout) AutoScrollDim(dim Dims2D, st, pos int) bool {
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
			dst = Min32(dst, sc.Value)
			sc.SetValueAction(sc.Value - dst)
			return true
		}
	} else {
		pct := float32(maxd) / float32(vissz)
		if pct < .1 && sc.Value < scrange {
			dst = Min32(dst, (scrange - sc.Value))
			sc.SetValueAction(sc.Value + dst)
			return true
		}
	}
	return false
}

// AutoScroll scrolls the layout based on mouse position, when appropriate (DND, menus)
func (ly *Layout) AutoScroll(pos image.Point) bool {
	did := false
	if ly.HasScroll[Y] && ly.HasScroll[X] {
		did = ly.AutoScrollDim(Y, ly.WinBBox.Min.Y, pos.Y)
		did = did || ly.AutoScrollDim(X, ly.WinBBox.Min.X, pos.X)
	} else if ly.HasScroll[Y] {
		did = ly.AutoScrollDim(Y, ly.WinBBox.Min.Y, pos.Y)
	} else if ly.HasScroll[X] {
		did = ly.AutoScrollDim(X, ly.WinBBox.Min.X, pos.X)
	}
	return did
}

// ScrollToBoxDim scrolls to ensure that given rect box along one dimension is
// in view -- returns true if scrolling was needed
func (ly *Layout) ScrollToBoxDim(dim Dims2D, minBox, maxBox int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
	vpMin := ly.VpBBox.Min.X
	if dim == Y {
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
	if ly.HasScroll[Y] && ly.HasScroll[X] {
		did = ly.ScrollToBoxDim(Y, box.Min.Y, box.Max.Y)
		did = did || ly.ScrollToBoxDim(X, box.Min.X, box.Max.X)
	} else if ly.HasScroll[Y] {
		did = ly.ScrollToBoxDim(Y, box.Min.Y, box.Max.Y)
	} else if ly.HasScroll[X] {
		did = ly.ScrollToBoxDim(X, box.Min.X, box.Max.X)
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
func (ly *Layout) ScrollDimToStart(dim Dims2D, pos int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
	vpMin := ly.VpBBox.Min.X
	if dim == Y {
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
func (ly *Layout) ScrollDimToEnd(dim Dims2D, pos int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
	vpMin := ly.VpBBox.Min.X
	if dim == Y {
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
func (ly *Layout) ScrollDimToCenter(dim Dims2D, pos int) bool {
	if !ly.HasScroll[dim] {
		return false
	}
	vpMin := ly.VpBBox.Min.X
	if dim == Y {
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
	win := ly.ParentWindow()
	if win == nil {
		return nil, -1
	}
	for i, k := range ly.Kids {
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
	win := ly.ParentWindow()
	cur := win.Focus
	nxti := idx + 1
	if ly.Lay == LayoutGrid && updn {
		nxti = idx + ly.Sty.Layout.Columns
	}
	did := false
	if nxti < sz {
		did = win.FocusOnOrNext(ly.KnownChild(nxti))
	} else {
		did = win.FocusOnOrNext(ly.KnownChild(0))
	}
	if !did || win.Focus == cur {
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
	win := ly.ParentWindow()
	cur := win.Focus
	nxti := idx - 1
	if ly.Lay == LayoutGrid && updn {
		nxti = idx - ly.Sty.Layout.Columns
	}
	did := false
	if nxti >= 0 {
		did = win.FocusOnOrNext(ly.KnownChild(nxti))
	} else {
		did = win.FocusOnOrNext(ly.KnownChild(sz - 1))
	}
	if !did || win.Focus == cur {
		return false
	}
	return true
}

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
		}
	}
	if nf, ok := ly.Prop("no-focus-name"); ok {
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
		ly.FocusName += string(kt.Rune)
		ly.FocusNameLast = nil // only use last if tabbing
	}
	kt.SetProcessed()
	// fmt.Printf("searching for: %v\n", ly.FocusName)
	focel, found := ly.ChildByLabelStartsCanFocus(ly.FocusName, ly.FocusNameLast)
	if found {
		ly.ParentWindow().SetFocus(focel) // this will also scroll by default!
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
	ly.Style2DWidget()
	tprops := *kit.Types.Properties(ly.Type(), true) // true = makeNew
	kit.TypesMu.RLock()
	LayoutFields.Style(ly, nil, tprops)
	kit.TypesMu.RUnlock()
	LayoutFields.Style(ly, nil, ly.Props)
	LayoutFields.ToDots(ly, &ly.Sty.UnContext)
}

func (ly *Layout) Style2D() {
	ly.StyleLayout()
	ly.LayData.SetFromStyle(&ly.Sty.Layout) // also does reset
}

func (ly *Layout) Size2D(iter int) {
	ly.InitLayout2D()
	if ly.Lay == LayoutGrid {
		ly.GatherSizesGrid()
	} else {
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
	switch ly.Lay {
	case LayoutHoriz:
		ly.LayoutAlongDim(X)
		ly.LayoutSharedDim(Y)
	case LayoutVert:
		ly.LayoutAlongDim(Y)
		ly.LayoutSharedDim(X)
	case LayoutGrid:
		ly.LayoutGrid()
	case LayoutStacked:
		ly.LayoutSharedDim(X)
		ly.LayoutSharedDim(Y)
	case LayoutNil:
		// nothing
	}
	ly.FinalizeLayout()
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

var StretchProps = ki.Props{
	"max-width":  -1.0,
	"max-height": -1.0,
}

func (st *Stretch) Style2D() {
	st.Style2DWidget()
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

var SpaceProps = ki.Props{
	"width":  units.NewValue(1, units.Ch),
	"height": units.NewValue(1, units.Em),
}

func (sp *Space) Style2D() {
	sp.Style2DWidget()
	sp.LayData.SetFromStyle(&sp.Sty.Layout) // also does reset
}

func (sp *Space) Layout2D(parBBox image.Rectangle, iter int) bool {
	sp.Layout2DBase(parBBox, true, iter) // init style
	return sp.Layout2DChildren(iter)
}
