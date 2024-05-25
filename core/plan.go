// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/base/plan"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Plan represents a plan for how a widget should be configured and updated.
// An instance of it is passed to [Widget.Make], which is responsible for making
// the plan that is then used to configure and update the widget in [Widget.Build].
// To add a child item to a plan, use [Add], [AddAt], or [AddNew]. Those functions
// return a Plan that you can use as an argument to future Add calls to add more
// Plans for children nested under the already added Plan. Plan contains closures
// responsible for creating and updating a widget.
type Plan struct {

	// Name is the name of the planned widget. If it is blank, this [Plan]
	// is assumed to be the root Plan, and only its children are handled.
	Name string

	// New returns a new [Widget] of the correct type for this element,
	// fully configured and ready for use.
	New func() Widget

	// Update updates the widget based on current state so that it
	// propertly represents the correct information.
	Update func(w Widget)

	// Children is a list of [Plan]s that are used to configure and update children
	// nested under the widget of this [Plan].
	Children []*Plan
}

// Add adds a new [Plan] item to the given [Plan] for a widget with
// the given optional function(s). The first function is called
// to initially configure the widget, and the second function is called
// to update the widget. The name of the widget is automatically generated
// based on the file and line number of the calling function.
// It returns the new [Plan] item.
func Add[T Widget](p *Plan, funcs ...func(w T)) *Plan {
	return AddAt(p, autoPlanPath(2), funcs...)
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
// the given name and optional function(s). The first function is called
// to initially configure the widget, and the second function is called
// to update the widget. It returns the new [Plan] item.
func AddAt[T Widget](p *Plan, path string, funcs ...func(w T)) *Plan {
	switch len(funcs) {
	case 0:
		return p.Add(path, func() Widget { return tree.New[T]() }, nil)
	case 1:
		init := funcs[0]
		return p.Add(path, func() Widget {
			w := tree.New[T]()
			init(w)
			return w
		}, nil)
	default:
		init := funcs[0]
		update := funcs[1]
		return p.Add(path, func() Widget {
			w := tree.New[T]()
			init(w)
			return w
		}, func(w Widget) {
			update(w.(T))
		})
	}
}

// AddNew adds a new [Plan] item to the given [Plan] for a widget with
// the given name, function for constructing the widget, and optional
// function for updating the widget. It returns the new [Plan] item.
// It should only be called instead of [Add] and [AddAt] when the widget
// must be made new, like when using [NewValue].
func AddNew[T Widget](p *Plan, path string, new func() T, update ...func(w T)) *Plan {
	if len(update) == 0 {
		return p.Add(path, func() Widget { return new() }, nil)
	}
	u := update[0]
	return p.Add(path, func() Widget { return new() }, func(w Widget) { u(w.(T)) })
}

// Add adds a new [Plan] item to the given [Plan] with the given name and functions.
// It should typically not be called by end-user code; see the generic
// [Add], [AddAt], and [AddNew] functions instead. It returns the new [Plan] item.
func (p *Plan) Add(name string, new func() Widget, update func(w Widget)) *Plan {
	newPlan := &Plan{Name: name, New: new, Update: update}
	p.Children = append(p.Children, newPlan)
	return newPlan
}

// BuildWidget builds (configures and updates) the given widget and
// all of its children in accordance with the [Plan].
func (p *Plan) BuildWidget(w Widget) {
	p.buildWidget(w)
	p.UpdateWidget(w) // this gets everything
}

// buildWidget is the recursive implementation of [Plan.BuildWidget].
func (p *Plan) buildWidget(w Widget) {
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
			child.buildWidget(wparts)
		}
		p.Children = slices.Delete(p.Children, i, i+1) // not a real child
		// if we only have parts, then we must bail to avoid affecting real children
		if len(p.Children) == 0 {
			return
		}
		break
	}
	wb.Kids, _ = plan.Build(wb.Kids, len(p.Children),
		func(i int) string { return p.Children[i].Name },
		func(name string, i int) tree.Node {
			child := p.Children[i]
			cw := child.New()
			cw.SetName(name)
			tree.SetParent(cw, wb)
			if child.Update != nil { // do initial setting in case children might reference
				child.Update(cw)
			}
			return cw
		}, func(n tree.Node) { n.Destroy() })
	for i, child := range p.Children { // always build children even if not new
		cw := wb.Child(i).(Widget)
		child.buildWidget(cw)
	}
}

// UpdateWidget updates the given widget and all of its children to reflect
// the current state in accordance with the [Plan]. It does not change the
// actual structure of the widget tree; see [Plan.BuildWidget] for that.
func (p *Plan) UpdateWidget(w Widget) {
	wb := w.AsWidget()
	for i, child := range p.Children {
		cw := wb.Child(i).(Widget)
		if child.Update != nil {
			child.Update(cw)
		}
		child.UpdateWidget(cw)
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

// Build is the base implementation of [Widget.Build] that
// configures the widget by doing steps that apply to all widgets and then
// calling [Widget.Config] for widget-specific configuration steps.
func (wb *WidgetBase) Build() {
	if wb.ValueUpdate != nil {
		wb.ValueUpdate()
	}
	p := Plan{}
	wb.This().(Widget).Make(&p)
	p.BuildWidget(wb.This().(Widget))
}

// Config is the interface method called by [Widget.Build] that
// should be defined for each [Widget] type, which actually does
// the configuration work.
func (wb *WidgetBase) Make(p *Plan) {
	// this must be defined for each widget type
}

// ConfigTree calls [Widget.Build] on every Widget in the tree from me.
func (wb *WidgetBase) ConfigTree() {
	if wb.This() == nil {
		return
	}
	// pr := profile.Start(wb.This().NodeType().ShortName())
	wb.WidgetWalkDown(func(wi Widget, wb *WidgetBase) bool {
		wi.Build()
		return tree.Continue
	})
	// pr.End()
}

// Update does a general purpose update of the widget and everything
// below it by reconfiguring it, applying its styles, and indicating
// that it needs a new layout pass. It is the main way that end users
// should update widgets, and it should be called after making any
// changes to the core properties of a widget (for example, the text
// of [Text], the icon of a [Button], or the slice of a table view).
//
// If you are calling this in a separate goroutine outside of the main
// configuration, rendering, and event handling structure, you need to
// call [WidgetBase.AsyncLock] and [WidgetBase.AsyncUnlock] before and
// after this, respectively.
func (wb *WidgetBase) Update() { //types:add
	if wb == nil || wb.This() == nil {
		return
	}
	if DebugSettings.UpdateTrace {
		fmt.Println("\tDebugSettings.UpdateTrace Update:", wb)
	}
	wb.WidgetWalkDown(func(wi Widget, wb *WidgetBase) bool {
		wi.Build()
		wi.ApplyStyle()
		return tree.Continue
	})
	wb.NeedsLayout()
}
