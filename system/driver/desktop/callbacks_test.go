// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/system"
	"github.com/cogentcore/glfw4/v3.4/glfw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestWindow returns a new window for a test, which is closed after it.
func newTestWindow(t *testing.T) *Window {
	t.Helper()
	requireGUI(t)
	gpu.SkipIfNoGPU(t)
	sw, err := TheApp.NewWindow(&system.NewWindowOptions{Title: "callbacks test", Size: image.Pt(320, 240)})
	require.NoError(t, err)
	w := sw.(*Window)
	t.Cleanup(w.Close)
	return w
}

// drainEvents returns the events of the window that arrive within the
// given time, after any already queued.
func drainEvents(w *Window, wait time.Duration) []events.Event {
	var evs []events.Event
	end := time.Now().Add(wait)
	for time.Now().Before(end) {
		e, ok := w.Events().Deque.PollEvent()
		if !ok {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		evs = append(evs, e)
	}
	return evs
}

// hasEvent returns the first event of the given type, or nil.
func hasEvent(evs []events.Event, typ events.Types) events.Event {
	for _, e := range evs {
		if e.Type() == typ {
			return e
		}
	}
	return nil
}

// TestWindowCallbacks tests the events that the glfw callbacks of a window
// send, called directly (on the main thread, as glfw calls them).
func TestWindowCallbacks(t *testing.T) {
	w := newTestWindow(t)
	drainEvents(w, 100*time.Millisecond)
	gw := w.Glw
	onMain := func(f func()) { TheApp.RunOnMain(f) }

	onMain(func() {
		w.MouseButtonEvent(gw, glfw.MouseButtonLeft, glfw.Press, 0)
		w.MouseButtonEvent(gw, glfw.MouseButtonMiddle, glfw.Release, 0)
		w.MouseButtonEvent(gw, glfw.MouseButtonLeft, glfw.Press, glfw.ModControl)
		w.ScrollEvent(gw, 0, 1)
		w.CursorPosEvent(gw, 10, 20)
		w.CursorEnterEvent(gw, true)
		w.DropEvent(gw, nil)
		w.DropEvent(gw, []string{"/tmp/a.txt", "/tmp/b.txt"})
	})
	evs := drainEvents(w, 100*time.Millisecond)
	var downs, ups []*events.Mouse
	for _, e := range evs {
		if m, ok := e.(*events.Mouse); ok {
			switch e.Type() {
			case events.MouseDown:
				downs = append(downs, m)
			case events.MouseUp:
				ups = append(ups, m)
			}
		}
	}
	require.Len(t, downs, 2)
	assert.Equal(t, events.Left, downs[0].Button)
	assert.Equal(t, events.Right, downs[1].Button, "control-click is a right click")
	require.Len(t, ups, 1)
	assert.Equal(t, events.Middle, ups[0].Button)

	sc := hasEvent(evs, events.Scroll)
	require.NotNil(t, sc, "scroll")
	assert.Negative(t, sc.(*events.MouseScroll).Delta.Y, "scrolling up is negative")
	mv := hasEvent(evs, events.MouseDrag) // (the left button is pressed)
	require.NotNil(t, mv, "move")
	assert.Equal(t, w.MousePosToPoint(10, 20), mv.WindowPos())
	drop := hasEvent(evs, events.Drop)
	require.NotNil(t, drop, "the dropped files")
	assert.Len(t, drop.(*events.DragDrop).Data, 2)

	// the window callbacks
	onMain(func() {
		w.Focused(gw, false)
		w.Focused(gw, true)
		w.Iconify(gw, true)
		w.Iconify(gw, false)
	})
	evs = drainEvents(w, 100*time.Millisecond)
	assert.NotNil(t, hasEvent(evs, events.Window), "window events")
	var acts []events.WinActions
	for _, e := range evs {
		if we, ok := e.(*events.WindowEvent); ok {
			acts = append(acts, we.Action)
		}
	}
	assert.Contains(t, acts, events.WinFocusLost)
	assert.Contains(t, acts, events.WinFocus)
	assert.Contains(t, acts, events.WinMinimize)
	assert.Equal(t, image.Pt(-1, -1), w.Event.Last.MousePos, "reset by losing the focus")
	assert.False(t, w.Is(system.Minimized), "restored")

	onMain(func() {
		x, y := gw.GetPos()
		w.Moved(gw, x, y)
		ww, wh := gw.GetSize()
		w.WinResized(gw, ww, wh)
		fw, fh := gw.GetFramebufferSize()
		w.FbResized(gw, fw+1, fh) // a different size updates the geometry
	})
	evs = drainEvents(w, 300*time.Millisecond)
	assert.NotNil(t, hasEvent(evs, events.WindowResize), "the geometry is updated: %v", evs)

	closeReq := make(chan bool, 1)
	w.SetCloseReqFunc(func(win system.Window) { closeReq <- true })
	onMain(func() { w.OnCloseReq(gw) })
	select {
	case <-closeReq:
	case <-time.After(2 * time.Second):
		t.Error("no close request")
	}
}

// TestWindowGeometry tests the position, geometry, visibility, mouse,
// cursor mode, and icon of a (hidden) window.
func TestWindowGeometry(t *testing.T) {
	w := newTestWindow(t)
	assert.True(t, w.IsVisible())
	sc := w.Screen()
	pos := w.Position(nil)
	assert.Equal(t, pos.Sub(sc.Geometry.Min), w.Position(sc))

	w.SetPos(image.Pt(30, 40), sc)
	w.SetGeometry(false, image.Point{}, image.Point{}, nil) // keeps them
	w.SetGeometry(false, image.Pt(30, 40), w.Size(), sc)
	w.SetIcon([]image.Image{image.NewRGBA(image.Rect(0, 0, 16, 16))})
	w.SetMousePos(5, 5)
	TheApp.RunOnMain(func() {
		w.SetCursorEnabled(false, true)
		assert.False(t, w.CursorEnabled)
		w.CursorPosEvent(w.Glw, 50, 50) // the cursor is put back
		w.SetCursorEnabled(true, false)
		assert.True(t, w.CursorEnabled)
	})
	// (Raise and Minimize are not called, as raising a hidden window shows it)
	drainEvents(w, 100*time.Millisecond)
	assert.False(t, w.IsClosed())
}

// TestAppMisc tests the clipboard of a window, the thread priority, and
// the theme changes, which update the screens of the windows.
func TestAppMisc(t *testing.T) {
	w := newTestWindow(t)
	assert.Same(t, TheClipboard, TheApp.Clipboard(w))
	drainEvents(w, 50*time.Millisecond)
	dark := TheApp.IsDark()
	TheApp.setDark(!dark)
	assert.Equal(t, !dark, TheApp.IsDark())
	evs := drainEvents(w, 200*time.Millisecond)
	var screenUpdate bool
	for _, e := range evs {
		if we, ok := e.(*events.WindowEvent); ok && we.Action == events.ScreenUpdate {
			screenUpdate = true
		}
	}
	assert.True(t, screenUpdate, "a change of the theme updates the windows")
	TheApp.setDark(dark)
	TheApp.setDark(dark) // no change
	assert.Equal(t, dark, TheApp.IsDark())

	TheApp.RunOnMain(func() { TheApp.MonitorChange(nil, glfw.Connected) })
	assert.Positive(t, TheApp.NScreens())
}
