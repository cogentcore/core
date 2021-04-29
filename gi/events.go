// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
)

//go:generate stringer -type=EventPris

// EventPris for different queues of event signals, processed in priority order
type EventPris int32

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

	EventPrisN

	// AllPris = -1 = all priorities (for delete cases only)
	AllPris EventPris = -1
)

const (
	// Popups means include popups
	Popups = true

	// NoPopups means exclude popups
	NoPopups = false
)

// EventMgr is an event manager that handles distributing events to nodes.
// It relies on the EventMaster for a few things outside of its scope.
type EventMgr struct {
	Master          EventMaster                             `desc:"master of this event mangager -- handles broader scope issues"`
	EventSigs       [oswin.EventTypeN][EventPrisN]ki.Signal `desc:"signals for communicating each type of event, organized by priority"`
	EventMu         sync.Mutex                              `desc:"mutex that protects event sending"`
	TimerMu         sync.Mutex                              `desc:"mutex that protects timer variable updates (e.g., hover AfterFunc's)"`
	Dragging        ki.Ki                                   `desc:"node receiving mouse dragging events -- not for DND but things like sliders -- anchor to same"`
	Scrolling       ki.Ki                                   `desc:"node receiving mouse scrolling events -- anchor to same"`
	DNDStage        DNDStages                               `desc:"stage of DND process"`
	DNDData         mimedata.Mimes                          `desc:"drag-n-drop data -- if non-nil, then DND is taking place"`
	DNDSource       ki.Ki                                   `desc:"drag-n-drop source node"`
	DNDFinalEvent   *dnd.Event                              `desc:"final event for DND which is sent if a finalize is received"`
	DNDDropMod      dnd.DropMods                            `desc:"modifier in place at time of drop event (DropMove or DropCopy)"`
	Focus           ki.Ki                                   `desc:"node receiving keyboard events -- use SetFocus, CurFocus"`
	FocusMu         sync.RWMutex                            `desc:"mutex that protects focus updating"`
	FocusStack      []ki.Ki                                 `desc:"stack of focus"`
	StartFocus      ki.Ki                                   `desc:"node to focus on at start when no other focus has been set yet -- use SetStartFocus"`
	LastModBits     int32                                   `desc:"Last modifier key bits from most recent Mouse, Keyboard events"`
	LastSelMode     mouse.SelectModes                       `desc:"Last Select Mode from most recent Mouse, Keyboard events"`
	LastMousePos    image.Point                             `desc:"Last mouse position from most recent Mouse events"`
	LagSkipDeltaPos image.Point                             `desc:"change in position accumulated from skipped-over laggy mouse move events"`
	LagLastSkipped  bool                                    `desc:"true if last event was skipped due to lag"`
	startDrag       *mouse.DragEvent
	dragStarted     bool
	startDND        *mouse.DragEvent
	dndStarted      bool
	startHover      *mouse.MoveEvent
	curHover        *mouse.HoverEvent
	hoverStarted    bool
	hoverTimer      *time.Timer
	startDNDHover   *mouse.DragEvent
	curDNDHover     *mouse.DragEvent
	dndHoverStarted bool
	dndHoverTimer   *time.Timer
}

// WinEventRecv is used to hold info about widgets receiving event signals to
// given function, used for sorting and delayed sending.
type WinEventRecv struct {
	Recv ki.Ki
	Func ki.RecvFunc
	Data int
}

// Set sets the recv and fun
func (we *WinEventRecv) Set(r ki.Ki, f ki.RecvFunc, data int) {
	we.Recv = r
	we.Func = f
	we.Data = data
}

// Call calls the function on the recv with the args
func (we *WinEventRecv) Call(send ki.Ki, sig int64, data interface{}) {
	if EventTrace {
		fmt.Printf("calling event: %v method on: %v\n", data, we.Recv.Path())
	}
	we.Func(we.Recv, send, sig, data)
}

type WinEventRecvList []WinEventRecv

func (wl *WinEventRecvList) Add(recv ki.Ki, fun ki.RecvFunc, data int) {
	rr := WinEventRecv{recv, fun, data}
	*wl = append(*wl, rr)
}

func (wl *WinEventRecvList) AddDepth(recv ki.Ki, fun ki.RecvFunc, par ki.Ki) {
	wl.Add(recv, fun, recv.ParentLevel(par))
}

// ConnectEvent adds a Signal connection for given event type and
// priority to given receiver
func (em *EventMgr) ConnectEvent(recv ki.Ki, et oswin.EventType, pri EventPris, fun ki.RecvFunc) {
	if et >= oswin.EventTypeN {
		log.Printf("EventMgr ConnectEvent type: %v is not a known event type\n", et)
		return
	}
	em.EventSigs[et][pri].Connect(recv, fun)
}

// DisconnectEvent removes Signal connection for given event type to given
// receiver -- pri is priority -- pass AllPris for all priorities
func (em *EventMgr) DisconnectEvent(recv ki.Ki, et oswin.EventType, pri EventPris) {
	if et >= oswin.EventTypeN {
		log.Printf("EventMgr DisconnectEvent type: %v is not a known event type\n", et)
		return
	}
	if pri == AllPris {
		for p := HiPri; p < EventPrisN; p++ {
			em.EventSigs[et][p].Disconnect(recv)
		}
	} else {
		em.EventSigs[et][pri].Disconnect(recv)
	}
}

// DisconnectAllEvents disconnect node from all event signals -- pri is
// priority -- pass AllPris for all priorities
func (em *EventMgr) DisconnectAllEvents(recv ki.Ki, pri EventPris) {
	if pri == AllPris {
		for et := oswin.EventType(0); et < oswin.EventTypeN; et++ {
			for p := HiPri; p < EventPrisN; p++ {
				em.EventSigs[et][p].Disconnect(recv)
			}
		}
	} else {
		for et := oswin.EventType(0); et < oswin.EventTypeN; et++ {
			em.EventSigs[et][pri].Disconnect(recv)
		}
	}
}

// SendEventSignal sends given event signal to all receivers that want it --
// note that because there is a different EventSig for each event type, we are
// ONLY looking at nodes that have registered to receive that type of event --
// the further filtering is just to ensure that they are in the right position
// to receive the event (focus, popup filtering, etc).  If popup is true, then
// only items on popup are in scope, otherwise items NOT on popup are in scope
// (if no popup, everything is in scope).
func (em *EventMgr) SendEventSignal(evi oswin.Event, popup bool) {
	et := evi.Type()
	if et > oswin.EventTypeN || et < 0 {
		return // can't handle other types of events here due to EventSigs[et] size
	}

	em.EventMu.Lock()

	send := em.Master.EventTopNode()

	// fmt.Printf("got event type: %v\n", et)
	for pri := HiPri; pri < EventPrisN; pri++ {
		if pri != LowRawPri && evi.IsProcessed() { // someone took care of it
			continue
		}

		// we take control of signal process to sort elements by depth, and
		// dispatch to inner-most one first
		rvs := make(WinEventRecvList, 0, 10)

		esig := &em.EventSigs[et][pri]
		esig.ConsFunc(func(recv ki.Ki, fun ki.RecvFunc) bool {
			if recv.IsDeleted() {
				return ki.Continue
			}
			cont := em.SendEventSignalFunc(evi, popup, &rvs, recv, fun)
			return cont // false = break
		})

		if len(rvs) == 0 {
			continue
		}

		// deepest first
		sort.Slice(rvs, func(i, j int) bool {
			return rvs[i].Data > rvs[j].Data
		})

		for _, rr := range rvs {
			switch evi.(type) {
			case *mouse.DragEvent:
				if em.Dragging == nil {
					rr.Recv.SetFlag(int(NodeDragging)) // PROVISIONAL!
				}
			}
			em.EventMu.Unlock()
			rr.Call(send, int64(et), evi) // could call further event loops..
			em.EventMu.Lock()
			if pri != LowRawPri && evi.IsProcessed() { // someone took care of it
				switch evi.(type) { // only grab events if processed
				case *mouse.DragEvent:
					if em.Dragging == nil {
						em.Dragging = rr.Recv
						rr.Recv.SetFlag(int(NodeDragging))
					}
				case *mouse.ScrollEvent:
					if em.Scrolling == nil {
						em.Scrolling = rr.Recv
					}
				}
				break
			} else {
				switch evi.(type) {
				case *mouse.DragEvent:
					if em.Dragging == nil {
						rr.Recv.ClearFlag(int(NodeDragging)) // clear provisional
					}
				}
			}
		}
	}
	em.EventMu.Unlock()
}

// SendEventSignalFunc is the inner loop of the SendEventSignal -- needed to deal with
// map iterator locking logic in a cleaner way.  Returns true to continue, false to break
func (em *EventMgr) SendEventSignalFunc(evi oswin.Event, popup bool, rvs *WinEventRecvList, recv ki.Ki, fun ki.RecvFunc) bool {
	if !em.Master.IsInScope(recv, popup) {
		return ki.Continue
	}
	nii, ni := KiToNode2D(recv)
	if ni != nil {
		if evi.OnFocus() {
			if !nii.HasFocus2D() { // note: HasFocus2D is a separate interface method, containers also set to true
				return ki.Continue
			}
			if EventTrace && recv == em.CurFocus() {
				fmt.Printf("Event: cur focus: %v\n", recv.Path())
			}
			if !em.Master.IsFocusActive() { // reactivate on keyboard input
				em.Master.SetFocusActiveState(true)
				if EventTrace {
					fmt.Printf("Event: set focus active, was not: %v\n", ni.Path())
				}
				nii.FocusChanged2D(FocusActive)
			}
		}
	}
	top := em.Master.EventTopNode()
	// remainder is done using generic node interface, for 2D and 3D
	gni := recv.(Node)
	gn := gni.AsGiNode()
	if evi.HasPos() {
		pos := evi.Pos()
		switch evi.(type) {
		case *mouse.DragEvent:
			if em.Dragging != nil {
				if em.Dragging == gn.This() {
					if EventTrace {
						fmt.Printf("Event: dragging top pri: %v\n", recv.Path())
					}
					rvs.Add(recv, fun, 10000)
					return ki.Break
				} else {
					return ki.Continue
				}
			} else {
				if gn.PosInWinBBox(pos) {
					rvs.AddDepth(recv, fun, top)
					return ki.Break
				}
				return ki.Continue
			}
		case *mouse.ScrollEvent:
			if em.Scrolling != nil {
				if em.Scrolling == gn.This() {
					if EventTrace {
						fmt.Printf("Event: scrolling top pri: %v\n", recv.Path())
					}
					rvs.Add(recv, fun, 10000)
				} else {
					return ki.Continue
				}
			} else {
				if gn.PosInWinBBox(pos) {
					rvs.AddDepth(recv, fun, top)
					return ki.Break
				}
				return ki.Continue
			}
		default:
			if em.Dragging == gn.This() { // dragger always gets it
				if EventTrace {
					fmt.Printf("Event: dragging, non drag top pri: %v\n", recv.Path())
				}
				rvs.Add(recv, fun, 10000) // top priority -- can't steal!
				return ki.Break
			}
			if !gn.PosInWinBBox(pos) {
				return ki.Continue
			}
		}
	}
	rvs.AddDepth(recv, fun, top)
	return ki.Continue
}

// SendSig directly calls SendSig from given recv, sender for given event
// across all priorities.
func (em *EventMgr) SendSig(recv, sender ki.Ki, evi oswin.Event) {
	et := evi.Type()
	for pri := HiPri; pri < EventPrisN; pri++ {
		em.EventSigs[et][pri].SendSig(recv, sender, int64(et), evi)
	}
}

///////////////////////////////////////////////////////////////////////////
//  Mouse event processing

// MouseEvents processes mouse drag and move events
func (em *EventMgr) MouseEvents(evi oswin.Event) {
	et := evi.Type()
	if et == oswin.MouseDragEvent {
		em.MouseDragEvents(evi)
	} else if et != oswin.KeyEvent { // allow modifier keypress
		em.ResetMouseDrag()
	}

	if et == oswin.MouseMoveEvent {
		em.MouseMoveEvents(evi)
	} else {
		em.ResetMouseMove()
	}

	if et == oswin.MouseEvent {
		me := evi.(*mouse.Event)
		em.LastModBits = me.Modifiers
		em.LastSelMode = me.SelectMode()
		em.LastMousePos = me.Pos()
	}
	if et == oswin.KeyChordEvent {
		ke := evi.(*key.ChordEvent)
		em.LastModBits = ke.Modifiers
		em.LastSelMode = mouse.SelectModeBits(ke.Modifiers)
	}
}

// MouseEventReset resets state for "catch" events (Dragging, Scrolling)
func (em *EventMgr) MouseEventReset(evi oswin.Event) {
	et := evi.Type()
	if em.Dragging != nil && et != oswin.MouseDragEvent {
		em.Dragging.ClearFlag(int(NodeDragging))
		em.Dragging = nil
	}
	if em.Scrolling != nil && et != oswin.MouseScrollEvent {
		em.Scrolling = nil
	}
}

// MouseDragEvents processes MouseDragEvent to Detect start of drag and DND.
// These require timing and delays, e.g., due to minor wiggles when pressing
// the mouse button
func (em *EventMgr) MouseDragEvents(evi oswin.Event) {
	me := evi.(*mouse.DragEvent)
	em.LastModBits = me.Modifiers
	em.LastSelMode = me.SelectMode()
	em.LastMousePos = me.Pos()
	now := time.Now()
	if !em.dragStarted {
		if em.startDrag == nil {
			em.startDrag = me
		} else {
			if em.DoInstaDrag(em.startDrag, !em.Master.CurPopupIsTooltip()) {
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
				em.dndHoverStarted = false
				em.startDNDHover = nil
				em.curDNDHover = nil
				em.dndHoverTimer = nil
				em.TimerMu.Unlock()
				em.SendDNDHoverEvent(hoe)
			})
		} else {
			dst := int(mat32.Hypot(float32(em.startDNDHover.Where.X-me.Pos().X), float32(em.startDNDHover.Where.Y-me.Pos().Y)))
			if dst > HoverMaxPix {
				em.dndHoverTimer.Stop()
				em.dndHoverStarted = false
				em.startDNDHover = nil
				em.dndHoverTimer = nil
			} else {
				em.curDNDHover = me
			}
		}
		em.TimerMu.Unlock()
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
func (em *EventMgr) MouseMoveEvents(evi oswin.Event) {
	me := evi.(*mouse.MoveEvent)
	em.LastModBits = me.Modifiers
	em.LastSelMode = me.SelectMode()
	em.LastMousePos = me.Pos()
	em.TimerMu.Lock()
	if !em.hoverStarted {
		em.hoverStarted = true
		em.startHover = me
		em.curHover = &mouse.HoverEvent{Event: me.Event}
		em.hoverTimer = time.AfterFunc(time.Duration(HoverStartMSec)*time.Millisecond, func() {
			em.TimerMu.Lock()
			hoe := em.curHover
			em.hoverStarted = false
			em.startHover = nil
			em.curHover = nil
			em.hoverTimer = nil
			em.TimerMu.Unlock()
			if hoe != nil {
				em.SendHoverEvent(hoe)
			}
		})
	} else {
		dst := int(mat32.Hypot(float32(em.startHover.Where.X-me.Pos().X), float32(em.startHover.Where.Y-me.Pos().Y)))
		if dst > HoverMaxPix {
			em.hoverTimer.Stop()
			em.hoverStarted = false
			em.startHover = nil
			em.hoverTimer = nil
			em.Master.DeleteTooltip()
		} else {
			em.curHover = &mouse.HoverEvent{Event: me.Event}
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

// GenMouseFocusEvents processes mouse.MoveEvent to generate mouse.FocusEvent
// events -- returns true if any such events were sent.  If popup is true,
// then only items on popup are in scope, otherwise items NOT on popup are in
// scope (if no popup, everything is in scope).
func (em *EventMgr) GenMouseFocusEvents(mev *mouse.MoveEvent, popup bool) bool {
	fe := mouse.FocusEvent{Event: mev.Event}
	pos := mev.Pos()
	ftyp := oswin.MouseFocusEvent
	updated := false
	updt := false
	send := em.Master.EventTopNode()
	for pri := HiPri; pri < EventPrisN; pri++ {
		em.EventSigs[ftyp][pri].EmitFiltered(send, int64(ftyp), &fe, func(k ki.Ki) bool {
			if k.IsDeleted() { // destroyed is filtered upstream
				return ki.Break
			}
			if !em.Master.IsInScope(k, popup) {
				return ki.Break
			}
			_, ni := KiToNode2D(k)
			if ni != nil {
				in := ni.PosInWinBBox(pos)
				if in {
					if !ni.HasFlag(int(MouseHasEntered)) {
						fe.Action = mouse.Enter
						ni.SetFlag(int(MouseHasEntered))
						if !updated {
							updt = em.Master.EventTopUpdateStart()
							updated = true
						}
						return ki.Continue // send event
					} else {
						return ki.Break // already in
					}
				} else { // mouse not in object
					if ni.HasFlag(int(MouseHasEntered)) {
						fe.Action = mouse.Exit
						ni.ClearFlag(int(MouseHasEntered))
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
}

// DoInstaDrag tests whether the given mouse DragEvent is on a widget marked
// with InstaDrag
func (em *EventMgr) DoInstaDrag(me *mouse.DragEvent, popup bool) bool {
	et := me.Type()
	for pri := HiPri; pri < EventPrisN; pri++ {
		esig := &em.EventSigs[et][pri]
		gotOne := false
		esig.ConsFunc(func(recv ki.Ki, fun ki.RecvFunc) bool {
			if recv.IsDeleted() {
				return ki.Continue
			}
			if !em.Master.IsInScope(recv, popup) {
				return ki.Continue
			}
			_, ni := KiToNode2D(recv)
			if ni != nil {
				pos := me.Pos()
				if ni.PosInWinBBox(pos) {
					if ni.IsInstaDrag() {
						em.Dragging = ni.This()
						ni.SetFlag(int(NodeDragging))
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
	return ki.Break
}

// SendHoverEvent sends mouse hover event, based on last mouse move event
func (em *EventMgr) SendHoverEvent(he *mouse.HoverEvent) {
	he.ClearProcessed()
	he.Action = mouse.Hover
	em.SendEventSignal(he, Popups)
}

//////////////////////////////////////////////////////////////////////
//  Drag-n-Drop = DND

//go:generate stringer -type=DNDStages

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
func (em *EventMgr) DNDStartEvent(e *mouse.DragEvent) {
	de := dnd.Event{EventBase: e.EventBase, Where: e.Where, Modifiers: e.Modifiers}
	de.ClearProcessed()
	de.Action = dnd.Start
	de.DefaultMod() // based on current key modifiers
	em.DNDStage = DNDStartSent
	if DNDTrace {
		fmt.Printf("\nDNDStartSent\n")
	}
	em.SendEventSignal(&de, NoPopups)
	// now up to receiver to call StartDragNDrop if they want to..
}

// DNDStart is driven by node responding to start event, actually starts DND
func (em *EventMgr) DNDStart(src ki.Ki, data mimedata.Mimes) {
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
func (em *EventMgr) SendDNDHoverEvent(e *mouse.DragEvent) {
	if e == nil {
		return
	}
	he := dnd.FocusEvent{Event: dnd.Event{EventBase: e.EventBase, Where: e.Where, Modifiers: e.Modifiers}}
	he.ClearProcessed()
	he.Action = dnd.Hover
	em.SendEventSignal(&he, NoPopups)
}

// SendDNDMoveEvent sends DND move event
func (em *EventMgr) SendDNDMoveEvent(e *mouse.DragEvent) *dnd.MoveEvent {
	// todo: when e.Where goes negative, transition to OS DND
	// todo: send move / enter / exit events to anyone listening
	de := &dnd.MoveEvent{Event: dnd.Event{EventBase: e.Event.EventBase, Where: e.Event.Where, Modifiers: e.Event.Modifiers}, From: e.From, LastTime: e.LastTime}
	de.ClearProcessed()
	de.DefaultMod() // based on current key modifiers
	de.Action = dnd.Move
	em.SendEventSignal(de, NoPopups)
	em.GenDNDFocusEvents(de, NoPopups)
	return de
}

// SendDNDDropEvent sends DND drop event -- returns false if drop event was not processed
// in which case the event should be cleared (by the Window)
func (em *EventMgr) SendDNDDropEvent(e *mouse.Event) bool {
	de := dnd.Event{EventBase: e.EventBase, Where: e.Where, Modifiers: e.Modifiers}
	de.ClearProcessed()
	de.DefaultMod()
	de.Action = dnd.DropOnTarget
	de.Data = em.DNDData
	de.Source = em.DNDSource
	em.DNDSource.ClearFlag(int(NodeDragging))
	em.Dragging = nil
	em.DNDFinalEvent = &de
	em.DNDDropMod = de.Mod
	em.DNDStage = DNDDropped
	if DNDTrace {
		fmt.Printf("DNDDropped\n")
	}
	e.SetProcessed()
	em.SendEventSignal(&de, NoPopups)
	return de.IsProcessed()
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

// GenDNDFocusEvents processes mouse.MoveEvent to generate dnd.FocusEvent
// events -- returns true if any such events were sent.  If popup is true,
// then only items on popup are in scope, otherwise items NOT on popup are in
// scope (if no popup, everything is in scope).  Extra work is done to ensure
// that Exit from prior widget is always sent before Enter to next one.
func (em *EventMgr) GenDNDFocusEvents(mev *dnd.MoveEvent, popup bool) bool {
	fe := dnd.FocusEvent{Event: mev.Event}
	pos := mev.Pos()
	ftyp := oswin.DNDFocusEvent

	// first pass is just to get all the ins and outs
	var ins, outs WinEventRecvList

	send := em.Master.EventTopNode()
	for pri := HiPri; pri < EventPrisN; pri++ {
		esig := &em.EventSigs[ftyp][pri]
		esig.ConsFunc(func(recv ki.Ki, fun ki.RecvFunc) bool {
			if recv.IsDeleted() {
				return ki.Continue
			}
			if !em.Master.IsInScope(recv, popup) {
				return ki.Continue
			}
			_, ni := KiToNode2D(recv)
			if ni != nil {
				in := ni.PosInWinBBox(pos)
				if in {
					if !ni.HasFlag(int(DNDHasEntered)) {
						ni.SetFlag(int(DNDHasEntered))
						ins.Add(recv, fun, 0)
					}
				} else { // mouse not in object
					if ni.HasFlag(int(DNDHasEntered)) {
						ni.ClearFlag(int(DNDHasEntered))
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

///////////////////////////////////////////////////////////////////
//   Key events

// SendKeyChordEvent sends a KeyChord event with given values.  If popup is
// true, then only items on popup are in scope, otherwise items NOT on popup
// are in scope (if no popup, everything is in scope).
func (em *EventMgr) SendKeyChordEvent(popup bool, r rune, mods ...key.Modifiers) {
	ke := key.ChordEvent{}
	ke.SetTime()
	ke.SetModifiers(mods...)
	ke.Rune = r
	ke.Action = key.Press
	em.SendEventSignal(&ke, popup)
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
	ke := key.ChordEvent{}
	ke.SetTime()
	ke.Modifiers = mods
	ke.Rune = r
	ke.Action = key.Press
	em.SendEventSignal(&ke, popup)
}

// CurFocus gets the current focus node under mutex protection
func (em *EventMgr) CurFocus() ki.Ki {
	em.FocusMu.RLock()
	defer em.FocusMu.RUnlock()
	return em.Focus
}

// setFocusPtr JUST sets the focus pointer under mutex protection --
// use SetFocus for end-user setting of focus
func (em *EventMgr) setFocusPtr(k ki.Ki) {
	em.FocusMu.Lock()
	em.Focus = k
	em.FocusMu.Unlock()
}

// SetFocus sets focus to given item -- returns true if focus changed.
// If item is nil, then nothing has focus.
func (em *EventMgr) SetFocus(k ki.Ki) bool {
	cfoc := em.CurFocus()
	if cfoc == k {
		if k != nil {
			_, ni := KiToNode2D(k)
			if ni != nil && ni.This() != nil {
				ni.SetFocusState(true) // ensure focus flag always set
			}
		}
		return false
	}

	updt := em.Master.EventTopUpdateStart()
	defer em.Master.EventTopUpdateEnd(updt)

	if cfoc != nil {
		nii, ni := KiToNode2D(cfoc)
		if ni != nil && ni.This() != nil {
			ni.SetFocusState(false)
			// fmt.Printf("clear foc: %v\n", ni.Path())
			nii.FocusChanged2D(FocusLost)
		}
	}
	em.setFocusPtr(k)
	if k == nil {
		return true
	}
	nii, ni := KiToNode2D(k)
	if ni == nil || ni.This() == nil { // only 2d for now
		em.setFocusPtr(nil)
		return false
	}
	ni.SetFocusState(true)
	em.Master.SetFocusActiveState(true)
	// fmt.Printf("set foc: %v\n", ni.Path())
	em.ClearNonFocus(k) // shouldn't need this but actually sometimes do
	nii.FocusChanged2D(FocusGot)
	return true
}

// 	FocusNext sets the focus on the next item that can accept focus after the
// given item (can be nil) -- returns true if a focus item found.
func (em *EventMgr) FocusNext(foc ki.Ki) bool {
	gotFocus := false
	focusNext := false // get the next guy
	if foc == nil {
		focusNext = true
	}

	focRoot := em.Master.FocusTopNode()

	for i := 0; i < 2; i++ {
		focRoot.FuncDownMeFirst(0, focRoot, func(k ki.Ki, level int, d interface{}) bool {
			if gotFocus {
				return ki.Break
			}
			_, ni := KiToNode2D(k)
			if ni == nil || ni.This() == nil {
				return ki.Continue
			}
			if foc == k { // current focus can be a non-can-focus item
				focusNext = true
				return ki.Continue
			}
			if !focusNext {
				return ki.Continue
			}
			if !ni.CanFocus() {
				return ki.Continue
			}
			em.SetFocus(k)
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
func (em *EventMgr) FocusOnOrNext(foc ki.Ki) bool {
	cfoc := em.CurFocus()
	if cfoc == foc {
		return true
	}
	_, ni := KiToNode2D(foc)
	if ni == nil || ni.This() == nil {
		return false
	}
	if ni.CanFocus() {
		em.SetFocus(foc)
		return true
	}
	return em.FocusNext(foc)
}

// FocusOnOrPrev sets the focus on the given item, or the previous one that can
// accept focus -- returns true if a new focus item found.
func (em *EventMgr) FocusOnOrPrev(foc ki.Ki) bool {
	cfoc := em.CurFocus()
	if cfoc == foc {
		return true
	}
	_, ni := KiToNode2D(foc)
	if ni == nil || ni.This() == nil {
		return false
	}
	if ni.CanFocus() {
		em.SetFocus(foc)
		return true
	}
	return em.FocusPrev(foc)
}

// FocusPrev sets the focus on the previous item before the given item (can be nil)
func (em *EventMgr) FocusPrev(foc ki.Ki) bool {
	if foc == nil { // must have a current item here
		em.FocusLast()
		return false
	}

	gotFocus := false
	var prevItem ki.Ki

	focRoot := em.Master.FocusTopNode()

	focRoot.FuncDownMeFirst(0, focRoot, func(k ki.Ki, level int, d interface{}) bool {
		if gotFocus {
			return ki.Break
		}
		_, ni := KiToNode2D(k)
		if ni == nil || ni.This() == nil {
			return ki.Continue
		}
		if foc == k {
			gotFocus = true
			return ki.Break
		}
		if !ni.CanFocus() {
			return ki.Continue
		}
		prevItem = k
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
	var lastItem ki.Ki

	focRoot := em.Master.FocusTopNode()

	focRoot.FuncDownMeFirst(0, focRoot, func(k ki.Ki, level int, d interface{}) bool {
		_, ni := KiToNode2D(k)
		if ni == nil || ni.This() == nil {
			return ki.Continue
		}
		if !ni.CanFocus() {
			return ki.Continue
		}
		lastItem = k
		return ki.Continue
	})
	em.SetFocus(lastItem)
	if lastItem == nil {
		return false
	}
	return true
}

// ClearNonFocus clears the focus of any non-w.Focus item.
func (em *EventMgr) ClearNonFocus(foc ki.Ki) {
	focRoot := em.Master.FocusTopNode()

	updated := false
	updt := false

	focRoot.FuncDownMeFirst(0, focRoot, func(k ki.Ki, level int, d interface{}) bool {
		if k == focRoot { // skip top-level
			return ki.Continue
		}
		nii, ni := KiToNode2D(k)
		if ni == nil || ni.This() == nil {
			return ki.Continue
		}
		if foc == k {
			return ki.Continue
		}
		if ni.HasFocus() {
			if EventTrace {
				fmt.Printf("ClearNonFocus: had focus: %v\n", ni.Path())
			}
			if !updated {
				updated = true
				updt = em.Master.EventTopUpdateStart()
			}
			ni.ClearFlag(int(HasFocus))
			nii.FocusChanged2D(FocusLost)
		}
		return ki.Continue
	})
	if updated {
		em.Master.EventTopUpdateEnd(updt)
	}
}

// PushFocus pushes current focus onto stack and sets new focus.
func (em *EventMgr) PushFocus(p ki.Ki) {
	em.FocusMu.Lock()
	if em.FocusStack == nil {
		em.FocusStack = make([]ki.Ki, 0, 50)
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
	_, ni := KiToNode2D(nxtf)
	if ni != nil && ni.This() != nil {
		em.FocusMu.Unlock()
		em.SetFocus(nxtf)
		em.FocusMu.Lock()
	}
	em.FocusStack = em.FocusStack[:sz-1]
	em.FocusMu.Unlock()
}

// SetStartFocus sets the given item to be first focus when window opens.
func (em *EventMgr) SetStartFocus(k ki.Ki) {
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
//   Filter Laggy Events

// FilterLaggyEvents filters repeated laggy events -- key for responsive resize, scroll, etc
// returns false if event should not be processed further, and true if it should.
// Should only be called when the current event is the same type as last time.
// Accumulates mouse deltas in LagSkipDeltaPos.
func (em *EventMgr) FilterLaggyEvents(evi oswin.Event) bool {
	et := evi.Type()
	now := time.Now()
	lag := now.Sub(evi.Time())
	lagMs := int(lag / time.Millisecond)

	switch et {
	case oswin.MouseScrollEvent:
		me := evi.(*mouse.ScrollEvent)
		if lagMs > EventSkipLagMSec {
			// fmt.Printf("skipped et %v lag %v\n", et, lag)
			if !em.LagLastSkipped {
				em.LagSkipDeltaPos = me.Delta
			} else {
				em.LagSkipDeltaPos = em.LagSkipDeltaPos.Add(me.Delta)
			}
			em.LagLastSkipped = true
			return false
		} else {
			if em.LagLastSkipped {
				me.Delta = em.LagSkipDeltaPos.Add(me.Delta)
			}
			em.LagLastSkipped = false
		}
	case oswin.MouseDragEvent:
		me := evi.(*mouse.DragEvent)
		if lagMs > EventSkipLagMSec {
			// fmt.Printf("skipped et %v lag %v\n", et, lag)
			if !em.LagLastSkipped {
				em.LagSkipDeltaPos = me.From
			}
			em.LagLastSkipped = true
			return false
		} else {
			if em.LagLastSkipped {
				me.From = em.LagSkipDeltaPos
			}
			em.LagLastSkipped = false
		}
	case oswin.MouseMoveEvent:
		me := evi.(*mouse.MoveEvent)
		if lagMs > EventSkipLagMSec {
			// fmt.Printf("skipped et %v lag %v\n", et, lag)
			if !em.LagLastSkipped {
				em.LagSkipDeltaPos = me.From
			}
			em.LagLastSkipped = true
			return false
		} else {
			if em.LagLastSkipped {
				me.From = em.LagSkipDeltaPos
			}
			em.LagLastSkipped = false
		}
	case oswin.KeyEvent:
		if lagMs > EventSkipLagMSec {
			// fmt.Printf("skipped et %v lag %v\n", et, lag)
			em.LagLastSkipped = true
			return false
		} else {
			em.LagLastSkipped = false
		}
	}
	return true
}

///////////////////////////////////////////////////////////////////
//   Manager-level event processing

// MangerKeyChordEvents handles lower-priority manager-level key events.
// Mainly tab, shift-tab, and GoGiEditor and Prefs.
// event will be marked as processed if handled here.
func (em *EventMgr) ManagerKeyChordEvents(e *key.ChordEvent) {
	if e.IsProcessed() {
		return
	}
	cs := e.Chord()
	kf := KeyFun(cs)
	switch kf {
	case KeyFunFocusNext: // tab
		em.FocusNext(em.CurFocus())
		e.SetProcessed()
	case KeyFunFocusPrev: // shift-tab
		em.FocusPrev(em.CurFocus())
		e.SetProcessed()
	case KeyFunGoGiEditor:
		TheViewIFace.GoGiEditor(em.Master.FocusTopNode())
		e.SetProcessed()
	case KeyFunPrefs:
		TheViewIFace.PrefsView(&Prefs)
		e.SetProcessed()
	}
}

///////////////////////////////////////////////////////////////////
//   Master interface

// EventMaster provides additional control methods for the
// event manager, for things beyond its immediate scope
type EventMaster interface {
	// EventTopNode returns the top-level node for this event scope.
	// This is also the node that serves as the event sender.
	// By default it is the Window.
	EventTopNode() ki.Ki

	// FocusTopNode returns the top-level node for key event focusing.
	FocusTopNode() ki.Ki

	// EventTopUpdateStart does an UpdateStart on top-level node, for batch updates.
	// This may not be identical to EventTopNode().UpdateStart() for
	// embedded case where Viewport is the EventTopNode.
	EventTopUpdateStart() bool

	// EventTopUpdateEnd does an UpdateEnd on top-level node, for batch updates.
	// This may not be identical to EventTopNode().UpdateEnd() for
	// embedded case where Viewport is the EventTopNode.
	EventTopUpdateEnd(updt bool)

	// IsInScope returns whether given node is in scope for receiving events
	IsInScope(node ki.Ki, popup bool) bool

	// CurPopupIsTooltip returns true if current popup is a tooltip
	CurPopupIsTooltip() bool

	// DeleteTooltip deletes any tooltip popup (called when hover ends)
	DeleteTooltip()

	// IsFocusActive returns true if focus is active in this master
	IsFocusActive() bool

	// SetFocusActiveState sets focus active state
	SetFocusActiveState(active bool)
}
