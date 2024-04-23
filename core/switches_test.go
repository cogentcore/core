// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
)

func TestSwitches(t *testing.T) {
	b := NewBody()
	NewSwitches(b).SetStrings("Go", "Python", "C++")
	b.AssertRender(t, "switches/basic")
}

func TestSwitchesItems(t *testing.T) {
	b := NewBody()
	sw := NewSwitches(b).SetItems(
		SwitchItem{Text: "Go", Tooltip: "Elegant, fast, and easy-to-use"},
		SwitchItem{Text: "Python", Tooltip: "Slow and duck-typed"},
		SwitchItem{Text: "C++", Tooltip: "Hard to use and slow to compile"},
	)
	b.AssertRender(t, "switches/items", func() {
		assert.Equal(t, "Slow and duck-typed", sw.Child(1).(Widget).AsWidget().Tooltip)
	})
}

func TestSwitchesMutex(t *testing.T) {
	b := NewBody()
	sw := NewSwitches(b).SetMutex(true).SetStrings("Go", "Python", "C++")
	b.AssertRender(t, "switches/mutex", func() {
		sw.Child(0).(Widget).Send(events.Click)
		sw.Child(1).(Widget).Send(events.Click)
		assert.Equal(t, "Python", sw.SelectedItem().Text)
		assert.Equal(t, []SwitchItem{{Text: "Python"}}, sw.SelectedItems())
	})
}

func TestSwitchesChips(t *testing.T) {
	b := NewBody()
	NewSwitches(b).SetType(SwitchChip).SetStrings("Go", "Python", "C++")
	b.AssertRender(t, "switches/chips")
}

func TestSwitchesCheckboxes(t *testing.T) {
	b := NewBody()
	NewSwitches(b).SetType(SwitchCheckbox).SetStrings("Go", "Python", "C++")
	b.AssertRender(t, "switches/checkboxes")
}

func TestSwitchesRadioButtons(t *testing.T) {
	b := NewBody()
	NewSwitches(b).SetType(SwitchRadioButton).SetStrings("Go", "Python", "C++")
	b.AssertRender(t, "switches/radio-buttons")
}

func TestSwitchesSegmentedButtons(t *testing.T) {
	b := NewBody()
	NewSwitches(b).SetType(SwitchSegmentedButton).SetStrings("Go", "Python", "C++")
	b.AssertRender(t, "switches/segmented-buttons")
}

func TestSwitchesChange(t *testing.T) {
	b := NewBody()
	sw := NewSwitches(b).SetStrings("Go", "Python", "C++")
	selected := []SwitchItem{}
	sw.OnChange(func(e events.Event) {
		selected = sw.SelectedItems()
	})
	b.AssertRender(t, "switches/change", func() {
		sw.Child(0).(Widget).Send(events.Click)
		sw.Child(2).(Widget).Send(events.Click)
		assert.Equal(t, []SwitchItem{{Text: "Go"}, {Text: "C++"}}, selected)
	})
}
