// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/golang/mobile
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ios

package ios

/*
#cgo CFLAGS: -x objective-c -DGL_SILENCE_DEPRECATION
#cgo LDFLAGS: -framework Foundation -framework UIKit -framework MobileCoreServices -framework QuartzCore -framework UserNotifications
#include <sys/utsname.h>
#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>
#import <UIKit/UIKit.h>
#import <MobileCoreServices/MobileCoreServices.h>
#include <UIKit/UIDevice.h>

extern struct utsname sysInfo;

void runApp(void);
uint64_t threadID();

UIEdgeInsets getDevicePadding();
bool isDark();
void showKeyboard(int keyboardType);
void hideKeyboard();
*/
import "C"
import (
	"image"
	"log"
	"log/slog"
	"runtime"
	"strings"
	"unsafe"

	"goki.dev/goosi"
	"goki.dev/goosi/events"
)

var initThreadID uint64

func init() {
	// Lock the goroutine responsible for initialization to an OS thread.
	// This means the goroutine running main (and calling the run function
	// below) is locked to the OS thread that started the program. This is
	// necessary for the correct delivery of UIKit events to the process.
	//
	// A discussion on this topic:
	// https://groups.google.com/forum/#!msg/golang-nuts/IiWZ2hUuLDA/SNKYYZBelsYJ
	runtime.LockOSThread()
	initThreadID = uint64(C.threadID())
}

// MainLoop is the main app loop.
//
// We process UIKit events in runApp on the initial OS thread and run the
// standard goosi main loop in another goroutine.
func (a *App) MainLoop() {
	if tid := uint64(C.threadID()); tid != initThreadID {
		log.Fatalf("App.MainLoop called on thread %d, but init ran on %d", tid, initThreadID)
	}

	go a.App.MainLoop()

	C.runApp()
	log.Fatalln("unexpected return from runApp")
}

// DisplayMetrics contains information about the current display information
var DisplayMetrics struct {
	// WidthPx is the width of the screen in pixels
	WidthPx int

	// HeightPx is the height of the screen in pixels
	HeightPx int

	// DPI is the current raw display dots per inch
	DPI float32

	// ScreenScale is the current [UIScreen mainScreen].scale, which is either 1, 2, or 3.
	ScreenScale int
}

//export setWindowPtr
func setWindowPtr(window *C.void) {
	TheApp.Mu.Lock()
	defer TheApp.Mu.Unlock()
	TheApp.setSysWindow(uintptr(unsafe.Pointer(window)))
}

//export setDisplayMetrics
func setDisplayMetrics(width, height int, scale int) {
	DisplayMetrics.WidthPx = width
	DisplayMetrics.HeightPx = height
	DisplayMetrics.ScreenScale = scale
}

//export setScreen
func setScreen(scale int) {
	C.uname(&C.sysInfo)
	name := C.GoString(&C.sysInfo.machine[0])

	var v float32

	switch {
	case strings.HasPrefix(name, "iPhone"):
		v = 163
	case strings.HasPrefix(name, "iPad"):
		// TODO: is there a better way to distinguish the iPad Mini?
		switch name {
		case "iPad2,5", "iPad2,6", "iPad2,7", "iPad4,4", "iPad4,5", "iPad4,6", "iPad4,7":
			v = 163 // iPad Mini
		default:
			v = 132
		}
	default:
		v = 163 // names like i386 and x86_64 are the simulator
	}

	if v == 0 {
		slog.Warn("unknown machine: %s", name)
		v = 163 // emergency fallback
	}

	DisplayMetrics.DPI = v * float32(scale)
	DisplayMetrics.ScreenScale = scale
}

//export updateConfig
func updateConfig(width, height, orientation int32) {
	TheApp.Mu.Lock()
	defer TheApp.Mu.Unlock()
	TheApp.Scrn.Orientation = goosi.OrientationUnknown
	switch orientation {
	case C.UIDeviceOrientationPortrait, C.UIDeviceOrientationPortraitUpsideDown:
		TheApp.Scrn.Orientation = goosi.Portrait
	case C.UIDeviceOrientationLandscapeLeft, C.UIDeviceOrientationLandscapeRight:
		TheApp.Scrn.Orientation = goosi.Landscape
		width, height = height, width
	}
	insets := C.getDevicePadding()
	fscale := float32(DisplayMetrics.ScreenScale)
	TheApp.Win.Insts.Set(
		float32(insets.top)*fscale,
		float32(insets.right)*fscale,
		float32(insets.bottom)*fscale,
		float32(insets.left)*fscale,
	)

	TheApp.Scrn.DevicePixelRatio = fscale // TODO(kai): is this actually DevicePixelRatio?
	TheApp.Scrn.PixSize = image.Pt(int(width), int(height))
	TheApp.Scrn.Geometry.Max = TheApp.Scrn.PixSize

	TheApp.Scrn.PhysicalDPI = DisplayMetrics.DPI
	TheApp.Scrn.LogicalDPI = DisplayMetrics.DPI

	physX := 25.4 * float32(width) / DisplayMetrics.DPI
	physY := 25.4 * float32(height) / DisplayMetrics.DPI
	TheApp.Scrn.PhysicalSize = image.Pt(int(physX), int(physY))

	TheApp.Dark = bool(C.isDark())
}

//export lifecycleDead
func lifecycleDead() {
	TheApp.fullDestroyVk()
}

//export lifecycleAlive
func lifecycleAlive() {
}

//export lifecycleVisible
func lifecycleVisible() {
	if TheApp.Win != nil {
		TheApp.Win.EvMgr.Window(events.WinShow)
	}
}

//export lifecycleFocused
func lifecycleFocused() {
	if TheApp.Win != nil {
		TheApp.Win.EvMgr.Window(events.WinFocus)
	}
}

// ShowVirtualKeyboard requests the driver to show a virtual keyboard for text input
func (app *App) ShowVirtualKeyboard(typ goosi.VirtualKeyboardTypes) {
	C.showKeyboard(C.int(int32(typ)))
}

// HideVirtualKeyboard requests the driver to hide any visible virtual keyboard
func (app *App) HideVirtualKeyboard() {
	C.hideKeyboard()
}
