// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"reflect"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type task struct {
	Name     string
	Done     bool
	Hours    float32
	Children []*task
}

func testTasks() []*task {
	return []*task{
		{Name: "Work", Children: []*task{
			{Name: "Email", Done: true, Hours: 1},
			{Name: "Report", Hours: 3, Children: []*task{
				{Name: "Outline", Hours: 1},
			}},
		}},
		{Name: "Home", Children: []*task{
			{Name: "Dishes", Hours: 0.5},
		}},
	}
}

// rowNames returns the names of the visible items, with two spaces of
// indentation for each level.
func rowNames(tt *TreeTable) []string {
	names := make([]string, len(tt.rows))
	for i, r := range tt.rows {
		for range r.depth {
			names[i] += "  "
		}
		names[i] += r.item.Elem().FieldByName("Name").String()
	}
	return names
}

func TestTreeTable(t *testing.T) {
	b := NewBody()
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	assert.Equal(t, []string{"Work", "  Email", "  Report", "    Outline", "Home", "  Dishes"}, rowNames(tt))
	assert.Equal(t, 3, tt.numVisibleFields, "no column for the children")
	tt.Close(2)
	b.AssertRender(t, "tree-table/basic")
}

func TestTreeTableReadOnly(t *testing.T) {
	b := NewBody()
	tasks := testTasks()
	NewTreeTable(b).SetItems(&tasks).SetReadOnly(true)
	b.AssertRender(t, "tree-table/read-only")
}

func TestTreeTableOpenClose(t *testing.T) {
	tasks := testTasks()
	tt := NewTreeTable().SetItems(&tasks)
	tt.SelectedIndex = 4 // Home
	tt.SelectIndex(4)
	tt.Close(0)
	assert.Equal(t, []string{"Work", "Home", "  Dishes"}, rowNames(tt))
	assert.Equal(t, 1, tt.SelectedIndex, "the selection stays with Home")
	assert.True(t, tt.indexIsSelected(1))
	tt.Close(2) // a leaf
	assert.False(t, tt.IsClosed(2))
	tt.CloseAll()
	assert.Equal(t, []string{"Work", "Home"}, rowNames(tt))
	tt.OpenAll()
	assert.Len(t, tt.rows, 6)
}

func TestTreeTableEdit(t *testing.T) {
	tasks := testTasks()
	tt := NewTreeTable().SetItems(&tasks)

	tt.InsertAfter(1) // after Email
	tt.rows[2].item.Interface().(*task).Name = "New"
	assert.Equal(t, []string{"Work", "  Email", "  New", "  Report", "    Outline", "Home", "  Dishes"}, rowNames(tt))
	assert.Equal(t, 2, tt.SelectedIndex)

	tt.Indent(2) // into Email
	assert.Equal(t, []string{"Work", "  Email", "    New", "  Report", "    Outline", "Home", "  Dishes"}, rowNames(tt))
	tt.Outdent(2)
	tt.Outdent(2)
	assert.Equal(t, []string{"Work", "  Email", "  Report", "    Outline", "New", "Home", "  Dishes"}, rowNames(tt))
	tt.MoveItemDown(4)
	assert.Equal(t, []string{"Work", "  Email", "  Report", "    Outline", "Home", "  Dishes", "New"}, rowNames(tt))
	tt.Indent(0) // nothing before it
	assert.Len(t, tasks, 3)

	tt.AddChild(6)
	assert.Equal(t, []string{"New", "  "}, rowNames(tt)[6:])
	tt.DeleteAt(6)
	assert.Equal(t, []string{"Work", "  Email", "  Report", "    Outline", "Home", "  Dishes"}, rowNames(tt))
	assert.Len(t, tasks, 2)

	tt.Duplicate() // nothing selected
	tt.SelectIndex(2) // Report
	tt.Duplicate()
	assert.Equal(t, []string{"Work", "  Email", "  Report", "    Outline", "  Report", "    Outline", "Home", "  Dishes"}, rowNames(tt))
	assert.NotSame(t, tasks[0].Children[1], tasks[0].Children[2], "a copy")
}

func TestTreeTableSort(t *testing.T) {
	tasks := testTasks()
	tt := NewTreeTable().SetItems(&tasks)
	tt.sortIndex = 0
	tt.sortDescending = true
	tt.sortItems()
	tt.updateRows()
	assert.Equal(t, []string{"Work", "  Report", "    Outline", "  Email", "Home", "  Dishes"}, rowNames(tt))
}

// TestTreeTableMove tests moving items by copying them and deleting
// the dragged items, as dragging does.
func TestTreeTableMove(t *testing.T) {
	tasks := testTasks()
	tt := NewTreeTable().SetItems(&tasks)
	tt.SelectIndex(2) // Report
	tt.SelectIndex(3) // Outline, within Report
	md := tt.CopySelectToMime()
	assert.Len(t, md, 1, "only Report, with Outline")
	tt.pasteChildren(md, 5) // into Dishes
	tt.dropDeleteItems()
	assert.Equal(t, []string{"Work", "  Email", "Home", "  Dishes", "    Report", "      Outline"}, rowNames(tt))
}

// TestTreeTableStructOf tests items of a struct type made with
// [reflect.StructOf], whose children are a []any.
func TestTreeTableStructOf(t *testing.T) {
	typ := reflect.StructOf([]reflect.StructField{
		{Name: "Name", Type: reflect.TypeFor[string]()},
		{Name: "Count", Type: reflect.TypeFor[int](), Tag: `default:"2"`},
		{Name: "Closed", Type: reflect.TypeFor[bool](), Tag: `treetable:"closed"`},
		{Name: "Children", Type: reflect.TypeFor[[]any](), Tag: `display:"-"`},
	})
	items := reflect.New(reflect.SliceOf(reflect.PointerTo(typ)))
	b := NewBody()
	tt := NewTreeTable(b).SetItems(items.Interface())
	assert.Equal(t, 2, tt.numVisibleFields)

	tt.NewAt(-1)
	tt.AddChild(0)
	tt.AddChild(1)
	assert.Equal(t, 2, tt.rows[2].depth)
	assert.Equal(t, int64(2), tt.rows[2].item.Elem().Field(1).Int(), "the default")
	tt.Close(0)
	assert.True(t, tt.rows[0].item.Elem().Field(2).Bool(), "the closed field")
	assert.Len(t, tt.rows, 1)

	tt.SelectIndex(0)
	md := tt.CopySelectToMime()
	tt.PasteAtIndex(md, 0)
	assert.Equal(t, 2, items.Elem().Len())
	tt.Open(0)
	assert.Equal(t, []int{0, 1, 2, 0}, []int{tt.rows[0].depth, tt.rows[1].depth, tt.rows[2].depth, tt.rows[3].depth})
	assert.Equal(t, typ, tt.rows[2].item.Elem().Type(), "the grandchild is an item, not a map")
	b.AssertRender(t, "tree-table/struct-of")
}

// TestTreeTableDragDrop tests dragging an item by its tree cell with the
// mouse and moving it into another item (with Shift), with the menu of
// the drop.
func TestTreeTableDragDrop(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(600), units.Dp(500)) })
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b.AssertRender(t, "tree-table/drag-drop", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		b.AsyncLock()
		cell := func(row int) *WidgetBase {
			nw, _ := tt.RowWidgetNs()
			return tt.ListGrid.Child(row * nw).(Widget).AsWidget()
		}
		email := cell(1).Geom.TotalBBox
		from := image.Pt(email.Min.X+4, (email.Min.Y+email.Max.Y)/2) // in the indentation
		dishes := cell(5).Geom.TotalBBox
		to := dishes.Min.Add(dishes.Size().Div(2))
		b.AsyncUnlock()

		var shift key.Modifiers
		shift.SetFlag(true, key.Shift)
		src.MouseButton(events.MouseDown, events.Left, from, shift)
		for i := 1; i <= 10; i++ {
			src.MouseMove(from.Add(to.Sub(from).Mul(i).Div(10)))
			time.Sleep(20 * time.Millisecond)
		}
		src.MouseButton(events.MouseUp, events.Left, to, shift)

		var menu *Stage
		for range 100 {
			b.AsyncLock()
			menu = b.Scene.Stage.popups.top()
			b.AsyncUnlock()
			if menu != nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		require.NotNil(t, menu, "the drop menu")
		b.AsyncLock()
		var child *Button
		menu.Scene.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
			if bt, ok := cw.(*Button); ok && bt.Text == "Insert as child" {
				child = bt
			}
			return child == nil
		})
		require.NotNil(t, child)
		child.Send(events.Click)
		b.AsyncUnlock()
		time.Sleep(100 * time.Millisecond)
		b.AsyncLock()
		assert.Equal(t, []string{"Work", "  Report", "    Outline", "Home", "  Dishes", "    Email"}, rowNames(tt))
		b.AsyncUnlock()
	})
}

// TestTreeTableBranchKeys tests opening and closing items by clicking
// their branches, and with the arrow keys when the tree table is read-only.
func TestTreeTableBranchKeys(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(600), units.Dp(500)) })
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	tt.SetReadOnly(true)
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b.AssertRender(t, "tree-table/branch-keys", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		center := func(w *WidgetBase) image.Point {
			bb := w.Geom.TotalBBox
			return bb.Min.Add(bb.Size().Div(2))
		}
		click := func(pt image.Point) {
			src.MouseButton(events.MouseDown, events.Left, pt, 0)
			src.MouseButton(events.MouseUp, events.Left, pt, 0)
			time.Sleep(200 * time.Millisecond)
		}
		press := func(code key.Codes) {
			src.Key(events.KeyDown, 0, code, 0)
			src.Key(events.KeyUp, 0, code, 0)
			time.Sleep(200 * time.Millisecond)
		}
		b.AsyncLock()
		branch := center(tt.ListGrid.Child(0).AsTree().Child(0).(Widget).AsWidget())
		b.AsyncUnlock()
		click(branch)
		b.AsyncLock()
		assert.Equal(t, []string{"Work", "Home", "  Dishes"}, rowNames(tt), "click the branch")
		tt.Open(0) // not by clicking again, which would be a double click
		tt.Update()
		b.AsyncUnlock()
		time.Sleep(200 * time.Millisecond)

		b.AsyncLock()
		nw, _ := tt.RowWidgetNs()
		report := center(tt.ListGrid.Child(2 * nw).AsTree().Child(1).(Widget).AsWidget())
		b.AsyncUnlock()
		click(report)
		press(key.CodeLeftArrow)
		b.AsyncLock()
		assert.True(t, tt.ContainsFocus())
		assert.Equal(t, 2, tt.SelectedIndex)
		assert.True(t, tt.IsClosed(2), "left closes")
		b.AsyncUnlock()
		press(key.CodeLeftArrow)
		b.AsyncLock()
		assert.Equal(t, 0, tt.SelectedIndex, "left again goes to the parent")
		b.AsyncUnlock()
		press(key.CodeDownArrow)
		press(key.CodeDownArrow)
		press(key.CodeRightArrow)
		b.AsyncLock()
		assert.Equal(t, 2, tt.SelectedIndex)
		assert.False(t, tt.IsClosed(2), "right opens")
		b.AsyncUnlock()
	})
}

// TestTreeTableClickSelect tests that clicking into the text field of a
// row selects it, as does moving the focus to a row, so that the keys
// insert items there.
func TestTreeTableClickSelect(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(600), units.Dp(500)) })
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b.AssertRender(t, "tree-table/click-select", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		insertAfter := func() { // with the chord of the active key map
			for ch, kf := range *keymap.ActiveMap {
				if kf != keymap.InsertAfter {
					continue
				}
				r, code, mods, err := ch.Decode()
				require.NoError(t, err)
				src.Key(events.KeyDown, r, code, mods)
				src.Key(events.KeyUp, r, code, mods)
				time.Sleep(200 * time.Millisecond)
				return
			}
			t.Fatal("no chord for InsertAfter")
		}
		nw, _ := tt.RowWidgetNs()
		value := func(row int) *WidgetBase { // the text field of the first field
			cell := tt.ListGrid.Child(row * nw).AsTree()
			return cell.Child(cell.NumChildren() - 1).(Widget).AsWidget()
		}
		b.AsyncLock()
		bb := value(3).Geom.TotalBBox // Outline
		b.AsyncUnlock()
		pt := bb.Min.Add(bb.Size().Div(2))
		src.MouseButton(events.MouseDown, events.Left, pt, 0)
		src.MouseButton(events.MouseUp, events.Left, pt, 0)
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		assert.Equal(t, 3, tt.SelectedIndex, "the clicked row")
		assert.True(t, value(3).StateIs(states.Focused) || value(3).ContainsFocus())
		b.AsyncUnlock()
		insertAfter()
		b.AsyncLock()
		assert.Equal(t, []string{"Work", "  Email", "  Report", "    Outline", "    ", "Home", "  Dishes"}, rowNames(tt))

		value(5).SetFocus() // Home, as with the tab key
		b.AsyncUnlock()
		insertAfter()
		b.AsyncLock()
		assert.Equal(t, []string{"Work", "  Email", "  Report", "    Outline", "    ", "Home", "  Dishes", ""}, rowNames(tt))
		b.AsyncUnlock()
	})
}
