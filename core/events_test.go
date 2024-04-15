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

func TestHandleWidgetStateFromMouse(t *testing.T) {
	b := NewBody()
	w := NewBox(b)

	addAbility := func(ability abilities.Abilities) {
		w.Style(func(s *styles.Style) {
			w.SetAbilities(true, ability)
		})
		w.ApplyStyle()
	}

	expect := states.States(0)
	assert.Equal(t, expect, w.Styles.State)

	w.Send(events.MouseDown)
	assert.Equal(t, expect, w.Styles.State)

	addAbility(abilities.Activatable)
	w.Send(events.MouseDown)
	expect.SetFlag(true, states.Active)
	assert.Equal(t, expect, w.Styles.State)

	w.Send(events.MouseUp)
	expect.SetFlag(false, states.Active)
	assert.Equal(t, expect, w.Styles.State)

	addAbility(abilities.LongPressable)
	w.Send(events.LongPressStart)
	expect.SetFlag(true, states.LongPressed)
	assert.Equal(t, expect, w.Styles.State)

	w.Send(events.LongPressEnd)
	expect.SetFlag(false, states.LongPressed)
	assert.Equal(t, expect, w.Styles.State)
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
