// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"sync"
	"time"
)

// Blinker manages the logistics of blinking things, such as cursors.
type Blinker struct {

	// Ticker is the [time.Ticker] used to control the blinking.
	Ticker *time.Ticker

	// Widget is the current widget subject to blinking.
	Widget Widget

	// Func is the function called every tick.
	// The mutex is locked at the start but must be unlocked
	// when transitioning to locking the render context mutex.
	Func func()

	// Use Lock and Unlock on blinker directly.
	sync.Mutex
}

// Blink sets up the blinking; does nothing if already set up.
func (bl *Blinker) Blink(dur time.Duration) {
	bl.Lock()
	defer bl.Unlock()
	if bl.Ticker != nil {
		return
	}
	bl.Ticker = time.NewTicker(dur)
	go bl.blinkLoop()
}

// SetWidget sets the [Blinker.Widget] under mutex lock.
func (bl *Blinker) SetWidget(w Widget) {
	bl.Lock()
	defer bl.Unlock()
	bl.Widget = w
}

// ResetWidget sets [Blinker.Widget] to nil if it is currently set to the given one.
func (bl *Blinker) ResetWidget(w Widget) {
	bl.Lock()
	defer bl.Unlock()
	if bl.Widget == w {
		bl.Widget = nil
	}
}

// blinkLoop is the blinker's main control loop.
func (bl *Blinker) blinkLoop() {
	for {
		bl.Lock()
		if bl.Ticker == nil {
			bl.Unlock()
			return // shutdown..
		}
		bl.Unlock()
		<-bl.Ticker.C
		bl.Lock()
		if bl.Widget == nil {
			bl.Unlock()
			continue
		}
		wb := bl.Widget.AsWidget()
		if wb.Scene == nil || wb.Scene.Stage.Main == nil {
			bl.Widget = nil
			bl.Unlock()
			continue
		}
		bl.Func() // we enter the function locked
	}

}

// QuitClean is a cleanup function to pass to [TheApp.AddQuitCleanFunc]
// that breaks out of the ticker loop.
func (bl *Blinker) QuitClean() {
	bl.Lock()
	defer bl.Unlock()
	if bl.Ticker != nil {
		tck := bl.Ticker
		bl.Ticker = nil
		bl.Widget = nil
		tck.Stop()
	}
}
