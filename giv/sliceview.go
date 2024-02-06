// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/mimedata"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
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
	StyleFunc SliceViewStyleFunc `copier:"-" view:"-" json:"-" xml:"-"`
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
// * For a given slice type, the full set of widgets for VisRows is created
//   during the Layout process (Initially MinRows are created to get row height,
//   then the full set of visible rows is created during SizeFinal).  The
//   SliceViewConfiged flag indicates that this has been done -- when the slice
//   type changes (SetSlice), this flag is reset and a new layout is triggered.
//   Other externally-driven layout changes just update VisRows accordingly.
//
// * UpdateWidgets updates the view based on any changes in the slice data,
//   scrolling, etc.
//
// * The standard Update call will do the right thing: Config does UpdateWidgets
//   whenever SliceViewConfiged is set, and layout makes widgets as needed.
//   ApplyStyle is generally neeed after UpdateWidgets (state flag changes)
//   followed by Render.
//
// * SliceViewGrid handles all the layout logic to start with a minimum number of
//   rows and then computes the total number visible based on allocated size.

// SliceViewFlags extend WidgetFlags to hold SliceView state
type SliceViewFlags gi.WidgetFlags //enums:bitflag -trim-prefix SliceView

const (
	// SliceViewConfigured indicates that the widgets have been configured
	SliceViewConfigured SliceViewFlags = SliceViewFlags(gi.WidgetFlagsN) + iota

	// SliceViewIsArray is whether the slice is actually an array -- no modifications -- set by SetSlice
	SliceViewIsArray

	// SliceViewShowIndex is whether to show index or not
	SliceViewShowIndex

	// SliceViewReadOnlyKeyNav is whether support key navigation when ReadOnly (default true).
	// uses a capture of up / down events to manipulate selection, not focus.
	SliceViewReadOnlyKeyNav

	// SliceViewSelectMode is whether to be in select rows mode or editing mode
	SliceViewSelectMode

	// SliceViewReadOnlyMultiSel: if view is ReadOnly, default selection mode is to choose one row only.
	// If this is true, standard multiple selection logic with modifier keys is instead supported
	SliceViewReadOnlyMultiSel

	// SliceViewInFocusGrab is a guard for recursive focus grabbing
	SliceViewInFocusGrab

	// SliceViewInFullRebuild is a guard for recursive rebuild
	SliceViewInFullRebuild
)

// SliceViewer is the interface used by SliceViewBase to
// support any abstractions needed for different types of slice views.
type SliceViewer interface {
	// AsSliceViewBase returns the base for direct access to relevant fields etc
	AsSliceViewBase() *SliceViewBase

	// SliceGrid returns the SliceViewGrid grid Layout widget,
	// which contains all the fields and values
	SliceGrid() *SliceViewGrid

	// RowWidgetNs returns number of widgets per row and
	// offset for index label
	RowWidgetNs() (nWidgPerRow, idxOff int)

	// UpdtSliceSize updates the current size of the slice
	// and sets SliceSize if changed.
	UpdtSliceSize() int

	// StyleValueWidget performs additional value widget styling
	StyleValueWidget(w gi.Widget, s *styles.Style, row, col int)

	// ConfigRows configures VisRows worth of widgets
	// to display slice data.
	ConfigRows()

	// UpdateWidgets updates the row widget display to
	// represent the current state of the slice data,
	// including which range of data is being displayed.
	// This is called for scrolling, navigation etc.
	UpdateWidgets()

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

	// SliceNewAt inserts a new blank element at given
	// index in the slice. -1 means the end.
	SliceNewAt(idx int)

	// SliceDeleteAt deletes element at given index from slice
	// if updt is true, then update the grid after
	SliceDeleteAt(idx int)

	// WidgetIndex returns the row and column indexes for given widget.
	// Typically this is decoded from the name of the widget.
	WidgetIndex(w gi.Widget) (row, col int)

	// MimeDataType returns the data type for mime clipboard
	// (copy / paste) data e.g., fi.DataJson
	MimeDataType() string

	// CopySelToMime copies selected rows to mime data
	CopySelToMime() mimedata.Mimes

	// PasteAssign assigns mime data (only the first one!) to this idx
	PasteAssign(md mimedata.Mimes, idx int)

	// PasteAtIdx inserts object(s) from mime data at
	// (before) given slice index
	PasteAtIdx(md mimedata.Mimes, idx int)

	MakePasteMenu(m *gi.Scene, md mimedata.Mimes, idx int, mod events.DropMods, fun func())
	DragStart(e events.Event)
	DragDrop(e events.Event)
	DropFinalize(de *events.DragDrop)
	DropDeleteSource(e events.Event)
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
	Slice any `set:"-" json:"-" xml:"-"`

	// MinRows specifies the minimum number of rows to display, to ensure
	// at least this amount is displayed.
	MinRows int `default:"4"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// optional mutex that, if non-nil, will be used around any updates that
	// read / modify the underlying Slice data.
	// Can be used to protect against random updating if your code has specific
	// update points that can be likewise protected with this same mutex.
	ViewMu *sync.Mutex `copier:"-" view:"-" json:"-" xml:"-"`

	// Changed indicates whether the underlying slice
	// has been edited in any way
	Changed bool `set:"-"`

	// current selection value -- initially select this value if set
	SelVal any `copier:"-" view:"-" json:"-" xml:"-"`

	// index of currently selected item
	SelIdx int `copier:"-" json:"-" xml:"-"`

	// index of row to select at start
	InitSelIdx int `copier:"-" json:"-" xml:"-"`

	// list of currently-selected slice indexes
	SelIdxs map[int]struct{} `set:"-" copier:"-"`

	// LastClick is the last row that has been clicked on.
	// This is used to prevent erroneous double click events
	// from being sent when the user clicks on multiple different
	// rows in quick succession.
	LastClick int `set:"-" copier:"-" json:"-" xml:"-"`

	// NormalCursor is the cached cursor to display when there
	// is no row being hovered.
	NormalCursor cursors.Cursor `copier:"-" xml:"-" json:"-" set:"-"`

	// non-ptr reflect.Value of the slice
	SliceNPVal reflect.Value `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// Value for the slice itself, if this was created within value view framework -- otherwise nil
	SliceValView Value `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// Value representations of the slice values
	Values []Value `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// currently-hovered row
	HoverRow int `set:"-" view:"-" copier:"-" json:"-" xml:"-"`

	// list of currently-dragged indexes
	DraggedIdxs []int `set:"-" view:"-" copier:"-" json:"-" xml:"-"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `view:"-" copier:"-" json:"-" xml:"-"`

	// total number of rows visible in allocated display size
	VisRows int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// starting slice index of visible rows
	StartIdx int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// size of slice
	SliceSize int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// iteration through the configuration process, reset when a new slice type is set
	ConfigIter int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// temp idx state for e.g., dnd
	TmpIdx int `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// ElVal is a Value representation of the underlying element type
	// which is used whenever there are no slice elements available
	ElVal reflect.Value `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// maximum width of value column in chars, if string
	MaxWidth int `set:"-" copier:"-" json:"-" xml:"-"`
}

func (sv *SliceViewBase) FlagType() enums.BitFlagSetter {
	return (*SliceViewFlags)(&sv.Flags)
}

func (sv *SliceViewBase) OnInit() {
	sv.Frame.OnInit()
	sv.HandleEvents()
	sv.SetStyles()
	sv.AddContextMenu(sv.ContextMenu)
}

func (sv *SliceViewBase) SetStyles() {
	sv.InitSelIdx = -1
	sv.HoverRow = -1
	sv.MinRows = 4
	sv.SetFlag(false, SliceViewSelectMode)
	sv.SetFlag(true, SliceViewShowIndex)
	sv.SetFlag(true, SliceViewReadOnlyKeyNav)
	svi := sv.This().(SliceViewer)

	sv.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable, abilities.DoubleClickable, abilities.TripleClickable)
		s.Direction = styles.Column
		// absorb horizontal here, vertical in view
		s.Overflow.X = styles.OverflowAuto
		s.Grow.Set(1, 1)
	})
	sv.StyleFinal(func(s *styles.Style) {
		sv.NormalCursor = s.Cursor
	})
	sv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(sv) {
		case "grid": // slice grid
			sg := w.(*SliceViewGrid)
			sg.Style(func(s *styles.Style) {
				sg.MinRows = sv.MinRows
				s.Display = styles.Grid
				nWidgPerRow, _ := svi.RowWidgetNs()
				s.Columns = nWidgPerRow
				s.Grow.Set(1, 1)
				s.Overflow.Y = styles.OverflowAuto
				s.Gap.Set(units.Em(0.5)) // note: match header
				s.Align.Items = styles.Center
				s.Justify.Items = styles.Center
				// baseline mins:
				s.Min.X.Ch(20)
				s.Min.Y.Em(6)
			})
			sg.OnClick(func(e events.Event) {
				sv.SetFocusEvent()
				row, _ := sg.IndexFromPixel(e.Pos())
				sv.UpdateSelectRow(row, e.SelectMode())
				sv.LastClick = row + sv.StartIdx
			})
			sg.ContextMenus = sv.ContextMenus
		}
		if w.Parent().PathFrom(sv) == "grid" {
			switch {
			case strings.HasPrefix(w.Name(), "index-"):
				wb := w.AsWidget()
				w.Style(func(s *styles.Style) {
					s.SetAbilities(true, abilities.Draggable, abilities.Droppable, abilities.DoubleClickable)
					s.Cursor = cursors.None
					nd := mat32.Log10(float32(sv.SliceSize))
					nd = max(nd, 3)
					s.Min.X.Ch(nd + 2)
					s.Padding.Right.Dp(4)
					s.Text.Align = styles.End
					s.Min.Y.Em(1)
					s.GrowWrap = false
				})
				wb.ContextMenus = sv.ContextMenus
				wb.OnDoubleClick(func(e events.Event) {
					sv.Send(events.DoubleClick, e)
				})
				w.On(events.DragStart, func(e events.Event) {
					if sv.This() == nil || sv.Is(ki.Deleted) {
						return
					}
					svi.DragStart(e)
				})
				w.On(events.DragEnter, func(e events.Event) {
					if sv.This() == nil || sv.Is(ki.Deleted) {
						return
					}
					sv.SetState(true, states.DragHovered)
					sv.ApplyStyle()
					sv.SetNeedsRender(true)
					e.SetHandled()
				})
				w.On(events.DragLeave, func(e events.Event) {
					if sv.This() == nil || sv.Is(ki.Deleted) {
						return
					}
					sv.SetState(false, states.DragHovered)
					sv.ApplyStyle()
					sv.SetNeedsRender(true)
					e.SetHandled()
				})
				w.On(events.Drop, func(e events.Event) {
					if sv.This() == nil || sv.Is(ki.Deleted) {
						return
					}
					svi.DragDrop(e)
				})
				w.On(events.DropDeleteSource, func(e events.Event) {
					if sv.This() == nil || sv.Is(ki.Deleted) {
						return
					}
					svi.DropDeleteSource(e)
				})
			case strings.HasPrefix(w.Name(), "value-"):
				wb := w.AsWidget()
				w.Style(func(s *styles.Style) {
					if sv.IsReadOnly() {
						s.SetAbilities(true, abilities.DoubleClickable)
						s.SetAbilities(false, abilities.Hoverable, abilities.Focusable, abilities.Activatable, abilities.TripleClickable)
						wb.SetReadOnly(true)
					}
					row, col := sv.This().(SliceViewer).WidgetIndex(w)
					sv.This().(SliceViewer).StyleValueWidget(w, s, row, col)
					if row < sv.SliceSize {
						sv.This().(SliceViewer).StyleRow(w, row, col)
					}
				})
				wb.OnSelect(func(e events.Event) {
					e.SetHandled()
					row, _ := sv.This().(SliceViewer).WidgetIndex(w)
					sv.UpdateSelectRow(row, e.SelectMode())
					sv.LastClick = row + sv.StartIdx
				})
				wb.OnDoubleClick(func(e events.Event) {
					sv.Send(events.DoubleClick, e)
				})
				wb.ContextMenus = sv.ContextMenus
			}
		}
	})
}

// StyleValueWidget performs additional value widget styling
func (sv *SliceViewBase) StyleValueWidget(w gi.Widget, s *styles.Style, row, col int) {
	if sv.MaxWidth > 0 {
		hv := units.Ch(float32(sv.MaxWidth))
		s.Min.X.Val = max(s.Min.X.Val, hv.Convert(s.Min.X.Un, &s.UnContext).Val)
		s.Max.X.Val = max(s.Max.X.Val, hv.Convert(s.Max.X.Un, &s.UnContext).Val)
	}
}

func (sv *SliceViewBase) AsSliceViewBase() *SliceViewBase {
	return sv
}

func (sv *SliceViewBase) SetSliceBase() {
	sv.SetFlag(false, SliceViewConfigured, SliceViewSelectMode)
	sv.ConfigIter = 0
	sv.StartIdx = 0
	sv.VisRows = sv.MinRows
	if !sv.IsReadOnly() {
		sv.SelIdx = -1
	}
	sv.ResetSelectedIdxs()
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
		newslc = sv.Slice != sl
	}
	if !newslc && sv.Is(SliceViewConfigured) {
		sv.ConfigIter = 0
		sv.Update()
		return sv
	}
	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	sv.SetSliceBase()
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
	sv.Update()
	return sv
}

// IsNil returns true if the Slice is nil
func (sv *SliceViewBase) IsNil() bool {
	return laser.AnyIsNil(sv.Slice)
}

// RowFromEventPos returns the widget row, slice index, and
// whether the index is in slice range, for given event position.
func (sv *SliceViewBase) RowFromEventPos(e events.Event) (row, idx int, isValid bool) {
	sg := sv.This().(SliceViewer).SliceGrid()
	row, _ = sg.IndexFromPixel(e.Pos())
	idx = row + sv.StartIdx
	isValid = true
	if row < 0 || idx >= sv.SliceSize {
		isValid = false
	}
	return
}

// ClickSelectEvent is a helper for processing selection events
// based on a mouse click, which could be a double or triple
// in addition to a regular click.
// Returns false if no further processing should occur,
// because the user clicked outside the range of active rows.
func (sv *SliceViewBase) ClickSelectEvent(e events.Event) bool {
	row, _, isValid := sv.RowFromEventPos(e)
	if !isValid {
		e.SetHandled()
	} else {
		sv.UpdateSelectRow(row, e.SelectMode())
	}
	return isValid
}

// BindSelect makes the slice view a read-only selection slice view and then
// binds its events to its scene and its current selection index to the given value.
func (sv *SliceViewBase) BindSelect(val *int) *SliceViewBase {
	sv.SetReadOnly(true)
	sv.OnSelect(func(e events.Event) {
		*val = sv.SelIdx
	})
	sv.OnDoubleClick(func(e events.Event) {
		if sv.ClickSelectEvent(e) {
			*val = sv.SelIdx
			sv.Scene.SendKeyFun(keyfun.Accept, e) // activates Ok button code
		}
	})
	return sv
}

// ConfigWidget configures a standard setup of the overall Frame
func (sv *SliceViewBase) ConfigWidget() {
	sv.ConfigSliceView()
}

// ConfigSliceView handles entire config.
// ReConfig calls this, followed by ApplyStyleTree so we don't need to call that.
func (sv *SliceViewBase) ConfigSliceView() {
	if sv.Is(SliceViewConfigured) {
		sv.This().(SliceViewer).UpdateWidgets()
		return
	}
	updt := sv.UpdateStart()
	sv.ConfigFrame()
	sv.This().(SliceViewer).ConfigRows()
	sv.This().(SliceViewer).UpdateWidgets()
	sv.ApplyStyleTree()
	sv.UpdateEndLayout(updt)
}

func (sv *SliceViewBase) ConfigFrame() {
	if sv.HasChildren() {
		return
	}
	sv.VisRows = sv.MinRows
	NewSliceViewGrid(sv, "grid")
}

// SliceGrid returns the SliceGrid grid frame widget, which contains all the
// fields and values
func (sv *SliceViewBase) SliceGrid() *SliceViewGrid {
	return sv.ChildByName("grid", 0).(*SliceViewGrid)
}

// RowWidgetNs returns number of widgets per row and offset for index label
func (sv *SliceViewBase) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 2
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

// WidgetIndex returns the row and column indexes for given widget.
// Typically this is decoded from the name of the widget.
func (sv *SliceViewBase) WidgetIndex(w gi.Widget) (row, col int) {
	nm := w.Name()
	if strings.Contains(nm, "value-") {
		idx := grr.Log1(strconv.Atoi(strings.TrimPrefix(nm, "value-")))
		row = sv.StartIdx + idx
	} else if strings.Contains(nm, "index-") {
		idx := grr.Log1(strconv.Atoi(strings.TrimPrefix(nm, "index-")))
		row = sv.StartIdx + idx
	}
	return
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

// UpdateScroll updates the scroll value
func (sv *SliceViewBase) UpdateScroll() {
	sg := sv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}
	sg.UpdateScroll(sv.StartIdx)
}

// ConfigRows configures VisRows worth of widgets
// to display slice data.
func (sv *SliceViewBase) ConfigRows() {
	sg := sv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}
	sv.SetFlag(true, SliceViewConfigured)
	sg.SetFlag(true, gi.LayoutNoKeys)

	sv.ViewMuLock()
	defer sv.ViewMuUnlock()

	sg.DeleteChildren(ki.DestroyKids)
	sv.Values = nil

	sv.This().(SliceViewer).UpdtSliceSize()

	if sv.IsNil() {
		return
	}

	nWidgPerRow, idxOff := sv.RowWidgetNs()
	nWidg := nWidgPerRow * sv.VisRows
	sg.Styles.Columns = nWidgPerRow

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
			idxlab.SetText(sitxt)
			idxlab.OnSelect(func(e events.Event) {
				e.SetHandled()
				sv.UpdateSelectRow(i, e.SelectMode())
				sv.LastClick = i + sv.StartIdx
			})
		}

		w := ki.NewOfType(vtyp).(gi.Widget)
		sg.SetChild(w, ridx+idxOff, valnm)
		vv.ConfigWidget(w)

		if !sv.IsReadOnly() {
			vvb := vv.AsValueBase()
			vvb.OnChange(func(e events.Event) {
				sv.SendChange()
			})
		}
		if i == 0 {
			sv.MaxWidth = 0
			_, isbase := vv.(*ValueBase)
			npv := laser.NonPtrValue(val)
			if isbase && sv.SliceSize > 0 && npv.Kind() == reflect.String {
				mxw := 0
				for rw := 0; rw < sv.SliceSize; rw++ {
					val := laser.OnePtrUnderlyingValue(sv.SliceNPVal.Index(rw)).Elem()
					str := val.String()
					mxw = max(mxw, len(str))
				}
				sv.MaxWidth = mxw
			}
		}
	}

	sv.ConfigTree()
	sv.ApplyStyleTree()
}

// UpdateWidgets updates the row widget display to
// represent the current state of the slice data,
// including which range of data is being displayed.
// This is called for scrolling, navigation etc.
func (sv *SliceViewBase) UpdateWidgets() {
	sg := sv.This().(SliceViewer).SliceGrid()
	if sg == nil || sv.VisRows == 0 || sg.VisRows == 0 || !sg.HasChildren() {
		return
	}
	updt := sg.UpdateStart()
	defer sg.UpdateEndRender(updt)

	sv.ViewMuLock()
	defer sv.ViewMuUnlock()

	sv.This().(SliceViewer).UpdtSliceSize()

	nWidgPerRow, idxOff := sv.RowWidgetNs()

	scrollTo := -1
	if sv.SelVal != nil {
		idx, ok := SliceIdxByValue(sv.Slice, sv.SelVal)
		if ok {
			sv.SelIdx = idx
			scrollTo = sv.SelIdx
		}
		sv.SelVal = nil
		sv.InitSelIdx = -1
	} else if sv.InitSelIdx >= 0 {
		sv.SelIdx = sv.InitSelIdx
		sv.InitSelIdx = -1
		scrollTo = sv.SelIdx
	}

	sv.UpdateStartIdx()
	for i := 0; i < sv.VisRows; i++ {
		i := i
		ridx := i * nWidgPerRow
		w := sg.Kids[ridx+idxOff].(gi.Widget)
		vv := sv.Values[i]
		si := sv.StartIdx + i // slice idx
		invis := si >= sv.SliceSize

		var idxlab *gi.Label
		if sv.Is(SliceViewShowIndex) {
			idxlab = sg.Kids[ridx].(*gi.Label)
			idxlab.SetTextUpdate(strconv.Itoa(si))
			idxlab.SetState(invis, states.Invisible)
		}
		w.SetState(invis, states.Invisible)
		if si < sv.SliceSize {
			val := laser.OnePtrUnderlyingValue(sv.SliceNPVal.Index(si)) // deal with pointer lists
			vv.SetSliceValue(val, sv.Slice, si, sv.TmpSave, sv.ViewPath)
			vv.SetReadOnly(sv.IsReadOnly())
			vv.UpdateWidget()

			if sv.IsReadOnly() {
				w.AsWidget().SetReadOnly(true)
			}
		} else {
			vv.SetSliceValue(sv.ElVal, sv.Slice, 0, sv.TmpSave, sv.ViewPath)
			vv.UpdateWidget()
			w.AsWidget().SetSelected(false)
			if sv.Is(SliceViewShowIndex) {
				idxlab.SetSelected(false)
			}
		}
	}
	if scrollTo >= 0 {
		sv.ScrollToIdx(scrollTo)
	}
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
				d := gi.NewBody().AddTitle("Slice New").AddText("Number and Type of Items to Insert:")
				nd := &gi.NewItemsData{}
				w := NewValue(d, nd).AsWidget()
				ki.ChildByType[*gi.Chooser](w, ki.Embeds).SetTypes(gti.AllEmbeddersOf(ownki.BaseType())).SetCurrentIndex(0)
				d.AddBottomBar(func(pw gi.Widget) {
					d.AddCancel(pw)
					d.AddOk(pw).OnClick(func(e events.Event) {
						updt := ownki.UpdateStart()
						for i := 0; i < nd.Number; i++ {
							nm := fmt.Sprintf("New%v%v", nd.Type.Name, idx+1+i)
							ownki.InsertNewChild(nd.Type, idx+1+i, nm)
						}
						sv.SetChanged()
						ownki.UpdateEnd(updt)
					})
				})
				d.NewDialog(sv).Run()
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
	sv.This().(SliceViewer).UpdateWidgets()
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
	defer sv.UpdateEndRender(updt)

	sv.SliceDeleteAtSel(idx)

	laser.SliceDeleteAt(sv.Slice, idx)

	sv.This().(SliceViewer).UpdtSliceSize()

	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}

	sv.ViewMuUnlock()
	sv.SetChanged()
	sv.This().(SliceViewer).UpdateWidgets()
}

// ConfigToolbar configures a [gi.Toolbar] for this view
func (sv *SliceViewBase) ConfigToolbar(tb *gi.Toolbar) {
	if laser.AnyIsNil(sv.Slice) {
		return
	}
	if sv.Is(SliceViewIsArray) || sv.IsReadOnly() {
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
	w.SetFocusEvent()
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
			if w.Geom.TotalBBox.Min.Y < posY && posY < w.Geom.TotalBBox.Max.Y {
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
	}
	if idx >= sv.StartIdx+sv.VisRows {
		sv.StartIdx = idx - (sv.VisRows - 4)
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
			sv.UpdateSelectIdx(idx, true, events.SelectOne)
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

// RowWidgetsFunc calls function on each widget in given row
// (row, not index), with an UpdateStart / EndRender wrapper
func (sv *SliceViewBase) RowWidgetsFunc(row int, fun func(w gi.Widget)) {
	if row < 0 {
		return
	}
	updt := sv.UpdateStart()
	defer sv.UpdateEndRender(updt)

	sg := sv.This().(SliceViewer).SliceGrid()
	nWidgPerRow, _ := sv.This().(SliceViewer).RowWidgetNs()
	rowidx := row * nWidgPerRow
	for col := 0; col < nWidgPerRow; col++ {
		kidx := rowidx + col
		if sg.Kids.IsValidIndex(kidx) == nil {
			w := sg.Child(rowidx).(gi.Widget)
			fun(w)
		}
	}
}

// SetRowWidgetsStateEvent sets given state conditional on given
// ability for widget, for given event.
// returns the row and whether it represents an valid slice idex
func (sv *SliceViewBase) SetRowWidgetsStateEvent(e events.Event, ability abilities.Abilities, on bool, state states.States) (int, bool) {
	row, _, isValid := sv.RowFromEventPos(e)
	if isValid {
		sv.SetRowWidgetsState(row, ability, on, state)
	}
	return row, isValid
}

// SetRowWidgetsState sets given state conditional on given
// ability for widget
func (sv *SliceViewBase) SetRowWidgetsState(row int, ability abilities.Abilities, on bool, state states.States) {
	sv.RowWidgetsFunc(row, func(w gi.Widget) {
		wb := w.AsWidget()
		if wb.AbilityIs(ability) {
			wb.SetState(on, state)
		}
	})
}

// SelectRowWidgets sets the selection state of given row of widgets
func (sv *SliceViewBase) SelectRowWidgets(row int, sel bool) {
	if row < 0 {
		return
	}
	sv.RowWidgetsFunc(row, func(w gi.Widget) {
		w.AsWidget().SetSelected(sel)
	})
}

// SelectIdxWidgets sets the selection state of given slice index
// returns false if index is not visible
func (sv *SliceViewBase) SelectIdxWidgets(idx int, sel bool) bool {
	if !sv.IsIdxVisible(idx) {
		return false
	}
	sv.SelectRowWidgets(idx-sv.StartIdx, sel)
	return true
}

// UpdateSelectRow updates the selection for the given row
func (sv *SliceViewBase) UpdateSelectRow(row int, selMode events.SelectModes) {
	idx := row + sv.StartIdx
	if row < 0 || idx >= sv.SliceSize {
		return
	}
	sel := !sv.IdxIsSelected(idx)
	sv.UpdateSelectIdx(idx, sel, selMode)
}

// UpdateSelectIdx updates the selection for the given index
func (sv *SliceViewBase) UpdateSelectIdx(idx int, sel bool, selMode events.SelectModes) {
	if sv.IsReadOnly() && !sv.Is(SliceViewReadOnlyMultiSel) {
		updt := sv.UpdateStart()
		defer sv.UpdateEndRender(updt)

		sv.UnselectAllIdxs()
		if sel || sv.SelIdx == idx {
			sv.SelIdx = idx
			sv.SelectIdx(idx)
		}
		sv.ApplyStyleTree()
		sv.This().(SliceViewer).UpdateWidgets()
		sv.Send(events.Select)
	} else {
		sv.SelectIdxAction(idx, selMode)
	}
}

// IdxIsSelected returns the selected status of given slice index
func (sv *SliceViewBase) IdxIsSelected(idx int) bool {
	sv.ViewMuLock()
	defer sv.ViewMuUnlock()
	if sv.IsReadOnly() {
		return idx == sv.SelIdx
	}
	_, ok := sv.SelIdxs[idx]
	return ok
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
	// sv.SelectIdxWidgets(idx, true)
}

// UnselectIdx unselects given idx (if selected)
func (sv *SliceViewBase) UnselectIdx(idx int) {
	if sv.IdxIsSelected(idx) {
		delete(sv.SelIdxs, idx)
	}
	// sv.SelectIdxWidgets(idx, false)
}

// UnselectAllIdxs unselects all selected idxs
func (sv *SliceViewBase) UnselectAllIdxs() {
	// for r := range sv.SelIdxs {
	// 	sv.SelectIdxWidgets(r, false)
	// }
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
		// sv.SelectIdxWidgets(idx, true)
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
	sv.ApplyStyleTree()
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
		*md = append(*md, &mimedata.Data{Type: fi.DataJson, Data: b})
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
		if d.Type == fi.DataJson {
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
// e.g., fi.DataJson
func (sv *SliceViewBase) MimeDataType() string {
	return fi.DataJson
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

// CopyIdxs copies selected idxs to goosi.Clipboard, optionally resetting the selection
func (sv *SliceViewBase) CopyIdxs(reset bool) { //gti:add
	nitms := len(sv.SelIdxs)
	if nitms == 0 {
		return
	}
	md := sv.This().(SliceViewer).CopySelToMime()
	if md != nil {
		sv.Clipboard().Write(md)
	}
	if reset {
		sv.UnselectAllIdxs()
	}
}

// DeleteIdxs deletes all selected indexes
func (sv *SliceViewBase) DeleteIdxs() { //gti:add
	if len(sv.SelIdxs) == 0 {
		return
	}
	updt := sv.UpdateStart()
	defer sv.UpdateEndRender(updt)

	ixs := sv.SelectedIdxsList(true) // descending sort
	for _, i := range ixs {
		sv.This().(SliceViewer).SliceDeleteAt(i)
	}
	sv.SetChanged()
	sv.This().(SliceViewer).UpdateWidgets()
}

// CutIdxs copies selected indexes to goosi.Clipboard and deletes selected indexes
func (sv *SliceViewBase) CutIdxs() { //gti:add
	if len(sv.SelIdxs) == 0 {
		return
	}
	updt := sv.UpdateStart()
	defer sv.UpdateEndRender(updt)

	sv.CopyIdxs(false)
	ixs := sv.SelectedIdxsList(true) // descending sort
	idx := ixs[0]
	sv.UnselectAllIdxs()
	for _, i := range ixs {
		sv.This().(SliceViewer).SliceDeleteAt(i)
	}
	sv.SetChanged()
	sv.SelectIdxAction(idx, events.SelectOne)
	sv.This().(SliceViewer).UpdateWidgets()
}

// PasteIdx pastes clipboard at given idx
func (sv *SliceViewBase) PasteIdx(idx int) { //gti:add
	sv.TmpIdx = idx
	dt := sv.This().(SliceViewer).MimeDataType()
	md := sv.Clipboard().Read([]string{dt})
	if md != nil {
		sv.PasteMenu(md, sv.TmpIdx)
	}
}

// MakePasteMenu makes the menu of options for paste events
func (sv *SliceViewBase) MakePasteMenu(m *gi.Scene, md mimedata.Mimes, idx int, mod events.DropMods, fun func()) {
	svi := sv.This().(SliceViewer)
	if mod == events.DropCopy {
		gi.NewButton(m).SetText("Assign to").OnClick(func(e events.Event) {
			svi.PasteAssign(md, idx)
			if fun != nil {
				fun()
			}
		})
	}
	gi.NewButton(m).SetText("Insert before").OnClick(func(e events.Event) {
		svi.PasteAtIdx(md, idx)
		if fun != nil {
			fun()
		}
	})
	gi.NewButton(m).SetText("Insert after").OnClick(func(e events.Event) {
		svi.PasteAtIdx(md, idx+1)
		if fun != nil {
			fun()
		}
	})
	gi.NewButton(m).SetText("Cancel")
}

// PasteMenu performs a paste from the clipboard using given data -- pops up
// a menu to determine what specifically to do
func (sv *SliceViewBase) PasteMenu(md mimedata.Mimes, idx int) {
	sv.UnselectAllIdxs()
	mf := func(m *gi.Scene) {
		sv.MakePasteMenu(m, md, idx, events.DropCopy, nil)
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
	defer sv.UpdateEndRender(updt)

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
	sv.This().(SliceViewer).UpdateWidgets()
}

// Duplicate copies selected items and inserts them after current selection --
// return idx of start of duplicates if successful, else -1
func (sv *SliceViewBase) Duplicate() int { //gti:add
	nitms := len(sv.SelIdxs)
	if nitms == 0 {
		return -1
	}
	ixs := sv.SelectedIdxsList(true) // descending sort -- last first
	pasteAt := ixs[0]
	sv.CopyIdxs(true)
	dt := sv.This().(SliceViewer).MimeDataType()
	md := sv.Clipboard().Read([]string{dt})
	sv.This().(SliceViewer).PasteAtIdx(md, pasteAt)
	return pasteAt
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

func (sv *SliceViewBase) DragStart(e events.Event) {
	nitms := len(sv.SelIdxs)
	if nitms == 0 {
		return
	}
	md := sv.This().(SliceViewer).CopySelToMime()
	ixs := sv.SelectedIdxsList(false) // ascending
	w, ok := sv.This().(SliceViewer).RowFirstWidget(ixs[0])
	if ok {
		sv.Scene.EventMgr.DragStart(w, md, e)
	}
}

func (sv *SliceViewBase) DragDrop(e events.Event) {
	de := e.(*events.DragDrop)
	svi := sv.This().(SliceViewer)
	pos := de.Pos()
	idx, ok := sv.IdxFromPos(pos.Y)
	if ok {
		// sv.DraggedIdxs = nil
		sv.TmpIdx = idx
		sv.SaveDraggedIdxs(idx)
		md := de.Data.(mimedata.Mimes)
		mf := func(m *gi.Scene) {
			sv.Scene.EventMgr.DragMenuAddModLabel(m, de.DropMod)
			svi.MakePasteMenu(m, md, idx, de.DropMod, func() {
				svi.DropFinalize(de)
			})
		}
		pos := sv.IdxPos(sv.TmpIdx)
		gi.NewMenu(mf, sv.This().(gi.Widget), pos).Run()
	}
}

// DropFinalize is called to finalize Drop actions on the Source node.
// Only relevant for DropMod == DropMove.
func (sv *SliceViewBase) DropFinalize(de *events.DragDrop) {
	sv.UnselectAllIdxs()
	sv.Scene.EventMgr.DropFinalize(de) // sends DropDeleteSource to Source
}

// DropDeleteSource handles delete source event for DropMove case
func (sv *SliceViewBase) DropDeleteSource(e events.Event) {
	// de := e.(*events.DragDrop)
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

//////////////////////////////////////////////////////////////////////////////
//    Events

func (sv *SliceViewBase) ContextMenu(m *gi.Scene) {
	if sv.IsReadOnly() || sv.Is(SliceViewIsArray) {
		return
	}
	gi.NewButton(m).SetText("Add row").SetIcon(icons.Add).OnClick(func(e events.Event) {
		sv.SliceNewAtRow(sv.SelIdx + 1)
	})
	gi.NewButton(m).SetText("Delete row").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		sv.SliceDeleteAtRow(sv.SelIdx)
	})
	gi.NewSeparator(m)
	gi.NewButton(m).SetText("Copy").SetIcon(icons.Copy).OnClick(func(e events.Event) {
		sv.CopyIdxs(true)
	})
	gi.NewButton(m).SetText("Cut").SetIcon(icons.Cut).OnClick(func(e events.Event) {
		sv.CutIdxs()
	})
	gi.NewButton(m).SetText("Paste").SetIcon(icons.Paste).OnClick(func(e events.Event) {
		sv.PasteIdx(sv.SelIdx)
	})
	gi.NewButton(m).SetText("Duplicate").SetIcon(icons.Copy).OnClick(func(e events.Event) {
		sv.Duplicate()
	})
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
	if gi.DebugSettings.KeyEventTrace {
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
	if gi.DebugSettings.KeyEventTrace {
		fmt.Printf("SliceViewBase ReadOnly KeyInput: %v\n", sv.Path())
	}
	if sv.Is(SliceViewReadOnlyMultiSel) {
		sv.KeyInputNav(kt)
		if kt.IsHandled() {
			return
		}
	}
	selMode := kt.SelectMode()
	if sv.Is(SliceViewSelectMode) {
		selMode = events.ExtendOne
	}
	kf := keyfun.Of(kt.KeyChord())
	idx := sv.SelIdx
	switch {
	case kf == keyfun.MoveDown:
		ni := idx + 1
		if ni < sv.SliceSize {
			sv.ScrollToIdx(ni)
			sv.UpdateSelectIdx(ni, true, selMode)
			kt.SetHandled()
		}
	case kf == keyfun.MoveUp:
		ni := idx - 1
		if ni >= 0 {
			sv.ScrollToIdx(ni)
			sv.UpdateSelectIdx(ni, true, selMode)
			kt.SetHandled()
		}
	case kf == keyfun.PageDown:
		ni := min(idx+sv.VisRows-1, sv.SliceSize-1)
		sv.ScrollToIdx(ni)
		sv.UpdateSelectIdx(ni, true, selMode)
		kt.SetHandled()
	case kf == keyfun.PageUp:
		ni := max(idx-(sv.VisRows-1), 0)
		sv.ScrollToIdx(ni)
		sv.UpdateSelectIdx(ni, true, selMode)
		kt.SetHandled()
	case kf == keyfun.Enter || kf == keyfun.Accept || kt.KeyRune() == ' ':
		sv.Send(events.DoubleClick, kt)
		kt.SetHandled()
	}
}

func (sv *SliceViewBase) HandleEvents() {
	sv.OnFinal(events.KeyChord, func(e events.Event) {
		if sv.IsReadOnly() {
			if sv.Is(SliceViewReadOnlyKeyNav) {
				sv.KeyInputReadOnly(e)
			}
		} else {
			sv.KeyInputEditable(e)
		}
	})
	sv.On(events.MouseMove, func(e events.Event) {
		row, _, isValid := sv.RowFromEventPos(e)
		if !isValid {
			sv.HoverRow = -1
			sv.Styles.Cursor = sv.NormalCursor
		} else {
			if row != sv.HoverRow {
				sv.HoverRow = row
			}
			sv.Styles.Cursor = cursors.Pointer
		}
		sv.SetNeedsRender(true)
	})
	sv.OnFirst(events.DoubleClick, func(e events.Event) {
		row, _, isValid := sv.RowFromEventPos(e)
		if !isValid {
			return
		}
		if sv.LastClick != row+sv.StartIdx {
			sv.This().(SliceViewer).SliceGrid().Send(events.Click, e)
			e.SetHandled()
		}
	})
	// we must interpret triple click events as double click
	// events for rapid cross-row double clicking to work correctly
	sv.OnFirst(events.TripleClick, func(e events.Event) {
		sv.Send(events.DoubleClick, e)
	})
}

func (sv *SliceViewBase) SizeFinal() {
	sg := sv.This().(SliceViewer).SliceGrid()
	localIter := 0
	for (sv.ConfigIter < 2 || sv.VisRows != sg.VisRows) && localIter < 2 {
		if sv.VisRows != sg.VisRows {
			sv.VisRows = sg.VisRows
			sv.This().(SliceViewer).ConfigRows()
		} else {
			sg.ApplyStyleTree()
		}
		sg.SizeFinalUpdateChildrenSizes()
		sv.ConfigIter++
		localIter++
	}
	sv.Frame.SizeFinal()
}

//////////////////////////////////////////////////////
// 	SliceViewGrid and Layout

// SliceViewGrid handles the resizing logic for SliceView, TableView.
type SliceViewGrid struct {
	gi.Frame // note: must be a frame to support stripes!

	// MinRows is set from parent SV
	MinRows int `set:"-" edit:"-"`

	// height of a single row, computed during layout
	RowHeight float32 `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// total number of rows visible in allocated display size
	VisRows int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// Various computed backgrounds
	BgStripe, BgSelect, BgSelectStripe, BgHover, BgHoverStripe, BgHoverSelect, BgHoverSelectStripe image.Image `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// LastBackground is the background for which modified
	// backgrounds were computed -- don't update if same
	LastBackground image.Image
}

func (sg *SliceViewGrid) OnInit() {
	sg.Frame.OnInit()
	sg.Style(func(s *styles.Style) {
		s.Display = styles.Grid
	})
}

func (sg *SliceViewGrid) SizeFromChildren(iter int, pass gi.LayoutPasses) mat32.Vec2 {
	csz := sg.Frame.SizeFromChildren(iter, pass)
	rht, err := sg.LayImpl.RowHeight(0, 0)
	if err != nil {
		fmt.Println("SliceViewGrid Sizing Error:", err)
		sg.RowHeight = 42
	}
	if sg.NeedsRebuild() { // rebuilding = reset
		sg.RowHeight = rht
	} else {
		sg.RowHeight = max(sg.RowHeight, rht)
	}
	if sg.RowHeight == 0 {
		fmt.Println("SliceViewGrid Sizing Error: RowHeight should not be 0!", sg)
		sg.RowHeight = 42
	}
	allocHt := sg.Geom.Size.Alloc.Content.Y - sg.Geom.Size.InnerSpace.Y
	if allocHt > sg.RowHeight {
		sg.VisRows = int(mat32.Floor(allocHt / sg.RowHeight))
	}
	sg.VisRows = max(sg.VisRows, sg.MinRows)
	minHt := sg.RowHeight * float32(sg.MinRows)
	// visHt := sg.RowHeight * float32(sg.VisRows)
	csz.Y = minHt
	return csz
}

func (sg *SliceViewGrid) SetScrollParams(d mat32.Dims, sb *gi.Slider) {
	if d == mat32.X {
		sg.Frame.SetScrollParams(d, sb)
		return
	}
	sb.Min = 0
	sb.Step = 1
	if sg.VisRows > 0 {
		sb.PageStep = float32(sg.VisRows)
	} else {
		sb.PageStep = 10
	}
	sb.InputThreshold = sb.Step
}

func (sg *SliceViewGrid) SliceView() (SliceViewer, *SliceViewBase) {
	svi := sg.ParentByType(SliceViewBaseType, ki.Embeds)
	if svi == nil {
		return nil, nil
	}
	sv := svi.(SliceViewer)
	return sv, sv.AsSliceViewBase()
}

func (sg *SliceViewGrid) ScrollChanged(d mat32.Dims, sb *gi.Slider) {
	if d == mat32.X {
		sg.Frame.ScrollChanged(d, sb)
		return
	}
	_, sv := sg.SliceView()
	if sv == nil {
		return
	}
	updt := sg.UpdateStart()
	sv.StartIdx = int(mat32.Round(sb.Value))
	sv.This().(SliceViewer).UpdateWidgets()
	sg.UpdateEndRender(updt)
}

func (sg *SliceViewGrid) ScrollValues(d mat32.Dims) (maxSize, visSize, visPct float32) {
	if d == mat32.X {
		return sg.Frame.ScrollValues(d)
	}
	_, sv := sg.SliceView()
	if sv == nil {
		return
	}
	maxSize = float32(max(sv.SliceSize, 1))
	visSize = float32(sg.VisRows)
	visPct = visSize / maxSize
	return
}

func (sg *SliceViewGrid) UpdateScroll(idx int) {
	if !sg.HasScroll[mat32.Y] || sg.Scrolls[mat32.Y] == nil {
		return
	}
	sb := sg.Scrolls[mat32.Y]
	sb.SetValue(float32(idx))
}

func (sg *SliceViewGrid) UpdateBackgrounds() {
	bg := sg.Styles.ActualBackground
	if sg.LastBackground == bg {
		return
	}
	sg.LastBackground = bg

	// we take our zebra intensity applied foreground color and then overlay it onto our background color

	zclr := colors.WithAF32(sg.Styles.Color, gi.SystemSettings.ZebraStripesWeight())
	sg.BgStripe = gradient.Apply(bg, func(c color.Color) color.Color {
		return colors.AlphaBlend(c, zclr)
	})

	hclr := colors.WithAF32(sg.Styles.Color, 0.08)
	sg.BgHover = gradient.Apply(bg, func(c color.Color) color.Color {
		return colors.AlphaBlend(c, hclr)
	})

	zhclr := colors.WithAF32(sg.Styles.Color, gi.SystemSettings.ZebraStripesWeight()+0.08)
	sg.BgHoverStripe = gradient.Apply(bg, func(c color.Color) color.Color {
		return colors.AlphaBlend(c, zhclr)
	})

	sg.BgSelect = colors.C(colors.Scheme.Select.Container)

	sg.BgSelectStripe = colors.C(colors.AlphaBlend(colors.Scheme.Select.Container, zclr))

	sg.BgHoverSelect = colors.C(colors.AlphaBlend(colors.Scheme.Select.Container, hclr))

	sg.BgHoverSelectStripe = colors.C(colors.AlphaBlend(colors.Scheme.Select.Container, zhclr))

}

func (sg *SliceViewGrid) RowBackground(sel, stripe, hover bool) image.Image {
	switch {
	case sel && stripe && hover:
		return sg.BgHoverSelectStripe
	case sel && stripe:
		return sg.BgSelectStripe
	case sel && hover:
		return sg.BgHoverSelect
	case sel:
		return sg.BgSelect
	case stripe && hover:
		return sg.BgHoverStripe
	case stripe:
		return sg.BgStripe
	case hover:
		return sg.BgHover
	default:
		return sg.Styles.ActualBackground
	}
}

func (sg *SliceViewGrid) ChildBackground(child gi.Widget) image.Image {
	bg := sg.Styles.ActualBackground
	svi, sv := sg.SliceView()
	if sv == nil {
		return bg
	}
	sg.UpdateBackgrounds()
	si, _ := svi.WidgetIndex(child)
	row := si - sv.StartIdx
	return sg.RowBackground(sv.IdxIsSelected(si), si%2 == 1, row == sv.HoverRow)
}

func (sg *SliceViewGrid) RenderStripes() {
	pos := sg.Geom.Pos.Content
	sz := sg.Geom.Size.Actual.Content
	if sg.VisRows == 0 || sz.Y == 0 {
		return
	}
	sg.UpdateBackgrounds()

	pc := &sg.Scene.PaintContext
	rows := sg.LayImpl.Shape.Y
	cols := sg.LayImpl.Shape.X
	st := pos
	offset := 0
	_, sv := sg.SliceView()
	startIdx := 0
	if sv != nil {
		startIdx = sv.StartIdx
		offset = startIdx % 2
	}
	for r := 0; r < rows; r++ {
		si := r + startIdx
		ht, _ := sg.LayImpl.RowHeight(r, 0)
		miny := st.Y
		for c := 0; c < cols; c++ {
			kw := sg.Child(r*cols + c).(gi.Widget).AsWidget()
			pyi := mat32.Floor(kw.Geom.Pos.Total.Y)
			if pyi < miny {
				miny = pyi
			}
		}
		st.Y = miny
		ssz := sz
		ssz.Y = ht
		stripe := (r+offset)%2 == 1
		sbg := sg.RowBackground(sv.IdxIsSelected(si), stripe, r == sv.HoverRow)
		pc.BlitBox(st, ssz, sbg)
		st.Y += ht + sg.LayImpl.Gap.Y
	}
}

// IndexFromPixel returns the row, column indexes of given pixel point within grid.
// Takes a scene-level position.
func (sg *SliceViewGrid) IndexFromPixel(pt image.Point) (row, col int) {
	ptf := mat32.V2FromPoint(sg.PointToRelPos(pt))
	sz := sg.Geom.Size.Actual.Content
	if sg.VisRows == 0 || sz.Y == 0 {
		return
	}
	if ptf.Y < 0 || ptf.Y >= sz.Y {
		return -1, 0
	}
	rows := sg.LayImpl.Shape.Y
	cols := sg.LayImpl.Shape.X
	st := mat32.Vec2{}
	got := false
	for r := 0; r < rows; r++ {
		ht, _ := sg.LayImpl.RowHeight(r, 0)
		ht += sg.LayImpl.Gap.Y
		miny := st.Y
		if r > 0 {
			for c := 0; c < cols; c++ {
				kw := sg.Child(r*cols + c).(gi.Widget).AsWidget()
				pyi := mat32.Floor(kw.Geom.Pos.Total.Y)
				if pyi < miny {
					miny = pyi
				}
			}
		}
		st.Y = miny
		ssz := sz
		ssz.Y = ht
		if ptf.Y >= st.Y && ptf.Y < st.Y+ssz.Y {
			row = r
			got = true
			break
			// todo: col
		}
		st.Y += ht
	}
	if !got {
		row = rows - 1
	}
	return
}

func (sg *SliceViewGrid) Render() {
	if sg.PushBounds() {
		sg.RenderStripes()
		sg.RenderChildren()
		sg.RenderScrolls()
		sg.PopBounds()
	}
}
