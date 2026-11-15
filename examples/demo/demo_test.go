// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
)

// TestDemo tests the home tab of the demo, and its widgets tab.
func TestDemo(t *testing.T) {
	b := core.NewBody("Cogent Core Demo")
	makeBody(b)
	core.AssertRender(t, b, "demo")

	b = core.NewBody("Cogent Core Demo")
	makeBody(b)
	ts := tree.ChildByType[*core.Tabs](b)
	core.AssertRender(t, b, "demo-widgets", core.Locked(b, func() {
		ts.SelectTabByName("Widgets")
	}))
}

// TestDemoTabs tests that every tab of the demo, and every tab in them,
// can be shown (their content is made when they are first shown).
func TestDemoTabs(t *testing.T) {
	b := core.NewBody("Cogent Core Demo")
	makeBody(b)
	ts := tree.ChildByType[*core.Tabs](b)
	var steps []func()
	for i := range 5 {
		steps = append(steps, core.Locked(b, func() {
			ts.SelectTabIndex(i)
		}), core.Locked(b, func() {
			fr, _ := ts.TabAtIndex(i)
			if sub := tree.ChildByType[*core.Tabs](fr); sub != nil {
				for j := range sub.NumTabs() {
					sub.SelectTabIndex(j)
				}
			}
		}))
	}
	img := core.RenderImage(b, steps...)
	if img == nil {
		t.Fatal("the window was closed")
	}
}
