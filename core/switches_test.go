// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "testing"

func TestSwitches(t *testing.T) {
	b := NewBody()
	NewSwitches(b).SetStrings("Go", "Python", "C++")
	b.AssertRender(t, "switches/basic")
}

func TestSwitchesItems(t *testing.T) {
	b := NewBody()
	NewSwitches(b).SetItems(
		SwitchItem{Label: "Go", Tooltip: "Elegant, fast, and easy-to-use"},
		SwitchItem{Label: "Python", Tooltip: "Slow and duck-typed"},
		SwitchItem{Label: "C++", Tooltip: "Hard to use and slow to compile"},
	)
	b.AssertRender(t, "switches/items")
}
