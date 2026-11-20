// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ios

package ios

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework UserNotifications
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>
void notifyRequestPermission(uintptr_t h);
void notifyShow(char* nid, char* title, char* body);
void notifyClearScheduled(void);
void notifySchedule(char* nid, char* title, char* body, double unix);
*/
import "C"

import (
	"runtime/cgo"
	"time"
	"unsafe"

	"cogentcore.org/core/system"
)

// Notifier is the [system.Notifier] of iOS, which uses the
// UserNotifications framework, whose delegate is the app delegate.
type Notifier struct {
	system.NotifierClicks
}

// TheNotifier is the single [Notifier] of the app.
var TheNotifier = &Notifier{}

func (a *App) Notifier() system.Notifier {
	return TheNotifier
}

func (nt *Notifier) RequestPermission(fun func(allowed bool)) {
	C.notifyRequestPermission(C.uintptr_t(cgo.NewHandle(fun)))
}

//export iosNotifyPermission
func iosNotifyPermission(h C.uintptr_t, allowed C.bool) {
	ch := cgo.Handle(h)
	fun := ch.Value().(func(allowed bool))
	ch.Delete()
	fun(bool(allowed))
}

//export iosNotifyClicked
func iosNotifyClicked(id *C.char) {
	go TheNotifier.Clicked(C.GoString(id)) // not on the main thread
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

func (nt *Notifier) Show(n *system.Notification) error {
	id, title, body, free := cNotification(n)
	defer free()
	C.notifyShow(id, title, body)
	return nil
}

func (nt *Notifier) Sync(ns []*system.Notification) error {
	C.notifyClearScheduled()
	for _, n := range system.ScheduleOrder(ns, time.Now()) {
		id, title, body, free := cNotification(n)
		C.notifySchedule(id, title, body, C.double(float64(n.Time.UnixMilli())/1000))
		free()
	}
	return nil
}
