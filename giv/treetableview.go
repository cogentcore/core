// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"math/rand"
	"reflect"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/styles"
)

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
			case reflect.Array: //render indent and elem to table row

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
func NewHSplits(par ki.Ki) *gi.Splits { return newSplits(par, true) }
func NewVSplits(par ki.Ki) *gi.Splits { return newSplits(par, false) }

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
