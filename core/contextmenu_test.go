// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContextMenuKeyMaps tests the context menu bindings of the standard
// key maps: Shift+F10 and the Menu key, and on macOS Option+Return.
func TestContextMenuKeyMaps(t *testing.T) {
	for _, km := range keymap.StandardMaps {
		chords := []key.Chord{"Shift+F10", "ContextMenu"}
		if strings.HasPrefix(km.Name, "Mac") {
			chords = append(chords, "Alt+ReturnEnter")
		}
		for _, ch := range chords {
			assert.Equal(t, keymap.ContextMenu, km.Map[ch], "%s: %s", km.Name, ch)
		}
	}
}

// TestContextMenuKey tests that the context menu key functions show the
// context menu of the focused widget, at the widget.
func TestContextMenuKey(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("First")
	bt := NewButton(b).SetText("Button")
	bt.AddContextMenu(func(m *Scene, pos image.Point) {
		NewButton(m).SetText("Item")
	})
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b.AssertRender(t, "context-menu-key", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		var shift key.Modifiers
		shift.SetFlag(true, key.Shift)
		menu := func() *Stage {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return b.Scene.Stage.popups.top()
		}
		for _, ch := range []struct {
			name string
			code key.Codes
			mods key.Modifiers
		}{
			{"Shift+F10", key.CodeF10, shift},
			{"ContextMenu", key.CodeContextMenu, 0},
		} {
			assert.Equal(t, keymap.ContextMenu, keymap.Of(key.Chord(ch.name)), ch.name)
			b.AsyncLock()
			bt.SetFocus()
			b.AsyncUnlock()
			src.Key(events.KeyDown, 0, ch.code, ch.mods)
			src.Key(events.KeyUp, 0, ch.code, ch.mods)
			var st *Stage
			for range 100 {
				if st = menu(); st != nil {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			require.NotNil(t, st, ch.name)
			b.AsyncLock()
			bb := bt.winBBox()
			assert.Equal(t, MenuStage, st.Type, ch.name)
			assert.True(t, st.Pos.In(bb.Inset(-1)) || st.Pos == bb.Max, "%s: the menu at %v is at the button %v", ch.name, st.Pos, bb)
			st.ClosePopup()
			b.AsyncUnlock()
			for range 100 {
				if menu() == nil {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
		}
	})
}

// TestNewWindow tests that the Window menu of the standard context menu
// starts with New window, with the New key, when the app sets
// [NewWindowFunc], and that the New key, sent to the window, makes a new
// window with it, from the scene of the window.
func TestNewWindow(t *testing.T) {
	skipOnWeb(t, "there is one window")
	b := NewBody() // (loads the settings, with the key map, the first time)
	NewText(b).SetText("Main")
	prevMap := keymap.ActiveMapName
	keymap.SetActiveMapName("MacStandard")
	defer keymap.SetActiveMapName(prevMap)
	var from *Scene
	var made []*Body
	NewWindowFunc = func(sc *Scene) *Body {
		from = sc
		nb := NewBody("New")
		NewText(nb).SetText("New window")
		made = append(made, nb)
		return nb
	}
	defer func() { NewWindowFunc = nil }()
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		m := NewScene()
		b.Scene.standardContextMenu(m, image.Point{})
		var win *Button
		for _, c := range m.Children {
			if bt, ok := c.(*Button); ok && bt.Text == "Window" {
				win = bt
			}
		}
		require.NotNil(t, win)
		wm := NewScene()
		win.Menu(wm, image.Point{})
		first, ok := wm.Child(0).(*Button)
		require.True(t, ok)
		assert.Equal(t, "New window", first.Text)
		assert.Equal(t, keymap.New.Chord(), first.Shortcut)
	}, func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		var meta key.Modifiers
		meta.SetFlag(true, key.Meta)
		src.Key(events.KeyDown, 'n', key.CodeN, meta)
		src.Key(events.KeyUp, 'n', key.CodeN, meta)
		assert.Eventually(t, func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return len(made) == 1 && made[0].Scene.RenderWindow() != nil
		}, 3*time.Second, 20*time.Millisecond, "a new window")
		b.AsyncLock()
		assert.Equal(t, b.Scene, from)
		b.AsyncUnlock()
		if len(made) == 1 {
			made[0].AsyncLock()
			made[0].Close()
			made[0].AsyncUnlock()
		}
	})
}
