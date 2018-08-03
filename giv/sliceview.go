// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  SliceView

// SliceView represents a slice, creating a property editor of the values --
// constructs Children widgets to show the index / value pairs, within an
// overall frame with a button box at the bottom where methods can be invoked
// -- set to Inactive for select-only mode, which emits WidgetSig
// WidgetSelected signals when selection is updated
type SliceView struct {
	gi.Frame
	Slice       interface{} `desc:"the slice that we are a view onto -- must be a pointer to that slice"`
	Values      []ValueView `json:"-" xml:"-" desc:"ValueView representations of the slice values"`
	ShowIndex   bool        `xml:"index" desc:"whether to show index or not -- updated from "index" property (bool) -- index is required for copy / paste and DND of rows"`
	SelectedIdx int         `json:"-" xml:"-" desc:"index of currently-selected item, in Inactive mode only"`
	BuiltSlice  interface{}
	BuiltSize   int
	ViewSig     ki.Signal `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
	TmpSave     ValueView `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
}

var KiT_SliceView = kit.Types.AddType(&SliceView{}, SliceViewProps)

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// SetSlice sets the source slice that we are viewing -- rebuilds the children
// to represent this slice
func (sv *SliceView) SetSlice(sl interface{}, tmpSave ValueView) {
	updt := false
	if sv.Slice != sl {
		updt = sv.UpdateStart()
		sv.Slice = sl
	}
	sv.TmpSave = tmpSave
	sv.UpdateFromSlice()
	sv.UpdateEnd(updt)
}

var SliceViewProps = ki.Props{
	"background-color": &gi.Prefs.BackgroundColor,
}

// SetFrame configures view as a frame
func (sv *SliceView) SetFrame() {
	sv.Lay = gi.LayoutCol
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *SliceView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Frame, "slice-grid")
	config.Add(gi.KiT_Space, "grid-space")
	config.Add(gi.KiT_Layout, "buttons")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (sv *SliceView) StdConfig() (mods, updt bool) {
	sv.SetFrame()
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

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (sv *SliceView) ButtonBox() (*gi.Layout, int) {
	idx, ok := sv.Children().IndexByName("buttons", 0)
	if !ok {
		return nil, -1
	}
	return sv.KnownChild(idx).(*gi.Layout), idx
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
func (sv *SliceView) ConfigSliceGrid() {
	if kit.IfaceIsNil(sv.Slice) {
		return
	}
	mv := reflect.ValueOf(sv.Slice)
	mvnp := kit.NonPtrValue(mv)
	sz := mvnp.Len()

	if sv.BuiltSlice == sv.Slice && sv.BuiltSize == sz {
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
					svv := recv.EmbeddedStruct(KiT_SliceView).(*SliceView)
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
					if sig == int64(gi.WidgetSelected) {
						wbb := send.(gi.Node2D).AsWidget()
						idx := wbb.KnownProp("slv-index").(int)
						svv := recv.EmbeddedStruct(KiT_SliceView).(*SliceView)
						svv.UpdateSelect(idx, wbb.IsSelected())
					}
				})
			}
		} else {
			vvb := vv.AsValueViewBase()
			vvb.ViewSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				svv, _ := recv.EmbeddedStruct(KiT_SliceView).(*SliceView)
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
				svv := recv.EmbeddedStruct(KiT_SliceView).(*SliceView)
				svv.SliceNewAt(act.Data.(int) + 1)
			})
			delact.SetIcon("minus")
			delact.Tooltip = "delete this element"
			delact.Data = i
			delact.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				act := send.(*gi.Action)
				svv := recv.EmbeddedStruct(KiT_SliceView).(*SliceView)
				svv.SliceDelete(act.Data.(int))
			})
		}
	}
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1 means the end
func (sv *SliceView) SliceNewAt(idx int) {
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
	sv.ConfigSliceGrid()
	sv.ViewSig.Emit(sv.This, 0, nil)
}

// SliceDelete deletes element at given index from slice
func (sv *SliceView) SliceDelete(idx int) {
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
	sv.ConfigSliceGrid()
	sv.ViewSig.Emit(sv.This, 0, nil)
}

// SelectRowWidgets sets the selection state of given row of widgets
func (sv *SliceView) SelectRowWidgets(idx int, sel bool) {
	sg, _ := sv.SliceGrid()
	nWidgPerRow, _ := sv.RowWidgetNs()
	rowidx := idx * nWidgPerRow
	if sv.ShowIndex {
		if sg.Kids.IsValidIndex(rowidx) {
			widg := sg.KnownChild(rowidx).(gi.Node2D).AsNode2D()
			widg.SetSelectedState(sel)
			widg.UpdateSig()
		}
	}
	if sg.Kids.IsValidIndex(rowidx + 1) {
		widg := sg.KnownChild(rowidx + 1).(gi.Node2D).AsNode2D()
		widg.SetSelectedState(sel)
		widg.UpdateSig()
	}
}

// UpdateSelect updates the selection for the given index
func (sv *SliceView) UpdateSelect(idx int, sel bool) {
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
}

// ConfigSliceButtons configures the buttons for map functions
func (sv *SliceView) ConfigSliceButtons() {
	if kit.IfaceIsNil(sv.Slice) {
		return
	}
	bb, _ := sv.ButtonBox()
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Button, "Add")
	mods, updt := bb.ConfigChildren(config, false)
	addb := bb.KnownChildByName("Add", 0).EmbeddedStruct(gi.KiT_Button).(*gi.Button)
	addb.SetText("Add")
	addb.ButtonSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			svv := recv.EmbeddedStruct(KiT_SliceView).(*SliceView)
			svv.SliceNewAt(-1)
		}
	})
	if mods {
		bb.UpdateEnd(updt)
	}
}

func (sv *SliceView) UpdateFromSlice() {
	mods, updt := sv.StdConfig()
	sv.ConfigSliceGrid()
	sv.ConfigSliceButtons()
	if mods {
		sv.SetFullReRender()
		sv.UpdateEnd(updt)
	}
}

func (sv *SliceView) UpdateValues() {
	updt := sv.UpdateStart()
	for _, vv := range sv.Values {
		vv.UpdateWidget()
	}
	sv.UpdateEnd(updt)
}

////////////////////////////////////////////////////////////////////////////////////////
//  SliceViewInline

// SliceViewInline represents a slice as a single line widget, for smaller slices and those explicitly marked inline -- constructs widgets in Parts to show the key names and editor vals for each value
type SliceViewInline struct {
	gi.PartsWidgetBase
	Slice   interface{} `desc:"the slice that we are a view onto"`
	Values  []ValueView `json:"-" xml:"-" desc:"ValueView representations of the fields"`
	TmpSave ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig ki.Signal   `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
}

var KiT_SliceViewInline = kit.Types.AddType(&SliceViewInline{}, SliceViewInlineProps)

// SetSlice sets the source slice that we are viewing -- rebuilds the children to represent this slice
func (sv *SliceViewInline) SetSlice(sl interface{}, tmpSave ValueView) {
	updt := false
	if sv.Slice != sl {
		updt = sv.UpdateStart()
		sv.Slice = sl
	}
	sv.TmpSave = tmpSave
	sv.UpdateFromSlice()
	sv.UpdateEnd(updt)
}

var SliceViewInlineProps = ki.Props{
	"min-width": units.NewValue(20, units.Ex),
}

// ConfigParts configures Parts for the current slice
func (sv *SliceViewInline) ConfigParts() {
	if kit.IfaceIsNil(sv.Slice) {
		return
	}
	sv.Parts.Lay = gi.LayoutRow
	config := kit.TypeAndNameList{}
	// always start fresh!
	sv.Values = make([]ValueView, 0)

	mv := reflect.ValueOf(sv.Slice)
	mvnp := kit.NonPtrValue(mv)

	sz := mvnp.Len()
	for i := 0; i < sz; i++ {
		val := kit.OnePtrValue(mvnp.Index(i)) // deal with pointer lists
		vv := ToValueView(val.Interface())
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetSliceValue(val, sv.Slice, i, sv.TmpSave)
		vtyp := vv.WidgetType()
		idxtxt := fmt.Sprintf("%05d", i)
		labnm := fmt.Sprintf("index-%v", idxtxt)
		valnm := fmt.Sprintf("value-%v", idxtxt)
		config.Add(gi.KiT_Label, labnm)
		config.Add(vtyp, valnm)
		sv.Values = append(sv.Values, vv)
	}
	config.Add(gi.KiT_Action, "EditAction")
	mods, updt := sv.Parts.ConfigChildren(config, false)
	if !mods {
		updt = sv.Parts.UpdateStart()
	}
	for i, vv := range sv.Values {
		vvb := vv.AsValueViewBase()
		vvb.ViewSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.EmbeddedStruct(KiT_SliceViewInline).(*SliceViewInline)
			svv.UpdateSig()
			svv.ViewSig.Emit(svv.This, 0, nil)
		})
		lbl := sv.Parts.KnownChild(i * 2).(*gi.Label)
		idxtxt := fmt.Sprintf("%05d", i)
		lbl.Text = idxtxt
		widg := sv.Parts.KnownChild((i * 2) + 1).(gi.Node2D)
		vv.ConfigWidget(widg)
	}
	edack, ok := sv.Parts.Children().ElemFromEnd(0)
	if ok {
		edac := edack.(*gi.Action)
		edac.SetIcon("edit")
		edac.Tooltip = "edit slice in a dialog window"
		edac.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.EmbeddedStruct(KiT_SliceViewInline).(*SliceViewInline)
			dlg := SliceViewDialog(svv.Viewport, svv.Slice, false, svv.TmpSave, "Slice Value View", "", nil, nil)
			svvvk, ok := dlg.Frame().Children().ElemByType(KiT_SliceView, true, 2)
			if ok {
				svvv := svvvk.(*SliceView)
				svvv.ViewSig.ConnectOnly(svv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					svvvv, _ := recv.EmbeddedStruct(KiT_SliceViewInline).(*SliceViewInline)
					svvvv.ViewSig.Emit(svvvv.This, 0, nil)
				})
			}
		})
	}
	sv.Parts.UpdateEnd(updt)
}

func (sv *SliceViewInline) UpdateFromSlice() {
	sv.ConfigParts()
}

func (sv *SliceViewInline) UpdateValues() {
	updt := sv.UpdateStart()
	for _, vv := range sv.Values {
		vv.UpdateWidget()
	}
	sv.UpdateEnd(updt)
}

func (sv *SliceViewInline) Render2D() {
	if sv.FullReRenderIfNeeded() {
		return
	}
	if sv.PushBounds() {
		sv.ConfigParts()
		sv.Render2DParts()
		sv.Render2DChildren()
		sv.PopBounds()
	}
}
