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
void makeCurrentContext(GLintptr ctx);
void swapBuffers(GLintptr ctx);
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
	"goki.dev/mobile/event/touch"
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

func main(f func(App)) {
	//if tid := uint64(C.threadID()); tid != initThreadID {
	//	log.Fatalf("app.Run called on thread %d, but app.init ran on %d", tid, initThreadID)
	//}

	log.Println("in mobile main")
	go func() {
		f(theApp)
		// TODO(crawshaw): trigger runApp to return
	}()
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
	theApp.screen.Orientation = goosi.OrientationUnknown
	switch orientation {
	case C.UIDeviceOrientationPortrait, C.UIDeviceOrientationPortraitUpsideDown:
		theApp.screen.Orientation = goosi.Portrait
	case C.UIDeviceOrientationLandscapeLeft, C.UIDeviceOrientationLandscapeRight:
		theApp.screen.Orientation = goosi.Landscape
		width, height = height, width
	}
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

	// TODO(kai): system dark mode

	theApp.window.PhysDPI = theApp.screen.PhysicalDPI
	theApp.window.LogDPI = theApp.screen.LogicalDPI
	theApp.window.PxSize = theApp.screen.PixSize
	theApp.window.WnSize = theApp.screen.Geometry.Max
	theApp.window.DevPixRatio = theApp.screen.DevicePixelRatio

	theApp.window.EvMgr.WindowResize()
	theApp.window.EvMgr.WindowPaint()
}

// touchIDs is the current active touches. The position in the array
// is the ID, the value is the UITouch* pointer value.
//
// It is widely reported that the iPhone can handle up to 5 simultaneous
// touch events, while the iPad can handle 11.
var touchIDs [11]uintptr

//export sendTouch
func sendTouch(cTouch, cTouchType uintptr, x, y float32) {
	id := -1
	for i, val := range touchIDs {
		if val == cTouch {
			id = i
			break
		}
	}
	if id == -1 {
		for i, val := range touchIDs {
			if val == 0 {
				touchIDs[i] = cTouch
				id = i
				break
			}
		}
		if id == -1 {
			panic("out of touchIDs")
		}
	}
	t := events.TouchStart
	switch cTouchType {
	case 0:
		t = events.TouchStart
	case 1:
		t = events.TouchMove
	case 2:
		t = events.TouchEnd
	}
	if t == events.TouchEnd {
		// Clear all touchIDs when touch ends. The UITouch pointers are unique
		// at every multi-touch event. See:
		// https://github.com/fyne-io/fyne/issues/2407
		// https://developer.apple.com/documentation/uikit/touches_presses_and_gestures?language=objc
		for idx := range touchIDs {
			touchIDs[idx] = 0
		}
	}

	theApp.eventsIn <- touch.Event{
		X:        x,
		Y:        y,
		Sequence: touch.Sequence(id),
		Type:     t,
	}
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
	theApp.window.EvMgr.Window(events.WinShow)
}

//export lifecycleFocused
func lifecycleFocused() {
	fmt.Println("lifecycle focused")
	theApp.window.EvMgr.Window(events.WinFocus)
}

//export drawloop
func drawloop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for {
		select {
		case <-theApp.publish:
			theApp.publishResult <- PublishResult{}
			return
		case <-time.After(100 * time.Millisecond): // incase the method blocked!!
			return
		}
	}
}

//export startloop
func startloop(ctx C.GLintptr) {
	go theApp.loop(ctx)
}

// loop is the primary drawing loop.
//
// After UIKit has captured the initial OS thread for processing UIKit
// events in runApp, it starts loop on another goroutine. It is locked
// to an OS thread for its OpenGL context.
func (a *app) loop(ctx C.GLintptr) {
	runtime.LockOSThread()

	for {
		select {
		case <-theApp.publish:
			theApp.publishResult <- PublishResult{}
		}
	}
}

// driverShowVirtualKeyboard requests the driver to show a virtual keyboard for text input
func driverShowVirtualKeyboard(keyboard KeyboardType) {
	C.showKeyboard(C.int(int32(keyboard)))
}

// driverHideVirtualKeyboard requests the driver to hide any visible virtual keyboard
func driverHideVirtualKeyboard() {
	C.hideKeyboard()
}
