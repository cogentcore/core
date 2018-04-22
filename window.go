// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

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
	"github.com/rcoreilly/prof"

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
	FontLibrary.InitFontPaths("/Library/Fonts")
	win := &Window{}
	win.InitName(win, name)
	win.SetOnlySelfUpdate() // has its own FlushImage update logic
	var err error
	sz := image.Point{width, height}
	winDPI := float32(96.0)
	if oswin.TheApp.NScreens() > 0 {
		sc := oswin.TheApp.Screen(0)
		winDPI = float32(sc.LogicalDPI)
		fmt.Printf("screen logical dpi is: %v geom %v phys size: %v dpr: %v\n",
			winDPI, sc.Geometry, sc.PhysicalSize, sc.DevicePixelRatio)
	}
	if stdPixels {
		unctx := units.Context{}
		unctx.Defaults()
		unctx.DPI = winDPI
		sz.X = int(unctx.ToDots(float32(width), units.Px))
		sz.Y = int(unctx.ToDots(float32(height), units.Px))
	}
	win.OSWin, err = oswin.TheApp.NewWindow(&oswin.NewWindowOptions{
		Title: name, Width: sz.X, Height: sz.Y,
	})
	if err != nil {
		fmt.Printf("GoGi NewWindow error: %v \n", err)
		return nil
	}
	win.WinTex, err = oswin.TheApp.NewTexture(win.OSWin, sz)
	if err != nil {
		fmt.Printf("GoGi NewTexture error: %v \n", err)
		return nil
	}
	win.OSWin.SetName(name)
	win.OSWin.SetLogicalDPI(float32(winDPI)) // will also be updated by resize events
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

// LogicalDPI returns the current logical dots-per-inch resolution of the
// window, which should be used for most conversion of standard units --
// physical DPI can be found in the Screen
func (w *Window) LogicalDPI() float32 {
	if w.OSWin == nil {
		return 96.0 // null default
	}
	return float32(w.OSWin.LogicalDPI())
}

func (w *Window) WinViewport2D() *Viewport2D {
	vpi := w.ChildByType(KiT_Viewport2D, true, 0)
	vp, _ := vpi.EmbeddedStruct(KiT_Viewport2D).(*Viewport2D)
	return vp
}

func (w *Window) Resize(width, height int) {
	if w.WinTex != nil {
		w.WinTex.Release()
	}
	w.WinTex, _ = oswin.TheApp.NewTexture(w.OSWin, image.Point{width, height})
	w.Viewport.Resize(width, height)
}

// UpdateVpRegion updates pixels for one viewport region on the screen, using
// vpBBox bounding box for the viewport, and winBBox bounding box for the
// window (which should not be empty given the overall logic driving updates)
// -- the Window has a its OnlySelfUpdate logic for determining when to flush
// changes to the underlying OSWindow -- wrap updates in win.UpdateStart /
// win.UpdateEnd to actually flush the updates to be visible
func (w *Window) UpdateVpRegion(vp *Viewport2D, vpBBox, winBBox image.Rectangle) {
	pr := prof.Start("win.UpdateVpRegion")
	w.WinTex.Upload(winBBox.Min, vp.OSImage, vpBBox)
	pr.End()
}

// UpdateVpPixels updates pixels for one viewport region on the screen, in its entirety
func (w *Window) UpdateFullVpRegion(vp *Viewport2D, vpBBox, winBBox image.Rectangle) {
	pr := prof.Start("win.UpdateFullVpRegion")
	w.WinTex.Upload(winBBox.Min, vp.OSImage, vp.OSImage.Bounds())
	pr.End()
}

// UpdateVpRegionFromMain basically clears the region where the vp would show
// up, from the main
func (w *Window) UpdateVpRegionFromMain(winBBox image.Rectangle) {
	pr := prof.Start("win.UpdateVpRegionFromMain")
	w.WinTex.Upload(winBBox.Min, w.Viewport.OSImage, winBBox)
	pr.End()
}

// FullUpdate does a complete update of window pixels -- grab pixels from all
// the different active viewports
func (w *Window) FullUpdate() {
	pr := prof.Start("win.FullUpdate")
	updt := w.UpdateStart()
	if Render2DTrace {
		fmt.Printf("Window: %v uploading full Vp, image bound: %v, bounds: %v\n", w.PathUnique(), w.Viewport.OSImage.Bounds(), w.WinTex.Bounds())
	}
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
	pr.End()
	w.UpdateEnd(updt) // drives the flush
}

func (w *Window) Publish() {
	// fmt.Printf("Win %v doing publish\n", w.Nm)
	pr := prof.Start("win.Publish.Copy")
	w.OSWin.Copy(image.ZP, w.WinTex, w.WinTex.Bounds(), oswin.Over, nil)
	pr.End()
	pr2 := prof.Start("win.Publish.Publish")
	w.OSWin.Publish()
	pr2.End()
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

var cpuprofile *string
var memprofile *string

func (w *Window) StartProfile() {
	if cpuprofile != nil {
		return
	}

	// to read profile: go tool pprof -http=localhost:5555 cpu.prof
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
	profFlag := flag.Bool("prof", false, "turn on targeted profiling")
	flag.Parse()
	prof.Profiling = *profFlag
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
}

func (w *Window) EndProfile() {
	prof.Report(time.Millisecond)

	if *cpuprofile != "" {
		pprof.StopCPUProfile()
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
}

func (w *Window) StartEventLoop() {
	// w.DoFullRender = true
	// var wg sync.WaitGroup
	// wg.Add(1)
	w.EventLoop()
	// wg.Wait()
	fmt.Printf("stop event loop\n")
}

func (w *Window) StartEventLoopNoWait() {
	// w.DoFullRender = true
	w.EventLoop()
}

// IsInScope returns true if the given object is in scope for receiving events
func (w *Window) IsInScope(gii Node2D, gi *Node2DBase) bool {
	if w.Popup == nil {
		return true
	}
	if gi.This == w.Popup {
		return true
	}
	if gi.Viewport == nil {
		return false
	}
	if gi.Viewport.This == w.Popup {
		return true
	}
	return false
}

// SendEventSignal sends given event signal to all receivers that want it --
// note that because there is a different EventSig for each event type, we are
// ONLY looking at nodes that have registered to receive that type of event --
// the further filtering is just to ensure that they are in the right position
// to receive the event (focus, popup filtering, etc)
func (w *Window) SendEventSignal(evi oswin.Event) {
	if evi.IsProcessed() { // someone took care of it
		return
	}
	et := evi.Type()
	if et > oswin.EventTypeN || et < 0 {
		return // can't handle other types of events here due to EventSigs[et] size
	}

	// always process the popup last as a second pass -- this is index if found
	var popupCon *ki.Connection

	// fmt.Printf("got event type: %v\n", et)
	// first just process all the events straight-up
	w.EventSigs[et].EmitFiltered(w.This, int64(et), evi, func(k ki.Ki, idx int, con *ki.Connection) bool {
		if k.IsDeleted() { // destroyed is filtered upstream
			return false
		}
		if evi.IsProcessed() { // someone took care of it
			return false
		}
		gii, gi := KiToNode2D(k)
		if gi != nil {
			if gi.This == w.Popup { // do this last
				popupCon = con
				return false
			}
			if !w.IsInScope(gii, gi) { // no
				return false
			}
			if evi.OnFocus() && !gii.HasFocus2D() {
				return false
			} else if evi.HasPos() {
				pos := evi.Pos()
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
						return false
					}
				}
			}
		} else {
			// todo: get a 3D
			return false
		}
		return true
	})

	// send events to the popup last so e.g., dialog can do Accept / Cancel
	// after other events have been processed
	if popupCon != nil {
		popupCon.Func(popupCon.Recv, w.Popup, int64(et), evi)
	}

}

// process mouse.MoveEvent to generate mouse.FocusEvent events
func (w *Window) GenMouseFocusEvents(mev *mouse.MoveEvent) {
	fe := mouse.FocusEvent{Event: mev.Event}
	pos := mev.Pos()
	ftyp := oswin.MouseFocusEvent
	updated := false
	updt := false
	w.EventSigs[ftyp].EmitFiltered(w.This, int64(ftyp), &fe, func(k ki.Ki, idx int, con *ki.Connection) bool {
		if k.IsDeleted() { // destroyed is filtered upstream
			return false
		}
		gii, gi := KiToNode2D(k)
		if gi != nil {
			if !w.IsInScope(gii, gi) { // no
				return false
			}
			in := pos.In(gi.WinBBox)
			if in {
				if !bitflag.Has(gi.Flag, int(MouseHasEntered)) {
					fe.Action = mouse.Enter
					bitflag.Set(&gi.Flag, int(MouseHasEntered))
					if !updated {
						updt = w.UpdateStart()
						updated = true
					}
					return true // send event
				} else {
					return false // already in
				}
			} else { // mouse not in object
				if bitflag.Has(gi.Flag, int(MouseHasEntered)) {
					fe.Action = mouse.Exit
					bitflag.Clear(&gi.Flag, int(MouseHasEntered))
					if !updated {
						updt = w.UpdateStart()
						updated = true
					}
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
	if updated {
		w.UpdateEnd(updt)
	}
}

// PopupIsMenu returns true if the given popup item is a menu
func PopupIsMenu(pop ki.Ki) bool {
	gii, gi := KiToNode2D(pop)
	if gi == nil {
		return false
	}
	vp := gii.AsViewport2D()
	if vp == nil {
		return false
	}
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
					if PopupIsMenu(w.Popup) { // remove menus automatically after mouse release
						delPop = true
					}
				}
			}
		}
		switch e := evi.(type) {
		case *lifecycle.Event:
			if e.To == lifecycle.StageDead {
				fmt.Println("close")
				evi.SetProcessed()
				break
			} else {
				// fmt.Printf("lifecycle from: %v to %v\n", e.From, e.To)
				if e.Crosses(lifecycle.StageFocused) == lifecycle.CrossOff {
					w.EndProfile()
				}
				evi.SetProcessed()
			}
		case *paint.Event:
			w.Viewport.FullRender2DTree()
			w.SetNextFocusItem()
			w.StartProfile()
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
					if PopupIsMenu(w.Popup) {
						delPop = true
						e.SetProcessed()
					}
				}
			case KeyFunAccept:
				if w.Popup != nil && w.Popup == curPop {
					if PopupIsMenu(w.Popup) {
						delPop = true
					}
				}
			case KeyFunGoGiEditor:
				GoGiEditorOf(w.Viewport.This)
				e.SetProcessed()
			}
			// fmt.Printf("key chord: rune: %v Chord: %v\n", e.Rune, e.ChordString())
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

// ClearNonFocus clears the focus of any non-w.Focus item -- sometimes can get
// off
func (w *Window) ClearNonFocus() {
	focRoot := w.Viewport.This
	if w.Popup != nil {
		focRoot = w.Popup
	}

	updated := false
	updt := false

	focRoot.FuncDownMeFirst(0, w, func(k ki.Ki, level int, d interface{}) bool {
		if k == focRoot { // skip top-level
			return true
		}
		// todo: see about 3D guys
		gii, gi := KiToNode2D(k)
		if gi == nil {
			return true
		}
		if gi.Paint.Off { // off below this
			return false
		}
		if w.Focus == k {
			return true
		}
		if gi.HasFocus() {
			if !updated {
				updated = true
				updt = w.UpdateStart()
			}
			bitflag.Clear(&gi.Flag, int(HasFocus))
			gii.FocusChanged2D(false)
		}
		return true
	})
	if updated {
		w.UpdateEnd(updt)
	}
}

// set focus to given item -- returns true if focus changed
func (w *Window) SetFocusItem(k ki.Ki) bool {
	if w.Focus == k {
		return false
	}
	updt := w.UpdateStart()
	if w.Focus != nil {
		gii, gi := KiToNode2D(w.Focus)
		if gi != nil {
			bitflag.Clear(&gi.Flag, int(HasFocus))
			gii.FocusChanged2D(false)
		}
	}
	w.Focus = k
	if k == nil {
		w.UpdateEnd(updt)
		return true
	}
	gii, gi := KiToNode2D(k)
	if gi != nil {
		bitflag.Set(&gi.Flag, int(HasFocus))
		gii.FocusChanged2D(true)
	}
	w.ClearNonFocus()
	w.UpdateEnd(updt)
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
			if !gi.CanFocus() || gi.VpBBox.Empty() {
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
		if !gi.CanFocus() || gi.VpBBox.Empty() {
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
