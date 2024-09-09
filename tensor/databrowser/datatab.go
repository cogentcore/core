// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/plot/plotcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorcore"
	"cogentcore.org/core/texteditor"
)

// NewTab creates a tab with given label, or returns the existing one
// with given type of widget within it. mkfun function is called to create
// and configure a new widget if not already existing.
func NewTab[T any](br *Browser, label string, mkfun func(tab *core.Frame) T) T {
	tab := br.tabs.RecycleTab(label)
	if tab.HasChildren() {
		return tab.Child(1).(T)
	}
	w := mkfun(tab)
	return w
}

// NewTabTensorTable creates a tab with a tensorcore.Table widget
// to view given table.Table, using its own table.IndexView as tv.Table.
// Use tv.Table.Table to get the underlying *table.Table
// Use tv.Table.Sequential to update the IndexView to view
// all of the rows when done updating the Table, and then call br.Update()
func (br *Browser) NewTabTensorTable(label string, dt *table.Table) *tensorcore.Table {
	tv := NewTab[*tensorcore.Table](br, label, func(tab *core.Frame) *tensorcore.Table {
		tb := core.NewToolbar(tab)
		tv := tensorcore.NewTable(tab)
		tb.Maker(tv.MakeToolbar)
		return tv
	})
	tv.SetTable(dt)
	br.Update()
	return tv
}

// NewTabTensorEditor creates a tab with a tensorcore.TensorEditor widget
// to view given Tensor.
func (br *Browser) NewTabTensorEditor(label string, tsr tensor.Tensor) *tensorcore.TensorEditor {
	tv := NewTab[*tensorcore.TensorEditor](br, label, func(tab *core.Frame) *tensorcore.TensorEditor {
		tb := core.NewToolbar(tab)
		tv := tensorcore.NewTensorEditor(tab)
		tb.Maker(tv.MakeToolbar)
		return tv
	})
	tv.SetTensor(tsr)
	br.Update()
	return tv
}

// NewTabTensorGrid creates a tab with a tensorcore.TensorGrid widget
// to view given Tensor.
func (br *Browser) NewTabTensorGrid(label string, tsr tensor.Tensor) *tensorcore.TensorGrid {
	tv := NewTab[*tensorcore.TensorGrid](br, label, func(tab *core.Frame) *tensorcore.TensorGrid {
		// tb := core.NewToolbar(tab)
		tv := tensorcore.NewTensorGrid(tab)
		// tb.Maker(tv.MakeToolbar)
		return tv
	})
	tv.SetTensor(tsr)
	br.Update()
	return tv
}

// NewTabPlot creates a tab with a Plot of given table.Table.
func (br *Browser) NewTabPlot(label string, dt *table.Table) *plotcore.PlotEditor {
	pl := NewTab[*plotcore.PlotEditor](br, label, func(tab *core.Frame) *plotcore.PlotEditor {
		return plotcore.NewSubPlot(tab)
	})
	pl.SetTable(dt)
	br.Update()
	return pl
}

// NewTabSliceTable creates a tab with a core.Table widget
// to view the given slice of structs.
func (br *Browser) NewTabSliceTable(label string, slc any) *core.Table {
	tv := NewTab[*core.Table](br, label, func(tab *core.Frame) *core.Table {
		return core.NewTable(tab)
	})
	tv.SetSlice(slc)
	br.Update()
	return tv
}

// NewTabEditor opens a texteditor.Editor tab, displaying given string.
func (br *Browser) NewTabEditor(label, content string) *texteditor.Editor {
	ed := NewTab[*texteditor.Editor](br, label, func(tab *core.Frame) *texteditor.Editor {
		ed := texteditor.NewEditor(tab)
		ed.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
		return ed
	})
	if content != "" {
		ed.Buffer.SetText([]byte(content))
	}
	br.Update()
	return ed
}

// NewTabEditorFile opens an editor tab for given file
func (br *Browser) NewTabEditorFile(label, filename string) *texteditor.Editor {
	ed := NewTab[*texteditor.Editor](br, label, func(tab *core.Frame) *texteditor.Editor {
		ed := texteditor.NewEditor(tab)
		ed.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
		return ed
	})
	ed.Buffer.Open(core.Filename(filename))
	br.Update()
	return ed
}
