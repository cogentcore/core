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

	// LongPressStopDist is the pixel distance beyond which the LongPressEnd
	// event is sent
	LongPressStopDist = 50
)

const (
	DragSpriteName = "__DragSprite__"
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

	// stack of widgets with mouse pointer in BBox, and are not Disabled.
	// Last item in the stack is the deepest nested widget (smallest BBox).
	MouseInBBox []Widget

	// stack of hovered widgets: have mouse pointer in BBox and have Hoverable flag
	Hovers []Widget

	// the current candidate for a long hover event
	LongHoverWidget Widget

	// the position of the mouse at the start of LongHoverTimer
	LongHoverPos image.Point

	// the timer for the LongHover event, started with time.AfterFunc
	LongHoverTimer *time.Timer

	// the current candidate for a long press event
	LongPressWidget Widget

	// the position of the mouse at the start of LongPressTimer
	LongPressPos image.Point

	// the timer for the LongPress event, started with time.AfterFunc
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

	// if StartFocus not set, activate starting focus on first element
	StartFocusFirst bool

	// previously-focused widget -- what was in Focus when FocusClear is called
	PrevFocus Widget

	// stack of focus within elements
	FocusWithinStack []Widget

	// Last Select Mode from most recent Mouse, Keyboard events
	LastSelMode events.SelectModes

	// Currently active shortcuts for this window (shortcuts are always window-wide.
	// Use widget key event processing for more local key functions)
	Shortcuts Shortcuts

	// PriorityFocus are widgets with Focus PriorityEvents
	PriorityFocus []Widget

	// PriorityOther are widgets with other PriorityEvents types
	PriorityOther []Widget

	// source data from DragStart event
	DragData any
}

// MainStageMgr returns the MainStageMgr for our Main Stage
func (em *EventMgr) MainStageMgr() *StageMgr {
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

func (em *EventMgr) HandleEvent(e events.Event) {
	// et := evi.Type()
	// fmt.Printf("got event type: %v: %v\n", et, evi)
	if e.IsHandled() {
		return
	}
	switch {
	case e.HasPos():
		em.HandlePosEvent(e)
	case e.NeedsFocus():
		em.HandleFocusEvent(e)
	default:
		em.HandleOtherEvent(e)
	}
}

func (em *EventMgr) HandleOtherEvent(e events.Event) {
	fmt.Println("TODO: Other event not handled", e)
}

func (em *EventMgr) HandleFocusEvent(e events.Event) {
	if em.Focus == nil {
		switch {
		case em.StartFocus != nil:
			if FocusTrace {
				fmt.Println(em.Scene, "StartFocus:", em.StartFocus)
			}
			em.SetFocusEvent(em.StartFocus)
		case em.PrevFocus != nil:
			if FocusTrace {
				fmt.Println(em.Scene, "PrevFocus:", em.PrevFocus)
			}
			em.SetFocusEvent(em.PrevFocus)
			em.PrevFocus = nil
		default:
			em.FocusFirst()
		}
	}
	if em.PriorityFocus != nil {
		for _, wi := range em.PriorityFocus {
			wi.HandleEvent(e)
			if e.IsHandled() {
				if FocusTrace {
					fmt.Println(em.Scene, "PriorityFocus Handled:", wi)
				}
				break
			}
		}
	}
	if !e.IsHandled() && em.Focus != nil {
		em.Focus.HandleEvent(e)
	}
	if !e.IsHandled() && em.FocusWithins() {
		for _, fw := range em.FocusWithinStack {
			fw.HandleEvent(e)
			if e.IsHandled() {
				if FocusTrace {
					fmt.Println(em.Scene, "FocusWithin Handled:", fw)
				}
				break
			}
		}
	}
	em.ManagerKeyChordEvents(e)
}

func (em *EventMgr) ResetOnMouseDown() {
	em.Press = nil
	em.Drag = nil
	em.Slide = nil

	// if we have sent a long hover start event, we send an end
	// event (non-nil widget plus nil timer means we already sent)
	if em.LongHoverWidget != nil && em.LongHoverTimer == nil {
		em.LongHoverWidget.Send(events.LongHoverEnd)
	}
	em.LongHoverWidget = nil
	em.LongHoverPos = image.Point{}
	if em.LongHoverTimer != nil {
		em.LongHoverTimer.Stop()
		em.LongHoverTimer = nil
	}
}

func (em *EventMgr) HandlePosEvent(e events.Event) {
	pos := e.LocalPos()
	et := e.Type()
	sc := em.Scene

	isDrag := false
	switch et {
	case events.MouseDown:
		em.ResetOnMouseDown()
	case events.MouseDrag:
		isDrag = true
		switch {
		case em.Slide != nil:
			em.Slide.HandleEvent(e)
			em.Slide.Send(events.SlideMove, e)
			return // nothing further
		}
	case events.Scroll:
		switch {
		case em.Scroll != nil:
			em.Scroll.HandleEvent(e)
			return
		}
	}

	em.MouseInBBox = nil
	em.GetMouseInBBox(sc, pos)

	n := len(em.MouseInBBox)
	if n == 0 {
		if EventTrace && et != events.MouseMove {
			log.Println("Nothing in bbox:", sc.Geom.TotalBBox, "pos:", pos)
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
			w.HandleEvent(e) // everyone gets the primary event who is in scope, deepest first
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
		em.HandleLongPress(e)
	case events.MouseMove:
		hovs := make([]Widget, 0, len(em.MouseInBBox))
		for _, w := range em.MouseInBBox { // requires forward iter through em.MouseInBBox
			wb := w.AsWidget()
			if wb.Styles.Abilities.IsHoverable() {
				hovs = append(hovs, w)
			}
		}
		em.Hovers = em.UpdateHovers(hovs, em.Hovers, e, events.MouseEnter, events.MouseLeave)
		em.HandleLongHover(e)
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
			em.DragHovers = em.UpdateHovers(hovs, em.DragHovers, e, events.DragEnter, events.DragLeave)
			em.DragMove(e)                   // updates sprite position
			em.Drag.HandleEvent(e)           // raw drag
			em.Drag.Send(events.DragMove, e) // usually ignored
		case em.Slide != nil:
		case em.Press != nil && em.Press.AbilityIs(abilities.Slideable):
			if em.DragStartCheck(e, SlideStartTime, SlideStartDist) {
				em.Slide = em.Press
				em.Slide.Send(events.SlideStart, e)
			}
		case em.Press != nil && em.Press.AbilityIs(abilities.Draggable):
			if em.DragStartCheck(e, DragStartTime, DragStartDist) {
				em.Drag = em.Press
				em.Drag.Send(events.DragStart, e)
			}
		}
		// if we already have a long press widget, we update it based on our dragging movement
		if em.LongPressWidget != nil {
			em.HandleLongPress(e)
		}
	case events.MouseUp:
		switch {
		case em.Slide != nil:
			em.Slide.Send(events.SlideStop, e)
			em.Slide = nil
		case em.Drag != nil:
			em.DragDrop(em.Drag, e)
		// if we have sent a long press start event, we don't send click
		// events (non-nil widget plus nil timer means we already sent)
		case em.Press == up && up != nil && !(em.LongPressWidget != nil && em.LongPressTimer == nil):
			switch e.MouseButton() {
			case events.Left:
				if sc.SelectedWidgetChan != nil {
					sc.SelectedWidgetChan <- up
				}
				up.Send(events.Click, e)
			case events.Right: // note: automatically gets Control+Left
				up.Send(events.ContextMenu, e)
			}
		}
		em.Press = nil

		// if we have sent a long press start event, we send an end
		// event (non-nil widget plus nil timer means we already sent)
		if em.LongPressWidget != nil && em.LongPressTimer == nil {
			em.LongPressWidget.Send(events.LongPressEnd, e)
		}
		em.LongPressWidget = nil
		em.LongPressPos = image.Point{}
		if em.LongPressTimer != nil {
			em.LongPressTimer.Stop()
			em.LongPressTimer = nil
		}
		// a mouse up event acts also acts as a mouse leave
		// event on mobile, as that is needed to clear any
		// hovered state
		if up != nil && goosi.TheApp.Platform().IsMobile() {
			up.Send(events.MouseLeave, e)
		}
	case events.Scroll:
		switch {
		case em.Slide != nil:
			em.Slide.HandleEvent(e)
		case em.Drag != nil:
			em.Drag.HandleEvent(e)
		case em.Press != nil:
			em.Press.HandleEvent(e)
		default:
			em.Scene.HandleEvent(e)
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
func (em *EventMgr) UpdateHovers(hov, prev []Widget, e events.Event, enter, leave events.Types) []Widget {
	for _, prv := range prev {
		stillIn := false
		for _, cur := range hov {
			if prv == cur {
				stillIn = true
				break
			}
		}
		if !stillIn && prv.This() != nil && !prv.Is(ki.Deleted) {
			prv.Send(leave, e)
		}
	}

	for _, cur := range hov {
		wasIn := false
		for _, prv := range prev {
			if prv == cur {
				wasIn = true
				break
			}
		}
		if !wasIn {
			cur.Send(enter, e)
		}
	}
	// todo: detect change in top one, use to update cursor
	return hov
}

// TopLongHover returns the top-most LongHoverable widget among the Hovers
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
func (em *EventMgr) HandleLongHover(e events.Event) {
	em.HandleLong(e, em.TopLongHover(), &em.LongHoverWidget, &em.LongHoverPos, &em.LongHoverTimer, events.LongHoverStart, events.LongHoverEnd, LongHoverTime, LongHoverStopDist)
}

// HandleLongPress handles long press events
func (em *EventMgr) HandleLongPress(e events.Event) {
	em.HandleLong(e, em.Press, &em.LongPressWidget, &em.LongPressPos, &em.LongPressTimer, events.LongPressStart, events.LongPressEnd, LongPressTime, LongPressStopDist)
}

// HandleLong is the implementation of [EventMgr.HandleLongHover] and
// [EventManger.HandleLongPress]. It handles the logic to do with tracking
// long events using the given pointers to event manager fields and
// constant type, time, and distance properties. It should not need to
// be called by anything except for the aforementioned functions.
func (em *EventMgr) HandleLong(e events.Event, deep Widget, w *Widget, pos *image.Point, t **time.Timer, styp, etyp events.Types, stime time.Duration, sdist int) {
	em.TimerMu.Lock()
	defer em.TimerMu.Unlock()

	// fmt.Println("em:", em.Scene.Name())

	clearLong := func() {
		if *t != nil {
			(*t).Stop() // TODO: do we need to close this?
			*t = nil
		}
		*w = nil
		*pos = image.Point{}
		// fmt.Println("cleared hover")
	}

	cpos := e.Pos()
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
			(*w).Send(etyp, e)
		}
		clearLong()
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
		(*w).Send(etyp, e)
		clearLong()
		// fmt.Println("fallthrough after clear")
	}

	// if we have changed and still have the timer, we never
	// sent a start event, so we just bail
	if *t != nil {
		clearLong()
		// fmt.Println("timer non-nil, cleared")
		return
	}

	// we now know we don't have the timer and thus sent the start
	// event already, so we need to send a end event
	if *w != nil {
		(*w).Send(etyp, e)
		clearLong()
		// fmt.Println("lhw, send end, cleared")
		return
	}

	// now we can set it to our new widget
	*w = deep
	// fmt.Println("setting new:", deep)
	*pos = e.Pos()
	*t = time.AfterFunc(stime, func() {
		em.TimerMu.Lock()
		defer em.TimerMu.Unlock()
		if *w == nil {
			return
		}
		(*w).Send(styp, e)
		// we are done with the timer, and this indicates that
		// we have sent a start event
		*t = nil
	})
}

func (em *EventMgr) GetMouseInBBox(w Widget, pos image.Point) {
	wb := w.AsWidget()
	wb.WidgetWalkPre(func(kwi Widget, kwb *WidgetBase) bool {
		// we do not handle disabled here so that
		// we correctly process cursors for disabled elements.
		// it needs to be handled downstream by anyone who needs it.
		if !kwb.IsVisible() {
			return ki.Break
		}
		if !kwb.PosInScBBox(pos) {
			return ki.Break
		}
		// fmt.Println("in bb:", kwi, kwb.Styles.State)
		em.MouseInBBox = append(em.MouseInBBox, kwi)
		if kwb.Parts != nil {
			em.GetMouseInBBox(kwb.Parts, pos)
		}
		ly := AsLayout(kwi)
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

func (em *EventMgr) DragStartCheck(e events.Event, dur time.Duration, dist int) bool {
	since := e.SinceStart()
	if since < dur {
		return false
	}
	dst := int(mat32.NewVec2FmPoint(e.StartDelta()).Length())
	return dst >= dist
}

// DragStart starts a drag event, capturing a sprite image of the given widget
// and storing the data for later use during Drop
func (em *EventMgr) DragStart(w Widget, data any, e events.Event) {
	ms := em.Scene.Stage.Main
	if ms == nil {
		return
	}
	em.DragData = data
	sp := NewSprite(DragSpriteName, image.Point{}, e.Pos())
	sp.GrabRenderFrom(w) // todo: show number of items?
	ImageClearer(sp.Pixels, 50.0)
	sp.On = true
	ms.Sprites.Add(sp)
}

// DragMove is generally handled entirely by the event manager
func (em *EventMgr) DragMove(e events.Event) {
	ms := em.Scene.Stage.Main
	if ms == nil {
		return
	}
	sp, ok := ms.Sprites.SpriteByName(DragSpriteName)
	if !ok {
		fmt.Println("Drag sprite not found")
		return
	}
	sp.Geom.Pos = e.Pos()
	em.Scene.SetNeedsRender(true)
}

func (em *EventMgr) DragClearSprite() {
	ms := em.Scene.Stage.Main
	if ms == nil {
		return
	}
	ms.Sprites.InactivateSprite(DragSpriteName)
}

func (em *EventMgr) DragMenuAddModLabel(m *Scene, mod events.DropMods) {
	switch mod {
	case events.DropCopy:
		NewLabel(m).SetText("Copy (Use Shift to Move):")
	case events.DropMove:
		NewLabel(m).SetText("Move:")
	}
}

// DragDrop sends the events.Drop event to the top of the DragHovers stack.
// clearing the current dragging sprite before doing anything.
// It is up to the target to call
func (em *EventMgr) DragDrop(drag Widget, e events.Event) {
	em.DragClearSprite()
	data := em.DragData
	em.Drag = nil
	if len(em.DragHovers) == 0 {
		if EventTrace {
			fmt.Println(drag, "Drop has no target")
		}
		return
	}
	for _, dwi := range em.DragHovers {
		dwi.SetState(false, states.DragHovered)
	}
	targ := em.DragHovers[len(em.DragHovers)-1]
	de := events.NewDragDrop(events.Drop, e.(*events.Mouse)) // gets the actual mod at this point
	de.Data = data
	de.Source = drag
	de.Target = targ
	if EventTrace {
		fmt.Println(targ, "Drop with mod:", de.DropMod, "source:", de.Source)
	}
	targ.HandleEvent(de)
}

// DropFinalize should be called as the last step in the Drop event processing,
// to send the DropDeleteSource event to the source in case of DropMod == DropMove.
// Otherwise, nothing actually happens.
func (em *EventMgr) DropFinalize(de *events.DragDrop) {
	if de.DropMod != events.DropMove {
		return
	}
	de.Typ = events.DropDeleteSource
	de.ClearHandled()
	de.Source.(Widget).HandleEvent(de)
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
// 	r, code, mods, err := chord.Decode()
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
	grr.Log(goosi.TheApp.Cursor(win.GoosiWin).Set(cur))
}

// FocusClear saves current focus to FocusPrev
func (em *EventMgr) FocusClear() bool {
	if em.Focus != nil {
		if FocusTrace {
			fmt.Println(em.Scene, "FocusClear:", em.Focus)
		}
		em.PrevFocus = em.Focus
	}
	return em.SetFocusEvent(nil)
}

// SetFocus sets focus to given item, and returns true if focus changed.
// If item is nil, then nothing has focus.
// This does NOT send the events.Focus event to the widget.
// See [SetFocusEvent] for version that does send event.
func (em *EventMgr) SetFocus(w Widget) bool {
	if FocusTrace {
		fmt.Println(em.Scene, "SetFocus:", w)
	}
	got := em.SetFocusImpl(w, false) // no event
	if !got {
		if FocusTrace {
			fmt.Println(em.Scene, "SetFocus: Failed", w)
		}
		return false
	}
	if w != nil {
		w.AsWidget().ScrollToMe()
	}
	return got
}

// SetFocusEvent sets focus to given item, and returns true if focus changed.
// If item is nil, then nothing has focus.
// This sends the [events.Focus] event to the widget.
// See [SetFocus] for a version that does not.
func (em *EventMgr) SetFocusEvent(w Widget) bool {
	if FocusTrace {
		fmt.Println(em.Scene, "SetFocusEvent:", w)
	}
	got := em.SetFocusImpl(w, true) // sends event
	if !got {
		if FocusTrace {
			fmt.Println(em.Scene, "SetFocusEvent: Failed", w)
		}
		return false
	}
	if w != nil {
		w.AsWidget().ScrollToMe()
	}
	return got
}

// SetFocusImpl sets focus to given item -- returns true if focus changed.
// If item is nil, then nothing has focus.
// sendEvent determines whether the events.Focus event is sent to the focused item.
func (em *EventMgr) SetFocusImpl(w Widget, sendEvent bool) bool {
	cfoc := em.Focus
	if cfoc == nil || cfoc.This() == nil || cfoc.Is(ki.Deleted) {
		em.Focus = nil
		// fmt.Println("nil foc impl")
		cfoc = nil
	}
	if cfoc != nil && w != nil && cfoc.This() == w.This() {
		if FocusTrace {
			fmt.Println(em.Scene, "Already Focus:", cfoc)
		}
		// if sendEvent { // still send event
		// 	w.Send(events.Focus)
		// }
		return false
	}
	if cfoc != nil {
		if FocusTrace {
			fmt.Println(em.Scene, "Losing focus:", cfoc)
		}
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
	if em.Focus == nil {
		return em.FocusFirst()
	}
	return em.FocusNextFrom(em.Focus)
}

// FocusNextFrom sets the focus on the next item
// that can accept focus after the given item.
// returns true if a focus item found.
func (em *EventMgr) FocusNextFrom(from Widget) bool {
	var next Widget
	wi := from
	wb := wi.AsWidget()

	for wi != nil {
		if wb.Parts != nil {
			if em.FocusNextFrom(wb.Parts) {
				return true
			}
		}
		wi, wb = wb.WidgetNextVisible()
		if wi == nil {
			break
		}
		if wb.AbilityIs(abilities.Focusable) {
			next = wi
			break
		}
	}
	em.SetFocusEvent(next)
	return next != nil
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
		em.SetFocusEvent(foc)
		return true
	}
	return em.FocusNextFrom(foc)
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
		em.SetFocusEvent(foc)
		return true
	}
	em.Focus = foc
	fmt.Println("on or prev:", foc)
	return em.FocusPrevFrom(foc)
}

// FocusPrev sets the focus on the previous item before the
// current focus item.
func (em *EventMgr) FocusPrev() bool {
	if em.Focus == nil {
		return em.FocusLast()
	}
	return em.FocusPrevFrom(em.Focus)
}

// FocusPrevFrom sets the focus on the previous item before the given item
// (can be nil).
func (em *EventMgr) FocusPrevFrom(from Widget) bool {
	var prev Widget
	wi := from
	wb := wi.AsWidget()

	for wi != nil {
		wi, wb = wb.WidgetPrevVisible()
		if wi == nil {
			break
		}
		if wb.AbilityIs(abilities.Focusable) {
			prev = wi
			break
		}
		if wb.Parts != nil {
			if em.FocusLastFrom(wb.Parts) {
				return true
			}
		}
	}
	em.SetFocusEvent(prev)
	return prev != nil
}

// FocusFirst sets the focus on the first focusable item in the tree.
// returns true if a focusable item was found.
func (em *EventMgr) FocusFirst() bool {
	return em.FocusNextFrom(em.Scene.This().(Widget))
}

// FocusLast sets the focus on the last focusable item in the tree.
// returns true if a focusable item was found.
func (em *EventMgr) FocusLast() bool {
	return em.FocusLastFrom(em.Scene)
}

// FocusLastFrom sets the focus on the last focusable item in the given tree.
// returns true if a focusable item was found.
func (em *EventMgr) FocusLastFrom(from Widget) bool {
	last := ki.Last(from.This()).(Widget)
	return em.FocusOnOrPrev(last)
}

// ClearNonFocus clears the focus of any non-w.Focus item.
func (em *EventMgr) ClearNonFocus(foc Widget) {
	focRoot := em.Scene

	focRoot.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
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
	if em.StartFocus == nil && !em.StartFocusFirst {
		// fmt.Println("no start focus")
		return false
	}
	sf := em.StartFocus
	em.StartFocus = nil
	if sf == nil {
		em.FocusFirst()
	} else {
		// fmt.Println("start focus on:", sf)
		em.SetFocusEvent(sf)
	}
	return true
}

// MangerKeyChordEvents handles lower-priority manager-level key events.
// Mainly tab, shift-tab, and Inspector and Prefs.
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
	case keyfun.Inspector:
		TheViewIFace.Inspector(em.Scene)
		e.SetHandled()
	case keyfun.Prefs:
		TheViewIFace.PrefsView(&Prefs)
		e.SetHandled()
	case keyfun.WinClose:
		win.CloseReq()
		e.SetHandled()
	case keyfun.Menu:
		if tb := sc.GetTopAppBar(); tb != nil {
			chi := tb.ChildByType(ChooserType, ki.Embeds)
			if chi != nil {
				_, ch := AsWidget(chi)
				ch.Update()
				ch.SetFocusEvent()
			} else {
				tb.SetFocusEvent()
			}
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

/////////////////////////////////////////////////////////////////////////////////
// Shortcuts

// GetPriorityWidgets gathers Widgets with PriorityEvents set
// and also all widgets with Shortcuts
func (em *EventMgr) GetPriorityWidgets() {
	em.PriorityFocus = nil
	em.PriorityOther = nil
	em.Shortcuts = nil
	em.Scene.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		if bt := AsButton(wi.This()); bt != nil {
			if bt.Shortcut != "" {
				em.AddShortcut(bt.Shortcut, bt)
			}
		}
		if wb.PriorityEvents == nil {
			return ki.Continue
		}
		for _, tp := range wb.PriorityEvents {
			if tp.IsKey() {
				em.PriorityFocus = append(em.PriorityFocus, wi)
			} else {
				em.PriorityOther = append(em.PriorityOther, wi)
			}
		}
		return ki.Continue
	})
}

// Shortcuts is a map between a key chord and a specific Button that can be
// triggered.  This mapping must be unique, in that each chord has unique
// Button, and generally each Button only has a single chord as well, though
// this is not strictly enforced.  Shortcuts are evaluated *after* the
// standard KeyMap event processing, so any conflicts are resolved in favor of
// the local widget's key event processing, with the shortcut only operating
// when no conflicting widgets are in focus.  Shortcuts are always window-wide
// and are intended for global window / toolbar buttons.  Widget-specific key
// functions should be handled directly within widget key event
// processing.
type Shortcuts map[key.Chord]*Button

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
