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
	"cogentcore.org/core/types"
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
	mpv := reflect.ValueOf(mv.Map)
	mpvnp := reflectx.NonPointerValue(reflectx.OnePointerUnderlyingValue(mpv)) // from inline
	// mpvnp := reflectx.NonPointerValue(mpv) // original

	valtyp := reflectx.NonPointerType(reflect.TypeOf(mv.Map)).Elem()
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
	valtypes := types.AllEmbeddersOf(tree.NodeBaseType) // todo: this is not right

	mv.ncols = ncol

	keys := reflectx.MapSort(mv.Map, !mv.SortValues, true) // note: this is a slice of reflect.Value!
	for _, key := range keys {
		keytxt := reflectx.ToString(key.Interface())
		keynm := "key-" + keytxt
		valnm := "value-" + keytxt
		val := reflectx.OnePointerUnderlyingValue(mpvnp.MapIndex(key))

		core.Configure(c, keynm, func() core.Value {
			w := core.NewValue(key.Interface(), "")
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
			w := core.NewValue(val.Interface(), "")
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

		if ifaceType {
			typnm := "type-" + keytxt
			core.Configure(c, typnm, func() *core.Chooser {
				w := core.NewChooser()
				w.SetTypes(valtypes...)
				vtyp := reflectx.NonPointerType(reflect.TypeOf(val.Interface()))
				if vtyp == nil {
					vtyp = reflect.TypeOf("") // default to string
				}
				w.SetCurrentValue(vtyp)
				return w
			})
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
func (mv *MapView) ConfigToolbar(tb *core.Toolbar) {
	if reflectx.AnyIsNil(mv.Map) {
		return
	}
	core.NewButton(tb).SetText("Sort").SetIcon(icons.Sort).SetTooltip("Switch between sorting by the keys vs. the values").
		OnClick(func(e events.Event) {
			mv.ToggleSort()
		})
	if !mv.IsReadOnly() {
		core.NewButton(tb).SetText("Add").SetIcon(icons.Add).SetTooltip("Add a new element to the map").
			OnClick(func(e events.Event) {
				mv.MapAdd()
			})
	}
}
