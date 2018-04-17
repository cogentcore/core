// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
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
	Keys    []ValueView `desc:"ValueView representations of the map keys"`
	Values  []ValueView `desc:"ValueView representations of the map values"`
	TmpSave ValueView   `desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
}

var KiT_MapView = kit.Types.AddType(&MapView{}, MapViewProps)

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// SetMap sets the source map that we are viewing -- rebuilds the children to represent this map
func (sv *MapView) SetMap(mp interface{}, tmpSave ValueView) {
	sv.UpdateStart()
	sv.Map = mp
	sv.TmpSave = tmpSave
	sv.UpdateFromMap()
	sv.UpdateEnd()
}

var MapViewProps = map[string]interface{}{
	"#frame": map[string]interface{}{
		"border-width":        units.NewValue(2, units.Px),
		"margin":              units.NewValue(8, units.Px),
		"padding":             units.NewValue(4, units.Px),
		"box-shadow.h-offset": units.NewValue(4, units.Px),
		"box-shadow.v-offset": units.NewValue(4, units.Px),
		"box-shadow.blur":     units.NewValue(4, units.Px),
		"box-shadow.color":    "#CCC",
	},
	"#title": map[string]interface{}{
		// todo: add "bigger" font
		"max-width":        units.NewValue(-1, units.Px),
		"text-align":       AlignCenter,
		"vertical-align":   AlignTop,
		"background-color": "none",
	},
	"#prompt": map[string]interface{}{
		"max-width":        units.NewValue(-1, units.Px),
		"text-align":       AlignLeft,
		"vertical-align":   AlignTop,
		"background-color": "none",
	},
}

// SetFrame configures view as a frame
func (sv *MapView) SetFrame() {
	sv.Lay = LayoutCol
	sv.PartStyleProps(sv, MapViewProps)
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *MapView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// config.Add(KiT_Label, "title")
	// config.Add(KiT_Space, "title-space")
	config.Add(KiT_Layout, "map-grid")
	config.Add(KiT_Space, "grid-space")
	config.Add(KiT_Layout, "buttons")
	return config
}

// StdConfig configures a standard setup of the overall Frame
func (sv *MapView) StdConfig() {
	sv.SetFrame()
	config := sv.StdFrameConfig()
	sv.ConfigChildren(config, false)
}

// SetTitle sets the title and updates the Title label
func (sv *MapView) SetTitle(title string) {
	sv.Title = title
	lab, _ := sv.TitleWidget()
	if lab != nil {
		lab.Text = title
	}
}

// Title returns the title label widget, and its index, within frame -- nil, -1 if not found
func (sv *MapView) TitleWidget() (*Label, int) {
	idx := sv.ChildIndexByName("title", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Label), idx
}

// MapGrid returns the MapGrid grid layout widget, which contains all the fields and values, and its index, within frame -- nil, -1 if not found
func (sv *MapView) MapGrid() (*Layout, int) {
	idx := sv.ChildIndexByName("map-grid", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (sv *MapView) ButtonBox() (*Layout, int) {
	idx := sv.ChildIndexByName("buttons", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ConfigMapGrid configures the MapGrid for the current map
func (sv *MapView) ConfigMapGrid() {
	if kit.IsNil(sv.Map) {
		return
	}
	sg, _ := sv.MapGrid()
	if sg == nil {
		return
	}
	sg.Lay = LayoutGrid
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// always start fresh!
	sv.Keys = make([]ValueView, 0)
	sv.Values = make([]ValueView, 0)

	mv := reflect.ValueOf(sv.Map)
	mvnp := kit.NonPtrValue(mv)

	valtyp := kit.NonPtrType(reflect.TypeOf(sv.Map)).Elem()
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

	keys := mvnp.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return kit.ToString(keys[i]) < kit.ToString(keys[j])
	})
	for _, key := range keys {
		kv := ToValueView(key.Interface())
		if kv == nil { // shouldn't happen
			continue
		}
		kv.SetMapKey(key, sv.Map, sv.TmpSave)

		val := mvnp.MapIndex(key)
		vv := ToValueView(val.Interface())
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetMapValue(val, sv.Map, key.Interface(), kv, sv.TmpSave) // needs key value view to track updates

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
		sv.Keys = append(sv.Keys, kv)
		sv.Values = append(sv.Values, vv)
	}
	updt := sg.ConfigChildren(config, false)
	if updt {
		sv.SetFullReRender()
	}
	for i, vv := range sv.Values {
		keyw := sg.Child(i * ncol).(Node2D)
		keyw.SetProp("vertical-align", AlignMiddle)
		widg := sg.Child(i*ncol + 1).(Node2D)
		widg.SetProp("vertical-align", AlignMiddle)
		kv := sv.Keys[i]
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
			typw.ComboSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				cb := send.(*ComboBox)
				idx := cb.Prop("mapview-index", false, false).(int)
				svv := recv.EmbeddedStruct(KiT_MapView).(*MapView)
				svv.UpdateStart()
				typ := cb.CurVal.(reflect.Type)
				keyv := svv.Keys[idx]
				ck := keyv.Val() // current key value
				valv := svv.Values[idx]
				cv := kit.NonPtrValue(valv.Val()) // current val value

				// create a new item of selected type, and attempt to convert existing to it
				evn := reflect.New(typ)
				evi := evn.Interface()
				kit.SetRobust(evi, cv.Interface())
				ov := kit.NonPtrValue(reflect.ValueOf(svv.Map))
				ov.SetMapIndex(ck, reflect.ValueOf(evi).Elem())
				if svv.TmpSave != nil {
					svv.TmpSave.SaveTmp()
				}
				svv.SetFullReRender()
				svv.UpdateEnd()
			})
		}
		delact := sg.Child(i*ncol + ncol - 1).(*Action)
		delact.SetProp("vertical-align", AlignMiddle)
		delact.Text = "  --"
		delact.Data = kv
		// delact.ActionSig.DisconnectAll()
		delact.ActionSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			act := send.(*Action)
			svv := recv.EmbeddedStruct(KiT_MapView).(*MapView)
			svv.UpdateStart()
			svv.MapDelete(act.Data.(ValueView).Val())
			svv.SetFullReRender()
			svv.UpdateEnd()
		})
	}
}

func (sv *MapView) MapAdd() {
	if kit.IsNil(sv.Map) {
		return
	}
	sv.UpdateStart()
	mv := reflect.ValueOf(sv.Map)
	mvnp := kit.NonPtrValue(mv)
	mvtyp := mvnp.Type()
	valtyp := kit.NonPtrType(reflect.TypeOf(sv.Map)).Elem()
	if valtyp.Kind() == reflect.Interface && valtyp.String() == "interface {}" {
		str := ""
		valtyp = reflect.TypeOf(str)
	}
	nkey := reflect.New(mvtyp.Key())
	nval := reflect.New(valtyp)
	mvnp.SetMapIndex(nkey.Elem(), nval.Elem())
	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.UpdateEnd()
}

func (sv *MapView) MapDelete(key reflect.Value) {
	if kit.IsNil(sv.Map) {
		return
	}
	sv.UpdateStart()
	mv := reflect.ValueOf(sv.Map)
	mvnp := kit.NonPtrValue(mv)
	mvnp.SetMapIndex(key, reflect.Value{}) // delete
	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.UpdateEnd()
}

// ConfigMapButtons configures the buttons for map functions
func (sv *MapView) ConfigMapButtons() {
	if kit.IsNil(sv.Map) {
		return
	}
	bb, _ := sv.ButtonBox()
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	config.Add(KiT_Button, "Add")
	bb.ConfigChildren(config, false)
	addb := bb.ChildByName("Add", 0).EmbeddedStruct(KiT_Button).(*Button)
	addb.SetText("Add")
	addb.ButtonSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			svv := recv.EmbeddedStruct(KiT_MapView).(*MapView)
			svv.UpdateStart()
			svv.MapAdd()
			svv.SetFullReRender()
			svv.UpdateEnd()
		}
	})
}

func (sv *MapView) UpdateFromMap() {
	sv.StdConfig()
	// typ := kit.NonPtrType(reflect.TypeOf(sv.Map))
	// sv.SetTitle(fmt.Sprintf("%v Values", typ.Name()))
	sv.ConfigMapGrid()
	sv.ConfigMapButtons()
}

// needs full rebuild and this is where we do it:
func (sv *MapView) Style2D() {
	sv.ConfigMapGrid()
	sv.Frame.Style2D()
}

func (sv *MapView) Render2D() {
	sv.ClearFullReRender()
	sv.Frame.Render2D()
}

func (sv *MapView) ReRender2D() (node Node2D, layout bool) {
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
var _ Node2D = &MapView{}

////////////////////////////////////////////////////////////////////////////////////////
//  MapViewInline

// MapViewInline represents a map as a single line widget, for smaller maps and those explicitly marked inline -- constructs widgets in Parts to show the key names and editor vals for each value
type MapViewInline struct {
	WidgetBase
	Map        interface{} `desc:"the map that we are a view onto"`
	MapViewSig ki.Signal   `json:"-" desc:"signal for MapView -- see MapViewSignals for the types"`
	Keys       []ValueView `desc:"ValueView representations of the map keys"`
	Values     []ValueView `desc:"ValueView representations of the fields"`
	TmpSave    ValueView   `desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
}

var KiT_MapViewInline = kit.Types.AddType(&MapViewInline{}, nil)

// SetMap sets the source map that we are viewing -- rebuilds the children to represent this map
func (sv *MapViewInline) SetMap(st interface{}, tmpSave ValueView) {
	sv.UpdateStart()
	sv.Map = st
	sv.TmpSave = tmpSave
	sv.UpdateFromMap()
	sv.UpdateEnd()
}

var MapViewInlineProps = map[string]interface{}{}

// todo: maybe figure out a way to share some of this redundant code..

// ConfigParts configures Parts for the current map
func (sv *MapViewInline) ConfigParts() {
	if kit.IsNil(sv.Map) {
		return
	}
	sv.Parts.Lay = LayoutRow
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// always start fresh!
	sv.Keys = make([]ValueView, 0)
	sv.Values = make([]ValueView, 0)

	mv := reflect.ValueOf(sv.Map)
	mvnp := kit.NonPtrValue(mv)

	keys := mvnp.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return kit.ToString(keys[i]) < kit.ToString(keys[j])
	})
	for _, key := range keys {
		kv := ToValueView(key.Interface())
		if kv == nil { // shouldn't happen
			continue
		}
		kv.SetMapKey(key, sv.Map, sv.TmpSave)

		val := mvnp.MapIndex(key)
		vv := ToValueView(val.Interface())
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetMapValue(val, sv.Map, key.Interface(), kv, sv.TmpSave) // needs key value view to track updates

		keytxt := kit.ToString(key.Interface())
		keynm := fmt.Sprintf("key-%v", keytxt)
		valnm := fmt.Sprintf("value-%v", keytxt)

		config.Add(kv.WidgetType(), keynm)
		config.Add(vv.WidgetType(), valnm)
		sv.Keys = append(sv.Keys, kv)
		sv.Values = append(sv.Values, vv)
	}
	config.Add(KiT_Action, "EditAction")
	sv.Parts.ConfigChildren(config, false)
	for i, vv := range sv.Values {
		keyw := sv.Parts.Child(i * 2).(Node2D)
		keyw.SetProp("vertical-align", AlignMiddle)
		widg := sv.Parts.Child((i * 2) + 1).(Node2D)
		widg.SetProp("vertical-align", AlignMiddle)
		kv := sv.Keys[i]
		kv.ConfigWidget(keyw)
		vv.ConfigWidget(widg)
	}
	edac := sv.Parts.Child(-1).(*Action)
	edac.SetProp("vertical-align", AlignMiddle)
	edac.Text = "  ..."
	// edac.ActionSig.DisconnectAll()
	edac.ActionSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.EmbeddedStruct(KiT_MapViewInline).(*MapViewInline)
		MapViewDialog(svv.Viewport, svv.Map, svv.TmpSave, "Map Value View", "", svv.This,
			func(recv, send ki.Ki, sig int64, data interface{}) {
				svvv := recv.EmbeddedStruct(KiT_MapViewInline).(*MapViewInline)
				svpar := svvv.ParentByType(KiT_StructView, true).EmbeddedStruct(KiT_StructView).(*StructView)
				if svpar != nil {
					svpar.SetFullReRender() // todo: not working to re-generate item
					svpar.UpdateStart()
					svpar.UpdateEnd()
				}
			})
	})
}

func (sv *MapViewInline) UpdateFromMap() {
	sv.ConfigParts()
}

func (sv *MapViewInline) Style2D() {
	sv.ConfigParts()
	sv.WidgetBase.Style2D()
}

func (sv *MapViewInline) Render2D() {
	if sv.PushBounds() {
		sv.Render2DParts()
		sv.Render2DChildren()
		sv.PopBounds()
	}
}

// todo: see notes on treeview
func (sv *MapViewInline) ReRender2D() (node Node2D, layout bool) {
	node = sv.This.(Node2D)
	layout = true
	return
}

// check for interface implementation
var _ Node2D = &MapViewInline{}
