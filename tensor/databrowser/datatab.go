// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"fmt"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/plot/plotcore"
	"cogentcore.org/core/shell/cosh"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorcore"
	"cogentcore.org/core/texteditor"
)

// NewTabTensorTable creates a tab with a tensor table and a tensorcore table.
// Use tv.Table.Table to get the underlying *table.Table
// and tv.Table is the table.IndexView onto the table.
// Use tv.Table.Sequential to update the IndexView to view
// all of the rows when done updating the Table, and then call br.Update()
func (br *Browser) NewTabTensorTable(label string) *tensorcore.Table {
	tabs := br.Tabs()
	tab := tabs.RecycleTab(label)
	if tab.HasChildren() {
		tv := tab.Child(1).(*tensorcore.Table)
		return tv
	}
	dt := table.NewTable()
	tb := core.NewToolbar(tab)
	tv := tensorcore.NewTable(tab)
	tv.SetReadOnlyMultiSelect(true)
	tv.Styler(func(s *styles.Style) {
		s.SetReadOnly(true) // todo: not taking effect
	})
	tb.Maker(tv.MakeToolbar)
	tv.SetTable(dt)
	br.Update()
	return tv
}

// NewTabTable creates a tab with a slice Table.
// Sets the slice if tab already exists
func (br *Browser) NewTabTable(label string, slc any) *core.Table {
	tabs := br.Tabs()
	tab := tabs.RecycleTab(label)
	if tab.HasChildren() {
		tv := tab.Child(0).(*core.Table)
		tv.SetSlice(slc)
		return tv
	}
	tv := core.NewTable(tab)
	tv.SetReadOnlyMultiSelect(true)
	tv.Styler(func(s *styles.Style) {
		s.SetReadOnly(true) // todo: not taking effect
	})
	tv.SetSlice(slc)
	br.Update()
	return tv
}

// NewTabPlot creates a tab with a SubPlot PlotEditor.
// Set the table and call br.Update after this.
func (br *Browser) NewTabPlot(label string) *plotcore.PlotEditor {
	tabs := br.Tabs()
	tab := tabs.RecycleTab(label)
	if tab.HasChildren() {
		pl := tab.Child(0).AsTree().Child(1).(*plotcore.PlotEditor)
		return pl
	}
	pl := plotcore.NewSubPlot(tab)
	return pl
}

// NewTabEditorString opens an editor tab to display given string
func (br *Browser) NewTabEditorString(label, content string) *texteditor.Editor {
	tabs := br.Tabs()
	tab := tabs.RecycleTab(label)
	if tab.HasChildren() {
		ed := tab.Child(0).(*texteditor.Editor)
		ed.Buffer.SetText([]byte(content))
		return ed
	}
	ed := texteditor.NewEditor(tab)
	ed.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	ed.Buffer.SetText([]byte(content))
	br.Update()
	return ed
}

// FormatTableFromCSV formats the columns of the given table according to the
// Name, Type values in given format CSV file.
func (br *Browser) FormatTableFromCSV(dt *table.Table, format string) error {
	ft := table.NewTable()
	if err := errors.Log(ft.OpenCSV(core.Filename(format), table.Comma)); err != nil {
		return err
	}
	// todo: need a config mode for this!
	for i := range ft.Rows {
		name := ft.StringValue("Name", i)
		typ := ft.StringValue("Type", i)
		switch typ {
		case "string":
			dt.AddStringColumn(name)
		case "time":
			dt.AddIntColumn(name)
		}
	}
	return nil
}

// OpenTOML opens given .toml formatted file with name = value
// entries, as a map.
func (br *Browser) OpenTOML(filename string) (map[string]string, error) {
	md := make(map[string]string)
	err := tomlx.Open(&md, filename)
	errors.Log(err)
	return md, err
}

// TableWithNewKeyColumns returns a copy of the Table with new columns
// having given values, inserted at the start, used as legend keys etc.
// args are column name, value pairs.
func (br *Browser) TableWithNewKeyColumns(dt *table.Table, args ...string) *table.Table {
	n := len(args)
	if n%2 != 0 {
		fmt.Println("TableWithNewColumns requires even number of args as colnm, value pairs")
		return dt
	}
	c := dt.Clone()
	nc := n / 2
	for j := 0; j < nc; j++ {
		colNm := args[2*j]
		val := args[2*j+1]
		col := tensor.NewString([]int{c.Rows})
		c.InsertColumn(col, colNm, 0)
		for i := range col.Values {
			col.Values[i] = val
		}
	}
	return c
}

// FirstComment returns the first comment lines from given .cosh file,
// which is used to set the tooltip for scripts.
func FirstComment(sc string) string {
	sl := cosh.SplitLines(sc)
	cmt := ""
	for _, l := range sl {
		if !strings.HasPrefix(l, "// ") {
			return cmt
		}
		cmt += strings.TrimSpace(l[3:]) + " "
	}
	return cmt
}
