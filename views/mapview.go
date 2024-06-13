// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/types"
)

// KeyValueTable represents a map value using two columns of editable key and value widgets.
type KeyValueTable struct {
	core.Frame

	// Map is the pointer to the map that we are viewing.
	Map any

	// Inline is whether to display the map in one line.
	Inline bool

	// SortValue is whether to sort by values instead of keys.
	SortValues bool

	// ncols is the number of columns to display if the key value table is not inline.
	ncols int
}

func (kv *KeyValueTable) WidgetValue() any { return &kv.Map }

func (kv *KeyValueTable) Init() {
	kv.Frame.Init()
	kv.Styler(func(s *styles.Style) {
		if kv.Inline {
			return
		}
		s.Display = styles.Grid
		s.Columns = kv.ncols
		s.Overflow.Set(styles.OverflowAuto)
		s.Grow.Set(1, 1)
		s.Min.X.Em(20)
		s.Min.Y.Em(10)
	})

	kv.Maker(func(p *core.Plan) {
		if reflectx.AnyIsNil(kv.Map) {
			return
		}
		mapv := reflectx.Underlying(reflect.ValueOf(kv.Map))

		kv.ncols = 2
		typeAny := false
		valueType := mapv.Type().Elem()
		if valueType.String() == "interface {}" {
			kv.ncols = 3
			typeAny = true
		}

		builtinTypes := types.BuiltinTypes()

		keys := reflectx.MapSort(kv.Map, !kv.SortValues, true)
		for _, key := range keys {
			keytxt := reflectx.ToString(key.Interface())
			keynm := "key-" + keytxt
			valnm := "value-" + keytxt

			core.AddNew(p, keynm, func() core.Value {
				return core.ToValue(key.Interface(), "")
			}, func(w core.Value) {
				BindMapKey(mapv, key, w)
				wb := w.AsWidget()
				wb.SetReadOnly(kv.IsReadOnly())
				wb.Styler(func(s *styles.Style) {
					s.SetReadOnly(kv.IsReadOnly())
					s.SetTextWrap(false)
				})
				wb.OnChange(func(e events.Event) {
					kv.SendChange(e)
					kv.Update()
				})
				wb.SetReadOnly(kv.IsReadOnly())
				wb.OnInput(kv.HandleEvent)
				if !kv.IsReadOnly() {
					wb.AddContextMenu(func(m *core.Scene) {
						kv.ContextMenu(m, key)
					})
				}
				wb.Updater(func() {
					BindMapKey(mapv, key, w)
					wb.SetReadOnly(kv.IsReadOnly())
				})
			})
			core.AddNew(p, valnm, func() core.Value {
				val := mapv.MapIndex(key).Interface()
				w := core.ToValue(val, "")
				return BindMapValue(mapv, key, w)
			}, func(w core.Value) {
				wb := w.AsWidget()
				wb.SetReadOnly(kv.IsReadOnly())
				wb.OnChange(func(e events.Event) { kv.SendChange(e) })
				wb.OnInput(kv.HandleEvent)
				wb.Styler(func(s *styles.Style) {
					s.SetReadOnly(kv.IsReadOnly())
					s.SetTextWrap(false)
				})
				if !kv.IsReadOnly() {
					wb.AddContextMenu(func(m *core.Scene) {
						kv.ContextMenu(m, key)
					})
				}
				wb.Updater(func() {
					BindMapValue(mapv, key, w)
					wb.SetReadOnly(kv.IsReadOnly())
				})
			})

			if typeAny {
				typnm := "type-" + keytxt
				core.AddAt(p, typnm, func(w *core.Chooser) {
					w.SetTypes(builtinTypes...)
					w.OnChange(func(e events.Event) {
						typ := reflect.TypeOf(w.CurrentItem.Value.(*types.Type).Instance)
						newVal := reflect.New(typ)
						// try our best to convert the existing value to the new type
						reflectx.SetRobust(newVal.Interface(), mapv.MapIndex(key).Interface())
						mapv.SetMapIndex(key, newVal.Elem())
						kv.DeleteChildByName(valnm) // force it to be updated
						kv.Update()
					})
					w.Updater(func() {
						w.SetReadOnly(kv.IsReadOnly())
						vtyp := types.TypeByValue(mapv.MapIndex(key).Interface())
						if vtyp == nil {
							vtyp = types.TypeByName("string") // default to string
						}
						w.SetCurrentValue(vtyp)
					})
				})
			}
		}
		if kv.Inline {
			if !kv.IsReadOnly() {
				core.AddAt(p, "add-button", func(w *core.Button) {
					w.SetIcon(icons.Add).SetType(core.ButtonTonal)
					w.Tooltip = "Add an element"
					w.OnClick(func(e events.Event) {
						kv.MapAdd()
					})
				})
			}
			core.AddAt(p, "edit-button", func(w *core.Button) {
				w.SetIcon(icons.Edit).SetType(core.ButtonTonal)
				w.Tooltip = "Edit in a dialog"
				w.OnClick(func(e events.Event) {
					d := core.NewBody().AddTitle(kv.ValueTitle).AddText(kv.Tooltip)
					NewKeyValueTable(d).SetMap(kv.Map).SetValueTitle(kv.ValueTitle)
					d.OnClose(func(e events.Event) {
						kv.Update()
						kv.SendChange()
					})
					d.RunFullDialog(kv)
				})
			})
		}
	})
}

func (kv *KeyValueTable) ContextMenu(m *core.Scene, keyv reflect.Value) {
	if kv.IsReadOnly() {
		return
	}
	core.NewButton(m).SetText("Add").SetIcon(icons.Add).OnClick(func(e events.Event) {
		kv.MapAdd()
	})
	core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		kv.MapDelete(keyv)
	})
}

// ToggleSort toggles sorting by values vs. keys
func (kv *KeyValueTable) ToggleSort() {
	kv.SortValues = !kv.SortValues
	kv.Update()
}

// MapAdd adds a new entry to the map
func (kv *KeyValueTable) MapAdd() {
	if reflectx.AnyIsNil(kv.Map) {
		return
	}
	reflectx.MapAdd(kv.Map)

	kv.SendChange()
	kv.Update()
}

// MapDelete deletes a key-value from the map
func (kv *KeyValueTable) MapDelete(key reflect.Value) {
	if reflectx.AnyIsNil(kv.Map) {
		return
	}
	reflectx.MapDelete(kv.Map, reflectx.NonPointerValue(key))

	kv.SendChange()
	kv.Update()
}

// MakeToolbar configures a [core.Toolbar] for this view
func (kv *KeyValueTable) MakeToolbar(p *core.Plan) {
	if reflectx.AnyIsNil(kv.Map) {
		return
	}
	core.Add(p, func(w *core.Button) {
		w.SetText("Sort").SetIcon(icons.Sort).SetTooltip("Switch between sorting by the keys and the values").
			OnClick(func(e events.Event) {
				kv.ToggleSort()
			})
	})
	if !kv.IsReadOnly() {
		core.Add(p, func(w *core.Button) {
			w.SetText("Add").SetIcon(icons.Add).SetTooltip("Add a new element to the map").
				OnClick(func(e events.Event) {
					kv.MapAdd()
				})
		})
	}
}

// BindMapKey is a version of [core.Bind] that works for keys in a map.
func BindMapKey[T core.Value](mapv reflect.Value, key reflect.Value, vw T) T {
	wb := vw.AsWidget()
	wb.ValueUpdate = func() {
		if vws, ok := any(vw).(core.ValueSetter); ok {
			core.ErrorSnackbar(vw, vws.SetWidgetValue(key.Interface()))
		} else {
			core.ErrorSnackbar(vw, reflectx.SetRobust(vw.WidgetValue(), key.Interface()))
		}
	}
	wb.ValueOnChange = func() {
		newKey := reflect.New(key.Type())
		core.ErrorSnackbar(vw, reflectx.SetRobust(newKey.Interface(), vw.WidgetValue()))
		newKey = newKey.Elem()
		if !mapv.MapIndex(newKey).IsValid() { // not already taken
			mapv.SetMapIndex(newKey, mapv.MapIndex(key))
			mapv.SetMapIndex(key, reflect.Value{})
			return
		}
		d := core.NewBody().AddTitle("Key already exists").AddText(fmt.Sprintf("The key %q already exists", reflectx.ToString(newKey.Interface())))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			d.AddOK(parent).SetText("Overwrite").OnClick(func(e events.Event) {
				mapv.SetMapIndex(newKey, mapv.MapIndex(key))
				mapv.SetMapIndex(key, reflect.Value{})
				wb.SendChange()
			})
		})
		d.RunDialog(vw)
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
