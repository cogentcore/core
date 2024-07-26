// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

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

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// List represents a slice value with a list of value widgets and optional index widgets.
// Use [ListBase.BindSelect] to make the list designed for item selection.
type List struct {
	ListBase

	// ListStyler is an optional styler for list items.
	ListStyler ListStyler `copier:"-" json:"-" xml:"-"`
}

// ListStyler is a styling function for custom styling and
// configuration of elements in the list.
type ListStyler func(w Widget, s *styles.Style, row int)

func (ls *List) HasStyler() bool {
	return ls.ListStyler != nil
}

func (ls *List) StyleRow(w Widget, idx, fidx int) {
	if ls.ListStyler != nil {
		ls.ListStyler(w, &w.AsWidget().Styles, idx)
	}
}

// note on implementation:
// * ListGrid handles all the layout logic to start with a minimum number of
//   rows and then computes the total number visible based on allocated size.

const (
	// ListRowProperty is the tree property name for the row of a list element.
	ListRowProperty = "ls-row"

	// ListColProperty is the tree property name for the column of a list element.
	ListColProperty = "ls-col"
)

// Lister is the interface used by [ListBase] to
// support any abstractions needed for different types of lists.
type Lister interface {
	tree.Node

	// AsListBase returns the base for direct access to relevant fields etc
	AsListBase() *ListBase

	// RowWidgetNs returns number of widgets per row and
	// offset for index label
	RowWidgetNs() (nWidgPerRow, idxOff int)

	// UpdateSliceSize updates the current size of the slice
	// and sets SliceSize if changed.
	UpdateSliceSize() int

	// UpdateMaxWidths updates the maximum widths per column based
	// on estimates from length of strings (for string values)
	UpdateMaxWidths()

	// SliceIndex returns the logical slice index: si = i + StartIndex,
	// the actual value index vi into the slice value (typically = si),
	// which can be different if there is an index indirection as in
	// tensorcore table.IndexView), and a bool that is true if the
	// index is beyond the available data and is thus invisible,
	// given the row index provided.
	SliceIndex(i int) (si, vi int, invis bool)

	// MakeRow adds config for one row at given widget row index.
	// Plan must be the StructGrid Plan.
	MakeRow(p *tree.Plan, i int)

	// StyleValue performs additional value widget styling
	StyleValue(w Widget, s *styles.Style, row, col int)

	// HasStyler returns whether there is a custom style function.
	HasStyler() bool

	// StyleRow calls a custom style function on given row (and field)
	StyleRow(w Widget, idx, fidx int)

	// RowGrabFocus grabs the focus for the first focusable
	// widget in given row.
	// returns that element or nil if not successful
	// note: grid must have already rendered for focus to be grabbed!
	RowGrabFocus(row int) *WidgetBase

	// NewAt inserts a new blank element at the given index in the slice.
	// -1 indicates to insert the element at the end.
	NewAt(idx int)

	// DeleteAt deletes the element at the given index from the slice.
	DeleteAt(idx int)

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
}

var _ Lister = &List{}

// ListBase is the base for [List] and [Table] and any other displays
// of array-like data. It automatically computes the number of rows that fit
// within its allocated space, and manages the offset view window into the full
// list of items, and supports row selection, copy / paste, Drag-n-Drop, etc.
// Use [ListBase.BindSelect] to make the list designed for item selection.
type ListBase struct { //core:no-new
	Frame

	// Slice is the pointer to the slice that we are viewing.
	Slice any `set:"-"`

	// ShowIndexes is whether to show the indexes of rows or not (default false).
	ShowIndexes bool

	// MinRows specifies the minimum number of rows to display, to ensure
	// at least this amount is displayed.
	MinRows int `default:"4"`

	// SelectedValue is the current selection value.
	// If it is set, it is used as the initially selected value.
	SelectedValue any `copier:"-" display:"-" json:"-" xml:"-"`

	// SelectedIndex is the index of the currently selected item.
	SelectedIndex int `copier:"-" json:"-" xml:"-"`

	// InitSelectedIndex is the index of the row to select at the start.
	InitSelectedIndex int `copier:"-" json:"-" xml:"-"`

	// SelectedIndexes is a list of currently selected slice indexes.
	SelectedIndexes map[int]struct{} `set:"-" copier:"-"`

	// lastClick is the last row that has been clicked on.
	// This is used to prevent erroneous double click events
	// from being sent when the user clicks on multiple different
	// rows in quick succession.
	lastClick int

	// normalCursor is the cached cursor to display when there
	// is no row being hovered.
	normalCursor cursors.Cursor

	// currentCursor is the cached cursor that should currently be
	// displayed.
	currentCursor cursors.Cursor

	// sliceUnderlying is the underlying slice value.
	sliceUnderlying reflect.Value

	// currently hovered row
	hoverRow int

	// list of currently dragged indexes
	draggedIndexes []int

	// VisibleRows is the total number of rows visible in allocated display size.
	VisibleRows int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// StartIndex is the starting slice index of visible rows.
	StartIndex int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// SliceSize is the size of the slice.
	SliceSize int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// MakeIter is the iteration through the configuration process,
	// which is reset when a new slice type is set.
	MakeIter int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// temp idx state for e.g., dnd
	tmpIndex int

	// elementValue is a [reflect.Value] representation of the underlying element type
	// which is used whenever there are no slice elements available
	elementValue reflect.Value

	// maximum width of value column in chars, if string
	maxWidth int

	// ReadOnlyKeyNav is whether support key navigation when ReadOnly (default true).
	// It uses a capture of up / down events to manipulate selection, not focus.
	ReadOnlyKeyNav bool `default:"true"`

	// SelectMode is whether to be in select rows mode or editing mode.
	SelectMode bool `set:"-" copier:"-" json:"-" xml:"-"`

	// ReadOnlyMultiSelect: if list is ReadOnly, default selection mode is to
	// choose one row only. If this is true, standard multiple selection logic
	// with modifier keys is instead supported.
	ReadOnlyMultiSelect bool

	// InFocusGrab is a guard for recursive focus grabbing.
	InFocusGrab bool `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// isArray is whether the slice is actually an array.
	isArray bool

	// ListGrid is the [ListGrid] widget.
	ListGrid *ListGrid `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`
}

func (lb *ListBase) WidgetValue() any { return &lb.Slice }

func (lb *ListBase) Init() {
	lb.Frame.Init()
	lb.AddContextMenu(lb.contextMenu)
	lb.InitSelectedIndex = -1
	lb.hoverRow = -1
	lb.MinRows = 4
	lb.ReadOnlyKeyNav = true

	lb.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable, abilities.DoubleClickable, abilities.TripleClickable)
		s.SetAbilities(!lb.IsReadOnly(), abilities.Draggable, abilities.Droppable)
		s.Cursor = lb.currentCursor
		s.Direction = styles.Column
		// absorb horizontal here, vertical in view
		s.Overflow.X = styles.OverflowAuto
		s.Grow.Set(1, 1)
	})
	if !lb.IsReadOnly() {
		lb.On(events.DragStart, func(e events.Event) {
			lb.dragStart(e)
		})
		lb.On(events.DragEnter, func(e events.Event) {
			e.SetHandled()
		})
		lb.On(events.DragLeave, func(e events.Event) {
			e.SetHandled()
		})
		lb.On(events.Drop, func(e events.Event) {
			lb.dragDrop(e)
		})
		lb.On(events.DropDeleteSource, func(e events.Event) {
			lb.dropDeleteSource(e)
		})
	}
	lb.FinalStyler(func(s *styles.Style) {
		lb.normalCursor = s.Cursor
	})

	lb.OnFinal(events.KeyChord, func(e events.Event) {
		if lb.IsReadOnly() {
			if lb.ReadOnlyKeyNav {
				lb.keyInputReadOnly(e)
			}
		} else {
			lb.keyInputEditable(e)
		}
	})
	lb.On(events.MouseMove, func(e events.Event) {
		row, _, isValid := lb.rowFromEventPos(e)
		prevHoverRow := lb.hoverRow
		if !isValid {
			lb.hoverRow = -1
			lb.Styles.Cursor = lb.normalCursor
		} else {
			lb.hoverRow = row
			lb.Styles.Cursor = cursors.Pointer
		}
		lb.currentCursor = lb.Styles.Cursor
		if lb.hoverRow != prevHoverRow {
			lb.NeedsRender()
		}
	})
	lb.On(events.MouseDrag, func(e events.Event) {
		row, idx, isValid := lb.rowFromEventPos(e)
		if !isValid {
			return
		}
		lb.ListGrid.AutoScroll(math32.Vec2(0, float32(idx)))
		prevHoverRow := lb.hoverRow
		if !isValid {
			lb.hoverRow = -1
			lb.Styles.Cursor = lb.normalCursor
		} else {
			lb.hoverRow = row
			lb.Styles.Cursor = cursors.Pointer
		}
		lb.currentCursor = lb.Styles.Cursor
		if lb.hoverRow != prevHoverRow {
			lb.NeedsRender()
		}
	})
	lb.OnFirst(events.DoubleClick, func(e events.Event) {
		row, _, isValid := lb.rowFromEventPos(e)
		if !isValid {
			return
		}
		if lb.lastClick != row+lb.StartIndex {
			lb.ListGrid.Send(events.Click, e)
			e.SetHandled()
		}
	})
	// we must interpret triple click events as double click
	// events for rapid cross-row double clicking to work correctly
	lb.OnFirst(events.TripleClick, func(e events.Event) {
		lb.Send(events.DoubleClick, e)
	})

	lb.Maker(func(p *tree.Plan) {
		svi := lb.This.(Lister)
		svi.UpdateSliceSize()

		scrollTo := -1
		if lb.SelectedValue != nil {
			idx, ok := sliceIndexByValue(lb.Slice, lb.SelectedValue)
			if ok {
				lb.SelectedIndex = idx
				scrollTo = lb.SelectedIndex
			}
			lb.SelectedValue = nil
			lb.InitSelectedIndex = -1
		} else if lb.InitSelectedIndex >= 0 {
			lb.SelectedIndex = lb.InitSelectedIndex
			lb.InitSelectedIndex = -1
			scrollTo = lb.SelectedIndex
		}
		if scrollTo >= 0 {
			lb.ScrollToIndex(scrollTo)
		}

		lb.Updater(func() {
			lb.UpdateStartIndex()
			svi.UpdateMaxWidths()
		})

		lb.MakeGrid(p, func(p *tree.Plan) {
			for i := 0; i < lb.VisibleRows; i++ {
				svi.MakeRow(p, i)
			}
		})
	})
}

func (lb *ListBase) SliceIndex(i int) (si, vi int, invis bool) {
	si = lb.StartIndex + i
	vi = si
	invis = si >= lb.SliceSize
	return
}

// StyleValue performs additional value widget styling
func (lb *ListBase) StyleValue(w Widget, s *styles.Style, row, col int) {
	if lb.maxWidth > 0 {
		hv := units.Ch(float32(lb.maxWidth))
		s.Min.X.Value = max(s.Min.X.Value, hv.Convert(s.Min.X.Unit, &s.UnitContext).Value)
	}
	s.SetTextWrap(false)
}

func (lb *ListBase) AsListBase() *ListBase {
	return lb
}

func (lb *ListBase) SetSliceBase() {
	lb.SelectMode = false
	lb.MakeIter = 0
	lb.StartIndex = 0
	lb.VisibleRows = lb.MinRows
	if !lb.IsReadOnly() {
		lb.SelectedIndex = -1
	}
	lb.ResetSelectedIndexes()
}

// SetSlice sets the source slice that we are viewing.
// This ReMakes the view for this slice if different.
// Note: it is important to at least set an empty slice of
// the desired type at the start to enable initial configuration.
func (lb *ListBase) SetSlice(sl any) *ListBase {
	if reflectx.AnyIsNil(sl) {
		lb.Slice = nil
		return lb
	}
	// TODO: a lot of this garbage needs to be cleaned up.
	// New is not working!
	newslc := false
	if reflect.TypeOf(sl).Kind() != reflect.Pointer { // prevent crash on non-comparable
		newslc = true
	} else {
		newslc = lb.Slice != sl
	}
	if !newslc {
		lb.MakeIter = 0
		return lb
	}

	lb.SetSliceBase()
	lb.Slice = sl
	lb.sliceUnderlying = reflectx.Underlying(reflect.ValueOf(lb.Slice))
	lb.isArray = reflectx.NonPointerType(reflect.TypeOf(sl)).Kind() == reflect.Array
	lb.elementValue = reflectx.SliceElementValue(sl)
	return lb
}

// rowFromEventPos returns the widget row, slice index, and
// whether the index is in slice range, for given event position.
func (lb *ListBase) rowFromEventPos(e events.Event) (row, idx int, isValid bool) {
	sg := lb.ListGrid
	row, _, isValid = sg.indexFromPixel(e.Pos())
	if !isValid {
		return
	}
	idx = row + lb.StartIndex
	if row < 0 || idx >= lb.SliceSize {
		isValid = false
	}
	return
}

// clickSelectEvent is a helper for processing selection events
// based on a mouse click, which could be a double or triple
// in addition to a regular click.
// Returns false if no further processing should occur,
// because the user clicked outside the range of active rows.
func (lb *ListBase) clickSelectEvent(e events.Event) bool {
	row, _, isValid := lb.rowFromEventPos(e)
	if !isValid {
		e.SetHandled()
	} else {
		lb.updateSelectRow(row, e.SelectMode())
	}
	return isValid
}

// BindSelect makes the list a read-only selection list and then
// binds its events to its scene and its current selection index to the given value.
// It will send an [events.Change] event when the user changes the selection row.
func (lb *ListBase) BindSelect(val *int) *ListBase {
	lb.SetReadOnly(true)
	lb.OnSelect(func(e events.Event) {
		*val = lb.SelectedIndex
		lb.SendChange(e)
	})
	lb.OnDoubleClick(func(e events.Event) {
		if lb.clickSelectEvent(e) {
			*val = lb.SelectedIndex
			lb.Scene.sendKey(keymap.Accept, e) // activate OK button
			if lb.Scene.Stage.Type == DialogStage {
				lb.Scene.Close() // also directly close dialog for value dialogs without OK button
			}
		}
	})
	return lb
}

func (lb *ListBase) UpdateMaxWidths() {
	lb.maxWidth = 0
	npv := reflectx.NonPointerValue(lb.elementValue)
	isString := npv.Type().Kind() == reflect.String && npv.Type() != reflect.TypeFor[icons.Icon]()
	if !isString || lb.SliceSize == 0 {
		return
	}
	mxw := 0
	for rw := 0; rw < lb.SliceSize; rw++ {
		str := reflectx.ToString(lb.sliceElementValue(rw).Interface())
		mxw = max(mxw, len(str))
	}
	lb.maxWidth = mxw
}

// sliceElementValue returns an underlying non-pointer [reflect.Value]
// of slice element at given index or ElementValue if out of range.
func (lb *ListBase) sliceElementValue(si int) reflect.Value {
	var val reflect.Value
	if si < lb.SliceSize {
		val = reflectx.Underlying(lb.sliceUnderlying.Index(si)) // deal with pointer lists
	} else {
		val = lb.elementValue
	}
	if !val.IsValid() {
		val = lb.elementValue
	}
	return val
}

func (lb *ListBase) MakeGrid(p *tree.Plan, maker func(p *tree.Plan)) {
	tree.AddAt(p, "grid", func(w *ListGrid) {
		lb.ListGrid = w
		w.Styler(func(s *styles.Style) {
			nWidgPerRow, _ := lb.This.(Lister).RowWidgetNs()
			w.minRows = lb.MinRows
			s.Display = styles.Grid
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
			lb.SetFocusEvent()
			row, _, isValid := w.indexFromPixel(e.Pos())
			if isValid {
				lb.updateSelectRow(row, e.SelectMode())
				lb.lastClick = row + lb.StartIndex
			}
		}
		w.OnClick(oc)
		w.On(events.ContextMenu, func(e events.Event) {
			// we must select the row on right click so that the context menu
			// corresponds to the right row
			oc(e)
			lb.HandleEvent(e)
		})
		w.Updater(func() {
			nWidgPerRow, _ := lb.This.(Lister).RowWidgetNs()
			w.Styles.Columns = nWidgPerRow
		})
		w.Maker(maker)
	})
}

func (lb *ListBase) MakeValue(w Value, i int) {
	svi := lb.This.(Lister)
	wb := w.AsWidget()
	wb.SetProperty(ListRowProperty, i)
	wb.Styler(func(s *styles.Style) {
		if lb.IsReadOnly() {
			s.SetAbilities(true, abilities.DoubleClickable)
			s.SetAbilities(false, abilities.Hoverable, abilities.Focusable, abilities.Activatable, abilities.TripleClickable)
			s.SetReadOnly(true)
		}
		row, col := lb.widgetIndex(w)
		row += lb.StartIndex
		svi.StyleValue(w, s, row, col)
		if row < lb.SliceSize {
			svi.StyleRow(w, row, col)
		}
	})
	wb.OnSelect(func(e events.Event) {
		e.SetHandled()
		row, _ := lb.widgetIndex(w)
		lb.updateSelectRow(row, e.SelectMode())
		lb.lastClick = row + lb.StartIndex
	})
	wb.OnDoubleClick(lb.HandleEvent)
	wb.On(events.ContextMenu, lb.HandleEvent)
	wb.OnFirst(events.ContextMenu, func(e events.Event) {
		wb.Send(events.Select, e) // we must select the row for context menu actions
	})
	if !lb.IsReadOnly() {
		wb.OnInput(lb.HandleEvent)
	}
}

func (lb *ListBase) MakeRow(p *tree.Plan, i int) {
	svi := lb.This.(Lister)
	si, vi, invis := svi.SliceIndex(i)
	itxt := strconv.Itoa(i)
	val := lb.sliceElementValue(vi)

	if lb.ShowIndexes {
		lb.MakeGridIndex(p, i, si, itxt, invis)
	}

	valnm := fmt.Sprintf("value-%s-%s", itxt, reflectx.ShortTypeName(lb.elementValue.Type()))
	tree.AddNew(p, valnm, func() Value {
		return NewValue(val.Addr().Interface(), "")
	}, func(w Value) {
		wb := w.AsWidget()
		lb.MakeValue(w, i)
		if !lb.IsReadOnly() {
			wb.OnChange(func(e events.Event) {
				lb.SendChange(e)
			})
		}
		wb.Updater(func() {
			wb := w.AsWidget()
			_, vi, invis := svi.SliceIndex(i)
			val := lb.sliceElementValue(vi)
			Bind(val.Addr().Interface(), w)
			wb.SetReadOnly(lb.IsReadOnly())
			wb.SetState(invis, states.Invisible)
			if lb.This.(Lister).HasStyler() {
				w.Style()
			}
			if invis {
				wb.SetSelected(false)
			}
		})
	})

}

func (lb *ListBase) MakeGridIndex(p *tree.Plan, i, si int, itxt string, invis bool) {
	ls := lb.This.(Lister)
	tree.AddAt(p, "index-"+itxt, func(w *Text) {
		w.SetProperty(ListRowProperty, i)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.DoubleClickable)
			s.SetAbilities(!lb.IsReadOnly(), abilities.Draggable, abilities.Droppable)
			s.Cursor = cursors.None
			nd := math32.Log10(float32(lb.SliceSize))
			nd = max(nd, 3)
			s.Min.X.Ch(nd + 2)
			s.Padding.Right.Dp(4)
			s.Text.Align = styles.End
			s.Min.Y.Em(1)
			s.GrowWrap = false
		})
		w.OnSelect(func(e events.Event) {
			e.SetHandled()
			lb.updateSelectRow(i, e.SelectMode())
			lb.lastClick = si
		})
		w.OnDoubleClick(lb.HandleEvent)
		w.On(events.ContextMenu, lb.HandleEvent)
		if !lb.IsReadOnly() {
			w.On(events.DragStart, func(e events.Event) {
				lb.dragStart(e)
			})
			w.On(events.DragEnter, func(e events.Event) {
				e.SetHandled()
			})
			w.On(events.DragLeave, func(e events.Event) {
				e.SetHandled()
			})
			w.On(events.Drop, func(e events.Event) {
				lb.dragDrop(e)
			})
			w.On(events.DropDeleteSource, func(e events.Event) {
				lb.dropDeleteSource(e)
			})
		}
		w.Updater(func() {
			si, _, invis := ls.SliceIndex(i)
			sitxt := strconv.Itoa(si)
			w.SetText(sitxt)
			w.SetReadOnly(lb.IsReadOnly())
			w.SetState(invis, states.Invisible)
			if invis {
				w.SetSelected(false)
			}
		})
	})
}

// RowWidgetNs returns number of widgets per row and offset for index label
func (lb *ListBase) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 2
	idxOff = 1
	if !lb.ShowIndexes {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

// UpdateSliceSize updates and returns the size of the slice
// and sets SliceSize
func (lb *ListBase) UpdateSliceSize() int {
	sz := lb.sliceUnderlying.Len()
	lb.SliceSize = sz
	return sz
}

// widgetIndex returns the row and column indexes for given widget,
// from the properties set during construction.
func (lb *ListBase) widgetIndex(w Widget) (row, col int) {
	if rwi := w.AsTree().Property(ListRowProperty); rwi != nil {
		row = rwi.(int)
	}
	if cli := w.AsTree().Property(ListColProperty); cli != nil {
		col = cli.(int)
	}
	return
}

// UpdateStartIndex updates StartIndex to fit current view
func (lb *ListBase) UpdateStartIndex() {
	sz := lb.This.(Lister).UpdateSliceSize()
	if sz > lb.VisibleRows {
		lastSt := sz - lb.VisibleRows
		lb.StartIndex = min(lastSt, lb.StartIndex)
		lb.StartIndex = max(0, lb.StartIndex)
	} else {
		lb.StartIndex = 0
	}
}

// updateScroll updates the scroll value
func (lb *ListBase) updateScroll() {
	sg := lb.ListGrid
	if sg == nil {
		return
	}
	sg.updateScroll(lb.StartIndex)
}

// newAtRow inserts a new blank element at the given display row.
func (lb *ListBase) newAtRow(row int) {
	lb.This.(Lister).NewAt(lb.StartIndex + row)
}

// NewAt inserts a new blank element at the given index in the slice.
// -1 indicates to insert the element at the end.
func (lb *ListBase) NewAt(idx int) {
	if lb.isArray {
		return
	}

	lb.NewAtSelect(idx)
	reflectx.SliceNewAt(lb.Slice, idx)
	if idx < 0 {
		idx = lb.SliceSize
	}

	lb.This.(Lister).UpdateSliceSize()
	lb.SelectIndexEvent(idx, events.SelectOne)
	lb.UpdateChange()
	lb.IndexGrabFocus(idx)
}

// deleteAtRow deletes the element at the given display row.
func (lb *ListBase) deleteAtRow(row int) {
	lb.This.(Lister).DeleteAt(lb.StartIndex + row)
}

// NewAtSelect updates the selected rows based on
// inserting a new element at the given index.
func (lb *ListBase) NewAtSelect(i int) {
	sl := lb.SelectedIndexesList(false) // ascending
	lb.ResetSelectedIndexes()
	for _, ix := range sl {
		if ix >= i {
			ix++
		}
		lb.SelectedIndexes[ix] = struct{}{}
	}
}

// DeleteAtSelect updates the selected rows based on
// deleting the element at the given index.
func (lb *ListBase) DeleteAtSelect(i int) {
	sl := lb.SelectedIndexesList(true) // desscending
	lb.ResetSelectedIndexes()
	for _, ix := range sl {
		switch {
		case ix == i:
			continue
		case ix > i:
			ix--
		}
		lb.SelectedIndexes[ix] = struct{}{}
	}
}

// DeleteAt deletes the element at the given index from the slice.
func (lb *ListBase) DeleteAt(i int) {
	if lb.isArray {
		return
	}
	if i < 0 || i >= lb.SliceSize {
		return
	}
	lb.DeleteAtSelect(i)
	reflectx.SliceDeleteAt(lb.Slice, i)
	lb.This.(Lister).UpdateSliceSize()
	lb.UpdateChange()
}

func (lb *ListBase) MakeToolbar(p *tree.Plan) {
	if reflectx.AnyIsNil(lb.Slice) {
		return
	}
	if lb.isArray || lb.IsReadOnly() {
		return
	}
	tree.Add(p, func(w *Button) {
		w.SetText("Add").SetIcon(icons.Add).SetTooltip("add a new element to the slice").
			OnClick(func(e events.Event) {
				lb.This.(Lister).NewAt(-1)
			})
	})
}

////////////////////////////////////////////////////////////
//  Row access methods
//  NOTE: row = physical GUI display row, idx = slice index
//  not the same!

// sliceValue returns value interface at given slice index.
func (lb *ListBase) sliceValue(idx int) any {
	if idx < 0 || idx >= lb.SliceSize {
		fmt.Printf("core.ListBase: slice index out of range: %v\n", idx)
		return nil
	}
	val := reflectx.UnderlyingPointer(lb.sliceUnderlying.Index(idx)) // deal with pointer lists
	vali := val.Interface()
	return vali
}

// IsRowInBounds returns true if disp row is in bounds
func (lb *ListBase) IsRowInBounds(row int) bool {
	return row >= 0 && row < lb.VisibleRows
}

// rowFirstWidget returns the first widget for given row (could be index or
// not) -- false if out of range
func (lb *ListBase) rowFirstWidget(row int) (*WidgetBase, bool) {
	if !lb.ShowIndexes {
		return nil, false
	}
	if !lb.IsRowInBounds(row) {
		return nil, false
	}
	nWidgPerRow, _ := lb.This.(Lister).RowWidgetNs()
	sg := lb.ListGrid
	w := sg.Children[row*nWidgPerRow].(Widget).AsWidget()
	return w, true
}

// RowGrabFocus grabs the focus for the first focusable widget
// in given row.  returns that element or nil if not successful
// note: grid must have already rendered for focus to be grabbed!
func (lb *ListBase) RowGrabFocus(row int) *WidgetBase {
	if !lb.IsRowInBounds(row) || lb.InFocusGrab { // range check
		return nil
	}
	nWidgPerRow, idxOff := lb.This.(Lister).RowWidgetNs()
	ridx := nWidgPerRow * row
	sg := lb.ListGrid
	w := sg.Child(ridx + idxOff).(Widget).AsWidget()
	if w.StateIs(states.Focused) {
		return w
	}
	lb.InFocusGrab = true
	w.SetFocusEvent()
	lb.InFocusGrab = false
	return w
}

// IndexGrabFocus grabs the focus for the first focusable widget
// in given idx.  returns that element or nil if not successful.
func (lb *ListBase) IndexGrabFocus(idx int) *WidgetBase {
	lb.ScrollToIndex(idx)
	return lb.This.(Lister).RowGrabFocus(idx - lb.StartIndex)
}

// indexPos returns center of window position of index label for idx (ContextMenuPos)
func (lb *ListBase) indexPos(idx int) image.Point {
	row := idx - lb.StartIndex
	if row < 0 {
		row = 0
	}
	if row > lb.VisibleRows-1 {
		row = lb.VisibleRows - 1
	}
	var pos image.Point
	w, ok := lb.rowFirstWidget(row)
	if ok {
		pos = w.ContextMenuPos(nil)
	}
	return pos
}

// rowFromPos returns the row that contains given vertical position, false if not found
func (lb *ListBase) rowFromPos(posY int) (int, bool) {
	// todo: could optimize search to approx loc, and search up / down from there
	for rw := 0; rw < lb.VisibleRows; rw++ {
		w, ok := lb.rowFirstWidget(rw)
		if ok {
			if w.Geom.TotalBBox.Min.Y < posY && posY < w.Geom.TotalBBox.Max.Y {
				return rw, true
			}
		}
	}
	return -1, false
}

// indexFromPos returns the idx that contains given vertical position, false if not found
func (lb *ListBase) indexFromPos(posY int) (int, bool) {
	row, ok := lb.rowFromPos(posY)
	if !ok {
		return -1, false
	}
	return row + lb.StartIndex, true
}

// ScrollToIndexNoUpdate ensures that given slice idx is visible
// by scrolling display as needed.
// This version does not update the slicegrid.
// Just computes the StartIndex and updates the scrollbar
func (lb *ListBase) ScrollToIndexNoUpdate(idx int) bool {
	if lb.VisibleRows == 0 {
		return false
	}
	if idx < lb.StartIndex {
		lb.StartIndex = idx
		lb.StartIndex = max(0, lb.StartIndex)
		lb.updateScroll()
		return true
	}
	if idx >= lb.StartIndex+lb.VisibleRows {
		lb.StartIndex = idx - (lb.VisibleRows - 4)
		lb.StartIndex = max(0, lb.StartIndex)
		lb.updateScroll()
		return true
	}
	return false
}

// ScrollToIndex ensures that given slice idx is visible
// by scrolling display as needed.
func (lb *ListBase) ScrollToIndex(idx int) bool {
	update := lb.ScrollToIndexNoUpdate(idx)
	if update {
		lb.Update()
	}
	return update
}

// sliceIndexByValue searches for first index that contains given value in slice;
// returns false if not found
func sliceIndexByValue(slc any, fldVal any) (int, bool) {
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

// moveDown moves the selection down to next row, using given select mode
// (from keyboard modifiers) -- returns newly selected row or -1 if failed
func (lb *ListBase) moveDown(selMode events.SelectModes) int {
	if lb.SelectedIndex >= lb.SliceSize-1 {
		lb.SelectedIndex = lb.SliceSize - 1
		return -1
	}
	lb.SelectedIndex++
	lb.SelectIndexEvent(lb.SelectedIndex, selMode)
	return lb.SelectedIndex
}

// moveDownEvent moves the selection down to next row, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected
// row
func (lb *ListBase) moveDownEvent(selMode events.SelectModes) int {
	nidx := lb.moveDown(selMode)
	if nidx >= 0 {
		lb.ScrollToIndex(nidx)
		lb.Send(events.Select) // todo: need to do this for the item?
	}
	return nidx
}

// moveUp moves the selection up to previous idx, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (lb *ListBase) moveUp(selMode events.SelectModes) int {
	if lb.SelectedIndex <= 0 {
		lb.SelectedIndex = 0
		return -1
	}
	lb.SelectedIndex--
	lb.SelectIndexEvent(lb.SelectedIndex, selMode)
	return lb.SelectedIndex
}

// moveUpEvent moves the selection up to previous idx, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (lb *ListBase) moveUpEvent(selMode events.SelectModes) int {
	nidx := lb.moveUp(selMode)
	if nidx >= 0 {
		lb.ScrollToIndex(nidx)
		lb.Send(events.Select)
	}
	return nidx
}

// movePageDown moves the selection down to next page, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (lb *ListBase) movePageDown(selMode events.SelectModes) int {
	if lb.SelectedIndex >= lb.SliceSize-1 {
		lb.SelectedIndex = lb.SliceSize - 1
		return -1
	}
	lb.SelectedIndex += lb.VisibleRows
	lb.SelectedIndex = min(lb.SelectedIndex, lb.SliceSize-1)
	lb.SelectIndexEvent(lb.SelectedIndex, selMode)
	return lb.SelectedIndex
}

// movePageDownEvent moves the selection down to next page, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (lb *ListBase) movePageDownEvent(selMode events.SelectModes) int {
	nidx := lb.movePageDown(selMode)
	if nidx >= 0 {
		lb.ScrollToIndex(nidx)
		lb.Send(events.Select)
	}
	return nidx
}

// movePageUp moves the selection up to previous page, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (lb *ListBase) movePageUp(selMode events.SelectModes) int {
	if lb.SelectedIndex <= 0 {
		lb.SelectedIndex = 0
		return -1
	}
	lb.SelectedIndex -= lb.VisibleRows
	lb.SelectedIndex = max(0, lb.SelectedIndex)
	lb.SelectIndexEvent(lb.SelectedIndex, selMode)
	return lb.SelectedIndex
}

// movePageUpEvent moves the selection up to previous page, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (lb *ListBase) movePageUpEvent(selMode events.SelectModes) int {
	nidx := lb.movePageUp(selMode)
	if nidx >= 0 {
		lb.ScrollToIndex(nidx)
		lb.Send(events.Select)
	}
	return nidx
}

//////////////////////////////////////////////////////////
//    Selection: user operates on the index labels

// updateSelectRow updates the selection for the given row
func (lb *ListBase) updateSelectRow(row int, selMode events.SelectModes) {
	idx := row + lb.StartIndex
	if row < 0 || idx >= lb.SliceSize {
		return
	}
	sel := !lb.indexIsSelected(idx)
	lb.updateSelectIndex(idx, sel, selMode)
}

// updateSelectIndex updates the selection for the given index
func (lb *ListBase) updateSelectIndex(idx int, sel bool, selMode events.SelectModes) {
	if lb.IsReadOnly() && !lb.ReadOnlyMultiSelect {
		lb.unselectAllIndexes()
		if sel || lb.SelectedIndex == idx {
			lb.SelectedIndex = idx
			lb.SelectIndex(idx)
		}
		lb.Send(events.Select)
		lb.Restyle()
	} else {
		lb.SelectIndexEvent(idx, selMode)
	}
}

// indexIsSelected returns the selected status of given slice index
func (lb *ListBase) indexIsSelected(idx int) bool {
	if lb.IsReadOnly() && !lb.ReadOnlyMultiSelect {
		return idx == lb.SelectedIndex
	}
	_, ok := lb.SelectedIndexes[idx]
	return ok
}

func (lb *ListBase) ResetSelectedIndexes() {
	lb.SelectedIndexes = make(map[int]struct{})
}

// SelectedIndexesList returns list of selected indexes,
// sorted either ascending or descending
func (lb *ListBase) SelectedIndexesList(descendingSort bool) []int {
	rws := make([]int, len(lb.SelectedIndexes))
	i := 0
	for r := range lb.SelectedIndexes {
		if r >= lb.SliceSize { // double safety check at this point
			delete(lb.SelectedIndexes, r)
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
func (lb *ListBase) SelectIndex(idx int) {
	lb.SelectedIndexes[idx] = struct{}{}
}

// unselectIndex unselects given idx (if selected)
func (lb *ListBase) unselectIndex(idx int) {
	if lb.indexIsSelected(idx) {
		delete(lb.SelectedIndexes, idx)
	}
}

// unselectAllIndexes unselects all selected idxs
func (lb *ListBase) unselectAllIndexes() {
	lb.ResetSelectedIndexes()
}

// selectAllIndexes selects all idxs
func (lb *ListBase) selectAllIndexes() {
	lb.unselectAllIndexes()
	lb.SelectedIndexes = make(map[int]struct{}, lb.SliceSize)
	for idx := 0; idx < lb.SliceSize; idx++ {
		lb.SelectedIndexes[idx] = struct{}{}
	}
	lb.NeedsRender()
}

// SelectIndexEvent is called when a select event has been received (e.g., a
// mouse click) -- translates into selection updates -- gets selection mode
// from mouse event (ExtendContinuous, ExtendOne)
func (lb *ListBase) SelectIndexEvent(idx int, mode events.SelectModes) {
	if mode == events.NoSelect {
		return
	}
	idx = min(idx, lb.SliceSize-1)
	if idx < 0 {
		lb.ResetSelectedIndexes()
		return
	}
	// row := idx - sv.StartIndex // note: could be out of bounds

	switch mode {
	case events.SelectOne:
		if lb.indexIsSelected(idx) {
			if len(lb.SelectedIndexes) > 1 {
				lb.unselectAllIndexes()
			}
			lb.SelectedIndex = idx
			lb.SelectIndex(idx)
			lb.IndexGrabFocus(idx)
		} else {
			lb.unselectAllIndexes()
			lb.SelectedIndex = idx
			lb.SelectIndex(idx)
			lb.IndexGrabFocus(idx)
		}
		lb.Send(events.Select) //  sv.SelectedIndex)
	case events.ExtendContinuous:
		if len(lb.SelectedIndexes) == 0 {
			lb.SelectedIndex = idx
			lb.SelectIndex(idx)
			lb.IndexGrabFocus(idx)
			lb.Send(events.Select) //  sv.SelectedIndex)
		} else {
			minIndex := -1
			maxIndex := 0
			for r := range lb.SelectedIndexes {
				if minIndex < 0 {
					minIndex = r
				} else {
					minIndex = min(minIndex, r)
				}
				maxIndex = max(maxIndex, r)
			}
			cidx := idx
			lb.SelectedIndex = idx
			lb.SelectIndex(idx)
			if idx < minIndex {
				for cidx < minIndex {
					r := lb.moveDown(events.SelectQuiet) // just select
					cidx = r
				}
			} else if idx > maxIndex {
				for cidx > maxIndex {
					r := lb.moveUp(events.SelectQuiet) // just select
					cidx = r
				}
			}
			lb.IndexGrabFocus(idx)
			lb.Send(events.Select) //  sv.SelectedIndex)
		}
	case events.ExtendOne:
		if lb.indexIsSelected(idx) {
			lb.unselectIndexEvent(idx)
			lb.Send(events.Select) //  sv.SelectedIndex)
		} else {
			lb.SelectedIndex = idx
			lb.SelectIndex(idx)
			lb.IndexGrabFocus(idx)
			lb.Send(events.Select) //  sv.SelectedIndex)
		}
	case events.Unselect:
		lb.SelectedIndex = idx
		lb.unselectIndexEvent(idx)
	case events.SelectQuiet:
		lb.SelectedIndex = idx
		lb.SelectIndex(idx)
	case events.UnselectQuiet:
		lb.SelectedIndex = idx
		lb.unselectIndex(idx)
	}
	lb.Restyle()
}

// unselectIndexEvent unselects this idx (if selected) -- and emits a signal
func (lb *ListBase) unselectIndexEvent(idx int) {
	if lb.indexIsSelected(idx) {
		lb.unselectIndex(idx)
	}
}

///////////////////////////////////////////////////
//    Copy / Cut / Paste

// mimeDataIndex adds mimedata for given idx: an application/json of the struct
func (lb *ListBase) mimeDataIndex(md *mimedata.Mimes, idx int) {
	val := lb.sliceValue(idx)
	b, err := json.MarshalIndent(val, "", "  ")
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: fileinfo.DataJson, Data: b})
	} else {
		log.Printf("ListBase MimeData JSON Marshall error: %v\n", err)
	}
}

// fromMimeData creates a slice of structs from mime data
func (lb *ListBase) fromMimeData(md mimedata.Mimes) []any {
	svtyp := lb.sliceUnderlying.Type()
	sl := make([]any, 0, len(md))
	for _, d := range md {
		if d.Type == fileinfo.DataJson {
			nval := reflect.New(svtyp.Elem()).Interface()
			err := json.Unmarshal(d.Data, nval)
			if err == nil {
				sl = append(sl, nval)
			} else {
				log.Printf("ListBase FromMimeData: JSON load error: %v\n", err)
			}
		}
	}
	return sl
}

// MimeDataType returns the data type for mime clipboard (copy / paste) data
// e.g., fileinfo.DataJson
func (lb *ListBase) MimeDataType() string {
	return fileinfo.DataJson
}

// CopySelectToMime copies selected rows to mime data
func (lb *ListBase) CopySelectToMime() mimedata.Mimes {
	nitms := len(lb.SelectedIndexes)
	if nitms == 0 {
		return nil
	}
	ixs := lb.SelectedIndexesList(false) // ascending
	md := make(mimedata.Mimes, 0, nitms)
	for _, i := range ixs {
		lb.mimeDataIndex(&md, i)
	}
	return md
}

// copyIndexes copies selected idxs to system.Clipboard, optionally resetting the selection
func (lb *ListBase) copyIndexes(reset bool) { //types:add
	nitms := len(lb.SelectedIndexes)
	if nitms == 0 {
		return
	}
	md := lb.This.(Lister).CopySelectToMime()
	if md != nil {
		lb.Clipboard().Write(md)
	}
	if reset {
		lb.unselectAllIndexes()
	}
}

// cutIndexes copies selected indexes to system.Clipboard and deletes selected indexes
func (lb *ListBase) cutIndexes() { //types:add
	if len(lb.SelectedIndexes) == 0 {
		return
	}

	lb.copyIndexes(false)
	ixs := lb.SelectedIndexesList(true) // descending sort
	idx := ixs[0]
	lb.unselectAllIndexes()
	for _, i := range ixs {
		lb.This.(Lister).DeleteAt(i)
	}
	lb.SendChange()
	lb.SelectIndexEvent(idx, events.SelectOne)
	lb.Update()
}

// pasteIndex pastes clipboard at given idx
func (lb *ListBase) pasteIndex(idx int) { //types:add
	lb.tmpIndex = idx
	dt := lb.This.(Lister).MimeDataType()
	md := lb.Clipboard().Read([]string{dt})
	if md != nil {
		lb.pasteMenu(md, lb.tmpIndex)
	}
}

// makePasteMenu makes the menu of options for paste events
func (lb *ListBase) makePasteMenu(m *Scene, md mimedata.Mimes, idx int, mod events.DropMods, fun func()) {
	svi := lb.This.(Lister)
	if mod == events.DropCopy {
		NewButton(m).SetText("Assign to").OnClick(func(e events.Event) {
			svi.PasteAssign(md, idx)
			if fun != nil {
				fun()
			}
		})
	}
	NewButton(m).SetText("Insert before").OnClick(func(e events.Event) {
		svi.PasteAtIndex(md, idx)
		if fun != nil {
			fun()
		}
	})
	NewButton(m).SetText("Insert after").OnClick(func(e events.Event) {
		svi.PasteAtIndex(md, idx+1)
		if fun != nil {
			fun()
		}
	})
	NewButton(m).SetText("Cancel")
}

// pasteMenu performs a paste from the clipboard using given data -- pops up
// a menu to determine what specifically to do
func (lb *ListBase) pasteMenu(md mimedata.Mimes, idx int) {
	lb.unselectAllIndexes()
	mf := func(m *Scene) {
		lb.makePasteMenu(m, md, idx, events.DropCopy, nil)
	}
	pos := lb.indexPos(idx)
	NewMenu(mf, lb.This.(Widget), pos).Run()
}

// PasteAssign assigns mime data (only the first one!) to this idx
func (lb *ListBase) PasteAssign(md mimedata.Mimes, idx int) {
	sl := lb.fromMimeData(md)
	if len(sl) == 0 {
		return
	}
	ns := sl[0]
	lb.sliceUnderlying.Index(idx).Set(reflect.ValueOf(ns).Elem())
	lb.UpdateChange()
}

// PasteAtIndex inserts object(s) from mime data at (before) given slice index
func (lb *ListBase) PasteAtIndex(md mimedata.Mimes, idx int) {
	sl := lb.fromMimeData(md)
	if len(sl) == 0 {
		return
	}
	svl := reflect.ValueOf(lb.Slice)
	svnp := lb.sliceUnderlying

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

	lb.sliceUnderlying = reflectx.NonPointerValue(reflect.ValueOf(lb.Slice)) // need to update after changes

	lb.SendChange()
	lb.SelectIndexEvent(idx, events.SelectOne)
	lb.Update()
}

// duplicate copies selected items and inserts them after current selection --
// return idx of start of duplicates if successful, else -1
func (lb *ListBase) duplicate() int { //types:add
	nitms := len(lb.SelectedIndexes)
	if nitms == 0 {
		return -1
	}
	ixs := lb.SelectedIndexesList(true) // descending sort -- last first
	pasteAt := ixs[0]
	lb.copyIndexes(true)
	dt := lb.This.(Lister).MimeDataType()
	md := lb.Clipboard().Read([]string{dt})
	lb.This.(Lister).PasteAtIndex(md, pasteAt)
	return pasteAt
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// selectRowIfNone selects the row the mouse is on if there
// are no currently selected items.  Returns false if no valid mouse row.
func (lb *ListBase) selectRowIfNone(e events.Event) bool {
	nitms := len(lb.SelectedIndexes)
	if nitms > 0 {
		return true
	}
	row, _, isValid := lb.ListGrid.indexFromPixel(e.Pos())
	if !isValid {
		return false
	}
	lb.updateSelectRow(row, e.SelectMode())
	return true
}

// mousePosInGrid returns true if the event mouse position is
// located within the slicegrid.
func (lb *ListBase) mousePosInGrid(e events.Event) bool {
	return lb.ListGrid.mousePosInGrid(e.Pos())
}

func (lb *ListBase) dragStart(e events.Event) {
	if !lb.selectRowIfNone(e) || !lb.mousePosInGrid(e) {
		return
	}
	ixs := lb.SelectedIndexesList(false) // ascending
	if len(ixs) == 0 {
		return
	}
	md := lb.This.(Lister).CopySelectToMime()
	w, ok := lb.rowFirstWidget(ixs[0] - lb.StartIndex)
	if ok {
		lb.Scene.Events.dragStart(w, md, e)
		e.SetHandled()
		// } else {
		// 	fmt.Println("List DND programmer error")
	}
}

func (lb *ListBase) dragDrop(e events.Event) {
	de := e.(*events.DragDrop)
	if de.Data == nil {
		return
	}
	pos := de.Pos()
	idx, ok := lb.indexFromPos(pos.Y)
	if ok {
		// sv.DraggedIndexes = nil
		lb.tmpIndex = idx
		lb.saveDraggedIndexes(idx)
		md := de.Data.(mimedata.Mimes)
		mf := func(m *Scene) {
			lb.Scene.Events.dragMenuAddModText(m, de.DropMod)
			lb.makePasteMenu(m, md, idx, de.DropMod, func() {
				lb.dropFinalize(de)
			})
		}
		pos := lb.indexPos(lb.tmpIndex)
		NewMenu(mf, lb.This.(Widget), pos).Run()
	}
}

// dropFinalize is called to finalize Drop actions on the Source node.
// Only relevant for DropMod == DropMove.
func (lb *ListBase) dropFinalize(de *events.DragDrop) {
	lb.NeedsLayout()
	lb.unselectAllIndexes()
	lb.Scene.Events.dropFinalize(de) // sends DropDeleteSource to Source
}

// dropDeleteSource handles delete source event for DropMove case
func (lb *ListBase) dropDeleteSource(e events.Event) {
	sort.Slice(lb.draggedIndexes, func(i, j int) bool {
		return lb.draggedIndexes[i] > lb.draggedIndexes[j]
	})
	idx := lb.draggedIndexes[0]
	for _, i := range lb.draggedIndexes {
		lb.This.(Lister).DeleteAt(i)
	}
	lb.draggedIndexes = nil
	lb.SelectIndexEvent(idx, events.SelectOne)
}

// saveDraggedIndexes saves selectedindexes into dragged indexes
// taking into account insertion at idx
func (lb *ListBase) saveDraggedIndexes(idx int) {
	sz := len(lb.SelectedIndexes)
	if sz == 0 {
		lb.draggedIndexes = nil
		return
	}
	ixs := lb.SelectedIndexesList(false) // ascending
	lb.draggedIndexes = make([]int, len(ixs))
	for i, ix := range ixs {
		if ix > idx {
			lb.draggedIndexes[i] = ix + sz // make room for insertion
		} else {
			lb.draggedIndexes[i] = ix
		}
	}
}

func (lb *ListBase) contextMenu(m *Scene) {
	if lb.IsReadOnly() || lb.isArray {
		NewButton(m).SetText("Copy").SetIcon(icons.Copy).OnClick(func(e events.Event) {
			lb.copyIndexes(true)
		})
		NewSeparator(m)
		NewButton(m).SetText("Toggle indexes").SetIcon(icons.Numbers).OnClick(func(e events.Event) {
			lb.ShowIndexes = !lb.ShowIndexes
			lb.Update()
		})
		return
	}
	NewButton(m).SetText("Add row").SetIcon(icons.Add).OnClick(func(e events.Event) {
		lb.newAtRow((lb.SelectedIndex - lb.StartIndex) + 1)
	})
	NewButton(m).SetText("Delete row").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		lb.deleteAtRow(lb.SelectedIndex - lb.StartIndex)
	})
	NewSeparator(m)
	NewButton(m).SetText("Copy").SetIcon(icons.Copy).OnClick(func(e events.Event) {
		lb.copyIndexes(true)
	})
	NewButton(m).SetText("Cut").SetIcon(icons.Cut).OnClick(func(e events.Event) {
		lb.cutIndexes()
	})
	NewButton(m).SetText("Paste").SetIcon(icons.Paste).OnClick(func(e events.Event) {
		lb.pasteIndex(lb.SelectedIndex)
	})
	NewButton(m).SetText("Duplicate").SetIcon(icons.Copy).OnClick(func(e events.Event) {
		lb.duplicate()
	})
	NewSeparator(m)
	NewButton(m).SetText("Toggle indexes").SetIcon(icons.Numbers).OnClick(func(e events.Event) {
		lb.ShowIndexes = !lb.ShowIndexes
		lb.Update()
	})
}

// keyInputNav supports multiple selection navigation keys
func (lb *ListBase) keyInputNav(kt events.Event) {
	kf := keymap.Of(kt.KeyChord())
	selMode := events.SelectModeBits(kt.Modifiers())
	if selMode == events.SelectOne {
		if lb.SelectMode {
			selMode = events.ExtendContinuous
		}
	}
	switch kf {
	case keymap.CancelSelect:
		lb.unselectAllIndexes()
		lb.SelectMode = false
		kt.SetHandled()
	case keymap.MoveDown:
		lb.moveDownEvent(selMode)
		kt.SetHandled()
	case keymap.MoveUp:
		lb.moveUpEvent(selMode)
		kt.SetHandled()
	case keymap.PageDown:
		lb.movePageDownEvent(selMode)
		kt.SetHandled()
	case keymap.PageUp:
		lb.movePageUpEvent(selMode)
		kt.SetHandled()
	case keymap.SelectMode:
		lb.SelectMode = !lb.SelectMode
		kt.SetHandled()
	case keymap.SelectAll:
		lb.selectAllIndexes()
		lb.SelectMode = false
		kt.SetHandled()
	}
}

func (lb *ListBase) keyInputEditable(kt events.Event) {
	lb.keyInputNav(kt)
	if kt.IsHandled() {
		return
	}
	idx := lb.SelectedIndex
	kf := keymap.Of(kt.KeyChord())
	if DebugSettings.KeyEventTrace {
		slog.Info("ListBase KeyInput", "widget", lb, "keyFunction", kf)
	}
	switch kf {
	// case keymap.Delete: // too dangerous
	// 	sv.This.(Lister).SliceDeleteAt(sv.SelectedIndex)
	// 	sv.SelectMode = false
	// 	sv.SelectIndexEvent(idx, events.SelectOne)
	// 	kt.SetHandled()
	case keymap.Duplicate:
		nidx := lb.duplicate()
		lb.SelectMode = false
		if nidx >= 0 {
			lb.SelectIndexEvent(nidx, events.SelectOne)
		}
		kt.SetHandled()
	case keymap.Insert:
		lb.This.(Lister).NewAt(idx)
		lb.SelectMode = false
		lb.SelectIndexEvent(idx+1, events.SelectOne) // todo: somehow nidx not working
		kt.SetHandled()
	case keymap.InsertAfter:
		lb.This.(Lister).NewAt(idx + 1)
		lb.SelectMode = false
		lb.SelectIndexEvent(idx+1, events.SelectOne)
		kt.SetHandled()
	case keymap.Copy:
		lb.copyIndexes(true)
		lb.SelectMode = false
		lb.SelectIndexEvent(idx, events.SelectOne)
		kt.SetHandled()
	case keymap.Cut:
		lb.cutIndexes()
		lb.SelectMode = false
		kt.SetHandled()
	case keymap.Paste:
		lb.pasteIndex(lb.SelectedIndex)
		lb.SelectMode = false
		kt.SetHandled()
	}
}

func (lb *ListBase) keyInputReadOnly(kt events.Event) {
	if lb.ReadOnlyMultiSelect {
		lb.keyInputNav(kt)
		if kt.IsHandled() {
			return
		}
	}
	selMode := kt.SelectMode()
	if lb.SelectMode {
		selMode = events.ExtendOne
	}
	kf := keymap.Of(kt.KeyChord())
	if DebugSettings.KeyEventTrace {
		slog.Info("ListBase ReadOnly KeyInput", "widget", lb, "keyFunction", kf)
	}
	idx := lb.SelectedIndex
	switch {
	case kf == keymap.MoveDown:
		ni := idx + 1
		if ni < lb.SliceSize {
			lb.ScrollToIndex(ni)
			lb.updateSelectIndex(ni, true, selMode)
			kt.SetHandled()
		}
	case kf == keymap.MoveUp:
		ni := idx - 1
		if ni >= 0 {
			lb.ScrollToIndex(ni)
			lb.updateSelectIndex(ni, true, selMode)
			kt.SetHandled()
		}
	case kf == keymap.PageDown:
		ni := min(idx+lb.VisibleRows-1, lb.SliceSize-1)
		lb.ScrollToIndex(ni)
		lb.updateSelectIndex(ni, true, selMode)
		kt.SetHandled()
	case kf == keymap.PageUp:
		ni := max(idx-(lb.VisibleRows-1), 0)
		lb.ScrollToIndex(ni)
		lb.updateSelectIndex(ni, true, selMode)
		kt.SetHandled()
	case kf == keymap.Enter || kf == keymap.Accept || kt.KeyRune() == ' ':
		lb.Send(events.DoubleClick, kt)
		kt.SetHandled()
	}
}

func (lb *ListBase) SizeFinal() {
	sg := lb.ListGrid
	if sg == nil {
		lb.Frame.SizeFinal()
		return
	}
	localIter := 0
	for (lb.MakeIter < 2 || lb.VisibleRows != sg.visibleRows) && localIter < 2 {
		if lb.VisibleRows != sg.visibleRows {
			lb.VisibleRows = sg.visibleRows
			lb.Update()
		} else {
			sg.StyleTree()
		}
		sg.sizeFinalUpdateChildrenSizes()
		lb.MakeIter++
		localIter++
	}
	lb.Frame.SizeFinal()
}

// ListGrid handles the resizing logic for all [Lister]s.
type ListGrid struct { //core:no-new
	Frame

	// minRows is set from parent [List]
	minRows int

	// height of a single row, computed during layout
	rowHeight float32

	// total number of rows visible in allocated display size
	visibleRows int

	// Various computed backgrounds
	bgStripe, bgSelect, bgSelectStripe, bgHover, bgHoverStripe, bgHoverSelect, bgHoverSelectStripe image.Image

	// lastBackground is the background for which modified
	// backgrounds were computed -- don't update if same
	lastBackground image.Image
}

func (lg *ListGrid) Init() {
	lg.Frame.Init()
	lg.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
	})
}

func (lg *ListGrid) SizeFromChildren(iter int, pass LayoutPasses) math32.Vector2 {
	csz := lg.Frame.SizeFromChildren(iter, pass)
	rht, err := lg.layout.rowHeight(0, 0)
	if err != nil {
		// fmt.Println("ListGrid Sizing Error:", err)
		lg.rowHeight = 42
	}
	if lg.NeedsRebuild() { // rebuilding = reset
		lg.rowHeight = rht
	} else {
		lg.rowHeight = max(lg.rowHeight, rht)
	}
	if lg.rowHeight == 0 {
		// fmt.Println("ListGrid Sizing Error: RowHeight should not be 0!", sg)
		lg.rowHeight = 42
	}
	allocHt := lg.Geom.Size.Alloc.Content.Y - lg.Geom.Size.InnerSpace.Y
	if allocHt > lg.rowHeight {
		lg.visibleRows = int(math32.Floor(allocHt / lg.rowHeight))
	}
	lg.visibleRows = max(lg.visibleRows, lg.minRows)
	minHt := lg.rowHeight * float32(lg.minRows)
	// fmt.Println("VisRows:", sg.VisRows, "rh:", sg.RowHeight, "ht:", minHt)
	// visHt := sg.RowHeight * float32(sg.VisRows)
	csz.Y = minHt
	return csz
}

func (lg *ListGrid) SetScrollParams(d math32.Dims, sb *Slider) {
	if d == math32.X {
		lg.Frame.SetScrollParams(d, sb)
		return
	}
	sb.Min = 0
	sb.Step = 1
	if lg.visibleRows > 0 {
		sb.PageStep = float32(lg.visibleRows)
	} else {
		sb.PageStep = 10
	}
	sb.InputThreshold = sb.Step
}

func (lg *ListGrid) list() (Lister, *ListBase) {
	ls := tree.ParentByType[Lister](lg)
	if ls == nil {
		return nil, nil
	}
	return ls, ls.AsListBase()
}

func (lg *ListGrid) ScrollChanged(d math32.Dims, sb *Slider) {
	if d == math32.X {
		lg.Frame.ScrollChanged(d, sb)
		return
	}
	_, sv := lg.list()
	if sv == nil {
		return
	}
	sv.StartIndex = int(math32.Round(sb.Value))
	sv.Update()
}

func (lg *ListGrid) ScrollValues(d math32.Dims) (maxSize, visSize, visPct float32) {
	if d == math32.X {
		return lg.Frame.ScrollValues(d)
	}
	_, sv := lg.list()
	if sv == nil {
		return
	}
	maxSize = float32(max(sv.SliceSize, 1))
	visSize = float32(lg.visibleRows)
	visPct = visSize / maxSize
	return
}

func (lg *ListGrid) updateScroll(idx int) {
	if !lg.HasScroll[math32.Y] || lg.scrolls[math32.Y] == nil {
		return
	}
	sb := lg.scrolls[math32.Y]
	sb.SetValue(float32(idx))
}

func (lg *ListGrid) updateBackgrounds() {
	bg := lg.Styles.ActualBackground
	if lg.lastBackground == bg {
		return
	}
	lg.lastBackground = bg

	// we take our zebra intensity applied foreground color and then overlay it onto our background color

	zclr := colors.WithAF32(colors.ToUniform(lg.Styles.Color), AppearanceSettings.ZebraStripesWeight())
	lg.bgStripe = gradient.Apply(bg, func(c color.Color) color.Color {
		return colors.AlphaBlend(c, zclr)
	})

	hclr := colors.WithAF32(colors.ToUniform(lg.Styles.Color), 0.08)
	lg.bgHover = gradient.Apply(bg, func(c color.Color) color.Color {
		return colors.AlphaBlend(c, hclr)
	})

	zhclr := colors.WithAF32(colors.ToUniform(lg.Styles.Color), AppearanceSettings.ZebraStripesWeight()+0.08)
	lg.bgHoverStripe = gradient.Apply(bg, func(c color.Color) color.Color {
		return colors.AlphaBlend(c, zhclr)
	})

	lg.bgSelect = colors.Scheme.Select.Container

	lg.bgSelectStripe = colors.Uniform(colors.AlphaBlend(colors.ToUniform(colors.Scheme.Select.Container), zclr))

	lg.bgHoverSelect = colors.Uniform(colors.AlphaBlend(colors.ToUniform(colors.Scheme.Select.Container), hclr))

	lg.bgHoverSelectStripe = colors.Uniform(colors.AlphaBlend(colors.ToUniform(colors.Scheme.Select.Container), zhclr))

}

func (lg *ListGrid) rowBackground(sel, stripe, hover bool) image.Image {
	switch {
	case sel && stripe && hover:
		return lg.bgHoverSelectStripe
	case sel && stripe:
		return lg.bgSelectStripe
	case sel && hover:
		return lg.bgHoverSelect
	case sel:
		return lg.bgSelect
	case stripe && hover:
		return lg.bgHoverStripe
	case stripe:
		return lg.bgStripe
	case hover:
		return lg.bgHover
	default:
		return lg.Styles.ActualBackground
	}
}

func (lg *ListGrid) ChildBackground(child Widget) image.Image {
	bg := lg.Styles.ActualBackground
	_, sv := lg.list()
	if sv == nil {
		return bg
	}
	lg.updateBackgrounds()
	row, _ := sv.widgetIndex(child)
	si := row + sv.StartIndex
	return lg.rowBackground(sv.indexIsSelected(si), si%2 == 1, row == sv.hoverRow)
}

func (lg *ListGrid) renderStripes() {
	pos := lg.Geom.Pos.Content
	sz := lg.Geom.Size.Actual.Content
	if lg.visibleRows == 0 || sz.Y == 0 {
		return
	}
	lg.updateBackgrounds()

	pc := &lg.Scene.PaintContext
	rows := lg.layout.Shape.Y
	cols := lg.layout.Shape.X
	st := pos
	offset := 0
	_, sv := lg.list()
	startIndex := 0
	if sv != nil {
		startIndex = sv.StartIndex
		offset = startIndex % 2
	}
	for r := 0; r < rows; r++ {
		si := r + startIndex
		ht, _ := lg.layout.rowHeight(r, 0)
		miny := st.Y
		for c := 0; c < cols; c++ {
			ki := r*cols + c
			if ki < lg.NumChildren() {
				kw := lg.Child(ki).(Widget).AsWidget()
				pyi := math32.Floor(kw.Geom.Pos.Total.Y)
				if pyi < miny {
					miny = pyi
				}
			}
		}
		st.Y = miny
		ssz := sz
		ssz.Y = ht
		stripe := (r+offset)%2 == 1
		sbg := lg.rowBackground(sv.indexIsSelected(si), stripe, r == sv.hoverRow)
		pc.BlitBox(st, ssz, sbg)
		st.Y += ht + lg.layout.Gap.Y
	}
}

// mousePosInGrid returns true if the event mouse position is
// located within the slicegrid.
func (lg *ListGrid) mousePosInGrid(pt image.Point) bool {
	ptrel := lg.PointToRelPos(pt)
	sz := lg.Geom.ContentBBox.Size()
	if lg.visibleRows == 0 || sz.Y == 0 {
		return false
	}
	if ptrel.Y < 0 || ptrel.Y >= sz.Y || ptrel.X < 0 || ptrel.X >= sz.X-50 { // leave margin on rhs around scroll
		return false
	}
	return true
}

// indexFromPixel returns the row, column indexes of given pixel point within grid.
// Takes a scene-level position.
func (lg *ListGrid) indexFromPixel(pt image.Point) (row, col int, isValid bool) {
	if !lg.mousePosInGrid(pt) {
		return
	}
	ptf := math32.Vector2FromPoint(lg.PointToRelPos(pt))
	sz := math32.Vector2FromPoint(lg.Geom.ContentBBox.Size())
	isValid = true
	rows := lg.layout.Shape.Y
	cols := lg.layout.Shape.X
	st := math32.Vector2{}
	got := false
	for r := 0; r < rows; r++ {
		ht, _ := lg.layout.rowHeight(r, 0)
		ht += lg.layout.Gap.Y
		miny := st.Y
		if r > 0 {
			for c := 0; c < cols; c++ {
				kw := lg.Child(r*cols + c).(Widget).AsWidget()
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

func (lg *ListGrid) Render() {
	lg.WidgetBase.Render()
	lg.renderStripes()
}
