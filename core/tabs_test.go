// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles/states"
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
		b.AsyncLock()
		defer b.AsyncUnlock()
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
		b.AsyncLock()
		defer b.AsyncUnlock()
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
		b.AsyncLock()
		defer b.AsyncUnlock()
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
		b.AsyncLock()
		defer b.AsyncUnlock()
		AsWidget(ts.getTabs().ChildByName("new-tab-button")).Send(events.Click)
		assert.Equal(t, ts.NumTabs(), 3)
	})
}

// TestTabsShowBeforeShown tests that the widgets of a tab that is selected
// before its window is shown get the show event once its scene has been
// shown, when they can be updated from other goroutines
// (AsyncLockIfShown), not before (as the scripts of the pages of the tabs
// of the web browser of Cogent, which were stopped as their page seemed
// to be gone).
func TestTabsShowBeforeShown(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b)
	ts.NewTab("First")
	second, _ := ts.NewTab("Second")
	tx := NewText(second).SetText("second")
	var shows, early int
	tx.OnShow(func(e events.Event) {
		shows++
		if !tx.Scene.hasShown.Load() {
			early++
		}
	})
	ts.SelectTabIndex(1)
	b.RenderImage()
	assert.Equal(t, 0, early, "shown before its scene was")
	assert.GreaterOrEqual(t, shows, 1)
}

// TestTabsKeys tests that the keys for the next and previous tabs
// (Shift+RightArrow and Shift+LeftArrow), sent to the window, select them
// when no widget in the tabs handles them: after a press in a tab whose
// content is not focusable, which gives the tabs the focus, cycling around
// and focusing the text field of a tab, which handles them (selecting),
// and the button of another, which does not.
func TestTabsKeys(t *testing.T) {
	b := NewBody() // (loads the settings, with the key map, the first time)
	prevMap := keymap.ActiveMapName
	keymap.SetActiveMapName("MacStandard")
	defer keymap.SetActiveMapName(prevMap)
	ts := NewTabs(b)
	f0, _ := ts.NewTab("Field")
	tf := NewTextField(f0).SetText("text")
	f1, _ := ts.NewTab("Text")
	tx := NewText(f1).SetText("Not focusable")
	f2, _ := ts.NewTab("Other")
	bt := NewButton(f2).SetText("Button")
	var shift key.Modifiers
	shift.SetFlag(true, key.Shift)
	current := func() int {
		b.AsyncLock()
		defer b.AsyncUnlock()
		_, idx := ts.CurrentTab()
		return idx
	}
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ts.SelectTabIndex(1)
	}, func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		press := func(code key.Codes) {
			src.Key(events.KeyDown, 0, code, shift)
			src.Key(events.KeyUp, 0, code, shift)
			time.Sleep(100 * time.Millisecond)
		}
		b.AsyncLock()
		pt := tx.Geom.TotalBBox.Min.Add(image.Pt(2, 2))
		b.AsyncUnlock()
		src.MouseButton(events.MouseDown, events.Left, pt, 0)
		src.MouseButton(events.MouseUp, events.Left, pt, 0)
		time.Sleep(100 * time.Millisecond)
		press(key.CodeRightArrow)
		assert.Equal(t, 2, current(), "next")
		press(key.CodeRightArrow)
		assert.Equal(t, 0, current(), "next, around")
		b.AsyncLock()
		assert.True(t, tf.StateIs(states.Focused), "the text field of the tab is focused")
		b.AsyncUnlock()
		press(key.CodeRightArrow)
		assert.Equal(t, 0, current(), "the text field handles it")
		b.AsyncLock()
		ts.Events().setFocusQuiet(ts) // (as by a press in the tabs)
		b.AsyncUnlock()
		press(key.CodeLeftArrow)
		assert.Equal(t, 2, current(), "previous, around")
		b.AsyncLock()
		assert.True(t, bt.StateIs(states.Focused), "the button of the tab is focused")
		b.AsyncUnlock()
		press(key.CodeLeftArrow)
		assert.Equal(t, 1, current(), "previous, from the button, which does not handle it")
	})
}
