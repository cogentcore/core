// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
)

// ArgView represents a slice of reflect.Value's and associated names, for the
// purpose of supplying arguments to methods called via the MethodView
// framework.
type ArgView struct {
	gi.Frame

	// Args are the args that we are a view onto
	Args []Value
}

func (av *ArgView) OnInit() {
	av.Frame.OnInit()
	av.SetStyles()
}

func (av *ArgView) SetStyles() {
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
				s.Text.Align = styles.Center
			})
		case "args-grid":
			w.Style(func(s *styles.Style) {
				s.Display = styles.Grid
				s.Columns = 2
				s.Min.X.Ch(60)
				s.Min.Y.Em(10)
				s.Grow.Set(1, 1)
				s.Overflow.Set(styles.OverflowAuto)
			})
		}
	})
}

// Config configures the view
func (av *ArgView) Config() {
	config := ki.Config{}
	config.Add(gi.LabelType, "title")
	config.Add(gi.FrameType, "args-grid")
	av.ConfigChildren(config)
	av.ConfigArgsGrid()
}

// TitleWidget returns the title label widget
func (av *ArgView) TitleWidget() *gi.Label {
	return av.ChildByName("title", 0).(*gi.Label)
}

// ArgsGrid returns the grid layout widget, which contains all the fields
// and values
func (av *ArgView) ArgsGrid() *gi.Frame {
	return av.ChildByName("args-grid", 0).(*gi.Frame)
}

// ConfigArgsGrid configures the ArgsGrid for the current struct
func (av *ArgView) ConfigArgsGrid() {
	if laser.AnyIsNil(av.Args) {
		return
	}
	sg := av.ArgsGrid()
	config := ki.Config{}
	argnms := make(map[string]bool)
	for i := range av.Args {
		arg := av.Args[i]
		if view, _ := arg.Tag("view"); view == "-" {
			continue
		}
		knm := strcase.ToKebab(arg.Name())
		if _, has := argnms[knm]; has {
			knm += fmt.Sprintf("%d", i)
		}
		argnms[knm] = true
		labnm := "label-" + knm
		valnm := "value-" + knm
		config.Add(gi.LabelType, labnm)
		config.Add(arg.WidgetType(), valnm)
	}
	if sg.ConfigChildren(config) {
		av.NeedsLayout()
	}
	idx := 0
	for i := range av.Args {
		arg := av.Args[i]
		if view, _ := arg.Tag("view"); view == "-" {
			continue
		}
		arg.SetTag("grow", "1")
		lbl := sg.Child(idx * 2).(*gi.Label)
		lbl.Text = arg.Label()
		lbl.Tooltip = arg.Doc()
		w, _ := gi.AsWidget(sg.Child((idx * 2) + 1))
		Config(arg, w)
		idx++
	}
}
