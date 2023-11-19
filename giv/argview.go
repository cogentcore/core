// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"github.com/iancoleman/strcase"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

// ArgView represents a slice of reflect.Value's and associated names, for the
// purpose of supplying arguments to methods called via the MethodView
// framework.
type ArgView struct {
	gi.Frame

	// title / prompt to show above the editor fields
	Title string

	// the args that we are a view onto
	Args []Value

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string
}

func (av *ArgView) OnInit() {
	av.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	av.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(av) {
		case "title":
			title := w.(*gi.Label)
			title.Type = gi.LabelTitleLarge
			title.Style(func(s *styles.Style) {
				s.Grow.Set(1, 0)
				s.Text.Align = styles.AlignCenter
				s.Align.Y = styles.AlignStart
			})
		case "args-grid":
			w.Style(func(s *styles.Style) {
				s.Display = styles.Grid
				s.Columns = 2
				s.Min.X.Ch(20)
				s.Min.Y.Em(10)
				s.Grow.Set(1, 1)
				s.Overflow.Set(styles.OverflowAuto)
			})
		}
		if w.Parent().Name() == "args-grid" {
			w.Style(func(s *styles.Style) {
				s.Align.X = styles.AlignCenter
			})
		}
	})
}

// Config configures the view
func (av *ArgView) ConfigWidget(vp *gi.Scene) {
	config := ki.Config{}
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

// // SetTitle sets the optional title and updates the Title label
// func (av *ArgView) SetTitle(title string) {
// 	av.Title = title
// 	if av.Title != "" {
// 		lab := av.TitleWidget()
// 		if lab != nil {
// 			lab.Text = title
// 		}
// 	}
// }

// ConfigArgsGrid configures the ArgsGrid for the current struct
func (av *ArgView) ConfigArgsGrid() {
	if laser.AnyIsNil(av.Args) {
		return
	}
	sg := av.ArgsGrid()
	sg.Stripes = gi.RowStripes
	config := ki.Config{}
	for i := range av.Args {
		arg := av.Args[i]
		if view, _ := arg.Tag("view"); view == "-" {
			continue
		}
		vtyp := arg.WidgetType()
		knm := strcase.ToKebab(arg.Name())
		labnm := "label-" + knm
		valnm := "value-" + knm
		config.Add(gi.LabelType, labnm)
		config.Add(vtyp, valnm)
	}
	mods, updt := sg.ConfigChildren(config) // not sure if always unique?
	if mods {
		av.SetNeedsLayoutUpdate(av.Sc, updt)
	} else {
		updt = sg.UpdateStart()
	}
	for i := range av.Args {
		arg := av.Args[i]
		if view, _ := arg.Tag("view"); view == "-" {
			continue
		}
		lbl := sg.Child(i * 2).(*gi.Label)
		lbl.Text = arg.Label()
		lbl.Tooltip = arg.Doc()
		w, wb := gi.AsWidget(sg.Child((i * 2) + 1))
		if wb.Class == "" {
			wb.Class = "configed"
			arg.ConfigWidget(w, av.Sc)
		} else {
			arg.AsValueBase().Widget = w
			arg.UpdateWidget()
		}
	}
	sg.UpdateEnd(updt)
}

// UpdateArgs updates each of the value-view widgets for the args
func (av *ArgView) UpdateArgs() {
	updt := av.UpdateStart()
	for i := range av.Args {
		ad := av.Args[i]
		ad.UpdateWidget()
	}
	av.UpdateEnd(updt)
}
