// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
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
	imagex.UpdateTestImages = true // the images are not compared
	defer func() { imagex.UpdateTestImages = false }()
	b.AssertRender(t, "title", func() {
		waitTitle("App", "the title of the window")
		MessageSnackbar(b, "A message")
	}, func() {
		time.Sleep(300 * time.Millisecond) // the snackbar is shown
		waitTitle("App", "a snackbar does not change the title")
		d = NewBody("Dlg")
		NewText(d).SetText("A dialog")
		d.AddOKOnly().RunDialog(b)
	}, func() {
		waitTitle("Dlg • App", "the title of the dialog")
		d.Close()
	}, func() {
		waitTitle("App", "back after the dialog")
	})
}
