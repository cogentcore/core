// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// KeyedList represents a map value using two columns of editable key and value widgets.
type KeyedList struct {
	Frame

	// Map is the pointer to the map that we are viewing.
	Map any

	// Inline is whether to display the map in one line.
	Inline bool

	// SortByValues is whether to sort by values instead of keys.
	SortByValues bool

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

	kl.Maker(func(p *tree.Plan) {
		mapv := reflectx.Underlying(reflect.ValueOf(kl.Map))
		if reflectx.IsNil(mapv) {
			return
		}

		kl.ncols = 2
		typeAny := false
		valueType := mapv.Type().Elem()
		if valueType.String() == "interface {}" {
			kl.ncols = 3
			typeAny = true
		}

		builtinTypes := types.BuiltinTypes()

		keys := reflectx.MapSort(kl.Map, !kl.SortByValues, true)
		for _, key := range keys {
			keytxt := reflectx.ToString(key.Interface())
			keynm := "key-" + keytxt
			valnm := "value-" + keytxt

			tree.AddNew(p, keynm, func() Value {
				return toValue(key.Interface(), "")
			}, func(w Value) {
				bindMapKey(mapv, key, w)
				wb := w.AsWidget()
				wb.SetReadOnly(kl.IsReadOnly())
				wb.Styler(func(s *styles.Style) {
					s.SetReadOnly(kl.IsReadOnly())
					s.SetTextWrap(false)
				})
				wb.OnChange(func(e events.Event) {
					kl.UpdateChange(e)
				})
				wb.SetReadOnly(kl.IsReadOnly())
				wb.OnInput(kl.HandleEvent)
				if !kl.IsReadOnly() {
					wb.AddContextMenu(func(m *Scene) {
						kl.contextMenu(m, key)
					})
				}
				wb.Updater(func() {
					bindMapKey(mapv, key, w)
					wb.SetReadOnly(kl.IsReadOnly())
				})
			})
			tree.AddNew(p, valnm, func() Value {
				val := mapv.MapIndex(key).Interface()
				w := toValue(val, "")
				return bindMapValue(mapv, key, w)
			}, func(w Value) {
				wb := w.AsWidget()
				wb.SetReadOnly(kl.IsReadOnly())
				wb.OnChange(func(e events.Event) { kl.SendChange(e) })
				wb.OnInput(kl.HandleEvent)
				wb.Styler(func(s *styles.Style) {
					s.SetReadOnly(kl.IsReadOnly())
					s.SetTextWrap(false)
				})
				if !kl.IsReadOnly() {
					wb.AddContextMenu(func(m *Scene) {
						kl.contextMenu(m, key)
					})
				}
				wb.Updater(func() {
					bindMapValue(mapv, key, w)
					wb.SetReadOnly(kl.IsReadOnly())
				})
			})

			if typeAny {
				typnm := "type-" + keytxt
				tree.AddAt(p, typnm, func(w *Chooser) {
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
		if kl.Inline && !kl.IsReadOnly() {
			tree.AddAt(p, "add-button", func(w *Button) {
				w.SetIcon(icons.Add).SetType(ButtonTonal)
				w.Tooltip = "Add an element"
				w.OnClick(func(e events.Event) {
					kl.AddItem()
				})
			})
		}
	})
}

func (kl *KeyedList) contextMenu(m *Scene, keyv reflect.Value) {
	if kl.IsReadOnly() {
		return
	}
	NewButton(m).SetText("Add").SetIcon(icons.Add).OnClick(func(e events.Event) {
		kl.AddItem()
	})
	NewButton(m).SetText("Delete").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		kl.DeleteItem(keyv)
	})
	if kl.Inline {
		NewButton(m).SetText("Open in dialog").SetIcon(icons.OpenInNew).OnClick(func(e events.Event) {
			d := NewBody(kl.ValueTitle)
			NewText(d).SetType(TextSupporting).SetText(kl.Tooltip)
			NewKeyedList(d).SetMap(kl.Map).SetValueTitle(kl.ValueTitle).SetReadOnly(kl.IsReadOnly())
			d.OnClose(func(e events.Event) {
				kl.UpdateChange(e)
			})
			d.RunFullDialog(kl)
		})
	}
}

// toggleSort toggles sorting by values vs. keys
func (kl *KeyedList) toggleSort() {
	kl.SortByValues = !kl.SortByValues
	kl.Update()
}

// AddItem adds a new key-value item to the map.
func (kl *KeyedList) AddItem() {
	if reflectx.IsNil(reflect.ValueOf(kl.Map)) {
		return
	}
	reflectx.MapAdd(kl.Map)
	kl.UpdateChange()
}

// DeleteItem deletes a key-value item from the map.
func (kl *KeyedList) DeleteItem(key reflect.Value) {
	if reflectx.IsNil(reflect.ValueOf(kl.Map)) {
		return
	}
	reflectx.MapDelete(kl.Map, reflectx.NonPointerValue(key))
	kl.UpdateChange()
}

func (kl *KeyedList) MakeToolbar(p *tree.Plan) {
	if reflectx.IsNil(reflect.ValueOf(kl.Map)) {
		return
	}
	tree.Add(p, func(w *Button) {
		w.SetText("Sort").SetIcon(icons.Sort).SetTooltip("Switch between sorting by the keys and the values").
			OnClick(func(e events.Event) {
				kl.toggleSort()
			})
	})
	if !kl.IsReadOnly() {
		tree.Add(p, func(w *Button) {
			w.SetText("Add").SetIcon(icons.Add).SetTooltip("Add a new element to the map").
				OnClick(func(e events.Event) {
					kl.AddItem()
				})
		})
	}
}

// bindMapKey is a version of [Bind] that works for keys in a map.
func bindMapKey[T Value](mapv reflect.Value, key reflect.Value, vw T) T {
	wb := vw.AsWidget()
	alreadyBound := wb.ValueUpdate != nil
	wb.ValueUpdate = func() {
		if vws, ok := any(vw).(ValueSetter); ok {
			ErrorSnackbar(vw, vws.SetWidgetValue(key.Interface()))
		} else {
			ErrorSnackbar(vw, reflectx.SetRobust(vw.WidgetValue(), key.Interface()))
		}
	}
	wb.ValueOnChange = func() {
		newKey := reflect.New(key.Type())
		ErrorSnackbar(vw, reflectx.SetRobust(newKey.Interface(), vw.WidgetValue()))
		newKey = newKey.Elem()
		if !mapv.MapIndex(newKey).IsValid() { // not already taken
			mapv.SetMapIndex(newKey, mapv.MapIndex(key))
			mapv.SetMapIndex(key, reflect.Value{})
			return
		}
		d := NewBody("Key already exists")
		NewText(d).SetType(TextSupporting).SetText(fmt.Sprintf("The key %q already exists", reflectx.ToString(newKey.Interface())))
		d.AddBottomBar(func(bar *Frame) {
			d.AddCancel(bar)
			d.AddOK(bar).SetText("Overwrite").OnClick(func(e events.Event) {
				mapv.SetMapIndex(newKey, mapv.MapIndex(key))
				mapv.SetMapIndex(key, reflect.Value{})
				wb.SendChange()
			})
		})
		d.RunDialog(vw)
	}
	if alreadyBound {
		ResetWidgetValue(vw)
	}
	if ob, ok := any(vw).(OnBinder); ok {
		ob.OnBind(key.Interface(), "")
	}
	wb.ValueUpdate() // we update it with the initial value immediately
	return vw
}

// bindMapValue is a version of [Bind] that works for values in a map.
func bindMapValue[T Value](mapv reflect.Value, key reflect.Value, vw T) T {
	wb := vw.AsWidget()
	alreadyBound := wb.ValueUpdate != nil
	wb.ValueUpdate = func() {
		value := mapv.MapIndex(key).Interface()
		if vws, ok := any(vw).(ValueSetter); ok {
			ErrorSnackbar(vw, vws.SetWidgetValue(value))
		} else {
			ErrorSnackbar(vw, reflectx.SetRobust(vw.WidgetValue(), value))
		}
	}
	wb.ValueOnChange = func() {
		value := reflectx.NonNilNew(mapv.Type().Elem())
		err := reflectx.SetRobust(value.Interface(), vw.WidgetValue())
		if err != nil {
			ErrorSnackbar(vw, err)
			return
		}
		mapv.SetMapIndex(key, value.Elem())
	}
	if alreadyBound {
		ResetWidgetValue(vw)
	}
	if ob, ok := any(vw).(OnBinder); ok {
		value := mapv.MapIndex(key).Interface()
		ob.OnBind(value, "")
	}
	wb.ValueUpdate() // we update it with the initial value immediately
	return vw
}
