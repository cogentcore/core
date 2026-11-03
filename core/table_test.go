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
