// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

func TestButtonText(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("Download")
	b.AssertRender(t, "button/text")
}

func TestButtonIcon(t *testing.T) {
	b := NewBody()
	NewButton(b).SetIcon(icons.Download)
	b.AssertRender(t, "button/icon")
}

func TestButtonTextIcon(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("Download").SetIcon(icons.Download)
	b.AssertRender(t, "button/text-icon")
}

func TestButtonClick(t *testing.T) {
	b := NewBody()
	clicked := false
	bt := NewButton(b)
	bt.OnClick(func(e events.Event) {
		clicked = true
	})
	bt.Send(events.Click)
	assert.True(t, clicked)
}

func TestButtonMenu(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("Share").SetIcon(icons.Share).SetMenu(func(m *Scene, pos image.Point) {
		NewButton(m).SetText("Copy link")
		NewButton(m).SetText("Send message")
	})
	b.AssertRender(t, "button/menu")
}

func TestButtonMenuClick(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	bt := NewButton(b).SetText("Share").SetIcon(icons.Share).SetMenu(func(m *Scene, pos image.Point) {
		NewButton(m).SetText("Copy link")
		NewButton(m).SetText("Send message")
	})
	b.AssertRender(t, "button/menu-click", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		bt.Send(events.Click)
	})
}

func TestButtonShortcut(t *testing.T) {
	b := NewBody()
	bt := NewButton(b).SetShortcut("Command+S")
	tt, _ := bt.WidgetTooltip(image.Point{})
	assert.Equal(t, "["+commandLabel()+"+S]", tt)
}

func TestButtonShortcutWithTooltip(t *testing.T) {
	b := NewBody()
	bt := NewButton(b).SetShortcut("Command+S")
	bt.SetTooltip("Test")
	tt, _ := bt.WidgetTooltip(image.Point{})
	assert.Equal(t, "["+commandLabel()+"+S] Test", tt)
}

func TestButtonShortcutKey(t *testing.T) {
	b := NewBody()
	bt := NewButton(b).SetKey(keymap.Open)
	tt, _ := bt.WidgetTooltip(image.Point{})
	assert.Equal(t, "["+commandLabel()+"+O]", tt)
}

func TestButtonShortcutMenu(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("Save").SetShortcut("Command+S").SetType(ButtonMenu)
	b.AssertRender(t, "button/shortcut-menu")
}

func TestButtonTypes(t *testing.T) {
	for _, typ := range ButtonTypesValues() {
		b := NewBody()
		NewButton(b).SetType(typ).SetText(typ.String())
		b.AssertRender(t, "button/type-"+strings.ToLower(typ.String()))
	}
}

func TestButtonFontSize(t *testing.T) {
	b := NewBody()
	bt := NewButton(b).SetText("Hello")
	bt.Styler(func(s *styles.Style) {
		s.Font.Size.Dp(48)
	})
	b.AssertRender(t, "button/font-size")
}

// TestButtonShortcutFocus tests that triggering a button with its shortcut
// does not move the focus to it, as clicking it does.
func TestButtonShortcutFocus(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b)
	clicked := 0
	bt := NewButton(b).SetText("Action").SetShortcut(key.Chord("Command+Shift+L").PlatformChord())
	bt.OnClick(func(e events.Event) { clicked++ })
	b.AssertRender(t, "button/shortcut-focus", func() {
		b.AsyncLock()
		tf.SetFocus()
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		r, code, mods, err := bt.Shortcut.Decode()
		assert.NoError(t, err)
		src.Key(events.KeyDown, r, code, mods)
		src.Key(events.KeyUp, r, code, mods)
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, 1, clicked)
		assert.True(t, tf.StateIs(states.Focused) || b.Scene.Events.focus == tf.This, "the text field keeps the focus")
	})
}

// commandLabel returns the label of the Command modifier of shortcuts on
// the platform of the tests: Cmd on macOS, as on the web in a Mac, and
// otherwise Ctrl.
func commandLabel() string {
	if key.SystemPlatform == "MacOS" {
		return "Cmd"
	}
	return "Ctrl"
}
