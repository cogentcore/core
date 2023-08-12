// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/iancoleman/strcase"
)

// ArgView represents a slice of reflect.Value's and associated names, for the
// purpose of supplying arguments to methods called via the MethView
// framework.
type ArgView struct {
	gi.Frame

	// the args that we are a view onto
	Args []ArgData `desc:"the args that we are a view onto"`

	// title / prompt to show above the editor fields
	Title string `desc:"title / prompt to show above the editor fields"`

	// signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update
	ViewSig ki.Signal `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string `desc:"a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows"`
}

var TypeArgView = kit.Types.AddType(&ArgView{}, ArgViewProps)

func (av *ArgView) OnInit() {
	av.AddStyleFunc(gi.StyleFuncDefault, func() {
		s := &av.Style
		s.BackgroundColor.SetColor(gi.ColorScheme.Background)
		s.Color = gi.ColorScheme.OnBackground
		s.MaxWidth.SetPx(-1)
		s.MaxHeight.SetPx(-1)
	})
}

func (av *ArgView) OnChildAdded(child ki.Ki) {
	switch child.Name() {
	case "title":
		title := child.(*gi.Label)
		title.Type = gi.LabelTitleLarge
		av.AddStyleFunc(gi.StyleFuncParent(av), func() {
			s := &title.Style
			s.MaxWidth.SetPx(-1)
			s.Text.Align = gist.AlignCenter
			s.AlignV = gist.AlignTop
		})
	case "args-grid":
		grid := child.(*gi.Layout)
		grid.AddStyleFunc(gi.StyleFuncParent(av), func() {
			s := &grid.Style
			// setting a pref here is key for giving it a scrollbar in larger context
			s.MinWidth.SetEm(1.5)
			s.Width.SetEm(1.5)
			s.MaxWidth.SetPx(-1) // for this to work, ALL layers above need it too
			s.MinHeight.SetEm(10)
			s.Height.SetEm(10)
			s.MaxHeight.SetPx(-1)            // for this to work, ALL layers above need it too
			s.Overflow = gist.OverflowScroll // this still gives it true size during PrefSize
			s.Columns = 2
		})
	}
	if child.Parent().Name() == "args-grid" {
		if widg, ok := child.(*gi.WidgetBase); ok {
			widg.AddStyleFunc(gi.StyleFuncParent(av), func() {
				s := &widg.Style
				s.AlignH = gist.AlignCenter
			})
		}
	}
}

func (av *ArgView) Disconnect() {
	av.Frame.Disconnect()
	av.ViewSig.DisconnectAll()
}

var ArgViewProps = ki.Props{
	ki.EnumTypeFlag: gi.TypeNodeFlags,
}

// SetArgs sets the source args that we are viewing -- rebuilds the children
// to represent
func (av *ArgView) SetArgs(arg []ArgData) {
	updt := false
	updt = av.UpdateStart()
	av.Args = arg
	av.Config()
	av.UpdateEnd(updt)
}

// Config configures the view
func (av *ArgView) Config() {
	av.Lay = gi.LayoutVert
	av.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.TypeAndNameList{}
	config.Add(gi.TypeLabel, "title")
	config.Add(gi.TypeFrame, "args-grid")
	mods, updt := av.ConfigChildren(config)
	av.ConfigArgsGrid()
	if mods {
		av.UpdateEnd(updt)
	}
}

// Title returns the title label widget, and its index, within frame
func (av *ArgView) TitleWidget() *gi.Label {
	return av.ChildByName("title", 0).(*gi.Label)
}

// ArgsGrid returns the grid layout widget, which contains all the fields
// and values, and its index, within frame
func (av *ArgView) ArgsGrid() *gi.Frame {
	return av.ChildByName("args-grid", 0).(*gi.Frame)
}

// SetTitle sets the optional title and updates the Title label
func (av *ArgView) SetTitle(title string) {
	av.Title = title
	if av.Title != "" {
		lab := av.TitleWidget()
		if lab != nil {
			lab.Text = title
		}
	}
}

// ConfigArgsGrid configures the ArgsGrid for the current struct
func (av *ArgView) ConfigArgsGrid() {
	if kit.IfaceIsNil(av.Args) {
		return
	}
	sg := av.ArgsGrid()
	sg.Lay = gi.LayoutGrid
	sg.Stripes = gi.RowStripes
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
		config.Add(gi.TypeLabel, labnm)
		config.Add(vtyp, valnm)
	}
	mods, updt := sg.ConfigChildren(config) // not sure if always unique?
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
		lbl := sg.Child(i * 2).(*gi.Label)
		vvb := ad.View.AsValueViewBase()
		vvb.ViewSig.ConnectOnly(av.This(), func(recv, send ki.Ki, sig int64, data any) {
			avv, _ := recv.Embed(TypeArgView).(*ArgView)
			// note: updating here is redundant -- relevant field will have already updated
			avv.ViewSig.Emit(avv.This(), 0, nil)
		})
		lbl.Text = ad.Name
		lbl.Tooltip = ad.Desc
		widg := sg.Child((i * 2) + 1).(*gi.WidgetBase)
		ad.View.ConfigWidget(widg)
	}
	sg.UpdateEnd(updt)
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
