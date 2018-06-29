// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"reflect"

	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  StructTableView

// StructTableView represents a slice of a struct as a table, where the fields
// are the columns, within an overall frame and a button box at the bottom
// where methods can be invoked -- set to Inactive for select-only mode, which
// emits SelectSig signals when selection is updated
type StructTableView struct {
	Frame
	Slice       interface{}              `desc:"the slice that we are a view onto -- must be a pointer to that slice"`
	StyleFunc   StructTableViewStyleFunc `json:"-" xml:"-" desc:"optional styling function"`
	Values      [][]ValueView            `json:"-" xml:"-" desc:"ValueView representations of the slice field values -- outer dimension is fields, inner is rows (generally more rows than fields, so this minimizes number of slices allocated)"`
	TmpSave     ValueView                `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig     ki.Signal                `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
	SelectedIdx int                      `json:"-" xml:"-" desc:"index of currently-selected item, in Inactive mode only"`
	SelectSig   ki.Signal                `json:"-" xml:"-" desc:"signal for selection changes, in Inactive mode only"`
	builtSlice  interface{}
	builtSize   int
}

var KiT_StructTableView = kit.Types.AddType(&StructTableView{}, StructTableViewProps)

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// StructTableViewStyleFunc is a styling function for custom styling /
// configuration of elements in the view
type StructTableViewStyleFunc func(slice interface{}, widg Node2D, row, col int, vv ValueView)

// SetSlice sets the source slice that we are viewing -- rebuilds the children
// to represent this slice
func (sv *StructTableView) SetSlice(sl interface{}, tmpSave ValueView) {
	updt := false
	if sv.Slice != sl {
		slpTyp := reflect.TypeOf(sl)
		if slpTyp.Kind() != reflect.Ptr {
			log.Printf("StructTableView requires that you pass a pointer to a slice of struct elements -- type is not a Ptr: %v\n", slpTyp.String())
			return
		}
		if slpTyp.Elem().Kind() != reflect.Slice {
			log.Printf("StructTableView requires that you pass a pointer to a slice of struct elements -- ptr doesn't point to a slice: %v\n", slpTyp.Elem().String())
			return
		}
		sv.Slice = sl
		struTyp := sv.StructType()
		if struTyp.Kind() != reflect.Struct {
			log.Printf("StructTableView requires that you pass a slice of struct elements -- type is not a Struct: %v\n", struTyp.String())
			return
		}
		updt = sv.UpdateStart()
		sv.SelectedIdx = -1
		sv.SetFullReRender()
	}
	sv.TmpSave = tmpSave
	sv.UpdateFromSlice()
	sv.UpdateEnd(updt)
}

var StructTableViewProps = ki.Props{
	"background-color": &Prefs.BackgroundColor,
}

// StructType returns the type of the struct within the slice
func (sv *StructTableView) StructType() reflect.Type {
	return kit.NonPtrType(reflect.TypeOf(sv.Slice).Elem().Elem())
}

// SetFrame configures view as a frame
func (sv *StructTableView) SetFrame() {
	sv.Lay = LayoutCol
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *StructTableView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(KiT_Layout, "slice-grid")
	config.Add(KiT_Space, "grid-space")
	config.Add(KiT_Layout, "buttons")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (sv *StructTableView) StdConfig() (mods, updt bool) {
	sv.SetFrame()
	config := sv.StdFrameConfig()
	mods, updt = sv.ConfigChildren(config, false)
	return
}

// SliceGrid returns the SliceGrid grid layout widget, which contains all the
// fields and values, and its index, within frame -- nil, -1 if not found
func (sv *StructTableView) SliceGrid() (*Layout, int) {
	idx := sv.ChildIndexByName("slice-grid", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (sv *StructTableView) ButtonBox() (*Layout, int) {
	idx := sv.ChildIndexByName("buttons", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// StdGridConfig returns a TypeAndNameList for configuring the slice-grid
func (sv *StructTableView) StdGridConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(KiT_Layout, "header")
	config.Add(KiT_Separator, "head-sepe")
	config.Add(KiT_Frame, "grid")
	return config
}

// ConfigSliceGrid configures the SliceGrid for the current slice
func (sv *StructTableView) ConfigSliceGrid() {
	if kit.IfaceIsNil(sv.Slice) {
		return
	}
	mv := reflect.ValueOf(sv.Slice)
	mvnp := kit.NonPtrValue(mv)
	sz := mvnp.Len()

	if sv.builtSlice == sv.Slice && sv.builtSize == sz {
		return
	}
	sv.builtSlice = sv.Slice
	sv.builtSize = sz

	sv.SelectedIdx = -1

	// this is the type of element within slice -- already checked that it is a struct
	struTyp := sv.StructType()
	nfld := struTyp.NumField()

	nWidgPerRow := 1 + nfld
	if !sv.IsInactive() {
		nWidgPerRow += 2
	}

	// always start fresh!
	sv.Values = make([][]ValueView, nfld)
	for fli := 0; fli < nfld; fli++ {
		sv.Values[fli] = make([]ValueView, sz)
	}

	sg, _ := sv.SliceGrid()
	if sg == nil {
		return
	}
	sg.Lay = LayoutCol
	sg.SetStretchMaxHeight() // for this to work, ALL layers above need it too
	sg.SetStretchMaxWidth()  // for this to work, ALL layers above need it too

	sgcfg := sv.StdGridConfig()
	modsg, updtg := sg.ConfigChildren(sgcfg, false)
	if modsg {
		sv.SetFullReRender()
	} else {
		updtg = sg.UpdateStart()
	}

	sgh := sg.Child(0).(*Layout)
	sgh.Lay = LayoutRow
	sgh.SetStretchMaxWidth()

	sep := sg.Child(1).(*Separator)
	sep.Horiz = true
	sep.SetStretchMaxWidth()

	sgf := sg.Child(2).(*Frame)
	sgf.Lay = LayoutGrid

	// setting a pref here is key for giving it a scrollbar in larger context
	sgf.SetMinPrefHeight(units.NewValue(10, units.Em))
	sgf.SetMinPrefWidth(units.NewValue(10, units.Em))
	sgf.SetStretchMaxHeight() // for this to work, ALL layers above need it too
	sgf.SetStretchMaxWidth()  // for this to work, ALL layers above need it too
	if sv.IsInactive() {
		sgf.SetProp("columns", nfld+1)
	} else {
		sgf.SetProp("columns", nfld+3)
	}

	// Configure Header
	hcfg := kit.TypeAndNameList{}
	hcfg.Add(KiT_Label, "head-idx")
	for fli := 0; fli < nfld; fli++ {
		fld := struTyp.Field(fli)
		labnm := fmt.Sprintf("head-%v", fld.Name)
		hcfg.Add(KiT_Label, labnm)
	}
	if !sv.IsInactive() {
		hcfg.Add(KiT_Label, "head-add")
		hcfg.Add(KiT_Label, "head-del")
	}

	modsh, updth := sgh.ConfigChildren(hcfg, false)
	if modsh {
		sv.SetFullReRender()
	} else {
		updth = sgh.UpdateStart()
	}
	lbl := sgh.Child(0).(*Label)
	lbl.SetProp("vertical-align", AlignMiddle)
	lbl.Text = "Index"
	for fli := 0; fli < nfld; fli++ {
		fld := struTyp.Field(fli)
		lbl := sgh.Child(1 + fli).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		idxtxt := fmt.Sprintf("%v", fld.Name) // todo: add RTF
		lbl.Text = idxtxt
	}
	if !sv.IsInactive() {
		lbl := sgh.Child(nfld + 1).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		lbl.Text = "Add"
		lbl = sgh.Child(nfld + 2).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		lbl.Text = "Del"
	}

	sgf.Kids = make(ki.Slice, nWidgPerRow*sz)

	// Render Data
	for i := 0; i < sz; i++ {
		ridx := i * nWidgPerRow
		val := kit.OnePtrValue(mvnp.Index(i)) // deal with pointer lists
		stru := val.Interface()
		idxtxt := fmt.Sprintf("%05d", i)
		labnm := fmt.Sprintf("index-%v", idxtxt)
		idxlab := Label{}
		sgf.SetChild(&idxlab, ridx, labnm)
		idxlab.Text = idxtxt
		idxlab.SetProp("vertical-align", AlignMiddle)

		for fli := 0; fli < nfld; fli++ {
			fval := val.Elem().Field(fli)
			vv := ToValueView(fval.Interface())
			if vv == nil { // shouldn't happen
				continue
			}
			field := struTyp.Field(fli)
			vv.SetStructValue(fval.Addr(), stru, &field, sv.TmpSave)
			vtyp := vv.WidgetType()
			valnm := fmt.Sprintf("value-%v.%v", fli, idxtxt)
			sv.Values[fli][i] = vv
			widg := ki.NewOfType(vtyp).(Node2D)
			sgf.SetChild(widg, ridx+1+fli, valnm)
			widg.SetProp("vertical-align", AlignMiddle)
			vv.ConfigWidget(widg)
			if sv.IsInactive() {
				widg.AsNode2D().SetInactive()
			}
			if !sv.IsInactive() {
				vvb := vv.AsValueViewBase()
				vvb.ViewSig.ConnectOnly(sv.This, // todo: do we need this?
					func(recv, send ki.Ki, sig int64, data interface{}) {
						svv, _ := recv.EmbeddedStruct(KiT_StructTableView).(*StructTableView)
						svv.UpdateSig()
						svv.ViewSig.Emit(svv.This, 0, nil)
					})

				addnm := fmt.Sprintf("add-%v", idxtxt)
				delnm := fmt.Sprintf("del-%v", idxtxt)
				addact := Action{}
				delact := Action{}
				sgf.SetChild(&addact, ridx+1+nfld, addnm)
				sgf.SetChild(&delact, ridx+1+nfld+1, delnm)

				addact.SetProp("vertical-align", AlignMiddle)
				addact.SetIcon("plus")
				addact.Data = i
				addact.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					act := send.(*Action)
					svv := recv.EmbeddedStruct(KiT_StructTableView).(*StructTableView)
					svv.SliceNewAt(act.Data.(int) + 1)
				})
				delact.SetProp("vertical-align", AlignMiddle)
				delact.SetIcon("minus")
				delact.Data = i
				delact.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					act := send.(*Action)
					svv := recv.EmbeddedStruct(KiT_StructTableView).(*StructTableView)
					svv.SliceDelete(act.Data.(int))
				})
			} else { // inactive
				if widg.TypeEmbeds(KiT_TextField) {
					tf := widg.EmbeddedStruct(KiT_TextField).(*TextField)
					tf.SetProp("stv-index", i)
					tf.ClearSelected()
					tf.TextFieldSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
						if sig == int64(TextFieldSelected) {
							tff := send.(*TextField)
							idx := tff.Prop("stv-index", false, false).(int)
							svv := recv.EmbeddedStruct(KiT_StructTableView).(*StructTableView)
							svv.UpdateSelect(idx, tff.IsSelected())
						}
					})
				}
			}
			if sv.StyleFunc != nil {
				sv.StyleFunc(sv.Slice, widg, i, fli, vv)
			}
		}
	}
	sgh.UpdateEnd(updth)
	sg.UpdateEnd(updtg)
}

// UpdateSelect updates the selection for the given index
func (sv *StructTableView) UpdateSelect(idx int, sel bool) {
	struTyp := sv.StructType()
	nfld := struTyp.NumField()
	sg, _ := sv.SliceGrid()
	sgf := sg.Child(2).(*Frame)

	nWidgPerRow := nfld + 1 // !interact

	if sv.SelectedIdx >= 0 { // unselect current
		for fli := 0; fli < nfld; fli++ {
			seldx := sv.SelectedIdx*nWidgPerRow + 1 + fli
			if sgf.Kids.IsValidIndex(seldx) {
				widg := sgf.Child(sv.SelectedIdx*nWidgPerRow + 1 + fli).(Node2D)
				if widg.TypeEmbeds(KiT_TextField) {
					tf := widg.EmbeddedStruct(KiT_TextField).(*TextField)
					tf.ClearSelected()
					tf.UpdateSig()
				}
			}
		}
	}
	if sel {
		sv.SelectedIdx = idx
		for fli := 0; fli < nfld; fli++ {
			seldx := idx*nWidgPerRow + 1 + fli
			if sgf.Kids.IsValidIndex(seldx) {
				widg := sgf.Child(seldx).(Node2D)
				if widg.TypeEmbeds(KiT_TextField) {
					tf := widg.EmbeddedStruct(KiT_TextField).(*TextField)
					tf.SetSelected()
					tf.UpdateSig()
				}
			}
		}
	} else {
		sv.SelectedIdx = -1
	}
	sv.SelectSig.Emit(sv.This, 0, sv.SelectedIdx)
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1 means the end
func (sv *StructTableView) SliceNewAt(idx int) {
	updt := sv.UpdateStart()
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
	sv.SetFullReRender()
	sv.UpdateEnd(updt)
	sv.ViewSig.Emit(sv.This, 0, nil)
}

// SliceDelete deletes element at given index from slice
func (sv *StructTableView) SliceDelete(idx int) {
	updt := sv.UpdateStart()
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
	sv.SetFullReRender()
	sv.UpdateEnd(updt)
	sv.ViewSig.Emit(sv.This, 0, nil)
}

// ConfigSliceButtons configures the buttons for map functions
func (sv *StructTableView) ConfigSliceButtons() {
	if kit.IfaceIsNil(sv.Slice) {
		return
	}
	if sv.IsInactive() {
		return
	}
	bb, _ := sv.ButtonBox()
	config := kit.TypeAndNameList{}
	config.Add(KiT_Button, "Add")
	mods, updt := bb.ConfigChildren(config, false)
	addb := bb.ChildByName("Add", 0).EmbeddedStruct(KiT_Button).(*Button)
	addb.SetText("Add")
	addb.ButtonSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			svv := recv.EmbeddedStruct(KiT_StructTableView).(*StructTableView)
			svv.SliceNewAt(-1)
		}
	})
	if mods {
		bb.UpdateEnd(updt)
	}
}

func (sv *StructTableView) UpdateFromSlice() {
	mods, updt := sv.StdConfig()
	sv.ConfigSliceGrid()
	sv.ConfigSliceButtons()
	if mods {
		sv.SetFullReRender()
		sv.UpdateEnd(updt)
	}
}

func (sv *StructTableView) UpdateValues() {
	updt := sv.UpdateStart()
	for _, vv := range sv.Values {
		for _, vvf := range vv {
			vvf.UpdateWidget()
		}
	}
	sv.UpdateEnd(updt)
}

func (sv *StructTableView) Layout2D(parBBox image.Rectangle) {
	sv.Frame.Layout2D(parBBox)
	sg, _ := sv.SliceGrid()
	if sg == nil {
		return
	}
	struTyp := sv.StructType()
	nfld := struTyp.NumField()
	sgh := sg.Child(0).(*Layout)
	sgf := sg.Child(2).(*Frame)
	if len(sgf.Kids) >= 1+nfld {
		for fli := 0; fli < nfld; fli++ {
			lbl := sgh.Child(1 + fli).(*Label)
			widg := sgf.Child(1 + fli).(Node2D).AsWidget()
			lbl.SetProp("width", units.NewValue(widg.LayData.AllocSize.X, units.Dot))
		}
		sgh.Layout2D(parBBox)
	}
}

func (sv *StructTableView) Render2D() {
	sv.ClearFullReRender()
	sv.Frame.Render2D()
}

func (sv *StructTableView) ReRender2D() (node Node2D, layout bool) {
	if sv.NeedsFullReRender() {
		node = nil
		layout = false
	} else {
		node = sv.This.(Node2D)
		layout = true
	}
	return
}
