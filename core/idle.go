// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"time"

	"cogentcore.org/core/enums"
)

// idle.go has the pausing of the paint events of a window while there is
// nothing to update in it (see [system.Window.PaintIdle]), so that an
// idle app does not use the CPU, as on mobile in the background (#1659).
// The window is woken (see [renderWindow.wake]) by everything that needs
// an update: the flags of its scenes that drive updates, changes to its
// stages and sprites, animations, and all events.

// wake resumes the paint events of the window, if they are paused, as
// something in it needs to be updated. It can be called from any
// goroutine, and is cheap when the window is already awake.
func (w *renderWindow) wake() {
	if w == nil || w.paintWanted.Swap(true) {
		return
	}
	if sw := w.SystemWindow; sw != nil {
		sw.PaintWake()
	}
}

// goIdle pauses the paint events of the window, as there is nothing to
// update in it, until it is woken; it is woken right away if something
// needed an update since this render pass started.
func (w *renderWindow) goIdle() {
	sw := w.SystemWindow
	if sw == nil {
		return
	}
	sw.PaintIdle()
	if w.paintWanted.Load() {
		sw.PaintWake()
	}
}

// idleUntilBlink pauses the paint events of the window until the next
// blink of its active sprites, such as the text cursor, if it has focus.
func (w *renderWindow) idleUntilBlink(top *Stage, blink time.Duration) {
	if w.flags.HasFlag(winGotFocus) && top.Sprites.hasActive() {
		d := max(blink-time.Since(w.lastSpriteDraw), time.Millisecond)
		if w.blinkTimer == nil {
			w.blinkTimer = time.AfterFunc(d, w.wake)
		} else {
			w.blinkTimer.Reset(d)
		}
	}
	w.goIdle()
}

// hasAnimations returns whether any scene of the main stages, or of
// their popups, has animations, which run at every paint event.
func (sm *stages) hasAnimations() bool {
	for _, st := range sm.stack.Values() {
		if st.Scene != nil && len(st.Scene.Animations) > 0 {
			return true
		}
		if st.Type.isMain() && st.popups != nil && st.popups != sm && st.popups.hasAnimations() {
			return true
		}
	}
	return false
}

// wake wakes the window of the stages (see [renderWindow.wake]).
func (sm *stages) wake() {
	if sm.renderWindow != nil {
		sm.renderWindow.wake()
		return
	}
	if sm.main != nil {
		sm.main.wake()
	}
}

// setModified records that the stages have been modified, waking their
// window to update them.
func (sm *stages) setModified() {
	sm.modified = true
	sm.wake()
}

// wake wakes the window of the stage (see [renderWindow.wake]).
func (st *Stage) wake() {
	if st == nil {
		return
	}
	if sm := st.mains.Load(); sm != nil && sm.renderWindow != nil {
		sm.renderWindow.wake()
	}
}

// wake wakes the window of the scene (see [renderWindow.wake]).
func (sc *Scene) wake() {
	sc.RenderWindow().wake()
}

// wakeFlags returns whether any of the given flags drives an update of
// a scene, for which its window is woken when one is set.
func wakeFlags(f []enums.BitFlag) bool {
	for _, fl := range f {
		switch fl {
		case sceneNeedsRender, sceneNeedsLayout, sceneHasDeferred:
			return true
		}
	}
	return false
}

// setModifiedNoLock records that the sprites have been modified, waking
// the window of their stage to draw them.
func (ss *Sprites) setModifiedNoLock() {
	ss.modified = true
	ss.stage.wake()
}
