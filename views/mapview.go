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

// MapView represents a map using two columns of editable key and value widgets.
type MapView struct {
	core.Frame

	// Map is the pointer to the map that we are viewing.
	Map any

	// SortValue is whether to sort by values instead of keys
	SortValues bool

	// ViewPath is a record of parent view names that have led up to this view.
	// It is displayed as extra contextual information in view dialogs.
	ViewPath string

	// ncols is the number of columns in the map
	ncols int
}

func (mv *MapView) OnInit() {
	mv.Frame.OnInit()
	mv.SetStyles()
}

func (mv *MapView) SetStyles() {
	mv.Style(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = mv.ncols
		s.Overflow.Set(styles.OverflowAuto)
		s.Grow.Set(1, 1)
		s.Min.X.Em(20)
		s.Min.Y.Em(10)
	})
}

func (mv *MapView) Config(c *core.Config) {
	if reflectx.AnyIsNil(mv.Map) {
		return
	}
	mapv := reflectx.Underlying(reflect.ValueOf(mv.Map))

	valtyp := mapv.Type().Elem()
	ncol := 2
	ifaceType := false
	if valtyp.Kind() == reflect.Interface && valtyp.String() == "interface {}" {
		ifaceType = true
		ncol = 3
		// todo: need some way of setting & getting
		// this for given domain mapview could have a structview parent and
		// the source node of that struct, if a tree, could have a property --
		// unlike inline case, plain mapview is not a child of struct view
		// directly -- but field on struct view does create the mapview
		// dialog.. a bit hacky and indirect..
	}

	// valtypes := append(kit.Types.AllTagged(typeTag), kit.Enums.AllTagged(typeTag)...)
	// valtypes = append(valtypes, kit.Types.AllTagged("basic-type")...)
	// valtypes = append(valtypes, kit.TypeFor[reflect.Type]())
	// valtypes := types.AllEmbeddersOf(tree.NodeBaseType) // todo: this is not right

	mv.ncols = ncol

	keys := reflectx.MapSort(mv.Map, !mv.SortValues, true) // note: this is a slice of reflect.Value!
	for _, key := range keys {
		keytxt := reflectx.ToString(key.Interface())
		keynm := "key-" + keytxt
		valnm := "value-" + keytxt

		core.ConfigureNew(c, keynm, func() core.Value {
			w := core.ToValue(key.Interface(), "")
			BindMapKey(mapv, key, w)
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

		if ifaceType {
			// TODO(config)
			// typnm := "type-" + keytxt
			// core.Configure(c, typnm, func(w *core.Chooser) {
			// 	w.SetTypes(valtypes...)
			// 	vtyp := reflectx.NonPointerType(reflect.TypeOf(val))
			// 	if vtyp == nil {
			// 		vtyp = reflect.TypeOf("") // default to string
			// 	}
			// 	w.SetCurrentValue(vtyp)
			// })
		}
	}
}

func (mv *MapView) ContextMenu(m *core.Scene, keyv reflect.Value) {
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

// ToggleSort toggles sorting by values vs. keys
func (mv *MapView) ToggleSort() {
	mv.SortValues = !mv.SortValues
	mv.Update()
}

// MapAdd adds a new entry to the map
func (mv *MapView) MapAdd() {
	if reflectx.AnyIsNil(mv.Map) {
		return
	}
	reflectx.MapAdd(mv.Map)

	mv.SendChange()
	mv.Update()
}

// MapDelete deletes a key-value from the map
func (mv *MapView) MapDelete(key reflect.Value) {
	if reflectx.AnyIsNil(mv.Map) {
		return
	}
	reflectx.MapDelete(mv.Map, reflectx.NonPointerValue(key))

	mv.SendChange()
	mv.Update()
}

// ConfigToolbar configures a [core.Toolbar] for this view
func (mv *MapView) ConfigToolbar(c *core.Config) {
	if reflectx.AnyIsNil(mv.Map) {
		return
	}
	core.Configure(c, "", func(w *core.Button) {
		w.SetText("Sort").SetIcon(icons.Sort).SetTooltip("Switch between sorting by the keys vs. the values").
			OnClick(func(e events.Event) {
				mv.ToggleSort()
			})
	})
	if !mv.IsReadOnly() {
		core.Configure(c, "", func(w *core.Button) {
			w.SetText("Add").SetIcon(icons.Add).SetTooltip("Add a new element to the map").
				OnClick(func(e events.Event) {
					mv.MapAdd()
				})
		})
	}
}

// BindMapKey is a version of [core.Bind] that works for keys in a map.
func BindMapKey[T core.Value](mapv reflect.Value, key reflect.Value, vw T) T {
	// We must have an addressable key so that we can use Addr when we set it down below.
	// This address doesn't point to the actual key, but it serves as a fake pointer we
	// can use to keep the key in sync locally here.
	key = reflectx.NewFrom(key).Elem()
	wb := vw.AsWidget()
	wb.ValueUpdate = func() {
		if vws, ok := any(vw).(core.ValueSetter); ok {
			core.ErrorSnackbar(vw, vws.SetWidgetValue(key.Interface()))
		} else {
			core.ErrorSnackbar(vw, reflectx.SetRobust(vw.WidgetValue(), key.Interface()))
		}
	}
	wb.ValueOnChange = func() {
		// TODO(config): check for duplicates
		value := mapv.MapIndex(key)
		mapv.SetMapIndex(key, reflect.ValueOf(nil))
		core.ErrorSnackbar(vw, reflectx.SetRobust(key.Addr().Interface(), vw.WidgetValue())) // must set using address
		mapv.SetMapIndex(key, value)
	}
	if ob, ok := any(vw).(core.OnBinder); ok {
		ob.OnBind(key.Interface())
	}
	return vw
}

// BindMapValue is a version of [core.Bind] that works for values in a map.
func BindMapValue[T core.Value](mapv reflect.Value, key reflect.Value, vw T) T {
	wb := vw.AsWidget()
	wb.ValueUpdate = func() {
		value := mapv.MapIndex(key).Interface()
		if vws, ok := any(vw).(core.ValueSetter); ok {
			core.ErrorSnackbar(vw, vws.SetWidgetValue(value))
		} else {
			core.ErrorSnackbar(vw, reflectx.SetRobust(vw.WidgetValue(), value))
		}
	}
	wb.ValueOnChange = func() {
		value := reflect.New(mapv.Type().Elem())
		core.ErrorSnackbar(vw, reflectx.SetRobust(value.Interface(), vw.WidgetValue()))
		mapv.SetMapIndex(key, value.Elem())
	}
	if ob, ok := any(vw).(core.OnBinder); ok {
		value := mapv.MapIndex(key).Interface()
		ob.OnBind(value)
	}
	return vw
}
