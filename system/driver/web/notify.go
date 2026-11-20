// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"errors"
	"sync"
	"syscall/js"

	"cogentcore.org/core/system"
)

var (
	// theNotifier is the notifier of the app, made by [App.Notifier].
	theNotifier *system.NotifierTimers

	notifierOnce sync.Once
)

// Notifier returns the notifier of the browser, with its Notification
// API, which only shows notifications while the page is open, as it cannot
// schedule them.
func (a *App) Notifier() system.Notifier {
	notifierOnce.Do(func() {
		if js.Global().Get("Notification").IsUndefined() {
			theNotifier = &system.NotifierTimers{ShowFunc: func(n *system.Notification) error {
				return errors.ErrUnsupported
			}}
			return
		}
		theNotifier = &system.NotifierTimers{ShowFunc: showNotification, PermissionFunc: requestNotifyPermission}
	})
	return theNotifier
}

// requestNotifyPermission asks the user for permission to show
// notifications, which must be in the handler of a click.
func requestNotifyPermission(fun func(allowed bool)) {
	nf := js.Global().Get("Notification")
	if p := nf.Get("permission").String(); p != "default" { // answered
		go fun(p == "granted")
		return
	}
	var then js.Func
	then = js.FuncOf(func(this js.Value, args []js.Value) any {
		then.Release()
		go fun(len(args) > 0 && args[0].String() == "granted")
		return nil
	})
	nf.Call("requestPermission").Call("then", then)
}

// errNotAllowed is the error for showing a notification without
// permission.
var errNotAllowed = errors.New("web: notifications are not allowed")

// showNotification shows the given notification now, which focuses the
// page when it is clicked.
func showNotification(n *system.Notification) error {
	nf := js.Global().Get("Notification")
	if nf.Get("permission").String() != "granted" {
		return errNotAllowed
	}
	opts := map[string]any{"body": n.Body, "tag": n.ID}
	jn := nf.New(n.Title, opts)
	id := n.ID
	var click js.Func
	click = js.FuncOf(func(this js.Value, args []js.Value) any {
		click.Release()
		js.Global().Call("focus")
		jn.Call("close")
		go theNotifier.Clicked(id)
		return nil
	})
	jn.Set("onclick", click)
	return nil
}
