// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"sync"
	"time"

	"goki.dev/girl/states"
	"goki.dev/goosi"
	"goki.dev/goosi/dnd"
	"goki.dev/goosi/key"
	"goki.dev/goosi/mimedata"
	"goki.dev/goosi/mouse"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// EventPris for different queues of event signals, processed in priority order
type EventPris int32 //enums:enum

const (
	// HiPri = high priority -- event receivers processed first -- can be used
	// to override default behavior
	HiPri EventPris = iota

	// RegPri = default regular priority -- most should be here
	RegPri

	// LowPri = low priority -- processed last -- typically for containers /
	// dialogs etc
	LowPri

	// LowRawPri = unfiltered (raw) low priority -- ignores whether the event
	// was already processed.
	LowRawPri
)

const (
	// Popups means include popups
	Popups = true

	// NoPopups means exclude popups
	NoPopups = false
)

// EventMgr is an event manager that handles incoming events for a
// MainStage object (Window, Dialog, Sheet).  It distributes events
// to a Scene based on position or focus, and deals with more complex
// cases such as dragging, drag-n-drop, and hovering.
type EventMgr struct {

	// Stage is the owning MainStage that we manage events for
	Main *MainStage

	// mutex that protects timer variable updates (e.g., hover AfterFunc's)
	TimerMu sync.Mutex `desc:"mutex that protects timer variable updates (e.g., hover AfterFunc's)"`

	// node receiving mouse dragging events -- not for DND but things like sliders -- anchor to same
	Dragging Widget `desc:"node receiving mouse dragging events -- not for DND but things like sliders -- anchor to same"`

	// node receiving mouse scrolling events -- anchor to same
	Scrolling Widget `desc:"node receiving mouse scrolling events -- anchor to same"`

	// stage of DND process
	DNDStage DNDStages `desc:"stage of DND process"`

	// drag-n-drop data -- if non-nil, then DND is taking place
	DNDData mimedata.Mimes `desc:"drag-n-drop data -- if non-nil, then DND is taking place"`

	// drag-n-drop source node
	DNDSource Widget `desc:"drag-n-drop source node"`

	// final event for DND which is sent if a finalize is received
	DNDFinalEvent *dnd.Event `desc:"final event for DND which is sent if a finalize is received"`

	// modifier in place at time of drop event (DropMove or DropCopy)
	DNDDropMod dnd.DropMods `desc:"modifier in place at time of drop event (DropMove or DropCopy)"`

	// node receiving keyboard events -- use SetFocus, CurFocus
	Focus Widget `desc:"node receiving keyboard events -- use SetFocus, CurFocus"`

	// mutex that protects focus updating
	FocusMu sync.RWMutex `desc:"mutex that protects focus updating"`

	// stack of focus
	FocusStack []Widget `desc:"stack of focus"`

	// node to focus on at start when no other focus has been set yet -- use SetStartFocus
	StartFocus Widget `desc:"node to focus on at start when no other focus has been set yet -- use SetStartFocus"`

	// Last modifier key bits from most recent Mouse, Keyboard events
	LastModBits goosi.Modifiers `desc:"Last modifier key bits from most recent Mouse, Keyboard events"`

	// Last Select Mode from most recent Mouse, Keyboard events
	LastSelMode mouse.SelectModes `desc:"Last Select Mode from most recent Mouse, Keyboard events"`

	// Last mouse position from most recent Mouse events
	LastMousePos image.Point `desc:"Last mouse position from most recent Mouse events"`

	// change in position accumulated from skipped-over laggy mouse move events
	LagSkipDeltaPos image.Point `desc:"change in position accumulated from skipped-over laggy mouse move events"`

	// true if last event was skipped due to lag
	LagLastSkipped  bool `desc:"true if last event was skipped due to lag"`
	startDrag       *mouse.Event
	dragStarted     bool
	startDND        *mouse.Event
	dndStarted      bool
	startHover      *mouse.Event
	curHover        *mouse.Event
	hoverStarted    bool
	hoverTimer      *time.Timer
	startDNDHover   *mouse.Event
	curDNDHover     *mouse.Event
	dndHoverStarted bool
	dndHoverTimer   *time.Timer
}

// WinEventRecv is used to hold info about widgets receiving event signals to
// given function, used for sorting and delayed sending.
type WinEventRecv struct {
	Recv Widget
	Func func()
	Data int
}

// Set sets the recv and fun
func (we *WinEventRecv) Set(r Widget, f func()) {
	we.Recv = r
	we.Func = f
}

// Call calls the function on the recv with the args
func (we *WinEventRecv) Call(send Widget, sig int64) {
	if EventTrace {
		// fmt.Printf("calling event: %v method on: %v\n", we.Recv.Path())
	}
	we.Func() // we.Recv, send, sig, data)
}

type WinEventRecvList []WinEventRecv

func (wl *WinEventRecvList) Add(recv Widget, fun func()) {
	rr := WinEventRecv{Recv: recv, Func: fun}
	*wl = append(*wl, rr)
}

func (wl *WinEventRecvList) AddDepth(recv Widget, fun func(), par Widget) {
	wl.Add(recv, fun)
}

// MainStageMgr returns the MainStageMgr for our Main Stage
func (em *EventMgr) MainStageMgr() *MainStageMgr {
	if em.Main == nil {
		return nil
	}
	return em.Main.StageMgr
}

// RenderWin returns the overall render window, if possible.
func (em *EventMgr) RenderWin() *RenderWin {
	mgr := em.MainStageMgr()
	if mgr == nil {
		return nil
	}
	return mgr.RenderWin
}

///////////////////////////////////////////////////////////////////////
// 	HandleEvent

func (em *EventMgr) HandleEvent(sc *Scene, evi goosi.Event) {
	et := evi.Type()
	if et > goosi.EventTypesN || et < 0 {
		return // can't handle other types of events here
	}
	// fmt.Printf("got event type: %v: %v\n", et, evi)

	switch {
	case evi.HasPos():
		em.HandlePosEvent(sc, evi)
	case evi.OnFocus():
		em.HandleFocusEvent(sc, evi)
	default:
		em.HandleOtherEvent(sc, evi)
	}
}

func (em *EventMgr) SetRenderWinFocusActive(active bool) {
	win := em.RenderWin()
	if win == nil {
		return
	}
	win.SetFocusActive(active)
}

func (em *EventMgr) HandleOtherEvent(sc *Scene, evi goosi.Event) {
}

func (em *EventMgr) HandleFocusEvent(sc *Scene, evi goosi.Event) {
	et := evi.Type()
	foc := em.CurFocus()
	if foc == nil {
		return
	}
	fw := foc.AsWidget()
	if win := em.RenderWin(); win != nil {
		if !win.IsFocusActive() { // reactivate on keyboard input
			win.SetFocusActive(true)
			if EventTrace {
				fmt.Printf("Event: set focus active, was not: %v\n", fw.Path())
			}
			foc.FocusChanged(FocusActive)
		}
	}
	wants := fw.Events.Filter.HasFlag(et)
	if wants {
		fw.Events.Funcs[et].Func() // foc.This(), nil, int64(et), evi)
	}
}

func (em *EventMgr) HandlePosEvent(sc *Scene, evi goosi.Event) {
	pos := evi.LocalPos()
	et := evi.Type()

	// todo: all the stuff about dragging here
	// todo: sc.Decor needs to be processed too!  do that first!

	// note: we don't really have a ki sender -- window is no longer a Ki
	var send Widget

	for pri := HiPri; pri < EventPrisN; pri++ {
		if pri != LowRawPri && evi.IsHandled() { // someone took care of it
			continue
		}

		// we take control of signal process to sort elements by depth, and
		// dispatch to inner-most one first
		rvs := make(WinEventRecvList, 0, 10)

		sc.Frame.WalkPre(func(k ki.Ki) bool {
			wi, wb := AsWidget(k)
			if wb == nil || wb.Is(ki.Deleted) || wb.Is(ki.Destroyed) {
				return ki.Break
			}
			if !wb.PosInBBox(pos) {
				return ki.Break
			}
			wants := wb.Events.Matches(et, pri)
			if wants {
				// fmt.Printf("pri: %s  et: %s   wb: %s\n", pri.String(), et.BitIndexString(), wb.Path())
				rvs.Add(wi, wb.Events.Funcs[et].Func)
			}
			return ki.Continue
		})

		nrv := len(rvs)

		if nrv == 0 {
			continue
		}

		// // deepest first
		// sort.Slice(rvs, func(i, j int) bool {
		// 	return rvs[i].Data > rvs[j].Data
		// })

		// reverse order
		for i := nrv - 1; i >= 0; i-- {
			rr := rvs[i]
			switch evi.Type() {
			case goosi.MouseDragEvent:
				if em.Dragging == nil {
					rr.Recv.SetFlag(true, NodeDragging) // PROVISIONAL!
				}
			}

			// fmt.Printf("proc event type: %v: %v %v\n", et.BitIndexString(), evi, rr.Recv.Path())
			// actually call the thing!
			rr.Call(send, int64(et))

			if pri != LowRawPri && evi.IsHandled() { // someone took care of it
				switch evi.Type() { // only grab events if processed
				case goosi.MouseDragEvent:
					if em.Dragging == nil {
						em.Dragging = rr.Recv
						rr.Recv.SetFlag(true, NodeDragging)
					}
				case goosi.MouseScrollEvent:
					if em.Scrolling == nil {
						em.Scrolling = rr.Recv
					}
				}
				break
			} else {
				switch evi.Type() {
				case goosi.MouseDragEvent:
					if em.Dragging == nil {
						rr.Recv.SetFlag(false, NodeDragging) // clear provisional
					}
				}
			}
		}
	}
}

/*
// SendEventSignalFunc is the inner loop of the SendEventSignal -- needed to deal with
// map iterator locking logic in a cleaner way.  Returns true to continue, false to break
func (em *EventMgr) SendEventSignalFunc(evi goosi.Event, popup bool, rvs *WinEventRecvList, recv Widget, fun func()) bool {
	if !em.Master.IsInScope(recv, popup) {
		return ki.Continue
	}
	nii, ni := AsWidget(recv)
	if ni != nil {
		if evi.OnFocus() {
		}
	}
	top := em.Master.EventTopNode()
	// remainder is done using generic node interface, for 2D and 3D
	_, wb := AsWidget(recv)
	if evi.HasPos() {
		pos := evi.LocalPos()
		switch evi.Type() {
		case goosi.MouseDragEvent:
			if em.Dragging != nil {
				if em.Dragging == wb.This() {
					if EventTrace {
						fmt.Printf("Event: dragging top pri: %v\n", recv.Path())
					}
					rvs.Add(recv, fun, 10000)
					return ki.Break
				} else {
					return ki.Continue
				}
			} else {
				if wb.PosInBBox(pos) {
					rvs.AddDepth(recv, fun, top)
					return ki.Break
				}
				return ki.Continue
			}
		case goosi.MouseScrollEvent:
			if em.Scrolling != nil {
				if em.Scrolling == wb.This() {
					if EventTrace {
						fmt.Printf("Event: scrolling top pri: %v\n", recv.Path())
					}
					rvs.Add(recv, fun, 10000)
				} else {
					return ki.Continue
				}
			} else {
				if wb.PosInBBox(pos) {
					rvs.AddDepth(recv, fun, top)
					return ki.Break
				}
				return ki.Continue
			}
		default:
			if em.Dragging == wb.This() { // dragger always gets it
				if EventTrace {
					fmt.Printf("Event: dragging, non drag top pri: %v\n", recv.Path())
				}
				rvs.Add(recv, fun, 10000) // top priority -- can't steal!
				return ki.Break
			}
			if !wb.PosInBBox(pos) {
				return ki.Continue
			}
		}
	}
	rvs.AddDepth(recv, fun, top)
	return ki.Continue
}
*/

// // SendSig directly calls SendSig from given recv, sender for given event
// // across all priorities.
// func (em *EventMgr) SendSig(recv, sender Widget, evi goosi.Event) {
// 	et := evi.Type()
// 	for pri := HiPri; pri < EventPrisN; pri++ {
// 		em.EventSigs[et][pri].SendSig(recv, sender, int64(et), evi)
// 	}
// }

///////////////////////////////////////////////////////////////////////////
//  Mouse event processing

// MouseEvents processes mouse drag and move events
func (em *EventMgr) MouseEvents(evi goosi.Event) {
	et := evi.Type()
	if et == goosi.MouseDragEvent {
		em.MouseDragEvents(evi)
	} else if et != goosi.KeyEvent { // allow modifier keypress
		em.ResetMouseDrag()
	}

	if et == goosi.MouseMoveEvent {
		em.MouseMoveEvents(evi)
	} else {
		em.ResetMouseMove()
	}

	if et == goosi.MouseButtonEvent {
		me := evi.(*mouse.Event)
		em.LastModBits = me.Mods
		em.LastSelMode = me.SelectMode()
		em.LastMousePos = me.Pos()
	}
	if et == goosi.KeyChordEvent {
		ke := evi.(*key.Event)
		em.LastModBits = ke.Mods
		em.LastSelMode = mouse.SelectModeBits(ke.Mods)
	}
}

// MouseEventReset resets state for "catch" events (Dragging, Scrolling)
func (em *EventMgr) MouseEventReset(evi goosi.Event) {
	et := evi.Type()
	if em.Dragging != nil && et != goosi.MouseDragEvent {
		em.Dragging.SetFlag(false, NodeDragging)
		em.Dragging = nil
	}
	if em.Scrolling != nil && et != goosi.MouseScrollEvent {
		em.Scrolling = nil
	}
}

// MouseDragEvents processes MouseDragEvent to Detect start of drag and DND.
// These require timing and delays, e.g., due to minor wiggles when pressing
// the mouse button
func (em *EventMgr) MouseDragEvents(evi goosi.Event) {
	me := evi.(*mouse.Event)
	em.LastModBits = me.Mods
	em.LastSelMode = me.SelectMode()
	em.LastMousePos = me.Pos()
	now := time.Now()
	if !em.dragStarted {
		if em.startDrag == nil {
			em.startDrag = me
		} else {
			if em.DoInstaDrag(em.startDrag, false) { // !em.Master.CurPopupIsTooltip()) {
				em.dragStarted = true
				em.startDrag = nil
			} else {
				delayMs := int(now.Sub(em.startDrag.Time()) / time.Millisecond)
				if delayMs >= DragStartMSec {
					dst := int(mat32.Hypot(float32(em.startDrag.Where.X-me.Pos().X), float32(em.startDrag.Where.Y-me.Pos().Y)))
					if dst >= DragStartPix {
						em.dragStarted = true
						em.startDrag = nil
					}
				}
			}
		}
	}
	if em.Dragging == nil && !em.dndStarted {
		if em.startDND == nil {
			em.startDND = me
		} else {
			delayMs := int(now.Sub(em.startDND.Time()) / time.Millisecond)
			if delayMs >= DNDStartMSec {
				dst := int(mat32.Hypot(float32(em.startDND.Where.X-me.Pos().X), float32(em.startDND.Where.Y-me.Pos().Y)))
				if dst >= DNDStartPix {
					em.dndStarted = true
					em.DNDStartEvent(em.startDND)
					em.startDND = nil
				}
			}
		}
	} else { // em.dndStarted
		em.TimerMu.Lock()
		if !em.dndHoverStarted {
			em.dndHoverStarted = true
			em.startDNDHover = me
			em.curDNDHover = em.startDNDHover
			em.dndHoverTimer = time.AfterFunc(time.Duration(HoverStartMSec)*time.Millisecond, func() {
				em.TimerMu.Lock()
				hoe := em.curDNDHover
				if hoe != nil {
					// em.TimerMu.Unlock()
					em.SendDNDHoverEvent(hoe)
					// em.TimerMu.Lock()
				}
				em.startDNDHover = nil
				em.curDNDHover = nil
				em.dndHoverTimer = nil
				em.dndHoverStarted = false
				em.TimerMu.Unlock()
			})
		} else {
			dst := int(mat32.Hypot(float32(em.startDNDHover.Where.X-me.Pos().X), float32(em.startDNDHover.Where.Y-me.Pos().Y)))
			if dst > HoverMaxPix {
				em.dndHoverTimer.Stop()
				em.startDNDHover = nil
				em.dndHoverTimer = nil
				em.dndHoverStarted = false
			} else {
				em.curDNDHover = me
			}
		}
		em.TimerMu.Unlock()
	}
	// if we have started dragging but aren't dragging anything, scroll
	if (em.dragStarted || em.dndStarted) && em.Dragging == nil && em.DNDSource == nil {
		scev := mouse.NewScrollEvent(me.Pos(), me.Pos().Sub(me.Start).Mul(-1), me.Mods)
		scev.Init()
		// em.HandleEvent(sc, scev)
	}
}

// ResetMouseDrag resets all the mouse dragging variables after last drag
func (em *EventMgr) ResetMouseDrag() {
	em.dragStarted = false
	em.startDrag = nil
	em.dndStarted = false
	em.startDND = nil

	em.TimerMu.Lock()
	em.dndHoverStarted = false
	em.startDNDHover = nil
	em.curDNDHover = nil
	if em.dndHoverTimer != nil {
		em.dndHoverTimer.Stop()
		em.dndHoverTimer = nil
	}
	em.TimerMu.Unlock()
}

// MouseMoveEvents processes MouseMoveEvent to detect start of hover events.
// These require timing and delays
func (em *EventMgr) MouseMoveEvents(evi goosi.Event) {
	me := evi.(*mouse.Event)
	em.LastModBits = me.Mods
	em.LastSelMode = me.SelectMode()
	em.LastMousePos = me.Pos()
	em.TimerMu.Lock()
	if !em.hoverStarted {
		em.hoverStarted = true
		em.startHover = me
		em.curHover = mouse.NewEventCopy(goosi.MouseHoverEvent, me)
		em.hoverTimer = time.AfterFunc(time.Duration(HoverStartMSec)*time.Millisecond, func() {
			em.TimerMu.Lock()
			hoe := em.curHover
			if hoe != nil {
				// em.TimerMu.Unlock()
				em.SendHoverEvent(hoe) // this attempts to lock focus
				// em.TimerMu.Lock()
			}
			em.startHover = nil
			em.curHover = nil
			em.hoverTimer = nil
			em.hoverStarted = false
			em.TimerMu.Unlock()
		})
	} else {
		dst := int(mat32.Hypot(float32(em.startHover.Where.X-me.Pos().X), float32(em.startHover.Where.Y-me.Pos().Y)))
		if dst > HoverMaxPix {
			em.hoverTimer.Stop()
			// em.Master.DeleteTooltip()
			em.startHover = nil
			em.hoverTimer = nil
			em.hoverStarted = false
		} else {
			em.curHover = mouse.NewEventCopy(goosi.MouseHoverEvent, me)
		}
	}
	em.TimerMu.Unlock()
}

// ResetMouseMove resets all the mouse moving variables after last move
func (em *EventMgr) ResetMouseMove() {
	em.TimerMu.Lock()
	em.hoverStarted = false
	em.startHover = nil
	em.curHover = nil
	if em.hoverTimer != nil {
		em.hoverTimer.Stop()
		em.hoverTimer = nil
	}
	em.TimerMu.Unlock()
}

// GenMouseFocusEvents processes mouse.Event to generate mouse.Event
// events -- returns true if any such events were sent.  If popup is true,
// then only items on popup are in scope, otherwise items NOT on popup are in
// scope (if no popup, everything is in scope).
func (em *EventMgr) GenMouseFocusEvents(mev *mouse.Event, popup bool) bool {
	/*
		fe := mouse.NewEventCopy(goosi.MouseFocusEvent, mev)
		pos := mev.LocalPos()
		ftyp := goosi.MouseFocusEvent
		updated := false
		updt := false
		send := em.Master.EventTopNode()
		for pri := HiPri; pri < EventPrisN; pri++ {
			em.EventSigs[ftyp][pri].EmitFiltered(send, int64(ftyp), &fe, func(k Widget) bool {
				if k.Is(ki.Deleted) { // destroyed is filtered upstream
					return ki.Break
				}
				if !em.Master.IsInScope(k, popup) {
					return ki.Break
				}
				_, ni := AsWidget(k)
				if ni != nil {
					in := ni.PosInBBox(pos)
					if in {
						if !ni.HasFlag(MouseHasEntered) {
							fe.Action = mouse.Enter
							ni.SetFlag(true, MouseHasEntered)
							if !updated {
								updt = em.Master.EventTopUpdateStart()
								updated = true
							}
							return ki.Continue // send event
						} else {
							return ki.Break // already in
						}
					} else { // mouse not in object
						if ni.HasFlag(MouseHasEntered) {
							fe.Action = mouse.Exit
							ni.SetFlag(false, MouseHasEntered)
							if !updated {
								updt = em.Master.EventTopUpdateStart()
								updated = true
							}
							return ki.Continue // send event
						} else {
							return ki.Break // already out
						}
					}
				} else {
					// 3D
					return ki.Break
				}
			})
		}
		if updated {
			em.Master.EventTopUpdateEnd(updt)
		}
		return updated
	*/
	return false
}

// DoInstaDrag tests whether the given mouse DragEvent is on a widget marked
// with InstaDrag
func (em *EventMgr) DoInstaDrag(me *mouse.Event, popup bool) bool {
	/*
		et := me.Type()
		for pri := HiPri; pri < EventPrisN; pri++ {
			esig := &em.EventSigs[et][pri]
			gotOne := false
			esig.ConsFunc(func(recv Widget, fun func()) bool {
				if recv.Is(ki.Deleted) {
					return ki.Continue
				}
				if !em.Master.IsInScope(recv, popup) {
					return ki.Continue
				}
				_, wb := AsWidget(recv)
				if wb != nil {
					pos := me.LocalPos()
					if wb.PosInBBox(pos) {
						if wb.HasFlag(InstaDrag) {
							em.Dragging = wb.This()
							wb.SetFlag(true, NodeDragging)
							gotOne = true
							return ki.Break
						}
					}
				}
				return ki.Continue
			})
			if gotOne {
				return ki.Continue
			}
		}
	*/
	return ki.Break
}

// SendHoverEvent sends mouse hover event, based on last mouse move event
func (em *EventMgr) SendHoverEvent(he *mouse.Event) {
	he.ClearHandled()
	he.Action = mouse.Hover
	// em.HandleEvent(he)
}

//////////////////////////////////////////////////////////////////////
//  Drag-n-Drop = DND

// DNDStages indicates stage of DND process
type DNDStages int32

const (
	// DNDNotStarted = nothing happening
	DNDNotStarted DNDStages = iota

	// DNDStartSent means that the Start event was sent out, but receiver has
	// not yet started the DND on its end by calling StartDragNDrop
	DNDStartSent

	// DNDStarted means that a node called StartDragNDrop
	DNDStarted

	// DNDDropped means that drop event has been sent
	DNDDropped

	DNDStagesN
)

// DNDTrace can be set to true to get a trace of the DND process
var DNDTrace = false

// DNDStartEvent handles drag-n-drop start events.
func (em *EventMgr) DNDStartEvent(e *mouse.Event) {
	de := dnd.NewEvent(dnd.Start, e.Where, e.Mods)
	de.Start = e.Where
	de.StTime = e.GenTime
	de.DefaultMod() // based on current key modifiers
	em.DNDStage = DNDStartSent
	if DNDTrace {
		fmt.Printf("\nDNDStartSent\n")
	}
	// em.HandleEvent(&de)
	// now up to receiver to call StartDragNDrop if they want to..
}

// DNDStart is driven by node responding to start event, actually starts DND
func (em *EventMgr) DNDStart(src Widget, data mimedata.Mimes) {
	em.DNDStage = DNDStarted
	em.DNDSource = src
	em.DNDData = data
	if DNDTrace {
		fmt.Printf("DNDStarted on: %v\n", src.Path())
	}
}

// DNDIsInternalSrc returns true if the source of the DND operation is internal to GoGi
// system -- otherwise it originated from external OS source.
func (em *EventMgr) DNDIsInternalSrc() bool {
	return em.DNDSource != nil
}

// SendDNDHoverEvent sends DND hover event, based on last mouse move event
func (em *EventMgr) SendDNDHoverEvent(e *mouse.Event) {
	if e == nil {
		return
	}
	he := &dnd.Event{}
	he.EventBase = e.EventBase
	he.ClearHandled()
	he.Action = dnd.Hover
	// em.HandleEvent(&he)
}

// SendDNDMoveEvent sends DND move event
func (em *EventMgr) SendDNDMoveEvent(e *mouse.Event) *dnd.Event {
	// todo: when e.Where goes negative, transition to OS DND
	// todo: send move / enter / exit events to anyone listening
	de := &dnd.Event{}
	de.EventBase = e.EventBase
	de.ClearHandled()
	de.DefaultMod() // based on current key modifiers
	de.Action = dnd.Move
	// em.HandleEvent(de)
	// em.GenDNDFocusEvents(de)
	return de
}

// SendDNDDropEvent sends DND drop event -- returns false if drop event was not processed
// in which case the event should be cleared (by the RenderWin)
func (em *EventMgr) SendDNDDropEvent(e *mouse.Event) bool {
	de := &dnd.Event{}
	de.EventBase = e.EventBase
	de.ClearHandled()
	de.DefaultMod()
	de.Action = dnd.DropOnTarget
	de.Data = em.DNDData
	de.Source = em.DNDSource
	em.DNDSource.SetFlag(false, NodeDragging)
	em.Dragging = nil
	em.DNDFinalEvent = de
	em.DNDDropMod = de.Mod
	em.DNDStage = DNDDropped
	if DNDTrace {
		fmt.Printf("DNDDropped\n")
	}
	e.SetHandled()
	// em.HandleEvent(&de)
	return de.IsHandled()
}

// ClearDND clears DND state
func (em *EventMgr) ClearDND() {
	em.DNDStage = DNDNotStarted
	em.DNDSource = nil
	em.DNDData = nil
	em.Dragging = nil
	em.DNDFinalEvent = nil
	if DNDTrace {
		fmt.Printf("DNDCleared\n")
	}
}

/*
// GenDNDFocusEvents processes mouse.Event to generate dnd.FocusEvent
// events -- returns true if any such events were sent.  If popup is true,
// then only items on popup are in scope, otherwise items NOT on popup are in
// scope (if no popup, everything is in scope).  Extra work is done to ensure
// that Exit from prior widget is always sent before Enter to next one.
func (em *EventMgr) GenDNDFocusEvents(mev *dnd.Event, popup bool) bool {
	fe := &dnd.Event{}
	*fe = *mev
	pos := mev.LocalPos()
	ftyp := goosi.DNDFocusEvent

	// first pass is just to get all the ins and outs
	var ins, outs WinEventRecvList

	send := em.Master.EventTopNode()
	for pri := HiPri; pri < EventPrisN; pri++ {
		esig := &em.EventSigs[ftyp][pri]
		esig.ConsFunc(func(recv Widget, fun func()) bool {
			if recv.Is(ki.Deleted) {
				return ki.Continue
			}
			if !em.Master.IsInScope(recv, popup) {
				return ki.Continue
			}
			_, wb := AsWidget(recv)
			if wb != nil {
				in := wb.PosInBBox(pos)
				if in {
					if !wb.HasFlag(DNDHasEntered) {
						wb.SetFlag(true, DNDHasEntered)
						ins.Add(recv, fun, 0)
					}
				} else { // mouse not in object
					if wb.HasFlag(DNDHasEntered) {
						wb.SetFlag(false, DNDHasEntered)
						outs.Add(recv, fun, 0)
					}
				}
			} else {
				// 3D
			}
			return ki.Continue
		})
	}
	if len(outs)+len(ins) > 0 {
		updt := em.Master.EventTopUpdateStart()
		// now send all the exits before the enters..
		fe.Action = dnd.Exit
		for i := range outs {
			outs[i].Call(send, int64(ftyp), &fe)
		}
		fe.Action = dnd.Enter
		for i := range ins {
			ins[i].Call(send, int64(ftyp), &fe)
		}
		em.Master.EventTopUpdateEnd(updt)
		return ki.Continue
	}
	return ki.Break
}

*/

///////////////////////////////////////////////////////////////////
//   Key events

// SendKeyChordEvent sends a KeyChord event with given values.  If popup is
// true, then only items on popup are in scope, otherwise items NOT on popup
// are in scope (if no popup, everything is in scope).
func (em *EventMgr) SendKeyChordEvent(popup bool, r rune, mods ...goosi.Modifiers) {
	ke := key.NewEvent(r, 0, key.Press, 0)
	ke.SetTime()
	// ke.SetModifiers(mods...)
	// em.HandleEvent(ke)
}

// SendKeyFunEvent sends a KeyChord event with params from the given KeyFun.
// If popup is true, then only items on popup are in scope, otherwise items
// NOT on popup are in scope (if no popup, everything is in scope).
func (em *EventMgr) SendKeyFunEvent(kf KeyFuns, popup bool) {
	chord := ActiveKeyMap.ChordForFun(kf)
	if chord == "" {
		return
	}
	r, mods, err := chord.Decode()
	if err != nil {
		return
	}
	ke := key.NewEvent(r, 0, key.Press, mods)
	ke.SetTime()
	// em.HandleEvent(&ke)
}

// CurFocus gets the current focus node under mutex protection
func (em *EventMgr) CurFocus() Widget {
	em.FocusMu.RLock()
	defer em.FocusMu.RUnlock()
	return em.Focus
}

// setFocusPtr JUST sets the focus pointer under mutex protection --
// use SetFocus for end-user setting of focus
func (em *EventMgr) setFocusPtr(k Widget) {
	em.FocusMu.Lock()
	em.Focus = k
	em.FocusMu.Unlock()
}

// SetFocus sets focus to given item -- returns true if focus changed.
// If item is nil, then nothing has focus.
func (em *EventMgr) SetFocus(k Widget) bool {
	cfoc := em.CurFocus()
	if cfoc == k {
		if k != nil {
			_, wb := AsWidget(k)
			if wb != nil && wb.This() != nil {
				wb.SetFocusState(true) // ensure focus flag always set
			}
		}
		return false
	}

	if cfoc != nil {
		wi, wb := AsWidget(cfoc)
		if wb != nil && wb.This() != nil {
			wb.SetFocusState(false)
			// fmt.Printf("clear foc: %v\n", ni.Path())
			wi.FocusChanged(FocusLost)
		}
	}
	em.setFocusPtr(k)
	if k == nil {
		return true
	}
	wi, wb := AsWidget(k)
	if wb == nil || wb.This() == nil { // only 2d for now
		em.setFocusPtr(nil)
		return false
	}
	wb.SetFocusState(true)
	em.SetRenderWinFocusActive(true)
	// fmt.Printf("set foc: %v\n", ni.Path())
	em.ClearNonFocus(k) // shouldn't need this but actually sometimes do
	wi.FocusChanged(FocusGot)
	return true
}

//	FocusNext sets the focus on the next item that can accept focus after the
//
// given item (can be nil) -- returns true if a focus item found.
func (em *EventMgr) FocusNext(foc Widget) bool {
	gotFocus := false
	focusNext := false // get the next guy
	if foc == nil {
		focusNext = true
	}

	focRoot := em.CurFocus() // em.Master.FocusTopNode()

	for i := 0; i < 2; i++ {
		focRoot.WalkPre(func(k ki.Ki) bool {
			if gotFocus {
				return ki.Break
			}
			wi, wb := AsWidget(k)
			if wb == nil || wb.This() == nil {
				return ki.Continue
			}
			if foc == wi { // current focus can be a non-can-focus item
				focusNext = true
				return ki.Continue
			}
			if !focusNext {
				return ki.Continue
			}
			if !wb.CanFocus() {
				return ki.Continue
			}
			em.SetFocus(wi)
			gotFocus = true
			return ki.Break // done
		})
		if gotFocus {
			return true
		}
		focusNext = true // this time around, just get the first one
	}
	return gotFocus
}

// FocusOnOrNext sets the focus on the given item, or the next one that can
// accept focus -- returns true if a new focus item found.
func (em *EventMgr) FocusOnOrNext(foc Widget) bool {
	cfoc := em.CurFocus()
	if cfoc == foc {
		return true
	}
	_, wb := AsWidget(foc)
	if wb == nil || wb.This() == nil {
		return false
	}
	if wb.CanFocus() {
		em.SetFocus(foc)
		return true
	}
	return em.FocusNext(foc)
}

// FocusOnOrPrev sets the focus on the given item, or the previous one that can
// accept focus -- returns true if a new focus item found.
func (em *EventMgr) FocusOnOrPrev(foc Widget) bool {
	cfoc := em.CurFocus()
	if cfoc == foc {
		return true
	}
	_, wb := AsWidget(foc)
	if wb == nil || wb.This() == nil {
		return false
	}
	if wb.CanFocus() {
		em.SetFocus(foc)
		return true
	}
	return em.FocusPrev(foc)
}

// FocusPrev sets the focus on the previous item before the given item (can be nil)
func (em *EventMgr) FocusPrev(foc Widget) bool {
	if foc == nil { // must have a current item here
		em.FocusLast()
		return false
	}

	gotFocus := false
	var prevItem Widget

	focRoot := em.CurFocus() // em.Master.FocusTopNode()

	focRoot.WalkPre(func(k ki.Ki) bool {
		if gotFocus {
			return ki.Break
		}
		wi, wb := AsWidget(k)
		if wb == nil || wb.This() == nil {
			return ki.Continue
		}
		if foc == wi {
			gotFocus = true
			return ki.Break
		}
		if !wb.CanFocus() {
			return ki.Continue
		}
		prevItem = wi
		return ki.Continue
	})
	if gotFocus && prevItem != nil {
		em.SetFocus(prevItem)
		return true
	} else {
		return em.FocusLast()
	}
}

// FocusLast sets the focus on the last item in the tree -- returns true if a
// focusable item was found
func (em *EventMgr) FocusLast() bool {
	var lastItem Widget

	focRoot := em.CurFocus() // em.Master.FocusTopNode()

	focRoot.WalkPre(func(k ki.Ki) bool {
		wi, wb := AsWidget(k)
		if wb == nil || wb.This() == nil {
			return ki.Continue
		}
		if !wb.CanFocus() {
			return ki.Continue
		}
		lastItem = wi
		return ki.Continue
	})
	em.SetFocus(lastItem)
	if lastItem == nil {
		return false
	}
	return true
}

// ClearNonFocus clears the focus of any non-w.Focus item.
func (em *EventMgr) ClearNonFocus(foc Widget) {
	focRoot := em.CurFocus() // em.Master.FocusTopNode()

	focRoot.WalkPre(func(k ki.Ki) bool {
		if k == focRoot { // skip top-level
			return ki.Continue
		}
		wi, wb := AsWidget(k)
		if wb == nil || wb.This() == nil {
			return ki.Continue
		}
		if foc == k {
			return ki.Continue
		}
		if wb.StateIs(states.Focused) {
			if EventTrace {
				fmt.Printf("ClearNonFocus: had focus: %v\n", wb.Path())
			}
			wb.SetFlag(false, HasFocus)
			wi.FocusChanged(FocusLost)
		}
		return ki.Continue
	})
}

// PushFocus pushes current focus onto stack and sets new focus.
func (em *EventMgr) PushFocus(p Widget) {
	em.FocusMu.Lock()
	if em.FocusStack == nil {
		em.FocusStack = make([]Widget, 0, 50)
	}
	em.FocusStack = append(em.FocusStack, em.Focus)
	em.Focus = nil // don't un-focus on prior item when pushing
	em.FocusMu.Unlock()
	em.FocusOnOrNext(p)
}

// PopFocus pops off the focus stack and sets prev to current focus.
func (em *EventMgr) PopFocus() {
	em.FocusMu.Lock()
	if em.FocusStack == nil || len(em.FocusStack) == 0 {
		em.Focus = nil
		return
	}
	sz := len(em.FocusStack)
	em.Focus = nil
	nxtf := em.FocusStack[sz-1]
	_, wb := AsWidget(nxtf)
	if wb != nil && wb.This() != nil {
		em.FocusMu.Unlock()
		em.SetFocus(nxtf)
		em.FocusMu.Lock()
	}
	em.FocusStack = em.FocusStack[:sz-1]
	em.FocusMu.Unlock()
}

// SetStartFocus sets the given item to be first focus when window opens.
func (em *EventMgr) SetStartFocus(k Widget) {
	em.FocusMu.Lock()
	em.StartFocus = k
	em.FocusMu.Unlock()
}

// ActivateStartFocus activates start focus if there is no current focus
// and StartFocus is set -- returns true if activated
func (em *EventMgr) ActivateStartFocus() bool {
	em.FocusMu.RLock()
	if em.StartFocus == nil {
		em.FocusMu.RUnlock()
		return false
	}
	em.FocusMu.RUnlock()
	em.FocusMu.Lock()
	sf := em.StartFocus
	em.StartFocus = nil
	em.FocusMu.Unlock()
	em.FocusOnOrNext(sf)
	return true
}

// InitialFocus establishes the initial focus for the window if no focus
// is set -- uses ActivateStartFocus or FocusNext as backup.
func (em *EventMgr) InitialFocus() {
	if em.CurFocus() == nil {
		if !em.ActivateStartFocus() {
			em.FocusNext(em.CurFocus())
		}
	}
}

///////////////////////////////////////////////////////////////////
//   Manager-level event processing

// MangerKeyChordEvents handles lower-priority manager-level key events.
// Mainly tab, shift-tab, and GoGiEditor and Prefs.
// event will be marked as processed if handled here.
func (em *EventMgr) ManagerKeyChordEvents(e *key.Event) {
	if e.IsHandled() {
		return
	}
	cs := e.Chord()
	kf := KeyFun(cs)
	switch kf {
	case KeyFunFocusNext: // tab
		em.FocusNext(em.CurFocus())
		e.SetHandled()
	case KeyFunFocusPrev: // shift-tab
		em.FocusPrev(em.CurFocus())
		e.SetHandled()
	case KeyFunGoGiEditor:
		// todo:
		// TheViewIFace.GoGiEditor(em.Master.EventTopNode())
		e.SetHandled()
	case KeyFunPrefs:
		TheViewIFace.PrefsView(&Prefs)
		e.SetHandled()
	}
}
