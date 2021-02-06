// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// MapViewInline represents a map as a single line widget, for smaller maps
// and those explicitly marked inline -- constructs widgets in Parts to show
// the key names and editor vals for each value.
type MapViewInline struct {
	gi.PartsWidgetBase
	Map        interface{} `desc:"the map that we are a view onto"`
	MapValView ValueView   `desc:"ValueView for the map itself, if this was created within value view framework -- otherwise nil"`
	Changed    bool        `desc:"has the map been edited?"`
	Keys       []ValueView `json:"-" xml:"-" desc:"ValueView representations of the map keys"`
	Values     []ValueView `json:"-" xml:"-" desc:"ValueView representations of the fields"`
	TmpSave    ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig    ki.Signal   `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
	ViewPath   string      `desc:"a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows"`
}

var KiT_MapViewInline = kit.Types.AddType(&MapViewInline{}, MapViewInlineProps)

func (mv *MapViewInline) Disconnect() {
	mv.PartsWidgetBase.Disconnect()
	mv.ViewSig.DisconnectAll()
}

var MapViewInlineProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
	"min-width":     units.NewEx(60),
}

// SetMap sets the source map that we are viewing -- rebuilds the children to represent this map
func (mv *MapViewInline) SetMap(mp interface{}) {
	// note: because we make new maps, and due to the strangeness of reflect, they
	// end up not being comparable types, so we can't check if equal
	mv.Map = mp
	mv.UpdateFromMap()
}

// ConfigParts configures Parts for the current map
func (mv *MapViewInline) ConfigParts() {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	mv.Parts.Lay = gi.LayoutHoriz
	mv.Parts.SetProp("overflow", gist.OverflowHidden) // no scrollbars!
	config := kit.TypeAndNameList{}
	// always start fresh!
	mv.Keys = make([]ValueView, 0)
	mv.Values = make([]ValueView, 0)

	mpv := reflect.ValueOf(mv.Map)
	mpvnp := kit.NonPtrValue(mpv)

	keys := mpvnp.MapKeys() // this is a slice of reflect.Value
	kit.ValueSliceSort(keys, true)
	for i, key := range keys {
		if i >= MapInlineLen {
			break
		}
		kv := ToValueView(key.Interface(), "")
		if kv == nil { // shouldn't happen
			continue
		}
		kv.SetMapKey(key, mv.Map, mv.TmpSave)

		val := kit.OnePtrUnderlyingValue(mpvnp.MapIndex(key))
		vv := ToValueView(val.Interface(), "")
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetMapValue(val, mv.Map, key.Interface(), kv, mv.TmpSave, mv.ViewPath) // needs key value view to track updates

		keytxt := kit.ToString(key.Interface())
		keynm := fmt.Sprintf("key-%v", keytxt)
		valnm := fmt.Sprintf("value-%v", keytxt)

		config.Add(kv.WidgetType(), keynm)
		config.Add(vv.WidgetType(), valnm)
		mv.Keys = append(mv.Keys, kv)
		mv.Values = append(mv.Values, vv)
	}
	config.Add(gi.KiT_Action, "add-action")
	config.Add(gi.KiT_Action, "edit-action")
	mods, updt := mv.Parts.ConfigChildren(config)
	if !mods {
		updt = mv.Parts.UpdateStart()
	}
	for i, vv := range mv.Values {
		vvb := vv.AsValueViewBase()
		vvb.ViewSig.ConnectOnly(mv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			mvv, _ := recv.Embed(KiT_MapViewInline).(*MapViewInline)
			mvv.SetChanged()
		})
		keyw := mv.Parts.Child(i * 2).(gi.Node2D)
		widg := mv.Parts.Child((i * 2) + 1).(gi.Node2D)
		kv := mv.Keys[i]
		kv.ConfigWidget(keyw)
		vv.ConfigWidget(widg)
		if mv.IsInactive() {
			widg.AsNode2D().SetInactive()
			keyw.AsNode2D().SetInactive()
		}
	}
	adack, err := mv.Parts.Children().ElemFromEndTry(1)
	if err == nil {
		adac := adack.(*gi.Action)
		adac.SetIcon("plus")
		adac.Tooltip = "add an entry to the map"
		adac.ActionSig.ConnectOnly(mv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			mvv, _ := recv.Embed(KiT_MapViewInline).(*MapViewInline)
			mvv.MapAdd()
		})
	}
	edack, err := mv.Parts.Children().ElemFromEndTry(0)
	if err == nil {
		edac := edack.(*gi.Action)
		edac.SetIcon("edit")
		edac.Tooltip = "map edit dialog"
		edac.ActionSig.ConnectOnly(mv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			mvv, _ := recv.Embed(KiT_MapViewInline).(*MapViewInline)
			vpath := mvv.ViewPath
			title := ""
			if mvv.MapValView != nil {
				newPath := ""
				isZero := false
				title, newPath, isZero = mvv.MapValView.AsValueViewBase().Label()
				if isZero {
					return
				}
				vpath = mvv.ViewPath + "/" + newPath
			} else {
				tmptyp := kit.NonPtrType(reflect.TypeOf(mvv.Map))
				title = "Map of " + tmptyp.String()
				// if tynm == "" {
				// 	tynm = tmptyp.String()
				// }
			}
			dlg := MapViewDialog(mvv.Viewport, mvv.Map, DlgOpts{Title: title, Prompt: mvv.Tooltip, TmpSave: mvv.TmpSave, ViewPath: vpath}, nil, nil)
			mvvvk := dlg.Frame().ChildByType(KiT_MapView, ki.Embeds, 2)
			if mvvvk != nil {
				mvvv := mvvvk.(*MapView)
				mvvv.MapValView = mvv.MapValView
				mvvv.ViewSig.ConnectOnly(mvv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
					mvvvv, _ := recv.Embed(KiT_MapViewInline).(*MapViewInline)
					mvvvv.ViewSig.Emit(mvvvv.This(), 0, nil)
				})
			}
		})
	}
	mv.Parts.UpdateEnd(updt)
}

// SetChanged sets the Changed flag and emits the ViewSig signal for the
// SliceView, indicating that some kind of edit / change has taken place to
// the table data.  It isn't really practical to record all the different
// types of changes, so this is just generic.
func (mv *MapViewInline) SetChanged() {
	mv.Changed = true
	mv.ViewSig.Emit(mv.This(), 0, nil)
}

// MapAdd adds a new entry to the map
func (mv *MapViewInline) MapAdd() {
	if kit.IfaceIsNil(mv.Map) {
		return
	}
	updt := mv.UpdateStart()
	defer mv.UpdateEnd(updt)

	kit.MapAdd(mv.Map)

	if mv.TmpSave != nil {
		mv.TmpSave.SaveTmp()
	}
	mv.SetChanged()
	mv.SetFullReRender()
	mv.UpdateFromMap()
}

func (mv *MapViewInline) UpdateFromMap() {
	mv.ConfigParts()
}

func (mv *MapViewInline) UpdateValues() {
	// maps have to re-read their values because they can't get pointers!
	mv.ConfigParts()
}

func (mv *MapViewInline) Style2D() {
	mv.ConfigParts()
	mv.PartsWidgetBase.Style2D()
}

func (mv *MapViewInline) Render2D() {
	if mv.FullReRenderIfNeeded() {
		return
	}
	if mv.PushBounds() {
		mv.Render2DParts()
		mv.Render2DChildren()
		mv.PopBounds()
	}
}
