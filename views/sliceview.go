// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"log/slog"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/reflectx"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
	"cogentcore.org/core/units"
)

// SliceView represents a slice value with index and value widgets.
// Use [SliceViewBase.BindSelect] to make the slice view designed for item selection.
type SliceView struct {
	SliceViewBase

	// StyleFunc is an optional styling function.
	StyleFunc SliceViewStyleFunc `copier:"-" view:"-" json:"-" xml:"-"`
}

// SliceViewStyleFunc is a styling function for custom styling and
// configuration of elements in the slice view.
type SliceViewStyleFunc func(w core.Widget, s *styles.Style, row int)

func (sv *SliceView) HasStyleFunc() bool {
	return sv.StyleFunc != nil
}

func (sv *SliceView) StyleRow(w core.Widget, idx, fidx int) {
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
//   Other externally driven layout changes just update VisRows accordingly.
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
type SliceViewFlags core.WidgetFlags //enums:bitflag -trim-prefix SliceView

const (
	// SliceViewConfigured indicates that the widgets have been configured
	SliceViewConfigured SliceViewFlags = SliceViewFlags(core.WidgetFlagsN) + iota

	// SliceViewIsArray is whether the slice is actually an array -- no modifications -- set by SetSlice
	SliceViewIsArray

	// SliceViewShowIndex is whether to show index or not
	SliceViewShowIndex

	// SliceViewReadOnlyKeyNav is whether support key navigation when ReadOnly (default true).
	// uses a capture of up / down events to manipulate selection, not focus.
	SliceViewReadOnlyKeyNav

	// SliceViewSelectMode is whether to be in select rows mode or editing mode
	SliceViewSelectMode

	// SliceViewReadOnlyMultiSelect: if view is ReadOnly, default selection mode is to choose one row only.
	// If this is true, standard multiple selection logic with modifier keys is instead supported
	SliceViewReadOnlyMultiSelect

	// SliceViewInFocusGrab is a guard for recursive focus grabbing
	SliceViewInFocusGrab

	// SliceViewInFullRebuild is a guard for recursive rebuild
	SliceViewInFullRebuild
)

const (
	SliceViewRowProperty = "sv-row"
	SliceViewColProperty = "sv-col"
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

	// UpdateSliceSize updates the current size of the slice
	// and sets SliceSize if changed.
	UpdateSliceSize() int

	// StyleValueWidget performs additional value widget styling
	StyleValueWidget(w core.Widget, s *styles.Style, row, col int)

	// ConfigRows configures VisRows worth of widgets
	// to display slice data.
	ConfigRows()

	// UpdateWidgets updates the row widget display to
	// represent the current state of the slice data,
	// including which range of data is being displayed.
	// This is called for scrolling, navigation etc.
	UpdateWidgets()

	// HasStyleFunc returns whether there is a custom style function.
	HasStyleFunc() bool

	// StyleRow calls a custom style function on given row (and field)
	StyleRow(w core.Widget, idx, fidx int)

	// RowFirstWidget returns the first widget for given row
	// (could be index or not) -- false if out of range
	RowFirstWidget(row int) (*core.WidgetBase, bool)

	// RowGrabFocus grabs the focus for the first focusable
	// widget in given row.
	// returns that element or nil if not successful
	// note: grid must have already rendered for focus to be grabbed!
	RowGrabFocus(row int) *core.WidgetBase

	// SliceNewAt inserts a new blank element at given
	// index in the slice. -1 means the end.
	SliceNewAt(idx int)

	// SliceDeleteAt deletes element at given index from slice
	SliceDeleteAt(idx int)

	// MimeDataType returns the data type for mime clipboard
	// (copy / paste) data e.g., fileinfo.DataJson
	MimeDataType() string

	// CopySelectToMime copies selected rows to mime data
	CopySelectToMime() mimedata.Mimes

	// PasteAssign assigns mime data (only the first one!) to this idx
	PasteAssign(md mimedata.Mimes, idx int)

	// PasteAtIndex inserts object(s) from mime data at
	// (before) given slice index
	PasteAtIndex(md mimedata.Mimes, idx int)

	MakePasteMenu(m *core.Scene, md mimedata.Mimes, idx int, mod events.DropMods, fun func())
	DragStart(e events.Event)
	DragDrop(e events.Event)
	DropFinalize(de *events.DragDrop)
	DropDeleteSource(e events.Event)
}

// SliceViewBase is the base for [SliceView] and [TableView] and any other viewers
// of array-like data. It automatically computes the number of rows that fit
// within its allocated space, and manages the offset view window into the full
// list of items, and supports row selection, copy / paste, Drag-n-Drop, etc.
// Use [SliceViewBase.BindSelect] to make the slice view designed for item selection.
type SliceViewBase struct {
	core.Frame

	// Slice is the pointer to the slice that we are viewing.
	Slice any `set:"-" json:"-" xml:"-"`

	// MinRows specifies the minimum number of rows to display, to ensure
	// at least this amount is displayed.
	MinRows int `default:"4"`

	// ViewPath is a record of parent view names that have led up to this view.
	// It is displayed as extra contextual information in view dialogs.
	ViewPath string

	// ViewMu is an optional mutex that, if non-nil, will be used around any updates that
	// read / modify the underlying Slice data.
	// Can be used to protect against random updating if your code has specific
	// update points that can be likewise protected with this same mutex.
	ViewMu *sync.Mutex `copier:"-" view:"-" json:"-" xml:"-" set:"-"`

	// SelectedValue is the current selection value; initially select this value if set.
	SelectedValue any `copier:"-" view:"-" json:"-" xml:"-"`

	// index of currently selected item
	SelectedIndex int `copier:"-" json:"-" xml:"-"`

	// index of row to select at start
	InitSelectedIndex int `copier:"-" json:"-" xml:"-"`

	// list of currently selected slice indexes
	SelectedIndexes map[int]struct{} `set:"-" copier:"-"`

	// LastClick is the last row that has been clicked on.
	// This is used to prevent erroneous double click events
	// from being sent when the user clicks on multiple different
	// rows in quick succession.
	LastClick int `set:"-" copier:"-" json:"-" xml:"-"`

	// NormalCursor is the cached cursor to display when there
	// is no row being hovered.
	NormalCursor cursors.Cursor `copier:"-" xml:"-" json:"-" set:"-"`

	// CurrentCursor is the cached cursor that should currently be
	// displayed.
	CurrentCursor cursors.Cursor `copier:"-" xml:"-" json:"-" set:"-"`

	// non-ptr reflect.Value of the slice
	SliceNPVal reflect.Value `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// SliceValue is the [Value] associated with this slice view, if any.
	SliceValue Value `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// Value representations of the slice values
	Values []Value `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// currently hovered row
	HoverRow int `set:"-" view:"-" copier:"-" json:"-" xml:"-"`

	// list of currently dragged indexes
	DraggedIndexes []int `set:"-" view:"-" copier:"-" json:"-" xml:"-"`

	// total number of rows visible in allocated display size
	VisRows int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// starting slice index of visible rows
	StartIndex int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// size of slice
	SliceSize int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// iteration through the configuration process, reset when a new slice type is set
	ConfigIter int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// temp idx state for e.g., dnd
	TmpIndex int `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

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
	sv.InitSelectedIndex = -1
	sv.HoverRow = -1
	sv.MinRows = 4
	sv.SetFlag(false, SliceViewSelectMode)
	sv.SetFlag(true, SliceViewShowIndex)
	sv.SetFlag(true, SliceViewReadOnlyKeyNav)
	svi := sv.This().(SliceViewer)

	sv.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable, abilities.DoubleClickable, abilities.TripleClickable)
		s.SetAbilities(!sv.IsReadOnly(), abilities.Draggable, abilities.Droppable)
		s.Cursor = sv.CurrentCursor
		s.Direction = styles.Column
		// absorb horizontal here, vertical in view
		s.Overflow.X = styles.OverflowAuto
		s.Grow.Set(1, 1)
	})
	if !sv.IsReadOnly() {
		sv.On(events.DragStart, func(e events.Event) {
			svi.DragStart(e)
		})
		sv.On(events.DragEnter, func(e events.Event) {
			e.SetHandled()
		})
		sv.On(events.DragLeave, func(e events.Event) {
			e.SetHandled()
		})
		sv.On(events.Drop, func(e events.Event) {
			svi.DragDrop(e)
		})
		sv.On(events.DropDeleteSource, func(e events.Event) {
			svi.DropDeleteSource(e)
		})
	}
	sv.StyleFinal(func(s *styles.Style) {
		sv.NormalCursor = s.Cursor
	})
	sv.OnWidgetAdded(func(w core.Widget) {
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
				// baseline mins:
				s.Min.X.Ch(20)
				s.Min.Y.Em(6)
			})
			oc := func(e events.Event) {
				sv.SetFocusEvent()
				row, _, isValid := sg.IndexFromPixel(e.Pos())
				if isValid {
					sv.UpdateSelectRow(row, e.SelectMode())
					sv.LastClick = row + sv.StartIndex
				}
			}
			sg.OnClick(oc)
			sg.On(events.ContextMenu, func(e events.Event) {
				// we must select the row on right click so that the context menu
				// corresponds to the right row
				oc(e)
				sv.HandleEvent(e)
			})
		}
		if w.Parent().PathFrom(sv) == "grid" {
			switch {
			case strings.HasPrefix(w.Name(), "index-"):
				wb := w.AsWidget()
				w.Style(func(s *styles.Style) {
					s.SetAbilities(true, abilities.DoubleClickable)
					s.SetAbilities(!sv.IsReadOnly(), abilities.Draggable, abilities.Droppable)
					s.Cursor = cursors.None
					nd := math32.Log10(float32(sv.SliceSize))
					nd = max(nd, 3)
					s.Min.X.Ch(nd + 2)
					s.Padding.Right.Dp(4)
					s.Text.Align = styles.End
					s.Min.Y.Em(1)
					s.GrowWrap = false
				})
				wb.OnDoubleClick(sv.HandleEvent)
				wb.On(events.ContextMenu, sv.HandleEvent)
				if !sv.IsReadOnly() {
					w.On(events.DragStart, func(e events.Event) {
						svi.DragStart(e)
					})
					w.On(events.DragEnter, func(e events.Event) {
						e.SetHandled()
					})
					w.On(events.DragLeave, func(e events.Event) {
						e.SetHandled()
					})
					w.On(events.Drop, func(e events.Event) {
						svi.DragDrop(e)
					})
					w.On(events.DropDeleteSource, func(e events.Event) {
						svi.DropDeleteSource(e)
					})
				}
			case strings.HasPrefix(w.Name(), "value-"):
				wb := w.AsWidget()
				w.Style(func(s *styles.Style) {
					if sv.IsReadOnly() {
						s.SetAbilities(true, abilities.DoubleClickable)
						s.SetAbilities(false, abilities.Hoverable, abilities.Focusable, abilities.Activatable, abilities.TripleClickable)
						s.SetReadOnly(true)
					}
					row, col := sv.WidgetIndex(w)
					row += sv.StartIndex
					sv.This().(SliceViewer).StyleValueWidget(w, s, row, col)
					if row < sv.SliceSize {
						sv.This().(SliceViewer).StyleRow(w, row, col)
					}
				})
				wb.OnSelect(func(e events.Event) {
					e.SetHandled()
					row, _ := sv.WidgetIndex(w)
					sv.UpdateSelectRow(row, e.SelectMode())
					sv.LastClick = row + sv.StartIndex
				})
				wb.OnDoubleClick(sv.HandleEvent)
				wb.On(events.ContextMenu, sv.HandleEvent)
			}
		}
	})
}

// StyleValueWidget performs additional value widget styling
func (sv *SliceViewBase) StyleValueWidget(w core.Widget, s *styles.Style, row, col int) {
	if sv.MaxWidth > 0 {
		hv := units.Ch(float32(sv.MaxWidth))
		s.Min.X.Value = max(s.Min.X.Value, hv.Convert(s.Min.X.Unit, &s.UnitContext).Value)
	}
	s.SetTextWrap(false)
}

func (sv *SliceViewBase) AsSliceViewBase() *SliceViewBase {
	return sv
}

func (sv *SliceViewBase) SetSliceBase() {
	sv.SetFlag(false, SliceViewConfigured, SliceViewSelectMode)
	sv.ConfigIter = 0
	sv.StartIndex = 0
	sv.VisRows = sv.MinRows
	if !sv.IsReadOnly() {
		sv.SelectedIndex = -1
	}
	sv.ResetSelectedIndexes()
}

// SetSlice sets the source slice that we are viewing.
// This ReConfigs the view for this slice if different.
// Note: it is important to at least set an empty slice of
// the desired type at the start to enable initial configuration.
func (sv *SliceViewBase) SetSlice(sl any) *SliceViewBase {
	if reflectx.AnyIsNil(sl) {
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

	sv.SetSliceBase()
	sv.Slice = sl
	sv.SliceNPVal = reflectx.NonPointerValue(reflect.ValueOf(sv.Slice))
	isArray := reflectx.NonPointerType(reflect.TypeOf(sl)).Kind() == reflect.Array
	sv.SetFlag(isArray, SliceViewIsArray)
	// make sure elements aren't nil to prevent later panics
	for i := 0; i < sv.SliceNPVal.Len(); i++ {
		val := sv.SliceNPVal.Index(i)
		k := val.Kind()
		if (k == reflect.Chan || k == reflect.Func || k == reflect.Interface || k == reflect.Map || k == reflect.Pointer || k == reflect.Slice) && val.IsNil() {
			val.Set(reflect.New(reflectx.NonPointerType(val.Type())))
		}
	}
	sv.ElVal = reflectx.SliceElementValue(sl)
	sv.Update()
	return sv
}

// IsNil returns true if the Slice is nil
func (sv *SliceViewBase) IsNil() bool {
	return reflectx.AnyIsNil(sv.Slice)
}

// RowFromEventPos returns the widget row, slice index, and
// whether the index is in slice range, for given event position.
func (sv *SliceViewBase) RowFromEventPos(e events.Event) (row, idx int, isValid bool) {
	sg := sv.This().(SliceViewer).SliceGrid()
	row, _, isValid = sg.IndexFromPixel(e.Pos())
	if !isValid {
		return
	}
	idx = row + sv.StartIndex
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
		*val = sv.SelectedIndex
	})
	sv.OnDoubleClick(func(e events.Event) {
		if sv.ClickSelectEvent(e) {
			*val = sv.SelectedIndex
			sv.Scene.SendKey(keymap.Accept, e) // activates Ok button code
		}
	})
	return sv
}

// Config configures a standard setup of the overall Frame
func (sv *SliceViewBase) Config() {
	sv.ConfigSliceView()
}

// ConfigSliceView handles entire config.
// ReConfig calls this, followed by ApplyStyleTree so we don't need to call that.
func (sv *SliceViewBase) ConfigSliceView() {
	if sv.Is(SliceViewConfigured) {
		sv.This().(SliceViewer).UpdateWidgets()
		return
	}
	sv.ConfigFrame()
	sv.This().(SliceViewer).ConfigRows()
	sv.This().(SliceViewer).UpdateWidgets()
	sv.ApplyStyleTree()
	sv.NeedsLayout()
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

// UpdateSliceSize updates and returns the size of the slice
// and sets SliceSize
func (sv *SliceViewBase) UpdateSliceSize() int {
	sz := sv.SliceNPVal.Len()
	sv.SliceSize = sz
	return sz
}

// WidgetIndex returns the row and column indexes for given widget,
// from the properties set during construction.
func (sv *SliceViewBase) WidgetIndex(w core.Widget) (row, col int) {
	if rwi := w.Property(SliceViewRowProperty); rwi != nil {
		row = rwi.(int)
	}
	if cli := w.Property(SliceViewColProperty); cli != nil {
		col = cli.(int)
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

// UpdateStartIndex updates StartIndex to fit current view
func (sv *SliceViewBase) UpdateStartIndex() {
	sz := sv.This().(SliceViewer).UpdateSliceSize()
	if sz > sv.VisRows {
		lastSt := sz - sv.VisRows
		sv.StartIndex = min(lastSt, sv.StartIndex)
		sv.StartIndex = max(0, sv.StartIndex)
	} else {
		sv.StartIndex = 0
	}
}

// UpdateScroll updates the scroll value
func (sv *SliceViewBase) UpdateScroll() {
	sg := sv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}
	sg.UpdateScroll(sv.StartIndex)
}

// ConfigRows configures VisRows worth of widgets
// to display slice data.
func (sv *SliceViewBase) ConfigRows() {
	sg := sv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}
	sv.SetFlag(true, SliceViewConfigured)
	sg.SetFlag(true, core.LayoutNoKeys)

	sv.ViewMuLock()
	defer sv.ViewMuUnlock()

	sg.DeleteChildren()
	sv.Values = nil

	sv.This().(SliceViewer).UpdateSliceSize()

	if sv.IsNil() {
		return
	}

	nWidgPerRow, idxOff := sv.RowWidgetNs()
	nWidg := nWidgPerRow * sv.VisRows
	sg.Styles.Columns = nWidgPerRow

	sv.Values = make([]Value, sv.VisRows)
	sg.Kids = make(tree.Slice, nWidg)

	for i := 0; i < sv.VisRows; i++ {
		si := i
		ridx := i * nWidgPerRow
		var val reflect.Value
		if si < sv.SliceSize {
			val = reflectx.OnePointerUnderlyingValue(sv.SliceNPVal.Index(si)) // deal with pointer lists
		} else {
			val = sv.ElVal
		}
		vv := ToValue(val.Interface(), "")
		sv.Values[i] = vv
		vv.SetSliceValue(val, sv.Slice, si, sv.ViewPath)
		vv.SetReadOnly(sv.IsReadOnly())

		vtyp := vv.WidgetType()
		itxt := strconv.Itoa(i)
		sitxt := strconv.Itoa(si)
		labnm := "index-" + itxt
		valnm := "value-" + itxt

		if sv.Is(SliceViewShowIndex) {
			idxlab := &core.Text{}
			sg.SetChild(idxlab, ridx, labnm)
			idxlab.SetText(sitxt)
			idxlab.OnSelect(func(e events.Event) {
				e.SetHandled()
				sv.UpdateSelectRow(i, e.SelectMode())
				sv.LastClick = i + sv.StartIndex
			})
			idxlab.SetProperty(SliceViewRowProperty, i)
		}

		w := tree.NewOfType(vtyp).(core.Widget)
		sg.SetChild(w, ridx+idxOff, valnm)
		Config(vv, w)
		w.SetProperty(SliceViewRowProperty, i)

		if !sv.IsReadOnly() {
			vv.OnChange(func(e events.Event) {
				sv.SendChange(e)
			})
			vv.AsWidgetBase().OnInput(sv.HandleEvent)
		}
		if i == 0 {
			sv.MaxWidth = 0
			_, isString := vv.(*StringValue)
			npv := reflectx.NonPointerValue(val)
			if isString && sv.SliceSize > 0 && npv.Kind() == reflect.String {
				mxw := 0
				for rw := 0; rw < sv.SliceSize; rw++ {
					val := reflectx.OnePointerUnderlyingValue(sv.SliceNPVal.Index(rw)).Elem()
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

	sv.ViewMuLock()
	defer sv.ViewMuUnlock()

	sv.This().(SliceViewer).UpdateSliceSize()

	nWidgPerRow, idxOff := sv.RowWidgetNs()

	scrollTo := -1
	if sv.SelectedValue != nil {
		idx, ok := SliceIndexByValue(sv.Slice, sv.SelectedValue)
		if ok {
			sv.SelectedIndex = idx
			scrollTo = sv.SelectedIndex
		}
		sv.SelectedValue = nil
		sv.InitSelectedIndex = -1
	} else if sv.InitSelectedIndex >= 0 {
		sv.SelectedIndex = sv.InitSelectedIndex
		sv.InitSelectedIndex = -1
		scrollTo = sv.SelectedIndex
	}
	if scrollTo >= 0 {
		sv.ScrollToIndex(scrollTo)
	}
	sv.UpdateStartIndex()

	for i := 0; i < sv.VisRows; i++ {
		ridx := i * nWidgPerRow
		w := sg.Kids[ridx+idxOff].(core.Widget)
		vv := sv.Values[i]
		si := sv.StartIndex + i // slice idx
		invis := si >= sv.SliceSize

		var idxlab *core.Text
		if sv.Is(SliceViewShowIndex) {
			idxlab = sg.Kids[ridx].(*core.Text)
			idxlab.SetText(strconv.Itoa(si)).Config()
			idxlab.SetState(invis, states.Invisible)
		}
		w.SetState(invis, states.Invisible)
		if si < sv.SliceSize {
			val := reflectx.OnePointerUnderlyingValue(sv.SliceNPVal.Index(si)) // deal with pointer lists
			vv.SetSliceValue(val, sv.Slice, si, sv.ViewPath)
			vv.SetReadOnly(sv.IsReadOnly())
			vv.Update()

			if sv.IsReadOnly() {
				w.AsWidget().SetReadOnly(true)
			}
		} else {
			vv.SetSliceValue(sv.ElVal, sv.Slice, 0, sv.ViewPath)
			vv.Update()
			w.AsWidget().SetSelected(false)
			if sv.Is(SliceViewShowIndex) {
				idxlab.SetSelected(false)
			}
		}
		if sv.This().(SliceViewer).HasStyleFunc() {
			w.ApplyStyle()
		}
	}
	sg.NeedsRender()
}

// SliceNewAtRow inserts a new blank element at given display row
func (sv *SliceViewBase) SliceNewAtRow(row int) {
	sv.This().(SliceViewer).SliceNewAt(sv.StartIndex + row)
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (sv *SliceViewBase) SliceNewAt(idx int) {
	if sv.Is(SliceViewIsArray) {
		return
	}

	sv.ViewMuLock() // no return!  must unlock before return below

	sv.SliceNewAtSelect(idx)

	sltyp := reflectx.SliceElementType(sv.Slice) // has pointer if it is there
	isNode := tree.IsNode(sltyp)
	slptr := sltyp.Kind() == reflect.Pointer

	svl := reflect.ValueOf(sv.Slice)
	sz := sv.SliceSize

	svnp := sv.SliceNPVal

	if isNode && sv.SliceValue != nil {
		vd := sv.SliceValue.AsValueData()
		if vd.Owner != nil {
			if owntree, ok := vd.Owner.(tree.Node); ok {
				d := core.NewBody().AddTitle("Add list items").AddText("Number and type of items to insert:")
				nd := &core.NewItemsData{}
				w := NewValue(d, nd).AsWidget()
				tree.ChildByType[*core.Chooser](w, tree.Embeds).SetTypes(types.AllEmbeddersOf(owntree.BaseType())...).SetCurrentIndex(0)
				d.AddBottomBar(func(parent core.Widget) {
					d.AddCancel(parent)
					d.AddOK(parent).OnClick(func(e events.Event) {
						for i := 0; i < nd.Number; i++ {
							nm := fmt.Sprintf("New%v%v", nd.Type.Name, idx+1+i)
							owntree.InsertNewChild(nd.Type, idx+1+i, nm)
						}
						sv.SendChange()
					})
				})
				d.RunDialog(sv)
			}
		}
	} else {
		nval := reflect.New(reflectx.NonPointerType(sltyp)) // make the concrete el
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

	sv.SliceNPVal = reflectx.NonPointerValue(reflect.ValueOf(sv.Slice)) // need to update after changes

	sv.This().(SliceViewer).UpdateSliceSize()

	sv.SelectIndexAction(idx, events.SelectOne)
	sv.ViewMuUnlock()
	sv.SendChange()
	sv.This().(SliceViewer).UpdateWidgets()
	sv.IndexGrabFocus(idx)
	sv.NeedsLayout()
}

// SliceDeleteAtRow deletes element at given display row
// if updt is true, then update the grid after
func (sv *SliceViewBase) SliceDeleteAtRow(row int) {
	sv.This().(SliceViewer).SliceDeleteAt(sv.StartIndex + row)
}

// SliceNewAtSelect updates selected rows based on
// inserting new element at given index.
// must be called with successful SliceNewAt
func (sv *SliceViewBase) SliceNewAtSelect(i int) {
	sl := sv.SelectedIndexesList(false) // ascending
	sv.ResetSelectedIndexes()
	for _, ix := range sl {
		if ix >= i {
			ix++
		}
		sv.SelectedIndexes[ix] = struct{}{}
	}
}

// SliceDeleteAtSelect updates selected rows based on
// deleting element at given index
// must be called with successful SliceDeleteAt
func (sv *SliceViewBase) SliceDeleteAtSelect(i int) {
	sl := sv.SelectedIndexesList(true) // desscending
	sv.ResetSelectedIndexes()
	for _, ix := range sl {
		switch {
		case ix == i:
			continue
		case ix > i:
			ix--
		}
		sv.SelectedIndexes[ix] = struct{}{}
	}
}

// SliceDeleteAt deletes element at given index from slice
func (sv *SliceViewBase) SliceDeleteAt(i int) {
	if sv.Is(SliceViewIsArray) {
		return
	}
	if i < 0 || i >= sv.SliceSize {
		return
	}
	sv.ViewMuLock()

	sv.SliceDeleteAtSelect(i)

	reflectx.SliceDeleteAt(sv.Slice, i)

	sv.This().(SliceViewer).UpdateSliceSize()

	sv.ViewMuUnlock()
	sv.SendChange()
	sv.This().(SliceViewer).UpdateWidgets()
	sv.NeedsRender()
}

// ConfigToolbar configures a [core.Toolbar] for this view
func (sv *SliceViewBase) ConfigToolbar(tb *core.Toolbar) {
	if reflectx.AnyIsNil(sv.Slice) {
		return
	}
	if sv.Is(SliceViewIsArray) || sv.IsReadOnly() {
		return
	}
	core.NewButton(tb, "slice-add").SetText("Add").SetIcon(icons.Add).SetTooltip("add a new element to the slice").
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
		fmt.Printf("views.SliceViewBase: slice index out of range: %v\n", idx)
		return nil
	}
	val := reflectx.OnePointerUnderlyingValue(sv.SliceNPVal.Index(idx)) // deal with pointer lists
	vali := val.Interface()
	return vali
}

// IsRowInBounds returns true if disp row is in bounds
func (sv *SliceViewBase) IsRowInBounds(row int) bool {
	return row >= 0 && row < sv.VisRows
}

// IsIndexVisible returns true if slice index is currently visible
func (sv *SliceViewBase) IsIndexVisible(idx int) bool {
	return sv.IsRowInBounds(idx - sv.StartIndex)
}

// RowFirstWidget returns the first widget for given row (could be index or
// not) -- false if out of range
func (sv *SliceViewBase) RowFirstWidget(row int) (*core.WidgetBase, bool) {
	if !sv.Is(SliceViewShowIndex) {
		return nil, false
	}
	if !sv.IsRowInBounds(row) {
		return nil, false
	}
	nWidgPerRow, _ := sv.This().(SliceViewer).RowWidgetNs()
	sg := sv.This().(SliceViewer).SliceGrid()
	w := sg.Kids[row*nWidgPerRow].(core.Widget).AsWidget()
	return w, true
}

// RowGrabFocus grabs the focus for the first focusable widget
// in given row.  returns that element or nil if not successful
// note: grid must have already rendered for focus to be grabbed!
func (sv *SliceViewBase) RowGrabFocus(row int) *core.WidgetBase {
	if !sv.IsRowInBounds(row) || sv.Is(SliceViewInFocusGrab) { // range check
		return nil
	}
	nWidgPerRow, idxOff := sv.This().(SliceViewer).RowWidgetNs()
	ridx := nWidgPerRow * row
	sg := sv.This().(SliceViewer).SliceGrid()
	w := sg.Child(ridx + idxOff).(core.Widget).AsWidget()
	if w.StateIs(states.Focused) {
		return w
	}
	sv.SetFlag(true, SliceViewInFocusGrab)
	w.SetFocusEvent()
	sv.SetFlag(false, SliceViewInFocusGrab)
	return w
}

// IndexGrabFocus grabs the focus for the first focusable widget
// in given idx.  returns that element or nil if not successful.
func (sv *SliceViewBase) IndexGrabFocus(idx int) *core.WidgetBase {
	sv.ScrollToIndex(idx)
	return sv.This().(SliceViewer).RowGrabFocus(idx - sv.StartIndex)
}

// IndexPos returns center of window position of index label for idx (ContextMenuPos)
func (sv *SliceViewBase) IndexPos(idx int) image.Point {
	row := idx - sv.StartIndex
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

// IndexFromPos returns the idx that contains given vertical position, false if not found
func (sv *SliceViewBase) IndexFromPos(posY int) (int, bool) {
	row, ok := sv.RowFromPos(posY)
	if !ok {
		return -1, false
	}
	return row + sv.StartIndex, true
}

// ScrollToIndexNoUpdate ensures that given slice idx is visible
// by scrolling display as needed.
// This version does not update the slicegrid.
// Just computes the StartIndex and updates the scrollbar
func (sv *SliceViewBase) ScrollToIndexNoUpdate(idx int) bool {
	if sv.VisRows == 0 {
		return false
	}
	if idx < sv.StartIndex {
		sv.StartIndex = idx
		sv.StartIndex = max(0, sv.StartIndex)
		sv.UpdateScroll()
		return true
	}
	if idx >= sv.StartIndex+sv.VisRows {
		sv.StartIndex = idx - (sv.VisRows - 4)
		sv.StartIndex = max(0, sv.StartIndex)
		sv.UpdateScroll()
		return true
	}
	return false
}

// ScrollToIndex ensures that given slice idx is visible
// by scrolling display as needed.
func (sv *SliceViewBase) ScrollToIndex(idx int) bool {
	updt := sv.ScrollToIndexNoUpdate(idx)
	if updt {
		sv.This().(SliceViewer).UpdateWidgets()
	}
	return updt
}

// SelectValue sets SelVal and attempts to find corresponding row, setting
// SelectedIndex and selecting row if found -- returns true if found, false
// otherwise.
func (sv *SliceViewBase) SelectValue(val string) bool {
	sv.SelectedValue = val
	if sv.SelectedValue != nil {
		sv.ViewMuLock()
		idx, _ := SliceIndexByValue(sv.Slice, sv.SelectedValue)
		sv.ViewMuUnlock()
		if idx >= 0 {
			sv.UpdateSelectIndex(idx, true, events.SelectOne)
			sv.ScrollToIndex(idx)
			return true
		}
	}
	return false
}

// SliceIndexByValue searches for first index that contains given value in slice
// -- returns false if not found
func SliceIndexByValue(slc any, fldVal any) (int, bool) {
	svnp := reflectx.NonPointerValue(reflect.ValueOf(slc))
	sz := svnp.Len()
	for idx := 0; idx < sz; idx++ {
		rval := reflectx.NonPointerValue(svnp.Index(idx))
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
	if sv.SelectedIndex >= sv.SliceSize-1 {
		sv.SelectedIndex = sv.SliceSize - 1
		return -1
	}
	sv.SelectedIndex++
	sv.SelectIndexAction(sv.SelectedIndex, selMode)
	return sv.SelectedIndex
}

// MoveDownAction moves the selection down to next row, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected
// row
func (sv *SliceViewBase) MoveDownAction(selMode events.SelectModes) int {
	nidx := sv.MoveDown(selMode)
	if nidx >= 0 {
		sv.ScrollToIndex(nidx)
		sv.Send(events.Select) // todo: need to do this for the item?
	}
	return nidx
}

// MoveUp moves the selection up to previous idx, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (sv *SliceViewBase) MoveUp(selMode events.SelectModes) int {
	if sv.SelectedIndex <= 0 {
		sv.SelectedIndex = 0
		return -1
	}
	sv.SelectedIndex--
	sv.SelectIndexAction(sv.SelectedIndex, selMode)
	return sv.SelectedIndex
}

// MoveUpAction moves the selection up to previous idx, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (sv *SliceViewBase) MoveUpAction(selMode events.SelectModes) int {
	nidx := sv.MoveUp(selMode)
	if nidx >= 0 {
		sv.ScrollToIndex(nidx)
		sv.Send(events.Select)
	}
	return nidx
}

// MovePageDown moves the selection down to next page, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (sv *SliceViewBase) MovePageDown(selMode events.SelectModes) int {
	if sv.SelectedIndex >= sv.SliceSize-1 {
		sv.SelectedIndex = sv.SliceSize - 1
		return -1
	}
	sv.SelectedIndex += sv.VisRows
	sv.SelectedIndex = min(sv.SelectedIndex, sv.SliceSize-1)
	sv.SelectIndexAction(sv.SelectedIndex, selMode)
	return sv.SelectedIndex
}

// MovePageDownAction moves the selection down to next page, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (sv *SliceViewBase) MovePageDownAction(selMode events.SelectModes) int {
	nidx := sv.MovePageDown(selMode)
	if nidx >= 0 {
		sv.ScrollToIndex(nidx)
		sv.Send(events.Select)
	}
	return nidx
}

// MovePageUp moves the selection up to previous page, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (sv *SliceViewBase) MovePageUp(selMode events.SelectModes) int {
	if sv.SelectedIndex <= 0 {
		sv.SelectedIndex = 0
		return -1
	}
	sv.SelectedIndex -= sv.VisRows
	sv.SelectedIndex = max(0, sv.SelectedIndex)
	sv.SelectIndexAction(sv.SelectedIndex, selMode)
	return sv.SelectedIndex
}

// MovePageUpAction moves the selection up to previous page, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (sv *SliceViewBase) MovePageUpAction(selMode events.SelectModes) int {
	nidx := sv.MovePageUp(selMode)
	if nidx >= 0 {
		sv.ScrollToIndex(nidx)
		sv.Send(events.Select)
	}
	return nidx
}

//////////////////////////////////////////////////////////
//    Selection: user operates on the index labels

// RowWidgetsFunc calls function on each widget in given row
// (row, not index), with an UpdateStart / EndRender wrapper
func (sv *SliceViewBase) RowWidgetsFunc(row int, fun func(w core.Widget)) {
	if row < 0 {
		return
	}

	sg := sv.This().(SliceViewer).SliceGrid()
	nWidgPerRow, _ := sv.This().(SliceViewer).RowWidgetNs()
	rowidx := row * nWidgPerRow
	for col := 0; col < nWidgPerRow; col++ {
		kidx := rowidx + col
		if sg.Kids.IsValidIndex(kidx) == nil {
			w := sg.Child(rowidx).(core.Widget)
			fun(w)
		}
	}
	sv.NeedsRender()
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
	sv.RowWidgetsFunc(row, func(w core.Widget) {
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
	sv.RowWidgetsFunc(row, func(w core.Widget) {
		w.AsWidget().SetSelected(sel)
	})
}

// SelectIndexWidgets sets the selection state of given slice index
// returns false if index is not visible
func (sv *SliceViewBase) SelectIndexWidgets(idx int, sel bool) bool {
	if !sv.IsIndexVisible(idx) {
		return false
	}
	sv.SelectRowWidgets(idx-sv.StartIndex, sel)
	return true
}

// UpdateSelectRow updates the selection for the given row
func (sv *SliceViewBase) UpdateSelectRow(row int, selMode events.SelectModes) {
	idx := row + sv.StartIndex
	if row < 0 || idx >= sv.SliceSize {
		return
	}
	sel := !sv.IndexIsSelected(idx)
	sv.UpdateSelectIndex(idx, sel, selMode)
}

// UpdateSelectIndex updates the selection for the given index
func (sv *SliceViewBase) UpdateSelectIndex(idx int, sel bool, selMode events.SelectModes) {
	if sv.IsReadOnly() && !sv.Is(SliceViewReadOnlyMultiSelect) {
		sv.UnselectAllIndexes()
		if sel || sv.SelectedIndex == idx {
			sv.SelectedIndex = idx
			sv.SelectIndex(idx)
		}
		sv.ApplyStyleTree()
		sv.This().(SliceViewer).UpdateWidgets()
		sv.Send(events.Select)
		sv.NeedsRender()
	} else {
		sv.SelectIndexAction(idx, selMode)
	}
}

// IndexIsSelected returns the selected status of given slice index
func (sv *SliceViewBase) IndexIsSelected(idx int) bool {
	sv.ViewMuLock()
	defer sv.ViewMuUnlock()
	if sv.IsReadOnly() {
		return idx == sv.SelectedIndex
	}
	_, ok := sv.SelectedIndexes[idx]
	return ok
}

func (sv *SliceViewBase) ResetSelectedIndexes() {
	sv.SelectedIndexes = make(map[int]struct{})
}

// SelectedIndexesList returns list of selected indexes,
// sorted either ascending or descending
func (sv *SliceViewBase) SelectedIndexesList(descendingSort bool) []int {
	rws := make([]int, len(sv.SelectedIndexes))
	i := 0
	for r := range sv.SelectedIndexes {
		if r >= sv.SliceSize { // double safety check at this point
			delete(sv.SelectedIndexes, r)
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

// SelectIndex selects given idx (if not already selected) -- updates select
// status of index label
func (sv *SliceViewBase) SelectIndex(idx int) {
	sv.SelectedIndexes[idx] = struct{}{}
	// sv.SelectIndexWidgets(idx, true)
}

// UnselectIndex unselects given idx (if selected)
func (sv *SliceViewBase) UnselectIndex(idx int) {
	if sv.IndexIsSelected(idx) {
		delete(sv.SelectedIndexes, idx)
	}
	// sv.SelectIndexWidgets(idx, false)
}

// UnselectAllIndexes unselects all selected idxs
func (sv *SliceViewBase) UnselectAllIndexes() {
	// for r := range sv.SelectedIndexes {
	// 	sv.SelectIndexWidgets(r, false)
	// }
	sv.ResetSelectedIndexes()
}

// SelectAllIndexes selects all idxs
func (sv *SliceViewBase) SelectAllIndexes() {
	sv.UnselectAllIndexes()
	sv.SelectedIndexes = make(map[int]struct{}, sv.SliceSize)
	for idx := 0; idx < sv.SliceSize; idx++ {
		sv.SelectedIndexes[idx] = struct{}{}
		// sv.SelectIndexWidgets(idx, true)
	}
	sv.NeedsRender()
}

// SelectIndexAction is called when a select action has been received (e.g., a
// mouse click) -- translates into selection updates -- gets selection mode
// from mouse event (ExtendContinuous, ExtendOne)
func (sv *SliceViewBase) SelectIndexAction(idx int, mode events.SelectModes) {
	if mode == events.NoSelect {
		return
	}
	idx = min(idx, sv.SliceSize-1)
	if idx < 0 {
		sv.ResetSelectedIndexes()
		return
	}
	// row := idx - sv.StartIndex // note: could be out of bounds

	switch mode {
	case events.SelectOne:
		if sv.IndexIsSelected(idx) {
			if len(sv.SelectedIndexes) > 1 {
				sv.UnselectAllIndexes()
			}
			sv.SelectedIndex = idx
			sv.SelectIndex(idx)
			sv.IndexGrabFocus(idx)
		} else {
			sv.UnselectAllIndexes()
			sv.SelectedIndex = idx
			sv.SelectIndex(idx)
			sv.IndexGrabFocus(idx)
		}
		sv.Send(events.Select) //  sv.SelectedIndex)
	case events.ExtendContinuous:
		if len(sv.SelectedIndexes) == 0 {
			sv.SelectedIndex = idx
			sv.SelectIndex(idx)
			sv.IndexGrabFocus(idx)
			sv.Send(events.Select) //  sv.SelectedIndex)
		} else {
			minIndex := -1
			maxIndex := 0
			for r := range sv.SelectedIndexes {
				if minIndex < 0 {
					minIndex = r
				} else {
					minIndex = min(minIndex, r)
				}
				maxIndex = max(maxIndex, r)
			}
			cidx := idx
			sv.SelectedIndex = idx
			sv.SelectIndex(idx)
			if idx < minIndex {
				for cidx < minIndex {
					r := sv.MoveDown(events.SelectQuiet) // just select
					cidx = r
				}
			} else if idx > maxIndex {
				for cidx > maxIndex {
					r := sv.MoveUp(events.SelectQuiet) // just select
					cidx = r
				}
			}
			sv.IndexGrabFocus(idx)
			sv.Send(events.Select) //  sv.SelectedIndex)
		}
	case events.ExtendOne:
		if sv.IndexIsSelected(idx) {
			sv.UnselectIndexAction(idx)
			sv.Send(events.Select) //  sv.SelectedIndex)
		} else {
			sv.SelectedIndex = idx
			sv.SelectIndex(idx)
			sv.IndexGrabFocus(idx)
			sv.Send(events.Select) //  sv.SelectedIndex)
		}
	case events.Unselect:
		sv.SelectedIndex = idx
		sv.UnselectIndexAction(idx)
	case events.SelectQuiet:
		sv.SelectedIndex = idx
		sv.SelectIndex(idx)
	case events.UnselectQuiet:
		sv.SelectedIndex = idx
		sv.UnselectIndex(idx)
	}
	sv.This().(SliceViewer).UpdateWidgets()
	sv.ApplyStyleTree()
	sv.NeedsRender()
}

// UnselectIndexAction unselects this idx (if selected) -- and emits a signal
func (sv *SliceViewBase) UnselectIndexAction(idx int) {
	if sv.IndexIsSelected(idx) {
		sv.UnselectIndex(idx)
	}
}

///////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeDataIndex adds mimedata for given idx: an application/json of the struct
func (sv *SliceViewBase) MimeDataIndex(md *mimedata.Mimes, idx int) {
	sv.ViewMuLock()
	val := sv.SliceVal(idx)
	b, err := json.MarshalIndent(val, "", "  ")
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: fileinfo.DataJson, Data: b})
	} else {
		log.Printf("core.SliceViewBase MimeData JSON Marshall error: %v\n", err)
	}
	sv.ViewMuUnlock()
}

// FromMimeData creates a slice of structs from mime data
func (sv *SliceViewBase) FromMimeData(md mimedata.Mimes) []any {
	svtyp := sv.SliceNPVal.Type()
	sl := make([]any, 0, len(md))
	for _, d := range md {
		if d.Type == fileinfo.DataJson {
			nval := reflect.New(svtyp.Elem()).Interface()
			err := json.Unmarshal(d.Data, nval)
			if err == nil {
				sl = append(sl, nval)
			} else {
				log.Printf("core.SliceViewBase FromMimeData: JSON load error: %v\n", err)
			}
		}
	}
	return sl
}

// MimeDataType returns the data type for mime clipboard (copy / paste) data
// e.g., fileinfo.DataJson
func (sv *SliceViewBase) MimeDataType() string {
	return fileinfo.DataJson
}

// CopySelectToMime copies selected rows to mime data
func (sv *SliceViewBase) CopySelectToMime() mimedata.Mimes {
	nitms := len(sv.SelectedIndexes)
	if nitms == 0 {
		return nil
	}
	ixs := sv.SelectedIndexesList(false) // ascending
	md := make(mimedata.Mimes, 0, nitms)
	for _, i := range ixs {
		sv.MimeDataIndex(&md, i)
	}
	return md
}

// CopyIndexes copies selected idxs to system.Clipboard, optionally resetting the selection
func (sv *SliceViewBase) CopyIndexes(reset bool) { //types:add
	nitms := len(sv.SelectedIndexes)
	if nitms == 0 {
		return
	}
	md := sv.This().(SliceViewer).CopySelectToMime()
	if md != nil {
		sv.Clipboard().Write(md)
	}
	if reset {
		sv.UnselectAllIndexes()
	}
}

// DeleteIndexes deletes all selected indexes
func (sv *SliceViewBase) DeleteIndexes() { //types:add
	if len(sv.SelectedIndexes) == 0 {
		return
	}

	ixs := sv.SelectedIndexesList(true) // descending sort
	for _, i := range ixs {
		sv.This().(SliceViewer).SliceDeleteAt(i)
	}
	sv.SendChange()
	sv.This().(SliceViewer).UpdateWidgets()
	sv.NeedsRender()
}

// CutIndexes copies selected indexes to system.Clipboard and deletes selected indexes
func (sv *SliceViewBase) CutIndexes() { //types:add
	if len(sv.SelectedIndexes) == 0 {
		return
	}

	sv.CopyIndexes(false)
	ixs := sv.SelectedIndexesList(true) // descending sort
	idx := ixs[0]
	sv.UnselectAllIndexes()
	for _, i := range ixs {
		sv.This().(SliceViewer).SliceDeleteAt(i)
	}
	sv.SendChange()
	sv.SelectIndexAction(idx, events.SelectOne)
	sv.This().(SliceViewer).UpdateWidgets()
	sv.NeedsRender()
}

// PasteIndex pastes clipboard at given idx
func (sv *SliceViewBase) PasteIndex(idx int) { //types:add
	sv.TmpIndex = idx
	dt := sv.This().(SliceViewer).MimeDataType()
	md := sv.Clipboard().Read([]string{dt})
	if md != nil {
		sv.PasteMenu(md, sv.TmpIndex)
	}
}

// MakePasteMenu makes the menu of options for paste events
func (sv *SliceViewBase) MakePasteMenu(m *core.Scene, md mimedata.Mimes, idx int, mod events.DropMods, fun func()) {
	svi := sv.This().(SliceViewer)
	if mod == events.DropCopy {
		core.NewButton(m).SetText("Assign to").OnClick(func(e events.Event) {
			svi.PasteAssign(md, idx)
			if fun != nil {
				fun()
			}
		})
	}
	core.NewButton(m).SetText("Insert before").OnClick(func(e events.Event) {
		svi.PasteAtIndex(md, idx)
		if fun != nil {
			fun()
		}
	})
	core.NewButton(m).SetText("Insert after").OnClick(func(e events.Event) {
		svi.PasteAtIndex(md, idx+1)
		if fun != nil {
			fun()
		}
	})
	core.NewButton(m).SetText("Cancel")
}

// PasteMenu performs a paste from the clipboard using given data -- pops up
// a menu to determine what specifically to do
func (sv *SliceViewBase) PasteMenu(md mimedata.Mimes, idx int) {
	sv.UnselectAllIndexes()
	mf := func(m *core.Scene) {
		sv.MakePasteMenu(m, md, idx, events.DropCopy, nil)
	}
	pos := sv.IndexPos(idx)
	core.NewMenu(mf, sv.This().(core.Widget), pos).Run()
}

// PasteAssign assigns mime data (only the first one!) to this idx
func (sv *SliceViewBase) PasteAssign(md mimedata.Mimes, idx int) {
	sl := sv.FromMimeData(md)
	if len(sl) == 0 {
		return
	}
	ns := sl[0]
	sv.SliceNPVal.Index(idx).Set(reflect.ValueOf(ns).Elem())
	sv.SendChange()
	sv.NeedsRender()
}

// PasteAtIndex inserts object(s) from mime data at (before) given slice index
func (sv *SliceViewBase) PasteAtIndex(md mimedata.Mimes, idx int) {
	sl := sv.FromMimeData(md)
	if len(sl) == 0 {
		return
	}
	svl := reflect.ValueOf(sv.Slice)
	svnp := sv.SliceNPVal

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

	sv.SliceNPVal = reflectx.NonPointerValue(reflect.ValueOf(sv.Slice)) // need to update after changes

	sv.SendChange()
	sv.SelectIndexAction(idx, events.SelectOne)
	sv.This().(SliceViewer).UpdateWidgets()
	sv.NeedsRender()
}

// Duplicate copies selected items and inserts them after current selection --
// return idx of start of duplicates if successful, else -1
func (sv *SliceViewBase) Duplicate() int { //types:add
	nitms := len(sv.SelectedIndexes)
	if nitms == 0 {
		return -1
	}
	ixs := sv.SelectedIndexesList(true) // descending sort -- last first
	pasteAt := ixs[0]
	sv.CopyIndexes(true)
	dt := sv.This().(SliceViewer).MimeDataType()
	md := sv.Clipboard().Read([]string{dt})
	sv.This().(SliceViewer).PasteAtIndex(md, pasteAt)
	return pasteAt
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// SelectRowIfNone selects the row the mouse is on if there
// are no currently selected items.  Returns false if no valid mouse row.
func (sv *SliceViewBase) SelectRowIfNone(e events.Event) bool {
	nitms := len(sv.SelectedIndexes)
	if nitms > 0 {
		return true
	}
	row, _, isValid := sv.This().(SliceViewer).SliceGrid().IndexFromPixel(e.Pos())
	if !isValid {
		return false
	}
	sv.UpdateSelectRow(row, e.SelectMode())
	return true
}

// MousePosInGrid returns true if the event mouse position is
// located within the slicegrid.
func (sv *SliceViewBase) MousePosInGrid(e events.Event) bool {
	return sv.This().(SliceViewer).SliceGrid().MousePosInGrid(e.Pos())
}

func (sv *SliceViewBase) DragStart(e events.Event) {
	if !sv.SelectRowIfNone(e) || !sv.MousePosInGrid(e) {
		return
	}
	ixs := sv.SelectedIndexesList(false) // ascending
	if len(ixs) == 0 {
		return
	}
	md := sv.This().(SliceViewer).CopySelectToMime()
	w, ok := sv.This().(SliceViewer).RowFirstWidget(ixs[0] - sv.StartIndex)
	if ok {
		sv.Scene.Events.DragStart(w, md, e)
		e.SetHandled()
		// } else {
		// 	fmt.Println("SliceView DND programmer error")
	}
}

func (sv *SliceViewBase) DragDrop(e events.Event) {
	de := e.(*events.DragDrop)
	if de.Data == nil {
		return
	}
	svi := sv.This().(SliceViewer)
	pos := de.Pos()
	idx, ok := sv.IndexFromPos(pos.Y)
	if ok {
		// sv.DraggedIndexes = nil
		sv.TmpIndex = idx
		sv.SaveDraggedIndexes(idx)
		md := de.Data.(mimedata.Mimes)
		mf := func(m *core.Scene) {
			sv.Scene.Events.DragMenuAddModText(m, de.DropMod)
			svi.MakePasteMenu(m, md, idx, de.DropMod, func() {
				svi.DropFinalize(de)
			})
		}
		pos := sv.IndexPos(sv.TmpIndex)
		core.NewMenu(mf, sv.This().(core.Widget), pos).Run()
	}
}

// DropFinalize is called to finalize Drop actions on the Source node.
// Only relevant for DropMod == DropMove.
func (sv *SliceViewBase) DropFinalize(de *events.DragDrop) {
	sv.NeedsLayout()
	sv.UnselectAllIndexes()
	sv.Scene.Events.DropFinalize(de) // sends DropDeleteSource to Source
}

// DropDeleteSource handles delete source event for DropMove case
func (sv *SliceViewBase) DropDeleteSource(e events.Event) {
	sort.Slice(sv.DraggedIndexes, func(i, j int) bool {
		return sv.DraggedIndexes[i] > sv.DraggedIndexes[j]
	})
	idx := sv.DraggedIndexes[0]
	for _, i := range sv.DraggedIndexes {
		sv.This().(SliceViewer).SliceDeleteAt(i)
	}
	sv.DraggedIndexes = nil
	sv.SelectIndexAction(idx, events.SelectOne)
}

// SaveDraggedIndexes saves selectedindexes into dragged indexes
// taking into account insertion at idx
func (sv *SliceViewBase) SaveDraggedIndexes(idx int) {
	sz := len(sv.SelectedIndexes)
	if sz == 0 {
		sv.DraggedIndexes = nil
		return
	}
	ixs := sv.SelectedIndexesList(false) // ascending
	sv.DraggedIndexes = make([]int, len(ixs))
	for i, ix := range ixs {
		if ix > idx {
			sv.DraggedIndexes[i] = ix + sz // make room for insertion
		} else {
			sv.DraggedIndexes[i] = ix
		}
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Events

func (sv *SliceViewBase) ContextMenu(m *core.Scene) {
	if sv.IsReadOnly() || sv.Is(SliceViewIsArray) {
		return
	}
	core.NewButton(m).SetText("Add row").SetIcon(icons.Add).OnClick(func(e events.Event) {
		sv.SliceNewAtRow((sv.SelectedIndex - sv.StartIndex) + 1)
	})
	core.NewButton(m).SetText("Delete row").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		sv.SliceDeleteAtRow(sv.SelectedIndex - sv.StartIndex)
	})
	core.NewSeparator(m)
	core.NewButton(m).SetText("Copy").SetIcon(icons.Copy).OnClick(func(e events.Event) {
		sv.CopyIndexes(true)
	})
	core.NewButton(m).SetText("Cut").SetIcon(icons.Cut).OnClick(func(e events.Event) {
		sv.CutIndexes()
	})
	core.NewButton(m).SetText("Paste").SetIcon(icons.Paste).OnClick(func(e events.Event) {
		sv.PasteIndex(sv.SelectedIndex)
	})
	core.NewButton(m).SetText("Duplicate").SetIcon(icons.Copy).OnClick(func(e events.Event) {
		sv.Duplicate()
	})
}

// KeyInputNav supports multiple selection navigation keys
func (sv *SliceViewBase) KeyInputNav(kt events.Event) {
	kf := keymap.Of(kt.KeyChord())
	selMode := events.SelectModeBits(kt.Modifiers())
	if selMode == events.SelectOne {
		if sv.Is(SliceViewSelectMode) {
			selMode = events.ExtendContinuous
		}
	}
	switch kf {
	case keymap.CancelSelect:
		sv.UnselectAllIndexes()
		sv.SetFlag(false, SliceViewSelectMode)
		kt.SetHandled()
	case keymap.MoveDown:
		sv.MoveDownAction(selMode)
		kt.SetHandled()
	case keymap.MoveUp:
		sv.MoveUpAction(selMode)
		kt.SetHandled()
	case keymap.PageDown:
		sv.MovePageDownAction(selMode)
		kt.SetHandled()
	case keymap.PageUp:
		sv.MovePageUpAction(selMode)
		kt.SetHandled()
	case keymap.SelectMode:
		sv.SetFlag(!sv.Is(SliceViewSelectMode), SliceViewSelectMode)
		kt.SetHandled()
	case keymap.SelectAll:
		sv.SelectAllIndexes()
		sv.SetFlag(false, SliceViewSelectMode)
		kt.SetHandled()
	}
}

func (sv *SliceViewBase) KeyInputEditable(kt events.Event) {
	sv.KeyInputNav(kt)
	if kt.IsHandled() {
		return
	}
	idx := sv.SelectedIndex
	kf := keymap.Of(kt.KeyChord())
	if core.DebugSettings.KeyEventTrace {
		slog.Info("SliceViewBase KeyInput", "widget", sv, "keyFunction", kf)
	}
	switch kf {
	// case keymap.Delete: // too dangerous
	// 	sv.This().(SliceViewer).SliceDeleteAt(sv.SelectedIndex)
	// 	sv.SelectMode = false
	// 	sv.SelectIndexAction(idx, events.SelectOne)
	// 	kt.SetHandled()
	case keymap.Duplicate:
		nidx := sv.Duplicate()
		sv.SetFlag(false, SliceViewSelectMode)
		if nidx >= 0 {
			sv.SelectIndexAction(nidx, events.SelectOne)
		}
		kt.SetHandled()
	case keymap.Insert:
		sv.This().(SliceViewer).SliceNewAt(idx)
		sv.SetFlag(false, SliceViewSelectMode)
		sv.SelectIndexAction(idx+1, events.SelectOne) // todo: somehow nidx not working
		kt.SetHandled()
	case keymap.InsertAfter:
		sv.This().(SliceViewer).SliceNewAt(idx + 1)
		sv.SetFlag(false, SliceViewSelectMode)
		sv.SelectIndexAction(idx+1, events.SelectOne)
		kt.SetHandled()
	case keymap.Copy:
		sv.CopyIndexes(true)
		sv.SetFlag(false, SliceViewSelectMode)
		sv.SelectIndexAction(idx, events.SelectOne)
		kt.SetHandled()
	case keymap.Cut:
		sv.CutIndexes()
		sv.SetFlag(false, SliceViewSelectMode)
		kt.SetHandled()
	case keymap.Paste:
		sv.PasteIndex(sv.SelectedIndex)
		sv.SetFlag(false, SliceViewSelectMode)
		kt.SetHandled()
	}
}

func (sv *SliceViewBase) KeyInputReadOnly(kt events.Event) {
	if sv.Is(SliceViewReadOnlyMultiSelect) {
		sv.KeyInputNav(kt)
		if kt.IsHandled() {
			return
		}
	}
	selMode := kt.SelectMode()
	if sv.Is(SliceViewSelectMode) {
		selMode = events.ExtendOne
	}
	kf := keymap.Of(kt.KeyChord())
	if core.DebugSettings.KeyEventTrace {
		slog.Info("SliceViewBase ReadOnly KeyInput", "widget", sv, "keyFunction", kf)
	}
	idx := sv.SelectedIndex
	switch {
	case kf == keymap.MoveDown:
		ni := idx + 1
		if ni < sv.SliceSize {
			sv.ScrollToIndex(ni)
			sv.UpdateSelectIndex(ni, true, selMode)
			kt.SetHandled()
		}
	case kf == keymap.MoveUp:
		ni := idx - 1
		if ni >= 0 {
			sv.ScrollToIndex(ni)
			sv.UpdateSelectIndex(ni, true, selMode)
			kt.SetHandled()
		}
	case kf == keymap.PageDown:
		ni := min(idx+sv.VisRows-1, sv.SliceSize-1)
		sv.ScrollToIndex(ni)
		sv.UpdateSelectIndex(ni, true, selMode)
		kt.SetHandled()
	case kf == keymap.PageUp:
		ni := max(idx-(sv.VisRows-1), 0)
		sv.ScrollToIndex(ni)
		sv.UpdateSelectIndex(ni, true, selMode)
		kt.SetHandled()
	case kf == keymap.Enter || kf == keymap.Accept || kt.KeyRune() == ' ':
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
		prevHoverRow := sv.HoverRow
		if !isValid {
			sv.HoverRow = -1
			sv.Styles.Cursor = sv.NormalCursor
		} else {
			sv.HoverRow = row
			sv.Styles.Cursor = cursors.Pointer
		}
		sv.CurrentCursor = sv.Styles.Cursor
		if sv.HoverRow != prevHoverRow {
			sv.NeedsRender()
		}
	})
	sv.On(events.MouseDrag, func(e events.Event) {
		row, idx, isValid := sv.RowFromEventPos(e)
		if !isValid {
			return
		}
		sv.This().(SliceViewer).SliceGrid().AutoScroll(math32.Vec2(0, float32(idx)))
		prevHoverRow := sv.HoverRow
		if !isValid {
			sv.HoverRow = -1
			sv.Styles.Cursor = sv.NormalCursor
		} else {
			sv.HoverRow = row
			sv.Styles.Cursor = cursors.Pointer
		}
		sv.CurrentCursor = sv.Styles.Cursor
		if sv.HoverRow != prevHoverRow {
			sv.NeedsRender()
		}
	})
	sv.OnFirst(events.DoubleClick, func(e events.Event) {
		row, _, isValid := sv.RowFromEventPos(e)
		if !isValid {
			return
		}
		if sv.LastClick != row+sv.StartIndex {
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
	core.Frame // note: must be a frame to support stripes!

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

func (sg *SliceViewGrid) SizeFromChildren(iter int, pass core.LayoutPasses) math32.Vector2 {
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
		sg.VisRows = int(math32.Floor(allocHt / sg.RowHeight))
	}
	sg.VisRows = max(sg.VisRows, sg.MinRows)
	minHt := sg.RowHeight * float32(sg.MinRows)
	// visHt := sg.RowHeight * float32(sg.VisRows)
	csz.Y = minHt
	return csz
}

func (sg *SliceViewGrid) SetScrollParams(d math32.Dims, sb *core.Slider) {
	if d == math32.X {
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
	svi := sg.ParentByType(SliceViewBaseType, tree.Embeds)
	if svi == nil {
		return nil, nil
	}
	sv := svi.(SliceViewer)
	return sv, sv.AsSliceViewBase()
}

func (sg *SliceViewGrid) ScrollChanged(d math32.Dims, sb *core.Slider) {
	if d == math32.X {
		sg.Frame.ScrollChanged(d, sb)
		return
	}
	_, sv := sg.SliceView()
	if sv == nil {
		return
	}
	sv.StartIndex = int(math32.Round(sb.Value))
	sv.This().(SliceViewer).UpdateWidgets()
	sg.NeedsRender()
}

func (sg *SliceViewGrid) ScrollValues(d math32.Dims) (maxSize, visSize, visPct float32) {
	if d == math32.X {
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
	if !sg.HasScroll[math32.Y] || sg.Scrolls[math32.Y] == nil {
		return
	}
	sb := sg.Scrolls[math32.Y]
	sb.SetValue(float32(idx))
}

func (sg *SliceViewGrid) UpdateBackgrounds() {
	bg := sg.Styles.ActualBackground
	if sg.LastBackground == bg {
		return
	}
	sg.LastBackground = bg

	// we take our zebra intensity applied foreground color and then overlay it onto our background color

	zclr := colors.WithAF32(colors.ToUniform(sg.Styles.Color), core.AppearanceSettings.ZebraStripesWeight())
	sg.BgStripe = gradient.Apply(bg, func(c color.Color) color.Color {
		return colors.AlphaBlend(c, zclr)
	})

	hclr := colors.WithAF32(colors.ToUniform(sg.Styles.Color), 0.08)
	sg.BgHover = gradient.Apply(bg, func(c color.Color) color.Color {
		return colors.AlphaBlend(c, hclr)
	})

	zhclr := colors.WithAF32(colors.ToUniform(sg.Styles.Color), core.AppearanceSettings.ZebraStripesWeight()+0.08)
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

func (sg *SliceViewGrid) ChildBackground(child core.Widget) image.Image {
	bg := sg.Styles.ActualBackground
	_, sv := sg.SliceView()
	if sv == nil {
		return bg
	}
	sg.UpdateBackgrounds()
	row, _ := sv.WidgetIndex(child)
	si := row + sv.StartIndex
	return sg.RowBackground(sv.IndexIsSelected(si), si%2 == 1, row == sv.HoverRow)
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
	startIndex := 0
	if sv != nil {
		startIndex = sv.StartIndex
		offset = startIndex % 2
	}
	for r := 0; r < rows; r++ {
		si := r + startIndex
		ht, _ := sg.LayImpl.RowHeight(r, 0)
		miny := st.Y
		for c := 0; c < cols; c++ {
			kw := sg.Child(r*cols + c).(core.Widget).AsWidget()
			pyi := math32.Floor(kw.Geom.Pos.Total.Y)
			if pyi < miny {
				miny = pyi
			}
		}
		st.Y = miny
		ssz := sz
		ssz.Y = ht
		stripe := (r+offset)%2 == 1
		sbg := sg.RowBackground(sv.IndexIsSelected(si), stripe, r == sv.HoverRow)
		pc.BlitBox(st, ssz, sbg)
		st.Y += ht + sg.LayImpl.Gap.Y
	}
}

// MousePosInGrid returns true if the event mouse position is
// located within the slicegrid.
func (sg *SliceViewGrid) MousePosInGrid(pt image.Point) bool {
	ptrel := sg.PointToRelPos(pt)
	sz := sg.Geom.ContentBBox.Size()
	if sg.VisRows == 0 || sz.Y == 0 {
		return false
	}
	if ptrel.Y < 0 || ptrel.Y >= sz.Y || ptrel.X < 0 || ptrel.X >= sz.X-50 { // leave margin on rhs around scroll
		return false
	}
	return true
}

// IndexFromPixel returns the row, column indexes of given pixel point within grid.
// Takes a scene-level position.
func (sg *SliceViewGrid) IndexFromPixel(pt image.Point) (row, col int, isValid bool) {
	if !sg.MousePosInGrid(pt) {
		return
	}
	ptf := math32.Vector2FromPoint(sg.PointToRelPos(pt))
	sz := math32.Vector2FromPoint(sg.Geom.ContentBBox.Size())
	isValid = true
	rows := sg.LayImpl.Shape.Y
	cols := sg.LayImpl.Shape.X
	st := math32.Vector2{}
	got := false
	for r := 0; r < rows; r++ {
		ht, _ := sg.LayImpl.RowHeight(r, 0)
		ht += sg.LayImpl.Gap.Y
		miny := st.Y
		if r > 0 {
			for c := 0; c < cols; c++ {
				kw := sg.Child(r*cols + c).(core.Widget).AsWidget()
				pyi := math32.Floor(kw.Geom.Pos.Total.Y)
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

func (sg *SliceViewGrid) RenderWidget() {
	if sg.PushBounds() {
		sg.Frame.Render()
		sg.RenderStripes()
		sg.RenderChildren()
		sg.RenderScrolls()
		sg.PopBounds()
	}
}
