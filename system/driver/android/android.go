// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/golang/mobile
// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

/*
Package android implements system interfaces on Android mobile devices.

Android Apps are built with -buildmode=c-shared. They are loaded by a
running Java process.

Before any entry point is reached, a global constructor initializes the
Go runtime, calling all Go init functions. All cgo calls will block
until this is complete. Next JNI_OnLoad is called. When that is
complete, one of two entry points is called.

All-Go apps built using NativeActivity enter at ANativeActivity_onCreate.
*/
package android

/*
#cgo LDFLAGS: -landroid -llog

#include <android/configuration.h>
#include <android/input.h>
#include <android/keycodes.h>
#include <android/looper.h>
#include <android/native_activity.h>
#include <android/native_window.h>
#include <jni.h>
#include <pthread.h>
#include <stdlib.h>
#include <stdbool.h>

void showKeyboard(JNIEnv* env, int keyboardType);
void hideKeyboard(JNIEnv* env);
*/
import "C"
import (
	"image"
	"log"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"time"
	"unsafe"

	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/driver/base"
	"cogentcore.org/core/system/driver/mobile/callfn"
	"cogentcore.org/core/system/driver/mobile/mobileinit"
)

// mimeMap contains standard mime entries that are missing on Android
var mimeMap = map[string]string{
	".txt": "text/plain",
}

// RunOnJVM runs fn on a new goroutine locked to an OS thread with a JNIEnv.
//
// RunOnJVM blocks until the call to fn is complete. Any Java
// exception or failure to attach to the JVM is returned as an error.
//
// The function fn takes vm, the current JavaVM*,
// env, the current JNIEnv*, and
// ctx, a jobject representing the global android.context.Context.
func RunOnJVM(fn func(vm, jniEnv, ctx uintptr) error) error {
	return mobileinit.RunOnJVM(fn)
}

//export setCurrentContext
func setCurrentContext(vm *C.JavaVM, ctx C.jobject) {
	mobileinit.SetCurrentContext(unsafe.Pointer(vm), uintptr(ctx))
}

//export callMain
func callMain(mainPC uintptr) {
	for _, name := range []string{"FILESDIR", "TMPDIR", "PATH", "LD_LIBRARY_PATH"} {
		n := C.CString(name)
		os.Setenv(name, C.GoString(C.getenv(n)))
		C.free(unsafe.Pointer(n))
	}

	// Set timezone.
	//
	// Note that Android zoneinfo is stored in /system/usr/share/zoneinfo,
	// but it is in some kind of packed TZiff file that we do not support
	// yet. As a stopgap, we build a fixed zone using the tm_zone name.
	var curtime C.time_t
	var curtm C.struct_tm
	C.time(&curtime)
	C.localtime_r(&curtime, &curtm)
	tzOffset := int(curtm.tm_gmtoff)
	tz := C.GoString(curtm.tm_zone)
	time.Local = time.FixedZone(tz, tzOffset)

	go callfn.CallFn(mainPC)
}

//export onSaveInstanceState
func onSaveInstanceState(activity *C.ANativeActivity, outSize *C.size_t) unsafe.Pointer {
	return nil
}

//export onCreate
func onCreate(activity *C.ANativeActivity) {
	// Set the initial configuration.
	//
	// Note we use unbuffered channels to talk to the activity loop, and
	// NativeActivity calls these callbacks sequentially, so configuration
	// will be set before <-windowRedrawNeeded is processed.
	windowConfigChange <- windowConfigRead(activity)
}

//export onDestroy
func onDestroy(activity *C.ANativeActivity) {
	activityDestroyed <- struct{}{}
}

//export onWindowFocusChanged
func onWindowFocusChanged(activity *C.ANativeActivity, hasFocus C.int) {
	TheApp.Mu.Lock()
	defer TheApp.Mu.Unlock()
	if hasFocus > 0 {
		TheApp.Event.Window(events.WinFocus)
	} else {
		TheApp.Event.Window(events.WinFocusLost)
	}
}

//export onNativeWindowCreated
func onNativeWindowCreated(activity *C.ANativeActivity, window *C.ANativeWindow) {
	TheApp.Mu.Lock()
	defer TheApp.Mu.Unlock()
	TheApp.SetSystemWindow(uintptr(unsafe.Pointer(window)))
}

//export onNativeWindowRedrawNeeded
func onNativeWindowRedrawNeeded(activity *C.ANativeActivity, window *C.ANativeWindow) {
	// Called on orientation change and window resize.
	// Send a request for redraw, and block this function
	// until a complete draw and buffer swap is completed.
	// This is required by the redraw documentation to
	// avoid bad draws.
	windowRedrawNeeded <- window
	// TODO(kai): do we need windowRedrawDone here?
	// It is unclear how to implement it with the way system is structured.
	// <-windowRedrawDone
}

//export onNativeWindowDestroyed
func onNativeWindowDestroyed(activity *C.ANativeActivity, window *C.ANativeWindow) {
	windowDestroyed <- window
}

//export onInputQueueCreated
func onInputQueueCreated(activity *C.ANativeActivity, q *C.AInputQueue) {
	inputQueue <- q
	<-inputQueueDone
}

//export onInputQueueDestroyed
func onInputQueueDestroyed(activity *C.ANativeActivity, q *C.AInputQueue) {
	inputQueue <- nil
	<-inputQueueDone
}

//export onContentRectChanged
func onContentRectChanged(activity *C.ANativeActivity, rect *C.ARect) {
}

//export setDarkMode
func setDarkMode(dark C.bool) {
	TheApp.Dark = bool(dark)
}

// windowConfig contains the window configuration information fetched from the native activity
type windowConfig struct {
	Orientation system.ScreenOrientation
	DPI         float32 // raw display dots per inch
}

func windowConfigRead(activity *C.ANativeActivity) windowConfig {
	defer func() { system.HandleRecover(recover()) }()

	aconfig := C.AConfiguration_new()
	C.AConfiguration_fromAssetManager(aconfig, activity.assetManager)
	orient := C.AConfiguration_getOrientation(aconfig)
	density := C.AConfiguration_getDensity(aconfig)
	C.AConfiguration_delete(aconfig)

	// Calculate the screen resolution. This value is approximate. For example,
	// a physical resolution of 200 DPI may be quantized to one of the
	// ACONFIGURATION_DENSITY_XXX values such as 160 or 240.
	//
	// A more accurate DPI could possibly be calculated from
	// https://developer.android.com/reference/android/util/DisplayMetrics.html#xdpi
	// but this does not appear to be accessible via the NDK. In any case, the
	// hardware might not even provide a more accurate number, as the system
	// does not apparently use the reported value. See golang.org/issue/13366
	// for a discussion.
	var dpi int
	switch density {
	case C.ACONFIGURATION_DENSITY_DEFAULT:
		dpi = 160
	case C.ACONFIGURATION_DENSITY_LOW,
		C.ACONFIGURATION_DENSITY_MEDIUM,
		213, // C.ACONFIGURATION_DENSITY_TV
		C.ACONFIGURATION_DENSITY_HIGH,
		320, // ACONFIGURATION_DENSITY_XHIGH
		480, // ACONFIGURATION_DENSITY_XXHIGH
		640: // ACONFIGURATION_DENSITY_XXXHIGH
		dpi = int(density)
	case C.ACONFIGURATION_DENSITY_NONE:
		slog.Warn("android device reports no screen density")
		dpi = 72
	default:
		// TODO: fix this always happening with value 240
		slog.Warn("android device reports unknown screen density", "density", density)
		// All we can do is guess.
		if density > 0 {
			dpi = int(density)
		} else {
			dpi = 72
		}
	}

	o := system.OrientationUnknown
	switch orient {
	case C.ACONFIGURATION_ORIENTATION_PORT:
		o = system.Portrait
	case C.ACONFIGURATION_ORIENTATION_LAND:
		o = system.Landscape
	}

	return windowConfig{
		Orientation: o,
		DPI:         float32(dpi),
	}
}

//export onConfigurationChanged
func onConfigurationChanged(activity *C.ANativeActivity) {
	// A rotation event first triggers onConfigurationChanged, then
	// calls onNativeWindowRedrawNeeded. We extract the orientation
	// here and save it for the redraw event.
	windowConfigChange <- windowConfigRead(activity)
}

//export onLowMemory
func onLowMemory(activity *C.ANativeActivity) {
	runtime.GC()
	debug.FreeOSMemory()
}

var (
	inputQueue         = make(chan *C.AInputQueue)
	inputQueueDone     = make(chan struct{})
	windowDestroyed    = make(chan *C.ANativeWindow)
	windowRedrawNeeded = make(chan *C.ANativeWindow)
	windowRedrawDone   = make(chan struct{})
	windowConfigChange = make(chan windowConfig)
	activityDestroyed  = make(chan struct{})
)

func (a *App) MainLoop() {
	a.MainQueue = make(chan base.FuncRun)
	a.MainDone = make(chan struct{})
	// TODO: maybe merge the RunInputQueue and MainUI functions?
	go func() {
		defer func() { system.HandleRecover(recover()) }()
		if err := mobileinit.RunOnJVM(RunInputQueue); err != nil {
			log.Fatalf("app: %v", err)
		}
	}()
	// Preserve this OS thread for the attached JNI thread
	if err := mobileinit.RunOnJVM(TheApp.MainUI); err != nil {
		log.Fatalf("app: %v", err)
	}
}

// ShowVirtualKeyboard requests the driver to show a virtual keyboard for text input
func (a *App) ShowVirtualKeyboard(typ styles.VirtualKeyboards) {
	err := mobileinit.RunOnJVM(func(vm, jniEnv, ctx uintptr) error {
		env := (*C.JNIEnv)(unsafe.Pointer(jniEnv)) // not a Go heap pointer
		C.showKeyboard(env, C.int(int32(typ)))
		return nil
	})
	if err != nil {
		log.Fatalf("app: %v", err)
	}
}

// HideVirtualKeyboard requests the driver to hide any visible virtual keyboard
func (a *App) HideVirtualKeyboard() {
	if err := mobileinit.RunOnJVM(hideSoftInput); err != nil {
		log.Fatalf("app: %v", err)
	}
}

func hideSoftInput(vm, jniEnv, ctx uintptr) error {
	env := (*C.JNIEnv)(unsafe.Pointer(jniEnv)) // not a Go heap pointer
	C.hideKeyboard(env)
	return nil
}

//export insetsChanged
func insetsChanged(top, bottom, left, right int) {
	TheApp.Insets.Set(top, right, bottom, left)
	TheApp.Event.WindowResize()
}

// MainUI runs the main UI loop of the app.
func (a *App) MainUI(vm, jniEnv, ctx uintptr) error {
	defer func() { system.HandleRecover(recover()) }()

	var dpi float32
	var orientation system.ScreenOrientation

	for {
		select {
		case <-a.MainDone:
			a.FullDestroyGPU()
			return nil
		case f := <-a.MainQueue:
			f.F()
			if f.Done != nil {
				f.Done <- struct{}{}
			}
		case cfg := <-windowConfigChange:
			dpi = cfg.DPI
			orientation = cfg.Orientation
		case w := <-windowRedrawNeeded:
			widthPx := int(C.ANativeWindow_getWidth(w))
			heightPx := int(C.ANativeWindow_getHeight(w))

			a.Scrn.Orientation = orientation

			a.Scrn.DevicePixelRatio = dpi / 160
			a.Scrn.PixelSize = image.Pt(widthPx, heightPx)
			a.Scrn.Geometry.Max = a.Scrn.PixelSize

			a.Scrn.PhysicalDPI = dpi
			a.Scrn.LogicalDPI = dpi

			if system.InitScreenLogicalDPIFunc != nil {
				system.InitScreenLogicalDPIFunc()
			}

			physX := 25.4 * float32(widthPx) / dpi
			physY := 25.4 * float32(heightPx) / dpi
			a.Scrn.PhysicalSize = image.Pt(int(physX), int(physY))

			// we only send OnSystemWindowCreated after we get the screen info
			if system.OnSystemWindowCreated != nil {
				system.OnSystemWindowCreated <- struct{}{}
			}

			a.Draw.System.Renderer.SetSize(a.Scrn.PixelSize)
			a.Event.WindowResize()
		case <-windowDestroyed:
			// we need to set the size of the window to 0 so that it detects a size difference
			// and lets the size event go through when we come back later
			a.Win.SetSize(image.Point{})
			a.Event.Window(events.WinMinimize)
			a.DestroyGPU()
		case <-activityDestroyed:
			// TODO(kai): needed to stop crashing when changing orientation
			// a.Event.Window(events.WinClose)
		}
	}
}

// RunInputQueue runs the input queue for the app.
func RunInputQueue(vm, jniEnv, ctx uintptr) error {
	env := (*C.JNIEnv)(unsafe.Pointer(jniEnv)) // not a Go heap pointer

	// Android loopers select on OS file descriptors, not Go channels, so we
	// translate the inputQueue channel to an ALooper_wake call.
	l := C.ALooper_prepare(C.ALOOPER_PREPARE_ALLOW_NON_CALLBACKS)
	pending := make(chan *C.AInputQueue, 1)
	go func() {
		for q := range inputQueue {
			pending <- q
			C.ALooper_wake(l)
		}
	}()

	var q *C.AInputQueue
	for {
		if C.ALooper_pollOnce(-1, nil, nil, nil) == C.ALOOPER_POLL_WAKE {
			select {
			default:
			case p := <-pending:
				if q != nil {
					ProcessEvents(env, q)
					C.AInputQueue_detachLooper(q)
				}
				q = p
				if q != nil {
					C.AInputQueue_attachLooper(q, l, 0, nil, nil)
				}
				inputQueueDone <- struct{}{}
			}
		}
		if q != nil {
			ProcessEvents(env, q)
		}
	}
}
