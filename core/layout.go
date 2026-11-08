// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log/slog"
	"slices"

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

// GeomSize has all of the relevant Layout sizes
type GeomSize struct {
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

	// slot is the size that the parent layout would have allocated
	// to a growing widget whose allocation is limited by its Max size,
	// which it is aligned within, and 0 otherwise.
	slot math32.Vector2

	// extent is the total size of the widget including any content of it
	// that overflows it visibly (as for a frame with a Max narrower than
	// its content), which is beyond its Actual size; it is zero or Actual
	// when there is none. See [Frame.sizeOverflow].
	extent math32.Vector2

	// preferred is the actual total size of the widget from the SizeUp
	// pass, before its parent allocates it less, which is its max-content
	// size for widgets that shrink ([styles.Style.Shrink]), so that they
	// are as wide as they can be in the space that they have.
	preferred math32.Vector2

	// minWidth is the min-content width of the widget from the SizeUp
	// pass, as in CSS: its total width (with its Space) below which its
	// content overflows, such as the width of the longest word of text.
	// The layout does not narrow a widget below it.
	minWidth float32

	// prefWidth is the max-content (preferred) width of the widget from
	// the SizeUp pass: its total width without any wrapping, such as the
	// width of text on one line. A widget starts at this width, and is
	// narrowed toward its minWidth only when there is too little space.
	prefWidth float32
}

// setIntrinsicWidths sets the minWidth and prefWidth from the given
// min-content and max-content widths of the content, within the Min and
// Max sizes, with the Space added.
func (ls *GeomSize) setIntrinsicWidths(minw, prefw float32) {
	minw = max(minw, ls.Min.X)
	prefw = max(prefw, minw)
	if mx := ls.Max.X; mx > 0 {
		minw = min(minw, mx)
		prefw = min(prefw, mx)
	}
	ls.minWidth = math32.Ceil(minw + ls.Space.X)
	ls.prefWidth = math32.Ceil(prefw + ls.Space.X)
}

func (ls GeomSize) String() string {
	return fmt.Sprintf("Actual: %v, \tAlloc: %v", ls.Actual, ls.Alloc)
}

// setInitContentMin sets initial Actual.Content size from given Styles.Min,
// further subject to the current Max constraint.
func (ls *GeomSize) setInitContentMin(styMin math32.Vector2) {
	csz := &ls.Actual.Content
	*csz = styMin
	styles.SetClampMaxVector(csz, ls.Max)
}

// maxTotal returns the Max size as a Total size, adding the Space
// to each dimension that has a Max (which is 0 for no Max).
func (ls *GeomSize) maxTotal() math32.Vector2 {
	mt := ls.Max
	for d := math32.X; d <= math32.Y; d++ {
		if mt.Dim(d) > 0 {
			mt.SetDim(d, mt.Dim(d)+ls.Space.Dim(d))
		}
	}
	return mt
}

// fitTotalMax increases the given Total size to fit the given fm value,
// subject to the Max constraint on the content, as [GeomSize.FitSizeMax]
// does for a content size.
func (ls *GeomSize) fitTotalMax(to *math32.Vector2, fm math32.Vector2) {
	styles.SetClampMinVector(to, fm)
	styles.SetClampMaxVector(to, ls.maxTotal())
}

// FitSizeMax increases given size to fit given fm value, subject to Max constraints
func (ls *GeomSize) FitSizeMax(to *math32.Vector2, fm math32.Vector2) {
	styles.SetClampMinVector(to, fm)
	styles.SetClampMaxVector(to, ls.Max)
}

// SetTotalFromContent sets the Total size as Content plus Space
func (ls *GeomSize) SetTotalFromContent(ct *geomCT) {
	ct.Total = ct.Content.Add(ls.Space)
}

// SetContentFromTotal sets the Content from Total size,
// subtracting Space
func (ls *GeomSize) SetContentFromTotal(ct *geomCT) {
	ct.Content = ct.Total.Sub(ls.Space)
}

// GeomState contains the the layout geometry state for each widget.
// Set by the parent Layout during the Layout process.
type GeomState struct {
	// Size has sizing data for the widget: use Actual for rendering.
	// Alloc shows the potentially larger space top-down allocated.
	Size GeomSize `display:"add-fields"`

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

	// contentOffset is the offset of Pos.Content from Pos.Total, the
	// position of the style's BoxSpace. It is set with Size.Space, so that
	// positioning, as when scrolling, need not compute it again.
	contentOffset math32.Vector2
}

func (ls *GeomState) String() string {
	return "Size: " + ls.Size.String() + "\nPos: " + ls.Pos.String() + "\tCell: " + ls.Cell.String() +
		"\tRelPos: " + ls.RelPos.String() + "\tScroll: " + ls.Scroll.String()
}

// contentRangeDim returns the Content bounding box min, max
// along given dimension
func (ls *GeomState) contentRangeDim(d math32.Dims) (cmin, cmax float32) {
	cmin = float32(math32.PointDim(ls.ContentBBox.Min, d))
	cmax = float32(math32.PointDim(ls.ContentBBox.Max, d))
	return
}

// totalRect returns Pos.Total -- Size.Actual.Total
// as an image.Rectangle, e.g., for bounding box
func (ls *GeomState) totalRect() image.Rectangle {
	return math32.RectFromPosSizeMax(ls.Pos.Total, ls.Size.Actual.Total)
}

// contentRect returns Pos.Content, Size.Actual.Content
// as an image.Rectangle, e.g., for bounding box.
func (ls *GeomState) contentRect() image.Rectangle {
	return math32.RectFromPosSizeMax(ls.Pos.Content, ls.Size.Actual.Content)
}

// ScrollOffset computes the net scrolling offset as a function of
// the difference between the allocated position and the actual
// content position according to the clipped bounding box.
func (ls *GeomState) ScrollOffset() image.Point {
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
			if md == nil {
				continue
			}
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

	// spans is whether a grid has cells that span columns.
	spans bool

	// rowSpans is whether a grid has cells that span rows.
	rowSpans bool

	// floats is the layout of a column with floating children, if it has any.
	floats *floatLayout

	// growExtra is the extra space that children with a Grow factor grow
	// into, which is within the actual size of a frame that does not grow
	// (see [Frame.childSpace]), while other children (such as GrowWrap
	// text) are allocated all of the extra allocation of the frame.
	growExtra math32.Vector2

	// minContent is the min-content width of the content of the frame,
	// from its children (see [Frame.sizeUpIntrinsicFrame]), which it
	// scrolls in width only if it is narrower than.
	minContent float32

	// overflow is the size of the content of the frame including the
	// content of its children that overflows them visibly (see
	// [Frame.sizeOverflow]), which the frame scrolls over.
	overflow math32.Vector2

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
	// wrapped content is sized by its allocation, except that wrapped
	// rows that shrink are as wide as they are on one line, until then
	nosz := pass == SizeUpPass && fr.Styles.IsFlexWrap() &&
		!(fr.Styles.Shrink.X > 0 && fr.Styles.Direction == styles.Row)
	mx := sz.Max
	for d := math32.X; d <= math32.Y; d++ {
		if nosz || fr.contained(d) {
			continue // (contained: as if it had no content)
		}
		// a frame that scrolls is as large as its style, unless it shrinks
		// in width (as in CSS, where it starts at the width of its content,
		// and is narrowed to fit, down to nothing: see sizeUpIntrinsicFrame),
		// which is then as wide as its content up to its allocation
		shrinks := d == math32.X && fr.Styles.Shrink.X > 0
		scrolls := !(fr.Scene != nil && fr.Scene.hasFlag(sceneContentSizing)) && oflow.Dim(d) >= styles.OverflowAuto && fr.Parent != nil
		switch {
		case scrolls && shrinks && pass != SizeUpPass:
			w := nsz.X
			if al := sz.Alloc.Content.X; al > 0 {
				w = min(w, al)
			}
			asz.X = styles.ClampMin(asz.X, w)
		case scrolls && !shrinks:
			if mx.Dim(d) > 0 {
				asz.SetDim(d, styles.ClampMax(styles.ClampMin(asz.Dim(d), nsz.Dim(d)), mx.Dim(d)))
			}
		default:
			asz.SetDim(d, styles.ClampMin(asz.Dim(d), nsz.Dim(d)))
		}
	}
	styles.SetClampMaxVector(asz, mx)
	sz.SetTotalFromContent(&sz.Actual)
}

// SizeUp (bottom-up) gathers Actual sizes from our Children & Parts,
// based on Styles.Min / Max sizes and actual content sizing
// (e.g., text size), along with the intrinsic min-content and preferred
// widths (see layoutintrinsic.go). Wrapping [Text] starts out at its
// preferred width (on one line), and is re-wrapped to its allocated width
// in SizeDown; flex wrap layouts are wrapped in SizeDown.
func (wb *WidgetBase) SizeUp() {
	wb.SizeUpWidget()
}

// SizeUpWidget is the standard Widget SizeUp pass
func (wb *WidgetBase) SizeUpWidget() {
	wb.sizeFromStyle()
	wb.sizeUpParts()
	sz := &wb.Geom.Size
	sz.SetTotalFromContent(&sz.Actual)
	wb.sizeAspect(false)
	wb.sizeUpIntrinsic()
}

// sizeUpIntrinsic sets the intrinsic widths of a widget that has no
// children of its own in the layout: both are its actual width. A widget
// with parts (such as a button, switch, or chooser) is not narrowed, as
// its parts are laid out within its actual size: its label stays on one
// line, and the frame that it is in scrolls if it does not fit.
func (wb *WidgetBase) sizeUpIntrinsic() {
	sz := &wb.Geom.Size
	sz.minWidth, sz.prefWidth = sz.Actual.Total.X, sz.Actual.Total.X
}

// spaceFromStyle sets the Space based on Style BoxSpace().Size(),
// and the content offset from its position.
func (wb *WidgetBase) spaceFromStyle() {
	space := wb.Styles.BoxSpace()
	wb.Geom.Size.Space = space.Size().Ceil()
	wb.Geom.contentOffset = space.Pos().Floor()
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
	if bs := wb.boxSizingSpace(); bs != (math32.Vector2{}) {
		for d := math32.X; d <= math32.Y; d++ {
			if m := sz.Min.Dim(d); m > 0 {
				sz.Min.SetDim(d, max(m-bs.Dim(d), 0))
			}
			if m := sz.Max.Dim(d); m > 0 {
				sz.Max.SetDim(d, max(m-bs.Dim(d), 1))
			}
		}
	}
	sz.Internal.SetZero()
	sz.setInitContentMin(sz.Min)
	sz.SetTotalFromContent(&sz.Actual)
	if DebugSettings.LayoutTrace && (sz.Actual.Content.X > 0 || sz.Actual.Content.Y > 0) {
		fmt.Println(wb, "SizeUp from Style:", sz.Actual.Content.String())
	}
}

// specifiedHeight returns the height of the content of the widget that is
// specified by its style, if it is: its Min height in units that are not
// relative to its parent, or a percentage of the specified height of its
// parent, if it has a Min height and no Max height (as for a height in CSS).
func (wb *WidgetBase) specifiedHeight() (float32, bool) {
	s := &wb.Styles
	mn := s.Min.Y
	if mn.Value <= 0 || mn.Custom != nil || s.Max.Y.Value > 0 {
		return 0, false
	}
	switch mn.Unit {
	case units.UnitPw:
		return 0, false
	case units.UnitPh:
		pwb := wb.parentWidget()
		if pwb == nil {
			return 0, false
		}
		ph, ok := pwb.specifiedHeight()
		if !ok {
			return 0, false
		}
		return max(ph*0.01*mn.Value-wb.boxSizingSpace().Y, 0), true
	}
	return math32.Ceil(mn.Dots) - wb.boxSizingSpace().Y, true
}

// styleMinDots returns the Min size of the style in dots, rounded up,
// without the dimensions that are relative to the size of the parent
// (pw and ph units), which are not known until the parent is sized (see
// [WidgetBase.updateParentRelSizes]), as in [WidgetBase.sizeFromStyle]:
// their dots are of the size of the parent when the widget was styled (as
// the scene, for a text field of a height of 100% of the search box of
// duckduckgo.com, which was as high as the window).
func (wb *WidgetBase) styleMinDots() math32.Vector2 {
	s := &wb.Styles
	m := s.Min.Dots().Ceil()
	if s.Min.X.Unit == units.UnitPw || s.Min.X.Unit == units.UnitPh {
		m.X = 0
	}
	if s.Min.Y.Unit == units.UnitPw || s.Min.Y.Unit == units.UnitPh {
		m.Y = 0
	}
	return m
}

// boxSizingSpace returns the size of the padding and border of the widget
// if its Min and Max sizes include them ([styles.BoxBorderBox]), which is
// subtracted from them for the size of the content, or zero otherwise.
func (wb *WidgetBase) boxSizingSpace() math32.Vector2 {
	s := &wb.Styles
	if s.BoxSizing != styles.BoxBorderBox {
		return math32.Vector2{}
	}
	return s.Padding.Dots().Size().Add(s.Border.Width.Dots().Size())
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
	if h, ok := pwb.specifiedHeight(); ok {
		// a percentage of the height of the parent is of its specified
		// height, as in CSS, not of the height that it is given, which can
		// be that of its content: with that, a text field of a height of
		// 100% in frames of 100% (as the search box of duckduckgo.com)
		// made them higher by its padding on each pass
		psz.Y = h
	}
	bs := wb.boxSizingSpace()
	got := false
	for d := math32.X; d <= math32.Y; d++ {
		if s.Min.Dim(d).Unit == units.UnitPw {
			got = true
			effmin.SetDim(d, max(psz.X*0.01*s.Min.Dim(d).Value-bs.Dim(d), 0))
		}
		if s.Min.Dim(d).Unit == units.UnitPh {
			got = true
			effmin.SetDim(d, max(psz.Y*0.01*s.Min.Dim(d).Value-bs.Dim(d), 0))
		}
		if s.Max.Dim(d).Unit == units.UnitPw {
			got = true
			sz.Max.SetDim(d, max(psz.X*0.01*s.Max.Dim(d).Value-bs.Dim(d), 1))
		}
		if s.Max.Dim(d).Unit == units.UnitPh {
			got = true
			sz.Max.SetDim(d, max(psz.Y*0.01*s.Max.Dim(d).Value-bs.Dim(d), 1))
		}
		// custom sizes are computed with the size of the parent too (as a
		// calc() of CSS with a percentage and a length, as the width of
		// calc(100% - 256px) of the grids of github.com)
		if mn := s.Min.Dim(d); mn.Custom != nil {
			got = true
			uc := s.UnitContext
			uc.Paw, uc.Pah = psz.X, psz.Y
			v := max(math32.Ceil(mn.Custom(&uc))-bs.Dim(d), 0)
			effmin.SetDim(d, v)
			sz.Min.SetDim(d, v)
		}
		if mx := s.Max.Dim(d); mx.Custom != nil {
			got = true
			uc := s.UnitContext
			uc.Paw, uc.Pah = psz.X, psz.Y
			sz.Max.SetDim(d, max(math32.Ceil(mx.Custom(&uc))-bs.Dim(d), 1))
		}
	}
	if got {
		// the min is of the content, and the total has the space around it
		// (as the padding of the 21.5% navigation of pkg.go.dev, which was
		// narrower by its padding)
		effmin = effmin.Add(sz.Space)
		sz.fitTotalMax(&sz.Actual.Total, effmin)
		sz.fitTotalMax(&sz.Alloc.Total, effmin)
		sz.SetContentFromTotal(&sz.Actual)
		sz.SetContentFromTotal(&sz.Alloc)
	}
	return got
}

// contained returns whether the size of the widget in the given dimension
// does not depend on its content ([styles.Style.Contain]).
func (wb *WidgetBase) contained(d math32.Dims) bool {
	c := wb.Styles.Contain
	return c == styles.ContainSize || (c == styles.ContainInlineSize && d == math32.X)
}

// aspectHeight returns the height of the content of the widget for its
// [styles.Style.AspectRatio] at the given width of its content, or 0 if
// it has none, or if its height is set (by its Min).
func (wb *WidgetBase) aspectHeight(w float32) float32 {
	r := wb.Styles.AspectRatio
	if r <= 0 || w <= 0 || wb.Styles.Min.Y.Value > 0 {
		return 0
	}
	if wb.Styles.BoxSizing == styles.BoxBorderBox {
		bs := wb.boxSizingSpace()
		return max(math32.Ceil((w+bs.X)/r-bs.Y), 0)
	}
	return math32.Ceil(w / r)
}

// sizeAspect makes the actual height of the content of the widget at
// least the height for its aspect ratio at its actual width (see
// [WidgetBase.aspectHeight]), and its allocated height too if alloc,
// for its children (in SizeDown). It returns whether the actual height
// changed.
func (wb *WidgetBase) sizeAspect(alloc bool) bool {
	sz := &wb.Geom.Size
	w := sz.Actual.Content.X
	if alloc && wb.Styles.Grow.X > 0 && sz.Alloc.Content.X > w {
		w = sz.Alloc.Content.X // as wide as its allocation (as a block)
		if sz.Max.X > 0 {
			w = min(w, sz.Max.X)
		}
	}
	h := wb.aspectHeight(w)
	if h <= 0 {
		return false
	}
	if alloc && sz.Alloc.Content.Y < h {
		sz.Alloc.Content.Y = h
		sz.SetTotalFromContent(&sz.Alloc)
	}
	if sz.Actual.Content.Y >= h {
		return false
	}
	sz.Actual.Content.Y = h
	sz.SetTotalFromContent(&sz.Actual)
	return true
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
	fr.sizeUpIntrinsicFrame()
	fr.sizeAspect(false)
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
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
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
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
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
	if fr.Styles.Shrink.X > 0 && li.MainAxis == math32.X {
		// all on one line, its widest, which wraps in its allocation
		li.Wraps = []int{ni}
		fr.setWrapIndexes()
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
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
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
	if cols == 0 {
		fmt.Println(fr, "no cols:", cols)
	}
	cols = max(cols, 1)
	ci := 0
	ri := 0
	fr.layout.spans = false
	fr.layout.rowSpans = false
	// occupied has the cells occupied by cells that span rows from above,
	// which the cells after them are placed after.
	occupied := map[image.Point]bool{}
	free := func(c, r, cs int) bool {
		for x := c; x < c+cs; x++ {
			if occupied[image.Point{x, r}] {
				return false
			}
		}
		return true
	}
	rows := 0
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cs := min(max(cwb.Styles.ColSpan, 1), cols)
		rs := max(cwb.Styles.RowSpan, 1)
		if cs > 1 {
			fr.layout.spans = true
		}
		if rs > 1 {
			fr.layout.rowSpans = true
		}
		for {
			if ci > 0 && ci+cs > cols { // a spanning cell that does not fit starts a new row
				ci = 0
				ri++
			}
			if free(ci, ri, cs) {
				break
			}
			ci++
		}
		cwb.Geom.Cell = image.Point{ci, ri}
		for r := ri + 1; r < ri+rs; r++ {
			for x := ci; x < ci+cs; x++ {
				occupied[image.Point{x, r}] = true
			}
		}
		rows = max(rows, ri+rs)
		ci += cs
		if ci >= cols {
			ci = 0
			ri++
		}
		return tree.Continue
	})
	fr.layout.Shape = image.Point{cols, max(rows, 1)}
}

// colSpan returns the number of grid columns that the widget spans.
func (fr *Frame) colSpan(cwb *WidgetBase) int {
	if fr.Styles.Display != styles.Grid {
		return 1
	}
	return min(max(cwb.Styles.ColSpan, 1), fr.layout.Shape.X-cwb.Geom.Cell.X)
}

// rowSpan returns the number of grid rows that the widget spans.
func (fr *Frame) rowSpan(cwb *WidgetBase) int {
	if fr.Styles.Display != styles.Grid {
		return 1
	}
	return min(max(cwb.Styles.RowSpan, 1), fr.layout.Shape.Y-cwb.Geom.Cell.Y)
}

// rowSpanAlloc returns the allocated height of a grid cell at the given
// cell that spans the given number of rows: the total of the heights of
// the rows and the gaps between them, with the given extra height that
// the rows grow into, in proportion to their grow factors.
func (fr *Frame) rowSpanAlloc(cidx image.Point, span int, extra float32) float32 {
	li := &fr.layout
	gsum := float32(0)
	for r := range li.Shape.Y {
		gsum += li.cell(math32.Y, r, cidx.X).Grow.Y
	}
	sum := li.Gap.Y * float32(span-1)
	for r := range span {
		cd := li.cell(math32.Y, cidx.Y+r, cidx.X)
		sum += cd.Size.Y
		if gsum > 0 && extra > 0 {
			sum += math32.Round(extra * cd.Grow.Y / gsum)
		}
	}
	return sum
}

// sizeFromChildrenFit gathers Actual size from kids, and calls LaySetContentFitOverflow
// to update Actual and Internal size based on this.
func (fr *Frame) sizeFromChildrenFit(iter int, pass LayoutPasses) {
	ksz := fr.This.(Layouter).SizeFromChildren(iter, pass)
	fr.laySetContentFitOverflow(ksz, pass)
	fr.sizeOverflow()
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
	type spanning struct {
		cell  image.Point
		span  int
		width float32
	}
	var spans, rowSpans []spanning
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cidx := cwb.Geom.Cell
		sz := cwb.Geom.Size.Actual.Total
		grw := cwb.Styles.Grow
		span := fr.colSpan(cwb)
		rspan := fr.rowSpan(cwb)
		if pass == SizeFinalPass {
			// the cell is the allocation of the child, unless its content
			// overflows it (as a frame narrowed below its minimum), which
			// then counts for the frame to scroll
			if grw.X == 0 && !cwb.Styles.GrowWrap {
				sz.X = max(sz.X, cwb.Geom.Size.Alloc.Total.X)
			}
			if grw.Y == 0 {
				sz.Y = max(sz.Y, cwb.Geom.Size.Alloc.Total.Y)
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
				continue
			}
			msz := sz.Dim(ma) // main axis size dim: X, Y
			mx := md.Size.Dim(ma)
			if ma == math32.X && span > 1 { // sized below, over its columns
				spans = append(spans, spanning{cidx, span, msz})
			} else if ma == math32.Y && rspan > 1 { // sized below, over its rows
				rowSpans = append(rowSpans, spanning{cidx, rspan, msz})
			} else {
				mx = max(mx, msz) // Col, max widths of all elements; Row, max heights of all elements
			}
			md.Size.SetDim(ma, mx)
			sm := cd.Size.Dim(ma)
			sm += msz
			cd.Size.SetDim(ma, sm) // Row, sum widths of all elements; Col, sum heights of all elements
			gsz := grw.Dim(ma)
			mx = md.Grow.Dim(ma)
			mx = max(mx, gsz)
			md.Grow.SetDim(ma, mx)
			if ma == math32.X && span > 1 {
				// all of the columns that it spans grow (as the columns
				// of a grid of 1fr that cells span), not only the first
				for k := 1; k < span; k++ {
					if kd := li.cell(ma, mi+k, ci); kd != nil {
						kd.Grow.SetDim(ma, max(kd.Grow.Dim(ma), gsz))
					}
				}
			}
			sm = cd.Grow.Dim(ma)
			sm += gsz
			cd.Grow.SetDim(ma, sm)
		}
		return tree.Continue
	})
	// The columns that a cell spans are widened evenly to fit it.
	for _, sp := range spans {
		sum := li.Gap.X * float32(sp.span-1)
		for c := range sp.span {
			sum += li.cell(math32.X, sp.cell.X+c, sp.cell.Y).Size.X
		}
		if extra := sp.width - sum; extra > 0 {
			for c := range sp.span {
				cd := li.cell(math32.X, sp.cell.X+c, sp.cell.Y)
				cd.Size.X += extra / float32(sp.span)
			}
		}
	}
	// The rows that a cell spans are heightened evenly to fit it.
	for _, sp := range rowSpans {
		sum := li.Gap.Y * float32(sp.span-1)
		for r := range sp.span {
			sum += li.cell(math32.Y, sp.cell.Y+r, sp.cell.X).Size.Y
		}
		if extra := sp.width - sum; extra > 0 {
			if fr.FrLastRow { // all in the last one (see Frame.FrLastRow)
				li.cell(math32.Y, sp.cell.Y+sp.span-1, sp.cell.X).Size.Y += extra
				continue
			}
			for r := range sp.span {
				cd := li.cell(math32.Y, sp.cell.Y+r, sp.cell.X)
				cd.Size.Y += extra / float32(sp.span)
			}
		}
	}
	if DebugSettings.LayoutTraceDetail {
		fmt.Println(fr, "SizeFromChildren")
		fmt.Println(li.String())
	}
	ksz := li.cellsSize()
	if fl := li.floats; fl != nil { // the column is as tall as its floats
		ksz.Y = max(ksz.Y, fl.size.Y-li.GapSize.Y)
		if fr.onlyFloats() { // and as wide, when it only has floats
			ksz.X = max(ksz.X, fl.floatsWidth)
		}
	}
	sizing := fr.Scene != nil && fr.Scene.hasFlag(sceneContentSizing)
	if pass > SizeUpPass && li.Wraps != nil && li.MainAxis == math32.X && !sizing {
		// The lines of a wrapped row whose items are wider than it only by
		// the rounding of their sizes to whole dots, which sizeDownWrap
		// allows, fit it (as their exact sizes do), so that it does not
		// overflow by those dots (except when the scene is sized to fit
		// its content, which it is then as wide as).
		fit := fr.Geom.Size.Alloc.Content.X
		if over := ksz.X - fit; over > 0 && over <= float32(slices.Max(li.Wraps)) {
			ksz.X = fit
		}
	}
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
			if md == nil {
				continue
			}
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
	aspect := wb.sizeAspect(true)
	redo := wb.sizeDownParts(iter)
	return redo || aspect
}

func (wb *WidgetBase) sizeDownParts(iter int) bool {
	if wb.Parts == nil {
		return false
	}
	sz := &wb.Geom.Size
	psz := &wb.Parts.Geom.Size
	pgrow, _ := wb.growToAllocSize(sz.Actual.Content, sz.Alloc.Content, sz.Max)
	psz.Alloc.Total = pgrow // parts = content
	psz.SetContentFromTotal(&psz.Alloc)
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
// The given max is the Max size for the given sizes, which are both either
// Content or Total sizes, with 0 for no Max.
func (wb *WidgetBase) growToAllocSize(act, alloc, mx math32.Vector2) (math32.Vector2, bool) {
	change := false
	for d := math32.X; d <= math32.Y; d++ {
		grw := wb.Styles.Grow.Dim(d)
		allocd := alloc.Dim(d)
		actd := act.Dim(d)
		if grw > 0 && allocd > actd {
			grw := min(1, grw)
			nsz := math32.Ceil(actd + grw*(allocd-actd))
			styles.SetClampMax(&nsz, mx.Dim(d))
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
	if iter == 0 { // the sizes of the children from SizeUp
		fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
			cwb.Geom.Size.preferred = cwb.Geom.Size.Actual.Total
			return tree.Continue
		})
	}
	sz := &fr.Geom.Size
	if fr.Parent != nil {
		// A frame is never allocated less than its Min (as a min-width in
		// CSS), even when its parent narrows it (as a block in a column):
		// its children are laid out in at least that, and it then
		// overflows its parent, which scrolls if it can.
		styles.SetClampMinVector(&sz.Alloc.Content, sz.Min)
	}
	styles.SetClampMaxVector(&sz.Alloc.Content, sz.Max) // can't be more than max..
	sz.SetTotalFromContent(&sz.Alloc)
	// as high as its aspect ratio makes it, for its children too
	aspect := fr.sizeAspect(true)
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
	fr.sizeDownOutOfFlow()
	floatsChanged := fr.layoutFloats()
	redo := fr.sizeDownChildren(iter) || floatsChanged
	if redo || wrapped {
		fr.sizeFromChildrenFit(iter, SizeDownPass)
	}
	fr.sizeDownParts(iter)    // no std role, just get sizes
	if fr.sizeAspect(false) { // after its content (which can make it higher)
		aspect = true
	}
	return chg || wrapped || redo || aspect
}

// SizeDownSetAllocs is the key SizeDown step that sets the allocations
// in the children, based on our allocation.  In the default implementation
// this calls SizeDownGrow if there is extra space to grow, or
// SizeDownAllocActual to set the allocations as they currrently are.
func (fr *Frame) SizeDownSetAllocs(iter int) {
	sz := &fr.Geom.Size
	extra := sz.Alloc.Content.Sub(sz.Internal) // note: critical to use internal to be accurate
	// Children that grow only grow within the space of the frame.
	fr.layout.growExtra = fr.childSpace().Sub(sz.Internal)
	if extra.X > 0 || extra.Y > 0 {
		if DebugSettings.LayoutTrace {
			fmt.Println(fr, "SizeDown extra:", extra, "Internal:", sz.Internal, "Alloc:", sz.Alloc.Content)
		}
		fr.sizeDownGrow(iter, extra)
	} else {
		fr.sizeDownAllocActual(iter) // set allocations as is
	}
}

// childSpace returns the content size that the frame allocates to its
// children: its allocation in the dimensions in which it grows, and
// otherwise its actual size, which is all that it takes up even when it
// is allocated more (for example, as the width of the column that it is
// in), so that its children only grow within it.
func (fr *Frame) childSpace() math32.Vector2 {
	sz := &fr.Geom.Size
	csz := sz.Alloc.Content
	if fr.Parent == nil || (fr.Scene != nil && fr.Scene.hasFlag(sceneContentSizing)) {
		return csz
	}
	for d := math32.X; d <= math32.Y; d++ {
		grows := fr.Styles.Grow.Dim(d) > 0 || (d == math32.X && fr.Styles.GrowWrap)
		if !grows && fr.Styles.Overflow.Dim(d) < styles.OverflowAuto {
			csz.SetDim(d, min(csz.Dim(d), sz.Actual.Content.Dim(d)))
		}
	}
	return csz
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
	// The scene is being sized to fit its content, from no size,
	// so it does not overflow: scrollbars would only make it bigger.
	sizing := fr.Parent == nil && fr.Scene != nil && fr.Scene.hasFlag(sceneContentSizing)
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
		if d == math32.X {
			maxSize = fr.overflowWidth(iter, maxSize)
		}
		ofd := maxSize - visSize
		switch fr.Styles.Overflow.Dim(d) {
		// case styles.OverflowVisible:
		// note: this shouldn't happen -- just have this in here for monitoring
		// fmt.Println(ly, "OverflowVisible ERROR -- shouldn't have overflow:", d, ofd)
		case styles.OverflowAuto:
			if ofd < 0.5 || sizing {
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
		sz.SetTotalFromContent(&sz.Actual)
		sz.SetContentFromTotal(&sz.Alloc) // alloc is *decreased* from any increase in space
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
	var widths []float32
	colWidth, colBlock := fr.columnWidth()
	// columns that shrink are narrowed to fit, or widened back to their
	// widths from SizeUp after they were narrower
	widths, shrunk := fr.shrinkWidths()
	if fw := fr.frWidths(alloc.X); fw != nil { // (as fr tracks)
		widths, shrunk = fw, true
	}
	// When the columns fit at the widths of shrinkWidths (widened back to
	// their preferred widths), those are their base widths, and the columns
	// that grow share the width that is left, not the extra width from
	// their earlier (narrower) sizes, which would make them overflow.
	if widths != nil && !shrunk {
		widths = fr.growWidths(widths, alloc.X)
	}
	for exitr := range 2 {
		fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
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
				if cwb.Styles.Grow.Dim(ma) > 0 {
					exd = min(exd, li.growExtra.Dim(ma))
				}
				if exd < 0 {
					exd = 0
				}
				mi := math32.PointDim(cidx, ma) // X, Y
				ci := math32.PointDim(cidx, ca) // Y, X
				md := li.cell(ma, mi, ci)       // X, Y
				cd := li.cell(ca, ci, mi)       // Y, X
				if md == nil || cd == nil {
					continue
				}
				mx := md.Size.Dim(ma)
				asz := mx
				gsum := cd.Grow.Dim(ma)
				if (li.spans || fr.Styles.Display == styles.Grid) && ma == math32.X {
					// the columns of a grid grow consistently in every row,
					// by the grow of the column, also with spanning cells
					// (and in a last row that is not full, whose cells took
					// all of the width of the row)
					gr = md.Grow.X
					gsum = 0
					for c := range li.Shape.X {
						gsum += li.cell(math32.X, c, ci).Grow.X
					}
				}
				ksz.slot.SetDim(ma, 0)
				if gsum > 0 && exd > 0 {
					if gr > gsum {
						fmt.Println(fr, "SizeDownGrowCells error: grow > grow sum:", gr, gsum)
						gr = gsum
					}
					redo = true
					asz = math32.Round(mx + exd*(gr/gsum))
					oasz := asz
					styles.SetClampMax(&asz, ksz.maxTotal().Dim(ma))
					if asz < oasz { // didn't consume its full amount
						ksz.slot.SetDim(ma, oasz)
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
						if DebugSettings.LayoutTrace {
							fmt.Println(fr, "SizeDownGrowCells error: sub alloc > total to alloc:", asz, alloc.Dim(ma))
							fmt.Println("ma:", ma, "mi:", mi, "ci:", ci, "mx:", mx, "gsum:", gsum, "gr:", gr, "ex:", exd, "par act:", sz.Actual.Content.Dim(ma))
							fmt.Println(fr.layout.String())
							fmt.Println(fr.layout.cellsSize())
						}
					}
				}
				if span := fr.colSpan(cwb); ma == math32.X && span > 1 {
					asz = fr.spanAlloc(cidx, span, exd)
				}
				if span := fr.rowSpan(cwb); ma == math32.Y && span > 1 {
					asz = fr.rowSpanAlloc(cidx, span, exd)
				}
				if ma == math32.X && widths != nil {
					// from the widths of shrinkWidths, which those that grow
					// grow from into the width that is left (see above)
					asz = fr.shrinkAlloc(widths, cidx, fr.colSpan(cwb))
				}
				if ma == math32.X && colWidth > 0 {
					// as wide as it can be (see sizeDownAllocActualCells)
					asz = max(asz, ksz.preferred.X)
					asz = narrowTo(cwb, asz, colWidth, colBlock)
				}
				if DebugSettings.LayoutTraceDetail {
					fmt.Println(cwb, ma, "alloc:", asz, "was act:", sz.Actual.Total.Dim(ma), "mx:", mx, "gsum:", gsum, "gr:", gr, "ex:", exd)
				}
				ksz.Alloc.Total.SetDim(ma, asz)
			}
			ksz.SetContentFromTotal(&ksz.Alloc)
			return tree.Continue
		})
		if exn.X == 0 && exn.Y == 0 {
			break
		}
	}
	return redo
}

// spanAlloc returns the allocated width of a grid cell at the given cell
// that spans the given number of columns, given the extra width that the
// columns grow into: the total of the widths of the columns and the gaps
// between them.
func (fr *Frame) spanAlloc(cidx image.Point, span int, extra float32) float32 {
	li := &fr.layout
	gsum := float32(0)
	for c := range li.Shape.X {
		gsum += li.cell(math32.X, c, cidx.Y).Grow.X
	}
	sum := li.Gap.X * float32(span-1)
	for c := range span {
		cd := li.cell(math32.X, cidx.X+c, cidx.Y)
		sum += cd.Size.X
		if gsum > 0 && extra > 0 {
			sum += math32.Round(extra * cd.Grow.X / gsum)
		}
	}
	return sum
}

// shrinkWidths returns the widths of the columns of the frame (its children
// in a row, or the columns of a grid), narrowed to fit within its allocated
// width, in proportion to their width and the [styles.Style.Shrink] of the
// children in them, or nil if it fits or none shrink. It is not for wrapped
// layouts. shrunk is whether they were narrowed to fit, in which case their
// widths are final (they do not also grow), as in CSS.
func (fr *Frame) shrinkWidths() (widths []float32, shrunk bool) {
	li := &fr.layout
	sz := &fr.Geom.Size
	rowOrGrid := fr.Styles.Display == styles.Grid || (fr.Styles.Display == styles.Flex && fr.Styles.Direction == styles.Row)
	if li.Wraps != nil || len(li.Cells) == 0 || !rowOrGrid || fr.Styles.MaxContentWidth {
		// (the content of a frame as wide as its content is not narrowed)
		return nil, false
	}
	n := li.Shape.X
	shrink := make([]float32, n)
	minw := make([]float32, n)
	pref := make([]float32, n) // the widths of the columns from SizeUp
	any := false
	type spanning struct {
		col, span int
		minw      float32
	}
	var spans []spanning
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		c := cwb.Geom.Cell.X
		if c >= n {
			return tree.Continue
		}
		ksz := &cwb.Geom.Size
		if span := fr.colSpan(cwb); span > 1 {
			// a cell that spans columns (as a footer over the columns of a
			// grid) shrinks with them: they shrink when it can, down to
			// the widths that it needs (see below)
			spans = append(spans, spanning{c, span, ksz.minWidth})
			if ksz.minWidth < ksz.preferred.X {
				sh := cwb.Styles.Shrink.X
				if sh == 0 {
					sh = 1
				}
				for k := c; k < min(c+span, n); k++ {
					shrink[k] = max(shrink[k], sh)
				}
				any = true
			}
			return tree.Continue
		}
		// every widget that can be narrower than it is shrinks, by 1
		// unless its style says otherwise
		if ksz.minWidth < ksz.preferred.X {
			sh := cwb.Styles.Shrink.X
			if sh == 0 {
				sh = 1
			}
			shrink[c] = max(shrink[c], sh)
			any = true
		}
		p := ksz.preferred.X
		if cwb.Styles.Grow.X > 0 {
			// a widget that grows up to its Max (as for a width in CSS)
			// starts from it, so that those that have the same Max shrink
			// to the same width, as for the flex basis of CSS
			if mw := fr.growMaxWidth(cwb); mw > 0 {
				p = mw
			}
		}
		pref[c] = max(pref[c], p)
		return tree.Continue
	})
	if !any {
		return nil, false
	}
	// The columns that shrink start from their widths from SizeUp, so
	// that they are as wide as they can be when they were narrower before.
	widths = make([]float32, n)
	cur := float32(0)
	for c := range n {
		widths[c] = li.cell(math32.X, c, 0).Size.X
		cur += widths[c]
		if shrink[c] > 0 {
			widths[c] = max(widths[c], pref[c])
		}
	}
	total := float32(0)
	for c := range n {
		total += widths[c]
	}
	deficit := sz.Internal.X - cur + total - sz.Alloc.Content.X
	if deficit <= 0.5 {
		if total > cur+0.5 {
			return widths, false
		}
		return nil, false
	}
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if c := cwb.Geom.Cell.X; c < n && fr.colSpan(cwb) == 1 && shrink[c] > 0 {
			minw[c] = max(minw[c], cwb.Geom.Size.minWidth)
		}
		return tree.Continue
	})
	// The columns that a cell spans are at least as wide as it needs to be,
	// widened evenly (as in SizeUp), so that its row is not wider than the
	// others (as a header over two columns of a table).
	for _, sp := range spans {
		end := min(sp.col+sp.span, n)
		sum := fr.layout.Gap.X * float32(end-sp.col-1)
		nshrink := 0
		for c := sp.col; c < end; c++ {
			if shrink[c] > 0 {
				sum += minw[c]
				nshrink++
			} else {
				sum += widths[c]
			}
		}
		if extra := sp.minw - sum; extra > 0 && nshrink > 0 {
			for c := sp.col; c < end; c++ {
				if shrink[c] > 0 {
					minw[c] += extra / float32(nshrink)
				}
			}
		}
	}
	for range 2 { // again for those that are not at their minimum
		sum := float32(0)
		for c := range n {
			if shrink[c] > 0 {
				sum += shrink[c] * widths[c]
			}
		}
		if sum <= 0 { // none can shrink any more
			break
		}
		left := deficit
		for c := range n {
			if shrink[c] == 0 {
				continue
			}
			w := widths[c]
			nw := max(math32.Round(w-deficit*shrink[c]*w/sum), minw[c])
			left -= w - nw
			widths[c] = nw
			if nw == minw[c] {
				shrink[c] = 0 // cannot shrink any more
			}
		}
		if left <= 0.5 {
			break
		}
		deficit = left // those that can shrink more do
	}
	return widths, true
}

// frWidths returns the widths of the columns of a grid whose columns all
// grow, from the given width of its content, as the fr tracks of CSS: each
// is its share of the width by its grow, or its minimum (the min-content
// width of the cells in it) when that is more, with the others sharing the
// rest, or nil if it is not such a grid. Their preferred widths do not
// count (as the columns of the top stories of bbc.com, whose wider photos
// made their columns wider).
func (fr *Frame) frWidths(width float32) []float32 {
	li := &fr.layout
	if !fr.FrColumns || fr.Styles.Display != styles.Grid || li.Wraps != nil || fr.Styles.MaxContentWidth {
		return nil
	}
	n := li.Shape.X
	if n < 2 || width <= 0 {
		return nil
	}
	grow := make([]float32, n)
	for c := range n {
		grow[c] = li.cell(math32.X, c, 0).Grow.X
		if grow[c] <= 0 {
			return nil
		}
	}
	minw := make([]float32, n)
	type spanning struct {
		col, span int
		minw      float32
	}
	var spans []spanning
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		c := cwb.Geom.Cell.X
		if c >= n {
			return tree.Continue
		}
		if span := fr.colSpan(cwb); span > 1 {
			spans = append(spans, spanning{c, span, cwb.Geom.Size.minWidth})
		} else {
			minw[c] = max(minw[c], cwb.Geom.Size.minWidth)
		}
		return tree.Continue
	})
	for _, sp := range spans { // spread over their columns, as in SizeUp
		end := min(sp.col+sp.span, n)
		sum := li.Gap.X * float32(end-sp.col-1)
		for c := sp.col; c < end; c++ {
			sum += minw[c]
		}
		if extra := sp.minw - sum; extra > 0 {
			for c := sp.col; c < end; c++ {
				minw[c] += extra / float32(end-sp.col)
			}
		}
	}
	fixed := make([]bool, n) // (the width is without the gaps)
	widths := make([]float32, n)
	for range n {
		left, flex := width, float32(0)
		for c := range n {
			if fixed[c] {
				left -= minw[c]
			} else {
				flex += grow[c]
			}
		}
		if flex <= 0 {
			break
		}
		unit := max(left, 0) / flex
		changed := false
		for c := range n {
			if !fixed[c] && minw[c] > unit*grow[c] {
				fixed[c], changed = true, true
			}
		}
		if changed {
			continue
		}
		for c := range n {
			if fixed[c] {
				widths[c] = minw[c]
			} else {
				widths[c] = unit * grow[c]
			}
		}
		return widths
	}
	return minw
}

// shrinkAlloc returns the allocated width of the given child in the column
// at the given index, from the given shrunk column widths.
func (fr *Frame) shrinkAlloc(widths []float32, cidx image.Point, span int) float32 {
	if span <= 1 {
		return widths[cidx.X]
	}
	w := float32(0)
	for c := cidx.X; c < min(cidx.X+span, len(widths)); c++ {
		w += widths[c]
	}
	return w + float32(span-1)*fr.layout.Gap.X
}

// columnWidth returns the width of the content of the frame if it is a
// column (block is true) or a wrapped row, within which each of its
// children is allocated at most that width (see [narrowTo]), so that its
// content (such as text) can wrap in it, or -1 otherwise.
func (fr *Frame) columnWidth() (width float32, block bool) {
	s := &fr.Styles
	if s.Display != styles.Flex {
		return -1, false
	}
	if s.Direction == styles.Column && !s.Wrap {
		return fr.Geom.Size.Alloc.Content.X - fr.Geom.Size.InnerSpace.X, true
	}
	if s.Direction == styles.Row && s.Wrap {
		return fr.Geom.Size.Alloc.Content.X, false
	}
	return -1, false
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
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		ksz := cwb.Geom.Size.Actual.Total
		if d == math32.X && cwb.HasChildren() {
			// A frame in a wrapped row is as wide as it can be in it
			// (see sizeDownAllocActualCells): its preferred width, within
			// the row, which its line needs room for, as for the
			// hypothetical main size of a flex item in CSS.
			ksz.X = max(ksz.X, min(cwb.Geom.Size.preferred.X, fit))
		}
		if first {
			n = 1
			sum = ksz.Dim(d) + gap
			first = false
			return tree.Continue
		}
		// The sizes of the items are each rounded up to whole dots, so
		// items whose exact sizes fit together (as widths in CSS that add
		// up to that of their container) can be up to a dot each over it.
		if sum+ksz.Dim(d) > fit+float32(n+1) { // sum has the gap before it
			if DebugSettings.LayoutTraceDetail {
				fmt.Println(fr, "wrapped:", i, sum, ksz.Dim(d), fit)
			}
			wraps = append(wraps, n)
			sum = ksz.Dim(d) + gap
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
			ksz.SetContentFromTotal(&ksz.Alloc)
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
		ksz.SetContentFromTotal(&ksz.Alloc)
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
	// In a column that is narrower than its content, the children get at
	// most its width (but not less than their minimum), so that text can
	// wrap in it.
	colWidth, colBlock := fr.columnWidth()
	widths, _ := fr.shrinkWidths()
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		ksz := &cwb.Geom.Size
		cidx := cwb.Geom.Cell
		for ma := math32.X; ma <= math32.Y; ma++ { // main axis = X then Y
			ca := ma.Other()                 // cross axis = Y then X
			mi := math32.PointDim(cidx, ma)  // X, Y
			ci := math32.PointDim(cidx, ca)  // Y, X
			md := fr.layout.cell(ma, mi, ci) // X, Y
			if md == nil {
				continue
			}
			asz := md.Size.Dim(ma)
			if span := fr.colSpan(cwb); ma == math32.X && span > 1 {
				asz = fr.spanAlloc(cidx, span, 0)
			}
			if span := fr.rowSpan(cwb); ma == math32.Y && span > 1 {
				asz = fr.rowSpanAlloc(cidx, span, 0)
			}
			if ma == math32.X && widths != nil {
				asz = fr.shrinkAlloc(widths, cidx, fr.colSpan(cwb))
			}
			if ma == math32.X && colWidth > 0 {
				// as wide as it can be: its preferred width, within the
				// column, even after it was narrower in an earlier iteration
				asz = max(asz, cwb.Geom.Size.preferred.X)
				asz = narrowTo(cwb, asz, colWidth, colBlock)
			}
			ksz.Alloc.Total.SetDim(ma, asz)
		}
		ksz.SetContentFromTotal(&ksz.Alloc)
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
			ksz.SetContentFromTotal(&ksz.Alloc)
		}
		return
	}
	// note: allocate everyone in case they are flipped to top
	// need a new layout if size is actually different
	fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		ksz := &cwb.Geom.Size
		ksz.Alloc.Total = asz
		ksz.SetContentFromTotal(&ksz.Alloc)
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
		ksz.SetContentFromTotal(&ksz.Alloc)
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
	wb.sizeAspect(false) // for its final width
	wb.styleSizeUpdate() // now that sizes are stable, ensure styling based on size is updated
	wb.sizeFinalParts()
	sz.SetTotalFromContent(&sz.Actual)
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
	act, change := wb.growToAllocSize(sz.Actual.Total, sz.Alloc.Total, sz.maxTotal())
	if change {
		if DebugSettings.LayoutTrace {
			fmt.Println(wb, "GrowToAlloc:", sz.Alloc.Total, "from actual:", sz.Actual.Total)
		}
		sz.Actual.Total = act // already has max constraint
		sz.SetContentFromTotal(&sz.Actual)
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
	fr.sizeDownOutOfFlow() // their final size is still that of their inset
	fr.layoutFloats()      // with the final sizes
	fr.sizeFromChildrenFit(0, SizeFinalPass)
	fr.growToAlloc()
	fr.sizeAspect(false) // for its final width
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
	psz := wb.percentBase(pwb)
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

// alignSize returns the size that the widget is aligned within:
// its allocation, or the slot that its Max size limited it within.
func (wb *WidgetBase) alignSize() math32.Vector2 {
	sz := &wb.Geom.Size
	return sz.Alloc.Total.Max(sz.slot)
}

// positionWithinAllocMainX positions the widget within its allocation,
// in a row (main axis X), and within the given cross size (the height
// of its line), if larger, for its alignment on the cross axis.
func (wb *WidgetBase) positionWithinAllocMainX(pos math32.Vector2, parJustify, parAlign styles.Aligns, cross float32) {
	sz := &wb.Geom.Size
	asz := wb.alignSize()
	asz.Y = max(asz.Y, cross)
	pos.X += styles.AlignPos(styles.ItemAlign(parJustify, wb.Styles.Justify.Self), sz.Actual.Total.X, asz.X)
	pos.Y += styles.AlignPos(styles.ItemAlign(parAlign, wb.Styles.Align.Self), sz.Actual.Total.Y, asz.Y)
	wb.Geom.RelPos = pos
	if DebugSettings.LayoutTrace {
		fmt.Println(wb, "Position within Main=X:", pos)
	}
}

// positionWithinAllocMainY positions the widget within its allocation,
// in a column (main axis Y), and within the given cross size (the width
// of its line), if larger, for its alignment on the cross axis.
func (wb *WidgetBase) positionWithinAllocMainY(pos math32.Vector2, parJustify, parAlign styles.Aligns, cross float32) {
	sz := &wb.Geom.Size
	asz := wb.alignSize()
	asz.X = max(asz.X, cross)
	pos.Y += styles.AlignPos(styles.ItemAlign(parJustify, wb.Styles.Justify.Self), sz.Actual.Total.Y, asz.Y)
	pos.X += styles.AlignPos(styles.ItemAlign(parAlign, wb.Styles.Align.Self), sz.Actual.Total.X, asz.X)
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
		fr.positionWithinAllocMainY(math32.Vector2{}, fr.Styles.Justify.Items, fr.Styles.Align.Items, 0)
	}
	fr.ConfigScrolls() // and configure the scrolls
	if fr.Styles.Display == styles.Stacked {
		fr.positionStacked()
	} else {
		fr.positionCells()
		fr.positionStatic()
		fr.positionChildren()
	}
	fr.positionParts()
}

// positionStatic sets the positions in the layout of the children of the
// frame that are out of its flow (absolute and fixed ones), which they have
// in the dimensions in which their inset is not set (their static position
// in CSS): right after the child in the flow before them, below it in a
// column (as the dropdown panels of the menus of MDN, below their buttons,
// which covered them), and beside it in a row, or at the start.
func (fr *Frame) positionStatic() {
	var prev *WidgetBase
	column := fr.Styles.Display != styles.Grid && fr.Styles.Direction == styles.Column
	fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if cwb.StateIs(states.Invisible) {
			return tree.Continue
		}
		if !cwb.outOfFlow() {
			prev = cwb
			return tree.Continue
		}
		var pos math32.Vector2
		if prev != nil {
			pos = prev.Geom.RelPos
			if column {
				pos.X = 0
				pos.Y += prev.Geom.Size.Actual.Total.Y
			} else {
				pos.X += prev.Geom.Size.Actual.Total.X
			}
		}
		cwb.Geom.RelPos = pos
		return tree.Continue
	})
}

func (fr *Frame) positionCells() {
	if fr.Styles.Display == styles.Grid && fr.layout.rowSpans {
		fr.positionCellsGrid()
		return
	}
	if fr.Styles.Display == styles.Flex && fr.Styles.Direction == styles.Column {
		fr.positionCellsMainY()
		return
	}
	fr.positionCellsMainX()
}

// Main axis = X
// positionCellsGrid positions the cells of a grid that has cells that
// span rows, at the positions of their columns and rows, which are as wide
// and high as the widest and highest of their cells that do not span.
func (fr *Frame) positionCellsGrid() {
	li := &fr.layout
	gap := li.Gap
	sz := &fr.Geom.Size
	colW := make([]float32, li.Shape.X)
	rowH := make([]float32, li.Shape.Y)
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		c := cwb.Geom.Cell
		alloc := cwb.Geom.Size.Alloc.Total
		if fr.colSpan(cwb) == 1 && c.X < len(colW) {
			colW[c.X] = max(colW[c.X], alloc.X)
		}
		if fr.rowSpan(cwb) == 1 && c.Y < len(rowH) {
			rowH[c.Y] = max(rowH[c.Y], alloc.Y)
		}
		return tree.Continue
	})
	// the columns that cells span are widened evenly to fit them (as for
	// the sizes of the cells), as when all of the cells span columns (as
	// the stories of npr.org, of 8 and 4 of 12 columns)
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		c, span := cwb.Geom.Cell.X, fr.colSpan(cwb)
		if span <= 1 || c+span > len(colW) {
			return tree.Continue
		}
		sum := gap.X * float32(span-1)
		for k := range span {
			sum += colW[c+k]
		}
		if extra := cwb.Geom.Size.Alloc.Total.X - sum; extra > 0 {
			for k := range span {
				colW[c+k] += extra / float32(span)
			}
		}
		return tree.Continue
	})
	var stPos math32.Vector2
	stPos.X = styles.AlignPos(fr.Styles.Justify.Content, sz.Internal.X, sz.Actual.Content.X)
	stPos.Y = styles.AlignPos(fr.Styles.Align.Content, sz.Internal.Y, sz.Actual.Content.Y)
	colX := make([]float32, li.Shape.X)
	rowY := make([]float32, li.Shape.Y)
	for c := range colX {
		if c > 0 {
			colX[c] = colX[c-1] + colW[c-1] + gap.X
		}
	}
	for r := range rowY {
		if r > 0 {
			rowY[r] = rowY[r-1] + rowH[r-1] + gap.Y
		}
	}
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		c := cwb.Geom.Cell
		pos := stPos.Add(math32.Vec2(colX[min(c.X, len(colX)-1)], rowY[min(c.Y, len(rowY)-1)]))
		cwb.positionWithinAllocMainX(pos, fr.Styles.Justify.Items, fr.Styles.Align.Items, 0)
		return tree.Continue
	})
}

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
	spaces := fr.lineSpaces(math32.X, sz.Actual.Content.X)
	crosses := fr.lineCrosses(math32.X)
	baselines := fr.lineBaselines()
	// the cells of a grid are at the positions of their columns, which a
	// cell narrower than its column (as one with a max width) does not move
	var colW []float32
	if fr.Styles.Display == styles.Grid && fr.layout.Shape.X > 0 {
		colW = make([]float32, fr.layout.Shape.X)
		fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
			if c := cwb.Geom.Cell.X; c < len(colW) && fr.colSpan(cwb) == 1 {
				colW[c] = max(colW[c], cwb.Geom.Size.Alloc.Total.X)
			}
			return tree.Continue
		})
	}
	pos := stPos
	var lastSz math32.Vector2
	idx := 0
	line := 0
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cidx := cwb.Geom.Cell
		if cidx.X == 0 && idx > 0 {
			pos.X = stPos.X
			pos.Y += lastSz.Y + gap.Y
			line++
		}
		if spaces != nil && line < len(spaces) {
			if cidx.X == 0 {
				pos.X += spaces[line].X
			} else {
				pos.X += spaces[line].Y
			}
		}
		cwb.positionWithinAllocMainX(pos, fr.Styles.Justify.Items, fr.Styles.Align.Items, crosses[min(line, len(crosses)-1)])
		if baselines != nil && styles.ItemAlign(fr.Styles.Align.Items, cwb.Styles.Align.Self) == styles.Baseline {
			// the baselines of the items of the line with text are at the
			// same place, within the line; the others are at its end
			cross := crosses[min(line, len(crosses)-1)]
			h := cwb.Geom.Size.Actual.Total.Y
			if b, ok := cwb.firstBaseline(); ok {
				off := min(cross-baselines[min(line, len(baselines)-1)]-b, cross-h)
				cwb.Geom.RelPos.Y = pos.Y + math32.Floor(max(off, 0))
			} else {
				cwb.Geom.RelPos.Y = pos.Y + math32.Floor(max(cross-h, 0))
			}
		}
		alloc := cwb.Geom.Size.Alloc.Total
		if c := cidx.X; colW != nil && c < len(colW) && fr.colSpan(cwb) == 1 {
			pos.X += max(alloc.X, colW[c]) + gap.X
		} else {
			pos.X += alloc.X + gap.X
		}
		lastSz = alloc
		idx++
		return tree.Continue
	})
}

// lineCrosses returns, for each line of the children along the given main
// axis, its size on the cross axis: the largest allocation of its children,
// within which they are aligned on the cross axis (as a child that is
// centered in a column next to a child that grows to its width). It has
// one line (of size 0) if there are no children.
func (fr *Frame) lineCrosses(ma math32.Dims) []float32 {
	ca := ma.Other()
	crosses := []float32{0}
	idx := 0
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if math32.PointDim(cwb.Geom.Cell, ma) == 0 && idx > 0 {
			crosses = append(crosses, 0)
		}
		ln := len(crosses) - 1
		crosses[ln] = max(crosses[ln], cwb.Geom.Size.Alloc.Total.Dim(ca))
		idx++
		return tree.Continue
	})
	return crosses
}

// lineBaselines returns, for each line of the children of the frame in
// a row, the largest distance from the baselines of its children that are
// aligned on their baselines (see [styles.Baseline]) and have text to
// their bottoms: they are aligned on a baseline that far from the end of
// the line (and the children without text are at the end of the line, as
// with [styles.End]). It returns nil if no children are aligned on their
// baselines.
func (fr *Frame) lineBaselines() []float32 {
	var bases []float32
	any := false
	idx := 0
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if cwb.Geom.Cell.X == 0 || idx == 0 {
			bases = append(bases, 0)
		}
		idx++
		if styles.ItemAlign(fr.Styles.Align.Items, cwb.Styles.Align.Self) != styles.Baseline {
			return tree.Continue
		}
		any = true
		if b, ok := cwb.firstBaseline(); ok {
			ln := len(bases) - 1
			bases[ln] = max(bases[ln], cwb.Geom.Size.Actual.Total.Y-b)
		}
		return tree.Continue
	})
	if !any {
		return nil
	}
	return bases
}

// firstBaseline returns the distance from the top of the widget to the
// baseline of its first line of text: that of a [Text], or of the first
// child of a [Frame] that has one, and whether there is one.
func (wb *WidgetBase) firstBaseline() (float32, bool) {
	top := wb.Styles.BoxSpace().Top
	switch w := wb.This.(type) {
	case *Text:
		if w.paintText == nil || len(w.paintText.Lines) == 0 {
			return 0, false
		}
		ls := w.paintText
		return top + ls.Lines[0].Offset.Y - ls.Bounds.Min.Y, true
	case *Frame:
		if b, ok := w.rowBaseline(); ok {
			return top + b, true
		}
		var base float32
		found := false
		w.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
			if b, ok := cwb.firstBaseline(); ok {
				base, found = top+b, true
				return tree.Break
			}
			return tree.Continue
		})
		return base, found
	}
	return 0, false
}

// rowBaseline returns the baseline of the first line of a frame that is
// a row aligned on baselines (see [Frame.lineBaselines]), from the top of
// its content, and whether it has one: where its items with text are
// aligned, in the line of the size of its largest item.
func (fr *Frame) rowBaseline() (float32, bool) {
	if fr.Styles.Direction != styles.Row || fr.Styles.Align.Items != styles.Baseline {
		return 0, false
	}
	var cross, desc float32
	found := false
	idx := 0
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if cwb.Geom.Cell.X == 0 && idx > 0 {
			return tree.Break // the first line only
		}
		idx++
		h := cwb.Geom.Size.Actual.Total.Y
		cross = max(cross, h)
		if styles.ItemAlign(fr.Styles.Align.Items, cwb.Styles.Align.Self) != styles.Baseline {
			return tree.Continue
		}
		if b, ok := cwb.firstBaseline(); ok {
			desc, found = max(desc, h-b), true
		}
		return tree.Continue
	})
	if !found {
		return 0, false
	}
	return cross - desc, true
}

// lineSpaces returns, for each line of the children along the given main
// axis, the space before the first child (X) and between the children (Y)
// for the distributed [styles.Style.Justify] Content values (SpaceBetween,
// SpaceAround, SpaceEvenly) of the frame, given its content size along
// the axis. It returns nil for the other values, which position all of
// the children as a block.
func (fr *Frame) lineSpaces(ma math32.Dims, size float32) []math32.Vector2 {
	jc := fr.Styles.Justify.Content
	if jc != styles.SpaceBetween && jc != styles.SpaceAround && jc != styles.SpaceEvenly {
		return nil
	}
	gap := fr.layout.Gap.Dim(ma)
	var widths []float32
	var counts []int
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if math32.PointDim(cwb.Geom.Cell, ma) == 0 || len(widths) == 0 {
			widths = append(widths, 0)
			counts = append(counts, 0)
		}
		ln := len(widths) - 1
		if counts[ln] > 0 {
			widths[ln] += gap
		}
		widths[ln] += cwb.Geom.Size.Alloc.Total.Dim(ma)
		counts[ln]++
		return tree.Continue
	})
	spaces := make([]math32.Vector2, len(widths))
	for ln, w := range widths {
		extra := size - w
		n := float32(counts[ln])
		if extra <= 0 || n == 0 {
			continue
		}
		switch jc {
		case styles.SpaceBetween:
			if n > 1 {
				spaces[ln].Y = extra / (n - 1)
			}
		case styles.SpaceAround:
			spaces[ln].Y = extra / n
			spaces[ln].X = spaces[ln].Y / 2
		case styles.SpaceEvenly:
			spaces[ln].Y = extra / (n + 1)
			spaces[ln].X = spaces[ln].Y
		}
	}
	return spaces
}

// Main axis = Y
func (fr *Frame) positionCellsMainY() {
	if fr.positionFloats() {
		return
	}
	gap := fr.layout.Gap
	sz := &fr.Geom.Size
	if DebugSettings.LayoutTraceDetail {
		fmt.Println(fr, "PositionCells, actual", sz.Actual.Content, "internal:", sz.Internal)
	}
	var lastSz math32.Vector2
	var stPos math32.Vector2
	stPos.Y = styles.AlignPos(fr.Styles.Justify.Content, sz.Internal.Y, sz.Actual.Content.Y)
	stPos.X = styles.AlignPos(fr.Styles.Align.Content, sz.Internal.X, sz.Actual.Content.X)
	spaces := fr.lineSpaces(math32.Y, sz.Actual.Content.Y)
	crosses := fr.lineCrosses(math32.Y)
	pos := stPos
	idx := 0
	line := 0
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cidx := cwb.Geom.Cell
		if cidx.Y == 0 && idx > 0 {
			pos.Y = stPos.Y
			pos.X += lastSz.X + gap.X
			line++
		}
		if spaces != nil && line < len(spaces) {
			if cidx.Y == 0 {
				pos.Y += spaces[line].X
			} else {
				pos.Y += spaces[line].Y
			}
		}
		cwb.positionWithinAllocMainY(pos, fr.Styles.Justify.Items, fr.Styles.Align.Items, crosses[min(line, len(crosses)-1)])
		alloc := cwb.Geom.Size.Alloc.Total
		pos.Y += alloc.Y + gap.Y
		lastSz = alloc
		idx++
		return tree.Continue
	})
}

func (fr *Frame) positionStacked() {
	avail := fr.Geom.Size.Alloc.Content.Sub(fr.Geom.Size.InnerSpace)
	fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cwb.Geom.RelPos.SetZero()
		// a child that is smaller than the stack is aligned in it by its
		// Justify.Self and Align.Self, if they are set (as the logo of
		// Google, in a grid cell that it is centered in)
		for d, al := range []styles.Aligns{cwb.Styles.Justify.Self, cwb.Styles.Align.Self} {
			dim := math32.Dims(d)
			extra := avail.Dim(dim) - cwb.Geom.Size.Actual.Total.Dim(dim)
			if extra <= 0 {
				continue
			}
			switch al {
			case styles.Center:
				cwb.Geom.RelPos.SetDim(dim, math32.Floor(extra/2))
			case styles.End:
				cwb.Geom.RelPos.SetDim(dim, extra)
			}
		}
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
	wb.Geom.Pos.Content = wb.Geom.Pos.Total.Add(wb.Geom.contentOffset)
}

func (wb *WidgetBase) setPosFromParent() {
	pwb := wb.parentWidget()
	var parPos math32.Vector2
	if pwb != nil {
		// critical that parent adds here but not to self; the scroll
		// offset is in whole dots, so that the content is always at the
		// same position relative to the pixels (otherwise lines thinner
		// than a pixel come and go while scrolling)
		parPos = pwb.Geom.Pos.Content.Add(pwb.Geom.Scroll.Round())
	}
	switch {
	case pwb != nil && wb.outOfFlow():
		wb.setPosOutOfFlow(parPos)
	case wb.Styles.Position == styles.PositionRelative:
		wb.Geom.Pos.Total = wb.Geom.RelPos.Add(parPos).Add(wb.relativeOffset())
	case pwb != nil && wb.Styles.Position == styles.PositionSticky:
		wb.Geom.Pos.Total = wb.Geom.RelPos.Add(parPos).Add(wb.stickyOffset(pwb, wb.Geom.RelPos.Add(parPos)))
	default:
		wb.Geom.Pos.Total = wb.Geom.RelPos.Add(parPos)
	}
	wb.Geom.Pos.Total.SetAdd(wb.translateOffset())
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

// childClipBBox returns the bounding box that the children of the widget
// are clipped to: its content box, or, if it does not clip its children
// ([styles.Style.NoClip]), that of the nearest widget that it is in that does.
func (wb *WidgetBase) childClipBBox() image.Rectangle {
	clip := wb
	for clip.noClip() {
		p := clip.parentWidget()
		if p == nil {
			break
		}
		clip = p
	}
	return clip.Geom.ContentBBox
}

// noClip returns whether the widget does not clip its children
// ([styles.Style.NoClip], with overflow that does not scroll).
func (wb *WidgetBase) noClip() bool {
	s := &wb.Styles
	return s.NoClip && s.Overflow.X < styles.OverflowAuto && s.Overflow.Y < styles.OverflowAuto
}

func (wb *WidgetBase) setBBoxes() {
	pwb := wb.parentWidget()
	var parBB image.Rectangle
	if pwb == nil { // scene
		sz := &wb.Geom.Size
		wb.Geom.TotalBBox = math32.RectFromPosSizeMax(math32.Vector2{}, sz.Alloc.Total)
		wb.Geom.ContentBBox = math32.RectFromPosSizeMax(wb.Geom.contentOffset, sz.Alloc.Content)
		if DebugSettings.LayoutTrace {
			fmt.Println(wb, "Total BBox:", wb.Geom.TotalBBox)
			fmt.Println(wb, "Content BBox:", wb.Geom.ContentBBox)
		}
	} else {
		clip := pwb
		if wb.outOfFlow() { // not clipped by its parent (see [WidgetBase.RenderWidget])
			if gp := pwb.parentWidget(); gp != nil {
				clip = gp
			}
		}
		parBB = clip.childClipBBox()
		bb := wb.Geom.totalRect()
		if m := wb.Styles.Margin.Dots(); m.Left < 0 || m.Right < 0 || m.Top < 0 || m.Bottom < 0 {
			// The box inside negative margins (as for a CSS float pulled
			// beside another with a negative margin) is larger than the
			// total box: it is visible where it is, not only in that.
			pos := wb.Geom.Pos.Total.Add(math32.Vec2(m.Left, m.Top))
			sz := wb.Geom.Size.Actual.Total.Sub(math32.Vec2(m.Left+m.Right, m.Top+m.Bottom))
			bb = bb.Union(math32.RectFromPosSizeMax(pos, sz))
		}
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
	if pos.Y < tbb.Min.Y { // scrolled off top
		srcBBox.Min.Y = tbb.Min.Y - pos.Y
	}
	sz := srcBBox.Size()
	if sz.X <= 0 || sz.Y <= 0 {
		empty = true
	}
	return
}
