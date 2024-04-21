// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"reflect"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/reflectx"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// MapView represents a map using two columns of editable key and value widgets.
type MapView struct {
	core.Frame

	// Map is the pointer to the map that we are viewing.
	Map any

	// Keys are [Value] representations of the map keys.
	Keys []Value `json:"-" xml:"-" set:"-"`

	// Values are [Value] representations of the map values.
	Values []Value `json:"-" xml:"-" set:"-"`

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
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	mv.OnWidgetAdded(func(w core.Widget) {
		switch w.PathFrom(mv) {
		case "map-grid":
			w.Style(func(s *styles.Style) {
				s.Display = styles.Grid
				s.Columns = mv.ncols
				s.Overflow.Set(styles.OverflowAuto)
				s.Grow.Set(1, 1)
				s.Min.X.Em(20)
				s.Min.Y.Em(10)
			})
		}
	})
}

func (mv *MapView) Config() {
	if !mv.HasChildren() {
		core.NewFrame(mv, "map-grid")
	}
	mv.ConfigMapGrid()
}

// MapGrid returns the map grid frame widget, which contains all the fields and values.
func (mv *MapView) MapGrid() *core.Frame {
	return mv.ChildByName("map-grid", 0).(*core.Frame)
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

// ConfigMapGrid configures the MapGrid for the current map
func (mv *MapView) ConfigMapGrid() {
	if reflectx.AnyIsNil(mv.Map) {
		return
	}
	sg := mv.MapGrid()
	config := tree.Config{}
	// always start fresh!
	mv.Keys = make([]Value, 0)
	mv.Values = make([]Value, 0)
	sg.DeleteChildren()

	mpv := reflect.ValueOf(mv.Map)
	mpvnp := reflectx.NonPointerValue(mpv)

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
		vv.SetMapValue(val, mv.Map, key.Interface(), kv, mv.ViewPath) // needs key value value to track updates

		keytxt := reflectx.ToString(key.Interface())
		keynm := "key-" + keytxt
		valnm := "value-" + keytxt

		config.Add(kv.WidgetType(), keynm)
		config.Add(vv.WidgetType(), valnm)
		if ifaceType {
			typnm := "type-" + keytxt
			config.Add(core.ChooserType, typnm)
		}
		mv.Keys = append(mv.Keys, kv)
		mv.Values = append(mv.Values, vv)
	}
	if sg.ConfigChildren(config) {
		sg.NeedsLayout()
	}
	for i, vv := range mv.Values {
		kv := mv.Keys[i]
		vv.OnChange(func(e events.Event) { mv.SendChange(e) })
		kv.OnChange(func(e events.Event) {
			mv.SendChange(e)
			mv.Update()
		})
		keyw := sg.Child(i * ncol).(core.Widget)
		w := sg.Child(i*ncol + 1).(core.Widget)
		keyw.AsWidget().SetReadOnly(mv.IsReadOnly())
		w.AsWidget().SetReadOnly(mv.IsReadOnly())
		Config(kv, keyw)
		Config(vv, w)
		vv.AsWidgetBase().OnInput(mv.HandleEvent)
		kv.AsWidgetBase().OnInput(mv.HandleEvent)
		w.Style(func(s *styles.Style) {
			s.SetReadOnly(mv.IsReadOnly())
			s.SetTextWrap(false)
		})
		keyw.Style(func(s *styles.Style) {
			s.SetReadOnly(mv.IsReadOnly())
			s.SetTextWrap(false)
		})
		w.AddContextMenu(func(m *core.Scene) {
			mv.ContextMenu(m, kv.Val())
		})
		keyw.AddContextMenu(func(m *core.Scene) {
			mv.ContextMenu(m, kv.Val())
		})
		if ifaceType {
			typw := sg.Child(i*ncol + 2).(*core.Chooser)
			typw.SetTypes(valtypes...)
			vtyp := reflectx.NonPointerType(reflect.TypeOf(vv.Val().Interface()))
			if vtyp == nil {
				vtyp = reflect.TypeOf("") // default to string
			}
			typw.SetCurrentValue(vtyp)
		}
	}
}

// ToggleSort toggles sorting by values vs. keys
func (mv *MapView) ToggleSort() {
	mv.SortValues = !mv.SortValues
	mv.ConfigMapGrid()
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
	core.NewButton(tb, "sort").SetText("Sort").SetIcon(icons.Sort).SetTooltip("Switch between sorting by the keys vs. the values").
		OnClick(func(e events.Event) {
			mv.ToggleSort()
		})
	if !mv.IsReadOnly() {
		core.NewButton(tb, "add").SetText("Add").SetIcon(icons.Add).SetTooltip("Add a new element to the map").
			OnClick(func(e events.Event) {
				mv.MapAdd()
			})
	}
}
