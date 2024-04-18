// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
	"github.com/stretchr/testify/assert"
)

func TestChooserSetStrings(t *testing.T) {
	b := NewBody()
	NewChooser(b).SetStrings("macOS", "Windows", "Linux")
	b.AssertRender(t, "chooser/set-strings")
}

func TestChooserClick(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	ch := NewChooser(b).SetStrings("macOS", "Windows", "Linux")
	b.AssertRenderScreen(t, "chooser/click", func() {
		ch.Send(events.Click)
	})
}

func TestChooserSetItems(t *testing.T) {
	b := NewBody()
	ch := NewChooser(b).SetItems(
		ChooserItem{Value: "Computer", Icon: icons.Computer, Tooltip: "Use a computer"},
		ChooserItem{Value: "Phone", Icon: icons.Smartphone, Tooltip: "Use a phone"},
	)
	b.AssertRender(t, "chooser/set-items", func() {
		assert.Equal(t, "", ch.Tooltip)
		assert.Equal(t, "Use a computer", ch.WidgetTooltip())
	})
}

func TestChooserPlaceholder(t *testing.T) {
	b := NewBody()
	NewChooser(b).SetPlaceholder("Choose a platform").SetStrings("macOS", "Windows", "Linux")
	b.AssertRender(t, "chooser/placeholder")
}

func TestChooserCurrentValue(t *testing.T) {
	b := NewBody()
	ch := NewChooser(b).SetStrings("Apple", "Orange", "Strawberry").SetCurrentValue("Orange")
	assert.Equal(t, 1, ch.CurrentIndex)
	assert.Equal(t, ChooserItem{Value: "Orange"}, ch.CurrentItem)
	b.AssertRender(t, "chooser/current-value")
}

func TestChooserOutlined(t *testing.T) {
	b := NewBody()
	NewChooser(b).SetType(ChooserOutlined).SetStrings("Apple", "Orange", "Strawberry")
	b.AssertRender(t, "chooser/outlined")
}

func TestChooserIcon(t *testing.T) {
	b := NewBody()
	NewChooser(b).SetIcon(icons.Sort).SetStrings("Newest", "Oldest", "Popular")
	b.AssertRender(t, "chooser/icon")
}

func TestChooserEditable(t *testing.T) {
	b := NewBody()
	NewChooser(b).SetEditable(true).SetStrings("Newest", "Oldest", "Popular")
	b.AssertRender(t, "chooser/editable")
}

func TestChooserEditableClick(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	ch := NewChooser(b).SetEditable(true).SetStrings("Newest", "Oldest", "Popular")
	b.AssertRenderScreen(t, "chooser/editable-click", func() {
		ch.Send(events.Click)
	})
}

func TestChooserEditableTextFieldClick(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	ch := NewChooser(b).SetEditable(true).SetStrings("Newest", "Oldest", "Popular")
	b.AssertRenderScreen(t, "chooser/editable-text-field-click", func() {
		ch.TextField().Send(events.Click)
	})
}

func TestChooserAllowNewClick(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	ch := NewChooser(b).SetAllowNew(true).SetStrings("Newest", "Oldest", "Popular")
	b.AssertRenderScreen(t, "chooser/allow-new-click", func() {
		ch.Send(events.Click)
	})
}

func TestChooserEditableAllowNewClick(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	ch := NewChooser(b).SetEditable(true).SetAllowNew(true).SetStrings("Newest", "Oldest", "Popular")
	b.AssertRenderScreen(t, "chooser/editable-allow-new-click", func() {
		ch.Send(events.Click)
	})
}

func TestChooserEditableAllowNewTextFieldClick(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	ch := NewChooser(b).SetEditable(true).SetAllowNew(true).SetStrings("Newest", "Oldest", "Popular")
	b.AssertRenderScreen(t, "chooser/editable-allow-new-text-field-click", func() {
		ch.TextField().HandleEvent(events.NewKey(events.KeyChord, 'O', 0, 0))
	}, func() {
		ch.TextField().Send(events.Click)
	})
}
