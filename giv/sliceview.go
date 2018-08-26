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

	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  SliceView

// SliceView represents a slice, creating a property editor of the values --
// constructs Children widgets to show the index / value pairs, within an
// overall frame. Set to Inactive for select-only mode, which emits WidgetSig
// WidgetSelected signals when selection is updated
type SliceView struct {
	gi.Frame
	Slice        interface{}        `desc:"the slice that we are a view onto -- must be a pointer to that slice"`
	StyleFunc    SliceViewStyleFunc `view:"-" json:"-" xml:"-" desc:"optional styling function"`
	Values       []ValueView        `json:"-" xml:"-" desc:"ValueView representations of the slice values"`
	ShowIndex    bool               `xml:"index" desc:"whether to show index or not -- updated from "index" property (bool)"`
	InactKeyNav  bool               `xml:"inact-key-nav" desc:"support key navigation when inactive (default true) -- updated from "intact-key-nav" property (bool) -- no focus really plausible in inactive case, so it uses a low-pri capture of up / down events"`
	SelVal       interface{}        `view:"-" json:"-" xml:"-" desc:"current selection value -- initially select this value if set"`
	SelectedIdx  int                `json:"-" xml:"-" desc:"index of currently-selected item, in Inactive mode only"`
	SelectMode   bool               `desc:"editing-mode select rows mode"`
	SelectedRows map[int]struct{}   `desc:"list of currently-selected rows"`
	DraggedRows  []int              `desc:"list of currently-dragged rows"`
	SliceViewSig ki.Signal          `json:"-" xml:"-" desc:"slice view interactive editing signals"`
	ViewSig      ki.Signal          `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
	TmpSave      ValueView          `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	BuiltSlice   interface{}        `view:"-" json:"-" xml:"-" desc:"the built slice"`
	BuiltSize    int
	ToolbarSlice interface{} `desc:"the slice that we successfully set a toolbar for"`
	inFocusGrab  bool
}

var KiT_SliceView = kit.Types.AddType(&SliceView{}, SliceViewProps)

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// SliceViewStyleFunc is a styling function for custom styling /
// configuration of elements in the view
type SliceViewStyleFunc func(sv *SliceView, slice interface{}, widg gi.Node2D, row int, vv ValueView)

// SetSlice sets the source slice that we are viewing -- rebuilds the children
// to represent this slice
func (sv *SliceView) SetSlice(sl interface{}, tmpSave ValueView) {
	updt := false
	if sv.Slice != sl {
		updt = sv.UpdateStart()
		sv.Slice = sl
		if !sv.IsInactive() {
			sv.SelectedIdx = -1
		}
		sv.SelectedRows = make(map[int]struct{}, 10)
		sv.SelectMode = false
		sv.SetFullReRender()
	}
	sv.ShowIndex = true
	if sidxp, ok := sv.Prop("index"); ok {
		sv.ShowIndex, _ = kit.ToBool(sidxp)
	}
	sv.InactKeyNav = true
	if siknp, ok := sv.Prop("inact-key-nav"); ok {
		sv.InactKeyNav, _ = kit.ToBool(siknp)
	}
	sv.TmpSave = tmpSave
	sv.UpdateFromSlice()
	sv.UpdateEnd(updt)
}

var SliceViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"max-width":        -1,
	"max-height":       -1,
}

// SliceViewSignals are signals that sliceview can send, mostly for editing
// mode.  Selection events are sent on WidgetSig WidgetSelected signals in
// both modes.
type SliceViewSignals int64

const (
	// SliceViewDoubleClicked emitted during inactive mode when item
	// double-clicked -- can be used for accepting dialog.
	SliceViewDoubleClicked SliceViewSignals = iota

	// todo: add more signals as needed

	SliceViewSignalsN
)

//go:generate stringer -type=SliceViewSignals

// UpdateFromSlice performs overall configuration for given slice
func (sv *SliceView) UpdateFromSlice() {
	mods, updt := sv.StdConfig()
	sv.ConfigSliceGrid(false)
	sv.ConfigToolbar()
	if mods {
		sv.SetFullReRender()
		sv.UpdateEnd(updt)
	}
}

// UpdateValues updates the widget display of slice values, assuming same slice config
func (sv *SliceView) UpdateValues() {
	updt := sv.UpdateStart()
	for _, vv := range sv.Values {
		vv.UpdateWidget()
	}
	sv.UpdateEnd(updt)
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *SliceView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_Frame, "slice-grid")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (sv *SliceView) StdConfig() (mods, updt bool) {
	sv.Lay = gi.LayoutVert
	sv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := sv.StdFrameConfig()
	mods, updt = sv.ConfigChildren(config, false)
	return
}

// SliceGrid returns the SliceGrid grid frame widget, which contains all the
// fields and values, and its index, within frame -- nil, -1 if not found
func (sv *SliceView) SliceGrid() (*gi.Frame, int) {
	idx, ok := sv.Children().IndexByName("slice-grid", 0)
	if !ok {
		return nil, -1
	}
	return sv.KnownChild(idx).(*gi.Frame), idx
}

// ToolBar returns the toolbar widget
func (sv *SliceView) ToolBar() *gi.ToolBar {
	idx, ok := sv.Children().IndexByName("toolbar", 0)
	if !ok {
		return nil
	}
	return sv.KnownChild(idx).(*gi.ToolBar)
}

// RowWidgetNs returns number of widgets per row and offset for index label
func (sv *SliceView) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 2
	if !sv.IsInactive() {
		nWidgPerRow += 2
	}
	idxOff = 1
	if !sv.ShowIndex {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

// ConfigSliceGrid configures the SliceGrid for the current slice
func (sv *SliceView) ConfigSliceGrid(forceUpdt bool) {
	if kit.IfaceIsNil(sv.Slice) {
		return
	}
	mv := reflect.ValueOf(sv.Slice)
	mvnp := kit.NonPtrValue(mv)
	sz := mvnp.Len()

	if !forceUpdt && sv.BuiltSlice == sv.Slice && sv.BuiltSize == sz {
		return
	}
	sv.BuiltSlice = sv.Slice
	sv.BuiltSize = sz

	sg, _ := sv.SliceGrid()
	if sg == nil {
		return
	}
	updt := sg.UpdateStart()
	sg.SetFullReRender()
	defer sg.UpdateEnd(updt)

	nWidgPerRow, _ := sv.RowWidgetNs()

	sg.Lay = gi.LayoutGrid
	sg.SetProp("columns", nWidgPerRow)
	// setting a pref here is key for giving it a scrollbar in larger context
	sg.SetMinPrefHeight(units.NewValue(10, units.Em))
	sg.SetMinPrefWidth(units.NewValue(10, units.Em))
	sg.SetStretchMaxHeight() // for this to work, ALL layers above need it too
	sg.SetStretchMaxWidth()  // for this to work, ALL layers above need it too

	sv.Values = make([]ValueView, sz)

	sg.DeleteChildren(true)
	sg.Kids = make(ki.Slice, nWidgPerRow*sz)

	sv.ConfigSliceGridRows()
}

// ConfigSliceGridRows configures the SliceGrid rows for the current slice --
// assumes .Kids is created at the right size -- only call this for a direct
// re-render e.g., after sorting
func (sv *SliceView) ConfigSliceGridRows() {
	mv := reflect.ValueOf(sv.Slice)
	mvnp := kit.NonPtrValue(mv)
	sz := mvnp.Len()
	sg, _ := sv.SliceGrid()

	nWidgPerRow, idxOff := sv.RowWidgetNs()
	updt := sg.UpdateStart()
	defer sg.UpdateEnd(updt)

	for i := 0; i < sz; i++ {
		ridx := i * nWidgPerRow
		val := kit.OnePtrValue(mvnp.Index(i)) // deal with pointer lists
		vv := ToValueView(val.Interface())
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetSliceValue(val, sv.Slice, i, sv.TmpSave)
		sv.Values[i] = vv
		vtyp := vv.WidgetType()
		idxtxt := fmt.Sprintf("%05d", i)
		labnm := fmt.Sprintf("index-%v", idxtxt)
		valnm := fmt.Sprintf("value-%v", idxtxt)

		if sv.ShowIndex {
			var idxlab *gi.Label
			if sg.Kids[ridx] != nil {
				idxlab = sg.Kids[ridx].(*gi.Label)
			} else {
				idxlab = &gi.Label{}
				sg.SetChild(idxlab, ridx, labnm)
			}
			idxlab.Text = idxtxt
			idxlab.SetProp("slv-index", i)
			idxlab.Selectable = true
			idxlab.WidgetSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.WidgetSelected) {
					wbb := send.(gi.Node2D).AsWidget()
					idx := wbb.KnownProp("slv-index").(int)
					svv := recv.Embed(KiT_SliceView).(*SliceView)
					svv.UpdateSelect(idx, wbb.IsSelected())
				}
			})
		}

		var widg gi.Node2D
		if sg.Kids[ridx+idxOff] != nil {
			widg = sg.Kids[ridx+idxOff].(gi.Node2D)
		} else {
			widg = ki.NewOfType(vtyp).(gi.Node2D)
			sg.SetChild(widg, ridx+idxOff, valnm)
		}
		vv.ConfigWidget(widg)

		if sv.IsInactive() {
			widg.AsNode2D().SetInactive()
			wb := widg.AsWidget()
			if wb != nil {
				wb.SetProp("slv-index", i)
				wb.ClearSelected()
				wb.WidgetSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					if sig == int64(gi.WidgetSelected) || sig == int64(gi.WidgetFocused) {
						wbb := send.(gi.Node2D).AsWidget()
						idx := wbb.KnownProp("slv-index").(int)
						svv := recv.Embed(KiT_SliceView).(*SliceView)
						svv.UpdateSelect(idx, wbb.IsSelected())
					}
				})
			}
		} else {
			vvb := vv.AsValueViewBase()
			vvb.ViewSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				svv, _ := recv.Embed(KiT_SliceView).(*SliceView)
				svv.UpdateSig()
				svv.ViewSig.Emit(svv.This, 0, nil)
			})
			addnm := fmt.Sprintf("add-%v", idxtxt)
			delnm := fmt.Sprintf("del-%v", idxtxt)
			addact := gi.Action{}
			delact := gi.Action{}
			sg.SetChild(&addact, ridx+idxOff+1, addnm)
			sg.SetChild(&delact, ridx+idxOff+2, delnm)

			addact.SetIcon("plus")
			addact.Tooltip = "insert a new element at this index"
			addact.Data = i
			addact.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				act := send.(*gi.Action)
				svv := recv.Embed(KiT_SliceView).(*SliceView)
				svv.SliceNewAt(act.Data.(int)+1, true)
			})
			delact.SetIcon("minus")
			delact.Tooltip = "delete this element"
			delact.Data = i
			delact.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				act := send.(*gi.Action)
				svv := recv.Embed(KiT_SliceView).(*SliceView)
				svv.SliceDelete(act.Data.(int), true)
			})
		}
		if sv.StyleFunc != nil {
			sv.StyleFunc(sv, mvnp.Interface(), widg, i, vv)
		}

	}
	if sv.SelVal != nil {
		sv.SelectedIdx, _ = SliceRowByValue(sv.Slice, sv.SelVal)
	}
	if sv.IsInactive() && sv.SelectedIdx >= 0 {
		sv.SelectRowWidgets(sv.SelectedIdx, true)
	}
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1 means the end
func (sv *SliceView) SliceNewAt(idx int, reconfig bool) {
	updt := sv.UpdateStart()
	defer sv.UpdateEnd(updt)

	svl := reflect.ValueOf(sv.Slice)
	svnp := kit.NonPtrValue(svl)
	svtyp := svnp.Type()
	nval := reflect.New(svtyp.Elem())
	sz := svnp.Len()
	svnp = reflect.Append(svnp, nval.Elem())
	if idx >= 0 && idx < sz-1 {
		reflect.Copy(svnp.Slice(idx+1, sz+1), svnp.Slice(idx, sz))
		svnp.Index(idx).Set(nval.Elem())
	}
	svl.Elem().Set(svnp)
	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	if reconfig {
		sv.ConfigSliceGrid(true)
	}
	sv.ViewSig.Emit(sv.This, 0, nil)
}

// SliceDelete deletes element at given index from slice
func (sv *SliceView) SliceDelete(idx int, reconfig bool) {
	updt := sv.UpdateStart()
	defer sv.UpdateEnd(updt)

	svl := reflect.ValueOf(sv.Slice)
	svnp := kit.NonPtrValue(svl)
	svtyp := svnp.Type()
	nval := reflect.New(svtyp.Elem())
	sz := svnp.Len()
	reflect.Copy(svnp.Slice(idx, sz-1), svnp.Slice(idx+1, sz))
	svnp.Index(sz - 1).Set(nval.Elem())
	svl.Elem().Set(svnp.Slice(0, sz-1))
	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	if reconfig {
		sv.ConfigSliceGrid(true)
	}
	sv.ViewSig.Emit(sv.This, 0, nil)
}

// ConfigToolbar configures the toolbar actions
func (sv *SliceView) ConfigToolbar() {
	if kit.IfaceIsNil(sv.Slice) || sv.IsInactive() {
		return
	}
	if sv.ToolbarSlice == sv.Slice {
		return
	}
	tb := sv.ToolBar()
	if len(*tb.Children()) == 0 {
		tb.SetStretchMaxWidth()
		addac := tb.AddNewChild(gi.KiT_Action, "Add").(*gi.Action)
		addac.SetText("Add")
		addac.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SliceView).(*SliceView)
			svv.SliceNewAt(-1, true)
		})
	}
	sz := len(*tb.Children())
	if sz > 1 {
		for i := sz - 1; i >= 1; i-- {
			tb.DeleteChildAtIndex(i, true)
		}
	}
	if HasToolBarView(sv.Slice) {
		ToolBarView(sv.Slice, sv.Viewport, tb)
	}
	sv.ToolbarSlice = sv.Slice
}

func (sv *SliceView) Render2D() {
	sv.ToolBar().UpdateActions()
	if win := sv.ParentWindow(); win != nil {
		win.MainMenuUpdateActives()
	}
	if sv.FullReRenderIfNeeded() {
		return
	}
	if sv.PushBounds() {
		sv.FrameStdRender()
		sv.SliceViewEvents()
		sv.RenderScrolls()
		sv.Render2DChildren()
		sv.PopBounds()
		if sv.SelectedIdx > -1 {
			sv.ScrollToRow(sv.SelectedIdx)
		}
	} else {
		sv.DisconnectAllEvents(gi.AllPris)
	}
}

func (sv *SliceView) HasFocus2D() bool {
	if sv.IsInactive() {
		return sv.InactKeyNav
	}
	return sv.ContainsFocus() // anyone within us gives us focus..
}

//////////////////////////////////////////////////////////////////////////////
//  Row access methods

// RowVal returns value interface at given row
func (sv *SliceView) RowVal(row int) interface{} {
	mv := reflect.ValueOf(sv.Slice)
	mvnp := kit.NonPtrValue(mv)
	sz := mvnp.Len()
	if row < 0 || row >= sz {
		fmt.Printf("giv.SliceView: row index out of range: %v\n", row)
		return nil
	}
	val := kit.OnePtrValue(mvnp.Index(row)) // deal with pointer lists
	vali := val.Interface()
	return vali
}

// RowFirstWidget returns the first widget for given row (could be index or
// not) -- false if out of range
func (sv *SliceView) RowFirstWidget(row int) (*gi.WidgetBase, bool) {
	if !sv.ShowIndex {
		return nil, false
	}
	if sv.RowVal(row) == nil { // range check
		return nil, false
	}
	nWidgPerRow, _ := sv.RowWidgetNs()
	sg, _ := sv.SliceGrid()
	if sg == nil {
		return nil, false
	}
	widg := sg.Kids[row*nWidgPerRow].(gi.Node2D).AsWidget()
	return widg, true
}

// RowGrabFocus grabs the focus for the first focusable widget in given row --
// returns that element or nil if not successful -- note: grid must have
// already rendered for focus to be grabbed!
func (sv *SliceView) RowGrabFocus(row int) *gi.WidgetBase {
	if sv.RowVal(row) == nil || sv.inFocusGrab { // range check
		return nil
	}
	nWidgPerRow, idxOff := sv.RowWidgetNs()
	sg, _ := sv.SliceGrid()
	if sg == nil {
		return nil
	}
	ridx := nWidgPerRow * row
	widg := sg.KnownChild(ridx + idxOff).(gi.Node2D).AsWidget()
	if widg.HasFocus() {
		return widg
	}
	sv.inFocusGrab = true
	defer func() { sv.inFocusGrab = false }()
	if widg.CanFocus() {
		widg.GrabFocus()
		return widg
	}
	return nil
}

// RowPos returns center of window position of index label for row (ContextMenuPos)
func (sv *SliceView) RowPos(row int) image.Point {
	var pos image.Point
	widg, ok := sv.RowFirstWidget(row)
	if ok {
		pos = widg.ContextMenuPos()
	}
	return pos
}

// RowFromPos returns the row that contains given vertical position, false if not found
func (sv *SliceView) RowFromPos(posY int) (int, bool) {
	// todo: could optimize search to approx loc, and search up / down from there
	for rw := 0; rw < sv.BuiltSize; rw++ {
		widg, ok := sv.RowFirstWidget(rw)
		if ok {
			if widg.WinBBox.Min.Y < posY && posY < widg.WinBBox.Max.Y {
				return rw, true
			}
		}
	}
	return -1, false
}

// ScrollToRow ensures that given row is visible by scrolling layout as needed
// -- returns true if any scrolling was performed
func (sv *SliceView) ScrollToRow(row int) bool {
	sg, _ := sv.SliceGrid()
	if widg, ok := sv.RowFirstWidget(row); ok {
		return sg.ScrollToItem(widg)
	}
	return false
}

// SelectVal sets SelVal and attempts to find corresponding row, setting
// SelectedIdx and selecting row if found -- returns true if found, false
// otherwise.
func (sv *SliceView) SelectVal(val string) bool {
	sv.SelVal = val
	if sv.SelVal != nil {
		idx, _ := SliceRowByValue(sv.Slice, sv.SelVal)
		if idx >= 0 {
			sv.ScrollToRow(idx)
			sv.UpdateSelect(idx, true)
			return true
		}
	}
	return false
}

// SliceRowByValue searches for first row that contains given value in slice
// -- returns false if not found
func SliceRowByValue(struSlice interface{}, fldVal interface{}) (int, bool) {
	mv := reflect.ValueOf(struSlice)
	mvnp := kit.NonPtrValue(mv)
	sz := mvnp.Len()
	for row := 0; row < sz; row++ {
		rval := kit.NonPtrValue(mvnp.Index(row))
		if rval.Interface() == fldVal {
			return row, true
		}
	}
	return -1, false
}

/////////////////////////////////////////////////////////////////////////////
//    Moving

// MoveDown moves the selection down to next row, using given select mode
// (from keyboard modifiers) -- returns newly selected row or -1 if failed
func (sv *SliceView) MoveDown(selMode mouse.SelectModes) int {
	if selMode == mouse.NoSelectMode {
		if sv.SelectMode {
			selMode = mouse.ExtendContinuous
		}
	}
	if sv.SelectedIdx >= sv.BuiltSize-1 {
		sv.SelectedIdx = sv.BuiltSize - 1
		return -1
	}
	sv.SelectedIdx++
	sv.SelectRowAction(sv.SelectedIdx, selMode)
	return sv.SelectedIdx
}

// MoveDownAction moves the selection down to next row, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected
// row
func (sv *SliceView) MoveDownAction(selMode mouse.SelectModes) int {
	nrow := sv.MoveDown(selMode)
	if nrow >= 0 {
		sv.ScrollToRow(nrow)
		sv.WidgetSig.Emit(sv.This, int64(gi.WidgetSelected), nrow)
	}
	return nrow
}

// MoveUp moves the selection up to previous row, using given select mode
// (from keyboard modifiers) -- returns newly selected row or -1 if failed
func (sv *SliceView) MoveUp(selMode mouse.SelectModes) int {
	if selMode == mouse.NoSelectMode {
		if sv.SelectMode {
			selMode = mouse.ExtendContinuous
		}
	}
	if sv.SelectedIdx <= 0 {
		sv.SelectedIdx = 0
		return -1
	}
	sv.SelectedIdx--
	sv.SelectRowAction(sv.SelectedIdx, selMode)
	return sv.SelectedIdx
}

// MoveUpAction moves the selection up to previous row, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected
// row
func (sv *SliceView) MoveUpAction(selMode mouse.SelectModes) int {
	nrow := sv.MoveUp(selMode)
	if nrow >= 0 {
		sv.ScrollToRow(nrow)
		sv.WidgetSig.Emit(sv.This, int64(gi.WidgetSelected), nrow)
	}
	return nrow
}

//////////////////////////////////////////////////////////////////////////////
//    Selection: user operates on the index labels

// SelectRowWidgets sets the selection state of given row of widgets
func (sv *SliceView) SelectRowWidgets(idx int, sel bool) {
	sg, _ := sv.SliceGrid()
	nWidgPerRow, idxOff := sv.RowWidgetNs()
	rowidx := idx * nWidgPerRow
	if sv.ShowIndex {
		if sg.Kids.IsValidIndex(rowidx) {
			widg := sg.KnownChild(rowidx).(gi.Node2D).AsNode2D()
			widg.SetSelectedState(sel)
			widg.UpdateSig()
		}
	}
	if sg.Kids.IsValidIndex(rowidx + idxOff) {
		widg := sg.KnownChild(rowidx + idxOff).(gi.Node2D).AsNode2D()
		widg.SetSelectedState(sel)
		widg.UpdateSig()
	}
}

// UpdateSelect updates the selection for the given index -- callback from widgetsig select
func (sv *SliceView) UpdateSelect(idx int, sel bool) {
	if sv.IsInactive() {
		if sv.SelectedIdx >= 0 { // unselect current
			sv.SelectRowWidgets(sv.SelectedIdx, false)
		}
		if sel {
			sv.SelectedIdx = idx
			sv.SelectRowWidgets(sv.SelectedIdx, true)
		} else {
			sv.SelectedIdx = -1
		}
		sv.WidgetSig.Emit(sv.This, int64(gi.WidgetSelected), sv.SelectedIdx)
	} else {
		selMode := mouse.NoSelectMode
		win := sv.Viewport.Win
		if win != nil {
			selMode = win.LastSelMode
		}
		sv.SelectRowAction(idx, selMode)
	}
}

// RowIsSelected returns the selected status of given row index
func (sv *SliceView) RowIsSelected(row int) bool {
	if _, ok := sv.SelectedRows[row]; ok {
		return true
	}
	return false
}

// SelectedRowsList returns list of selected rows, sorted either ascending or descending
func (sv *SliceView) SelectedRowsList(descendingSort bool) []int {
	rws := make([]int, len(sv.SelectedRows))
	i := 0
	for r, _ := range sv.SelectedRows {
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

// SelectRow selects given row (if not already selected) -- updates select
// status of index label
func (sv *SliceView) SelectRow(row int) {
	sv.SelectedRows[row] = struct{}{}
	sv.SelectRowWidgets(row, true)
}

// UnselectRow unselects given row (if selected)
func (sv *SliceView) UnselectRow(row int) {
	if sv.RowIsSelected(row) {
		delete(sv.SelectedRows, row)
	}
	sv.SelectRowWidgets(row, false)
}

// UnselectAllRows unselects all selected rows
func (sv *SliceView) UnselectAllRows() {
	win := sv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	for r, _ := range sv.SelectedRows {
		sv.SelectRowWidgets(r, false)
	}
	sv.SelectedRows = make(map[int]struct{}, 10)
	if win != nil {
		win.UpdateEnd(updt)
	}
}

// SelectAllRows selects all rows
func (sv *SliceView) SelectAllRows() {
	win := sv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	sv.UnselectAllRows()
	sv.SelectedRows = make(map[int]struct{}, sv.BuiltSize)
	for row := 0; row < sv.BuiltSize; row++ {
		sv.SelectedRows[row] = struct{}{}
		sv.SelectRowWidgets(row, true)
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
}

// SelectRowAction is called when a select action has been received (e.g., a
// mouse click) -- translates into selection updates -- gets selection mode
// from mouse event (ExtendContinuous, ExtendOne)
func (sv *SliceView) SelectRowAction(row int, mode mouse.SelectModes) {
	if row >= sv.BuiltSize {
		row = sv.BuiltSize - 1
	}
	if row < 0 {
		row = 0
	}
	win := sv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	switch mode {
	case mouse.ExtendContinuous:
		if len(sv.SelectedRows) == 0 {
			sv.SelectedIdx = row
			sv.SelectRow(row)
			sv.RowGrabFocus(row)
			sv.WidgetSig.Emit(sv.This, int64(gi.WidgetSelected), sv.SelectedIdx)
		} else {
			minIdx := -1
			maxIdx := 0
			for r, _ := range sv.SelectedRows {
				if minIdx < 0 {
					minIdx = r
				} else {
					minIdx = kit.MinInt(minIdx, r)
				}
				maxIdx = kit.MaxInt(maxIdx, r)
			}
			cidx := row
			sv.SelectedIdx = row
			sv.SelectRow(row)
			if row < minIdx {
				for cidx < minIdx {
					r := sv.MoveDown(mouse.SelectModesN) // just select
					cidx = r
				}
			} else if row > maxIdx {
				for cidx > maxIdx {
					r := sv.MoveUp(mouse.SelectModesN) // just select
					cidx = r
				}
			}
			sv.RowGrabFocus(row)
			sv.WidgetSig.Emit(sv.This, int64(gi.WidgetSelected), sv.SelectedIdx)
		}
	case mouse.ExtendOne:
		if sv.RowIsSelected(row) {
			sv.UnselectRowAction(row)
		} else {
			sv.SelectedIdx = row
			sv.SelectRow(row)
			sv.RowGrabFocus(row)
			sv.WidgetSig.Emit(sv.This, int64(gi.WidgetSelected), sv.SelectedIdx)
		}
	case mouse.NoSelectMode:
		if sv.RowIsSelected(row) {
			if len(sv.SelectedRows) > 1 {
				sv.UnselectAllRows()
			}
			sv.SelectedIdx = row
			sv.SelectRow(row)
			sv.RowGrabFocus(row)
		} else {
			sv.UnselectAllRows()
			sv.SelectedIdx = row
			sv.SelectRow(row)
			sv.RowGrabFocus(row)
		}
		sv.WidgetSig.Emit(sv.This, int64(gi.WidgetSelected), sv.SelectedIdx)
	default: // anything else
		sv.SelectedIdx = row
		sv.SelectRow(row)
		sv.RowGrabFocus(row)
		sv.WidgetSig.Emit(sv.This, int64(gi.WidgetSelected), sv.SelectedIdx)
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
}

// UnselectRowAction unselects this row (if selected) -- and emits a signal
func (sv *SliceView) UnselectRowAction(row int) {
	if sv.RowIsSelected(row) {
		sv.UnselectRow(row)
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeDataRow adds mimedata for given row: an application/json of the struct
func (sv *SliceView) MimeDataRow(md *mimedata.Mimes, row int) {
	val := sv.RowVal(row)
	b, err := json.MarshalIndent(val, "", "  ")
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: mimedata.AppJSON, Data: b})
	} else {
		log.Printf("gi.SliceView MimeData JSON Marshall error: %v\n", err)
	}
}

// RowsFromMimeData creates a slice of structs from mime data
func (sv *SliceView) RowsFromMimeData(md mimedata.Mimes) []interface{} {
	tvl := reflect.ValueOf(sv.Slice)
	tvnp := kit.NonPtrValue(tvl)
	tvtyp := tvnp.Type()
	sl := make([]interface{}, 0, len(md))
	for _, d := range md {
		if d.Type == mimedata.AppJSON {
			nval := reflect.New(tvtyp.Elem()).Interface()
			err := json.Unmarshal(d.Data, nval)
			if err == nil {
				sl = append(sl, nval)
			} else {
				log.Printf("gi.SliceView RowsFromMimeData: JSON load error: %v\n", err)
			}
		}
	}
	return sl
}

// CopyRows copies selected rows to clip.Board, optionally resetting the selection
func (sv *SliceView) CopyRows(reset bool) {
	nitms := len(sv.SelectedRows)
	if nitms == 0 {
		return
	}
	md := make(mimedata.Mimes, 0, nitms)
	for r, _ := range sv.SelectedRows {
		sv.MimeDataRow(&md, r)
	}
	oswin.TheApp.ClipBoard().Write(md)
	if reset {
		sv.UnselectAllRows()
	}
}

// DeleteRows deletes all selected rows
func (sv *SliceView) DeleteRows() {
	if len(sv.SelectedRows) == 0 {
		return
	}
	updt := sv.UpdateStart()
	rws := sv.SelectedRowsList(true) // descending sort
	for _, r := range rws {
		sv.SliceDelete(r, false)
	}
	sv.ConfigSliceGrid(true)
	sv.UpdateEnd(updt)
}

// CutRows copies selected rows to clip.Board and deletes selected rows
func (sv *SliceView) CutRows() {
	if len(sv.SelectedRows) == 0 {
		return
	}
	updt := sv.UpdateStart()
	sv.CopyRows(false)
	rws := sv.SelectedRowsList(true) // descending sort
	row := rws[0]
	sv.UnselectAllRows()
	for _, r := range rws {
		sv.SliceDelete(r, false)
	}
	sv.ConfigSliceGrid(true)
	sv.UpdateEnd(updt)
	sv.SelectRowAction(row, mouse.NoSelectMode)
}

// Paste pastes clipboard at given row
func (sv *SliceView) Paste(row int) {
	md := oswin.TheApp.ClipBoard().Read([]string{mimedata.AppJSON})
	if md != nil {
		sv.PasteAction(md, row)
	}
}

// MakePasteMenu makes the menu of options for paste events
func (sv *SliceView) MakePasteMenu(m *gi.Menu, data interface{}, row int) {
	if len(*m) > 0 {
		return
	}
	m.AddAction(gi.ActOpts{Label: "Assign To", Data: data}, sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_SliceView).(*SliceView)
		tvv.PasteAssign(data.(mimedata.Mimes), row)
	})
	m.AddAction(gi.ActOpts{Label: "Insert Before", Data: data}, sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_SliceView).(*SliceView)
		tvv.PasteAtRow(data.(mimedata.Mimes), row)
	})
	m.AddAction(gi.ActOpts{Label: "Insert After", Data: data}, sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_SliceView).(*SliceView)
		tvv.PasteAtRow(data.(mimedata.Mimes), row+1)
	})
	m.AddAction(gi.ActOpts{Label: "Cancel", Data: data}, sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
	})
}

// PasteAction performs a paste from the clipboard using given data -- pops up
// a menu to determine what specifically to do
func (sv *SliceView) PasteAction(md mimedata.Mimes, row int) {
	sv.UnselectAllRows()
	var men gi.Menu
	sv.MakePasteMenu(&men, md, row)
	pos := sv.RowPos(row)
	gi.PopupMenu(men, pos.X, pos.Y, sv.Viewport, "tvPasteMenu")
}

// PasteAssign assigns mime data (only the first one!) to this row
func (sv *SliceView) PasteAssign(md mimedata.Mimes, row int) {
	tvl := reflect.ValueOf(sv.Slice)
	tvnp := kit.NonPtrValue(tvl)

	sl := sv.RowsFromMimeData(md)
	updt := sv.UpdateStart()
	if len(sl) == 0 {
		return
	}
	ns := sl[0]
	tvnp.Index(row).Set(reflect.ValueOf(ns).Elem())
	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.ConfigSliceGridRows() // no change in length
	sv.UpdateEnd(updt)
}

// PasteAtRow inserts object(s) from mime data at (before) given row
func (sv *SliceView) PasteAtRow(md mimedata.Mimes, row int) {
	tvl := reflect.ValueOf(sv.Slice)
	tvnp := kit.NonPtrValue(tvl)

	sl := sv.RowsFromMimeData(md)
	updt := sv.UpdateStart()
	for _, ns := range sl {
		sz := tvnp.Len()
		tvnp = reflect.Append(tvnp, reflect.ValueOf(ns).Elem())
		tvl.Elem().Set(tvnp)
		if row >= 0 && row < sz {
			reflect.Copy(tvnp.Slice(row+1, sz+1), tvnp.Slice(row, sz))
			tvnp.Index(row).Set(reflect.ValueOf(ns).Elem())
			tvl.Elem().Set(tvnp)
		}
		row++
	}
	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.ConfigSliceGrid(true)
	sv.UpdateEnd(updt)
	sv.SelectRowAction(row, mouse.NoSelectMode)
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// DragNDropStart starts a drag-n-drop
func (sv *SliceView) DragNDropStart() {
	nitms := len(sv.SelectedRows)
	if nitms == 0 {
		return
	}
	md := make(mimedata.Mimes, 0, nitms)
	for r, _ := range sv.SelectedRows {
		sv.MimeDataRow(&md, r)
	}
	rws := sv.SelectedRowsList(true) // descending sort
	widg, ok := sv.RowFirstWidget(rws[0])
	if ok {
		bi := &gi.Bitmap{}
		bi.InitName(bi, sv.UniqueName())
		bi.GrabRenderFrom(widg)
		gi.ImageClearer(bi.Pixels, 50.0)
		sv.Viewport.Win.StartDragNDrop(sv.This, md, bi)
	}
}

// DragNDropTarget handles a drag-n-drop drop
func (sv *SliceView) DragNDropTarget(de *dnd.Event) {
	de.Target = sv.This
	if de.Mod == dnd.DropLink {
		de.Mod = dnd.DropCopy // link not supported -- revert to copy
	}
	row, ok := sv.RowFromPos(de.Where.Y)
	if ok {
		de.SetProcessed()
		sv.DropAction(de.Data, de.Mod, row)
	}
}

// MakeDropMenu makes the menu of options for dropping on a target
func (sv *SliceView) MakeDropMenu(m *gi.Menu, data interface{}, mod dnd.DropMods, row int) {
	if len(*m) > 0 {
		return
	}
	switch mod {
	case dnd.DropCopy:
		m.AddLabel("Copy (Shift=Move):")
	case dnd.DropMove:
		m.AddLabel("Move:")
	}
	if mod == dnd.DropCopy {
		m.AddAction(gi.ActOpts{Label: "Assign To", Data: data}, sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_SliceView).(*SliceView)
			tvv.DropAssign(data.(mimedata.Mimes), row)
		})
	}
	m.AddAction(gi.ActOpts{Label: "Insert Before", Data: data}, sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_SliceView).(*SliceView)
		tvv.DropBefore(data.(mimedata.Mimes), mod, row) // captures mod
	})
	m.AddAction(gi.ActOpts{Label: "Insert After", Data: data}, sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_SliceView).(*SliceView)
		tvv.DropAfter(data.(mimedata.Mimes), mod, row) // captures mod
	})
	m.AddAction(gi.ActOpts{Label: "Cancel", Data: data}, sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_SliceView).(*SliceView)
		tvv.DropCancel()
	})
}

// DropAction pops up a menu to determine what specifically to do with dropped items
func (sv *SliceView) DropAction(md mimedata.Mimes, mod dnd.DropMods, row int) {
	var men gi.Menu
	sv.MakeDropMenu(&men, md, mod, row)
	pos := sv.RowPos(row)
	gi.PopupMenu(men, pos.X, pos.Y, sv.Viewport, "tvDropMenu")
}

// DropAssign assigns mime data (only the first one!) to this node
func (sv *SliceView) DropAssign(md mimedata.Mimes, row int) {
	sv.DraggedRows = nil
	sv.PasteAssign(md, row)
	sv.DragNDropFinalize(dnd.DropCopy)
}

// DragNDropFinalize is called to finalize actions on the Source node prior to
// performing target actions -- mod must indicate actual action taken by the
// target, including ignore -- ends up calling DragNDropSource if us..
func (sv *SliceView) DragNDropFinalize(mod dnd.DropMods) {
	sv.UnselectAllRows()
	sv.Viewport.Win.FinalizeDragNDrop(mod)
}

// DragNDropSource is called after target accepts the drop -- we just remove
// elements that were moved
func (sv *SliceView) DragNDropSource(de *dnd.Event) {
	if de.Mod != dnd.DropMove || len(sv.DraggedRows) == 0 {
		return
	}
	updt := sv.UpdateStart()
	sort.Slice(sv.DraggedRows, func(i, j int) bool {
		return sv.DraggedRows[i] > sv.DraggedRows[j]
	})
	row := sv.DraggedRows[0]
	for _, r := range sv.DraggedRows {
		sv.SliceDelete(r, false)
	}
	sv.DraggedRows = nil
	sv.ConfigSliceGrid(true)
	sv.UpdateEnd(updt)
	sv.SelectRowAction(row, mouse.NoSelectMode)
}

// SaveDraggedRows saves selectedrows into dragged rows taking into account insertion at rows
func (sv *SliceView) SaveDraggedRows(row int) {
	sz := len(sv.SelectedRows)
	if sz == 0 {
		sv.DraggedRows = nil
		return
	}
	sv.DraggedRows = make([]int, len(sv.SelectedRows))
	idx := 0
	for r, _ := range sv.SelectedRows {
		if r > row {
			sv.DraggedRows[idx] = r + sz // make room for insertion
		} else {
			sv.DraggedRows[idx] = r
		}
		idx++
	}
}

// DropBefore inserts object(s) from mime data before this node
func (sv *SliceView) DropBefore(md mimedata.Mimes, mod dnd.DropMods, row int) {
	sv.SaveDraggedRows(row)
	sv.PasteAtRow(md, row)
	sv.DragNDropFinalize(mod)
}

// DropAfter inserts object(s) from mime data after this node
func (sv *SliceView) DropAfter(md mimedata.Mimes, mod dnd.DropMods, row int) {
	sv.SaveDraggedRows(row + 1)
	sv.PasteAtRow(md, row+1)
	sv.DragNDropFinalize(mod)
}

// DropCancel cancels the drop action e.g., preventing deleting of source
// items in a Move case
func (sv *SliceView) DropCancel() {
	sv.DragNDropFinalize(dnd.DropIgnore)
}

func (sv *SliceView) KeyInputActive(kt *key.ChordEvent) {
	kf := gi.KeyFun(kt.ChordString())
	selMode := mouse.SelectModeBits(kt.Modifiers)
	row := sv.SelectedIdx
	switch kf {
	case gi.KeyFunCancelSelect:
		sv.UnselectAllRows()
		sv.SelectMode = false
		kt.SetProcessed()
	case gi.KeyFunMoveDown:
		sv.MoveDownAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunMoveUp:
		sv.MoveUpAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunSelectMode:
		sv.SelectMode = !sv.SelectMode
		kt.SetProcessed()
	case gi.KeyFunSelectAll:
		sv.SelectAllRows()
		sv.SelectMode = false
		kt.SetProcessed()
	case gi.KeyFunDelete:
		sv.SliceDelete(sv.SelectedIdx, true)
		sv.SelectMode = false
		sv.SelectRowAction(row, mouse.NoSelectMode)
		kt.SetProcessed()
	// case gi.KeyFunDuplicate:
	// 	sv.SrcDuplicate() // todo: dupe
	// 	kt.SetProcessed()
	case gi.KeyFunInsert:
		sv.SliceNewAt(row, true)
		sv.SelectMode = false
		sv.SelectRowAction(row+1, mouse.NoSelectMode) // todo: somehow nrow not working
		kt.SetProcessed()
	case gi.KeyFunInsertAfter:
		sv.SliceNewAt(row+1, true)
		sv.SelectMode = false
		sv.SelectRowAction(row+1, mouse.NoSelectMode)
		kt.SetProcessed()
	case gi.KeyFunCopy:
		sv.CopyRows(true)
		sv.SelectMode = false
		sv.SelectRowAction(row, mouse.NoSelectMode)
		kt.SetProcessed()
	case gi.KeyFunCut:
		sv.CutRows()
		sv.SelectMode = false
		kt.SetProcessed()
	case gi.KeyFunPaste:
		sv.Paste(sv.SelectedIdx)
		sv.SelectMode = false
		kt.SetProcessed()
	}
}

func (sv *SliceView) KeyInputInactive(kt *key.ChordEvent) {
	kf := gi.KeyFun(kt.ChordString())
	row := sv.SelectedIdx
	switch kf {
	case gi.KeyFunMoveDown:
		nr := row + 1
		if nr < sv.BuiltSize {
			sv.ScrollToRow(nr)
			sv.UpdateSelect(nr, true)
			kt.SetProcessed()
		}
	case gi.KeyFunMoveUp:
		nr := row - 1
		if nr >= 0 {
			sv.ScrollToRow(nr)
			sv.UpdateSelect(nr, true)
			kt.SetProcessed()
		}
	}
}

func (sv *SliceView) SliceViewEvents() {
	if sv.IsInactive() {
		if sv.InactKeyNav {
			sv.ConnectEvent(oswin.KeyChordEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
				tvv := recv.Embed(KiT_SliceView).(*SliceView)
				kt := d.(*key.ChordEvent)
				tvv.KeyInputInactive(kt)
			})
		}
		sv.ConnectEvent(oswin.MouseEvent, gi.LowRawPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			me := d.(*mouse.Event)
			svv := recv.Embed(KiT_SliceView).(*SliceView)
			if me.Button == mouse.Left && me.Action == mouse.DoubleClick {
				svv.SliceViewSig.Emit(svv.This, int64(SliceViewDoubleClicked), svv.SelectedIdx)
				me.SetProcessed()
			}
		})
	} else {
		sv.ConnectEvent(oswin.KeyChordEvent, gi.HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			tvv := recv.Embed(KiT_SliceView).(*SliceView)
			kt := d.(*key.ChordEvent)
			tvv.KeyInputActive(kt)
		})
		sv.ConnectEvent(oswin.DNDEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			de := d.(*dnd.Event)
			tvv := recv.Embed(KiT_SliceView).(*SliceView)
			switch de.Action {
			case dnd.Start:
				tvv.DragNDropStart()
			case dnd.DropOnTarget:
				tvv.DragNDropTarget(de)
			case dnd.DropFmSource:
				tvv.DragNDropSource(de)
			}
		})
	}
}
