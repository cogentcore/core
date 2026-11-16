// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"slices"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

func TestDialogMessage(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(300))
	})
	AssertRender(t, b, "dialog/message", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		MessageDialog(b, "Something happened", "Message")
	})
}

// TestDialogMinSize tests that the [Stage.MinSize] of a dialog sets its
// minimum initial size, in dp, larger than its content.
func TestDialogMinSize(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1000), units.Dp(800)) })
	b.runAndShowNewWindow()
	rw := b.Scene.RenderWindow()
	sizeOf := func(minSize math32.Vector2) image.Point {
		d := NewBody()
		NewText(d).SetText("Small")
		b.AsyncLock()
		st := d.NewDialog(b).SetMinSize(minSize).Run()
		b.AsyncUnlock()
		b.waitNoEvents(rw)
		b.AsyncLock()
		defer b.AsyncUnlock()
		sz := st.Scene.SceneGeom.Size
		st.ClosePopup()
		d.Close()
		return sz
	}
	dp := b.Scene.Styles.UnitContext.Dp(1)
	small := sizeOf(math32.Vector2{})
	assert.Less(t, float32(small.X), 700*dp, "the default")
	big := sizeOf(math32.Vec2(700, 500))
	assert.GreaterOrEqual(t, float32(big.X), 700*dp)
	assert.GreaterOrEqual(t, float32(big.Y), 500*dp)
}

// TestDialogWindowCloseFromOwnLoop tests that a dialog in a new window that
// is closed by its own event loop, as by a key that presses its OK button,
// is removed from the windows, and its event loop stops: its system window
// is closed before the loop gets the close event, so it cannot be locked
// then. It stayed, keeping the keyboard focus of the system from the
// window behind it.
func TestDialogWindowCloseFromOwnLoop(t *testing.T) {
	b := NewBody()
	b.runAndShowNewWindow()
	defer func() {
		b.AsyncLock()
		b.Close()
		b.AsyncUnlock()
	}()
	d := NewBody("dialog")
	NewText(d).SetText("Dialog")
	d.AddBottomBar(func(bar *Frame) {
		d.AddCancel(bar)
		d.AddOK(bar)
	})
	showed := make(chan struct{})
	d.OnFinal(events.Show, func(e events.Event) { close(showed) })
	b.AsyncLock()
	d.NewWindowDialog(b).Run()
	b.AsyncUnlock()
	<-showed
	dw := d.Scene.RenderWindow()
	assert.NotSame(t, b.Scene.RenderWindow(), dw, "in a new window")
	dw.SystemWindow.Events().KeyChord(0, key.CodeReturnEnter, key.Meta) // the OK button
	deadline := time.Now().Add(5 * time.Second)
	for slices.Contains(AllRenderWindows.windows(), dw) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	assert.NotContains(t, AllRenderWindows.windows(), dw, "the dialog window is closed")
}
