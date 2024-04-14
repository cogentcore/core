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
)

// MapViewInline represents a map as a single line widget,
// for smaller maps and those explicitly marked inline.
type MapViewInline struct {
	core.Layout

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
	if reflectx.AnyIsNil(mv.Map) {
		mv.ConfigSize = 0
		return
	}
	config := tree.Config{}
	mv.Keys = make([]Value, 0)
	mv.Values = make([]Value, 0)

	mpv := reflect.ValueOf(mv.Map)
	mpvnp := reflectx.NonPointerValue(reflectx.OnePointerUnderlyingValue(mpv))
	keys := mpvnp.MapKeys() // this is a slice of reflect.Value
	mv.ConfigSize = len(keys)

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
	if reflectx.AnyIsNil(mv.Map) {
		return
	}
	reflectx.MapAdd(mv.Map)

	mv.SetChanged()
	mv.Update()
}

// MapDelete deletes a key-value from the map
func (mv *MapViewInline) MapDelete(key reflect.Value) {
	if reflectx.AnyIsNil(mv.Map) {
		return
	}
	reflectx.MapDelete(mv.Map, reflectx.NonPointerValue(key))

	mv.SetChanged()
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
		return mv.ConfigSize != 0
	}
	mpv := reflect.ValueOf(mv.Map)
	mpvnp := reflectx.NonPointerValue(reflectx.OnePointerUnderlyingValue(mpv))
	keys := mpvnp.MapKeys()
	return mv.ConfigSize != len(keys)
}

func (mv *MapViewInline) SizeUp() {
	if mv.MapSizeChanged() {
		mv.Update()
	}
	mv.Layout.SizeUp()
}
