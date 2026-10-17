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

func TestTabsUpdateSelected(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b)
	first, _ := ts.NewTab("First")
	NewText(first).SetText("I am first!")
	second, _ := ts.NewTab("Second")
	updates := 0
	text := NewText(second)
	text.Updater(func() {
		updates++
		text.SetText("I am second!")
	})
	b.AssertRender(t, "tabs/click", func() {
		// The content of a tab is updated when it is selected, not before.
		assert.Zero(t, updates)
		ts.getTabs().Child(1).(Widget).AsWidget().Send(events.Click)
	})
	assert.NotZero(t, updates)
}

func TestTabsLazy(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b)
	builds := 0
	ts.NewLazyTab("First", func(tab *Frame) {
		builds++
		NewText(tab).SetText("I am first!")
	})
	ts.NewLazyTab("Second", func(tab *Frame) {
		builds++
		NewText(tab).SetText("I am second!")
	})
	// Only the content of the selected tab is made, and only once.
	assert.Equal(t, 1, builds)
	b.AssertRender(t, "tabs/click", func() {
		assert.Equal(t, 1, builds)
		ts.getTabs().Child(1).(Widget).AsWidget().Send(events.Click)
	})
	assert.Equal(t, 2, builds)
	ts.SelectTabIndex(0)
	ts.SelectTabIndex(1)
	assert.Equal(t, 2, builds)
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
