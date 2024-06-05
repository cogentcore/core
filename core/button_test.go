// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"strings"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
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
	bt := NewButton(b).OnClick(func(e events.Event) {
		clicked = true
	})
	bt.Send(events.Click)
	assert.True(t, clicked)
}

func TestButtonMenu(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("Share").SetIcon(icons.Share).SetMenu(func(m *Scene) {
		NewButton(m).SetText("Copy link")
		NewButton(m).SetText("Send message")
	})
	b.AssertRender(t, "button/menu")
}

func TestButtonMenuClick(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	bt := NewButton(b).SetText("Share").SetIcon(icons.Share).SetMenu(func(m *Scene) {
		NewButton(m).SetText("Copy link")
		NewButton(m).SetText("Send message")
	})
	b.AssertRenderScreen(t, "button/menu-click", func() {
		bt.Send(events.Click)
	})
}

func TestButtonShortcut(t *testing.T) {
	b := NewBody()
	bt := NewButton(b).SetShortcut("Command+S")
	tt, _ := bt.WidgetTooltip(image.Point{})
	assert.Equal(t, "[Ctrl+S]", tt)
}

func TestButtonShortcutWithTooltip(t *testing.T) {
	b := NewBody()
	bt := NewButton(b).SetShortcut("Command+S")
	bt.SetTooltip("Test")
	tt, _ := bt.WidgetTooltip(image.Point{})
	assert.Equal(t, "[Ctrl+S] Test", tt)
}

func TestButtonShortcutKey(t *testing.T) {
	b := NewBody()
	bt := NewButton(b).SetKey(keymap.Open)
	tt, _ := bt.WidgetTooltip(image.Point{})
	assert.Equal(t, "[Ctrl+O]", tt)
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
