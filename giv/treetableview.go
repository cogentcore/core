// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/styles"
	"math/rand"
	"reflect"
)

type (
	TableRowData interface {
		OnInit()
		SetStyles()
		SetSlice(sl any) *TableView
		StructType() reflect.Type
		CacheVisFields()
		ConfigWidget()
		ConfigTableView()
		ConfigFrame()
		ConfigHeader()
		SliceGrid() *SliceViewGrid
		SliceHeader() *gi.Frame
		RowWidgetNs() (nWidgPerRow int, idxOff int)
		ConfigRows()
		UpdateWidgets()
		StyleRow(w gi.Widget, idx int, fidx int)
		SliceNewAt(idx int)
		SliceDeleteAt(idx int)
		SortSlice()
		SortSliceAction(fldIdx int)
		SortFieldName() string
		SetSortFieldName(nm string)
		RowFirstVisWidget(row int) (*gi.WidgetBase, bool)
		RowGrabFocus(row int) *gi.WidgetBase
		SelectRowWidgets(row int, sel bool)
		SelectFieldVal(fld string, val string) bool
		EditIdx(idx int)
		ContextMenu(m *gi.Scene)
		SizeFinal()
	}
)

type TreeTableView struct { //gti:add

	TreeHeaderFrame *gi.Frame
	TreeView        *TreeView

	SliceViewBase

	// optional styling function
	StyleFunc TableViewStyleFunc `copier:"-" view:"-" json:"-" xml:"-"`

	// current selection field -- initially select value in this field
	SelField string `copier:"-" view:"-" json:"-" xml:"-"`

	// current sort index
	SortIdx int

	// whether current sort order is descending
	SortDesc bool

	// struct type for each row
	StruType reflect.Type `copier:"-" view:"-" json:"-" xml:"-"`

	// the visible fields
	VisFields []reflect.StructField `copier:"-" view:"-" json:"-" xml:"-"`

	// number of visible fields
	NVisFields int `copier:"-" view:"-" json:"-" xml:"-"`

	// HeaderWidths has number of characters in each header, per visfields
	HeaderWidths []int `copier:"-" view:"-" json:"-" xml:"-"`
}

//go:generate core generate

var _ SliceViewer = (*TreeTableView)(nil)

func (t *TreeTableView) OnInit() {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SetStyles() {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SetSlice(sl any) *TableView {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) StructType() reflect.Type {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) CacheVisFields() {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) ConfigWidget() {
	hSplits := NewHSplits(t)
	treeFrame := gi.NewFrame(hSplits)  //left
	tableFrame := gi.NewFrame(hSplits) //right
	hSplits.SetSplits(.2, .8)

	treeFrame.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	treeHeaderFrame := gi.NewFrame(treeFrame) //treeHeader for align table header
	treeHeaderFrame.Style(func(s *styles.Style) {
		s.Direction = styles.Row
	})
	gi.NewTextField(treeHeaderFrame).SetPlaceholder("filter content")
	gi.NewButton(treeHeaderFrame).SetIcon("hierarchy")
	gi.NewButton(treeHeaderFrame).SetIcon("circled_add")
	gi.NewButton(treeHeaderFrame).SetIcon("trash")
	gi.NewButton(treeHeaderFrame).SetIcon("star")

	treeView := NewTreeView(treeFrame)
	treeView.IconOpen = icons.ExpandCircleDown
	treeView.IconClosed = icons.ExpandCircleRight
	treeView.IconLeaf = icons.Blank
}

func (t *TreeTableView) ConfigTableView() {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) ConfigFrame() {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) ConfigHeader() {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SliceGrid() *SliceViewGrid {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SliceHeader() *gi.Frame {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) RowWidgetNs() (nWidgPerRow int, idxOff int) {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) ConfigRows() {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) UpdateWidgets() {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) StyleRow(w gi.Widget, idx int, fidx int) {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SliceNewAt(idx int) {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SliceDeleteAt(idx int) {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SortSlice() {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SortSliceAction(fldIdx int) {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SortFieldName() string {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SetSortFieldName(nm string) {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) RowFirstVisWidget(row int) (*gi.WidgetBase, bool) {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) RowGrabFocus(row int) *gi.WidgetBase {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SelectRowWidgets(row int, sel bool) {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SelectFieldVal(fld string, val string) bool {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) EditIdx(idx int) {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) ContextMenu(m *gi.Scene) {
	//TODO implement me
	panic("implement me")
}

func (t *TreeTableView) SizeFinal() {
	//TODO implement me
	panic("implement me")
}

// TreeTable todo set struct or dynamic creat node
func TreeTable(b *gi.Body, nodes []any) {
	hSplits := NewHSplits(b)
	treeFrame := gi.NewFrame(hSplits)  //left
	tableFrame := gi.NewFrame(hSplits) //right
	hSplits.SetSplits(.2, .8)

	treeFrame.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	treeHeaderFrame := gi.NewFrame(treeFrame) //treeHeader for align table header
	treeHeaderFrame.Style(func(s *styles.Style) {
		s.Direction = styles.Row
	})
	gi.NewTextField(treeHeaderFrame).SetPlaceholder("filter content")
	gi.NewButton(treeHeaderFrame).SetIcon("hierarchy")
	gi.NewButton(treeHeaderFrame).SetIcon("circled_add")
	gi.NewButton(treeHeaderFrame).SetIcon("trash")
	gi.NewButton(treeHeaderFrame).SetIcon("star")

	treeView := NewTreeView(treeFrame)
	treeView.IconOpen = icons.ExpandCircleDown
	treeView.IconClosed = icons.ExpandCircleRight
	treeView.IconLeaf = icons.Blank

	//todo merge struct field
	for _, node := range nodes {
		fields := reflect.VisibleFields(reflect.TypeOf(node))
		for _, field := range fields {
			switch field.Type.Kind() {
			case reflect.Struct: //render tree
			case reflect.Pointer:
				reflect.Indirect(reflect.ValueOf(field)) //todo
			case reflect.Slice: //render indent and elem to table row
				//gi.NewSpace(field) //row 是水平布局全部cell
			case reflect.Array: //render indent and elem to table row
				//gi.NewSpace(field)
			}
		}
	}

	MakeTree(treeView, 0, 3, 5)
	tableView := NewTableView(tableFrame)

	tableView.SetReadOnly(true)
	tableView.SetSlice(&nodes)
}

// MakeTree todo remove
func MakeTree(tv *TreeView, iter, maxIter, maxKids int) {
	if iter > maxIter {
		return
	}
	n := rand.Intn(maxKids)
	if iter == 0 {
		n = maxKids
	}
	iter++
	parnm := tv.Name() + "_"
	tv.SetNChildren(n, TreeViewType, parnm+"ch")
	for j := 0; j < n; j++ {
		kt := tv.Child(j).(*TreeView)
		MakeTree(kt, iter, maxIter, maxKids)
	}
}

// util
func NewHSplits(parent ki.Ki) *gi.Splits { return newSplits(parent, true) }
func NewVSplits(parent ki.Ki) *gi.Splits { return newSplits(parent, false) }

func newSplits(parent ki.Ki, isHorizontal bool) *gi.Splits { // Horizontal and vertical
	splits := gi.NewSplits(parent)
	splits.Style(func(s *styles.Style) {
		if isHorizontal {
			s.Direction = styles.Row
		} else {
			s.Direction = styles.Column
		}
		s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
	})
	return splits
}
