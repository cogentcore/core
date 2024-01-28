// Copyright (c) 2018, Cogent Core. All rights reserved.
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

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
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

	// SliceViewNoAdd indicates whether the user cannot add elements to the slice
	SliceViewNoAdd

	// SliceViewNoDelete indicates whether the user cannot delete elements from the slice
	SliceViewNoDelete

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

	// SelectRowWidgets sets the selection state of given row of widgets
	SelectRowWidgets(row int, sel bool)

	// SliceNewAt inserts a new blank element at given
	// index in the slice. -1 means the end.
	SliceNewAt(idx int)

	// SliceDeleteAt deletes element at given index from slice
	// if updt is true, then update the grid after
	SliceDeleteAt(idx int)

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

	// optional mutex that, if non-nil, will be used around any updates that
	// read / modify the underlying Slice data.
	// Can be used to protect against random updating if your code has specific
	// update points that can be likewise protected with this same mutex.
	ViewMu *sync.Mutex `copier:"-" view:"-" json:"-" xml:"-"`

	// Changed indicates whether the underlying slice
	// has been edited in any way
	Changed bool `set:"-"`

	// non-ptr reflect.Value of the slice
	SliceNPVal reflect.Value `copier:"-" view:"-" json:"-" xml:"-"`

	// Value for the slice itself, if this was created within value view framework -- otherwise nil
	SliceValView Value `copier:"-" view:"-" json:"-" xml:"-"`

	// Value representations of the slice values
	Values []Value `copier:"-" view:"-" json:"-" xml:"-"`

	// current selection value -- initially select this value if set
	SelVal any `copier:"-" view:"-" json:"-" xml:"-"`

	// index of currently-selected item, in ReadOnly mode only
	SelIdx int `copier:"-" json:"-" xml:"-"`

	// list of currently-selected slice indexes
	SelIdxs map[int]struct{} `copier:"-"`

	// index of row to select at start
	InitSelIdx int `copier:"-" json:"-" xml:"-"`

	// list of currently-dragged indexes
	DraggedIdxs []int `copier:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `copier:"-" json:"-" xml:"-"`

	// total number of rows visible in allocated display size
	VisRows int `edit:"-" copier:"-" json:"-" xml:"-"`

	// starting slice index of visible rows
	StartIdx int `edit:"-" copier:"-" json:"-" xml:"-"`

	// size of slice
	SliceSize int `edit:"-" copier:"-" json:"-" xml:"-"`

	// iteration through the configuration process, reset when a new slice type is set
	ConfigIter int `edit:"-" copier:"-" json:"-" xml:"-"`

	// temp idx state for e.g., dnd
	TmpIdx int `copier:"-" view:"-" json:"-" xml:"-"`

	// ElVal is a Value representation of the underlying element type
	// which is used whenever there are no slice elements available
	ElVal reflect.Value `copier:"-" view:"-" json:"-" xml:"-"`
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
	sv.MinRows = 4
	sv.SetFlag(false, SliceViewSelectMode)
	sv.SetFlag(true, SliceViewShowIndex)
	sv.SetFlag(true, SliceViewReadOnlyKeyNav)
	svi := sv.This().(SliceViewer)

	sv.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.DoubleClickable)
		s.Direction = styles.Column
		// absorb horizontal here, vertical in view
		s.Overflow.X = styles.OverflowAuto
		s.Grow.Set(1, 1)
	})
	sv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(sv) {
		case "grid": // slice grid
			sg := w.(*SliceViewGrid)
			sg.Stripes = gi.RowStripes
			sg.Style(func(s *styles.Style) {
				sg.MinRows = sv.MinRows
				s.Display = styles.Grid
				nWidgPerRow, _ := sv.RowWidgetNs()
				s.Columns = nWidgPerRow
				s.Grow.Set(1, 1)
				s.Overflow.Set(styles.OverflowAuto)
				// baseline mins:
				s.Min.X.Ch(20)
				s.Min.Y.Em(6)
			})
		}
		if w.Parent().PathFrom(sv) == "grid" {
			switch {
			case strings.HasPrefix(w.Name(), "index-"):
				w.Style(func(s *styles.Style) {
					s.SetAbilities(true, abilities.Activatable, abilities.Selectable, abilities.Draggable, abilities.Droppable)
					s.Min.X.Em(1.5)
					s.Padding.Right.Dp(4)
					s.Text.Align = styles.End
					s.Min.Y.Em(1)
					s.GrowWrap = false
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
					idx := grr.Log1(strconv.Atoi(strings.TrimPrefix(w.Name(), "value-")))
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
		newslc = sv.Slice != sl
	}
	if !newslc && sv.Is(SliceViewConfigured) {
		sv.ConfigIter = 0
		sv.Update()
		return sv
	}
	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	sv.SetFlag(false, SliceViewConfigured)
	sv.StartIdx = 0
	sv.VisRows = sv.MinRows
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
	sv.Update()
	return sv
}

// IsNil returns true if the Slice is nil
func (sv *SliceViewBase) IsNil() bool {
	return laser.AnyIsNil(sv.Slice)
}

// BindSelectDialog makes the slice view a read-only selection slice view and then
// binds its events to its scene and its current selection index to the given value.
func (sv *SliceViewBase) BindSelectDialog(val *int) *SliceViewBase {
	sv.SetReadOnly(true)
	sv.OnSelect(func(e events.Event) {
		*val = sv.SelIdx
	})
	sv.OnDoubleClick(func(e events.Event) {
		*val = sv.SelIdx
		sv.Scene.SendKeyFun(keyfun.Accept, e) // activates Ok button code
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
			idxlab.OnSelect(func(e events.Event) {
				e.SetHandled()
				sv.UpdateSelectRow(i)
			})
			idxlab.SetText(sitxt)
			idxlab.ContextMenus = sv.ContextMenus
		}

		w := ki.NewOfType(vtyp).(gi.Widget)
		sg.SetChild(w, ridx+idxOff, valnm)
		vv.ConfigWidget(w)
		wb := w.AsWidget()
		wb.OnSelect(func(e events.Event) {
			e.SetHandled()
			sv.UpdateSelectRow(i)
		})
		wb.ContextMenus = sv.ContextMenus

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
					addact.SetType(gi.ButtonAction).SetIcon(icons.Add).
						SetTooltip("insert a new element at this index").OnClick(func(e events.Event) {
						sv.SliceNewAtRow(i + 1)
					})
				}

				if !sv.Is(SliceViewNoDelete) {
					cidx++
					delnm := fmt.Sprintf("del-%v", itxt)
					delact := gi.Button{}
					sg.SetChild(&delact, cidx, delnm)
					delact.SetType(gi.ButtonAction).SetIcon(icons.Delete).
						SetTooltip("delete this element").OnClick(func(e events.Event) {
						sv.SliceDeleteAtRow(i)
					})
				}
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
	// fmt.Println("updtw:", updt)

	sv.ViewMuLock()
	defer sv.ViewMuUnlock()

	sv.This().(SliceViewer).UpdtSliceSize()

	nWidgPerRow, idxOff := sv.RowWidgetNs()
	// sc := sv.Sc

	if sv.InitSelIdx >= 0 {
		sv.SelectIdxAction(sv.InitSelIdx, events.SelectOne)
		fmt.Println("selected init:", sv.InitSelIdx)
		sv.InitSelIdx = -1
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
			issel := sv.IdxIsSelected(si)
			w.AsWidget().SetSelected(issel)
			if sv.Is(SliceViewShowIndex) {
				idxlab.SetSelected(issel)
			}
			if !sv.IsReadOnly() && !sv.Is(SliceViewIsArray) {
				cidx := ridx + idxOff
				if !sv.Is(SliceViewNoAdd) {
					cidx++
					addact := sg.Kids[cidx].(*gi.Button)
					addact.SetState(invis, states.Invisible)
				}
				if !sv.Is(SliceViewNoDelete) {
					cidx++
					delact := sg.Kids[cidx].(*gi.Button)
					delact.SetState(invis, states.Invisible)
				}
			}
		} else {
			vv.SetSliceValue(sv.ElVal, sv.Slice, 0, sv.TmpSave, sv.ViewPath)
			vv.UpdateWidget()
			w.AsWidget().SetSelected(false)
			if sv.Is(SliceViewShowIndex) {
				idxlab.SetSelected(false)
			}
			if !sv.IsReadOnly() && !sv.Is(SliceViewIsArray) {
				cidx := ridx + idxOff
				if !sv.Is(SliceViewNoAdd) {
					cidx++
					addact := sg.Kids[cidx].(*gi.Button)
					addact.SetState(invis, states.Invisible)
				}
				if !sv.Is(SliceViewNoDelete) {
					cidx++
					delact := sg.Kids[cidx].(*gi.Button)
					delact.SetState(invis, states.Invisible)
				}
			}
		}
	}
	if sv.SelVal != nil {
		sv.SelIdx, _ = SliceIdxByValue(sv.Slice, sv.SelVal)
		sv.SelVal = nil
		sv.ScrollToIdx(sv.SelIdx)
		// sv.SetFocusEvent() // todo: doesn't work -- probably need priority events or something?
	} else if sv.InitSelIdx >= 0 {
		sv.SelIdx = sv.InitSelIdx
		sv.InitSelIdx = -1
		sv.ScrollToIdx(sv.SelIdx)
		// sv.SetFocusEvent()
	}
	if sv.IsReadOnly() && sv.SelIdx >= 0 {
		sv.SelectIdx(sv.SelIdx)
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
				ki.ChildByType[*gi.Chooser](w, ki.Embeds).SetTypes(gti.AllEmbeddersOf(ownki.BaseType()), true, true)
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
		sv.ApplyStyleTree()
		sv.This().(SliceViewer).UpdateWidgets()
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
	sv.ViewMuLock()
	defer sv.ViewMuUnlock()
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
	gi.NewButton(m).SetText("Copy").OnClick(func(e events.Event) {
		sv.CopyIdxs(true)
	})
	gi.NewButton(m).SetText("Cut").OnClick(func(e events.Event) {
		sv.CutIdxs()
	})
	gi.NewButton(m).SetText("Paste").OnClick(func(e events.Event) {
		sv.PasteIdx(sv.SelIdx)
	})
	gi.NewButton(m).SetText("Duplicate").OnClick(func(e events.Event) {
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
	sv.OnClick(func(e events.Event) {
		sv.SetFocusEvent()
	})
}

//////////////////////////////////////////////////////
// 	SliceViewGrid and Layout

// SliceViewGrid handles the resizing logic for SliceView, TableView.
type SliceViewGrid struct {
	gi.Frame // note: must be a frame to support stripes!

	// MinRows is set from parent SV
	MinRows int `edit:"-"`

	// height of a single row, computed during layout
	RowHeight float32 `edit:"-" copier:"-" json:"-" xml:"-"`

	// total number of rows visible in allocated display size
	VisRows int `edit:"-" copier:"-" json:"-" xml:"-"`
}

func (sg *SliceViewGrid) OnInit() {
	sg.Frame.OnInit()
	sg.Styles.Display = styles.Grid
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
	visHt := sg.RowHeight * float32(sg.VisRows)
	// fmt.Println("rowht:", sg.RowHeight, "allocht:", allocHt, "visrows:", sg.VisRows)
	_ = visHt
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
	maxSize = float32(max(sv.SliceSize, 1)) // + 0.01 // bit of extra to ensure last line always shows up
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

func (sv *SliceViewBase) SizeFinal() {
	sg := sv.This().(SliceViewer).SliceGrid()
	localIter := 0
	for (sv.ConfigIter < 2 || sv.VisRows != sg.VisRows) && localIter < 2 {
		// fmt.Println("sv:", sv.VisRows, "sg:", sg.VisRows)
		sv.VisRows = sg.VisRows
		sv.This().(SliceViewer).ConfigRows()
		sg.SizeFinalUpdateChildrenSizes()
		sv.ConfigIter++
		localIter++
	}
	sv.Frame.SizeFinal()
}
