// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"github.com/stretchr/testify/assert"
)

func TestHandleWidgetState(t *testing.T) {
	b := NewBody()
	w := NewBox(b)

	test := func(ability abilities.Abilities, state states.States, event events.Types, endEvent events.Types) {
		expect := states.States(0)
		assert.Equal(t, expect, w.Styles.State)

		w.Send(event)
		assert.Equal(t, expect, w.Styles.State)

		w.Style(func(s *styles.Style) {
			w.SetAbilities(true, ability)
		})
		w.ApplyStyle()

		w.Send(event)
		expect.SetFlag(true, state)
		assert.Equal(t, expect, w.Styles.State)

		w.Send(endEvent)
		expect.SetFlag(false, state)
		assert.Equal(t, expect, w.Styles.State)
	}

	test(abilities.Activatable, states.Active, events.MouseDown, events.MouseUp)
	test(abilities.LongPressable, states.LongPressed, events.LongPressStart, events.LongPressEnd)
	test(abilities.Hoverable, states.Hovered, events.MouseEnter, events.MouseLeave)
	test(abilities.LongHoverable, states.LongHovered, events.LongHoverStart, events.LongHoverEnd)
	test(abilities.Slideable, states.Sliding, events.SlideStart, events.SlideStop)
	test(abilities.Draggable, states.Dragging, events.DragStart, events.Drop)
	test(abilities.Focusable, states.Focused, events.Focus, events.FocusLost)
	test(abilities.Checkable, states.Checked, events.Click, events.Click)

	w.HandleSelectToggle()
	test(abilities.Selectable, states.Selected, events.Select, events.Select)
}

func TestWidgetPrev(t *testing.T) {
	b := NewBody()
	NewTextField(b, "tf1").AddClearButton()
	NewTextField(b, "tf2").SetLeadingIcon(icons.Search)
	lt := NewTextField(b, "tf3")
	b.ConfigTree()

	paths := []string{
		"/body scene/body/tf2.parts/lead-icon.parts/icon",
		"/body scene/body/tf2.parts/lead-icon",
		"/body scene/body/tf2",
		"/body scene/body/tf1.parts/trail-icon.parts/icon",
		"/body scene/body/tf1.parts/trail-icon",
		"/body scene/body/tf1.parts/trail-icon-str",
		"/body scene/body/tf1",
		"/body scene/body",
		"/body scene",
	}
	i := 0
	WidgetPrevFunc(lt, func(w Widget) bool {
		have := w.Path()
		want := paths[i]
		if have != want {
			t.Errorf("expected\n%s\n\tbut got\n%s", want, have)
		}
		i++
		return false
	})
}

func TestWidgetNext(t *testing.T) {
	b := NewBody()
	ft := NewTextField(b, "tf1").AddClearButton()
	NewTextField(b, "tf2").SetLeadingIcon(icons.Search)
	NewTextField(b, "tf3")
	b.ConfigTree()

	paths := []string{
		"/body scene/body/tf1.parts/trail-icon-str",
		"/body scene/body/tf1.parts/trail-icon",
		"/body scene/body/tf1.parts/trail-icon.parts/icon",
		"/body scene/body/tf2",
		"/body scene/body/tf2.parts/lead-icon",
		"/body scene/body/tf2.parts/lead-icon.parts/icon",
		"/body scene/body/tf3",
	}
	i := 0
	WidgetNextFunc(ft, func(w Widget) bool {
		have := w.Path()
		want := paths[i]
		if have != want {
			t.Errorf("expected\n%s\n\tbut got\n%s", want, have)
		}
		i++
		return false
	})
}

func ExampleWidgetBase_AddCloseDialog() {
	b := NewBody()
	b.AddCloseDialog(func(d *Body) bool {
		d.AddTitle("Are you sure?").AddText("Are you sure you want to close the Cogent Core Demo?")
		d.AddBottomBar(func(parent Widget) {
			d.AddOK(parent).SetText("Close").OnClick(func(e events.Event) {
				b.Scene.Close()
			})
		})
		return true
	})
	b.RunMainWindow()
}
