// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"reflect"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// MapView represents a map, creating a property editor of the values --
// constructs Children widgets to show the key / value pairs, within an
// overall frame.
type MapView struct {
	core.Frame

	// the map that we are a view onto
	Map any `set:"-"`

	// Value for the map itself, if this was created within value view framework; otherwise nil
	MapValView Value

	// has the map been edited?
	Changed bool `set:"-"`

	// Value representations of the map keys
	Keys []Value `json:"-" xml:"-" set:"-"`

	// Value representations of the map values
	Values []Value `json:"-" xml:"-" set:"-"`

	// sort by values instead of keys
	SortVals bool

	// the number of columns in the map; do not set externally;
	// generally only access internally
	NCols int `set:"-"`

	// a record of parent View names that have led up to this view.
	// Displayed as extra contextual information in view dialog windows.
	ViewPath string
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
				s.Columns = mv.NCols
				s.Overflow.Set(styles.OverflowAuto)
				s.Grow.Set(1, 1)
				s.Min.X.Em(20)
				s.Min.Y.Em(10)
			})
		}
		// if w.Parent().Name() == "map-grid" {
		// }
	})
}

// SetMap sets the source map that we are viewing.
// Rebuilds the children to represent this map
func (mv *MapView) SetMap(mp any) *MapView {
	// note: because we make new maps, and due to the strangeness of reflect, they
	// end up not being comparable types, so we can't check if equal
	mv.Map = mp
	mv.Update()
	return mv
}

// UpdateValues updates the widget display of slice values, assuming same slice config
func (mv *MapView) UpdateValues() {
	// maps have to re-read their values -- can't get pointers
	mv.Update()
}

// Config configures the view
func (mv *MapView) Config() {
	if !mv.HasChildren() {
		core.NewFrame(mv, "map-grid")
	}
	mv.ConfigMapGrid()
}

// IsConfiged returns true if the widget is fully configured
func (mv *MapView) IsConfiged() bool {
	return len(mv.Kids) != 0
}

// MapGrid returns the MapGrid grid layout widget, which contains all the fields and values
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
	if laser.AnyIsNil(mv.Map) {
		return
	}
	sg := mv.MapGrid()
	config := tree.Config{}
	// always start fresh!
	mv.Keys = make([]Value, 0)
	mv.Values = make([]Value, 0)
	sg.DeleteChildren()

	mpv := reflect.ValueOf(mv.Map)
	mpvnp := laser.NonPtrValue(mpv)

	valtyp := laser.NonPtrType(reflect.TypeOf(mv.Map)).Elem()
	ncol := 2
	ifaceType := false
	if valtyp.Kind() == reflect.Interface && valtyp.String() == "interface {}" {
		ifaceType = true
		ncol = 3
		// todo: need some way of setting & getting
		// this for given domain mapview could have a structview parent and
		// the source node of that struct, if a Ki, could have a property --
		// unlike inline case, plain mapview is not a child of struct view
		// directly -- but field on struct view does create the mapview
		// dialog.. a bit hacky and indirect..
	}

	// valtypes := append(kit.Types.AllTagged(typeTag), kit.Enums.AllTagged(typeTag)...)
	// valtypes = append(valtypes, kit.Types.AllTagged("basic-type")...)
	// valtypes = append(valtypes, kit.TypeFor[reflect.Type]())
	valtypes := gti.AllEmbeddersOf(tree.NodeBaseType) // todo: this is not right

	mv.NCols = ncol

	keys := laser.MapSort(mv.Map, !mv.SortVals, true) // note: this is a slice of reflect.Value!
	for _, key := range keys {
		kv := ToValue(key.Interface(), "")
		if kv == nil { // shouldn't happen
			continue
		}
		kv.SetMapKey(key, mv.Map)

		val := laser.OnePtrUnderlyingValue(mpvnp.MapIndex(key))
		vv := ToValue(val.Interface(), "")
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetMapValue(val, mv.Map, key.Interface(), kv, mv.ViewPath) // needs key value value to track updates

		keytxt := laser.ToString(key.Interface())
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
		Config(kv, keyw)
		Config(vv, w)
		vv.AsWidgetBase().OnInput(mv.HandleEvent)
		kv.AsWidgetBase().OnInput(mv.HandleEvent)
		w.Style(func(s *styles.Style) {
			s.SetTextWrap(false)
		})
		keyw.Style(func(s *styles.Style) {
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
			vtyp := laser.NonPtrType(reflect.TypeOf(vv.Val().Interface()))
			if vtyp == nil {
				vtyp = reflect.TypeOf("") // default to string
			}
			typw.SetCurrentValue(vtyp)
		}
	}
}

// SetChanged sets the Changed flag and emits the ViewSig signal for the
// MapView, indicating that some kind of edit / change has taken place to
// the table data.
func (mv *MapView) SetChanged() {
	mv.Changed = true
	mv.SendChange()
}

// MapChangeValueType changes the type of the value for given map element at
// idx -- for maps with any values
func (mv *MapView) MapChangeValueType(idx int, typ reflect.Type) {
	if laser.AnyIsNil(mv.Map) {
		return
	}

	keyv := mv.Keys[idx]
	ck := laser.NonPtrValue(keyv.Val()) // current key value
	valv := mv.Values[idx]
	cv := laser.NonPtrValue(valv.Val()) // current val value

	// create a new item of selected type, and attempt to convert existing to it
	var evn reflect.Value
	if cv.IsZero() {
		evn = laser.MakeOfType(typ)
	} else {
		evn = laser.CloneToType(typ, cv.Interface())
	}
	ov := laser.NonPtrValue(reflect.ValueOf(mv.Map))
	valv.AsValueData().Value = evn.Elem()
	ov.SetMapIndex(ck, evn.Elem())
	mv.ConfigMapGrid()
	mv.SetChanged()
	mv.NeedsRender()
}

// ToggleSort toggles sorting by values vs. keys
func (mv *MapView) ToggleSort() {
	mv.SortVals = !mv.SortVals
	mv.ConfigMapGrid()
}

// MapAdd adds a new entry to the map
func (mv *MapView) MapAdd() {
	if laser.AnyIsNil(mv.Map) {
		return
	}
	laser.MapAdd(mv.Map)

	mv.SetChanged()
	mv.Update()
}

// MapDelete deletes a key-value from the map
func (mv *MapView) MapDelete(key reflect.Value) {
	if laser.AnyIsNil(mv.Map) {
		return
	}
	laser.MapDeleteValue(mv.Map, laser.NonPtrValue(key))

	mv.SetChanged()
	mv.Update()
}

// ConfigToolbar configures a [core.Toolbar] for this view
func (mv *MapView) ConfigToolbar(tb *core.Toolbar) {
	if laser.AnyIsNil(mv.Map) {
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
