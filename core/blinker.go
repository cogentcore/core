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
	// Ticker is the time.Ticker
	Ticker *time.Ticker

	// Widget is the current widget subject to blinking
	Widget Widget

	// Func is the function called every tick -- under Mu mutex protection
	Func func(w Widget)

	Quit chan struct{}

	Mu sync.Mutex
}

// Blink sets up the blinking -- does nothing if already setup
func (bl *Blinker) Blink(dur time.Duration, fun func(w Widget)) {
	bl.Mu.Lock()
	defer bl.Mu.Unlock()
	if bl.Ticker != nil {
		return
	}
	bl.Func = fun
	bl.Ticker = time.NewTicker(dur)
	bl.Quit = make(chan struct{})
	go bl.BlinkLoop()
}

// SetWidget sets Widget to given one, under mutex lock
func (bl *Blinker) SetWidget(w Widget) {
	bl.Mu.Lock()
	defer bl.Mu.Unlock()
	bl.Widget = w
}

// ResetWidget sets Widget = nil if it is currently set to given one
func (bl *Blinker) ResetWidget(w Widget) {
	bl.Mu.Lock()
	defer bl.Mu.Unlock()
	if bl.Widget == w {
		bl.Widget = nil
	}
}

// BlinkLoop is the blinker's main control loop
func (bl *Blinker) BlinkLoop() {
	for {
		bl.Mu.Lock()
		if bl.Ticker == nil {
			bl.Mu.Unlock()
			return // shutdown..
		}
		bl.Mu.Unlock()
		select {
		case <-bl.Ticker.C:
		case <-bl.Quit:
			return
		}
		bl.Mu.Lock()
		if bl.Widget == nil || bl.Widget.This() == nil {
			bl.Mu.Unlock()
			continue
		}
		wb := bl.Widget.AsWidget()
		if wb.Scene == nil || wb.Scene.Stage.Main == nil {
			bl.Widget = nil
			bl.Mu.Unlock()
			continue
		}
		bl.Func(bl.Widget)
		bl.Mu.Unlock()
	}

}

// QuitClean is a cleanup function to call during Quit that
// breaks out of the ticker loop
func (bl *Blinker) QuitClean() {
	bl.Mu.Lock()
	defer bl.Mu.Unlock()
	if bl.Ticker != nil {
		bl.Widget = nil
		bl.Ticker.Stop()
		bl.Ticker = nil
		bl.Quit <- struct{}{}
	}
}
