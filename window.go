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
	// "reflect"
	"runtime"
	"sync"
)

// todo: could have two subtypes of windows, one a native 3D with OpenGl etc.

// Window provides an OS-specific window and all the associated event handling
type Window struct {
	NodeBase
	Win           OSWindow              `json:"-",desc:"OS-specific window interface"`
	EventSigs     [EventTypeN]ki.Signal `json:"-",desc:"signals for communicating each type of window (wde) event"`
	Focus         ki.Ki                 `json:"-",desc:"node receiving keyboard events"`
	Dragging      ki.Ki                 `json:"-",desc:"node receiving mouse dragging events"`
	stopEventLoop bool                  `json:"-",desc:"signal for communicating all user events (mouse, keyboard, etc)"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Window = ki.Types.AddType(&Window{}, nil)

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
	// we signal ourselves!
	win.NodeSig.Connect(win.This, SignalWindow)
	return win
}

// create a new window with given name and sizing, and initialize a 2D viewport within it
func NewWindow2D(name string, width, height int) *Window {
	win := NewWindow(name, width, height)
	vp := NewViewport2D(width, height)
	win.AddChildNamed(vp, "WinVp")
	// vp.NodeSig.Connect(win.This, SignalWindow)
	return win
}

func (w *Window) WinViewport2D() *Viewport2D {
	vpi := w.FindChildByType(KiT_Viewport2D)
	vp, _ := vpi.(*Viewport2D)
	return vp
}

func (w *Window) Resize(width, height int) {
	vp := w.WinViewport2D()
	if vp != nil {
		// fmt.Printf("resize to: %v, %v\n", width, height)
		vp.Resize(width, height)
	}
}

func SignalWindow(winki, node ki.Ki, sig int64, data interface{}) {
	win, ok := winki.(*Window) // will fail if not a window
	if !ok {
		return
	}
	vpki := win.FindChildByType(KiT_Viewport2D) // should be first one
	if vpki == nil {
		fmt.Print("vpki not found\n")
		return
	}
	vp, ok := vpki.(*Viewport2D)
	if !ok {
		fmt.Print("vp not a vp\n")
		return
	}
	// fmt.Printf("window: %v rendering due to signal: %v from node: %v\n", win.PathUnique(), ki.NodeSignals(sig), node.PathUnique())

	vp.FullRender2DRoot()
}

func (w *Window) ReceiveEventType(recv ki.Ki, et EventType, fun ki.RecvFun) {
	if et >= EventTypeN {
		log.Printf("Window ReceiveEventType type: %v is not a known event type\n", et)
		return
	}
	w.EventSigs[et].Connect(recv, fun)
}

// disconnect node from all signals
func (w *Window) DisconnectNode(recv ki.Ki) {
	for _, es := range w.EventSigs {
		es.Disconnect(recv, nil)
	}
}

// tell the event loop to stop running
func (w *Window) StopEventLoop() {
	w.stopEventLoop = true
}

func (w *Window) StartEventLoop() {
	vp := w.WinViewport2D()
	if vp != nil {
		vp.FullRender2DRoot()
	}
	w.SetNextFocusItem()
	var wg sync.WaitGroup
	wg.Add(1)
	go w.EventLoop()
	wg.Wait()
}

// send given event signal to all receivers that want it -- note that because
// there is a different EventSig for each event type, we are ONLY looking at
// nodes that have registered to receive that type of event -- the further
// filtering is just to ensure that they are in the right position to receive
// the event (focus, etc)
func (w *Window) SendEventSignal(ei interface{}) {
	evi, ok := ei.(Event)
	if !ok {
		return
	}
	et := evi.EventType()
	if et > EventTypeN || et < 0 {
		return // can't handle other types of events here due to EventSigs[et] size
	}
	// fmt.Printf("got event type: %v\n", et)
	// first just process all the events straight-up
	w.EventSigs[et].EmitFiltered(w.This, int64(et), ei, func(k ki.Ki) bool {
		_, gi := KiToNode2D(k)
		if gi != nil {
			if evi.EventOnFocus() {
				if gi.This != w.Focus { // todo: could use GiNodeI interface
					return false
				}
			} else if evi.EventHasPos() {
				pos := evi.EventPos()
				// fmt.Printf("checking pos %v of: %v\n", pos, gi.PathUnique())

				// drag events start with node but can go beyond it..
				_, ok := evi.(MouseDraggedEvent)
				if ok {
					if w.Dragging != nil {
						return (w.Dragging == gi.This) // only dragger gets events
					} else {
						if pos.In(gi.WinBBox) {
							w.Dragging = gi.This
							ki.SetBitFlag(&gi.NodeFlags, int(NodeDragging))
							return true
						}
						return false
					}
				} else {
					if w.Dragging != nil {
						_, dg := KiToNode2D(w.Dragging)
						if dg != nil {
							ki.ClearBitFlag(&dg.NodeFlags, int(NodeDragging))
						}
						w.Dragging = nil
						return true // send event just after dragging for sure
					}
					if !pos.In(gi.WinBBox) {
						return false // todo: we should probably check entered / existed events and set flags accordingly -- this is a diff pathway for that
					}
				}
			}
		} else {
			// todo: get a 3D
			return false
		}
		return true
	})
}

// process MouseMoved events for enter / exit status
func (w *Window) ProcessMouseMovedEvent(ei interface{}) {
	evi, ok := ei.(Event)
	if !ok {
		return
	}
	pos := evi.EventPos()
	var mene MouseEnteredEvent
	mene.From = pos
	var mexe MouseExitedEvent
	mexe.From = pos
	enex := []EventType{MouseEnteredEventType, MouseExitedEventType}
	for _, ete := range enex {
		nwei := interface{}(nil)
		if ete == MouseEnteredEventType {
			nwei = mene
		} else {
			nwei = mexe
		}
		w.EventSigs[ete].EmitFiltered(w.This, int64(ete), nwei, func(k ki.Ki) bool {
			_, gi := KiToNode2D(k)
			if gi != nil {
				in := pos.In(gi.WinBBox)
				if in {
					if ete == MouseEnteredEventType {
						if ki.HasBitFlag(gi.NodeFlags, int(MouseHasEntered)) {
							return false // already in
						}
						ki.SetBitFlag(&gi.NodeFlags, int(MouseHasEntered)) // we'll send the event, and now set the flag
					} else {
						return false // don't send any exited events if in
					}
				} else { // mouse not in object
					if ete == MouseExitedEventType {
						if ki.HasBitFlag(gi.NodeFlags, int(MouseHasEntered)) {
							ki.ClearBitFlag(&gi.NodeFlags, int(MouseHasEntered)) // we'll send the event, and now set the flag
						} else {
							return false // already out..
						}
					} else {
						return false // don't send any exited events if in
					}
				}
			} else {
				// todo: 3D
				return false
			}
			return true
		})
	}

}

// start the event loop running -- runs in a separate goroutine
func (w *Window) EventLoop() {
	// todo: separate the inner and outer loops here?  not sure if events needs to be outside?
	events := w.Win.EventChan()

	lastResize := interface{}(nil)

	for ei := range events {
		if w.stopEventLoop {
			w.stopEventLoop = false
			fmt.Println("stop event loop")
		}
		runtime.Gosched()

		evi, ok := ei.(Event)
		if !ok {
			log.Printf("Gi Window: programmer error -- got a non-Event -- event does not define all EventI interface methods\n")
			continue
		}
		et := evi.EventType()
		if et > EventTypeN || et < 0 { // we don't handle other types of events here
			continue
		}
		w.SendEventSignal(ei)
		if et == MouseMovedEventType {
			w.ProcessMouseMovedEvent(ei)
		}
		// todo: what about iconify events!?
		if et == CloseEventType {
			fmt.Println("close")
			w.Win.Close()
			// todo: only if last one..
			StopBackendEventLoop()
		}
		if et == ResizeEventType {
			lastResize = ei
		} else {
			if lastResize != nil { // only do last one
				rev, ok := lastResize.(ResizeEvent)
				lastResize = nil
				if ok {
					w.Resize(rev.Width, rev.Height)
				}
			}
		}
		if et == KeyTypedEventType {
			kt, ok := ei.(KeyTypedEvent)
			if ok {
				kf := KeyFun(kt.Key, kt.Chord)
				switch kf {
				case KeyFunFocusNext: // todo: should we absorb this event or not?  if so, goes first..
					w.SetNextFocusItem()
				}
				// fmt.Printf("key typed: key: %v glyph: %v Chord: %v\n", kt.Key, kt.Glyph, kt.Chord)
			}
		}
	}
	fmt.Println("end of events")
}

// set focus to given item -- returns true if focus changed
func (w *Window) SetFocusItem(k ki.Ki) bool {
	if w.Focus == k {
		return false
	}
	if w.Focus != nil {
		gii, gi := KiToNode2D(w.Focus)
		if gi != nil {
			ki.ClearBitFlag(&gi.NodeFlags, int(HasFocus))
			gii.FocusChanged2D(false)
		}
	}
	w.Focus = k
	if k == nil {
		return true
	}
	gii, gi := KiToNode2D(k)
	if gi != nil {
		ki.SetBitFlag(&gi.NodeFlags, int(HasFocus))
		gii.FocusChanged2D(true)
	}
	return true
}

// set the focus on the next item that can accept focus -- returns true if a focus item found
// todo: going the other direction is going to be tricky!
func (w *Window) SetNextFocusItem() bool {
	gotFocus := false
	focusNext := false // get the next guy
	if w.Focus == nil {
		focusNext = true
	}

	for i := 0; i < 2; i++ {
		w.FunDownMeFirst(0, w, func(k ki.Ki, level int, d interface{}) bool {
			if gotFocus {
				return false
			}
			// todo: see about 3D guys
			_, gi := KiToNode2D(k)
			if gi == nil {
				return true
			}
			if gi.Paint.Off { // off below this
				return false
			}
			if !ki.HasBitFlag(gi.NodeFlags, int(CanFocus)) {
				return true
			}
			if focusNext {
				w.SetFocusItem(k)
				gotFocus = true
				return false // done
			}
			if w.Focus == k {
				focusNext = true
			}
			return true
		})
		if gotFocus {
			return true
		}
		focusNext = true // this time around, just get the first one
	}
	return false
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
		gi.RunBackendEventLoop()
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
gi.RunBackendEventLoop(), gi.StopBackendEventLoop() and gi.NewOSWindow() without referring to the
backend explicitly.

If you pupt the registration import in a separate file filtered for
the correct platform, your project will work on all three major
platforms without configuration.

That is, if you import gi/xgb in a file named "gi_linux.go",
gi/win in a file named "gi_windows.go" and gi/cocoa in a
file named "gi_darwin.go", the go tool will import the correct one.

*/
func RunBackendEventLoop() {
	BackendRun()
}

var BackendRun = func() {
	panic("no gi backend imported")
}

/*
Call this when you want gi.Run() to return. Usually to allow your
program to exit gracefully.
*/
func StopBackendEventLoop() {
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
