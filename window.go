// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"

	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/oswin/key"
	"github.com/rcoreilly/goki/gi/oswin/lifecycle"
	"github.com/rcoreilly/goki/gi/oswin/mouse"
	"github.com/rcoreilly/goki/gi/oswin/paint"
	"github.com/rcoreilly/goki/gi/oswin/window"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
	// "reflect"

	"time"
)

// notes: oswin/Image is the thing that a Vp should have uploader uploads the
// buffer/image to the window -- can also render directly onto window using
// textures using the drawer interface, but..

// todo: could have two subtypes of windows, one a native 3D with OpenGl etc.

// Window provides an OS-specific window and all the associated event handling
type Window struct {
	NodeBase
	Viewport      *Viewport2D                 `json:"-" xml:"-" desc:"convenience pointer to our viewport child that handles all of our rendering"`
	OSWin         oswin.Window                `json:"-" xml:"-" desc:"OS-specific window interface"`
	WinTex        oswin.Texture               `json:"-" xml:"-" desc:"texture for the entire window -- all rendering is done onto this texture, which then updates the window"`
	EventSigs     [oswin.EventTypeN]ki.Signal `json:"-" xml:"-" desc:"signals for communicating each type of event"`
	Focus         ki.Ki                       `json:"-" xml:"-" desc:"node receiving keyboard events"`
	Dragging      ki.Ki                       `json:"-" xml:"-" desc:"node receiving mouse dragging events"`
	Popup         ki.Ki                       `jsom:"-" xml:"-" desc:"Current popup viewport that gets all events"`
	PopupStack    []ki.Ki                     `jsom:"-" xml:"-" desc:"stack of popups"`
	FocusStack    []ki.Ki                     `jsom:"-" xml:"-" desc:"stack of focus"`
	LastDrag      time.Time                   `json:"-" xml:"-" desc:"time since last drag event"`
	LastSentDrag  mouse.DragEvent             `json:"-" xml:"-" desc:"last drag that we actually sent"`
	stopEventLoop bool                        `json:"-" xml:"-" desc:"signal for communicating all user events (mouse, keyboard, etc)"`
	DoFullRender  bool                        `json:"-" xml:"-" desc:"triggers a full re-render of the window within the event loop -- cleared once done"`
}

var KiT_Window = kit.Types.AddType(&Window{}, nil)

// NewWindow creates a new window with given name and sizing (0 = some kind of
// default) -- stdPixels means use standardized "pixel" units for the display
// size (96 per inch), not the actual underlying raw display dot pixels
func NewWindow(name string, width, height int, stdPixels bool) *Window {
	FontLibrary.AddFontPaths("/Library/Fonts")
	win := &Window{}
	win.InitName(win, name)
	win.SetOnlySelfUpdate() // has its own FlushImage update logic
	var err error
	sz := image.Point{width, height}
	if stdPixels {
		unctx := units.Context{}
		unctx.Defaults()
		if oswin.TheApp.NScreens() > 0 {
			sc := oswin.TheApp.Screen(0)
			unctx.DPI = float64(sc.LogicalDPI)
			fmt.Printf("screen logical dpi is: %v\n", sc.LogicalDPI)
			sz.X = int(unctx.ToDots(float64(width), units.Px))
			sz.Y = int(unctx.ToDots(float64(height), units.Px))
		}
	}
	win.OSWin, err = oswin.TheApp.NewWindow(&oswin.NewWindowOptions{
		Title: name, Width: sz.X, Height: sz.Y,
	})
	if err != nil {
		fmt.Printf("GoGi NewWindow error: %v \n", err)
		return nil
	}
	win.WinTex, err = oswin.TheApp.NewTexture(sz)
	if err != nil {
		fmt.Printf("GoGi NewTexture error: %v \n", err)
		return nil
	}
	win.OSWin.SetName(name)
	win.NodeSig.Connect(win.This, SignalWindowFlush)
	return win
}

// NewWindow2D creates a new window with given name and sizing, and initializes
// a 2D viewport within it -- stdPixels means use standardized "pixel" units for
// the display size (96 per inch), not the actual underlying raw display dot
// pixels
func NewWindow2D(name string, width, height int, stdPixels bool) *Window {
	win := NewWindow(name, width, height, stdPixels)
	if win == nil {
		return nil
	}
	vp := NewViewport2D(width, height)
	vp.SetName("WinVp")
	win.AddChild(vp)
	win.Viewport = vp
	return win
}

func (w *Window) WinViewport2D() *Viewport2D {
	vpi := w.ChildByType(KiT_Viewport2D, true, 0)
	vp, _ := vpi.EmbeddedStruct(KiT_Viewport2D).(*Viewport2D)
	return vp
}

func (w *Window) Resize(width, height int) {
	w.WinTex, _ = oswin.TheApp.NewTexture(image.Point{width, height})
	w.Viewport.Resize(width, height)
}

// UpdateVpRegion updates pixels for one viewport region on the screen, using
// vpBBox bounding box for the viewport, and winBBox bounding box for the
// window (which should not be empty given the overall logic driving updates)
// -- the Window has a its OnlySelfUpdate logic for determining when to flush
// changes to the underlying OSWindow -- wrap updates in win.UpdateStart /
// win.UpdateEnd to actually flush the updates to be visible
func (w *Window) UpdateVpRegion(vp *Viewport2D, vpBBox, winBBox image.Rectangle) {
	w.WinTex.Upload(winBBox.Min, vp.OSImage, vpBBox)
}

// UpdateVpPixels updates pixels for one viewport region on the screen, in its entirety
func (w *Window) UpdateFullVpRegion(vp *Viewport2D, vpBBox, winBBox image.Rectangle) {
	w.WinTex.Upload(winBBox.Min, vp.OSImage, vp.OSImage.Bounds())
}

// UpdateVpRegionFromMain basically clears the region where the vp would show
// up, from the main
func (w *Window) UpdateVpRegionFromMain(winBBox image.Rectangle) {
	w.WinTex.Upload(winBBox.Min, w.Viewport.OSImage, winBBox)
}

// FullUpdate does a complete update of window pixels -- grab pixels from all the different active viewports
func (w *Window) FullUpdate() {
	w.UpdateStart()
	w.WinTex.Upload(image.ZP, w.Viewport.OSImage, w.Viewport.OSImage.Bounds())
	// then all the current popups
	if w.PopupStack != nil {
		for _, pop := range w.PopupStack {
			gii, _ := KiToNode2D(pop)
			if gii != nil {
				vp := gii.AsViewport2D()
				r := vp.ViewBox.Bounds()
				w.WinTex.Upload(r.Min, vp.OSImage, vp.OSImage.Bounds())
			}
		}
	}
	if w.Popup != nil {
		gii, _ := KiToNode2D(w.Popup)
		if gii != nil {
			vp := gii.AsViewport2D()
			r := vp.ViewBox.Bounds()
			w.WinTex.Upload(r.Min, vp.OSImage, vp.OSImage.Bounds())
		}
	}
	w.UpdateEnd() // drives the flush
}

func (w *Window) Publish() {
	w.OSWin.Copy(image.ZP, w.WinTex, w.WinTex.Bounds(), oswin.Over, nil)
	w.OSWin.Publish()

}

func SignalWindowFlush(winki, node ki.Ki, sig int64, data interface{}) {
	win := winki.EmbeddedStruct(KiT_Window).(*Window)
	if Render2DTrace {
		fmt.Printf("Window: %v flushing image due to signal: %v from node: %v\n", win.PathUnique(), ki.NodeSignals(sig), node.PathUnique())
	}
	win.Publish()
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
	// w.DoFullRender = true
	// var wg sync.WaitGroup
	// wg.Add(1)
	w.EventLoop()
	// wg.Wait()
}

func (w *Window) StartEventLoopNoWait() {
	w.DoFullRender = true
	go w.EventLoop()
}

// send given event signal to all receivers that want it -- note that because
// there is a different EventSig for each event type, we are ONLY looking at
// nodes that have registered to receive that type of event -- the further
// filtering is just to ensure that they are in the right position to receive
// the event (focus, etc)
func (w *Window) SendEventSignal(evi oswin.Event) {
	if evi.IsProcessed() { // someone took care of it
		return
	}
	et := evi.Type()
	if et > oswin.EventTypeN || et < 0 {
		return // can't handle other types of events here due to EventSigs[et] size
	}

	// if et == oswin.MouseDraggedEventType {
	// 	mde, ok := evi.(*oswin.MouseDraggedEvent)
	// 	if ok {
	// 		if w.Dragging != nil {
	// 			// td := time.Now().Sub(w.LastDrag)
	// 			// ed := time.Now().Sub(mde.Time)
	// 			// fmt.Printf("td %v  ed %v\n", td, ed)
	// 			// if td < 10*time.Millisecond {
	// 			// 	// fmt.Printf("skipping td %v\n", td)
	// 			// 	return // too laggy, bail on sending this event
	// 			// }
	// 			// lsd := w.LastSentDrag
	// 			// w.LastSentDrag = *mde
	// 			// mde.From = lsd.From
	// 			// w.LastDrag = time.Now()
	// 			// // evi = mde // reset interface to us
	// 		}
	// 	}
	// }

	// fmt.Printf("got event type: %v\n", et)
	// first just process all the events straight-up
	w.EventSigs[et].EmitFiltered(w.This, int64(et), evi, func(k ki.Ki) bool {
		if k.IsDeleted() { // destroyed is filtered upstream
			return false
		}
		if evi.IsProcessed() { // someone took care of it
			return false
		}
		_, gi := KiToNode2D(k)
		if gi != nil {
			if w.Popup != nil && (gi.Viewport == nil || gi.Viewport.This != w.Popup) {
				return false
			}
			if evi.OnFocus() {
				if gi.This != w.Focus { // todo: could use GiNodeI interface
					return false
				}
			} else if evi.HasPos() {
				pos := evi.Pos()
				// fmt.Printf("checking pos %v of: %v\n", pos, gi.PathUnique())

				// drag events start with node but can go beyond it..
				mde, ok := evi.(*mouse.DragEvent)
				if ok {
					if w.Dragging == gi.This {
						return true
					} else if w.Dragging != nil {
						return false
					} else {
						if pos.In(gi.WinBBox) {
							w.LastDrag = time.Now()
							w.Dragging = gi.This
							w.LastSentDrag = *mde
							bitflag.Set(&gi.Flag, int(NodeDragging))
							return true
						}
						return false
					}
				} else {
					if w.Dragging == gi.This {
						_, dg := KiToNode2D(w.Dragging)
						if dg != nil {
							bitflag.Clear(&dg.Flag, int(NodeDragging))
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

// process mouse.MoveEvent to generate mouse.FocusEvent events
func (w *Window) GenMouseFocusEvents(mev *mouse.MoveEvent) {
	fe := mouse.FocusEvent{Event: mev.Event}
	pos := mev.Pos()
	ftyp := oswin.MouseFocusEvent
	w.EventSigs[ftyp].EmitFiltered(w.This, int64(ftyp), &fe, func(k ki.Ki) bool {
		if k.IsDeleted() { // destroyed is filtered upstream
			return false
		}
		_, gi := KiToNode2D(k)
		if gi != nil {
			if w.Popup != nil && (gi.Viewport == nil || gi.Viewport.This != w.Popup) {
				return false
			}
			in := pos.In(gi.WinBBox)
			if in {
				if !bitflag.Has(gi.Flag, int(MouseHasEntered)) {
					fe.Action = mouse.Enter
					bitflag.Set(&gi.Flag, int(MouseHasEntered))
					fmt.Printf("sending enter to: %v\n", gi.PathUnique())
					return true // send event
				} else {
					return false // already in
				}
			} else { // mouse not in object
				if bitflag.Has(gi.Flag, int(MouseHasEntered)) {
					fe.Action = mouse.Exit
					bitflag.Clear(&gi.Flag, int(MouseHasEntered))
					fmt.Printf("sending exit to: %v\n", gi.PathUnique())
					return true // send event
				} else {
					return false // already out
				}
			}
		} else {
			// todo: 3D
			return false
		}
	})
}

// process Mouseup events during popup for possible closing of popup -- returns true if popup should be deleted
func (w *Window) PopupMouseUpEvent(evi oswin.Event) bool {
	gii, gi := KiToNode2D(w.Popup)
	if gi == nil {
		return false
	}
	vp := gii.AsViewport2D()
	if vp == nil {
		return false
	}
	// pos := evi.Pos()
	if vp.IsMenu() {
		return true
	}
	return false
}

// start the event loop running -- runs in a separate goroutine
func (w *Window) EventLoop() {
	var lastResize *window.Event
	resizing := false

	for {
		evi := w.OSWin.NextEvent()

		// format := "got %#v\n"
		// if _, ok := evi.(fmt.Stringer); ok {
		// 	format = "got %v\n"
		// }
		// fmt.Printf(format, evi)

		if w.stopEventLoop {
			w.stopEventLoop = false
			fmt.Println("stop event loop")
		}
		if w.DoFullRender {
			fmt.Printf("Doing full render\n")
			w.DoFullRender = false
			w.Viewport.FullRender2DTree()
			w.SetNextFocusItem()
		}
		curPop := w.Popup
		delPop := false // if true, delete this popup after event loop

		// if curPop != nil {
		// 	fmt.Printf("curpop: %v\n", curPop.Name())
		// }

		if rs, ok := evi.(*window.Event); ok {
			if rs.Action == window.Resize {
				resizing = true
				lastResize = rs
				continue
			}
		} else {
			if resizing {
				w.Resize(lastResize.Size.X, lastResize.Size.Y)
				resizing = false
				lastResize = nil
			}
		}

		et := evi.Type()
		if et > oswin.EventTypeN || et < 0 { // we don't handle other types of events here
			continue
		}
		if w.Popup != nil {
			if me, ok := evi.(*mouse.Event); ok {
				if me.Action == mouse.Release {
					delPop = w.PopupMouseUpEvent(evi) // popup before processing event
					// if curPop != nil {
					// 	fmt.Printf("curpop: %v delpop: %v\n", curPop.Name(), delPop)
					// }
				}
			}
		}
		switch e := evi.(type) {
		case *lifecycle.Event:
			if e.To == lifecycle.StageDead {
				fmt.Println("close")
				// oswin.StopBackendEventLoop()
				evi.SetProcessed()
				return // break out of our event loop
			}
		case *paint.Event:
			fmt.Printf("Got paint event!\n")
			w.Viewport.FullRender2DTree()
			w.SetNextFocusItem()
			continue
		case *key.ChordEvent:
			kf := KeyFun(e.ChordString())
			switch kf {
			case KeyFunFocusNext:
				w.SetNextFocusItem()
				e.SetProcessed()
			case KeyFunFocusPrev:
				w.SetPrevFocusItem()
				e.SetProcessed()
			case KeyFunAbort:
				if w.Popup != nil && w.Popup == curPop {
					delPop = true
					e.SetProcessed()
				}
			case KeyFunGoGiEditor:
				GoGiEditorOf(w.Viewport.This)
				e.SetProcessed()
			}
			fmt.Printf("key chord: rune: %v Chord: %v\n", e.Rune, e.ChordString())
		}

		if delPop {
			// fmt.Printf("delpop disconnecting curpop: %v delpop: %v w.Popup %v\n", curPop.Name(), delPop, w.Popup)
			w.DisconnectPopup(curPop)
		}

		if !evi.IsProcessed() {
			w.SendEventSignal(evi)
			if !delPop && et == oswin.MouseMoveEvent {
				w.GenMouseFocusEvents(evi.(*mouse.MoveEvent))
			}
		}

		if delPop {
			// fmt.Printf("delpop poping curpop: %v delpop: %v w.Popup %v\n", curPop.Name(), delPop, w.Popup)
			w.PopPopup(curPop)
		}
	}
	fmt.Println("end of events")
}

// set focus to given item -- returns true if focus changed
func (w *Window) SetFocusItem(k ki.Ki) bool {
	if w.Focus == k {
		return false
	}
	w.UpdateStart()
	// defer w.UpdateEnd()
	if w.Focus != nil {
		gii, gi := KiToNode2D(w.Focus)
		if gi != nil {
			bitflag.Clear(&gi.Flag, int(HasFocus))
			gii.FocusChanged2D(false)
		}
	}
	w.Focus = k
	if k == nil {
		w.UpdateEnd()
		return true
	}
	gii, gi := KiToNode2D(k)
	if gi != nil {
		bitflag.Set(&gi.Flag, int(HasFocus))
		gii.FocusChanged2D(true)
	}
	w.UpdateEnd()
	return true
}

// set the focus on the next item that can accept focus -- returns true if a focus item found
func (w *Window) SetNextFocusItem() bool {
	gotFocus := false
	focusNext := false // get the next guy
	if w.Focus == nil {
		focusNext = true
	}

	focRoot := w.Viewport.This
	if w.Popup != nil {
		focRoot = w.Popup
	}

	for i := 0; i < 2; i++ {
		focRoot.FuncDownMeFirst(0, w, func(k ki.Ki, level int, d interface{}) bool {
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
			if w.Focus == k { // current focus can be a non-can-focus item
				focusNext = true
				return true
			}
			if !bitflag.Has(gi.Flag, int(CanFocus)) || gi.VpBBox.Empty() {
				return true
			}
			if focusNext {
				w.SetFocusItem(k)
				gotFocus = true
				return false // done
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

// set the focus on the previous item
func (w *Window) SetPrevFocusItem() bool {
	if w.Focus == nil { // must have a current item here
		w.SetNextFocusItem()
		return false
	}

	gotFocus := false
	var prevItem ki.Ki

	focRoot := w.Viewport.This
	if w.Popup != nil {
		focRoot = w.Popup
	}

	focRoot.FuncDownMeFirst(0, w, func(k ki.Ki, level int, d interface{}) bool {
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
		if w.Focus == k {
			gotFocus = true
			return false
		}
		if !bitflag.Has(gi.Flag, int(CanFocus)) || gi.VpBBox.Empty() {
			return true
		}
		prevItem = k
		return true
	})
	if gotFocus && prevItem != nil {
		w.SetFocusItem(prevItem)
	} else {
		w.SetNextFocusItem()
	}
	return true
}

// push current popup onto stack and set new popup
func (w *Window) PushPopup(pop ki.Ki) {
	if w.PopupStack == nil {
		w.PopupStack = make([]ki.Ki, 0, 50)
	}
	pop.SetParent(w.This) // popup has parent as window -- draws directly in to assoc vp
	w.PopupStack = append(w.PopupStack, w.Popup)
	w.Popup = pop
	_, gi := KiToNode2D(pop)
	if gi != nil {
		gi.FullRender2DTree()
	}
	w.PushFocus(pop)
}

// disconnect given popup -- typically the current one
func (w *Window) DisconnectPopup(pop ki.Ki) {
	w.DisconnectNode(pop)
	pop.SetParent(nil) // don't redraw the popup anymore
	w.Viewport.DrawIntoWindow()
}

// close given popup -- must be the current one -- returns false if not
func (w *Window) ClosePopup(pop ki.Ki) bool {
	if pop != w.Popup {
		return false
	}
	w.DisconnectPopup(pop)
	w.PopPopup(pop)
	return true
}

// pop current popup off the popup stack and set to current popup
func (w *Window) PopPopup(pop ki.Ki) {
	gii, ok := pop.(Node2D)
	if ok {
		pvp := gii.AsViewport2D()
		if pvp != nil {
			pvp.DeletePopup()
		}
	}
	sz := len(w.PopupStack)
	if w.Popup == pop {
		if w.PopupStack == nil || sz == 0 {
			w.Popup = nil
		} else {
			w.Popup = w.PopupStack[sz-1]
			w.PopupStack = w.PopupStack[:sz-1]
		}
		w.PopFocus()
	} else {
		for i := sz - 1; i >= 0; i-- {
			pp := w.PopupStack[i]
			if pp == pop {
				w.PopupStack = w.PopupStack[:i+copy(w.PopupStack[i:], w.PopupStack[i+1:])]
				break
			}
		}
		// do nothing
	}
	w.FullUpdate()
}

// push current focus onto stack and set new focus
func (w *Window) PushFocus(p ki.Ki) {
	if w.FocusStack == nil {
		w.FocusStack = make([]ki.Ki, 0, 50)
	}
	w.FocusStack = append(w.FocusStack, w.Focus)
	w.Focus = p
	w.SetNextFocusItem()
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
