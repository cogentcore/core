// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/system"
)

// osDragger is implemented by a widget whose drags have data that is not
// [mimedata.Mimes], to give the data for the operating system to drag
// when a drag of it leaves its window (see [Events.dragToOS]), or nil.
type osDragger interface {
	osDragData() mimedata.Mimes
}

// osDropper is implemented by a widget whose drags, handed to the operating
// system (see [Events.dragToOS]), do something when they are dropped where
// nothing accepts them, as onto the desktop: a tab of [Tabs] with
// [Tabs.DragToWindow] then moves into a new window there.
type osDropper interface {
	// droppedNowhere is called with the event of the end of the drag, and
	// the position of the drop on the screen, in the units of the window
	// manager (see [system.Window.Position]).
	droppedNowhere(e events.Event, screen image.Point)
}

// dragOSDroppedNowhere calls [osDropper.droppedNowhere] for a drag that
// the operating system has dragged, if it was dropped where nothing
// accepted it, at the position of the given event of its end.
func (em *Events) dragOSDroppedNowhere(e events.Event) {
	od, ok := em.drag.(osDropper)
	rw := em.RenderWindow()
	if !ok || rw == nil || rw.SystemWindow == nil || rw.SystemWindow.DragAccepted() {
		return
	}
	od.droppedNowhere(e, screenPos(rw.SystemWindow, e.WindowPos()))
}

// screenPos returns the position on the screen, in the units of the window
// manager, of the given position in the given window, in its pixels.
func screenPos(w system.Window, pos image.Point) image.Point {
	return w.Position(nil).Add(scalePoint(pos, 1/screenPixelRatio(w)))
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
