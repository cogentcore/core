// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "time"

// Animation represents the data for a widget animation.
// You can call [WidgetBase.Animate] to create a widget animation.
// Animations are stored on the [Scene].
type Animation struct {

	// Func is the animation function, which is run every time the [Scene]
	// receives a paint tick, which is usually at the same rate as the refresh
	// rate of the monitor. It receives the [Animation] object so that
	// it can references things such as [Animation.Delta] and set things such as
	// [Animation.Done].
	Func func(a *Animation)

	// Widget is the widget associated with the animation. The animation will
	// pause if the widget is not visible, and it will end if the widget is destroyed.
	Widget Widget

	// Delta is the amount of time that has passed since the
	// last animation frame/step.
	Delta time.Duration

	// Done can be set to true to permanently stop the animation; the [Animation] object
	// will be removed from the [Scene] at the next frame.
	Done bool
}

// Animate adds a new [Animation] to the [Scene] for the widget. The given function is run
// at every tick, and it receives the [Animation] object so that it can reference and modify
// things on it; see the [Animation] docs for more information on things such as [Animation.Delta]
// and [Animation.Done].
func (wb *WidgetBase) Animate(f func(a *Animation)) {
	a := &Animation{
		Func:   f,
		Widget: wb.This.(Widget),
	}
	wb.Scene.Animations = append(wb.Scene.Animations, a)
}
