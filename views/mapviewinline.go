// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"reflect"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// MapViewInline represents a map within a single line of key and value widgets.
// This is typically used for smaller maps.
type MapViewInline struct {
	core.Layout

	// Map is the pointer to the map that we are viewing.
	Map any

	// MapValue is the [Value] associated with this map view, if there is one.
	MapValue Value `set:"-"`

	// Keys are [Value] representations of the map keys.
	Keys []Value `json:"-" xml:"-" set:"-"`

	// Values are [Value] representations of the map values.
	Values []Value `json:"-" xml:"-" set:"-"`

	// ViewPath is a record of parent view names that have led up to this view.
	// It is displayed as extra contextual information in view dialogs.
	ViewPath string

	// configSize is the size of the map when the widget was configured.
	configSize int
}

func (mv *MapViewInline) OnInit() {
	mv.Layout.OnInit()
	mv.SetStyles()
}

func (mv *MapViewInline) SetStyles() {
	mv.Style(func(s *styles.Style) {
		s.Grow.Set(0, 0)
	})
	mv.OnWidgetAdded(func(w core.Widget) {
		switch w.PathFrom(mv) {
		case "add-button":
			ab := w.(*core.Button)
			w.Style(func(s *styles.Style) {
				ab.SetType(core.ButtonTonal)
			})
			w.OnClick(func(e events.Event) {
				mv.MapAdd()
			})
		case "edit-button":
			w.Style(func(s *styles.Style) {
				w.(*core.Button).SetType(core.ButtonTonal)
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
					tmptyp := reflectx.NonPointerType(reflect.TypeOf(mv.Map))
					title = "Map of " + tmptyp.String()
				}
				d := core.NewBody().AddTitle(title).AddText(mv.Tooltip)
				NewMapView(d).SetViewPath(vpath).SetMap(mv.Map)
				d.OnClose(func(e events.Event) {
					mv.Update()
					mv.SendChange()
				})
				d.RunFullDialog(mv)
			})
		}
	})
}

func (mv *MapViewInline) Config(c *core.Config) {
	mv.DeleteChildren()
	if reflectx.AnyIsNil(mv.Map) {
		mv.configSize = 0
		return
	}
	config := tree.Config{}
	mv.Keys = make([]Value, 0)
	mv.Values = make([]Value, 0)

	mpv := reflect.ValueOf(mv.Map)
	mpvnp := reflectx.NonPointerValue(reflectx.OnePointerUnderlyingValue(mpv))
	keys := mpvnp.MapKeys() // this is a slice of reflect.Value
	mv.configSize = len(keys)

	reflectx.ValueSliceSort(keys, true)
	for i, key := range keys {
		if i >= core.SystemSettings.MapInlineLength {
			break
		}
		kv := ToValue(key.Interface(), "")
		if kv == nil { // shouldn't happen
			continue
		}
		kv.SetMapKey(key, mv.Map)

		val := reflectx.OnePointerUnderlyingValue(mpvnp.MapIndex(key))
		vv := ToValue(val.Interface(), "")
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetMapValue(val, mv.Map, key.Interface(), kv, mv.ViewPath) // needs key value to track updates

		keytxt := reflectx.ToString(key.Interface())
		keynm := "key-" + keytxt
		valnm := "value-" + keytxt

		config.Add(kv.WidgetType(), keynm)
		config.Add(vv.WidgetType(), valnm)
		mv.Keys = append(mv.Keys, kv)
		mv.Values = append(mv.Values, vv)
	}
	config.Add(core.ButtonType, "add-button")
	config.Add(core.ButtonType, "edit-button")
	mv.ConfigChildren(config)
	for i, vv := range mv.Values {
		kv := mv.Keys[i]
		vv.OnChange(func(e events.Event) { mv.SendChange() })
		kv.OnChange(func(e events.Event) {
			mv.SendChange()
			mv.Update()
		})
		w, wb := core.AsWidget(mv.Child((i * 2) + 1))
		kw, kwb := core.AsWidget(mv.Child(i * 2))
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
			wb.AddContextMenu(func(m *core.Scene) {
				mv.ContextMenu(m, kv.Val())
			})
			kwb.AddContextMenu(func(m *core.Scene) {
				mv.ContextMenu(m, kv.Val())
			})
		}
	}
	adack, err := mv.Children().ElemFromEndTry(1)
	if err == nil {
		adbt := adack.(*core.Button)
		adbt.SetType(core.ButtonTonal)
		adbt.SetIcon(icons.Add)
		adbt.Tooltip = "add an entry to the map"

	}
	edack, err := mv.Children().ElemFromEndTry(0)
	if err == nil {
		edbt := edack.(*core.Button)
		edbt.SetType(core.ButtonTonal)
		edbt.SetIcon(icons.Edit)
		edbt.Tooltip = "map edit dialog"
	}
}

// MapAdd adds a new entry to the map
func (mv *MapViewInline) MapAdd() {
	if reflectx.AnyIsNil(mv.Map) {
		return
	}
	reflectx.MapAdd(mv.Map)

	mv.SendChange()
	mv.Update()
}

// MapDelete deletes a key-value from the map
func (mv *MapViewInline) MapDelete(key reflect.Value) {
	if reflectx.AnyIsNil(mv.Map) {
		return
	}
	reflectx.MapDelete(mv.Map, reflectx.NonPointerValue(key))

	mv.SendChange()
	mv.Update()
}

func (mv *MapViewInline) ContextMenu(m *core.Scene, keyv reflect.Value) {
	if mv.IsReadOnly() {
		return
	}
	core.NewButton(m).SetText("Add").SetIcon(icons.Add).OnClick(func(e events.Event) {
		mv.MapAdd()
	})
	core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		mv.MapDelete(keyv)
	})
}

func (mv *MapViewInline) UpdateValues() {
	// maps have to re-read their values because they can't get pointers!
	mv.Update()
}

func (mv *MapViewInline) MapSizeChanged() bool {
	if reflectx.AnyIsNil(mv.Map) {
		return mv.configSize != 0
	}
	mpv := reflect.ValueOf(mv.Map)
	mpvnp := reflectx.NonPointerValue(reflectx.OnePointerUnderlyingValue(mpv))
	keys := mpvnp.MapKeys()
	return mv.configSize != len(keys)
}

func (mv *MapViewInline) SizeUp() {
	if mv.MapSizeChanged() {
		mv.Update()
	}
	mv.Layout.SizeUp()
}
