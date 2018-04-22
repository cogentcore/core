// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"reflect"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  SliceView

// SliceView represents a slice, creating a property editor of the values -- constructs Children widgets to show the index / value pairs, within an overall frame with an optional title, and a button box at the bottom where methods can be invoked
type SliceView struct {
	Frame
	Slice   interface{} `desc:"the slice that we are a view onto"`
	Title   string      `desc:"title / prompt to show above the editor fields"`
	Values  []ValueView `json:"-" xml:"-" desc:"ValueView representations of the slice values"`
	TmpSave ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
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
	"#frame": ki.Props{
		"border-width":        units.NewValue(2, units.Px),
		"margin":              units.NewValue(8, units.Px),
		"padding":             units.NewValue(4, units.Px),
		"box-shadow.h-offset": units.NewValue(4, units.Px),
		"box-shadow.v-offset": units.NewValue(4, units.Px),
		"box-shadow.blur":     units.NewValue(4, units.Px),
		"box-shadow.color":    "#CCC",
	},
	"#title": ki.Props{
		// todo: add "bigger" font
		"max-width":        units.NewValue(-1, units.Px),
		"text-align":       AlignCenter,
		"vertical-align":   AlignTop,
		"background-color": "none",
	},
	"#prompt": ki.Props{
		"max-width":        units.NewValue(-1, units.Px),
		"text-align":       AlignLeft,
		"vertical-align":   AlignTop,
		"background-color": "none",
	},
}

// SetFrame configures view as a frame
func (sv *SliceView) SetFrame() {
	sv.Lay = LayoutCol
	sv.PartStyleProps(sv, SliceViewProps)
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *SliceView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// config.Add(KiT_Label, "title")
	// config.Add(KiT_Space, "title-space")
	config.Add(KiT_Layout, "slice-grid")
	// config.Add(KiT_Space, "grid-space")
	// config.Add(KiT_Layout, "buttons")
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

// SetTitle sets the title and updates the Title label
func (sv *SliceView) SetTitle(title string) {
	sv.Title = title
	lab, _ := sv.TitleWidget()
	if lab != nil {
		lab.Text = title
	}
}

// Title returns the title label widget, and its index, within frame -- nil, -1 if not found
func (sv *SliceView) TitleWidget() (*Label, int) {
	idx := sv.ChildIndexByName("title", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Label), idx
}

// SliceGrid returns the SliceGrid grid layout widget, which contains all the fields and values, and its index, within frame -- nil, -1 if not found
func (sv *SliceView) SliceGrid() (*Layout, int) {
	idx := sv.ChildIndexByName("slice-grid", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (sv *SliceView) ButtonBox() (*Layout, int) {
	idx := sv.ChildIndexByName("buttons", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ConfigSliceGrid configures the SliceGrid for the current slice
func (sv *SliceView) ConfigSliceGrid() {
	if kit.IsNil(sv.Slice) {
		return
	}
	sg, _ := sv.SliceGrid()
	if sg == nil {
		return
	}
	sg.Lay = LayoutGrid
	sg.SetProp("columns", 4)
	config := kit.TypeAndNameList{} // note: slice is already a pointer
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
		addnm := fmt.Sprintf("add-%v", idxtxt)
		delnm := fmt.Sprintf("del-%v", idxtxt)
		config.Add(KiT_Label, labnm)
		config.Add(vtyp, valnm)
		config.Add(KiT_Action, addnm)
		config.Add(KiT_Action, delnm)
		sv.Values = append(sv.Values, vv)
	}
	mods, updt := sg.ConfigChildren(config, false)
	if mods {
		sv.SetFullReRender()
	} else {
		updt = sg.UpdateStart()
	}
	for i, vv := range sv.Values {
		lbl := sg.Child(i * 4).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		idxtxt := fmt.Sprintf("%05d", i)
		lbl.Text = idxtxt
		widg := sg.Child((i * 4) + 1).(Node2D)
		widg.SetProp("vertical-align", AlignMiddle)
		vv.ConfigWidget(widg)
		addact := sg.Child(i*4 + 2).(*Action)
		addact.SetProp("vertical-align", AlignMiddle)
		addact.Text = " + "
		addact.Data = i
		addact.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			act := send.(*Action)
			svv := recv.EmbeddedStruct(KiT_SliceView).(*SliceView)
			svv.SliceNewAt(act.Data.(int) + 1)
		})
		delact := sg.Child(i*4 + 3).(*Action)
		delact.SetProp("vertical-align", AlignMiddle)
		delact.Text = "  --"
		delact.Data = i
		delact.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			act := send.(*Action)
			svv := recv.EmbeddedStruct(KiT_SliceView).(*SliceView)
			svv.SliceDelete(act.Data.(int))
		})
	}
	sg.UpdateEnd(updt)
}

// SliceNewAt inserts a new blank element at given index in the slice
func (sv *SliceView) SliceNewAt(idx int) {
	updt := sv.UpdateStart()
	svl := reflect.ValueOf(sv.Slice)
	svnp := kit.NonPtrValue(svl)
	svtyp := svnp.Type()
	nval := reflect.New(svtyp.Elem())
	sz := svnp.Len()
	svnp = reflect.Append(svnp, nval.Elem())
	if idx < sz-1 {
		reflect.Copy(svnp.Slice(idx+1, sz+1), svnp.Slice(idx, sz))
		svnp.Index(idx).Set(nval.Elem())
	}
	svl.Elem().Set(svnp)
	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.SetFullReRender()
	sv.UpdateEnd(updt)
}

// SliceDelete deletes element at given index from slice
func (sv *SliceView) SliceDelete(idx int) {
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
}

func (sv *SliceView) UpdateFromSlice() {
	mods, updt := sv.StdConfig()
	// typ := kit.NonPtrType(reflect.TypeOf(sv.Slice))
	// sv.SetTitle(fmt.Sprintf("%v Values", typ.Name()))
	sv.ConfigSliceGrid()
	if mods {
		sv.UpdateEnd(updt)
	}
}

// needs full rebuild and this is where we do it:
func (sv *SliceView) Style2D() {
	sv.ConfigSliceGrid()
	sv.Frame.Style2D()
}

func (sv *SliceView) Render2D() {
	sv.ClearFullReRender()
	sv.Frame.Render2D()
}

func (sv *SliceView) ReRender2D() (node Node2D, layout bool) {
	if sv.NeedsFullReRender() {
		node = nil
		layout = false
	} else {
		node = sv.This.(Node2D)
		layout = true
	}
	return
}

// check for interface implementation
var _ Node2D = &SliceView{}

////////////////////////////////////////////////////////////////////////////////////////
//  SliceViewInline

// SliceViewInline represents a slice as a single line widget, for smaller slices and those explicitly marked inline -- constructs widgets in Parts to show the key names and editor vals for each value
type SliceViewInline struct {
	WidgetBase
	Slice        interface{} `desc:"the slice that we are a view onto"`
	SliceViewSig ki.Signal   `json:"-" xml:"-" desc:"signal for SliceView -- see SliceViewSignals for the types"`
	Values       []ValueView `json:"-" xml:"-" desc:"ValueView representations of the fields"`
	TmpSave      ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
}

var KiT_SliceViewInline = kit.Types.AddType(&SliceViewInline{}, nil)

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

var SliceViewInlineProps = ki.Props{}

// ConfigParts configures Parts for the current slice
func (sv *SliceViewInline) ConfigParts() {
	if kit.IsNil(sv.Slice) {
		return
	}
	sv.Parts.Lay = LayoutRow
	config := kit.TypeAndNameList{} // note: slice is already a pointer
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
		config.Add(KiT_Label, labnm)
		config.Add(vtyp, valnm)
		sv.Values = append(sv.Values, vv)
	}
	config.Add(KiT_Action, "EditAction")
	mods, updt := sv.Parts.ConfigChildren(config, false)
	if !mods {
		updt = sv.Parts.UpdateStart()
	}
	for i, vv := range sv.Values {
		lbl := sv.Parts.Child(i * 2).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		idxtxt := fmt.Sprintf("%05d", i)
		lbl.Text = idxtxt
		widg := sv.Parts.Child((i * 2) + 1).(Node2D)
		widg.SetProp("vertical-align", AlignMiddle)
		vv.ConfigWidget(widg)
	}
	edac := sv.Parts.Child(-1).(*Action)
	edac.SetProp("vertical-align", AlignMiddle)
	edac.Text = "  ..."
	edac.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.EmbeddedStruct(KiT_SliceViewInline).(*SliceViewInline)
		SliceViewDialog(svv.Viewport, svv.Slice, svv.TmpSave, "Slice Value View", "", svv.This,
			func(recv, send ki.Ki, sig int64, data interface{}) {
				svvv := recv.EmbeddedStruct(KiT_SliceViewInline).(*SliceViewInline)
				svpar := svvv.ParentByType(KiT_StructView, true).EmbeddedStruct(KiT_StructView).(*StructView)
				if svpar != nil {
					svpar.SetFullReRender()
					svpar.UpdateSig()
				}
			})
	})
	sv.Parts.UpdateEnd(updt)
}

func (sv *SliceViewInline) UpdateFromSlice() {
	sv.ConfigParts()
}

func (sv *SliceViewInline) Style2D() {
	sv.ConfigParts()
	sv.WidgetBase.Style2D()
}

func (sv *SliceViewInline) Render2D() {
	if sv.PushBounds() {
		sv.ConfigParts()
		sv.Render2DParts()
		sv.Render2DChildren()
		sv.PopBounds()
	}
}

// todo: see notes on treeview
func (sv *SliceViewInline) ReRender2D() (node Node2D, layout bool) {
	node = sv.This.(Node2D)
	layout = true
	return
}

// check for interface implementation
var _ Node2D = &SliceViewInline{}
