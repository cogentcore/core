// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
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
	SizeFromChildren(iter int, pass LayoutPasses) mat32.Vec2

	// SizeDownSetAllocs is the key SizeDown step that sets the allocations
	// in the children, based on our allocation.  In the default implementation
	// this calls SizeDownGrow if there is extra space to grow, or
	// SizeDownAllocActual to set the allocations as they currrently are.
	SizeDownSetAllocs(iter int)

	// ManageOverflow uses overflow settings to determine if scrollbars
	// are needed, based on difference between ActualOverflow (full actual size)
	// and Alloc allocation.  Returns true if size changes as a result.
	// If updtSize is false, then the Actual and Alloc sizes are NOT
	// updated as a result of change from adding scrollbars
	// (generally should be true, but some cases not)
	ManageOverflow(iter int, updtSize bool) bool

	// ScrollChanged is called in the OnInput event handler for updating
	// when the scrollbar value has changed, for given dimension
	ScrollChanged(d mat32.Dims, sb *Slider)

	// ScrollValues returns the maximum size that could be scrolled,
	// the visible size (which could be less than the max size, in which
	// case no scrollbar is needed), and visSize / maxSize as the VisiblePct.
	// This is used in updating the scrollbar and determining whether one is
	// needed in the first place
	ScrollValues(d mat32.Dims) (maxSize, visSize, visPct float32)

	// ScrollGeom returns the target position and size for scrollbars
	ScrollGeom(d mat32.Dims) (pos, sz mat32.Vec2)

	// SetScrollParams sets scrollbar parameters.  Must set Step and PageStep,
	// but can also set others as needed.
	// Max and VisiblePct are automatically set based on ScrollValues maxSize, visPct.
	SetScrollParams(d mat32.Dims, sb *Slider)
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
func (l *Layout) AsLayout() *Layout {
	return l
}

//////////////////////////////////////////////////////////////
//  GeomSize

// GeomCT has core layout elements: Content and Total
type GeomCT struct { //gti:add
	// Content is for the contents (children, parts) of the widget,
	// excluding the Space (margin, padding, scrollbars).
	// This content includes the InnerSpace factor (Gaps in Layout)
	// which must therefore be subtracted when allocating down to children.
	Content mat32.Vec2

	// Total is for the total exterior of the widget: Content + Space
	Total mat32.Vec2
}

func (ct GeomCT) String() string {
	return fmt.Sprintf("Content: %v, \tTotal: %v", ct.Content, ct.Total)
}

// GeomSize has all of the relevant Layout sizes
type GeomSize struct { //gti:add
	// Actual is the actual size for the purposes of rendering, representing
	// the "external" demands of the widget for space from its parent.
	// This is initially the bottom-up constraint computed by SizeUp,
	// and only changes during SizeDown when wrapping elements are reshaped
	// based on allocated size, or when scrollbars are added.
	// For elements with scrollbars (OverflowAuto), the Actual size remains
	// at the initial style minimums, "absorbing" is internal size,
	// while Internal records the true size of the contents.
	// For SizeFinal, Actual size can Grow up to the final Alloc size,
	// while Internal records the actual bottom-up contents size.
	Actual GeomCT `view:"inline"`

	// Alloc is the top-down allocated size, based on available visible space,
	// starting from the Scene geometry and working downward, attempting to
	// accommodate the Actual contents, and allocating extra space based on
	// Grow factors.  When Actual < Alloc, alignment factors determine positioning
	// within the allocated space.
	Alloc GeomCT `view:"inline"`

	// Internal is the internal size representing the true size of all contents
	// of the widget.  This can be less than Actual.Content if widget has Grow
	// factors but its internal contents did not grow accordingly, or it can
	// be more than Actual.Content if it has scrollbars (OverflowAuto).
	// Note that this includes InnerSpace (Gap).
	Internal mat32.Vec2

	// Space is the padding, total effective margin (border, shadow, etc),
	// and scrollbars that subtracts from Total size to get Content size.
	Space mat32.Vec2

	// InnerSpace is total extra space that is included within the Content Size region
	// and must be subtracted from Content when passing sizes down to children.
	InnerSpace mat32.Vec2

	// Min is the Styles.Min.Dots() (Ceil int) that constrains the Actual.Content size
	Min mat32.Vec2

	// Max is the Styles.Max.Dots() (Ceil int) that constrains the Actual.Content size
	Max mat32.Vec2
}

func (ls GeomSize) String() string {
	return fmt.Sprintf("Actual: %v, \tAlloc: %v", ls.Actual, ls.Alloc)
}

// SetInitContentMin sets initial Actual.Content size from given Styles.Min,
// further subject to the current Max constraint.
func (ls *GeomSize) SetInitContentMin(styMin mat32.Vec2) {
	csz := &ls.Actual.Content
	*csz = styMin
	styles.SetClampMaxVec(csz, ls.Max)
}

// FitSizeMax increases given size to fit given fm value, subject to Max constraints
func (ls *GeomSize) FitSizeMax(to *mat32.Vec2, fm mat32.Vec2) {
	styles.SetClampMinVec(to, fm)
	styles.SetClampMaxVec(to, ls.Max)
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
type GeomState struct { //gti:add
	// Size has sizing data for the widget: use Actual for rendering.
	// Alloc shows the potentially larger space top-down allocated.
	Size GeomSize `view:"add-fields"`

	// Pos is position within the overall Scene that we render into,
	// including effects of scroll offset, for both Total outer dimension
	// and inner Content dimension.
	Pos GeomCT `view:"inline" edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// Cell is the logical X, Y index coordinates (col, row) of element
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
	TotalBBox image.Rectangle `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// 2D bounding box for our Content, which excludes our padding, margin, etc.
	// starting at Pos.Content and ending at Pos.Content + Size.Content.
	// It is intersected with parent bounding boxes.
	ContentBBox image.Rectangle `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`
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

// TotalRect returns Pos.Total -- Size.Actual.Total
// as an image.Rectangle, e.g., for bounding box
func (ls *GeomState) TotalRect() image.Rectangle {
	return mat32.RectFromPosSizeMax(ls.Pos.Total, ls.Size.Actual.Total)
}

// ContentRect returns Pos.Content, Size.Actual.Content
// as an image.Rectangle, e.g., for bounding box.
func (ls *GeomState) ContentRect() image.Rectangle {
	return mat32.RectFromPosSizeMax(ls.Pos.Content, ls.Size.Actual.Content)
}

// ScrollOffset computes the net scrolling offset as a function of
// the difference between the allocated position and the actual
// content position according to the clipped bounding box.
func (ls *GeomState) ScrollOffset() image.Point {
	return ls.ContentBBox.Min.Sub(ls.Pos.Content.ToPoint())
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

// LayCells holds one set of LayCell cell elements for rows, cols.
// There can be multiple of these for Wrap case.
type LayCells struct {
	// Shape is number of cells along each dimension for our ColRow cells,
	Shape image.Point `edit:"-"`

	// ColRow has the data for the columns in [0] and rows in [1]:
	// col Size.X = max(X over rows) (cross axis), .Y = sum(Y over rows) (main axis for col)
	// row Size.X = sum(X over cols) (main axis for row), .Y = max(Y over cols) (cross axis)
	// see: https://docs.google.com/spreadsheets/d/1eimUOIJLyj60so94qUr4Buzruj2ulpG5o6QwG2nyxRw/edit?usp=sharing
	ColRow [2][]LayCell `edit:"-"`
}

// Cell returns the cell for given dimension and index along that
// dimension (X = Cols, idx = col, Y = Rows, idx = row)
func (lc *LayCells) Cell(d mat32.Dims, idx int) *LayCell {
	if len(lc.ColRow[d]) <= idx {
		return nil
	}
	return &(lc.ColRow[d][idx])
}

// Init initializes Cells for given shape
func (lc *LayCells) Init(shape image.Point) {
	lc.Shape = shape
	for d := mat32.X; d <= mat32.Y; d++ {
		n := mat32.PointDim(lc.Shape, d)
		if len(lc.ColRow[d]) != n {
			lc.ColRow[d] = make([]LayCell, n)
		}
		for i := 0; i < n; i++ {
			lc.ColRow[d][i].Reset()
		}
	}
}

// CellsSize returns the total Size represented by the current Cells,
// which is the Sum of the Max values along each dimension.
func (lc *LayCells) CellsSize() mat32.Vec2 {
	var ksz mat32.Vec2
	for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
		n := mat32.PointDim(lc.Shape, ma) // cols, rows
		sum := float32(0)
		for mi := 0; mi < n; mi++ {
			md := lc.Cell(ma, mi) // X, Y
			mx := md.Size.Dim(ma)
			sum += mx // sum of maxes
		}
		ksz.SetDim(ma, sum)
	}
	return ksz.Ceil()
}

// GapSizeDim returns the gap size for given dimension, based on Shape and given gap size
func (lc *LayCells) GapSizeDim(d mat32.Dims, gap float32) float32 {
	n := mat32.PointDim(lc.Shape, d)
	return float32(n-1) * gap
}

func (lc *LayCells) String() string {
	s := ""
	n := lc.Shape.X
	for i := 0; i < n; i++ {
		col := lc.Cell(mat32.X, i)
		s += fmt.Sprintln("col:", i, "\tmax X:", col.Size.X, "\tsum Y:", col.Size.Y, "\tmax grX:", col.Grow.X, "\tsum grY:", col.Grow.Y)
	}
	n = lc.Shape.Y
	for i := 0; i < n; i++ {
		row := lc.Cell(mat32.Y, i)
		s += fmt.Sprintln("row:", i, "\tsum X:", row.Size.X, "\tmax Y:", row.Size.Y, "\tsum grX:", row.Grow.X, "\tmax grY:", row.Grow.Y)
	}
	return s
}

// LayImplState has internal state for implementing layout
type LayImplState struct {
	// Shape is number of cells along each dimension,
	// computed for each layout type:
	// For Grid: Max Col, Row.
	// For Flex no Wrap: Cols,1 (X) or 1,Rows (Y).
	// For Flex Wrap: Cols,Max(Rows) or Max(Cols),Rows
	// For Stacked: 1,1
	Shape image.Point `edit:"-"`

	// MainAxis cached here from Styles, to enable Wrap-based access.
	MainAxis mat32.Dims

	// Wraps is the number of actual items in each Wrap for Wrap case:
	// MainAxis X: Len = Rows, Val = Cols; MainAxis Y: Len = Cols, Val = Rows.
	// This should be nil for non-Wrap case.
	Wraps []int

	// Cells has the Actual size and Grow factor data for each of the child elements,
	// organized according to the Shape and Display type.
	// For non-Wrap, has one element in slice, with cells based on Shape.
	// For Wrap, slice is number of CrossAxis wraps allocated:
	// MainAxis X = Rows; MainAxis Y = Cols, and Cells are MainAxis layout x 1 CrossAxis.
	Cells []LayCells `edit:"-"`

	// ScrollSize has the scrollbar sizes (widths) for each dim, which adds to Space.
	// If there is a vertical scrollbar, X has width; if horizontal, Y has "width" = height
	ScrollSize mat32.Vec2

	// Gap is the Styles.Gap size
	Gap mat32.Vec2

	// GapSize has the total extra gap sizing between elements, which adds to Space.
	// This depends on cell layout so it can vary for Wrap case.
	// For SizeUp / Down Gap contributes to Space like other cases,
	// but for BoundingBox rendering and Alignment, it does NOT, and must be
	// subtracted.  This happens in the Position phase.
	GapSize mat32.Vec2
}

// InitCells initializes the Cells based on Shape, MainAxis, and Wraps
// which must be set before calling.
func (ls *LayImplState) InitCells() {
	if ls.Wraps == nil {
		if len(ls.Cells) != 1 {
			ls.Cells = make([]LayCells, 1)
		}
		ls.Cells[0].Init(ls.Shape)
		return
	}
	ma := ls.MainAxis
	ca := ma.Other()
	nw := len(ls.Wraps)
	if len(ls.Cells) != nw {
		ls.Cells = make([]LayCells, nw)
	}
	for wi, wn := range ls.Wraps {
		var shp image.Point
		mat32.SetPointDim(&shp, ma, wn)
		mat32.SetPointDim(&shp, ca, 1)
		ls.Cells[wi].Init(shp)
	}
}

func (ls *LayImplState) ShapeCheck(w Widget, phase string) bool {
	zp := image.Point{}
	if w.HasChildren() && (ls.Shape == zp || len(ls.Cells) == 0) {
		// fmt.Println(w, "Shape is nil in:", phase)
		return false
	}
	return true
}

// Cell returns the cell for given dimension and index along that
// dimension, and given other-dimension axis which is ignored for non-Wrap cases.
// Does no range checking and will crash if out of bounds.
func (ls *LayImplState) Cell(d mat32.Dims, dIdx, odIdx int) *LayCell {
	if ls.Wraps == nil {
		return ls.Cells[0].Cell(d, dIdx)
	}
	if ls.MainAxis == d {
		return ls.Cells[odIdx].Cell(d, dIdx)
	}
	return ls.Cells[dIdx].Cell(d, 0)
}

// WrapIdxToCoord returns the X,Y coordinates in Wrap case for given sequential idx
func (ls *LayImplState) WrapIdxToCoord(index int) image.Point {
	y := 0
	x := 0
	sum := 0
	if ls.MainAxis == mat32.X {
		for _, nx := range ls.Wraps {
			if index >= sum && index < sum+nx {
				x = index - sum
				break
			}
			sum += nx
			y++
		}
	} else {
		for _, ny := range ls.Wraps {
			if index >= sum && index < sum+ny {
				y = index - sum
				break
			}
			sum += ny
			x++
		}
	}
	return image.Point{x, y}
}

// CellsSize returns the total Size represented by the current Cells,
// which is the Sum of the Max values along each dimension within each Cell,
// Maxed over cross-axis dimension for Wrap case, plus GapSize.
func (ls *LayImplState) CellsSize() mat32.Vec2 {
	if ls.Wraps == nil {
		return ls.Cells[0].CellsSize().Add(ls.GapSize)
	}
	var ksz mat32.Vec2
	d := ls.MainAxis
	od := d.Other()
	gap := ls.Gap.Dim(d)
	for wi := range ls.Wraps {
		wsz := ls.Cells[wi].CellsSize()
		wsz.SetDim(d, wsz.Dim(d)+ls.Cells[wi].GapSizeDim(d, gap))
		if wi == 0 {
			ksz = wsz
		} else {
			ksz.SetDim(d, max(ksz.Dim(d), wsz.Dim(d)))
			ksz.SetDim(od, ksz.Dim(od)+wsz.Dim(od))
		}
	}
	ksz.SetDim(od, ksz.Dim(od)+ls.GapSize.Dim(od))
	return ksz.Ceil()
}

// ColWidth returns the width of given column for given row index
// (ignored for non-Wrap), with full bounds checking.
// Returns error if out of range.
func (ls *LayImplState) ColWidth(row, col int) (float32, error) {
	if ls.Wraps == nil {
		n := mat32.PointDim(ls.Shape, mat32.X)
		if col >= n {
			return 0, fmt.Errorf("Layout.ColWidth: col: %d > number of columns: %d", col, n)
		}
		return ls.Cell(mat32.X, col, 0).Size.X, nil
	}
	nw := len(ls.Wraps)
	if ls.MainAxis == mat32.X {
		if row >= nw {
			return 0, fmt.Errorf("Layout.ColWidth: row: %d > number of rows: %d", row, nw)
		}
		wn := ls.Wraps[row]
		if col >= wn {
			return 0, fmt.Errorf("Layout.ColWidth: col: %d > number of columns: %d", col, wn)
		}
		return ls.Cell(mat32.X, col, row).Size.X, nil
	}
	if col >= nw {
		return 0, fmt.Errorf("Layout.ColWidth: col: %d > number of columns: %d", col, nw)
	}
	wn := ls.Wraps[col]
	if row >= wn {
		return 0, fmt.Errorf("Layout.ColWidth: row: %d > number of rows: %d", row, wn)
	}
	return ls.Cell(mat32.X, col, row).Size.X, nil
}

// RowHeight returns the height of given row for given
// column (ignored for non-Wrap), with full bounds checking.
// Returns error if out of range.
func (ls *LayImplState) RowHeight(row, col int) (float32, error) {
	if ls.Wraps == nil {
		n := mat32.PointDim(ls.Shape, mat32.Y)
		if row >= n {
			return 0, fmt.Errorf("Layout.RowHeight: row: %d > number of rows: %d", row, n)
		}
		return ls.Cell(mat32.Y, 0, row).Size.Y, nil
	}
	nw := len(ls.Wraps)
	if ls.MainAxis == mat32.Y {
		if col >= nw {
			return 0, fmt.Errorf("Layout.RowHeight: col: %d > number of columns: %d", col, nw)
		}
		wn := ls.Wraps[row]
		if col >= wn {
			return 0, fmt.Errorf("Layout.RowHeight: row: %d > number of rows: %d", row, wn)
		}
		return ls.Cell(mat32.Y, col, row).Size.Y, nil
	}
	if row >= nw {
		return 0, fmt.Errorf("Layout.RowHeight: row: %d > number of rows: %d", row, nw)
	}
	wn := ls.Wraps[col]
	if col >= wn {
		return 0, fmt.Errorf("Layout.RowHeight: col: %d > number of columns: %d", col, wn)
	}
	return ls.Cell(mat32.Y, row, col).Size.Y, nil
}

func (ls *LayImplState) String() string {
	if ls.Wraps == nil {
		return ls.Cells[0].String()
	}
	s := ""
	ods := ls.MainAxis.Other().String()
	for wi := range ls.Wraps {
		s += fmt.Sprintf("%s: %d Shape: %v\n", ods, wi, ls.Cells[wi].Shape) + ls.Cells[wi].String()
	}
	return s
}

// StackTopWidget returns the StackTop element as a widget
func (l *Layout) StackTopWidget() (Widget, *WidgetBase) {
	sn := l.Child(l.StackTop)
	return AsWidget(sn)
}

// LaySetContentFitOverflow sets Internal and Actual.Content size to fit given
// new content size, depending on the Styles Overflow: Auto and Scroll types do NOT
// expand Actual and remain at their current styled actual values,
// absorbing the extra content size within their own scrolling zone
// (full size recorded in Internal).
func (l *Layout) LaySetContentFitOverflow(nsz mat32.Vec2, pass LayoutPasses) {
	// todo: potentially the diff between Visible & Hidden is
	// that Hidden also does Not expand beyond Alloc?
	// can expt with that.
	sz := &l.Geom.Size
	asz := &sz.Actual.Content
	isz := &sz.Internal
	sz.SetInitContentMin(sz.Min) // start from style
	*isz = nsz                   // internal is always accurate!
	oflow := &l.Styles.Overflow
	nosz := pass == SizeUpPass && l.Styles.IsFlexWrap()
	for d := mat32.X; d <= mat32.Y; d++ {
		if (nosz || (!(l.Scene != nil && l.Scene.Is(ScPrefSizing)) && oflow.Dim(d) >= styles.OverflowAuto)) && l.Par != nil {
			continue
		}
		asz.SetDim(d, styles.ClampMin(asz.Dim(d), nsz.Dim(d)))
	}
	mx := sz.Max
	styles.SetClampMaxVec(isz, mx)
	styles.SetClampMaxVec(asz, mx)
	sz.SetTotalFromContent(&sz.Actual)
}

//////////////////////////////////////////////////////////////////////
//		SizeUp

// SizeUp (bottom-up) gathers Actual sizes from our Children & Parts,
// based on Styles.Min / Max sizes and actual content sizing
// (e.g., text size).  Flexible elements (e.g., Label, Flex Wrap,
// TopAppBar) should reserve the _minimum_ size possible at this stage,
// and then Grow based on SizeDown allocation.
func (wb *WidgetBase) SizeUp() {
	wb.SizeUpWidget()
}

// SizeUpWidget is the standard Widget SizeUp pass
func (wb *WidgetBase) SizeUpWidget() {
	wb.SizeFromStyle()
	wb.SizeUpParts()
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
	sz.Min = wb.Styles.Min.Dots().Ceil()
	sz.Max = wb.Styles.Max.Dots().Ceil()
	sz.Internal.SetZero()
	sz.SetInitContentMin(sz.Min)
	sz.SetTotalFromContent(&sz.Actual)
	if DebugSettings.LayoutTrace && (sz.Actual.Content.X > 0 || sz.Actual.Content.Y > 0) {
		fmt.Println(wb, "SizeUp from Style:", sz.Actual.Content.String())
	}
}

// SizeUpParts adjusts the Content size to hold the Parts layout if present
func (wb *WidgetBase) SizeUpParts() {
	if wb.Parts == nil {
		return
	}
	wb.Parts.SizeUp()
	sz := &wb.Geom.Size
	sz.FitSizeMax(&sz.Actual.Content, wb.Parts.Geom.Size.Actual.Total)
}

/////////////// Layout

func (l *Layout) SizeUp() {
	l.SizeUpLay()
}

// SizeUpLay is the Layout standard SizeUp pass
func (l *Layout) SizeUpLay() {
	if !l.HasChildren() {
		l.SizeUpWidget() // behave like a widget
		return
	}
	l.SizeFromStyle()
	l.LayImpl.ScrollSize.SetZero() // we don't know yet
	l.LaySetInitCells()
	l.This().(Layouter).LayoutSpace()
	l.SizeUpChildren() // kids do their own thing
	l.SizeFromChildrenFit(0, SizeUpPass)
	if l.Parts != nil {
		l.Parts.SizeUp() // just to get sizes -- no std role in layout
	}
}

// LayoutSpace sets our Space based on Styles and Scroll.
// Other layout types can change this if they want to.
func (l *Layout) LayoutSpace() {
	l.SpaceFromStyle()
	l.Geom.Size.Space.SetAdd(l.LayImpl.ScrollSize)
}

// SizeUpChildren calls SizeUp on all the children of this node
func (wb *WidgetBase) SizeUpChildren() {
	wb.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.SizeUp()
		return ki.Continue
	})
}

// SizeUpChildren calls SizeUp on all the children of this node
func (l *Layout) SizeUpChildren() {
	if l.Styles.Display == styles.Stacked && !l.Is(LayoutStackTopOnly) {
		l.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			kwi.SizeUp()
			return ki.Continue
		})
		return
	}
	l.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.SizeUp()
		return ki.Continue
	})
}

// SetInitCells sets the initial default assignment of cell indexes
// to each widget, based on layout type.
func (l *Layout) LaySetInitCells() {
	switch {
	case l.Styles.Display == styles.Flex:
		if l.Styles.Wrap {
			l.LaySetInitCellsWrap()
		} else {
			l.LaySetInitCellsFlex()
		}
	case l.Styles.Display == styles.Stacked:
		l.LaySetInitCellsStacked()
	case l.Styles.Display == styles.Grid:
		l.LaySetInitCellsGrid()
	default:
		l.LaySetInitCellsStacked() // whatever
	}
	l.LayImpl.InitCells()
	l.LaySetGapSizeFromCells()
	l.LayImpl.ShapeCheck(l, "SizeUp")
	// fmt.Println(l, "SzUp Init", l.LayImpl.Shape)
}

func (l *Layout) LaySetGapSizeFromCells() {
	li := &l.LayImpl
	li.Gap = l.Styles.Gap.Dots().Floor()
	// note: this is not accurate for flex
	li.GapSize.X = max(float32(li.Shape.X-1)*li.Gap.X, 0)
	li.GapSize.Y = max(float32(li.Shape.Y-1)*li.Gap.Y, 0)
	l.Geom.Size.InnerSpace = li.GapSize
}

func (l *Layout) LaySetInitCellsFlex() {
	li := &l.LayImpl
	li.MainAxis = mat32.Dims(l.Styles.Direction)
	ca := li.MainAxis.Other()
	li.Wraps = nil
	idx := 0
	l.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		mat32.SetPointDim(&kwb.Geom.Cell, li.MainAxis, idx)
		mat32.SetPointDim(&kwb.Geom.Cell, ca, 0)
		idx++
		return ki.Continue
	})
	if idx == 0 {
		if DebugSettings.LayoutTrace {
			fmt.Println(l, "no items:", idx)
		}
	}
	mat32.SetPointDim(&li.Shape, li.MainAxis, max(idx, 1)) // must be at least 1
	mat32.SetPointDim(&li.Shape, ca, 1)
}

func (l *Layout) LaySetInitCellsWrap() {
	li := &l.LayImpl
	li.MainAxis = mat32.Dims(l.Styles.Direction)
	ni := 0
	l.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		ni++
		return ki.Continue
	})
	if ni == 0 {
		li.Shape = image.Point{1, 1}
		li.Wraps = nil
		li.GapSize.SetZero()
		l.Geom.Size.InnerSpace.SetZero()
		if DebugSettings.LayoutTrace {
			fmt.Println(l, "no items:", ni)
		}
		return
	}
	nm := max(int(mat32.Sqrt(float32(ni))), 1)
	nc := max(ni/nm, 1)
	for nm*nc < ni {
		nm++
	}
	li.Wraps = make([]int, nc)
	sum := 0
	for i := range li.Wraps {
		n := min(ni-sum, nm)
		li.Wraps[i] = n
		sum += n
	}
	l.LaySetWrapIdxs()
}

// LaySetWrapIdxs sets indexes for Wrap case
func (l *Layout) LaySetWrapIdxs() {
	li := &l.LayImpl
	idx := 0
	var maxc image.Point
	l.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		ic := li.WrapIdxToCoord(idx)
		kwb.Geom.Cell = ic
		if ic.X > maxc.X {
			maxc.X = ic.X
		}
		if ic.Y > maxc.Y {
			maxc.Y = ic.Y
		}
		idx++
		return ki.Continue
	})
	maxc.X++
	maxc.Y++
	li.Shape = maxc
}

// UpdateStackedVisbility updates the visibility for Stacked layouts
// so the StackTop widget is visible, and others are Invisible.
func (l *Layout) UpdateStackedVisibility() {
	l.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwb.SetState(i != l.StackTop, states.Invisible)
		kwb.Geom.Cell = image.Point{0, 0}
		return ki.Continue
	})
}

func (l *Layout) LaySetInitCellsStacked() {
	l.UpdateStackedVisibility()
	l.LayImpl.Shape = image.Point{1, 1}
}

func (l *Layout) LaySetInitCellsGrid() {
	n := len(*l.Children())
	cols := l.Styles.Columns
	if cols == 0 {
		cols = int(mat32.Sqrt(float32(n)))
	}
	rows := n / cols
	for rows*cols < n {
		rows++
	}
	if rows == 0 || cols == 0 {
		fmt.Println(l, "no rows or cols:", rows, cols)
	}
	l.LayImpl.Shape = image.Point{max(cols, 1), max(rows, 1)}
	ci := 0
	ri := 0
	l.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
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
}

//////////////////////////////////////////////////////////////////////
//		SizeFromChildren

// SizeFromChildrenFit gathers Actual size from kids, and calls LaySetContentFitOverflow
// to update Actual and Internal size based on this.
func (l *Layout) SizeFromChildrenFit(iter int, pass LayoutPasses) {
	ksz := l.This().(Layouter).SizeFromChildren(iter, SizeDownPass)
	l.LaySetContentFitOverflow(ksz, pass)
	if DebugSettings.LayoutTrace {
		sz := &l.Geom.Size
		fmt.Println(l, pass, "FromChildren:", ksz, "Content:", sz.Actual.Content, "Internal:", sz.Internal)
	}
}

// SizeFromChildren gathers Actual size from kids.
// Different Layout types can alter this to present different Content
// sizes for the layout process, e.g., if Content is sized to fit allocation,
// as in the TopAppBar and Sliceview types.
func (l *Layout) SizeFromChildren(iter int, pass LayoutPasses) mat32.Vec2 {
	var ksz mat32.Vec2
	if l.Styles.Display == styles.Stacked {
		ksz = l.SizeFromChildrenStacked()
	} else {
		ksz = l.SizeFromChildrenCells(iter, pass)
	}
	return ksz
}

// SizeFromChildrenCells for Flex, Grid
func (l *Layout) SizeFromChildrenCells(iter int, pass LayoutPasses) mat32.Vec2 {
	// r   0   1   col X = max(X over rows), Y = sum(Y over rows)
	//   +--+--+
	// 0 |  |  |   row X = sum(X over cols), Y = max(Y over cols)
	//   +--+--+
	// 1 |  |  |
	//   +--+--+
	li := &l.LayImpl
	li.InitCells()
	l.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Geom.Cell
		sz := kwb.Geom.Size.Actual.Total
		grw := kwb.Styles.Grow
		if pass <= SizeDownPass && iter == 0 && kwb.Styles.GrowWrap {
			grw.Set(1, 0)
		}
		if DebugSettings.LayoutTraceDetail {
			fmt.Println("SzUp i:", i, kwb, "cidx:", cidx, "sz:", sz, "grw:", grw)
		}
		for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
			ca := ma.Other()               // cross axis = Y then X
			mi := mat32.PointDim(cidx, ma) // X, Y
			ci := mat32.PointDim(cidx, ca) // Y, X

			md := li.Cell(ma, mi, ci) // X, Y
			cd := li.Cell(ca, ci, mi) // Y, X
			if md == nil || cd == nil {
				break
			}
			msz := sz.Dim(ma) // main axis size dim: X, Y
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
	if DebugSettings.LayoutTraceDetail {
		fmt.Println(l, "SizeFromChildren")
		fmt.Println(li.String())
	}
	ksz := li.CellsSize()
	return ksz
}

// SizeFromChildrenStacked for stacked case
func (l *Layout) SizeFromChildrenStacked() mat32.Vec2 {
	l.LayImpl.InitCells()
	_, kwb := l.StackTopWidget()
	li := &l.LayImpl
	var ksz mat32.Vec2
	if kwb != nil {
		ksz = kwb.Geom.Size.Actual.Total
		kgrw := kwb.Styles.Grow
		for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
			md := li.Cell(ma, 0, 0)
			md.Size = ksz
			md.Grow = kgrw
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
func (wb *WidgetBase) SizeDown(iter int) bool {
	return wb.SizeDownWidget(iter)
}

// SizeDownWidget is the standard widget implementation of SizeDown.
// It just delegates to the Parts.
func (wb *WidgetBase) SizeDownWidget(iter int) bool {
	return wb.SizeDownParts(iter)
}

func (wb *WidgetBase) SizeDownParts(iter int) bool {
	if wb.Parts == nil {
		return false
	}
	sz := &wb.Geom.Size
	psz := &wb.Parts.Geom.Size
	pgrow, _ := wb.GrowToAllocSize(sz.Actual.Content, sz.Alloc.Content)
	psz.Alloc.Total = pgrow // parts = content
	psz.SetContentFromTotal(&psz.Alloc)
	redo := wb.Parts.SizeDown(iter)
	if redo && DebugSettings.LayoutTrace {
		fmt.Println(wb, "Parts triggered redo")
	}
	return redo
}

// SizeDownChildren calls SizeDown on the Children.
// The kids must have their Size.Alloc set prior to this, which
// is what Layout type does.  Other special widget types can
// do custom layout and call this too.
func (wb *WidgetBase) SizeDownChildren(iter int) bool {
	redo := false
	wb.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		re := kwi.SizeDown(iter)
		if re && DebugSettings.LayoutTrace {
			fmt.Println(wb, "SizeDownChildren child:", kwb.Nm, "triggered redo")
		}
		redo = redo || re
		return ki.Continue
	})
	return redo
}

// SizeDownChildren calls SizeDown on the Children.
// The kids must have their Size.Alloc set prior to this, which
// is what Layout type does.  Other special widget types can
// do custom layout and call this too.
func (l *Layout) SizeDownChildren(iter int) bool {
	if l.Styles.Display == styles.Stacked && !l.Is(LayoutStackTopOnly) {
		redo := false
		l.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			re := kwi.SizeDown(iter)
			if i == l.StackTop {
				redo = redo || re
			}
			return ki.Continue
		})
		return redo
	}
	return l.WidgetBase.SizeDownChildren(iter)
}

// GrowToAllocSize returns the potential size that widget could grow,
// for any dimension with a non-zero Grow factor.
// If Grow is < 1, then the size is increased in proportion, but
// any factor > 1 produces a full fill along that dimension.
// Returns true if this resulted in a change.
func (wb *WidgetBase) GrowToAllocSize(act, alloc mat32.Vec2) (mat32.Vec2, bool) {
	change := false
	for d := mat32.X; d <= mat32.Y; d++ {
		grw := wb.Styles.Grow.Dim(d)
		allocd := alloc.Dim(d)
		actd := act.Dim(d)
		if grw > 0 && allocd > actd {
			grw := min(1, grw)
			nsz := mat32.Ceil(actd + grw*(allocd-actd))
			styles.SetClampMax(&nsz, wb.Geom.Size.Max.Dim(d))
			if nsz != actd {
				change = true
			}
			act.SetDim(d, nsz)
		}
	}
	return act.Ceil(), change
}

/////////////// Layout

func (l *Layout) SizeDown(iter int) bool {
	redo := l.SizeDownLay(iter)
	if redo && DebugSettings.LayoutTrace {
		fmt.Println(l, "SizeDown redo")
	}
	return redo
}

// SizeDownLay is the Layout standard SizeDown pass, returning true if another
// iteration is required.  It allocates sizes to fit given parent-allocated
// total size.
func (l *Layout) SizeDownLay(iter int) bool {
	if !l.HasChildren() || !l.LayImpl.ShapeCheck(l, "SizeDown") {
		return l.SizeDownWidget(iter) // behave like a widget
	}
	sz := &l.Geom.Size
	styles.SetClampMaxVec(&sz.Alloc.Content, sz.Max) // can't be more than max..
	sz.SetTotalFromContent(&sz.Alloc)
	if DebugSettings.LayoutTrace {
		fmt.Println(l, "Managing Alloc:", sz.Alloc.Content)
	}
	chg := l.This().(Layouter).ManageOverflow(iter, true) // this must go first.
	wrapped := false
	if iter <= 1 && l.Styles.IsFlexWrap() {
		wrapped = l.SizeDownWrap(iter) // first recompute wrap
		if iter == 0 {
			wrapped = true // always update
		}
	}
	l.This().(Layouter).SizeDownSetAllocs(iter)
	redo := l.SizeDownChildren(iter)
	if redo || wrapped {
		l.SizeFromChildrenFit(iter, SizeDownPass)
	}
	l.SizeDownParts(iter) // no std role, just get sizes
	return chg || wrapped || redo
}

// SizeDownSetAllocs is the key SizeDown step that sets the allocations
// in the children, based on our allocation.  In the default implementation
// this calls SizeDownGrow if there is extra space to grow, or
// SizeDownAllocActual to set the allocations as they currrently are.
func (l *Layout) SizeDownSetAllocs(iter int) {
	sz := &l.Geom.Size
	extra := sz.Alloc.Content.Sub(sz.Internal) // note: critical to use internal to be accurate
	if extra.X > 0 || extra.Y > 0 {
		if DebugSettings.LayoutTrace {
			fmt.Println(l, "SizeDown extra:", extra, "Internal:", sz.Internal, "Alloc:", sz.Alloc.Content)
		}
		l.SizeDownGrow(iter, extra)
	} else {
		l.SizeDownAllocActual(iter) // set allocations as is
	}
}

// ManageOverflow uses overflow settings to determine if scrollbars
// are needed (Internal > Alloc).  Returns true if size changes as a result.
// If updtSize is false, then the Actual and Alloc sizes are NOT
// updated as a result of change from adding scrollbars
// (generally should be true, but some cases not)
func (l *Layout) ManageOverflow(iter int, updtSize bool) bool {
	sz := &l.Geom.Size
	sbw := mat32.Ceil(l.Styles.ScrollBarWidth.Dots)
	change := false
	if iter == 0 {
		l.LayImpl.ScrollSize.SetZero()
		l.SetScrollsOff()
		for d := mat32.X; d <= mat32.Y; d++ {
			if l.Styles.Overflow.Dim(d) == styles.OverflowScroll {
				if !l.HasScroll[d] {
					change = true
				}
				l.HasScroll[d] = true
				l.LayImpl.ScrollSize.SetDim(d.Other(), sbw)
			}
		}
	}
	for d := mat32.X; d <= mat32.Y; d++ {
		maxSize, visSize, _ := l.This().(Layouter).ScrollValues(d)
		ofd := maxSize - visSize
		switch l.Styles.Overflow.Dim(d) {
		// case styles.OverflowVisible:
		// note: this shouldn't happen -- just have this in here for monitoring
		// fmt.Println(l, "OverflowVisible ERROR -- shouldn't have overflow:", d, ofd)
		case styles.OverflowAuto:
			if ofd <= 1 {
				if l.HasScroll[d] {
					if DebugSettings.LayoutTrace {
						fmt.Println(l, "turned off scroll", d)
					}
					change = true
					l.HasScroll[d] = false
					l.LayImpl.ScrollSize.SetDim(d.Other(), 0)
				}
				continue
			}
			if !l.HasScroll[d] {
				change = true
			}
			l.HasScroll[d] = true
			l.LayImpl.ScrollSize.SetDim(d.Other(), sbw)
			if change && DebugSettings.LayoutTrace {
				fmt.Println(l, "OverflowAuto enabling scrollbars for dim for overflow:", d, ofd, "alloc:", sz.Alloc.Content.Dim(d), "internal:", sz.Internal.Dim(d))
			}
		}
	}
	l.This().(Layouter).LayoutSpace() // adds the scroll space
	if updtSize {
		sz.SetTotalFromContent(&sz.Actual)
		sz.SetContentFromTotal(&sz.Alloc) // alloc is *decreased* from any increase in space
	}
	if change && DebugSettings.LayoutTrace {
		fmt.Println(l, "ManageOverflow changed")
	}
	return change
}

// SizeDownGrow grows the element sizes based on total extra and Grow
func (l *Layout) SizeDownGrow(iter int, extra mat32.Vec2) bool {
	redo := false
	if l.Styles.Display == styles.Stacked {
		redo = l.SizeDownGrowStacked(iter, extra)
	} else {
		redo = l.SizeDownGrowCells(iter, extra)
	}
	return redo
}

func (l *Layout) SizeDownGrowCells(iter int, extra mat32.Vec2) bool {
	redo := false
	sz := &l.Geom.Size
	alloc := sz.Alloc.Content.Sub(sz.InnerSpace) // inner is fixed
	// todo: use max growth values instead of individual ones to ensure consistency!
	li := &l.LayImpl
	if len(li.Cells) == 0 {
		panic(fmt.Sprintf("%v Has not been initialized -- UpdateStart / End error!", l.String()))
	}
	l.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Geom.Cell
		ksz := &kwb.Geom.Size
		grw := kwb.Styles.Grow
		if iter == 0 && kwb.Styles.GrowWrap {
			grw.Set(1, 0)
		}
		// if DebugSettings.LayoutTrace {
		// 	fmt.Println("szdn i:", i, kwb, "cidx:", cidx, "sz:", sz, "grw:", grw)
		// }
		for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
			gr := grw.Dim(ma)
			ca := ma.Other()     // cross axis = Y then X
			exd := extra.Dim(ma) // row.X = extra width for cols; col.Y = extra height for rows in this col
			if exd < 0 {
				exd = 0
			}
			mi := mat32.PointDim(cidx, ma) // X, Y
			ci := mat32.PointDim(cidx, ca) // Y, X
			md := li.Cell(ma, mi, ci)      // X, Y
			cd := li.Cell(ca, ci, mi)      // Y, X
			if md == nil || cd == nil {
				break
			}
			mx := md.Size.Dim(ma)
			asz := mx
			gsum := cd.Grow.Dim(ma)
			if gsum > 0 && exd > 0 {
				if gr > gsum {
					fmt.Println(l, "SizeDownGrowCells error: grow > grow sum:", gr, gsum)
					gr = gsum
				}
				redo = true
				asz = mat32.Round(mx + exd*(gr/gsum))  // todo: could use Floor
				if asz > mat32.Ceil(alloc.Dim(ma))+1 { // bug!
					fmt.Println(l, "SizeDownGrowCells error: sub alloc > total to alloc:", asz, alloc.Dim(ma))
					fmt.Println("ma:", ma, "mi:", mi, "ci:", ci, "mx:", mx, "gsum:", gsum, "gr:", gr, "ex:", exd, "par act:", sz.Actual.Content.Dim(ma))
					fmt.Println(l.LayImpl.String())
					fmt.Println(l.LayImpl.CellsSize())
				}
			}
			if DebugSettings.LayoutTraceDetail {
				fmt.Println(kwb, ma, "alloc:", asz, "was act:", sz.Actual.Total.Dim(ma), "mx:", mx, "gsum:", gsum, "gr:", gr, "ex:", exd)
			}
			ksz.Alloc.Total.SetDim(ma, asz)
		}
		ksz.SetContentFromTotal(&ksz.Alloc)
		return ki.Continue
	})
	return redo
}

func (l *Layout) SizeDownWrap(iter int) bool {
	wrapped := false
	li := &l.LayImpl
	sz := &l.Geom.Size
	d := li.MainAxis
	alloc := sz.Alloc.Content
	gap := li.Gap.Dim(d)
	fit := alloc.Dim(d)
	if DebugSettings.LayoutTrace {
		fmt.Println(l, "SizeDownWrap fitting into:", d, fit)
	}
	first := true
	var sum float32
	var n int
	var wraps []int
	l.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		ksz := kwb.Geom.Size.Actual.Total
		if first {
			n = 1
			sum = ksz.Dim(d) + gap
			first = false
			return ki.Continue
		}
		if sum+ksz.Dim(d)+gap >= fit {
			if DebugSettings.LayoutTraceDetail {
				fmt.Println(l, "wrapped:", i, sum, ksz.Dim(d), fit)
			}
			wraps = append(wraps, n)
			sum = ksz.Dim(d)
			n = 1 // this guy is on next line
		} else {
			sum += ksz.Dim(d) + gap
			n++
		}
		return ki.Continue
	})
	if n > 0 {
		wraps = append(wraps, n)
	}
	wrapped = false
	if len(wraps) != len(li.Wraps) {
		wrapped = true
	} else {
		for i := range wraps {
			if wraps[i] != li.Wraps[i] {
				wrapped = true
				break
			}
		}
	}
	if !wrapped {
		return false
	}
	if DebugSettings.LayoutTrace {
		fmt.Println(l, "wrapped:", wraps)
	}
	li.Wraps = wraps
	l.LaySetWrapIdxs()
	li.InitCells()
	l.LaySetGapSizeFromCells()
	l.SizeFromChildrenCells(iter, SizeDownPass)
	return wrapped
}

func (l *Layout) SizeDownGrowStacked(iter int, extra mat32.Vec2) bool {
	// stack just gets everything from us
	chg := false
	asz := l.Geom.Size.Alloc.Content
	// todo: could actually use the grow factors to decide if growing here?
	if l.Is(LayoutStackTopOnly) {
		_, kwb := l.StackTopWidget()
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
	l.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
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
func (l *Layout) SizeDownAllocActual(iter int) {
	if l.Styles.Display == styles.Stacked {
		l.SizeDownAllocActualStacked(iter)
		return
	}
	// todo: wrap needs special case
	l.SizeDownAllocActualCells(iter)
}

// SizeDownAllocActualCells sets Alloc to Actual for no-extra case.
// Note however that due to max sizing for row / column,
// this size can actually be different than original actual.
func (l *Layout) SizeDownAllocActualCells(iter int) {
	l.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		ksz := &kwb.Geom.Size
		cidx := kwb.Geom.Cell
		for ma := mat32.X; ma <= mat32.Y; ma++ { // main axis = X then Y
			ca := ma.Other()                 // cross axis = Y then X
			mi := mat32.PointDim(cidx, ma)   // X, Y
			ci := mat32.PointDim(cidx, ca)   // Y, X
			md := l.LayImpl.Cell(ma, mi, ci) // X, Y
			asz := md.Size.Dim(ma)
			ksz.Alloc.Total.SetDim(ma, asz)
		}
		ksz.SetContentFromTotal(&ksz.Alloc)
		return ki.Continue
	})
}

func (l *Layout) SizeDownAllocActualStacked(iter int) {
	// stack just gets everything from us
	asz := l.Geom.Size.Actual.Content
	// todo: could actually use the grow factors to decide if growing here?
	if l.Is(LayoutStackTopOnly) {
		_, kwb := l.StackTopWidget()
		if kwb != nil {
			ksz := &kwb.Geom.Size
			ksz.Alloc.Total = asz
			ksz.SetContentFromTotal(&ksz.Alloc)
		}
		return
	}
	// note: allocate everyone in case they are flipped to top
	// need a new layout if size is actually different
	l.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
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
func (l *Layout) SizeFinalUpdateChildrenSizes() {
	l.SizeUpLay()
	iter := 3 // late stage..
	l.This().(Layouter).SizeDownSetAllocs(iter)
	l.SizeDownChildren(iter)
	l.SizeDownParts(iter) // no std role, just get sizes
}

// SizeFinal: (bottom-up) similar to SizeUp but done at the end of the
// Sizing phase: first grows widget Actual sizes based on their Grow
// factors, up to their Alloc sizes.  Then gathers this updated final
// actual Size information for layouts to register their actual sizes
// prior to positioning, which requires accurate Actual vs. Alloc
// sizes to perform correct alignment calculations.
func (wb *WidgetBase) SizeFinal() {
	wb.SizeFinalWidget()
}

// SizeFinalWidget is the standard Widget SizeFinal pass
func (wb *WidgetBase) SizeFinalWidget() {
	wb.Geom.RelPos.SetZero()
	sz := &wb.Geom.Size
	sz.Internal = sz.Actual.Content // keep it before we grow
	wb.GrowToAlloc()
	wb.StyleSizeUpdate() // now that sizes are stable, ensure styling based on size is updated
	wb.SizeFinalParts()
	sz.SetTotalFromContent(&sz.Actual)
}

// GrowToAlloc grows our Actual size up to current Alloc size
// for any dimension with a non-zero Grow factor.
// If Grow is < 1, then the size is increased in proportion, but
// any factor > 1 produces a full fill along that dimension.
// Returns true if this resulted in a change in our Total size.
func (wb *WidgetBase) GrowToAlloc() bool {
	if (wb.Scene != nil && wb.Scene.Is(ScPrefSizing)) || wb.Styles.GrowWrap {
		return false
	}
	sz := &wb.Geom.Size
	act, change := wb.GrowToAllocSize(sz.Actual.Total, sz.Alloc.Total)
	if change {
		if DebugSettings.LayoutTrace {
			fmt.Println(wb, "GrowToAlloc:", sz.Alloc.Total, "from actual:", sz.Actual.Total)
		}
		sz.Actual.Total = act // already has max constraint
		sz.SetContentFromTotal(&sz.Actual)
		if wb.Styles.LayoutHasParSizing() {
			// todo: requires some additional logic to see if actually changes something
		}
	}
	return change
}

// SizeFinalParts adjusts the Content size to hold the Parts Final sizes
func (wb *WidgetBase) SizeFinalParts() {
	if wb.Parts == nil {
		return
	}
	wb.Parts.SizeFinal()
	sz := &wb.Geom.Size
	sz.FitSizeMax(&sz.Actual.Content, wb.Parts.Geom.Size.Actual.Total)
}

/////////////// Layout

func (l *Layout) SizeFinal() {
	l.SizeFinalLay()
}

// SizeFinalLay is the Layout standard SizeFinal pass
func (l *Layout) SizeFinalLay() {
	if !l.HasChildren() || !l.LayImpl.ShapeCheck(l, "SizeFinal") {
		l.SizeFinalWidget() // behave like a widget
		return
	}
	l.Geom.RelPos.SetZero()
	l.SizeFinalChildren() // kids do their own thing
	l.SizeFromChildrenFit(0, SizeFinalPass)
	l.GrowToAlloc()
	l.StyleSizeUpdate() // now that sizes are stable, ensure styling based on size is updated
}

// SizeFinalChildren calls SizeFinal on all the children of this node
func (wb *WidgetBase) SizeFinalChildren() {
	wb.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.SizeFinal()
		return ki.Continue
	})
}

// SizeFinalChildren calls SizeFinal on all the children of this node
func (l *Layout) SizeFinalChildren() {
	if l.Styles.Display == styles.Stacked && !l.Is(LayoutStackTopOnly) {
		l.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			kwi.SizeFinal()
			return ki.Continue
		})
		return
	}
	l.WidgetBase.SizeFinalChildren()
}

// StyleSizeUpdate updates styling size values for widget and its parent,
// which should be called after these are updated.  Returns true if any changed.
func (wb *WidgetBase) StyleSizeUpdate() bool {
	el := wb.Geom.Size.Actual.Content
	var par mat32.Vec2
	pwb := wb.ParentWidget()
	if pwb != nil {
		par = pwb.Geom.Size.Actual.Content
	}
	sz := wb.Scene.SceneGeom.Size
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
func (wb *WidgetBase) Position() {
	wb.PositionWidget()
}

func (wb *WidgetBase) PositionWidget() {
	wb.PositionParts()
}

func (wb *WidgetBase) PositionWithinAllocMainX(pos mat32.Vec2, parJustify, parAlign styles.Aligns) {
	sz := &wb.Geom.Size
	pos.X += styles.AlignPos(styles.ItemAlign(parJustify, wb.Styles.Justify.Self), sz.Actual.Total.X, sz.Alloc.Total.X)
	pos.Y += styles.AlignPos(styles.ItemAlign(parAlign, wb.Styles.Align.Self), sz.Actual.Total.Y, sz.Alloc.Total.Y)
	wb.Geom.RelPos = pos
	if DebugSettings.LayoutTrace {
		fmt.Println(wb, "Position within Main=X:", pos)
	}
}

func (wb *WidgetBase) PositionWithinAllocMainY(pos mat32.Vec2, parJustify, parAlign styles.Aligns) {
	sz := &wb.Geom.Size
	pos.Y += styles.AlignPos(styles.ItemAlign(parJustify, wb.Styles.Justify.Self), sz.Actual.Total.Y, sz.Alloc.Total.Y)
	pos.X += styles.AlignPos(styles.ItemAlign(parAlign, wb.Styles.Align.Self), sz.Actual.Total.X, sz.Alloc.Total.X)
	wb.Geom.RelPos = pos
	if DebugSettings.LayoutTrace {
		fmt.Println(wb, "Position within Main=Y:", pos)
	}
}

func (wb *WidgetBase) PositionParts() {
	if wb.Parts == nil {
		return
	}
	sz := &wb.Geom.Size
	pgm := &wb.Parts.Geom
	pgm.RelPos.X = styles.AlignPos(wb.Parts.Styles.Justify.Content, pgm.Size.Actual.Total.X, sz.Actual.Content.X)
	pgm.RelPos.Y = styles.AlignPos(wb.Parts.Styles.Align.Content, pgm.Size.Actual.Total.Y, sz.Actual.Content.Y)
	if DebugSettings.LayoutTrace {
		fmt.Println(wb.Parts, "parts align pos:", pgm.RelPos)
	}
	wb.Parts.This().(Widget).Position()
}

// PositionChildren runs Position on the children
func (wb *WidgetBase) PositionChildren() {
	wb.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.Position()
		return ki.Continue
	})
}

// Position: uses the final sizes to position everything within layouts
// according to alignment settings.
func (l *Layout) Position() {
	l.PositionLay()
}

func (l *Layout) PositionLay() {
	if !l.HasChildren() || !l.LayImpl.ShapeCheck(l, "Position") {
		l.PositionWidget() // behave like a widget
		return
	}
	if l.Par == nil {
		l.PositionWithinAllocMainY(mat32.Vec2{}, l.Styles.Justify.Items, l.Styles.Align.Items)
	}
	l.ConfigScrolls() // and configure the scrolls
	if l.Styles.Display == styles.Stacked {
		l.PositionStacked()
	} else {
		l.PositionCells()
		l.PositionChildren()
	}
}

func (l *Layout) PositionCells() {
	if l.Styles.Display == styles.Flex && l.Styles.Direction == styles.Column {
		l.PositionCellsMainY()
		return
	}
	l.PositionCellsMainX()
}

// Main axis = X
func (l *Layout) PositionCellsMainX() {
	// todo: can break apart further into Flex rows
	gap := l.LayImpl.Gap
	sz := &l.Geom.Size
	if DebugSettings.LayoutTraceDetail {
		fmt.Println(l, "PositionCells Main X, alloc:", sz.Alloc.Content, "internal:", sz.Internal)
	}
	var stPos mat32.Vec2
	stPos.X = styles.AlignPos(l.Styles.Justify.Content, sz.Internal.X, sz.Alloc.Content.X)
	stPos.Y = styles.AlignPos(l.Styles.Align.Content, sz.Internal.Y, sz.Alloc.Content.Y)
	pos := stPos
	var lastSz mat32.Vec2
	idx := 0
	l.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Geom.Cell
		if cidx.X == 0 && idx > 0 {
			pos.X = stPos.X
			pos.Y += lastSz.Y + gap.Y
		}
		kwb.PositionWithinAllocMainX(pos, l.Styles.Justify.Items, l.Styles.Align.Items)
		alloc := kwb.Geom.Size.Alloc.Total
		pos.X += alloc.X + gap.X
		lastSz = alloc
		idx++
		return ki.Continue
	})
}

// Main axis = Y
func (l *Layout) PositionCellsMainY() {
	gap := l.LayImpl.Gap
	sz := &l.Geom.Size
	if DebugSettings.LayoutTraceDetail {
		fmt.Println(l, "PositionCells, alloc:", sz.Alloc.Content, "internal:", sz.Internal)
	}
	var lastSz mat32.Vec2
	var stPos mat32.Vec2
	stPos.Y = styles.AlignPos(l.Styles.Justify.Content, sz.Internal.Y, sz.Alloc.Content.Y)
	stPos.X = styles.AlignPos(l.Styles.Align.Content, sz.Internal.X, sz.Alloc.Content.X)
	pos := stPos
	idx := 0
	l.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Geom.Cell
		if cidx.Y == 0 && idx > 0 {
			pos.Y = stPos.Y
			pos.X += lastSz.X + gap.X
		}
		kwb.PositionWithinAllocMainY(pos, l.Styles.Justify.Items, l.Styles.Align.Items)
		alloc := kwb.Geom.Size.Alloc.Total
		pos.Y += alloc.Y + gap.Y
		lastSz = alloc
		idx++
		return ki.Continue
	})
}

func (l *Layout) PositionWrap() {
}

func (l *Layout) PositionStacked() {
	l.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwb.Geom.RelPos.SetZero()
		if !l.Is(LayoutStackTopOnly) || i == l.StackTop {
			kwi.Position()
		}
		return ki.Continue
	})
}

//////////////////////////////////////////////////////////////////////
//		ScenePos

// ScenePos computes scene-based absolute positions and final BBox
// bounding boxes for rendering, based on relative positions from
// Position step and parents accumulated position and scroll offset.
// This is the only step needed when scrolling (very fast).
func (wb *WidgetBase) ScenePos() {
	wb.ScenePosWidget()
}

func (wb *WidgetBase) ScenePosWidget() {
	wb.SetPosFromParent()
	wb.SetBBoxes()
}

// SetContentPosFromPos sets the Pos.Content position based on current Pos
// plus the BoxSpace position offset.
func (wb *WidgetBase) SetContentPosFromPos() {
	off := wb.Styles.BoxSpace().Pos().Floor()
	wb.Geom.Pos.Content = wb.Geom.Pos.Total.Add(off)
}

func (wb *WidgetBase) SetPosFromParent() {
	pwb := wb.ParentWidget()
	var parPos mat32.Vec2
	if pwb != nil {
		parPos = pwb.Geom.Pos.Content.Add(pwb.Geom.Scroll) // critical that parent adds here but not to self
	}
	wb.Geom.Pos.Total = wb.Geom.RelPos.Add(parPos)
	wb.SetContentPosFromPos()
	if DebugSettings.LayoutTrace {
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

func (wb *WidgetBase) SetBBoxes() {
	pwb := wb.ParentWidget()
	var parBB image.Rectangle
	if pwb == nil { // scene
		sz := &wb.Geom.Size
		wb.Geom.TotalBBox = mat32.RectFromPosSizeMax(mat32.Vec2{}, sz.Alloc.Total)
		off := wb.Styles.BoxSpace().Pos().Floor()
		wb.Geom.ContentBBox = mat32.RectFromPosSizeMax(off, sz.Alloc.Content)
		if DebugSettings.LayoutTrace {
			fmt.Println(wb, "Total BBox:", wb.Geom.TotalBBox)
			fmt.Println(wb, "Content BBox:", wb.Geom.ContentBBox)
		}
	} else {
		parBB = pwb.Geom.ContentBBox
		bb := wb.Geom.TotalRect()
		wb.Geom.TotalBBox = parBB.Intersect(bb)
		if DebugSettings.LayoutTrace {
			fmt.Println(wb, "Total BBox:", bb, "parBB:", parBB, "BBox:", wb.Geom.TotalBBox)
		}

		cbb := wb.Geom.ContentRect()
		wb.Geom.ContentBBox = parBB.Intersect(cbb)
		if DebugSettings.LayoutTrace {
			fmt.Println(wb, "Content BBox:", cbb, "parBB:", parBB, "BBox:", wb.Geom.ContentBBox)
		}
	}
	wb.ScenePosParts()
}

func (wb *WidgetBase) ScenePosParts() {
	if wb.Parts == nil {
		return
	}
	wb.Parts.ScenePos()
}

// ScenePosChildren runs ScenePos on the children
func (wb *WidgetBase) ScenePosChildren() {
	wb.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.ScenePos()
		return ki.Continue
	})
}

// ScenePosChildren runs ScenePos on the children
func (l *Layout) ScenePosChildren() {
	if l.Styles.Display == styles.Stacked && !l.Is(LayoutStackTopOnly) {
		l.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			kwi.ScenePos()
			return ki.Continue
		})
		return
	}
	l.WidgetBase.ScenePosChildren()
}

// ScenePos: scene-based position and final BBox is computed based on
// parents accumulated position and scrollbar position.
// This step can be performed when scrolling after updating Scroll.
func (l *Layout) ScenePos() {
	l.ScenePosLay()
}

func (l *Layout) ScenePosLay() {
	if !l.HasChildren() || !l.LayImpl.ShapeCheck(l, "ScenePos") {
		l.ScenePosWidget() // behave like a widget
		return
	}
	// note: l.Geom.Scroll has the X, Y scrolling offsets, set by Layouter.ScrollChanged function
	if !l.HasScroll[mat32.X] {
		l.Geom.Scroll.X = 0
	}
	if !l.HasScroll[mat32.Y] {
		l.Geom.Scroll.Y = 0
	}
	l.ScenePosWidget()
	l.ScenePosChildren()
	l.PositionScrolls()
	l.ScenePosParts() // in case they fit inside parent
	// otherwise handle separately like scrolls on layout
}
