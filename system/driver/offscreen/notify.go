// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"slices"
	"sync"

	"cogentcore.org/core/system"
)

// Notifier is the [system.Notifier] of the offscreen app, which records
// the notifications that are shown, so that they can be tested.
// The notifications of [system.Notifier.Sync] are shown with timers.
type Notifier struct {
	system.NotifierTimers

	mu    sync.Mutex
	shown []system.Notification
}

// TheNotifier is the single [Notifier] of the offscreen app.
var TheNotifier = newNotifier()

func newNotifier() *Notifier {
	nt := &Notifier{}
	nt.ShowFunc = func(n *system.Notification) error {
		nt.mu.Lock()
		nt.shown = append(nt.shown, *n)
		nt.mu.Unlock()
		return nil
	}
	return nt
}

func (a *App) Notifier() system.Notifier {
	return TheNotifier
}

// Shown returns the notifications that have been shown, in order.
func (nt *Notifier) Shown() []system.Notification {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	return slices.Clone(nt.shown)
}

// ClearShown clears the list of the notifications that have been shown.
func (nt *Notifier) ClearShown() {
	nt.mu.Lock()
	nt.shown = nil
	nt.mu.Unlock()
}
