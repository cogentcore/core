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

// KeyedList represents a map value using two columns of editable key and value widgets.
type KeyedList struct {
	core.Frame

	// Map is the pointer to the map that we are viewing.
	Map any

	// Inline is whether to display the map in one line.
	Inline bool

	// SortValue is whether to sort by values instead of keys.
	SortValues bool

	// ncols is the number of columns to display if the keyed list is not inline.
	ncols int
}

func (kl *KeyedList) WidgetValue() any { return &kl.Map }

func (kl *KeyedList) Init() {
	kl.Frame.Init()
	kl.Styler(func(s *styles.Style) {
		if kl.Inline {
			return
		}
		s.Display = styles.Grid
		s.Columns = kl.ncols
		s.Overflow.Set(styles.OverflowAuto)
		s.Grow.Set(1, 1)
		s.Min.X.Em(20)
		s.Min.Y.Em(10)
	})

	kl.Maker(func(p *core.Plan) {
		if reflectx.AnyIsNil(kl.Map) {
			return
		}
		mapv := reflectx.Underlying(reflect.ValueOf(kl.Map))

		kl.ncols = 2
		typeAny := false
		valueType := mapv.Type().Elem()
		if valueType.String() == "interface {}" {
			kl.ncols = 3
			typeAny = true
		}

		builtinTypes := types.BuiltinTypes()

		keys := reflectx.MapSort(kl.Map, !kl.SortValues, true)
		for _, key := range keys {
			keytxt := reflectx.ToString(key.Interface())
			keynm := "key-" + keytxt
			valnm := "value-" + keytxt

			core.AddNew(p, keynm, func() core.Value {
				return core.ToValue(key.Interface(), "")
			}, func(w core.Value) {
				BindMapKey(mapv, key, w)
				wb := w.AsWidget()
				wb.SetReadOnly(kl.IsReadOnly())
				wb.Styler(func(s *styles.Style) {
					s.SetReadOnly(kl.IsReadOnly())
					s.SetTextWrap(false)
				})
				wb.OnChange(func(e events.Event) {
					kl.SendChange(e)
					kl.Update()
				})
				wb.SetReadOnly(kl.IsReadOnly())
				wb.OnInput(kl.HandleEvent)
				if !kl.IsReadOnly() {
					wb.AddContextMenu(func(m *core.Scene) {
						kl.ContextMenu(m, key)
					})
				}
				wb.Updater(func() {
					BindMapKey(mapv, key, w)
					wb.SetReadOnly(kl.IsReadOnly())
				})
			})
			core.AddNew(p, valnm, func() core.Value {
				val := mapv.MapIndex(key).Interface()
				w := core.ToValue(val, "")
				return BindMapValue(mapv, key, w)
			}, func(w core.Value) {
				wb := w.AsWidget()
				wb.SetReadOnly(kl.IsReadOnly())
				wb.OnChange(func(e events.Event) { kl.SendChange(e) })
				wb.OnInput(kl.HandleEvent)
				wb.Styler(func(s *styles.Style) {
					s.SetReadOnly(kl.IsReadOnly())
					s.SetTextWrap(false)
				})
				if !kl.IsReadOnly() {
					wb.AddContextMenu(func(m *core.Scene) {
						kl.ContextMenu(m, key)
					})
				}
				wb.Updater(func() {
					BindMapValue(mapv, key, w)
					wb.SetReadOnly(kl.IsReadOnly())
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
						kl.DeleteChildByName(valnm) // force it to be updated
						kl.Update()
					})
					w.Updater(func() {
						w.SetReadOnly(kl.IsReadOnly())
						vtyp := types.TypeByValue(mapv.MapIndex(key).Interface())
						if vtyp == nil {
							vtyp = types.TypeByName("string") // default to string
						}
						w.SetCurrentValue(vtyp)
					})
				})
			}
		}
		if kl.Inline {
			if !kl.IsReadOnly() {
				core.AddAt(p, "add-button", func(w *core.Button) {
					w.SetIcon(icons.Add).SetType(core.ButtonTonal)
					w.Tooltip = "Add an element"
					w.OnClick(func(e events.Event) {
						kl.MapAdd()
					})
				})
			}
			core.AddAt(p, "edit-button", func(w *core.Button) {
				w.SetIcon(icons.Edit).SetType(core.ButtonTonal)
				w.Tooltip = "Edit in a dialog"
				w.OnClick(func(e events.Event) {
					d := core.NewBody().AddTitle(kl.ValueTitle).AddText(kl.Tooltip)
					NewKeyedList(d).SetMap(kl.Map).SetValueTitle(kl.ValueTitle)
					d.OnClose(func(e events.Event) {
						kl.Update()
						kl.SendChange()
					})
					d.RunFullDialog(kl)
				})
			})
		}
	})
}

func (kl *KeyedList) ContextMenu(m *core.Scene, keyv reflect.Value) {
	if kl.IsReadOnly() {
		return
	}
	core.NewButton(m).SetText("Add").SetIcon(icons.Add).OnClick(func(e events.Event) {
		kl.MapAdd()
	})
	core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		kl.MapDelete(keyv)
	})
}

// ToggleSort toggles sorting by values vs. keys
func (kl *KeyedList) ToggleSort() {
	kl.SortValues = !kl.SortValues
	kl.Update()
}

// MapAdd adds a new entry to the map
func (kl *KeyedList) MapAdd() {
	if reflectx.AnyIsNil(kl.Map) {
		return
	}
	reflectx.MapAdd(kl.Map)

	kl.SendChange()
	kl.Update()
}

// MapDelete deletes a key-value from the map
func (kl *KeyedList) MapDelete(key reflect.Value) {
	if reflectx.AnyIsNil(kl.Map) {
		return
	}
	reflectx.MapDelete(kl.Map, reflectx.NonPointerValue(key))

	kl.SendChange()
	kl.Update()
}

// MakeToolbar configures a [core.Toolbar] for this view
func (kl *KeyedList) MakeToolbar(p *core.Plan) {
	if reflectx.AnyIsNil(kl.Map) {
		return
	}
	core.Add(p, func(w *core.Button) {
		w.SetText("Sort").SetIcon(icons.Sort).SetTooltip("Switch between sorting by the keys and the values").
			OnClick(func(e events.Event) {
				kl.ToggleSort()
			})
	})
	if !kl.IsReadOnly() {
		core.Add(p, func(w *core.Button) {
			w.SetText("Add").SetIcon(icons.Add).SetTooltip("Add a new element to the map").
				OnClick(func(e events.Event) {
					kl.MapAdd()
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
