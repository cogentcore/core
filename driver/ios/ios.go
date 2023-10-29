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
	"fmt"
	"image"
	"log"
	"runtime"
	"strings"
	"time"
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

func main(f func(goosi.App)) {
	//if tid := uint64(C.threadID()); tid != initThreadID {
	//	log.Fatalf("app.Run called on thread %d, but app.init ran on %d", tid, initThreadID)
	//}

	log.Println("in mobile main")
	go func() {
		f(theApp)
		// TODO(crawshaw): trigger runApp to return
	}()
	log.Println("running c app")
	C.runApp()
	panic("unexpected return from app.runApp")
}

var dpi float32     // raw display dots per inch
var screenScale int // [UIScreen mainScreen].scale, either 1, 2, or 3.

var DisplayMetrics struct {
	WidthPx  int
	HeightPx int
}

//export setWindowPtr
func setWindowPtr(window *C.void) {
	theApp.mu.Lock()
	defer theApp.mu.Unlock()
	theApp.setSysWindow(uintptr(unsafe.Pointer(window)))
}

//export setDisplayMetrics
func setDisplayMetrics(width, height int, scale int) {
	DisplayMetrics.WidthPx = width
	DisplayMetrics.HeightPx = height
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
		log.Printf("unknown machine: %s", name)
		v = 163 // emergency fallback
	}

	dpi = v * float32(scale)
	screenScale = scale
}

//export updateConfig
func updateConfig(width, height, orientation int32) {
	theApp.mu.Lock()
	defer theApp.mu.Unlock()
	theApp.screen.Orientation = goosi.OrientationUnknown
	switch orientation {
	case C.UIDeviceOrientationPortrait, C.UIDeviceOrientationPortraitUpsideDown:
		theApp.screen.Orientation = goosi.Portrait
	case C.UIDeviceOrientationLandscapeLeft, C.UIDeviceOrientationLandscapeRight:
		theApp.screen.Orientation = goosi.Landscape
		width, height = height, width
	}
	fmt.Println("getting device padding")
	insets := C.getDevicePadding()
	fscale := float32(screenScale)
	theApp.insets.Set(
		float32(insets.top)*fscale,
		float32(insets.right)*fscale,
		float32(insets.bottom)*fscale,
		float32(insets.left)*fscale,
	)

	theApp.screen.DevicePixelRatio = fscale // TODO(kai): is this actually DevicePixelRatio?
	theApp.screen.PixSize = image.Pt(int(width), int(height))
	theApp.screen.Geometry.Max = theApp.screen.PixSize

	theApp.screen.PhysicalDPI = dpi
	theApp.screen.LogicalDPI = dpi

	physX := 25.4 * float32(width) / dpi
	physY := 25.4 * float32(height) / dpi
	theApp.screen.PhysicalSize = image.Pt(int(physX), int(physY))

	fmt.Println("getting is dark")
	theApp.isDark = bool(C.isDark())
	fmt.Println("got is dark")
}

//export lifecycleDead
func lifecycleDead() {
	fmt.Println("lifecycle dead")
	theApp.fullDestroyVk()
}

//export lifecycleAlive
func lifecycleAlive() {
	fmt.Println("lifecycle alive")
}

//export lifecycleVisible
func lifecycleVisible() {
	fmt.Println("lifecycle visible")
	if theApp.window != nil {
		theApp.window.EvMgr.Window(events.WinShow)
	}
}

//export lifecycleFocused
func lifecycleFocused() {
	fmt.Println("lifecycle focused")
	if theApp.window != nil {
		theApp.window.EvMgr.Window(events.WinFocus)
	}
}

//export drawloop
func drawloop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for {
		select {
		case <-theApp.window.publish:
			theApp.window.publishDone <- struct{}{}
			return
		case <-time.After(100 * time.Millisecond): // in case the method blocked!
			return
		}
	}
}

//export startloop
func startloop() {
	go theApp.loop()
}

// loop is the primary drawing loop.
//
// After UIKit has captured the initial OS thread for processing UIKit
// events in runApp, it starts loop on another goroutine. It is locked
// to an OS thread for its OpenGL context.
func (app *appImpl) loop() {
	runtime.LockOSThread()

	for {
		select {
		case <-app.mainDone:
			app.fullDestroyVk()
			return
		case f := <-app.mainQueue:
			f.f()
			if f.done != nil {
				f.done <- true
			}
		case <-theApp.window.publish:
			theApp.window.publishDone <- struct{}{}
		}
	}
}

// ShowVirtualKeyboard requests the driver to show a virtual keyboard for text input
func (app *appImpl) ShowVirtualKeyboard(typ goosi.VirtualKeyboardTypes) {
	C.showKeyboard(C.int(int32(typ)))
}

// HideVirtualKeyboard requests the driver to hide any visible virtual keyboard
func (app *appImpl) HideVirtualKeyboard() {
	C.hideKeyboard()
}
