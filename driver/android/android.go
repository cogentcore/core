// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/golang/mobile
// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

/*
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
void showFileOpen(JNIEnv* env, char* mimes);
void showFileSave(JNIEnv* env, char* mimes, char* filename);
*/
import "C"
import (
	"fmt"
	"image"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"time"
	"unsafe"

	"goki.dev/goosi"
	"goki.dev/goosi/driver/mobile/callfn"
	"goki.dev/goosi/driver/mobile/mobileinit"
	"goki.dev/goosi/events"
	"goki.dev/mobile/event/size"
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
	fmt.Println("calling main")
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

//export onStart
func onStart(activity *C.ANativeActivity) {
	fmt.Println("started")
}

//export onResume
func onResume(activity *C.ANativeActivity) {
}

//export onSaveInstanceState
func onSaveInstanceState(activity *C.ANativeActivity, outSize *C.size_t) unsafe.Pointer {
	return nil
}

//export onPause
func onPause(activity *C.ANativeActivity) {
}

//export onStop
func onStop(activity *C.ANativeActivity) {
}

//export onCreate
func onCreate(activity *C.ANativeActivity) {
	fmt.Println("created")
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
}

//export onNativeWindowCreated
func onNativeWindowCreated(activity *C.ANativeActivity, window *C.ANativeWindow) {
	theApp.mu.Lock()
	defer theApp.mu.Unlock()
	theApp.winptr = uintptr(unsafe.Pointer(window))
	fmt.Println("win created", theApp.winptr)
	theApp.setSysWindow(nil, theApp.winptr)
}

//export onNativeWindowRedrawNeeded
func onNativeWindowRedrawNeeded(activity *C.ANativeActivity, window *C.ANativeWindow) {
	// Called on orientation change and window resize.
	// Send a request for redraw, and block this function
	// until a complete draw and buffer swap is completed.
	// This is required by the redraw documentation to
	// avoid bad draws.
	windowRedrawNeeded <- window
	<-windowRedrawDone
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
	theApp.darkMode = bool(dark)
}

type windowConfig struct {
	orientation size.Orientation
	dotsPerPx   float32 // raw display dots per standard pixel (1/96 of 1 in)
}

func windowConfigRead(activity *C.ANativeActivity) windowConfig {
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
		log.Print("android device reports no screen density")
		dpi = 72
	default:
		// TODO: fix this always happening with value 240
		log.Printf("android device reports unknown density: %d", density)
		// All we can do is guess.
		if density > 0 {
			dpi = int(density)
		} else {
			dpi = 72
		}
	}

	o := size.OrientationUnknown
	switch orient {
	case C.ACONFIGURATION_ORIENTATION_PORT:
		o = size.OrientationPortrait
	case C.ACONFIGURATION_ORIENTATION_LAND:
		o = size.OrientationLandscape
	}

	return windowConfig{
		orientation: o,
		dotsPerPx:   float32(dpi) / 96,
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

func (app *appImpl) mainLoop() {
	fmt.Println("in main")
	app.mainQueue = make(chan funcRun)
	app.mainDone = make(chan struct{})
	// TODO: merge the runInputQueue and mainUI functions?
	go func() {
		defer func() { handleRecover(recover()) }()
		fmt.Println("running input queue")
		if err := mobileinit.RunOnJVM(runInputQueue); err != nil {
			log.Fatalf("app: %v", err)
		}
	}()
	// Preserve this OS thread for:
	//	1. the attached JNI thread
	fmt.Println("running main UI")
	if err := mobileinit.RunOnJVM(theApp.mainUI); err != nil {
		log.Fatalf("app: %v", err)
	}
}

// ShowVirtualKeyboard requests the driver to show a virtual keyboard for text input
func (a *appImpl) ShowVirtualKeyboard(typ goosi.VirtualKeyboardTypes) {
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
func (a *appImpl) HideVirtualKeyboard() {
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
	theApp.insets.Set(float32(top), float32(right), float32(bottom), float32(left))
}

func (app *appImpl) mainUI(vm, jniEnv, ctx uintptr) error {
	go func() {
		defer func() { handleRecover(recover()) }()
		mainCallback(theApp)
		app.stopMain()
	}()

	var dotsPerPx float32

	for {
		select {
		case <-app.mainDone:
			app.fullDestroyVk()
			return nil
		case f := <-app.mainQueue:
			f.f()
			if f.done != nil {
				f.done <- true
			}
		case cfg := <-windowConfigChange:
			dotsPerPx = cfg.dotsPerPx
		case w := <-windowRedrawNeeded:
			app.window.EvMgr.Window(events.Focus)

			widthDots := int(C.ANativeWindow_getWidth(w))
			heightDots := int(C.ANativeWindow_getHeight(w))

			app.screen.ScreenNumber = 0
			app.screen.DevicePixelRatio = dotsPerPx
			wsz := image.Point{widthDots, heightDots}
			app.screen.Geometry = image.Rectangle{Max: wsz}
			app.screen.PixSize = app.screen.WinSizeToPix(wsz)
			app.screen.Orientation = screenOrientation(widthDots, heightDots)
			app.screen.UpdatePhysicalDPI()
			app.screen.UpdateLogicalDPI()

			app.window.PhysDPI = app.screen.PhysicalDPI
			app.window.PxSize = app.screen.PixSize
			app.window.WnSize = wsz

			app.window.EvMgr.WindowPaint()
		case <-windowDestroyed:
			// we need to set the size of the window to 0 so that it detects a size difference
			// and lets the size event go through when we come back later
			app.window.SetSize(image.Point{})
			app.window.EvMgr.Window(events.Minimize)
			app.destroyVk()
		case <-activityDestroyed:
			app.window.EvMgr.Window(events.Close)
		}
	}
}

func screenOrientation(width, height int) goosi.ScreenOrientation {
	if width > height {
		return goosi.Landscape
	}
	return goosi.Portrait
}

func runInputQueue(vm, jniEnv, ctx uintptr) error {
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
		if C.ALooper_pollAll(-1, nil, nil, nil) == C.ALOOPER_POLL_WAKE {
			select {
			default:
			case p := <-pending:
				if q != nil {
					processEvents(env, q)
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
			processEvents(env, q)
		}
	}
}
