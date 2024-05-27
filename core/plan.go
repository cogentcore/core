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

// Plan represents a plan for how a widget should be constructed and initialized.
// An instance of it is passed to [Widget.Make], which is responsible for making
// the plan that is then used to configure and update the widget in [Widget.Build].
// To add a child item to a plan, use [Add], [AddAt], or [AddNew]. To extend an
// existing child item, use [AddInit].
type Plan struct {

	// Name is the name of the planned widget. If it is blank, this [Plan]
	// is assumed to be the root Plan, and only its children are handled.
	Name string

	// New returns a new [Widget] of the correct type for this element,
	// fully configured and ready for use.
	New func() Widget

	// Init is a list of functions that are called once in sequential ascending order
	// after [Plan.New] to initialize the widget for the first time; see [AddInit].
	Init []func(w Widget)

	// Children is a list of [Plan]s that are used to configure and update children
	// nested under the widget of this [Plan].
	Children []*Plan
}

// Add adds a new [Plan] item to the given [Plan] for a widget with
// the given optional function to initialize the widget. The name of
// the widget is automatically generated based on the file and line
// number of the calling function.
// It returns the new [Plan] item.
func Add[T Widget](p *Plan, init ...func(w T)) *Plan {
	return AddAt(p, autoPlanPath(2), init...)
}

// autoPlanPath returns the dir-filename of [runtime.Caller](level),
// with all / . replaced to -, which is suitable as a unique name
// for a [PlanItem.Path].
func autoPlanPath(level int) string {
	_, file, line, _ := runtime.Caller(level)
	name := filepath.Base(file)
	dir := filepath.Base(filepath.Dir(file))
	path := dir + "-" + name
	path = strings.ReplaceAll(strings.ReplaceAll(path, "/", "-"), ".", "-") + "-" + strconv.Itoa(line)
	return path
}

// AddAt adds a new [Plan] item to the given [Plan] for a widget with
// the given name and optional function to initialize the widget. The
// widget is guaranteed to have its parent set before the init function
// is called.
// It returns the new [Plan] item.
func AddAt[T Widget](p *Plan, path string, init ...func(w T)) *Plan {
	new := func() Widget {
		return tree.New[T]()
	}
	if len(init) == 0 {
		return p.Add(path, new)
	}
	f := init[0]
	return p.Add(path, new, func(w Widget) {
		f(w.(T))
	})
}

// AddNew adds a new [Plan] item to the given [Plan] for a widget with
// the given name, function for constructing the widget, and function
// for initializing the widget. The widget is guaranteed to
// have its parent set before the init function is called. It returns
// the new [Plan] item.
// It should only be called instead of [Add] and [AddAt] when the widget
// must be made new, like when using [NewValue].
func AddNew[T Widget](p *Plan, path string, new func() T, init func(w T)) *Plan {
	return p.Add(path, func() Widget {
		return new()
	}, func(w Widget) {
		init(w.(T))
	})
}

// AddInit adds a new function for initializing the widget with the given name
// in the given [Plan]. The widget must already exist in the plan; this is for
// extending an existing [Plan] item, not adding a new one. The widget is guaranteed
// to have its parent set before the init function is called. The init functions are
// called in sequential ascending order.
func AddInit[T Widget](p *Plan, name string, init func(w T)) {
	for _, child := range p.Children {
		if child.Name == name {
			child.Init = append(child.Init, func(w Widget) {
				init(w.(T))
			})
			return
		}
	}
	slog.Error("core.AddInit: child not found", "name", name)
}

// Add adds a new [Plan] item to the given [Plan] with the given name and functions.
// It should typically not be called by end-user code; see the generic
// [Add], [AddAt], and [AddNew] functions instead. It returns the new [Plan] item.
func (p *Plan) Add(name string, new func() Widget, init ...func(w Widget)) *Plan {
	newPlan := &Plan{Name: name, New: new, Init: init}
	p.Children = append(p.Children, newPlan)
	return newPlan
}

// BuildWidget builds (configures and updates) the given widget and
// all of its children in accordance with the [Plan].
func (p *Plan) BuildWidget(w Widget) {
	if len(p.Children) == 0 { // TODO(config): figure out a better way to handle this?
		return
	}
	wb := w.AsWidget()
	for i, child := range p.Children { // TODO(config): figure out a better way to handle this?
		if child.Name != "parts" {
			continue
		}
		if wb.Parts == nil {
			wparts := child.New()
			wparts.SetName("parts")
			wb.Parts = wparts.(*Frame)
			tree.SetParent(wb.Parts, wb)
			child.BuildWidget(wparts)
		}
		p.Children = slices.Delete(p.Children, i, i+1) // not a real child
		break
	}
	if len(p.Children) == 0 { // check again after potentially removing parts
		return
	}
	wb.Kids, _ = plan.Build(wb.Kids, len(p.Children),
		func(i int) string { return p.Children[i].Name },
		func(name string, i int) tree.Node {
			child := p.Children[i]
			cw := child.New()
			cw.SetName(name)
			tree.SetParent(cw, wb)
			for _, f := range child.Init {
				f(cw)
			}
			return cw
		}, func(n tree.Node) { n.Destroy() })
	for i, child := range p.Children { // always build children even if not new
		cw := wb.Child(i).(Widget)
		child.BuildWidget(cw)
	}
}

// InitParts configures the given [Frame] to be ready
// to serve as [WidgetBase.Parts] in a [Add] context.
func InitParts(w *Frame) {
	w.SetName("parts")
	w.SetFlag(true, tree.Field)
	w.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.RenderBox = false
	})
}
