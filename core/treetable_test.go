// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
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

	tt.Duplicate()    // nothing selected
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

// TestTreeTableContextMenu tests that right clicks on the tree cell
// outside of its value, and on its branch, select the row and show the
// context menu of the tree table.
func TestTreeTableContextMenu(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(600), units.Dp(500)) })
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b.AssertRender(t, "tree-table/context-menu", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		b.AsyncLock()
		nw, _ := tt.RowWidgetNs()
		cell := tt.ListGrid.Child(1 * nw).(Widget).AsWidget().Geom.TotalBBox
		branch := tt.ListGrid.Child(2 * nw).AsTree().Child(0).(Widget).AsWidget().Geom.TotalBBox
		b.AsyncUnlock()
		for _, c := range []struct {
			name string
			pt   image.Point
			row  int
		}{
			{"indentation", image.Pt(cell.Min.X+4, (cell.Min.Y+cell.Max.Y)/2), 1},
			{"branch", branch.Min.Add(branch.Size().Div(2)), 2},
		} {
			src.MouseButton(events.MouseDown, events.Right, c.pt, 0)
			src.MouseButton(events.MouseUp, events.Right, c.pt, 0)
			time.Sleep(300 * time.Millisecond)
			b.AsyncLock()
			st := b.Scene.Stage.popups.top()
			require.NotNil(t, st, c.name)
			hasInsert := false
			st.Scene.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
				if bt, ok := cw.(*Button); ok && bt.Text == "Insert after" {
					hasInsert = true
				}
				return !hasInsert
			})
			assert.True(t, hasInsert, "%s: the menu of the tree table", c.name)
			assert.Equal(t, c.row, tt.SelectedIndex, c.name)
			st.ClosePopup()
			b.AsyncUnlock()
			time.Sleep(200 * time.Millisecond)
		}
	})
}

// TestTreeTableAcceptDelete tests that the delete key deletes a character
// in a text field, and the item after its edits are accepted.
func TestTreeTableAcceptDelete(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(600), units.Dp(500)) })
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b.AssertRender(t, "tree-table/accept-delete", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		press := func(kf keymap.Functions) { // with the chord of the active key map
			for ch, f := range *keymap.ActiveMap {
				if f != kf {
					continue
				}
				r, code, mods, err := ch.Decode()
				require.NoError(t, err)
				src.Key(events.KeyDown, r, code, mods)
				src.Key(events.KeyUp, r, code, mods)
				time.Sleep(200 * time.Millisecond)
				return
			}
			t.Fatal("no chord for", kf)
		}
		nw, _ := tt.RowWidgetNs()
		b.AsyncLock()
		cell := tt.ListGrid.Child(1 * nw).AsTree()
		tf := cell.Child(cell.NumChildren() - 1).(*TextField) // Email
		bb := tf.Geom.TotalBBox
		b.AsyncUnlock()
		pt := image.Pt(bb.Min.X+8, (bb.Min.Y+bb.Max.Y)/2) // before the text
		src.MouseButton(events.MouseDown, events.Left, pt, 0)
		src.MouseButton(events.MouseUp, events.Left, pt, 0)
		time.Sleep(200 * time.Millisecond)

		press(keymap.Delete)
		b.AsyncLock()
		assert.Equal(t, "mail", tf.Text(), "a character is deleted while editing")
		b.AsyncUnlock()
		press(keymap.Accept)
		b.AsyncLock()
		assert.Equal(t, "mail", tasks[0].Children[0].Name, "the edit is accepted")
		b.AsyncUnlock()
		press(keymap.Delete)
		b.AsyncLock()
		assert.Equal(t, []string{"Work", "  Report", "    Outline", "Home", "  Dishes"}, rowNames(tt), "the item is deleted")
		b.AsyncUnlock()
	})
}

// TestTreeTableDragMoves tests that dragging from a tree table with
// DragMoves moves without modifier keys and copies with Shift, and that
// the drag image shows whether the drop would copy.
func TestTreeTableDragMoves(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(600), units.Dp(500)) })
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	tt.DragMoves = true
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b.AssertRender(t, "tree-table/drag-moves", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		nw, _ := tt.RowWidgetNs()
		drag := func(from, to int, mods key.Modifiers) (dragCopy bool) {
			b.AsyncLock()
			fbb := tt.ListGrid.Child(from * nw).(Widget).AsWidget().Geom.TotalBBox
			tbb := tt.ListGrid.Child(to * nw).(Widget).AsWidget().Geom.TotalBBox
			b.AsyncUnlock()
			fp := image.Pt(fbb.Min.X+4, (fbb.Min.Y+fbb.Max.Y)/2)
			tp := tbb.Min.Add(tbb.Size().Div(2))
			src.MouseButton(events.MouseDown, events.Left, fp, mods)
			for i := 1; i <= 10; i++ {
				src.MouseMove(fp.Add(tp.Sub(fp).Mul(i).Div(10)))
				time.Sleep(20 * time.Millisecond)
			}
			b.AsyncLock()
			dragCopy = b.Scene.Events.dragCopy
			b.AsyncUnlock()
			src.MouseButton(events.MouseUp, events.Left, tp, mods)
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
			return
		}
		assert.False(t, drag(1, 5, 0), "no plus for a move") // Email into Dishes
		b.AsyncLock()
		assert.Equal(t, []string{"Work", "  Report", "    Outline", "Home", "  Dishes", "    Email"}, rowNames(tt), "moved")
		b.AsyncUnlock()
		var shift key.Modifiers
		shift.SetFlag(true, key.Shift)
		assert.True(t, drag(4, 1, shift), "a plus for a copy") // Dishes into Report
		b.AsyncLock()
		assert.Equal(t, []string{"Work", "  Report", "    Outline", "    Dishes", "      Email", "Home", "  Dishes", "    Email"}, rowNames(tt), "copied")
		b.AsyncUnlock()
	})
}

func TestTreeTableColumnWidths(t *testing.T) {
	b := NewBody()
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	tt.ColumnWidths = map[string]float32{"Name": 90, "Hours": 130}
	b.AssertRender(t, "tree-table/column-widths", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		dots := tt.Styles.UnitContext.Dots(units.UnitDp)
		assert.InDelta(t, 90*dots, tt.columnDots(0), 1)
		assert.InDelta(t, 130*dots, tt.columnDots(2), 1)
	})
}

// TestTreeTableColumnFill tests that the values of the tree cells fill
// the width of the tree column when it is set wider than they would be.
func TestTreeTableColumnFill(t *testing.T) {
	b := NewBody()
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	tt.ColumnWidths = map[string]float32{"Name": 600}
	b.AssertRender(t, "tree-table/column-fill", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		nw, _ := tt.RowWidgetNs()
		for row := range 4 {
			cell := AsWidget(tt.ListGrid.Child(row * nw))
			value := AsWidget(cell.Child(cell.NumChildren() - 1))
			assert.InDelta(t, cell.Geom.TotalBBox.Max.X, value.Geom.TotalBBox.Max.X, 2, "row %d", row)
		}
	})
}

// TestTreeTableFewerFields tests setting the items of a tree table
// to those of a type with fewer fields while it is shown, from its
// updater, as an app does when the fields of the type change.
func TestTreeTableFewerFields(t *testing.T) {
	b := NewBody()
	tasks := testTasks()
	type short struct { // without Done, before Hours
		Name     string
		Hours    float32
		Children []*short
	}
	shorts := []*short{{Name: "a", Children: []*short{{Name: "b"}}}}
	var items any = &tasks
	tt := NewTreeTable(b)
	tt.Updater(func() {
		tt.SetItems(items)
	})
	b.AssertRender(t, "tree-table/fewer-fields", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		items = &shorts
		tt.Update()
		tt.Restyle()
		assert.Equal(t, 2, tt.numVisibleFields)
		assert.Equal(t, []string{"a", "  b"}, rowNames(tt))
	})
}

// TestTreeTableInsertChild tests adding a child with the keys of
// [keymap.InsertChild], while editing the text field of an item.
func TestTreeTableInsertChild(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(600), units.Dp(500)) })
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b.AssertRender(t, "tree-table/insert-child", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		nw, _ := tt.RowWidgetNs()
		b.AsyncLock()
		cell := tt.ListGrid.Child(1 * nw).AsTree() // Email
		bb := cell.Child(cell.NumChildren() - 1).(Widget).AsWidget().Geom.TotalBBox
		b.AsyncUnlock()
		pt := bb.Min.Add(bb.Size().Div(2))
		src.MouseButton(events.MouseDown, events.Left, pt, 0)
		src.MouseButton(events.MouseUp, events.Left, pt, 0)
		time.Sleep(200 * time.Millisecond)
		n := 0
		for ch, kf := range *keymap.ActiveMap {
			if kf != keymap.InsertChild {
				continue
			}
			r, code, mods, err := ch.Decode()
			require.NoError(t, err)
			src.Key(events.KeyDown, r, code, mods)
			src.Key(events.KeyUp, r, code, mods)
			time.Sleep(200 * time.Millisecond)
			n++
			b.AsyncLock()
			assert.Len(t, tasks[0].Children[0].Children, n, string(ch))
			nr := tt.rowOf(reflect.ValueOf(tasks[0].Children[0].Children[n-1]))
			assert.Equal(t, nr, tt.SelectedIndex, "the new child is selected")
			b.AsyncUnlock()
			// back to Email, whose text field the keys then go to
			src.MouseButton(events.MouseDown, events.Left, pt, 0)
			src.MouseButton(events.MouseUp, events.Left, pt, 0)
			time.Sleep(200 * time.Millisecond)
		}
		assert.GreaterOrEqual(t, n, 2, "the key map has chords for it")
	})
}

// TestTreeTableSameType tests that setting new items of the same type,
// as after an undo, keeps the selection, the widgets, and the focus.
func TestTreeTableSameType(t *testing.T) {
	b := NewBody()
	tasks := testTasks()
	var items any = &tasks
	tt := NewTreeTable(b)
	tt.Updater(func() {
		tt.SetItems(items)
	})
	b.AssertRender(t, "tree-table/same-type", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		nw, _ := tt.RowWidgetNs()
		cell := tt.ListGrid.Child(2 * nw).AsTree() // Report
		tf := cell.Child(cell.NumChildren() - 1).(*TextField)
		tf.SetFocus()
		tt.SelectIndexEvent(2, events.SelectOne)

		again := testTasks()
		again[0].Children[1].Name = "Report v2"
		items = &again
		tt.Update()
		assert.Equal(t, 2, tt.SelectedIndex)
		assert.True(t, tt.indexIsSelected(2))
		cell = tt.ListGrid.Child(2 * nw).AsTree()
		assert.Same(t, tf, cell.Child(cell.NumChildren()-1), "the same widget")
		assert.Equal(t, "Report v2", tf.Text())
		assert.True(t, tf.StateIs(states.Focused), "with the focus")
	})
}

// TestTreeTableEmpty tests that an empty tree table takes the focus,
// and that the insert keys then add its first item.
func TestTreeTableEmpty(t *testing.T) {
	b := NewBody()
	var tasks []*task
	tt := NewTreeTable(b).SetItems(&tasks)
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b.AssertRender(t, "tree-table/empty", func() {
		b.AsyncLock()
		assert.True(t, tt.CanFocus())
		tt.SetFocus()
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		for ch, kf := range *keymap.ActiveMap {
			if kf != keymap.Insert {
				continue
			}
			r, code, mods, err := ch.Decode()
			require.NoError(t, err)
			src.Key(events.KeyDown, r, code, mods)
			src.Key(events.KeyUp, r, code, mods)
			break
		}
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Len(t, tasks, 1, "the first item")
		assert.False(t, tt.CanFocus(), "the rows have the focus now")
	})
}

// TestTreeTableSortBool tests sorting by a bool column by clicking its
// header, within the children of each item.
func TestTreeTableSortBool(t *testing.T) {
	b := NewBody()
	tasks := testTasks()
	tasks = append(tasks, &task{Name: "Errands", Done: true})
	tt := NewTreeTable(b).SetItems(&tasks)
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b.AssertRender(t, "tree-table/sort-bool", func() {
		b.AsyncLock()
		bb := AsWidget(tt.header.Child(2)).Geom.TotalBBox // the header of Done
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		pt := bb.Min.Add(bb.Size().Div(2))
		src.MouseButton(events.MouseDown, events.Left, pt, 0)
		src.MouseButton(events.MouseUp, events.Left, pt, 0)
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, 1, tt.sortIndex)
		assert.Equal(t, []string{"Work", "  Report", "    Outline", "  Email", "Home", "  Dishes", "Errands"}, rowNames(tt))
	})
}

func TestTreeTableHideGroupValues(t *testing.T) {
	b := NewBody()
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	tt.HideGroupValues = true
	b.AssertRender(t, "tree-table/hide-group-values", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		nw, _ := tt.RowWidgetNs()
		hidden := func(row, col int) bool {
			_, ok := tt.ListGrid.Child(row*nw + col).(*Frame) // empty, instead of the value
			return ok && col > 0
		}
		assert.True(t, hidden(0, 1), "Work, a group")
		assert.True(t, hidden(0, 2))
		assert.False(t, hidden(0, 0), "with its tree")
		assert.False(t, hidden(1, 1), "Email, an item")
		tt.HideGroupValues = false
		tt.Update()
		assert.False(t, hidden(0, 1))
		tt.HideGroupValues = true // for the image
		tt.Update()
	})
}

// TestDragCopySprite tests that the sprite of the green plus on a drag
// shows whether the drop would copy, as the shift key is pressed and
// released during the drag, without moving the mouse.
func TestDragCopySprite(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(600), units.Dp(500)) })
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	tt.DragMoves = true
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b.AssertRender(t, "tree-table/drag-copy-sprite", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		nw, _ := tt.RowWidgetNs()
		b.AsyncLock()
		bb := tt.ListGrid.Child(1 * nw).(Widget).AsWidget().Geom.TotalBBox
		b.AsyncUnlock()
		from := image.Pt(bb.Min.X+4, (bb.Min.Y+bb.Max.Y)/2)
		src.MouseButton(events.MouseDown, events.Left, from, 0)
		time.Sleep(2 * TimingSettings.DragStartTime)
		for i := 1; i <= 5; i++ {
			src.MouseMove(from.Add(image.Pt(0, 10*i)))
			time.Sleep(20 * time.Millisecond)
		}
		copySprite := func() (active bool, pos image.Point) {
			b.AsyncLock()
			defer b.AsyncUnlock()
			ms := b.Scene.Stage.Main
			sp, ok := ms.Sprites.SpriteByName(dragCopySpriteName)
			require.True(t, ok)
			ds, _ := ms.Sprites.SpriteByName(dragSpriteName)
			return sp.Active, sp.EventBBox.Min.Sub(ds.EventBBox.Min)
		}
		active, off := copySprite()
		assert.False(t, active, "a move, without the plus")
		// the key events have NumLock, as on macOS
		var lock, shift key.Modifiers
		lock.SetFlag(true, key.NumLock)
		shift.SetFlag(true, key.Shift, key.NumLock)
		src.Key(events.KeyDown, 0, key.CodeLeftShift, lock)
		time.Sleep(100 * time.Millisecond)
		active, _ = copySprite()
		assert.True(t, active, "a copy with shift, with the plus")
		src.Key(events.KeyUp, 0, key.CodeLeftShift, shift)
		time.Sleep(100 * time.Millisecond)
		active, _ = copySprite()
		assert.False(t, active, "a move again")
		assert.Equal(t, 0, off.X, "at the start of the drag image")
		src.MouseButton(events.MouseUp, events.Left, from.Add(image.Pt(0, 50)), 0)
		time.Sleep(100 * time.Millisecond)
		b.AsyncLock()
		defer b.AsyncUnlock()
		for range 50 { // the drop menu
			if st := b.Scene.Stage.popups.top(); st != nil {
				st.ClosePopup()
				break
			}
		}
	})
}

// TestTreeTableColumnLongText tests that the text fields of the tree cells
// fit in the set width of the tree column, so that they scroll their text,
// for text longer than that.
func TestTreeTableColumnLongText(t *testing.T) {
	long := strings.Repeat("long text ", 20)
	tasks := []*task{{Name: long, Children: []*task{{Name: long}}}}
	b := NewBody()
	tt := NewTreeTable(b).SetItems(&tasks)
	tt.ColumnWidths = map[string]float32{"Name": 200}
	b.AssertRender(t, "tree-table/column-long-text", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		nw, _ := tt.RowWidgetNs()
		for row := range 2 {
			cell := AsWidget(tt.ListGrid.Child(row * nw))
			value := AsWidget(cell.Child(cell.NumChildren() - 1))
			assert.InDelta(t, cell.Geom.TotalBBox.Max.X, value.Geom.TotalBBox.Max.X, 2, "row %d", row)
		}
	})
}

// TestTreeTableInsertSorted tests that inserting an item while the items
// are sorted puts the focus on its row after it is sorted.
func TestTreeTableInsertSorted(t *testing.T) {
	b := NewBody()
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	tt.SetSortFieldName("Done:down") // Email, done, before Report
	b.AssertRender(t, "tree-table/insert-sorted", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tt.Update()
		assert.Equal(t, []string{"Work", "  Email", "  Report", "    Outline", "Home", "  Dishes"}, rowNames(tt))
		tt.InsertBefore(1) // before Email, but not done, so after it
		nw, _ := tt.RowWidgetNs()
		idx := slices.Index(rowNames(tt), "  ")
		assert.Equal(t, 2, idx, "sorted after Email")
		assert.Equal(t, idx, tt.SelectedIndex)
		cell := tt.ListGrid.Child(idx * nw).AsTree()
		assert.True(t, cell.Child(cell.NumChildren()-1).(Widget).AsWidget().StateIs(states.Focused), "the focus is on its row")
	})
}

// TestTreeTableShowItem tests filtering the items with ShowItem.
func TestTreeTableShowItem(t *testing.T) {
	tasks := testTasks()
	tt := NewTreeTable().SetItems(&tasks)
	tt.ShowItem = func(item any) bool { return !item.(*task).Done }
	tt.updateRows()
	assert.Equal(t, []string{"Work", "  Report", "    Outline", "Home", "  Dishes"}, rowNames(tt), "without Email, which is done")
	tasks[0].Done = true
	tt.updateRows()
	assert.Equal(t, []string{"Home", "  Dishes"}, rowNames(tt), "without Work, with its children")
	tt.ShowItem = nil
	tt.updateRows()
	assert.Len(t, tt.rows, 6)
}

// TestRowButton tests the button at the end of each row of a [Table] and
// a [TreeTable], which calls its function with the item of its row.
func TestRowButton(t *testing.T) {
	b := NewBody()
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	var clicked *task
	tt.SetRowButton(icons.Edit, "Edit the task", func(item any) {
		clicked = item.(*task)
	})
	b.AssertRender(t, "tree-table/row-button", func() {
		b.AsyncLock()
		nw, idxOff := tt.RowWidgetNs()
		assert.Equal(t, tt.numVisibleFields+1, nw-idxOff, "the button after the values")
		bt, ok := tt.ListGrid.Child(2*nw + nw - 1).(*Button) // of the third row
		require.True(t, ok)
		assert.Equal(t, icons.Edit, bt.Icon)
		bb := bt.Geom.TotalBBox
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		pt := bb.Min.Add(bb.Size().Div(2))
		src.MouseButton(events.MouseDown, events.Left, pt, 0)
		src.MouseButton(events.MouseUp, events.Left, pt, 0)
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		defer b.AsyncUnlock()
		require.NotNil(t, clicked)
		assert.Equal(t, "Report", clicked.Name)

		tt.SetRowButton("", "", nil)
		nw2, _ := tt.RowWidgetNs()
		assert.Equal(t, nw-1, nw2, "removed")
	})
}

// TestTableShowField tests showing only some of the fields of the items
// of a [TreeTable], with [Table.ShowField].
func TestTableShowField(t *testing.T) {
	b := NewBody()
	tasks := testTasks()
	tt := NewTreeTable(b)
	tt.ShowField = func(field reflect.StructField) bool {
		return field.Name != "Hours"
	}
	tt.SetItems(&tasks)
	b.AssertRender(t, "tree-table/show-field", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		names := []string{}
		for _, f := range tt.visibleFields {
			names = append(names, f.Name)
		}
		assert.Equal(t, []string{"Name", "Done"}, names)
	})
}

// TestTreeTableFitWidth tests that the tree column of a [TreeTable] with
// [TreeTable.FitWidth] has the width that the other columns leave, so that
// the table fits in a narrow width without scrolling, with a long value.
func TestTreeTableFitWidth(t *testing.T) {
	for _, hide := range []bool{false, true} {
		t.Run(fmt.Sprintf("hide-group-values-%v", hide), func(t *testing.T) {
			testTreeTableFitWidth(t, hide)
		})
	}
}

// testTreeTableFitWidth tests [TreeTable.FitWidth], with the values of the
// groups hidden if hide is true, as the first row is a group, whose hidden
// values are empty frames, narrower than the values of the other rows.
func testTreeTableFitWidth(t *testing.T, hide bool) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.X.Dp(300)
		s.Max.X.Dp(300)
	})
	tasks := testTasks()
	tasks[0].Name = strings.Repeat("a long name ", 10)
	tt := NewTreeTable(b)
	tt.FitWidth = true
	tt.HideGroupValues = hide
	tt.ShowField = func(field reflect.StructField) bool { return field.Name != "Hours" }
	tt.SetRowButton(icons.Edit, "Edit", func(item any) {})
	tt.SetItems(&tasks)
	b.AssertRender(t, fmt.Sprintf("tree-table/fit-width-hide-%v", hide), func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		nw, _ := tt.RowWidgetNs()
		for row := range 3 { // also those of the rows that are not groups
			last := AsWidget(tt.ListGrid.Child(row*nw + nw - 1)).Geom.TotalBBox
			assert.LessOrEqual(t, last.Max.X, tt.Geom.TotalBBox.Max.X, "the row button is in the table")
		}
		assert.False(t, tt.HasScroll[math32.X], "no horizontal scrolling")
		assert.False(t, tt.ListGrid.HasScroll[math32.X], "no horizontal scrolling of the grid")
	})
}

// TestTreeTableScrollHidden tests that scrolling a [TreeTable] with
// [TreeTable.HideGroupValues] shows the values of the items that are not
// groups in the rows that showed groups, whose values are hidden.
func TestTreeTableScrollHidden(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Y.Dp(250)
		s.Max.Y.Dp(250)
	})
	var tasks []*task
	for g := range 10 {
		tasks = append(tasks, &task{Name: fmt.Sprintf("Group %d", g), Children: []*task{
			{Name: fmt.Sprintf("Task %d", g), Done: g%2 == 0},
		}})
	}
	tt := NewTreeTable(b).SetItems(&tasks)
	tt.HideGroupValues = true
	b.AssertRender(t, "tree-table/scroll-hidden", func() {
		b.AsyncLock()
		bb := tt.ListGrid.Geom.TotalBBox
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		odd := false // whether a scroll was by an odd number of rows, as the
		// rows of groups and items alternate
		for range 8 {
			src.Scroll(bb.Min.Add(bb.Size().Div(2)), math32.Vec2(0, 25))
			time.Sleep(100 * time.Millisecond)
			b.AsyncLock()
			odd = odd || tt.StartIndex%2 == 1
			nw, idxOff := tt.RowWidgetNs()
			for i := range tt.VisibleRows {
				si, _, invis := tt.SliceIndex(i)
				if invis {
					continue
				}
				done := tt.ListGrid.Child(i*nw + idxOff + 1)
				_, isSwitch := done.(*Switch)
				assert.Equal(t, !tt.hideValue(si, 1), isSwitch, "row %d, item %d, start %d", i, si, tt.StartIndex)
				if isSwitch { // laid out, after it is made in the place of a hidden value
					assert.Greater(t, AsWidget(done).Geom.Size.Actual.Total.X, float32(0), "row %d, item %d, start %d", i, si, tt.StartIndex)
				}
			}
			b.AsyncUnlock()
		}
		assert.True(t, odd, "scrolled by an odd number of rows")
	})
}

// TestTreeTableSortOnChange tests that clicking the check box of an item,
// in the column that the items are sorted by, sorts them again, with the
// item, which is selected, moving after those that are not checked.
func TestTreeTableSortOnChange(t *testing.T) {
	b := NewBody()
	tasks := []*task{{Name: "Work"}, {Name: "Home"}, {Name: "Errands"}}
	tt := NewTreeTable(b).SetItems(&tasks)
	tt.SetSortFieldName("Done")
	var cell *WidgetBase
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, []string{"Work", "Home", "Errands"}, rowNames(tt))
		for _, c := range tt.ListGrid.Children {
			cw := AsWidget(c)
			if cw.Property(ListRowProperty) == 0 && cw.Property(ListColProperty) == 1 {
				cell = cw
			}
		}
		require.NotNil(t, cell)
	}, func() {
		b.AsyncLock()
		bb := cell.Geom.TotalBBox
		b.AsyncUnlock()
		pt := bb.Min.Add(bb.Size().Div(2))
		src := b.Scene.RenderWindow().SystemWindow.Events()
		src.MouseButton(events.MouseDown, events.Left, pt, 0)
		src.MouseButton(events.MouseUp, events.Left, pt, 0)
		assert.Eventually(t, func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return tasks[0].Done || tasks[2].Done
		}, 2*time.Second, 20*time.Millisecond, "the item is done")
		time.Sleep(100 * time.Millisecond)
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, []string{"Home", "Errands", "Work"}, rowNames(tt))
		assert.Equal(t, 2, tt.SelectedIndex)
		_, _, foc := tt.focusedCell()
		_, isText := foc.(*TextField)
		assert.False(t, isText, "the focus does not go to a text field (with the keyboard on mobile)")
	})
}

// TestTreeTableNewFocus tests that a new item, added with a button,
// has the focus in its first cell after the click.
func TestTreeTableNewFocus(t *testing.T) {
	b := NewBody()
	tasks := []*task{{Name: "Work"}, {Name: "Home"}}
	bt := NewButton(b).SetText("Add item")
	tt := NewTreeTable(b).SetItems(&tasks)
	bt.OnClick(func(e events.Event) { tt.NewAt(-1) })
	b.RenderImage(func() {}, func() {
		b.AsyncLock()
		bb := bt.Geom.TotalBBox
		b.AsyncUnlock()
		pt := bb.Min.Add(bb.Size().Div(2))
		src := b.Scene.RenderWindow().SystemWindow.Events()
		src.MouseButton(events.MouseDown, events.Left, pt, 0)
		src.MouseButton(events.MouseUp, events.Left, pt, 0)
		assert.Eventually(t, func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			idx, _, foc := tt.focusedCell()
			_, ok := foc.(*TextField)
			return len(tasks) == 3 && idx == 2 && ok
		}, 2*time.Second, 20*time.Millisecond, "the text field of the new item is focused")
	})
}

// TestTreeTableFocusShrink tests that the text being typed into a focused
// item of a tree table, whose first cell has the text field in it, stays
// with the item, with its cursor, when the table gets fewer rows, as when
// the virtual keyboard shrinks the window on mobile (see TestTableFocusShrink).
func TestTreeTableFocusShrink(t *testing.T) {
	tasks := make([]*task, 12)
	for i := range tasks {
		tasks[i] = &task{}
	}
	b := NewBody()
	ht := float32(600)
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(300), units.Dp(ht))
		s.Max.Set(units.Dp(300), units.Dp(ht))
	})
	tt := NewTreeTable(b).SetItems(&tasks)
	typeKey := func(r rune, code key.Codes) {
		_, _, foc := tt.focusedCell()
		if tf, ok := foc.(*TextField); ok {
			tf.HandleEvent(events.NewKey(events.KeyChord, r, code, 0))
		}
	}
	last := 0
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		last = tt.StartIndex + tt.VisibleRows - 2
		tt.IndexGrabFocus(last)
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		typeKey('a', key.CodeA)
		typeKey('b', key.CodeB)
		ht = 250
		b.Restyle()
		b.NeedsLayout()
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		idx, _, foc := tt.focusedCell()
		_, ok := foc.(*TextField)
		require.True(t, ok, "a text field is focused")
		assert.Equal(t, last, idx, "of the same item")
		typeKey('x', key.CodeX)
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		_, _, foc := tt.focusedCell()
		if tf, ok := foc.(*TextField); ok {
			tf.editDone()
		}
		assert.Equal(t, "abx", tasks[last].Name, "typed into the item, at the end")
	})
}

// TestTreeTableMultiSelect tests selecting several rows of an editable
// tree table with clicks, sent to the window: with Shift from the last
// clicked, and with Command, which do not edit the cells (as checking a
// check box), and in the select mode, with plain clicks (taps on mobile),
// which do not focus a text field (showing the keyboard on mobile).
func TestTreeTableMultiSelect(t *testing.T) {
	b := NewBody()
	tasks := []*task{{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"}}
	tt := NewTreeTable(b).SetItems(&tasks)
	cellAt := func(row, col int) *WidgetBase {
		for _, c := range tt.ListGrid.Children {
			cw := AsWidget(c)
			if cw.Property(ListRowProperty) == row && cw.Property(ListColProperty) == col {
				return cw
			}
		}
		return nil
	}
	selected := func() []int {
		b.AsyncLock()
		defer b.AsyncUnlock()
		return tt.SelectedIndexesList(false)
	}
	b.RenderImage(func() {}, func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		click := func(row, col int, mods key.Modifiers) {
			b.AsyncLock()
			bb := cellAt(row, col).Geom.TotalBBox
			b.AsyncUnlock()
			pt := image.Pt((bb.Min.X+bb.Max.X)/2, (bb.Min.Y+bb.Max.Y)/2) // (the text, or the check box)
			src.MouseButton(events.MouseDown, events.Left, pt, mods)
			src.MouseButton(events.MouseUp, events.Left, pt, mods)
			time.Sleep(150 * time.Millisecond)
		}
		var shift, meta key.Modifiers
		shift.SetFlag(true, key.Shift)
		meta.SetFlag(true, key.Meta)
		click(0, 0, 0)
		assert.Equal(t, []int{0}, selected())
		click(2, 0, shift)
		assert.Equal(t, []int{0, 1, 2}, selected(), "Shift")
		click(3, 1, meta) // (the check box of D)
		assert.Equal(t, []int{0, 1, 2, 3}, selected(), "Command")
		assert.False(t, tasks[3].Done, "not checked")

		b.AsyncLock()
		tt.SetSelectMode(true)
		b.AsyncUnlock()
		click(1, 0, 0)
		assert.Equal(t, []int{0, 2, 3}, selected(), "a tap in the select mode")
		b.AsyncLock()
		_, isText := tt.Events().focus.(*TextField)
		b.AsyncUnlock()
		assert.False(t, isText, "no text field is focused")
	})
}

// TestTreeTableMoveSelected tests moving several selected items together:
// up and down among their siblings, keeping their order, blocked at the
// first, and indenting them into the same item, and outdenting them.
func TestTreeTableMoveSelected(t *testing.T) {
	b := NewBody()
	tasks := []*task{{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"}}
	tt := NewTreeTable(b).SetItems(&tasks)
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		sel := func(rows ...int) {
			tt.ResetSelectedIndexes()
			for _, r := range rows {
				tt.SelectIndex(r)
			}
			tt.SelectedIndex = rows[0]
		}
		selNames := func() []string {
			var ns []string
			for _, i := range tt.SelectedIndexesList(false) {
				ns = append(ns, strings.TrimSpace(rowNames(tt)[i]))
			}
			return ns
		}
		sel(1, 2)
		tt.MoveSelectedUp(1)
		assert.Equal(t, []string{"B", "C", "A", "D"}, rowNames(tt))
		assert.Equal(t, []string{"B", "C"}, selNames())
		tt.MoveSelectedUp(0)
		assert.Equal(t, []string{"B", "C", "A", "D"}, rowNames(tt), "blocked at the first")
		tt.MoveSelectedDown(0)
		tt.MoveSelectedDown(1)
		assert.Equal(t, []string{"A", "D", "B", "C"}, rowNames(tt))
		tt.IndentSelected(2)
		assert.Equal(t, []string{"A", "D", "  B", "  C"}, rowNames(tt))
		assert.Equal(t, []string{"B", "C"}, selNames())
		tt.OutdentSelected(2)
		assert.Equal(t, []string{"A", "D", "B", "C"}, rowNames(tt))
		assert.Equal(t, []string{"B", "C"}, selNames())
	})
}
