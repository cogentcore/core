// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"
	"strings"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

// MapView represents a map, creating a property editor of the values --
// constructs Children widgets to show the key / value pairs, within an
// overall frame.
// Automatically has a toolbar with Map Toolbar props if defined
// set prop toolbar = false to turn off
type MapView struct {
	gi.Frame

	// the map that we are a view onto
	Map any `set:"-"`

	// Value for the map itself, if this was created within value view framework -- otherwise nil
	MapValView Value

	// has the map been edited?
	Changed bool `set:"-"`

	// Value representations of the map keys
	Keys []Value `json:"-" xml:"-"`

	// Value representations of the map values
	Values []Value `json:"-" xml:"-"`

	// sort by values instead of keys
	SortVals bool

	// whether to show the toolbar or not
	ShowToolbar bool

	// the number of columns in the map; do not set externally; generally only access internally
	NCols int

	// WidgetConfiged tracks if the given Widget has been configured.
	// Widgets can only be configured once -- otherwise duplicate event
	// functions are registered.
	WidgetConfiged map[gi.Widget]bool `view:"-" json:"-" xml:"-"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `json:"-" xml:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// the map that we successfully set a toolbar for
	ToolbarMap any
}

func (mv *MapView) OnInit() {
	mv.MapViewStyles()
}

func (mv *MapView) MapViewStyles() {
	mv.WidgetConfiged = make(map[gi.Widget]bool)
	mv.ShowToolbar = true
	mv.Lay = gi.LayoutVert
	mv.Style(func(s *styles.Style) {
		mv.Spacing = gi.StdDialogVSpaceUnits
		s.SetStretchMax()
	})
	mv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(mv.This()) {
		case "map-grid":
			mg := w.(*gi.Frame)
			mg.Lay = gi.LayoutGrid
			mg.Stripes = gi.RowStripes
			w.Style(func(s *styles.Style) {
				// setting a pref here is key for giving it a scrollbar in larger context
				s.SetMinPrefHeight(units.Em(1.5))
				s.SetMinPrefWidth(units.Em(10))
				s.SetStretchMax()                  // for this to work, ALL layers above need it too
				s.Overflow = styles.OverflowScroll // this still gives it true size during PrefSize
				s.Columns = mv.NCols
			})
		}
		if w.Parent().Name() == "map-grid" {
			if strings.HasPrefix(w.Name(), "del-") {
				delbt := w.(*gi.Button)
				delbt.OnClick(func(e events.Event) {
					mv.MapDelete(delbt.Data.(Value).Val())
				})
				delbt.Style(func(s *styles.Style) {
					delbt.SetType(gi.ButtonAction)
					s.Color = colors.Scheme.Error.Base
				})
			}
		}
	})
}

// SetMap sets the source map that we are viewing.
// Rebuilds the children to represent this map
func (mv *MapView) SetMap(mp any) *MapView {
	// note: because we make new maps, and due to the strangeness of reflect, they
	// end up not being comparable types, so we can't check if equal
	mv.Map = mp
	mv.Update()
	return mv
}

// UpdateValues updates the widget display of slice values, assuming same slice config
func (mv *MapView) UpdateValues() {
	// maps have to re-read their values -- can't get pointers
	mv.Update()
}

// Config configures the view
func (mv *MapView) ConfigWidget(sc *gi.Scene) {
	mv.Sc = sc
	config := ki.Config{}
	config.Add(gi.ToolbarType, "toolbar")
	config.Add(gi.FrameType, "map-grid")
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

// Toolbar returns the toolbar widget
func (mv *MapView) Toolbar() *gi.Toolbar {
	return mv.ChildByName("toolbar", 0).(*gi.Toolbar)
}

// KiPropTag returns the PropTag value from Ki owner of this map, if it is..
func (mv *MapView) KiPropTag() string {
	if mv.MapValView == nil {
		return ""
	}
	vvb := mv.MapValView.AsValueBase()
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
	if laser.AnyIsNil(mv.Map) {
		return
	}
	sc := mv.Sc
	sg := mv.MapGrid()
	config := ki.Config{}
	// always start fresh!
	mv.Keys = make([]Value, 0)
	mv.Values = make([]Value, 0)

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
	valtypes := gti.AllEmbeddersOf(ki.NodeType) // todo: this is not right

	mv.NCols = ncol

	keys := laser.MapSort(mv.Map, !mv.SortVals, true) // note: this is a slice of reflect.Value!
	for _, key := range keys {
		kv := ToValue(key.Interface(), "")
		if kv == nil { // shouldn't happen
			continue
		}
		kv.SetMapKey(key, mv.Map, mv.TmpSave)

		val := laser.OnePtrUnderlyingValue(mpvnp.MapIndex(key))
		vv := ToValue(val.Interface(), "")
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
			config.Add(gi.ChooserType, typnm)
		}
		config.Add(gi.ButtonType, delnm)
		mv.Keys = append(mv.Keys, kv)
		mv.Values = append(mv.Values, vv)
	}
	mods, updt := sg.ConfigChildren(config)
	if mods {
		sg.SetNeedsLayoutUpdate(sc, updt)
	} else {
		updt = sg.UpdateStart() // cover rest of updates, which can happen even if same config
	}
	for i, vv := range mv.Values {
		kv := mv.Keys[i]
		vvb := vv.AsValueBase()
		kvb := kv.AsValueBase()
		vvb.OnChange(func(e events.Event) { mv.SendChange() })
		kvb.OnChange(func(e events.Event) {
			mv.SendChange()
			mv.Update()
		})
		keyw := sg.Child(i * ncol).(gi.Widget)
		widg := sg.Child(i*ncol + 1).(gi.Widget)
		if _, cfg := mv.WidgetConfiged[widg]; cfg { // already configured
			vvb.Widget = widg
			vv.UpdateWidget()
			kvb.Widget = keyw
			kv.UpdateWidget()
			continue
		}
		mv.WidgetConfiged[widg] = true
		kv.ConfigWidget(keyw, sc)
		vv.ConfigWidget(widg, sc)
		if ifaceType {
			typw := sg.Child(i*ncol + 2).(*gi.Chooser)
			typw.ItemsFromTypes(valtypes, false, true, 50)
			vtyp := laser.NonPtrType(reflect.TypeOf(vv.Val().Interface()))
			if vtyp == nil {
				vtyp = strtyp // default to string
			}
			typw.SetCurVal(vtyp)
			// typw.SetProp("mapview-index", i)
			// typw.OnChange(func(e events.Event) {
			// 	typ := typw.CurVal.(reflect.Type)
			// 	idx := typw.Prop("mapview-index").(int) // todo: does anything?
			// 	mv.SendChange()
			// })
		}
		delbt := sg.Child(i*ncol + ncol - 1).(*gi.Button)
		delbt.SetType(gi.ButtonAction)
		delbt.SetIcon(icons.Delete)
		delbt.Tooltip = "delete item"
		delbt.Data = kv
	}
	sg.UpdateEnd(updt)
}

// SetChanged sets the Changed flag and emits the ViewSig signal for the
// MapView, indicating that some kind of edit / change has taken place to
// the table data.
func (mv *MapView) SetChanged() {
	mv.Changed = true
	mv.SendChange()
	mv.Toolbar().UpdateButtons() // nil safe
}

// MapChangeValueType changes the type of the value for given map element at
// idx -- for maps with any values
func (mv *MapView) MapChangeValueType(idx int, typ reflect.Type) {
	if laser.AnyIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	defer mv.UpdateEndRender(updt)

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
	valv.AsValueBase().Value = evn.Elem()
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
	if laser.AnyIsNil(mv.Map) {
		return
	}
	laser.MapAdd(mv.Map)

	if mv.TmpSave != nil {
		mv.TmpSave.SaveTmp()
	}
	mv.SetChanged()
	mv.Update()
}

// MapDelete deletes a key-value from the map
func (mv *MapView) MapDelete(key reflect.Value) {
	if laser.AnyIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	// kvi := laser.NonPtrValue(key).Interface()
	laser.MapDeleteValue(mv.Map, laser.NonPtrValue(key))

	if mv.TmpSave != nil {
		mv.TmpSave.SaveTmp()
	}
	mv.SetChanged()
	mv.UpdateEnd(updt)
	mv.Update()
}

// ConfigToolbar configures the toolbar actions
func (mv *MapView) ConfigToolbar() {
	if laser.AnyIsNil(mv.Map) {
		return
	}
	if &mv.ToolbarMap == &mv.Map { // maps are not comparable
		return
	}
	if !mv.ShowToolbar {
		mv.ToolbarMap = mv.Map
		return
	}
	tb := mv.Toolbar()
	ndef := 2 // number of default actions
	if mv.IsReadOnly() {
		ndef = 1
	}
	if len(*tb.Children()) == 0 {
		gi.NewButton(tb, "sort").SetText("Sort").SetIcon(icons.Sort).SetTooltip("Switch between sorting by the keys vs. the values").
			OnClick(func(e events.Event) {
				mv.ToggleSort()
			})
		if ndef > 1 {
			gi.NewButton(tb, "add").SetText("Add").SetIcon(icons.Add).SetTooltip("add a new element to the map").
				OnClick(func(e events.Event) {
					mv.MapAdd()
				})
		}
	}
	sz := len(*tb.Children())
	if sz > ndef {
		for i := sz - 1; i >= ndef; i-- {
			tb.DeleteChildAtIndex(i, ki.DestroyKids)
		}
	}
	gi.ToolbarFor(mv.Map, tb)
	mv.ToolbarMap = mv.Map
}

func (mv *MapView) Render(sc *gi.Scene) {
	if mv.IsConfiged() {
		mv.Toolbar().UpdateButtons() // nil safe..
	}
	mv.Frame.Render(sc)
}
