// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package desktop

import (
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/system"
	"cogentcore.org/core/system/driver/desktop/wintoast"
)

// winNotifier is the [system.Notifier] of Windows, which shows and
// schedules toast notifications (see [wintoast]). Clicks on the toasts
// are not reported yet, which needs a COM activator registered for the app.
type winNotifier struct {
	system.NotifierClicks

	toasts *wintoast.Notifier
}

var (
	// theNotifier is the notifier of the app, made by [App.Notifier].
	theNotifier system.Notifier

	notifierOnce sync.Once
)

// Notifier returns the notifier of Windows, which registers the
// AppUserModelID of the app for the toasts (see [appUserModelID]).
func (a *App) Notifier() system.Notifier {
	notifierOnce.Do(func() {
		tn, err := wintoast.New(appUserModelID(), a.Name())
		if err != nil {
			theNotifier = &system.NotifierBase{}
			return
		}
		theNotifier = &winNotifier{toasts: tn}
	})
	return theNotifier
}

// appUserModelID returns the AppUserModelID of the app for its toasts:
// [system.AppID] if it is set, as by core build, and otherwise one made
// from the name of the app.
func appUserModelID() string {
	if system.AppID != "" {
		return system.AppID
	}
	name := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\\' {
			return -1
		}
		return r
	}, system.TheApp.Name())
	return "CogentCore." + name
}

func (wn *winNotifier) RequestPermission(fun func(allowed bool)) {
	go fun(wn.toasts.Enabled())
}

// toast returns the toast of the given notification.
func toast(n *system.Notification) *wintoast.Toast {
	return &wintoast.Toast{ID: n.ID, Title: n.Title, Body: n.Body, Time: n.Time}
}

func (wn *winNotifier) Show(n *system.Notification) error {
	return wn.toasts.Show(toast(n))
}

func (wn *winNotifier) Sync(ns []*system.Notification) error {
	var ts []*wintoast.Toast
	for _, n := range system.ScheduleOrder(ns, time.Now()) {
		ts = append(ts, toast(n))
	}
	return wn.toasts.Sync(ts)
}
