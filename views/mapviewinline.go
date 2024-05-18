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
)

// MapViewInline represents a map within a single line of key and value widgets.
// This is typically used for smaller maps.
type MapViewInline struct {
	core.Frame

	// Map is the pointer to the map that we are viewing.
	Map any

	// MapValue is the [Value] associated with this map view, if there is one.
	MapValue Value `set:"-"`

	// ViewPath is a record of parent view names that have led up to this view.
	// It is displayed as extra contextual information in view dialogs.
	ViewPath string

	// configSize is the size of the map when the widget was configured.
	configSize int
}

func (mv *MapViewInline) OnInit() {
	mv.Frame.OnInit()
	mv.SetStyles()
}

func (mv *MapViewInline) Config(c *core.Config) {
	if reflectx.AnyIsNil(mv.Map) {
		mv.configSize = 0
		return
	}
	mapv := reflectx.Underlying(reflect.ValueOf(mv.Map))
	keys := mapv.MapKeys()
	reflectx.ValueSliceSort(keys, true)
	mv.configSize = len(keys)

	for i, key := range keys {
		if i >= core.SystemSettings.MapInlineLength {
			break
		}
		keytxt := reflectx.ToString(key.Interface())
		keynm := "key-" + keytxt
		valnm := "value-" + keytxt

		core.ConfigureNew(c, keynm, func() core.Value {
			w := core.ToValue(key.Interface(), "")
			BindMapKey(mapv, key, w)
			wb := w.AsWidget()
			wb.SetReadOnly(mv.IsReadOnly())
			w.Style(func(s *styles.Style) {
				s.SetReadOnly(mv.IsReadOnly())
				s.SetTextWrap(false)
			})
			wb.OnChange(func(e events.Event) {
				mv.SendChange(e)
				mv.Update()
			})
			wb.SetReadOnly(mv.IsReadOnly())
			wb.OnInput(mv.HandleEvent)
			if !mv.IsReadOnly() {
				w.AddContextMenu(func(m *core.Scene) {
					mv.ContextMenu(m, key)
				})
			}
			return w
		}, func(w core.Value) {
			wb := w.AsWidget()
			BindMapKey(mapv, key, w)
			wb.SetReadOnly(mv.IsReadOnly())
		})
		core.ConfigureNew(c, valnm, func() core.Value {
			val := mapv.MapIndex(key).Interface()
			w := core.ToValue(val, "")
			BindMapValue(mapv, key, w)
			wb := w.AsWidget()
			wb.SetReadOnly(mv.IsReadOnly())
			wb.OnChange(func(e events.Event) { mv.SendChange(e) })
			wb.OnInput(mv.HandleEvent)
			w.Style(func(s *styles.Style) {
				s.SetReadOnly(mv.IsReadOnly())
				s.SetTextWrap(false)
			})
			if !mv.IsReadOnly() {
				w.AddContextMenu(func(m *core.Scene) {
					mv.ContextMenu(m, key)
				})
			}
			return w
		}, func(w core.Value) {
			wb := w.AsWidget()
			BindMapValue(mapv, key, w)
			wb.SetReadOnly(mv.IsReadOnly())
		})
	}
	if !mv.IsReadOnly() {
		core.Configure(c, "add-button", func(w *core.Button) {
			w.SetIcon(icons.Add).SetType(core.ButtonTonal)
			w.Tooltip = "Add an element"
			w.OnClick(func(e events.Event) {
				mv.MapAdd()
			})
		})
	}
	core.Configure(c, "edit-button", func(w *core.Button) {
		w.SetIcon(icons.Edit).SetType(core.ButtonTonal)
		w.Tooltip = "Edit in a dialog"
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
	})
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

func (mv *MapViewInline) MapSizeChanged() bool {
	if reflectx.AnyIsNil(mv.Map) {
		return mv.configSize != 0
	}
	mpv := reflect.ValueOf(mv.Map)
	mpvnp := reflectx.NonPointerValue(reflectx.UnderlyingPointer(mpv))
	keys := mpvnp.MapKeys()
	return mv.configSize != len(keys)
}

func (mv *MapViewInline) SizeUp() {
	if mv.MapSizeChanged() {
		mv.Update()
	}
	mv.Frame.SizeUp()
}
