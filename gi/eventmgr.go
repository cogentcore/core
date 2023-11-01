// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"path/filepath"
	"sync"
	"time"

	"goki.dev/cursors"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/grows/images"
	"goki.dev/grr"
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

	// LongHoverStopDist is the pixel distance beyond which the LongHoverEnd
	// event is sent
	LongHoverStopDist = 50

	// LongPressTime is the time to wait before sending a LongPress event
	LongPressTime = 500 * time.Millisecond
)

// note: EventMgr should be in _exclusive_ control of its own state
// and IF we end up needing a mutex, it should be global on main
// entry points (HandleEvent, anything else?)

// EventMgr is an event manager that handles incoming events for a Scene.
// It creates all the derived event types (Hover, Sliding, Dragging)
// and Focus management for keyboard events.
type EventMgr struct {

	// Scene is the scene that we manage events for
	Scene *Scene

	// mutex that protects timer variable updates (e.g., hover AfterFunc's)
	TimerMu sync.Mutex

	// stack of widgets with mouse pointer in BBox, and are not Disabled
	MouseInBBox []Widget

	// stack of hovered widgets: have mouse pointer in BBox and have Hoverable flag
	Hovers []Widget

	// the current candidate for a long hover event
	LongHoverWidget Widget

	// the position of the mouse at the start of LongHoverTimer
	LongHoverPos image.Point

	// the timer for the LongHover event, started with time.AfterFunc
	LongHoverTimer *time.Timer

	// LongPressWIdget is the current candidate for a long press event
	LongPressWidget Widget

	// LongPressTimer is the timer for the LongPress event, started with time.AfterFunc
	LongPressTimer *time.Timer

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

	// node receiving keyboard events -- use SetFocus, CurFocus
	Focus Widget

	// node to focus on at start when no other focus has been set yet -- use SetStartFocus
	StartFocus Widget

	// previously-focused widget -- what was in Focus when FocusClear is called
	PrevFocus Widget

	// stack of focus within elements
	FocusWithinStack []Widget

	// Last Select Mode from most recent Mouse, Keyboard events
	LastSelMode events.SelectModes

	// currently active shortcuts for this window (shortcuts are always window-wide -- use widget key event processing for more local key functions)
	Shortcuts Shortcuts `json:"-" xml:"-"`

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
	if em.Scene == nil {
		return nil
	}
	return em.Scene.MainStageMgr()
}

// RenderWin returns the overall render window, which could be nil
func (em *EventMgr) RenderWin() *RenderWin {
	mgr := em.MainStageMgr()
	if mgr == nil {
		return nil
	}
	return mgr.RenderWin
}

///////////////////////////////////////////////////////////////////////
// 	HandleEvent

func (em *EventMgr) HandleEvent(evi events.Event) {
	// et := evi.Type()
	// fmt.Printf("got event type: %v: %v\n", et, evi)
	if evi.IsHandled() {
		return
	}
	switch {
	case evi.HasPos():
		em.HandlePosEvent(evi)
	case evi.NeedsFocus():
		em.HandleFocusEvent(evi)
	default:
		em.HandleOtherEvent(evi)
	}
}

func (em *EventMgr) HandleOtherEvent(evi events.Event) {
	fmt.Println("TODO: Other event not handled", evi)
}

func (em *EventMgr) HandleFocusEvent(evi events.Event) {
	if em.Focus == nil {
		switch {
		case em.PrevFocus != nil:
			em.SetFocus(em.PrevFocus)
			em.PrevFocus = nil
		case em.StartFocus != nil:
			em.SetFocus(em.PrevFocus)
		default:
			em.FocusFirst()
		}
	}
	if em.Focus != nil {
		em.Focus.HandleEvent(evi)
	}
	if !evi.IsHandled() && em.FocusWithins() {
		for _, fw := range em.FocusWithinStack {
			fw.HandleEvent(evi)
			if evi.IsHandled() {
				break
			}
		}
	}
	em.ManagerKeyChordEvents(evi)
}

func (em *EventMgr) ResetOnMouseDown() {
	em.Press = nil
	em.Drag = nil
	em.Slide = nil
}

func (em *EventMgr) HandlePosEvent(evi events.Event) {
	pos := evi.LocalPos()
	et := evi.Type()
	sc := em.Scene

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
		}
	case events.Scroll:
		switch {
		case em.Scroll != nil:
			em.Scroll.HandleEvent(evi)
			return
		}
	}

	em.MouseInBBox = nil
	em.GetMouseInBBox(sc, pos)

	n := len(em.MouseInBBox)
	if n == 0 {
		if EventTrace && et != events.MouseMove {
			log.Println("Nothing in bbox:", sc.ScBBox, "pos:", pos)
		}
		return
	}

	var press, move, up Widget
	for i := n - 1; i >= 0; i-- {
		w := em.MouseInBBox[i]
		wb := w.AsWidget()

		// we need to handle this here and not in [EventMgr.GetMouseInBBox] so that
		// we correctly process cursors for disabled elements.
		if wb.StateIs(states.Disabled) {
			continue
		}

		if !isDrag {
			w.HandleEvent(evi) // everyone gets the primary event who is in scope, deepest first
		}
		switch et {
		case events.MouseMove:
			if move == nil && wb.Styles.Abilities.IsHoverable() {
				move = w
			}
		case events.MouseDown:
			if press == nil && wb.Styles.Abilities.IsPressable() {
				press = w
			}
		case events.MouseUp:
			if up == nil && wb.Styles.Abilities.IsPressable() {
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
		hovs := make([]Widget, 0, len(em.MouseInBBox))
		for _, w := range em.MouseInBBox { // requires forward iter through em.MouseInBBox
			wb := w.AsWidget()
			if wb.Styles.Abilities.IsHoverable() {
				hovs = append(hovs, w)
			}
		}
		em.Hovers = em.UpdateHovers(hovs, em.Hovers, evi, events.MouseEnter, events.MouseLeave)
		if em.MainStageMgr() != nil && em.MainStageMgr().Top() != nil {
			top := em.MainStageMgr().Top().AsMain()
			if top != nil && top.Scene != nil {
				top.Scene.EventMgr.HandleLongHover(evi, em.TopLongHover())
			}
		}
	case events.MouseDrag:
		switch {
		case em.Drag != nil:
			hovs := make([]Widget, 0, len(em.MouseInBBox))
			for _, w := range em.MouseInBBox { // requires forward iter through em.MouseInBBox
				wb := w.AsWidget()
				if wb.AbilityIs(abilities.Droppable) {
					hovs = append(hovs, w)
				}
			}
			em.DragHovers = em.UpdateHovers(hovs, em.DragHovers, evi, events.DragEnter, events.DragLeave)
		case em.Slide != nil:
		case em.Press != nil && em.Press.AbilityIs(abilities.Slideable):
			if em.DragStartCheck(evi, SlideStartTime, SlideStartDist) {
				em.Slide = em.Press
				em.Slide.Send(events.SlideStart, evi)
			}
		case em.Press != nil && em.Press.AbilityIs(abilities.Draggable):
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
			switch evi.MouseButton() {
			case events.Left:
				up.Send(events.Click, evi)
			case events.Right: // note: automatically gets Control+Left
				up.Send(events.ContextMenu, evi)
			}
		}
		em.Press = nil
		// a mouse up event acts also acts as a mouse leave
		// and long hover end event on mobile, as those events
		// are needed to clear any hovered state and tooltip
		if goosi.TheApp.Platform().IsMobile() {
			up.Send(events.MouseLeave, evi)
			up.Send(events.LongHoverEnd, evi)
		}
	case events.Scroll:
		switch {
		case em.Slide != nil:
			em.Slide.HandleEvent(evi)
		case em.Drag != nil:
			em.Drag.HandleEvent(evi)
		case em.Press != nil:
			em.Press.HandleEvent(evi)
		default:
			em.Scene.HandleEvent(evi)
		}
	}

	// we need to handle cursor after all of the events so that
	// we get the latest cursor if it changes based on the state

	cursorSet := false
	for i := n - 1; i >= 0; i-- {
		w := em.MouseInBBox[i]
		wb := w.AsWidget()
		if !cursorSet && wb.Styles.Cursor != cursors.None {
			em.SetCursor(wb.Styles.Cursor)
			cursorSet = true
		}
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
		if !stillIn && prv.This() != nil && !prv.Is(ki.Deleted) {
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

// TopLongHover returns the top-most LongHoverable among the Hovers
func (em *EventMgr) TopLongHover() Widget {
	var deep Widget
	for i := len(em.Hovers) - 1; i >= 0; i-- {
		h := em.Hovers[i]
		if h.AbilityIs(abilities.LongHoverable) {
			deep = h
			break
		}
	}
	return deep
}

// HandleLongHover handles long hover events
func (em *EventMgr) HandleLongHover(evi events.Event, deep Widget) {
	em.HandleLong(evi, deep, &em.LongHoverWidget, &em.LongHoverPos, &em.LongHoverTimer, events.LongHoverStart, events.LongHoverEnd, LongHoverTime, LongHoverStopDist)
}

// HandleLong is the implementation of [EventMgr.HandleLongHover] and
// [EventManger.HandleLongPress]. It handles the logic to do with tracking
// long events using the given pointers to event manager fields and
// constant type, time, and distance properties. It should not need to
// be called by anything except for the aforementioned functions.
func (em *EventMgr) HandleLong(evi events.Event, deep Widget, w *Widget, pos *image.Point, t **time.Timer, styp, etyp events.Types, stime time.Duration, sdist int) {
	em.TimerMu.Lock()
	defer em.TimerMu.Unlock()

	// fmt.Println("em:", em.Scene.Name())

	clearLongHover := func() {
		if *t != nil {
			(*t).Stop() // TODO: do we need to close this?
			*t = nil
		}
		*w = nil
		*pos = image.Point{}
		// fmt.Println("cleared hover")
	}

	cpos := evi.Pos()
	dst := int(mat32.Hypot(float32(pos.X-cpos.X), float32(pos.Y-cpos.Y)))
	// fmt.Println("dist:", dst)

	// we have no long hovers, so we must be done
	if deep == nil {
		// fmt.Println("no deep")
		if *w == nil {
			// fmt.Println("no lhw")
			return
		}
		// if we have already finished the timer, then we have already
		// sent the start event, so we have to send the end one
		if *t == nil {
			(*w).Send(etyp, evi)
		}
		clearLongHover()
		// fmt.Println("cleared")
		return
	}

	// we still have the current one, so there is nothing to do
	// but make sure our position hasn't changed too much
	if deep == *w {
		// if we haven't gone too far, we have nothing to do
		if dst <= sdist {
			// fmt.Println("bail on dist:", dst)
			return
		}
		// If we have gone too far, we are done with the long hover and
		// we must clear it. However, critically, we do not return, as
		// we must make a new tooltip immediately; otherwise, we may end
		// up not getting another mouse move event, so we will be on the
		// element with no tooltip, which is a bug. Not returning here is
		// the solution to https://github.com/goki/gi/issues/553
		(*w).Send(etyp, evi)
		clearLongHover()
		// fmt.Println("fallthrough after clear")
	}

	// if we have changed and still have the timer, we never
	// sent a start event, so we just bail
	if *t != nil {
		clearLongHover()
		// fmt.Println("timer non-nil, cleared")
		return
	}

	// we now know we don't have the timer and thus sent the start
	// event already, so we need to send a end event
	if *w != nil {
		(*w).Send(etyp, evi)
		clearLongHover()
		// fmt.Println("lhw, send end, cleared")
		return
	}

	// now we can set it to our new widget
	*w = deep
	// fmt.Println("setting new:", deep)
	*pos = evi.Pos()
	*t = time.AfterFunc(stime, func() {
		em.TimerMu.Lock()
		defer em.TimerMu.Unlock()
		if *w == nil {
			return
		}
		(*w).Send(styp, evi)
		// we are done with the timer, and this indicates that
		// we have sent a start event
		*t = nil
	})
}

func (em *EventMgr) GetMouseInBBox(w Widget, pos image.Point) {
	w.WalkPre(func(k ki.Ki) bool {
		wi, wb := AsWidget(k)
		// we do not handle disabled here so that
		// we correctly process cursors for disabled elements.
		// it needs to be handled downstream by anyone who needs it.
		if !wb.IsVisible() {
			return ki.Break
		}
		if !wb.PosInScBBox(pos) {
			return ki.Break
		}
		// fmt.Println("in bb:", wi, wb.Styles.State)
		em.MouseInBBox = append(em.MouseInBBox, wi)
		if wb.Parts != nil {
			em.GetMouseInBBox(wb.Parts, pos)
		}
		ly := AsLayout(k)
		if ly != nil {
			for d := mat32.X; d <= mat32.Y; d++ {
				if ly.HasScroll[d] {
					sb := ly.Scrolls[d]
					em.GetMouseInBBox(sb, pos)
				}
			}
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
	return dst >= dist
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

// Sendkeyfun.Event sends a KeyChord event with params from the given keyfun..
// If popup is true, then only items on popup are in scope, otherwise items
// NOT on popup are in scope (if no popup, everything is in scope).
// func (em *EventMgr) Sendkeyfun.Event(kf keyfun.Funs, popup bool) {
// 	chord := ActiveKeyMap.ChordFor(kf)
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

// ClipBoard returns the goosi clip.Board, supplying the window context
// if available.
func (em *EventMgr) ClipBoard() clip.Board {
	var gwin goosi.Window
	if win := em.RenderWin(); win != nil {
		gwin = win.GoosiWin
	}
	return goosi.TheApp.ClipBoard(gwin)
}

// SetCursor sets window cursor to given Cursor
func (em *EventMgr) SetCursor(cur cursors.Cursor) {
	win := em.RenderWin()
	if win == nil {
		return
	}
	if win.Is(WinClosing) {
		return
	}
	grr.Log0(goosi.TheApp.Cursor(win.GoosiWin).Set(cur))
}

// FocusClear saves current focus to FocusPrev
func (em *EventMgr) FocusClear() bool {
	if em.Focus != nil {
		em.PrevFocus = em.Focus
	}
	return em.GrabFocus(nil)
}

// GrabFocus sets focus to given item -- returns true if focus changed.
// If item is nil, then nothing has focus.
// This does NOT send the events.Focus event to the widget.
func (em *EventMgr) GrabFocus(w Widget) bool {
	return em.SetFocusImpl(w, false) // no event
}

// SetFocus sets focus to given item -- returns true if focus changed.
// If item is nil, then nothing has focus.
// This sends the events.Focus event to the widget -- see GrabFocus
// for a version that does not.
func (em *EventMgr) SetFocus(w Widget) bool {
	return em.SetFocusImpl(w, true) // sends event
}

// SetFocusImpl sets focus to given item -- returns true if focus changed.
// If item is nil, then nothing has focus.
// sendEvent determines whether the events.Focus event is sent to the focused item.
func (em *EventMgr) SetFocusImpl(w Widget, sendEvent bool) bool {
	cfoc := em.Focus
	if cfoc == nil || cfoc.This() == nil || cfoc.Is(ki.Deleted) {
		em.Focus = nil
		cfoc = nil
	}
	if cfoc == w {
		return false
	}
	if KeyEventTrace {
		fmt.Println("EventMgr set focus to:", w)
	}

	if cfoc != nil {
		cfoc.Send(events.FocusLost)
	}
	em.Focus = w
	if sendEvent && w != nil {
		w.Send(events.Focus)
	}
	return true
}

// FocusWithins gets the FocusWithin containers of the current Focus event
func (em *EventMgr) FocusWithins() bool {
	em.FocusWithinStack = nil
	if em.Focus == nil {
		return false
	}
	em.Focus.WalkUpParent(func(k ki.Ki) bool {
		wi, wb := AsWidget(k)
		if !wb.IsVisible() {
			return ki.Break
		}
		if wb.AbilityIs(abilities.FocusWithinable) {
			em.FocusWithinStack = append(em.FocusWithinStack, wi)
		}
		return ki.Continue
	})
	return true
}

// FocusNext sets the focus on the next item
// that can accept focus after the current Focus item.
// returns true if a focus item found.
func (em *EventMgr) FocusNext() bool {
	foc := em.Focus
	gotFocus := false
	focusNext := false // get the next guy
	if foc == nil {
		focusNext = true
	}

	focRoot := em.Scene

	for i := 0; i < 2; i++ {
		focRoot.WalkPre(func(k ki.Ki) bool {
			if gotFocus {
				return ki.Break
			}
			wi, wb := AsWidget(k)
			if !wb.IsVisible() {
				return ki.Continue
			}
			if foc == wi { // current focus can be a non-can-focus item
				focusNext = true
				return ki.Continue
			}
			if !focusNext {
				return ki.Continue
			}
			if !wi.AbilityIs(abilities.Focusable) {
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
	if !wb.IsVisible() {
		return false
	}
	if wb.AbilityIs(abilities.Focusable) {
		em.SetFocus(foc)
		return true
	}
	em.Focus = foc
	return em.FocusNext()
}

// FocusOnOrPrev sets the focus on the given item, or the previous one that can
// accept focus -- returns true if a new focus item found.
func (em *EventMgr) FocusOnOrPrev(foc Widget) bool {
	cfoc := em.Focus
	if cfoc == foc {
		return true
	}
	_, wb := AsWidget(foc)
	if !wb.IsVisible() {
		return false
	}
	if wb.AbilityIs(abilities.Focusable) {
		em.SetFocus(foc)
		return true
	}
	em.Focus = foc
	return em.FocusPrev()
}

// FocusPrev sets the focus on the previous item before the given item (can be nil)
func (em *EventMgr) FocusPrev() bool {
	foc := em.Focus
	if foc == nil { // must have a current item here
		em.FocusLast()
		return false
	}

	gotFocus := false
	var prevItem Widget

	focRoot := em.Scene

	focRoot.WalkPre(func(k ki.Ki) bool {
		if gotFocus {
			return ki.Break
		}
		wi, wb := AsWidget(k)
		if !wb.IsVisible() {
			return ki.Continue
		}
		if foc == wi {
			gotFocus = true
			return ki.Break
		}
		if !wb.AbilityIs(abilities.Focusable) {
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

// FocusFirst sets the focus on the first focusable item in the tree.
// returns true if a focusable item was found.
func (em *EventMgr) FocusFirst() bool {
	var firstItem Widget
	focRoot := em.Scene

	focRoot.WalkPre(func(k ki.Ki) bool {
		wi, wb := AsWidget(k)
		if !wb.IsVisible() {
			return ki.Continue
		}
		if !wb.AbilityIs(abilities.Focusable) {
			return ki.Continue
		}
		firstItem = wi
		return ki.Break
	})
	em.SetFocus(firstItem)
	return firstItem != nil
}

// FocusLast sets the focus on the last focusable item in the tree.
// returns true if a focusable item was found.
func (em *EventMgr) FocusLast() bool {
	var lastItem Widget
	focRoot := em.Scene

	// todo: could use walking functions in ki
	focRoot.WalkPre(func(k ki.Ki) bool {
		wi, wb := AsWidget(k)
		if !wb.IsVisible() {
			return ki.Continue
		}
		if !wb.AbilityIs(abilities.Focusable) {
			return ki.Continue
		}
		lastItem = wi
		return ki.Continue
	})
	em.SetFocus(lastItem)
	return lastItem != nil
}

// ClearNonFocus clears the focus of any non-w.Focus item.
func (em *EventMgr) ClearNonFocus(foc Widget) {
	focRoot := em.Scene

	focRoot.WalkPre(func(k ki.Ki) bool {
		wi, wb := AsWidget(k)
		if wi == focRoot { // skip top-level
			return ki.Continue
		}
		if !wb.IsVisible() {
			return ki.Continue
		}
		if foc == wi {
			return ki.Continue
		}
		if wb.StateIs(states.Focused) {
			if EventTrace {
				fmt.Printf("ClearNonFocus: had focus: %v\n", wb.Path())
			}
			wi.Send(events.FocusLost)
		}
		return ki.Continue
	})
}

// SetStartFocus sets the given item to be first focus when window opens.
func (em *EventMgr) SetStartFocus(k Widget) {
	em.StartFocus = k
}

// ActivateStartFocus activates start focus if there is no current focus
// and StartFocus is set -- returns true if activated
func (em *EventMgr) ActivateStartFocus() bool {
	if em.StartFocus == nil {
		return false
	}
	sf := em.StartFocus
	em.StartFocus = nil
	em.GrabFocus(sf)
	return true
}

// InitialFocus establishes the initial focus for the window if no focus
// is set -- uses ActivateStartFocus or FocusNext as backup.
func (em *EventMgr) InitialFocus() {
	if em.Focus == nil {
		if !em.ActivateStartFocus() {
			em.FocusNext()
		}
	}
}

// MangerKeyChordEvents handles lower-priority manager-level key events.
// Mainly tab, shift-tab, and GoGiEditor and Prefs.
// event will be marked as processed if handled here.
func (em *EventMgr) ManagerKeyChordEvents(e events.Event) {
	if e.IsHandled() {
		return
	}
	if e.Type() != events.KeyChord {
		return
	}
	win := em.RenderWin()
	if win == nil {
		return
	}
	sc := em.Scene
	cs := e.KeyChord()
	kf := keyfun.Of(cs)
	// fmt.Println(kf, cs)
	switch kf {
	case keyfun.GoGiEditor:
		TheViewIFace.GoGiEditor(em.Scene)
		e.SetHandled()
	case keyfun.Prefs:
		TheViewIFace.PrefsView(&Prefs)
		e.SetHandled()
	case keyfun.WinClose:
		win.CloseReq()
		e.SetHandled()
	case keyfun.Menu:
		if win.MainMenu != nil {
			win.MainMenu.GrabFocus()
			e.SetHandled()
		}
	case keyfun.WinSnapshot:
		dstr := time.Now().Format("Mon_Jan_2_15:04:05_MST_2006")
		fnm, _ := filepath.Abs("./GrabOf_" + sc.Name() + "_" + dstr + ".png")
		images.Save(sc.Pixels, fnm)
		fmt.Printf("Saved RenderWin Image to: %s\n", fnm)
		e.SetHandled()
	case keyfun.ZoomIn:
		win.ZoomDPI(1)
		e.SetHandled()
	case keyfun.ZoomOut:
		win.ZoomDPI(-1)
		e.SetHandled()
	case keyfun.Refresh:
		e.SetHandled()
		fmt.Printf("Win: %v display refreshed\n", sc.Name())
		goosi.TheApp.GetScreens()
		Prefs.UpdateAll()
		WinGeomMgr.RestoreAll()
		// w.FocusInactivate()
		// w.FullReRender()
		// sz := w.GoosiWin.Size()
		// w.SetSize(sz)
	case keyfun.WinFocusNext:
		e.SetHandled()
		AllRenderWins.FocusNext()
	}
	switch cs { // some other random special codes, during dev..
	case "Control+Alt+R":
		ProfileToggle()
		e.SetHandled()
	case "Control+Alt+F":
		sc.BenchmarkFullRender()
		e.SetHandled()
	case "Control+Alt+H":
		sc.BenchmarkReRender()
		e.SetHandled()
	}
	if !e.IsHandled() {
		em.TriggerShortcut(cs)
	}
}

// AddShortcut adds given shortcut to given button.
func (em *EventMgr) AddShortcut(chord key.Chord, bt *Button) {
	if chord == "" {
		return
	}
	if em.Shortcuts == nil {
		em.Shortcuts = make(Shortcuts, 100)
	}
	sa, exists := em.Shortcuts[chord]
	if exists && sa != bt && sa.Text != bt.Text {
		if KeyEventTrace {
			log.Printf("gi.RenderWin shortcut: %v already exists on button: %v -- will be overwritten with button: %v\n", chord, sa.Text, bt.Text)
		}
	}
	em.Shortcuts[chord] = bt
}

// DeleteShortcut deletes given shortcut
func (em *EventMgr) DeleteShortcut(chord key.Chord, bt *Button) {
	if chord == "" {
		return
	}
	if em.Shortcuts == nil {
		return
	}
	sa, exists := em.Shortcuts[chord]
	if exists && sa == bt {
		delete(em.Shortcuts, chord)
	}
}

// TriggerShortcut attempts to trigger a shortcut, returning true if one was
// triggered, and false otherwise.  Also eliminates any shortcuts with deleted
// buttons, and does not trigger for Disabled buttons.
func (em *EventMgr) TriggerShortcut(chord key.Chord) bool {
	if KeyEventTrace {
		fmt.Printf("Shortcut chord: %v -- looking for button\n", chord)
	}
	if em.Shortcuts == nil {
		return false
	}
	sa, exists := em.Shortcuts[chord]
	if !exists {
		return false
	}
	if sa.Is(ki.Destroyed) {
		delete(em.Shortcuts, chord)
		return false
	}
	if sa.IsDisabled() {
		if KeyEventTrace {
			fmt.Printf("Shortcut chord: %v, button: %v -- is inactive, not fired\n", chord, sa.Text)
		}
		return false
	}

	if KeyEventTrace {
		fmt.Printf("Shortcut chord: %v, button: %v triggered\n", chord, sa.Text)
	}
	sa.Send(events.Click)
	return true
}

// TODO: all of the code below should be deleted once the corresponding DND
// functionality has been implemented in a much cleaner way.  Most of the
// logic should already be in place above.  Just need to check drop targets,
// update cursor, grab the initial sprite, etc.

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
					if wb.PosInScBBox(pos) {
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
				in := wb.PosInScBBox(pos)
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

/*
/////////////////////////////////////////////////////////////////////////////
//   Window level DND: Drag-n-Drop

const DNDSpriteName = "gi.RenderWin:DNDSprite"

// StartDragNDrop is called by a node to start a drag-n-drop operation on
// given source node, which is responsible for providing the data and Sprite
// representation of the node.
func (w *RenderWin) StartDragNDrop(src ki.Ki, data mimedata.Mimes, sp *Sprite) {
	w.EventMgr.DNDStart(src, data)
	if _, sw := AsWidget(src); sw != nil {
		sp.SetBottomPos(sw.LayState.Alloc.Pos.ToPo)
	}
	w.DeleteSprite(DNDSpriteName)
	sp.Name = DNDSpriteName
	sp.On = true
	w.AddSprite(sp)
	w.DNDSetCursor(dnd.DefaultModBits(w.EventMgr.LastModBits))
}

// DNDMoveEvent handles drag-n-drop move events.
func (w *RenderWin) DNDMoveEvent(e events.Event) {
	sp, ok := w.SpriteByName(DNDSpriteName)
	if ok {
		sp.SetBottomPos(e.Pos())
	}
	de := w.EventMgr.SendDNDMoveEvent(e)
	w.DNDUpdateCursor(de.Mod)
	e.SetHandled()
}

// DNDDropEvent handles drag-n-drop drop event (action = release).
func (w *RenderWin) DNDDropEvent(e events.Event) {
	proc := w.EventMgr.SendDNDDropEvent(e)
	if !proc {
		w.ClearDragNDrop()
	}
}

// FinalizeDragNDrop is called by a node to finalize the drag-n-drop
// operation, after given action has been performed on the target -- allows
// target to cancel, by sending dnd.DropIgnore.
func (w *RenderWin) FinalizeDragNDrop(action dnd.DropMods) {
	if w.EventMgr.DNDStage != DNDDropped {
		w.ClearDragNDrop()
		return
	}
	if w.EventMgr.DNDFinalEvent == nil { // shouldn't happen...
		w.ClearDragNDrop()
		return
	}
	de := w.EventMgr.DNDFinalEvent
	de.ClearHandled()
	de.Mod = action
	if de.Source != nil {
		de.Action = dnd.DropFmSource
		w.EventMgr.SendSig(de.Source, w, de)
	}
	w.ClearDragNDrop()
}

// ClearDragNDrop clears any existing DND values.
func (w *RenderWin) ClearDragNDrop() {
	w.EventMgr.ClearDND()
	w.DeleteSprite(DNDSpriteName)
	w.DNDClearCursor()
}

// DNDModCursor gets the appropriate cursor based on the DND event mod.
func DNDModCursor(dmod dnd.DropMods) cursor.Shapes {
	switch dmod {
	case dnd.DropCopy:
		return cursor.DragCopy
	case dnd.DropMove:
		return cursor.DragMove
	case dnd.DropLink:
		return cursor.DragLink
	}
	return cursor.Not
}

// DNDSetCursor sets the cursor based on the DND event mod -- does a
// "PushIfNot" so safe for multiple calls.
func (w *RenderWin) DNDSetCursor(dmod dnd.DropMods) {
	dndc := DNDModCursor(dmod)
	goosi.TheApp.Cursor(w.GoosiWin).PushIfNot(dndc)
}

// DNDNotCursor sets the cursor to Not = can't accept a drop
func (w *RenderWin) DNDNotCursor() {
	goosi.TheApp.Cursor(w.GoosiWin).PushIfNot(cursor.Not)
}

// DNDUpdateCursor updates the cursor based on the current DND event mod if
// different from current (but no update if Not)
func (w *RenderWin) DNDUpdateCursor(dmod dnd.DropMods) bool {
	dndc := DNDModCursor(dmod)
	curs := goosi.TheApp.Cursor(w.GoosiWin)
	if !curs.IsDrag() || curs.Current() == dndc {
		return false
	}
	curs.Push(dndc)
	return true
}

// DNDClearCursor clears any existing DND cursor that might have been set.
func (w *RenderWin) DNDClearCursor() {
	curs := goosi.TheApp.Cursor(w.GoosiWin)
	for curs.IsDrag() || curs.Current() == cursor.Not {
		curs.Pop()
	}
}

// HiProrityEvents processes High-priority events for RenderWin.
// RenderWin gets first crack at these events, and handles window-specific ones
// returns true if processing should continue and false if was handled
func (w *RenderWin) HiPriorityEvents(evi events.Event) bool {
	switch evi.(type) {
	case events.Event:
		// if w.EventMgr.DNDStage == DNDStarted {
		// 	w.DNDMoveEvent(e)
		// } else {
		// 	w.SelSpriteEvent(evi)
		// 	if !w.EventMgr.dragStarted {
		// 		e.SetHandled() // ignore
		// 	}
		// }
		// case events.Event:
		// if w.EventMgr.DNDStage == DNDStarted && e.Action == events.Release {
		// 	w.DNDDropEvent(e)
		// }
		// w.FocusActiveClick(e)
		// w.SelSpriteEvent(evi)
		// if w.NeedWinMenuUpdate() {
		// 	w.MainMenuUpdateRenderWins()
		// }
	// case *dnd.Event:
	// if e.Action == dnd.External {
	// 	w.EventMgr.DNDDropMod = e.Mod
	// }
	}
	return true
}

*/
