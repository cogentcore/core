// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
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
	b.AssertRender(t, "table/basic")
}

func TestTableReadOnly(t *testing.T) {
	b := NewBody()
	NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}}).SetReadOnly(true)
	b.AssertRender(t, "table/read-only")
}

// TestTableReadOnlyFocus tests that a read-only table takes the focus when
// its rows are clicked, as they are not focusable, and that the keys then
// move the selection among them.
func TestTableReadOnlyFocus(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b)
	tb := NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}, {"C", 3}})
	tb.SetReadOnly(true)
	b.AssertRender(t, "table/read-only-focus", func() {
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
	b.AssertRender(t, "table/column-widths", func() {
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
	b.RenderImage(func() {
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
			return tb.SelectedIndexesList(false)
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
	b.RenderImage(func() {
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
	b.RenderImage(func() {
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
	b.AssertRender(t, "table/empty-header", func() {
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
