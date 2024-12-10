// Copyright 2023 Cogent Core. All rights reserved.
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
	"strings"
	"unsafe"

	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
)

// MainLoop is the main app loop.
//
// We process UIKit events in runApp on the initial OS thread and run the
// standard system main loop in another goroutine.
func (a *App) MainLoop() {
	go a.AppSingle.MainLoop()

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
	TheApp.SetSystemWindow(uintptr(unsafe.Pointer(window)))
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
	TheApp.Scrn.Orientation = system.OrientationUnknown
	switch orientation {
	case C.UIDeviceOrientationPortrait, C.UIDeviceOrientationPortraitUpsideDown:
		TheApp.Scrn.Orientation = system.Portrait
	case C.UIDeviceOrientationLandscapeLeft, C.UIDeviceOrientationLandscapeRight:
		TheApp.Scrn.Orientation = system.Landscape
		width, height = height, width
	}
	insets := C.getDevicePadding()
	s := DisplayMetrics.ScreenScale
	TheApp.Insets.Set(
		int(insets.top)*s,
		int(insets.right)*s,
		int(insets.bottom)*s,
		int(insets.left)*s,
	)

	TheApp.Scrn.DevicePixelRatio = float32(s) // TODO(kai): is this actually DevicePixelRatio?
	TheApp.Scrn.PixelSize = image.Pt(int(width), int(height))
	TheApp.Scrn.Geometry.Max = TheApp.Scrn.PixelSize

	TheApp.Scrn.PhysicalDPI = DisplayMetrics.DPI
	TheApp.Scrn.LogicalDPI = DisplayMetrics.DPI

	if system.InitScreenLogicalDPIFunc != nil {
		system.InitScreenLogicalDPIFunc()
	}

	physX := 25.4 * float32(width) / DisplayMetrics.DPI
	physY := 25.4 * float32(height) / DisplayMetrics.DPI
	TheApp.Scrn.PhysicalSize = image.Pt(int(physX), int(physY))

	TheApp.Dark = bool(C.isDark())

	// we only send OnSystemWindowCreated after we get the screen info
	if system.OnSystemWindowCreated != nil {
		system.OnSystemWindowCreated <- struct{}{}
	}
	if TheApp.Draw != nil {
		TheApp.Draw.System.Renderer.SetSize(TheApp.Scrn.PixelSize)
	}
	TheApp.Event.WindowResize()
}

//export lifecycleDead
func lifecycleDead() {
	TheApp.FullDestroyGPU()
}

//export lifecycleAlive
func lifecycleAlive() {
}

//export lifecycleVisible
func lifecycleVisible() {
	if TheApp.Win != nil {
		TheApp.Event.Window(events.WinShow)
	}
}

//export lifecycleFocused
func lifecycleFocused() {
	if TheApp.Win != nil {
		TheApp.Event.Window(events.WinFocus)
	}
}

// ShowVirtualKeyboard requests the driver to show a virtual keyboard for text input
func (a *App) ShowVirtualKeyboard(typ styles.VirtualKeyboards) {
	C.showKeyboard(C.int(int32(typ)))
}

// HideVirtualKeyboard requests the driver to hide any visible virtual keyboard
func (a *App) HideVirtualKeyboard() {
	C.hideKeyboard()
}
