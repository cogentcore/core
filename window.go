// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/skelterjohn/go.wde"
	// "image"
	"github.com/rcoreilly/goki/ki"
	"log"
	"reflect"
	"runtime"
)

// todo: could have two subtypes of windows, one a native 3D with OpenGl etc.

// Window provides an OS window using go.wde package
type Window struct {
	GiNode
	Win           wde.Window            `json:"-",desc:"OS-specific window interface"`
	EventSigs     [EventTypeN]ki.Signal `json:"-",desc:"signals for communicating each type of window (wde) event"`
	Focus         *GiNode               `json:"-",desc:"node receiving keyboard events"`
	stopEventLoop bool                  `json:"-",desc:"signal for communicating all user events (mouse, keyboard, etc)"`
}

// create a new window with given name and sizing
func NewWindow(name string, width, height int) *Window {
	win := &Window{}
	win.SetThisName(win, name)
	var err error
	win.Win, err = wde.NewWindow(width, height)
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

// start the event loop running -- runs in a separate goroutine
func (w *Window) StartEventLoop() {
	// todo: separate the inner and outer loops here?  not sure if events needs to be outside?
	events := w.Win.EventChan()

	done := make(chan bool)

	go func() {
	loop:
		for ei := range events {
			if w.stopEventLoop {
				w.stopEventLoop = false
				fmt.Println("stop event loop")
				done <- true
				break loop
			}
			runtime.Gosched()
			et := EventTypeFromEvent(ei)
			if et < EventTypeN {
				w.EventSigs[et].EmitFiltered(w.This, ki.SendCustomSignal(int64(et)), ei, func(k ki.Ki) bool {
					gi, ok := k.(*GiNode)
					if !ok {
						return false
					}
					if et <= MouseDraggedEvent {
						// me := ei.(*wde.MouseEvent)
						return true
						// return me.Where.In(gi.WinBBox)
					} else if et == MagnifyEvent { // todo: better if these are all GestureEvent
						me := ei.(*wde.MagnifyEvent)
						return me.Where.In(gi.WinBBox)
					} else if et == RotateEvent { // todo: better if these are all GestureEvent
						me := ei.(*wde.RotateEvent)
						return me.Where.In(gi.WinBBox)
					} else if et == ScrollEvent { // todo: better if these are all GestureEvent
						me := ei.(*wde.ScrollEvent)
						return me.Where.In(gi.WinBBox)
					} else if et >= KeyDownEvent && et <= KeyTypedEvent {
						return gi == w.Focus // todo: could use GiNodeI interface
					}
					return false
				})
			}
			// todo: deal with resize event -- also what about iconify events!?
			if et == CloseEvent {
				fmt.Println("close")
				w.Win.Close()
				done <- true
				break loop
			}
		}
		// todo: never seems to get here
		done <- true
		fmt.Println("end of events")
	}()
}
