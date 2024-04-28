// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"fmt"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// ArgView represents a slice of reflect.Value's and associated names, for the
// purpose of supplying arguments to methods called via the MethodView
// framework.
type ArgView struct {
	core.Frame

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
	av.OnWidgetAdded(func(w core.Widget) {
		switch w.PathFrom(av) {
		case "title":
			title := w.(*core.Text)
			title.Type = core.TextTitleLarge
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
	config := tree.Config{}
	config.Add(core.TextType, "title")
	config.Add(core.FrameType, "args-grid")
	av.ConfigChildren(config)
	av.ConfigArgsGrid()
}

// TitleWidget returns the title label widget
func (av *ArgView) TitleWidget() *core.Text {
	return av.ChildByName("title", 0).(*core.Text)
}

// ArgsGrid returns the grid layout widget, which contains all the fields
// and values
func (av *ArgView) ArgsGrid() *core.Frame {
	return av.ChildByName("args-grid", 0).(*core.Frame)
}

// ConfigArgsGrid configures the ArgsGrid for the current struct
func (av *ArgView) ConfigArgsGrid() {
	if reflectx.AnyIsNil(av.Args) {
		return
	}
	sg := av.ArgsGrid()
	config := tree.Config{}
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
		config.Add(core.TextType, labnm)
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
		text := sg.Child(idx * 2).(*core.Text)
		text.Text = arg.Label()
		text.Tooltip = arg.Doc()
		w, _ := core.AsWidget(sg.Child((idx * 2) + 1))
		Config(arg, w)
		idx++
	}
}
