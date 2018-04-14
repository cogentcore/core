// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"reflect"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  MapView

// MapView represents a map, creating a property editor of the values -- constructs Children widgets to show the key / value pairs, within an overall frame with an optional title, and a button box at the bottom where methods can be invoked
type MapView struct {
	Frame
	Map    interface{} `desc:"the map that we are a view onto"`
	Title  string      `desc:"title / prompt to show above the editor fields"`
	Values []ValueView `desc:"ValueView representations of the map values"`
}

var KiT_MapView = kit.Types.AddType(&MapView{}, nil)

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// SetMap sets the source map that we are viewing -- rebuilds the children to represent this map
func (sv *MapView) SetMap(mp interface{}) {
	sv.UpdateStart()
	sv.Map = mp
	sv.UpdateFromMap()
	sv.UpdateEnd()
}

var MapViewProps = map[string]interface{}{
	"#frame": map[string]interface{}{
		"border-width":        units.NewValue(2, units.Px),
		"margin":              units.NewValue(8, units.Px),
		"padding":             units.NewValue(4, units.Px),
		"box-shadow.h-offset": units.NewValue(4, units.Px),
		"box-shadow.v-offset": units.NewValue(4, units.Px),
		"box-shadow.blur":     units.NewValue(4, units.Px),
		"box-shadow.color":    "#CCC",
	},
	"#title": map[string]interface{}{
		// todo: add "bigger" font
		"max-width":        units.NewValue(-1, units.Px),
		"text-align":       AlignCenter,
		"vertical-align":   AlignTop,
		"background-color": "none",
	},
	"#prompt": map[string]interface{}{
		"max-width":        units.NewValue(-1, units.Px),
		"text-align":       AlignLeft,
		"vertical-align":   AlignTop,
		"background-color": "none",
	},
}

// SetFrame configures view as a frame
func (sv *MapView) SetFrame() {
	sv.Lay = LayoutCol
	sv.PartStyleProps(sv, MapViewProps)
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *MapView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	config.Add(KiT_Label, "Title")
	config.Add(KiT_Space, "TitleSpace")
	config.Add(KiT_Layout, "MapGrid")
	config.Add(KiT_Space, "GridSpace")
	config.Add(KiT_Layout, "ButtonBox")
	return config
}

// StdConfig configures a standard setup of the overall Frame
func (sv *MapView) StdConfig() {
	sv.SetFrame()
	config := sv.StdFrameConfig()
	sv.ConfigChildren(config, false)
}

// SetTitle sets the title and updates the Title label
func (sv *MapView) SetTitle(title string) {
	sv.Title = title
	lab, _ := sv.TitleWidget()
	if lab != nil {
		lab.Text = title
	}
}

// Title returns the title label widget, and its index, within frame -- nil, -1 if not found
func (sv *MapView) TitleWidget() (*Label, int) {
	idx := sv.ChildIndexByName("Title", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Label), idx
}

// MapGrid returns the MapGrid grid layout widget, which contains all the fields and values, and its index, within frame -- nil, -1 if not found
func (sv *MapView) MapGrid() (*Layout, int) {
	idx := sv.ChildIndexByName("MapGrid", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (sv *MapView) ButtonBox() (*Layout, int) {
	idx := sv.ChildIndexByName("ButtonBox", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ConfigMapGrid configures the MapGrid for the current map
func (sv *MapView) ConfigMapGrid() {
	if kit.IsNil(sv.Map) {
		return
	}
	sg, _ := sv.MapGrid()
	if sg == nil {
		return
	}
	sg.Lay = LayoutGrid
	sg.SetProp("columns", 2)
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// always start fresh!
	sv.Values = make([]ValueView, 0)

	mv := reflect.ValueOf(sv.Map)
	mvnp := kit.NonPtrValue(mv)

	// todo: could sort keys
	keys := mvnp.MapKeys()
	for _, key := range keys {
		val := mvnp.MapIndex(key)
		vv := ToValueView(val.Interface())
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetMapValue(val, sv.Map, key.Interface())
		vtyp := vv.WidgetType()
		lbltxt := kit.ToString(key.Interface())
		labnm := fmt.Sprintf("Lbl%v", lbltxt)
		valnm := fmt.Sprintf("Val%v", lbltxt)
		config.Add(KiT_Label, labnm)
		config.Add(vtyp, valnm)
		sv.Values = append(sv.Values, vv)
	}
	updt := sg.ConfigChildren(config, false)
	if updt {
		bitflag.Set(&sv.Flag, int(NodeFlagFullReRender))
	}
	for i, vv := range sv.Values {
		lbl := sg.Child(i * 2).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		vvb := vv.AsValueViewBase()
		lbltxt := kit.ToString(vvb.Key)
		lbl.Text = lbltxt
		widg := sg.Child((i * 2) + 1).(Node2D)
		widg.SetProp("vertical-align", AlignMiddle)
		vv.ConfigWidget(widg)
	}
}

func (sv *MapView) UpdateFromMap() {
	sv.StdConfig()
	typ := kit.NonPtrType(reflect.TypeOf(sv.Map))
	sv.SetTitle(fmt.Sprintf("%v Values", typ.Name()))
	sv.ConfigMapGrid()
}

func (sv *MapView) Render2D() {
	bitflag.Clear(&sv.Flag, int(NodeFlagFullReRender))
	sv.Frame.Render2D()
}

func (sv *MapView) ReRender2D() (node Node2D, layout bool) {
	if bitflag.Has(sv.Flag, int(NodeFlagFullReRender)) {
		node = nil
		layout = false
	} else {
		node = sv.This.(Node2D)
		layout = true
	}
	return
}

// check for interface implementation
var _ Node2D = &MapView{}

////////////////////////////////////////////////////////////////////////////////////////
//  MapViewInline

// MapViewInline represents a map as a single line widget, for smaller maps and those explicitly marked inline -- constructs widgets in Parts to show the key names and editor vals for each value
type MapViewInline struct {
	WidgetBase
	Map        interface{} `desc:"the map that we are a view onto"`
	MapViewSig ki.Signal   `json:"-" desc:"signal for MapView -- see MapViewSignals for the types"`
	Values     []ValueView `desc:"ValueView representations of the fields"`
}

var KiT_MapViewInline = kit.Types.AddType(&MapViewInline{}, nil)

// SetMap sets the source map that we are viewing -- rebuilds the children to represent this map
func (sv *MapViewInline) SetMap(st interface{}) {
	sv.UpdateStart()
	sv.Map = st
	sv.UpdateFromMap()
	sv.UpdateEnd()
}

var MapViewInlineProps = map[string]interface{}{}

// ConfigParts configures Parts for the current map
func (sv *MapViewInline) ConfigParts() {
	if kit.IsNil(sv.Map) {
		return
	}
	sv.Parts.Lay = LayoutRow
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// always start fresh!
	sv.Values = make([]ValueView, 0)

	mv := reflect.ValueOf(sv.Map)
	mvnp := kit.NonPtrValue(mv)

	keys := mvnp.MapKeys()
	for _, key := range keys {
		val := mvnp.MapIndex(key)
		vv := ToValueView(val.Interface())
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetMapValue(val, sv.Map, key.Interface())
		vtyp := vv.WidgetType()
		lbltxt := kit.ToString(key.Interface())
		labnm := fmt.Sprintf("Lbl%v", lbltxt)
		valnm := fmt.Sprintf("Val%v", lbltxt)
		config.Add(KiT_Label, labnm)
		config.Add(vtyp, valnm)
		sv.Values = append(sv.Values, vv)
	}
	config.Add(KiT_Action, "EditAction")
	sv.Parts.ConfigChildren(config, false)
	for i, vv := range sv.Values {
		lbl := sv.Parts.Child(i * 2).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		vvb := vv.AsValueViewBase()
		lbltxt := kit.ToString(vvb.Key)
		lbl.Text = lbltxt
		widg := sv.Parts.Child((i * 2) + 1).(Node2D)
		widg.SetProp("vertical-align", AlignMiddle)
		vv.ConfigWidget(widg)
	}
	edac := sv.Parts.Child(-1).(*Action)
	edac.SetProp("vertical-align", AlignMiddle)
	edac.Text = "  ..."
	edac.ActionSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.EmbeddedStruct(KiT_MapViewInline).(*MapViewInline)
		MapViewDialog(svv.Viewport, svv.Map, "Map Value View", "", nil, nil)
	})
}

func (sv *MapViewInline) UpdateFromMap() {
	sv.ConfigParts()
}

func (sv *MapViewInline) Render2D() {
	if sv.PushBounds() {
		sv.Render2DParts()
		sv.Render2DChildren()
		sv.PopBounds()
	}
}

// todo: see notes on treeview
func (sv *MapViewInline) ReRender2D() (node Node2D, layout bool) {
	node = sv.This.(Node2D)
	layout = true
	return
}

// check for interface implementation
var _ Node2D = &MapViewInline{}
