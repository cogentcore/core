package giv

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/styles"
	"math/rand"
)

// TreeTable todo set struct or dynamic creat node
func TreeTable(b *gi.Body) {
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
	MakeTree(treeView, 0, 3, 5)
	tableView := NewTableView(tableFrame)

	//mock
	nodes := make([]*Node, 0)
	for i := 0; i < 10; i++ {
		nodes = append(nodes, &Node{
			Column1: "Column1",
			Column2: "Column2",
			Column3: "Column3",
			Column4: "Column4",
			Nested: &Node{
				Column1: "Column1",
				Column2: "Column2",
				Column3: "Column3",
				Column4: "Column4",
				Nested:  nil,
				Nested2: nil,
			},
			Nested2: &Node{
				Column1: "Column1",
				Column2: "Column2",
				Column3: "Column3",
				Column4: "Column4",
				Nested:  nil,
				Nested2: nil,
			},
		})
	}
	tableView.SetReadOnly(true)
	tableView.SetSlice(&nodes)
}

// Node todo move to input arg
type Node struct {
	Column1 string
	Column2 string
	Column3 string
	Column4 string
	Nested  *Node
	Nested2 *Node
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
