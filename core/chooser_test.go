// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

func TestChooser(t *testing.T) {
	b := NewBody()
	NewChooser(b).SetStrings("macOS", "Windows", "Linux")
	AssertRender(t, b, "chooser/basic")
}

func TestChooserClick(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	ch := NewChooser(b).SetStrings("macOS", "Windows", "Linux")
	AssertRender(t, b, "chooser/click", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ch.Send(events.Click)
	})
}

func TestChooserItems(t *testing.T) {
	b := NewBody()
	ch := NewChooser(b).SetItems(
		ChooserItem{Value: "Computer", Icon: icons.Computer, Tooltip: "Use a computer"},
		ChooserItem{Value: "Phone", Icon: icons.Smartphone, Tooltip: "Use a phone"},
	)
	AssertRender(t, b, "chooser/items", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, "", ch.Tooltip)
		tt, _ := ch.WidgetTooltip(image.Point{})
		assert.Equal(t, "Use a computer", tt)
	})
}

func TestChooserPlaceholder(t *testing.T) {
	b := NewBody()
	NewChooser(b).SetPlaceholder("Choose a platform").SetStrings("macOS", "Windows", "Linux")
	AssertRender(t, b, "chooser/placeholder")
}

func TestChooserCurrentValue(t *testing.T) {
	b := NewBody()
	ch := NewChooser(b).SetStrings("Apple", "Orange", "Strawberry").SetCurrentValue("Orange")
	assert.Equal(t, 1, ch.CurrentIndex)
	assert.Equal(t, ChooserItem{Value: "Orange"}, ch.CurrentItem)
	AssertRender(t, b, "chooser/current-value")
}

func TestChooserOutlined(t *testing.T) {
	b := NewBody()
	NewChooser(b).SetType(ChooserOutlined).SetStrings("Apple", "Orange", "Strawberry")
	AssertRender(t, b, "chooser/outlined")
}

func TestChooserIcon(t *testing.T) {
	b := NewBody()
	NewChooser(b).SetIcon(icons.Sort).SetStrings("Newest", "Oldest", "Popular")
	AssertRender(t, b, "chooser/icon")
}

func TestChooserEditable(t *testing.T) {
	b := NewBody()
	NewChooser(b).SetEditable(true).SetStrings("Newest", "Oldest", "Popular")
	AssertRender(t, b, "chooser/editable")
}

func TestChooserEditableClick(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	ch := NewChooser(b).SetEditable(true).SetStrings("Newest", "Oldest", "Popular")
	AssertRender(t, b, "chooser/editable-click", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ch.Send(events.Click)
	})
}

func TestChooserEditableTextFieldClick(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	ch := NewChooser(b).SetEditable(true).SetStrings("Newest", "Oldest", "Popular")
	AssertRender(t, b, "chooser/editable-text-field-click", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ch.textField.Send(events.Click)
	})
}

func TestChooserAllowNewClick(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	ch := NewChooser(b).SetAllowNew(true).SetStrings("Newest", "Oldest", "Popular")
	AssertRender(t, b, "chooser/allow-new-click", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ch.Send(events.Click)
	})
}

func TestChooserEditableAllowNewClick(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	ch := NewChooser(b).SetEditable(true).SetAllowNew(true).SetStrings("Newest", "Oldest", "Popular")
	AssertRender(t, b, "chooser/editable-allow-new-click", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ch.Send(events.Click)
	})
}

func TestChooserEditableAllowNewTextFieldClick(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	ch := NewChooser(b).SetEditable(true).SetAllowNew(true).SetStrings("Newest", "Oldest", "Popular")
	AssertRender(t, b, "chooser/editable-allow-new-text-field-click", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ch.textField.HandleEvent(events.NewKey(events.KeyChord, 'O', 0, 0))
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ch.textField.Send(events.Click)
	})
}

func TestChooserChange(t *testing.T) {
	b := NewBody()
	ch := NewChooser(b).SetStrings("Newest", "Oldest", "Popular")
	index := -1
	item := ChooserItem{}
	ch.OnChange(func(e events.Event) {
		index = ch.CurrentIndex
		item = ch.CurrentItem
	})
	AssertRender(t, b, "chooser/change", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ch.selectItemEvent(1)
		assert.Equal(t, 1, index)
		assert.Equal(t, ChooserItem{Value: "Oldest"}, item)
	})
}
