// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"reflect"
	"testing"
)

func TestTreeTableView(t *testing.T) {
	//todo make a func for save canvas to png ?
	b := gi.NewAppBody("TreeTableView")
	treeTableView := NewTreeTableView(b)
	rows := mockRows(treeTableView)
	treeTableView.SetSlice(rows)
	b.RunMainWindow()
}

type demoRow struct {
	table        *TreeTableView
	parent       *demoRow
	id           uuid.UUID
	text         string
	text2        string
	children     []*demoRow
	checkbox     *gi.Switches
	container    bool
	open         bool
	doubleHeight bool
}

func mockRows(table *TreeTableView) []*demoRow {
	rows := make([]*demoRow, 100)
	for i := range rows {
		row := &demoRow{
			table: table,
			id:    uuid.New(),
			text:  fmt.Sprintf("Row %d", i+1),
			text2: fmt.Sprintf("Some longer content for Row %d", i+1),
		}
		if i%10 == 3 {
			if i == 3 {
				row.doubleHeight = true
			}
			row.container = true
			row.open = true
			row.children = make([]*demoRow, 5)
			for j := range row.children {
				child := &demoRow{
					table:  table,
					parent: row,
					id:     uuid.New(),
					text:   fmt.Sprintf("Sub Row %d", j+1),
				}
				row.children[j] = child
				if j < 2 {
					child.container = true
					child.open = true
					child.children = make([]*demoRow, 2)
					for k := range child.children {
						child.children[k] = &demoRow{
							table:  table,
							parent: child,
							id:     uuid.New(),
							text:   fmt.Sprintf("Sub Sub Row %d", k+1),
						}
					}
				}
			}
		}
		rows[i] = row
	}
	return rows
}

// delete
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
