// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

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

	// Alloc is the top-down Total allocated size of the widget, computed by the Layout
	// it can be larger than the Total in case of non-Grow (stretch) elements.
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
	ls.Content.SetToCeil()
}

// SetContentToFit sets the Content size subject to given Max values
// which only apply if they are > 0.  Values are automatically
// rounded up to Ceil ints
func (ls *LaySize) SetContentToFit(sz mat32.Vec2, mx mat32.Vec2) {
	ls.Content.SetMaxPos(sz)
	ls.Content.SetMinPos(mx)
	ls.Content.SetToCeil()
}

// SetTotalFromContent sets the Total size as Content plus given extra space
func (ls *LaySize) SetTotalFromContent() {
	ls.Total = ls.Content.Add(ls.Space)
}

// SetContentFromTotal sets the Content from Total size subtracting extra space
func (ls *LaySize) SetContentFromTotal() {
	ls.Content = ls.Total.Sub(ls.Space)
}

// LayoutState contains the the layout state for each widget,
// Set by the parent Layout during the Layout process.
type LayoutState struct {
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

func (ls LayoutState) String() string {
	return "Size:" + ls.Size.String() + "\tCell:" + ls.Cell.String() +
		"\tRelPos:" + ls.RelPos.String() + "\tPos:" + ls.Pos.String()
}

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

	// ScrollSize has the scrollbar sizes (widths) for each dim, which adds extra space,
	// that is subtracted from Total to get Content size.
	// If there is a vertical scrollbar, X has width; if horizontal, Y has "width" = height
	ScrollSize mat32.Vec2

	// GapSize has the extra gap sizing between elements for each dimension,
	// which is part of Content for computing rendering bbox, but is
	// subtracted when allocating sizes to elements.
	GapSize mat32.Vec2

	// ContentMinusGap is the amount to allocate
	ContentMinusGap mat32.Vec2
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
	ls.WrapBreaks = nil
}

func (ls *LayImplState) ContentSize() mat32.Vec2 {
	var csz mat32.Vec2
	for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
		n := mat32.PointDim(ls.Cells, ma) // cols, rows
		sum := float32(0)
		for mi := 0; mi < n; mi++ {
			md := &ls.Sizes[ma][mi] // X, Y
			mx := md.Size.Dim(ma)
			sum += mx // sum of maxes
		}
		csz.SetDim(ma, sum)
	}
	return csz
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
	wb.Alloc.Size.SetContentMax(wb.Styles.Min.Dots(), wb.Styles.Max.Dots())
	wb.Alloc.Size.Space = wb.Styles.BoxSpace().Size()
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
	ly.SizeFromStyle()
	ly.SizeUpKids(sc)
	ly.SetInitCells()
	ly.SizeUpCells(sc)
	ly.Alloc.Size.SetTotalFromContent()
}

// SizeUpKids calls SizeUp on all the children of this node
func (ly *Layout) SizeUpKids(sc *Scene) {
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
	ly.LayImpl.GapSize.X = float32(ly.LayImpl.Cells.X-1) * ly.Styles.Gap.X.Dots
	ly.LayImpl.GapSize.Y = float32(ly.LayImpl.Cells.Y-1) * ly.Styles.Gap.Y.Dots
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
	mat32.SetPointDim(&ly.LayImpl.Cells, ma, idx)
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
	ly.LayImpl.Cells = image.Point{cols, rows}
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

// SizeUpCells gathers sizing data for cells
func (ly *Layout) SizeUpCells(sc *Scene) {
	if ly.Styles.Display == styles.DisplayStacked {
		ly.SizeUpCellsStacked(sc)
		return
	}
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
		if LayoutTrace {
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
	if LayoutTrace {
		fmt.Println("szup:", ly, "\n", ly.LayImpl.String())
	}

	csz := ly.LayImpl.ContentSize()
	if LayoutTrace {
		fmt.Println("szup Content Size:", csz)
	}
	ly.Alloc.Size.SetContentToFit(csz, ly.Styles.Max.Dots())

}

// SizeUpCellsStacked
func (ly *Layout) SizeUpCellsStacked(sc *Scene) {
	_, kwb := ly.StackTopWidget()
	if kwb == nil {
		return
	}
	ly.LayImpl.Sizes[0][0].Size = kwb.Alloc.Size.Total
	ly.LayImpl.Sizes[1][0].Size = kwb.Alloc.Size.Total
	ly.LayImpl.Sizes[0][0].Grow = kwb.Styles.Grow
	ly.LayImpl.Sizes[0][0].Grow = kwb.Styles.Grow
	ly.Alloc.Size.SetContentToFit(ly.LayImpl.Sizes[0][0].Size, ly.Styles.Max.Dots())
}

//////////////////////////////////////////////////////////////////////
//		SizeDown

func (wb *WidgetBase) SizeDown(sc *Scene, iter int, allocTotal mat32.Vec2) bool {
	return wb.SizeDownWidget(sc, iter, allocTotal)
}

// SizeDownWidget is the standard widget implementation of SizeDown,
// computing updated Content size from given total allocation
// and giving that content to its parts if they exist.
func (wb *WidgetBase) SizeDownWidget(sc *Scene, iter int, allocTotal mat32.Vec2) bool {
	wb.Alloc.Size.Total = allocTotal
	wb.Alloc.Size.SetContentFromTotal()
	if wb.Styles.LayoutHasParSizing() {
		// todo: requires some additional logic to see if actually changes something
	}
	redo := wb.SizeDownParts(sc, iter, wb.Alloc.Size.Content) // give our content to parts
	return redo
}

func (wb *WidgetBase) SizeDownParts(sc *Scene, iter int, allocTotal mat32.Vec2) bool {
	if wb.Parts == nil {
		return false
	}
	return wb.Parts.SizeDown(sc, iter, allocTotal)
}

func (ly *Layout) SizeDown(sc *Scene, iter int, allocTotal mat32.Vec2) bool {
	redo := ly.SizeDownLay(sc, iter, allocTotal)
	re := ly.SizeDownChildren(sc, iter)
	return redo || re
}

// SizeDownLay is the Layout standard SizeDown pass, returning true if another
// iteration is required.  It allocates sizes to fit given parent-allocated
// total size.
func (ly *Layout) SizeDownLay(sc *Scene, iter int, allocTotal mat32.Vec2) bool {
	// todo: need to add gap sizes!
	ourTotal := ly.Alloc.Size.Total
	var totSz mat32.Vec2
	for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
		allTot := allocTotal.Dim(ma)
		ourTot := ourTotal.Dim(ma)
		var tsz float32
		switch ly.Styles.Overflow.Dim(ma) {
		case styles.OverflowVisible:
			tsz = max(ourTot, allTot) // get everything we need or are given
			// no scrollbars
		case styles.OverflowHidden:
			tsz = allTot // get what we are given
			// no scrollbars
		case styles.OverflowAuto, styles.OverflowScroll:
			tsz = max(ourTot, allTot) // get everything we need or are given
			if ourTot > allTot {
				ly.LayImpl.ScrollSize.SetDim(ma.OtherDim(), ly.Styles.ScrollBarWidth.Dots)
			}
		}
		totSz.SetDim(ma, tsz)
	}
	prevContent := ly.Alloc.Size.Content
	if iter > 0 {
		prevContent = ly.LayImpl.ContentMinusGap
	}
	ly.Alloc.Size.Total = totSz
	spctot := ly.LayImpl.ScrollSize.Add(ly.Styles.BoxSpace().Size())
	ly.Alloc.Size.Space = spctot
	ly.Alloc.Size.SetContentFromTotal()
	ly.LayImpl.ContentMinusGap = ly.Alloc.Size.Content.Sub(ly.LayImpl.GapSize)

	conDiff := ly.LayImpl.ContentMinusGap.Sub(prevContent)
	redo := false
	if conDiff.X > 0 || conDiff.Y > 0 {
		if LayoutTrace {
			fmt.Println("szdn growing:", ly, "diff:", conDiff, "was:", prevContent, "now:", ly.LayImpl.ContentMinusGap, "gapsize:", ly.LayImpl.GapSize)
		}
		redo = ly.SizeDownGrow(sc, iter, conDiff)
	} else {
		ly.SizeDownAlloc(sc, iter) // set allocations as is
	}
	return redo
}

// SizeDownGrow grows the element sizes based on total extra and Grow
// factors
func (ly *Layout) SizeDownGrow(sc *Scene, iter int, diff mat32.Vec2) bool {
	redo := false
	if ly.Styles.Display == styles.DisplayFlex && ly.Styles.Wrap {
		redo = ly.SizeDownGrowWrap(sc, iter, diff) // first recompute wrap
		// todo: use special version of grow
	} else if ly.Styles.Display == styles.DisplayStacked {
		redo = ly.SizeDownGrowStacked(sc, iter, diff)
	} else {
		redo = ly.SizeDownGrowGrid(sc, iter, diff)
	}
	return redo
}

func (ly *Layout) SizeDownGrowGrid(sc *Scene, iter int, diff mat32.Vec2) bool {
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
	if redo {
		ly.SizeUpCells(sc)
	}
	return redo
}

func (ly *Layout) SizeDownGrowWrap(sc *Scene, iter int, diff mat32.Vec2) bool {
	// todo
	return false
}

func (ly *Layout) SizeDownGrowStacked(sc *Scene, iter int, diff mat32.Vec2) bool {
	// todo
	return false
}

func (ly *Layout) SizeDownChildren(sc *Scene, iter int) bool {
	redo := false
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		asz := kwb.Alloc.Size.Alloc
		re := kwi.SizeDown(sc, iter, asz)
		redo = redo || re
		return ki.Continue
	})
	return redo
}

// SizeDownAlloc calls size down on kids with newly allocated sizes.
// returns true if any report needing a redo.
func (ly *Layout) SizeDownAlloc(sc *Scene, iter int) {
	if ly.Styles.Display == styles.DisplayStacked {
		ly.SizeDownAllocStacked(sc, iter)
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
}

//////////////////////////////////////////////////////////////////////
//		Position

// Position: uses the final sizes to position everything within layouts
// according to alignment settings.
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
	return wb.Styles.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, par.X, par.Y)
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
	ly.StyleSizeUpdate(sc) // now that sizes are stable, ensure styling based on size is updated
	if ly.Styles.Display == styles.DisplayFlex && ly.Styles.Wrap {
		ly.PositionWrap(sc)
	} else if ly.Styles.Display == styles.DisplayStacked {
		ly.PositionStacked(sc)
	} else {
		ly.PositionGrid(sc)
	}
	ly.PositionChildren(sc)
}

func (ly *Layout) PositionGrid(sc *Scene) {
	var pos mat32.Vec2
	var lastAsz mat32.Vec2
	gap := ly.Styles.Gap.Dots()

	var stspc mat32.Vec2
	extot := ly.LayImpl.ScrollSize.Add(ly.LayImpl.GapSize).Add(ly.Styles.BoxSpace().Size())
	contAvail := ly.Alloc.Size.Alloc.Sub(extot)
	cdiff := contAvail.Sub(ly.Alloc.Size.Content)
	if cdiff.X > 0 {
		if LayoutTrace {
			stspc.X += styles.AlignFactor(ly.Styles.Align.X) * cdiff.X
			if LayoutTrace {
				fmt.Println("pos grid:", ly, "extra X:", cdiff.X, "start X:", stspc.X, "align:", ly.Styles.Align.X, "factor:", styles.AlignFactor(ly.Styles.Align.X))
			}
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
		if LayoutTrace {
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
	_, pwb := wb.ParentWidget()
	var parPos mat32.Vec2
	var parBB image.Rectangle
	if pwb != nil {
		parPos = pwb.Alloc.ContentPos
		parBB = pwb.Alloc.ContentBBox
	} else {
		parBB.Max = sc.Geom.Size
	}
	wb.Alloc.Pos = wb.Alloc.RelPos.Add(parPos).Add(wb.Alloc.Scroll)
	bb := mat32.RectFromPosSizeMax(wb.Alloc.Pos, wb.Alloc.Size.Total)
	wb.Alloc.BBox = parBB.Intersect(bb)
	if LayoutTrace {
		fmt.Println(wb, "pos:", wb.Alloc.Pos, "parPos:", parPos, "Total BBox:", bb, "parBB:", parBB, "BBox:", wb.Alloc.BBox)
	}

	spc := wb.Styles.BoxSpace()
	off := spc.Pos()
	wb.Alloc.ContentPos = wb.Alloc.Pos.Add(off)
	cbb := mat32.RectFromPosSizeMax(wb.Alloc.Pos.Add(off), wb.Alloc.Size.Content)
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
	ly.ScenePosWidget(sc)
	ly.ScenePosChildren(sc)
	// todo: do scrollbars here
}

/*
// LayoutSharedDim implements calculations to layout for the shared dimension
// (i.e., Vertical for Horizontal layout). Returns pos and size.
func LayoutSharedDimImpl(ly *Layout, avail, need, pref, max float32, spc styles.SideFloats, al styles.Align) (pos, size float32) {
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

	if ly.Lay == LayoutHoriz || ly.Lay == LayoutHorizFlow {
		pos = spc.Pos().Dim(mat32.Y)
	} else {
		pos = spc.Pos().Dim(mat32.X)
	}

	size = need
	if usePref {
		size = pref
	}
	if stretchMax || stretchNeed {
		size += extra
	} else {
		if styles.IsAlignMiddle(al) {
			pos += 0.5 * extra
		} else if styles.IsAlignEnd(al) {
			pos += extra
		} else if al == styles.AlignJustify { // treat justify as stretch
			size += extra
		}
	}

	// if LayoutTrace {
	// 	fmt.Printf("ly %v avail: %v targ: %v, extra %v, strMax: %v, strNeed: %v, pos: %v size: %v spc: %v\n", ly.Nm, avail, targ, extra, stretchMax, stretchNeed, pos, size, spc)
	// }

	return
}

// LayoutSharedDim lays out items along a shared dimension, where all elements
// share the same space, e.g., Horiz for a Vert layout, and vice-versa.
func LayoutSharedDim(ly *Layout, dim mat32.Dims) {
	spc := ly.BoxSpace()
	avail := ly.Alloc.Size.Total.Dim(dim) - spc.Size().Dim(dim)
	for i, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Widget).AsWidget()
		if ni == nil {
			continue
		}
		if ly.Lay == LayoutStacked && ly.Is(LayoutStackTopOnly) && i != ly.StackTop {
			continue
		}
		ni.StyMu.RLock()
		al := ni.Styles.AlignDim(dim)
		ni.StyMu.RUnlock()
		pref := ni.LayState.Size.Pref.Dim(dim)
		need := ni.LayState.Size.Need.Dim(dim)
		max := ni.LayState.Size.Max.Dim(dim)
		pos, size := LayoutSharedDimImpl(ly, avail, need, pref, max, spc, al)
		ni.Alloc.Size.Total.SetDim(dim, size)
		ni.Alloc.PosRel.SetDim(dim, pos)
	}
}

// LayoutAlongDim lays out all children along given dim -- only affects that dim --
// e.g., use LayoutSharedDim for other dim.
func LayoutAlongDim(ly *Layout, dim mat32.Dims) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	elspc := float32(sz-1) * ly.Styles.Gap.Dots
	al := ly.Styles.AlignDim(dim)
	spc := ly.BoxSpace()
	exspc := spc.Size().Dim(dim) + elspc
	avail := ly.Alloc.Size.Total.Dim(dim) - exspc
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
			ni := c.(Widget).AsWidget()
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
			ni := c.(Widget).AsWidget()
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
	if sz > 1 && extra > 0.0 && al == styles.AlignJustify && !stretchNeed && !stretchMax {
		addSpace = true
		// if neither, then just distribute as spacing for justify
		extraSpace = extra / float32(sz-1)
	}

	// now arrange everyone
	pos := spc.Pos().Dim(dim)

	// todo: need a direction setting too
	if styles.IsAlignEnd(al) && !stretchNeed && !stretchMax {
		pos += extra
	}

	if LayoutTrace {
		fmt.Printf("Layout: %v Along dim %v, avail: %v elspc: %v need: %v pref: %v targ: %v, extra %v, strMax: %v, strNeed: %v, nstr %v, strTot %v\n", ly.Path(), dim, avail, elspc, need, pref, targ, extra, stretchMax, stretchNeed, nstretch, stretchTot)
	}

	for i, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Widget).AsWidget()
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

		ni.Alloc.Size.Total.SetDim(dim, size)
		ni.Alloc.PosRel.SetDim(dim, pos)
		if LayoutTrace {
			fmt.Printf("Layout: %v Child: %v, pos: %v, size: %v, need: %v, pref: %v\n", ly.Path(), ni.Nm, pos, size, ni.LayState.Size.Need.Dim(dim), ni.LayState.Size.Pref.Dim(dim))
		}
		pos += size + ly.Styles.Gap.Dots
	}
}

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

// todo: grid does not process spans at all yet -- assumes = 1

// GatherSizesGrid is size first pass: gather the size information from the
// children, grid version
func GatherSizesGrid(ly *Layout) {
	if len(ly.Kids) == 0 {
		return
	}

	cols := ly.Styles.Columns
	rows := 0

	sz := len(ly.Kids)
	// collect overall size
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Widget).AsWidget()
		if ni == nil {
			continue
		}
		ni.StyMu.RLock()
		st := &ni.Styles
		ni.StyMu.RUnlock()
		if st.Col > 0 {
			cols = max(cols, st.Col+st.ColSpan)
		}
		if st.Row > 0 {
			rows = max(rows, st.Row+st.RowSpan)
		}
	}

	if cols == 0 {
		cols = int(mat32.Sqrt(float32(sz))) // whatever -- not well defined
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
		ni := c.(Widget).AsWidget()
		if ni == nil {
			continue
		}
		ni.LayState.UpdateSizes()
		ni.StyMu.RLock()
		st := &ni.Styles
		ni.StyMu.RUnlock()
		if st.Col > 0 {
			col = st.Col
		}
		if st.Row > 0 {
			row = st.Row
		}

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
	sc := ly.Sc
	if sc != nil && sc.Is(ScPrefSizing) {
		prefSizing = ly.Styles.Overflow == styles.OverflowScroll // special case
	}

	// if there aren't existing prefs, we need to compute size
	if prefSizing || ly.LayState.Size.Pref.X == 0 || ly.LayState.Size.Pref.Y == 0 {
		sbw := ly.Styles.ScrollBarWidth.Dots
		maxRow := len(ly.GridData[Row])
		maxCol := len(ly.GridData[Col])
		if prefSizing {
			maxRow = min(LayoutPrefMaxRows, maxRow)
			maxCol = min(LayoutPrefMaxCols, maxCol)
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

	spc := ly.BoxSpace()
	ly.LayState.Size.Need.SetAdd(spc.Size())
	ly.LayState.Size.Pref.SetAdd(spc.Size())

	ly.LayState.Size.Need.X += float32(cols-1) * ly.Styles.Gap.Dots
	ly.LayState.Size.Pref.X += float32(cols-1) * ly.Styles.Gap.Dots
	ly.LayState.Size.Need.Y += float32(rows-1) * ly.Styles.Gap.Dots
	ly.LayState.Size.Pref.Y += float32(rows-1) * ly.Styles.Gap.Dots

	ly.LayState.UpdateSizes() // enforce max and normal ordering, etc
	if LayoutTrace {
		fmt.Printf("Size:   %v gather sizes grid need: %v, pref: %v\n", ly.Path(), ly.LayState.Size.Need, ly.LayState.Size.Pref)
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

// LayoutGridDim lays out grid data along each dimension (row, Y; col, X),
// same as LayoutAlongDim.  For cols, X has width prefs of each -- turn that
// into an actual allocated width for each column, and likewise for rows.
func LayoutGridDim(ly *Layout, rowcol RowCol, dim mat32.Dims) {
	gds := ly.GridData[rowcol]
	sz := len(gds)
	if sz == 0 {
		return
	}
	elspc := float32(sz-1) * ly.Styles.Gap.Dots
	al := ly.Styles.AlignDim(dim)
	spc := ly.BoxSpace()
	exspc := spc.Size().Dim(dim) + elspc
	avail := ly.Alloc.Size.Total.Dim(dim) - exspc
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
	if sz > 1 && extra > 0.0 && al == styles.AlignJustify && !stretchNeed && !stretchMax {
		addSpace = true
		// if neither, then just distribute as spacing for justify
		extraSpace = extra / float32(sz-1)
	}

	// now arrange everyone
	pos := spc.Pos().Dim(dim)

	// todo: need a direction setting too
	if styles.IsAlignEnd(al) && !stretchNeed && !stretchMax {
		pos += extra
	}

	if LayoutTrace {
		fmt.Printf("Layout Grid Dim: %v All on dim %v, avail: %v need: %v pref: %v targ: %v, extra %v, strMax: %v, strNeed: %v, nstr %v, strTot %v\n", ly.Path(), dim, avail, need, pref, targ, extra, stretchMax, stretchNeed, nstretch, stretchTot)
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
		if LayoutTrace {
			fmt.Printf("Grid %v pos: %v, size: %v\n", rowcol, pos, size)
		}
		pos += size + ly.Styles.Gap.Dots
	}
}

// LayoutGridLay manages overall grid layout of children
func LayoutGridLay(ly *Layout) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	LayoutGridDim(ly, Row, mat32.Y)
	LayoutGridDim(ly, Col, mat32.X)

	col := 0
	row := 0
	cols := ly.GridSize.X
	rows := ly.GridSize.Y

	if cols*rows != ly.NumChildren() {
		GatherSizesGrid(ly)
	}

	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Widget).AsWidget()
		if ni == nil {
			continue
		}

		ni.StyMu.RLock()
		st := &ni.Styles
		ni.StyMu.RUnlock()
		if st.Col > 0 {
			col = st.Col
		}
		if st.Row > 0 {
			row = st.Row
		}

		{ // col, X dim
			dim := mat32.X
			gd := ly.GridData[Col][col]
			avail := gd.AllocSize
			al := st.AlignDim(dim)
			pref := ni.LayState.Size.Pref.Dim(dim)
			need := ni.LayState.Size.Need.Dim(dim)
			max := ni.LayState.Size.Max.Dim(dim)
			pos, size := LayoutSharedDimImpl(ly, avail, need, pref, max, styles.SideFloats{}, al)
			ni.Alloc.Size.Total.SetDim(dim, size)
			ni.Alloc.PosRel.SetDim(dim, pos+gd.AllocPosRel)

		}
		{ // row, Y dim
			dim := mat32.Y
			gd := ly.GridData[Row][row]
			avail := gd.AllocSize
			al := st.AlignDim(dim)
			pref := ni.LayState.Size.Pref.Dim(dim)
			need := ni.LayState.Size.Need.Dim(dim)
			max := ni.LayState.Size.Max.Dim(dim)
			pos, size := LayoutSharedDimImpl(ly, avail, need, pref, max, styles.SideFloats{}, al)
			ni.Alloc.Size.Total.SetDim(dim, size)
			ni.Alloc.PosRel.SetDim(dim, pos+gd.AllocPosRel)
		}

		if LayoutTrace {
			fmt.Printf("Layout: %v grid col: %v row: %v pos: %v size: %v\n", ly.Path(), col, row, ni.Alloc.PosRel, ni.Alloc.Size.Total)
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
	if ly.Lay == LayoutStacked && ly.Is(LayoutStackTopOnly) {
		sn, err := ly.ChildTry(ly.StackTop)
		if err != nil {
			return
		}
		ni := sn.(Widget).AsWidget()
		if ni == nil {
			return
		}
		ly.ChildSize.SetMax(ni.Alloc.PosRel.Add(ni.Alloc.Size.Total))
		ni.Alloc.Size.TotalOrig = ni.Alloc.Size.Total
		return
	}
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Widget).AsWidget()
		if ni == nil {
			continue
		}
		ly.ChildSize.SetMax(ni.Alloc.PosRel.Add(ni.Alloc.Size.Total))
		ni.Alloc.Size.TotalOrig = ni.Alloc.Size.Total
	}
}

*/
