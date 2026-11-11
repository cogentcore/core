// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"sync"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/system"
)

// windowDragger is implemented by a widget that, when a drag of it leaves
// its window, moves its content into a new window, which is then moved
// with the mouse until the drag ends, as for a tab of [Tabs] with
// [Tabs.DragToWindow], instead of handing the drag to the operating system.
type windowDragger interface {
	// dragToWindow is called when a drag that the widget started leaves
	// its window, with the event. If it moves its content into a new
	// window, it returns the body of that window, and the function that
	// returns the position in it (in its pixels) that is kept under the
	// mouse while it is moved.
	dragToWindow(e events.Event) (*Body, func() image.Point)
}

// windowDrag is the state of a new window that is moved with the mouse (see
// [windowDragger]).
type windowDrag struct {
	// body is the body of the new window.
	body *Body

	// grab returns the position in the window that is kept under the mouse.
	grab func() image.Point
}

// dragToWindow moves the content of the source of the current drag into a
// new window when the drag leaves the window, if the source does that (see
// [windowDragger]), after which the window is moved with the mouse. It
// returns whether it did.
func (em *Events) dragToWindow(e events.Event) bool {
	if em.InWindow(e.WindowPos()) || !windowDragSupported() {
		return false
	}
	wd, ok := em.drag.(windowDragger)
	if !ok {
		return false
	}
	bd, grab := wd.dragToWindow(e)
	if bd == nil {
		return false
	}
	em.dragClearSprite()
	em.dragHoversClear(e)
	if DebugSettings.EventTrace {
		fmt.Println(em.drag, "dragged into a new window")
	}
	em.drag = nil
	em.dragData = nil
	em.dragImage = nil
	em.dragFiles = nil
	em.press = nil
	em.dragPress = nil
	em.dragWindow = &windowDrag{body: bd, grab: grab}
	em.moveDragWindow(e)
	e.SetHandled()
	return true
}

// windowDragSupported returns whether a drag can move content into a new
// window on the current platform: one with windows that can be moved.
func windowDragSupported() bool {
	p := TheApp.Platform()
	return !p.IsMobile() && p != system.Web && p != system.Offscreen
}

// handleWindowDrag handles a mouse event while a new window is moved with
// the mouse: it moves the window, until the mouse button is released.
func (em *Events) handleWindowDrag(e events.Event) bool {
	switch e.Type() {
	case events.MouseDrag:
		em.moveDragWindow(e)
	case events.MouseUp, events.MouseMove: // (a move if the release was missed)
		em.endWindowDrag()
		if e.Type() == events.MouseMove {
			return false
		}
	default:
		return false
	}
	e.SetHandled()
	return true
}

// moveDragWindow moves the window that is moved with the mouse, for the
// position of the mouse in the given event, so that its grab position is
// under the mouse.
func (em *Events) moveDragWindow(e events.Event) {
	wd := em.dragWindow
	src := em.RenderWindow()
	win := wd.body.Scene.RenderWindow()
	if src == nil || win == nil || src.SystemWindow == nil || win.SystemWindow == nil {
		return
	}
	// the position of the mouse on the screen, in the units of the window
	// manager (positions in windows are in pixels)
	dpr := screenPixelRatio(src.SystemWindow)
	mouse := src.SystemWindow.Position(nil).Add(scalePoint(e.WindowPos(), 1/dpr))
	grab := scalePoint(wd.grab(), 1/screenPixelRatio(win.SystemWindow))
	win.SystemWindow.SetPos(mouse.Sub(grab), nil)
}

// endWindowDrag ends the moving of a new window with the mouse, and raises
// the window, which gets the focus.
func (em *Events) endWindowDrag() {
	wd := em.dragWindow
	em.dragWindow = nil
	if win := wd.body.Scene.RenderWindow(); win != nil && win.SystemWindow != nil {
		win.SystemWindow.Raise()
	}
}

// screenPixelRatio returns the device pixel ratio of the screen of the given
// window: the number of its pixels in a unit of the window manager.
func screenPixelRatio(w system.Window) float32 {
	if sc := w.Screen(); sc != nil && sc.DevicePixelRatio > 0 {
		return sc.DevicePixelRatio
	}
	return 1
}

// scalePoint returns the given point scaled by the given factor.
func scalePoint(p image.Point, f float32) image.Point {
	return image.Pt(int(math32.Round(float32(p.X)*f)), int(math32.Round(float32(p.Y)*f)))
}

// grabPoint is a position that is set in one goroutine (as when a window is
// shown) and read in another.
type grabPoint struct {
	sync.Mutex
	p image.Point
}

func (g *grabPoint) get() image.Point {
	g.Lock()
	defer g.Unlock()
	return g.p
}

func (g *grabPoint) set(p image.Point) {
	g.Lock()
	g.p = p
	g.Unlock()
}
