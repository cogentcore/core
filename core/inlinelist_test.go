// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInlineList(t *testing.T) {
	b := NewBody()
	NewInlineList(b).SetSlice(&[]int{1, 3, 5})
	AssertRender(t, b, "inline-list/basic")
}

func TestInlineListUpdate(t *testing.T) {
	b := NewBody()
	il := NewInlineList(b).SetSlice(&[]int{1, 2, 3})
	il.Update()
	il.SetSlice(&[]int{4, 5, 6})
	il.Update()
	AssertRender(t, b, "inline-list/update")
}

// TestInlineListItemMenu tests that the widgets in an item of an inline
// list, as the fields of a struct, have the context menu of the item, as
// for deleting it, once, also after updates.
func TestInlineListItemMenu(t *testing.T) {
	type labeled struct{ Label, Value string }
	items := []labeled{{"work", "a@x.test"}, {"home", "b@x.test"}}
	b := NewBody()
	il := NewInlineList(b).SetSlice(&items)
	RenderImage(b, Locked(b, func() {
		il.Update()
		il.Update()
		var tf *TextField
		for w := range il.Child(0).(Widget).AsWidget().AllWidgets() {
			if t, ok := w.(*TextField); ok && tf == nil {
				tf = t
			}
		}
		require.NotNil(t, tf, "a field of the first item")
		m := NewScene()
		tf.AsWidget().applyContextMenus(m, image.Point{})
		n := 0
		for w := range m.AllWidgets() {
			if bt, ok := w.(*Button); ok && bt.Text == "Delete" {
				n++
			}
		}
		assert.Equal(t, 1, n, "Delete, once")
		ButtonByText(m, "Delete").Send(events.Click)
		assert.Equal(t, []labeled{{"home", "b@x.test"}}, items, "the item deleted")
	}))
}

// TestInlineListWrapInForm tests that an inline list of text fields that
// wraps in a form fits in the width of its column: the text fields of each
// line grow only into the width left in their line (they grew to their
// max, wider than the form, with the width that the field of another line
// could not use, as the accounts in the settings of Cogent Mail).
func TestInlineListWrapInForm(t *testing.T) {
	type st struct {
		Accounts            []string
		LongerLabelOfAField string
	}
	v := &st{Accounts: []string{"someone8@gmail.com", "someone5@gmail.com", "someone.else@example.org"}}
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.X.Dp(600)
		s.Max.X.Dp(600)
	})
	tb := NewTabs(b)
	fr, _ := tb.NewTab("Tab")
	fm := NewForm(fr).SetStruct(v)
	RenderImage(b, Locked(b, func() {
		var il *InlineList
		fm.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
			if l, ok := cw.(*InlineList); ok {
				il = l
			}
			return il == nil
		})
		require.NotNil(t, il)
		assert.Greater(t, il.Geom.Size.Actual.Total.Y, float32(60), "wrapped")
		right := fr.Geom.Pos.Content.X + fr.Geom.Size.Actual.Content.X
		assert.LessOrEqual(t, fm.Geom.Pos.Total.X+fm.Geom.Size.Actual.Total.X, right+0.5, "the form fits")
		assert.LessOrEqual(t, il.Geom.Pos.Total.X+il.Geom.Size.Actual.Total.X, right+0.5, "the list fits")
	}))
}
