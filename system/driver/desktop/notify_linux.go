// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && !android

package desktop

import (
	"sync"

	"cogentcore.org/core/system"
	"github.com/godbus/dbus/v5"
)

// The D-Bus names of the notification service of freedesktop.org, which
// the notification daemons of the desktops implement.
const (
	notifyService = "org.freedesktop.Notifications"
	notifyPath    = "/org/freedesktop/Notifications"
)

// linuxNotifier shows notifications with the notification service of
// freedesktop.org, over D-Bus. It cannot schedule them, so they are
// shown with timers while the app is running (see [system.NotifierTimers]).
type linuxNotifier struct {
	system.NotifierTimers

	mu   sync.Mutex
	conn *dbus.Conn

	// ids are the IDs of the notifications by the IDs of the service,
	// for the clicks.
	ids map[uint32]string
}

var (
	// theNotifier is the notifier of the app, made by [App.Notifier].
	theNotifier *linuxNotifier

	notifierOnce sync.Once
)

// Notifier returns the notifier of Linux, which only shows notifications
// while the app is running, as the notification service cannot schedule
// them.
func (a *App) Notifier() system.Notifier {
	notifierOnce.Do(func() {
		ln := &linuxNotifier{ids: map[uint32]string{}}
		ln.ShowFunc = ln.show
		theNotifier = ln
	})
	return theNotifier
}

// connect connects to the session bus, once, and listens for clicks.
// It must be called with the mutex locked.
func (ln *linuxNotifier) connect() error {
	if ln.conn != nil {
		return nil
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	err = conn.AddMatchSignal(dbus.WithMatchObjectPath(notifyPath), dbus.WithMatchInterface(notifyService))
	if err != nil {
		conn.Close()
		return err
	}
	signals := make(chan *dbus.Signal, 10)
	conn.Signal(signals)
	go ln.signals(signals)
	ln.conn = conn
	return nil
}

// signals handles the signals of the notification service: the click of
// a notification (its default action), and its closing.
func (ln *linuxNotifier) signals(signals chan *dbus.Signal) {
	for sig := range signals {
		if len(sig.Body) < 2 {
			continue
		}
		nid, ok := sig.Body[0].(uint32)
		if !ok {
			continue
		}
		ln.mu.Lock()
		id, has := ln.ids[nid]
		ln.mu.Unlock()
		if !has {
			continue
		}
		switch sig.Name {
		case notifyService + ".ActionInvoked":
			if action, _ := sig.Body[1].(string); action == "default" {
				ln.Clicked(id)
			}
		case notifyService + ".NotificationClosed":
			ln.mu.Lock()
			delete(ln.ids, nid)
			ln.mu.Unlock()
		}
	}
}

// show shows the given notification now.
func (ln *linuxNotifier) show(n *system.Notification) error {
	ln.mu.Lock()
	defer ln.mu.Unlock()
	if err := ln.connect(); err != nil {
		return err
	}
	// the default action is the click of the notification
	actions := []string{"default", "Open"}
	obj := ln.conn.Object(notifyService, notifyPath)
	var nid uint32
	err := obj.Call(notifyService+".Notify", 0, system.TheApp.Name(), uint32(0), "",
		n.Title, n.Body, actions, map[string]dbus.Variant{}, int32(-1)).Store(&nid)
	if err != nil {
		return err
	}
	ln.ids[nid] = n.ID
	return nil
}
