// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"github.com/stretchr/testify/assert"
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
	first, _ := ts.NewTab("First")
	NewText(first).SetText("I am first!")
	second, _ := ts.NewTab("Second")
	NewText(second).SetText("I am second!")
	b.AssertRender(t, "tabs/widgets")
}

func TestTabsClick(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b)
	first, _ := ts.NewTab("First")
	NewText(first).SetText("I am first!")
	second, _ := ts.NewTab("Second")
	NewText(second).SetText("I am second!")
	b.AssertRender(t, "tabs/click", func() {
		ts.getTabs().Child(1).(Widget).AsWidget().Send(events.Click)
	})
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
	_, tb := ts.NewTab("First")
	tb.SetIcon(icons.Home)
	_, tb = ts.NewTab("Second")
	tb.SetIcon(icons.Explore)
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
	_, tb := ts.NewTab("First")
	tb.SetIcon(icons.Home)
	_, tb = ts.NewTab("Second")
	tb.SetIcon(icons.Explore)
	_, tb = ts.NewTab("Third")
	tb.SetIcon(icons.History)
	b.AssertRender(t, "tabs/navigation")
}

func TestTabsNew(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b).SetNewTabButton(true)
	ts.NewTab("First")
	ts.NewTab("Second")
	b.AssertRender(t, "tabs/new")
}

func TestTabsNewClick(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b).SetNewTabButton(true)
	ts.NewTab("First")
	ts.NewTab("Second")
	b.AssertRender(t, "tabs/new-click", func() {
		AsWidget(ts.getTabs().ChildByName("new-tab-button")).Send(events.Click)
		assert.Equal(t, ts.NumTabs(), 3)
	})
}
