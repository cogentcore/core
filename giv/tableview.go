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
	"strings"
	"time"

	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// todo:
// * search option, both as a search field and as simple type-to-search
// * popup menu option -- when user does right-mouse on item, a provided func is called
//   -- use in fileview
// * could have a native context menu for add / delete etc.
// * emit TableViewSigs

// TableViewWaitCursorSize is the length of the slice above which a wait
// cursor will be displayed while updating the table
var TableViewWaitCursorSize = 5000

// TableView represents a slice-of-structs as a table, where the fields are
// the columns, within an overall frame.  It has two modes, determined by
// Inactive flag: if Inactive, it functions as a mutually-exclusive item
// selector, highlighting the selected row and emitting a WidgetSig
// WidgetSelected signal, and TableViewDoubleClick for double clicks (can be
// used for closing dialogs).  If !Inactive, it is a full-featured editor with
// multiple-selection, cut-and-paste, and drag-and-drop, reporting each action
// taken using the TableViewSig signals
type TableView struct {
	gi.Frame
	Slice        interface{}        `view:"-" json:"-" xml:"-" desc:"the slice that we are a view onto -- must be a pointer to that slice"`
	StyleFunc    TableViewStyleFunc `view:"-" json:"-" xml:"-" desc:"optional styling function"`
	Values       [][]ValueView      `json:"-" xml:"-" desc:"ValueView representations of the slice field values -- outer dimension is fields, inner is rows (generally more rows than fields, so this minimizes number of slices allocated)"`
	ShowIndex    bool               `xml:"index" desc:"whether to show index or not (default true) -- updated from "index" property (bool)"`
	InactKeyNav  bool               `xml:"inact-key-nav" desc:"support key navigation when inactive (default true) -- updated from "intact-key-nav" property (bool) -- no focus really plausible in inactive case, so it uses a low-pri capture of up / down events"`
	SelField     string             `view:"-" json:"-" xml:"-" desc:"current selection field -- initially select value in this field"`
	SelVal       interface{}        `view:"-" json:"-" xml:"-" desc:"current selection value -- initially select this value in SelField"`
	SelectedIdx  int                `json:"-" xml:"-" desc:"index (row) of currently-selected item (-1 if none) -- see SelectedRows for full set of selected rows in active editing mode"`
	SortIdx      int                `desc:"current sort index"`
	SortDesc     bool               `desc:"whether current sort order is descending"`
	SelectMode   bool               `desc:"editing-mode select rows mode"`
	SelectedRows map[int]bool       `desc:"list of currently-selected rows"`
	DraggedRows  []int              `desc:"list of currently-dragged rows"`
	TableViewSig ki.Signal          `json:"-" xml:"-" desc:"table view interactive editing signals"`
	ViewSig      ki.Signal          `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`

	TmpSave      ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	BuiltSlice   interface{} `view:"-" json:"-" xml:"-" desc:"the built slice"`
	BuiltSize    int
	ToolbarSlice interface{} `desc:"the slice that we successfully set a toolbar for"`
	StruType     reflect.Type
	NVisFields   int
	VisFields    []reflect.StructField `view:"-" json:"-" xml:"-" desc:"the visible fields"`
	inFocusGrab  bool
}

var KiT_TableView = kit.Types.AddType(&TableView{}, TableViewProps)

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// TableViewStyleFunc is a styling function for custom styling /
// configuration of elements in the view
type TableViewStyleFunc func(tv *TableView, slice interface{}, widg gi.Node2D, row, col int, vv ValueView)

// SetSlice sets the source slice that we are viewing -- rebuilds the children
// to represent this slice
func (tv *TableView) SetSlice(sl interface{}, tmpSave ValueView) {
	updt := false
	if tv.Slice != sl {
		if !tv.IsInactive() {
			tv.SelectedIdx = -1
		}
		tv.SortIdx = -1
		tv.SortDesc = false
		slpTyp := reflect.TypeOf(sl)
		if slpTyp.Kind() != reflect.Ptr {
			log.Printf("TableView requires that you pass a pointer to a slice of struct elements -- type is not a Ptr: %v\n", slpTyp.String())
			return
		}
		if slpTyp.Elem().Kind() != reflect.Slice {
			log.Printf("TableView requires that you pass a pointer to a slice of struct elements -- ptr doesn't point to a slice: %v\n", slpTyp.Elem().String())
			return
		}
		tv.Slice = sl
		struTyp := tv.StructType()
		if struTyp.Kind() != reflect.Struct {
			log.Printf("TableView requires that you pass a slice of struct elements -- type is not a Struct: %v\n", struTyp.String())
			return
		}
		updt = tv.UpdateStart()
		tv.SelectedRows = make(map[int]bool, 10)
		tv.SelectMode = false
		tv.SetFullReRender()
	}
	tv.ShowIndex = true
	if sidxp, ok := tv.Prop("index"); ok {
		tv.ShowIndex, _ = kit.ToBool(sidxp)
	}
	tv.InactKeyNav = true
	if siknp, ok := tv.Prop("inact-key-nav"); ok {
		tv.InactKeyNav, _ = kit.ToBool(siknp)
	}
	tv.TmpSave = tmpSave
	tv.UpdateFromSlice()
	tv.UpdateEnd(updt)
}

var TableViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}

// TableViewSignals are signals that tableview can send, mostly for editing
// mode.  Selection events are sent on WidgetSig WidgetSelected signals in
// both modes.
type TableViewSignals int64

const (
	// TableViewDoubleClicked emitted during inactive mode when item
	// double-clicked -- can be used for accepting dialog.
	TableViewDoubleClicked TableViewSignals = iota

	// todo: add more signals as needed

	TableViewSignalsN
)

//go:generate stringer -type=TableViewSignals

// StructType returns the type of the struct within the slice, and the number
// of visible fields
func (tv *TableView) StructType() reflect.Type {
	tv.StruType = kit.NonPtrType(reflect.TypeOf(tv.Slice).Elem().Elem())
	return tv.StruType
}

// CacheVisFields computes the number of visible fields in nVisFields and
// caches those to skip in fieldSkip
func (tv *TableView) CacheVisFields() {
	tv.StructType()
	nfld := tv.StruType.NumField()
	tv.VisFields = make([]reflect.StructField, 0, nfld)
	for fli := 0; fli < nfld; fli++ {
		fld := tv.StruType.Field(fli)
		tvtag := fld.Tag.Get("tableview")
		if tvtag != "" {
			if tvtag == "-" {
				continue
			} else if tvtag == "-select" && tv.IsInactive() {
				continue
			} else if tvtag == "-edit" && !tv.IsInactive() {
				continue
			}
		}
		tv.VisFields = append(tv.VisFields, fld)
	}
	tv.NVisFields = len(tv.VisFields)
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (tv *TableView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_Frame, "slice-frame")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (tv *TableView) StdConfig() (mods, updt bool) {
	tv.Lay = gi.LayoutVert
	tv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := tv.StdFrameConfig()
	mods, updt = tv.ConfigChildren(config, false)
	return
}

// SliceFrame returns the outer frame widget, which contains all the header,
// fields and values, and its index, within frame -- nil, -1 if not found
func (tv *TableView) SliceFrame() (*gi.Frame, int) {
	idx, ok := tv.Children().IndexByName("slice-frame", 0)
	if !ok {
		return nil, -1
	}
	return tv.KnownChild(idx).(*gi.Frame), idx
}

// SliceGrid returns the SliceGrid grid frame widget, which contains all the
// fields and values, within SliceFrame
func (tv *TableView) SliceGrid() *gi.Frame {
	sf, _ := tv.SliceFrame()
	if sf == nil {
		return nil
	}
	return sf.KnownChild(2).(*gi.Frame)
}

// ToolBar returns the toolbar widget
func (tv *TableView) ToolBar() *gi.ToolBar {
	idx, ok := tv.Children().IndexByName("toolbar", 0)
	if !ok {
		return nil
	}
	return tv.KnownChild(idx).(*gi.ToolBar)
}

// StdSliceFrameConfig returns a TypeAndNameList for configuring the slice-frame
func (tv *TableView) StdSliceFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "header")
	config.Add(gi.KiT_Separator, "head-sepe")
	config.Add(gi.KiT_Frame, "grid")
	return config
}

// RowWidgetNs returns number of widgets per row and offset for index label
func (tv *TableView) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 1 + tv.NVisFields
	if !tv.IsInactive() {
		nWidgPerRow += 2
	}
	idxOff = 1
	if !tv.ShowIndex {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

// ConfigSliceGrid configures the SliceGrid for the current slice
func (tv *TableView) ConfigSliceGrid(forceUpdt bool) {
	if kit.IfaceIsNil(tv.Slice) {
		return
	}
	mv := reflect.ValueOf(tv.Slice)
	mvnp := kit.NonPtrValue(mv)
	sz := mvnp.Len()

	if !forceUpdt && tv.BuiltSlice == tv.Slice && tv.BuiltSize == sz {
		return
	}
	tv.BuiltSlice = tv.Slice
	tv.BuiltSize = sz

	tv.CacheVisFields()

	nWidgPerRow, idxOff := tv.RowWidgetNs()

	// always start fresh!
	tv.Values = make([][]ValueView, tv.NVisFields)
	for fli := 0; fli < tv.NVisFields; fli++ {
		tv.Values[fli] = make([]ValueView, sz)
	}

	sg, _ := tv.SliceFrame()
	if sg == nil {
		return
	}
	sg.Lay = gi.LayoutVert
	sg.SetMinPrefWidth(units.NewValue(10, units.Em))
	sg.SetStretchMaxHeight() // for this to work, ALL layers above need it too
	sg.SetStretchMaxWidth()  // for this to work, ALL layers above need it too

	if sz > TableViewWaitCursorSize {
		oswin.TheApp.Cursor().Push(cursor.Wait)
		defer oswin.TheApp.Cursor().Pop()
	}

	sgcfg := tv.StdSliceFrameConfig()
	modsg, updtg := sg.ConfigChildren(sgcfg, false)
	if modsg {
		tv.SetFullReRender()
	} else {
		updtg = sg.UpdateStart()
	}

	sgh := sg.KnownChild(0).(*gi.ToolBar)
	sgh.Lay = gi.LayoutHoriz
	// sgh.SetStretchMaxWidth()

	sep := sg.KnownChild(1).(*gi.Separator)
	sep.Horiz = true
	sep.SetStretchMaxWidth()

	sgf := sg.KnownChild(2).(*gi.Frame)
	sgf.Lay = gi.LayoutGrid
	sgf.Stripes = gi.RowStripes

	// setting a pref here is key for giving it a scrollbar in larger context
	sgf.SetMinPrefHeight(units.NewValue(10, units.Em))
	sgf.SetStretchMaxHeight() // for this to work, ALL layers above need it too
	sgf.SetStretchMaxWidth()  // for this to work, ALL layers above need it too
	sgf.SetProp("columns", nWidgPerRow)

	// Configure Header
	hcfg := kit.TypeAndNameList{}
	if tv.ShowIndex {
		hcfg.Add(gi.KiT_Label, "head-idx")
	}
	for fli := 0; fli < tv.NVisFields; fli++ {
		fld := tv.VisFields[fli]
		labnm := fmt.Sprintf("head-%v", fld.Name)
		hcfg.Add(gi.KiT_Action, labnm)
	}
	if !tv.IsInactive() {
		hcfg.Add(gi.KiT_Label, "head-add")
		hcfg.Add(gi.KiT_Label, "head-del")
	}

	modsh, updth := sgh.ConfigChildren(hcfg, false)
	if modsh {
		tv.SetFullReRender()
	} else {
		updth = sgh.UpdateStart()
	}
	if tv.ShowIndex {
		lbl := sgh.KnownChild(0).(*gi.Label)
		lbl.Text = "Index"
	}
	for fli := 0; fli < tv.NVisFields; fli++ {
		fld := tv.VisFields[fli]
		hdr := sgh.KnownChild(idxOff + fli).(*gi.Action)
		hdr.SetText(fld.Name)
		if fli == tv.SortIdx {
			if tv.SortDesc {
				hdr.SetIcon("widget-wedge-down")
			} else {
				hdr.SetIcon("widget-wedge-up")
			}
		}
		hdr.Data = fli
		hdr.Tooltip = "click to sort by this column -- toggles direction of sort too"
		hdr.ActionSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TableView).(*TableView)
			act := send.(*gi.Action)
			fldIdx := act.Data.(int)
			tvv.SortSliceAction(fldIdx)
		})
	}
	if !tv.IsInactive() {
		lbl := sgh.KnownChild(tv.NVisFields + idxOff).(*gi.Label)
		lbl.Text = "+"
		lbl.Tooltip = "insert row"
		lbl = sgh.KnownChild(tv.NVisFields + idxOff + 1).(*gi.Label)
		lbl.Text = "-"
		lbl.Tooltip = "delete row"
	}

	sgf.DeleteChildren(true)
	sgf.Kids = make(ki.Slice, nWidgPerRow*sz)

	if tv.SortIdx >= 0 {
		StructSliceSort(tv.Slice, tv.SortIdx, !tv.SortDesc)
	}
	tv.ConfigSliceGridRows()

	sg.SetFullReRender()
	sgh.UpdateEnd(updth)
	sg.UpdateEnd(updtg)
}

// ConfigSliceGridRows configures the SliceGrid rows for the current slice --
// assumes .Kids is created at the right size -- only call this for a direct
// re-render e.g., after sorting
func (tv *TableView) ConfigSliceGridRows() {
	mv := reflect.ValueOf(tv.Slice)
	mvnp := kit.NonPtrValue(mv)
	sz := mvnp.Len()

	if sz > TableViewWaitCursorSize {
		oswin.TheApp.Cursor().Push(cursor.Wait)
		defer oswin.TheApp.Cursor().Pop()
	}

	nWidgPerRow, idxOff := tv.RowWidgetNs()
	sg, _ := tv.SliceFrame()
	sgf := sg.KnownChild(2).(*gi.Frame)

	updt := sgf.UpdateStart()
	defer sgf.UpdateEnd(updt)

	for i := 0; i < sz; i++ {
		ridx := i * nWidgPerRow
		val := kit.OnePtrValue(mvnp.Index(i)) // deal with pointer lists
		stru := val.Interface()
		idxtxt := fmt.Sprintf("%05d", i)
		labnm := fmt.Sprintf("index-%v", idxtxt)
		if tv.ShowIndex {
			var idxlab *gi.Label
			if sgf.Kids[ridx] != nil {
				idxlab = sgf.Kids[ridx].(*gi.Label)
			} else {
				idxlab = &gi.Label{}
				sgf.SetChild(idxlab, ridx, labnm)
			}
			idxlab.Text = idxtxt
			idxlab.SetProp("tv-index", i)
			idxlab.Selectable = true
			idxlab.WidgetSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.WidgetSelected) {
					wbb := send.(gi.Node2D).AsWidget()
					idx := wbb.KnownProp("tv-index").(int)
					tvv := recv.Embed(KiT_TableView).(*TableView)
					tvv.UpdateSelect(idx, wbb.IsSelected())
				}
			})
		}

		for fli := 0; fli < tv.NVisFields; fli++ {
			field := tv.VisFields[fli]
			fval := val.Elem().Field(field.Index[0])
			vv := ToValueView(fval.Interface())
			if vv == nil { // shouldn't happen
				continue
			}
			vv.SetStructValue(fval.Addr(), stru, &field, tv.TmpSave)
			vtyp := vv.WidgetType()
			valnm := fmt.Sprintf("value-%v.%v", fli, idxtxt)
			cidx := ridx + idxOff + fli
			var widg gi.Node2D
			if sgf.Kids[cidx] != nil {
				widg = sgf.Kids[cidx].(gi.Node2D)
			} else {
				tv.Values[fli][i] = vv
				widg = ki.NewOfType(vtyp).(gi.Node2D)
				sgf.SetChild(widg, cidx, valnm)
			}
			vv.ConfigWidget(widg)
			wb := widg.AsWidget()
			if wb != nil {
				wb.SetProp("tv-index", i)
				wb.ClearSelected()
				wb.WidgetSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					if sig == int64(gi.WidgetSelected) || sig == int64(gi.WidgetFocused) {
						wbb := send.(gi.Node2D).AsWidget()
						idx := wbb.KnownProp("tv-index").(int)
						tvv := recv.Embed(KiT_TableView).(*TableView)
						if sig != int64(gi.WidgetFocused) || !tvv.inFocusGrab {
							tvv.UpdateSelect(idx, wbb.IsSelected())
						}
					}
				})
			}
			if tv.IsInactive() {
				widg.AsNode2D().SetInactive()
			} else {
				vvb := vv.AsValueViewBase()
				vvb.ViewSig.ConnectOnly(tv.This, // todo: do we need this?
					func(recv, send ki.Ki, sig int64, data interface{}) {
						tvv, _ := recv.Embed(KiT_TableView).(*TableView)
						tvv.UpdateSig()
						tvv.ViewSig.Emit(tvv.This, 0, nil)
					})

				addnm := fmt.Sprintf("add-%v", idxtxt)
				delnm := fmt.Sprintf("del-%v", idxtxt)
				addact := gi.Action{}
				delact := gi.Action{}
				sgf.SetChild(&addact, ridx+1+tv.NVisFields, addnm)
				sgf.SetChild(&delact, ridx+1+tv.NVisFields+1, delnm)

				addact.SetIcon("plus")
				addact.Tooltip = "insert a new element at this index"
				addact.Data = i
				addact.ActionSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					act := send.(*gi.Action)
					tvv := recv.Embed(KiT_TableView).(*TableView)
					tvv.SliceNewAt(act.Data.(int)+1, true)
				})
				delact.SetIcon("minus")
				delact.Tooltip = "delete this element"
				delact.Data = i
				delact.ActionSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					act := send.(*gi.Action)
					tvv := recv.Embed(KiT_TableView).(*TableView)
					tvv.SliceDelete(act.Data.(int), true)
				})
			}
			if tv.StyleFunc != nil {
				tv.StyleFunc(tv, mvnp.Interface(), widg, i, fli, vv)
			}
		}
	}
	if tv.SelField != "" && tv.SelVal != nil {
		tv.SelectedIdx, _ = StructSliceRowByValue(tv.Slice, tv.SelField, tv.SelVal)
	}
	if tv.IsInactive() && tv.SelectedIdx >= 0 {
		tv.SelectRow(tv.SelectedIdx)
	}
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end -- reconfig means call ConfigSliceGrid to update display
func (tv *TableView) SliceNewAt(idx int, reconfig bool) {
	updt := tv.UpdateStart()
	defer tv.UpdateEnd(updt)

	tvl := reflect.ValueOf(tv.Slice)
	tvnp := kit.NonPtrValue(tvl)
	tvtyp := tvnp.Type()
	nval := reflect.New(tvtyp.Elem())
	sz := tvnp.Len()
	tvnp = reflect.Append(tvnp, nval.Elem())
	if idx >= 0 && idx < sz-1 {
		reflect.Copy(tvnp.Slice(idx+1, sz+1), tvnp.Slice(idx, sz))
		tvnp.Index(idx).Set(nval.Elem())
	}
	tvl.Elem().Set(tvnp)
	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	if reconfig {
		tv.ConfigSliceGrid(true)
	}
	tv.ViewSig.Emit(tv.This, 0, nil)
}

// SliceDelete deletes element at given index from slice -- reconfig means
// call ConfigSliceGrid to update display
func (tv *TableView) SliceDelete(idx int, reconfig bool) {
	if idx < 0 {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEnd(updt)

	tvl := reflect.ValueOf(tv.Slice)
	tvnp := kit.NonPtrValue(tvl)
	tvtyp := tvnp.Type()
	nval := reflect.New(tvtyp.Elem())
	sz := tvnp.Len()
	reflect.Copy(tvnp.Slice(idx, sz-1), tvnp.Slice(idx+1, sz))
	tvnp.Index(sz - 1).Set(nval.Elem())
	tvl.Elem().Set(tvnp.Slice(0, sz-1))
	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	if reconfig {
		tv.ConfigSliceGrid(true)
	}
	tv.ViewSig.Emit(tv.This, 0, nil)
}

// SortSliceAction sorts the slice for given field index -- toggles ascending
// vs. descending if already sorting on this dimension
func (tv *TableView) SortSliceAction(fldIdx int) {
	oswin.TheApp.Cursor().Push(cursor.Wait)
	defer oswin.TheApp.Cursor().Pop()

	sg, _ := tv.SliceFrame()
	sgh := sg.KnownChild(0).(*gi.ToolBar)
	sgh.SetFullReRender()
	idxOff := 1
	if !tv.ShowIndex {
		idxOff = 0
	}

	ascending := true

	for fli := 0; fli < tv.NVisFields; fli++ {
		hdr := sgh.KnownChild(idxOff + fli).(*gi.Action)
		if fli == fldIdx {
			if tv.SortIdx == fli {
				tv.SortDesc = !tv.SortDesc
				ascending = !tv.SortDesc
			} else {
				tv.SortDesc = false
			}
			if ascending {
				hdr.SetIcon("widget-wedge-up")
			} else {
				hdr.SetIcon("widget-wedge-down")
			}
		} else {
			hdr.SetIcon("none")
		}
	}

	tv.SortIdx = fldIdx
	rawIdx := tv.VisFields[fldIdx].Index[0]

	sgf := sg.KnownChild(2).(*gi.Frame)
	sgf.SetFullReRender()

	StructSliceSort(tv.Slice, rawIdx, !tv.SortDesc)
	tv.ConfigSliceGridRows()
}

// StructSliceSort sorts a slice of a struct according to the given field
// (specified by first-order index) and sort direction, using int, float,
// string kind conversions through reflect, and supporting time.Time as well
// -- todo: could extend with a function that handles specific fields
func StructSliceSort(struSlice interface{}, fldIdx int, ascending bool) error {
	mv := reflect.ValueOf(struSlice)
	mvnp := kit.NonPtrValue(mv)
	struTyp := kit.NonPtrType(reflect.TypeOf(struSlice).Elem().Elem())
	if fldIdx < 0 || fldIdx >= struTyp.NumField() {
		err := fmt.Errorf("gi.StructSliceSort: field index out of range: %v must be < %v\n", fldIdx, struTyp.NumField())
		log.Println(err)
		return err
	}
	fld := struTyp.Field(fldIdx)
	vk := fld.Type.Kind()

	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		sort.Slice(mvnp.Interface(), func(i, j int) bool {
			ival := kit.OnePtrValue(mvnp.Index(i))
			iv := ival.Elem().Field(fldIdx).Int()
			jval := kit.OnePtrValue(mvnp.Index(j))
			jv := jval.Elem().Field(fldIdx).Int()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		sort.Slice(mvnp.Interface(), func(i, j int) bool {
			ival := kit.OnePtrValue(mvnp.Index(i))
			iv := ival.Elem().Field(fldIdx).Uint()
			jval := kit.OnePtrValue(mvnp.Index(j))
			jv := jval.Elem().Field(fldIdx).Uint()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		sort.Slice(mvnp.Interface(), func(i, j int) bool {
			ival := kit.OnePtrValue(mvnp.Index(i))
			iv := ival.Elem().Field(fldIdx).Float()
			jval := kit.OnePtrValue(mvnp.Index(j))
			jv := jval.Elem().Field(fldIdx).Float()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
	case vk == reflect.String:
		sort.Slice(mvnp.Interface(), func(i, j int) bool {
			ival := kit.OnePtrValue(mvnp.Index(i))
			iv := ival.Elem().Field(fldIdx).String()
			jval := kit.OnePtrValue(mvnp.Index(j))
			jv := jval.Elem().Field(fldIdx).String()
			if ascending {
				return strings.ToLower(iv) < strings.ToLower(jv)
			} else {
				return strings.ToLower(iv) > strings.ToLower(jv)
			}
		})
	case vk == reflect.Struct && kit.FullTypeName(fld.Type) == "giv.FileTime":
		sort.Slice(mvnp.Interface(), func(i, j int) bool {
			ival := kit.OnePtrValue(mvnp.Index(i))
			iv := (time.Time)(ival.Elem().Field(fldIdx).Interface().(FileTime))
			jval := kit.OnePtrValue(mvnp.Index(j))
			jv := (time.Time)(jval.Elem().Field(fldIdx).Interface().(FileTime))
			if ascending {
				return iv.Before(jv)
			} else {
				return jv.Before(iv)
			}
		})
	case vk == reflect.Struct && kit.FullTypeName(fld.Type) == "time.Time":
		sort.Slice(mvnp.Interface(), func(i, j int) bool {
			ival := kit.OnePtrValue(mvnp.Index(i))
			iv := ival.Elem().Field(fldIdx).Interface().(time.Time)
			jval := kit.OnePtrValue(mvnp.Index(j))
			jv := jval.Elem().Field(fldIdx).Interface().(time.Time)
			if ascending {
				return iv.Before(jv)
			} else {
				return jv.Before(iv)
			}
		})
	default:
		err := fmt.Errorf("SortStructSlice: unable to sort on field of type: %v\n", fld.Type.String())
		log.Println(err)
		return err
	}
	return nil
}

// ConfigToolbar configures the toolbar actions
func (tv *TableView) ConfigToolbar() {
	if kit.IfaceIsNil(tv.Slice) || tv.IsInactive() {
		return
	}
	if tv.ToolbarSlice == tv.Slice {
		return
	}
	tb := tv.ToolBar()
	if len(*tb.Children()) == 0 {
		tb.SetStretchMaxWidth()
		addac := tb.AddNewChild(gi.KiT_Action, "Add").(*gi.Action)
		addac.SetText("Add")
		addac.ActionSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TableView).(*TableView)
			tvv.SliceNewAt(-1, true)
		})
	}
	sz := len(*tb.Children())
	if sz > 1 {
		for i := sz - 1; i >= 1; i-- {
			tb.DeleteChildAtIndex(i, true)
		}
	}
	if HasToolBarView(tv.Slice) {
		ToolBarView(tv.Slice, tv.Viewport, tb)
	}
	tv.ToolbarSlice = tv.Slice
}

// SortFieldName returns the name of the field being sorted, along with :up or
// :down depending on descending
func (tv *TableView) SortFieldName() string {
	if tv.SortIdx >= 0 && tv.SortIdx < tv.NVisFields {
		nm := tv.VisFields[tv.SortIdx].Name
		if tv.SortDesc {
			nm += ":down"
		} else {
			nm += ":up"
		}
		return nm
	}
	return ""
}

// SetSortField sets sorting to happen on given field and direction -- see
// SortFieldName for details
func (tv *TableView) SetSortFieldName(nm string) {
	if nm == "" {
		return
	}
	spnm := strings.Split(nm, ":")
	for fli := 0; fli < tv.NVisFields; fli++ {
		fld := tv.VisFields[fli]
		if fld.Name == spnm[0] {
			tv.SortIdx = fli
		}
	}
	if len(spnm) == 2 {
		if spnm[1] == "down" {
			tv.SortDesc = true
		} else {
			tv.SortDesc = false
		}
	}
}

func (tv *TableView) UpdateFromSlice() {
	mods, updt := tv.StdConfig()
	tv.ConfigSliceGrid(false)
	tv.ConfigToolbar()
	if mods {
		tv.SetFullReRender()
		tv.UpdateEnd(updt)
	}
}

func (tv *TableView) UpdateValues() {
	updt := tv.UpdateStart()
	for _, vv := range tv.Values {
		for _, vvf := range vv {
			vvf.UpdateWidget()
		}
	}
	tv.UpdateEnd(updt)
}

func (tv *TableView) Layout2D(parBBox image.Rectangle, iter int) bool {
	redo := tv.Frame.Layout2D(parBBox, iter)
	sg, _ := tv.SliceFrame()
	if sg == nil {
		return redo
	}
	idxOff := 1
	if !tv.ShowIndex {
		idxOff = 0
	}

	nfld := tv.NVisFields + idxOff
	sgh := sg.KnownChild(0).(*gi.ToolBar)
	sgf := sg.KnownChild(2).(*gi.Frame)
	if len(sgf.Kids) >= nfld {
		sgh.SetProp("max-width", units.NewValue(sgf.LayData.AllocSize.X, units.Dot))
		for fli := 0; fli < nfld; fli++ {
			lbl := sgh.KnownChild(fli).(gi.Node2D).AsWidget()
			widg := sgf.KnownChild(fli).(gi.Node2D).AsWidget()
			lbl.SetProp("width", units.NewValue(widg.LayData.AllocSize.X, units.Dot))
		}
		sgh.Layout2D(parBBox, iter)
	}
	return redo
}

func (tv *TableView) Render2D() {
	if tv.FullReRenderIfNeeded() {
		return
	}
	if tv.PushBounds() {
		tv.FrameStdRender()
		tv.TableViewEvents()
		tv.RenderScrolls()
		tv.Render2DChildren()
		tv.PopBounds()
		if tv.SelectedIdx > -1 {
			tv.ScrollToRow(tv.SelectedIdx)
		}
	} else {
		tv.DisconnectAllEvents(gi.AllPris)
	}
}

func (tv *TableView) HasFocus2D() bool {
	if tv.IsInactive() {
		return tv.InactKeyNav
	}
	return tv.ContainsFocus() // anyone within us gives us focus..
}

//////////////////////////////////////////////////////////////////////////////
//  Row access methods

// RowStruct returns struct interface at given row
func (tv *TableView) RowStruct(row int) interface{} {
	mv := reflect.ValueOf(tv.Slice)
	mvnp := kit.NonPtrValue(mv)
	sz := mvnp.Len()
	if row < 0 || row >= sz {
		fmt.Printf("giv.TableView: row index out of range: %v\n", row)
		return nil
	}
	val := kit.OnePtrValue(mvnp.Index(row)) // deal with pointer lists
	stru := val.Interface()
	return stru
}

// RowFirstWidget returns the first widget for given row (could be index or
// not) -- false if out of range
func (tv *TableView) RowFirstWidget(row int) (*gi.WidgetBase, bool) {
	if tv.RowStruct(row) == nil { // range check
		return nil, false
	}
	nWidgPerRow, _ := tv.RowWidgetNs()
	sg, _ := tv.SliceFrame()
	if sg == nil {
		return nil, false
	}
	sgf := sg.KnownChild(2).(*gi.Frame)
	widg := sgf.Kids[row*nWidgPerRow].(gi.Node2D).AsWidget()
	return widg, true
}

// RowGrabFocus grabs the focus for the first focusable widget in given row --
// returns that element or nil if not successful -- note: grid must have
// already rendered for focus to be grabbed!
func (tv *TableView) RowGrabFocus(row int) *gi.WidgetBase {
	if tv.RowStruct(row) == nil || tv.inFocusGrab { // range check
		return nil
	}
	// fmt.Printf("grab row focus: %v\n", row)
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	sg, _ := tv.SliceFrame()
	if sg == nil {
		return nil
	}
	ridx := nWidgPerRow * row
	sgf := sg.KnownChild(2).(*gi.Frame)
	// first check if we already have focus
	for fli := 0; fli < tv.NVisFields; fli++ {
		widg := sgf.KnownChild(ridx + idxOff + fli).(gi.Node2D).AsWidget()
		if widg.HasFocus() {
			return widg
		}
	}
	tv.inFocusGrab = true
	defer func() { tv.inFocusGrab = false }()
	for fli := 0; fli < tv.NVisFields; fli++ {
		widg := sgf.KnownChild(ridx + idxOff + fli).(gi.Node2D).AsWidget()
		if widg.CanFocus() {
			widg.GrabFocus()
			return widg
		}
	}
	return nil
}

// RowPos returns center of window position of index label for row (ContextMenuPos)
func (tv *TableView) RowPos(row int) image.Point {
	var pos image.Point
	widg, ok := tv.RowFirstWidget(row)
	if ok {
		pos = widg.ContextMenuPos()
	}
	return pos
}

// RowFromPos returns the row that contains given vertical position, false if not found
func (tv *TableView) RowFromPos(posY int) (int, bool) {
	// todo: could optimize search to approx loc, and search up / down from there
	for rw := 0; rw < tv.BuiltSize; rw++ {
		widg, ok := tv.RowFirstWidget(rw)
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
func (tv *TableView) ScrollToRow(row int) bool {
	sg, _ := tv.SliceFrame()
	sgf := sg.KnownChild(2).(*gi.Frame)
	if widg, ok := tv.RowFirstWidget(row); ok {
		return sgf.ScrollToItem(widg)
	}
	return false
}

// SelectFieldVal sets SelField and SelVal and attempts to find corresponding
// row, setting SelectedIdx and selecting row if found -- returns true if
// found, false otherwise
func (tv *TableView) SelectFieldVal(fld, val string) bool {
	tv.SelField = fld
	tv.SelVal = val
	if tv.SelField != "" && tv.SelVal != nil {
		idx, _ := StructSliceRowByValue(tv.Slice, tv.SelField, tv.SelVal)
		if idx >= 0 {
			tv.ScrollToRow(idx)
			tv.UpdateSelect(idx, true)
			return true
		}
	}
	return false
}

// StructSliceRowByValue searches for first row that contains given value in field of
// given name.
func StructSliceRowByValue(struSlice interface{}, fldName string, fldVal interface{}) (int, error) {
	mv := reflect.ValueOf(struSlice)
	mvnp := kit.NonPtrValue(mv)
	sz := mvnp.Len()
	struTyp := kit.NonPtrType(reflect.TypeOf(struSlice).Elem().Elem())
	fld, ok := struTyp.FieldByName(fldName)
	if !ok {
		err := fmt.Errorf("gi.StructSliceRowByValue: field name: %v not found\n", fldName)
		log.Println(err)
		return -1, err
	}
	fldIdx := fld.Index[0]
	for row := 0; row < sz; row++ {
		rval := kit.OnePtrValue(mvnp.Index(row))
		fval := rval.Elem().Field(fldIdx)
		if fval.Interface() == fldVal {
			return row, nil
		}
	}
	return -1, nil
}

/////////////////////////////////////////////////////////////////////////////
//    Moving

// MoveDown moves the selection down to next row, using given select mode
// (from keyboard modifiers) -- returns newly selected row or -1 if failed
func (tv *TableView) MoveDown(selMode mouse.SelectModes) int {
	if selMode == mouse.NoSelectMode {
		if tv.SelectMode {
			selMode = mouse.ExtendContinuous
		}
	}
	if tv.SelectedIdx >= tv.BuiltSize-1 {
		tv.SelectedIdx = tv.BuiltSize - 1
		return -1
	}
	tv.SelectedIdx++
	tv.SelectRowAction(tv.SelectedIdx, selMode)
	return tv.SelectedIdx
}

// MoveDownAction moves the selection down to next row, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected
// row
func (tv *TableView) MoveDownAction(selMode mouse.SelectModes) int {
	nrow := tv.MoveDown(selMode)
	if nrow >= 0 {
		tv.ScrollToRow(nrow)
		tv.WidgetSig.Emit(tv.This, int64(gi.WidgetSelected), nrow)
	}
	return nrow
}

// MoveUp moves the selection up to previous row, using given select mode
// (from keyboard modifiers) -- returns newly selected row or -1 if failed
func (tv *TableView) MoveUp(selMode mouse.SelectModes) int {
	if selMode == mouse.NoSelectMode {
		if tv.SelectMode {
			selMode = mouse.ExtendContinuous
		}
	}
	if tv.SelectedIdx <= 0 {
		tv.SelectedIdx = 0
		return -1
	}
	tv.SelectedIdx--
	tv.SelectRowAction(tv.SelectedIdx, selMode)
	return tv.SelectedIdx
}

// MoveUpAction moves the selection up to previous row, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected
// row
func (tv *TableView) MoveUpAction(selMode mouse.SelectModes) int {
	nrow := tv.MoveUp(selMode)
	if nrow >= 0 {
		tv.ScrollToRow(nrow)
		tv.WidgetSig.Emit(tv.This, int64(gi.WidgetSelected), nrow)
	}
	return nrow
}

//////////////////////////////////////////////////////////////////////////////
//    Selection: user operates on the index labels

// SelectRowWidgets sets the selection state of given row of widgets
func (tv *TableView) SelectRowWidgets(idx int, sel bool) {
	if idx < 0 {
		return
	}
	var win *gi.Window
	if tv.Viewport != nil {
		win = tv.Viewport.Win
	}
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	sg, _ := tv.SliceFrame()
	sgf := sg.KnownChild(2).(*gi.Frame)
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	ridx := idx * nWidgPerRow
	for fli := 0; fli < tv.NVisFields; fli++ {
		seldx := ridx + idxOff + fli
		if sgf.Kids.IsValidIndex(seldx) {
			widg := sgf.KnownChild(seldx).(gi.Node2D).AsNode2D()
			widg.SetSelectedState(sel)
			widg.UpdateSig()
		}
	}
	if idxOff == 1 {
		if sgf.Kids.IsValidIndex(ridx) {
			widg := sgf.KnownChild(ridx).(gi.Node2D).AsNode2D()
			widg.SetSelectedState(sel)
			widg.UpdateSig()
		}
	}

	if win != nil {
		win.UpdateEnd(updt)
	}
}

// UpdateSelect updates the selection for the given index -- callback from widgetsig select
func (tv *TableView) UpdateSelect(idx int, sel bool) {
	if tv.IsInactive() {
		if tv.SelectedIdx >= 0 { // unselect current
			tv.SelectRowWidgets(tv.SelectedIdx, false)
		}
		if sel {
			tv.SelectedIdx = idx
			tv.SelectRowWidgets(tv.SelectedIdx, true)
		} else {
			tv.SelectedIdx = -1
		}
		tv.WidgetSig.Emit(tv.This, int64(gi.WidgetSelected), tv.SelectedIdx)
	} else {
		selMode := mouse.NoSelectMode
		win := tv.Viewport.Win
		if win != nil {
			selMode = win.LastSelMode
		}
		tv.SelectRowAction(idx, selMode)
	}
}

// RowIsSelected returns the selected status of given row index
func (tv *TableView) RowIsSelected(row int) bool {
	if _, ok := tv.SelectedRows[row]; ok {
		return true
	}
	return false
}

// SelectedRowsList returns list of selected rows, sorted either ascending or descending
func (tv *TableView) SelectedRowsList(descendingSort bool) []int {
	rws := make([]int, len(tv.SelectedRows))
	i := 0
	for r, _ := range tv.SelectedRows {
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
func (tv *TableView) SelectRow(row int) {
	tv.SelectedRows[row] = true
	tv.SelectRowWidgets(row, true)
}

// UnselectRow unselects given row (if selected)
func (tv *TableView) UnselectRow(row int) {
	if tv.RowIsSelected(row) {
		delete(tv.SelectedRows, row)
	}
	tv.SelectRowWidgets(row, false)
}

// UnselectAllRows unselects all selected rows
func (tv *TableView) UnselectAllRows() {
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	for r, _ := range tv.SelectedRows {
		tv.SelectRowWidgets(r, false)
	}
	tv.SelectedRows = make(map[int]bool, 10)
	if win != nil {
		win.UpdateEnd(updt)
	}
}

// SelectAllRows selects all rows
func (tv *TableView) SelectAllRows() {
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	tv.UnselectAllRows()
	tv.SelectedRows = make(map[int]bool, tv.BuiltSize)
	for row := 0; row < tv.BuiltSize; row++ {
		tv.SelectedRows[row] = true
		tv.SelectRowWidgets(row, true)
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
}

// SelectRowAction is called when a select action has been received (e.g., a
// mouse click) -- translates into selection updates -- gets selection mode
// from mouse event (ExtendContinuous, ExtendOne)
func (tv *TableView) SelectRowAction(row int, mode mouse.SelectModes) {
	if row >= tv.BuiltSize {
		row = tv.BuiltSize - 1
	}
	if row < 0 {
		row = 0
	}
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	switch mode {
	case mouse.ExtendContinuous:
		if len(tv.SelectedRows) == 0 {
			tv.SelectedIdx = row
			tv.SelectRow(row)
			tv.RowGrabFocus(row)
			tv.WidgetSig.Emit(tv.This, int64(gi.WidgetSelected), tv.SelectedIdx)
		} else {
			minIdx := -1
			maxIdx := 0
			for r, _ := range tv.SelectedRows {
				if minIdx < 0 {
					minIdx = r
				} else {
					minIdx = kit.MinInt(minIdx, r)
				}
				maxIdx = kit.MaxInt(maxIdx, r)
			}
			cidx := row
			tv.SelectedIdx = row
			tv.SelectRow(row)
			if row < minIdx {
				for cidx < minIdx {
					r := tv.MoveDown(mouse.SelectModesN) // just select
					cidx = r
				}
			} else if row > maxIdx {
				for cidx > maxIdx {
					r := tv.MoveUp(mouse.SelectModesN) // just select
					cidx = r
				}
			}
			tv.RowGrabFocus(row)
			tv.WidgetSig.Emit(tv.This, int64(gi.WidgetSelected), tv.SelectedIdx)
		}
	case mouse.ExtendOne:
		if tv.RowIsSelected(row) {
			tv.UnselectRowAction(row)
		} else {
			tv.SelectedIdx = row
			tv.SelectRow(row)
			tv.RowGrabFocus(row)
			tv.WidgetSig.Emit(tv.This, int64(gi.WidgetSelected), tv.SelectedIdx)
		}
	case mouse.NoSelectMode:
		if tv.RowIsSelected(row) {
			if len(tv.SelectedRows) > 1 {
				tv.UnselectAllRows()
			}
			tv.SelectedIdx = row
			tv.SelectRow(row)
			tv.RowGrabFocus(row)
		} else {
			tv.UnselectAllRows()
			tv.SelectedIdx = row
			tv.SelectRow(row)
			tv.RowGrabFocus(row)
		}
		tv.WidgetSig.Emit(tv.This, int64(gi.WidgetSelected), tv.SelectedIdx)
	default: // anything else
		tv.SelectedIdx = row
		tv.SelectRow(row)
		tv.RowGrabFocus(row)
		tv.WidgetSig.Emit(tv.This, int64(gi.WidgetSelected), tv.SelectedIdx)
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
}

// UnselectRowAction unselects this row (if selected) -- and emits a signal
func (tv *TableView) UnselectRowAction(row int) {
	if tv.RowIsSelected(row) {
		tv.UnselectRow(row)
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeDataRow adds mimedata for given row: an application/json of the struct
func (tv *TableView) MimeDataRow(md *mimedata.Mimes, row int) {
	stru := tv.RowStruct(row)
	b, err := json.MarshalIndent(stru, "", "  ")
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: mimedata.AppJSON, Data: b})
	} else {
		log.Printf("gi.TableView MimeData JSON Marshall error: %v\n", err)
	}
}

// RowsFromMimeData creates a slice of structs from mime data
func (tv *TableView) RowsFromMimeData(md mimedata.Mimes) []interface{} {
	tvl := reflect.ValueOf(tv.Slice)
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
				log.Printf("gi.TableView RowsFromMimeData: JSON load error: %v\n", err)
			}
		}
	}
	return sl
}

// CopyRows copies selected rows to clip.Board, optionally resetting the selection
func (tv *TableView) CopyRows(reset bool) {
	nitms := len(tv.SelectedRows)
	if nitms == 0 {
		return
	}
	md := make(mimedata.Mimes, 0, nitms)
	for r, _ := range tv.SelectedRows {
		tv.MimeDataRow(&md, r)
	}
	oswin.TheApp.ClipBoard().Write(md)
	if reset {
		tv.UnselectAllRows()
	}
}

// DeleteRows deletes all selected rows
func (tv *TableView) DeleteRows() {
	if len(tv.SelectedRows) == 0 {
		return
	}
	updt := tv.UpdateStart()
	rws := tv.SelectedRowsList(true) // descending sort
	for _, r := range rws {
		tv.SliceDelete(r, false)
	}
	tv.ConfigSliceGrid(true)
	tv.UpdateEnd(updt)
}

// CutRows copies selected rows to clip.Board and deletes selected rows
func (tv *TableView) CutRows() {
	if len(tv.SelectedRows) == 0 {
		return
	}
	updt := tv.UpdateStart()
	tv.CopyRows(false)
	rws := tv.SelectedRowsList(true) // descending sort
	row := rws[0]
	tv.UnselectAllRows()
	for _, r := range rws {
		tv.SliceDelete(r, false)
	}
	tv.ConfigSliceGrid(true)
	tv.UpdateEnd(updt)
	tv.SelectRowAction(row, mouse.NoSelectMode)
}

// Paste pastes clipboard at given row
func (tv *TableView) Paste(row int) {
	md := oswin.TheApp.ClipBoard().Read([]string{mimedata.AppJSON})
	if md != nil {
		tv.PasteAction(md, row)
	}
}

// MakePasteMenu makes the menu of options for paste events
func (tv *TableView) MakePasteMenu(m *gi.Menu, data interface{}, row int) {
	if len(*m) > 0 {
		return
	}
	m.AddAction(gi.ActOpts{Label: "Assign To", Data: data}, tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TableView).(*TableView)
		tvv.PasteAssign(data.(mimedata.Mimes), row)
	})
	m.AddAction(gi.ActOpts{Label: "Insert Before", Data: data}, tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TableView).(*TableView)
		tvv.PasteAtRow(data.(mimedata.Mimes), row)
	})
	m.AddAction(gi.ActOpts{Label: "Insert After", Data: data}, tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TableView).(*TableView)
		tvv.PasteAtRow(data.(mimedata.Mimes), row+1)
	})
	m.AddAction(gi.ActOpts{Label: "Cancel", Data: data}, tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
	})
}

// PasteAction performs a paste from the clipboard using given data -- pops up
// a menu to determine what specifically to do
func (tv *TableView) PasteAction(md mimedata.Mimes, row int) {
	tv.UnselectAllRows()
	var men gi.Menu
	tv.MakePasteMenu(&men, md, row)
	pos := tv.RowPos(row)
	gi.PopupMenu(men, pos.X, pos.Y, tv.Viewport, "tvPasteMenu")
}

// PasteAssign assigns mime data (only the first one!) to this row
func (tv *TableView) PasteAssign(md mimedata.Mimes, row int) {
	tvl := reflect.ValueOf(tv.Slice)
	tvnp := kit.NonPtrValue(tvl)

	sl := tv.RowsFromMimeData(md)
	updt := tv.UpdateStart()
	if len(sl) == 0 {
		return
	}
	ns := sl[0]
	tvnp.Index(row).Set(reflect.ValueOf(ns).Elem())
	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	tv.ConfigSliceGridRows() // no change in length
	tv.UpdateEnd(updt)
}

// PasteAtRow inserts object(s) from mime data at (before) given row
func (tv *TableView) PasteAtRow(md mimedata.Mimes, row int) {
	tvl := reflect.ValueOf(tv.Slice)
	tvnp := kit.NonPtrValue(tvl)

	sl := tv.RowsFromMimeData(md)
	updt := tv.UpdateStart()
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
	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	tv.ConfigSliceGrid(true)
	tv.UpdateEnd(updt)
	tv.SelectRowAction(row, mouse.NoSelectMode)
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// DragNDropStart starts a drag-n-drop
func (tv *TableView) DragNDropStart() {
	nitms := len(tv.SelectedRows)
	if nitms == 0 {
		return
	}
	md := make(mimedata.Mimes, 0, nitms)
	for r, _ := range tv.SelectedRows {
		tv.MimeDataRow(&md, r)
	}
	rws := tv.SelectedRowsList(true) // descending sort
	widg, ok := tv.RowFirstWidget(rws[0])
	if ok {
		bi := &gi.Bitmap{}
		bi.InitName(bi, tv.UniqueName())
		bi.GrabRenderFrom(widg)
		gi.ImageClearer(bi.Pixels, 50.0)
		tv.Viewport.Win.StartDragNDrop(tv.This, md, bi)
	}
}

// DragNDropTarget handles a drag-n-drop drop
func (tv *TableView) DragNDropTarget(de *dnd.Event) {
	de.Target = tv.This
	if de.Mod == dnd.DropLink {
		de.Mod = dnd.DropCopy // link not supported -- revert to copy
	}
	row, ok := tv.RowFromPos(de.Where.Y)
	if ok {
		de.SetProcessed()
		tv.DropAction(de.Data, de.Mod, row)
	}
}

// MakeDropMenu makes the menu of options for dropping on a target
func (tv *TableView) MakeDropMenu(m *gi.Menu, data interface{}, mod dnd.DropMods, row int) {
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
		m.AddAction(gi.ActOpts{Label: "Assign To", Data: data}, tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TableView).(*TableView)
			tvv.DropAssign(data.(mimedata.Mimes), row)
		})
	}
	m.AddAction(gi.ActOpts{Label: "Insert Before", Data: data}, tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TableView).(*TableView)
		tvv.DropBefore(data.(mimedata.Mimes), mod, row) // captures mod
	})
	m.AddAction(gi.ActOpts{Label: "Insert After", Data: data}, tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TableView).(*TableView)
		tvv.DropAfter(data.(mimedata.Mimes), mod, row) // captures mod
	})
	m.AddAction(gi.ActOpts{Label: "Cancel", Data: data}, tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TableView).(*TableView)
		tvv.DropCancel()
	})
}

// DropAction pops up a menu to determine what specifically to do with dropped items
func (tv *TableView) DropAction(md mimedata.Mimes, mod dnd.DropMods, row int) {
	var men gi.Menu
	tv.MakeDropMenu(&men, md, mod, row)
	pos := tv.RowPos(row)
	gi.PopupMenu(men, pos.X, pos.Y, tv.Viewport, "tvDropMenu")
}

// DropAssign assigns mime data (only the first one!) to this node
func (tv *TableView) DropAssign(md mimedata.Mimes, row int) {
	tv.DraggedRows = nil
	tv.PasteAssign(md, row)
	tv.DragNDropFinalize(dnd.DropCopy)
}

// DragNDropFinalize is called to finalize actions on the Source node prior to
// performing target actions -- mod must indicate actual action taken by the
// target, including ignore -- ends up calling DragNDropSource if us..
func (tv *TableView) DragNDropFinalize(mod dnd.DropMods) {
	tv.UnselectAllRows()
	tv.Viewport.Win.FinalizeDragNDrop(mod)
}

// DragNDropSource is called after target accepts the drop -- we just remove
// elements that were moved
func (tv *TableView) DragNDropSource(de *dnd.Event) {
	if de.Mod != dnd.DropMove || len(tv.DraggedRows) == 0 {
		return
	}
	updt := tv.UpdateStart()
	sort.Slice(tv.DraggedRows, func(i, j int) bool {
		return tv.DraggedRows[i] > tv.DraggedRows[j]
	})
	row := tv.DraggedRows[0]
	for _, r := range tv.DraggedRows {
		tv.SliceDelete(r, false)
	}
	tv.DraggedRows = nil
	tv.ConfigSliceGrid(true)
	tv.UpdateEnd(updt)
	tv.SelectRowAction(row, mouse.NoSelectMode)
}

// SaveDraggedRows saves selectedrows into dragged rows taking into account insertion at rows
func (tv *TableView) SaveDraggedRows(row int) {
	sz := len(tv.SelectedRows)
	if sz == 0 {
		tv.DraggedRows = nil
		return
	}
	tv.DraggedRows = make([]int, len(tv.SelectedRows))
	idx := 0
	for r, _ := range tv.SelectedRows {
		if r > row {
			tv.DraggedRows[idx] = r + sz // make room for insertion
		} else {
			tv.DraggedRows[idx] = r
		}
		idx++
	}
}

// DropBefore inserts object(s) from mime data before this node
func (tv *TableView) DropBefore(md mimedata.Mimes, mod dnd.DropMods, row int) {
	tv.SaveDraggedRows(row)
	tv.PasteAtRow(md, row)
	tv.DragNDropFinalize(mod)
}

// DropAfter inserts object(s) from mime data after this node
func (tv *TableView) DropAfter(md mimedata.Mimes, mod dnd.DropMods, row int) {
	tv.SaveDraggedRows(row + 1)
	tv.PasteAtRow(md, row+1)
	tv.DragNDropFinalize(mod)
}

// DropCancel cancels the drop action e.g., preventing deleting of source
// items in a Move case
func (tv *TableView) DropCancel() {
	tv.DragNDropFinalize(dnd.DropIgnore)
}

func (tv *TableView) KeyInputActive(kt *key.ChordEvent) {
	kf := gi.KeyFun(kt.ChordString())
	selMode := mouse.SelectModeBits(kt.Modifiers)
	row := tv.SelectedIdx
	switch kf {
	case gi.KeyFunCancelSelect:
		tv.UnselectAllRows()
		tv.SelectMode = false
		kt.SetProcessed()
	case gi.KeyFunMoveDown:
		tv.MoveDownAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunMoveUp:
		tv.MoveUpAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunSelectMode:
		tv.SelectMode = !tv.SelectMode
		kt.SetProcessed()
	case gi.KeyFunSelectAll:
		tv.SelectAllRows()
		tv.SelectMode = false
		kt.SetProcessed()
	case gi.KeyFunDelete:
		tv.SliceDelete(tv.SelectedIdx, true)
		tv.SelectMode = false
		tv.SelectRowAction(row, mouse.NoSelectMode)
		kt.SetProcessed()
	// case gi.KeyFunDuplicate:
	// 	tv.SrcDuplicate() // todo: dupe
	// 	kt.SetProcessed()
	case gi.KeyFunInsert:
		tv.SliceNewAt(row, true)
		tv.SelectMode = false
		tv.SelectRowAction(row+1, mouse.NoSelectMode) // todo: somehow nrow not working
		kt.SetProcessed()
	case gi.KeyFunInsertAfter:
		tv.SliceNewAt(row+1, true)
		tv.SelectMode = false
		tv.SelectRowAction(row+1, mouse.NoSelectMode)
		kt.SetProcessed()
	case gi.KeyFunCopy:
		tv.CopyRows(true)
		tv.SelectMode = false
		tv.SelectRowAction(row, mouse.NoSelectMode)
		kt.SetProcessed()
	case gi.KeyFunCut:
		tv.CutRows()
		tv.SelectMode = false
		kt.SetProcessed()
	case gi.KeyFunPaste:
		tv.Paste(tv.SelectedIdx)
		tv.SelectMode = false
		kt.SetProcessed()
	}
}

func (tv *TableView) KeyInputInactive(kt *key.ChordEvent) {
	kf := gi.KeyFun(kt.ChordString())
	row := tv.SelectedIdx
	switch {
	case kf == gi.KeyFunMoveDown:
		nr := row + 1
		if nr < tv.BuiltSize {
			tv.ScrollToRow(nr)
			tv.UpdateSelect(nr, true)
			kt.SetProcessed()
		}
	case kf == gi.KeyFunMoveUp:
		nr := row - 1
		if nr >= 0 {
			tv.ScrollToRow(nr)
			tv.UpdateSelect(nr, true)
			kt.SetProcessed()
		}
	case kf == gi.KeyFunSelectItem || kf == gi.KeyFunAccept || kt.Rune == ' ':
		tv.TableViewSig.Emit(tv.This, int64(TableViewDoubleClicked), tv.SelectedIdx)
		kt.SetProcessed()
	}
}

func (tv *TableView) TableViewEvents() {
	if tv.IsInactive() {
		if tv.InactKeyNav {
			tv.ConnectEvent(oswin.KeyChordEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
				tvv := recv.Embed(KiT_TableView).(*TableView)
				kt := d.(*key.ChordEvent)
				tvv.KeyInputInactive(kt)
			})
		}
		tv.ConnectEvent(oswin.MouseEvent, gi.LowRawPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			me := d.(*mouse.Event)
			tvv := recv.Embed(KiT_TableView).(*TableView)
			if me.Button == mouse.Left && me.Action == mouse.DoubleClick {
				tvv.TableViewSig.Emit(tvv.This, int64(TableViewDoubleClicked), tvv.SelectedIdx)
				me.SetProcessed()
			}
		})
	} else {
		tv.ConnectEvent(oswin.KeyChordEvent, gi.HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			tvv := recv.Embed(KiT_TableView).(*TableView)
			kt := d.(*key.ChordEvent)
			tvv.KeyInputActive(kt)
		})
		tv.ConnectEvent(oswin.DNDEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			de := d.(*dnd.Event)
			tvv := recv.Embed(KiT_TableView).(*TableView)
			switch de.Action {
			case dnd.Start:
				tvv.DragNDropStart()
			case dnd.DropOnTarget:
				tvv.DragNDropTarget(de)
			case dnd.DropFmSource:
				tvv.DragNDropSource(de)
			}
		})
		sgf := tv.SliceGrid()
		sgf.ConnectEvent(oswin.DNDFocusEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			de := d.(*dnd.FocusEvent)
			switch de.Action {
			case dnd.Enter:
				gi.DNDSetCursor(de.Mod)
			case dnd.Exit:
				gi.DNDNotCursor()
			case dnd.Hover:
				// nothing here?
			}
		})
	}
}
