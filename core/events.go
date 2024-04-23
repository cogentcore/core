// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/errors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/iox/imagex"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/units"
	"github.com/anthonynsimon/bild/clone"
)

// DragSpriteName is the name of the sprite added when dragging an object.
const DragSpriteName = "__DragSprite__"

// note: Events should be in exclusive control of its own state
// and IF we end up needing a mutex, it should be global on main
// entry points (HandleEvent, anything else?)

// Events is an event manager that handles incoming events for a [Scene].
// It creates all the derived event types (Hover, Sliding, Dragging)
// and Focus management for keyboard events.
type Events struct {

	// Scene is the scene that we manage events for
	Scene *Scene

	// mutex that protects timer variable updates (e.g., hover AfterFunc's)
	TimerMu sync.Mutex

	// stack of widgets with mouse pointer in BBox, and are not Disabled.
	// Last item in the stack is the deepest nested widget (smallest BBox).
	MouseInBBox []Widget

	// stack of hovered widgets: have mouse pointer in BBox and have Hoverable flag
	Hovers []Widget

	// LastClickWidget is the last widget that has been clicked on
	LastClickWidget Widget

	// LastDoubleClickWidget is the last widget that has been clicked on
	LastDoubleClickWidget Widget

	// LastClickTime is the time the last widget was clicked on
	LastClickTime time.Time

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

	// the deepest widget that was just pressed
	Press Widget

	// widget receiving mouse dragging events -- for drag-n-drop
	Drag Widget

	// the deepest draggable widget that was just pressed
	DragPress Widget

	// widget receiving mouse sliding events
	Slide Widget

	// the deepest slideable widget that was just pressed
	SlidePress Widget

	// widget receiving mouse scrolling events
	Scroll Widget

	// widget being held down with RepeatClickable ability
	RepeatClick Widget

	// the timer for RepeatClickable items
	RepeatClickTimer *time.Timer

	// widget receiving keyboard events -- use SetFocus, CurFocus
	Focus Widget

	// widget to focus on at start when no other focus has been set yet -- use SetStartFocus
	StartFocus Widget

	// if StartFocus not set, activate starting focus on first element
	StartFocusFirst bool

	// previously-focused widget -- what was in Focus when FocusClear is called
	PrevFocus Widget

	// Last Select Mode from most recent Mouse, Keyboard events
	LastSelMode events.SelectModes

	// Currently active shortcuts for this window (shortcuts are always window-wide.
	// Use widget key event processing for more local key functions)
	Shortcuts Shortcuts

	// source data from DragStart event
	DragData any
}

// Mains returns the stack of main stages for our scene.
func (em *Events) Mains() *Stages {
	if em.Scene == nil {
		return nil
	}
	return em.Scene.Stage.Mains
}

// RenderWindow returns the overall render window in which we reside,
// which could be nil.
func (em *Events) RenderWindow() *RenderWindow {
	mgr := em.Mains()
	if mgr == nil {
		return nil
	}
	return mgr.RenderWindow
}

///////////////////////////////////////////////////////////////////////
// 	HandleEvent

func (em *Events) HandleEvent(e events.Event) {
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

func (em *Events) HandleOtherEvent(e events.Event) {
	fmt.Println("TODO: Other event not handled", e)
}

func (em *Events) HandleFocusEvent(e events.Event) {
	// key down and key up can not give active focus, only key chord
	if em.Focus == nil && e.Type() != events.KeyDown && e.Type() != events.KeyUp {
		switch {
		case em.StartFocus != nil:
			if DebugSettings.FocusTrace {
				fmt.Println(em.Scene, "StartFocus:", em.StartFocus)
			}
			em.SetFocusEvent(em.StartFocus)
		case em.PrevFocus != nil:
			if DebugSettings.FocusTrace {
				fmt.Println(em.Scene, "PrevFocus:", em.PrevFocus)
			}
			em.SetFocusEvent(em.PrevFocus)
			em.PrevFocus = nil
		}
	}
	if em.Focus != nil {
		em.Focus.WalkUpParent(func(k tree.Node) bool {
			_, wb := AsWidget(k)
			if !wb.IsVisible() {
				return tree.Break
			}
			wb.FirstHandleEvent(e)
			return !e.IsHandled()
		})
		if !e.IsHandled() {
			em.Focus.HandleEvent(e)
		}
		if !e.IsHandled() {
			em.Focus.WalkUpParent(func(k tree.Node) bool {
				_, wb := AsWidget(k)
				if !wb.IsVisible() {
					return tree.Break
				}
				wb.FinalHandleEvent(e)
				return !e.IsHandled()
			})
		}
	}
	em.ManagerKeyChordEvents(e)
}

func (em *Events) ResetOnMouseDown() {
	em.Press = nil
	em.Drag = nil
	em.DragPress = nil
	em.Slide = nil
	em.SlidePress = nil

	em.CancelRepeatClick()

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

func (em *Events) HandlePosEvent(e events.Event) {
	pos := e.Pos()
	et := e.Type()
	sc := em.Scene

	switch et {
	case events.MouseDown:
		em.ResetOnMouseDown()
	case events.MouseDrag:
		if em.Slide != nil {
			em.Slide.HandleEvent(e)
			em.Slide.Send(events.SlideMove, e)
			return
		}
	case events.Scroll:
		if em.Scroll != nil {
			em.Scroll.HandleEvent(e)
			return
		}
	}

	em.MouseInBBox = nil
	em.GetMouseInBBox(sc, pos)

	n := len(em.MouseInBBox)
	if n == 0 {
		if DebugSettings.EventTrace && et != events.MouseMove {
			log.Println("Nothing in bbox:", sc.Geom.TotalBBox, "pos:", pos)
		}
		return
	}

	var press, dragPress, slidePress, move, up, repeatClick Widget
	for i := n - 1; i >= 0; i-- {
		w := em.MouseInBBox[i]
		wb := w.AsWidget()

		// we need to handle this here and not in [Events.GetMouseInBBox] so that
		// we correctly process cursors for disabled elements.
		// in ScRenderBBoxes, everyone is effectively enabled
		if wb.StateIs(states.Disabled) && !sc.Is(ScRenderBBoxes) {
			continue
		}

		w.HandleEvent(e) // everyone gets the primary event who is in scope, deepest first
		switch et {
		case events.MouseMove:
			if move == nil && wb.Styles.Abilities.IsHoverable() {
				move = w
			}
		case events.MouseDown:
			// in ScRenderBBoxes, everyone is effectively pressable
			if press == nil && (wb.Styles.Abilities.IsPressable() || sc.Is(ScRenderBBoxes)) {
				press = w
			}
			if dragPress == nil && wb.Styles.Abilities.Is(abilities.Draggable) {
				dragPress = w
			}
			if slidePress == nil && wb.Styles.Abilities.Is(abilities.Slideable) {
				slidePress = w
			}
			if repeatClick == nil && wb.Styles.Abilities.Is(abilities.RepeatClickable) {
				repeatClick = w
			}
		case events.MouseUp:
			// in ScRenderBBoxes, everyone is effectively pressable
			if up == nil && (wb.Styles.Abilities.IsPressable() || sc.Is(ScRenderBBoxes)) {
				up = w
			}
		}
	}
	switch et {
	case events.MouseDown:
		if press != nil {
			em.Press = press
		}
		if dragPress != nil {
			em.DragPress = dragPress
		}
		if slidePress != nil {
			em.SlidePress = slidePress
		}
		if repeatClick != nil {
			em.RepeatClick = repeatClick
			em.StartRepeatClickTimer()
		}
		em.HandleLongPress(e)
	case events.MouseMove:
		hovs := make([]Widget, 0, len(em.MouseInBBox))
		for _, w := range em.MouseInBBox { // requires forward iter through em.MouseInBBox
			wb := w.AsWidget()
			// in ScRenderBBoxes, everyone is effectively hoverable
			if wb.Styles.Abilities.IsHoverable() || sc.Is(ScRenderBBoxes) {
				hovs = append(hovs, w)
			}
		}
		if sc.Is(ScRenderBBoxes) {
			pselw := sc.SelectedWidget
			if len(em.Hovers) > 0 {
				sc.SelectedWidget = em.Hovers[len(em.Hovers)-1]
			} else {
				sc.SelectedWidget = nil
			}
			if sc.SelectedWidget != pselw {
				if pselw != nil {
					pselw.AsWidget().NeedsRender()
				}
				if sc.SelectedWidget != nil {
					sc.SelectedWidget.AsWidget().NeedsRender()
				}
			}
		}
		em.Hovers = em.UpdateHovers(hovs, em.Hovers, e, events.MouseEnter, events.MouseLeave)
		em.HandleLongHover(e)
	case events.MouseDrag:
		if em.Drag != nil {
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
		} else {
			if em.DragPress != nil && em.DragStartCheck(e, DeviceSettings.DragStartTime, DeviceSettings.DragStartDistance) {
				em.CancelRepeatClick()
				em.CancelLongPress()
				em.DragPress.Send(events.DragStart, e)
			} else if em.SlidePress != nil && em.DragStartCheck(e, DeviceSettings.SlideStartTime, DeviceSettings.DragStartDistance) {
				em.CancelRepeatClick()
				em.CancelLongPress()
				em.Slide = em.SlidePress
				em.Slide.Send(events.SlideStart, e)
			}
		}
		// if we already have a long press widget, we update it based on our dragging movement
		if em.LongPressWidget != nil {
			em.HandleLongPress(e)
		}
	case events.MouseUp:
		em.CancelRepeatClick()
		if em.Slide != nil {
			em.Slide.Send(events.SlideStop, e)
			em.Slide = nil
			em.Press = nil
		}
		if em.Drag != nil {
			em.DragDrop(em.Drag, e)
			em.Press = nil
		}
		// if we have sent a long press start event, we don't send click
		// events (non-nil widget plus nil timer means we already sent)
		if em.Press == up && up != nil && !(em.LongPressWidget != nil && em.LongPressTimer == nil) {
			em.CancelLongPress()
			switch e.MouseButton() {
			case events.Left:
				if sc.SelectedWidgetChan != nil {
					sc.SelectedWidgetChan <- up
					return
				}
				dcInTime := time.Since(em.LastClickTime) < DeviceSettings.DoubleClickInterval
				em.LastClickTime = time.Now()
				sentMulti := false
				switch {
				case em.LastDoubleClickWidget == up && dcInTime:
					tce := e.NewFromClone(events.TripleClick)
					for i := n - 1; i >= 0; i-- {
						w := em.MouseInBBox[i]
						wb := w.AsWidget()
						if !wb.StateIs(states.Disabled) && wb.AbilityIs(abilities.TripleClickable) {
							sentMulti = true
							w.HandleEvent(tce)
							break
						}
					}
				case em.LastClickWidget == up && dcInTime:
					dce := e.NewFromClone(events.DoubleClick)
					for i := n - 1; i >= 0; i-- {
						w := em.MouseInBBox[i]
						wb := w.AsWidget()
						if !wb.StateIs(states.Disabled) && wb.AbilityIs(abilities.DoubleClickable) {
							em.LastDoubleClickWidget = up // not actually who gets the event
							sentMulti = true
							w.HandleEvent(dce)
							break
						}
					}
				}
				if !sentMulti {
					em.LastDoubleClickWidget = nil
					em.LastClickWidget = up
					up.Send(events.Click, e)
				}
			case events.Right: // note: automatically gets Control+Left
				up.Send(events.ContextMenu, e)
			}
		}
		// if our original pressed widget is different from the one we are
		// going up on, then it has not gotten a mouse up event yet, so
		// we need to send it one
		if em.Press != up && em.Press != nil {
			em.Press.HandleEvent(e)
		}
		em.Press = nil

		// if we have sent a long press start event, we send an end
		// event (non-nil widget plus nil timer means we already sent)
		if em.LongPressWidget != nil && em.LongPressTimer == nil {
			em.LongPressWidget.Send(events.LongPressEnd, e)
		}
		em.CancelLongPress()
		// a mouse up event acts also acts as a mouse leave
		// event on mobile, as that is needed to clear any
		// hovered state
		if up != nil && TheApp.Platform().IsMobile() {
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
func (em *Events) UpdateHovers(hov, prev []Widget, e events.Event, enter, leave events.Types) []Widget {
	for _, prv := range prev {
		stillIn := false
		for _, cur := range hov {
			if prv == cur {
				stillIn = true
				break
			}
		}
		if !stillIn && prv.This() != nil {
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
func (em *Events) TopLongHover() Widget {
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
func (em *Events) HandleLongHover(e events.Event) {
	em.HandleLong(e, em.TopLongHover(), &em.LongHoverWidget, &em.LongHoverPos, &em.LongHoverTimer, events.LongHoverStart, events.LongHoverEnd, DeviceSettings.LongHoverTime, DeviceSettings.LongHoverStopDistance)
}

// HandleLongPress handles long press events
func (em *Events) HandleLongPress(e events.Event) {
	em.HandleLong(e, em.Press, &em.LongPressWidget, &em.LongPressPos, &em.LongPressTimer, events.LongPressStart, events.LongPressEnd, DeviceSettings.LongPressTime, DeviceSettings.LongPressStopDistance)
}

// HandleLong is the implementation of [Events.HandleLongHover] and
// [EventManger.HandleLongPress]. It handles the logic to do with tracking
// long events using the given pointers to event manager fields and
// constant type, time, and distance properties. It should not need to
// be called by anything except for the aforementioned functions.
func (em *Events) HandleLong(e events.Event, deep Widget, w *Widget, pos *image.Point, t **time.Timer, styp, etyp events.Types, stime time.Duration, sdist int) {
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

	cpos := e.WindowPos()
	dst := int(math32.Hypot(float32(pos.X-cpos.X), float32(pos.Y-cpos.Y)))
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
		// the solution to https://github.com/cogentcore/core/issues/553
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
	*pos = e.WindowPos()
	*t = time.AfterFunc(stime, func() {
		win := em.RenderWindow()
		if win == nil {
			return
		}
		rc := win.RenderContext() // have to get this one first
		rc.Lock()
		defer rc.Unlock()

		em.TimerMu.Lock() // then can get this
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

func (em *Events) GetMouseInBBox(w Widget, pos image.Point) {
	wb := w.AsWidget()
	wb.WidgetWalkPre(func(kwi Widget, kwb *WidgetBase) bool {
		// we do not handle disabled here so that
		// we correctly process cursors for disabled elements.
		// it needs to be handled downstream by anyone who needs it.
		if !kwb.IsVisible() {
			return tree.Break
		}
		if !kwb.PosInScBBox(pos) {
			return tree.Break
		}
		em.MouseInBBox = append(em.MouseInBBox, kwi)
		if kwb.Parts != nil {
			em.GetMouseInBBox(kwb.Parts, pos)
		}
		if ly := AsLayout(kwi); ly != nil {
			for d := math32.X; d <= math32.Y; d++ {
				if ly.HasScroll[d] {
					sb := ly.Scrolls[d]
					em.GetMouseInBBox(sb, pos)
				}
			}
		}
		return tree.Continue
	})
}

func (em *Events) CancelLongPress() {
	em.LongPressWidget = nil
	em.LongPressPos = image.Point{}
	if em.LongPressTimer != nil {
		em.LongPressTimer.Stop()
		em.LongPressTimer = nil
	}
}

func (em *Events) CancelRepeatClick() {
	em.RepeatClick = nil
	if em.RepeatClickTimer != nil {
		em.RepeatClickTimer.Stop()
		em.RepeatClickTimer = nil
	}
}

func (em *Events) StartRepeatClickTimer() {
	if em.RepeatClick == nil || !em.RepeatClick.IsVisible() {
		return
	}
	delay := DeviceSettings.RepeatClickTime
	if em.RepeatClickTimer == nil {
		delay *= 8
	}
	em.RepeatClickTimer = time.AfterFunc(delay, func() {
		if em.RepeatClick == nil || !em.RepeatClick.IsVisible() {
			return
		}
		em.RepeatClick.Send(events.Click)
		em.StartRepeatClickTimer()
	})
}

func (em *Events) DragStartCheck(e events.Event, dur time.Duration, dist int) bool {
	since := e.SinceStart()
	if since < dur {
		return false
	}
	dst := int(math32.Vector2FromPoint(e.StartDelta()).Length())
	return dst >= dist
}

// DragStart starts a drag event, capturing a sprite image of the given widget
// and storing the data for later use during Drop.
// A drag does not officially start until this is called.
func (em *Events) DragStart(w Widget, data any, e events.Event) {
	ms := em.Scene.Stage.Main
	if ms == nil {
		return
	}
	em.Drag = w
	em.DragData = data
	sp := NewSprite(DragSpriteName, image.Point{}, e.WindowPos())
	sp.GrabRenderFrom(w) // TODO: maybe show the number of items being dragged
	sp.Pixels = clone.AsRGBA(gradient.ApplyOpacityImage(sp.Pixels, 0.5))
	sp.On = true
	ms.Sprites.Add(sp)
}

// DragMove is generally handled entirely by the event manager
func (em *Events) DragMove(e events.Event) {
	ms := em.Scene.Stage.Main
	if ms == nil {
		return
	}
	sp, ok := ms.Sprites.SpriteByName(DragSpriteName)
	if !ok {
		fmt.Println("Drag sprite not found")
		return
	}
	sp.Geom.Pos = e.WindowPos()
	em.Scene.NeedsRender()
}

func (em *Events) DragClearSprite() {
	ms := em.Scene.Stage.Main
	if ms == nil {
		return
	}
	ms.Sprites.InactivateSprite(DragSpriteName)
}

func (em *Events) DragMenuAddModText(m *Scene, mod events.DropMods) {
	text := ""
	switch mod {
	case events.DropCopy:
		text = "Copy (use Shift to move):"
	case events.DropMove:
		text = "Move:"
	}
	NewText(m).SetType(TextLabelLarge).SetText(text).Style(func(s *styles.Style) {
		s.Margin.Set(units.Em(0.5))
	})
}

// DragDrop sends the events.Drop event to the top of the DragHovers stack.
// clearing the current dragging sprite before doing anything.
// It is up to the target to call
func (em *Events) DragDrop(drag Widget, e events.Event) {
	em.DragClearSprite()
	data := em.DragData
	em.Drag = nil
	if len(em.DragHovers) == 0 {
		if DebugSettings.EventTrace {
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
	if DebugSettings.EventTrace {
		fmt.Println(targ, "Drop with mod:", de.DropMod, "source:", de.Source)
	}
	targ.HandleEvent(de)
}

// DropFinalize should be called as the last step in the Drop event processing,
// to send the DropDeleteSource event to the source in case of DropMod == DropMove.
// Otherwise, nothing actually happens.
func (em *Events) DropFinalize(de *events.DragDrop) {
	if de.DropMod != events.DropMove {
		return
	}
	de.Typ = events.DropDeleteSource
	de.ClearHandled()
	de.Source.(Widget).HandleEvent(de)
}

// Clipboard returns the system system.Clipboard, supplying the window context
// if available.
func (em *Events) Clipboard() system.Clipboard {
	var gwin system.Window
	if win := em.RenderWindow(); win != nil {
		gwin = win.SystemWindow
	}
	return system.TheApp.Clipboard(gwin)
}

// SetCursor sets window cursor to given Cursor
func (em *Events) SetCursor(cur cursors.Cursor) {
	win := em.RenderWindow()
	if win == nil {
		return
	}
	if !win.IsVisible() {
		return
	}
	errors.Log(system.TheApp.Cursor(win.SystemWindow).Set(cur))
}

// FocusClear saves current focus to FocusPrev
func (em *Events) FocusClear() bool {
	if em.Focus != nil {
		if DebugSettings.FocusTrace {
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
func (em *Events) SetFocus(w Widget) bool {
	if DebugSettings.FocusTrace {
		fmt.Println(em.Scene, "SetFocus:", w)
	}
	got := em.SetFocusImpl(w, false) // no event
	if !got {
		if DebugSettings.FocusTrace {
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
func (em *Events) SetFocusEvent(w Widget) bool {
	if DebugSettings.FocusTrace {
		fmt.Println(em.Scene, "SetFocusEvent:", w)
	}
	got := em.SetFocusImpl(w, true) // sends event
	if !got {
		if DebugSettings.FocusTrace {
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
func (em *Events) SetFocusImpl(w Widget, sendEvent bool) bool {
	cfoc := em.Focus
	if cfoc == nil || cfoc.This() == nil {
		em.Focus = nil
		// fmt.Println("nil foc impl")
		cfoc = nil
	}
	if cfoc != nil && w != nil && cfoc.This() == w.This() {
		if DebugSettings.FocusTrace {
			fmt.Println(em.Scene, "Already Focus:", cfoc)
		}
		// if sendEvent { // still send event
		// 	w.Send(events.Focus)
		// }
		return false
	}
	if cfoc != nil {
		if DebugSettings.FocusTrace {
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

// FocusNext sets the focus on the next item
// that can accept focus after the current Focus item.
// returns true if a focus item found.
func (em *Events) FocusNext() bool {
	if em.Focus == nil {
		return em.FocusFirst()
	}
	return em.FocusNextFrom(em.Focus)
}

// FocusNextFrom sets the focus on the next item
// that can accept focus after the given item.
// returns true if a focus item found.
func (em *Events) FocusNextFrom(from Widget) bool {
	next := WidgetNextFunc(from, func(w Widget) bool {
		return w.IsVisible() && !w.StateIs(states.Disabled) && w.AbilityIs(abilities.Focusable)
	})
	em.SetFocusEvent(next)
	return next != nil
}

// FocusOnOrNext sets the focus on the given item, or the next one that can
// accept focus -- returns true if a new focus item found.
func (em *Events) FocusOnOrNext(foc Widget) bool {
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
func (em *Events) FocusOnOrPrev(foc Widget) bool {
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
	return em.FocusPrevFrom(foc)
}

// FocusPrev sets the focus on the previous item before the
// current focus item.
func (em *Events) FocusPrev() bool {
	if em.Focus == nil {
		return em.FocusLast()
	}
	return em.FocusPrevFrom(em.Focus)
}

// FocusPrevFrom sets the focus on the previous item before the given item
// (can be nil).
func (em *Events) FocusPrevFrom(from Widget) bool {
	prev := WidgetPrevFunc(from, func(w Widget) bool {
		return w.IsVisible() && !w.StateIs(states.Disabled) && w.AbilityIs(abilities.Focusable)
	})
	em.SetFocusEvent(prev)
	return prev != nil
}

// FocusFirst sets the focus on the first focusable item in the tree.
// returns true if a focusable item was found.
func (em *Events) FocusFirst() bool {
	return em.FocusNextFrom(em.Scene.This().(Widget))
}

// FocusLast sets the focus on the last focusable item in the tree.
// returns true if a focusable item was found.
func (em *Events) FocusLast() bool {
	return em.FocusLastFrom(em.Scene)
}

// FocusLastFrom sets the focus on the last focusable item in the given tree.
// returns true if a focusable item was found.
func (em *Events) FocusLastFrom(from Widget) bool {
	last := tree.Last(from.This()).(Widget)
	// fmt.Println("last:", last, "from:", from)
	return em.FocusOnOrPrev(last)
}

// ClearNonFocus clears the focus of any non-w.Focus item.
func (em *Events) ClearNonFocus(foc Widget) {
	focRoot := em.Scene

	focRoot.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		if wi == focRoot { // skip top-level
			return tree.Continue
		}
		if !wb.IsVisible() {
			return tree.Continue
		}
		if foc == wi {
			return tree.Continue
		}
		if wb.StateIs(states.Focused) {
			if DebugSettings.EventTrace {
				fmt.Printf("ClearNonFocus: had focus: %v\n", wb.Path())
			}
			wi.Send(events.FocusLost)
		}
		return tree.Continue
	})
}

// SetStartFocus sets the given item to be first focus when window opens.
func (em *Events) SetStartFocus(k Widget) {
	em.StartFocus = k
}

// ActivateStartFocus activates start focus if there is no current focus
// and StartFocus is set -- returns true if activated
func (em *Events) ActivateStartFocus() bool {
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
// Mainly tab, shift-tab, and Inspector and Settings.
// event will be marked as processed if handled here.
func (em *Events) ManagerKeyChordEvents(e events.Event) {
	if e.IsHandled() {
		return
	}
	if e.Type() != events.KeyChord {
		return
	}
	win := em.RenderWindow()
	if win == nil {
		return
	}
	sc := em.Scene
	cs := e.KeyChord()
	kf := keymap.Of(cs)
	// fmt.Println(kf, cs)
	switch kf {
	case keymap.FocusNext: // tab
		if em.FocusNext() {
			e.SetHandled()
		}
	case keymap.FocusPrev: // shift-tab
		if em.FocusPrev() {
			e.SetHandled()
		}
	case keymap.Menu:
		if tb := sc.GetTopAppBar(); tb != nil {
			ch := tree.ChildByType[*Chooser](tb, true)
			if ch != nil {
				ch.SetFocusEvent()
				ch.TextField().OfferComplete()
			} else {
				tb.SetFocusEvent()
			}
			e.SetHandled()
		}
	case keymap.WinSnapshot:
		dstr := time.Now().Format("Mon_Jan_2_15:04:05_MST_2006")
		fnm, _ := filepath.Abs("./GrabOf_" + sc.Name() + "_" + dstr + ".png")
		imagex.Save(sc.Pixels, fnm)
		fmt.Printf("Saved RenderWindow Image to: %s\n", fnm)
		e.SetHandled()
	case keymap.ZoomIn:
		win.StepZoom(1)
		e.SetHandled()
	case keymap.ZoomOut:
		win.StepZoom(-1)
		e.SetHandled()
	case keymap.Refresh:
		e.SetHandled()
		system.TheApp.GetScreens()
		UpdateAll()
		TheWindowGeometrySaver.RestoreAll()
		// w.FocusInactivate()
		// w.FullReRender()
		// sz := w.SystemWin.Size()
		// w.SetSize(sz)
	case keymap.WinFocusNext:
		e.SetHandled()
		AllRenderWindows.FocusNext()
	}
	// TODO(kai): maybe clean up / document this
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

// GetShortcuts gathers all [Button]s in the Scene with a shortcut specified.
// It recursively navigates [Button.Menu]s.
func (em *Events) GetShortcuts() {
	em.Shortcuts = nil
	em.GetShortcutsIn(em.Scene)
}

// GetShortcutsIn gathers all [Button]s in the given parent widget with
// a shortcut specified. It recursively navigates [Button.Menu]s.
func (em *Events) GetShortcutsIn(parent Widget) {
	parent.AsWidget().WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		bt := AsButton(wi.This())
		if bt == nil {
			return tree.Continue
		}
		if bt.Shortcut != "" {
			em.AddShortcut(bt.Shortcut, bt)
		}
		if bt.HasMenu() {
			tmps := NewScene()
			bt.Menu(tmps)
			em.GetShortcutsIn(tmps)
		}
		return tree.Continue
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
func (em *Events) AddShortcut(chord key.Chord, bt *Button) {
	if chord == "" {
		return
	}
	if em.Shortcuts == nil {
		em.Shortcuts = Shortcuts{}
	}
	chords := strings.Split(string(chord), "\n")
	for _, c := range chords {
		cc := key.Chord(c)
		if DebugSettings.KeyEventTrace {
			old, exists := em.Shortcuts[cc]
			if exists && old != bt {
				slog.Error("core.Events.AddShortcut: overwriting duplicate shortcut", "shortcut", cc, "originalButton", old, "newButton", bt)
			}
		}
		em.Shortcuts[cc] = bt
	}
}

// DeleteShortcut deletes given shortcut
func (em *Events) DeleteShortcut(chord key.Chord, bt *Button) {
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
func (em *Events) TriggerShortcut(chord key.Chord) bool {
	if DebugSettings.KeyEventTrace {
		fmt.Printf("Shortcut chord: %v -- looking for button\n", chord)
	}
	if em.Shortcuts == nil {
		return false
	}
	sa, exists := em.Shortcuts[chord]
	if !exists {
		return false
	}
	if sa == nil || sa.This() == nil {
		delete(em.Shortcuts, chord)
		return false
	}
	if sa.IsDisabled() {
		if DebugSettings.KeyEventTrace {
			fmt.Printf("Shortcut chord: %v, button: %v -- is inactive, not fired\n", chord, sa.Text)
		}
		return false
	}

	if DebugSettings.KeyEventTrace {
		fmt.Printf("Shortcut chord: %v, button: %v triggered\n", chord, sa.Text)
	}
	sa.Send(events.Click)
	return true
}
