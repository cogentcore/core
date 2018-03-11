// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/rcoreilly/goki/ki"
	"image"
	"image/draw"
	"log"
	"reflect"
	"runtime"
	"sync"
)

// todo: could have two subtypes of windows, one a native 3D with OpenGl etc.

// Window provides an OS window using go.wde package
type Window struct {
	GiNode
	Win           OSWindow              `json:"-",desc:"OS-specific window interface"`
	EventSigs     [EventTypeN]ki.Signal `json:"-",desc:"signals for communicating each type of window (wde) event"`
	Focus         *GiNode               `json:"-",desc:"node receiving keyboard events"`
	stopEventLoop bool                  `json:"-",desc:"signal for communicating all user events (mouse, keyboard, etc)"`
}

// create a new window with given name and sizing
func NewWindow(name string, width, height int) *Window {
	win := &Window{}
	win.SetThisName(win, name)
	var err error
	win.Win, err = NewOSWindow(width, height)
	if err != nil {
		fmt.Printf("gogi NewWindow error: %v \n", err)
		return nil
	}
	win.Win.SetTitle(name)
	return win
}

// create a new window with given name and sizing, and initialize a 2D viewport within it
func NewWindow2D(name string, width, height int) *Window {
	win := NewWindow(name, width, height)
	vp := NewViewport2D(width, height)
	win.AddChild(vp)
	return win
}

func (w *Window) WinViewport2D() *Viewport2D {
	vpi := w.FindChildByType(reflect.TypeOf(Viewport2D{}))
	vp, _ := vpi.(*Viewport2D)
	return vp
}

func (w *Window) ReceiveEventType(recv ki.Ki, et EventType, fun ki.RecvFun) {
	if et >= EventTypeN {
		log.Printf("Window ReceiveEventType type: %v is not a known event type\n", et)
		return
	}
	w.EventSigs[et].Connect(recv, fun)
}

// tell the event loop to stop running
func (w *Window) StopEventLoop() {
	w.stopEventLoop = true
}

func (w *Window) StartEventLoop() {
	var wg sync.WaitGroup
	wg.Add(1)
	go w.EventLoop()
	wg.Wait()
}

// start the event loop running -- runs in a separate goroutine
func (w *Window) EventLoop() {
	// todo: separate the inner and outer loops here?  not sure if events needs to be outside?
	events := w.Win.EventChan()

	for ei := range events {
		if w.stopEventLoop {
			w.stopEventLoop = false
			fmt.Println("stop event loop")
		}
		runtime.Gosched()
		evi, ok := ei.(EventI)
		if !ok {
			log.Printf("GoGi Window: programmer error -- got a non-EventI -- event does not define all EventI interface methods\n")
			continue
		}
		et := evi.EventTyp()
		// fmt.Printf("got event type: %v\n", et)
		if et < EventTypeN {
			w.EventSigs[et].EmitFiltered(w.This, ki.SendCustomSignal(int64(et)), ei, func(k ki.Ki) bool {
				gii, ok := k.(GiNode2DI)
				if ok {
					gi := gii.Node2D()
					if evi.EventOnFocus() {
						return &(gi.GiNode) == w.Focus // todo: could use GiNodeI interface
					} else if evi.EventHasPos() {
						pos := evi.EventPos()
						fmt.Printf("checking pos %v of: %v\n", pos, gi.PathUnique())
						return pos.In(gi.WinBBox)
					} else {
						return true
					}
				} else {
					// todo: get a 3D
					return false
				}
				return true
			})
		}
		// todo: deal with resize event -- also what about iconify events!?
		if et == CloseEventType {
			fmt.Println("close")
			w.Win.Close()
		}
	}
	fmt.Println("end of events")
}

////////////////////////////////////////////////////////////////////////////////////////
// OS-specific window

// general interface into the operating-specific window structure
type OSWindow interface {
	SetTitle(title string)
	SetSize(width, height int)
	Size() (width, height int)
	LockSize(lock bool)
	Show()
	Screen() (im WinImage)
	FlushImage(bounds ...image.Rectangle)
	EventChan() (events <-chan interface{})
	Close() (err error)
	SetCursor(cursor Cursor)
}

// window image
type WinImage interface {
	draw.Image
	// CopyRGBA() copies the source image to this image, translating
	// the source image to the provided bounds.
	CopyRGBA(src *image.RGBA, bounds image.Rectangle)
}

/*
Some wde backends (cocoa) require that this function be called in the
main thread. To make your code as cross-platform as possible, it is
recommended that your main function look like the the code below.

	func main() {
		go theRestOfYourProgram()
		gi.Run()
	}

gi.Run() will return when gi.Stop() is called.

For this to work, you must import one of the gi backends. For
instance,

	import _ "github.com/rcoreilly/goki/gi/xgb"

or

	import _ "github.com/rcoreilly/goki/gi/win"

or

	import _ "github.com/rcoreilly/goki/gi/cocoa"


will register a backend with GoGi, allowing you to call
gi.Run(), gi.Stop() and gi.NewOSWindow() without referring to the
backend explicitly.

If you pupt the registration import in a separate file filtered for
the correct platform, your project will work on all three major
platforms without configuration.

That is, if you import gi/xgb in a file named "gi_linux.go",
gi/win in a file named "gi_windows.go" and gi/cocoa in a
file named "gi_darwin.go", the go tool will import the correct one.

*/
func Run() {
	BackendRun()
}

var BackendRun = func() {
	panic("no gi backend imported")
}

/*
Call this when you want gi.Run() to return. Usually to allow your
program to exit gracefully.
*/
func Stop() {
	BackendStop()
}

var BackendStop = func() {
	panic("no gi backend imported")
}

/*
Create a new OS window with the specified width and height.
*/
func NewOSWindow(width, height int) (OSWindow, error) {
	return BackendNewWindow(width, height)
}

var BackendNewWindow = func(width, height int) (OSWindow, error) {
	panic("no gi backend imported")
}
