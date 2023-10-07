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
	"goki.dev/girl/states"
	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/grr"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/svg"
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
	TimerMu sync.Mutex `desc:"mutex that protects timer variable updates (e.g., hover AfterFunc's)"`

	// stack of widgets with mouse pointer in BBox, and are not Disabled
	MouseInBBox []Widget

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

	// node receiving keyboard events -- use SetFocus, CurFocus
	Focus Widget `desc:"node receiving keyboard events -- use SetFocus, CurFocus"`

	// node to focus on at start when no other focus has been set yet -- use SetStartFocus
	StartFocus Widget `desc:"node to focus on at start when no other focus has been set yet -- use SetStartFocus"`

	// stack of focus
	FocusStack []Widget

	// stack of focus within elements
	FocusWithinStack []Widget

	// Last Select Mode from most recent Mouse, Keyboard events
	LastSelMode events.SelectModes `desc:"Last Select Mode from most recent Mouse, Keyboard events"`

	// currently active shortcuts for this window (shortcuts are always window-wide -- use widget key event processing for more local key functions)
	Shortcuts Shortcuts `json:"-" xml:"-" desc:"currently active shortcuts for this window (shortcuts are always window-wide -- use widget key event processing for more local key functions)"`

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
}

func (em *EventMgr) HandleFocusEvent(evi events.Event) {
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

	em.MouseInBBox = nil
	em.GetMouseInBBox(sc, pos)

	n := len(em.MouseInBBox)
	if n == 0 {
		if EventTrace && et != events.MouseMove {
			log.Println("Nothing in bbox:", sc.ScBBox, "pos:", pos)
		}
		return
	}

	cursorSet := false

	var press, move, up Widget
	for i := n - 1; i >= 0; i-- {
		w := em.MouseInBBox[i]
		wb := w.AsWidget()
		if !cursorSet && wb.Style.Cursor != cursors.None {
			em.SetCursor(wb.Style.Cursor)
			cursorSet = true
			// fmt.Println(wb.Style.Cursor)
		}

		if !isDrag {
			w.HandleEvent(evi) // everyone gets the primary event who is in scope, deepest first
		}
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
		hovs := make([]Widget, 0, len(em.MouseInBBox))
		for _, w := range em.MouseInBBox { // requires forward iter through em.MouseInBBox
			wb := w.AsWidget()
			if wb.Style.Abilities.IsHoverable() {
				hovs = append(hovs, w)
			}
		}
		em.Hovers = em.UpdateHovers(hovs, em.Hovers, evi, events.MouseEnter, events.MouseLeave)
	case events.MouseDrag:
		switch {
		case em.Drag != nil:
			hovs := make([]Widget, 0, len(em.MouseInBBox))
			for _, w := range em.MouseInBBox { // requires forward iter through em.MouseInBBox
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

func (em *EventMgr) GetMouseInBBox(w Widget, pos image.Point) {
	w.WalkPre(func(k ki.Ki) bool {
		wi, wb := AsWidget(k)
		if wb == nil || wb.Is(ki.Deleted) || wb.Is(ki.Destroyed) || wb.StateIs(states.Disabled) {
			return ki.Break
		}
		if !wb.PosInBBox(pos) {
			return ki.Break
		}
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
	var gwin goosi.Window
	if win := em.RenderWin(); win != nil {
		gwin = win.GoosiWin
	}
	grr.Log0(goosi.TheApp.Cursor(gwin).Set(cur))
	// todo: this would be simpler:
	// gwin.SetCursor(cur)
}

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
	// fmt.Println("set focus:", w)

	if cfoc != nil {
		cfoc.Send(events.FocusLost, nil)
	}
	em.Focus = w
	if w != nil {
		w.Send(events.Focus, nil)
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
		if wi == nil {
			return ki.Break
		}
		if wb.AbilityIs(states.FocusWithinable) {
			em.FocusWithinStack = append(em.FocusWithinStack, wi)
		}
		return ki.Continue
	})
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

	focRoot := em.Scene

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
			if !wi.AbilityIs(states.Focusable) {
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

	focRoot := em.Scene

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

	focRoot := em.Scene

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
	focRoot := em.Scene

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
	if em.FocusStack == nil {
		em.FocusStack = make([]Widget, 0, 50)
	}
	em.FocusStack = append(em.FocusStack, em.Focus)
	em.Focus = nil // don't un-focus on prior item when pushing
	em.FocusOnOrNext(p)
}

// PopFocus pops off the focus stack and sets prev to current focus.
func (em *EventMgr) PopFocus() {
	if em.FocusStack == nil || len(em.FocusStack) == 0 {
		em.Focus = nil
		return
	}
	sz := len(em.FocusStack)
	em.Focus = nil
	nxtf := em.FocusStack[sz-1]
	_, wb := AsWidget(nxtf)
	if wb != nil && wb.This() != nil {
		em.SetFocus(nxtf)
	}
	em.FocusStack = em.FocusStack[:sz-1]
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
	kf := KeyFun(cs)
	// fmt.Println(kf, cs)
	switch kf {
	case KeyFunGoGiEditor:
		// todo:
		// TheViewIFace.GoGiEditor(em.Master.EventTopNode())
		e.SetHandled()
	case KeyFunPrefs:
		// TheViewIFace.PrefsView(&Prefs)
		e.SetHandled()
	case KeyFunWinClose:
		win.CloseReq()
		e.SetHandled()
	case KeyFunMenu:
		if win.MainMenu != nil {
			win.MainMenu.GrabFocus()
			e.SetHandled()
		}
	case KeyFunWinSnapshot:
		dstr := time.Now().Format("Mon_Jan_2_15:04:05_MST_2006")
		fnm, _ := filepath.Abs("./GrabOf_" + sc.Name() + "_" + dstr + ".png")
		svg.SaveImage(fnm, sc.Pixels)
		fmt.Printf("Saved RenderWin Image to: %s\n", fnm)
		e.SetHandled()
	case KeyFunZoomIn:
		win.ZoomDPI(1)
		e.SetHandled()
	case KeyFunZoomOut:
		win.ZoomDPI(-1)
		e.SetHandled()
	case KeyFunRefresh:
		e.SetHandled()
		fmt.Printf("Win: %v display refreshed\n", sc.Name())
		goosi.TheApp.GetScreens()
		Prefs.UpdateAll()
		WinGeomMgr.RestoreAll()
		// w.FocusInactivate()
		// w.FullReRender()
		// sz := w.GoosiWin.Size()
		// w.SetSize(sz)
	case KeyFunWinFocusNext:
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

// AddShortcut adds given shortcut to given action.
func (em *EventMgr) AddShortcut(chord key.Chord, act *Action) {
	if chord == "" {
		return
	}
	if em.Shortcuts == nil {
		em.Shortcuts = make(Shortcuts, 100)
	}
	sa, exists := em.Shortcuts[chord]
	if exists && sa != act && sa.Text != act.Text {
		if KeyEventTrace {
			log.Printf("gi.RenderWin shortcut: %v already exists on action: %v -- will be overwritten with action: %v\n", chord, sa.Text, act.Text)
		}
	}
	em.Shortcuts[chord] = act
}

// DeleteShortcut deletes given shortcut
func (em *EventMgr) DeleteShortcut(chord key.Chord, act *Action) {
	if chord == "" {
		return
	}
	if em.Shortcuts == nil {
		return
	}
	sa, exists := em.Shortcuts[chord]
	if exists && sa == act {
		delete(em.Shortcuts, chord)
	}
}

// TriggerShortcut attempts to trigger a shortcut, returning true if one was
// triggered, and false otherwise.  Also eliminates any shortcuts with deleted
// actions, and does not trigger for Inactive actions.
func (em *EventMgr) TriggerShortcut(chord key.Chord) bool {
	if KeyEventTrace {
		fmt.Printf("Shortcut chord: %v -- looking for action\n", chord)
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
			fmt.Printf("Shortcut chord: %v, action: %v -- is inactive, not fired\n", chord, sa.Text)
		}
		return false
	}

	if KeyEventTrace {
		fmt.Printf("Shortcut chord: %v, action: %v triggered\n", chord, sa.Text)
	}
	sa.Send(events.Click, nil)
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
