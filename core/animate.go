// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"slices"
	"time"
)

// Animation represents the data for a widget animation.
// You can call [WidgetBase.Animate] to create a widget animation.
// Animations are stored on the [Scene].
type Animation struct {

	// Func is the animation function, which is run every time the [Scene]
	// receives a paint tick, which is usually at the same rate as the refresh
	// rate of the monitor. It receives the [Animation] object so that
	// it can references things such as [Animation.Dt] and set things such as
	// [Animation.Done].
	Func func(a *Animation)

	// Widget is the widget associated with the animation. The animation will
	// pause if the widget is not visible, and it will end if the widget is destroyed.
	Widget *WidgetBase

	// Dt is the amount of time in milliseconds that has passed since the
	// last animation frame/step/tick.
	Dt float32

	// Done can be set to true to permanently stop the animation; the [Animation] object
	// will be removed from the [Scene] at the next frame.
	Done bool

	// lastTime is the last time this animation was run.
	lastTime time.Time
}

// Animate adds a new [Animation] to the [Scene] for the widget. The given function is run
// at every tick, and it receives the [Animation] object so that it can reference and modify
// things on it; see the [Animation] docs for more information on things such as [Animation.Dt]
// and [Animation.Done].
func (wb *WidgetBase) Animate(f func(a *Animation)) {
	a := &Animation{
		Func:   f,
		Widget: wb,
	}
	wb.Scene.Animations = append(wb.Scene.Animations, a)
}

// runAnimations runs the [Scene.Animations].
func (sc *Scene) runAnimations() {
	if len(sc.Animations) == 0 {
		return
	}

	for _, a := range sc.Animations {
		if a.Widget == nil || a.Widget.This == nil {
			a.Done = true
		}
		if a.Done || !a.Widget.IsVisible() {
			continue
		}
		if a.lastTime.IsZero() {
			a.Dt = 16.66666667 // 60 FPS fallback
		} else {
			a.Dt = float32(time.Since(a.lastTime).Seconds()) * 1000
		}
		a.Func(a)
		a.lastTime = time.Now()
	}

	sc.Animations = slices.DeleteFunc(sc.Animations, func(a *Animation) bool {
		return a.Done
	})
}
