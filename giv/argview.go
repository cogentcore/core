// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/iancoleman/strcase"
)

// ArgView represents a slice of reflect.Value's and associated names, for the
// purpose of supplying arguments to methods called via the MethView
// framework.
type ArgView struct {
	gi.Frame
	Args    []ArgData `desc:"the args that we are a view onto"`
	Title   string    `desc:"title / prompt to show above the editor fields"`
	ViewSig ki.Signal `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
}

var KiT_ArgView = kit.Types.AddType(&ArgView{}, ArgViewProps)

var ArgViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
	"#title": ki.Props{
		"max-width":      -1,
		"text-align":     gi.AlignCenter,
		"vertical-align": gi.AlignTop,
	},
}

// SetArgs sets the source args that we are viewing -- rebuilds the children
// to represent
func (av *ArgView) SetArgs(arg []ArgData) {
	updt := false
	updt = av.UpdateStart()
	av.Args = arg
	av.UpdateFromArgs()
	av.UpdateEnd(updt)
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (av *ArgView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "title")
	config.Add(gi.KiT_Frame, "args-grid")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (av *ArgView) StdConfig() (mods, updt bool) {
	av.Lay = gi.LayoutVert
	av.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := av.StdFrameConfig()
	mods, updt = av.ConfigChildren(config, false)
	return
}

// SetTitle sets the optional title and updates the Title label
func (av *ArgView) SetTitle(title string) {
	av.Title = title
	if av.Title != "" {
		lab, _ := av.TitleWidget()
		if lab != nil {
			lab.Text = title
		}
	}
}

// Title returns the title label widget, and its index, within frame -- nil,
// -1 if not found
func (av *ArgView) TitleWidget() (*gi.Label, int) {
	idx, ok := av.Children().IndexByName("title", 0)
	if !ok {
		return nil, -1
	}
	return av.KnownChild(idx).(*gi.Label), idx
}

// ArgsGrid returns the grid layout widget, which contains all the fields
// and values, and its index, within frame -- nil, -1 if not found
func (av *ArgView) ArgsGrid() (*gi.Frame, int) {
	idx, ok := av.Children().IndexByName("args-grid", 0)
	if !ok {
		return nil, -1
	}
	return av.KnownChild(idx).(*gi.Frame), idx
}

// ConfigArgsGrid configures the ArgsGrid for the current struct
func (av *ArgView) ConfigArgsGrid() {
	if kit.IfaceIsNil(av.Args) {
		return
	}
	sg, _ := av.ArgsGrid()
	if sg == nil {
		return
	}
	sg.Lay = gi.LayoutGrid
	sg.Stripes = gi.RowStripes
	// setting a pref here is key for giving it a scrollbar in larger context
	sg.SetMinPrefHeight(units.NewValue(1.5, units.Em))
	sg.SetMinPrefWidth(units.NewValue(10, units.Em))
	sg.SetStretchMaxHeight() // for this to work, ALL layers above need it too
	sg.SetStretchMaxWidth()  // for this to work, ALL layers above need it too
	sg.SetProp("columns", 2)
	config := kit.TypeAndNameList{}
	for i := range av.Args {
		ad := &av.Args[i]
		if ad.HasValSet() {
			continue
		}
		vtyp := ad.View.WidgetType()
		knm := strcase.ToKebab(ad.Name)
		labnm := fmt.Sprintf("label-%v", knm)
		valnm := fmt.Sprintf("value-%v", knm)
		config.Add(gi.KiT_Label, labnm)
		config.Add(vtyp, valnm)
	}
	mods, updt := sg.ConfigChildren(config, false)
	if mods {
		av.SetFullReRender()
	} else {
		updt = sg.UpdateStart()
	}
	for i := range av.Args {
		ad := &av.Args[i]
		if ad.HasValSet() {
			continue
		}
		lbl := sg.KnownChild(i * 2).(*gi.Label)
		vvb := ad.View.AsValueViewBase()
		vvb.ViewSig.ConnectOnly(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			avv, _ := recv.Embed(KiT_ArgView).(*ArgView)
			// note: updating here is redundant -- relevant field will have already updated
			avv.ViewSig.Emit(avv.This(), 0, nil)
		})
		lbl.Text = ad.Name
		lbl.Tooltip = ad.Desc
		widg := sg.KnownChild((i * 2) + 1).(gi.Node2D)
		widg.SetProp("horizontal-align", gi.AlignLeft)
		ad.View.ConfigWidget(widg)
	}
	sg.UpdateEnd(updt)
}

// UpdateFromArgs updates full widget layout from structure
func (av *ArgView) UpdateFromArgs() {
	mods, updt := av.StdConfig()
	av.ConfigArgsGrid()
	if mods {
		av.UpdateEnd(updt)
	}
}

// UpdateArgs updates each of the value-view widgets for the args
func (av *ArgView) UpdateArgs() {
	updt := av.UpdateStart()
	for i := range av.Args {
		ad := &av.Args[i]
		ad.View.UpdateWidget()
	}
	av.UpdateEnd(updt)
}
