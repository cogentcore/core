// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

// MapViewInline represents a map as a single line widget,
// for smaller maps and those explicitly marked inline.
type MapViewInline struct {
	gi.Layout

	// the map that we are a view onto
	Map any `set:"-"`

	// Value for the map itself, if this was created within value view framework -- otherwise nil
	MapValView Value

	// has the map been edited?
	Changed bool `set:"-"`

	// Value representations of the map keys
	Keys []Value `json:"-" xml:"-"`

	// Value representations of the fields
	Values []Value `json:"-" xml:"-"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `view:"-" json:"-" xml:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string
}

func (mv *MapViewInline) OnInit() {
	mv.MapViewInlineStyles()
}

func (mv *MapViewInline) MapViewInlineStyles() {
	mv.Lay = gi.LayoutHoriz
	mv.Style(func(s *styles.Style) {
		s.MinWidth.Ex(60)
		s.Overflow = styles.OverflowHidden // no scrollbars!
	})
	mv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(mv) {
		case "add-action":
			ab := w.(*gi.Button)
			w.Style(func(s *styles.Style) {
				ab.SetType(gi.ButtonTonal)
			})
			w.OnClick(func(e events.Event) {
				mv.MapAdd()
			})
		case "edit-action":
			w.Style(func(s *styles.Style) {
				w.(*gi.Button).SetType(gi.ButtonTonal)
			})
			w.OnClick(func(e events.Event) {
				vpath := mv.ViewPath
				title := ""
				if mv.MapValView != nil {
					newPath := ""
					isZero := false
					title, newPath, isZero = mv.MapValView.AsValueBase().GetTitle()
					if isZero {
						return
					}
					vpath = mv.ViewPath + "/" + newPath
				} else {
					tmptyp := laser.NonPtrType(reflect.TypeOf(mv.Map))
					title = "Map of " + tmptyp.String()
					// if tynm == "" {
					// 	tynm = tmptyp.String()
					// }
				}
				d := gi.NewDialog(mv).Title(title).Prompt(mv.Tooltip)
				NewMapView(d).SetViewPath(vpath).SetMap(mv.Map).SetTmpSave(mv.TmpSave)
				d.OnAccept(func(e events.Event) {
					mv.SendChange()
				}).Run()
			})
		}
	})
}

// SetMap sets the source map that we are viewing -- rebuilds the children to represent this map
func (mv *MapViewInline) SetMap(mp any) *MapViewInline {
	// note: because we make new maps, and due to the strangeness of reflect, they
	// end up not being comparable types, so we can't check if equal
	mv.Map = mp
	mv.Update()
	return mv
}

func (mv *MapViewInline) ConfigWidget(sc *gi.Scene) {
	mv.ConfigMap(sc)
}

// ConfigMap configures children for map view
func (mv *MapViewInline) ConfigMap(sc *gi.Scene) bool {
	if laser.AnyIsNil(mv.Map) {
		return false
	}
	config := ki.Config{}
	// always start fresh!
	mv.Keys = make([]Value, 0)
	mv.Values = make([]Value, 0)

	mpv := reflect.ValueOf(mv.Map)
	mpvnp := laser.NonPtrValue(laser.OnePtrUnderlyingValue(mpv))

	keys := mpvnp.MapKeys() // this is a slice of reflect.Value
	laser.ValueSliceSort(keys, true)
	for i, key := range keys {
		if i >= MapInlineLen {
			break
		}
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

		config.Add(kv.WidgetType(), keynm)
		config.Add(vv.WidgetType(), valnm)
		mv.Keys = append(mv.Keys, kv)
		mv.Values = append(mv.Values, vv)
	}
	config.Add(gi.ButtonType, "add-action")
	config.Add(gi.ButtonType, "edit-action")
	mods, updt := mv.ConfigChildren(config)
	if !mods {
		updt = mv.UpdateStart()
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
		keyw := mv.Child(i * 2).(gi.Widget)
		w := mv.Child((i * 2) + 1).(gi.Widget)
		kv.ConfigWidget(keyw, sc)
		vv.ConfigWidget(w, sc)
		if mv.IsReadOnly() {
			w.AsWidget().SetState(true, states.ReadOnly)
			keyw.AsWidget().SetState(true, states.ReadOnly)
		}
	}
	adack, err := mv.Children().ElemFromEndTry(1)
	if err == nil {
		adbt := adack.(*gi.Button)
		adbt.SetType(gi.ButtonTonal)
		adbt.SetIcon(icons.Add)
		adbt.Tooltip = "add an entry to the map"

	}
	edack, err := mv.Children().ElemFromEndTry(0)
	if err == nil {
		edbt := edack.(*gi.Button)
		edbt.SetType(gi.ButtonTonal)
		edbt.SetIcon(icons.Edit)
		edbt.Tooltip = "map edit dialog"
	}
	mv.UpdateEnd(updt)
	return updt
}

// SetChanged sets the Changed flag and emits the ViewSig signal for the
// SliceView, indicating that some kind of edit / change has taken place to
// the table data.  It isn't really practical to record all the different
// types of changes, so this is just generic.
func (mv *MapViewInline) SetChanged() {
	mv.Changed = true
	mv.SendChange()
}

// MapAdd adds a new entry to the map
func (mv *MapViewInline) MapAdd() {
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

func (mv *MapViewInline) UpdateValues() {
	// maps have to re-read their values because they can't get pointers!
	mv.Update()
}
