// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/events"
	"testing"

	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// currentTabName returns the name of the current tab.
func currentTabName(ts *Tabs) string {
	w, _ := ts.CurrentTab()
	if w == nil {
		return ""
	}
	return w.AsTree().Name
}

// TestTabsByName tests finding, selecting, and recycling tabs by name.
func TestTabsByName(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b)
	ts.NewTab("A")
	ts.NewTab("B")
	RenderImage(b, Locked(b, func() {
		assert.NotNil(t, ts.TabByName("B"))
		assert.Nil(t, ts.TabByName("Z"))
		assert.Equal(t, 1, ts.tabIndexByName("B"))
		assert.Equal(t, -1, ts.tabIndexByName("Z"))
		assert.NotNil(t, ts.SelectTabByName("B"))
		assert.Equal(t, "B", currentTabName(ts))
		assert.Nil(t, ts.SelectTabByName("Z"))

		fr := ts.RecycleTab("A")
		assert.Same(t, ts.TabByName("A"), fr, "an existing tab")
		assert.Equal(t, "A", currentTabName(ts))
		ts.RecycleTab("C")
		assert.Equal(t, 3, ts.NumTabs(), "a new tab")
		assert.Equal(t, "C", currentTabName(ts))

		tx := RecycleTabWidget[Text](ts, "D")
		require.NotNil(t, tx)
		assert.Same(t, tx, RecycleTabWidget[Text](ts, "D"), "the widget of the existing tab")
		_, tab := ts.TabAtIndex(3)
		assert.Equal(t, "D", tab.Label())
	}))
}

// TestTabsDelete tests that deleting a tab keeps the current tab, unless
// it is the one deleted, which selects the one before it.
func TestTabsDelete(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b)
	for _, n := range []string{"A", "B", "C", "D"} {
		ts.NewTab(n)
	}
	closed := -1
	ts.CloseTabFunc = func(idx int) { closed = idx }
	RenderImage(b, Locked(b, func() {
		ts.SelectTabByName("C")
		require.True(t, ts.DeleteTabIndex(0))
		assert.Equal(t, 0, closed)
		assert.Equal(t, "C", currentTabName(ts), "deleting a tab before the current one keeps it")
		require.True(t, ts.DeleteTabIndex(2))
		assert.Equal(t, "C", currentTabName(ts), "deleting a tab after the current one keeps it")
		require.True(t, ts.DeleteTabIndex(1))
		assert.Equal(t, "B", currentTabName(ts), "deleting the current tab selects the one before")
		assert.False(t, ts.DeleteTabIndex(5))
		require.True(t, ts.DeleteTabIndex(0))
		assert.Equal(t, 0, ts.NumTabs())
		w, idx := ts.CurrentTab()
		assert.Nil(t, w)
		assert.Equal(t, -1, idx)
	}))
}

// TestTabKeys tests the keys for the next and previous tabs, which cycle,
// given to the tabs that a widget is in.
func TestTabKeys(t *testing.T) {
	b := NewBody()
	ts := NewTabs(b)
	fa, _ := ts.NewTab("A")
	ts.NewTab("B")
	tf := NewTextField(fa)
	RenderImage(b, Locked(b, func() {
		ts.SelectTabByName("A")
		e := keyEventOf(keymap.NextTab)
		TabKeys(tf, e)
		assert.True(t, e.IsHandled())
		assert.Equal(t, "B", currentTabName(ts))
		TabKeys(tf, keyEventOf(keymap.NextTab))
		assert.Equal(t, "A", currentTabName(ts), "cycles to the first")
		TabKeys(tf, keyEventOf(keymap.PrevTab))
		assert.Equal(t, "B", currentTabName(ts), "and back to the last")
		other := keyEventOf(keymap.Copy)
		TabKeys(tf, other)
		assert.False(t, other.IsHandled())
	}))
}

// keyEventOf returns a key chord event for the first chord of the given
// key function in the active keymap.
func keyEventOf(kf keymap.Functions) *events.Key {
	rn, code, mods, _ := kf.Chord().Decode()
	return events.NewKey(events.KeyChord, rn, code, mods)
}
