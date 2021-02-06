// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// MapView represents a map, creating a property editor of the values --
// constructs Children widgets to show the key / value pairs, within an
// overall frame.
// Automatically has a toolbar with Map ToolBar props if defined
// set prop toolbar = false to turn off
type MapView struct {
	gi.Frame
	Map        interface{} `desc:"the map that we are a view onto"`
	MapValView ValueView   `desc:"ValueView for the map itself, if this was created within value view framework -- otherwise nil"`
	Changed    bool        `desc:"has the map been edited?"`
	Keys       []ValueView `json:"-" xml:"-" desc:"ValueView representations of the map keys"`
	Values     []ValueView `json:"-" xml:"-" desc:"ValueView representations of the map values"`
	SortVals   bool        `desc:"sort by values instead of keys"`
	TmpSave    ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig    ki.Signal   `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
	MapViewSig ki.Signal   `copy:"-" json:"-" xml:"-" desc:"map view specific signals: add, delete, double-click"`
	ViewPath   string      `desc:"a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows"`
	ToolbarMap interface{} `desc:"the map that we successfully set a toolbar for"`
}

var KiT_MapView = kit.Types.AddType(&MapView{}, MapViewProps)

// AddNewMapView adds a new mapview to given parent node, with given name.
func AddNewMapView(parent ki.Ki, name string) *MapView {
	return parent.AddNewChild(KiT_MapView, name).(*MapView)
}

func (mv *MapView) Disconnect() {
	mv.Frame.Disconnect()
	mv.MapViewSig.DisconnectAll()
	mv.ViewSig.DisconnectAll()
}

// SetMap sets the source map that we are viewing -- rebuilds the children to
// represent this map
func (mv *MapView) SetMap(mp interface{}) {
	// note: because we make new maps, and due to the strangeness of reflect, they
	// end up not being comparable types, so we can't check if equal
	mv.Map = mp
	mv.Config()
}

var MapViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"max-width":        -1,
	"max-height":       -1,
}

// MapViewSignals are signals that mapview can send, mostly for editing
// mode.  Selection events are sent on WidgetSig WidgetSelected signals in
// both modes.
type MapViewSignals int

const (
	// MapViewDoubleClicked emitted during inactive mode when item
	// double-clicked -- can be used for accepting dialog.
	MapViewDoubleClicked MapViewSignals = iota

	// MapViewAdded emitted when a new blank item is added -- no data is sent.
	MapViewAdded

	// MapViewDeleted emitted when an item is deleted -- data is key of item deleted
	MapViewDeleted

	MapViewSignalsN
)

//go:generate stringer -type=MapViewSignals

// UpdateValues updates the widget display of slice values, assuming same slice config
func (mv *MapView) UpdateValues() {
	// maps have to re-read their values -- can't get pointers
	mv.ConfigMapGrid()
}

// Config configures the view
func (mv *MapView) Config() {
	mv.Lay = gi.LayoutVert
	mv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_Frame, "map-grid")
	mods, updt := mv.ConfigChildren(config)
	mv.ConfigMapGrid()
	mv.ConfigToolbar()
	if mods {
		mv.UpdateEnd(updt)
	}
}

// IsConfiged returns true if the widget is fully configured
func (mv *MapView) IsConfiged() bool {
	if len(mv.Kids) == 0 {
		return false
	}
	return true
}

// MapGrid returns the MapGrid grid layout widget, which contains all the fields and values
func (mv *MapView) MapGrid() *gi.Frame {
	return mv.ChildByName("map-grid", 0).(*gi.Frame)
}

// ToolBar returns the toolbar widget
func (mv *MapView) ToolBar() *gi.ToolBar {
	return mv.ChildByName("toolbar", 0).(*gi.ToolBar)
}

// KiPropTag returns the PropTag value from Ki owner of this map, if it is..
func (mv *MapView) KiPropTag() string {
	if mv.MapValView == nil {
		return ""
	}
	vvb := mv.MapValView.AsValueViewBase()
	if vvb.Owner == nil {
		return ""
	}
	if ownki, ok := vvb.Owner.(ki.Ki); ok {
		pt := ownki.PropTag()
		// fmt.Printf("got prop tag: %v\n", pt)
		return pt
	}
	return ""
}

// ConfigMapGrid configures the MapGrid for the current map
func (mv *MapView) ConfigMapGrid() {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	sg := mv.MapGrid()
	sg.Lay = gi.LayoutGrid
	sg.Stripes = gi.RowStripes
	// setting a pref here is key for giving it a scrollbar in larger context
	sg.SetMinPrefHeight(units.NewEm(1.5))
	sg.SetMinPrefWidth(units.NewEm(10))
	sg.SetStretchMax()                          // for this to work, ALL layers above need it too
	sg.SetProp("overflow", gist.OverflowScroll) // this still gives it true size during PrefSize
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
		typeTag = mv.KiPropTag()
		// todo: need some way of setting & getting
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

	keys := kit.MapSort(mv.Map, !mv.SortVals, true) // note: this is a slice of reflect.Value!
	for _, key := range keys {
		kv := ToValueView(key.Interface(), "")
		if kv == nil { // shouldn't happen
			continue
		}
		kv.SetMapKey(key, mv.Map, mv.TmpSave)

		val := kit.OnePtrUnderlyingValue(mpvnp.MapIndex(key))
		vv := ToValueView(val.Interface(), "")
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetMapValue(val, mv.Map, key.Interface(), kv, mv.TmpSave, mv.ViewPath) // needs key value view to track updates

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
	mods, updt := sg.ConfigChildren(config)
	if mods {
		sg.SetFullReRender()
	} else {
		updt = sg.UpdateStart() // cover rest of updates, which can happen even if same config
	}
	for i, vv := range mv.Values {
		vvb := vv.AsValueViewBase()
		vvb.ViewSig.ConnectOnly(mv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			mvv, _ := recv.Embed(KiT_MapView).(*MapView)
			mvv.SetChanged()
		})
		keyw := sg.Child(i * ncol).(gi.Node2D)
		widg := sg.Child(i*ncol + 1).(gi.Node2D)
		kv := mv.Keys[i]
		kvb := kv.AsValueViewBase()
		kvb.ViewSig.ConnectOnly(mv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			mvv, _ := recv.Embed(KiT_MapView).(*MapView)
			mvv.SetChanged()
		})
		kv.ConfigWidget(keyw)
		vv.ConfigWidget(widg)
		wb := widg.AsWidget()
		if wb != nil {
			wb.Sty.Template = "giv.MapView.ItemWidget." + vv.WidgetType().Name()
		}
		wb = keyw.AsWidget()
		if wb != nil {
			wb.Sty.Template = "giv.MapView.KeyWidget." + kv.WidgetType().Name()
		}
		if ifaceType {
			typw := sg.Child(i*ncol + 2).(*gi.ComboBox)
			typw.ItemsFromTypes(valtypes, false, true, 50)
			vtyp := kit.NonPtrType(reflect.TypeOf(vv.Val().Interface()))
			if vtyp == nil {
				vtyp = strtyp // default to string
			}
			typw.SetCurVal(vtyp)
			typw.SetProp("mapview-index", i)
			typw.ComboSig.ConnectOnly(mv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				cb := send.(*gi.ComboBox)
				typ := cb.CurVal.(reflect.Type)
				idx := cb.Prop("mapview-index").(int)
				mvv := recv.Embed(KiT_MapView).(*MapView)
				mvv.MapChangeValueType(idx, typ)
			})
		}
		delact := sg.Child(i*ncol + ncol - 1).(*gi.Action)
		delact.SetIcon("minus")
		delact.Tooltip = "delete item"
		delact.Data = kv
		delact.Sty.Template = "giv.MapView.DelAction"
		delact.ActionSig.ConnectOnly(mv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			act := send.(*gi.Action)
			mvv := recv.Embed(KiT_MapView).(*MapView)
			mvv.MapDelete(act.Data.(ValueView).Val())
		})
	}
	sg.UpdateEnd(updt)
}

// SetChanged sets the Changed flag and emits the ViewSig signal for the
// MapView, indicating that some kind of edit / change has taken place to
// the table data.  It isn't really practical to record all the different
// types of changes, so this is just generic.
func (mv *MapView) SetChanged() {
	mv.Changed = true
	mv.ViewSig.Emit(mv.This(), 0, nil)
	mv.ToolBar().UpdateActions() // nil safe
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
	ck := kit.NonPtrValue(keyv.Val()) // current key value
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
	mv.SetChanged()
}

// ToggleSort toggles sorting by values vs. keys
func (mv *MapView) ToggleSort() {
	mv.SortVals = !mv.SortVals
	mv.ConfigMapGrid()
}

// MapAdd adds a new entry to the map
func (mv *MapView) MapAdd() {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	defer mv.UpdateEnd(updt)

	kit.MapAdd(mv.Map)

	if mv.TmpSave != nil {
		mv.TmpSave.SaveTmp()
	}
	mv.ConfigMapGrid()
	mv.SetChanged()
	mv.MapViewSig.Emit(mv.This(), int64(MapViewAdded), nil)
}

// MapDelete deletes a key-value from the map
func (mv *MapView) MapDelete(key reflect.Value) {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	defer mv.UpdateEnd(updt)

	kvi := kit.NonPtrValue(key).Interface()

	kit.MapDeleteValue(mv.Map, kit.NonPtrValue(key))

	if mv.TmpSave != nil {
		mv.TmpSave.SaveTmp()
	}
	mv.ConfigMapGrid()
	mv.SetChanged()
	mv.MapViewSig.Emit(mv.This(), int64(MapViewDeleted), kvi)
}

// ConfigToolbar configures the toolbar actions
func (mv *MapView) ConfigToolbar() {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	if &mv.ToolbarMap == &mv.Map { // maps are not comparable
		return
	}
	if pv, ok := mv.PropInherit("toolbar", ki.NoInherit, ki.TypeProps); ok {
		pvb, _ := kit.ToBool(pv)
		if !pvb {
			mv.ToolbarMap = mv.Map
			return
		}
	}
	tb := mv.ToolBar()
	ndef := 3 // number of default actions
	if mv.IsInactive() {
		ndef = 2
	}
	if len(*tb.Children()) == 0 {
		tb.SetStretchMaxWidth()
		tb.AddAction(gi.ActOpts{Label: "UpdtView", Icon: "update", Tooltip: "update the view to reflect current state of map"},
			mv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				mvv := recv.Embed(KiT_MapView).(*MapView)
				mvv.UpdateValues()
			})
		tb.AddAction(gi.ActOpts{Label: "Sort", Icon: "update", Tooltip: "Switch between sorting by the keys vs. the values"},
			mv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				mvv := recv.Embed(KiT_MapView).(*MapView)
				mvv.ToggleSort()
			})
		if ndef > 2 {
			tb.AddAction(gi.ActOpts{Label: "Add", Icon: "plus", Tooltip: "add a new element to the map"},
				mv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
					mvv := recv.Embed(KiT_MapView).(*MapView)
					mvv.MapAdd()
				})
		}
	}
	sz := len(*tb.Children())
	if sz > ndef {
		for i := sz - 1; i >= ndef; i-- {
			tb.DeleteChildAtIndex(i, ki.DestroyKids)
		}
	}
	if HasToolBarView(mv.Map) {
		ToolBarView(mv.Map, mv.Viewport, tb)
		tb.SetFullReRender()
	}
	mv.ToolbarMap = mv.Map
}

func (mv *MapView) Style2D() {
	mvp := mv.ViewportSafe()
	if mvp != nil && mvp.IsDoingFullRender() {
		mv.Config()
	}
	mv.Frame.Style2D()
}

func (mv *MapView) Render2D() {
	if mv.IsConfiged() {
		mv.ToolBar().UpdateActions() // nil safe..
	}
	if win := mv.ParentWindow(); win != nil {
		if !win.IsResizing() {
			win.MainMenuUpdateActives()
		}
	}
	mv.Frame.Render2D()
}
