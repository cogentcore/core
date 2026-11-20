// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Foundation -framework UserNotifications
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>
bool notifyAvailable(void);
void notifyInit(void);
void notifyRequestPermission(uintptr_t h);
void notifyShow(char* nid, char* title, char* body);
void notifyClearScheduled(void);
void notifySchedule(char* nid, char* title, char* body, double unix);
*/
import "C"

import (
	"runtime/cgo"
	"sync"
	"time"
	"unsafe"

	"cogentcore.org/core/system"
)

// macNotifier is the [system.Notifier] of macOS, which uses the
// UserNotifications framework.
type macNotifier struct {
	system.NotifierClicks
}

var (
	// theNotifier is the notifier of the app, made by [App.Notifier].
	theNotifier system.Notifier

	notifierOnce sync.Once
)

// Notifier returns the notifier of macOS, which is only supported for an
// app in an app bundle, as made by core build. The notification system also
// refuses an app that LaunchServices does not validate, as one in a
// temporary directory: it works in /Applications or ~/Applications.
func (a *App) Notifier() system.Notifier {
	notifierOnce.Do(func() {
		if !bool(C.notifyAvailable()) {
			theNotifier = &system.NotifierBase{}
			return
		}
		C.notifyInit()
		theNotifier = &macNotifier{}
	})
	return theNotifier
}

func (mn *macNotifier) RequestPermission(fun func(allowed bool)) {
	C.notifyRequestPermission(C.uintptr_t(cgo.NewHandle(fun)))
}

//export macNotifyPermission
func macNotifyPermission(h C.uintptr_t, allowed C.bool) {
	ch := cgo.Handle(h)
	fun := ch.Value().(func(allowed bool))
	ch.Delete()
	fun(bool(allowed))
}

//export macNotifyClicked
func macNotifyClicked(id *C.char) {
	if mn, ok := theNotifier.(*macNotifier); ok {
		go mn.Clicked(C.GoString(id)) // not on the main thread
	}
}

// cNotification returns the C strings of the given notification,
// which must be freed with the returned function.
func cNotification(n *system.Notification) (id, title, body *C.char, free func()) {
	id, title, body = C.CString(n.ID), C.CString(n.Title), C.CString(n.Body)
	return id, title, body, func() {
		C.free(unsafe.Pointer(id))
		C.free(unsafe.Pointer(title))
		C.free(unsafe.Pointer(body))
	}
}

func (mn *macNotifier) Show(n *system.Notification) error {
	id, title, body, free := cNotification(n)
	defer free()
	C.notifyShow(id, title, body)
	return nil
}

func (mn *macNotifier) Sync(ns []*system.Notification) error {
	C.notifyClearScheduled()
	for _, n := range system.ScheduleOrder(ns, time.Now()) {
		id, title, body, free := cNotification(n)
		C.notifySchedule(id, title, body, C.double(float64(n.Time.UnixMilli())/1000))
		free()
	}
	return nil
}
