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
	core.Layout

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
	mv.Layout.OnInit()
	mv.SetStyles()
}

func (mv *MapViewInline) SetStyles() {
	mv.Style(func(s *styles.Style) {
		s.Grow.Set(0, 0)
	})
}

func (mv *MapViewInline) Config(c *core.Config) {
	if reflectx.AnyIsNil(mv.Map) {
		mv.configSize = 0
		return
	}
	mpv := reflect.ValueOf(mv.Map)
	mpvnp := reflectx.NonPointerValue(reflectx.OnePointerUnderlyingValue(mpv))
	keys := mpvnp.MapKeys() // this is a slice of reflect.Value
	reflectx.ValueSliceSort(keys, true)
	mv.configSize = len(keys)

	for i, key := range keys {
		if i >= core.SystemSettings.MapInlineLength {
			break
		}
		keytxt := reflectx.ToString(key.Interface())
		keynm := "key-" + keytxt
		valnm := "value-" + keytxt
		val := reflectx.OnePointerUnderlyingValue(mpvnp.MapIndex(key))

		core.Configure(c, keynm, func() core.Value {
			w := core.NewValue(key.Interface())
			wb := w.AsWidget()
			wb.SetReadOnly(mv.IsReadOnly())
			// kv.SetMapKey(key, mv.Map)
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
			core.Bind(key.Interface(), w)
			// vv.SetMapValue(val, mv.Map, key.Interface(), kv, mv.ViewPath) // needs key value value to track updates
			wb.SetReadOnly(mv.IsReadOnly())
		})
		core.Configure(c, valnm, func() core.Value {
			w := core.NewValue(val.Interface())
			wb := w.AsWidget()
			wb.SetReadOnly(mv.IsReadOnly())
			// vv.SetMapValue(val, mv.Map, key.Interface(), kv, mv.ViewPath) // needs key value value to track updates
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
			core.Bind(val.Interface(), w)
			// vv.SetMapValue(val, mv.Map, key.Interface(), kv, mv.ViewPath) // needs key value value to track updates
			wb.SetReadOnly(mv.IsReadOnly())
		})
	}
	if !mv.IsReadOnly() {
		core.Configure(c, "add-button", func() *core.Button {
			w := core.NewButton().SetIcon(icons.Add).SetType(core.ButtonTonal)
			w.Tooltip = "add an element to the map"
			w.OnClick(func(e events.Event) {
				mv.MapAdd()
			})
			return w
		})
	}
	core.Configure(c, "edit-button", func() *core.Button {
		w := core.NewButton().SetIcon(icons.Edit).SetType(core.ButtonTonal)
		w.Tooltip = "edit in a dialog"
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
		return w
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
