// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

package android

/*
#include <jni.h>
#include <stdbool.h>
void notifyCall(JNIEnv *env, int show, int n, const jchar *buf, const int *lens, const long long *times);
void notifyRequestPermission(JNIEnv *env);
*/
import "C"

import (
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"

	"cogentcore.org/core/system"
	"cogentcore.org/core/system/driver/mobile/mobileinit"
)

// Notifier is the [system.Notifier] of Android, which shows notifications
// with the NotificationManager and schedules them with the AlarmManager
// (see Notify.java). Scheduled notifications are lost when the device
// restarts, until the app syncs them again.
type Notifier struct {
	system.NotifierClicks

	mu sync.Mutex

	// permission are the functions waiting for the answer of the
	// permission request.
	permission []func(allowed bool)
}

// TheNotifier is the single [Notifier] of the app.
var TheNotifier = &Notifier{}

func (a *App) Notifier() system.Notifier {
	return TheNotifier
}

func (nt *Notifier) RequestPermission(fun func(allowed bool)) {
	nt.mu.Lock()
	nt.permission = append(nt.permission, fun)
	nt.mu.Unlock()
	err := mobileinit.RunOnJVM(func(vm, jniEnv, ctx uintptr) error {
		C.notifyRequestPermission((*C.JNIEnv)(unsafe.Pointer(jniEnv)))
		return nil
	})
	if err != nil {
		androidNotifyPermission(false)
	}
}

//export androidNotifyPermission
func androidNotifyPermission(allowed C.bool) {
	nt := TheNotifier
	nt.mu.Lock()
	funs := nt.permission
	nt.permission = nil
	nt.mu.Unlock()
	for _, fun := range funs {
		go fun(bool(allowed))
	}
}

//export androidNotifyClicked
func androidNotifyClicked(id *C.char) {
	go TheNotifier.Clicked(C.GoString(id)) // not on the UI thread
}

// notifyCall shows the given notification now if show is set, and
// otherwise schedules them (see notifyCall in android.c), passing their
// strings in UTF-16, as JNI's modified UTF-8 has no 4-byte characters.
func notifyCall(show bool, ns []*system.Notification) error {
	var buf []uint16
	var lens []C.int
	times := make([]C.longlong, len(ns))
	for i, n := range ns {
		for _, s := range []string{n.ID, n.Title, n.Body} {
			u := utf16.Encode([]rune(s))
			buf = append(buf, u...)
			lens = append(lens, C.int(len(u)))
		}
		times[i] = C.longlong(n.Time.UnixMilli())
	}
	buf = append(buf, 0) // not empty
	lens = append(lens, 0)
	times = append(times, 0)
	return mobileinit.RunOnJVM(func(vm, jniEnv, ctx uintptr) error {
		env := (*C.JNIEnv)(unsafe.Pointer(jniEnv))
		C.notifyCall(env, C.int(boolInt(show)), C.int(len(ns)), (*C.jchar)(unsafe.Pointer(&buf[0])), &lens[0], &times[0])
		return nil
	})
}

// boolInt returns 1 for true, and 0 for false.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (nt *Notifier) Show(n *system.Notification) error {
	return notifyCall(true, []*system.Notification{n})
}

func (nt *Notifier) Sync(ns []*system.Notification) error {
	return notifyCall(false, system.ScheduleOrder(ns, time.Now()))
}
