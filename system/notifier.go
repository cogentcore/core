// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"errors"
	"slices"
	"sync"
	"time"
)

// Notification is a notification shown by the operating system, as for
// a reminder of something that is due.
type Notification struct {

	// ID identifies the notification, which is passed to the function of
	// [Notifier.OnClick] when it is clicked. The app's notifications
	// should have different IDs.
	ID string

	// Title is the title of the notification, in bold.
	Title string

	// Body is the text of the notification, below the title.
	Body string

	// Time is when the notification is shown, for [Notifier.Sync].
	Time time.Time
}

// MaxScheduled is the maximum number of notifications that
// [Notifier.Sync] schedules, the soonest ones, as iOS only keeps
// that many for an app.
const MaxScheduled = 64

// Notifier shows notifications with the notification system of the
// operating system (see [App.Notifier]). The methods return
// [errors.ErrUnsupported] where notifications are not supported,
// as on macOS for an app that is not in an app bundle.
type Notifier interface {

	// RequestPermission asks the user for permission to show
	// notifications, if they have not already answered, and then calls
	// the given function with whether they are allowed. It returns
	// without waiting for the answer, and the function is called on
	// another goroutine. On the web, it must be called in the event
	// handler of a click, as for a button that turns notifications on.
	RequestPermission(fun func(allowed bool))

	// Show shows the given notification now (its time is not used).
	Show(n *Notification) error

	// Sync schedules the given notifications to be shown at their times,
	// replacing all of the notifications scheduled before, so it is
	// called again with all of them when any of them changes, and with
	// none to cancel them. Those whose time is past are not shown, and
	// only the [MaxScheduled] soonest are scheduled. On macOS, iOS,
	// Windows, and Android, the operating system shows them even when the
	// app is not running; elsewhere, they are only shown while it runs.
	Sync(ns []*Notification) error

	// OnClick sets the function that is called with the ID of a
	// notification when the user clicks it, which brings the app to the
	// front. A click that starts the app (on iOS and Android) is passed
	// to the function once it is set. Clicks are not reported on Windows
	// yet. It is called on another goroutine, so it must lock the app as
	// needed.
	OnClick(fun func(id string))
}

// ScheduleOrder returns the notifications of the given list whose time is
// after now, sorted by time, and at most [MaxScheduled] of them, which are
// the notifications that [Notifier.Sync] schedules.
func ScheduleOrder(ns []*Notification, now time.Time) []*Notification {
	var sched []*Notification
	for _, n := range ns {
		if n != nil && n.Time.After(now) {
			sched = append(sched, n)
		}
	}
	slices.SortStableFunc(sched, func(a, b *Notification) int {
		return a.Time.Compare(b.Time)
	})
	if len(sched) > MaxScheduled {
		sched = sched[:MaxScheduled]
	}
	return sched
}

// NotifierClicks has the click function of a [Notifier], for implementing
// [Notifier.OnClick].
type NotifierClicks struct {
	mu    sync.Mutex
	click func(id string)

	// pending is the ID of a notification clicked before the click
	// function was set, as when the click started the app.
	pending string
}

func (nc *NotifierClicks) OnClick(fun func(id string)) {
	nc.mu.Lock()
	nc.click = fun
	pending := nc.pending
	nc.pending = ""
	nc.mu.Unlock()
	if fun != nil && pending != "" {
		go fun(pending)
	}
}

// Clicked calls the click function for the notification with the given
// ID, which the user clicked. If it is not set yet, as when the click
// started the app, it is called with the ID once it is set.
func (nc *NotifierClicks) Clicked(id string) {
	nc.mu.Lock()
	fun := nc.click
	if fun == nil {
		nc.pending = id
	}
	nc.mu.Unlock()
	if fun != nil {
		fun(id)
	}
}

// NotifierBase is a [Notifier] for where notifications are not supported,
// whose methods return [errors.ErrUnsupported].
type NotifierBase struct{}

var _ Notifier = &NotifierBase{}

func (nb *NotifierBase) RequestPermission(fun func(allowed bool)) { go fun(false) }
func (nb *NotifierBase) Show(n *Notification) error               { return errors.ErrUnsupported }
func (nb *NotifierBase) Sync(ns []*Notification) error            { return errors.ErrUnsupported }
func (nb *NotifierBase) OnClick(fun func(id string))              {}

// NotifierTimers is a [Notifier] that shows the notifications of
// [Notifier.Sync] with timers, for where the operating system cannot
// schedule them, so they are only shown while the app is running.
type NotifierTimers struct {
	NotifierClicks

	// ShowFunc shows the given notification now.
	ShowFunc func(n *Notification) error

	// PermissionFunc, if set, asks for permission
	// (see [Notifier.RequestPermission]); otherwise it is allowed.
	PermissionFunc func(fun func(allowed bool))

	mu     sync.Mutex
	timers []*time.Timer
}

var _ Notifier = &NotifierTimers{}

func (nt *NotifierTimers) RequestPermission(fun func(allowed bool)) {
	if nt.PermissionFunc == nil {
		go fun(true)
		return
	}
	nt.PermissionFunc(fun)
}

func (nt *NotifierTimers) Show(n *Notification) error {
	return nt.ShowFunc(n)
}

func (nt *NotifierTimers) Sync(ns []*Notification) error {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	for _, t := range nt.timers {
		t.Stop()
	}
	nt.timers = nil
	now := time.Now()
	for _, n := range ScheduleOrder(ns, now) {
		nc := *n // a copy, as the caller can change it
		nt.timers = append(nt.timers, time.AfterFunc(n.Time.Sub(now), func() {
			nt.ShowFunc(&nc)
		}))
	}
	return nil
}
