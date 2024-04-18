// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/icons"
	"github.com/stretchr/testify/assert"
)

func TestChooserSetStrings(t *testing.T) {
	b := NewBody()
	NewChooser(b).SetStrings("macOS", "Windows", "Linux")
	b.AssertRender(t, "chooser/set-strings")
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
