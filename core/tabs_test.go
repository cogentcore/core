// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/icons"
)

func TestTabs(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b)
	ts.NewTab("First")
	ts.NewTab("Second")
	b.AssertRender(t, "tabs/basic")
}

func TestTabsWidgets(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b)
	first := ts.NewTab("First")
	NewLabel(first).SetText("I am first!")
	second := ts.NewTab("Second")
	NewLabel(second).SetText("I am second!")
	b.AssertRender(t, "tabs/widgets")
}

func TestTabsMany(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b)
	ts.NewTab("First")
	ts.NewTab("Second")
	ts.NewTab("Third")
	ts.NewTab("Fourth")
	b.AssertRender(t, "tabs/many")
}

func TestTabsIcons(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b)
	ts.NewTab("First", icons.Home)
	ts.NewTab("Second", icons.Explore)
	b.AssertRender(t, "tabs/icons")
}

func TestTabsFunctional(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b).SetType(FunctionalTabs)
	ts.NewTab("First")
	ts.NewTab("Second")
	ts.NewTab("Third")
	b.AssertRender(t, "tabs/functional")
}

func TestTabsNavigation(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b).SetType(NavigationAuto)
	ts.NewTab("First", icons.Home)
	ts.NewTab("Second", icons.Explore)
	ts.NewTab("Third", icons.History)
	b.AssertRender(t, "tabs/navigation")
}

func TestTabsNew(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b).SetNewTabButton(true)
	ts.NewTab("First")
	ts.NewTab("Second")
	b.AssertRender(t, "tabs/new")
}
