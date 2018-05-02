// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/goki/goki/gi/oswin"
	"github.com/goki/goki/gi/oswin/key"
	"github.com/goki/goki/gi/oswin/lifecycle"
	"github.com/goki/goki/gi/oswin/mouse"
	"github.com/goki/goki/gi/oswin/paint"
	"github.com/goki/goki/gi/oswin/window"
	"github.com/goki/goki/ki"
	"github.com/goki/goki/ki/bitflag"
	"github.com/goki/goki/ki/kit"
	"github.com/goki/prof"

	"time"
)

// EventSkipLagMSec is the number of milliseconds of lag between the time the
// event was sent to the time it is being processed, above which a repeated
// event type (scroll, drag, resize) is skipped
var EventSkipLagMSec = 50

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
	NextPopup     ki.Ki                       `json:"-" xml:"-" desc:"this popup will be pushed at the end of the current event cycle"`
	stopEventLoop bool                        `json:"-" xml:"-" desc:"signal for communicating all user events (mouse, keyboard, etc)"`
	DoFullRender  bool                        `json:"-" xml:"-" desc:"triggers a full re-render of the window within the event loop -- cleared once done"`
}

var KiT_Window = kit.Types.AddType(&Window{}, nil)

func (n *Window) New() ki.Ki { return &Window{} }

// NewWindow creates a new window with given name and options
func NewWindow(name string, opts *oswin.NewWindowOptions) *Window {
	Init() // overall gogi system initialization
	win := &Window{}
	win.InitName(win, name)
	win.SetOnlySelfUpdate() // has its own FlushImage update logic
	var err error
	win.OSWin, err = oswin.TheApp.NewWindow(opts)
	if err != nil {
		fmt.Printf("GoGi NewWindow error: %v \n", err)
		return nil
	}
	win.WinTex, err = oswin.TheApp.NewTexture(win.OSWin, opts.Size) // note size will be in dots
	if err != nil {
		fmt.Printf("GoGi NewTexture error: %v \n", err)
		return nil
	}
	win.OSWin.SetName(name)
	win.OSWin.SetParent(win.This)
	win.NodeSig.Connect(win.This, SignalWindowFlush)
	return win
}

// NewWindow2D creates a new standard 2D window with given name and sizing,
// with default positioning, and initializes a 2D viewport within it --
// stdPixels means use standardized "pixel" units for the display size (96 per
// inch), not the actual underlying raw display dot pixels
func NewWindow2D(name string, width, height int, stdPixels bool) *Window {
	opts := &oswin.NewWindowOptions{
		Title: name, Size: image.Point{width, height}, StdPixels: stdPixels,
	}
	win := NewWindow(name, opts)
	if win == nil {
		return nil
	}
	vp := NewViewport2D(width, height)
	vp.SetName("WinVp")
	win.AddChild(vp)
	win.Viewport = vp
	return win
}

// NewDialogWin creates a new dialog window with given name and sizing
// (assumed to be in raw dots), without setting its main viewport -- user
// should do win.AddChild(vp); win.Viewport = vp to set their own viewport
func NewDialogWin(name string, width, height int, modal bool) *Window {
	opts := &oswin.NewWindowOptions{
		Title: name, Size: image.Point{width, height}, StdPixels: false,
	}
	opts.SetDialog()
	if modal {
		opts.SetModal()
	}
	win := NewWindow(name, opts)
	if win == nil {
		return nil
	}
	return win
}

func (w *Window) StartEventLoop() {
	w.EventLoop()
}

func (w *Window) StartEventLoopNoWait() {
	go w.EventLoop()
}

// Init performs overall initialization of the gogi system: loading prefs, etc
func Init() {
	if Prefs.LogicalDPIScale == 0 {
		Prefs.Defaults()
		Prefs.Load()
		Prefs.Apply()
	}
}

// LogicalDPI returns the current logical dots-per-inch resolution of the
// window, which should be used for most conversion of standard units --
// physical DPI can be found in the Screen
func (w *Window) LogicalDPI() float32 {
	if w.OSWin == nil {
		return 96.0 // null default
	}
	return w.OSWin.LogicalDPI()
}

func (w *Window) WinViewport2D() *Viewport2D {
	vpi := w.ChildByType(KiT_Viewport2D, true, 0)
	vp, _ := vpi.EmbeddedStruct(KiT_Viewport2D).(*Viewport2D)
	return vp
}

func (w *Window) SetSize(sz image.Point) {
	w.OSWin.SetSize(sz)
	// wait for resize event for us to update?
	// w.Resized(sz.X, sz.Y)
}

func (w *Window) Resized(sz image.Point) {
	// if w.WinTex.Size() == sz {
	// 	return
	// }
	// fmt.Printf("resized to: %v\n", sz)
	if w.WinTex != nil {
		w.WinTex.Release()
	}
	w.WinTex, _ = oswin.TheApp.NewTexture(w.OSWin, sz)
	w.Viewport.Resize(sz.X, sz.Y)
}

// FullReRender can be called to trigger a full re-render of the window
func (w *Window) FullReRender() {
	if w.Viewport == nil {
		return
	}
	pdpi := w.OSWin.PhysicalDPI()
	dpi := oswin.LogicalFmPhysicalDPI(pdpi)
	w.OSWin.SetLogicalDPI(dpi)
	w.Viewport.FullRender2DTree()
	if w.Focus == nil {
		w.SetNextFocusItem()
	}
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
	// if w.Viewport.Fill {
	// 	w.WinTex.Fill(w.Viewport.OSImage.Bounds(), &w.Viewport.Style.Background.Color, oswin.Src)
	// }
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
	w.OSWin.Copy(image.ZP, w.WinTex, w.WinTex.Bounds(), oswin.Src, nil)
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

// Zoom -- positive steps increase logical DPI, negative steps decrease it
func (w *Window) ZoomDPI(steps int) {
	pdpi := w.OSWin.PhysicalDPI()
	dpi := oswin.LogicalFmPhysicalDPI(pdpi)
	dpi += float32(6 * steps)
	oswin.LogicalDPIScale = dpi / pdpi
	w.OSWin.SetLogicalDPI(dpi) // will also be updated by resize events
	fmt.Printf("LogicalDPI now: %v  PhysicalDPI: %v  Scale: %v\n", dpi, pdpi, oswin.LogicalDPIScale)
	w.FullReRender()
}

// ReceiveEventType adds a Signal connection for given event type to given receiver
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
			if gi.IsInactive() && !bitflag.Has(gi.Flag, int(InactiveEvents)) {
				return false
			}
			if gi.This == w.Popup || gi.This == w.Viewport.This { // do this last
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
				_, ok := evi.(*mouse.DragEvent)
				if ok {
					if w.Dragging == gi.This {
						return true
					} else if w.Dragging != nil {
						return false
					} else {
						if pos.In(gi.WinBBox) {
							w.Dragging = gi.This
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

// DeletePopupMenu returns true if the given popup item should be deleted
func (w *Window) DeletePopupMenu(pop ki.Ki) bool {
	if !PopupIsMenu(pop) {
		return false
	}
	if w.NextPopup != nil && PopupIsMenu(w.NextPopup) { // poping up another menu
		return false
	}
	return true
}

// start the event loop running -- runs in a separate goroutine
func (w *Window) EventLoop() {
	var skippedResize *window.Event

	lastEt := oswin.EventTypeN
	var skipDelta image.Point
	lastSkipped := false

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
			// fmt.Printf("Doing full render\n")
			w.DoFullRender = false
			w.FullReRender()
		}
		delPop := false // if true, delete this popup after event loop

		et := evi.Type()
		if et > oswin.EventTypeN || et < 0 { // we don't handle other types of events here
			continue
		}

		nw := time.Now()
		lag := nw.Sub(evi.Time())
		lagMs := int(lag / time.Millisecond)
		// fmt.Printf("et %v lag %v\n", et, lag)

		if et == lastEt || lastEt == oswin.WindowResizeEvent && et == oswin.PaintEvent {
			switch et {
			case oswin.MouseScrollEvent:
				me := evi.(*mouse.ScrollEvent)
				if lagMs > EventSkipLagMSec {
					// fmt.Printf("skipped et %v lag %v\n", et, lag)
					if !lastSkipped {
						skipDelta = me.Delta
					} else {
						skipDelta = skipDelta.Add(me.Delta)
					}
					lastSkipped = true
					continue
				} else {
					if lastSkipped {
						me.Delta = skipDelta.Add(me.Delta)
					}
					lastSkipped = false
				}
			case oswin.MouseDragEvent:
				me := evi.(*mouse.DragEvent)
				if lagMs > EventSkipLagMSec {
					// fmt.Printf("skipped et %v lag %v\n", et, lag)
					if !lastSkipped {
						skipDelta = me.From
					}
					lastSkipped = true
					continue
				} else {
					if lastSkipped {
						me.From = skipDelta
					}
					lastSkipped = false
				}
			case oswin.WindowResizeEvent:
				we := evi.(*window.Event)
				// fmt.Printf("resize %v\n", we.Size)
				if lagMs > EventSkipLagMSec {
					// fmt.Printf("skipped et %v lag %v\n", et, lag)
					lastSkipped = true
					skippedResize = we
					continue
				} else {
					w.Resized(we.Size)
					w.DoFullRender = true
					lastSkipped = false
					skippedResize = nil
					continue
				}
			case oswin.PaintEvent:
				// fmt.Printf("skipped paint\n")
				continue
			}
		}
		lastSkipped = false
		lastEt = et

		if skippedResize != nil {
			w.Resized(skippedResize.Size)
			w.DoFullRender = true
			skippedResize = nil
		}

		switch e := evi.(type) {
		case *lifecycle.Event:
			if e.To == lifecycle.StageDead {
				// fmt.Println("close")
				evi.SetProcessed()
				break
			} else {
				// fmt.Printf("lifecycle from: %v to %v\n", e.From, e.To)
				// if e.Crosses(lifecycle.StageFocused) == lifecycle.CrossOff {
				// }
				evi.SetProcessed()
			}
		case *paint.Event:
			w.FullReRender()
			// fmt.Println("doing paint")
			continue
		case *window.Event:
			if e.Action == window.Open || e.Action == window.Resize {
				// fmt.Printf("doing resize for action %v \n", e.Action)
				w.Resized(e.Size)
				w.FullReRender()
			}
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
				if w.Popup != nil {
					if PopupIsMenu(w.Popup) {
						delPop = true
						e.SetProcessed()
					}
				}
			case KeyFunAccept:
				if w.Popup != nil {
					if PopupIsMenu(w.Popup) {
						delPop = true
					}
				}
			case KeyFunGoGiEditor:
				GoGiEditorOf(w.Viewport.This)
				e.SetProcessed()
			case KeyFunZoomIn:
				w.ZoomDPI(1)
				e.SetProcessed()
			case KeyFunZoomOut:
				w.ZoomDPI(-1)
				e.SetProcessed()
			case KeyFunPrefs:
				Prefs.Edit()
				e.SetProcessed()
			case KeyFunRefresh:
				w.FullReRender()
				e.SetProcessed()
			}
			if !e.IsProcessed() {
				cs := e.ChordString()
				switch cs { // some other random special codes, during dev..
				case "Control+Alt+R":
					if prof.Profiling {
						w.EndTargProfile()
						w.EndCPUMemProfile()
					} else {
						w.StartTargProfile()
						w.StartCPUMemProfile()
					}
					e.SetProcessed()
				case "Control+Alt+F":
					w.BenchmarkFullRender()
					e.SetProcessed()
				case "Control+Alt+G":
					w.BenchmarkReRender()
					e.SetProcessed()
				}
			}
			// fmt.Printf("key chord: rune: %v Chord: %v\n", e.Rune, e.ChordString())
		}

		if !evi.IsProcessed() {
			w.SendEventSignal(evi)
			if !delPop && et == oswin.MouseMoveEvent {
				w.GenMouseFocusEvents(evi.(*mouse.MoveEvent))
			}
		}

		if w.Popup != nil {
			if me, ok := evi.(*mouse.Event); ok {
				if me.Action == mouse.Release {
					if w.DeletePopupMenu(w.Popup) {
						delPop = true
					}
				}
			}
		}

		if delPop {
			w.PopPopup(w.Popup)
		}

		if w.NextPopup != nil {
			w.PushPopup(w.NextPopup)
			w.NextPopup = nil
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

///////////////////////////////////////////////////////
// Profiling and Benchmarking, controlled by hot-keys (or buttons :)

func (w *Window) StartCPUMemProfile() {
	fmt.Println("Starting Std CPU / Mem Profiling")
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
}

func (w *Window) EndCPUMemProfile() {
	fmt.Println("Ending Std CPU / Mem Profiling")
	pprof.StopCPUProfile()
	f, err := os.Create("mem.prof")
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
	f.Close()
}

// start targeted profiling using prof package
func (w *Window) StartTargProfile() {
	nn := 0
	w.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		nn++
		return true
	})
	fmt.Printf("Starting Targeted Profiling, window has %v nodes\n", nn)
	prof.Reset()
	prof.Profiling = true
}

// end targeted profiling and print report
func (w *Window) EndTargProfile() {
	prof.Report(time.Millisecond)
	prof.Profiling = false
}

// run benchmark of 50 full re-renders, report targeted profile results
func (w *Window) BenchmarkFullRender() {
	fmt.Println("Starting BenchmarkFullRender")
	w.StartCPUMemProfile()
	w.StartTargProfile()
	ts := time.Now()
	n := 50
	for i := 0; i < n; i++ {
		w.Viewport.FullRender2DTree()
	}
	td := time.Now().Sub(ts)
	fmt.Printf("Time for %v Re-Renders: %12.2f s\n", n, float64(td)/float64(time.Second))
	w.EndTargProfile()
	w.EndCPUMemProfile()
}

// run benchmark of 50 just-re-renders, not full rebuilds
func (w *Window) BenchmarkReRender() {
	fmt.Println("Starting BenchmarkReRender")
	w.StartTargProfile()
	ts := time.Now()
	n := 50
	for i := 0; i < n; i++ {
		w.Viewport.Render2DTree()
	}
	td := time.Now().Sub(ts)
	fmt.Printf("Time for %v Re-Renders: %12.2f s\n", n, float64(td)/float64(time.Second))
	w.EndTargProfile()
}
