// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"strings"

	"goki.dev/girl/styles"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// Layout uses 2 Size passes, 2 Position passes:
//
// 1. SizeUp (bottom-up): gathers sizes from our Children & Parts,
//    based only on Min style sizes and actual content sizing.
//    Flexible elements (e.g., Text, Flex Wrap, TopAppBar) allocate
//    optimistically along their main axis, up to any optional Max size
//
// 2. SizeDown (top-down, multiple iterations possible): assigns sizes based
//    on allocated parent avail size, giving extra space based on Grow factors,
//    and flexible elements wrap / config to fit top-down constraint along main
//    axis, producing a (new) top-down size expanding in cross axis as needed
//    (or removing items that don't fit, etc).  Wrap & Grid layouts assign
//    X,Y index coordinates to items during this pass.
//
//    This iterates until the resulting sizes values are stable.
//
// 3. Position: uses the final sizes to position everything within the layout
//    according to alignment settings.
//
// 4. ScenePos: scene-based position and final BBox is computed based on scrollbar
//    position.  This step can be performed when scrolling.

// (Text) Wrapping key principles:
// * start with nothing (except styled) for SizeUp (non-wrap asserts needs)
// * use full Alloc for SizeDown
// * set content + total to what you actually use (key: start with only styled
//   so you don't get hysterisis)
// * Layout always re-gets the actuals for accurate KidsSize

//////////////////////////////////////////////////////////////
//  LaySize

// LaySize has sizing information for content and total size, and grow factors
type LaySize struct {
	// Content is size of the contents (children, parts) of the widget,
	// excluding all additional Space (but including Gaps for Layouts)
	Content mat32.Vec2

	// Space is total extra space that, when added to Content, results in the Total size
	// This includes padding, total effective margin (border, shadow, etc), and scrollbars
	// It does NOT include Gap for layout, which is part of Content.
	Space mat32.Vec2

	// Total is total size of the widget: Content + Space
	Total mat32.Vec2

	// Alloc is the top-down Total allocated size of the widget, computed by the Layout.
	// If we are not stretchy (Grow factor == 0), then our Total size can be less than
	// this allocation, in which case our alignment factors determine positioning within
	// the allocated space.
	Alloc mat32.Vec2
}

func (ls *LaySize) String() string {
	return fmt.Sprintf("Content: %v, \tSpace: %v, \tTotal: %v, \tAlloc: %v", ls.Content, ls.Space, ls.Total, ls.Alloc)
}

// SetContentMax sets the Content size subject to given Max values
// which only apply if they are > 0.  Values are automatically
// rounded up to Ceil ints
func (ls *LaySize) SetContentMax(sz mat32.Vec2, mx mat32.Vec2) {
	ls.Content = sz
	ls.Content.SetMinPos(mx)
	ls.Content.SetCeil()
}

// SetContentToFit sets the Content size subject to given Max values
// which only apply if they are > 0.  Values are automatically
// rounded up to Ceil ints
func (ls *LaySize) SetContentToFit(sz mat32.Vec2, mx mat32.Vec2) {
	ls.Content.SetMaxPos(sz)
	ls.Content.SetMinPos(mx)
	ls.Content.SetCeil()
}

// SetSpace sets the space and rounds up to Ceil ints
func (ls *LaySize) SetSpace(spc mat32.Vec2) {
	ls.Space = spc
	ls.Space.SetCeil()
}

// SetTotalFromContent sets the Total size as Content plus given extra space
func (ls *LaySize) SetTotalFromContent() {
	ls.Total = ls.Content.Add(ls.Space)
}

// SetTotalToFitContent increases the Total size to fit current content
func (ls *LaySize) SetTotalToFitContent() {
	ls.Total.SetMaxPos(ls.Content.Add(ls.Space))
}

// SetContentFromTotal sets the Content from Total size subtracting extra space
func (ls *LaySize) SetContentFromTotal() {
	ls.Content = ls.Total.Sub(ls.Space)
}

//////////////////////////////////////////////////////////////
//  LayState

// LayState contains the the layout state for each widget,
// Set by the parent Layout during the Layout process.
type LayState struct {
	// Size is the size of the element (updated during different passes,
	// and holding the final computed size).
	Size LaySize `view:"inline"`

	// Cell is the logical X, Y index coordinates (row, col) of element
	// within its parent layout
	Cell image.Point

	// RelPos is top, left position relative to parent Content size space
	RelPos mat32.Vec2

	// Scroll is additional scrolling offset within our parent layout
	Scroll mat32.Vec2

	// Pos is position within the overall Scene that we render into,
	// including effects of scroll offset
	Pos mat32.Vec2

	// ContentPos is Pos plus spacing offset for top, left of where
	// content starts rendering.
	ContentPos mat32.Vec2

	// 2D bounding box for Total size occupied within parent Scene object that we render onto,
	// starting at Pos and ending at Pos + Size.Total.
	// These are the pixels we can draw into, intersected with parent bounding boxes
	// (empty for invisible). Used for render Bounds clipping.
	// This includes all space (margin, padding etc).
	BBox image.Rectangle `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// 2D bounding box for our content, which excludes our padding, margin, etc.
	// starting at ContentPos and ending at Pos + Size.Content.
	// It is intersected with parent bounding boxes.
	ContentBBox image.Rectangle `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`
}

func (ls *LayState) String() string {
	return "Size:" + ls.Size.String() + "\tCell:" + ls.Cell.String() +
		"\tRelPos:" + ls.RelPos.String() + "\tPos:" + ls.Pos.String()
}

// ContentRangeDim returns the Content bounding box min, max
// along given dimension
func (ls *LayState) ContentRangeDim(d mat32.Dims) (cmin, cmax float32) {
	cmin = float32(mat32.PointDim(ls.ContentBBox.Min, d))
	cmax = float32(mat32.PointDim(ls.ContentBBox.Max, d))
	return
}

// TotalRect returns the Pos, Size.Total geom
// as an image.Rectangle, e.g., for bounding box
func (ls *LayState) TotalRect() image.Rectangle {
	return mat32.RectFromPosSizeMax(ls.Pos, ls.Size.Total)
}

// ContentRect returns the ContentPos, Size.Content geom
// as an image.Rectangle, e.g., for bounding box
func (ls *LayState) ContentRect() image.Rectangle {
	return mat32.RectFromPosSizeMax(ls.ContentPos, ls.Size.Content)
}

//////////////////////////////////////////////////////////////
//  LayImplState -- for Layout only

// LayImplSizes holds the layout implementation sizing data for col, row dims
type LayImplSizes struct {
	Size mat32.Vec2
	Grow mat32.Vec2
}

func (ls *LayImplSizes) String() string {
	return fmt.Sprintf("Sizet: %v, \tGrow: %g", ls.Size, ls.Grow)
}

func (ls *LayImplSizes) Reset() {
	ls.Size.SetZero()
	ls.Grow.SetZero()
}

// LayImplState has internal state for implementing layout
type LayImplState struct {
	// Cells is number of cells along each dimension,
	// computed for each layout type.
	Cells image.Point `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// line breaks for wrap layout
	WrapBreaks []int `edit:"-" copy:"-" json:"-" xml:"-"`

	// sizes has the data for the columns in [0] and rows in [1]:
	// col Size.X = max(X over rows) (cross axis), .Y = sum(Y over rows) (main axis for col)
	// row Size.X = sum(X over cols) (main axis for row), .Y = max(Y over cols) (cross axis)
	// see: https://docs.google.com/spreadsheets/d/1eimUOIJLyj60so94qUr4Buzruj2ulpG5o6QwG2nyxRw/edit?usp=sharing
	Sizes [2][]LayImplSizes `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// todo: Flex has slice of sizes, one for each line

	// KidsSize is the actual size of the Children in this layout.
	// Drives ContentSize, and also the need for scrollbars when
	// greater than ContentSubGap (kids don't fit in avail space).
	KidsSize mat32.Vec2

	// ScrollSize has the scrollbar sizes (widths) for each dim, which adds extra space,
	// that is subtracted from Total to get Content size.
	// If there is a vertical scrollbar, X has width; if horizontal, Y has "width" = height
	ScrollSize mat32.Vec2

	// GapSize has the extra gap sizing between elements for each dimension,
	// which is part of Content for computing rendering bbox, but is
	// subtracted when allocating sizes to elements.
	GapSize mat32.Vec2

	// ContentSubGap is the amount to allocate to the Kids
	ContentSubGap mat32.Vec2
}

// InitSizes initializes the Sizes based on Cells geom
func (ls *LayImplState) InitSizes() {
	for d := mat32.X; d <= mat32.Y; d++ {
		n := mat32.PointDim(ls.Cells, d)
		if len(ls.Sizes[d]) != n {
			ls.Sizes[d] = make([]LayImplSizes, n)
		}
		for i := 0; i < n; i++ {
			ls.Sizes[d][i].Reset()
		}
	}
}

// GetKidsSize returns the actual sizes of our kids, using Cells
func (ls *LayImplState) GetKidsSize() mat32.Vec2 {
	var ksz mat32.Vec2
	for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
		n := mat32.PointDim(ls.Cells, ma) // cols, rows
		sum := float32(0)
		for mi := 0; mi < n; mi++ {
			md := &ls.Sizes[ma][mi] // X, Y
			mx := md.Size.Dim(ma)
			sum += mx // sum of maxes
		}
		ksz.SetDim(ma, sum)
	}
	return ksz
}

// Overflow is the difference between KidsSize - ContentSubGap,
// which drives the need for scrollbars.
func (ls *LayImplState) Overflow() mat32.Vec2 {
	return ls.KidsSize.Sub(ls.ContentSubGap)
}

// StackTopWidget returns the StackTop element as a widget
func (ly *Layout) StackTopWidget() (Widget, *WidgetBase) {
	sn, err := ly.ChildTry(ly.StackTop)
	if err != nil {
		return nil, nil
	}
	return AsWidget(sn)
}

func (ls *LayImplState) String() string {
	s := ""
	n := ls.Cells.X
	for i := 0; i < n; i++ {
		col := ls.Sizes[mat32.X][i]
		s += fmt.Sprintln("col:", i, "\tmax X:", col.Size.X, "\tsum Y:", col.Size.Y, "\tmax grX:", col.Grow.X, "\tsum grY:", col.Grow.Y)
	}
	n = ls.Cells.Y
	for i := 0; i < n; i++ {
		row := ls.Sizes[mat32.Y][i]
		s += fmt.Sprintln("row:", i, "\tsum X:", row.Size.X, "\tmax Y:", row.Size.Y, "\tsum grX:", row.Grow.X, "\tmax grY:", row.Grow.Y)
	}
	return s
}

//////////////////////////////////////////////////////////////////////
//		SizeUp

func (wb *WidgetBase) SizeUp(sc *Scene) {
	wb.SizeUpWidget(sc)
}

// SizeUpWidget is the standard Widget SizeUp pass
func (wb *WidgetBase) SizeUpWidget(sc *Scene) {
	wb.SizeFromStyle()
	wb.SizeUpParts(sc)
	wb.Alloc.Size.SetTotalFromContent()
}

// SizeFromStyle sets the initial Content size from Min style size,
// subject to Max constraints.
func (wb *WidgetBase) SizeFromStyle() {
	sz := &wb.Alloc.Size
	sz.Alloc.SetZero()
	sz.SetContentMax(wb.Styles.Min.Dots(), wb.Styles.Max.Dots())
	sz.SetSpace(wb.Styles.BoxSpace().Size())
	sz.SetTotalFromContent()
	if LayoutTrace && (wb.Alloc.Size.Content.X > 0 || wb.Alloc.Size.Content.Y > 0) {
		fmt.Println(wb, "SizeUp from style:", wb.Alloc.Size.String())
	}
}

// SizeUpParts adjusts the Content size to hold the Parts layout if present
func (wb *WidgetBase) SizeUpParts(sc *Scene) {
	if wb.Parts == nil {
		return
	}
	wb.Parts.SizeUp(sc)
	wb.Alloc.Size.SetContentToFit(wb.Parts.Alloc.Size.Total, wb.Styles.Max.Dots())
}

func (ly *Layout) SizeUp(sc *Scene) {
	ly.SizeUpLay(sc)
}

// SizeUpLay is the Layout standard SizeUp pass
func (ly *Layout) SizeUpLay(sc *Scene) {
	if !ly.HasChildren() {
		return
	}
	ly.SizeFromStyle()
	ly.SizeUpChildren(sc)
	ly.SetInitCells()
	ly.SizeUpKids(sc)
	ly.Alloc.Size.SetContentToFit(ly.LayImpl.KidsSize, ly.Styles.Max.Dots())
	ly.Alloc.Size.SetTotalFromContent()
}

// SizeUpChildren calls SizeUp on all the children of this node
func (ly *Layout) SizeUpChildren(sc *Scene) {
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.SizeUp(sc)
		return ki.Continue
	})
	return
}

// SetInitCells sets the initial default assignment of cell indexes
// to each widget, based on layout type.
func (ly *Layout) SetInitCells() {
	switch {
	case ly.Styles.Display == styles.DisplayFlex:
		ly.SetInitCellsFlex()
	case ly.Styles.Display == styles.DisplayStacked:
		ly.SetInitCellsStacked()
	case ly.Styles.Display == styles.DisplayGrid:
		ly.SetInitCellsGrid()
	default:
		ly.SetInitCellsStacked() // whatever
	}
}

func (ly *Layout) SetGapSizeFromCells() {
	ly.LayImpl.GapSize.X = max(float32(ly.LayImpl.Cells.X-1)*ly.Styles.Gap.X.Dots, 0)
	ly.LayImpl.GapSize.Y = max(float32(ly.LayImpl.Cells.Y-1)*ly.Styles.Gap.Y.Dots, 0)
}

func (ly *Layout) SetInitCellsFlex() {
	ma := ly.Styles.MainAxis
	ca := ma.OtherDim()
	idx := 0
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		mat32.SetPointDim(&kwb.Alloc.Cell, ma, idx)
		idx++
		return ki.Continue
	})
	if idx == 0 {
		fmt.Println(ly, "no items:", idx)
	}
	mat32.SetPointDim(&ly.LayImpl.Cells, ma, max(idx, 1)) // must be at least 1
	mat32.SetPointDim(&ly.LayImpl.Cells, ca, 1)
	ly.SetGapSizeFromCells()
}

func (ly *Layout) SetInitCellsStacked() {
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwb.Alloc.Cell = image.Point{0, 0}
		return ki.Continue
	})
	ly.LayImpl.Cells = image.Point{1, 1}
	ly.SetGapSizeFromCells() // always 0
}

func (ly *Layout) SetInitCellsGrid() {
	n := len(*ly.Children())
	cols := ly.Styles.Columns
	if cols == 0 {
		cols = int(mat32.Sqrt(float32(n)))
	}
	rows := n / cols
	for rows*cols < n {
		rows++
	}
	if rows == 0 || cols == 0 {
		fmt.Println(ly, "no rows or cols:", rows, cols)
	}
	ly.LayImpl.Cells = image.Point{max(cols, 1), max(rows, 1)}
	ci := 0
	ri := 0
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwb.Alloc.Cell = image.Point{ci, ri}
		ci++
		cs := kwb.Styles.ColSpan
		if cs > 1 {
			ci += cs - 1
		}
		if ci >= cols {
			ci = 0
			ri++
		}
		return ki.Continue
	})
	ly.SetGapSizeFromCells()
}

// todo: wrap requires a different non-grid logic -- no constraint of same sizes across rows

//////////////////////////////////////////////////////////////////////
//		SizeUpKids

// SizeUpKids gathers actual size values from kids, puts in KidsSize
func (ly *Layout) SizeUpKids(sc *Scene) {
	// todo: flex
	if ly.Styles.Display == styles.DisplayStacked {
		ly.SizeUpKidsStacked(sc)
		return
	}
	ly.SizeUpKidsCells(sc)
}

// SizeUpKidsCells for Flex, Grid
func (ly *Layout) SizeUpKidsCells(sc *Scene) {
	// r   0   1   col X = max(X over rows), Y = sum(Y over rows)
	//   +--+--+
	// 0 |  |  |   row X = sum(X over cols), Y = max(Y over cols)
	//   +--+--+
	// 1 |  |  |
	//   +--+--+
	ly.LayImpl.InitSizes()
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Alloc.Cell
		sz := kwb.Alloc.Size.Total
		grw := kwb.Styles.Grow
		if LayoutTraceDetail {
			fmt.Println("szup i:", i, kwb, "cidx:", cidx, "sz:", sz, "grw:", grw, "csz:", kwb.Alloc.Size.Content)
		}
		for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
			ca := ma.OtherDim()             // cross axis = Y then X
			mi := mat32.PointDim(cidx, ma)  // X, Y
			ci := mat32.PointDim(cidx, ca)  // Y, X
			md := &ly.LayImpl.Sizes[ma][mi] // X, Y
			cd := &ly.LayImpl.Sizes[ca][ci] // Y, X
			msz := sz.Dim(ma)               // main axis size dim: X, Y
			mx := md.Size.Dim(ma)
			mx = max(mx, msz) // Col, max widths of all elements; Row, max heights of all elements
			md.Size.SetDim(ma, mx)
			sm := cd.Size.Dim(ma)
			sm += msz
			cd.Size.SetDim(ma, sm) // Row, sum widths of all elements; Col, sum heights of all elements
			gsz := grw.Dim(ma)
			mx = md.Grow.Dim(ma)
			mx = max(mx, gsz)
			md.Grow.SetDim(ma, mx)
			sm = cd.Grow.Dim(ma)
			sm += gsz
			cd.Grow.SetDim(ma, sm)
		}
		return ki.Continue
	})
	if LayoutTraceDetail {
		fmt.Println(ly, "SizeUp")
		fmt.Println(ly.LayImpl.String())
	}

	ksz := ly.LayImpl.GetKidsSize()
	ksz.SetCeil()
	if LayoutTrace {
		fmt.Println(ly, "SizeUpKids Cells KidsSize:", ksz)
	}
	ly.LayImpl.KidsSize = ksz
}

// SizeUpKidsStacked
func (ly *Layout) SizeUpKidsStacked(sc *Scene) {
	ly.LayImpl.InitSizes()
	_, kwb := ly.StackTopWidget()
	if kwb != nil {
		ly.LayImpl.Sizes[0][0].Size = kwb.Alloc.Size.Total
		ly.LayImpl.Sizes[1][0].Size = kwb.Alloc.Size.Total
		ly.LayImpl.Sizes[0][0].Grow = kwb.Styles.Grow
		ly.LayImpl.Sizes[1][0].Grow = kwb.Styles.Grow
	}
	ksz := ly.LayImpl.Sizes[0][0].Size // all the same
	ksz.SetCeil()
	if LayoutTrace {
		fmt.Println(ly, "SizeUpKids Stacked KidsSize:", ksz)
	}
	ly.LayImpl.KidsSize = ksz
}

//////////////////////////////////////////////////////////////////////
//		SizeDown

func (wb *WidgetBase) SizeDown(sc *Scene, iter int) bool {
	return wb.SizeDownWidget(sc, iter)
}

// SizeDownWidget is the standard widget implementation of SizeDown.
// Computes updated Content size from given total allocation
// and gives that content to its parts if they exist.
func (wb *WidgetBase) SizeDownWidget(sc *Scene, iter int) bool {
	// re := wb.SizeDownGrowToAlloc(sc, iter) // prevents word wrapping but also prevents super wide labels
	redo := wb.SizeDownParts(sc, iter) // give our content to parts
	return redo                        // || re
}

// SizeDownGrowToAlloc grows our Total size up to current Alloc size
// for any dimension with a non-zero Grow factor.
// The parent Layout uses the proportional nature of Grow to determine
// relative size of Alloc, but here we use all of Alloc if non-zero.
// Returns true if this resulted in a change in our Total size.
func (wb *WidgetBase) SizeDownGrowToAlloc(sc *Scene, iter int) bool {
	change := false
	sz := &wb.Alloc.Size
	s := &wb.Styles
	totWas := sz.Total
	if wb.Nm == "trow" || strings.HasPrefix(wb.Nm, "scrollbar") {
		fmt.Println(wb, s.Grow)
	}
	sz.Total.Clamp(mat32.Vec2Zero, sz.Alloc)
	if s.Grow.X > 0 && sz.Alloc.X > sz.Total.X {
		change = true
		sz.Total.X = sz.Alloc.X
	}
	if s.Grow.Y > 0 && sz.Alloc.Y > sz.Total.Y {
		change = true
		sz.Total.Y = sz.Alloc.Y
	}
	if change {
		sz.SetContentFromTotal()
		if wb.Styles.LayoutHasParSizing() {
			// todo: requires some additional logic to see if actually changes something
		}
		if LayoutTrace {
			fmt.Println(wb, "changed Total size based on Alloc.  Was:", totWas, "Alloc:", sz.Alloc)
		}
	}
	return change
}

func (wb *WidgetBase) SizeDownParts(sc *Scene, iter int) bool {
	if wb.Parts == nil {
		return false
	}
	wb.Parts.Alloc.Size.Alloc = wb.Alloc.Size.Content // parts are our content
	return wb.Parts.SizeDown(sc, iter)
}

// SizeDownChildren calls SizeDown on the Children.  The kids
// need to have their Alloc.Size.Alloc set prior to this, which
// is what Layout type does.  Other special widget types can
// do custom layout and call this too.
func (wb *WidgetBase) SizeDownChildren(sc *Scene, iter int) bool {
	redo := false
	wb.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		re := kwi.SizeDown(sc, iter)
		redo = redo || re
		return ki.Continue
	})
	return redo
}

func (ly *Layout) SizeDown(sc *Scene, iter int) bool {
	redo := ly.SizeDownLay(sc, iter)
	if redo && LayoutTrace {
		fmt.Println(ly, "SizeDown redo")
	}
	return redo
}

// SizeDownLay is the Layout standard SizeDown pass, returning true if another
// iteration is required.  It allocates sizes to fit given parent-allocated
// total size.
func (ly *Layout) SizeDownLay(sc *Scene, iter int) bool {
	if !ly.HasChildren() {
		return false
	}
	totalChanged := ly.SizeDownGrowToAlloc(sc, iter)
	chg := ly.ManageOverflow(sc, iter)
	sz := &ly.Alloc.Size
	ly.LayImpl.ContentSubGap = sz.Content.Sub(ly.LayImpl.GapSize)

	conDiff := ly.LayImpl.ContentSubGap.Sub(ly.LayImpl.KidsSize) // vs. actual kids
	if conDiff.X > 0 || conDiff.Y > 0 {
		if LayoutTrace {
			fmt.Println("szdn growing:", ly, "diff:", conDiff, "was:", ly.LayImpl.KidsSize, "now:", ly.LayImpl.ContentSubGap, "gapsize:", ly.LayImpl.GapSize)
		}
		ly.SizeDownGrow(sc, iter, conDiff)
	} else {
		ly.SizeDownAlloc(sc, iter) // set allocations as is
	}
	re := ly.SizeDownChildren(sc, iter)
	ly.SizeUpKids(sc)  // always update KidsSize
	if ly.Par != nil { // cannot change the scene content size
		ly.Alloc.Size.SetContentToFit(ly.LayImpl.KidsSize, ly.Styles.Max.Dots())
		ly.Alloc.Size.SetTotalToFitContent()
	}
	return chg || totalChanged || re
}

// ManageOverflow uses overflow settings to determine if scrollbars
// are needed, etc.  Returns true if size changes as a result.
// Based on the allocation mechanisms, our Total never exceeds Alloc
// but KidsSize represents the true actual needs of the child elements,
// so that is what drives scrolling.
func (ly *Layout) ManageOverflow(sc *Scene, iter int) bool {
	sz := &ly.Alloc.Size
	change := false
	if iter == 0 {
		ly.LayImpl.ScrollSize.SetZero()
		ly.SetScrollsOff()
	}
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.Styles.Overflow.Dim(d) == styles.OverflowScroll {
			if !ly.HasScroll[d] {
				change = true
			}
			ly.HasScroll[d] = true
			ly.LayImpl.ScrollSize.SetDim(d.OtherDim(), ly.Styles.ScrollBarWidth.Dots+4)
		}
	}
	if !sc.Is(ScPrefSizing) {
		oflow := ly.LayImpl.Overflow() // KidsSize - ContentSubGap
		for d := mat32.X; d <= mat32.Y; d++ {
			ofd := oflow.Dim(d)
			if ofd > 0 {
				switch ly.Styles.Overflow.Dim(d) {
				// case styles.OverflowVisible:
				// note: this shouldn't happen -- just have this in here for monitoring
				// fmt.Println(ly, "OverflowVisible ERROR -- shouldn't have overflow:", d, ofd)
				case styles.OverflowAuto:
					if !ly.HasScroll[d] {
						change = true
					}
					ly.HasScroll[d] = true
					ly.LayImpl.ScrollSize.SetDim(d.OtherDim(), ly.Styles.ScrollBarWidth.Dots+4)
					if LayoutTrace {
						fmt.Println(ly, "OverflowAuto enabling scrollbars for dim for overflow:", d, ofd)
					}
				}
			}
		}
	}
	sz.SetSpace(ly.LayImpl.ScrollSize.Add(ly.Styles.BoxSpace().Size()))
	if ly.Par == nil { // nowhere to go
		sz.SetContentFromTotal()
	} else {
		oldTot := sz.Total
		sz.SetTotalFromContent()
		if oldTot != sz.Total {
			change = true
		}
	}
	return change
}

// SizeDownGrow grows the element sizes based on total extra and Grow
// factors
func (ly *Layout) SizeDownGrow(sc *Scene, iter int, diff mat32.Vec2) bool {
	redo := false
	// if ly.Styles.Display == styles.DisplayFlex && ly.Styles.Wrap {
	// 	redo = ly.SizeDownGrowWrap(sc, iter, diff) // first recompute wrap
	// todo: use special version of grow
	// } else
	if ly.Styles.Display == styles.DisplayStacked {
		redo = ly.SizeDownGrowStacked(sc, iter, diff)
	} else {
		redo = ly.SizeDownGrowCells(sc, iter, diff)
	}
	return redo
}

func (ly *Layout) SizeDownGrowCells(sc *Scene, iter int, diff mat32.Vec2) bool {
	redo := false
	// todo: use max growth values instead of individual ones to ensure consistency!
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Alloc.Cell
		sz := kwb.Alloc.Size.Total
		_ = sz
		grw := kwb.Styles.Grow
		// if LayoutTrace {
		// 	fmt.Println("szdn i:", i, kwb, "cidx:", cidx, "sz:", sz, "grw:", grw)
		// }
		for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
			gr := grw.Dim(ma)
			ca := ma.OtherDim()   // cross axis = Y then X
			extra := diff.Dim(ma) // row.X = extra width for cols; col.Y = extra height for rows in this col
			if extra < 0 {
				extra = 0
			}
			mi := mat32.PointDim(cidx, ma)  // X, Y
			ci := mat32.PointDim(cidx, ca)  // Y, X
			md := &ly.LayImpl.Sizes[ma][mi] // X, Y
			cd := &ly.LayImpl.Sizes[ca][ci] // Y, X
			mx := md.Size.Dim(ma)
			asz := mx
			gsum := cd.Grow.Dim(ma)
			if gsum > 0 {
				redo = true
				asz = mx + extra*(gr/gsum)
			}
			kwb.Alloc.Size.Alloc.SetDim(ma, asz)
		}
		kwb.Alloc.Size.Alloc.SetFloor()
		return ki.Continue
	})
	return redo
}

func (ly *Layout) SizeDownGrowWrap(sc *Scene, iter int, diff mat32.Vec2) bool {
	// todo
	return false
}

func (ly *Layout) SizeDownGrowStacked(sc *Scene, iter int, diff mat32.Vec2) bool {
	// note: not much we can do here
	return false
}

// SizeDownAlloc calls size down on kids with newly allocated sizes.
// returns true if any report needing a redo.
func (ly *Layout) SizeDownAlloc(sc *Scene, iter int) {
	if ly.Styles.Display == styles.DisplayStacked {
		ly.SizeDownAllocStacked(sc, iter)
		return
	}
	// todo: wrap needs special case
	ly.SizeDownAllocCells(sc, iter)
}

// SizeDownAllocGrid calls size down on kids with newly allocated sizes.
// returns true if any report needing a redo.
func (ly *Layout) SizeDownAllocCells(sc *Scene, iter int) {
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Alloc.Cell
		for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
			mi := mat32.PointDim(cidx, ma)  // X, Y
			md := &ly.LayImpl.Sizes[ma][mi] // X, Y
			asz := md.Size.Dim(ma)
			kwb.Alloc.Size.Alloc.SetDim(ma, asz)
		}
		kwb.Alloc.Size.Alloc.SetFloor()
		return ki.Continue
	})
}

func (ly *Layout) SizeDownAllocStacked(sc *Scene, iter int) {
	if ly.Is(LayoutStackTopOnly) {
		_, kwb := ly.StackTopWidget()
		if kwb != nil {
			kwb.Alloc.Size.Alloc = ly.Alloc.Size.Content
		}
		return
	}

	// note: allocate everyone in case they are flipped to top
	// need a new layout if size is actually different
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwb.Alloc.Size.Alloc = ly.Alloc.Size.Content
		return ki.Continue
	})
}

//////////////////////////////////////////////////////////////////////
//		Position

// Position uses the final sizes to position everything within layouts
// according to alignment settings.  It only sets the PosRel relative positions.
// The final layout step of ScenePos computes scene-relative positions, and
// is called separately whenever scrolling happens.
func (wb *WidgetBase) Position(sc *Scene) {
	wb.PositionWidget(sc)
}

func (wb *WidgetBase) PositionWidget(sc *Scene) {
	wb.StyleSizeUpdate(sc) // now that sizes are stable, ensure styling based on size is updated
	wb.PositionParts(sc)
}

// StyleSizeUpdate updates styling size values for widget and its parent,
// which should be called after these are updated.  Returns true if any changed.
func (wb *WidgetBase) StyleSizeUpdate(sc *Scene) bool {
	el := wb.Alloc.Size.Content
	var par mat32.Vec2
	_, pwb := wb.ParentWidget()
	if pwb != nil {
		par = pwb.Alloc.Size.Content
	}
	sz := sc.Geom.Size
	chg := wb.Styles.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, par.X, par.Y)
	if chg {
		wb.Styles.ToDots()
	}
	return chg
}

func (wb *WidgetBase) PositionParts(sc *Scene) {
	if wb.Parts == nil {
		return
	}
	wb.Parts.Position(sc)
}

// PositionChildren runs Position on the children
func (wb *WidgetBase) PositionChildren(sc *Scene) {
	wb.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.Position(sc)
		return ki.Continue
	})
}

// Position: uses the final sizes to position everything within layouts
// according to alignment settings.
func (ly *Layout) Position(sc *Scene) {
	ly.PositionLay(sc)
}

func (ly *Layout) PositionLay(sc *Scene) {
	if !ly.HasChildren() {
		return
	}
	ly.StyleSizeUpdate(sc) // now that sizes are stable, ensure styling based on size is updated
	ly.ConfigScrolls(sc)   // and configure the scrolls
	if ly.Styles.Display == styles.DisplayStacked {
		ly.PositionStacked(sc)
	} else {
		ly.PositionCells(sc)
	}
	ly.PositionChildren(sc)
}

func (ly *Layout) PositionCells(sc *Scene) {
	var pos mat32.Vec2
	var lastAsz mat32.Vec2
	gap := ly.Styles.Gap.Dots()

	sz := &ly.Alloc.Size
	var stspc mat32.Vec2
	contAvail := sz.Alloc.Sub(sz.Space).Sub(ly.LayImpl.GapSize)
	cdiff := contAvail.Sub(ly.LayImpl.KidsSize)
	if cdiff.X > 0 {
		stspc.X += styles.AlignFactor(ly.Styles.Align.X) * cdiff.X
		if LayoutTrace {
			fmt.Println("pos grid:", ly, "extra X:", cdiff.X, "start X:", stspc.X, "align:", ly.Styles.Align.X, "factor:", styles.AlignFactor(ly.Styles.Align.X))
		}
	}
	if cdiff.Y > 0 {
		stspc.Y += styles.AlignFactor(ly.Styles.Align.Y) * cdiff.Y
		if LayoutTrace {
			fmt.Println("pos grid:", ly, "extra Y:", cdiff.Y, "start Y:", stspc.Y, "align:", ly.Styles.Align.Y, "factor:", styles.AlignFactor(ly.Styles.Align.Y))
		}
	}
	pos = stspc
	var maxs mat32.Vec2
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Alloc.Cell
		sz := kwb.Alloc.Size.Total
		asz := kwb.Alloc.Size.Alloc
		if cidx.X == 0 && i > 0 {
			pos.X = stspc.X
			pos.Y += lastAsz.Y + gap.Y
		}
		ep := pos
		if sz.X < asz.X {
			ep.X += styles.AlignFactor(kwb.Styles.Align.X) * (asz.X - sz.X)
		}
		if sz.Y < asz.Y {
			ep.Y += styles.AlignFactor(kwb.Styles.Align.Y) * (asz.Y - sz.Y)
		}
		ep.SetRound()
		if LayoutTraceDetail {
			fmt.Println("pos i:", i, kwb, "cidx:", cidx, "sz:", sz, "asz:", asz, "pos:", ep)
		}
		kwb.Alloc.RelPos = ep
		maxs.SetMax(ep.Add(asz))
		pos.X += asz.X + gap.X
		lastAsz = asz
		return ki.Continue
	})
	if LayoutTrace {
		fmt.Println("pos:", ly, "max:", maxs)
	}
}

func (ly *Layout) PositionWrap(sc *Scene) {
}

func (ly *Layout) PositionStacked(sc *Scene) {
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwb.Alloc.RelPos.SetZero()
		return ki.Continue
	})
}

//////////////////////////////////////////////////////////////////////
//		ScenePos

// ScenePos: scene-based position and final BBox is computed based on
// parents accumulated position and scrollbar position.
// This step can be performed when scrolling after updating Scroll.
func (wb *WidgetBase) ScenePos(sc *Scene) {
	wb.ScenePosWidget(sc)
}

func (wb *WidgetBase) ScenePosWidget(sc *Scene) {
	wb.SetPosFromParent(sc)
	wb.SetBBoxes(sc)
}

// SetContentPosFromPos sets the ContentPos position based on current Pos
// plus the BoxSpace position offset.
func (wb *WidgetBase) SetContentPosFromPos() {
	spc := wb.Styles.BoxSpace()
	off := spc.Pos()
	off.SetFloor()
	wb.Alloc.ContentPos = wb.Alloc.Pos.Add(off)
}

func (wb *WidgetBase) SetPosFromParent(sc *Scene) {
	_, pwb := wb.ParentWidget()
	var parPos mat32.Vec2
	if pwb != nil {
		parPos = pwb.Alloc.ContentPos.Add(pwb.Alloc.Scroll) // critical that parent adds here but not to self
	}
	wb.Alloc.Pos = wb.Alloc.RelPos.Add(parPos)
	wb.SetContentPosFromPos()
	if LayoutTrace {
		fmt.Println(wb, "pos:", wb.Alloc.Pos, "parPos:", parPos)
	}
}

// SetBBoxesFromAllocs sets BBox and ContentBBox from Alloc.Pos and .Size
// This does NOT intersect with parent content BBox, which is done in SetBBoxes.
// Use this for elements that are dynamically positioned outside of parent BBox.
func (wb *WidgetBase) SetBBoxesFromAllocs() {
	wb.Alloc.BBox = wb.Alloc.TotalRect()
	wb.Alloc.ContentBBox = wb.Alloc.ContentRect()
}

func (wb *WidgetBase) SetBBoxes(sc *Scene) {
	_, pwb := wb.ParentWidget()
	var parBB image.Rectangle
	if pwb != nil {
		parBB = pwb.Alloc.ContentBBox
	} else {
		parBB.Max = sc.Geom.Size
	}
	bb := mat32.RectFromPosSizeMax(wb.Alloc.Pos, wb.Alloc.Size.Total) // todo: alloc fixes widgets..
	wb.Alloc.BBox = parBB.Intersect(bb)
	if LayoutTrace {
		fmt.Println(wb, "Total BBox:", bb, "parBB:", parBB, "BBox:", wb.Alloc.BBox)
	}

	cbb := mat32.RectFromPosSizeMax(wb.Alloc.ContentPos, wb.Alloc.Size.Content)
	wb.Alloc.ContentBBox = parBB.Intersect(cbb)
	if LayoutTrace {
		fmt.Println(wb, "Content BBox:", cbb, "parBB:", parBB, "BBox:", wb.Alloc.ContentBBox)
	}
	wb.ScenePosParts(sc)
}

func (wb *WidgetBase) ScenePosParts(sc *Scene) {
	if wb.Parts == nil {
		return
	}
	wb.Parts.ScenePos(sc)
}

// ScenePosChildren runs ScenePos on the children
func (wb *WidgetBase) ScenePosChildren(sc *Scene) {
	wb.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.ScenePos(sc)
		return ki.Continue
	})
}

// ScenePos: scene-based position and final BBox is computed based on
// parents accumulated position and scrollbar position.
// This step can be performed when scrolling after updating Scroll.
func (ly *Layout) ScenePos(sc *Scene) {
	ly.GetScrollPosition(sc)
	ly.ScenePosWidget(sc)
	ly.ScenePosChildren(sc)
	ly.PositionScrolls(sc)
}

/*
//////////////////////////////////////////////////////
//  Wrap

// GatherSizesFlow is size first pass: gather the size information from the children
func GatherSizesFlow(ly *Layout, iter int) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	if iter > 0 {
		ly.ChildrenUpdateSizes() // essential to call this
		prv := ly.ChildSize
		ly.LayState.Size.Need = prv
		ly.LayState.Size.Pref = prv
		ly.Alloc.Size.Total = prv
		ly.LayState.UpdateSizes() // enforce max and normal ordering, etc
		if LayoutTrace {
			fmt.Printf("Size:   %v iter 1 fix size: %v\n", ly.Path(), prv)
		}
		return
	}

	sumPref, sumNeed, maxPref, maxNeed := GatherSizesSumMax(ly)

	// for flow, the need is always *maxNeed* (i.e., a single item)
	// and the pref is based on styled pref estimate

	sdim := LaySummedDim(ly.Lay)
	odim := mat32.OtherDim(sdim)
	pref := ly.LayState.Size.Pref.Dim(sdim)
	// opref := ly.LayState.Size.Pref.Dim(odim) // not using

	if pref == 0 {
		pref = 6 * maxNeed.Dim(sdim) // 6 items preferred backup default
	}
	if pref == 0 {
		pref = 200 // final backstop
	}

	if LayoutTrace {
		fmt.Printf("Size:   %v flow pref start: %v\n", ly.Path(), pref)
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

	spc := ly.BoxSpace()
	ly.LayState.Size.Need.SetAdd(spc.Size())
	ly.LayState.Size.Pref.SetAdd(spc.Size())

	elspc := float32(0.0)
	if sz >= 2 {
		elspc = float32(sz-1) * ly.Styles.Gap.Dots
	}
	if LaySumDim(ly.Lay, mat32.X) {
		ly.LayState.Size.Need.X += elspc
		ly.LayState.Size.Pref.X += elspc
	}
	if LaySumDim(ly.Lay, mat32.Y) {
		ly.LayState.Size.Need.Y += elspc
		ly.LayState.Size.Pref.Y += elspc
	}

	ly.LayState.UpdateSizes() // enforce max and normal ordering, etc
	if LayoutTrace {
		fmt.Printf("Size:   %v gather sizes need: %v, pref: %v, elspc: %v\n", ly.Path(), ly.LayState.Size.Need, ly.LayState.Size.Pref, elspc)
	}
}


////////////////////////////////////////////////////////////////////////////////////////
//     Layout children

// LayoutFlow manages the flow layout along given dimension
// returns true if needs another iteration (only if iter == 0)
func LayoutFlow(ly *Layout, dim mat32.Dims, iter int) bool {
	ly.FlowBreaks = nil
	sz := len(ly.Kids)
	if sz == 0 {
		return false
	}

	elspc := float32(sz-1) * ly.Styles.Gap.Dots
	spc := ly.BoxSpace()
	exspc := spc.Size().Dim(dim) + elspc

	avail := ly.Alloc.Size.Total.Dim(dim) - exspc
	odim := mat32.OtherDim(dim)

	// SidesTODO: might be odim
	pos := spc.Pos().Dim(dim)
	for i, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Widget).AsWidget()
		if ni == nil {
			continue
		}
		size := ni.LayState.Size.Need.Dim(dim)
		if pos+size > avail {
			ly.FlowBreaks = append(ly.FlowBreaks, i)
			pos = spc.Pos().Dim(dim)
		}
		ni.Alloc.Size.Total.SetDim(dim, size)
		ni.Alloc.PosRel.SetDim(dim, pos)
		if LayoutTrace {
			fmt.Printf("Layout: %v Child: %v, pos: %v, size: %v, need: %v, pref: %v\n", ly.Path(), ni.Nm, pos, size, ni.LayState.Size.Need.Dim(dim), ni.LayState.Size.Pref.Dim(dim))
		}
		pos += size + ly.Styles.Gap.Dots
	}
	ly.FlowBreaks = append(ly.FlowBreaks, len(ly.Kids))

	nrows := len(ly.FlowBreaks)
	oavail := ly.Alloc.Size.Total.Dim(odim) - exspc
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
			ni := c.(Widget).AsWidget()
			if ni == nil {
				continue
			}
			ni.StyMu.RLock()
			al := ni.Styles.AlignDim(odim)
			ni.StyMu.RUnlock()
			pref := ni.LayState.Size.Pref.Dim(odim)
			need := ni.LayState.Size.Need.Dim(odim)
			max := ni.LayState.Size.Max.Dim(odim)
			pos, size := LayoutSharedDimImpl(ly, oavPerRow, need, pref, max, spc, al)
			ni.Alloc.Size.Total.SetDim(odim, size)
			ni.Alloc.PosRel.SetDim(odim, rpos+pos)
			rmax = mat32.Max(rmax, size)
			nsz.X = mat32.Max(nsz.X, ni.Alloc.PosRel.X+ni.Alloc.Size.Total.X)
			nsz.Y = mat32.Max(nsz.Y, ni.Alloc.PosRel.Y+ni.Alloc.Size.Total.Y)
		}
		rpos += rmax + ly.Styles.Gap.Dots
		ci = bi
	}
	ly.LayState.Size.Need = nsz
	ly.LayState.Size.Pref = nsz
	if LayoutTrace {
		fmt.Printf("Layout: %v Flow final size: %v\n", ly.Path(), nsz)
	}
	// if nrows == 1 {
	// 	return false
	// }
	return true
}

*/
