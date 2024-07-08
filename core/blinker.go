// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"sync"
	"time"
)

// Blinker manages the logistics of blinking things, such as cursors
type Blinker struct {

	// Ticker is the [time.Ticker] used to control the blinking.
	Ticker *time.Ticker

	// Widget is the current widget subject to blinking
	Widget Widget

	// Func is the function called every tick under Mu mutex protection
	Func func()

	quit chan struct{}
	mu   sync.Mutex
}

// Blink sets up the blinking; does nothing if already set up.
func (bl *Blinker) Blink(dur time.Duration) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	if bl.Ticker != nil {
		return
	}
	bl.Ticker = time.NewTicker(dur)
	bl.quit = make(chan struct{})
	go bl.blinkLoop()
}

// SetWidget sets the [Blinker.Widget] under mutex lock.
func (bl *Blinker) SetWidget(w Widget) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.Widget = w
}

// ResetWidget sets [Blinker.Widget] to nil if it is currently set to the given one.
func (bl *Blinker) ResetWidget(w Widget) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	if bl.Widget == w {
		bl.Widget = nil
	}
}

// blinkLoop is the blinker's main control loop.
func (bl *Blinker) blinkLoop() {
	for {
		bl.mu.Lock()
		if bl.Ticker == nil {
			bl.mu.Unlock()
			return // shutdown..
		}
		bl.mu.Unlock()
		select {
		case <-bl.Ticker.C:
		case <-bl.quit:
			return
		}
		bl.mu.Lock()
		if bl.Widget == nil {
			bl.mu.Unlock()
			continue
		}
		wb := bl.Widget.AsWidget()
		if wb.Scene == nil || wb.Scene.Stage.Main == nil {
			bl.Widget = nil
			bl.mu.Unlock()
			continue
		}
		bl.Func()
		bl.mu.Unlock()
	}

}

// QuitClean is a cleanup function to pass to [TheApp.AddQuitCleanFunc]
// that breaks out of the ticker loop.
func (bl *Blinker) QuitClean() {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	if bl.Ticker != nil {
		bl.Widget = nil
		bl.Ticker.Stop()
		bl.Ticker = nil
		bl.quit <- struct{}{}
	}
}
