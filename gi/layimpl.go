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

// Layout uses 3 Size passes, 2 Position passes:
//
// SizeUp: (bottom-up) gathers Actual sizes from our Children & Parts,
// based on Styles.Min / Max sizes and actual content sizing
// (e.g., text size).  Flexible elements (e.g., Label, Flex Wrap,
// TopAppBar) should reserve the _minimum_ size possible at this stage,
// and then Grow based on SizeDown allocation.

// SizeDown: (top-down, multiple iterations possible) provides top-down
// size allocations based initially on Scene available size and
// the SizeUp Actual sizes.  If there is extra space available, it is
// allocated according to the Grow factors.
// Flexible elements (e.g., Flex Wrap layouts and Label with word wrap)
// update their Actual size based on available Alloc size (re-wrap),
// to fit the allocated shape vs. the initial bottom-up guess.
// However, do NOT grow the Actual size to match Alloc at this stage,
// as Actual sizes must always represent the minimums (see Position).
// Returns true if any change in Actual size occurred.

// SizeFinal: (bottom-up) similar to SizeUp but done at the end of the
// Sizing phase: first grows widget Actual sizes based on their Grow
// factors, up to their Alloc sizes.  Then gathers this updated final
// actual Size information for layouts to register their actual sizes
// prior to positioning, which requires accurate Actual vs. Alloc
// sizes to perform correct alignment calculations.

// Position: uses the final sizes to set relative positions within layouts
// according to alignment settings, and Grow elements to their actual
// Alloc size per Styles settings and widget-specific behavior.

// ScenePos: computes scene-based absolute positions and final BBox
// bounding boxes for rendering, based on relative positions from
// Position step and parents accumulated position and scroll offset.
// This is the only step needed when scrolling (very fast).

// (Text) Wrapping key principles:
// * Using a heuristic initial box based on expected text area from length
//   of Text and aspect ratio based on styled size to get initial layout size.
//   This avoids extremes of all horizontal or all vertical initial layouts.
// * Use full Alloc for SizeDown to allocate for what has been reserved.
// * Set Actual to what you actually use (key: start with only styled
//   so you don't get hysterisis)
// * Layout always re-gets the actuals for accurate Actual sizing

// Scroll areas are similar: don't request anything more than Min reservation
// and then expand to Alloc in Final.

// Note that it is critical to not actually change any bottom-up Actual
// sizing based on the Alloc, during the SizeDown process, as this will
// introduce false constraints on the process: only work with minimum
// Actual "hard" constraints to make sure those are satisfied.  Text
// and Wrap elements resize only enough to fit within the Alloc space
// to the extent possible, but do not Grow.
//
// The separate SizeFinal step then finally allows elements to grow
// into their final Alloc space, once all the constraints are satisfied.
//
// This overall top-down / bottom-up logic is used in Flutter:
// https://docs.flutter.dev/resources/architectural-overview#rendering-and-layout
// Here's more links to other layout algorithms:
// https://stackoverflow.com/questions/53911631/gui-layout-algorithms-overview

// LayoutPasses is used for the SizeFromChildren method,
// which can potentially compute different sizes for different passes.
type LayoutPasses int32 //enums:enum

const (
	SizeUpPass LayoutPasses = iota
	SizeDownPass
	SizeFinalPass
)

///////////////////////////////////////////////////////////////////
// Layouter

// Layouter is the interface for layout functions, called by Layout
// widget type during the various Layout passes.
type Layouter interface {
	Widget

	// AsLayout returns the base Layout type
	AsLayout() *Layout

	// LayoutSpace sets our Space based on Styles, Scroll, and Gap Spacing.
	// Other layout types can change this if they want to.
	LayoutSpace()

	// SizeFromChildren gathers Actual size from kids into our Actual.Content size.
	// Different Layout types can alter this to present different Content
	// sizes for the layout process, e.g., if Content is sized to fit allocation,
	// as in the TopAppBar and Sliceview types.
	SizeFromChildren(sc *Scene, iter int, pass LayoutPasses) mat32.Vec2

	// SizeDownSetAllocs is the key SizeDown step that sets the allocations
	// in the children, based on our allocation.  In the default implementation
	// this calls SizeDownGrow if there is extra space to grow, or
	// SizeDownAllocActual to set the allocations as they currrently are.
	SizeDownSetAllocs(sc *Scene, iter int)

	// ManageOverflow uses overflow settings to determine if scrollbars
	// are needed, based on difference between ActualOverflow (full actual size)
	// and Alloc allocation.  Returns true if size changes as a result.
	ManageOverflow(sc *Scene, iter int) bool
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

//////////////////////////////////////////////////////////////
//  GeomSize

// GeomCT has core layout elements: Content and Total
type GeomCT struct {
	// Content is for the contents (children, parts) of the widget,
	// excluding all additional Space (e.g., gaps, scrollbars for Layouts)
	Content mat32.Vec2

	// Total is for the total exterior of the widget: Content + Space
	Total mat32.Vec2
}

func (ct GeomCT) String() string {
	return fmt.Sprintf("Content: %v, \tTotal: %v", ct.Content, ct.Total)
}

// GeomSize has Layout sizes for Actual and Alloc, and Space for the difference
// between Content and Total
type GeomSize struct {
	// Actual is the actual size required by the element based on its content.
	// This is initially the bottom-up constraint computed by SizeUp.
	// For flexible elements, it can change based on resizing from Alloc
	// during SizeDown.  For SizeFinal, widgets can grow to take up the
	// full final Alloc size -- their size prior to that is recorded
	// in FinalUp which is used for alignment positioning.
	Actual GeomCT `view:"inline"`

	// Alloc is the top-down allocated size, based on available visible space,
	// starting from the Scene geometry and working downward, attempting to
	// accommodate the Actual contents, and allocating extra space based on
	// Grow factors.  When Actual < Alloc, alignment factors determine positioning
	// within the allocated space.
	Alloc GeomCT

	// FinalUp is the final bottom-up Actual size, prior to the Grow step during
	// SizeFinal.  This is the true bottom-up content size, and is used for alignment
	// positioning within the full allocated layout size.
	FinalUp GeomCT

	// Space is total extra space that, when added to Content, results in the Total size.
	// This includes padding, total effective margin (border, shadow, etc), scrollbars, Gap
	// spacing -- everything that doesn't go into child content.
	Space mat32.Vec2

	// InnerSpace is total extra space that is included within the Content Size region
	// and must be included for Bounding Box and other such computations, but
	// excluded from child sizing computations.  Gap size is the main example in std Styling.
	InnerSpace mat32.Vec2

	// Max is the Styles.Max.Dots() (Ceil int) that constrains the Actual.Content size
	Max mat32.Vec2
}

func (ls GeomSize) String() string {
	return fmt.Sprintf("Actual: %v, \tAlloc: %v", ls.Actual, ls.Alloc)
}

// SetSizeMax sets given size to given fm value, subject to Max constraints
func (ls *GeomSize) SetSizeMax(to *mat32.Vec2, fm mat32.Vec2) {
	*to = fm
	to.SetMinPos(ls.Max)
}

// FitSizeMax increases given size to fit given fm value, subject to Max constraints
func (ls *GeomSize) FitSizeMax(to *mat32.Vec2, fm mat32.Vec2) {
	to.SetMaxPos(fm)
	to.SetMinPos(ls.Max)
}

// SetTotalFromContent sets the Total size as Content plus Space
func (ls *GeomSize) SetTotalFromContent(ct *GeomCT) {
	ct.Total = ct.Content.Add(ls.Space)
}

// SetContentFromTotal sets the Content from Total size,
// subtracting Space
func (ls *GeomSize) SetContentFromTotal(ct *GeomCT) {
	ct.Content = ct.Total.Sub(ls.Space)
}

//////////////////////////////////////////////////////////////
//  GeomState

// GeomState contains the the layout geometry state for each widget.
// Set by the parent Layout during the Layout process.
type GeomState struct {
	// Size has sizing data for the widget: use Actual for rendering.
	// Alloc shows the potentially larger space top-down allocated.
	Size GeomSize `view:"inline"`

	// Pos is position within the overall Scene that we render into,
	// including effects of scroll offset, for both Total outer dimension
	// and inner Content dimension.
	Pos GeomCT `view:"inline" edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// Cell is the logical X, Y index coordinates (row, col) of element
	// within its parent layout
	Cell image.Point

	// RelPos is top, left position relative to parent Content size space
	RelPos mat32.Vec2

	// Scroll is additional scrolling offset within our parent layout
	Scroll mat32.Vec2

	// 2D bounding box for Actual.Total size occupied within parent Scene
	// that we render onto, starting at Pos.Total and ending at Pos.Total + Size.Total.
	// These are the pixels we can draw into, intersected with parent bounding boxes
	// (empty for invisible). Used for render Bounds clipping.
	// This includes all space (margin, padding etc).
	TotalBBox image.Rectangle `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// 2D bounding box for our Content, which excludes our padding, margin, etc.
	// starting at Pos.Content and ending at Pos.Content + Size.Content.
	// It is intersected with parent bounding boxes.
	ContentBBox image.Rectangle `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`
}

func (ls *GeomState) String() string {
	return "Size: " + ls.Size.String() + "\nPos: " + ls.Pos.String() + "\tCell: " + ls.Cell.String() +
		"\tRelPos: " + ls.RelPos.String() + "\tScroll: " + ls.Scroll.String()
}

// ContentRangeDim returns the Content bounding box min, max
// along given dimension
func (ls *GeomState) ContentRangeDim(d mat32.Dims) (cmin, cmax float32) {
	cmin = float32(mat32.PointDim(ls.ContentBBox.Min, d))
	cmax = float32(mat32.PointDim(ls.ContentBBox.Max, d))
	return
}

// TotalRect returns Pos.Total, Min(Size.Actual.Total, Size.Alloc.Total)
// as an image.Rectangle, e.g., for bounding box
func (ls *GeomState) TotalRect() image.Rectangle {
	return mat32.RectFromPosSizeMax(ls.Pos.Total, ls.Size.Actual.Total.Min(ls.Size.Alloc.Total))
}

// ContentRect returns Pos.Content, Min(Size.Actual.Content, Size.Alloc.Content)
// with InnerSpace added to Size, as an image.Rectangle, e.g., for bounding box.
func (ls *GeomState) ContentRect() image.Rectangle {
	act := ls.Size.Actual.Content.Add(ls.Size.InnerSpace)
	alloc := ls.Size.Alloc.Content.Add(ls.Size.InnerSpace)
	return mat32.RectFromPosSizeMax(ls.Pos.Content, act.Min(alloc))
}

//////////////////////////////////////////////////////////////
//  LayImplState -- for Layout only

// LayCell holds the layout implementation data for col, row Cells
type LayCell struct {
	// Size has the Actual size of elements (not Alloc)
	Size mat32.Vec2

	// Grow has the Grow factors
	Grow mat32.Vec2
}

func (ls *LayCell) String() string {
	return fmt.Sprintf("Size: %v, \tGrow: %g", ls.Size, ls.Grow)
}

func (ls *LayCell) Reset() {
	ls.Size.SetZero()
	ls.Grow.SetZero()
}

// LayImplState has internal state for implementing layout
type LayImplState struct {
	// todo: Shape, Sizes needs to be an array of that substructure, for Wrap

	// Shape is number of cells along each dimension,
	// computed for each layout type.
	Shape image.Point `edit:"-"`

	// sizes has the data for the columns in [0] and rows in [1]:
	// col Size.X = max(X over rows) (cross axis), .Y = sum(Y over rows) (main axis for col)
	// row Size.X = sum(X over cols) (main axis for row), .Y = max(Y over cols) (cross axis)
	// see: https://docs.google.com/spreadsheets/d/1eimUOIJLyj60so94qUr4Buzruj2ulpG5o6QwG2nyxRw/edit?usp=sharing
	Sizes [2][]LayCell `edit:"-"`

	// ScrollSize has the scrollbar sizes (widths) for each dim, which adds to Space.
	// If there is a vertical scrollbar, X has width; if horizontal, Y has "width" = height
	ScrollSize mat32.Vec2

	// ActualOverflow is the actual size without absorbing the overflow in
	// scroll elements -- this is needed for grow calculation and overflow computation
	// to see if scrollbars are required.
	ActualOverflow GeomCT

	// GapSize has the extra gap sizing between elements, which adds to Space.
	// This depends on cell layout so it can vary for Wrap case.
	// For SizeUp / Down Gap contributes to Space like other cases,
	// but for BoundingBox rendering and Alignment, it does NOT, and must be
	// subtracted.  This happens in the Position phase.
	GapSize mat32.Vec2
}

// InitSizes initializes the Sizes based on Shape geom
func (ls *LayImplState) InitSizes() {
	for d := mat32.X; d <= mat32.Y; d++ {
		n := mat32.PointDim(ls.Shape, d)
		if len(ls.Sizes[d]) != n {
			ls.Sizes[d] = make([]LayCell, n)
		}
		for i := 0; i < n; i++ {
			ls.Sizes[d][i].Reset()
		}
	}
}

// CellsSize returns the total Size represented by the current Cells,
// which is the Sum of the Max values along each dimension.
func (ls *LayImplState) CellsSize() mat32.Vec2 {
	var ksz mat32.Vec2
	for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
		n := mat32.PointDim(ls.Shape, ma) // cols, rows
		sum := float32(0)
		for mi := 0; mi < n; mi++ {
			md := &ls.Sizes[ma][mi] // X, Y
			mx := md.Size.Dim(ma)
			sum += mx // sum of maxes
		}
		ksz.SetDim(ma, sum)
	}
	return ksz.Ceil()
}

// ColWidth returns the width of given column.  Returns false if doesn't exist.
func (ls *LayImplState) ColWidth(col int) (float32, bool) {
	n := mat32.PointDim(ls.Shape, mat32.X)
	if col >= n {
		return 0, false
	}
	return ls.Sizes[mat32.X][col].Size.X, true
}

// RowHeight returns the height of given row.  Returns false if doesn't exist.
func (ls *LayImplState) RowHeight(row int) (float32, bool) {
	n := mat32.PointDim(ls.Shape, mat32.Y)
	if row >= n {
		return 0, false
	}
	return ls.Sizes[mat32.Y][row].Size.Y, true
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
	n := ls.Shape.X
	for i := 0; i < n; i++ {
		col := ls.Sizes[mat32.X][i]
		s += fmt.Sprintln("col:", i, "\tmax X:", col.Size.X, "\tsum Y:", col.Size.Y, "\tmax grX:", col.Grow.X, "\tsum grY:", col.Grow.Y)
	}
	n = ls.Shape.Y
	for i := 0; i < n; i++ {
		row := ls.Sizes[mat32.Y][i]
		s += fmt.Sprintln("row:", i, "\tsum X:", row.Size.X, "\tmax Y:", row.Size.Y, "\tsum grX:", row.Grow.X, "\tmax grY:", row.Grow.Y)
	}
	return s
}

// SetContentFitOverflow sets Actual.Content size to fit given new actual size
// depending on the Styles Overflow: Auto and Scroll types do NOT expand and
// remain at their current styled actual values, absorbing the extra content size
// within their own scrolling zone.  Scene root (parent == nil) always
// updates because the buck stops there.
// Full overflow actual is maintained for overflow computation in ActualOverflow.
func (ly *Layout) SetContentFitOverflow(nsz mat32.Vec2) {
	// todo: potentially the diff between Visible & Hidden is
	// that Hidden also does Not expand beyond Alloc?
	// can expt with that.
	sz := &ly.Geom.Size
	asz := &sz.Actual
	osz := &ly.LayImpl.ActualOverflow
	ly.Geom.Size.SetSizeMax(&asz.Content, ly.Styles.Min.Dots().Ceil()) // start with style
	*osz = *asz
	mx := ly.Geom.Size.Max
	oflow := &ly.Styles.Overflow
	for d := mat32.X; d <= mat32.Y; d++ {
		osz.Content.SetDim(d, mat32.MaxPos(osz.Content.Dim(d), nsz.Dim(d)))
		if oflow.Dim(d) < styles.OverflowAuto || ly.Par == nil {
			asz.Content.SetDim(d, mat32.MaxPos(asz.Content.Dim(d), nsz.Dim(d)))
		}
	}
	osz.Content.SetMinPos(mx)
	asz.Content.SetMinPos(mx)
	sz.SetTotalFromContent(asz)
	sz.SetTotalFromContent(osz)
}

//////////////////////////////////////////////////////////////////////
//		SizeUp

// SizeUp (bottom-up) gathers Actual sizes from our Children & Parts,
// based on Styles.Min / Max sizes and actual content sizing
// (e.g., text size).  Flexible elements (e.g., Label, Flex Wrap,
// TopAppBar) should reserve the _minimum_ size possible at this stage,
// and then Grow based on SizeDown allocation.
func (wb *WidgetBase) SizeUp(sc *Scene) {
	wb.SizeUpWidget(sc)
}

// SizeUpWidget is the standard Widget SizeUp pass
func (wb *WidgetBase) SizeUpWidget(sc *Scene) {
	wb.SizeFromStyle()
	wb.SizeUpParts(sc)
	sz := &wb.Geom.Size
	sz.SetTotalFromContent(&sz.Actual)
}

// SpaceFromStyle sets the Space based on Style BoxSpace().Size()
func (wb *WidgetBase) SpaceFromStyle() {
	wb.Geom.Size.Space = wb.Styles.BoxSpace().Size().Ceil()
}

// SizeFromStyle sets the initial Actual Sizes from Style.Min, Max.
// Required first call in SizeUp.
func (wb *WidgetBase) SizeFromStyle() {
	sz := &wb.Geom.Size
	wb.SpaceFromStyle()
	wb.Geom.Size.InnerSpace.SetZero()
	sz.Max = wb.Styles.Max.Dots().Ceil()
	sz.SetSizeMax(&sz.Actual.Content, wb.Styles.Min.Dots().Ceil())
	sz.SetTotalFromContent(&sz.Actual)
	if LayoutTrace && (sz.Actual.Content.X > 0 || sz.Actual.Content.Y > 0) {
		fmt.Println(wb, "SizeUp from Style:", sz.Actual.Content.String())
	}
}

// SizeUpParts adjusts the Content size to hold the Parts layout if present
func (wb *WidgetBase) SizeUpParts(sc *Scene) {
	if wb.Parts == nil {
		return
	}
	wb.Parts.SizeUp(sc)
	sz := &wb.Geom.Size
	sz.FitSizeMax(&sz.Actual.Content, wb.Parts.Geom.Size.Actual.Total)
}

/////////////// Layout

func (ly *Layout) SizeUp(sc *Scene) {
	ly.SizeUpLay(sc)
}

// SizeUpLay is the Layout standard SizeUp pass
func (ly *Layout) SizeUpLay(sc *Scene) {
	if !ly.HasChildren() {
		ly.SizeUpWidget(sc) // behave like a widget
		return
	}
	ly.SizeFromStyle()
	ly.LayImpl.ScrollSize.SetZero() // we don't know yet
	ly.SetInitCells()
	ly.This().(Layouter).LayoutSpace() // now includes initial GapSize from Cells
	ly.SizeUpChildren(sc)              // kids do their own thing
	ksz := ly.This().(Layouter).SizeFromChildren(sc, 0, SizeUpPass)
	sz := &ly.Geom.Size
	ly.SetContentFitOverflow(ksz)
	if LayoutTrace {
		fmt.Println(ly, "SizeUp FromChildren:", ksz, "Content:", sz.Actual.Content, "Overflow:", ly.LayImpl.ActualOverflow)
	}
	if ly.Parts != nil {
		ly.Parts.SizeUp(sc) // just to get sizes -- no std role in layout
	}
}

// LayoutSpace sets our Space based on Styles, Scroll, and Gap Spacing.
// Other layout types can change this if they want to.
func (ly *Layout) LayoutSpace() {
	ly.SpaceFromStyle()
	ly.Geom.Size.Space.SetAdd(ly.LayImpl.ScrollSize.Add(ly.LayImpl.GapSize))
}

// SizeUpChildren calls SizeUp on all the children of this node
func (wb *WidgetBase) SizeUpChildren(sc *Scene) {
	wb.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
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
		// todo: if wrap..
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
	ly.LayImpl.GapSize.X = max(float32(ly.LayImpl.Shape.X-1)*mat32.Ceil(ly.Styles.Gap.X.Dots), 0)
	ly.LayImpl.GapSize.Y = max(float32(ly.LayImpl.Shape.Y-1)*mat32.Ceil(ly.Styles.Gap.Y.Dots), 0)
	ly.LayImpl.GapSize.SetCeil()
	ly.Geom.Size.InnerSpace = ly.LayImpl.GapSize
}

func (ly *Layout) SetInitCellsFlex() {
	ma := ly.Styles.MainAxis
	ca := ma.Other()
	idx := 0
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		mat32.SetPointDim(&kwb.Geom.Cell, ma, idx)
		idx++
		return ki.Continue
	})
	if idx == 0 {
		fmt.Println(ly, "no items:", idx)
	}
	mat32.SetPointDim(&ly.LayImpl.Shape, ma, max(idx, 1)) // must be at least 1
	mat32.SetPointDim(&ly.LayImpl.Shape, ca, 1)
	ly.SetGapSizeFromCells()
}

func (ly *Layout) SetInitCellsStacked() {
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwb.Geom.Cell = image.Point{0, 0}
		return ki.Continue
	})
	ly.LayImpl.Shape = image.Point{1, 1}
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
	ly.LayImpl.Shape = image.Point{max(cols, 1), max(rows, 1)}
	ci := 0
	ri := 0
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwb.Geom.Cell = image.Point{ci, ri}
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
//		SizeFromChildren

// SizeFromChildren gathers Actual size from kids into our Actual.Content size.
// Different Layout types can alter this to present different Content
// sizes for the layout process, e.g., if Content is sized to fit allocation,
// as in the TopAppBar and Sliceview types.
func (ly *Layout) SizeFromChildren(sc *Scene, iter int, pass LayoutPasses) mat32.Vec2 {
	// todo: flex
	if ly.Styles.Display == styles.DisplayStacked {
		return ly.SizeFromChildrenStacked(sc)
	}
	return ly.SizeFromChildrenCells(sc)
}

// SizeFromChildrenCells for Flex, Grid
func (ly *Layout) SizeFromChildrenCells(sc *Scene) mat32.Vec2 {
	// r   0   1   col X = max(X over rows), Y = sum(Y over rows)
	//   +--+--+
	// 0 |  |  |   row X = sum(X over cols), Y = max(Y over cols)
	//   +--+--+
	// 1 |  |  |
	//   +--+--+
	ly.LayImpl.InitSizes()
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Geom.Cell
		sz := kwb.Geom.Size.Actual.Total
		grw := kwb.Styles.Grow
		if LayoutTraceDetail {
			fmt.Println("SzUp i:", i, kwb, "cidx:", cidx, "sz:", sz, "grw:", grw)
		}
		for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
			ca := ma.Other()                // cross axis = Y then X
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
		fmt.Println(ly, "SizeFromChildren")
		fmt.Println(ly.LayImpl.String())
	}
	ksz := ly.LayImpl.CellsSize()
	return ksz
}

// SizeFromChildrenStacked for stacked case
func (ly *Layout) SizeFromChildrenStacked(sc *Scene) mat32.Vec2 {
	ly.LayImpl.InitSizes()
	_, kwb := ly.StackTopWidget()
	li := &ly.LayImpl
	var ksz mat32.Vec2
	if kwb != nil {
		ksz = kwb.Geom.Size.Actual.Total
		kgrw := kwb.Styles.Grow
		for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
			li.Sizes[ma][0].Size = ksz
			li.Sizes[ma][0].Grow = kgrw
		}
	}
	return ksz
}

//////////////////////////////////////////////////////////////////////
//		SizeDown

// SizeDown (top-down, multiple iterations possible) provides top-down
// size allocations based initially on Scene available size and
// the SizeUp Actual sizes.  If there is extra space available, it is
// allocated according to the Grow factors.
// Flexible elements (e.g., Flex Wrap layouts and Label with word wrap)
// update their Actual size based on available Alloc size (re-wrap),
// to fit the allocated shape vs. the initial bottom-up guess.
// However, do NOT grow the Actual size to match Alloc at this stage,
// as Actual sizes must always represent the minimums (see Position).
// Returns true if any change in Actual size occurred.
func (wb *WidgetBase) SizeDown(sc *Scene, iter int) bool {
	return wb.SizeDownWidget(sc, iter)
}

// SizeDownWidget is the standard widget implementation of SizeDown.
// It just delegates to the Parts.
func (wb *WidgetBase) SizeDownWidget(sc *Scene, iter int) bool {
	return wb.SizeDownParts(sc, iter)
}

func (wb *WidgetBase) SizeDownParts(sc *Scene, iter int) bool {
	if wb.Parts == nil {
		return false
	}
	sz := &wb.Geom.Size
	psz := &wb.Parts.Geom.Size
	pgrow, _ := wb.GrowToAllocSize(sc, sz.Actual.Content, sz.Alloc.Content)
	psz.Alloc.Total = pgrow // parts = content
	psz.SetContentFromTotal(&psz.Alloc)
	redo := wb.Parts.SizeDown(sc, iter)
	if redo && LayoutTrace {
		fmt.Println(wb, "Parts triggered redo")
	}
	return redo
}

// SizeDownChildren calls SizeDown on the Children.
// The kids must have their Size.Alloc set prior to this, which
// is what Layout type does.  Other special widget types can
// do custom layout and call this too.
func (wb *WidgetBase) SizeDownChildren(sc *Scene, iter int) bool {
	redo := false
	wb.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		re := kwi.SizeDown(sc, iter)
		if re && LayoutTrace {
			fmt.Println(wb, "SizeDownChildren child:", kwb.Nm, "triggered redo")
		}
		redo = redo || re
		return ki.Continue
	})
	return redo
}

// GrowToAllocSize returns the potential size that widget could grow,
// for any dimension with a non-zero Grow factor.
// If Grow is < 1, then the size is increased in proportion, but
// any factor > 1 produces a full fill along that dimension.
// Returns true if this resulted in a change.
func (wb *WidgetBase) GrowToAllocSize(sc *Scene, act, alloc mat32.Vec2) (mat32.Vec2, bool) {
	if sc.Is(ScPrefSizing) {
		return act, false
	}
	change := false
	for d := mat32.X; d <= mat32.Y; d++ {
		grw := wb.Styles.Grow.Dim(d)
		allocd := alloc.Dim(d)
		actd := act.Dim(d)
		if grw > 0 && allocd > actd {
			grw := min(1, grw)
			nsz := mat32.Ceil(actd + grw*(allocd-actd))
			nsz = mat32.MinPos(nsz, wb.Geom.Size.Max.Dim(d))
			if nsz != actd {
				change = true
			}
			act.SetDim(d, nsz)
		}
	}
	return act, change
}

/////////////// Layout

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
		return ly.SizeDownWidget(sc, iter) // behave like a widget
	}
	ly.This().(Layouter).SizeDownSetAllocs(sc, iter)
	redo := ly.SizeDownChildren(sc, iter)
	if redo {
		ksz := ly.This().(Layouter).SizeFromChildren(sc, iter, SizeDownPass)
		ly.SetContentFitOverflow(ksz)
		if LayoutTrace {
			fmt.Println(ly, "SizeDown FromChildren:", ksz, "Content:", ly.Geom.Size.Actual.Content)
		}
	}
	chg := ly.This().(Layouter).ManageOverflow(sc, iter)
	ly.SizeDownParts(sc, iter) // no std role, just get sizes
	return chg || redo
}

// SizeDownSetAllocs is the key SizeDown step that sets the allocations
// in the children, based on our allocation.  In the default implementation
// this calls SizeDownGrow if there is extra space to grow, or
// SizeDownAllocActual to set the allocations as they currrently are.
func (ly *Layout) SizeDownSetAllocs(sc *Scene, iter int) {
	sz := &ly.Geom.Size
	extra := sz.Alloc.Content.Sub(ly.LayImpl.ActualOverflow.Content) // note: critical to use oflow
	if extra.X > 0 || extra.Y > 0 {
		if LayoutTrace {
			fmt.Println(ly, "SizeDown extra:", extra, "actual:", sz.Actual.Content, "alloc:", sz.Alloc.Content)
		}
		ly.SizeDownGrow(sc, iter, extra)
	} else {
		ly.SizeDownAllocActual(sc, iter) // set allocations as is
	}
}

// ManageOverflow uses overflow settings to determine if scrollbars
// are needed, etc.  Returns true if size changes as a result.
func (ly *Layout) ManageOverflow(sc *Scene, iter int) bool {
	sz := &ly.Geom.Size
	oflow := ly.LayImpl.ActualOverflow.Content.Sub(sz.Alloc.Content)
	change := false
	ly.LayImpl.ScrollSize.SetZero()
	ly.SetScrollsOff()
	for d := mat32.X; d <= mat32.Y; d++ {
		if ly.Styles.Overflow.Dim(d) == styles.OverflowScroll {
			if !ly.HasScroll[d] {
				change = true
			}
			ly.HasScroll[d] = true
			ly.LayImpl.ScrollSize.SetDim(d.Other(), mat32.Ceil(ly.Styles.ScrollBarWidth.Dots))
		}
	}
	for d := mat32.X; d <= mat32.Y; d++ {
		ofd := oflow.Dim(d)
		if ofd <= 1 {
			continue
		}
		switch ly.Styles.Overflow.Dim(d) {
		// case styles.OverflowVisible:
		// note: this shouldn't happen -- just have this in here for monitoring
		// fmt.Println(ly, "OverflowVisible ERROR -- shouldn't have overflow:", d, ofd)
		case styles.OverflowAuto:
			if !ly.HasScroll[d] {
				change = true
			}
			ly.HasScroll[d] = true
			ly.LayImpl.ScrollSize.SetDim(d.Other(), mat32.Ceil(ly.Styles.ScrollBarWidth.Dots))
			if LayoutTrace {
				fmt.Println(ly, "OverflowAuto enabling scrollbars for dim for overflow:", d, ofd)
			}
		}
	}
	ly.This().(Layouter).LayoutSpace() // adds the scroll space
	sz.SetTotalFromContent(&sz.Actual)
	sz.SetContentFromTotal(&sz.Alloc) // alloc is *decreased* from any increase in space
	if change && LayoutTrace {
		fmt.Println(ly, "ManageOverflow changed")
	}
	return change
}

// SizeDownGrow grows the element sizes based on total extra and Grow
func (ly *Layout) SizeDownGrow(sc *Scene, iter int, extra mat32.Vec2) bool {
	redo := false
	// if ly.Styles.Display == styles.DisplayFlex && ly.Styles.Wrap {
	// 	redo = ly.SizeDownGrowWrap(sc, iter, extra) // first recompute wrap
	// todo: use special version of grow
	// } else
	if ly.Styles.Display == styles.DisplayStacked {
		redo = ly.SizeDownGrowStacked(sc, iter, extra)
	} else {
		redo = ly.SizeDownGrowCells(sc, iter, extra)
	}
	return redo
}

func (ly *Layout) SizeDownGrowCells(sc *Scene, iter int, extra mat32.Vec2) bool {
	redo := false
	sz := &ly.Geom.Size
	alloc := sz.Alloc.Content
	// todo: use max growth values instead of individual ones to ensure consistency!
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Geom.Cell
		ksz := &kwb.Geom.Size
		grw := kwb.Styles.Grow
		// if LayoutTrace {
		// 	fmt.Println("szdn i:", i, kwb, "cidx:", cidx, "sz:", sz, "grw:", grw)
		// }
		for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
			gr := grw.Dim(ma)
			ca := ma.Other()     // cross axis = Y then X
			exd := extra.Dim(ma) // row.X = extra width for cols; col.Y = extra height for rows in this col
			if exd < 0 {
				exd = 0
			}
			mi := mat32.PointDim(cidx, ma)  // X, Y
			ci := mat32.PointDim(cidx, ca)  // Y, X
			md := &ly.LayImpl.Sizes[ma][mi] // X, Y
			cd := &ly.LayImpl.Sizes[ca][ci] // Y, X
			mx := md.Size.Dim(ma)
			asz := mx
			gsum := cd.Grow.Dim(ma)
			if gsum > 0 && exd > 0 {
				if gr > gsum {
					fmt.Println(ly, "SizeDownGrowCells error: grow > grow sum:", gr, gsum)
					gr = gsum
				}
				redo = true
				asz = mat32.Round(mx + exd*(gr/gsum))  // todo: could use Floor
				if asz > mat32.Ceil(alloc.Dim(ma))+1 { // bug!
					fmt.Println(ly, "SizeDownGrowCells error: sub alloc > total to alloc:", asz, alloc.Dim(ma))
					fmt.Println("ma:", ma, "mi:", mi, "ci:", ci, "mx:", mx, "gsum:", gsum, "gr:", gr, "ex:", exd, "act:", sz.Actual.Content.Dim(ma))
					fmt.Println(ly.LayImpl.String())
					fmt.Println(ly.LayImpl.CellsSize())
				}
			}
			ksz.Alloc.Total.SetDim(ma, asz)
		}
		ksz.SetContentFromTotal(&ksz.Alloc)
		return ki.Continue
	})
	return redo
}

func (ly *Layout) SizeDownGrowWrap(sc *Scene, iter int, extra mat32.Vec2) bool {
	// todo
	return false
}

func (ly *Layout) SizeDownGrowStacked(sc *Scene, iter int, extra mat32.Vec2) bool {
	// stack just gets everything from us
	chg := false
	asz := ly.Geom.Size.Alloc.Content
	// todo: could actually use the grow factors to decide if growing here?
	if ly.Is(LayoutStackTopOnly) {
		_, kwb := ly.StackTopWidget()
		if kwb != nil {
			ksz := &kwb.Geom.Size
			if ksz.Alloc.Total != asz {
				chg = true
			}
			ksz.Alloc.Total = asz
			ksz.SetContentFromTotal(&ksz.Alloc)
		}
		return chg
	}
	// note: allocate everyone in case they are flipped to top
	// need a new layout if size is actually different
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		ksz := &kwb.Geom.Size
		if ksz.Alloc.Total != asz {
			chg = true
		}
		ksz.Alloc.Total = asz
		ksz.SetContentFromTotal(&ksz.Alloc)
		return ki.Continue
	})
	return chg
}

// SizeDownAllocActual sets Alloc to Actual for no-extra case.
func (ly *Layout) SizeDownAllocActual(sc *Scene, iter int) {
	if ly.Styles.Display == styles.DisplayStacked {
		ly.SizeDownAllocActualStacked(sc, iter)
		return
	}
	// todo: wrap needs special case
	ly.SizeDownAllocActualCells(sc, iter)
}

// SizeDownAllocActualCells sets Alloc to Actual for no-extra case.
// Note however that due to max sizing for row / column,
// this size can actually be different than original actual.
func (ly *Layout) SizeDownAllocActualCells(sc *Scene, iter int) {
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		ksz := &kwb.Geom.Size
		cidx := kwb.Geom.Cell
		for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
			mi := mat32.PointDim(cidx, ma)  // X, Y
			md := &ly.LayImpl.Sizes[ma][mi] // X, Y
			asz := md.Size.Dim(ma)
			ksz.Alloc.Total.SetDim(ma, asz)
		}
		ksz.SetContentFromTotal(&ksz.Alloc)
		return ki.Continue
	})
}

func (ly *Layout) SizeDownAllocActualStacked(sc *Scene, iter int) {
	// stack just gets everything from us
	asz := ly.Geom.Size.Actual.Content
	// todo: could actually use the grow factors to decide if growing here?
	if ly.Is(LayoutStackTopOnly) {
		_, kwb := ly.StackTopWidget()
		if kwb != nil {
			ksz := &kwb.Geom.Size
			ksz.Alloc.Total = asz
			ksz.SetContentFromTotal(&ksz.Alloc)
		}
		return
	}
	// note: allocate everyone in case they are flipped to top
	// need a new layout if size is actually different
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		ksz := &kwb.Geom.Size
		ksz.Alloc.Total = asz
		ksz.SetContentFromTotal(&ksz.Alloc)
		return ki.Continue
	})
}

//////////////////////////////////////////////////////////////////////
//		SizeFinal

// SizeFinalUpdateChildrenSizes can optionally be called for layouts
// that dynamically create child elements based on final layout size.
// It ensures that the children are properly sized.
func (ly *Layout) SizeFinalUpdateChildrenSizes(sc *Scene) {
	ly.SizeUpLay(sc)
	iter := 3 // late stage..
	ly.This().(Layouter).SizeDownSetAllocs(sc, iter)
	ly.SizeDownChildren(sc, iter)
	ly.SizeDownParts(sc, iter) // no std role, just get sizes
}

// SizeFinal: (bottom-up) similar to SizeUp but done at the end of the
// Sizing phase: first grows widget Actual sizes based on their Grow
// factors, up to their Alloc sizes.  Then gathers this updated final
// actual Size information for layouts to register their actual sizes
// prior to positioning, which requires accurate Actual vs. Alloc
// sizes to perform correct alignment calculations.
func (wb *WidgetBase) SizeFinal(sc *Scene) {
	wb.SizeFinalWidget(sc)
}

// SizeFinalWidget is the standard Widget SizeFinal pass
func (wb *WidgetBase) SizeFinalWidget(sc *Scene) {
	sz := &wb.Geom.Size
	sz.FinalUp = sz.Actual // keep it before we grow
	wb.GrowToAlloc(sc)
	wb.StyleSizeUpdate(sc) // now that sizes are stable, ensure styling based on size is updated
	wb.SizeFinalParts(sc)
	sz.SetTotalFromContent(&sz.Actual)
}

// GrowToAlloc grows our Actual size up to current Alloc size
// for any dimension with a non-zero Grow factor.
// If Grow is < 1, then the size is increased in proportion, but
// any factor > 1 produces a full fill along that dimension.
// Returns true if this resulted in a change in our Total size.
func (wb *WidgetBase) GrowToAlloc(sc *Scene) bool {
	if sc.Is(ScPrefSizing) {
		return false
	}
	sz := &wb.Geom.Size
	act, change := wb.GrowToAllocSize(sc, sz.Actual.Total, sz.Alloc.Total)
	if change {
		sz.Actual.Total = act // already has max constraint
		sz.SetContentFromTotal(&sz.Actual)
		if wb.Styles.LayoutHasParSizing() {
			// todo: requires some additional logic to see if actually changes something
		}
		if LayoutTrace {
			fmt.Println(wb, "GrowToAlloc:", sz.Alloc.Total, "actual:", sz.Actual.Total)
		}
	}
	return change
}

// SizeFinalParts adjusts the Content size to hold the Parts Final sizes
func (wb *WidgetBase) SizeFinalParts(sc *Scene) {
	if wb.Parts == nil {
		return
	}
	wb.Parts.SizeFinal(sc)
	sz := &wb.Geom.Size
	sz.FitSizeMax(&sz.Actual.Content, wb.Parts.Geom.Size.Actual.Total)
}

/////////////// Layout

func (ly *Layout) SizeFinal(sc *Scene) {
	ly.SizeFinalLay(sc)
}

// SizeFinalLay is the Layout standard SizeFinal pass
func (ly *Layout) SizeFinalLay(sc *Scene) {
	if !ly.HasChildren() {
		ly.SizeFinalWidget(sc) // behave like a widget
		return
	}
	ly.SizeFinalChildren(sc) // kids do their own thing
	ksz := ly.This().(Layouter).SizeFromChildren(sc, 0, SizeFinalPass)
	sz := &ly.Geom.Size
	sz.FitSizeMax(&sz.Actual.Content, ksz) // note: get full amount here, regardless of overflow
	sz.SetTotalFromContent(&sz.Actual)
	sz.FinalUp = sz.Actual // keep it before we grow
	ly.GrowToAlloc(sc)
	ly.StyleSizeUpdate(sc) // now that sizes are stable, ensure styling based on size is updated
}

// SizeFinalChildren calls SizeFinal on all the children of this node
func (wb *WidgetBase) SizeFinalChildren(sc *Scene) {
	wb.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.SizeFinal(sc)
		return ki.Continue
	})
	return
}

// StyleSizeUpdate updates styling size values for widget and its parent,
// which should be called after these are updated.  Returns true if any changed.
func (wb *WidgetBase) StyleSizeUpdate(sc *Scene) bool {
	el := wb.Geom.Size.Actual.Content
	var par mat32.Vec2
	_, pwb := wb.ParentWidget()
	if pwb != nil {
		par = pwb.Geom.Size.Actual.Content
	}
	sz := sc.SceneGeom.Size
	chg := wb.Styles.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, par.X, par.Y)
	if chg {
		wb.Styles.ToDots()
	}
	return chg
}

//////////////////////////////////////////////////////////////////////
//		Position

// Position uses the final sizes to set relative positions within layouts
// according to alignment settings.
func (wb *WidgetBase) Position(sc *Scene) {
	wb.PositionWidget(sc)
}

func (wb *WidgetBase) PositionWidget(sc *Scene) {
	wb.PositionParts(sc)
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
		ly.PositionWidget(sc) // behave like a widget
		return
	}
	ly.ConfigScrolls(sc) // and configure the scrolls
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
	gap := ly.Styles.Gap.Dots().Floor()
	sz := &ly.Geom.Size
	csz := sz.Actual.Content.Add(sz.InnerSpace)

	if LayoutTraceDetail {
		fmt.Println(ly, "PositionCells, actual + inner:", csz, "alloc:", sz.Alloc.Content.Add(sz.InnerSpace), "finalup:", sz.FinalUp.Content.Add(sz.InnerSpace))
	}

	var stspc mat32.Vec2
	cdiff := sz.Actual.Content.Sub(sz.FinalUp.Content).Floor()
	if cdiff.X > 0 {
		stspc.X += mat32.Floor(styles.AlignFactor(ly.Styles.Align.X) * cdiff.X)
		if LayoutTrace {
			fmt.Println("pos grid:", ly, "extra X:", cdiff.X, "start X:", stspc.X, "align:", ly.Styles.Align.X, "factor:", styles.AlignFactor(ly.Styles.Align.X))
		}
	}
	if cdiff.Y > 0 {
		stspc.Y += mat32.Floor(styles.AlignFactor(ly.Styles.Align.Y) * cdiff.Y)
		if LayoutTrace {
			fmt.Println("pos grid:", ly, "extra Y:", cdiff.Y, "start Y:", stspc.Y, "align:", ly.Styles.Align.Y, "factor:", styles.AlignFactor(ly.Styles.Align.Y))
		}
	}
	pos = stspc
	var maxs mat32.Vec2
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Geom.Cell
		ksz := &kwb.Geom.Size
		sz := ksz.Actual.Total
		asz := ksz.Alloc.Total
		if cidx.X == 0 && i > 0 {
			pos.X = stspc.X
			pos.Y += lastAsz.Y + gap.Y
		}
		ep := pos
		if sz.X < asz.X {
			ex := styles.AlignFactor(kwb.Styles.Align.X) * (asz.X - sz.X)
			ep.X += ex
		}
		if sz.Y < asz.Y {
			ep.Y += styles.AlignFactor(kwb.Styles.Align.Y) * (asz.Y - sz.Y)
		}
		ep.SetFloor()
		endsz := ep.Add(asz)
		if LayoutTraceDetail {
			fmt.Println("pos i:", i, kwb, "cidx:", cidx, "sz:", sz, "asz:", asz, "pos:", ep, "end:", endsz)
		}
		kwb.Geom.RelPos = ep
		maxs.SetMax(endsz)
		pos.X += asz.X + gap.X
		lastAsz = asz
		return ki.Continue
	})
	if LayoutTrace {
		if maxs.X > csz.X+1 || maxs.Y > csz.Y+2 {
			fmt.Println(ly, "Layout Position error: max position exceeds actual content size:", maxs, "content:", sz.Actual.Content)
		}
		fmt.Println(ly, "Position max:", maxs)
	}
}

func (ly *Layout) PositionWrap(sc *Scene) {
}

func (ly *Layout) PositionStacked(sc *Scene) {
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwb.Geom.RelPos.SetZero()
		return ki.Continue
	})
}

//////////////////////////////////////////////////////////////////////
//		ScenePos

// ScenePos computes scene-based absolute positions and final BBox
// bounding boxes for rendering, based on relative positions from
// Position step and parents accumulated position and scroll offset.
// This is the only step needed when scrolling (very fast).
func (wb *WidgetBase) ScenePos(sc *Scene) {
	wb.ScenePosWidget(sc)
}

func (wb *WidgetBase) ScenePosWidget(sc *Scene) {
	wb.SetPosFromParent(sc)
	wb.SetBBoxes(sc)
}

// SetContentPosFromPos sets the Pos.Content position based on current Pos
// plus the BoxSpace position offset.
func (wb *WidgetBase) SetContentPosFromPos() {
	spc := wb.Styles.BoxSpace()
	off := spc.Pos()
	off.SetFloor()
	wb.Geom.Pos.Content = wb.Geom.Pos.Total.Add(off)
}

func (wb *WidgetBase) SetPosFromParent(sc *Scene) {
	_, pwb := wb.ParentWidget()
	var parPos mat32.Vec2
	if pwb != nil {
		parPos = pwb.Geom.Pos.Content.Add(pwb.Geom.Scroll) // critical that parent adds here but not to self
	}
	wb.Geom.Pos.Total = wb.Geom.RelPos.Add(parPos)
	wb.SetContentPosFromPos()
	if LayoutTrace {
		fmt.Println(wb, "pos:", wb.Geom.Pos.Total, "parPos:", parPos)
	}
}

// SetBBoxesFromAllocs sets BBox and ContentBBox from Geom.Pos and .Size
// This does NOT intersect with parent content BBox, which is done in SetBBoxes.
// Use this for elements that are dynamically positioned outside of parent BBox.
func (wb *WidgetBase) SetBBoxesFromAllocs() {
	wb.Geom.TotalBBox = wb.Geom.TotalRect()
	wb.Geom.ContentBBox = wb.Geom.ContentRect()
}

func (wb *WidgetBase) SetBBoxes(sc *Scene) {
	_, pwb := wb.ParentWidget()
	var parBB image.Rectangle
	if pwb == nil { // scene
		sz := &wb.Geom.Size
		wb.Geom.TotalBBox = mat32.RectFromPosSizeMax(mat32.Vec2{}, sz.Alloc.Total)
		csz := sz.Alloc.Total.Sub(sz.Space).Add(sz.InnerSpace)
		wb.Geom.ContentBBox = mat32.RectFromPosSizeMax(wb.Geom.Pos.Content, csz)
		if LayoutTrace {
			fmt.Println(wb, "Total BBox:", wb.Geom.TotalBBox)
			fmt.Println(wb, "Content BBox:", wb.Geom.ContentBBox)
		}
	} else {
		parBB = pwb.Geom.ContentBBox
		bb := wb.Geom.TotalRect()
		wb.Geom.TotalBBox = parBB.Intersect(bb)
		if LayoutTrace {
			fmt.Println(wb, "Total BBox:", bb, "parBB:", parBB, "BBox:", wb.Geom.TotalBBox)
		}

		cbb := wb.Geom.ContentRect()
		wb.Geom.ContentBBox = parBB.Intersect(cbb)
		if LayoutTrace {
			fmt.Println(wb, "Content BBox:", cbb, "parBB:", parBB, "BBox:", wb.Geom.ContentBBox)
		}
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
	if !ly.HasChildren() {
		ly.ScenePosWidget(sc) // behave like a widget
		return
	}
	ly.GetScrollPosition(sc)
	ly.ScenePosWidget(sc)
	ly.ScenePosChildren(sc)
	ly.PositionScrolls(sc)
	ly.ScenePosParts(sc) // in case they fit inside parent
	// otherwise handle separately like scrolls on layout
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
		ly.Geom.Size.Total = prv
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

	avail := ly.Geom.Size.Total.Dim(dim) - exspc
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
		ni.Geom.Size.Total.SetDim(dim, size)
		ni.Geom.PosRel.SetDim(dim, pos)
		if LayoutTrace {
			fmt.Printf("Layout: %v Child: %v, pos: %v, size: %v, need: %v, pref: %v\n", ly.Path(), ni.Nm, pos, size, ni.LayState.Size.Need.Dim(dim), ni.LayState.Size.Pref.Dim(dim))
		}
		pos += size + ly.Styles.Gap.Dots
	}
	ly.FlowBreaks = append(ly.FlowBreaks, len(ly.Kids))

	nrows := len(ly.FlowBreaks)
	oavail := ly.Geom.Size.Total.Dim(odim) - exspc
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
			ni.Geom.Size.Total.SetDim(odim, size)
			ni.Geom.PosRel.SetDim(odim, rpos+pos)
			rmax = mat32.Max(rmax, size)
			nsz.X = mat32.Max(nsz.X, ni.Geom.PosRel.X+ni.Geom.Size.Total.X)
			nsz.Y = mat32.Max(nsz.Y, ni.Geom.PosRel.Y+ni.Geom.Size.Total.Y)
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
