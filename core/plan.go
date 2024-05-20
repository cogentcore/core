// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"cogentcore.org/core/base/config"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Plan represents a plan for how a widget should be configured and updated.
// An instance of it is passed to [Widget.Make], which is responsible for making
// the plan that is then used to configure and update the widget in [Widget.Build].
// To add an item to a plan, use [Add], [AddAt], or [AddNew].
type Plan []*PlanItem

// PlanItem represents one item of a [Plan] that specifies how a single widget
// and its children should be configured and updated in [Widget.Build]. It contains
// closures responsible for creating and updating the widget. Also, it can contain
// an additional [Plan] for configuring the children of the widget. To add a new
// PlanItem, call [Add], [AddAt], or [AddNew]. Those functions return a [Plan]
// that you can use as an argument to future Add calls to add more [PlanItem]s
// for children nested under the already added widget.
type PlanItem struct {

	// Path is the forward slash delimited path to the element.
	Path string

	// New returns a new Widget of the correct type for this element,
	// fully configured and ready for use.
	New func() Widget

	// Update updates the widget based on current state, so that it
	// propertly renders the correct information.
	Update func(w Widget)

	// Config for Children elements.
	Children Plan
}

// Configure adds a new config item to the given [Plan] for a widget at the
// given forward slash separated path with the given optional function(s). The
// first function is called to initially configure the widget, and the second
// function is called to update the widget. If the given path is blank, it is
// automatically set to a unique name based on the filepath and line number of
// the calling function.
func Configure[T Widget](c *Plan, path string, funcs ...func(w T)) {
	switch len(funcs) {
	case 0:
		c.Add(path, func() Widget { return tree.New[T]() }, nil)
	case 1:
		init := funcs[0]
		c.Add(path, func() Widget {
			w := tree.New[T]()
			// w.SetName(path)
			init(w)
			return w
		}, nil)
	default:
		init := funcs[0]
		update := funcs[1]
		c.Add(path, func() Widget {
			w := tree.New[T]()
			// w.SetName(path)
			init(w)
			return w
		}, func(w Widget) {
			update(w.(T))
		})
	}
}

// ConfigureNew adds a new config item to the given [Plan] for a widget at the
// given forward slash separated path with the given function for constructing
// the widget and the given optional function for updating the widget. If the
// given path is blank, it is automatically set to a unique name based on the
// filepath and line number of the calling function.
func ConfigureNew[T Widget](c *Plan, path string, new func() T, update ...func(w T)) {
	if len(update) == 0 {
		c.Add(path, func() Widget { return new() }, nil)
	} else {
		u := update[0]
		c.Add(path, func() Widget { return new() }, func(w Widget) { u(w.(T)) })
	}
}

// Add adds a new config item for a widget at the given forward slash separated
// path with the given function for constructing the widget and the given function
// for updating the widget. This should be called on the root level Config
// list. Any items with nested paths are added to Children lists as
// appropriate. If the given path is blank, it is automatically set to
// a unique name based on the filepath and line number of the calling function.
// Consider using the [Configure] global generic function for
// better type safety and increased convenience.
func (c *Plan) Add(path string, new func() Widget, update func(w Widget)) {
	if path == "" {
		path = ConfigCallerPath(3)
	}
	itm := &PlanItem{Path: path, New: new, Update: update}
	plist := strings.Split(path, "/")
	if len(plist) == 1 {
		*c = append(*c, itm)
		return
	}
	next := c.FindMakeChild(plist[0])
	next.AddSubItem(plist[1:], itm)
}

// ConfigCallerPath returns the dir-filename of [runtime.Caller](level),
// with all / . replaced to -, which is suitable as a unique name
// for a [PlanItem.Path].
func ConfigCallerPath(level int) string {
	_, file, line, _ := runtime.Caller(level)
	dir, fn := filepath.Split(file)
	d1 := ""
	if len(dir) > 1 {
		_, d1 = filepath.Split(dir[:len(dir)-1])
	}
	path := fn + "-" + d1
	// need to get rid of slashes and dots for path name
	path = strings.ReplaceAll(strings.ReplaceAll(path, "/", "-"), ".", "-") + "-" + strconv.Itoa(line)
	return path
}

// ChildPath returns this item's path + "/" + child
func (c *PlanItem) ChildPath(child string) string {
	return c.Path + "/" + child
}

// ItemName returns this item's name from its path,
// as the last element in the path.
func (c *PlanItem) ItemName() string {
	pi := strings.LastIndex(c.Path, "/")
	if pi < 0 {
		return c.Path
	}
	return c.Path[pi+1:]
}

// AddSubItem adds given sub item to this config item, based on the
// list of path elements, where the first path element should be
// immediate Children of this item, etc.
func (c *PlanItem) AddSubItem(path []string, itm *PlanItem) {
	fpath := c.ChildPath(path[0])
	child := c.Children.FindMakeChild(fpath)
	if len(path) == 1 {
		*child = *itm
		return
	}
	child.AddSubItem(path[1:], itm)
}

// FindChild finds item with given path string within this list.
// Does not search within any Children of items here.
// Returns nil if not found.
func (c *Plan) FindChild(path string) *PlanItem {
	for _, itm := range *c {
		if itm.Path == path {
			return itm
		}
	}
	return nil
}

// FindMakeChild finds item with given path element within this list.
// Does not search within any Children of items here.
// Makes a new item if not found.
func (c *Plan) FindMakeChild(path string) *PlanItem {
	for _, itm := range *c {
		if itm.Path == path {
			return itm
		}
	}
	itm := &PlanItem{Path: path}
	*c = append(*c, itm)
	return itm
}

// String returns a newline separated list of paths for all the items,
// including the Children of items.
func (c *Plan) String() string {
	str := ""
	for _, itm := range *c {
		str += itm.Path + "\n"
		str += itm.Children.String()
	}
	return str
}

// SplitParts splits out a config item with sub-name "parts" from remainder
// returning both (each of which could be nil).  parpath contains any parent
// path to add to the path
func (c *Plan) SplitParts(parpath string) (parts *PlanItem, children Plan) {
	partnm := "parts"
	if parpath != "" {
		partnm = parpath + "/" + partnm
	}
	for _, itm := range *c {
		if itm.Path == partnm {
			parts = itm
		} else {
			children = append(children, itm)
		}
	}
	return
}

// ConfigWidget runs the Config on the given widget, ensuring that
// the widget has the specified parts and direct Children.
// The given parent path is used for recursion and should be blank
// when calling the function externally.
func (c *Plan) ConfigWidget(w Widget, parentPath string) {
	wb := w.AsWidget()
	parts, children := c.SplitParts(parentPath)
	if parts != nil {
		if wb.Parts == nil {
			wparts := parts.New()
			wparts.SetName("parts")
			wb.Parts = wparts.(*Frame)
			tree.SetParent(wb.Parts, wb)
			parts.Children.ConfigWidget(wparts, parts.ChildPath("parts"))
		}
	}
	n := len(children)
	if n > 0 {
		wb.Kids, _ = config.Config(wb.Kids, n,
			func(i int) string { return children[i].ItemName() },
			func(name string, i int) tree.Node {
				child := children[i]
				if child.New == nil {
					fmt.Println("core.Config child.New is nil:", child.ItemName())
					return nil
				}
				cw := child.New()
				cw.SetName(name)
				// fmt.Println(name, cw, wb)
				tree.SetParent(cw, wb)
				if child.Update != nil { // do initial setting in case children might reference
					child.Update(cw)
				}
				return cw
			})
		for i, child := range children { // always config children even if not new
			if len(child.Children) > 0 {
				cw := wb.Child(i).(Widget)
				child.Children.ConfigWidget(cw, child.Path)
			}
		}
	}
	if parentPath == "" { // top level
		c.UpdateWidget(w, parentPath) // this is recursive
	}
}

// UpdateWidget runs the [PlanItem.Update] functions on the given widget,
// and recursively on all of its children as specified in the Config.
// It is called at the end of [Plan.ConfigWidget].
// The given parent path is used for recursion and should be blank
// when calling the function externally.
func (c *Plan) UpdateWidget(w Widget, parentPath string) {
	wb := w.AsWidget()
	parts, children := c.SplitParts(parentPath)
	if parts != nil {
		parts.Children.UpdateWidget(wb.Parts, parts.ChildPath("parts"))
	}
	n := len(children)
	if n == 0 {
		return
	}
	for i, child := range children {
		cw := wb.Child(i).(Widget)
		if child.Update != nil {
			child.Update(cw)
		}
		if len(child.Children) > 0 {
			child.Children.UpdateWidget(cw, child.Path)
		}
	}
}

// ConfigParts configures the given [Frame] to be ready
// to serve as [WidgetBase.Parts] in a [Configure] context.
func ConfigParts(w *Frame) {
	w.SetName("parts")
	w.SetFlag(true, tree.Field)
	w.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.RenderBox = false
	})
}

// ConfigWidget is the base implementation of [Widget.ConfigWidget] that
// configures the widget by doing steps that apply to all widgets and then
// calling [Widget.Config] for widget-specific configuration steps.
func (wb *WidgetBase) ConfigWidget() {
	if wb.ValueUpdate != nil {
		wb.ValueUpdate()
	}
	c := Plan{}
	wb.This().(Widget).Config(&c)
	if len(c) > 0 {
		c.ConfigWidget(wb.This().(Widget), "")
	}
}

// Config is the interface method called by [Widget.ConfigWidget] that
// should be defined for each [Widget] type, which actually does
// the configuration work.
func (wb *WidgetBase) Config(c *Plan) {
	// this must be defined for each widget type
}

// ConfigTree calls [Widget.ConfigWidget] on every Widget in the tree from me.
func (wb *WidgetBase) ConfigTree() {
	if wb.This() == nil {
		return
	}
	// pr := profile.Start(wb.This().NodeType().ShortName())
	wb.WidgetWalkDown(func(wi Widget, wb *WidgetBase) bool {
		wi.ConfigWidget()
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
		wi.ConfigWidget()
		wi.ApplyStyle()
		return tree.Continue
	})
	wb.NeedsLayout()
}

//////////////////////////////////////////////////////////////////////////////
// 	ConfigFuncs

// ConfigFuncs is a stack of config functions, which take a Config
// and add to it.
type ConfigFuncs []func(c *Plan)

// Add adds the given function for configuring a toolbar
func (cf *ConfigFuncs) Add(fun ...func(c *Plan)) *ConfigFuncs {
	*cf = append(*cf, fun...)
	return cf
}

// Call calls all the functions for configuring given toolbar
func (cf *ConfigFuncs) Call(c *Plan) {
	for _, fun := range *cf {
		fun(c)
	}
}

// IsEmpty returns true if there are no functions added
func (cf *ConfigFuncs) IsEmpty() bool {
	return len(*cf) == 0
}
