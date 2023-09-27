// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"github.com/iancoleman/strcase"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/gist"
	"goki.dev/ki/v2"
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

func (av *ArgView) OnInit() {
	av.Lay = gi.LayoutVert
	av.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		av.Spacing = gi.StdDialogVSpaceUnits
		s.BackgroundColor.SetSolid(colors.Scheme.Background)
		s.Color = colors.Scheme.OnBackground
		s.MaxWidth.SetPx(-1)
		s.MaxHeight.SetPx(-1)
	})
}

func (av *ArgView) OnChildAdded(child ki.Ki) {
	if w := gi.AsWidget(child); w != nil {
		switch w.Name() {
		case "title":
			title := child.(*gi.Label)
			title.Type = gi.LabelTitleLarge
			title.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				s.MaxWidth.SetPx(-1)
				s.Text.Align = gist.AlignCenter
				s.AlignV = gist.AlignTop
			})
		case "args-grid":
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
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
		if w.Parent().Name() == "args-grid" {
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
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
func (av *ArgView) ConfigWidget(vp *Scene) {
	config := ki.TypeAndNameList{}
	config.Add(gi.LabelType, "title")
	config.Add(gi.FrameType, "args-grid")
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
	if laser.IfaceIsNil(av.Args) {
		return
	}
	sg := av.ArgsGrid()
	sg.Lay = gi.LayoutGrid
	sg.Stripes = gi.RowStripes
	config := ki.TypeAndNameList{}
	for i := range av.Args {
		ad := &av.Args[i]
		if ad.HasValSet() {
			continue
		}
		vtyp := ad.View.WidgetType()
		knm := strcase.ToKebab(ad.Name)
		labnm := "label-" + knm
		valnm := "value-" + knm
		config.Add(gi.LabelType, labnm)
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
		widg := sg.Child((i * 2) + 1).(gi.Node2D)
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
