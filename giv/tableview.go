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

////////////////////////////////////////////////////////////////////////////////////////
//  TableView

// TableView represents a slice-of-structs as a table, where the fields are
// the columns, within an overall frame and a button box at the bottom where
// methods can be invoked.  It has two modes, determined by Inactive flag: if
// Inactive, it functions as a mutually-exclusive item selector, highlighting
// the selected row and emitting a WidgetSig WidgetSelected signal.  If
// !Inactive, it is a full-featured editor with multiple-selection,
// cut-and-paste, and drag-and-drop, reporting each action taken using the
// TableViewSig signals
type TableView struct {
	gi.Frame
	Slice        interface{}        `view:"-" json:"-" xml:"-" desc:"the slice that we are a view onto -- must be a pointer to that slice"`
	StyleFunc    TableViewStyleFunc `view:"-" json:"-" xml:"-" desc:"optional styling function"`
	Values       [][]ValueView      `json:"-" xml:"-" desc:"ValueView representations of the slice field values -- outer dimension is fields, inner is rows (generally more rows than fields, so this minimizes number of slices allocated)"`
	ShowIndex    bool               `xml:"index" desc:"whether to show index or not -- updated from "index" property (bool) -- index is required for copy / paste and DND of rows"`
	SelectedIdx  int                `json:"-" xml:"-" desc:"index (row) of currently-selected item -- see SelectedRows for full set of selected rows in active editing mode"`
	SortIdx      int                `desc:"current sort index"`
	SortDesc     bool               `desc:"whether current sort order is descending"`
	SelectMode   bool               `desc:"editing-mode select rows mode"`
	SelectedRows map[int]bool       `desc:"list of currently-selected rows"`
	DraggedRows  map[int]bool       `desc:"list of currently-dragged rows"`
	TableViewSig ki.Signal          `json:"-" xml:"-" desc:"table view interactive editing signals"`
	ViewSig      ki.Signal          `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`

	TmpSave    ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	BuiltSlice interface{} `view:"-" json:"-" xml:"-" desc:"the built slice"`
	BuiltSize  int
	StruType   reflect.Type
	NVisFields int
	VisFields  []reflect.StructField `view:"-" json:"-" xml:"-" desc:"the visible fields"`
}

var KiT_TableView = kit.Types.AddType(&TableView{}, TableViewProps)

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// TableViewStyleFunc is a styling function for custom styling /
// configuration of elements in the view
type TableViewStyleFunc func(slice interface{}, widg gi.Node2D, row, col int, vv ValueView)

// SetSlice sets the source slice that we are viewing -- rebuilds the children
// to represent this slice
func (tv *TableView) SetSlice(sl interface{}, tmpSave ValueView) {
	updt := false
	if tv.Slice != sl {
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
	tv.TmpSave = tmpSave
	tv.UpdateFromSlice()
	tv.UpdateEnd(updt)
}

var TableViewProps = ki.Props{
	"background-color": &gi.Prefs.BackgroundColor,
	"color":            &gi.Prefs.FontColor,
}

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

// SetFrame configures view as a frame
func (tv *TableView) SetFrame() {
	tv.Lay = gi.LayoutCol
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (tv *TableView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Frame, "struct-grid")
	config.Add(gi.KiT_Space, "grid-space")
	config.Add(gi.KiT_Layout, "buttons")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (tv *TableView) StdConfig() (mods, updt bool) {
	tv.SetFrame()
	config := tv.StdFrameConfig()
	mods, updt = tv.ConfigChildren(config, false)
	return
}

// SliceGrid returns the SliceGrid grid frame widget, which contains all the
// fields and values, and its index, within frame -- nil, -1 if not found
func (tv *TableView) SliceGrid() (*gi.Frame, int) {
	idx, ok := tv.Children().IndexByName("struct-grid", 0)
	if !ok {
		return nil, -1
	}
	return tv.KnownChild(idx).(*gi.Frame), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (tv *TableView) ButtonBox() (*gi.Layout, int) {
	idx, ok := tv.Children().IndexByName("buttons", 0)
	if !ok {
		return nil, -1
	}
	return tv.KnownChild(idx).(*gi.Layout), idx
}

// StdGridConfig returns a TypeAndNameList for configuring the struct-grid
func (tv *TableView) StdGridConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Layout, "header")
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
func (tv *TableView) ConfigSliceGrid() {
	if kit.IfaceIsNil(tv.Slice) {
		return
	}
	mv := reflect.ValueOf(tv.Slice)
	mvnp := kit.NonPtrValue(mv)
	sz := mvnp.Len()

	if tv.BuiltSlice == tv.Slice && tv.BuiltSize == sz {
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

	sg, _ := tv.SliceGrid()
	if sg == nil {
		return
	}
	sg.Lay = gi.LayoutCol
	// sg.SetMinPrefHeight(units.NewValue(10, units.Em))
	sg.SetMinPrefWidth(units.NewValue(10, units.Em))
	sg.SetStretchMaxHeight() // for this to work, ALL layers above need it too
	sg.SetStretchMaxWidth()  // for this to work, ALL layers above need it too

	sgcfg := tv.StdGridConfig()
	modsg, updtg := sg.ConfigChildren(sgcfg, false)
	if modsg {
		tv.SetFullReRender()
	} else {
		updtg = sg.UpdateStart()
	}

	sgh := sg.KnownChild(0).(*gi.Layout)
	sgh.Lay = gi.LayoutRow
	sgh.SetStretchMaxWidth()

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
			tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
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
		SortStructSlice(tv.Slice, tv.SortIdx, !tv.SortDesc)
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

	nWidgPerRow, idxOff := tv.RowWidgetNs()
	sg, _ := tv.SliceGrid()
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
					tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
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
					if sig == int64(gi.WidgetSelected) {
						wbb := send.(gi.Node2D).AsWidget()
						idx := wbb.KnownProp("tv-index").(int)
						tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
						tvv.UpdateSelect(idx, wbb.IsSelected())
					}
				})
			}
			if tv.IsInactive() {
				widg.AsNode2D().SetInactive()
			} else {
				vvb := vv.AsValueViewBase()
				vvb.ViewSig.ConnectOnly(tv.This, // todo: do we need this?
					func(recv, send ki.Ki, sig int64, data interface{}) {
						tvv, _ := recv.EmbeddedStruct(KiT_TableView).(*TableView)
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
					tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
					tvv.SliceNewAt(act.Data.(int)+1, true)
				})
				delact.SetIcon("minus")
				delact.Tooltip = "delete this element"
				delact.Data = i
				delact.ActionSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					act := send.(*gi.Action)
					tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
					tvv.SliceDelete(act.Data.(int), true)
				})
			}
			if tv.StyleFunc != nil {
				tv.StyleFunc(mvnp.Interface(), widg, i, fli, vv)
			}
		}
	}
	if tv.SelectedIdx >= 0 {
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
		tv.ConfigSliceGrid()
	}
	tv.ViewSig.Emit(tv.This, 0, nil)
}

// SliceDelete deletes element at given index from slice -- reconfig means
// call ConfigSliceGrid to update display
func (tv *TableView) SliceDelete(idx int, reconfig bool) {
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
		tv.ConfigSliceGrid()
	}
	tv.ViewSig.Emit(tv.This, 0, nil)
}

// SortSliceAction sorts the slice for given field index -- toggles ascending
// vs. descending if already sorting on this dimension
func (tv *TableView) SortSliceAction(fldIdx int) {
	sg, _ := tv.SliceGrid()
	sgh := sg.KnownChild(0).(*gi.Layout)
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

	SortStructSlice(tv.Slice, rawIdx, !tv.SortDesc)
	tv.ConfigSliceGridRows()
}

// SortStructSlice sorts a slice of a struct according to the given field
// (specified by first-order index) and sort direction, using int, float,
// string kind conversions through reflect, and supporting time.Time as well
// -- todo: could extend with a function that handles specific fields
func SortStructSlice(struSlice interface{}, fldIdx int, ascending bool) error {
	mv := reflect.ValueOf(struSlice)
	mvnp := kit.NonPtrValue(mv)
	struTyp := kit.NonPtrType(reflect.TypeOf(struSlice).Elem().Elem())
	if fldIdx < 0 || fldIdx >= struTyp.NumField() {
		err := fmt.Errorf("gi.SortStructSlice: field index out of range: %v must be < %v\n", fldIdx, struTyp.NumField())
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

// ConfigSliceButtons configures the buttons for map functions
func (tv *TableView) ConfigSliceButtons() {
	if kit.IfaceIsNil(tv.Slice) {
		return
	}
	if tv.IsInactive() {
		return
	}
	bb, _ := tv.ButtonBox()
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Button, "Add")
	mods, updt := bb.ConfigChildren(config, false)
	addb := bb.KnownChildByName("Add", 0).EmbeddedStruct(gi.KiT_Button).(*gi.Button)
	addb.SetText("Add")
	addb.ButtonSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
			tvv.SliceNewAt(-1, true)
		}
	})
	if mods {
		bb.UpdateEnd(updt)
	}
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
	tv.ConfigSliceGrid()
	tv.ConfigSliceButtons()
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

func (tv *TableView) Layout2D(parBBox image.Rectangle) {
	tv.Frame.Layout2D(parBBox)
	sg, _ := tv.SliceGrid()
	if sg == nil {
		return
	}
	idxOff := 1
	if !tv.ShowIndex {
		idxOff = 0
	}

	nfld := tv.NVisFields + idxOff
	sgh := sg.KnownChild(0).(*gi.Layout)
	sgf := sg.KnownChild(2).(*gi.Frame)
	if len(sgf.Kids) >= nfld {
		sgh.SetProp("width", units.NewValue(sgf.LayData.AllocSize.X, units.Dot))
		for fli := 0; fli < nfld; fli++ {
			lbl := sgh.KnownChild(fli).(gi.Node2D).AsWidget()
			widg := sgf.KnownChild(fli).(gi.Node2D).AsWidget()
			lbl.SetProp("width", units.NewValue(widg.LayData.AllocSize.X, units.Dot))
		}
		sgh.Layout2D(parBBox)
	}
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
	} else {
		tv.DisconnectAllEvents(gi.AllPris)
	}
}

func (tv *TableView) HasFocus2D() bool {
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

// RowIndexLabel returns the index label for given row -- nil if no indexes or out of range
func (tv *TableView) RowIndexLabel(row int) (*gi.Label, bool) {
	if !tv.ShowIndex {
		return nil, false
	}
	if tv.RowStruct(row) == nil { // range check
		return nil, false
	}
	nWidgPerRow, _ := tv.RowWidgetNs()
	sg, _ := tv.SliceGrid()
	if sg == nil {
		return nil, false
	}
	sgf := sg.KnownChild(2).(*gi.Frame)
	idxlab := sgf.Kids[row*nWidgPerRow].(*gi.Label)
	return idxlab, true
}

// RowGrabFocus grabs the focus for the first focusable widget in given row
// -- returns that element or nil if not successful
func (tv *TableView) RowGrabFocus(row int) *gi.WidgetBase {
	if tv.RowStruct(row) == nil { // range check
		return nil
	}
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	sg, _ := tv.SliceGrid()
	if sg == nil {
		return nil
	}
	ridx := nWidgPerRow * row
	sgf := sg.KnownChild(2).(*gi.Frame)
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
	idxlab, ok := tv.RowIndexLabel(row)
	if ok {
		pos = idxlab.ContextMenuPos()
	}
	return pos
}

// RowFromPos returns the row that contains given vertical position, false if not found
func (tv *TableView) RowFromPos(posY int) (int, bool) {
	for rw := 0; rw < tv.BuiltSize; rw++ {
		idxlab, ok := tv.RowIndexLabel(rw)
		if ok {
			if idxlab.WinBBox.Min.Y < posY && posY < idxlab.WinBBox.Max.Y {
				return rw, true
			}
		}
	}
	return -1, false
}

//////////////////////////////////////////////////////////////////////////////
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
		tv.RowGrabFocus(nrow)
		// fw := tv.RowGrabFocus(nrow)
		// if fw != nil {
		// 	tv.RootView.TableViewSig.Emit(tv.RootView.This, int64(TableViewSelected), nrow)
		// }
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
		tv.RowGrabFocus(nrow)
		// fw := tv.RowGrabFocus(nrow)
		// if fw != nil {
		// 	tv.RootView.TableViewSig.Emit(tv.RootView.This, int64(TableViewSelected), nrow)
		// }
	}
	return nrow
}

//////////////////////////////////////////////////////////////////////////////
//    Selection: user operates on the index labels

// SelectRowWidgets sets the selection state of given row of widgets
func (tv *TableView) SelectRowWidgets(idx int, sel bool) {
	var win *gi.Window
	if tv.Viewport != nil {
		win = tv.Viewport.Win
	}
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	sg, _ := tv.SliceGrid()
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

// UpdateSelect updates the selection for the given index
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
	if !tv.RowIsSelected(row) {
		tv.SelectedRows[row] = true
		tv.SelectRowWidgets(row, true)
	}
}

// UnselectRow unselects given row (if selected)
func (tv *TableView) UnselectRow(row int) {
	if tv.RowIsSelected(row) {
		delete(tv.SelectedRows, row)
		tv.SelectRowWidgets(row, false)
	}
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
				tv.SelectedIdx = row
				tv.SelectRow(row)
				tv.RowGrabFocus(row)
			}
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
				log.Printf("gi.TableView NodesFromMimeData: JSON load error: %v\n", err)
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
	// updt := tv.UpdateStart()
	rws := tv.SelectedRowsList(true) // descending sort
	for _, r := range rws {
		tv.SliceDelete(r, false)
	}
	// tv.ConfigSliceGrid()
	// tv.UpdateEnd(updt)
}

// CutRows copies selected rows to clip.Board and deletes selected rows
func (tv *TableView) CutRows() {
	updt := tv.UpdateStart()
	tv.CopyRows(false)
	rws := tv.SelectedRowsList(true) // descending sort
	tv.UnselectAllRows()
	for _, r := range rws {
		tv.SliceDelete(r, false)
	}
	tv.ConfigSliceGrid()
	tv.UpdateEnd(updt)
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
	m.AddMenuText("Assign To", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
		tvv.PasteAssign(data.(mimedata.Mimes), row)
	})
	m.AddMenuText("Insert Before", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
		tvv.PasteAtRow(data.(mimedata.Mimes), row)
	})
	m.AddMenuText("Insert After", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
		tvv.PasteAtRow(data.(mimedata.Mimes), row+1)
	})
	m.AddMenuText("Cancel", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	tv.ConfigSliceGrid()
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
		if row >= 0 && row < sz-1 {
			reflect.Copy(tvnp.Slice(row+1, sz+1), tvnp.Slice(row, sz))
			tvnp.Index(row).Set(reflect.ValueOf(ns).Elem())
			tvl.Elem().Set(tvnp)
		}
	}
	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	tv.ConfigSliceGrid()
	tv.UpdateEnd(updt)
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
	idxlab, ok := tv.RowIndexLabel(rws[0])
	if ok {
		bi := &gi.Bitmap{}
		bi.InitName(bi, tv.UniqueName())
		bi.GrabRenderFrom(idxlab)
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

// DragNDropFinalize is called to finalize actions on the Source node prior to
// performing target actions -- mod must indicate actual action taken by the
// target, including ignore
func (tv *TableView) DragNDropFinalize(mod dnd.DropMods) {
	tv.DraggedRows = tv.SelectedRows
	tv.UnselectAllRows()
	tv.Viewport.Win.FinalizeDragNDrop(mod)
}

// DragNDropSource is called after target accepts the drop -- we just remove
// elements that were moved
func (tv *TableView) DragNDropSource(de *dnd.Event) {
	if de.Mod != dnd.DropMove {
		return
	}
	curSel := tv.SelectedRows
	tv.SelectedRows = tv.DraggedRows
	tv.DeleteRows()
	tv.SelectedRows = curSel
	tv.DraggedRows = nil
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
		m.AddMenuText("Assign To", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
			tvv.DropAssign(data.(mimedata.Mimes), row)
		})
	}
	m.AddMenuText("Insert Before", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
		tvv.DropBefore(data.(mimedata.Mimes), mod, row) // captures mod
	})
	m.AddMenuText("Insert After", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
		tvv.DropAfter(data.(mimedata.Mimes), mod, row) // captures mod
	})
	m.AddMenuText("Cancel", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
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
	tv.DragNDropFinalize(dnd.DropCopy)
	tv.PasteAssign(md, row)
}

// DropBefore inserts object(s) from mime data before this node
func (tv *TableView) DropBefore(md mimedata.Mimes, mod dnd.DropMods, row int) {
	tv.DragNDropFinalize(mod)
	tv.PasteAtRow(md, row)
}

// DropAfter inserts object(s) from mime data after this node
func (tv *TableView) DropAfter(md mimedata.Mimes, mod dnd.DropMods, row int) {
	tv.DragNDropFinalize(mod)
	tv.PasteAtRow(md, row+1)
}

// DropCancel cancels the drop action e.g., preventing deleting of source
// items in a Move case
func (tv *TableView) DropCancel() {
	tv.DragNDropFinalize(dnd.DropIgnore)
}

func (tv *TableView) KeyInput(kt *key.ChordEvent) {
	kf := gi.KeyFun(kt.ChordString())
	selMode := mouse.SelectModeMod(kt.Modifiers)
	row := tv.SelectedIdx
	switch kf {
	case gi.KeyFunSelectItem:
		tv.SelectRowAction(tv.SelectedIdx, selMode)
		kt.SetProcessed()
	case gi.KeyFunCancelSelect:
		tv.UnselectAllRows()
		kt.SetProcessed()
	// case gi.KeyFunMoveRight:
	// 	tv.Open()
	// 	kt.SetProcessed()
	// case gi.KeyFunMoveLeft:
	// 	tv.Close()
	// 	kt.SetProcessed()
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
		kt.SetProcessed()
	case gi.KeyFunDelete:
		tv.SliceDelete(tv.SelectedIdx, true)
		tv.SelectRowAction(row, mouse.NoSelectMode)
		kt.SetProcessed()
	// case gi.KeyFunDuplicate:
	// 	tv.SrcDuplicate() // todo: dupe
	// 	kt.SetProcessed()
	case gi.KeyFunInsert:
		tv.SliceNewAt(row, true)
		tv.SelectRowAction(row+1, mouse.NoSelectMode) // todo: somehow nrow not working
		kt.SetProcessed()
	case gi.KeyFunInsertAfter:
		tv.SliceNewAt(row+1, true)
		tv.SelectRowAction(row+1, mouse.NoSelectMode)
		kt.SetProcessed()
	case gi.KeyFunCopy:
		tv.CopyRows(true)
		tv.SelectRowAction(row, mouse.NoSelectMode)
		kt.SetProcessed()
	case gi.KeyFunCut:
		tv.CutRows()
		tv.SelectRowAction(row, mouse.NoSelectMode)
		kt.SetProcessed()
	case gi.KeyFunPaste:
		tv.Paste(tv.SelectedIdx)
		tv.SelectRowAction(row, mouse.NoSelectMode)
		kt.SetProcessed()
	}
}

func (tv *TableView) TableViewEvents() {
	tv.ConnectEventType(oswin.KeyChordEvent, gi.HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
		kt := d.(*key.ChordEvent)
		tvv.KeyInput(kt)
	})
	tv.ConnectEventType(oswin.DNDEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		de := d.(*dnd.Event)
		tvv := recv.EmbeddedStruct(KiT_TableView).(*TableView)
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
