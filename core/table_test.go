// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/styles/states"
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
