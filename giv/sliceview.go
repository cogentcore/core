// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"goki.dev/colors"
	"goki.dev/enums"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/abilities"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/goosi/mimedata"
	"goki.dev/grr"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/filecat"
)

////////////////////////////////////////////////////////
//  SliceView

// SliceView represents a slice, creating an interactive viewer / editor of the
// elements as rows in a table.  Widgets to show the index / value pairs, within an
// overall frame.
// Set to ReadOnly for select-only mode, which emits WidgetSig WidgetSelected
// signals when selection is updated.
type SliceView struct {
	SliceViewBase

	// optional styling function
	StyleFunc SliceViewStyleFunc `copy:"-" view:"-" json:"-" xml:"-"`
}

// check for interface impl
var _ SliceViewer = (*SliceView)(nil)

// SliceViewStyleFunc is a styling function for custom styling /
// configuration of elements in the view.  If style properties are set
// then you must call w.AsNode2dD().SetFullReRender() to trigger
// re-styling during re-render
type SliceViewStyleFunc func(w gi.Widget, s *styles.Style, row int)

func (sv *SliceView) StyleRow(w gi.Widget, idx, fidx int) {
	if sv.StyleFunc != nil {
		sv.StyleFunc(w, &w.AsWidget().Styles, idx)
	}
}

////////////////////////////////////////////////////////
//  SliceViewBase

// note on implementation:
// * Use ReConfig whenever elements added or deleted.
//   Value-only changes use Updatewidgets.
// * ConfigRows creates VisRows number of Values and associated Widgets
//   to represent the slice values.
// * UpdateWidgets goes through the existing values and widgets and updates
//   them based on the current starting index and state.
// * It is tricky to compute VisRows: depends on how big each row is, which
//   means you need to render it out and measure, then divide the available
//   layout space by that size.  ConfigSliceView starts by allocating a single
//   "test case" row, and then VisRowsAvail uses that first row size to do the
//   the math.
// * ConfigWidget (ReConfig) calls ConfigSliceView which decides what level of
//   config to perform: first ConfigOneRow, then if needs diff # of rows, ConfigRows,
//   else UpdateWidgets.
// * Multiple iterations are required to get this to work: ShowLayoutIter on Scene
//   manages these iterations to get it to work.

// SliceViewFlags extend WidgetFlags to hold SliceView state
type SliceViewFlags gi.WidgetFlags //enums:bitflag -trim-prefix SliceView

const (
	// flagged after first configuration
	SliceViewConfiged SliceViewFlags = SliceViewFlags(gi.WidgetFlagsN) + iota

	// if true, user cannot add elements to the slice
	SliceViewNoAdd

	// if true, user cannot delete elements from the slice
	SliceViewNoDelete

	// if the type we're viewing has its own CtxtMenu property defined, should we also still show the view's standard context menu?
	SliceViewShowViewCtxtMenu

	// whether the slice is actually an array -- no modifications -- set by SetSlice
	SliceViewIsArray

	// whether to show index or not
	SliceViewShowIndex

	// support key navigation when ReadOnly (default true) -- no focus really plausible in ReadOnly case, so it uses a low-pri capture of up / down events
	SliceViewReadOnlyKeyNav

	// editing-mode select rows mode
	SliceViewSelectMode

	// if view is ReadOnly, default selection mode is to choose one row only -- if this is true, standard multiple selection logic with modifier keys is instead supported
	SliceViewReadOnlyMultiSel

	// guard for recursive focus grabbing
	SliceViewInFocusGrab

	// guard for recursive rebuild
	SliceViewInFullRebuild
)

// SliceViewer is the interface used by SliceViewBase to
// support any abstractions needed for different types of slice views.
type SliceViewer interface {
	// AsSliceViewBase returns the base for direct access to relevant fields etc
	AsSliceViewBase() *SliceViewBase

	// SliceGrid returns the SliceGrid grid frame widget,
	// which contains all the fields and values
	SliceGrid() *gi.Frame

	// ScrollBar returns the SliceGrid scrollbar
	ScrollBar() *gi.Slider

	// RowWidgetNs returns number of widgets per row and
	// offset for index label
	RowWidgetNs() (nWidgPerRow, idxOff int)

	// UpdtSliceSize updates the current size of the slice
	// and sets SliceSize if changed.
	UpdtSliceSize() int

	// NeedsConfigRows returns true if a call to ConfigRows is needed,
	// whenever the current layout allocation requires a different
	// number of rows than currently configured.
	NeedsConfigRows() bool

	// ConfigRows configures VisRows worth of widgets
	// to display slice data.  It should only be called
	// when NeedsConfigRows is true: when VisRows changes.
	ConfigRows(sc *gi.Scene)

	// UpdateWidgets updates the row widget display to
	// represent the current state of the slice data,
	// including which range of data is being displayed.
	// This is called for scrolling, navigation etc.
	UpdateWidgets()

	// ConfigOneRow configures one row, just to get sizing,
	// only called at the start.
	ConfigOneRow(sc *gi.Scene)

	// StyleRow calls a custom style function on given row (and field)
	StyleRow(w gi.Widget, idx, fidx int)

	// RowFirstWidget returns the first widget for given row
	// (could be index or not) -- false if out of range
	RowFirstWidget(row int) (*gi.WidgetBase, bool)

	// RowGrabFocus grabs the focus for the first focusable
	// widget in given row.
	// returns that element or nil if not successful
	// note: grid must have already rendered for focus to be grabbed!
	RowGrabFocus(row int) *gi.WidgetBase

	// SelectRowWidgets sets the selection state of given row of widgets
	SelectRowWidgets(row int, sel bool)

	// SliceNewAt inserts a new blank element at given
	// index in the slice. -1 means the end.
	SliceNewAt(idx int)

	// SliceDeleteAt deletes element at given index from slice
	// if updt is true, then update the grid after
	SliceDeleteAt(idx int)

	// MimeDataType returns the data type for mime clipboard
	// (copy / paste) data e.g., filecat.DataJson
	MimeDataType() string

	// CopySelToMime copies selected rows to mime data
	CopySelToMime() mimedata.Mimes

	// PasteAssign assigns mime data (only the first one!) to this idx
	PasteAssign(md mimedata.Mimes, idx int)

	// PasteAtIdx inserts object(s) from mime data at
	// (before) given slice index
	PasteAtIdx(md mimedata.Mimes, idx int)

	// ItemCtxtMenu pulls up the context menu for given slice index
	ItemCtxtMenu(idx int)

	// StdCtxtMenu generates the standard context menu for this view
	StdCtxtMenu(m *gi.Scene, idx int)
}

// SliceViewBase is the base for SliceView and TableView and any other viewers
// of array-like data.  It automatically computes the number of rows that fit
// within its allocated space, and manages the offset view window into the full
// list of items, and supports row selection, copy / paste, Drag-n-Drop, etc.
// Set to ReadOnly for select-only mode, which emits WidgetSig WidgetSelected
// signals when selection is updated.
type SliceViewBase struct {
	gi.Frame

	// the slice that we are a view onto -- must be a pointer to that slice
	Slice any `set:"-" copy:"-" view:"-" json:"-" xml:"-"`

	// optional mutex that, if non-nil, will be used around any updates that read / modify the underlying Slice data -- can be used to protect against random updating if your code has specific update points that can be likewise protected with this same mutex
	ViewMu *sync.Mutex `copy:"-" view:"-" json:"-" xml:"-"`

	// Changed indicates whether the underlying slice
	// has been edited in any way
	Changed bool `set:"-"`

	// non-ptr reflect.Value of the slice
	SliceNPVal reflect.Value `copy:"-" view:"-" json:"-" xml:"-"`

	// Value for the slice itself, if this was created within value view framework -- otherwise nil
	SliceValView Value `copy:"-" view:"-" json:"-" xml:"-"`

	// Value representations of the slice values
	Values []Value `copy:"-" view:"-" json:"-" xml:"-"`

	// current selection value -- initially select this value if set
	SelVal any `copy:"-" view:"-" json:"-" xml:"-"`

	// index of currently-selected item, in ReadOnly mode only
	SelIdx int `copy:"-" json:"-" xml:"-"`

	// list of currently-selected slice indexes
	SelIdxs map[int]struct{} `copy:"-"`

	// list of currently-dragged indexes
	DraggedIdxs []int `copy:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `copy:"-" json:"-" xml:"-"`

	// height of a single row
	RowHeight float32 `edit:"-" copy:"-" json:"-" xml:"-"`

	// the height of grid from last layout -- determines when update needed
	LayoutHeight float32 `copy:"-" view:"-" json:"-" xml:"-"`

	// total number of rows visible in allocated display size
	VisRows int `edit:"-" copy:"-" json:"-" xml:"-"`

	// starting slice index of visible rows
	StartIdx int `edit:"-" copy:"-" json:"-" xml:"-"`

	// the number of rows rendered -- determines update
	RenderedRows int `copy:"-" view:"-" json:"-" xml:"-"`

	// size of slice
	SliceSize int `edit:"-" copy:"-" json:"-" xml:"-"`

	// temp idx state for e.g., dnd
	CurIdx int `copy:"-" view:"-" json:"-" xml:"-"`

	// ElVal is a Value representation of the underlying element type
	// which is used whenever there are no slice elements available
	ElVal reflect.Value `copy:"-" view:"-" json:"-" xml:"-"`
}

func (sv *SliceViewBase) FlagType() enums.BitFlag {
	return SliceViewFlags(sv.Flags)
}

func (sv *SliceViewBase) OnInit() {
	sv.SliceViewBaseInit()
}

func (sv *SliceViewBase) SliceViewBaseInit() {
	sv.SetFlag(false, SliceViewSelectMode)
	sv.SetFlag(true, SliceViewShowIndex)
	sv.SetFlag(true, SliceViewReadOnlyKeyNav)

	sv.HandleSliceViewEvents()

	sv.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.FocusWithinable)
		s.SetMainAxis(mat32.Y)
		s.Grow.Set(1, 1)
	})
	sv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(sv) {
		case "grid-lay": // grid layout
			w.Style(func(s *styles.Style) {
				s.SetMainAxis(mat32.X)
				s.Grow.Set(1, 1) // for this to work, ALL layers above need it too
			})
		case "grid-lay/grid": // slice grid
			sg := w.(*gi.Frame)
			sg.Stripes = gi.RowStripes
			sg.Style(func(s *styles.Style) {
				nWidgPerRow, _ := sv.RowWidgetNs()
				s.Display = styles.DisplayGrid
				s.Columns = nWidgPerRow
				s.Overflow.Set(styles.OverflowAuto)
				s.Grow.Set(1, 1)
				s.Min.X.Em(20)
				s.Min.Y.Ch(10)
				// s.Gap.Zero()
			})
		case "grid-lay/scrollbar":
			sb := w.(*gi.Slider)
			sb.Style(func(s *styles.Style) {
				sb.Type = gi.SliderScrollbar
				// s.Min.Y.Zero()
				s.Min.X = sv.Styles.ScrollBarWidth
				s.Grow.Set(0, 1)
				fmt.Println(sb, "set grow:", s.Grow)
			})
			sb.OnChange(func(e events.Event) {
				sv.StartIdx = int(sb.Value)
				sv.This().(SliceViewer).UpdateWidgets()
			})

		}
		if w.Parent().PathFrom(sv) == "grid-lay/grid" {
			switch {
			case strings.HasPrefix(w.Name(), "index-"):
				w.Style(func(s *styles.Style) {
					s.Min.X.Em(1.5)
					s.Padding.Right.Dp(4)
					s.Text.Align = styles.AlignEnd
					s.Min.Y.Em(1)
					s.Grow.Set(0, 0)
				})
			case strings.HasPrefix(w.Name(), "add-"):
				w.Style(func(s *styles.Style) {
					w.(*gi.Button).SetType(gi.ButtonAction)
					s.Color = colors.Scheme.Success.Base
				})
			case strings.HasPrefix(w.Name(), "del-"):
				w.Style(func(s *styles.Style) {
					w.(*gi.Button).SetType(gi.ButtonAction)
					s.Color = colors.Scheme.Error.Base
				})
			case strings.HasPrefix(w.Name(), "value-"):
				w.Style(func(s *styles.Style) {
					idx := grr.Log(strconv.Atoi(strings.TrimPrefix(w.Name(), "value-")))
					si := sv.StartIdx + idx
					if si < sv.SliceSize {
						sv.This().(SliceViewer).StyleRow(w, si, 0)
					}
				})
			}
		}
	})
}

func (sv *SliceViewBase) AsSliceViewBase() *SliceViewBase {
	return sv
}

// SetSlice sets the source slice that we are viewing.
// This ReConfigs the view for this slice if different.
// Note: it is important to at least set an empty slice of
// the desired type at the start to enable initial configuration.
func (sv *SliceViewBase) SetSlice(sl any) *SliceViewBase {
	if laser.AnyIsNil(sl) {
		sv.Slice = nil
		return sv
	}
	newslc := false
	if reflect.TypeOf(sl).Kind() != reflect.Pointer { // prevent crash on non-comparable
		newslc = true
	} else {
		newslc = (sv.Slice != sl)
	}
	if !newslc && sv.Is(SliceViewConfiged) {
		sv.Update()
		return sv
	}
	updt := sv.UpdateStart()
	sv.StartIdx = 0
	sv.Slice = sl
	sv.SliceNPVal = laser.NonPtrValue(reflect.ValueOf(sv.Slice))
	isArray := laser.NonPtrType(reflect.TypeOf(sl)).Kind() == reflect.Array
	sv.SetFlag(isArray, SliceViewIsArray)
	// make sure elements aren't nil to prevent later panics
	for i := 0; i < sv.SliceNPVal.Len(); i++ {
		val := sv.SliceNPVal.Index(i)
		k := val.Kind()
		if (k == reflect.Chan || k == reflect.Func || k == reflect.Interface || k == reflect.Map || k == reflect.Pointer || k == reflect.Slice) && val.IsNil() {
			val.Set(reflect.New(laser.NonPtrType(val.Type())))
		}
	}
	sv.ElVal = laser.SliceElValue(sl)
	if !sv.IsReadOnly() {
		sv.SelIdx = -1
	}
	sv.ResetSelectedIdxs()
	sv.UpdateEnd(updt)
	sv.Update()
	return sv
}

// IsNil returns true if the Slice is nil
func (sv *SliceViewBase) IsNil() bool {
	return laser.AnyIsNil(sv.Slice)
}

// BindSelectDialog makes the slice view a read-only selection slice view and then
// binds its events to the given dialog and its current selection index to the given value.
func (sv *SliceViewBase) BindSelectDialog(d *gi.Dialog, val *int) *SliceViewBase {
	sv.SetReadOnly(true)
	sv.OnSelect(func(e events.Event) {
		*val = sv.CurIdx
	})
	sv.OnDoubleClick(func(e events.Event) {
		*val = sv.CurIdx
		d.AcceptDialog()
	})
	return sv
}

// Config configures a standard setup of the overall Frame
func (sv *SliceViewBase) ConfigWidget(sc *gi.Scene) {
	sv.ConfigSliceView(sc)
}

// ConfigSliceView handles entire config.
// ReConfig calls this, followed by ApplyStyleTree so we don't need to call that.
func (sv *SliceViewBase) ConfigSliceView(sc *gi.Scene) {
	if sv.Is(SliceViewConfiged) {
		if sv.NeedsConfigRows() {
			sv.This().(SliceViewer).ConfigRows(sc)
		} else {
			sv.This().(SliceViewer).UpdateWidgets()
		}
		return
	}
	updt := sv.UpdateStart()
	sv.ConfigFrame(sc)
	sv.This().(SliceViewer).ConfigOneRow(sc)
	sv.ConfigScroll()
	sv.ApplyStyleTree(sc)
	sv.UpdateEndLayout(updt)
}

func (sv *SliceViewBase) ConfigFrame(sc *gi.Scene) {
	sv.SetFlag(true, SliceViewConfiged)
	sv.VisRows = 0
	gl := gi.NewLayout(sv, "grid-lay")
	gl.SetFlag(true, gi.LayoutNoKeys)
	gi.NewFrame(gl, "grid")
	gi.NewSlider(gl, "scrollbar")
}

// ConfigOneRow configures one row for initial row height measurement
func (sv *SliceViewBase) ConfigOneRow(sc *gi.Scene) {
	sg := sv.This().(SliceViewer).SliceGrid()
	sg.SetFlag(true, gi.LayoutNoKeys)
	if sg.HasChildren() {
		return
	}
	updt := sg.UpdateStart()
	defer sg.UpdateEnd(updt)

	sv.VisRows = 0
	if sv.IsNil() {
		return
	}

	nWidgPerRow, idxOff := sv.RowWidgetNs()
	sg.Kids = make(ki.Slice, nWidgPerRow)

	// at this point, we make one dummy row to get size of widgets
	val := sv.ElVal
	vv := ToValue(sv.ElVal.Interface(), "")
	if vv == nil { // shouldn't happen
		return
	}
	vv.SetSliceValue(val, sv.Slice, 0, sv.TmpSave, sv.ViewPath)
	vtyp := vv.WidgetType()
	itxt := fmt.Sprintf("%05d", 0)
	labnm := fmt.Sprintf("index-%v", itxt)
	valnm := fmt.Sprintf("value-%v", itxt)

	if sv.Is(SliceViewShowIndex) {
		idxlab := &gi.Label{}
		sg.SetChild(idxlab, 0, labnm)
		idxlab.Text = itxt
	}

	w := ki.NewOfType(vtyp).(gi.Widget)
	sg.SetChild(w, idxOff, valnm)
	vv.ConfigWidget(w, sc)
	vv.UpdateWidget()

	if !sv.IsReadOnly() && !sv.Is(SliceViewIsArray) {
		cidx := idxOff
		if !sv.Is(SliceViewNoAdd) {
			cidx++
			addnm := fmt.Sprintf("add-%v", itxt)
			addbt := gi.Button{}
			sg.SetChild(&addbt, cidx, addnm)
			addbt.SetType(gi.ButtonAction)
			addbt.SetIcon(icons.Add)
		}
		if !sv.Is(SliceViewNoDelete) {
			cidx++
			delnm := fmt.Sprintf("del-%v", itxt)
			delbt := gi.Button{}
			sg.SetChild(&delbt, cidx, delnm)
			delbt.SetType(gi.ButtonAction)
			delbt.SetIcon(icons.Delete)
		}
	}
}

// ConfigScroll configures the scrollbar
func (sv *SliceViewBase) ConfigScroll() {
	sb := sv.This().(SliceViewer).ScrollBar()
	sb.Type = gi.SliderScrollbar
	sb.Dim = mat32.Y
	sb.Tracking = true
	sb.Min = 0
	sb.Step = 1
	sv.UpdateScroll()
}

// GridLayout returns the Layout containing the Grid and the scrollbar
func (sv *SliceViewBase) GridLayout() *gi.Layout {
	return sv.ChildByName("grid-lay", 0).(*gi.Layout)
}

// SliceGrid returns the SliceGrid grid frame widget, which contains all the
// fields and values
func (sv *SliceViewBase) SliceGrid() *gi.Frame {
	return sv.GridLayout().ChildByName("grid", 0).(*gi.Frame)
}

// ScrollBar returns the SliceGrid scrollbar
func (sv *SliceViewBase) ScrollBar() *gi.Slider {
	return sv.GridLayout().ChildByName("scrollbar", 1).(*gi.Slider)
}

// RowWidgetNs returns number of widgets per row and offset for index label
func (sv *SliceViewBase) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 2
	if !sv.IsReadOnly() && !sv.Is(SliceViewIsArray) {
		if !sv.Is(SliceViewNoAdd) {
			nWidgPerRow += 1
		}
		if !sv.Is(SliceViewNoDelete) {
			nWidgPerRow += 1
		}
	}
	idxOff = 1
	if !sv.Is(SliceViewShowIndex) {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

// UpdtSliceSize updates and returns the size of the slice
// and sets SliceSize
func (sv *SliceViewBase) UpdtSliceSize() int {
	sz := sv.SliceNPVal.Len()
	sv.SliceSize = sz
	return sz
}

// ViewMuLock locks the ViewMu if non-nil
func (sv *SliceViewBase) ViewMuLock() {
	if sv.ViewMu == nil {
		return
	}
	sv.ViewMu.Lock()
}

// ViewMuUnlock Unlocks the ViewMu if non-nil
func (sv *SliceViewBase) ViewMuUnlock() {
	if sv.ViewMu == nil {
		return
	}
	sv.ViewMu.Unlock()
}

func (sv *SliceViewBase) SizeDown(sc *gi.Scene, iter int) bool {
	redo := sv.Frame.SizeDown(sc, iter)
	if iter == 0 && sv.This().(SliceViewer).NeedsConfigRows() {
		// fmt.Println("redo, iter:", iter)
		sv.Update()   // does applystyle
		sv.SizeUp(sc) // re-init everything
		redo = true
	}
	return redo
}

// UpdateStartIdx updates StartIdx to fit current view
func (sv *SliceViewBase) UpdateStartIdx() {
	sz := sv.This().(SliceViewer).UpdtSliceSize()
	if sz > sv.VisRows {
		lastSt := sz - sv.VisRows
		sv.StartIdx = min(lastSt, sv.StartIdx)
		sv.StartIdx = max(0, sv.StartIdx)
	} else {
		sv.StartIdx = 0
	}
}

// UpdateScroll updates grid scrollbar based on display
func (sv *SliceViewBase) UpdateScroll() {
	sb := sv.This().(SliceViewer).ScrollBar()
	updt := sb.UpdateStart()
	sb.Max = float32(sv.SliceSize) + 0.01 // bit of extra to ensure last line always shows up
	if sv.VisRows > 0 {
		sb.PageStep = float32(sv.VisRows) * sb.Step
		sb.SetVisiblePct(float32(sv.VisRows) / float32(sv.SliceSize))
	} else {
		sb.PageStep = 10 * sb.Step
		sb.SetVisiblePct(1)
	}
	sb.TrackThr = sb.Step
	sb.SetValue(float32(sv.StartIdx)) // essential for updating pos from value
	sb.SetState(sv.VisRows == sv.SliceSize, states.Invisible)
	sb.UpdateEnd(updt)
}

func (sv *SliceViewBase) AvailHeight() float32 {
	sg := sv.This().(SliceViewer).SliceGrid()
	// sgHt := sg.Alloc.Size.Content.Y
	sgHt := sg.LayImpl.ContentSubGap.Y
	// fmt.Println("sg:", sgHt, "sv:", sv.Alloc.Size.TotalOrig.Y)
	return sgHt
}

// VisRowsAvail returns the number of visible rows available
// to display given the current layout parameters.
func (sv *SliceViewBase) VisRowsAvail() (rows int, rowht, layht float32) {
	sg := sv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}
	if sg.HasChildren() {
		_, kwb := gi.AsWidget(sg.Child(0))
		rowht = kwb.Alloc.Size.Alloc.Y
		// fmt.Println(kwb, "row ht alloc:", rowht, "total:", kwb.Alloc.Size.Total.Y)
	}
	if !sv.NeedsRebuild() { // use existing unless rebuilding
		rowht = max(rowht, sv.RowHeight)
		// fmt.Println("rowht:", rowht, "svrh:", sv.RowHeight)
	}
	if sv.Styles.Font.Face == nil {
		sv.Styles.Font = paint.OpenFont(sv.Styles.FontRender(), &sv.Styles.UnContext)
	}
	rowht = mat32.Max(rowht, 2*sv.Styles.Font.Face.Metrics.Height)

	sc := sv.Sc
	if sc != nil && sc.Is(gi.ScPrefSizing) {
		rows = gi.LayoutPrefMaxRows
		layht = float32(rows) * rowht
	} else {
		sv.RowHeight = max(sv.RowHeight, rowht)
		sgHt := sv.AvailHeight()
		layht = sgHt
		if sgHt == 0 {
			return
		}
		rows = int(mat32.Floor(sgHt / rowht))
		// fmt.Println("rows:", rows, "sght:", sgHt, "rowht:", rowht, "sc ht:", sc.Geom.Size.Y)
	}
	return
}

// NeedsConfigRows returns true if layout size needs diff # of rows
func (sv *SliceViewBase) NeedsConfigRows() bool {
	if sv.IsNil() {
		sv.VisRows = 0
		return false
	}
	rows, _, _ := sv.VisRowsAvail()
	return rows != sv.VisRows || rows == 0
}

// ConfigRows configures VisRows worth of widgets
// to display slice data.  It should only be called
// when NeedsConfigRows is true: when VisRows changes.
func (sv *SliceViewBase) ConfigRows(sc *gi.Scene) {
	sg := sv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}
	sg.SetFlag(true, gi.LayoutNoKeys)

	updt := sg.UpdateStart()
	defer sg.UpdateEndLayout(updt)

	sv.ViewMuLock()
	defer sv.ViewMuUnlock()

	sg.DeleteChildren(ki.DestroyKids)
	sv.Values = nil
	sv.VisRows = 0

	if sv.IsNil() {
		return
	}

	sv.VisRows, sv.RowHeight, sv.LayoutHeight = sv.VisRowsAvail()

	// fmt.Println("vis:", sv.VisRows, "rowht:", sv.RowHeight, "layht:", sv.LayoutHeight)
	// fmt.Println("sg:", sg.LayState.String())

	nWidgPerRow, idxOff := sv.RowWidgetNs()
	nWidg := nWidgPerRow * sv.VisRows

	sv.Values = make([]Value, sv.VisRows)
	sg.Kids = make(ki.Slice, nWidg)

	for i := 0; i < sv.VisRows; i++ {
		i := i
		si := i
		ridx := i * nWidgPerRow
		var val reflect.Value
		if si < sv.SliceSize {
			val = laser.OnePtrUnderlyingValue(sv.SliceNPVal.Index(si)) // deal with pointer lists
		} else {
			val = sv.ElVal
		}
		vv := ToValue(val.Interface(), "")
		sv.Values[i] = vv
		vv.SetSliceValue(val, sv.Slice, si, sv.TmpSave, sv.ViewPath)
		vv.SetReadOnly(sv.IsReadOnly())

		vtyp := vv.WidgetType()
		itxt := strconv.Itoa(i)
		sitxt := strconv.Itoa(si)
		labnm := "index-" + itxt
		valnm := "value-" + itxt

		if sv.Is(SliceViewShowIndex) {
			idxlab := &gi.Label{}
			sg.SetChild(idxlab, ridx, labnm)
			idxlab.OnSelect(func(e events.Event) {
				e.SetHandled()
				sv.UpdateSelectRow(i)
			})
			idxlab.SetText(sitxt)
		}

		w := ki.NewOfType(vtyp).(gi.Widget)
		sg.SetChild(w, ridx+idxOff, valnm)
		vv.ConfigWidget(w, sc)
		wb := w.AsWidget()
		wb.OnSelect(func(e events.Event) {
			e.SetHandled()
			sv.UpdateSelectRow(i)
		})

		if sv.IsReadOnly() {
			w.AsWidget().SetReadOnly(true)
		} else {
			vvb := vv.AsValueBase()
			vvb.OnChange(func(e events.Event) {
				sv.SendChange()
			})
			if !sv.Is(SliceViewIsArray) {
				cidx := ridx + idxOff
				if !sv.Is(SliceViewNoAdd) {
					cidx++
					addnm := fmt.Sprintf("add-%v", itxt)
					addact := gi.Button{}
					sg.SetChild(&addact, cidx, addnm)
					addact.SetType(gi.ButtonAction)
					addact.SetIcon(icons.Add)
					addact.Tooltip = "insert a new element at this index"
					addact.Data = i
					addact.OnClick(func(e events.Event) {
						sv.SliceNewAtRow(i + 1)
					})
				}

				if !sv.Is(SliceViewNoDelete) {
					cidx++
					delnm := fmt.Sprintf("del-%v", itxt)
					delact := gi.Button{}
					sg.SetChild(&delact, cidx, delnm)
					delact.SetType(gi.ButtonAction)
					delact.SetIcon(icons.Delete)
					delact.Tooltip = "delete this element"
					delact.Data = i
					delact.OnClick(func(e events.Event) {
						sv.SliceDeleteAtRow(i)
					})
				}
			}
		}
	}
	sv.This().(SliceViewer).UpdateWidgets()
}

// UpdateWidgets updates the row widget display to
// represent the current state of the slice data,
// including which range of data is being displayed.
// This is called for scrolling, navigation etc.
func (sv *SliceViewBase) UpdateWidgets() {
	sg := sv.This().(SliceViewer).SliceGrid()
	if sg == nil || sv.VisRows == 0 || !sg.HasChildren() {
		return
	}
	updt := sg.UpdateStart()
	defer sg.UpdateEndLayout(updt)
	// fmt.Println("updtw:", updt)

	sv.ViewMuLock()
	defer sv.ViewMuUnlock()

	nWidgPerRow, idxOff := sv.RowWidgetNs()
	// sc := sv.Sc

	sv.UpdateStartIdx()
	for i := 0; i < sv.VisRows; i++ {
		i := i
		ridx := i * nWidgPerRow
		w := sg.Kids[ridx+idxOff].(gi.Widget)
		vv := sv.Values[i]
		si := sv.StartIdx + i // slice idx
		var idxlab *gi.Label
		if sv.Is(SliceViewShowIndex) {
			idxlab = sg.Kids[ridx].(*gi.Label)
			idxlab.SetText(strconv.Itoa(si))
			idxlab.SetNeedsRender()
		}
		if si < sv.SliceSize {
			w.SetState(false, states.Invisible)
			val := laser.OnePtrUnderlyingValue(sv.SliceNPVal.Index(si)) // deal with pointer lists
			vv.SetSliceValue(val, sv.Slice, si, sv.TmpSave, sv.ViewPath)
			vv.SetReadOnly(sv.IsReadOnly())
			vv.UpdateWidget()

			if sv.IsReadOnly() {
				w.AsWidget().SetReadOnly(true)
			}
			issel := sv.IdxIsSelected(si)
			w.AsWidget().SetSelected(issel)
			if sv.Is(SliceViewShowIndex) {
				idxlab.SetState(false, states.Invisible)
				idxlab.SetSelected(issel)
			}
			if !sv.IsReadOnly() && !sv.Is(SliceViewIsArray) {
				cidx := ridx + idxOff
				if !sv.Is(SliceViewNoAdd) {
					cidx++
					addact := sg.Kids[cidx].(*gi.Button)
					addact.SetState(false, states.Invisible)
				}
				if !sv.Is(SliceViewNoDelete) {
					cidx++
					delact := sg.Kids[cidx].(*gi.Button)
					delact.SetState(false, states.Invisible)
				}
			}
		} else {
			w.SetState(true, states.Invisible)
			vv.SetSliceValue(sv.ElVal, sv.Slice, 0, sv.TmpSave, sv.ViewPath)
			vv.UpdateWidget()
			w.AsWidget().SetSelected(false)
			if sv.Is(SliceViewShowIndex) {
				idxlab.SetState(true, states.Invisible)
				idxlab.SetSelected(false)
			}
			if !sv.IsReadOnly() && !sv.Is(SliceViewIsArray) {
				cidx := ridx + idxOff
				if !sv.Is(SliceViewNoAdd) {
					cidx++
					addact := sg.Kids[cidx].(*gi.Button)
					addact.SetState(true, states.Invisible)
				}
				if !sv.Is(SliceViewNoDelete) {
					cidx++
					delact := sg.Kids[cidx].(*gi.Button)
					delact.SetState(true, states.Invisible)
				}
			}
		}
	}
	if sv.SelVal != nil {
		sv.SelIdx, _ = SliceIdxByValue(sv.Slice, sv.SelVal)
	}
	if sv.IsReadOnly() && sv.SelIdx >= 0 {
		sv.SelectIdx(sv.SelIdx)
	}
	sv.UpdateScroll()
}

// SetChanged sets the Changed flag and emits the ViewSig signal for the
// SliceViewBase, indicating that some kind of edit / change has taken place to
// the table data.  It isn't really practical to record all the different
// types of changes, so this is just generic.
func (sv *SliceViewBase) SetChanged() {
	sv.Changed = true
	sv.SendChange()
}

// SliceNewAtRow inserts a new blank element at given display row
func (sv *SliceViewBase) SliceNewAtRow(row int) {
	sv.This().(SliceViewer).SliceNewAt(sv.StartIdx + row)
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (sv *SliceViewBase) SliceNewAt(idx int) {
	if sv.Is(SliceViewIsArray) {
		return
	}

	sv.ViewMuLock() // no return!  must unlock before return below

	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	sv.SliceNewAtSel(idx)

	sltyp := laser.SliceElType(sv.Slice) // has pointer if it is there
	iski := ki.IsKi(sltyp)
	slptr := sltyp.Kind() == reflect.Ptr

	svl := reflect.ValueOf(sv.Slice)
	sz := sv.SliceSize

	svnp := sv.SliceNPVal

	if iski && sv.SliceValView != nil {
		vvb := sv.SliceValView.AsValueBase()
		if vvb.Owner != nil {
			if ownki, ok := vvb.Owner.(ki.Ki); ok {
				d := gi.NewDialog(sv).Title("Slice New").Prompt("Number and Type of Items to Insert:")
				nd := &gi.NewItemsData{}
				w := NewValue(d, nd).AsWidget()
				ki.ChildByType[*gi.Chooser](w, true).SetTypes(gti.AllEmbeddersOf(ownki.BaseType()), true, true, 50)
				d.Cancel().Ok().OnAccept(func(e events.Event) {
					updt := ownki.UpdateStart()
					for i := 0; i < nd.Number; i++ {
						nm := fmt.Sprintf("New%v%v", nd.Type.Name, idx+1+i)
						ownki.InsertNewChild(nd.Type, idx+1+i, nm)
					}
					sv.SetChanged()
					ownki.UpdateEnd(updt)
				}).Run()
			}
		}
	} else {
		nval := reflect.New(laser.NonPtrType(sltyp)) // make the concrete el
		if !slptr {
			nval = nval.Elem() // use concrete value
		}
		svnp = reflect.Append(svnp, nval)
		if idx >= 0 && idx < sz {
			reflect.Copy(svnp.Slice(idx+1, sz+1), svnp.Slice(idx, sz))
			svnp.Index(idx).Set(nval)
		}
		svl.Elem().Set(svnp)
	}
	if idx < 0 {
		idx = sz
	}

	sv.SliceNPVal = laser.NonPtrValue(reflect.ValueOf(sv.Slice)) // need to update after changes

	sv.This().(SliceViewer).UpdtSliceSize()

	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.ViewMuUnlock()
	sv.SetChanged()
	sv.Update()
}

// SliceDeleteAtRow deletes element at given display row
// if updt is true, then update the grid after
func (sv *SliceViewBase) SliceDeleteAtRow(row int) {
	sv.This().(SliceViewer).SliceDeleteAt(sv.StartIdx + row)
}

// SliceNewAtSel updates selected rows based on
// inserting new element at given index.
// must be called with successful SliceNewAt
func (sv *SliceViewBase) SliceNewAtSel(idx int) {
	sl := sv.SelectedIdxsList(false) // ascending
	sv.ResetSelectedIdxs()
	for _, ix := range sl {
		if ix >= idx {
			ix++
		}
		sv.SelIdxs[ix] = struct{}{}
	}
}

// SliceDeleteAtSel updates selected rows based on
// deleting element at given index
// must be called with successful SliceDeleteAt
func (sv *SliceViewBase) SliceDeleteAtSel(idx int) {
	sl := sv.SelectedIdxsList(true) // desscending
	sv.ResetSelectedIdxs()
	for _, ix := range sl {
		switch {
		case ix == idx:
			continue
		case ix > idx:
			ix--
		}
		sv.SelIdxs[ix] = struct{}{}
	}
}

// SliceDeleteAt deletes element at given index from slice
func (sv *SliceViewBase) SliceDeleteAt(idx int) {
	if sv.Is(SliceViewIsArray) {
		return
	}
	if idx < 0 || idx >= sv.SliceSize {
		return
	}
	sv.ViewMuLock()
	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	sv.SliceDeleteAtSel(idx)

	laser.SliceDeleteAt(sv.Slice, idx)

	sv.This().(SliceViewer).UpdtSliceSize()

	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}

	sv.ViewMuUnlock()
	sv.SetChanged()
	sv.Update()
}

// SliceDefaultTopAppBar is a TopAppBar function that adds an "Add" button for slice
func (sv *SliceViewBase) SliceDefaultTopAppBar(tb *gi.TopAppBar) {
	if laser.AnyIsNil(sv.Slice) {
		return
	}
	if sv.Is(SliceViewIsArray) || sv.IsReadOnly() || sv.Is(SliceViewNoAdd) {
		return
	}
	gi.NewButton(tb, "slice-add").SetText("Add").SetIcon(icons.Add).SetTooltip("add a new element to the slice").
		OnClick(func(e events.Event) {
			sv.This().(SliceViewer).SliceNewAt(-1)
		})
}

////////////////////////////////////////////////////////////
//  Row access methods
//  NOTE: row = physical GUI display row, idx = slice index
//  not the same!

// SliceVal returns value interface at given slice index
// must be protected by mutex
func (sv *SliceViewBase) SliceVal(idx int) any {
	if idx < 0 || idx >= sv.SliceSize {
		fmt.Printf("giv.SliceViewBase: slice index out of range: %v\n", idx)
		return nil
	}
	val := laser.OnePtrUnderlyingValue(sv.SliceNPVal.Index(idx)) // deal with pointer lists
	vali := val.Interface()
	return vali
}

// IsRowInBounds returns true if disp row is in bounds
func (sv *SliceViewBase) IsRowInBounds(row int) bool {
	return row >= 0 && row < sv.VisRows
}

// IsIdxVisible returns true if slice index is currently visible
func (sv *SliceViewBase) IsIdxVisible(idx int) bool {
	return sv.IsRowInBounds(idx - sv.StartIdx)
}

// RowFirstWidget returns the first widget for given row (could be index or
// not) -- false if out of range
func (sv *SliceViewBase) RowFirstWidget(row int) (*gi.WidgetBase, bool) {
	if !sv.Is(SliceViewShowIndex) {
		return nil, false
	}
	if !sv.IsRowInBounds(row) {
		return nil, false
	}
	nWidgPerRow, _ := sv.This().(SliceViewer).RowWidgetNs()
	sg := sv.This().(SliceViewer).SliceGrid()
	w := sg.Kids[row*nWidgPerRow].(gi.Widget).AsWidget()
	return w, true
}

// RowGrabFocus grabs the focus for the first focusable widget
// in given row.  returns that element or nil if not successful
// note: grid must have already rendered for focus to be grabbed!
func (sv *SliceViewBase) RowGrabFocus(row int) *gi.WidgetBase {
	if !sv.IsRowInBounds(row) || sv.Is(SliceViewInFocusGrab) { // range check
		return nil
	}
	nWidgPerRow, idxOff := sv.This().(SliceViewer).RowWidgetNs()
	ridx := nWidgPerRow * row
	sg := sv.This().(SliceViewer).SliceGrid()
	w := sg.Child(ridx + idxOff).(gi.Widget).AsWidget()
	if w.StateIs(states.Focused) {
		return w
	}
	sv.SetFlag(true, SliceViewInFocusGrab)
	w.GrabFocus()
	sv.SetFlag(false, SliceViewInFocusGrab)
	return w
}

// IdxGrabFocus grabs the focus for the first focusable widget
// in given idx.  returns that element or nil if not successful.
func (sv *SliceViewBase) IdxGrabFocus(idx int) *gi.WidgetBase {
	sv.ScrollToIdx(idx)
	return sv.This().(SliceViewer).RowGrabFocus(idx - sv.StartIdx)
}

// IdxPos returns center of window position of index label for idx (ContextMenuPos)
func (sv *SliceViewBase) IdxPos(idx int) image.Point {
	row := idx - sv.StartIdx
	if row < 0 {
		row = 0
	}
	if row > sv.VisRows-1 {
		row = sv.VisRows - 1
	}
	var pos image.Point
	w, ok := sv.This().(SliceViewer).RowFirstWidget(row)
	if ok {
		pos = w.ContextMenuPos(nil)
	}
	return pos
}

// RowFromPos returns the row that contains given vertical position, false if not found
func (sv *SliceViewBase) RowFromPos(posY int) (int, bool) {
	// todo: could optimize search to approx loc, and search up / down from there
	for rw := 0; rw < sv.VisRows; rw++ {
		w, ok := sv.This().(SliceViewer).RowFirstWidget(rw)
		if ok {
			if w.Alloc.BBox.Min.Y < posY && posY < w.Alloc.BBox.Max.Y {
				return rw, true
			}
		}
	}
	return -1, false
}

// IdxFromPos returns the idx that contains given vertical position, false if not found
func (sv *SliceViewBase) IdxFromPos(posY int) (int, bool) {
	row, ok := sv.RowFromPos(posY)
	if !ok {
		return -1, false
	}
	return row + sv.StartIdx, true
}

// ScrollToIdxNoUpdt ensures that given slice idx is visible
// by scrolling display as needed.
// This version does not update the slicegrid.
// Just computes the StartIdx and updates the scrollbar
func (sv *SliceViewBase) ScrollToIdxNoUpdt(idx int) bool {
	if sv.VisRows == 0 {
		return false
	}
	if idx < sv.StartIdx {
		sv.StartIdx = idx
		sv.StartIdx = max(0, sv.StartIdx)
		sv.UpdateScroll()
		return true
	} else if idx >= sv.StartIdx+sv.VisRows {
		sv.StartIdx = idx - (sv.VisRows - 1)
		sv.StartIdx = max(0, sv.StartIdx)
		sv.UpdateScroll()
		return true
	}
	return false
}

// ScrollToIdx ensures that given slice idx is visible
// by scrolling display as needed.
func (sv *SliceViewBase) ScrollToIdx(idx int) bool {
	updt := sv.ScrollToIdxNoUpdt(idx)
	if updt {
		sv.This().(SliceViewer).UpdateWidgets()
	}
	return updt
}

// SelectVal sets SelVal and attempts to find corresponding row, setting
// SelectedIdx and selecting row if found -- returns true if found, false
// otherwise.
func (sv *SliceViewBase) SelectVal(val string) bool {
	sv.SelVal = val
	if sv.SelVal != nil {
		sv.ViewMuLock()
		idx, _ := SliceIdxByValue(sv.Slice, sv.SelVal)
		sv.ViewMuUnlock()
		if idx >= 0 {
			sv.UpdateSelectIdx(idx, true)
			sv.ScrollToIdx(idx)
			return true
		}
	}
	return false
}

// SliceIdxByValue searches for first index that contains given value in slice
// -- returns false if not found
func SliceIdxByValue(slc any, fldVal any) (int, bool) {
	svnp := laser.NonPtrValue(reflect.ValueOf(slc))
	sz := svnp.Len()
	for idx := 0; idx < sz; idx++ {
		rval := laser.NonPtrValue(svnp.Index(idx))
		if rval.Interface() == fldVal {
			return idx, true
		}
	}
	return -1, false
}

/////////////////////////////////////////////////////////////////////////////
//    Moving

// MoveDown moves the selection down to next row, using given select mode
// (from keyboard modifiers) -- returns newly selected row or -1 if failed
func (sv *SliceViewBase) MoveDown(selMode events.SelectModes) int {
	if sv.SelIdx >= sv.SliceSize-1 {
		sv.SelIdx = sv.SliceSize - 1
		return -1
	}
	sv.SelIdx++
	sv.SelectIdxAction(sv.SelIdx, selMode)
	return sv.SelIdx
}

// MoveDownAction moves the selection down to next row, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected
// row
func (sv *SliceViewBase) MoveDownAction(selMode events.SelectModes) int {
	nidx := sv.MoveDown(selMode)
	if nidx >= 0 {
		sv.ScrollToIdx(nidx)
		sv.Send(events.Select) // todo: need to do this for the item?
	}
	return nidx
}

// MoveUp moves the selection up to previous idx, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (sv *SliceViewBase) MoveUp(selMode events.SelectModes) int {
	if sv.SelIdx <= 0 {
		sv.SelIdx = 0
		return -1
	}
	sv.SelIdx--
	sv.SelectIdxAction(sv.SelIdx, selMode)
	return sv.SelIdx
}

// MoveUpAction moves the selection up to previous idx, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (sv *SliceViewBase) MoveUpAction(selMode events.SelectModes) int {
	nidx := sv.MoveUp(selMode)
	if nidx >= 0 {
		sv.ScrollToIdx(nidx)
		sv.Send(events.Select)
	}
	return nidx
}

// MovePageDown moves the selection down to next page, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (sv *SliceViewBase) MovePageDown(selMode events.SelectModes) int {
	if sv.SelIdx >= sv.SliceSize-1 {
		sv.SelIdx = sv.SliceSize - 1
		return -1
	}
	sv.SelIdx += sv.VisRows
	sv.SelIdx = min(sv.SelIdx, sv.SliceSize-1)
	sv.SelectIdxAction(sv.SelIdx, selMode)
	return sv.SelIdx
}

// MovePageDownAction moves the selection down to next page, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (sv *SliceViewBase) MovePageDownAction(selMode events.SelectModes) int {
	nidx := sv.MovePageDown(selMode)
	if nidx >= 0 {
		sv.ScrollToIdx(nidx)
		sv.Send(events.Select)
	}
	return nidx
}

// MovePageUp moves the selection up to previous page, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (sv *SliceViewBase) MovePageUp(selMode events.SelectModes) int {
	if sv.SelIdx <= 0 {
		sv.SelIdx = 0
		return -1
	}
	sv.SelIdx -= sv.VisRows
	sv.SelIdx = max(0, sv.SelIdx)
	sv.SelectIdxAction(sv.SelIdx, selMode)
	return sv.SelIdx
}

// MovePageUpAction moves the selection up to previous page, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (sv *SliceViewBase) MovePageUpAction(selMode events.SelectModes) int {
	nidx := sv.MovePageUp(selMode)
	if nidx >= 0 {
		sv.ScrollToIdx(nidx)
		sv.Send(events.Select)
	}
	return nidx
}

//////////////////////////////////////////////////////////
//    Selection: user operates on the index labels

// SelectRowWidgets sets the selection state of given row of widgets
func (sv *SliceViewBase) SelectRowWidgets(row int, sel bool) {
	if row < 0 {
		return
	}
	updt := sv.UpdateStart()
	defer sv.UpdateEndRender(updt)

	sg := sv.This().(SliceViewer).SliceGrid()
	nWidgPerRow, idxOff := sv.This().(SliceViewer).RowWidgetNs()
	rowidx := row * nWidgPerRow
	if sv.Is(SliceViewShowIndex) {
		if sg.Kids.IsValidIndex(rowidx) == nil {
			w := sg.Child(rowidx).(gi.Widget).AsWidget()
			w.SetSelected(sel)
		}
	}
	if sg.Kids.IsValidIndex(rowidx+idxOff) == nil {
		w := sg.Child(rowidx + idxOff).(gi.Widget).AsWidget()
		w.SetSelected(sel)
	}
}

// SelectIdxWidgets sets the selection state of given slice index
// returns false if index is not visible
func (sv *SliceViewBase) SelectIdxWidgets(idx int, sel bool) bool {
	if !sv.IsIdxVisible(idx) {
		return false
	}
	sv.This().(SliceViewer).SelectRowWidgets(idx-sv.StartIdx, sel)
	return true
}

// UpdateSelectRow updates the selection for the given row
// callback from widgetsig select
func (sv *SliceViewBase) UpdateSelectRow(row int) {
	idx := row + sv.StartIdx
	sel := !sv.IdxIsSelected(idx)
	sv.UpdateSelectIdx(idx, sel)
}

// UpdateSelectIdx updates the selection for the given index
func (sv *SliceViewBase) UpdateSelectIdx(idx int, sel bool) {
	if sv.IsReadOnly() && !sv.Is(SliceViewReadOnlyMultiSel) {
		updt := sv.UpdateStart()
		defer sv.UpdateEndRender(updt)

		sv.UnselectAllIdxs()
		if sel || sv.SelIdx == idx {
			sv.SelIdx = idx
			sv.SelectIdx(idx)
		}
		// sv.This().(SliceViewer).UpdateWidgets()
		sv.ApplyStyleTree(sv.Sc)
		sv.Send(events.Select)
	} else {
		selMode := events.SelectOne
		em := sv.EventMgr()
		if em != nil {
			selMode = em.LastSelMode
		}
		sv.SelectIdxAction(idx, selMode)
	}
}

// IdxIsSelected returns the selected status of given slice index
func (sv *SliceViewBase) IdxIsSelected(idx int) bool {
	if _, ok := sv.SelIdxs[idx]; ok {
		return true
	}
	return false
}

func (sv *SliceViewBase) ResetSelectedIdxs() {
	sv.SelIdxs = make(map[int]struct{})
}

// SelectedIdxsList returns list of selected indexes,
// sorted either ascending or descending
func (sv *SliceViewBase) SelectedIdxsList(descendingSort bool) []int {
	rws := make([]int, len(sv.SelIdxs))
	i := 0
	for r := range sv.SelIdxs {
		if r >= sv.SliceSize { // double safety check at this point
			delete(sv.SelIdxs, r)
			rws = rws[:len(rws)-1]
			continue
		}
		rws[i] = r
		i++
	}
	if descendingSort {
		sort.Slice(rws, func(i, j int) bool {
			return rws[i] > rws[j]
		})
	} else {
		sort.Slice(rws, func(i, j int) bool {
			return rws[i] < rws[j]
		})
	}
	return rws
}

// SelectIdx selects given idx (if not already selected) -- updates select
// status of index label
func (sv *SliceViewBase) SelectIdx(idx int) {
	sv.SelIdxs[idx] = struct{}{}
	sv.SelectIdxWidgets(idx, true)
}

// UnselectIdx unselects given idx (if selected)
func (sv *SliceViewBase) UnselectIdx(idx int) {
	if sv.IdxIsSelected(idx) {
		delete(sv.SelIdxs, idx)
	}
	sv.SelectIdxWidgets(idx, false)
}

// UnselectAllIdxs unselects all selected idxs
func (sv *SliceViewBase) UnselectAllIdxs() {
	for r := range sv.SelIdxs {
		sv.SelectIdxWidgets(r, false)
	}
	sv.ResetSelectedIdxs()
}

// SelectAllIdxs selects all idxs
func (sv *SliceViewBase) SelectAllIdxs() {
	updt := sv.UpdateStart()
	defer sv.UpdateEndRender(updt)

	sv.UnselectAllIdxs()
	sv.SelIdxs = make(map[int]struct{}, sv.SliceSize)
	for idx := 0; idx < sv.SliceSize; idx++ {
		sv.SelIdxs[idx] = struct{}{}
		sv.SelectIdxWidgets(idx, true)
	}
}

// SelectIdxAction is called when a select action has been received (e.g., a
// mouse click) -- translates into selection updates -- gets selection mode
// from mouse event (ExtendContinuous, ExtendOne)
func (sv *SliceViewBase) SelectIdxAction(idx int, mode events.SelectModes) {
	if mode == events.NoSelect {
		return
	}
	idx = min(idx, sv.SliceSize-1)
	if idx < 0 {
		sv.ResetSelectedIdxs()
		return
	}
	// row := idx - sv.StartIdx // note: could be out of bounds
	updt := sv.UpdateStart()
	defer sv.UpdateEndRender(updt)

	switch mode {
	case events.SelectOne:
		if sv.IdxIsSelected(idx) {
			if len(sv.SelIdxs) > 1 {
				sv.UnselectAllIdxs()
			}
			sv.SelIdx = idx
			sv.SelectIdx(idx)
			sv.IdxGrabFocus(idx)
		} else {
			sv.UnselectAllIdxs()
			sv.SelIdx = idx
			sv.SelectIdx(idx)
			sv.IdxGrabFocus(idx)
		}
		sv.Send(events.Select) //  sv.SelectedIdx)
	case events.ExtendContinuous:
		if len(sv.SelIdxs) == 0 {
			sv.SelIdx = idx
			sv.SelectIdx(idx)
			sv.IdxGrabFocus(idx)
			sv.Send(events.Select) //  sv.SelectedIdx)
		} else {
			minIdx := -1
			maxIdx := 0
			for r := range sv.SelIdxs {
				if minIdx < 0 {
					minIdx = r
				} else {
					minIdx = min(minIdx, r)
				}
				maxIdx = max(maxIdx, r)
			}
			cidx := idx
			sv.SelIdx = idx
			sv.SelectIdx(idx)
			if idx < minIdx {
				for cidx < minIdx {
					r := sv.MoveDown(events.SelectQuiet) // just select
					cidx = r
				}
			} else if idx > maxIdx {
				for cidx > maxIdx {
					r := sv.MoveUp(events.SelectQuiet) // just select
					cidx = r
				}
			}
			sv.IdxGrabFocus(idx)
			sv.Send(events.Select) //  sv.SelectedIdx)
		}
	case events.ExtendOne:
		if sv.IdxIsSelected(idx) {
			sv.UnselectIdxAction(idx)
			sv.Send(events.Select) //  sv.SelectedIdx)
		} else {
			sv.SelIdx = idx
			sv.SelectIdx(idx)
			sv.IdxGrabFocus(idx)
			sv.Send(events.Select) //  sv.SelectedIdx)
		}
	case events.Unselect:
		sv.SelIdx = idx
		sv.UnselectIdxAction(idx)
	case events.SelectQuiet:
		sv.SelIdx = idx
		sv.SelectIdx(idx)
	case events.UnselectQuiet:
		sv.SelIdx = idx
		sv.UnselectIdx(idx)
	}
	sv.This().(SliceViewer).UpdateWidgets()
	sv.ApplyStyleTree(sv.Sc)
}

// UnselectIdxAction unselects this idx (if selected) -- and emits a signal
func (sv *SliceViewBase) UnselectIdxAction(idx int) {
	if sv.IdxIsSelected(idx) {
		sv.UnselectIdx(idx)
	}
}

///////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeDataIdx adds mimedata for given idx: an application/json of the struct
func (sv *SliceViewBase) MimeDataIdx(md *mimedata.Mimes, idx int) {
	sv.ViewMuLock()
	val := sv.SliceVal(idx)
	b, err := json.MarshalIndent(val, "", "  ")
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: filecat.DataJson, Data: b})
	} else {
		log.Printf("gi.SliceViewBase MimeData JSON Marshall error: %v\n", err)
	}
	sv.ViewMuUnlock()
}

// FromMimeData creates a slice of structs from mime data
func (sv *SliceViewBase) FromMimeData(md mimedata.Mimes) []any {
	svtyp := sv.SliceNPVal.Type()
	sl := make([]any, 0, len(md))
	for _, d := range md {
		if d.Type == filecat.DataJson {
			nval := reflect.New(svtyp.Elem()).Interface()
			err := json.Unmarshal(d.Data, nval)
			if err == nil {
				sl = append(sl, nval)
			} else {
				log.Printf("gi.SliceViewBase FromMimeData: JSON load error: %v\n", err)
			}
		}
	}
	return sl
}

// MimeDataType returns the data type for mime clipboard (copy / paste) data
// e.g., filecat.DataJson
func (sv *SliceViewBase) MimeDataType() string {
	return filecat.DataJson
}

// CopySelToMime copies selected rows to mime data
func (sv *SliceViewBase) CopySelToMime() mimedata.Mimes {
	nitms := len(sv.SelIdxs)
	if nitms == 0 {
		return nil
	}
	ixs := sv.SelectedIdxsList(false) // ascending
	md := make(mimedata.Mimes, 0, nitms)
	for _, i := range ixs {
		sv.MimeDataIdx(&md, i)
	}
	return md
}

// Copy copies selected rows to clip.Board, optionally resetting the selection
// satisfies gi.Clipper interface and can be overridden by subtypes
func (sv *SliceViewBase) Copy(reset bool) {
	nitms := len(sv.SelIdxs)
	if nitms == 0 {
		return
	}
	md := sv.This().(SliceViewer).CopySelToMime()
	if md != nil {
		sv.EventMgr().ClipBoard().Write(md)
	}
	if reset {
		sv.UnselectAllIdxs()
	}
}

// CopyIdxs copies selected idxs to clip.Board, optionally resetting the selection
func (sv *SliceViewBase) CopyIdxs(reset bool) {
	if cpr, ok := sv.This().(gi.Clipper); ok { // should always be true, but justin case..
		cpr.Copy(reset)
	} else {
		sv.Copy(reset)
	}
}

// DeleteIdxs deletes all selected indexes
func (sv *SliceViewBase) DeleteIdxs() {
	if len(sv.SelIdxs) == 0 {
		return
	}
	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	ixs := sv.SelectedIdxsList(true) // descending sort
	for _, i := range ixs {
		sv.This().(SliceViewer).SliceDeleteAt(i)
	}
	sv.SetChanged()
	sv.Update()
}

// Cut copies selected indexes to clip.Board and deletes selected indexes
// satisfies gi.Clipper interface and can be overridden by subtypes
func (sv *SliceViewBase) Cut() {
	if len(sv.SelIdxs) == 0 {
		return
	}
	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	sv.CopyIdxs(false)
	ixs := sv.SelectedIdxsList(true) // descending sort
	idx := ixs[0]
	sv.UnselectAllIdxs()
	for _, i := range ixs {
		sv.This().(SliceViewer).SliceDeleteAt(i)
	}
	sv.SetChanged()
	sv.SelectIdxAction(idx, events.SelectOne)
	sv.Update()
}

// CutIdxs copies selected indexes to clip.Board and deletes selected indexes
func (sv *SliceViewBase) CutIdxs() {
	if cpr, ok := sv.This().(gi.Clipper); ok { // should always be true, but justin case..
		cpr.Cut()
	} else {
		sv.Cut()
	}
}

// Paste pastes clipboard at CurIdx
// satisfies gi.Clipper interface and can be overridden by subtypes
func (sv *SliceViewBase) Paste() {
	dt := sv.This().(SliceViewer).MimeDataType()
	md := sv.EventMgr().ClipBoard().Read([]string{dt})
	if md != nil {
		sv.PasteMenu(md, sv.CurIdx)
	}
}

// PasteIdx pastes clipboard at given idx
func (sv *SliceViewBase) PasteIdx(idx int) {
	sv.CurIdx = idx
	if cpr, ok := sv.This().(gi.Clipper); ok { // should always be true, but justin case..
		cpr.Paste()
	} else {
		sv.Paste()
	}
}

// MakePasteMenu makes the menu of options for paste events
func (sv *SliceViewBase) MakePasteMenu(m *gi.Scene, data any, idx int) {
	gi.NewButton(m).SetText("Assign To").SetData(data).
		OnClick(func(e events.Event) {
			sv.This().(SliceViewer).PasteAssign(data.(mimedata.Mimes), idx)
		})
	gi.NewButton(m).SetText("Insert Before").SetData(data).
		OnClick(func(e events.Event) {
			sv.This().(SliceViewer).PasteAtIdx(data.(mimedata.Mimes), idx)
		})
	gi.NewButton(m).SetText("Insert After").SetData(data).
		OnClick(func(e events.Event) {
			sv.This().(SliceViewer).PasteAtIdx(data.(mimedata.Mimes), idx+1)
		})
	gi.NewButton(m).SetText("Cancel").SetData(data)
}

// PasteMenu performs a paste from the clipboard using given data -- pops up
// a menu to determine what specifically to do
func (sv *SliceViewBase) PasteMenu(md mimedata.Mimes, idx int) {
	sv.UnselectAllIdxs()
	mf := func(m *gi.Scene) {
		sv.MakePasteMenu(m, md, idx)
	}
	pos := sv.IdxPos(idx)
	gi.NewMenu(mf, sv.This().(gi.Widget), pos).Run()
}

// PasteAssign assigns mime data (only the first one!) to this idx
func (sv *SliceViewBase) PasteAssign(md mimedata.Mimes, idx int) {
	sl := sv.FromMimeData(md)
	if len(sl) == 0 {
		return
	}
	updt := sv.UpdateStart()
	ns := sl[0]
	sv.SliceNPVal.Index(idx).Set(reflect.ValueOf(ns).Elem())
	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.SetChanged()
	sv.UpdateEndRender(updt)
}

// PasteAtIdx inserts object(s) from mime data at (before) given slice index
func (sv *SliceViewBase) PasteAtIdx(md mimedata.Mimes, idx int) {
	sl := sv.FromMimeData(md)
	if len(sl) == 0 {
		return
	}
	svl := reflect.ValueOf(sv.Slice)
	svnp := sv.SliceNPVal
	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	for _, ns := range sl {
		sz := svnp.Len()
		svnp = reflect.Append(svnp, reflect.ValueOf(ns).Elem())
		svl.Elem().Set(svnp)
		if idx >= 0 && idx < sz {
			reflect.Copy(svnp.Slice(idx+1, sz+1), svnp.Slice(idx, sz))
			svnp.Index(idx).Set(reflect.ValueOf(ns).Elem())
			svl.Elem().Set(svnp)
		}
		idx++
	}

	sv.SliceNPVal = laser.NonPtrValue(reflect.ValueOf(sv.Slice)) // need to update after changes

	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.SetChanged()
	sv.SelectIdxAction(idx, events.SelectOne)
	sv.Update()
}

// Duplicate copies selected items and inserts them after current selection --
// return idx of start of duplicates if successful, else -1
func (sv *SliceViewBase) Duplicate() int {
	nitms := len(sv.SelIdxs)
	if nitms == 0 {
		return -1
	}
	ixs := sv.SelectedIdxsList(true) // descending sort -- last first
	pasteAt := ixs[0]
	sv.CopyIdxs(true)
	dt := sv.This().(SliceViewer).MimeDataType()
	md := sv.EventMgr().ClipBoard().Read([]string{dt})
	sv.This().(SliceViewer).PasteAtIdx(md, pasteAt)
	return pasteAt
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// DragNDropStart starts a drag-n-drop
func (sv *SliceViewBase) DragNDropStart() {
	nitms := len(sv.SelIdxs)
	if nitms == 0 {
		return
	}
	md := sv.This().(SliceViewer).CopySelToMime()
	_ = md
	ixs := sv.SelectedIdxsList(false) // ascending
	w, ok := sv.This().(SliceViewer).RowFirstWidget(ixs[0])
	if ok {
		sp := &gi.Sprite{}
		sp.GrabRenderFrom(w)
		gi.ImageClearer(sp.Pixels, 50.0)
		// todo:
		// sv.ParentRenderWin().StartDragNDrop(sv.This(), md, sp)
	}
}

// DragNDropTarget handles a drag-n-drop drop
func (sv *SliceViewBase) DragNDropTarget(de events.Event) {
	// de.Target = sv.This()
	// if de.Mod == events.DropLink {
	// 	de.Mod = events.DropCopy // link not supported -- revert to copy
	// }
	// idx, ok := sv.IdxFromPos(de.LocalPos().Y)
	// if ok {
	// 	de.SetHandled()
	// 	sv.CurIdx = idx
	// 	if dpr, ok := sv.This().(gi.DragNDropper); ok {
	// 		dpr.Drop(de.Data, de.Mod)
	// 	} else {
	// 		sv.Drop(de.Data, de.Mod)
	// 	}
	// }
}

// MakeDropMenu makes the menu of options for dropping on a target
func (sv *SliceViewBase) MakeDropMenu(m *gi.Scene, data any, mod events.DropMods, idx int) {
	switch mod {
	case events.DropCopy:
		gi.NewLabel(m, "copy").SetText("Copy (Shift=Move):")
	case events.DropMove:
		gi.NewLabel(m, "move").SetText("Move:")
	}
	if mod == events.DropCopy {
		gi.NewButton(m, "assign-to").SetText("Assign To").SetData(data).
			OnClick(func(e events.Event) {
				sv.DropAssign(data.(mimedata.Mimes), idx)
			})
	}
	gi.NewButton(m, "insert-before").SetText("Insert Before").SetData(data).
		OnClick(func(e events.Event) {
			sv.DropBefore(data.(mimedata.Mimes), mod, idx) // captures mod
		})
	gi.NewButton(m, "insert-after").SetText("Insert After").SetData(data).
		OnClick(func(e events.Event) {
			sv.DropAfter(data.(mimedata.Mimes), mod, idx) // captures mod
		})
	gi.NewButton(m, "cancel").SetText("Cancel").SetData(data).
		OnClick(func(e events.Event) {
			sv.DropCancel()
		})
}

// Drop pops up a menu to determine what specifically to do with dropped items
// this satisfies gi.DragNDropper interface, and can be overwritten in subtypes
func (sv *SliceViewBase) Drop(md mimedata.Mimes, mod events.DropMods) {
	mf := func(m *gi.Scene) {
		sv.MakeDropMenu(m, md, mod, sv.CurIdx)
	}
	pos := sv.IdxPos(sv.CurIdx)
	gi.NewMenu(mf, sv.This().(gi.Widget), pos).Run()
}

// DropAssign assigns mime data (only the first one!) to this node
func (sv *SliceViewBase) DropAssign(md mimedata.Mimes, idx int) {
	sv.DraggedIdxs = nil
	sv.This().(SliceViewer).PasteAssign(md, idx)
	sv.DragNDropFinalize(events.DropCopy)
}

// DragNDropFinalize is called to finalize actions on the Source node prior to
// performing target actions -- mod must indicate actual action taken by the
// target, including ignore -- ends up calling DragNDropSource if us..
func (sv *SliceViewBase) DragNDropFinalize(mod events.DropMods) {
	sv.UnselectAllIdxs()
	// sv.ParentRenderWin().FinalizeDragNDrop(mod)
}

// DragNDropSource is called after target accepts the drop -- we just remove
// elements that were moved
func (sv *SliceViewBase) DragNDropSource(de events.Event) {
	// if de.Mod != events.DropMove || len(sv.DraggedIdxs) == 0 {
	// 	return
	// }

	updt := sv.UpdateStart()
	defer sv.UpdateEnd(updt)

	sort.Slice(sv.DraggedIdxs, func(i, j int) bool {
		return sv.DraggedIdxs[i] > sv.DraggedIdxs[j]
	})
	idx := sv.DraggedIdxs[0]
	for _, i := range sv.DraggedIdxs {
		sv.This().(SliceViewer).SliceDeleteAt(i)
	}
	sv.DraggedIdxs = nil
	sv.SelectIdxAction(idx, events.SelectOne)
}

// SaveDraggedIdxs saves selectedindexes into dragged indexes
// taking into account insertion at idx
func (sv *SliceViewBase) SaveDraggedIdxs(idx int) {
	sz := len(sv.SelIdxs)
	if sz == 0 {
		sv.DraggedIdxs = nil
		return
	}
	ixs := sv.SelectedIdxsList(false) // ascending
	sv.DraggedIdxs = make([]int, len(ixs))
	for i, ix := range ixs {
		if ix > idx {
			sv.DraggedIdxs[i] = ix + sz // make room for insertion
		} else {
			sv.DraggedIdxs[i] = ix
		}
	}
}

// DropBefore inserts object(s) from mime data before this node
func (sv *SliceViewBase) DropBefore(md mimedata.Mimes, mod events.DropMods, idx int) {
	sv.SaveDraggedIdxs(idx)
	sv.This().(SliceViewer).PasteAtIdx(md, idx)
	sv.DragNDropFinalize(mod)
}

// DropAfter inserts object(s) from mime data after this node
func (sv *SliceViewBase) DropAfter(md mimedata.Mimes, mod events.DropMods, idx int) {
	sv.SaveDraggedIdxs(idx + 1)
	sv.This().(SliceViewer).PasteAtIdx(md, idx+1)
	sv.DragNDropFinalize(mod)
}

// DropCancel cancels the drop action e.g., preventing deleting of source
// items in a Move case
func (sv *SliceViewBase) DropCancel() {
	sv.DragNDropFinalize(events.DropIgnore)
}

//////////////////////////////////////////////////////////////////////////////
//    Events

func (sv *SliceViewBase) StdCtxtMenu(m *gi.Scene, idx int) {
	if sv.IsReadOnly() || sv.Is(SliceViewIsArray) {
		return
	}
	gi.NewButton(m).SetText("Copy").SetData(idx).
		OnClick(func(e events.Event) {
			sv.CopyIdxs(true)
		})
	gi.NewButton(m).SetText("Cut").SetData(idx).
		OnClick(func(e events.Event) {
			sv.CutIdxs()
		})
	gi.NewButton(m).SetText("Paste").SetData(idx).
		OnClick(func(e events.Event) {
			sv.PasteIdx(idx)
		})
	gi.NewButton(m).SetText("Duplicate").SetData(idx).
		OnClick(func(e events.Event) {
			sv.Duplicate()
		})
}

func (sv *SliceViewBase) ItemCtxtMenu(idx int) {
	val := sv.SliceVal(idx)
	if val == nil {
		return
	}

	// TODO(kai/menu): CtxtMenuView
	// if CtxtMenuView(val, sv.IsReadOnly(), sv.Sc, &menu) {
	// 	if sv.ShowViewCtxtMenu {
	// 		menu.AddSeparator("sep-svmenu")
	// 		sv.This().(SliceViewer).StdCtxtMenu(&menu, idx)
	// 	}
	// } else {

	// TODO(kai/menu): handle empty menu
	mf := func(m *gi.Scene) {
		sv.This().(SliceViewer).StdCtxtMenu(m, idx)
	}
	pos := sv.IdxPos(idx)
	// if pos == (image.Point{}) {
	// 	em := sv.EventMgr()
	// 	if em != nil {
	// 		pos = em.LastMousePos
	// 	}
	// }
	gi.NewMenu(mf, sv.This().(gi.Widget), pos).Run()
}

// KeyInputNav supports multiple selection navigation keys
func (sv *SliceViewBase) KeyInputNav(kt events.Event) {
	kf := keyfun.Of(kt.KeyChord())
	selMode := events.SelectModeBits(kt.Modifiers())
	if selMode == events.SelectOne {
		if sv.Is(SliceViewSelectMode) {
			selMode = events.ExtendContinuous
		}
	}
	switch kf {
	case keyfun.CancelSelect:
		sv.UnselectAllIdxs()
		sv.SetFlag(false, SliceViewSelectMode)
		kt.SetHandled()
	case keyfun.MoveDown:
		sv.MoveDownAction(selMode)
		kt.SetHandled()
	case keyfun.MoveUp:
		sv.MoveUpAction(selMode)
		kt.SetHandled()
	case keyfun.PageDown:
		sv.MovePageDownAction(selMode)
		kt.SetHandled()
	case keyfun.PageUp:
		sv.MovePageUpAction(selMode)
		kt.SetHandled()
	case keyfun.SelectMode:
		sv.SetFlag(!sv.Is(SliceViewSelectMode), SliceViewSelectMode)
		kt.SetHandled()
	case keyfun.SelectAll:
		sv.SelectAllIdxs()
		sv.SetFlag(false, SliceViewSelectMode)
		kt.SetHandled()
	}
}

func (sv *SliceViewBase) KeyInputEditable(kt events.Event) {
	if gi.KeyEventTrace {
		fmt.Printf("SliceViewBase KeyInput: %v\n", sv.Path())
	}
	sv.KeyInputNav(kt)
	if kt.IsHandled() {
		return
	}
	idx := sv.SelIdx
	kf := keyfun.Of(kt.KeyChord())
	switch kf {
	// case keyfun.Delete: // too dangerous
	// 	sv.This().(SliceViewer).SliceDeleteAt(sv.SelectedIdx)
	// 	sv.SelectMode = false
	// 	sv.SelectIdxAction(idx, events.SelectOne)
	// 	kt.SetHandled()
	case keyfun.Duplicate:
		nidx := sv.Duplicate()
		sv.SetFlag(false, SliceViewSelectMode)
		if nidx >= 0 {
			sv.SelectIdxAction(nidx, events.SelectOne)
		}
		kt.SetHandled()
	case keyfun.Insert:
		sv.This().(SliceViewer).SliceNewAt(idx)
		sv.SetFlag(false, SliceViewSelectMode)
		sv.SelectIdxAction(idx+1, events.SelectOne) // todo: somehow nidx not working
		kt.SetHandled()
	case keyfun.InsertAfter:
		sv.This().(SliceViewer).SliceNewAt(idx + 1)
		sv.SetFlag(false, SliceViewSelectMode)
		sv.SelectIdxAction(idx+1, events.SelectOne)
		kt.SetHandled()
	case keyfun.Copy:
		sv.CopyIdxs(true)
		sv.SetFlag(false, SliceViewSelectMode)
		sv.SelectIdxAction(idx, events.SelectOne)
		kt.SetHandled()
	case keyfun.Cut:
		sv.CutIdxs()
		sv.SetFlag(false, SliceViewSelectMode)
		kt.SetHandled()
	case keyfun.Paste:
		sv.PasteIdx(sv.SelIdx)
		sv.SetFlag(false, SliceViewSelectMode)
		kt.SetHandled()
	}
}

func (sv *SliceViewBase) KeyInputReadOnly(kt events.Event) {
	if gi.KeyEventTrace {
		fmt.Printf("SliceViewBase ReadOnly KeyInput: %v\n", sv.Path())
	}
	if sv.Is(SliceViewReadOnlyMultiSel) {
		sv.KeyInputNav(kt)
		if kt.IsHandled() {
			return
		}
	}
	kf := keyfun.Of(kt.KeyChord())
	idx := sv.SelIdx
	switch {
	case kf == keyfun.MoveDown:
		ni := idx + 1
		if ni < sv.SliceSize {
			sv.ScrollToIdx(ni)
			sv.UpdateSelectIdx(ni, true)
			kt.SetHandled()
		}
	case kf == keyfun.MoveUp:
		ni := idx - 1
		if ni >= 0 {
			sv.ScrollToIdx(ni)
			sv.UpdateSelectIdx(ni, true)
			kt.SetHandled()
		}
	case kf == keyfun.PageDown:
		ni := min(idx+sv.VisRows-1, sv.SliceSize-1)
		sv.ScrollToIdx(ni)
		sv.UpdateSelectIdx(ni, true)
		kt.SetHandled()
	case kf == keyfun.PageUp:
		ni := max(idx-(sv.VisRows-1), 0)
		sv.ScrollToIdx(ni)
		sv.UpdateSelectIdx(ni, true)
		kt.SetHandled()
	case kf == keyfun.Enter || kf == keyfun.Accept || kt.KeyRune() == ' ':
		sv.Send(events.DoubleClick, kt)
		kt.SetHandled()
	}
}

func (sv *SliceViewBase) HandleSliceViewEvents() {
	sv.On(events.Scroll, func(e events.Event) {
		e.SetHandled()
		se := e.(*events.MouseScroll)
		sbb := sv.This().(SliceViewer).ScrollBar()
		cur := float32(sbb.Pos)
		sbb.SetSliderPosAction(cur + float32(se.DimDelta(mat32.Y))) // reverse dir for "natural"
	})
	sv.OnKeyChord(func(e events.Event) {
		if sv.IsReadOnly() {
			if sv.Is(SliceViewReadOnlyKeyNav) {
				sv.KeyInputReadOnly(e)
			}
		} else {
			sv.KeyInputEditable(e)
		}
	})
	sv.OnClick(func(e events.Event) {
		sv.GrabFocus()
	})

	// todo: doubleclick unselectallidxs is crashing with recursive loop
	// sv.OnDoubleClick(func(e events.Event) {
	// 	si := sv.SelectedIdx
	// 	sv.UnselectAllIdxs()
	// 	sv.SelectIdx(si)
	// 	sv.Send(events.DoubleClick, e)
	// 	e.SetHandled()
	// })

	// todo ctxmenu
	// sv.Onwe.AddFunc(events.MouseUp, gi.LowRawPri, func(recv, send ki.Ki, sig int64, d any) {
	// 	me := d.(events.Event)
	// 	svv := recv.Embed(TypeSliceViewBase).(*SliceViewBase)
	// 	// if !svv.StateIs(states.Focused) {
	// 	// 	svv.GrabFocus()
	// 	// }
	// 	if me.Button == events.Right && me.Action == events.Release {
	// 		svv.This().(SliceViewer).ItemCtxtMenu(svv.SelectedIdx)
	// 		me.SetHandled()
	// 	}
	// })
	// svwe.AddFunc(goosi.DNDEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
	// 	de := d.(events.Event)
	// 	svv := recv.Embed(TypeSliceViewBase).(*SliceViewBase)
	// 	switch de.Action {
	// 	case events.Start:
	// 		svv.DragNDropStart()
	// 	case events.DropOnTarget:
	// 		svv.DragNDropTarget(de)
	// 	case events.DropFmSource:
	// 		svv.DragNDropSource(de)
	// 	}
	// })
	// sg := sv.This().(SliceViewer).SliceGrid()
	// if sg != nil {
	// 	sgwe.AddFunc(goosi.DNDFocusEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
	// 		de := d.(*events.FocusEvent)
	// 		sgg := recv.Embed(gi.FrameType).(*gi.Frame)
	// 		switch de.Action {
	// 		case events.Enter:
	// 			sgg.ParentRenderWin().DNDSetCursor(de.Mod)
	// 		case events.Exit:
	// 			sgg.ParentRenderWin().DNDNotCursor()
	// 		case events.Hover:
	// 			// nothing here?
	// 		}
	// 	})
	// }
}
