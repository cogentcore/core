// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"log/slog"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/base/plan"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Plan represents a plan for how the children of a widget should be configured.
// An instance of it is passed to [WidgetBase.Make], which is responsible
// for making the plan that is then used to configure the widget in [WidgetBase.Build].
// To add a child item to a plan, use [Add], [AddAt], or [AddNew]. To add a child
// item maker to a widget, use [AddChild] or [AddChildAt]. To extend an existing
// child item, use [AddInit] or [AddChildInit].
type Plan []*PlanItem

// PlanItem represents a plan for how a child widget should be constructed and initialized.
// See [Plan] for more information.
type PlanItem struct {

	// Name is the name of the planned widget.
	Name string

	// New returns a new widget of the correct type for this child.
	New func() Widget

	// Init is a list of functions that are called once in sequential ascending order
	// after [PlanItem.New] to initialize the widget for the first time.
	Init []func(w Widget)
}

// Add adds a new [PlanItem] to the given [Plan] for a widget with
// the given function to initialize the widget. The widget
// is guaranteed to have its parent set before the init function
// is called. The name of the widget is automatically generated based
// on the file and line number of the calling function.
func Add[T Widget](p *Plan, init func(w T)) {
	AddAt(p, autoPlanName(2), init)
}

// autoPlanName returns the dir-filename of [runtime.Caller](level),
// with all / . replaced to -, which is suitable as a unique name
// for a [PlanItem.Name].
func autoPlanName(level int) string {
	_, file, line, _ := runtime.Caller(level)
	name := filepath.Base(file)
	dir := filepath.Base(filepath.Dir(file))
	path := dir + "-" + name
	path = strings.ReplaceAll(strings.ReplaceAll(path, "/", "-"), ".", "-") + "-" + strconv.Itoa(line)
	return path
}

// AddAt adds a new [PlanItem] to the given [Plan] for a widget with
// the given name and function to initialize the widget. The widget
// is guaranteed to have its parent set before the init function
// is called.
func AddAt[T Widget](p *Plan, name string, init func(w T)) {
	p.Add(name, func() Widget {
		return tree.New[T]()
	}, func(w Widget) {
		init(w.(T))
	})
}

// AddNew adds a new [PlanItem] to the given [Plan] for a widget with
// the given name, function for constructing the widget, and function
// for initializing the widget. The widget is guaranteed to
// have its parent set before the init function is called.
// It should only be called instead of [Add] and [AddAt] when the widget
// must be made new, like when using [NewValue].
func AddNew[T Widget](p *Plan, name string, new func() T, init func(w T)) {
	p.Add(name, func() Widget {
		return new()
	}, func(w Widget) {
		init(w.(T))
	})
}

// AddInit adds a new function for initializing the widget with the given name
// in the given [Plan]. The widget must already exist in the plan; this is for
// extending an existing [PlanItem], not adding a new one. The widget is guaranteed
// to have its parent set before the init function is called. The init functions are
// called in sequential ascending order.
func AddInit[T Widget](p *Plan, name string, init func(w T)) {
	for _, child := range *p {
		if child.Name == name {
			child.Init = append(child.Init, func(w Widget) {
				init(w.(T))
			})
			return
		}
	}
	slog.Error("core.AddInit: child not found", "name", name)
}

// AddChild adds a new [WidgetBase.Maker] to the the given parent widget that
// adds a [PlanItem] with the given init function using [Add]. In other words,
// this adds a maker that will add a child to the given parent.
func AddChild[T Widget](parent Widget, init func(w T)) {
	name := autoPlanName(2) // must get here to get correct name
	parent.AsWidget().Maker(func(p *Plan) {
		AddAt(p, name, init)
	})
}

// AddChildAt adds a new [WidgetBase.Maker] to the the given parent widget that
// adds a [PlanItem] with the given name and init function using [AddAt]. In other
// words, this adds a maker that will add a child to the given parent.
func AddChildAt[T Widget](parent Widget, name string, init func(w T)) {
	parent.AsWidget().Maker(func(p *Plan) {
		AddAt(p, name, init)
	})
}

// AddChildInit adds a new [WidgetBase.Maker] to the the given parent widget that
// adds a new function for initializing the widget with the given name
// in the given [Plan]. The widget must already exist in the plan; this is for
// extending an existing [PlanItem], not adding a new one. The widget is guaranteed
// to have its parent set before the init function is called. The init functions are
// called in sequential ascending order.
func AddChildInit[T Widget](parent Widget, name string, init func(w T)) {
	parent.AsWidget().Maker(func(p *Plan) {
		AddInit(p, name, init)
	})
}

// Add adds a new [PlanItem] to the given [Plan] with the given name and functions.
// It should typically not be called by end-user code; see the generic
// [Add], [AddAt], [AddNew], [AddChild], [AddChildAt], [AddInit], and [AddChildInit]
// functions instead.
func (p *Plan) Add(name string, new func() Widget, init func(w Widget)) {
	*p = append(*p, &PlanItem{Name: name, New: new, Init: []func(w Widget){init}})
}

// Update updates the children of the given widget in accordance with the [Plan].
func (p *Plan) Update(w Widget) {
	if len(*p) == 0 { // TODO(config): figure out a better way to handle this?
		return
	}
	wb := w.AsWidget()
	makeNew := func(item *PlanItem) Widget {
		child := item.New()
		child.SetName(item.Name)
		tree.SetParent(child, wb)
		for _, f := range item.Init {
			f(child)
		}
		return child
	}
	for i, item := range *p { // TODO(config): figure out a better way to handle this?
		if item.Name != "parts" {
			continue
		}
		if wb.Parts == nil {
			wb.Parts = makeNew(item).(*Frame)
		}
		*p = slices.Delete(*p, i, i+1) // not a real child
		break
	}
	if len(*p) == 0 { // check again after potentially removing parts
		return
	}
	wb.Kids, _ = plan.Update(wb.Kids, len(*p),
		func(i int) string {
			return (*p)[i].Name
		}, func(name string, i int) tree.Node {
			return makeNew((*p)[i])
		}, func(n tree.Node) {
			n.Destroy()
		})
}

// InitParts configures the given [Frame] to be ready
// to serve as [WidgetBase.Parts] in a [Add] context.
func InitParts(w *Frame) {
	w.SetFlag(true, tree.Field)
	w.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.RenderBox = false
	})
}
