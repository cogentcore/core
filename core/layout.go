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
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// LayoutPasses is used for the SizeFromChildren method,
// which can potentially compute different sizes for different passes.
type LayoutPasses int32 //enums:enum

const (
	SizeUpPass LayoutPasses = iota
	SizeDownPass
	SizeFinalPass
)

// Layouter is an interface containing layout functions
// implemented by all types embedding [Frame].
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
	// as in the [Toolbar] and [List] types.
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

// AsFrame returns the given value as a [Frame] if it implements [Layouter]
// or nil otherwise.
func AsFrame(n tree.Node) *Frame {
	if t, ok := n.(Layouter); ok {
		return t.AsFrame()
	}
	return nil
}

func (t *Frame) AsFrame() *Frame {
	return t
}

var _ Layouter = &Frame{}

// geomCT has core layout elements: Content and Total
type geomCT struct {
	// Content is for the contents (children, parts) of the widget,
	// excluding the Space (margin, padding, scrollbars).
	// This content includes the InnerSpace factor (Gaps in Layout)
	// which must therefore be subtracted when allocating down to children.
	Content math32.Vector2

	// Total is for the total exterior of the widget: Content + Space
	Total math32.Vector2
}

func (ct geomCT) String() string {
	return fmt.Sprintf("Content: %v, \tTotal: %v", ct.Content, ct.Total)
}

// geomSize has all of the relevant Layout sizes
type geomSize struct {
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
	Actual geomCT `display:"inline"`

	// Alloc is the top-down allocated size, based on available visible space,
	// starting from the Scene geometry and working downward, attempting to
	// accommodate the Actual contents, and allocating extra space based on
	// Grow factors.  When Actual < Alloc, alignment factors determine positioning
	// within the allocated space.
	Alloc geomCT `display:"inline"`

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

func (ls geomSize) String() string {
	return fmt.Sprintf("Actual: %v, \tAlloc: %v", ls.Actual, ls.Alloc)
}

// setInitContentMin sets initial Actual.Content size from given Styles.Min,
// further subject to the current Max constraint.
func (ls *geomSize) setInitContentMin(styMin math32.Vector2) {
	csz := &ls.Actual.Content
	*csz = styMin
	styles.SetClampMaxVector(csz, ls.Max)
}

// FitSizeMax increases given size to fit given fm value, subject to Max constraints
func (ls *geomSize) FitSizeMax(to *math32.Vector2, fm math32.Vector2) {
	styles.SetClampMinVector(to, fm)
	styles.SetClampMaxVector(to, ls.Max)
}

// setTotalFromContent sets the Total size as Content plus Space
func (ls *geomSize) setTotalFromContent(ct *geomCT) {
	ct.Total = ct.Content.Add(ls.Space)
}

// setContentFromTotal sets the Content from Total size,
// subtracting Space
func (ls *geomSize) setContentFromTotal(ct *geomCT) {
	ct.Content = ct.Total.Sub(ls.Space)
}

// geomState contains the the layout geometry state for each widget.
// Set by the parent Layout during the Layout process.
type geomState struct {
	// Size has sizing data for the widget: use Actual for rendering.
	// Alloc shows the potentially larger space top-down allocated.
	Size geomSize `display:"add-fields"`

	// Pos is position within the overall Scene that we render into,
	// including effects of scroll offset, for both Total outer dimension
	// and inner Content dimension.
	Pos geomCT `display:"inline" edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

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

func (ls *geomState) String() string {
	return "Size: " + ls.Size.String() + "\nPos: " + ls.Pos.String() + "\tCell: " + ls.Cell.String() +
		"\tRelPos: " + ls.RelPos.String() + "\tScroll: " + ls.Scroll.String()
}

// contentRangeDim returns the Content bounding box min, max
// along given dimension
func (ls *geomState) contentRangeDim(d math32.Dims) (cmin, cmax float32) {
	cmin = float32(math32.PointDim(ls.ContentBBox.Min, d))
	cmax = float32(math32.PointDim(ls.ContentBBox.Max, d))
	return
}

// totalRect returns Pos.Total -- Size.Actual.Total
// as an image.Rectangle, e.g., for bounding box
func (ls *geomState) totalRect() image.Rectangle {
	return math32.RectFromPosSizeMax(ls.Pos.Total, ls.Size.Actual.Total)
}

// contentRect returns Pos.Content, Size.Actual.Content
// as an image.Rectangle, e.g., for bounding box.
func (ls *geomState) contentRect() image.Rectangle {
	return math32.RectFromPosSizeMax(ls.Pos.Content, ls.Size.Actual.Content)
}

// ScrollOffset computes the net scrolling offset as a function of
// the difference between the allocated position and the actual
// content position according to the clipped bounding box.
func (ls *geomState) ScrollOffset() image.Point {
	return ls.ContentBBox.Min.Sub(ls.Pos.Content.ToPoint())
}

// layoutCell holds the layout implementation data for col, row Cells
type layoutCell struct {

	// Size has the Actual size of elements (not Alloc)
	Size math32.Vector2

	// Grow has the Grow factors
	Grow math32.Vector2
}

func (ls *layoutCell) String() string {
	return fmt.Sprintf("Size: %v, \tGrow: %g", ls.Size, ls.Grow)
}

func (ls *layoutCell) reset() {
	ls.Size.SetZero()
	ls.Grow.SetZero()
}

// layoutCells holds one set of LayoutCell cell elements for rows, cols.
// There can be multiple of these for Wrap case.
type layoutCells struct {

	// Shape is number of cells along each dimension for our ColRow cells,
	Shape image.Point `edit:"-"`

	// ColRow has the data for the columns in [0] and rows in [1]:
	// col Size.X = max(X over rows) (cross axis), .Y = sum(Y over rows) (main axis for col)
	// row Size.X = sum(X over cols) (main axis for row), .Y = max(Y over cols) (cross axis)
	// see: https://docs.google.com/spreadsheets/d/1eimUOIJLyj60so94qUr4Buzruj2ulpG5o6QwG2nyxRw/edit?usp=sharing
	ColRow [2][]layoutCell `edit:"-"`
}

// cell returns the cell for given dimension and index along that
// dimension (X = Cols, idx = col, Y = Rows, idx = row)
func (lc *layoutCells) cell(d math32.Dims, idx int) *layoutCell {
	if len(lc.ColRow[d]) <= idx {
		return nil
	}
	return &(lc.ColRow[d][idx])
}

// init initializes Cells for given shape
func (lc *layoutCells) init(shape image.Point) {
	lc.Shape = shape
	for d := math32.X; d <= math32.Y; d++ {
		n := math32.PointDim(lc.Shape, d)
		if len(lc.ColRow[d]) != n {
			lc.ColRow[d] = make([]layoutCell, n)
		}
		for i := 0; i < n; i++ {
			lc.ColRow[d][i].reset()
		}
	}
}

// cellsSize returns the total Size represented by the current Cells,
// which is the Sum of the Max values along each dimension.
func (lc *layoutCells) cellsSize() math32.Vector2 {
	var ksz math32.Vector2
	for ma := math32.X; ma <= math32.Y; ma++ { // main axis = X then Y
		n := math32.PointDim(lc.Shape, ma) // cols, rows
		sum := float32(0)
		for mi := 0; mi < n; mi++ {
			md := lc.cell(ma, mi) // X, Y
			mx := md.Size.Dim(ma)
			sum += mx // sum of maxes
		}
		ksz.SetDim(ma, sum)
	}
	return ksz.Ceil()
}

// gapSizeDim returns the gap size for given dimension, based on Shape and given gap size
func (lc *layoutCells) gapSizeDim(d math32.Dims, gap float32) float32 {
	n := math32.PointDim(lc.Shape, d)
	return float32(n-1) * gap
}

func (lc *layoutCells) String() string {
	s := ""
	n := lc.Shape.X
	for i := 0; i < n; i++ {
		col := lc.cell(math32.X, i)
		s += fmt.Sprintln("col:", i, "\tmax X:", col.Size.X, "\tsum Y:", col.Size.Y, "\tmax grX:", col.Grow.X, "\tsum grY:", col.Grow.Y)
	}
	n = lc.Shape.Y
	for i := 0; i < n; i++ {
		row := lc.cell(math32.Y, i)
		s += fmt.Sprintln("row:", i, "\tsum X:", row.Size.X, "\tmax Y:", row.Size.Y, "\tsum grX:", row.Grow.X, "\tmax grY:", row.Grow.Y)
	}
	return s
}

// layoutState has internal state for implementing layout
type layoutState struct {
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
	Cells []layoutCells `edit:"-"`

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

// initCells initializes the Cells based on Shape, MainAxis, and Wraps
// which must be set before calling.
func (ls *layoutState) initCells() {
	if ls.Wraps == nil {
		if len(ls.Cells) != 1 {
			ls.Cells = make([]layoutCells, 1)
		}
		ls.Cells[0].init(ls.Shape)
		return
	}
	ma := ls.MainAxis
	ca := ma.Other()
	nw := len(ls.Wraps)
	if len(ls.Cells) != nw {
		ls.Cells = make([]layoutCells, nw)
	}
	for wi, wn := range ls.Wraps {
		var shp image.Point
		math32.SetPointDim(&shp, ma, wn)
		math32.SetPointDim(&shp, ca, 1)
		ls.Cells[wi].init(shp)
	}
}

func (ls *layoutState) shapeCheck(w Widget, phase string) bool {
	if w.AsTree().HasChildren() && (ls.Shape == (image.Point{}) || len(ls.Cells) == 0) {
		// fmt.Println(w, "Shape is nil in:", phase) // TODO: plan for this?
		return false
	}
	return true
}

// cell returns the cell for given dimension and index along that
// dimension, and given other-dimension axis which is ignored for non-Wrap cases.
// Does no range checking and will crash if out of bounds.
func (ls *layoutState) cell(d math32.Dims, dIndex, odIndex int) *layoutCell {
	if ls.Wraps == nil {
		return ls.Cells[0].cell(d, dIndex)
	}
	if ls.MainAxis == d {
		return ls.Cells[odIndex].cell(d, dIndex)
	}
	return ls.Cells[dIndex].cell(d, 0)
}

// wrapIndexToCoord returns the X,Y coordinates in Wrap case for given sequential idx
func (ls *layoutState) wrapIndexToCoord(idx int) image.Point {
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

// cellsSize returns the total Size represented by the current Cells,
// which is the Sum of the Max values along each dimension within each Cell,
// Maxed over cross-axis dimension for Wrap case, plus GapSize.
func (ls *layoutState) cellsSize() math32.Vector2 {
	if ls.Wraps == nil {
		return ls.Cells[0].cellsSize().Add(ls.GapSize)
	}
	var ksz math32.Vector2
	d := ls.MainAxis
	od := d.Other()
	gap := ls.Gap.Dim(d)
	for wi := range ls.Wraps {
		wsz := ls.Cells[wi].cellsSize()
		wsz.SetDim(d, wsz.Dim(d)+ls.Cells[wi].gapSizeDim(d, gap))
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

// colWidth returns the width of given column for given row index
// (ignored for non-Wrap), with full bounds checking.
// Returns error if out of range.
func (ls *layoutState) colWidth(row, col int) (float32, error) {
	if ls.Wraps == nil {
		n := math32.PointDim(ls.Shape, math32.X)
		if col >= n {
			return 0, fmt.Errorf("Layout.ColWidth: col: %d > number of columns: %d", col, n)
		}
		return ls.cell(math32.X, col, 0).Size.X, nil
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
		return ls.cell(math32.X, col, row).Size.X, nil
	}
	if col >= nw {
		return 0, fmt.Errorf("Layout.ColWidth: col: %d > number of columns: %d", col, nw)
	}
	wn := ls.Wraps[col]
	if row >= wn {
		return 0, fmt.Errorf("Layout.ColWidth: row: %d > number of rows: %d", row, wn)
	}
	return ls.cell(math32.X, col, row).Size.X, nil
}

// rowHeight returns the height of given row for given
// column (ignored for non-Wrap), with full bounds checking.
// Returns error if out of range.
func (ls *layoutState) rowHeight(row, col int) (float32, error) {
	if ls.Wraps == nil {
		n := math32.PointDim(ls.Shape, math32.Y)
		if row >= n {
			return 0, fmt.Errorf("Layout.RowHeight: row: %d > number of rows: %d", row, n)
		}
		return ls.cell(math32.Y, 0, row).Size.Y, nil
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
		return ls.cell(math32.Y, col, row).Size.Y, nil
	}
	if row >= nw {
		return 0, fmt.Errorf("Layout.RowHeight: row: %d > number of rows: %d", row, nw)
	}
	wn := ls.Wraps[col]
	if col >= wn {
		return 0, fmt.Errorf("Layout.RowHeight: col: %d > number of columns: %d", col, wn)
	}
	return ls.cell(math32.Y, row, col).Size.Y, nil
}

func (ls *layoutState) String() string {
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

// StackTopWidget returns the [Frame.StackTop] element as a [WidgetBase].
func (fr *Frame) StackTopWidget() *WidgetBase {
	n := fr.Child(fr.StackTop)
	return AsWidget(n)
}

// laySetContentFitOverflow sets Internal and Actual.Content size to fit given
// new content size, depending on the Styles Overflow: Auto and Scroll types do NOT
// expand Actual and remain at their current styled actual values,
// absorbing the extra content size within their own scrolling zone
// (full size recorded in Internal).
func (fr *Frame) laySetContentFitOverflow(nsz math32.Vector2, pass LayoutPasses) {
	sz := &fr.Geom.Size
	asz := &sz.Actual.Content
	isz := &sz.Internal
	sz.setInitContentMin(sz.Min) // start from style
	*isz = nsz                   // internal is always accurate!
	oflow := &fr.Styles.Overflow
	nosz := pass == SizeUpPass && fr.Styles.IsFlexWrap()
	mx := sz.Max
	for d := math32.X; d <= math32.Y; d++ {
		if nosz {
			continue
		}
		if !(fr.Scene != nil && fr.Scene.hasFlag(sceneContentSizing)) && oflow.Dim(d) >= styles.OverflowAuto && fr.Parent != nil {
			if mx.Dim(d) > 0 {
				asz.SetDim(d, styles.ClampMax(styles.ClampMin(asz.Dim(d), nsz.Dim(d)), mx.Dim(d)))
			}
		} else {
			asz.SetDim(d, styles.ClampMin(asz.Dim(d), nsz.Dim(d)))
		}
	}
	styles.SetClampMaxVector(asz, mx)
	sz.setTotalFromContent(&sz.Actual)
}

// SizeUp (bottom-up) gathers Actual sizes from our Children & Parts,
// based on Styles.Min / Max sizes and actual content sizing
// (e.g., text size).  Flexible elements (e.g., [Text], Flex Wrap,
// [Toolbar]) should reserve the _minimum_ size possible at this stage,
// and then Grow based on SizeDown allocation.
func (wb *WidgetBase) SizeUp() {
	wb.SizeUpWidget()
}

// SizeUpWidget is the standard Widget SizeUp pass
func (wb *WidgetBase) SizeUpWidget() {
	wb.sizeFromStyle()
	wb.sizeUpParts()
	sz := &wb.Geom.Size
	sz.setTotalFromContent(&sz.Actual)
}

// spaceFromStyle sets the Space based on Style BoxSpace().Size()
func (wb *WidgetBase) spaceFromStyle() {
	wb.Geom.Size.Space = wb.Styles.BoxSpace().Size().Ceil()
}

// sizeFromStyle sets the initial Actual Sizes from Style.Min, Max.
// Required first call in SizeUp.
func (wb *WidgetBase) sizeFromStyle() {
	sz := &wb.Geom.Size
	s := &wb.Styles
	wb.spaceFromStyle()
	wb.Geom.Size.InnerSpace.SetZero()
	sz.Min = s.Min.Dots().Ceil()
	sz.Max = s.Max.Dots().Ceil()
	if s.Min.X.Unit == units.UnitPw || s.Min.X.Unit == units.UnitPh {
		sz.Min.X = 0
	}
	if s.Min.Y.Unit == units.UnitPw || s.Min.Y.Unit == units.UnitPh {
		sz.Min.Y = 0
	}
	if s.Max.X.Unit == units.UnitPw || s.Max.X.Unit == units.UnitPh {
		sz.Max.X = 0
	}
	if s.Max.Y.Unit == units.UnitPw || s.Max.Y.Unit == units.UnitPh {
		sz.Max.Y = 0
	}
	sz.Internal.SetZero()
	sz.setInitContentMin(sz.Min)
	sz.setTotalFromContent(&sz.Actual)
	if DebugSettings.LayoutTrace && (sz.Actual.Content.X > 0 || sz.Actual.Content.Y > 0) {
		fmt.Println(wb, "SizeUp from Style:", sz.Actual.Content.String())
	}
}

// updateParentRelSizes updates any parent-relative Min, Max size values
// based on current actual parent sizes.
func (wb *WidgetBase) updateParentRelSizes() bool {
	pwb := wb.parentWidget()
	if pwb == nil {
		return false
	}
	sz := &wb.Geom.Size
	effmin := sz.Min
	s := &wb.Styles
	psz := pwb.Geom.Size.Alloc.Content.Sub(pwb.Geom.Size.InnerSpace)
	got := false
	for d := math32.X; d <= math32.Y; d++ {
		if s.Min.Dim(d).Unit == units.UnitPw {
			got = true
			effmin.SetDim(d, psz.X*0.01*s.Min.Dim(d).Value)
		}
		if s.Min.Dim(d).Unit == units.UnitPh {
			got = true
			effmin.SetDim(d, psz.Y*0.01*s.Min.Dim(d).Value)
		}
		if s.Max.Dim(d).Unit == units.UnitPw {
			got = true
			sz.Max.SetDim(d, psz.X*0.01*s.Max.Dim(d).Value)
		}
		if s.Max.Dim(d).Unit == units.UnitPh {
			got = true
			sz.Max.SetDim(d, psz.Y*0.01*s.Max.Dim(d).Value)
		}
	}
	if got {
		sz.FitSizeMax(&sz.Actual.Total, effmin)
		sz.FitSizeMax(&sz.Alloc.Total, effmin)
		sz.setContentFromTotal(&sz.Actual)
		sz.setContentFromTotal(&sz.Alloc)
	}
	return got
}

// sizeUpParts adjusts the Content size to hold the Parts layout if present
func (wb *WidgetBase) sizeUpParts() {
	if wb.Parts == nil {
		return
	}
	wb.Parts.SizeUp()
	sz := &wb.Geom.Size
	sz.FitSizeMax(&sz.Actual.Content, wb.Parts.Geom.Size.Actual.Total)
}

func (fr *Frame) SizeUp() {
	if fr.Styles.Display == styles.Custom {
		fr.SizeUpWidget()
		fr.sizeUpChildren()
		return
	}
	if !fr.HasChildren() {
		fr.SizeUpWidget() // behave like a widget
		return
	}
	fr.sizeFromStyle()
	fr.layout.ScrollSize.SetZero() // we don't know yet
	fr.setInitCells()
	fr.This.(Layouter).LayoutSpace()
	fr.sizeUpChildren() // kids do their own thing
	fr.sizeFromChildrenFit(0, SizeUpPass)
	if fr.Parts != nil {
		fr.Parts.SizeUp() // just to get sizes -- no std role in layout
	}
}

// LayoutSpace sets our Space based on Styles and Scroll.
// Other layout types can change this if they want to.
func (fr *Frame) LayoutSpace() {
	fr.spaceFromStyle()
	fr.Geom.Size.Space.SetAdd(fr.layout.ScrollSize)
}

// sizeUpChildren calls SizeUp on all the children of this node
func (fr *Frame) sizeUpChildren() {
	if fr.Styles.Display == styles.Stacked && !fr.LayoutStackTopOnly {
		fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
			cw.SizeUp()
			return tree.Continue
		})
		return
	}
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cw.SizeUp()
		return tree.Continue
	})
}

// setInitCells sets the initial default assignment of cell indexes
// to each widget, based on layout type.
func (fr *Frame) setInitCells() {
	switch {
	case fr.Styles.Display == styles.Flex:
		if fr.Styles.Wrap {
			fr.setInitCellsWrap()
		} else {
			fr.setInitCellsFlex()
		}
	case fr.Styles.Display == styles.Stacked:
		fr.setInitCellsStacked()
	case fr.Styles.Display == styles.Grid:
		fr.setInitCellsGrid()
	default:
		fr.setInitCellsStacked() // whatever
	}
	fr.layout.initCells()
	fr.setGapSizeFromCells()
	fr.layout.shapeCheck(fr, "SizeUp")
	// fmt.Println(ly, "SzUp Init", ly.Layout.Shape)
}

func (fr *Frame) setGapSizeFromCells() {
	li := &fr.layout
	li.Gap = fr.Styles.Gap.Dots().Floor()
	// note: this is not accurate for flex
	li.GapSize.X = max(float32(li.Shape.X-1)*li.Gap.X, 0)
	li.GapSize.Y = max(float32(li.Shape.Y-1)*li.Gap.Y, 0)
	fr.Geom.Size.InnerSpace = li.GapSize
}

func (fr *Frame) setInitCellsFlex() {
	li := &fr.layout
	li.MainAxis = math32.Dims(fr.Styles.Direction)
	ca := li.MainAxis.Other()
	li.Wraps = nil
	idx := 0
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		math32.SetPointDim(&cwb.Geom.Cell, li.MainAxis, idx)
		math32.SetPointDim(&cwb.Geom.Cell, ca, 0)
		idx++
		return tree.Continue
	})
	if idx == 0 {
		if DebugSettings.LayoutTrace {
			fmt.Println(fr, "no items:", idx)
		}
	}
	math32.SetPointDim(&li.Shape, li.MainAxis, max(idx, 1)) // must be at least 1
	math32.SetPointDim(&li.Shape, ca, 1)
}

func (fr *Frame) setInitCellsWrap() {
	li := &fr.layout
	li.MainAxis = math32.Dims(fr.Styles.Direction)
	ni := 0
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		ni++
		return tree.Continue
	})
	if ni == 0 {
		li.Shape = image.Point{1, 1}
		li.Wraps = nil
		li.GapSize.SetZero()
		fr.Geom.Size.InnerSpace.SetZero()
		if DebugSettings.LayoutTrace {
			fmt.Println(fr, "no items:", ni)
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
	fr.setWrapIndexes()
}

// setWrapIndexes sets indexes for Wrap case
func (fr *Frame) setWrapIndexes() {
	li := &fr.layout
	idx := 0
	var maxc image.Point
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		ic := li.wrapIndexToCoord(idx)
		cwb.Geom.Cell = ic
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

// UpdateStackedVisibility updates the visibility for Stacked layouts
// so the StackTop widget is visible, and others are Invisible.
func (fr *Frame) UpdateStackedVisibility() {
	fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cwb.SetState(i != fr.StackTop, states.Invisible)
		cwb.Geom.Cell = image.Point{0, 0}
		return tree.Continue
	})
}

func (fr *Frame) setInitCellsStacked() {
	fr.UpdateStackedVisibility()
	fr.layout.Shape = image.Point{1, 1}
}

func (fr *Frame) setInitCellsGrid() {
	n := len(fr.Children)
	cols := fr.Styles.Columns
	if cols == 0 {
		cols = int(math32.Sqrt(float32(n)))
	}
	rows := n / cols
	for rows*cols < n {
		rows++
	}
	if rows == 0 || cols == 0 {
		fmt.Println(fr, "no rows or cols:", rows, cols)
	}
	fr.layout.Shape = image.Point{max(cols, 1), max(rows, 1)}
	ci := 0
	ri := 0
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cwb.Geom.Cell = image.Point{ci, ri}
		ci++
		cs := cwb.Styles.ColSpan
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

// sizeFromChildrenFit gathers Actual size from kids, and calls LaySetContentFitOverflow
// to update Actual and Internal size based on this.
func (fr *Frame) sizeFromChildrenFit(iter int, pass LayoutPasses) {
	ksz := fr.This.(Layouter).SizeFromChildren(iter, pass)
	fr.laySetContentFitOverflow(ksz, pass)
	if DebugSettings.LayoutTrace {
		sz := &fr.Geom.Size
		fmt.Println(fr, pass, "FromChildren:", ksz, "Content:", sz.Actual.Content, "Internal:", sz.Internal)
	}
}

// SizeFromChildren gathers Actual size from kids.
// Different Layout types can alter this to present different Content
// sizes for the layout process, e.g., if Content is sized to fit allocation,
// as in the [Toolbar] and [List] types.
func (fr *Frame) SizeFromChildren(iter int, pass LayoutPasses) math32.Vector2 {
	var ksz math32.Vector2
	if fr.Styles.Display == styles.Stacked {
		ksz = fr.sizeFromChildrenStacked()
	} else {
		ksz = fr.sizeFromChildrenCells(iter, pass)
	}
	return ksz
}

// sizeFromChildrenCells for Flex, Grid
func (fr *Frame) sizeFromChildrenCells(iter int, pass LayoutPasses) math32.Vector2 {
	// r   0   1   col X = max(X over rows), Y = sum(Y over rows)
	//   +--+--+
	// 0 |  |  |   row X = sum(X over cols), Y = max(Y over cols)
	//   +--+--+
	// 1 |  |  |
	//   +--+--+
	li := &fr.layout
	li.initCells()
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cidx := cwb.Geom.Cell
		sz := cwb.Geom.Size.Actual.Total
		grw := cwb.Styles.Grow
		if pass == SizeFinalPass {
			if grw.X == 0 && !cwb.Styles.GrowWrap {
				sz.X = cwb.Geom.Size.Alloc.Total.X
			}
			if grw.Y == 0 {
				sz.Y = cwb.Geom.Size.Alloc.Total.Y
			}
		}
		if pass <= SizeDownPass && iter == 0 && cwb.Styles.GrowWrap {
			grw.Set(1, 0)
		}
		if DebugSettings.LayoutTraceDetail {
			fmt.Println("SzUp i:", i, cwb, "cidx:", cidx, "sz:", sz, "grw:", grw)
		}
		for ma := math32.X; ma <= math32.Y; ma++ { // main axis = X then Y
			ca := ma.Other()                // cross axis = Y then X
			mi := math32.PointDim(cidx, ma) // X, Y
			ci := math32.PointDim(cidx, ca) // Y, X

			md := li.cell(ma, mi, ci) // X, Y
			cd := li.cell(ca, ci, mi) // Y, X
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
		fmt.Println(fr, "SizeFromChildren")
		fmt.Println(li.String())
	}
	ksz := li.cellsSize()
	return ksz
}

// sizeFromChildrenStacked for stacked case
func (fr *Frame) sizeFromChildrenStacked() math32.Vector2 {
	fr.layout.initCells()
	kwb := fr.StackTopWidget()
	li := &fr.layout
	var ksz math32.Vector2
	if kwb != nil {
		ksz = kwb.Geom.Size.Actual.Total
		kgrw := kwb.Styles.Grow
		for ma := math32.X; ma <= math32.Y; ma++ { // main axis = X then Y
			md := li.cell(ma, 0, 0)
			md.Size = ksz
			md.Grow = kgrw
		}
	}
	return ksz
}

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
	wb.updateParentRelSizes()
	redo := wb.sizeDownParts(iter)
	return redo
}

func (wb *WidgetBase) sizeDownParts(iter int) bool {
	if wb.Parts == nil {
		return false
	}
	sz := &wb.Geom.Size
	psz := &wb.Parts.Geom.Size
	pgrow, _ := wb.growToAllocSize(sz.Actual.Content, sz.Alloc.Content)
	psz.Alloc.Total = pgrow // parts = content
	psz.setContentFromTotal(&psz.Alloc)
	redo := wb.Parts.SizeDown(iter)
	if redo && DebugSettings.LayoutTrace {
		fmt.Println(wb, "Parts triggered redo")
	}
	return redo
}

// sizeDownChildren calls SizeDown on the Children.
// The kids must have their Size.Alloc set prior to this, which
// is what Layout type does.  Other special widget types can
// do custom layout and call this too.
func (wb *WidgetBase) sizeDownChildren(iter int) bool {
	redo := false
	wb.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		re := cw.SizeDown(iter)
		if re && DebugSettings.LayoutTrace {
			fmt.Println(wb, "SizeDownChildren child:", cwb.Name, "triggered redo")
		}
		redo = redo || re
		return tree.Continue
	})
	return redo
}

// sizeDownChildren calls SizeDown on the Children.
// The kids must have their Size.Alloc set prior to this, which
// is what Layout type does.  Other special widget types can
// do custom layout and call this too.
func (fr *Frame) sizeDownChildren(iter int) bool {
	if fr.Styles.Display == styles.Stacked && !fr.LayoutStackTopOnly {
		redo := false
		fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
			re := cw.SizeDown(iter)
			if i == fr.StackTop {
				redo = redo || re
			}
			return tree.Continue
		})
		return redo
	}
	return fr.WidgetBase.sizeDownChildren(iter)
}

// growToAllocSize returns the potential size that widget could grow,
// for any dimension with a non-zero Grow factor.
// If Grow is < 1, then the size is increased in proportion, but
// any factor > 1 produces a full fill along that dimension.
// Returns true if this resulted in a change.
func (wb *WidgetBase) growToAllocSize(act, alloc math32.Vector2) (math32.Vector2, bool) {
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

func (fr *Frame) SizeDown(iter int) bool {
	redo := fr.sizeDownFrame(iter)
	if redo && DebugSettings.LayoutTrace {
		fmt.Println(fr, "SizeDown redo")
	}
	return redo
}

// sizeDownFrame is the [Frame] standard SizeDown pass, returning true if another
// iteration is required.  It allocates sizes to fit given parent-allocated
// total size.
func (fr *Frame) sizeDownFrame(iter int) bool {
	if fr.Styles.Display == styles.Custom {
		return fr.sizeDownCustom(iter)
	}
	if !fr.HasChildren() || !fr.layout.shapeCheck(fr, "SizeDown") {
		return fr.WidgetBase.SizeDown(iter) // behave like a widget
	}
	fr.updateParentRelSizes()
	sz := &fr.Geom.Size
	styles.SetClampMaxVector(&sz.Alloc.Content, sz.Max) // can't be more than max..
	sz.setTotalFromContent(&sz.Alloc)
	if DebugSettings.LayoutTrace {
		fmt.Println(fr, "Managing Alloc:", sz.Alloc.Content)
	}
	chg := fr.This.(Layouter).ManageOverflow(iter, true) // this must go first.
	wrapped := false
	if iter <= 1 && fr.Styles.IsFlexWrap() {
		wrapped = fr.sizeDownWrap(iter) // first recompute wrap
		if iter == 0 {
			wrapped = true // always update
		}
	}
	fr.This.(Layouter).SizeDownSetAllocs(iter)
	redo := fr.sizeDownChildren(iter)
	if redo || wrapped {
		fr.sizeFromChildrenFit(iter, SizeDownPass)
	}
	fr.sizeDownParts(iter) // no std role, just get sizes
	return chg || wrapped || redo
}

// SizeDownSetAllocs is the key SizeDown step that sets the allocations
// in the children, based on our allocation.  In the default implementation
// this calls SizeDownGrow if there is extra space to grow, or
// SizeDownAllocActual to set the allocations as they currrently are.
func (fr *Frame) SizeDownSetAllocs(iter int) {
	sz := &fr.Geom.Size
	extra := sz.Alloc.Content.Sub(sz.Internal) // note: critical to use internal to be accurate
	if extra.X > 0 || extra.Y > 0 {
		if DebugSettings.LayoutTrace {
			fmt.Println(fr, "SizeDown extra:", extra, "Internal:", sz.Internal, "Alloc:", sz.Alloc.Content)
		}
		fr.sizeDownGrow(iter, extra)
	} else {
		fr.sizeDownAllocActual(iter) // set allocations as is
	}
}

// ManageOverflow uses overflow settings to determine if scrollbars
// are needed (Internal > Alloc).  Returns true if size changes as a result.
// If updateSize is false, then the Actual and Alloc sizes are NOT
// updated as a result of change from adding scrollbars
// (generally should be true, but some cases not)
func (fr *Frame) ManageOverflow(iter int, updateSize bool) bool {
	sz := &fr.Geom.Size
	sbw := math32.Ceil(fr.Styles.ScrollbarWidth.Dots)
	change := false
	if iter == 0 {
		fr.layout.ScrollSize.SetZero()
		fr.setScrollsOff()
		for d := math32.X; d <= math32.Y; d++ {
			if fr.Styles.Overflow.Dim(d) == styles.OverflowScroll {
				if !fr.HasScroll[d] {
					change = true
				}
				fr.HasScroll[d] = true
				fr.layout.ScrollSize.SetDim(d.Other(), sbw)
			}
		}
	}
	for d := math32.X; d <= math32.Y; d++ {
		maxSize, visSize, _ := fr.This.(Layouter).ScrollValues(d)
		ofd := maxSize - visSize
		switch fr.Styles.Overflow.Dim(d) {
		// case styles.OverflowVisible:
		// note: this shouldn't happen -- just have this in here for monitoring
		// fmt.Println(ly, "OverflowVisible ERROR -- shouldn't have overflow:", d, ofd)
		case styles.OverflowAuto:
			if ofd < 0.5 {
				if fr.HasScroll[d] {
					if DebugSettings.LayoutTrace {
						fmt.Println(fr, "turned off scroll", d)
					}
					change = true
					fr.HasScroll[d] = false
					fr.layout.ScrollSize.SetDim(d.Other(), 0)
				}
				continue
			}
			if !fr.HasScroll[d] {
				change = true
			}
			fr.HasScroll[d] = true
			fr.layout.ScrollSize.SetDim(d.Other(), sbw)
			if change && DebugSettings.LayoutTrace {
				fmt.Println(fr, "OverflowAuto enabling scrollbars for dim for overflow:", d, ofd, "alloc:", sz.Alloc.Content.Dim(d), "internal:", sz.Internal.Dim(d))
			}
		}
	}
	fr.This.(Layouter).LayoutSpace() // adds the scroll space
	if updateSize {
		sz.setTotalFromContent(&sz.Actual)
		sz.setContentFromTotal(&sz.Alloc) // alloc is *decreased* from any increase in space
	}
	if change && DebugSettings.LayoutTrace {
		fmt.Println(fr, "ManageOverflow changed")
	}
	return change
}

// sizeDownGrow grows the element sizes based on total extra and Grow
func (fr *Frame) sizeDownGrow(iter int, extra math32.Vector2) bool {
	redo := false
	if fr.Styles.Display == styles.Stacked {
		redo = fr.sizeDownGrowStacked(iter, extra)
	} else {
		redo = fr.sizeDownGrowCells(iter, extra)
	}
	return redo
}

func (fr *Frame) sizeDownGrowCells(iter int, extra math32.Vector2) bool {
	redo := false
	sz := &fr.Geom.Size
	alloc := sz.Alloc.Content.Sub(sz.InnerSpace) // inner is fixed
	// todo: use max growth values instead of individual ones to ensure consistency!
	li := &fr.layout
	if len(li.Cells) == 0 {
		slog.Error("unexpected error: layout has not been initialized", "layout", fr.String())
		return false
	}
	var maxExtra, exn math32.Vector2
	for exitr := range 2 {
		fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
			cidx := cwb.Geom.Cell
			ksz := &cwb.Geom.Size
			grw := cwb.Styles.Grow
			if iter == 0 && cwb.Styles.GrowWrap {
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
				md := li.cell(ma, mi, ci)       // X, Y
				cd := li.cell(ca, ci, mi)       // Y, X
				if md == nil || cd == nil {
					break
				}
				mx := md.Size.Dim(ma)
				asz := mx
				gsum := cd.Grow.Dim(ma)
				if gsum > 0 && exd > 0 {
					if gr > gsum {
						fmt.Println(fr, "SizeDownGrowCells error: grow > grow sum:", gr, gsum)
						gr = gsum
					}
					redo = true
					asz = math32.Round(mx + exd*(gr/gsum))
					oasz := asz
					styles.SetClampMax(&asz, ksz.Max.Dim(ma))
					if asz < oasz { // didn't consume its full amount
						if exitr == 0 {
							maxExtra.SetDim(ma, maxExtra.Dim(ma)+(oasz-asz))
							exn.SetDim(ma, exn.Dim(ma)+gr)
						}
					} else {
						if exitr == 1 { // get more!
							nsum := gsum - exn.Dim(ma)
							if nsum > 0 {
								asz += math32.Round(maxExtra.Dim(ma) * (gr / nsum))
							}
						}
					}
					if asz > math32.Ceil(alloc.Dim(ma))+1 { // bug!
						// if DebugSettings.LayoutTrace {
						fmt.Println(fr, "SizeDownGrowCells error: sub alloc > total to alloc:", asz, alloc.Dim(ma))
						fmt.Println("ma:", ma, "mi:", mi, "ci:", ci, "mx:", mx, "gsum:", gsum, "gr:", gr, "ex:", exd, "par act:", sz.Actual.Content.Dim(ma))
						fmt.Println(fr.layout.String())
						fmt.Println(fr.layout.cellsSize())
						// }
					}
				}
				if DebugSettings.LayoutTraceDetail {
					fmt.Println(cwb, ma, "alloc:", asz, "was act:", sz.Actual.Total.Dim(ma), "mx:", mx, "gsum:", gsum, "gr:", gr, "ex:", exd)
				}
				ksz.Alloc.Total.SetDim(ma, asz)
			}
			ksz.setContentFromTotal(&ksz.Alloc)
			return tree.Continue
		})
		if exn.X == 0 && exn.Y == 0 {
			break
		}
	}
	return redo
}

func (fr *Frame) sizeDownWrap(iter int) bool {
	wrapped := false
	li := &fr.layout
	sz := &fr.Geom.Size
	d := li.MainAxis
	alloc := sz.Alloc.Content
	gap := li.Gap.Dim(d)
	fit := alloc.Dim(d)
	if DebugSettings.LayoutTrace {
		fmt.Println(fr, "SizeDownWrap fitting into:", d, fit)
	}
	first := true
	var sum float32
	var n int
	var wraps []int
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		ksz := cwb.Geom.Size.Actual.Total
		if first {
			n = 1
			sum = ksz.Dim(d) + gap
			first = false
			return tree.Continue
		}
		if sum+ksz.Dim(d)+gap >= fit {
			if DebugSettings.LayoutTraceDetail {
				fmt.Println(fr, "wrapped:", i, sum, ksz.Dim(d), fit)
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
		fmt.Println(fr, "wrapped:", wraps)
	}
	li.Wraps = wraps
	fr.setWrapIndexes()
	li.initCells()
	fr.setGapSizeFromCells()
	fr.sizeFromChildrenCells(iter, SizeDownPass)
	return wrapped
}

func (fr *Frame) sizeDownGrowStacked(iter int, extra math32.Vector2) bool {
	// stack just gets everything from us
	chg := false
	asz := fr.Geom.Size.Alloc.Content
	// todo: could actually use the grow factors to decide if growing here?
	if fr.LayoutStackTopOnly {
		kwb := fr.StackTopWidget()
		if kwb != nil {
			ksz := &kwb.Geom.Size
			if ksz.Alloc.Total != asz {
				chg = true
			}
			ksz.Alloc.Total = asz
			ksz.setContentFromTotal(&ksz.Alloc)
		}
		return chg
	}
	// note: allocate everyone in case they are flipped to top
	// need a new layout if size is actually different
	fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		ksz := &cwb.Geom.Size
		if ksz.Alloc.Total != asz {
			chg = true
		}
		ksz.Alloc.Total = asz
		ksz.setContentFromTotal(&ksz.Alloc)
		return tree.Continue
	})
	return chg
}

// sizeDownAllocActual sets Alloc to Actual for no-extra case.
func (fr *Frame) sizeDownAllocActual(iter int) {
	if fr.Styles.Display == styles.Stacked {
		fr.sizeDownAllocActualStacked(iter)
		return
	}
	// todo: wrap needs special case
	fr.sizeDownAllocActualCells(iter)
}

// sizeDownAllocActualCells sets Alloc to Actual for no-extra case.
// Note however that due to max sizing for row / column,
// this size can actually be different than original actual.
func (fr *Frame) sizeDownAllocActualCells(iter int) {
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		ksz := &cwb.Geom.Size
		cidx := cwb.Geom.Cell
		for ma := math32.X; ma <= math32.Y; ma++ { // main axis = X then Y
			ca := ma.Other()                 // cross axis = Y then X
			mi := math32.PointDim(cidx, ma)  // X, Y
			ci := math32.PointDim(cidx, ca)  // Y, X
			md := fr.layout.cell(ma, mi, ci) // X, Y
			asz := md.Size.Dim(ma)
			ksz.Alloc.Total.SetDim(ma, asz)
		}
		ksz.setContentFromTotal(&ksz.Alloc)
		return tree.Continue
	})
}

func (fr *Frame) sizeDownAllocActualStacked(iter int) {
	// stack just gets everything from us
	asz := fr.Geom.Size.Actual.Content
	// todo: could actually use the grow factors to decide if growing here?
	if fr.LayoutStackTopOnly {
		kwb := fr.StackTopWidget()
		if kwb != nil {
			ksz := &kwb.Geom.Size
			ksz.Alloc.Total = asz
			ksz.setContentFromTotal(&ksz.Alloc)
		}
		return
	}
	// note: allocate everyone in case they are flipped to top
	// need a new layout if size is actually different
	fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		ksz := &cwb.Geom.Size
		ksz.Alloc.Total = asz
		ksz.setContentFromTotal(&ksz.Alloc)
		return tree.Continue
	})
}

func (fr *Frame) sizeDownCustom(iter int) bool {
	fr.updateParentRelSizes()
	fr.growToAlloc()
	sz := &fr.Geom.Size
	if DebugSettings.LayoutTrace {
		fmt.Println(fr, "Custom Managing Alloc:", sz.Alloc.Content)
	}
	styles.SetClampMaxVector(&sz.Alloc.Content, sz.Max) // can't be more than max..
	// this allocates our full size to each child, same as ActualStacked all case
	asz := sz.Actual.Content
	fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		ksz := &cwb.Geom.Size
		ksz.Alloc.Total = asz
		ksz.setContentFromTotal(&ksz.Alloc)
		return tree.Continue
	})
	redo := fr.sizeDownChildren(iter)
	fr.sizeDownParts(iter) // no std role, just get sizes
	return redo
}

// sizeFinalUpdateChildrenSizes can optionally be called for layouts
// that dynamically create child elements based on final layout size.
// It ensures that the children are properly sized.
func (fr *Frame) sizeFinalUpdateChildrenSizes() {
	fr.SizeUp()
	iter := 3 // late stage..
	fr.This.(Layouter).SizeDownSetAllocs(iter)
	fr.sizeDownChildren(iter)
	fr.sizeDownParts(iter) // no std role, just get sizes
}

// SizeFinal: (bottom-up) similar to SizeUp but done at the end of the
// Sizing phase: first grows widget Actual sizes based on their Grow
// factors, up to their Alloc sizes.  Then gathers this updated final
// actual Size information for layouts to register their actual sizes
// prior to positioning, which requires accurate Actual vs. Alloc
// sizes to perform correct alignment calculations.
func (wb *WidgetBase) SizeFinal() {
	wb.Geom.RelPos.SetZero()
	sz := &wb.Geom.Size
	sz.Internal = sz.Actual.Content // keep it before we grow
	wb.growToAlloc()
	wb.styleSizeUpdate() // now that sizes are stable, ensure styling based on size is updated
	wb.sizeFinalParts()
	sz.setTotalFromContent(&sz.Actual)
}

// growToAlloc grows our Actual size up to current Alloc size
// for any dimension with a non-zero Grow factor.
// If Grow is < 1, then the size is increased in proportion, but
// any factor > 1 produces a full fill along that dimension.
// Returns true if this resulted in a change in our Total size.
func (wb *WidgetBase) growToAlloc() bool {
	if (wb.Scene != nil && wb.Scene.hasFlag(sceneContentSizing)) || wb.Styles.GrowWrap {
		return false
	}
	sz := &wb.Geom.Size
	act, change := wb.growToAllocSize(sz.Actual.Total, sz.Alloc.Total)
	if change {
		if DebugSettings.LayoutTrace {
			fmt.Println(wb, "GrowToAlloc:", sz.Alloc.Total, "from actual:", sz.Actual.Total)
		}
		sz.Actual.Total = act // already has max constraint
		sz.setContentFromTotal(&sz.Actual)
	}
	return change
}

// sizeFinalParts adjusts the Content size to hold the Parts Final sizes
func (wb *WidgetBase) sizeFinalParts() {
	if wb.Parts == nil {
		return
	}
	wb.Parts.SizeFinal()
	sz := &wb.Geom.Size
	sz.FitSizeMax(&sz.Actual.Content, wb.Parts.Geom.Size.Actual.Total)
}

func (fr *Frame) SizeFinal() {
	if fr.Styles.Display == styles.Custom {
		fr.WidgetBase.SizeFinal() // behave like a widget
		fr.WidgetBase.sizeFinalChildren()
		return
	}
	if !fr.HasChildren() || !fr.layout.shapeCheck(fr, "SizeFinal") {
		fr.WidgetBase.SizeFinal() // behave like a widget
		return
	}
	fr.Geom.RelPos.SetZero()
	fr.sizeFinalChildren() // kids do their own thing
	fr.sizeFromChildrenFit(0, SizeFinalPass)
	fr.growToAlloc()
	fr.styleSizeUpdate() // now that sizes are stable, ensure styling based on size is updated
	fr.sizeFinalParts()
}

// sizeFinalChildren calls SizeFinal on all the children of this node
func (wb *WidgetBase) sizeFinalChildren() {
	wb.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cw.SizeFinal()
		return tree.Continue
	})
}

// sizeFinalChildren calls SizeFinal on all the children of this node
func (fr *Frame) sizeFinalChildren() {
	if fr.Styles.Display == styles.Stacked && !fr.LayoutStackTopOnly {
		fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
			cw.SizeFinal()
			return tree.Continue
		})
		return
	}
	fr.WidgetBase.sizeFinalChildren()
}

// styleSizeUpdate updates styling size values for widget and its parent,
// which should be called after these are updated. Returns true if any changed.
func (wb *WidgetBase) styleSizeUpdate() bool {
	pwb := wb.parentWidget()
	if pwb == nil {
		return false
	}
	wb.updateParentRelSizes()
	scsz := wb.Scene.SceneGeom.Size
	sz := wb.Geom.Size.Alloc.Content
	psz := pwb.Geom.Size.Alloc.Content
	chg := wb.Styles.UnitContext.SetSizes(float32(scsz.X), float32(scsz.Y), sz.X, sz.Y, psz.X, psz.Y)
	if chg {
		wb.Styles.ToDots()
	}
	return chg
}

// Position uses the final sizes to set relative positions within layouts
// according to alignment settings.
func (wb *WidgetBase) Position() {
	wb.positionParts()
}

func (wb *WidgetBase) positionWithinAllocMainX(pos math32.Vector2, parJustify, parAlign styles.Aligns) {
	sz := &wb.Geom.Size
	pos.X += styles.AlignPos(styles.ItemAlign(parJustify, wb.Styles.Justify.Self), sz.Actual.Total.X, sz.Alloc.Total.X)
	pos.Y += styles.AlignPos(styles.ItemAlign(parAlign, wb.Styles.Align.Self), sz.Actual.Total.Y, sz.Alloc.Total.Y)
	wb.Geom.RelPos = pos
	if DebugSettings.LayoutTrace {
		fmt.Println(wb, "Position within Main=X:", pos)
	}
}

func (wb *WidgetBase) positionWithinAllocMainY(pos math32.Vector2, parJustify, parAlign styles.Aligns) {
	sz := &wb.Geom.Size
	pos.Y += styles.AlignPos(styles.ItemAlign(parJustify, wb.Styles.Justify.Self), sz.Actual.Total.Y, sz.Alloc.Total.Y)
	pos.X += styles.AlignPos(styles.ItemAlign(parAlign, wb.Styles.Align.Self), sz.Actual.Total.X, sz.Alloc.Total.X)
	wb.Geom.RelPos = pos
	if DebugSettings.LayoutTrace {
		fmt.Println(wb, "Position within Main=Y:", pos)
	}
}

func (wb *WidgetBase) positionParts() {
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

// positionChildren runs Position on the children
func (wb *WidgetBase) positionChildren() {
	wb.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cw.Position()
		return tree.Continue
	})
}

// Position: uses the final sizes to position everything within layouts
// according to alignment settings.
func (fr *Frame) Position() {
	if fr.Styles.Display == styles.Custom {
		fr.positionFromPos()
		fr.positionChildren()
		return
	}
	if !fr.HasChildren() || !fr.layout.shapeCheck(fr, "Position") {
		fr.WidgetBase.Position() // behave like a widget
		return
	}
	if fr.Parent == nil {
		fr.positionWithinAllocMainY(math32.Vector2{}, fr.Styles.Justify.Items, fr.Styles.Align.Items)
	}
	fr.ConfigScrolls() // and configure the scrolls
	if fr.Styles.Display == styles.Stacked {
		fr.positionStacked()
	} else {
		fr.positionCells()
		fr.positionChildren()
	}
	fr.positionParts()
}

func (fr *Frame) positionCells() {
	if fr.Styles.Display == styles.Flex && fr.Styles.Direction == styles.Column {
		fr.positionCellsMainY()
		return
	}
	fr.positionCellsMainX()
}

// Main axis = X
func (fr *Frame) positionCellsMainX() {
	// todo: can break apart further into Flex rows
	gap := fr.layout.Gap
	sz := &fr.Geom.Size
	if DebugSettings.LayoutTraceDetail {
		fmt.Println(fr, "PositionCells Main X, actual:", sz.Actual.Content, "internal:", sz.Internal)
	}
	var stPos math32.Vector2
	stPos.X = styles.AlignPos(fr.Styles.Justify.Content, sz.Internal.X, sz.Actual.Content.X)
	stPos.Y = styles.AlignPos(fr.Styles.Align.Content, sz.Internal.Y, sz.Actual.Content.Y)
	pos := stPos
	var lastSz math32.Vector2
	idx := 0
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cidx := cwb.Geom.Cell
		if cidx.X == 0 && idx > 0 {
			pos.X = stPos.X
			pos.Y += lastSz.Y + gap.Y
		}
		cwb.positionWithinAllocMainX(pos, fr.Styles.Justify.Items, fr.Styles.Align.Items)
		alloc := cwb.Geom.Size.Alloc.Total
		pos.X += alloc.X + gap.X
		lastSz = alloc
		idx++
		return tree.Continue
	})
}

// Main axis = Y
func (fr *Frame) positionCellsMainY() {
	gap := fr.layout.Gap
	sz := &fr.Geom.Size
	if DebugSettings.LayoutTraceDetail {
		fmt.Println(fr, "PositionCells, actual", sz.Actual.Content, "internal:", sz.Internal)
	}
	var lastSz math32.Vector2
	var stPos math32.Vector2
	stPos.Y = styles.AlignPos(fr.Styles.Justify.Content, sz.Internal.Y, sz.Actual.Content.Y)
	stPos.X = styles.AlignPos(fr.Styles.Align.Content, sz.Internal.X, sz.Actual.Content.X)
	pos := stPos
	idx := 0
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cidx := cwb.Geom.Cell
		if cidx.Y == 0 && idx > 0 {
			pos.Y = stPos.Y
			pos.X += lastSz.X + gap.X
		}
		cwb.positionWithinAllocMainY(pos, fr.Styles.Justify.Items, fr.Styles.Align.Items)
		alloc := cwb.Geom.Size.Alloc.Total
		pos.Y += alloc.Y + gap.Y
		lastSz = alloc
		idx++
		return tree.Continue
	})
}

func (fr *Frame) positionStacked() {
	fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cwb.Geom.RelPos.SetZero()
		if !fr.LayoutStackTopOnly || i == fr.StackTop {
			cw.Position()
		}
		return tree.Continue
	})
}

// ApplyScenePos computes scene-based absolute positions and final BBox
// bounding boxes for rendering, based on relative positions from
// Position step and parents accumulated position and scroll offset.
// This is the only step needed when scrolling (very fast).
func (wb *WidgetBase) ApplyScenePos() {
	wb.setPosFromParent()
	wb.setBBoxes()
}

// setContentPosFromPos sets the Pos.Content position based on current Pos
// plus the BoxSpace position offset.
func (wb *WidgetBase) setContentPosFromPos() {
	off := wb.Styles.BoxSpace().Pos().Floor()
	wb.Geom.Pos.Content = wb.Geom.Pos.Total.Add(off)
}

func (wb *WidgetBase) setPosFromParent() {
	pwb := wb.parentWidget()
	var parPos math32.Vector2
	if pwb != nil {
		parPos = pwb.Geom.Pos.Content.Add(pwb.Geom.Scroll) // critical that parent adds here but not to self
	}
	wb.Geom.Pos.Total = wb.Geom.RelPos.Add(parPos)
	wb.setContentPosFromPos()
	if DebugSettings.LayoutTrace {
		fmt.Println(wb, "pos:", wb.Geom.Pos.Total, "parPos:", parPos)
	}
}

// setBBoxesFromAllocs sets BBox and ContentBBox from Geom.Pos and .Size
// This does NOT intersect with parent content BBox, which is done in SetBBoxes.
// Use this for elements that are dynamically positioned outside of parent BBox.
func (wb *WidgetBase) setBBoxesFromAllocs() {
	wb.Geom.TotalBBox = wb.Geom.totalRect()
	wb.Geom.ContentBBox = wb.Geom.contentRect()
}

func (wb *WidgetBase) setBBoxes() {
	pwb := wb.parentWidget()
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
		bb := wb.Geom.totalRect()
		wb.Geom.TotalBBox = parBB.Intersect(bb)
		if DebugSettings.LayoutTrace {
			fmt.Println(wb, "Total BBox:", bb, "parBB:", parBB, "BBox:", wb.Geom.TotalBBox)
		}

		cbb := wb.Geom.contentRect()
		wb.Geom.ContentBBox = parBB.Intersect(cbb)
		if DebugSettings.LayoutTrace {
			fmt.Println(wb, "Content BBox:", cbb, "parBB:", parBB, "BBox:", wb.Geom.ContentBBox)
		}
	}
	wb.applyScenePosParts()
}

func (wb *WidgetBase) applyScenePosParts() {
	if wb.Parts == nil {
		return
	}
	wb.Parts.ApplyScenePos()
}

// applyScenePosChildren runs ApplyScenePos on the children
func (wb *WidgetBase) applyScenePosChildren() {
	wb.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cw.ApplyScenePos()
		return tree.Continue
	})
}

// applyScenePosChildren runs ScenePos on the children
func (fr *Frame) applyScenePosChildren() {
	if fr.Styles.Display == styles.Stacked && !fr.LayoutStackTopOnly {
		fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
			cw.ApplyScenePos()
			return tree.Continue
		})
		return
	}
	fr.WidgetBase.applyScenePosChildren()
}

// ApplyScenePos: scene-based position and final BBox is computed based on
// parents accumulated position and scrollbar position.
// This step can be performed when scrolling after updating Scroll.
func (fr *Frame) ApplyScenePos() {
	fr.scrollResetIfNone()
	if fr.Styles.Display == styles.Custom {
		fr.WidgetBase.ApplyScenePos()
		fr.applyScenePosChildren()
		fr.PositionScrolls()
		fr.applyScenePosParts() // in case they fit inside parent
		return
	}
	// note: ly.Geom.Scroll has the X, Y scrolling offsets, set by Layouter.ScrollChanged function
	if !fr.HasChildren() || !fr.layout.shapeCheck(fr, "ScenePos") {
		fr.WidgetBase.ApplyScenePos() // behave like a widget
		return
	}
	fr.WidgetBase.ApplyScenePos()
	fr.applyScenePosChildren()
	fr.PositionScrolls()
	fr.applyScenePosParts() // in case they fit inside parent
	// otherwise handle separately like scrolls on layout
}

// scrollResetIfNone resets the scroll offsets if there are no scrollbars
func (fr *Frame) scrollResetIfNone() {
	for d := math32.X; d <= math32.Y; d++ {
		if !fr.HasScroll[d] {
			fr.Geom.Scroll.SetDim(d, 0)
		}
	}
}

// positionFromPos does Custom positioning from style positions.
func (fr *Frame) positionFromPos() {
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cwb.Geom.RelPos.X = cwb.Styles.Pos.X.Dots
		cwb.Geom.RelPos.Y = cwb.Styles.Pos.Y.Dots
		return tree.Continue
	})
}

// DirectRenderDrawBBoxes returns the destination and source bounding boxes
// for RenderDraw call for widgets that do direct rendering.
// The destBBox.Min point can be passed as the dp destination point for Draw
// function, and srcBBox is the source region.  Empty flag indicates if either
// of the srcBBox dimensions are <= 0.
func (wb *WidgetBase) DirectRenderDrawBBoxes(srcFullBBox image.Rectangle) (destBBox, srcBBox image.Rectangle, empty bool) {
	tbb := wb.Geom.TotalBBox
	destBBox = tbb.Add(wb.Scene.SceneGeom.Pos)
	srcBBox = srcFullBBox
	pos := wb.Geom.Pos.Total.ToPoint()
	if pos.X < tbb.Min.X { // scrolled off left
		srcBBox.Min.X = tbb.Min.X - pos.X
	}
	if pos.Y < tbb.Min.Y {
		srcBBox.Min.X = tbb.Min.Y - pos.X
	}
	sz := srcBBox.Size()
	if sz.X <= 0 || sz.Y <= 0 {
		empty = true
	}
	return
}
