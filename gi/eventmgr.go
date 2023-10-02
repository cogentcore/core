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
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

var (
	// DragStartTime is the time to wait before DragStart
	DragStartTime = 200 * time.Millisecond

	// DragStartDist is pixel distance that must be moved before DragStart
	DragStartDist = 20

	// SlideStartTime is the time to wait before SlideStart
	SlideStartTime = 50 * time.Millisecond

	// SlideStartDist is pixel distance that must be moved before SlideStart
	SlideStartDist = 4

	// LongHoverTime is the time to wait before LongHoverStart event
	LongHoverTime = 500 * time.Millisecond

	// LongHoverStopDist is the pixel distance beyond which the LongHoverStop
	// event is sent
	LongHoverStopDist = 5
)

// // MgrActive indicates current active state
// type MgrActive int32 //enums:enum
//
// const (
// 	// NothingActive indicates no active state
// 	NothingActive MgrActive = iota
//
// 	Pressing
//
// 	Dragging
//
// 	Sliding
// )

// EventMgr is an event manager that handles incoming events for a
// MainStage object (Window, Dialog, Sheet).  It distributes events
// to a Scene based on position or focus, and deals with more complex
// cases such as dragging, drag-n-drop, and hovering.
type EventMgr struct {

	// Stage is the owning MainStage that we manage events for
	Main *MainStage

	// mutex that protects timer variable updates (e.g., hover AfterFunc's)
	TimerMu sync.Mutex `desc:"mutex that protects timer variable updates (e.g., hover AfterFunc's)"`

	// Current active state
	// Active MgrActive

	// stack of hovered widgets: have mouse pointer in BBox and have Hoverable flag
	Hovers []Widget

	// stack of drag-hovered widgets: have mouse pointer in BBox and have Droppable flag
	DragHovers []Widget

	// node that was just pressed
	Press Widget

	// node receiving mouse dragging events -- for drag-n-drop
	Drag Widget

	// node receiving mouse sliding events
	Slide Widget

	// node receiving mouse scrolling events
	Scroll Widget

	// stage of DND process
	// DNDStage DNDStages `desc:"stage of DND process"`
	//
	// // drag-n-drop data -- if non-nil, then DND is taking place
	// DNDData mimedata.Mimes `desc:"drag-n-drop data -- if non-nil, then DND is taking place"`
	//
	// // drag-n-drop source node
	// DNDSource Widget `desc:"drag-n-drop source node"`

	// 	// final event for DND which is sent if a finalize is received
	// 	DNDFinalEvent events.Event `desc:"final event for DND which is sent if a finalize is received"`
	//
	// 	// modifier in place at time of drop event (DropMove or DropCopy)
	// 	DNDDropMod events.DropMods `desc:"modifier in place at time of drop event (DropMove or DropCopy)"`

	// node receiving keyboard events -- use SetFocus, CurFocus
	Focus Widget `desc:"node receiving keyboard events -- use SetFocus, CurFocus"`

	// mutex that protects focus updating
	FocusMu sync.RWMutex `desc:"mutex that protects focus updating"`

	// stack of focus
	FocusStack []Widget `desc:"stack of focus"`

	// node to focus on at start when no other focus has been set yet -- use SetStartFocus
	StartFocus Widget `desc:"node to focus on at start when no other focus has been set yet -- use SetStartFocus"`

	// Last Select Mode from most recent Mouse, Keyboard events
	LastSelMode events.SelectModes `desc:"Last Select Mode from most recent Mouse, Keyboard events"`

	/*
		startDrag       events.Event
		dragStarted     bool
		startDND        events.Event
		dndStarted      bool
		startHover      events.Event
		curHover        events.Event
		hoverStarted    bool
		hoverTimer      *time.Timer
		startDNDHover   events.Event
		curDNDHover     events.Event
		dndHoverStarted bool
		dndHoverTimer   *time.Timer
	*/
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

func (em *EventMgr) HandleEvent(sc *Scene, evi events.Event) {
	// et := evi.Type()
	// fmt.Printf("got event type: %v: %v\n", et, evi)

	switch {
	case evi.HasPos():
		em.HandlePosEvent(sc, evi)
	case evi.NeedsFocus():
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

func (em *EventMgr) HandleOtherEvent(sc *Scene, evi events.Event) {
}

func (em *EventMgr) HandleFocusEvent(sc *Scene, evi events.Event) {
	if em.Focus == nil {
		fmt.Println("no focus")
		return
	}
	fi := em.Focus
	if win := em.RenderWin(); win != nil {
		if !win.IsFocusActive() { // reactivate on keyboard input
			win.SetFocusActive(true)
			if EventTrace {
				fmt.Printf("Event: set focus active, was not: %v\n", fi.Path())
			}
			fi.FocusChanged(FocusActive)
		}
	}
	fmt.Println("foc", fi.Path(), evi.String())
	fi.HandleEvent(evi)
}

func (em *EventMgr) ResetOnMouseDown() {
	em.Press = nil
	em.Drag = nil
	em.Slide = nil
}

func (em *EventMgr) HandlePosEvent(sc *Scene, evi events.Event) {
	pos := evi.LocalPos()
	et := evi.Type()

	isDrag := false
	switch et {
	case events.MouseDown:
		em.ResetOnMouseDown()
	case events.MouseDrag:
		switch {
		case em.Drag != nil:
			isDrag = true
			em.Drag.HandleEvent(evi)
			em.Drag.Send(events.DragMove, evi)
			// still needs to handle dragenter / leave
		case em.Slide != nil:
			em.Slide.HandleEvent(evi)
			em.Slide.Send(events.SlideMove, evi)
			return // nothing further
		case em.Press != nil:
			// todo: distance to start sliding, dragging
		}
	case events.Scroll:
		switch {
		case em.Scroll != nil:
			em.Scroll.HandleEvent(evi)
			return
		}
	}

	var allbb []Widget
	em.AllInBBox(&sc.Frame, &allbb, pos)

	n := len(allbb)
	if n == 0 {
		return
	}

	var press, move, up Widget
	for i := n - 1; i >= 0; i-- {
		w := allbb[i]
		if !isDrag {
			w.HandleEvent(evi) // everyone gets the primary event who is in scope, deepest first
		}
		wb := w.AsWidget()
		switch et {
		case events.MouseMove:
			if move == nil && wb.Style.Abilities.IsHoverable() {
				move = w
			}
		case events.MouseDown:
			if press == nil && wb.Style.Abilities.IsPressable() {
				press = w
			}
		case events.MouseUp:
			if up == nil && wb.Style.Abilities.IsPressable() {
				up = w
			}
		}
	}
	switch et {
	case events.MouseDown:
		if press != nil {
			em.Press = press
		}
	case events.MouseMove:
		hovs := make([]Widget, 0, len(allbb))
		for _, w := range allbb { // requires forward iter through allbb
			wb := w.AsWidget()
			if wb.Style.Abilities.IsHoverable() {
				hovs = append(hovs, w)
			}
		}
		em.Hovers = em.UpdateHovers(hovs, em.Hovers, evi, events.MouseEnter, events.MouseLeave)
	case events.MouseDrag:
		switch {
		case em.Drag != nil:
			hovs := make([]Widget, 0, len(allbb))
			for _, w := range allbb { // requires forward iter through allbb
				wb := w.AsWidget()
				if wb.AbilityIs(states.Droppable) {
					hovs = append(hovs, w)
				}
			}
			em.DragHovers = em.UpdateHovers(hovs, em.DragHovers, evi, events.DragEnter, events.DragLeave)
		case em.Slide != nil:
		case em.Press != nil && em.Press.AbilityIs(states.Slideable):
			if em.DragStartCheck(evi, SlideStartTime, SlideStartDist) {
				em.Slide = em.Press
				em.Slide.Send(events.SlideStart, evi)
			}
		case em.Press != nil && em.Press.AbilityIs(states.Draggable):
			if em.DragStartCheck(evi, DragStartTime, DragStartDist) {
				em.Drag = em.Press
				em.Drag.Send(events.DragStart, evi)
			}
		}
	case events.MouseUp:
		switch {
		case em.Slide != nil:
			em.Slide.Send(events.SlideStop, evi)
			em.Slide = nil
		case em.Drag != nil:
			em.Drag.Send(events.Drop, evi) // todo: all we need or what?
			em.Drag = nil
		case em.Press == up && up != nil:
			up.Send(events.Click, evi)
		}
		em.Press = nil
	}

}

// UpdateHovers updates the hovered widgets based on current
// widgets in bounding box.
func (em *EventMgr) UpdateHovers(hov, prev []Widget, evi events.Event, enter, leave events.Types) []Widget {
	for _, prv := range em.Hovers {
		stillIn := false
		for _, cur := range hov {
			if prv == cur {
				stillIn = true
				break
			}
		}
		if !stillIn {
			prv.Send(events.MouseLeave, evi)
		}
	}

	for _, cur := range hov {
		wasIn := false
		for _, prv := range em.Hovers {
			if prv == cur {
				wasIn = true
				break
			}
		}
		if !wasIn {
			cur.Send(events.MouseEnter, evi)
		}
	}
	// todo: detect change in top one, use to update cursor
	return hov
}

func (em *EventMgr) AllInBBox(w Widget, allbb *[]Widget, pos image.Point) {
	w.WalkPre(func(k ki.Ki) bool {
		wi, wb := AsWidget(k)
		if wb == nil || wb.Is(ki.Deleted) || wb.Is(ki.Destroyed) {
			return ki.Break
		}
		if !wb.PosInBBox(pos) {
			return ki.Break
		}
		*allbb = append(*allbb, wi)
		if wb.Parts != nil {
			em.AllInBBox(wb.Parts, allbb, pos)
		}
		return ki.Continue
	})
}

func (em *EventMgr) DragStartCheck(evi events.Event, dur time.Duration, dist int) bool {
	since := evi.SinceStart()
	if since < dur {
		return false
	}
	dst := int(mat32.NewVec2FmPoint(evi.StartDelta()).Length())
	if dst < dist {
		return false
	}
	return true
}

///////////////////////////////////////////////////////////////////
//   Key events

// SendKeyChordEvent sends a KeyChord event with given values.  If popup is
// true, then only items on popup are in scope, otherwise items NOT on popup
// are in scope (if no popup, everything is in scope).
// func (em *EventMgr) SendKeyChordEvent(popup bool, r rune, mods ...key.Modifiers) {
// 	ke := key.NewEvent(r, 0, key.Press, 0)
// 	ke.SetTime()
// 	// ke.SetModifiers(mods...)
// 	// em.HandleEvent(ke)
// }

// SendKeyFunEvent sends a KeyChord event with params from the given KeyFun.
// If popup is true, then only items on popup are in scope, otherwise items
// NOT on popup are in scope (if no popup, everything is in scope).
// func (em *EventMgr) SendKeyFunEvent(kf KeyFuns, popup bool) {
// 	chord := ActiveKeyMap.ChordForFun(kf)
// 	if chord == "" {
// 		return
// 	}
// 	r, mods, err := chord.Decode()
// 	if err != nil {
// 		return
// 	}
// 	ke := key.NewEvent(r, 0, key.Press, mods)
// 	ke.SetTime()
// 	// em.HandleEvent(&ke)
// }

// // CurFocus gets the current focus node under mutex protection
// func (em *EventMgr) CurFocus() Widget {
// 	em.FocusMu.RLock()
// 	defer em.FocusMu.RUnlock()
// 	return em.Focus
// }

// setFocusPtr JUST sets the focus pointer under mutex protection --
// use SetFocus for end-user setting of focus
// func (em *EventMgr) setFocusPtr(k Widget) {
// 	em.FocusMu.Lock()
// 	em.Focus = k
// 	em.FocusMu.Unlock()
// }

// SetFocus sets focus to given item -- returns true if focus changed.
// If item is nil, then nothing has focus.
func (em *EventMgr) SetFocus(w Widget) bool { // , evi events.Event
	cfoc := em.Focus
	if cfoc == nil || cfoc.This() == nil || cfoc.Is(ki.Deleted) || cfoc.Is(ki.Destroyed) {
		em.Focus = nil
		cfoc = nil
	}
	if cfoc == w {
		return false
	}

	if cfoc != nil {
		cfoc.Send(events.FocusLost, nil)
	}
	em.Focus = w
	if w != nil {
		w.Send(events.Focus, nil)
	}
	return true
}

// FocusNext sets the focus on the next item
// that can accept focus after the given item (can be nil).
// returns true if a focus item found.
func (em *EventMgr) FocusNext(foc Widget) bool {
	gotFocus := false
	focusNext := false // get the next guy
	if foc == nil {
		focusNext = true
	}

	focRoot := em.Main.Scene.Frame.This().(Widget)

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
			if !wb.AbilityIs(states.Focusable) {
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
	cfoc := em.Focus
	if cfoc == foc {
		return true
	}
	_, wb := AsWidget(foc)
	if wb == nil || wb.This() == nil {
		return false
	}
	if wb.AbilityIs(states.Focusable) {
		em.SetFocus(foc)
		return true
	}
	return em.FocusNext(foc)
}

// FocusOnOrPrev sets the focus on the given item, or the previous one that can
// accept focus -- returns true if a new focus item found.
func (em *EventMgr) FocusOnOrPrev(foc Widget) bool {
	cfoc := em.Focus
	if cfoc == foc {
		return true
	}
	_, wb := AsWidget(foc)
	if wb == nil || wb.This() == nil {
		return false
	}
	if wb.AbilityIs(states.Focusable) {
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

	focRoot := em.Main.Scene.Frame.This().(Widget)

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
		if !wb.AbilityIs(states.Focusable) {
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

	focRoot := em.Main.Scene.Frame.This().(Widget)

	focRoot.WalkPre(func(k ki.Ki) bool {
		wi, wb := AsWidget(k)
		if wb == nil || wb.This() == nil {
			return ki.Continue
		}
		if !wb.AbilityIs(states.Focusable) {
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
	focRoot := em.Main.Scene.Frame.This().(Widget)

	focRoot.WalkPre(func(k ki.Ki) bool {
		wi, wb := AsWidget(k)
		if wi == focRoot { // skip top-level
			return ki.Continue
		}
		if wb == nil || wb.This() == nil {
			return ki.Continue
		}
		if foc == wi {
			return ki.Continue
		}
		if wb.StateIs(states.Focused) {
			if EventTrace {
				fmt.Printf("ClearNonFocus: had focus: %v\n", wb.Path())
			}
			wi.Send(events.FocusLost, nil)
		}
		return ki.Continue
	})
}

// PushFocus pushes current focus onto stack and sets new focus.
func (em *EventMgr) PushFocus(p Widget) {
	// em.FocusMu.Lock()
	if em.FocusStack == nil {
		em.FocusStack = make([]Widget, 0, 50)
	}
	em.FocusStack = append(em.FocusStack, em.Focus)
	em.Focus = nil // don't un-focus on prior item when pushing
	// em.FocusMu.Unlock()
	em.FocusOnOrNext(p)
}

// PopFocus pops off the focus stack and sets prev to current focus.
func (em *EventMgr) PopFocus() {
	// em.FocusMu.Lock()
	if em.FocusStack == nil || len(em.FocusStack) == 0 {
		em.Focus = nil
		return
	}
	sz := len(em.FocusStack)
	em.Focus = nil
	nxtf := em.FocusStack[sz-1]
	_, wb := AsWidget(nxtf)
	if wb != nil && wb.This() != nil {
		// em.FocusMu.Unlock()
		em.SetFocus(nxtf)
		// em.FocusMu.Lock()
	}
	em.FocusStack = em.FocusStack[:sz-1]
	// em.FocusMu.Unlock()
}

// SetStartFocus sets the given item to be first focus when window opens.
func (em *EventMgr) SetStartFocus(k Widget) {
	// em.FocusMu.Lock()
	em.StartFocus = k
	// em.FocusMu.Unlock()
}

// ActivateStartFocus activates start focus if there is no current focus
// and StartFocus is set -- returns true if activated
func (em *EventMgr) ActivateStartFocus() bool {
	// em.FocusMu.RLock()
	if em.StartFocus == nil {
		em.FocusMu.RUnlock()
		return false
	}
	// em.FocusMu.RUnlock()
	// em.FocusMu.Lock()
	sf := em.StartFocus
	em.StartFocus = nil
	// em.FocusMu.Unlock()
	em.FocusOnOrNext(sf)
	return true
}

// InitialFocus establishes the initial focus for the window if no focus
// is set -- uses ActivateStartFocus or FocusNext as backup.
func (em *EventMgr) InitialFocus() {
	if em.Focus == nil {
		if !em.ActivateStartFocus() {
			em.FocusNext(em.Focus)
		}
	}
}

/*
///////////////////////////////////////////////////////////////////
//   Manager-level event processing

// MangerKeyChordEvents handles lower-priority manager-level key events.
// Mainly tab, shift-tab, and GoGiEditor and Prefs.
// event will be marked as processed if handled here.
func (em *EventMgr) ManagerKeyChordEvents(e events.Event) {
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
*/

/*

// MouseDragEvents processes MouseDragEvent to Detect start of drag and EVEnts.
// These require timing and delays, e.g., due to minor wiggles when pressing
// the mouse button
func (em *EventMgr) MouseDragEvents(evi events.Event) {
	me := evi.(events.Event)
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
					if dst >= DragStartDist {
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
			delayMs := int(now.Sub(em.startEVEnts.Time()) / time.Millisecond)
			if delayMs >= DNDStartMSec {
				dst := int(mat32.Hypot(float32(em.startEVEnts.Where.X-me.Pos().X), float32(em.startEVEnts.Where.Y-me.Pos().Y)))
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
		scev := events.NewScrollEvent(me.Pos(), me.Pos().Sub(me.Start).Mul(-1), me.Mods)
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
func (em *EventMgr) MouseMoveEvents(evi events.Event) {
	me := evi.(events.Event)
	em.LastModBits = me.Mods
	em.LastSelMode = me.SelectMode()
	em.LastMousePos = me.Pos()
	em.TimerMu.Lock()
	if !em.hoverStarted {
		em.hoverStarted = true
		em.startHover = me
		em.curHover = events.NewEventCopy(events.MouseHoverEvent, me)
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
			em.curHover = events.NewEventCopy(events.MouseHoverEvent, me)
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

// DoInstaDrag tests whether the given mouse DragEvent is on a widget marked
// with InstaDrag
func (em *EventMgr) DoInstaDrag(me events.Event, popup bool) bool {
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
	return ki.Break
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
func (em *EventMgr) DNDStartEvent(e events.Event) {
	de := events.NewEvent(events.Start, e.Pos(), e.Mods)
	de.Start = e.Pos()
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
func (em *EventMgr) SendDNDHoverEvent(e events.Event) {
	if e == nil {
		return
	}
	he := &events.Event{}
	he.EventBase = e.EventBase
	he.ClearHandled()
	he.Action = events.Hover
	// em.HandleEvent(&he)
}

// SendDNDMoveEvent sends DND move event
func (em *EventMgr) SendDNDMoveEvent(e events.Event) events.Event {
	// todo: when e.Pos() goes negative, transition to OS DND
	// todo: send move / enter / exit events to anyone listening
	de := &events.Event{}
	de.EventBase = e.EventBase
	de.ClearHandled()
	de.DefaultMod() // based on current key modifiers
	de.Action = events.Move
	// em.HandleEvent(de)
	// em.GenDNDFocusEvents(de)
	return de
}

// SendDNDDropEvent sends DND drop event -- returns false if drop event was not processed
// in which case the event should be cleared (by the RenderWin)
func (em *EventMgr) SendDNDDropEvent(e events.Event) bool {
	de := &events.Event{}
	de.EventBase = e.EventBase
	de.ClearHandled()
	de.DefaultMod()
	de.Action = events.DropOnTarget
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

// GenDNDFocusEvents processes events.Event to generate events.FocusEvent
// events -- returns true if any such events were sent.  If popup is true,
// then only items on popup are in scope, otherwise items NOT on popup are in
// scope (if no popup, everything is in scope).  Extra work is done to ensure
// that Exit from prior widget is always sent before Enter to next one.
func (em *EventMgr) GenDNDFocusEvents(mev events.Event, popup bool) bool {
	fe := &events.Event{}
	*fe = *mev
	pos := mev.LocalPos()
	ftyp := events.DNDFocusEvent

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
		fe.Action = events.Exit
		for i := range outs {
			outs[i].Call(send, int64(ftyp), &fe)
		}
		fe.Action = events.Enter
		for i := range ins {
			ins[i].Call(send, int64(ftyp), &fe)
		}
		em.Master.EventTopUpdateEnd(updt)
		return ki.Continue
	}
	return ki.Break
}
*/
