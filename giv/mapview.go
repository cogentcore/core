// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// MapView represents a map, creating a property editor of the values --
// constructs Children widgets to show the key / value pairs, within an
// overall frame with a button box at the bottom where methods can be invoked.
type MapView struct {
	gi.Frame
	Map     interface{} `desc:"the map that we are a view onto"`
	Keys    []ValueView `json:"-" xml:"-" desc:"ValueView representations of the map keys"`
	Values  []ValueView `json:"-" xml:"-" desc:"ValueView representations of the map values"`
	TmpSave ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig ki.Signal   `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
}

var KiT_MapView = kit.Types.AddType(&MapView{}, MapViewProps)

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
	"background-color": &gi.Prefs.BackgroundColor,
}

// SetFrame configures view as a frame
func (mv *MapView) SetFrame() {
	mv.Lay = gi.LayoutVert
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (mv *MapView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Frame, "map-grid")
	config.Add(gi.KiT_Space, "grid-space")
	config.Add(gi.KiT_Layout, "buttons")
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

// MapGrid returns the MapGrid grid layout widget, which contains all the fields and values, and its index, within frame -- nil, -1 if not found
func (mv *MapView) MapGrid() (*gi.Frame, int) {
	idx, ok := mv.Children().IndexByName("map-grid", 0)
	if !ok {
		return nil, -1
	}
	return mv.KnownChild(idx).(*gi.Frame), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (mv *MapView) ButtonBox() (*gi.Layout, int) {
	idx, ok := mv.Children().IndexByName("buttons", 0)
	if !ok {
		return nil, -1
	}
	return mv.KnownChild(idx).(*gi.Layout), idx
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
	sg.Lay = gi.LayoutGrid
	// setting a pref here is key for giving it a scrollbar in larger context
	sg.SetMinPrefHeight(units.NewValue(10, units.Em))
	sg.SetMinPrefWidth(units.NewValue(10, units.Em))
	sg.SetStretchMaxHeight() // for this to work, ALL layers above need it too
	sg.SetStretchMaxWidth()  // for this to work, ALL layers above need it too
	config := kit.TypeAndNameList{}
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
	valtypes = append(valtypes, reflect.TypeOf((*reflect.Type)(nil)).Elem())

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
			config.Add(gi.KiT_ComboBox, typnm)
		}
		config.Add(gi.KiT_Action, delnm)
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
		vvb := vv.AsValueViewBase()
		vvb.ViewSig.ConnectOnly(mv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			mvv, _ := recv.EmbeddedStruct(KiT_MapView).(*MapView)
			mvv.ViewSig.Emit(mvv.This, 0, nil)
		})
		keyw := sg.KnownChild(i * ncol).(gi.Node2D)
		widg := sg.KnownChild(i*ncol + 1).(gi.Node2D)
		kv := mv.Keys[i]
		kv.ConfigWidget(keyw)
		vv.ConfigWidget(widg)
		if ifaceType {
			typw := sg.KnownChild(i*ncol + 2).(*gi.ComboBox)
			typw.ItemsFromTypes(valtypes, false, true, 50)
			vtyp := kit.NonPtrType(reflect.TypeOf(vv.Val().Interface()))
			if vtyp == nil {
				vtyp = strtyp // default to string
			}
			typw.SetCurVal(vtyp)
			typw.SetProp("mapview-index", i)
			typw.ComboSig.ConnectOnly(mv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				cb := send.(*gi.ComboBox)
				typ := cb.CurVal.(reflect.Type)
				idx := cb.KnownProp("mapview-index").(int)
				mvv := recv.EmbeddedStruct(KiT_MapView).(*MapView)
				mvv.MapChangeValueType(idx, typ)
			})
		}
		delact := sg.KnownChild(i*ncol + ncol - 1).(*gi.Action)
		delact.SetIcon("minus")
		delact.Tooltip = "delete item"
		delact.Data = kv
		delact.ActionSig.ConnectOnly(mv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			act := send.(*gi.Action)
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
	defer mv.UpdateEnd(updt)

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
	mv.ConfigMapGrid()
	mv.ViewSig.Emit(mv.This, 0, nil)
}

func (mv *MapView) MapAdd() {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	defer mv.UpdateEnd(updt)

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
	mv.ConfigMapGrid()
	mv.ViewSig.Emit(mv.This, 0, nil)
}

func (mv *MapView) MapDelete(key reflect.Value) {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	defer mv.UpdateEnd(updt)

	mpv := reflect.ValueOf(mv.Map)
	mpvnp := kit.NonPtrValue(mpv)
	mpvnp.SetMapIndex(key, reflect.Value{}) // delete
	if mv.TmpSave != nil {
		mv.TmpSave.SaveTmp()
	}
	mv.ConfigMapGrid()
	mv.ViewSig.Emit(mv.This, 0, nil)
}

// ConfigMapButtons configures the buttons for map functions
func (mv *MapView) ConfigMapButtons() {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	bb, _ := mv.ButtonBox()
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Button, "Add")
	mods, updt := bb.ConfigChildren(config, false)
	addb := bb.KnownChildByName("Add", 0).EmbeddedStruct(gi.KiT_Button).(*gi.Button)
	addb.SetText("Add")
	addb.ButtonSig.ConnectOnly(mv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
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
	mv.ConfigMapGrid()
	mv.ConfigMapButtons()
	if mods {
		mv.UpdateEnd(updt)
	}
}

func (mv *MapView) UpdateValues() {
	// maps have to re-read their values -- can't get pointers
	mv.ConfigMapGrid()
}
