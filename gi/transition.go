// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"time"

	"goki.dev/grr"
	"goki.dev/laser"
)

// Transition transitions the given pointer value (typically a pointer to a style property
// that is a number, unit value, or color) to the given destination value (typically not a pointer)
// over the given duration, using the given timing function. The timing function takes the proportion
// of the time of the transition that has taken place (0-1) and returns the total proportion of the
// difference between the starting and ending value that should be applied at that point in time (0-1).
// For example, a standard linear timing function would just return the value it gets. The transition runs
// at the FPS of the window, and it tells the widget to render on every FPS tick. If the starting and ending
// value are the same, it does nothing. If not, it runs the transition in a separate goroutine.
func (wb *WidgetBase) Transition(value any, to any, duration time.Duration, timingFunc func(prop float32) float32) {
	vn := grr.Log(laser.ToFloat32(value))
	tn := grr.Log(laser.ToFloat32(to))
	diff := tn - vn
	if diff == 0 {
		return
	}
	rate := time.Second / 60
	propPer := float32(rate) / float32(duration)
	tick := time.NewTicker(rate)
	go func() {
		for i := 0; i < int(duration/rate); i++ {
			<-tick.C
			prop := float32(i) * propPer
			inc := timingFunc(prop)
			nv := vn + inc*diff
			grr.Log0(laser.SetRobust(value, nv))
			wb.SetNeedsRender()
		}
		tick.Stop()
	}()
}

// LinearTransition is a simple, linear, 1 to 1 timing function that can be passed to [WidgetBase.Transition]
func LinearTransition(prop float32) float32 {
	return prop
}
