// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"goki.dev/gi/v2/gi"
	"goki.dev/gicons"
	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
)

// MapView represents a map, creating a property editor of the values --
// constructs Children widgets to show the key / value pairs, within an
// overall frame.
// Automatically has a toolbar with Map ToolBar props if defined
// set prop toolbar = false to turn off
type MapView struct {
	gi.Frame

	// the map that we are a view onto
	Map any `desc:"the map that we are a view onto"`

	// ValueView for the map itself, if this was created within value view framework -- otherwise nil
	MapValView ValueView `desc:"ValueView for the map itself, if this was created within value view framework -- otherwise nil"`

	// has the map been edited?
	Changed bool `desc:"has the map been edited?"`

	// ValueView representations of the map keys
	Keys []ValueView `json:"-" xml:"-" desc:"ValueView representations of the map keys"`

	// ValueView representations of the map values
	Values []ValueView `json:"-" xml:"-" desc:"ValueView representations of the map values"`

	// sort by values instead of keys
	SortVals bool `desc:"sort by values instead of keys"`

	// whether to show the toolbar or not
	ShowToolBar bool `desc:"whether to show the toolbar or not"`

	// the number of columns in the map; do not set externally; generally only access internally
	NCols int `desc:"the number of columns in the map; do not set externally; generally only access internally"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave ValueView `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`

	// signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update
	ViewSig ki.Signal `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`

	// map view specific signals: add, delete, double-click
	MapViewSig ki.Signal `copy:"-" json:"-" xml:"-" desc:"map view specific signals: add, delete, double-click"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string `desc:"a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows"`

	// the map that we successfully set a toolbar for
	ToolbarMap any `desc:"the map that we successfully set a toolbar for"`
}

func (mv *MapView) OnInit() {
	mv.ShowToolBar = true
	mv.Lay = gi.LayoutVert
	mv.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		mv.Spacing = gi.StdDialogVSpaceUnits
		s.SetStretchMax()
	})
}

func (mv *MapView) OnChildAdded(child ki.Ki) {
	if w := gi.KiAsWidget(child); w != nil {
		switch w.Name() {
		case "map-grid":
			mg := child.(*gi.Frame)
			mg.Lay = gi.LayoutGrid
			mg.Stripes = gi.RowStripes
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				// setting a pref here is key for giving it a scrollbar in larger context
				s.SetMinPrefHeight(units.Em(1.5))
				s.SetMinPrefWidth(units.Em(10))
				s.SetStretchMax()                // for this to work, ALL layers above need it too
				s.Overflow = gist.OverflowScroll // this still gives it true size during PrefSize
				s.Columns = mv.NCols
			})
		}
	}
}

func (mv *MapView) Disconnect() {
	mv.Frame.Disconnect()
	mv.MapViewSig.DisconnectAll()
	mv.ViewSig.DisconnectAll()
}

// SetMap sets the source map that we are viewing -- rebuilds the children to
// represent this map
func (mv *MapView) SetMap(mp any) {
	// note: because we make new maps, and due to the strangeness of reflect, they
	// end up not being comparable types, so we can't check if equal
	mv.Map = mp
	mv.Config()
}

var MapViewProps = ki.Props{
	ki.EnumTypeFlag: gi.TypeNodeFlags,
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

// UpdateValues updates the widget display of slice values, assuming same slice config
func (mv *MapView) UpdateValues() {
	// maps have to re-read their values -- can't get pointers
	mv.ConfigMapGrid()
}

// Config configures the view
func (mv *MapView) Config() {
	config := ki.TypeAndNameList{}
	config.Add(gi.TypeToolBar, "toolbar")
	config.Add(gi.TypeFrame, "map-grid")
	mods, updt := mv.ConfigChildren(config)
	mv.ConfigMapGrid()
	mv.ConfigToolbar()
	if mods {
		mv.UpdateEnd(updt)
	}
}

// IsConfiged returns true if the widget is fully configured
func (mv *MapView) IsConfiged() bool {
	return len(mv.Kids) != 0
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
	if laser.IfaceIsNil(mv.Map) {
		return
	}
	sg := mv.MapGrid()
	config := ki.TypeAndNameList{}
	// always start fresh!
	mv.Keys = make([]ValueView, 0)
	mv.Values = make([]ValueView, 0)

	mpv := reflect.ValueOf(mv.Map)
	mpvnp := laser.NonPtrValue(mpv)

	valtyp := laser.NonPtrType(reflect.TypeOf(mv.Map)).Elem()
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

	// valtypes := append(kit.Types.AllTagged(typeTag), kit.Enums.AllTagged(typeTag)...)
	// valtypes = append(valtypes, kit.Types.AllTagged("basic-type")...)
	// valtypes = append(valtypes, kit.TypeFor[reflect.Type]())

	mv.NCols = ncol

	keys := laser.MapSort(mv.Map, !mv.SortVals, true) // note: this is a slice of reflect.Value!
	for _, key := range keys {
		kv := ToValueView(key.Interface(), "")
		if kv == nil { // shouldn't happen
			continue
		}
		kv.SetMapKey(key, mv.Map, mv.TmpSave)

		val := laser.OnePtrUnderlyingValue(mpvnp.MapIndex(key))
		vv := ToValueView(val.Interface(), "")
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetMapValue(val, mv.Map, key.Interface(), kv, mv.TmpSave, mv.ViewPath) // needs key value view to track updates

		keytxt := laser.ToString(key.Interface())
		keynm := "key-" + keytxt
		valnm := "value-" + keytxt
		delnm := "del-" + keytxt

		config.Add(kv.WidgetType(), keynm)
		config.Add(vv.WidgetType(), valnm)
		if ifaceType {
			typnm := "type-" + keytxt
			config.Add(gi.TypeComboBox, typnm)
		}
		config.Add(gi.TypeAction, delnm)
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
		vvb.ViewSig.ConnectOnly(mv.This(), func(recv, send ki.Ki, sig int64, data any) {
			mvv, _ := recv.Embed(TypeMapView).(*MapView)
			mvv.SetChanged()
		})
		keyw := sg.Child(i * ncol).(gi.Node2D)
		widg := sg.Child(i*ncol + 1).(gi.Node2D)
		kv := mv.Keys[i]
		kvb := kv.AsValueViewBase()
		kvb.ViewSig.ConnectOnly(mv.This(), func(recv, send ki.Ki, sig int64, data any) {
			mvv, _ := recv.Embed(TypeMapView).(*MapView)
			mvv.SetChanged()
		})
		kv.ConfigWidget(keyw)
		vv.ConfigWidget(widg)
		wb := widg.AsWidget()
		if wb != nil {
			wb.Style.Template = "giv.MapView.ItemWidget." + vv.WidgetType().Name()
		}
		wb = keyw.AsWidget()
		if wb != nil {
			wb.Style.Template = "giv.MapView.KeyWidget." + kv.WidgetType().Name()
		}
		if ifaceType {
			typw := sg.Child(i*ncol + 2).(*gi.ComboBox)
			typw.ItemsFromTypes(valtypes, false, true, 50)
			vtyp := laser.NonPtrType(reflect.TypeOf(vv.Val().Interface()))
			if vtyp == nil {
				vtyp = strtyp // default to string
			}
			typw.SetCurVal(vtyp)
			typw.SetProp("mapview-index", i)
			typw.ComboSig.ConnectOnly(mv.This(), func(recv, send ki.Ki, sig int64, data any) {
				cb := send.(*gi.ComboBox)
				typ := cb.CurVal.(reflect.Type)
				idx := cb.Prop("mapview-index").(int)
				mvv := recv.Embed(TypeMapView).(*MapView)
				mvv.MapChangeValueType(idx, typ)
			})
		}
		delact := sg.Child(i*ncol + ncol - 1).(*gi.Action)
		delact.SetIcon(gicons.Delete)
		delact.Tooltip = "delete item"
		delact.Data = kv
		delact.Style.Template = "giv.MapView.DelAction"
		delact.ActionSig.ConnectOnly(mv.This(), func(recv, send ki.Ki, sig int64, data any) {
			act := send.(*gi.Action)
			mvv := recv.Embed(TypeMapView).(*MapView)
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
// idx -- for maps with any values
func (mv *MapView) MapChangeValueType(idx int, typ reflect.Type) {
	if laser.IfaceIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	defer mv.UpdateEnd(updt)

	keyv := mv.Keys[idx]
	ck := laser.NonPtrValue(keyv.Val()) // current key value
	valv := mv.Values[idx]
	cv := laser.NonPtrValue(valv.Val()) // current val value

	// create a new item of selected type, and attempt to convert existing to it
	var evn reflect.Value
	if laser.ValueIsZero(cv) {
		evn = laser.MakeOfType(typ)
	} else {
		evn = laser.CloneToType(typ, cv.Interface())
	}
	ov := laser.NonPtrValue(reflect.ValueOf(mv.Map))
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
	if laser.IfaceIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	defer mv.UpdateEnd(updt)

	laser.MapAdd(mv.Map)

	if mv.TmpSave != nil {
		mv.TmpSave.SaveTmp()
	}
	mv.ConfigMapGrid()
	mv.SetChanged()
	mv.MapViewSig.Emit(mv.This(), int64(MapViewAdded), nil)
}

// MapDelete deletes a key-value from the map
func (mv *MapView) MapDelete(key reflect.Value) {
	if laser.IfaceIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	defer mv.UpdateEnd(updt)

	kvi := laser.NonPtrValue(key).Interface()

	laser.MapDeleteValue(mv.Map, laser.NonPtrValue(key))

	if mv.TmpSave != nil {
		mv.TmpSave.SaveTmp()
	}
	mv.ConfigMapGrid()
	mv.SetChanged()
	mv.MapViewSig.Emit(mv.This(), int64(MapViewDeleted), kvi)
}

// ConfigToolbar configures the toolbar actions
func (mv *MapView) ConfigToolbar() {
	if laser.IfaceIsNil(mv.Map) {
		return
	}
	if &mv.ToolbarMap == &mv.Map { // maps are not comparable
		return
	}
	if !mv.ShowToolBar {
		mv.ToolbarMap = mv.Map
		return
	}
	tb := mv.ToolBar()
	ndef := 3 // number of default actions
	if mv.IsDisabled() {
		ndef = 2
	}
	if len(*tb.Children()) == 0 {
		tb.SetStretchMaxWidth()
		tb.AddAction(gi.ActOpts{Label: "UpdtView", Icon: gicons.Refresh, Tooltip: "update the view to reflect current state of map"},
			mv.This(), func(recv, send ki.Ki, sig int64, data any) {
				mvv := recv.Embed(TypeMapView).(*MapView)
				mvv.UpdateValues()
			})
		tb.AddAction(gi.ActOpts{Label: "Sort", Icon: gicons.Sort, Tooltip: "Switch between sorting by the keys vs. the values"},
			mv.This(), func(recv, send ki.Ki, sig int64, data any) {
				mvv := recv.Embed(TypeMapView).(*MapView)
				mvv.ToggleSort()
			})
		if ndef > 2 {
			tb.AddAction(gi.ActOpts{Label: "Add", Icon: gicons.Add, Tooltip: "add a new element to the map"},
				mv.This(), func(recv, send ki.Ki, sig int64, data any) {
					mvv := recv.Embed(TypeMapView).(*MapView)
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

func (mv *MapView) SetStyle() {
	mvp := mv.ViewportSafe()
	if mvp != nil && mvp.IsDoingFullRender() {
		mv.Config()
	}
	mv.Frame.SetStyle()
}

func (mv *MapView) Render(vp *Viewport) {
	if mv.IsConfiged() {
		mv.ToolBar().UpdateActions() // nil safe..
	}
	if win := mv.ParentWindow(); win != nil {
		if !win.IsResizing() {
			win.MainMenuUpdateActives()
		}
	}
	mv.Frame.Render()
}
