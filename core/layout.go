// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log/slog"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
)

// Layout uses 3 Size passes, 2 Position passes:
//
// SizeUp: (bottom-up) gathers Actual sizes from our Children & Parts,
// based on Styles.Min / Max sizes and actual content sizing
// (e.g., text size).  Flexible elements (e.g., Text, Flex Wrap,
// TopAppBar) should reserve the _minimum_ size possible at this stage,
// and then Grow based on SizeDown allocation.

// SizeDown: (top-down, multiple iterations possible) provides top-down
// size allocations based initially on Scene available size and
// the SizeUp Actual sizes.  If there is extra space available, it is
// allocated according to the Grow factors.
// Flexible elements (e.g., Flex Wrap layouts and Text with word wrap)
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

// Layouter is the interface for layout functions, called by [Frame]
// during the various layout passes.
type Layouter interface {
	Widget

	// AsFrame returns the Layouter as a [Frame].
	AsFrame() *Frame

	// LayoutSpace sets our Space based on Styles, Scroll, and Gap Spacing.
	// Other layout types can change this if they want to.
	LayoutSpace()

	// SizeFromChildren gathers Actual size from kids into our Actual.Content size.
	// Different Layout types can alter this to present different Content
	// sizes for the layout process, e.g., if Content is sized to fit allocation,
	// as in the TopAppBar and List types.
	SizeFromChildren(iter int, pass LayoutPasses) math32.Vector2

	// SizeDownSetAllocs is the key SizeDown step that sets the allocations
	// in the children, based on our allocation.  In the default implementation
	// this calls SizeDownGrow if there is extra space to grow, or
	// SizeDownAllocActual to set the allocations as they currently are.
	SizeDownSetAllocs(iter int)

	// ManageOverflow uses overflow settings to determine if scrollbars
	// are needed, based on difference between ActualOverflow (full actual size)
	// and Alloc allocation.  Returns true if size changes as a result.
	// If updateSize is false, then the Actual and Alloc sizes are NOT
	// updated as a result of change from adding scrollbars
	// (generally should be true, but some cases not)
	ManageOverflow(iter int, updateSize bool) bool

	// ScrollChanged is called in the OnInput event handler for updating
	// when the scrollbar value has changed, for given dimension
	ScrollChanged(d math32.Dims, sb *Slider)

	// ScrollValues returns the maximum size that could be scrolled,
	// the visible size (which could be less than the max size, in which
	// case no scrollbar is needed), and visSize / maxSize as the VisiblePct.
	// This is used in updating the scrollbar and determining whether one is
	// needed in the first place
	ScrollValues(d math32.Dims) (maxSize, visSize, visPct float32)

	// ScrollGeom returns the target position and size for scrollbars
	ScrollGeom(d math32.Dims) (pos, sz math32.Vector2)

	// SetScrollParams sets scrollbar parameters.  Must set Step and PageStep,
	// but can also set others as needed.
	// Max and VisiblePct are automatically set based on ScrollValues maxSize, visPct.
	SetScrollParams(d math32.Dims, sb *Slider)
}

// AsFrame returns the given value as a value of type [Frame] if the type
// of the given value embeds [Frame], or nil otherwise.
func AsFrame(k tree.Node) *Frame {
	if t, ok := k.(Layouter); ok {
		return t.AsFrame()
	}
	return nil
}

// AsFrame satisfies the [Layouter] interface.
func (t *Frame) AsFrame() *Frame {
	return t
}

//////////////////////////////////////////////////////////////
//  GeomSize

// GeomCT has core layout elements: Content and Total
type GeomCT struct { //types:add
	// Content is for the contents (children, parts) of the widget,
	// excluding the Space (margin, padding, scrollbars).
	// This content includes the InnerSpace factor (Gaps in Layout)
	// which must therefore be subtracted when allocating down to children.
	Content math32.Vector2

	// Total is for the total exterior of the widget: Content + Space
	Total math32.Vector2
}

func (ct GeomCT) String() string {
	return fmt.Sprintf("Content: %v, \tTotal: %v", ct.Content, ct.Total)
}

// GeomSize has all of the relevant Layout sizes
type GeomSize struct { //types:add
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
	Actual GeomCT `display:"inline"`

	// Alloc is the top-down allocated size, based on available visible space,
	// starting from the Scene geometry and working downward, attempting to
	// accommodate the Actual contents, and allocating extra space based on
	// Grow factors.  When Actual < Alloc, alignment factors determine positioning
	// within the allocated space.
	Alloc GeomCT `display:"inline"`

	// Internal is the internal size representing the true size of all contents
	// of the widget.  This can be less than Actual.Content if widget has Grow
	// factors but its internal contents did not grow accordingly, or it can
	// be more than Actual.Content if it has scrollbars (OverflowAuto).
	// Note that this includes InnerSpace (Gap).
	Internal math32.Vector2

	// Space is the padding, total effective margin (border, shadow, etc),
	// and scrollbars that subtracts from Total size to get Content size.
	Space math32.Vector2

	// InnerSpace is total extra space that is included within the Content Size region
	// and must be subtracted from Content when passing sizes down to children.
	InnerSpace math32.Vector2

	// Min is the Styles.Min.Dots() (Ceil int) that constrains the Actual.Content size
	Min math32.Vector2

	// Max is the Styles.Max.Dots() (Ceil int) that constrains the Actual.Content size
	Max math32.Vector2
}

func (ls GeomSize) String() string {
	return fmt.Sprintf("Actual: %v, \tAlloc: %v", ls.Actual, ls.Alloc)
}

// SetInitContentMin sets initial Actual.Content size from given Styles.Min,
// further subject to the current Max constraint.
func (ls *GeomSize) SetInitContentMin(styMin math32.Vector2) {
	csz := &ls.Actual.Content
	*csz = styMin
	styles.SetClampMaxVector(csz, ls.Max)
}

// FitSizeMax increases given size to fit given fm value, subject to Max constraints
func (ls *GeomSize) FitSizeMax(to *math32.Vector2, fm math32.Vector2) {
	styles.SetClampMinVector(to, fm)
	styles.SetClampMaxVector(to, ls.Max)
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
type GeomState struct { //types:add
	// Size has sizing data for the widget: use Actual for rendering.
	// Alloc shows the potentially larger space top-down allocated.
	Size GeomSize `display:"add-fields"`

	// Pos is position within the overall Scene that we render into,
	// including effects of scroll offset, for both Total outer dimension
	// and inner Content dimension.
	Pos GeomCT `display:"inline" edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// Cell is the logical X, Y index coordinates (col, row) of element
	// within its parent layout
	Cell image.Point

	// RelPos is top, left position relative to parent Content size space
	RelPos math32.Vector2

	// Scroll is additional scrolling offset within our parent layout
	Scroll math32.Vector2

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
func (ls *GeomState) ContentRangeDim(d math32.Dims) (cmin, cmax float32) {
	cmin = float32(math32.PointDim(ls.ContentBBox.Min, d))
	cmax = float32(math32.PointDim(ls.ContentBBox.Max, d))
	return
}

// TotalRect returns Pos.Total -- Size.Actual.Total
// as an image.Rectangle, e.g., for bounding box
func (ls *GeomState) TotalRect() image.Rectangle {
	return math32.RectFromPosSizeMax(ls.Pos.Total, ls.Size.Actual.Total)
}

// ContentRect returns Pos.Content, Size.Actual.Content
// as an image.Rectangle, e.g., for bounding box.
func (ls *GeomState) ContentRect() image.Rectangle {
	return math32.RectFromPosSizeMax(ls.Pos.Content, ls.Size.Actual.Content)
}

// ScrollOffset computes the net scrolling offset as a function of
// the difference between the allocated position and the actual
// content position according to the clipped bounding box.
func (ls *GeomState) ScrollOffset() image.Point {
	return ls.ContentBBox.Min.Sub(ls.Pos.Content.ToPoint())
}

// LayCell holds the layout implementation data for col, row Cells
type LayCell struct {
	// Size has the Actual size of elements (not Alloc)
	Size math32.Vector2

	// Grow has the Grow factors
	Grow math32.Vector2
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
func (lc *LayCells) Cell(d math32.Dims, idx int) *LayCell {
	if len(lc.ColRow[d]) <= idx {
		return nil
	}
	return &(lc.ColRow[d][idx])
}

// Init initializes Cells for given shape
func (lc *LayCells) Init(shape image.Point) {
	lc.Shape = shape
	for d := math32.X; d <= math32.Y; d++ {
		n := math32.PointDim(lc.Shape, d)
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
func (lc *LayCells) CellsSize() math32.Vector2 {
	var ksz math32.Vector2
	for ma := math32.X; ma <= math32.Y; ma++ { // main axis = X then Y
		n := math32.PointDim(lc.Shape, ma) // cols, rows
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
func (lc *LayCells) GapSizeDim(d math32.Dims, gap float32) float32 {
	n := math32.PointDim(lc.Shape, d)
	return float32(n-1) * gap
}

func (lc *LayCells) String() string {
	s := ""
	n := lc.Shape.X
	for i := 0; i < n; i++ {
		col := lc.Cell(math32.X, i)
		s += fmt.Sprintln("col:", i, "\tmax X:", col.Size.X, "\tsum Y:", col.Size.Y, "\tmax grX:", col.Grow.X, "\tsum grY:", col.Grow.Y)
	}
	n = lc.Shape.Y
	for i := 0; i < n; i++ {
		row := lc.Cell(math32.Y, i)
		s += fmt.Sprintln("row:", i, "\tsum X:", row.Size.X, "\tmax Y:", row.Size.Y, "\tsum grX:", row.Grow.X, "\tmax grY:", row.Grow.Y)
	}
	return s
}

// LayoutState has internal state for implementing layout
type LayoutState struct {
	// Shape is number of cells along each dimension,
	// computed for each layout type:
	// For Grid: Max Col, Row.
	// For Flex no Wrap: Cols,1 (X) or 1,Rows (Y).
	// For Flex Wrap: Cols,Max(Rows) or Max(Cols),Rows
	// For Stacked: 1,1
	Shape image.Point `edit:"-"`

	// MainAxis cached here from Styles, to enable Wrap-based access.
	MainAxis math32.Dims

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
	ScrollSize math32.Vector2

	// Gap is the Styles.Gap size
	Gap math32.Vector2

	// GapSize has the total extra gap sizing between elements, which adds to Space.
	// This depends on cell layout so it can vary for Wrap case.
	// For SizeUp / Down Gap contributes to Space like other cases,
	// but for BoundingBox rendering and Alignment, it does NOT, and must be
	// subtracted.  This happens in the Position phase.
	GapSize math32.Vector2
}

// InitCells initializes the Cells based on Shape, MainAxis, and Wraps
// which must be set before calling.
func (ls *LayoutState) InitCells() {
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
		math32.SetPointDim(&shp, ma, wn)
		math32.SetPointDim(&shp, ca, 1)
		ls.Cells[wi].Init(shp)
	}
}

func (ls *LayoutState) ShapeCheck(w Widget, phase string) bool {
	if w.AsTree().HasChildren() && (ls.Shape == (image.Point{}) || len(ls.Cells) == 0) {
		// fmt.Println(w, "Shape is nil in:", phase) // TODO: plan for this?
		return false
	}
	return true
}

// Cell returns the cell for given dimension and index along that
// dimension, and given other-dimension axis which is ignored for non-Wrap cases.
// Does no range checking and will crash if out of bounds.
func (ls *LayoutState) Cell(d math32.Dims, dIndex, odIndex int) *LayCell {
	if ls.Wraps == nil {
		return ls.Cells[0].Cell(d, dIndex)
	}
	if ls.MainAxis == d {
		return ls.Cells[odIndex].Cell(d, dIndex)
	}
	return ls.Cells[dIndex].Cell(d, 0)
}

// WrapIndexToCoord returns the X,Y coordinates in Wrap case for given sequential idx
func (ls *LayoutState) WrapIndexToCoord(idx int) image.Point {
	y := 0
	x := 0
	sum := 0
	if ls.MainAxis == math32.X {
		for _, nx := range ls.Wraps {
			if idx >= sum && idx < sum+nx {
				x = idx - sum
				break
			}
			sum += nx
			y++
		}
	} else {
		for _, ny := range ls.Wraps {
			if idx >= sum && idx < sum+ny {
				y = idx - sum
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
func (ls *LayoutState) CellsSize() math32.Vector2 {
	if ls.Wraps == nil {
		return ls.Cells[0].CellsSize().Add(ls.GapSize)
	}
	var ksz math32.Vector2
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
func (ls *LayoutState) ColWidth(row, col int) (float32, error) {
	if ls.Wraps == nil {
		n := math32.PointDim(ls.Shape, math32.X)
		if col >= n {
			return 0, fmt.Errorf("Layout.ColWidth: col: %d > number of columns: %d", col, n)
		}
		return ls.Cell(math32.X, col, 0).Size.X, nil
	}
	nw := len(ls.Wraps)
	if ls.MainAxis == math32.X {
		if row >= nw {
			return 0, fmt.Errorf("Layout.ColWidth: row: %d > number of rows: %d", row, nw)
		}
		wn := ls.Wraps[row]
		if col >= wn {
			return 0, fmt.Errorf("Layout.ColWidth: col: %d > number of columns: %d", col, wn)
		}
		return ls.Cell(math32.X, col, row).Size.X, nil
	}
	if col >= nw {
		return 0, fmt.Errorf("Layout.ColWidth: col: %d > number of columns: %d", col, nw)
	}
	wn := ls.Wraps[col]
	if row >= wn {
		return 0, fmt.Errorf("Layout.ColWidth: row: %d > number of rows: %d", row, wn)
	}
	return ls.Cell(math32.X, col, row).Size.X, nil
}

// RowHeight returns the height of given row for given
// column (ignored for non-Wrap), with full bounds checking.
// Returns error if out of range.
func (ls *LayoutState) RowHeight(row, col int) (float32, error) {
	if ls.Wraps == nil {
		n := math32.PointDim(ls.Shape, math32.Y)
		if row >= n {
			return 0, fmt.Errorf("Layout.RowHeight: row: %d > number of rows: %d", row, n)
		}
		return ls.Cell(math32.Y, 0, row).Size.Y, nil
	}
	nw := len(ls.Wraps)
	if ls.MainAxis == math32.Y {
		if col >= nw {
			return 0, fmt.Errorf("Layout.RowHeight: col: %d > number of columns: %d", col, nw)
		}
		wn := ls.Wraps[row]
		if col >= wn {
			return 0, fmt.Errorf("Layout.RowHeight: row: %d > number of rows: %d", row, wn)
		}
		return ls.Cell(math32.Y, col, row).Size.Y, nil
	}
	if row >= nw {
		return 0, fmt.Errorf("Layout.RowHeight: row: %d > number of rows: %d", row, nw)
	}
	wn := ls.Wraps[col]
	if col >= wn {
		return 0, fmt.Errorf("Layout.RowHeight: col: %d > number of columns: %d", col, wn)
	}
	return ls.Cell(math32.Y, row, col).Size.Y, nil
}

func (ls *LayoutState) String() string {
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
func (ly *Frame) StackTopWidget() (Widget, *WidgetBase) {
	sn := ly.Child(ly.StackTop)
	return AsWidget(sn)
}

// LaySetContentFitOverflow sets Internal and Actual.Content size to fit given
// new content size, depending on the Styles Overflow: Auto and Scroll types do NOT
// expand Actual and remain at their current styled actual values,
// absorbing the extra content size within their own scrolling zone
// (full size recorded in Internal).
func (ly *Frame) LaySetContentFitOverflow(nsz math32.Vector2, pass LayoutPasses) {
	// todo: potentially the diff between Visible & Hidden is
	// that Hidden also does Not expand beyond Alloc?
	// can expt with that.
	sz := &ly.Geom.Size
	asz := &sz.Actual.Content
	isz := &sz.Internal
	sz.SetInitContentMin(sz.Min) // start from style
	*isz = nsz                   // internal is always accurate!
	oflow := &ly.Styles.Overflow
	nosz := pass == SizeUpPass && ly.Styles.IsFlexWrap()
	for d := math32.X; d <= math32.Y; d++ {
		if (nosz || (!(ly.Scene != nil && ly.Scene.prefSizing) && oflow.Dim(d) >= styles.OverflowAuto)) && ly.Parent != nil {
			continue
		}
		asz.SetDim(d, styles.ClampMin(asz.Dim(d), nsz.Dim(d)))
	}
	mx := sz.Max
	styles.SetClampMaxVector(isz, mx)
	styles.SetClampMaxVector(asz, mx)
	sz.SetTotalFromContent(&sz.Actual)
}

//////////////////////////////////////////////////////////////////////
//		SizeUp

// SizeUp (bottom-up) gathers Actual sizes from our Children & Parts,
// based on Styles.Min / Max sizes and actual content sizing
// (e.g., text size).  Flexible elements (e.g., Text, Flex Wrap,
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

func (ly *Frame) SizeUp() {
	ly.SizeUpLay()
}

// SizeUpLay is the Layout standard SizeUp pass
func (ly *Frame) SizeUpLay() {
	if !ly.HasChildren() {
		ly.SizeUpWidget() // behave like a widget
		return
	}
	ly.SizeFromStyle()
	ly.Layout.ScrollSize.SetZero() // we don't know yet
	ly.LaySetInitCells()
	ly.This.(Layouter).LayoutSpace()
	ly.SizeUpChildren() // kids do their own thing
	ly.SizeFromChildrenFit(0, SizeUpPass)
	if ly.Parts != nil {
		ly.Parts.SizeUp() // just to get sizes -- no std role in layout
	}
}

// LayoutSpace sets our Space based on Styles and Scroll.
// Other layout types can change this if they want to.
func (ly *Frame) LayoutSpace() {
	ly.SpaceFromStyle()
	ly.Geom.Size.Space.SetAdd(ly.Layout.ScrollSize)
}

// SizeUpChildren calls SizeUp on all the children of this node
func (wb *WidgetBase) SizeUpChildren() {
	wb.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.SizeUp()
		return tree.Continue
	})
}

// SizeUpChildren calls SizeUp on all the children of this node
func (ly *Frame) SizeUpChildren() {
	if ly.Styles.Display == styles.Stacked && !ly.LayoutStackTopOnly {
		ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			kwi.SizeUp()
			return tree.Continue
		})
		return
	}
	ly.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.SizeUp()
		return tree.Continue
	})
}

// SetInitCells sets the initial default assignment of cell indexes
// to each widget, based on layout type.
func (ly *Frame) LaySetInitCells() {
	switch {
	case ly.Styles.Display == styles.Flex:
		if ly.Styles.Wrap {
			ly.LaySetInitCellsWrap()
		} else {
			ly.LaySetInitCellsFlex()
		}
	case ly.Styles.Display == styles.Stacked:
		ly.LaySetInitCellsStacked()
	case ly.Styles.Display == styles.Grid:
		ly.LaySetInitCellsGrid()
	default:
		ly.LaySetInitCellsStacked() // whatever
	}
	ly.Layout.InitCells()
	ly.LaySetGapSizeFromCells()
	ly.Layout.ShapeCheck(ly, "SizeUp")
	// fmt.Println(ly, "SzUp Init", ly.Layout.Shape)
}

func (ly *Frame) LaySetGapSizeFromCells() {
	li := &ly.Layout
	li.Gap = ly.Styles.Gap.Dots().Floor()
	// note: this is not accurate for flex
	li.GapSize.X = max(float32(li.Shape.X-1)*li.Gap.X, 0)
	li.GapSize.Y = max(float32(li.Shape.Y-1)*li.Gap.Y, 0)
	ly.Geom.Size.InnerSpace = li.GapSize
}

func (ly *Frame) LaySetInitCellsFlex() {
	li := &ly.Layout
	li.MainAxis = math32.Dims(ly.Styles.Direction)
	ca := li.MainAxis.Other()
	li.Wraps = nil
	idx := 0
	ly.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		math32.SetPointDim(&kwb.Geom.Cell, li.MainAxis, idx)
		math32.SetPointDim(&kwb.Geom.Cell, ca, 0)
		idx++
		return tree.Continue
	})
	if idx == 0 {
		if DebugSettings.LayoutTrace {
			fmt.Println(ly, "no items:", idx)
		}
	}
	math32.SetPointDim(&li.Shape, li.MainAxis, max(idx, 1)) // must be at least 1
	math32.SetPointDim(&li.Shape, ca, 1)
}

func (ly *Frame) LaySetInitCellsWrap() {
	li := &ly.Layout
	li.MainAxis = math32.Dims(ly.Styles.Direction)
	ni := 0
	ly.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		ni++
		return tree.Continue
	})
	if ni == 0 {
		li.Shape = image.Point{1, 1}
		li.Wraps = nil
		li.GapSize.SetZero()
		ly.Geom.Size.InnerSpace.SetZero()
		if DebugSettings.LayoutTrace {
			fmt.Println(ly, "no items:", ni)
		}
		return
	}
	nm := max(int(math32.Sqrt(float32(ni))), 1)
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
	ly.LaySetWrapIndexes()
}

// LaySetWrapIndexes sets indexes for Wrap case
func (ly *Frame) LaySetWrapIndexes() {
	li := &ly.Layout
	idx := 0
	var maxc image.Point
	ly.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		ic := li.WrapIndexToCoord(idx)
		kwb.Geom.Cell = ic
		if ic.X > maxc.X {
			maxc.X = ic.X
		}
		if ic.Y > maxc.Y {
			maxc.Y = ic.Y
		}
		idx++
		return tree.Continue
	})
	maxc.X++
	maxc.Y++
	li.Shape = maxc
}

// UpdateStackedVisbility updates the visibility for Stacked layouts
// so the StackTop widget is visible, and others are Invisible.
func (ly *Frame) UpdateStackedVisibility() {
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwb.SetState(i != ly.StackTop, states.Invisible)
		kwb.Geom.Cell = image.Point{0, 0}
		return tree.Continue
	})
}

func (ly *Frame) LaySetInitCellsStacked() {
	ly.UpdateStackedVisibility()
	ly.Layout.Shape = image.Point{1, 1}
}

func (ly *Frame) LaySetInitCellsGrid() {
	n := len(ly.Children)
	cols := ly.Styles.Columns
	if cols == 0 {
		cols = int(math32.Sqrt(float32(n)))
	}
	rows := n / cols
	for rows*cols < n {
		rows++
	}
	if rows == 0 || cols == 0 {
		fmt.Println(ly, "no rows or cols:", rows, cols)
	}
	ly.Layout.Shape = image.Point{max(cols, 1), max(rows, 1)}
	ci := 0
	ri := 0
	ly.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
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
		return tree.Continue
	})
}

//////////////////////////////////////////////////////////////////////
//		SizeFromChildren

// SizeFromChildrenFit gathers Actual size from kids, and calls LaySetContentFitOverflow
// to update Actual and Internal size based on this.
func (ly *Frame) SizeFromChildrenFit(iter int, pass LayoutPasses) {
	ksz := ly.This.(Layouter).SizeFromChildren(iter, SizeDownPass)
	ly.LaySetContentFitOverflow(ksz, pass)
	if DebugSettings.LayoutTrace {
		sz := &ly.Geom.Size
		fmt.Println(ly, pass, "FromChildren:", ksz, "Content:", sz.Actual.Content, "Internal:", sz.Internal)
	}
}

// SizeFromChildren gathers Actual size from kids.
// Different Layout types can alter this to present different Content
// sizes for the layout process, e.g., if Content is sized to fit allocation,
// as in the TopAppBar and List types.
func (ly *Frame) SizeFromChildren(iter int, pass LayoutPasses) math32.Vector2 {
	var ksz math32.Vector2
	if ly.Styles.Display == styles.Stacked {
		ksz = ly.SizeFromChildrenStacked()
	} else {
		ksz = ly.SizeFromChildrenCells(iter, pass)
	}
	return ksz
}

// SizeFromChildrenCells for Flex, Grid
func (ly *Frame) SizeFromChildrenCells(iter int, pass LayoutPasses) math32.Vector2 {
	// r   0   1   col X = max(X over rows), Y = sum(Y over rows)
	//   +--+--+
	// 0 |  |  |   row X = sum(X over cols), Y = max(Y over cols)
	//   +--+--+
	// 1 |  |  |
	//   +--+--+
	li := &ly.Layout
	li.InitCells()
	ly.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Geom.Cell
		sz := kwb.Geom.Size.Actual.Total
		grw := kwb.Styles.Grow
		if pass <= SizeDownPass && iter == 0 && kwb.Styles.GrowWrap {
			grw.Set(1, 0)
		}
		if DebugSettings.LayoutTraceDetail {
			fmt.Println("SzUp i:", i, kwb, "cidx:", cidx, "sz:", sz, "grw:", grw)
		}
		for ma := math32.X; ma <= math32.Y; ma++ { // main axis = X then Y
			ca := ma.Other()                // cross axis = Y then X
			mi := math32.PointDim(cidx, ma) // X, Y
			ci := math32.PointDim(cidx, ca) // Y, X

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
		return tree.Continue
	})
	if DebugSettings.LayoutTraceDetail {
		fmt.Println(ly, "SizeFromChildren")
		fmt.Println(li.String())
	}
	ksz := li.CellsSize()
	return ksz
}

// SizeFromChildrenStacked for stacked case
func (ly *Frame) SizeFromChildrenStacked() math32.Vector2 {
	ly.Layout.InitCells()
	_, kwb := ly.StackTopWidget()
	li := &ly.Layout
	var ksz math32.Vector2
	if kwb != nil {
		ksz = kwb.Geom.Size.Actual.Total
		kgrw := kwb.Styles.Grow
		for ma := math32.X; ma <= math32.Y; ma++ { // main axis = X then Y
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
// Flexible elements (e.g., Flex Wrap layouts and Text with word wrap)
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
			fmt.Println(wb, "SizeDownChildren child:", kwb.Name, "triggered redo")
		}
		redo = redo || re
		return tree.Continue
	})
	return redo
}

// SizeDownChildren calls SizeDown on the Children.
// The kids must have their Size.Alloc set prior to this, which
// is what Layout type does.  Other special widget types can
// do custom layout and call this too.
func (ly *Frame) SizeDownChildren(iter int) bool {
	if ly.Styles.Display == styles.Stacked && !ly.LayoutStackTopOnly {
		redo := false
		ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			re := kwi.SizeDown(iter)
			if i == ly.StackTop {
				redo = redo || re
			}
			return tree.Continue
		})
		return redo
	}
	return ly.WidgetBase.SizeDownChildren(iter)
}

// GrowToAllocSize returns the potential size that widget could grow,
// for any dimension with a non-zero Grow factor.
// If Grow is < 1, then the size is increased in proportion, but
// any factor > 1 produces a full fill along that dimension.
// Returns true if this resulted in a change.
func (wb *WidgetBase) GrowToAllocSize(act, alloc math32.Vector2) (math32.Vector2, bool) {
	change := false
	for d := math32.X; d <= math32.Y; d++ {
		grw := wb.Styles.Grow.Dim(d)
		allocd := alloc.Dim(d)
		actd := act.Dim(d)
		if grw > 0 && allocd > actd {
			grw := min(1, grw)
			nsz := math32.Ceil(actd + grw*(allocd-actd))
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

func (ly *Frame) SizeDown(iter int) bool {
	redo := ly.SizeDownLay(iter)
	if redo && DebugSettings.LayoutTrace {
		fmt.Println(ly, "SizeDown redo")
	}
	return redo
}

// SizeDownLay is the Layout standard SizeDown pass, returning true if another
// iteration is required.  It allocates sizes to fit given parent-allocated
// total size.
func (ly *Frame) SizeDownLay(iter int) bool {
	if !ly.HasChildren() || !ly.Layout.ShapeCheck(ly, "SizeDown") {
		return ly.SizeDownWidget(iter) // behave like a widget
	}
	sz := &ly.Geom.Size
	styles.SetClampMaxVector(&sz.Alloc.Content, sz.Max) // can't be more than max..
	sz.SetTotalFromContent(&sz.Alloc)
	if DebugSettings.LayoutTrace {
		fmt.Println(ly, "Managing Alloc:", sz.Alloc.Content)
	}
	chg := ly.This.(Layouter).ManageOverflow(iter, true) // this must go first.
	wrapped := false
	if iter <= 1 && ly.Styles.IsFlexWrap() {
		wrapped = ly.SizeDownWrap(iter) // first recompute wrap
		if iter == 0 {
			wrapped = true // always update
		}
	}
	ly.This.(Layouter).SizeDownSetAllocs(iter)
	redo := ly.SizeDownChildren(iter)
	if redo || wrapped {
		ly.SizeFromChildrenFit(iter, SizeDownPass)
	}
	ly.SizeDownParts(iter) // no std role, just get sizes
	return chg || wrapped || redo
}

// SizeDownSetAllocs is the key SizeDown step that sets the allocations
// in the children, based on our allocation.  In the default implementation
// this calls SizeDownGrow if there is extra space to grow, or
// SizeDownAllocActual to set the allocations as they currrently are.
func (ly *Frame) SizeDownSetAllocs(iter int) {
	sz := &ly.Geom.Size
	extra := sz.Alloc.Content.Sub(sz.Internal) // note: critical to use internal to be accurate
	if extra.X > 0 || extra.Y > 0 {
		if DebugSettings.LayoutTrace {
			fmt.Println(ly, "SizeDown extra:", extra, "Internal:", sz.Internal, "Alloc:", sz.Alloc.Content)
		}
		ly.SizeDownGrow(iter, extra)
	} else {
		ly.SizeDownAllocActual(iter) // set allocations as is
	}
}

// ManageOverflow uses overflow settings to determine if scrollbars
// are needed (Internal > Alloc).  Returns true if size changes as a result.
// If updateSize is false, then the Actual and Alloc sizes are NOT
// updated as a result of change from adding scrollbars
// (generally should be true, but some cases not)
func (ly *Frame) ManageOverflow(iter int, updateSize bool) bool {
	sz := &ly.Geom.Size
	sbw := math32.Ceil(ly.Styles.ScrollBarWidth.Dots)
	change := false
	if iter == 0 {
		ly.Layout.ScrollSize.SetZero()
		ly.SetScrollsOff()
		for d := math32.X; d <= math32.Y; d++ {
			if ly.Styles.Overflow.Dim(d) == styles.OverflowScroll {
				if !ly.HasScroll[d] {
					change = true
				}
				ly.HasScroll[d] = true
				ly.Layout.ScrollSize.SetDim(d.Other(), sbw)
			}
		}
	}
	for d := math32.X; d <= math32.Y; d++ {
		maxSize, visSize, _ := ly.This.(Layouter).ScrollValues(d)
		ofd := maxSize - visSize
		switch ly.Styles.Overflow.Dim(d) {
		// case styles.OverflowVisible:
		// note: this shouldn't happen -- just have this in here for monitoring
		// fmt.Println(ly, "OverflowVisible ERROR -- shouldn't have overflow:", d, ofd)
		case styles.OverflowAuto:
			if ofd <= 1 {
				if ly.HasScroll[d] {
					if DebugSettings.LayoutTrace {
						fmt.Println(ly, "turned off scroll", d)
					}
					change = true
					ly.HasScroll[d] = false
					ly.Layout.ScrollSize.SetDim(d.Other(), 0)
				}
				continue
			}
			if !ly.HasScroll[d] {
				change = true
			}
			ly.HasScroll[d] = true
			ly.Layout.ScrollSize.SetDim(d.Other(), sbw)
			if change && DebugSettings.LayoutTrace {
				fmt.Println(ly, "OverflowAuto enabling scrollbars for dim for overflow:", d, ofd, "alloc:", sz.Alloc.Content.Dim(d), "internal:", sz.Internal.Dim(d))
			}
		}
	}
	ly.This.(Layouter).LayoutSpace() // adds the scroll space
	if updateSize {
		sz.SetTotalFromContent(&sz.Actual)
		sz.SetContentFromTotal(&sz.Alloc) // alloc is *decreased* from any increase in space
	}
	if change && DebugSettings.LayoutTrace {
		fmt.Println(ly, "ManageOverflow changed")
	}
	return change
}

// SizeDownGrow grows the element sizes based on total extra and Grow
func (ly *Frame) SizeDownGrow(iter int, extra math32.Vector2) bool {
	redo := false
	if ly.Styles.Display == styles.Stacked {
		redo = ly.SizeDownGrowStacked(iter, extra)
	} else {
		redo = ly.SizeDownGrowCells(iter, extra)
	}
	return redo
}

func (ly *Frame) SizeDownGrowCells(iter int, extra math32.Vector2) bool {
	redo := false
	sz := &ly.Geom.Size
	alloc := sz.Alloc.Content.Sub(sz.InnerSpace) // inner is fixed
	// todo: use max growth values instead of individual ones to ensure consistency!
	li := &ly.Layout
	if len(li.Cells) == 0 {
		slog.Error("unexpected error: layout has not been initialized", "layout", ly.String())
		return false
	}
	ly.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Geom.Cell
		ksz := &kwb.Geom.Size
		grw := kwb.Styles.Grow
		if iter == 0 && kwb.Styles.GrowWrap {
			grw.Set(1, 0)
		}
		// if DebugSettings.LayoutTrace {
		// 	fmt.Println("szdn i:", i, kwb, "cidx:", cidx, "sz:", sz, "grw:", grw)
		// }
		for ma := math32.X; ma <= math32.Y; ma++ { // main axis = X then Y
			gr := grw.Dim(ma)
			ca := ma.Other()     // cross axis = Y then X
			exd := extra.Dim(ma) // row.X = extra width for cols; col.Y = extra height for rows in this col
			if exd < 0 {
				exd = 0
			}
			mi := math32.PointDim(cidx, ma) // X, Y
			ci := math32.PointDim(cidx, ca) // Y, X
			md := li.Cell(ma, mi, ci)       // X, Y
			cd := li.Cell(ca, ci, mi)       // Y, X
			if md == nil || cd == nil {
				break
			}
			mx := md.Size.Dim(ma)
			asz := mx
			gsum := cd.Grow.Dim(ma)
			if gsum > 0 && exd > 0 {
				if gr > gsum {
					fmt.Println(ly, "SizeDownGrowCells error: grow > grow sum:", gr, gsum)
					gr = gsum
				}
				redo = true
				asz = math32.Round(mx + exd*(gr/gsum))
				styles.SetClampMax(&asz, ksz.Max.Dim(ma))
				if asz > math32.Ceil(alloc.Dim(ma))+1 { // bug!
					fmt.Println(ly, "SizeDownGrowCells error: sub alloc > total to alloc:", asz, alloc.Dim(ma))
					fmt.Println("ma:", ma, "mi:", mi, "ci:", ci, "mx:", mx, "gsum:", gsum, "gr:", gr, "ex:", exd, "par act:", sz.Actual.Content.Dim(ma))
					fmt.Println(ly.Layout.String())
					fmt.Println(ly.Layout.CellsSize())
				}
			}
			if DebugSettings.LayoutTraceDetail {
				fmt.Println(kwb, ma, "alloc:", asz, "was act:", sz.Actual.Total.Dim(ma), "mx:", mx, "gsum:", gsum, "gr:", gr, "ex:", exd)
			}
			ksz.Alloc.Total.SetDim(ma, asz)
		}
		ksz.SetContentFromTotal(&ksz.Alloc)
		return tree.Continue
	})
	return redo
}

func (ly *Frame) SizeDownWrap(iter int) bool {
	wrapped := false
	li := &ly.Layout
	sz := &ly.Geom.Size
	d := li.MainAxis
	alloc := sz.Alloc.Content
	gap := li.Gap.Dim(d)
	fit := alloc.Dim(d)
	if DebugSettings.LayoutTrace {
		fmt.Println(ly, "SizeDownWrap fitting into:", d, fit)
	}
	first := true
	var sum float32
	var n int
	var wraps []int
	ly.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		ksz := kwb.Geom.Size.Actual.Total
		if first {
			n = 1
			sum = ksz.Dim(d) + gap
			first = false
			return tree.Continue
		}
		if sum+ksz.Dim(d)+gap >= fit {
			if DebugSettings.LayoutTraceDetail {
				fmt.Println(ly, "wrapped:", i, sum, ksz.Dim(d), fit)
			}
			wraps = append(wraps, n)
			sum = ksz.Dim(d)
			n = 1 // this guy is on next line
		} else {
			sum += ksz.Dim(d) + gap
			n++
		}
		return tree.Continue
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
		fmt.Println(ly, "wrapped:", wraps)
	}
	li.Wraps = wraps
	ly.LaySetWrapIndexes()
	li.InitCells()
	ly.LaySetGapSizeFromCells()
	ly.SizeFromChildrenCells(iter, SizeDownPass)
	return wrapped
}

func (ly *Frame) SizeDownGrowStacked(iter int, extra math32.Vector2) bool {
	// stack just gets everything from us
	chg := false
	asz := ly.Geom.Size.Alloc.Content
	// todo: could actually use the grow factors to decide if growing here?
	if ly.LayoutStackTopOnly {
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
		return tree.Continue
	})
	return chg
}

// SizeDownAllocActual sets Alloc to Actual for no-extra case.
func (ly *Frame) SizeDownAllocActual(iter int) {
	if ly.Styles.Display == styles.Stacked {
		ly.SizeDownAllocActualStacked(iter)
		return
	}
	// todo: wrap needs special case
	ly.SizeDownAllocActualCells(iter)
}

// SizeDownAllocActualCells sets Alloc to Actual for no-extra case.
// Note however that due to max sizing for row / column,
// this size can actually be different than original actual.
func (ly *Frame) SizeDownAllocActualCells(iter int) {
	ly.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		ksz := &kwb.Geom.Size
		cidx := kwb.Geom.Cell
		for ma := math32.X; ma <= math32.Y; ma++ { // main axis = X then Y
			ca := ma.Other()                 // cross axis = Y then X
			mi := math32.PointDim(cidx, ma)  // X, Y
			ci := math32.PointDim(cidx, ca)  // Y, X
			md := ly.Layout.Cell(ma, mi, ci) // X, Y
			asz := md.Size.Dim(ma)
			ksz.Alloc.Total.SetDim(ma, asz)
		}
		ksz.SetContentFromTotal(&ksz.Alloc)
		return tree.Continue
	})
}

func (ly *Frame) SizeDownAllocActualStacked(iter int) {
	// stack just gets everything from us
	asz := ly.Geom.Size.Actual.Content
	// todo: could actually use the grow factors to decide if growing here?
	if ly.LayoutStackTopOnly {
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
		return tree.Continue
	})
}

//////////////////////////////////////////////////////////////////////
//		SizeFinal

// SizeFinalUpdateChildrenSizes can optionally be called for layouts
// that dynamically create child elements based on final layout size.
// It ensures that the children are properly sized.
func (ly *Frame) SizeFinalUpdateChildrenSizes() {
	ly.SizeUpLay()
	iter := 3 // late stage..
	ly.This.(Layouter).SizeDownSetAllocs(iter)
	ly.SizeDownChildren(iter)
	ly.SizeDownParts(iter) // no std role, just get sizes
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
	if (wb.Scene != nil && wb.Scene.prefSizing) || wb.Styles.GrowWrap {
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

func (ly *Frame) SizeFinal() {
	ly.SizeFinalLay()
}

// SizeFinalLay is the Layout standard SizeFinal pass
func (ly *Frame) SizeFinalLay() {
	if !ly.HasChildren() || !ly.Layout.ShapeCheck(ly, "SizeFinal") {
		ly.SizeFinalWidget() // behave like a widget
		return
	}
	ly.Geom.RelPos.SetZero()
	ly.SizeFinalChildren() // kids do their own thing
	ly.SizeFromChildrenFit(0, SizeFinalPass)
	ly.GrowToAlloc()
	ly.StyleSizeUpdate() // now that sizes are stable, ensure styling based on size is updated
	ly.SizeFinalParts()
}

// SizeFinalChildren calls SizeFinal on all the children of this node
func (wb *WidgetBase) SizeFinalChildren() {
	wb.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.SizeFinal()
		return tree.Continue
	})
}

// SizeFinalChildren calls SizeFinal on all the children of this node
func (ly *Frame) SizeFinalChildren() {
	if ly.Styles.Display == styles.Stacked && !ly.LayoutStackTopOnly {
		ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			kwi.SizeFinal()
			return tree.Continue
		})
		return
	}
	ly.WidgetBase.SizeFinalChildren()
}

// StyleSizeUpdate updates styling size values for widget and its parent,
// which should be called after these are updated.  Returns true if any changed.
func (wb *WidgetBase) StyleSizeUpdate() bool {
	return false
	// TODO(kai): this seems to break parent-relative units instead of making them work
	/*
		el := wb.Geom.Size.Alloc.Content
		var parent math32.Vector2
		pwb := wb.ParentWidget()
		if pwb != nil {
			parent = pwb.Geom.Size.Alloc.Content
		}
		sz := wb.Scene.SceneGeom.Size
		chg := wb.Styles.UnitContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, parent.X, parent.Y)
		if chg {
			wb.Styles.ToDots()
		}
		return chg
	*/
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

func (wb *WidgetBase) PositionWithinAllocMainX(pos math32.Vector2, parJustify, parAlign styles.Aligns) {
	sz := &wb.Geom.Size
	pos.X += styles.AlignPos(styles.ItemAlign(parJustify, wb.Styles.Justify.Self), sz.Actual.Total.X, sz.Alloc.Total.X)
	pos.Y += styles.AlignPos(styles.ItemAlign(parAlign, wb.Styles.Align.Self), sz.Actual.Total.Y, sz.Alloc.Total.Y)
	wb.Geom.RelPos = pos
	if DebugSettings.LayoutTrace {
		fmt.Println(wb, "Position within Main=X:", pos)
	}
}

func (wb *WidgetBase) PositionWithinAllocMainY(pos math32.Vector2, parJustify, parAlign styles.Aligns) {
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
	wb.Parts.This.(Widget).Position()
}

// PositionChildren runs Position on the children
func (wb *WidgetBase) PositionChildren() {
	wb.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.Position()
		return tree.Continue
	})
}

// Position: uses the final sizes to position everything within layouts
// according to alignment settings.
func (ly *Frame) Position() {
	ly.PositionLay()
}

func (ly *Frame) PositionLay() {
	if !ly.HasChildren() || !ly.Layout.ShapeCheck(ly, "Position") {
		ly.PositionWidget() // behave like a widget
		return
	}
	if ly.Parent == nil {
		ly.PositionWithinAllocMainY(math32.Vector2{}, ly.Styles.Justify.Items, ly.Styles.Align.Items)
	}
	ly.ConfigScrolls() // and configure the scrolls
	if ly.Styles.Display == styles.Stacked {
		ly.PositionStacked()
	} else {
		ly.PositionCells()
		ly.PositionChildren()
	}
	ly.PositionParts()
}

func (ly *Frame) PositionCells() {
	if ly.Styles.Display == styles.Flex && ly.Styles.Direction == styles.Column {
		ly.PositionCellsMainY()
		return
	}
	ly.PositionCellsMainX()
}

// Main axis = X
func (ly *Frame) PositionCellsMainX() {
	// todo: can break apart further into Flex rows
	gap := ly.Layout.Gap
	sz := &ly.Geom.Size
	if DebugSettings.LayoutTraceDetail {
		fmt.Println(ly, "PositionCells Main X, alloc:", sz.Alloc.Content, "internal:", sz.Internal)
	}
	var stPos math32.Vector2
	stPos.X = styles.AlignPos(ly.Styles.Justify.Content, sz.Internal.X, sz.Alloc.Content.X)
	stPos.Y = styles.AlignPos(ly.Styles.Align.Content, sz.Internal.Y, sz.Alloc.Content.Y)
	pos := stPos
	var lastSz math32.Vector2
	idx := 0
	ly.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Geom.Cell
		if cidx.X == 0 && idx > 0 {
			pos.X = stPos.X
			pos.Y += lastSz.Y + gap.Y
		}
		kwb.PositionWithinAllocMainX(pos, ly.Styles.Justify.Items, ly.Styles.Align.Items)
		alloc := kwb.Geom.Size.Alloc.Total
		pos.X += alloc.X + gap.X
		lastSz = alloc
		idx++
		return tree.Continue
	})
}

// Main axis = Y
func (ly *Frame) PositionCellsMainY() {
	gap := ly.Layout.Gap
	sz := &ly.Geom.Size
	if DebugSettings.LayoutTraceDetail {
		fmt.Println(ly, "PositionCells, alloc:", sz.Alloc.Content, "internal:", sz.Internal)
	}
	var lastSz math32.Vector2
	var stPos math32.Vector2
	stPos.Y = styles.AlignPos(ly.Styles.Justify.Content, sz.Internal.Y, sz.Alloc.Content.Y)
	stPos.X = styles.AlignPos(ly.Styles.Align.Content, sz.Internal.X, sz.Alloc.Content.X)
	pos := stPos
	idx := 0
	ly.VisibleKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		cidx := kwb.Geom.Cell
		if cidx.Y == 0 && idx > 0 {
			pos.Y = stPos.Y
			pos.X += lastSz.X + gap.X
		}
		kwb.PositionWithinAllocMainY(pos, ly.Styles.Justify.Items, ly.Styles.Align.Items)
		alloc := kwb.Geom.Size.Alloc.Total
		pos.Y += alloc.Y + gap.Y
		lastSz = alloc
		idx++
		return tree.Continue
	})
}

func (ly *Frame) PositionWrap() {
}

func (ly *Frame) PositionStacked() {
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwb.Geom.RelPos.SetZero()
		if !ly.LayoutStackTopOnly || i == ly.StackTop {
			kwi.Position()
		}
		return tree.Continue
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
	var parPos math32.Vector2
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
		wb.Geom.TotalBBox = math32.RectFromPosSizeMax(math32.Vector2{}, sz.Alloc.Total)
		off := wb.Styles.BoxSpace().Pos().Floor()
		wb.Geom.ContentBBox = math32.RectFromPosSizeMax(off, sz.Alloc.Content)
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
		return tree.Continue
	})
}

// ScenePosChildren runs ScenePos on the children
func (ly *Frame) ScenePosChildren() {
	if ly.Styles.Display == styles.Stacked && !ly.LayoutStackTopOnly {
		ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			kwi.ScenePos()
			return tree.Continue
		})
		return
	}
	ly.WidgetBase.ScenePosChildren()
}

// ScenePos: scene-based position and final BBox is computed based on
// parents accumulated position and scrollbar position.
// This step can be performed when scrolling after updating Scroll.
func (ly *Frame) ScenePos() {
	ly.ScenePosLay()
}

// ScrollResetIfNone resets the scroll offsets if there are no scrollbars
func (ly *Frame) ScrollResetIfNone() {
	for d := math32.X; d <= math32.Y; d++ {
		if !ly.HasScroll[d] {
			ly.Geom.Scroll.SetDim(d, 0)
		}
	}
}

func (ly *Frame) ScenePosLay() {
	ly.ScrollResetIfNone()
	// note: ly.Geom.Scroll has the X, Y scrolling offsets, set by Layouter.ScrollChanged function
	if !ly.HasChildren() || !ly.Layout.ShapeCheck(ly, "ScenePos") {
		ly.ScenePosWidget() // behave like a widget
		return
	}
	ly.ScenePosWidget()
	ly.ScenePosChildren()
	ly.PositionScrolls()
	ly.ScenePosParts() // in case they fit inside parent
	// otherwise handle separately like scrolls on layout
}
