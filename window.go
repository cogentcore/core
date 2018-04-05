// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log"

	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
	// "reflect"
	"runtime"
	"sync"
	"time"
)

// todo: could have two subtypes of windows, one a native 3D with OpenGl etc.

// Window provides an OS-specific window and all the associated event handling
type Window struct {
	NodeBase
	Win           oswin.OSWindow              `json:"-" xml:"-" desc:"OS-specific window interface"`
	EventSigs     [oswin.EventTypeN]ki.Signal `json:"-" xml:"-" desc:"signals for communicating each type of window (wde) event"`
	Focus         ki.Ki                       `json:"-" xml:"-" desc:"node receiving keyboard events"`
	Dragging      ki.Ki                       `json:"-" xml:"-" desc:"node receiving mouse dragging events"`
	Popup         ki.Ki                       `jsom:"-" xml:"-" desc:"Current popup viewport that gets all events"`
	PopupStack    []ki.Ki                     `jsom:"-" xml:"-" desc:"stack of popups"`
	FocusStack    []ki.Ki                     `jsom:"-" xml:"-" desc:"stack of focus"`
	LastDrag      time.Time                   `json:"-" xml:"-" desc:"time since last drag event"`
	LastSentDrag  oswin.MouseDraggedEvent     `json:"-" xml:"-" desc:"last drag that we actually sent"`
	stopEventLoop bool                        `json:"-" xml:"-" desc:"signal for communicating all user events (mouse, keyboard, etc)"`
}

var KiT_Window = kit.Types.AddType(&Window{}, nil)

// create a new window with given name and sizing
func NewWindow(name string, width, height int) *Window {
	FontLibrary.AddFontPaths("/Library/Fonts")
	win := &Window{}
	win.InitName(win, name)
	var err error
	win.Win, err = oswin.NewOSWindow(width, height)
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
	return win
}

func (w *Window) WinViewport2D() *Viewport2D {
	vpi := w.ChildByType(KiT_Viewport2D, true, 0)
	vp, _ := vpi.EmbeddedStruct(KiT_Viewport2D).(*Viewport2D)
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
	win := winki.EmbeddedStruct(KiT_Window).(*Window)
	vp := win.WinViewport2D()
	// fmt.Printf("window: %v rendering due to signal: %v from node: %v\n", win.PathUnique(), ki.NodeSignals(sig), node.PathUnique())
	vp.FullRender2DTree()
}

func (w *Window) ReceiveEventType(recv ki.Ki, et oswin.EventType, fun ki.RecvFunc) {
	if et >= oswin.EventTypeN {
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
func (w *Window) SendEventSignal(evi oswin.Event) {
	et := evi.EventType()
	if et > oswin.EventTypeN || et < 0 {
		return // can't handle other types of events here due to EventSigs[et] size
	}

	if et == oswin.MouseDraggedEventType {
		mde, ok := evi.(oswin.MouseDraggedEvent)
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
				mde, ok := evi.(oswin.MouseDraggedEvent)
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
func (w *Window) ProcessMouseMovedEvent(evi oswin.Event) {
	pos := evi.EventPos()
	var mene oswin.MouseEnteredEvent
	mene.From = pos
	var mexe oswin.MouseExitedEvent
	mexe.From = pos
	enex := []oswin.EventType{oswin.MouseEnteredEventType, oswin.MouseExitedEventType}
	for _, ete := range enex {
		nwei := interface{}(nil)
		if ete == oswin.MouseEnteredEventType {
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
					if ete == oswin.MouseEnteredEventType {
						if bitflag.Has(gi.NodeFlags, int(MouseHasEntered)) {
							return false // already in
						}
						bitflag.Set(&gi.NodeFlags, int(MouseHasEntered)) // we'll send the event, and now set the flag
					} else {
						return false // don't send any exited events if in
					}
				} else { // mouse not in object
					if ete == oswin.MouseExitedEventType {
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
func (w *Window) PopupMouseUpEvent(evi oswin.Event) {
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

		evi, ok := ei.(oswin.Event)
		if !ok {
			log.Printf("Gi Window: programmer error -- got a non-Event -- event does not define all EventI interface methods\n")
			continue
		}
		et := evi.EventType()
		if et > oswin.EventTypeN || et < 0 { // we don't handle other types of events here
			continue
		}
		w.SendEventSignal(evi)
		if et == oswin.MouseMovedEventType {
			w.ProcessMouseMovedEvent(evi)
		}
		if w.Popup != nil && w.Popup == curPop { // special processing of events during popups
			if et == oswin.MouseUpEventType {
				w.PopupMouseUpEvent(evi)
			}
		}
		// todo: what about iconify events!?
		if et == oswin.CloseEventType {
			fmt.Println("close")
			w.Win.Close()
			// todo: only if last one..
			oswin.StopBackendEventLoop()
		}
		if et == oswin.ResizeEventType {
			lastResize = ei
		} else {
			if lastResize != nil { // only do last one
				rev, ok := lastResize.(oswin.ResizeEvent)
				lastResize = nil
				if ok {
					w.Resize(rev.Width, rev.Height)
				}
			}
		}
		if et == oswin.KeyTypedEventType {
			kt, ok := ei.(oswin.KeyTypedEvent)
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
		w.FuncDownMeFirst(0, w, func(k ki.Ki, level int, d interface{}) bool {
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
