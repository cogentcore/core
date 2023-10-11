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

// MapViewInline represents a map as a single line widget, for smaller maps
// and those explicitly marked inline -- constructs widgets in Parts to show
// the key names and editor vals for each value.
type MapViewInline struct {
	gi.WidgetBase

	// the map that we are a view onto
	Map any `desc:"the map that we are a view onto"`

	// Value for the map itself, if this was created within value view framework -- otherwise nil
	MapValView Value `desc:"Value for the map itself, if this was created within value view framework -- otherwise nil"`

	// has the map been edited?
	Changed bool `desc:"has the map been edited?"`

	// Value representations of the map keys
	Keys []Value `json:"-" xml:"-" desc:"Value representations of the map keys"`

	// Value representations of the fields
	Values []Value `json:"-" xml:"-" desc:"Value representations of the fields"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string `desc:"a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows"`
}

func (mv *MapViewInline) OnInit() {
	mv.AddStyles(func(s *styles.Style) {
		s.MinWidth.SetEx(60)
	})
}

func (mv *MapViewInline) OnChildAdded(child ki.Ki) {
	w, _ := gi.AsWidget(child)
	switch w.Name() {
	case "parts":
		parts := w.(*gi.Layout)
		parts.Lay = gi.LayoutHoriz
		w.AddStyles(func(s *styles.Style) {
			s.Overflow = styles.OverflowHidden // no scrollbars!
		})
	}
}

// SetMap sets the source map that we are viewing -- rebuilds the children to represent this map
func (mv *MapViewInline) SetMap(mp any) {
	// note: because we make new maps, and due to the strangeness of reflect, they
	// end up not being comparable types, so we can't check if equal
	mv.Map = mp
	mv.UpdateFromMap()
}

// ConfigParts configures Parts for the current map
func (mv *MapViewInline) ConfigParts(sc *gi.Scene) {
	if laser.AnyIsNil(mv.Map) {
		return
	}
	parts := mv.NewParts(gi.LayoutHoriz)
	config := ki.Config{}
	// always start fresh!
	mv.Keys = make([]Value, 0)
	mv.Values = make([]Value, 0)

	mpv := reflect.ValueOf(mv.Map)
	mpvnp := laser.NonPtrValue(mpv)

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
	mods, updt := parts.ConfigChildren(config)
	if !mods {
		updt = parts.UpdateStart()
	}
	for i, vv := range mv.Values {
		vvb := vv.AsValueBase()
		vvb.OnChange(func(e events.Event) { mv.SendChange() })
		keyw := parts.Child(i * 2).(gi.Widget)
		widg := parts.Child((i * 2) + 1).(gi.Widget)
		kv := mv.Keys[i]
		kv.ConfigWidget(keyw)
		vv.ConfigWidget(widg)
		if mv.IsDisabled() {
			widg.AsWidget().SetState(true, states.Disabled)
			keyw.AsWidget().SetState(true, states.Disabled)
		}
	}
	adack, err := parts.Children().ElemFromEndTry(1)
	if err == nil {
		adac := adack.(*gi.Button)
		adac.SetIcon(icons.Add)
		adac.Tooltip = "add an entry to the map"
		adac.OnClick(func(e events.Event) {
			mv.MapAdd()
		})

	}
	edack, err := parts.Children().ElemFromEndTry(0)
	if err == nil {
		edac := edack.(*gi.Button)
		edac.SetIcon(icons.Edit)
		edac.Tooltip = "map edit dialog"
		edac.OnClick(func(e events.Event) {
			vpath := mv.ViewPath
			title := ""
			if mv.MapValView != nil {
				newPath := ""
				isZero := false
				title, newPath, isZero = mv.MapValView.AsValueBase().Label()
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
			MapViewDialog(mv, DlgOpts{Title: title, Prompt: mv.Tooltip, TmpSave: mv.TmpSave, ViewPath: vpath}, mv.Map, func(dlg *gi.Dialog) {
				mv.SendChange()
			})
		})
	}
	parts.UpdateEndLayout(updt)
	mv.SetNeedsLayout(sc, updt)
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
	mv.UpdateFromMap()
}

func (mv *MapViewInline) UpdateFromMap() {
	mv.ConfigParts(mv.Sc)
}

func (mv *MapViewInline) UpdateValues() {
	// maps have to re-read their values because they can't get pointers!
	mv.ConfigParts(mv.Sc)
}

func (mv *MapViewInline) Render(sc *gi.Scene) {
	if mv.PushBounds(sc) {
		mv.RenderParts(sc)
		mv.RenderChildren(sc)
		mv.PopBounds(sc)
	}
}
