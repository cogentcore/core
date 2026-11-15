// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSprites tests adding, activating, and inactivating sprites, and
// sending events to them.
func TestSprites(t *testing.T) {
	b := NewBody()
	RenderImage(b, Locked(b, func() {
		ss := &b.Scene.Stage.Main.Sprites
		sp := NewSprite("test", nil)
		sp.EventBBox = image.Rect(0, 0, 10, 10)
		clicks, slides := 0, 0
		sp.OnClick(func(e events.Event) { clicks++ })
		sp.OnSlideStart(func(e events.Event) { slides++ }).OnSlideMove(func(e events.Event) { slides++ }).OnSlideStop(func(e events.Event) { slides++ })
		ss.Add(sp)
		got, ok := ss.SpriteByName("test")
		require.True(t, ok)
		assert.Same(t, sp, got)

		assert.False(t, sp.Active)
		ss.ActivateSprite("test", "missing")
		assert.True(t, sp.Active)
		ss.InactivateSprite("test")
		assert.False(t, sp.Active)

		sp.send(events.Click)
		sp.send(events.SlideStart, events.NewMouse(events.MouseDown, 0, image.Pt(1, 1), 0))
		sp.send(events.SlideMove)
		sp.send(events.SlideStop)
		assert.Equal(t, 1, clicks)
		assert.Equal(t, 3, slides)
		sp.SetPos(image.Pt(5, 5))
		assert.Equal(t, image.Rect(5, 5, 15, 15), sp.EventBBox)

		assert.True(t, ss.Delete("test"))
		_, ok = ss.SpriteByName("test")
		assert.False(t, ok)
	}))
}

// TestWidgetEventHelpers tests sending input and key events to widgets,
// updating values and sending changes on input, and moving the focus back.
func TestWidgetEventHelpers(t *testing.T) {
	b := NewBody()
	tf1 := NewTextField(b)
	tf2 := NewTextField(b)
	sr := NewSlider(b)
	RenderImage(b, Locked(b, func() {
		inputs, changes := 0, 0
		tf1.OnInput(func(e events.Event) { inputs++ })
		tf1.UpdateInput()
		assert.Equal(t, 1, inputs)

		tf1.SetFocus()
		tf1.sendKeyChordRune('x', 0, 0)
		assert.Equal(t, "x", string(tf1.editText))
		tf1.sendKey(keymap.SelectAll)
		assert.True(t, tf1.hasSelection(), "the key of select all is sent to it")

		tf2.SetFocus()
		tf2.focusPrev()
		assert.Same(t, tf1.AsWidget(), b.Scene.Events.focus.AsWidget(), "the previous focusable widget")
		assert.True(t, b.ContainsFocus())

		vals := 0
		sr.ValueOnChange = func() { vals++ }
		sr.HandleValueOnInput()
		sr.SendChangeOnInput()
		sr.OnChange(func(e events.Event) { changes++ })
		sr.Send(events.Input)
		assert.Equal(t, 2, vals, "the value is updated on the input, and on the change it sends")
		assert.Equal(t, 1, changes, "and a change is sent")
		sr.SetAttend()
	}))
}

// TestWindowZoomTitle tests the zoom, name, and title of a window, and
// recycling a main window by its data.
func TestWindowZoomTitle(t *testing.T) {
	zoom := AppearanceSettings.Zoom
	defer func() {
		AppearanceSettings.Zoom = zoom
		AppearanceSettings.Apply()
	}()
	data := &struct{ X int }{}
	b := NewBody("zoom").SetData(data)
	RenderImage(b, Locked(b, func() {
		w := b.Scene.RenderWindow()
		require.NotNil(t, w)
		assert.Same(t, b.Scene, w.MainScene())
		assert.True(t, RecycleMainWindow(data), "the window with the data")
		assert.False(t, RecycleMainWindow(&struct{ X int }{}))
		assert.False(t, RecycleMainWindow(nil))

		w.setZoom(120, "")
		assert.Equal(t, float32(120), AppearanceSettings.Zoom)
		w.setZoom(1000, "")
		assert.Equal(t, float32(500), AppearanceSettings.Zoom, "clamped")
		w.setZoom(100, "")
		w.stepZoom(1)
		assert.Equal(t, float32(110), AppearanceSettings.Zoom)

		w.setTitle("New Title")
		assert.Equal(t, "New Title", w.title)
		w.setName("newname")
		assert.Equal(t, "newname", w.name)
		assert.Equal(t, "newname", w.name)
	}))
}
