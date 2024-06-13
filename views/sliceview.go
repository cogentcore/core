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

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/core"
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

	// StyleFunc is an optional styling function.
	StyleFunc ListStyleFunc `copier:"-" view:"-" json:"-" xml:"-"`
}

// ListStyleFunc is a styling function for custom styling and
// configuration of elements in the list.
type ListStyleFunc func(w core.Widget, s *styles.Style, row int)

func (ls *List) HasStyleFunc() bool {
	return ls.StyleFunc != nil
}

func (ls *List) StyleRow(w core.Widget, idx, fidx int) {
	if ls.StyleFunc != nil {
		ls.StyleFunc(w, &w.AsWidget().Styles, idx)
	}
}

////////////////////////////////////////////////////////
//  ListBase

// note on implementation:
// * ListGrid handles all the layout logic to start with a minimum number of
//   rows and then computes the total number visible based on allocated size.

const (
	ListRowProperty = "sv-row"
	ListColProperty = "sv-col"
)

// Lister is the interface used by [ListBase] to
// support any abstractions needed for different types of lists.
type Lister interface {
	// AsListBase returns the base for direct access to relevant fields etc
	AsListBase() *ListBase

	// SliceGrid returns the ListGrid grid Layout widget,
	// which contains all the fields and values
	SliceGrid() *ListGrid

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
	// tensorview table.IndexView), and a bool that is true if the
	// index is beyond the available data and is thus invisible,
	// given the row index provided.
	SliceIndex(i int) (si, vi int, invis bool)

	// MakeRow adds config for one row at given widget row index.
	// Plan must be the StructGrid Plan.
	MakeRow(p *core.Plan, i int)

	// StyleValue performs additional value widget styling
	StyleValue(w core.Widget, s *styles.Style, row, col int)

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

// ListBase is the base for [List] and [Table] and any other displays
// of array-like data. It automatically computes the number of rows that fit
// within its allocated space, and manages the offset view window into the full
// list of items, and supports row selection, copy / paste, Drag-n-Drop, etc.
// Use [ListBase.BindSelect] to make the list designed for item selection.
type ListBase struct {
	core.Frame

	// Slice is the pointer to the slice that we are viewing.
	Slice any `set:"-"`

	// ShowIndexes is whether to show the indexes of rows or not (default false).
	ShowIndexes bool

	// MinRows specifies the minimum number of rows to display, to ensure
	// at least this amount is displayed.
	MinRows int `default:"4"`

	// SelectedValue is the current selection value; initially select this value if set.
	SelectedValue any `copier:"-" view:"-" json:"-" xml:"-"`

	// index of currently selected item
	SelectedIndex int `copier:"-" json:"-" xml:"-"`

	// index of row to select at start
	InitSelectedIndex int `copier:"-" json:"-" xml:"-"`

	// list of currently selected slice indexes
	SelectedIndexes map[int]struct{} `set:"-" copier:"-"`

	// lastClick is the last row that has been clicked on.
	// This is used to prevent erroneous double click events
	// from being sent when the user clicks on multiple different
	// rows in quick succession.
	lastClick int

	// NormalCursor is the cached cursor to display when there
	// is no row being hovered.
	NormalCursor cursors.Cursor `copier:"-" xml:"-" json:"-" set:"-"`

	// CurrentCursor is the cached cursor that should currently be
	// displayed.
	CurrentCursor cursors.Cursor `copier:"-" xml:"-" json:"-" set:"-"`

	// SliceUnderlying is the underlying slice value.
	SliceUnderlying reflect.Value `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// currently hovered row
	hoverRow int

	// list of currently dragged indexes
	DraggedIndexes []int `set:"-" view:"-" copier:"-" json:"-" xml:"-"`

	// total number of rows visible in allocated display size
	VisRows int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// starting slice index of visible rows
	StartIndex int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// size of slice
	SliceSize int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// iteration through the configuration process, reset when a new slice type is set
	MakeIter int `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// temp idx state for e.g., dnd
	tmpIndex int

	// ElementValue is a [reflect.Value] representation of the underlying element type
	// which is used whenever there are no slice elements available
	ElementValue reflect.Value `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// maximum width of value column in chars, if string
	maxWidth int

	// ReadOnlyKeyNav is whether support key navigation when ReadOnly (default true).
	// It uses a capture of up / down events to manipulate selection, not focus.
	ReadOnlyKeyNav bool

	// SelectMode is whether to be in select rows mode or editing mode.
	SelectMode bool `set:"-" copier:"-" json:"-" xml:"-"`

	// ReadOnlyMultiSelect: if view is ReadOnly, default selection mode is to choose one row only.
	// If this is true, standard multiple selection logic with modifier keys is instead supported.
	ReadOnlyMultiSelect bool

	// InFocusGrab is a guard for recursive focus grabbing.
	InFocusGrab bool `set:"-" edit:"-" copier:"-" json:"-" xml:"-"`

	// isArray is whether the slice is actually an array.
	isArray bool
}

func (lb *ListBase) WidgetValue() any { return &lb.Slice }

func (lb *ListBase) Init() {
	lb.Frame.Init()
	lb.AddContextMenu(lb.ContextMenu)
	lb.InitSelectedIndex = -1
	lb.hoverRow = -1
	lb.MinRows = 4
	lb.ReadOnlyKeyNav = true
	svi := lb.This.(Lister)

	lb.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable, abilities.DoubleClickable, abilities.TripleClickable)
		s.SetAbilities(!lb.IsReadOnly(), abilities.Draggable, abilities.Droppable)
		s.Cursor = lb.CurrentCursor
		s.Direction = styles.Column
		// absorb horizontal here, vertical in view
		s.Overflow.X = styles.OverflowAuto
		s.Grow.Set(1, 1)
	})
	if !lb.IsReadOnly() {
		lb.On(events.DragStart, func(e events.Event) {
			svi.DragStart(e)
		})
		lb.On(events.DragEnter, func(e events.Event) {
			e.SetHandled()
		})
		lb.On(events.DragLeave, func(e events.Event) {
			e.SetHandled()
		})
		lb.On(events.Drop, func(e events.Event) {
			svi.DragDrop(e)
		})
		lb.On(events.DropDeleteSource, func(e events.Event) {
			svi.DropDeleteSource(e)
		})
	}
	lb.FinalStyler(func(s *styles.Style) {
		lb.NormalCursor = s.Cursor
	})

	lb.OnFinal(events.KeyChord, func(e events.Event) {
		if lb.IsReadOnly() {
			if lb.ReadOnlyKeyNav {
				lb.KeyInputReadOnly(e)
			}
		} else {
			lb.KeyInputEditable(e)
		}
	})
	lb.On(events.MouseMove, func(e events.Event) {
		row, _, isValid := lb.RowFromEventPos(e)
		prevHoverRow := lb.hoverRow
		if !isValid {
			lb.hoverRow = -1
			lb.Styles.Cursor = lb.NormalCursor
		} else {
			lb.hoverRow = row
			lb.Styles.Cursor = cursors.Pointer
		}
		lb.CurrentCursor = lb.Styles.Cursor
		if lb.hoverRow != prevHoverRow {
			lb.NeedsRender()
		}
	})
	lb.On(events.MouseDrag, func(e events.Event) {
		row, idx, isValid := lb.RowFromEventPos(e)
		if !isValid {
			return
		}
		lb.This.(Lister).SliceGrid().AutoScroll(math32.Vec2(0, float32(idx)))
		prevHoverRow := lb.hoverRow
		if !isValid {
			lb.hoverRow = -1
			lb.Styles.Cursor = lb.NormalCursor
		} else {
			lb.hoverRow = row
			lb.Styles.Cursor = cursors.Pointer
		}
		lb.CurrentCursor = lb.Styles.Cursor
		if lb.hoverRow != prevHoverRow {
			lb.NeedsRender()
		}
	})
	lb.OnFirst(events.DoubleClick, func(e events.Event) {
		row, _, isValid := lb.RowFromEventPos(e)
		if !isValid {
			return
		}
		if lb.lastClick != row+lb.StartIndex {
			lb.This.(Lister).SliceGrid().Send(events.Click, e)
			e.SetHandled()
		}
	})
	// we must interpret triple click events as double click
	// events for rapid cross-row double clicking to work correctly
	lb.OnFirst(events.TripleClick, func(e events.Event) {
		lb.Send(events.DoubleClick, e)
	})

	lb.Maker(func(p *core.Plan) {
		svi := lb.This.(Lister)
		svi.UpdateSliceSize()

		scrollTo := -1
		if lb.SelectedValue != nil {
			idx, ok := SliceIndexByValue(lb.Slice, lb.SelectedValue)
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

		lb.MakeGrid(p, func(p *core.Plan) {
			for i := 0; i < lb.VisRows; i++ {
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
func (lb *ListBase) StyleValue(w core.Widget, s *styles.Style, row, col int) {
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
	lb.VisRows = lb.MinRows
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
	lb.SliceUnderlying = reflectx.Underlying(reflect.ValueOf(lb.Slice))
	lb.isArray = reflectx.NonPointerType(reflect.TypeOf(sl)).Kind() == reflect.Array
	lb.ElementValue = reflectx.SliceElementValue(sl)
	return lb
}

// IsNil returns true if the Slice is nil
func (lb *ListBase) IsNil() bool {
	return reflectx.AnyIsNil(lb.Slice)
}

// RowFromEventPos returns the widget row, slice index, and
// whether the index is in slice range, for given event position.
func (lb *ListBase) RowFromEventPos(e events.Event) (row, idx int, isValid bool) {
	sg := lb.This.(Lister).SliceGrid()
	row, _, isValid = sg.IndexFromPixel(e.Pos())
	if !isValid {
		return
	}
	idx = row + lb.StartIndex
	if row < 0 || idx >= lb.SliceSize {
		isValid = false
	}
	return
}

// ClickSelectEvent is a helper for processing selection events
// based on a mouse click, which could be a double or triple
// in addition to a regular click.
// Returns false if no further processing should occur,
// because the user clicked outside the range of active rows.
func (lb *ListBase) ClickSelectEvent(e events.Event) bool {
	row, _, isValid := lb.RowFromEventPos(e)
	if !isValid {
		e.SetHandled()
	} else {
		lb.UpdateSelectRow(row, e.SelectMode())
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
		if lb.ClickSelectEvent(e) {
			*val = lb.SelectedIndex
			lb.Scene.SendKey(keymap.Accept, e) // activate OK button
			if lb.Scene.Stage.Type == core.DialogStage {
				lb.Scene.Close() // also directly close dialog for value dialogs without OK button
			}
		}
	})
	return lb
}

func (lb *ListBase) UpdateMaxWidths() {
	lb.maxWidth = 0
	npv := reflectx.NonPointerValue(lb.ElementValue)
	isString := npv.Type().Kind() == reflect.String
	if !isString || lb.SliceSize == 0 {
		return
	}
	mxw := 0
	for rw := 0; rw < lb.SliceSize; rw++ {
		str := reflectx.ToString(lb.SliceElementValue(rw).Interface())
		mxw = max(mxw, len(str))
	}
	lb.maxWidth = mxw
}

// SliceElementValue returns an underlying non-pointer [reflect.Value]
// of slice element at given index or ElementValue if out of range.
func (lb *ListBase) SliceElementValue(si int) reflect.Value {
	var val reflect.Value
	if si < lb.SliceSize {
		val = reflectx.Underlying(lb.SliceUnderlying.Index(si)) // deal with pointer lists
	} else {
		val = lb.ElementValue
	}
	if val.IsZero() {
		val = lb.ElementValue
	}
	return val
}

func (lb *ListBase) MakeGrid(p *core.Plan, maker func(p *core.Plan)) {
	core.AddAt(p, "grid", func(w *ListGrid) {
		w.Styler(func(s *styles.Style) {
			nWidgPerRow, _ := lb.This.(Lister).RowWidgetNs()
			w.MinRows = lb.MinRows
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
			row, _, isValid := w.IndexFromPixel(e.Pos())
			if isValid {
				lb.UpdateSelectRow(row, e.SelectMode())
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

func (lb *ListBase) MakeValue(w core.Value, i int) {
	svi := lb.This.(Lister)
	wb := w.AsWidget()
	wb.SetProperty(ListRowProperty, i)
	wb.Styler(func(s *styles.Style) {
		if lb.IsReadOnly() {
			s.SetAbilities(true, abilities.DoubleClickable)
			s.SetAbilities(false, abilities.Hoverable, abilities.Focusable, abilities.Activatable, abilities.TripleClickable)
			s.SetReadOnly(true)
		}
		row, col := lb.WidgetIndex(w)
		row += lb.StartIndex
		svi.StyleValue(w, s, row, col)
		if row < lb.SliceSize {
			svi.StyleRow(w, row, col)
		}
	})
	wb.OnSelect(func(e events.Event) {
		e.SetHandled()
		row, _ := lb.WidgetIndex(w)
		lb.UpdateSelectRow(row, e.SelectMode())
		lb.lastClick = row + lb.StartIndex
	})
	wb.OnDoubleClick(lb.HandleEvent)
	wb.On(events.ContextMenu, lb.HandleEvent)
	if !lb.IsReadOnly() {
		wb.OnInput(lb.HandleEvent)
	}
}

func (lb *ListBase) MakeRow(p *core.Plan, i int) {
	svi := lb.This.(Lister)
	si, vi, invis := svi.SliceIndex(i)
	itxt := strconv.Itoa(i)
	val := lb.SliceElementValue(vi)

	if lb.ShowIndexes {
		lb.MakeGridIndex(p, i, si, itxt, invis)
	}

	valnm := fmt.Sprintf("value-%s-%s", itxt, reflectx.ShortTypeName(lb.ElementValue.Type()))
	core.AddNew(p, valnm, func() core.Value {
		return core.NewValue(val.Addr().Interface(), "")
	}, func(w core.Value) {
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
			val := lb.SliceElementValue(vi)
			core.Bind(val.Addr().Interface(), w)
			wb.SetReadOnly(lb.IsReadOnly())
			wb.SetState(invis, states.Invisible)
			if lb.This.(Lister).HasStyleFunc() {
				w.Style()
			}
			if invis {
				wb.SetSelected(false)
			}
		})
	})

}

func (lb *ListBase) MakeGridIndex(p *core.Plan, i, si int, itxt string, invis bool) {
	svi := lb.This.(Lister)
	core.AddAt(p, "index-"+itxt, func(w *core.Text) {
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
			lb.UpdateSelectRow(i, e.SelectMode())
			lb.lastClick = si
		})
		w.OnDoubleClick(lb.HandleEvent)
		w.On(events.ContextMenu, lb.HandleEvent)
		if !lb.IsReadOnly() {
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
		w.Updater(func() {
			si, _, invis := svi.SliceIndex(i)
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

// SliceGrid returns the SliceGrid grid frame widget, which contains all the
// fields and values
func (lb *ListBase) SliceGrid() *ListGrid {
	sg := lb.ChildByName("grid", 0)
	if sg == nil {
		return nil
	}
	return sg.(*ListGrid)
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
	sz := lb.SliceUnderlying.Len()
	lb.SliceSize = sz
	return sz
}

// WidgetIndex returns the row and column indexes for given widget,
// from the properties set during construction.
func (lb *ListBase) WidgetIndex(w core.Widget) (row, col int) {
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
	if sz > lb.VisRows {
		lastSt := sz - lb.VisRows
		lb.StartIndex = min(lastSt, lb.StartIndex)
		lb.StartIndex = max(0, lb.StartIndex)
	} else {
		lb.StartIndex = 0
	}
}

// UpdateScroll updates the scroll value
func (lb *ListBase) UpdateScroll() {
	sg := lb.This.(Lister).SliceGrid()
	if sg == nil {
		return
	}
	sg.UpdateScroll(lb.StartIndex)
}

// SliceNewAtRow inserts a new blank element at given display row
func (lb *ListBase) SliceNewAtRow(row int) {
	lb.This.(Lister).SliceNewAt(lb.StartIndex + row)
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (lb *ListBase) SliceNewAt(idx int) {
	if lb.isArray {
		return
	}

	lb.SliceNewAtSelect(idx)

	sltyp := reflectx.SliceElementType(lb.Slice) // has pointer if it is there
	slptr := sltyp.Kind() == reflect.Pointer
	sz := lb.SliceSize
	svnp := lb.SliceUnderlying

	nval := reflect.New(reflectx.NonPointerType(sltyp)) // make the concrete el
	if !slptr {
		nval = nval.Elem() // use concrete value
	}
	svnp = reflect.Append(svnp, nval)
	if idx >= 0 && idx < sz {
		reflect.Copy(svnp.Slice(idx+1, sz+1), svnp.Slice(idx, sz))
		svnp.Index(idx).Set(nval)
	}
	svnp.Set(svnp)
	if idx < 0 {
		idx = sz
	}

	lb.SliceUnderlying = reflectx.NonPointerValue(reflect.ValueOf(lb.Slice)) // need to update after changes

	lb.This.(Lister).UpdateSliceSize()

	lb.SelectIndexAction(idx, events.SelectOne)
	lb.SendChange()
	lb.Update()
	lb.IndexGrabFocus(idx)
}

// SliceDeleteAtRow deletes element at given display row
// if update is true, then update the grid after
func (lb *ListBase) SliceDeleteAtRow(row int) {
	lb.This.(Lister).SliceDeleteAt(lb.StartIndex + row)
}

// SliceNewAtSelect updates selected rows based on
// inserting new element at given index.
// must be called with successful SliceNewAt
func (lb *ListBase) SliceNewAtSelect(i int) {
	sl := lb.SelectedIndexesList(false) // ascending
	lb.ResetSelectedIndexes()
	for _, ix := range sl {
		if ix >= i {
			ix++
		}
		lb.SelectedIndexes[ix] = struct{}{}
	}
}

// SliceDeleteAtSelect updates selected rows based on
// deleting element at given index
// must be called with successful SliceDeleteAt
func (lb *ListBase) SliceDeleteAtSelect(i int) {
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

// SliceDeleteAt deletes element at given index from slice
func (lb *ListBase) SliceDeleteAt(i int) {
	if lb.isArray {
		return
	}
	if i < 0 || i >= lb.SliceSize {
		return
	}

	lb.SliceDeleteAtSelect(i)

	reflectx.SliceDeleteAt(lb.Slice, i)

	lb.This.(Lister).UpdateSliceSize()

	lb.SendChange()
	lb.Update()
}

// MakeToolbar configures a [core.Toolbar] for this view
func (lb *ListBase) MakeToolbar(p *core.Plan) {
	if reflectx.AnyIsNil(lb.Slice) {
		return
	}
	if lb.isArray || lb.IsReadOnly() {
		return
	}
	core.Add(p, func(w *core.Button) {
		w.SetText("Add").SetIcon(icons.Add).SetTooltip("add a new element to the slice").
			OnClick(func(e events.Event) {
				lb.This.(Lister).SliceNewAt(-1)
			})
	})
}

////////////////////////////////////////////////////////////
//  Row access methods
//  NOTE: row = physical GUI display row, idx = slice index
//  not the same!

// SliceValue returns value interface at given slice index.
func (lb *ListBase) SliceValue(idx int) any {
	if idx < 0 || idx >= lb.SliceSize {
		fmt.Printf("views.ListBase: slice index out of range: %v\n", idx)
		return nil
	}
	val := reflectx.UnderlyingPointer(lb.SliceUnderlying.Index(idx)) // deal with pointer lists
	vali := val.Interface()
	return vali
}

// IsRowInBounds returns true if disp row is in bounds
func (lb *ListBase) IsRowInBounds(row int) bool {
	return row >= 0 && row < lb.VisRows
}

// IsIndexVisible returns true if slice index is currently visible
func (lb *ListBase) IsIndexVisible(idx int) bool {
	return lb.IsRowInBounds(idx - lb.StartIndex)
}

// RowFirstWidget returns the first widget for given row (could be index or
// not) -- false if out of range
func (lb *ListBase) RowFirstWidget(row int) (*core.WidgetBase, bool) {
	if !lb.ShowIndexes {
		return nil, false
	}
	if !lb.IsRowInBounds(row) {
		return nil, false
	}
	nWidgPerRow, _ := lb.This.(Lister).RowWidgetNs()
	sg := lb.This.(Lister).SliceGrid()
	w := sg.Children[row*nWidgPerRow].(core.Widget).AsWidget()
	return w, true
}

// RowGrabFocus grabs the focus for the first focusable widget
// in given row.  returns that element or nil if not successful
// note: grid must have already rendered for focus to be grabbed!
func (lb *ListBase) RowGrabFocus(row int) *core.WidgetBase {
	if !lb.IsRowInBounds(row) || lb.InFocusGrab { // range check
		return nil
	}
	nWidgPerRow, idxOff := lb.This.(Lister).RowWidgetNs()
	ridx := nWidgPerRow * row
	sg := lb.This.(Lister).SliceGrid()
	w := sg.Child(ridx + idxOff).(core.Widget).AsWidget()
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
func (lb *ListBase) IndexGrabFocus(idx int) *core.WidgetBase {
	lb.ScrollToIndex(idx)
	return lb.This.(Lister).RowGrabFocus(idx - lb.StartIndex)
}

// IndexPos returns center of window position of index label for idx (ContextMenuPos)
func (lb *ListBase) IndexPos(idx int) image.Point {
	row := idx - lb.StartIndex
	if row < 0 {
		row = 0
	}
	if row > lb.VisRows-1 {
		row = lb.VisRows - 1
	}
	var pos image.Point
	w, ok := lb.This.(Lister).RowFirstWidget(row)
	if ok {
		pos = w.ContextMenuPos(nil)
	}
	return pos
}

// RowFromPos returns the row that contains given vertical position, false if not found
func (lb *ListBase) RowFromPos(posY int) (int, bool) {
	// todo: could optimize search to approx loc, and search up / down from there
	for rw := 0; rw < lb.VisRows; rw++ {
		w, ok := lb.This.(Lister).RowFirstWidget(rw)
		if ok {
			if w.Geom.TotalBBox.Min.Y < posY && posY < w.Geom.TotalBBox.Max.Y {
				return rw, true
			}
		}
	}
	return -1, false
}

// IndexFromPos returns the idx that contains given vertical position, false if not found
func (lb *ListBase) IndexFromPos(posY int) (int, bool) {
	row, ok := lb.RowFromPos(posY)
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
	if lb.VisRows == 0 {
		return false
	}
	if idx < lb.StartIndex {
		lb.StartIndex = idx
		lb.StartIndex = max(0, lb.StartIndex)
		lb.UpdateScroll()
		return true
	}
	if idx >= lb.StartIndex+lb.VisRows {
		lb.StartIndex = idx - (lb.VisRows - 4)
		lb.StartIndex = max(0, lb.StartIndex)
		lb.UpdateScroll()
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

// SelectValue sets SelVal and attempts to find corresponding row, setting
// SelectedIndex and selecting row if found -- returns true if found, false
// otherwise.
func (lb *ListBase) SelectValue(val string) bool {
	lb.SelectedValue = val
	if lb.SelectedValue != nil {
		idx, _ := SliceIndexByValue(lb.Slice, lb.SelectedValue)
		if idx >= 0 {
			lb.UpdateSelectIndex(idx, true, events.SelectOne)
			lb.ScrollToIndex(idx)
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
func (lb *ListBase) MoveDown(selMode events.SelectModes) int {
	if lb.SelectedIndex >= lb.SliceSize-1 {
		lb.SelectedIndex = lb.SliceSize - 1
		return -1
	}
	lb.SelectedIndex++
	lb.SelectIndexAction(lb.SelectedIndex, selMode)
	return lb.SelectedIndex
}

// MoveDownAction moves the selection down to next row, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected
// row
func (lb *ListBase) MoveDownAction(selMode events.SelectModes) int {
	nidx := lb.MoveDown(selMode)
	if nidx >= 0 {
		lb.ScrollToIndex(nidx)
		lb.Send(events.Select) // todo: need to do this for the item?
	}
	return nidx
}

// MoveUp moves the selection up to previous idx, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (lb *ListBase) MoveUp(selMode events.SelectModes) int {
	if lb.SelectedIndex <= 0 {
		lb.SelectedIndex = 0
		return -1
	}
	lb.SelectedIndex--
	lb.SelectIndexAction(lb.SelectedIndex, selMode)
	return lb.SelectedIndex
}

// MoveUpAction moves the selection up to previous idx, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (lb *ListBase) MoveUpAction(selMode events.SelectModes) int {
	nidx := lb.MoveUp(selMode)
	if nidx >= 0 {
		lb.ScrollToIndex(nidx)
		lb.Send(events.Select)
	}
	return nidx
}

// MovePageDown moves the selection down to next page, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (lb *ListBase) MovePageDown(selMode events.SelectModes) int {
	if lb.SelectedIndex >= lb.SliceSize-1 {
		lb.SelectedIndex = lb.SliceSize - 1
		return -1
	}
	lb.SelectedIndex += lb.VisRows
	lb.SelectedIndex = min(lb.SelectedIndex, lb.SliceSize-1)
	lb.SelectIndexAction(lb.SelectedIndex, selMode)
	return lb.SelectedIndex
}

// MovePageDownAction moves the selection down to next page, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (lb *ListBase) MovePageDownAction(selMode events.SelectModes) int {
	nidx := lb.MovePageDown(selMode)
	if nidx >= 0 {
		lb.ScrollToIndex(nidx)
		lb.Send(events.Select)
	}
	return nidx
}

// MovePageUp moves the selection up to previous page, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (lb *ListBase) MovePageUp(selMode events.SelectModes) int {
	if lb.SelectedIndex <= 0 {
		lb.SelectedIndex = 0
		return -1
	}
	lb.SelectedIndex -= lb.VisRows
	lb.SelectedIndex = max(0, lb.SelectedIndex)
	lb.SelectIndexAction(lb.SelectedIndex, selMode)
	return lb.SelectedIndex
}

// MovePageUpAction moves the selection up to previous page, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (lb *ListBase) MovePageUpAction(selMode events.SelectModes) int {
	nidx := lb.MovePageUp(selMode)
	if nidx >= 0 {
		lb.ScrollToIndex(nidx)
		lb.Send(events.Select)
	}
	return nidx
}

//////////////////////////////////////////////////////////
//    Selection: user operates on the index labels

// UpdateSelectRow updates the selection for the given row
func (lb *ListBase) UpdateSelectRow(row int, selMode events.SelectModes) {
	idx := row + lb.StartIndex
	if row < 0 || idx >= lb.SliceSize {
		return
	}
	sel := !lb.IndexIsSelected(idx)
	lb.UpdateSelectIndex(idx, sel, selMode)
}

// UpdateSelectIndex updates the selection for the given index
func (lb *ListBase) UpdateSelectIndex(idx int, sel bool, selMode events.SelectModes) {
	if lb.IsReadOnly() && !lb.ReadOnlyMultiSelect {
		lb.UnselectAllIndexes()
		if sel || lb.SelectedIndex == idx {
			lb.SelectedIndex = idx
			lb.SelectIndex(idx)
		}
		lb.Send(events.Select)
		lb.Restyle()
	} else {
		lb.SelectIndexAction(idx, selMode)
	}
}

// IndexIsSelected returns the selected status of given slice index
func (lb *ListBase) IndexIsSelected(idx int) bool {
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

// UnselectIndex unselects given idx (if selected)
func (lb *ListBase) UnselectIndex(idx int) {
	if lb.IndexIsSelected(idx) {
		delete(lb.SelectedIndexes, idx)
	}
}

// UnselectAllIndexes unselects all selected idxs
func (lb *ListBase) UnselectAllIndexes() {
	lb.ResetSelectedIndexes()
}

// SelectAllIndexes selects all idxs
func (lb *ListBase) SelectAllIndexes() {
	lb.UnselectAllIndexes()
	lb.SelectedIndexes = make(map[int]struct{}, lb.SliceSize)
	for idx := 0; idx < lb.SliceSize; idx++ {
		lb.SelectedIndexes[idx] = struct{}{}
	}
	lb.NeedsRender()
}

// SelectIndexAction is called when a select action has been received (e.g., a
// mouse click) -- translates into selection updates -- gets selection mode
// from mouse event (ExtendContinuous, ExtendOne)
func (lb *ListBase) SelectIndexAction(idx int, mode events.SelectModes) {
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
		if lb.IndexIsSelected(idx) {
			if len(lb.SelectedIndexes) > 1 {
				lb.UnselectAllIndexes()
			}
			lb.SelectedIndex = idx
			lb.SelectIndex(idx)
			lb.IndexGrabFocus(idx)
		} else {
			lb.UnselectAllIndexes()
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
					r := lb.MoveDown(events.SelectQuiet) // just select
					cidx = r
				}
			} else if idx > maxIndex {
				for cidx > maxIndex {
					r := lb.MoveUp(events.SelectQuiet) // just select
					cidx = r
				}
			}
			lb.IndexGrabFocus(idx)
			lb.Send(events.Select) //  sv.SelectedIndex)
		}
	case events.ExtendOne:
		if lb.IndexIsSelected(idx) {
			lb.UnselectIndexAction(idx)
			lb.Send(events.Select) //  sv.SelectedIndex)
		} else {
			lb.SelectedIndex = idx
			lb.SelectIndex(idx)
			lb.IndexGrabFocus(idx)
			lb.Send(events.Select) //  sv.SelectedIndex)
		}
	case events.Unselect:
		lb.SelectedIndex = idx
		lb.UnselectIndexAction(idx)
	case events.SelectQuiet:
		lb.SelectedIndex = idx
		lb.SelectIndex(idx)
	case events.UnselectQuiet:
		lb.SelectedIndex = idx
		lb.UnselectIndex(idx)
	}
	lb.Restyle()
}

// UnselectIndexAction unselects this idx (if selected) -- and emits a signal
func (lb *ListBase) UnselectIndexAction(idx int) {
	if lb.IndexIsSelected(idx) {
		lb.UnselectIndex(idx)
	}
}

///////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeDataIndex adds mimedata for given idx: an application/json of the struct
func (lb *ListBase) MimeDataIndex(md *mimedata.Mimes, idx int) {
	val := lb.SliceValue(idx)
	b, err := json.MarshalIndent(val, "", "  ")
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: fileinfo.DataJson, Data: b})
	} else {
		log.Printf("core.ListBase MimeData JSON Marshall error: %v\n", err)
	}
}

// FromMimeData creates a slice of structs from mime data
func (lb *ListBase) FromMimeData(md mimedata.Mimes) []any {
	svtyp := lb.SliceUnderlying.Type()
	sl := make([]any, 0, len(md))
	for _, d := range md {
		if d.Type == fileinfo.DataJson {
			nval := reflect.New(svtyp.Elem()).Interface()
			err := json.Unmarshal(d.Data, nval)
			if err == nil {
				sl = append(sl, nval)
			} else {
				log.Printf("core.ListBase FromMimeData: JSON load error: %v\n", err)
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
		lb.MimeDataIndex(&md, i)
	}
	return md
}

// CopyIndexes copies selected idxs to system.Clipboard, optionally resetting the selection
func (lb *ListBase) CopyIndexes(reset bool) { //types:add
	nitms := len(lb.SelectedIndexes)
	if nitms == 0 {
		return
	}
	md := lb.This.(Lister).CopySelectToMime()
	if md != nil {
		lb.Clipboard().Write(md)
	}
	if reset {
		lb.UnselectAllIndexes()
	}
}

// DeleteIndexes deletes all selected indexes
func (lb *ListBase) DeleteIndexes() { //types:add
	if len(lb.SelectedIndexes) == 0 {
		return
	}

	ixs := lb.SelectedIndexesList(true) // descending sort
	for _, i := range ixs {
		lb.This.(Lister).SliceDeleteAt(i)
	}
	lb.SendChange()
	lb.Update()
}

// CutIndexes copies selected indexes to system.Clipboard and deletes selected indexes
func (lb *ListBase) CutIndexes() { //types:add
	if len(lb.SelectedIndexes) == 0 {
		return
	}

	lb.CopyIndexes(false)
	ixs := lb.SelectedIndexesList(true) // descending sort
	idx := ixs[0]
	lb.UnselectAllIndexes()
	for _, i := range ixs {
		lb.This.(Lister).SliceDeleteAt(i)
	}
	lb.SendChange()
	lb.SelectIndexAction(idx, events.SelectOne)
	lb.Update()
}

// PasteIndex pastes clipboard at given idx
func (lb *ListBase) PasteIndex(idx int) { //types:add
	lb.tmpIndex = idx
	dt := lb.This.(Lister).MimeDataType()
	md := lb.Clipboard().Read([]string{dt})
	if md != nil {
		lb.PasteMenu(md, lb.tmpIndex)
	}
}

// MakePasteMenu makes the menu of options for paste events
func (lb *ListBase) MakePasteMenu(m *core.Scene, md mimedata.Mimes, idx int, mod events.DropMods, fun func()) {
	svi := lb.This.(Lister)
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
func (lb *ListBase) PasteMenu(md mimedata.Mimes, idx int) {
	lb.UnselectAllIndexes()
	mf := func(m *core.Scene) {
		lb.MakePasteMenu(m, md, idx, events.DropCopy, nil)
	}
	pos := lb.IndexPos(idx)
	core.NewMenu(mf, lb.This.(core.Widget), pos).Run()
}

// PasteAssign assigns mime data (only the first one!) to this idx
func (lb *ListBase) PasteAssign(md mimedata.Mimes, idx int) {
	sl := lb.FromMimeData(md)
	if len(sl) == 0 {
		return
	}
	ns := sl[0]
	lb.SliceUnderlying.Index(idx).Set(reflect.ValueOf(ns).Elem())
	lb.SendChange()
	lb.Update()
}

// PasteAtIndex inserts object(s) from mime data at (before) given slice index
func (lb *ListBase) PasteAtIndex(md mimedata.Mimes, idx int) {
	sl := lb.FromMimeData(md)
	if len(sl) == 0 {
		return
	}
	svl := reflect.ValueOf(lb.Slice)
	svnp := lb.SliceUnderlying

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

	lb.SliceUnderlying = reflectx.NonPointerValue(reflect.ValueOf(lb.Slice)) // need to update after changes

	lb.SendChange()
	lb.SelectIndexAction(idx, events.SelectOne)
	lb.Update()
}

// Duplicate copies selected items and inserts them after current selection --
// return idx of start of duplicates if successful, else -1
func (lb *ListBase) Duplicate() int { //types:add
	nitms := len(lb.SelectedIndexes)
	if nitms == 0 {
		return -1
	}
	ixs := lb.SelectedIndexesList(true) // descending sort -- last first
	pasteAt := ixs[0]
	lb.CopyIndexes(true)
	dt := lb.This.(Lister).MimeDataType()
	md := lb.Clipboard().Read([]string{dt})
	lb.This.(Lister).PasteAtIndex(md, pasteAt)
	return pasteAt
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// SelectRowIfNone selects the row the mouse is on if there
// are no currently selected items.  Returns false if no valid mouse row.
func (lb *ListBase) SelectRowIfNone(e events.Event) bool {
	nitms := len(lb.SelectedIndexes)
	if nitms > 0 {
		return true
	}
	row, _, isValid := lb.This.(Lister).SliceGrid().IndexFromPixel(e.Pos())
	if !isValid {
		return false
	}
	lb.UpdateSelectRow(row, e.SelectMode())
	return true
}

// MousePosInGrid returns true if the event mouse position is
// located within the slicegrid.
func (lb *ListBase) MousePosInGrid(e events.Event) bool {
	return lb.This.(Lister).SliceGrid().MousePosInGrid(e.Pos())
}

func (lb *ListBase) DragStart(e events.Event) {
	if !lb.SelectRowIfNone(e) || !lb.MousePosInGrid(e) {
		return
	}
	ixs := lb.SelectedIndexesList(false) // ascending
	if len(ixs) == 0 {
		return
	}
	md := lb.This.(Lister).CopySelectToMime()
	w, ok := lb.This.(Lister).RowFirstWidget(ixs[0] - lb.StartIndex)
	if ok {
		lb.Scene.Events.DragStart(w, md, e)
		e.SetHandled()
		// } else {
		// 	fmt.Println("List DND programmer error")
	}
}

func (lb *ListBase) DragDrop(e events.Event) {
	de := e.(*events.DragDrop)
	if de.Data == nil {
		return
	}
	svi := lb.This.(Lister)
	pos := de.Pos()
	idx, ok := lb.IndexFromPos(pos.Y)
	if ok {
		// sv.DraggedIndexes = nil
		lb.tmpIndex = idx
		lb.SaveDraggedIndexes(idx)
		md := de.Data.(mimedata.Mimes)
		mf := func(m *core.Scene) {
			lb.Scene.Events.DragMenuAddModText(m, de.DropMod)
			svi.MakePasteMenu(m, md, idx, de.DropMod, func() {
				svi.DropFinalize(de)
			})
		}
		pos := lb.IndexPos(lb.tmpIndex)
		core.NewMenu(mf, lb.This.(core.Widget), pos).Run()
	}
}

// DropFinalize is called to finalize Drop actions on the Source node.
// Only relevant for DropMod == DropMove.
func (lb *ListBase) DropFinalize(de *events.DragDrop) {
	lb.NeedsLayout()
	lb.UnselectAllIndexes()
	lb.Scene.Events.DropFinalize(de) // sends DropDeleteSource to Source
}

// DropDeleteSource handles delete source event for DropMove case
func (lb *ListBase) DropDeleteSource(e events.Event) {
	sort.Slice(lb.DraggedIndexes, func(i, j int) bool {
		return lb.DraggedIndexes[i] > lb.DraggedIndexes[j]
	})
	idx := lb.DraggedIndexes[0]
	for _, i := range lb.DraggedIndexes {
		lb.This.(Lister).SliceDeleteAt(i)
	}
	lb.DraggedIndexes = nil
	lb.SelectIndexAction(idx, events.SelectOne)
}

// SaveDraggedIndexes saves selectedindexes into dragged indexes
// taking into account insertion at idx
func (lb *ListBase) SaveDraggedIndexes(idx int) {
	sz := len(lb.SelectedIndexes)
	if sz == 0 {
		lb.DraggedIndexes = nil
		return
	}
	ixs := lb.SelectedIndexesList(false) // ascending
	lb.DraggedIndexes = make([]int, len(ixs))
	for i, ix := range ixs {
		if ix > idx {
			lb.DraggedIndexes[i] = ix + sz // make room for insertion
		} else {
			lb.DraggedIndexes[i] = ix
		}
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Events

func (lb *ListBase) ContextMenu(m *core.Scene) {
	if lb.IsReadOnly() || lb.isArray {
		return
	}
	core.NewButton(m).SetText("Add row").SetIcon(icons.Add).OnClick(func(e events.Event) {
		lb.SliceNewAtRow((lb.SelectedIndex - lb.StartIndex) + 1)
	})
	core.NewButton(m).SetText("Delete row").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		lb.SliceDeleteAtRow(lb.SelectedIndex - lb.StartIndex)
	})
	core.NewSeparator(m)
	core.NewButton(m).SetText("Copy").SetIcon(icons.Copy).OnClick(func(e events.Event) {
		lb.CopyIndexes(true)
	})
	core.NewButton(m).SetText("Cut").SetIcon(icons.Cut).OnClick(func(e events.Event) {
		lb.CutIndexes()
	})
	core.NewButton(m).SetText("Paste").SetIcon(icons.Paste).OnClick(func(e events.Event) {
		lb.PasteIndex(lb.SelectedIndex)
	})
	core.NewButton(m).SetText("Duplicate").SetIcon(icons.Copy).OnClick(func(e events.Event) {
		lb.Duplicate()
	})
}

// KeyInputNav supports multiple selection navigation keys
func (lb *ListBase) KeyInputNav(kt events.Event) {
	kf := keymap.Of(kt.KeyChord())
	selMode := events.SelectModeBits(kt.Modifiers())
	if selMode == events.SelectOne {
		if lb.SelectMode {
			selMode = events.ExtendContinuous
		}
	}
	switch kf {
	case keymap.CancelSelect:
		lb.UnselectAllIndexes()
		lb.SelectMode = false
		kt.SetHandled()
	case keymap.MoveDown:
		lb.MoveDownAction(selMode)
		kt.SetHandled()
	case keymap.MoveUp:
		lb.MoveUpAction(selMode)
		kt.SetHandled()
	case keymap.PageDown:
		lb.MovePageDownAction(selMode)
		kt.SetHandled()
	case keymap.PageUp:
		lb.MovePageUpAction(selMode)
		kt.SetHandled()
	case keymap.SelectMode:
		lb.SelectMode = !lb.SelectMode
		kt.SetHandled()
	case keymap.SelectAll:
		lb.SelectAllIndexes()
		lb.SelectMode = false
		kt.SetHandled()
	}
}

func (lb *ListBase) KeyInputEditable(kt events.Event) {
	lb.KeyInputNav(kt)
	if kt.IsHandled() {
		return
	}
	idx := lb.SelectedIndex
	kf := keymap.Of(kt.KeyChord())
	if core.DebugSettings.KeyEventTrace {
		slog.Info("ListBase KeyInput", "widget", lb, "keyFunction", kf)
	}
	switch kf {
	// case keymap.Delete: // too dangerous
	// 	sv.This.(Lister).SliceDeleteAt(sv.SelectedIndex)
	// 	sv.SelectMode = false
	// 	sv.SelectIndexAction(idx, events.SelectOne)
	// 	kt.SetHandled()
	case keymap.Duplicate:
		nidx := lb.Duplicate()
		lb.SelectMode = false
		if nidx >= 0 {
			lb.SelectIndexAction(nidx, events.SelectOne)
		}
		kt.SetHandled()
	case keymap.Insert:
		lb.This.(Lister).SliceNewAt(idx)
		lb.SelectMode = false
		lb.SelectIndexAction(idx+1, events.SelectOne) // todo: somehow nidx not working
		kt.SetHandled()
	case keymap.InsertAfter:
		lb.This.(Lister).SliceNewAt(idx + 1)
		lb.SelectMode = false
		lb.SelectIndexAction(idx+1, events.SelectOne)
		kt.SetHandled()
	case keymap.Copy:
		lb.CopyIndexes(true)
		lb.SelectMode = false
		lb.SelectIndexAction(idx, events.SelectOne)
		kt.SetHandled()
	case keymap.Cut:
		lb.CutIndexes()
		lb.SelectMode = false
		kt.SetHandled()
	case keymap.Paste:
		lb.PasteIndex(lb.SelectedIndex)
		lb.SelectMode = false
		kt.SetHandled()
	}
}

func (lb *ListBase) KeyInputReadOnly(kt events.Event) {
	if lb.ReadOnlyMultiSelect {
		lb.KeyInputNav(kt)
		if kt.IsHandled() {
			return
		}
	}
	selMode := kt.SelectMode()
	if lb.SelectMode {
		selMode = events.ExtendOne
	}
	kf := keymap.Of(kt.KeyChord())
	if core.DebugSettings.KeyEventTrace {
		slog.Info("ListBase ReadOnly KeyInput", "widget", lb, "keyFunction", kf)
	}
	idx := lb.SelectedIndex
	switch {
	case kf == keymap.MoveDown:
		ni := idx + 1
		if ni < lb.SliceSize {
			lb.ScrollToIndex(ni)
			lb.UpdateSelectIndex(ni, true, selMode)
			kt.SetHandled()
		}
	case kf == keymap.MoveUp:
		ni := idx - 1
		if ni >= 0 {
			lb.ScrollToIndex(ni)
			lb.UpdateSelectIndex(ni, true, selMode)
			kt.SetHandled()
		}
	case kf == keymap.PageDown:
		ni := min(idx+lb.VisRows-1, lb.SliceSize-1)
		lb.ScrollToIndex(ni)
		lb.UpdateSelectIndex(ni, true, selMode)
		kt.SetHandled()
	case kf == keymap.PageUp:
		ni := max(idx-(lb.VisRows-1), 0)
		lb.ScrollToIndex(ni)
		lb.UpdateSelectIndex(ni, true, selMode)
		kt.SetHandled()
	case kf == keymap.Enter || kf == keymap.Accept || kt.KeyRune() == ' ':
		lb.Send(events.DoubleClick, kt)
		kt.SetHandled()
	}
}

func (lb *ListBase) SizeFinal() {
	sg := lb.This.(Lister).SliceGrid()
	if sg == nil {
		lb.Frame.SizeFinal()
		return
	}
	localIter := 0
	for (lb.MakeIter < 2 || lb.VisRows != sg.VisRows) && localIter < 2 {
		if lb.VisRows != sg.VisRows {
			lb.VisRows = sg.VisRows
			lb.Update()
		} else {
			sg.StyleTree()
		}
		sg.SizeFinalUpdateChildrenSizes()
		lb.MakeIter++
		localIter++
	}
	lb.Frame.SizeFinal()
}

//////////////////////////////////////////////////////
// 	ListGrid and Layout

// ListGrid handles the resizing logic for [List], [Table].
type ListGrid struct {
	core.Frame

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

func (lg *ListGrid) Init() {
	lg.Frame.Init()
	lg.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
	})
}

func (lg *ListGrid) SizeFromChildren(iter int, pass core.LayoutPasses) math32.Vector2 {
	csz := lg.Frame.SizeFromChildren(iter, pass)
	rht, err := lg.LayImpl.RowHeight(0, 0)
	if err != nil {
		// fmt.Println("ListGrid Sizing Error:", err)
		lg.RowHeight = 42
	}
	if lg.NeedsRebuild() { // rebuilding = reset
		lg.RowHeight = rht
	} else {
		lg.RowHeight = max(lg.RowHeight, rht)
	}
	if lg.RowHeight == 0 {
		// fmt.Println("ListGrid Sizing Error: RowHeight should not be 0!", sg)
		lg.RowHeight = 42
	}
	allocHt := lg.Geom.Size.Alloc.Content.Y - lg.Geom.Size.InnerSpace.Y
	if allocHt > lg.RowHeight {
		lg.VisRows = int(math32.Floor(allocHt / lg.RowHeight))
	}
	lg.VisRows = max(lg.VisRows, lg.MinRows)
	minHt := lg.RowHeight * float32(lg.MinRows)
	// fmt.Println("VisRows:", sg.VisRows, "rh:", sg.RowHeight, "ht:", minHt)
	// visHt := sg.RowHeight * float32(sg.VisRows)
	csz.Y = minHt
	return csz
}

func (lg *ListGrid) SetScrollParams(d math32.Dims, sb *core.Slider) {
	if d == math32.X {
		lg.Frame.SetScrollParams(d, sb)
		return
	}
	sb.Min = 0
	sb.Step = 1
	if lg.VisRows > 0 {
		sb.PageStep = float32(lg.VisRows)
	} else {
		sb.PageStep = 10
	}
	sb.InputThreshold = sb.Step
}

func (lg *ListGrid) List() (Lister, *ListBase) {
	svi := lg.ParentByType(ListBaseType, tree.Embeds)
	if svi == nil {
		return nil, nil
	}
	sv := svi.(Lister)
	return sv, sv.AsListBase()
}

func (lg *ListGrid) ScrollChanged(d math32.Dims, sb *core.Slider) {
	if d == math32.X {
		lg.Frame.ScrollChanged(d, sb)
		return
	}
	_, sv := lg.List()
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
	_, sv := lg.List()
	if sv == nil {
		return
	}
	maxSize = float32(max(sv.SliceSize, 1))
	visSize = float32(lg.VisRows)
	visPct = visSize / maxSize
	return
}

func (lg *ListGrid) UpdateScroll(idx int) {
	if !lg.HasScroll[math32.Y] || lg.Scrolls[math32.Y] == nil {
		return
	}
	sb := lg.Scrolls[math32.Y]
	sb.SetValue(float32(idx))
}

func (lg *ListGrid) UpdateBackgrounds() {
	bg := lg.Styles.ActualBackground
	if lg.LastBackground == bg {
		return
	}
	lg.LastBackground = bg

	// we take our zebra intensity applied foreground color and then overlay it onto our background color

	zclr := colors.WithAF32(colors.ToUniform(lg.Styles.Color), core.AppearanceSettings.ZebraStripesWeight())
	lg.BgStripe = gradient.Apply(bg, func(c color.Color) color.Color {
		return colors.AlphaBlend(c, zclr)
	})

	hclr := colors.WithAF32(colors.ToUniform(lg.Styles.Color), 0.08)
	lg.BgHover = gradient.Apply(bg, func(c color.Color) color.Color {
		return colors.AlphaBlend(c, hclr)
	})

	zhclr := colors.WithAF32(colors.ToUniform(lg.Styles.Color), core.AppearanceSettings.ZebraStripesWeight()+0.08)
	lg.BgHoverStripe = gradient.Apply(bg, func(c color.Color) color.Color {
		return colors.AlphaBlend(c, zhclr)
	})

	lg.BgSelect = colors.C(colors.Scheme.Select.Container)

	lg.BgSelectStripe = colors.C(colors.AlphaBlend(colors.Scheme.Select.Container, zclr))

	lg.BgHoverSelect = colors.C(colors.AlphaBlend(colors.Scheme.Select.Container, hclr))

	lg.BgHoverSelectStripe = colors.C(colors.AlphaBlend(colors.Scheme.Select.Container, zhclr))

}

func (lg *ListGrid) RowBackground(sel, stripe, hover bool) image.Image {
	switch {
	case sel && stripe && hover:
		return lg.BgHoverSelectStripe
	case sel && stripe:
		return lg.BgSelectStripe
	case sel && hover:
		return lg.BgHoverSelect
	case sel:
		return lg.BgSelect
	case stripe && hover:
		return lg.BgHoverStripe
	case stripe:
		return lg.BgStripe
	case hover:
		return lg.BgHover
	default:
		return lg.Styles.ActualBackground
	}
}

func (lg *ListGrid) ChildBackground(child core.Widget) image.Image {
	bg := lg.Styles.ActualBackground
	_, sv := lg.List()
	if sv == nil {
		return bg
	}
	lg.UpdateBackgrounds()
	row, _ := sv.WidgetIndex(child)
	si := row + sv.StartIndex
	return lg.RowBackground(sv.IndexIsSelected(si), si%2 == 1, row == sv.hoverRow)
}

func (lg *ListGrid) RenderStripes() {
	pos := lg.Geom.Pos.Content
	sz := lg.Geom.Size.Actual.Content
	if lg.VisRows == 0 || sz.Y == 0 {
		return
	}
	lg.UpdateBackgrounds()

	pc := &lg.Scene.PaintContext
	rows := lg.LayImpl.Shape.Y
	cols := lg.LayImpl.Shape.X
	st := pos
	offset := 0
	_, sv := lg.List()
	startIndex := 0
	if sv != nil {
		startIndex = sv.StartIndex
		offset = startIndex % 2
	}
	for r := 0; r < rows; r++ {
		si := r + startIndex
		ht, _ := lg.LayImpl.RowHeight(r, 0)
		miny := st.Y
		for c := 0; c < cols; c++ {
			ki := r*cols + c
			if ki < lg.NumChildren() {
				kw := lg.Child(ki).(core.Widget).AsWidget()
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
		sbg := lg.RowBackground(sv.IndexIsSelected(si), stripe, r == sv.hoverRow)
		pc.BlitBox(st, ssz, sbg)
		st.Y += ht + lg.LayImpl.Gap.Y
	}
}

// MousePosInGrid returns true if the event mouse position is
// located within the slicegrid.
func (lg *ListGrid) MousePosInGrid(pt image.Point) bool {
	ptrel := lg.PointToRelPos(pt)
	sz := lg.Geom.ContentBBox.Size()
	if lg.VisRows == 0 || sz.Y == 0 {
		return false
	}
	if ptrel.Y < 0 || ptrel.Y >= sz.Y || ptrel.X < 0 || ptrel.X >= sz.X-50 { // leave margin on rhs around scroll
		return false
	}
	return true
}

// IndexFromPixel returns the row, column indexes of given pixel point within grid.
// Takes a scene-level position.
func (lg *ListGrid) IndexFromPixel(pt image.Point) (row, col int, isValid bool) {
	if !lg.MousePosInGrid(pt) {
		return
	}
	ptf := math32.Vector2FromPoint(lg.PointToRelPos(pt))
	sz := math32.Vector2FromPoint(lg.Geom.ContentBBox.Size())
	isValid = true
	rows := lg.LayImpl.Shape.Y
	cols := lg.LayImpl.Shape.X
	st := math32.Vector2{}
	got := false
	for r := 0; r < rows; r++ {
		ht, _ := lg.LayImpl.RowHeight(r, 0)
		ht += lg.LayImpl.Gap.Y
		miny := st.Y
		if r > 0 {
			for c := 0; c < cols; c++ {
				kw := lg.Child(r*cols + c).(core.Widget).AsWidget()
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
	lg.RenderStripes()
}
