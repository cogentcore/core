// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// TestWindowTitle tests that the title of the window is that of its main
// stage on top: snackbars, which have no title, do not change it, and a
// dialog has its title while it is open (#1515).
func TestWindowTitle(t *testing.T) {
	b := NewBody("App")
	NewButton(b).SetText("Button")
	title := func() string {
		return b.Scene.RenderWindow().SystemWindow.Title()
	}
	// waitTitle waits for the window to have the given title, which is set
	// when the stage is shown, after it is rendered
	waitTitle := func(want, what string) {
		t.Helper()
		for range 100 {
			if title() == want {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		assert.Equal(t, want, title(), what)
	}
	var d *Body
	imagex.UpdateTestImagesFor(t) // the images are not compared
	AssertRender(t, b, "title", func() {
		waitTitle("App", "the title of the window")
		b.AsyncLock() // (it is rendered meanwhile)
		MessageSnackbar(b, "A message")
		b.AsyncUnlock()
	}, func() {
		time.Sleep(300 * time.Millisecond) // the snackbar is shown
		waitTitle("App", "a snackbar does not change the title")
		b.AsyncLock()
		d = NewBody("Dlg")
		NewText(d).SetText("A dialog")
		d.AddOKOnly().RunDialog(b)
		b.AsyncUnlock()
	}, func() {
		waitTitle("Dlg • App", "the title of the dialog")
		b.AsyncLock()
		d.Close()
		b.AsyncUnlock()
	}, Locked(b, func() {
		waitTitle("App", "back after the dialog")
	}))
}

// TestWindowTitleMenuSearch tests that the window gets its title back after
// the menu search dialog, whether it is canceled or an item is chosen in it,
// including one that runs a dialog, also in a window of its own, which is
// the current window when the menu search is closed.
func TestWindowTitleMenuSearch(t *testing.T) {
	b := NewBody("App")
	var winDlg *Body
	b.Scene.AddContextMenu(func(m *Scene, _ image.Point) {
		NewButton(m).SetText("Nothing")
		NewButton(m).SetText("Open dialog").OnClick(func(e events.Event) {
			d := NewBody("Opened")
			d.AddOKOnly().RunDialog(b)
		})
		NewButton(m).SetText("Open window dialog").OnClick(func(e events.Event) {
			winDlg = NewBody("Window dialog")
			winDlg.AddOKOnly().RunWindowDialog(b)
		})
	})
	title := func() string {
		return b.Scene.RenderWindow().SystemWindow.Title()
	}
	waitTitle := func(want, what string) {
		t.Helper()
		for range 100 {
			b.AsyncLock()
			tl := title()
			b.AsyncUnlock()
			if tl == want {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		b.AsyncLock()
		assert.Equal(t, want, title(), what)
		b.AsyncUnlock()
	}
	b.RunWindow()
	defer func() {
		b.AsyncLock()
		b.Close()
		b.AsyncUnlock()
	}()
	waitTitle("App", "the title of the window")
	// search opens the menu search, and returns its chooser and dialog
	search := func() (*Chooser, *Scene) {
		b.AsyncLock()
		b.Scene.MenuSearchDialog("Menu search", "Search")
		b.AsyncUnlock()
		waitTitle("Menu search • App", "the title of the menu search")
		b.AsyncLock()
		defer b.AsyncUnlock()
		sc := b.Scene.Stage.Mains.top().Scene
		var ch *Chooser
		sc.WalkDown(func(n tree.Node) bool {
			if c, ok := n.(*Chooser); ok && ch == nil {
				ch = c
			}
			return ch == nil
		})
		ch.Items = nil
		ch.CallItemsFuncs()
		return ch, sc
	}
	choose := func(ch *Chooser, text string) {
		b.AsyncLock()
		defer b.AsyncUnlock()
		for i, it := range ch.Items {
			if it.Text == text {
				ch.selectItemEvent(i)
				return
			}
		}
		t.Fatalf("no item %q in %v", text, ch.Items)
	}

	_, sc := search()
	b.AsyncLock()
	sc.Close()
	b.AsyncUnlock()
	waitTitle("App", "after canceling")

	ch, _ := search()
	choose(ch, "Nothing")
	waitTitle("App", "after choosing an item")

	ch, _ = search()
	choose(ch, "Open dialog")
	waitTitle("Opened • App", "the dialog of the item")
	b.AsyncLock()
	b.Scene.Stage.Mains.top().Scene.Close()
	b.AsyncUnlock()
	waitTitle("App", "after the dialog of the item")

	ch, _ = search()
	choose(ch, "Open window dialog")
	waitTitle("App", "after a dialog in a window of its own")
	if winDlg != nil {
		winDlg.AsyncLock()
		winDlg.Close()
		winDlg.AsyncUnlock()
	}
}
