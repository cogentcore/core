// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFocusKeys tests moving the focus with tab and shift-tab, focusing the last widget, and opening the context menu of the
// focused widget with a key.
func TestFocusKeys(t *testing.T) {
	b := NewBody()
	b1 := NewButton(b).SetText("One")
	b2 := NewButton(b).SetText("Two")
	b3 := NewButton(b).SetText("Three")
	menus := 0
	b2.AddContextMenu(func(m *Scene, pos image.Point) {
		menus++
		NewButton(m).SetText("Item")
	})
	RenderImage(b, Locked(b, func() {
		em := &b.Scene.Events
		send := func(kf keymap.Functions) {
			rn, code, mods, err := kf.Chord().Decode()
			require.NoError(t, err)
			e := events.NewKey(events.KeyChord, rn, code, mods)
			em.handleEvent(e)
		}
		b1.SetFocus()
		send(keymap.FocusNext)
		assert.Equal(t, Widget(b2), em.focus)
		send(keymap.FocusNext)
		assert.Equal(t, Widget(b3), em.focus)
		send(keymap.FocusPrev)
		assert.Equal(t, Widget(b2), em.focus)

		send(keymap.ContextMenu)
		assert.Equal(t, 1, menus, "the context menu of the focused widget")

		require.True(t, em.focusLast())
		assert.Equal(t, Widget(b3), em.focus)
		// past the last widget, there is no focus, as in a browser, and
		// then the first one again
		send(keymap.FocusNext)
		assert.Nil(t, em.focus)
		send(keymap.FocusNext)
		assert.Equal(t, Widget(b1), em.focus)
		send(keymap.FocusPrev)
		assert.Nil(t, em.focus)
		send(keymap.FocusPrev)
		assert.Equal(t, Widget(b3), em.focus, "the last one again")
	}))
}

// TestWinSnapshotKey tests saving screenshots of a window with its key:
// an image, an SVG, and a PDF, in the data directory of the app.
func TestWinSnapshotKey(t *testing.T) {
	skipOnWeb(t, "writes files")
	b := NewBody()
	NewText(b).SetText("Snapshot")
	RenderImage(b, Locked(b, func() {
		rn, code, mods, err := keymap.WinSnapshot.Chord().Decode()
		require.NoError(t, err)
		b.Scene.Events.handleEvent(events.NewKey(events.KeyChord, rn, code, mods))
	}))
	saved := map[string]bool{}
	ents, err := os.ReadDir(TheApp.AppDataDir())
	require.NoError(t, err)
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), "screenshot-") {
			saved[filepath.Ext(e.Name())] = true
			os.Remove(filepath.Join(TheApp.AppDataDir(), e.Name()))
		}
	}
	assert.Equal(t, map[string]bool{".png": true, ".svg": true, ".pdf": true}, saved)
}

// TestRepeatClick tests that holding the mouse down on a button of a
// spinner clicks it repeatedly, until the mouse is released.
func TestRepeatClick(t *testing.T) {
	prev := TimingSettings.RepeatClickTime
	TimingSettings.RepeatClickTime = 10 * time.Millisecond
	defer func() { TimingSettings.RepeatClickTime = prev }()
	b := NewBody()
	sp := NewSpinner(b).SetStep(1)
	var p image.Point
	var after float32
	RenderImage(b, Locked(b, func() {
		bt := sp.ChildByName("trail-icon").(Widget).AsWidget()
		bb := bt.Geom.TotalBBox
		p = bb.Min.Add(bb.Max).Div(2)
	}), func() {
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.MouseButton(events.MouseDown, events.Left, p, 0)
		time.Sleep(300 * time.Millisecond) // 80ms to start, then every 10ms
		ev.MouseButton(events.MouseUp, events.Left, p, 0)
	}, Locked(b, func() {
		after = sp.Value
		assert.Greater(t, after, float32(3), "clicked repeatedly")
	}), func() {
		time.Sleep(100 * time.Millisecond)
	}, Locked(b, func() {
		assert.Equal(t, after, sp.Value, "stopped by the release")
	}))
}
