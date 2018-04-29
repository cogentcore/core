// Use of this source code is governed by a BSD-style
// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  MapView

// MapView represents a map, creating a property editor of the values -- constructs Children widgets to show the key / value pairs, within an overall frame with an optional title, and a button box at the bottom where methods can be invoked
type MapView struct {
	Frame
	Map     interface{} `desc:"the map that we are a view onto"`
	Title   string      `desc:"title / prompt to show above the editor fields"`
	Keys    []ValueView `json:"-" xml:"-" desc:"ValueView representations of the map keys"`
	Values  []ValueView `json:"-" xml:"-" desc:"ValueView representations of the map values"`
	TmpSave ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
}

var KiT_MapView = kit.Types.AddType(&MapView{}, MapViewProps)

func (n *MapView) New() ki.Ki { return &MapView{} }

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// SetMap sets the source map that we are viewing -- rebuilds the children to
// represent this map
func (mv *MapView) SetMap(mp interface{}, tmpSave ValueView) {
	// note: because we make new maps, and due to the strangeness of reflect, they
	// end up not being comparable types, so we can't check if equal
	mv.Map = mp
	mv.TmpSave = tmpSave
	mv.UpdateFromMap()
}

var MapViewProps = ki.Props{
	"background-color": &Prefs.BackgroundColor,
	"#title": ki.Props{
		"max-width":      units.NewValue(-1, units.Px),
		"text-align":     AlignCenter,
		"vertical-align": AlignTop,
	},
	"#prompt": ki.Props{
		"max-width":      units.NewValue(-1, units.Px),
		"text-align":     AlignLeft,
		"vertical-align": AlignTop,
	},
}

// SetFrame configures view as a frame
func (mv *MapView) SetFrame() {
	mv.Lay = LayoutCol
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (mv *MapView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// config.Add(KiT_Label, "title")
	// config.Add(KiT_Space, "title-space")
	config.Add(KiT_Layout, "map-grid")
	config.Add(KiT_Space, "grid-space")
	config.Add(KiT_Layout, "buttons")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (mv *MapView) StdConfig() (mods, updt bool) {
	mv.SetFrame()
	config := mv.StdFrameConfig()
	mods, updt = mv.ConfigChildren(config, false)
	return
}

// SetTitle sets the title and updates the Title label
func (mv *MapView) SetTitle(title string) {
	mv.Title = title
	lab, _ := mv.TitleWidget()
	if lab != nil {
		lab.Text = title
	}
}

// Title returns the title label widget, and its index, within frame -- nil, -1 if not found
func (mv *MapView) TitleWidget() (*Label, int) {
	idx := mv.ChildIndexByName("title", 0)
	if idx < 0 {
		return nil, -1
	}
	return mv.Child(idx).(*Label), idx
}

// MapGrid returns the MapGrid grid layout widget, which contains all the fields and values, and its index, within frame -- nil, -1 if not found
func (mv *MapView) MapGrid() (*Layout, int) {
	idx := mv.ChildIndexByName("map-grid", 0)
	if idx < 0 {
		return nil, -1
	}
	return mv.Child(idx).(*Layout), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (mv *MapView) ButtonBox() (*Layout, int) {
	idx := mv.ChildIndexByName("buttons", 0)
	if idx < 0 {
		return nil, -1
	}
	return mv.Child(idx).(*Layout), idx
}

// ConfigMapGrid configures the MapGrid for the current map
func (mv *MapView) ConfigMapGrid() {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	sg, _ := mv.MapGrid()
	if sg == nil {
		return
	}
	sg.Lay = LayoutGrid
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// always start fresh!
	mv.Keys = make([]ValueView, 0)
	mv.Values = make([]ValueView, 0)

	mpv := reflect.ValueOf(mv.Map)
	mpvnp := kit.NonPtrValue(mpv)

	valtyp := kit.NonPtrType(reflect.TypeOf(mv.Map)).Elem()
	ncol := 3
	ifaceType := false
	typeTag := ""
	strtyp := reflect.TypeOf(typeTag)
	if valtyp.Kind() == reflect.Interface && valtyp.String() == "interface {}" {
		ifaceType = true
		ncol = 4
		typeTag = "style-prop" // todo: need some way of setting & getting
		// this for given domain mapview could have a structview parent and
		// the source node of that struct, if a Ki, could have a property --
		// unlike inline case, plain mapview is not a child of struct view
		// directly -- but field on struct view does create the mapview
		// dialog.. a bit hacky and indirect..
	}

	valtypes := append(kit.Types.AllTagged(typeTag), kit.Enums.AllTagged(typeTag)...)
	valtypes = append(valtypes, kit.Types.AllTagged("basic-type")...)

	sg.SetProp("columns", ncol)

	keys := mpvnp.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return kit.ToString(keys[i]) < kit.ToString(keys[j])
	})
	for _, key := range keys {
		kv := ToValueView(key.Interface())
		if kv == nil { // shouldn't happen
			continue
		}
		kv.SetMapKey(key, mv.Map, mv.TmpSave)

		val := mpvnp.MapIndex(key)
		vv := ToValueView(val.Interface())
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetMapValue(val, mv.Map, key.Interface(), kv, mv.TmpSave) // needs key value view to track updates

		keytxt := kit.ToString(key.Interface())
		keynm := fmt.Sprintf("key-%v", keytxt)
		valnm := fmt.Sprintf("value-%v", keytxt)
		delnm := fmt.Sprintf("del-%v", keytxt)

		config.Add(kv.WidgetType(), keynm)
		config.Add(vv.WidgetType(), valnm)
		if ifaceType {
			typnm := fmt.Sprintf("type-%v", keytxt)
			config.Add(KiT_ComboBox, typnm)
		}
		config.Add(KiT_Action, delnm)
		mv.Keys = append(mv.Keys, kv)
		mv.Values = append(mv.Values, vv)
	}
	mods, updt := sg.ConfigChildren(config, false)
	if mods {
		mv.SetFullReRender()
	} else {
		updt = sg.UpdateStart() // cover rest of updates, which can happen even if same config
	}
	for i, vv := range mv.Values {
		keyw := sg.Child(i * ncol).(Node2D)
		keyw.SetProp("vertical-align", AlignMiddle)
		widg := sg.Child(i*ncol + 1).(Node2D)
		widg.SetProp("vertical-align", AlignMiddle)
		kv := mv.Keys[i]
		kv.ConfigWidget(keyw)
		vv.ConfigWidget(widg)
		if ifaceType {
			typw := sg.Child(i*ncol + 2).(*ComboBox)
			typw.ItemsFromTypes(valtypes, false, true, 50)
			vtyp := kit.NonPtrType(reflect.TypeOf(vv.Val().Interface()))
			if vtyp == nil {
				vtyp = strtyp // default to string
			}
			typw.SetCurVal(vtyp)
			typw.SetProp("mapview-index", i)
			typw.ComboSig.ConnectOnly(mv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				cb := send.(*ComboBox)
				typ := cb.CurVal.(reflect.Type)
				idx := cb.Prop("mapview-index", false, false).(int)
				mvv := recv.EmbeddedStruct(KiT_MapView).(*MapView)
				mvv.MapChangeValueType(idx, typ)
			})
		}
		delact := sg.Child(i*ncol + ncol - 1).(*Action)
		delact.SetProp("vertical-align", AlignMiddle)
		delact.Text = "  --"
		delact.Data = kv
		delact.ActionSig.ConnectOnly(mv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			act := send.(*Action)
			mvv := recv.EmbeddedStruct(KiT_MapView).(*MapView)
			mvv.MapDelete(act.Data.(ValueView).Val())
		})
	}
	sg.UpdateEnd(updt)
}

// MapChangeValueType changes the type of the value for given map element at
// idx -- for maps with interface{} values
func (mv *MapView) MapChangeValueType(idx int, typ reflect.Type) {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	keyv := mv.Keys[idx]
	ck := keyv.Val() // current key value
	valv := mv.Values[idx]
	cv := kit.NonPtrValue(valv.Val()) // current val value

	// create a new item of selected type, and attempt to convert existing to it
	var evn reflect.Value
	if kit.ValueIsZero(cv) {
		evn = kit.MakeOfType(typ)
	} else {
		evn = kit.CloneToType(typ, cv.Interface())
	}
	ov := kit.NonPtrValue(reflect.ValueOf(mv.Map))
	valv.AsValueViewBase().Value = evn.Elem()
	ov.SetMapIndex(ck, evn.Elem())
	if mv.TmpSave != nil {
		mv.TmpSave.SaveTmp()
	}
	mv.SetFullReRender()
	mv.UpdateEnd(updt)
	mv.FullReRenderParentStructView()
}

func (mv *MapView) MapAdd() {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	mpv := reflect.ValueOf(mv.Map)
	mpvnp := kit.NonPtrValue(mpv)
	mvtyp := mpvnp.Type()
	valtyp := kit.NonPtrType(reflect.TypeOf(mv.Map)).Elem()
	if valtyp.Kind() == reflect.Interface && valtyp.String() == "interface {}" {
		str := ""
		valtyp = reflect.TypeOf(str)
	}
	nkey := reflect.New(mvtyp.Key())
	nval := reflect.New(valtyp)
	if mpvnp.IsNil() { // make a new map
		nmp := kit.MakeMap(mvtyp)
		mpv.Elem().Set(nmp.Elem())
		mpvnp = kit.NonPtrValue(mpv)
	}
	mpvnp.SetMapIndex(nkey.Elem(), nval.Elem())
	if mv.TmpSave != nil {
		mv.TmpSave.SaveTmp()
	}
	mv.SetFullReRender()
	mv.UpdateEnd(updt)
	mv.FullReRenderParentStructView()
}

func (mv *MapView) MapDelete(key reflect.Value) {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	mpv := reflect.ValueOf(mv.Map)
	mpvnp := kit.NonPtrValue(mpv)
	mpvnp.SetMapIndex(key, reflect.Value{}) // delete
	if mv.TmpSave != nil {
		mv.TmpSave.SaveTmp()
	}
	mv.SetFullReRender()
	mv.UpdateEnd(updt)
	mv.FullReRenderParentStructView()
}

func (mv *MapView) FullReRenderParentStructView() {
	// todo: this will typically fail because MapView is in a separate dialog
	// need some way of hooking back to the struct view -- another TmpSave??
	if mv.TmpSave == nil {
		return
	}
	svp := mv.TmpSave.ParentByType(KiT_StructView, true)
	if svp == nil {
		return
	}
	svpar := svp.EmbeddedStruct(KiT_StructView).(*StructView)
	if svpar != nil {
		svpar.SetFullReRender()
		svpar.UpdateSig()
	}
}

// ConfigMapButtons configures the buttons for map functions
func (mv *MapView) ConfigMapButtons() {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	bb, _ := mv.ButtonBox()
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	config.Add(KiT_Button, "Add")
	mods, updt := bb.ConfigChildren(config, false)
	addb := bb.ChildByName("Add", 0).EmbeddedStruct(KiT_Button).(*Button)
	addb.SetText("Add")
	addb.ButtonSig.ConnectOnly(mv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			mvv := recv.EmbeddedStruct(KiT_MapView).(*MapView)
			mvv.MapAdd()
		}
	})
	if mods {
		bb.UpdateEnd(updt)
	}
}

func (mv *MapView) UpdateFromMap() {
	mods, updt := mv.StdConfig()
	// typ := kit.NonPtrType(reflect.TypeOf(mv.Map))
	// mv.SetTitle(fmt.Sprintf("%v Values", typ.Name()))
	mv.ConfigMapGrid()
	mv.ConfigMapButtons()
	if mods {
		mv.UpdateEnd(updt)
	}
}

// needs full rebuild and this is where we do it:
func (mv *MapView) Style2D() {
	mv.ConfigMapGrid()
	mv.Frame.Style2D()
}

func (mv *MapView) Render2D() {
	mv.ClearFullReRender()
	mv.Frame.Render2D()
}

func (mv *MapView) ReRender2D() (node Node2D, layout bool) {
	if mv.NeedsFullReRender() {
		node = nil
		layout = false
	} else {
		node = mv.This.(Node2D)
		layout = true
	}
	return
}

// check for interface implementation
var _ Node2D = &MapView{}

////////////////////////////////////////////////////////////////////////////////////////
//  MapViewInline

// MapViewInline represents a map as a single line widget, for smaller maps and those explicitly marked inline -- constructs widgets in Parts to show the key names and editor vals for each value
type MapViewInline struct {
	WidgetBase
	Map        interface{} `desc:"the map that we are a view onto"`
	MapViewSig ki.Signal   `json:"-" xml:"-" desc:"signal for MapView -- see MapViewSignals for the types"`
	Keys       []ValueView `json:"-" xml:"-" desc:"ValueView representations of the map keys"`
	Values     []ValueView `json:"-" xml:"-" desc:"ValueView representations of the fields"`
	TmpSave    ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
}

var KiT_MapViewInline = kit.Types.AddType(&MapViewInline{}, MapViewInlineProps)

func (n *MapViewInline) New() ki.Ki { return &MapViewInline{} }

// SetMap sets the source map that we are viewing -- rebuilds the children to represent this map
func (mv *MapViewInline) SetMap(mp interface{}, tmpSave ValueView) {
	// note: because we make new maps, and due to the strangeness of reflect, they
	// end up not being comparable types, so we can't check if equal
	mv.Map = mp
	mv.TmpSave = tmpSave
	mv.UpdateFromMap()
}

var MapViewInlineProps = ki.Props{
	"min-width": units.NewValue(60, units.Ex),
}

// todo: maybe figure out a way to share some of this redundant code..

// ConfigParts configures Parts for the current map
func (mv *MapViewInline) ConfigParts() {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	mv.Parts.Lay = LayoutRow
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// always start fresh!
	mv.Keys = make([]ValueView, 0)
	mv.Values = make([]ValueView, 0)

	mpv := reflect.ValueOf(mv.Map)
	mpvnp := kit.NonPtrValue(mpv)

	keys := mpvnp.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return kit.ToString(keys[i]) < kit.ToString(keys[j])
	})
	for _, key := range keys {
		kv := ToValueView(key.Interface())
		if kv == nil { // shouldn't happen
			continue
		}
		kv.SetMapKey(key, mv.Map, mv.TmpSave)

		val := mpvnp.MapIndex(key)
		vv := ToValueView(val.Interface())
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetMapValue(val, mv.Map, key.Interface(), kv, mv.TmpSave) // needs key value view to track updates

		keytxt := kit.ToString(key.Interface())
		keynm := fmt.Sprintf("key-%v", keytxt)
		valnm := fmt.Sprintf("value-%v", keytxt)

		config.Add(kv.WidgetType(), keynm)
		config.Add(vv.WidgetType(), valnm)
		mv.Keys = append(mv.Keys, kv)
		mv.Values = append(mv.Values, vv)
	}
	config.Add(KiT_Action, "EditAction")
	mods, updt := mv.Parts.ConfigChildren(config, false)
	if !mods {
		updt = mv.Parts.UpdateStart()
	}
	for i, vv := range mv.Values {
		keyw := mv.Parts.Child(i * 2).(Node2D)
		keyw.SetProp("vertical-align", AlignMiddle)
		widg := mv.Parts.Child((i * 2) + 1).(Node2D)
		widg.SetProp("vertical-align", AlignMiddle)
		kv := mv.Keys[i]
		kv.ConfigWidget(keyw)
		vv.ConfigWidget(widg)
	}
	edac := mv.Parts.Child(-1).(*Action)
	edac.SetProp("vertical-align", AlignMiddle)
	edac.Text = "  ..."
	edac.ActionSig.ConnectOnly(mv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		mvv, _ := recv.EmbeddedStruct(KiT_MapViewInline).(*MapViewInline)
		MapViewDialog(mvv.Viewport, mvv.Map, mvv.TmpSave, "Map Value View", "", mvv.This,
			func(recv, send ki.Ki, sig int64, data interface{}) {
				mvvv := recv.EmbeddedStruct(KiT_MapViewInline).(*MapViewInline)
				mvvv.FullReRenderParentStructView()
			})
	})
	mv.Parts.UpdateEnd(updt)
}

func (mv *MapViewInline) FullReRenderParentStructView() {
	svpar := mv.ParentByType(KiT_StructView, true).EmbeddedStruct(KiT_StructView).(*StructView)
	if svpar != nil {
		svpar.SetFullReRender()
		svpar.UpdateSig()
	}
}

func (mv *MapViewInline) UpdateFromMap() {
	mv.ConfigParts()
}

func (mv *MapViewInline) Style2D() {
	mv.ConfigParts()
	mv.WidgetBase.Style2D()
}

func (mv *MapViewInline) Render2D() {
	if mv.PushBounds() {
		mv.Render2DParts()
		mv.Render2DChildren()
		mv.PopBounds()
	}
}

// todo: see notes on treeview
func (mv *MapViewInline) ReRender2D() (node Node2D, layout bool) {
	node = mv.This.(Node2D)
	layout = true
	return
}

// check for interface implementation
var _ Node2D = &MapViewInline{}
