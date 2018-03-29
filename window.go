// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
	"image"
	"image/draw"
	"log"
	// "reflect"
	"runtime"
	"sync"
	"time"
)

// todo: could have two subtypes of windows, one a native 3D with OpenGl etc.

// Window provides an OS-specific window and all the associated event handling
type Window struct {
	NodeBase
	Win           OSWindow              `json:"-" xml:"-" desc:"OS-specific window interface"`
	EventSigs     [EventTypeN]ki.Signal `json:"-" xml:"-" desc:"signals for communicating each type of window (wde) event"`
	Focus         ki.Ki                 `json:"-" xml:"-" desc:"node receiving keyboard events"`
	Dragging      ki.Ki                 `json:"-" xml:"-" desc:"node receiving mouse dragging events"`
	Popup         ki.Ki                 `jsom:"-" xml:"-" desc:"Current popup viewport that gets all events"`
	PopupStack    []ki.Ki               `jsom:"-" xml:"-" desc:"stack of popups"`
	FocusStack    []ki.Ki               `jsom:"-" xml:"-" desc:"stack of focus"`
	LastDrag      time.Time             `json:"-" xml:"-" desc:"time since last drag event"`
	LastSentDrag  MouseDraggedEvent     `json:"-" xml:"-" desc:"last drag that we actually sent"`
	stopEventLoop bool                  `json:"-" xml:"-" desc:"signal for communicating all user events (mouse, keyboard, etc)"`
}

var KiT_Window = kit.Types.AddType(&Window{}, nil)

// create a new window with given name and sizing
func NewWindow(name string, width, height int) *Window {
	win := &Window{}
	win.SetThisName(win, name)
	var err error
	win.Win, err = NewOSWindow(width, height)
	if err != nil {
		fmt.Printf("GoGi NewWindow error: %v \n", err)
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

	vp.FullRender2DTree()
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
		vp.FullRender2DTree()
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
func (w *Window) SendEventSignal(evi Event) {
	et := evi.EventType()
	if et > EventTypeN || et < 0 {
		return // can't handle other types of events here due to EventSigs[et] size
	}

	if et == MouseDraggedEventType {
		mde, ok := evi.(MouseDraggedEvent)
		if ok {
			if w.Dragging != nil {
				td := time.Now().Sub(w.LastDrag)
				// ed := time.Now().Sub(mde.Time)
				// fmt.Printf("td %v  ed %v\n", td, ed)
				if td < 10*time.Millisecond {
					// fmt.Printf("skipping td %v\n", td)
					return // too laggy, bail on sending this event
				}
				lsd := w.LastSentDrag
				w.LastSentDrag = mde
				mde.From = lsd.From
				w.LastDrag = time.Now()
				evi = mde // reset interface to us
			}
		}
	}

	// fmt.Printf("got event type: %v\n", et)
	// first just process all the events straight-up
	w.EventSigs[et].EmitFiltered(w.This, int64(et), evi, func(k ki.Ki) bool {
		_, gi := KiToNode2D(k)
		if gi != nil {
			if w.Popup != nil && gi.Viewport.This != w.Popup { // only process popup events
				return false
			}
			if evi.EventOnFocus() {
				if gi.This != w.Focus { // todo: could use GiNodeI interface
					return false
				}
			} else if evi.EventHasPos() {
				pos := evi.EventPos()
				// fmt.Printf("checking pos %v of: %v\n", pos, gi.PathUnique())

				// drag events start with node but can go beyond it..
				mde, ok := evi.(MouseDraggedEvent)
				if ok {
					if w.Dragging == gi.This {
						return true
					} else if w.Dragging != nil {
						return false
					} else {
						if pos.In(gi.WinBBox) {
							w.LastDrag = time.Now()
							w.Dragging = gi.This
							w.LastSentDrag = mde
							bitflag.Set(&gi.NodeFlags, int(NodeDragging))
							return true
						}
						return false
					}
				} else {
					if w.Dragging == gi.This {
						_, dg := KiToNode2D(w.Dragging)
						if dg != nil {
							bitflag.Clear(&dg.NodeFlags, int(NodeDragging))
						}
						w.Dragging = nil
						return true
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
func (w *Window) ProcessMouseMovedEvent(evi Event) {
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
				if w.Popup != nil && gi.Viewport.This != w.Popup { // only process popup events
					return false
				}
				in := pos.In(gi.WinBBox)
				if in {
					if ete == MouseEnteredEventType {
						if bitflag.Has(gi.NodeFlags, int(MouseHasEntered)) {
							return false // already in
						}
						bitflag.Set(&gi.NodeFlags, int(MouseHasEntered)) // we'll send the event, and now set the flag
					} else {
						return false // don't send any exited events if in
					}
				} else { // mouse not in object
					if ete == MouseExitedEventType {
						if bitflag.Has(gi.NodeFlags, int(MouseHasEntered)) {
							bitflag.Clear(&gi.NodeFlags, int(MouseHasEntered)) // we'll send the event, and now set the flag
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

// process Mouseup events during popup for possible closing of popup
func (w *Window) PopupMouseUpEvent(evi Event) {
	gii, gi := KiToNode2D(w.Popup)
	if gi == nil {
		return
	}
	vp := gii.AsViewport2D()
	if vp == nil {
		return
	}
	// pos := evi.EventPos()
	if vp.IsMenu() {
		vp.DeletePopup()
		w.PopPopup()
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

		curPop := w.Popup

		evi, ok := ei.(Event)
		if !ok {
			log.Printf("Gi Window: programmer error -- got a non-Event -- event does not define all EventI interface methods\n")
			continue
		}
		et := evi.EventType()
		if et > EventTypeN || et < 0 { // we don't handle other types of events here
			continue
		}
		w.SendEventSignal(evi)
		if et == MouseMovedEventType {
			w.ProcessMouseMovedEvent(evi)
		}
		if w.Popup != nil && w.Popup == curPop { // special processing of events during popups
			if et == MouseUpEventType {
				w.PopupMouseUpEvent(evi)
			}
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
			bitflag.Clear(&gi.NodeFlags, int(HasFocus))
			gii.FocusChanged2D(false)
		}
	}
	w.Focus = k
	if k == nil {
		return true
	}
	gii, gi := KiToNode2D(k)
	if gi != nil {
		bitflag.Set(&gi.NodeFlags, int(HasFocus))
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
			if !bitflag.Has(gi.NodeFlags, int(CanFocus)) {
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

// push current popup onto stack and set new popup
func (w *Window) PushPopup(p ki.Ki) {
	if w.PopupStack == nil {
		w.PopupStack = make([]ki.Ki, 0, 50)
	}
	w.PopupStack = append(w.PopupStack, w.Popup)
	w.Popup = p
	w.PushFocus(p)
	w.SetNextFocusItem()
}

// pop Mask off the popup stack and set to current popup
func (w *Window) PopPopup() {
	if w.PopupStack == nil || len(w.PopupStack) == 0 {
		w.Popup = nil
		return
	}
	sz := len(w.PopupStack)
	w.Popup = w.PopupStack[sz-1]
	w.PopupStack = w.PopupStack[:sz-1]
	w.PopFocus() // always
}

// push current focus onto stack and set new focus
func (w *Window) PushFocus(p ki.Ki) {
	if w.FocusStack == nil {
		w.FocusStack = make([]ki.Ki, 0, 50)
	}
	w.FocusStack = append(w.FocusStack, w.Focus)
	w.Focus = p
}

// pop Mask off the focus stack and set to current focus
func (w *Window) PopFocus() {
	if w.FocusStack == nil || len(w.FocusStack) == 0 {
		w.Focus = nil
		return
	}
	sz := len(w.FocusStack)
	w.Focus = w.FocusStack[sz-1]
	w.FocusStack = w.FocusStack[:sz-1]
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
