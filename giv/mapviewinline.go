// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
)

// MapViewInline represents a map as a single line widget,
// for smaller maps and those explicitly marked inline.
type MapViewInline struct {
	gi.Layout

	// the map that we are a view onto
	Map any `set:"-"`

	// MapValue is the Value for the map itself
	// if this was created within the Value framework.
	// Otherwise, it is nil.
	MapValue Value `set:"-"`

	// has the map been edited?
	Changed bool `set:"-"`

	// Value representations of the map keys
	Keys []Value `json:"-" xml:"-" set:"-"`

	// Value representations of the fields
	Values []Value `json:"-" xml:"-" set:"-"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `view:"-" json:"-" xml:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// size of map when gui configed
	ConfigSize int `set:"-"`
}

func (mv *MapViewInline) OnInit() {
	mv.Layout.OnInit()
	mv.SetStyles()
}

func (mv *MapViewInline) SetStyles() {
	mv.Style(func(s *styles.Style) {
		s.Grow.Set(0, 0)
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
				if mv.MapValue != nil {
					newPath := ""
					isZero := false
					title, newPath, isZero = mv.MapValue.AsValueData().GetTitle()
					if isZero {
						return
					}
					vpath = JoinViewPath(mv.ViewPath, newPath)
				} else {
					tmptyp := laser.NonPtrType(reflect.TypeOf(mv.Map))
					title = "Map of " + tmptyp.String()
				}
				d := gi.NewBody().AddTitle(title).AddText(mv.Tooltip)
				NewMapView(d).SetViewPath(vpath).SetMap(mv.Map).SetTmpSave(mv.TmpSave)
				d.AddBottomBar(func(pw gi.Widget) {
					d.AddCancel(pw)
					d.AddOk(pw).OnClick(func(e events.Event) {
						mv.SendChange()
					})
				})
				d.NewFullDialog(mv).Run()
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

func (mv *MapViewInline) Config() {
	mv.DeleteChildren()
	if laser.AnyIsNil(mv.Map) {
		mv.ConfigSize = 0
		return
	}
	config := ki.Config{}
	mv.Keys = make([]Value, 0)
	mv.Values = make([]Value, 0)

	mpv := reflect.ValueOf(mv.Map)
	mpvnp := laser.NonPtrValue(laser.OnePtrUnderlyingValue(mpv))
	keys := mpvnp.MapKeys() // this is a slice of reflect.Value
	mv.ConfigSize = len(keys)

	laser.ValueSliceSort(keys, true)
	for i, key := range keys {
		if i >= gi.SystemSettings.MapInlineLength {
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
	mv.ConfigChildren(config)
	for i, vv := range mv.Values {
		kv := mv.Keys[i]
		vv.OnChange(func(e events.Event) { mv.SendChange() })
		kv.OnChange(func(e events.Event) {
			mv.SendChange()
			mv.Update()
		})
		w, wb := gi.AsWidget(mv.Child((i * 2) + 1))
		kw, kwb := gi.AsWidget(mv.Child(i * 2))
		Config(vv, w)
		Config(kv, kw)
		vv.AsWidgetBase().OnInput(mv.HandleEvent)
		kv.AsWidgetBase().OnInput(mv.HandleEvent)
		w.Style(func(s *styles.Style) {
			s.SetTextWrap(false)
		})
		kw.Style(func(s *styles.Style) {
			s.SetTextWrap(false)
		})
		if mv.IsReadOnly() {
			wb.SetReadOnly(true)
			kwb.SetReadOnly(true)
		} else {
			wb.AddContextMenu(func(m *gi.Scene) {
				mv.ContextMenu(m, kv.Val())
			})
			kwb.AddContextMenu(func(m *gi.Scene) {
				mv.ContextMenu(m, kv.Val())
			})
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

// MapDelete deletes a key-value from the map
func (mv *MapViewInline) MapDelete(key reflect.Value) {
	if laser.AnyIsNil(mv.Map) {
		return
	}
	laser.MapDeleteValue(mv.Map, laser.NonPtrValue(key))

	if mv.TmpSave != nil {
		mv.TmpSave.SaveTmp()
	}
	mv.SetChanged()
	mv.Update()
}

func (mv *MapViewInline) ContextMenu(m *gi.Scene, keyv reflect.Value) {
	if mv.IsReadOnly() {
		return
	}
	gi.NewButton(m).SetText("Add").SetIcon(icons.Add).OnClick(func(e events.Event) {
		mv.MapAdd()
	})
	gi.NewButton(m).SetText("Delete").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		mv.MapDelete(keyv)
	})
}

func (mv *MapViewInline) UpdateValues() {
	// maps have to re-read their values because they can't get pointers!
	mv.Update()
}

func (mv *MapViewInline) MapSizeChanged() bool {
	if laser.AnyIsNil(mv.Map) {
		return mv.ConfigSize != 0
	}
	mpv := reflect.ValueOf(mv.Map)
	mpvnp := laser.NonPtrValue(laser.OnePtrUnderlyingValue(mpv))
	keys := mpvnp.MapKeys()
	return mv.ConfigSize != len(keys)
}

func (mv *MapViewInline) SizeUp() {
	if mv.MapSizeChanged() {
		mv.Update()
	}
	mv.Layout.SizeUp()
}
