// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
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
	Slice       interface{}   `desc:"the slice that we are a view onto -- must be a pointer to that slice"`
	Values      [][]ValueView `json:"-" xml:"-" desc:"ValueView representations of the slice field values -- outer dimension is fields, inner is rows (generally more rows than fields, so this minimizes number of slices allocated)"`
	TmpSave     ValueView     `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig     ki.Signal     `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
	SelectedIdx int           `json:"-" xml:"-" desc:"index of currently-selected item, in Inactive mode only"`
	SelectSig   ki.Signal     `json:"-" xml:"-" desc:"signal for selection changes, in Inactive mode only"`
	builtSlice  interface{}
	builtSize   int
}

var KiT_StructTableView = kit.Types.AddType(&StructTableView{}, StructTableViewProps)

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// SetSlice sets the source slice that we are viewing -- rebuilds the children
// to represent this slice
func (sv *StructTableView) SetSlice(sl interface{}, tmpSave ValueView) {
	updt := false
	if sv.Slice != sl {
		struTyp := reflect.TypeOf(sl).Elem().Elem()
		if struTyp.Kind() != reflect.Struct {
			log.Printf("StructTableView requires that you pass a slice of struct elements -- type is not a Struct: %v\n", struTyp.String())
			return
		}
		updt = sv.UpdateStart()
		sv.SelectedIdx = -1
		sv.Slice = sl
	}
	sv.TmpSave = tmpSave
	sv.UpdateFromSlice()
	sv.UpdateEnd(updt)
}

var StructTableViewProps = ki.Props{
	"background-color": &Prefs.BackgroundColor,
}

// SetFrame configures view as a frame
func (sv *StructTableView) SetFrame() {
	sv.Lay = LayoutCol
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *StructTableView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(KiT_Frame, "slice-grid")
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

// SliceGrid returns the SliceGrid grid frame widget, which contains all the
// fields and values, and its index, within frame -- nil, -1 if not found
func (sv *StructTableView) SliceGrid() (*Frame, int) {
	idx := sv.ChildIndexByName("slice-grid", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Frame), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (sv *StructTableView) ButtonBox() (*Layout, int) {
	idx := sv.ChildIndexByName("buttons", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
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
	struTyp := reflect.TypeOf(sv.Slice).Elem().Elem()
	nfld := struTyp.NumField()

	// always start fresh!
	sv.Values = make([][]ValueView, nfld)
	for fli := 0; fli < nfld; fli++ {
		sv.Values[fli] = make([]ValueView, sz)
	}

	sg, _ := sv.SliceGrid()
	if sg == nil {
		return
	}
	sg.Lay = LayoutGrid
	sg.SetProp("max-height", units.NewValue(40, units.Em))
	if sv.IsInactive() {
		sg.SetProp("columns", nfld+1)
	} else {
		sg.SetProp("columns", nfld+3)
	}

	config := kit.TypeAndNameList{}

	config.Add(KiT_Label, "head-idx")
	for fli := 0; fli < nfld; fli++ {
		fld := struTyp.Field(fli)
		labnm := fmt.Sprintf("head-%v", fld.Name)
		config.Add(KiT_Label, labnm)
	}
	if !sv.IsInactive() {
		config.Add(KiT_Label, "head-add")
		config.Add(KiT_Label, "head-del")
	}

	for i := 0; i < sz; i++ {
		val := kit.OnePtrValue(mvnp.Index(i)) // deal with pointer lists
		stru := val.Interface()
		idxtxt := fmt.Sprintf("%05d", i)
		labnm := fmt.Sprintf("index-%v", idxtxt)
		config.Add(KiT_Label, labnm)
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
			config.Add(vtyp, valnm)
			sv.Values[fli][i] = vv
		}
		if !sv.IsInactive() {
			addnm := fmt.Sprintf("add-%v", idxtxt)
			delnm := fmt.Sprintf("del-%v", idxtxt)
			config.Add(KiT_Action, addnm)
			config.Add(KiT_Action, delnm)
		}
	}
	mods, updt := sg.ConfigChildren(config, false)
	if mods {
		sv.SetFullReRender()
	} else {
		updt = sg.UpdateStart()
	}
	nWidgPerRow := nfld + 1
	if !sv.IsInactive() {
		nWidgPerRow += 2
	}
	stidx := nfld + 1

	lbl := sg.Child(0).(*Label)
	lbl.SetProp("vertical-align", AlignMiddle)
	lbl.Text = "Index"
	for fli := 0; fli < nfld; fli++ {
		fld := struTyp.Field(fli)
		lbl := sg.Child(1 + fli).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		idxtxt := fmt.Sprintf("%v", fld.Name) // todo: add RTF
		lbl.Text = idxtxt
	}
	if !sv.IsInactive() {
		lbl := sg.Child(nfld + 1).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		lbl.Text = "Add"
		lbl = sg.Child(nfld + 2).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		lbl.Text = "Del"
		stidx += 2
	}
	for i := 0; i < sz; i++ {
		for fli := 0; fli < nfld; fli++ {
			vv := sv.Values[fli][i]
			if !sv.IsInactive() {
				vvb := vv.AsValueViewBase()
				vvb.ViewSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					svv, _ := recv.EmbeddedStruct(KiT_StructTableView).(*StructTableView)
					svv.UpdateSig()
					svv.ViewSig.Emit(svv.This, 0, nil)
				})
			}
			lbl := sg.Child(stidx + i*nWidgPerRow).(*Label)
			lbl.SetProp("vertical-align", AlignMiddle)
			idxtxt := fmt.Sprintf("%05d", i)
			lbl.Text = idxtxt
			widg := sg.Child(stidx + i*nWidgPerRow + 1 + fli).(Node2D)
			widg.SetProp("vertical-align", AlignMiddle)
			vv.ConfigWidget(widg)
			if sv.IsInactive() {
				widg.AsNode2D().SetInactive()
				if widg.TypeEmbeds(KiT_TextField) {
					tf := widg.EmbeddedStruct(KiT_TextField).(*TextField)
					tf.SetProp("stv-index", i)
					tf.Selected = false
					tf.TextFieldSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
						if sig == int64(TextFieldSelected) {
							tff := send.(*TextField)
							idx := tff.Prop("stv-index", false, false).(int)
							svv := recv.EmbeddedStruct(KiT_StructTableView).(*StructTableView)
							svv.UpdateSelect(idx, tff.Selected)
						}
					})
				}

			} else {
				addact := sg.Child(stidx + i*nWidgPerRow + nfld + 1).(*Action)
				addact.SetProp("vertical-align", AlignMiddle)
				addact.Text = " + "
				addact.Data = i
				addact.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					act := send.(*Action)
					svv := recv.EmbeddedStruct(KiT_StructTableView).(*StructTableView)
					svv.SliceNewAt(act.Data.(int) + 1)
				})
				delact := sg.Child(stidx + i*nWidgPerRow + nfld + 2).(*Action)
				delact.SetProp("vertical-align", AlignMiddle)
				delact.Text = "  --"
				delact.Data = i
				delact.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					act := send.(*Action)
					svv := recv.EmbeddedStruct(KiT_StructTableView).(*StructTableView)
					svv.SliceDelete(act.Data.(int))
				})
			}
		}
	}
	sg.UpdateEnd(updt)
}

// UpdateSelect updates the selection for the given index
func (sv *StructTableView) UpdateSelect(idx int, sel bool) {
	if sv.SelectedIdx == idx && sel { // already selected
		return
	}

	struTyp := reflect.TypeOf(sv.Slice).Elem().Elem()
	nfld := struTyp.NumField()
	sg, _ := sv.SliceGrid()

	nWidgPerRow := nfld + 1
	stidx := nfld + 1

	if sv.SelectedIdx >= 0 { // unselect current
		for fli := 0; fli < nfld; fli++ {
			widg := sg.Child(stidx + sv.SelectedIdx*nWidgPerRow + 1 + fli).(Node2D)
			if widg.TypeEmbeds(KiT_TextField) {
				tf := widg.EmbeddedStruct(KiT_TextField).(*TextField)
				tf.Selected = false
				tf.UpdateSig()
			}
		}
	}
	if sel {
		sv.SelectedIdx = idx
		for fli := 0; fli < nfld; fli++ {
			widg := sg.Child(stidx + sv.SelectedIdx*nWidgPerRow + 1 + fli).(Node2D)
			if widg.TypeEmbeds(KiT_TextField) {
				tf := widg.EmbeddedStruct(KiT_TextField).(*TextField)
				tf.Selected = true
				tf.UpdateSig()
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

// needs full rebuild and this is where we do it:
func (sv *StructTableView) Style2D() {
	sv.ConfigSliceGrid()
	sv.Frame.Style2D()
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
