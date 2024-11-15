// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"fmt"

	"cogentcore.org/core/core"
	"cogentcore.org/core/plot/plotcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorcore"
	"cogentcore.org/core/texteditor"
)

// Tabber is a [core.Tabs] based widget that has support for opening
// tabs for [plotcore.PlotEditor] and [tensorcore.Table] editors,
// among others.
type Tabber interface {

	// AsTabs returns the underlying [Tabs] widget.
	AsTabs() *Tabs

	// RecycleTab returns a tab with the given name, first by looking for an existing one,
	// and if not found, making a new one. It returns the frame for the tab.
	RecycleTab(name string) *core.Frame

	// TensorTable recycles a tab with a [tensorcore.Table] widget
	// to view given [table.Table], using its own table.Table.
	TensorTable(label string, dt *table.Table) *tensorcore.Table

	// TensorEditor recycles a tab with a [tensorcore.TensorEditor] widget
	// to view given Tensor.
	TensorEditor(label string, tsr tensor.Tensor) *tensorcore.TensorEditor

	// TensorGrid recycles a tab with a [tensorcore.TensorGrid] widget
	// to view given Tensor.
	TensorGrid(label string, tsr tensor.Tensor) *tensorcore.TensorGrid

	// PlotTable recycles a tab with a Plot of given [table.Table].
	PlotTable(label string, dt *table.Table) *plotcore.PlotEditor

	// todo: PlotData of plot.Data

	// SliceTable recycles a tab with a [core.Table] widget
	// to view the given slice of structs.
	SliceTable(label string, slc any) *core.Table

	// EditorString recycles a [texteditor.Editor] tab, displaying given string.
	EditorString(label, content string) *texteditor.Editor

	// EditorFile opens an editor tab for given file.
	EditorFile(label, filename string) *texteditor.Editor
}

// NewTab recycles a tab with given label, or returns the existing one
// with given type of widget within it. mkfun function is called to create
// and configure a new widget if not already existing.
func NewTab[T any](tb Tabber, label string, mkfun func(tab *core.Frame) T) T {
	tab := tb.RecycleTab(label)
	var zv T
	if tab.HasChildren() {
		if tt, ok := tab.Child(1).(T); ok {
			return tt
		}
		err := fmt.Errorf("Name / Type conflict: tab %q does not have the expected type of content", label)
		core.ErrorSnackbar(tb.AsTabs(), err)
		return zv
	}
	w := mkfun(tab)
	return w
}

// Tabs implements the [Tabber] interface.
type Tabs struct {
	core.Tabs
}

func (ts *Tabs) Init() {
	ts.Tabs.Init()
	ts.Type = core.FunctionalTabs
}

func (ts *Tabs) AsTabs() *Tabs {
	return ts
}

// TensorTable recycles a tab with a tensorcore.Table widget
// to view given table.Table, using its own table.Table as tv.Table.
// Use tv.Table.Table to get the underlying *table.Table
// Use tv.Table.Sequential to update the Indexed to view
// all of the rows when done updating the Table, and then call br.Update()
func (ts *Tabs) TensorTable(label string, dt *table.Table) *tensorcore.Table {
	tv := NewTab(ts, label, func(tab *core.Frame) *tensorcore.Table {
		tb := core.NewToolbar(tab)
		tv := tensorcore.NewTable(tab)
		tb.Maker(tv.MakeToolbar)
		return tv
	})
	tv.SetTable(dt)
	ts.Update()
	return tv
}

// TensorEditor recycles a tab with a tensorcore.TensorEditor widget
// to view given Tensor.
func (ts *Tabs) TensorEditor(label string, tsr tensor.Tensor) *tensorcore.TensorEditor {
	tv := NewTab(ts, label, func(tab *core.Frame) *tensorcore.TensorEditor {
		tb := core.NewToolbar(tab)
		tv := tensorcore.NewTensorEditor(tab)
		tb.Maker(tv.MakeToolbar)
		return tv
	})
	tv.SetTensor(tsr)
	ts.Update()
	return tv
}

// TensorGrid recycles a tab with a tensorcore.TensorGrid widget
// to view given Tensor.
func (ts *Tabs) TensorGrid(label string, tsr tensor.Tensor) *tensorcore.TensorGrid {
	tv := NewTab(ts, label, func(tab *core.Frame) *tensorcore.TensorGrid {
		// tb := core.NewToolbar(tab)
		tv := tensorcore.NewTensorGrid(tab)
		// tb.Maker(tv.MakeToolbar)
		return tv
	})
	tv.SetTensor(tsr)
	ts.Update()
	return tv
}

// PlotTable recycles a tab with a Plot of given table.Table.
func (ts *Tabs) PlotTable(label string, dt *table.Table) *plotcore.PlotEditor {
	pl := NewTab(ts, label, func(tab *core.Frame) *plotcore.PlotEditor {
		return plotcore.NewSubPlot(tab)
	})
	pl.SetTable(dt)
	ts.Update()
	return pl
}

// SliceTable recycles a tab with a core.Table widget
// to view the given slice of structs.
func (ts *Tabs) SliceTable(label string, slc any) *core.Table {
	tv := NewTab(ts, label, func(tab *core.Frame) *core.Table {
		return core.NewTable(tab)
	})
	tv.SetSlice(slc)
	ts.Update()
	return tv
}

// EditorString recycles a [texteditor.Editor] tab, displaying given string.
func (ts *Tabs) EditorString(label, content string) *texteditor.Editor {
	ed := NewTab(ts, label, func(tab *core.Frame) *texteditor.Editor {
		ed := texteditor.NewEditor(tab)
		ed.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
		return ed
	})
	if content != "" {
		ed.Buffer.SetText([]byte(content))
	}
	ts.Update()
	return ed
}

// EditorFile opens an editor tab for given file.
func (ts *Tabs) EditorFile(label, filename string) *texteditor.Editor {
	ed := NewTab(ts, label, func(tab *core.Frame) *texteditor.Editor {
		ed := texteditor.NewEditor(tab)
		ed.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
		return ed
	})
	ed.Buffer.Open(core.Filename(filename))
	ts.Update()
	return ed
}
