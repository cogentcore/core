// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type language struct {
	Name   string
	Rating int
}

func (l *language) FieldWidget(field string) Value {
	if field == "Rating" {
		return NewSlider().SetMin(1).SetMax(10).SetStep(1)
	}
	return nil
}

func TestTable(t *testing.T) {
	b := NewBody()
	NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
	AssertRender(t, b, "table/basic")
}

func TestTableReadOnly(t *testing.T) {
	b := NewBody()
	NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}}).SetReadOnly(true)
	AssertRender(t, b, "table/read-only")
}

// TestTableReadOnlyFocus tests that a read-only table takes the focus when
// its rows are clicked, as they are not focusable, and that the keys then
// move the selection among them.
func TestTableReadOnlyFocus(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b)
	tb := NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}, {"C", 3}})
	tb.SetReadOnly(true)
	AssertRender(t, b, "table/read-only-focus", func() {
		b.AsyncLock()
		tf.SetFocus()
		assert.True(t, tb.CanFocus())
		pt := tb.ListGrid.Child(tb.numVisibleFields + 1).(Widget).AsWidget().Geom.ContentBBox.Min.Add(image.Pt(4, 4))
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		src.MouseButton(events.MouseDown, events.Left, pt, 0)
		src.MouseButton(events.MouseUp, events.Left, pt, 0)
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		assert.True(t, tb.ContainsFocus(), "the table has the focus")
		assert.Equal(t, 1, tb.SelectedIndex)
		b.AsyncUnlock()
		src.Key(events.KeyDown, 0, key.CodeDownArrow, 0)
		src.Key(events.KeyUp, 0, key.CodeDownArrow, 0)
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		assert.Equal(t, 2, tb.SelectedIndex, "down arrow")
		tf.SetFocus()
		tb.Events().FocusNextFrom(tf) // tab into the table
		assert.True(t, tb.StateIs(states.Focused), "tab into the table, with the focus style")
		b.AsyncUnlock()
	})
}

// TestTableColumnWidths tests setting the width of a column by dragging
// the handle after its header, after holding the mouse down, and making
// it automatic again.
func TestTableColumnWidths(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(600), units.Dp(300)) })
	tb := NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
	changes := 0
	tb.OnChange(func(e events.Event) { changes++ })
	imagex.UpdateTestImagesFor(t) // the image is not compared
	AssertRender(t, b, "table/column-widths", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		dp := func(dots float32) float32 { return dots / tb.Styles.UnitContext.Dots(units.UnitDp) }
		b.AsyncLock()
		auto := tb.columnDots(0)
		hb := AsWidget(tb.header.Child(1)).Geom.TotalBBox // the handle of Name
		header := AsWidget(tb.header.Child(0)).Geom.TotalBBox
		b.AsyncUnlock()
		assert.InDelta(t, header.Dx()+hb.Dx(), int(auto+tb.ListGrid.Styles.Gap.X.Dots), 2, "the header and handle have the width of the column")

		pt := hb.Min.Add(hb.Size().Div(2))
		src.MouseButton(events.MouseDown, events.Left, pt, 0)
		// held longer than the drag start time, after which the table, which
		// is draggable, would drag, except that the handle has SlidePriority
		time.Sleep(2 * TimingSettings.DragStartTime)
		for i := 1; i <= 10; i++ {
			src.MouseMove(pt.Add(image.Pt(5*i, 0)))
			time.Sleep(20 * time.Millisecond)
		}
		src.MouseButton(events.MouseUp, events.Left, pt.Add(image.Pt(50, 0)), 0)
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		assert.InDelta(t, dp(auto+50), tb.ColumnWidths["Name"], 1)
		assert.InDelta(t, auto+50, tb.columnDots(0), 2, "the column has the width")
		assert.Equal(t, 1, changes, "a change at the end")
		hb = AsWidget(tb.header.Child(1)).Geom.TotalBBox
		b.AsyncUnlock()

		pt = hb.Min.Add(hb.Size().Div(2))
		for range 2 {
			src.MouseButton(events.MouseDown, events.Left, pt, 0)
			src.MouseButton(events.MouseUp, events.Left, pt, 0)
		}
		time.Sleep(300 * time.Millisecond)
		b.AsyncLock()
		assert.NotContains(t, tb.ColumnWidths, "Name", "double click for automatic")
		assert.InDelta(t, auto, tb.columnDots(0), 2)
		b.AsyncUnlock()
	})
}

// TestTableContextMenuSelect tests that a right click (a context menu) on
// a row of a read-only, multi-select table keeps the selection of several
// rows when the row is one of them, and otherwise selects only that row.
func TestTableContextMenuSelect(t *testing.T) {
	b := NewBody()
	tb := NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}, {"C", 3}})
	tb.SetReadOnly(true)
	tb.ReadOnlyMultiSelect = true
	RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tb.SelectIndexEvent(0, events.SelectOne)
		tb.SelectIndexEvent(1, events.ExtendContinuous)
	}, func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		rightClick := func(row int) {
			b.AsyncLock()
			n, _ := tb.RowWidgetNs()
			pt := tb.ListGrid.Child(row * n).(Widget).AsWidget().Geom.ContentBBox.Min.Add(image.Pt(4, 4))
			b.AsyncUnlock()
			src.MouseButton(events.MouseDown, events.Right, pt, 0)
			src.MouseButton(events.MouseUp, events.Right, pt, 0)
			time.Sleep(200 * time.Millisecond)
			src.Key(events.KeyDown, 0, key.CodeEscape, 0) // (closes the menu)
			src.Key(events.KeyUp, 0, key.CodeEscape, 0)
			time.Sleep(100 * time.Millisecond)
		}
		selected := func() []int {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return tb.SelectedIndexesList(reflectx.Ascending)
		}
		rightClick(1)
		assert.Equal(t, []int{0, 1}, selected(), "on a selected row")
		b.AsyncLock()
		assert.Equal(t, 1, tb.SelectedIndex)
		b.AsyncUnlock()
		rightClick(2)
		assert.Equal(t, []int{2}, selected(), "on another row")
	})
}

// TestTableFocusShrink tests that the item of a focused text field of a
// table stays focused and visible when the table gets fewer rows, as when
// the virtual keyboard shrinks the window on mobile, with the text typed
// before and after going to it, in order, not to the item that the
// removed or rebound row shows.
func TestTableFocusShrink(t *testing.T) {
	type item struct{ Name string }
	items := make([]item, 12)
	b := NewBody()
	ht := float32(600)
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(300), units.Dp(ht))
		s.Max.Set(units.Dp(300), units.Dp(ht))
	})
	tb := NewTable(b).SetSlice(&items)
	focusedItem := func() int {
		idx, _, foc := tb.focusedCell()
		if _, ok := foc.(*TextField); !ok {
			return -1
		}
		return idx
	}
	last := 0
	RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		last = tb.StartIndex + tb.VisibleRows - 2 // the last fully visible one
		tb.IndexGrabFocus(last)
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, last, focusedItem())
		_, _, foc := tb.focusedCell()
		if tf, ok := foc.(*TextField); ok {
			tf.HandleEvent(events.NewKey(events.KeyChord, 'a', key.CodeA, 0))
			tf.HandleEvent(events.NewKey(events.KeyChord, 'b', key.CodeB, 0))
		}
		ht = 250
		b.Restyle()
		b.NeedsLayout()
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Less(t, tb.VisibleRows, last, "fewer rows")
		assert.Equal(t, last, focusedItem(), "the same item is focused")
		assert.GreaterOrEqual(t, last, tb.StartIndex)
		assert.Less(t, last, tb.StartIndex+tb.VisibleRows, "in view")
		_, _, foc := tb.focusedCell()
		if tf, ok := foc.(*TextField); ok {
			tf.HandleEvent(events.NewKey(events.KeyChord, 'x', key.CodeX, 0))
		}
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		_, _, foc := tb.focusedCell()
		if tf, ok := foc.(*TextField); ok {
			tf.editDone()
		}
		assert.Equal(t, "abx", items[last].Name, "typed into the focused item, at the end")
	})
}

// TestTableRowButtonBackground tests that the buttons of the rows (see
// [Table.SetRowButton]) have the background of their own rows: that of
// the selected row only for the button of that row.
func TestTableRowButtonBackground(t *testing.T) {
	type item struct{ Name string }
	items := []item{{"a"}, {"b"}, {"c"}}
	b := NewBody()
	tb := NewTable(b).SetSlice(&items)
	tb.SetRowButton(icons.Edit, "Edit", func(item any) {})
	RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tb.SelectIndex(0)
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		lg := tb.ListGrid
		var buttons []*Button
		for _, c := range lg.Children {
			if bt, ok := c.(*Button); ok && bt.Icon == icons.Edit {
				buttons = append(buttons, bt)
			}
		}
		if assert.GreaterOrEqual(t, len(buttons), 3) { // (and empty rows)
			sel := lg.ChildBackground(buttons[0])
			assert.Equal(t, lg.bgSelect, sel, "the selected row")
			assert.NotEqual(t, sel, lg.ChildBackground(buttons[1]), "not the selected row")
			assert.NotEqual(t, sel, lg.ChildBackground(buttons[2]), "not the selected row")
		}
	})
}

// TestTableEmptyHeader tests that the header of an empty table has the
// names of its fields, with their widths, as its cells are invisible,
// and that of a table with rows the widths of its columns.
func TestTableEmptyHeader(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b := NewBody()
	var sl []language
	tb := NewTable(b).SetSlice(&sl)
	headWidth := func(fli int) float32 {
		return AsWidget(tb.header.Child(2 * fli)).Geom.Size.Actual.Total.X
	}
	AssertRender(t, b, "table/empty-header", func() {
		b.AsyncLock() // (it is rendered meanwhile)
		defer b.AsyncUnlock()
		for fli := range 2 {
			assert.Greater(t, headWidth(fli), float32(40), "the name of the field")
		}
		sl = append(sl, language{Name: "Go", Rating: 10})
		tb.Update()
	}, func() {
		// (laid out with the row, which can be after the window has
		// handled its events, as on a slower machine)
		b.AsyncLock()
		defer b.AsyncUnlock()
		b.Scene.LayoutScene()
		gap := tb.ListGrid.Styles.Gap.X.Dots
		for fli := range 2 {
			col := AsWidget(tb.ListGrid.Child(fli)).Geom.Size.Alloc.Total.X + gap
			handle := AsWidget(tb.header.Child(2*fli + 1)).Geom.Size.Actual.Total.X
			assert.Equal(t, col-handle, headWidth(fli), "the width of the column")
		}
	})
}

// keyBinding has a bit flag field, for [TestTableBitFlags].
type keyBinding struct {
	Name      string
	Modifiers key.Modifiers
}

// TestTableBitFlags tests that the switches of a bit flag field are on one
// line in a table, in a column that is wide enough for all of them to be
// checked, as the chips of checked flags are wider, with their check icons:
// checking all of the flags of a row does not change the width of the column.
func TestTableBitFlags(t *testing.T) {
	b := NewBody()
	all := key.Modifiers(0)
	for _, m := range key.ModifiersValues() {
		all.SetFlag(true, m)
	}
	some := key.Modifiers(0)
	some.SetFlag(true, key.Control, key.Shift)
	bindings := []keyBinding{{"Copy", some}, {"None", 0}}
	tb := NewTable(b).SetSlice(&bindings)
	// checks that the switches of each row are on one line, within the column
	check := func(msg string) {
		_, idxOff := tb.RowWidgetNs()
		nw, _ := tb.RowWidgetNs()
		for row := range len(bindings) {
			sw := tb.ListGrid.Child(row*nw + idxOff + 1).(*Switches)
			sb := sw.Geom.TotalBBox
			y := -1
			for _, c := range sw.Children {
				cb := AsWidget(c).Geom.TotalBBox
				if y < 0 {
					y = cb.Min.Y
				}
				assert.Equal(t, y, cb.Min.Y, "%s: row %d: %s on the same line", msg, row, AsWidget(c).Name)
				assert.LessOrEqual(t, cb.Max.X, sb.Max.X, "%s: row %d: %s within the switches", msg, row, AsWidget(c).Name)
			}
		}
	}
	AssertRender(t, b, "table/bit-flags", func() {
		b.AsyncLock()
		check("some checked")
		width := tb.columnDots(1)
		bindings[0].Modifiers = all
		bindings[1].Modifiers = all
		tb.Update()
		b.AsyncUnlock()
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		defer b.AsyncUnlock()
		check("all checked")
		assert.Equal(t, width, tb.columnDots(1), "the width of the column")
	})
}

// dragRow drags the given row of the given table to the given row of the
// other given table (which can be the same), with the given modifier keys,
// by its index if they are shown, and otherwise by the gap after its first
// value, and inserts it after that row with the menu of the drop, returning
// whether the drag image showed a copy.
func dragRow(t *testing.T, b *Body, tb *Table, from int, totb *Table, to int, mods key.Modifiers) (dragCopy bool) {
	src := b.Scene.RenderWindow().SystemWindow.Events()
	b.AsyncLock()
	cell := func(tb *Table, row int) image.Rectangle {
		nw, _ := tb.RowWidgetNs()
		return tb.ListGrid.Child(row * nw).(Widget).AsWidget().Geom.TotalBBox
	}
	fb, tbb := cell(tb, from), cell(totb, to)
	b.AsyncUnlock()
	fp := fb.Min.Add(fb.Size().Div(2))
	if !tb.ShowIndexes {
		fp.X = fb.Max.X + 1 // in the gap after the value, as pressing it edits it
	}
	tp := tbb.Min.Add(tbb.Size().Div(2))
	src.MouseButton(events.MouseDown, events.Left, fp, mods)
	for i := 1; i <= 10; i++ {
		src.MouseMove(fp.Add(tp.Sub(fp).Mul(i).Div(10)))
		time.Sleep(20 * time.Millisecond)
	}
	waitDragOver(b, src, tp)
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
	var after *Button
	menu.Scene.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		if bt, ok := cw.(*Button); ok && bt.Text == "Insert after" {
			after = bt
		}
		return after == nil
	})
	require.NotNil(t, after)
	after.Send(events.Click)
	b.AsyncUnlock()
	time.Sleep(200 * time.Millisecond)
	return
}

// bindingNames returns the names of the given key bindings.
func bindingNames(b *Body, bindings []*keyBinding) []string {
	b.AsyncLock()
	defer b.AsyncUnlock()
	var names []string
	for _, kb := range bindings {
		names = append(names, kb.Name)
	}
	return names
}

// TestTableDragMoves tests dragging a row of a table by its index, with
// DragMoves, which moves it without modifier keys and copies it with
// Shift, in a table with the switches of bit flags, as in the editor of
// the list types of Cogent Lists.
func TestTableDragMoves(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(700), units.Dp(400)) })
	bindings := []*keyBinding{{"A", 0}, {"B", 0}, {"C", 0}, {"D", 0}}
	tb := NewTable(b).SetSlice(&bindings)
	tb.SetShowIndexes(true)
	tb.DragMoves = true
	imagex.UpdateTestImagesFor(t) // the image is not compared
	AssertRender(t, b, "table/drag-moves", func() {
		assert.False(t, dragRow(t, b, tb, 0, tb, 2, 0), "no plus for a move")
		assert.Equal(t, []string{"B", "C", "A", "D"}, bindingNames(b, bindings), "A moved after C")
		var shift key.Modifiers
		shift.SetFlag(true, key.Shift)
		assert.True(t, dragRow(t, b, tb, 3, tb, 0, shift), "a plus for a copy")
		assert.Equal(t, []string{"B", "D", "C", "A", "D"}, bindingNames(b, bindings), "D copied after B")
	})
}

// TestTableDragShiftMoves tests moving a row of a table without indexes
// (for which its first value is the source of the drag) and without DragMoves, by dragging it with
// Shift, which moves the row that is pressed, and not the row under the
// mouse when the drag starts or the selected row, extended with Shift.
func TestTableDragShiftMoves(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(700), units.Dp(400)) })
	bindings := []*keyBinding{{"A", 0}, {"B", 0}, {"C", 0}, {"D", 0}, {"E", 0}}
	tb := NewTable(b).SetSlice(&bindings)
	imagex.UpdateTestImagesFor(t) // the image is not compared
	AssertRender(t, b, "table/drag-shift-moves", func() {
		b.AsyncLock()
		tb.SelectIndex(1) // another row is selected
		b.AsyncUnlock()
		var shift key.Modifiers
		shift.SetFlag(true, key.Shift)
		assert.False(t, dragRow(t, b, tb, 4, tb, 0, shift), "no plus for a move")
		assert.Equal(t, []string{"A", "E", "B", "C", "D"}, bindingNames(b, bindings), "E moved after A")
		assert.True(t, dragRow(t, b, tb, 3, tb, 4, 0), "a plus for a copy")
		assert.Equal(t, []string{"A", "E", "B", "C", "D", "C"}, bindingNames(b, bindings), "C copied after D")
	})
}

// TestTableDragBetween tests moving a row from one table to another,
// which deletes the row that was dragged from the first table.
func TestTableDragBetween(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(700), units.Dp(500)) })
	from := []*keyBinding{{"A", 0}, {"B", 0}, {"C", 0}, {"D", 0}}
	to := []*keyBinding{{"X", 0}, {"Y", 0}}
	ftb := NewTable(b).SetSlice(&from)
	ftb.SetShowIndexes(true)
	ftb.DragMoves = true
	ttb := NewTable(b).SetSlice(&to)
	ttb.SetShowIndexes(true)
	imagex.UpdateTestImagesFor(t) // the image is not compared
	AssertRender(t, b, "table/drag-between", func() {
		b.AsyncLock()
		ftb.SelectIndex(3) // another row is selected
		b.AsyncUnlock()
		assert.False(t, dragRow(t, b, ftb, 1, ttb, 0, 0), "no plus for a move")
		assert.Equal(t, []string{"X", "B", "Y"}, bindingNames(b, to), "B moved after X")
		assert.Equal(t, []string{"A", "C", "D"}, bindingNames(b, from), "B deleted")
	})
}

// TestDragMovesTag tests the drag-moves:"+" struct tag, which sets
// DragMoves for the list or table of a slice opened with its button,
// as for the fields of a type in a table of types.
func TestDragMovesTag(t *testing.T) {
	var bindings []*keyBinding
	lb, ok := NewValue(&bindings, `drag-moves:"+"`).(*ListButton)
	require.True(t, ok)
	assert.True(t, lb.DragMoves)
	lb, ok = NewValue(&bindings, "").(*ListButton)
	require.True(t, ok)
	assert.False(t, lb.DragMoves, "not by default")

	// the table of the dialog of the button
	b := NewBody()
	lb = NewValue(&bindings, `drag-moves:"+"`, b).(*ListButton)
	AssertRender(t, b, "table/drag-moves-tag", func() {
		b.AsyncLock()
		lb.Send(events.Click)
		b.AsyncUnlock()
		time.Sleep(200 * time.Millisecond)
		var tb *Table
		for _, rw := range AllRenderWindows {
			sc := rw.MainScene()
			sc.AsyncLock()
			sc.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
				if x, ok := cw.(*Table); ok {
					tb = x
				}
				return tb == nil
			})
			sc.AsyncUnlock()
		}
		require.NotNil(t, tb, "the table of the dialog")
		assert.True(t, tb.DragMoves)
	})
}

// keyMap has a slice field with the drag-moves tag, for [TestTableSliceTags].
type keyMap struct {
	Name     string
	Bindings []*keyBinding `drag-moves:"+"`
}

// TestTableSliceTags tests that the struct tags of a slice field in a
// table are kept for its value, a button that opens its table, along with
// the tags of the table for slices.
func TestTableSliceTags(t *testing.T) {
	b := NewBody()
	maps := []keyMap{{Name: "A"}}
	tb := NewTable(b).SetSlice(&maps)
	AssertRender(t, b, "table/slice-tags", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		var lb *ListButton
		tb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
			if x, ok := cw.(*ListButton); ok {
				lb = x
			}
			return lb == nil
		})
		require.NotNil(t, lb)
		assert.True(t, lb.DragMoves, "the drag-moves tag of the field")
		assert.True(t, lb.AsWidget().hasFlag(widgetValueNewWindow), "the new-window tag of the table")
	})
}
