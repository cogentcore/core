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

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	"github.com/anthonynsimon/bild/clone"
)

// dragSpriteName is the name of the sprite added when dragging an object.
const dragSpriteName = "__DragSprite__"

// note: Events should be in exclusive control of its own state
// and IF we end up needing a mutex, it should be global on main
// entry points (HandleEvent, anything else?)

// Events is an event manager that handles incoming events for a [Scene].
// It creates all the derived event types (Hover, Sliding, Dragging)
// and Focus management for keyboard events.
type Events struct {

	// scene is the scene that we manage events for.
	scene *Scene

	// mutex that protects timer variable updates (e.g., hover AfterFunc's).
	timerMu sync.Mutex

	// stack of sprites with mouse pointer in BBox, with any listeners present.
	spriteInBBox []*Sprite

	// currently pressing sprite.
	spritePress *Sprite

	// currently sliding (dragging) sprite.
	spriteSlide *Sprite

	// stack of widgets with mouse pointer in BBox, and are not Disabled.
	// Last item in the stack is the deepest nested widget (smallest BBox).
	mouseInBBox []Widget

	// stack of hovered widgets: have mouse pointer in BBox and have Hoverable flag.
	hovers []Widget

	// lastClickWidget is the last widget that has been clicked on.
	lastClickWidget Widget

	// lastDoubleClickWidget is the last widget that has been clicked on.
	lastDoubleClickWidget Widget

	// lastClickTime is the time the last widget was clicked on.
	lastClickTime time.Time

	// the current candidate for a long hover event.
	longHoverWidget Widget

	// the position of the mouse at the start of LongHoverTimer.
	longHoverPos image.Point

	// the timer for the LongHover event, started with time.AfterFunc.
	longHoverTimer *time.Timer

	// the current candidate for a long press event.
	longPressWidget Widget

	// the position of the mouse at the start of LongPressTimer.
	longPressPos image.Point

	// the timer for the LongPress event, started with time.AfterFunc.
	longPressTimer *time.Timer

	// stack of drag-hovered widgets: have mouse pointer in BBox and have Droppable flag.
	dragHovers []Widget

	// the deepest widget that was just pressed.
	press Widget

	// widget receiving mouse dragging events, for drag-n-drop.
	drag Widget

	// the deepest draggable widget that was just pressed.
	dragPress Widget

	// widget receiving mouse sliding events.
	slide Widget

	// the deepest slideable widget that was just pressed.
	slidePress Widget

	// widget receiving mouse scrolling events, has "scroll focus".
	scroll Widget

	lastScrollTime time.Time

	// widget being held down with RepeatClickable ability.
	repeatClick Widget

	// the timer for RepeatClickable items.
	repeatClickTimer *time.Timer

	// widget receiving keyboard events.  Use SetFocus, CurFocus.
	focus Widget

	// widget to focus on at start when no other focus has been
	// set yet. Use SetStartFocus.
	startFocus Widget

	// if StartFocus not set, activate starting focus on first element
	startFocusFirst bool

	// previously focused widget.  Was in Focus when FocusClear is called.
	prevFocus Widget

	// Currently active shortcuts for this window (shortcuts are always window-wide.
	// Use widget key event processing for more local key functions)
	shortcuts shortcuts

	// source data from DragStart event.
	dragData any
}

// mains returns the stack of main stages for our scene.
func (em *Events) mains() *stages {
	if em.scene == nil {
		return nil
	}
	return em.scene.Stage.Mains
}

// RenderWindow returns the overall render window in which we reside,
// which could be nil.
func (em *Events) RenderWindow() *renderWindow {
	mgr := em.mains()
	if mgr == nil {
		return nil
	}
	return mgr.renderWindow
}

func (em *Events) handleEvent(e events.Event) {
	if e.IsHandled() {
		return
	}
	switch {
	case e.HasPos():
		em.handlePosEvent(e)
	case e.NeedsFocus():
		em.handleFocusEvent(e)
	}
}

func (em *Events) handleFocusEvent(e events.Event) {
	// key down and key up can not give active focus, only key chord
	if em.focus == nil && e.Type() != events.KeyDown && e.Type() != events.KeyUp {
		switch {
		case em.startFocus != nil:
			if DebugSettings.FocusTrace {
				fmt.Println(em.scene, "StartFocus:", em.startFocus)
			}
			em.setFocusEvent(em.startFocus)
		case em.prevFocus != nil:
			if DebugSettings.FocusTrace {
				fmt.Println(em.scene, "PrevFocus:", em.prevFocus)
			}
			em.setFocusEvent(em.prevFocus)
			em.prevFocus = nil
		}
	}
	if em.focus != nil {
		em.focus.AsTree().WalkUpParent(func(k tree.Node) bool {
			wb := AsWidget(k)
			if !wb.IsVisible() {
				return tree.Break
			}
			wb.firstHandleEvent(e)
			return !e.IsHandled()
		})
		if !e.IsHandled() {
			em.focus.AsWidget().HandleEvent(e)
		}
		if !e.IsHandled() {
			em.focus.AsTree().WalkUpParent(func(k tree.Node) bool {
				wb := AsWidget(k)
				if !wb.IsVisible() {
					return tree.Break
				}
				wb.finalHandleEvent(e)
				return !e.IsHandled()
			})
		}
	}
	em.managerKeyChordEvents(e)
}

func (em *Events) resetOnMouseDown() {
	em.press = nil
	em.drag = nil
	em.dragPress = nil
	em.slide = nil
	em.slidePress = nil
	em.spriteSlide = nil
	em.spritePress = nil

	em.cancelRepeatClick()

	// if we have sent a long hover start event, we send an end
	// event (non-nil widget plus nil timer means we already sent)
	if em.longHoverWidget != nil && em.longHoverTimer == nil {
		em.longHoverWidget.AsWidget().Send(events.LongHoverEnd)
	}
	em.longHoverWidget = nil
	em.longHoverPos = image.Point{}
	if em.longHoverTimer != nil {
		em.longHoverTimer.Stop()
		em.longHoverTimer = nil
	}
}

func (em *Events) handlePosEvent(e events.Event) {
	pos := e.Pos()
	et := e.Type()
	sc := em.scene

	switch et {
	case events.MouseDown:
		em.resetOnMouseDown()
	case events.MouseDrag:
		if em.spriteSlide != nil {
			em.spriteSlide.handleEvent(e)
			em.spriteSlide.send(events.SlideMove, e)
			return
		}
		if em.slide != nil {
			em.slide.AsWidget().HandleEvent(e)
			em.slide.AsWidget().Send(events.SlideMove, e)
			return
		}
	case events.Scroll:
		if em.scroll != nil {
			scInTime := time.Since(em.lastScrollTime) < DeviceSettings.ScrollFocusTime
			if scInTime {
				em.scroll.AsWidget().HandleEvent(e)
				if e.IsHandled() {
					em.lastScrollTime = time.Now()
				}
				return
			} else {
				em.scroll = nil
			}
		}
	}

	em.spriteInBBox = nil
	if et != events.MouseMove {
		em.getSpriteInBBox(sc, e.WindowPos())

		if len(em.spriteInBBox) > 0 {
			if em.handleSpriteEvent(e) {
				return
			}
		}
	}

	em.mouseInBBox = nil
	em.getMouseInBBox(sc, pos)

	n := len(em.mouseInBBox)
	if n == 0 {
		if DebugSettings.EventTrace && et != events.MouseMove {
			log.Println("Nothing in bbox:", sc.Geom.TotalBBox, "pos:", pos)
		}
		return
	}

	var press, dragPress, slidePress, move, up, repeatClick Widget
	for i := n - 1; i >= 0; i-- {
		w := em.mouseInBBox[i]
		wb := w.AsWidget()

		// we need to handle this here and not in [Events.GetMouseInBBox] so that
		// we correctly process cursors for disabled elements.
		// in ScRenderBBoxes, everyone is effectively enabled
		if wb.StateIs(states.Disabled) && !sc.renderBBoxes {
			continue
		}

		w.AsWidget().HandleEvent(e) // everyone gets the primary event who is in scope, deepest first
		switch et {
		case events.MouseMove:
			em.scroll = nil
			if move == nil && wb.Styles.Abilities.IsHoverable() {
				move = w
			}
		case events.MouseDown:
			em.scroll = nil
			// in ScRenderBBoxes, everyone is effectively pressable
			if press == nil && (wb.Styles.Abilities.IsPressable() || sc.renderBBoxes) {
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
			em.scroll = nil
			// in ScRenderBBoxes, everyone is effectively pressable
			if up == nil && (wb.Styles.Abilities.IsPressable() || sc.renderBBoxes) {
				up = w
			}
		case events.Scroll:
			if e.IsHandled() {
				if em.scroll == nil {
					em.scroll = w
					em.lastScrollTime = time.Now()
				}
			}
		}
	}
	switch et {
	case events.MouseDown:
		if press != nil {
			em.press = press
		}
		if dragPress != nil {
			em.dragPress = dragPress
		}
		if slidePress != nil {
			em.slidePress = slidePress
		}
		if repeatClick != nil {
			em.repeatClick = repeatClick
			em.startRepeatClickTimer()
		}
		em.handleLongPress(e)
	case events.MouseMove:
		hovs := make([]Widget, 0, len(em.mouseInBBox))
		for _, w := range em.mouseInBBox { // requires forward iter through em.MouseInBBox
			wb := w.AsWidget()
			// in ScRenderBBoxes, everyone is effectively hoverable
			if wb.Styles.Abilities.IsHoverable() || sc.renderBBoxes {
				hovs = append(hovs, w)
			}
		}
		if sc.renderBBoxes {
			pselw := sc.selectedWidget
			if len(em.hovers) > 0 {
				sc.selectedWidget = em.hovers[len(em.hovers)-1]
			} else {
				sc.selectedWidget = nil
			}
			if sc.selectedWidget != pselw {
				if pselw != nil {
					pselw.AsWidget().NeedsRender()
				}
				if sc.selectedWidget != nil {
					sc.selectedWidget.AsWidget().NeedsRender()
				}
			}
		}
		em.hovers = em.updateHovers(hovs, em.hovers, e, events.MouseEnter, events.MouseLeave)
		em.handleLongHover(e)
	case events.MouseDrag:
		if em.drag != nil {
			hovs := make([]Widget, 0, len(em.mouseInBBox))
			for _, w := range em.mouseInBBox { // requires forward iter through em.MouseInBBox
				wb := w.AsWidget()
				if wb.AbilityIs(abilities.Droppable) {
					hovs = append(hovs, w)
				}
			}
			em.dragHovers = em.updateHovers(hovs, em.dragHovers, e, events.DragEnter, events.DragLeave)
			em.dragMove(e)                              // updates sprite position
			em.drag.AsWidget().HandleEvent(e)           // raw drag
			em.drag.AsWidget().Send(events.DragMove, e) // usually ignored
		} else {
			if em.dragPress != nil && em.dragStartCheck(e, DeviceSettings.DragStartTime, DeviceSettings.DragStartDistance) {
				em.cancelRepeatClick()
				em.cancelLongPress()
				em.dragPress.AsWidget().Send(events.DragStart, e)
			} else if em.slidePress != nil && em.dragStartCheck(e, DeviceSettings.SlideStartTime, DeviceSettings.DragStartDistance) {
				em.cancelRepeatClick()
				em.cancelLongPress()
				em.slide = em.slidePress
				em.slide.AsWidget().Send(events.SlideStart, e)
			}
		}
		// if we already have a long press widget, we update it based on our dragging movement
		if em.longPressWidget != nil {
			em.handleLongPress(e)
		}
	case events.MouseUp:
		em.cancelRepeatClick()
		if em.slide != nil {
			em.slide.AsWidget().Send(events.SlideStop, e)
			em.slide = nil
			em.press = nil
		}
		if em.drag != nil {
			em.dragDrop(em.drag, e)
			em.press = nil
		}
		// if we have sent a long press start event, we don't send click
		// events (non-nil widget plus nil timer means we already sent)
		if em.press == up && up != nil && !(em.longPressWidget != nil && em.longPressTimer == nil) {
			em.cancelLongPress()
			switch e.MouseButton() {
			case events.Left:
				if sc.selectedWidgetChan != nil {
					sc.selectedWidgetChan <- up
					return
				}
				dcInTime := time.Since(em.lastClickTime) < DeviceSettings.DoubleClickInterval
				em.lastClickTime = time.Now()
				sentMulti := false
				switch {
				case em.lastDoubleClickWidget == up && dcInTime:
					tce := e.NewFromClone(events.TripleClick)
					for i := n - 1; i >= 0; i-- {
						w := em.mouseInBBox[i]
						wb := w.AsWidget()
						if !wb.StateIs(states.Disabled) && wb.AbilityIs(abilities.TripleClickable) {
							sentMulti = true
							w.AsWidget().HandleEvent(tce)
							break
						}
					}
				case em.lastClickWidget == up && dcInTime:
					dce := e.NewFromClone(events.DoubleClick)
					for i := n - 1; i >= 0; i-- {
						w := em.mouseInBBox[i]
						wb := w.AsWidget()
						if !wb.StateIs(states.Disabled) && wb.AbilityIs(abilities.DoubleClickable) {
							em.lastDoubleClickWidget = up // not actually who gets the event
							sentMulti = true
							w.AsWidget().HandleEvent(dce)
							break
						}
					}
				}
				if !sentMulti {
					em.lastDoubleClickWidget = nil
					em.lastClickWidget = up
					up.AsWidget().Send(events.Click, e)
				}
			case events.Right: // note: automatically gets Control+Left
				up.AsWidget().Send(events.ContextMenu, e)
			}
		}
		// if our original pressed widget is different from the one we are
		// going up on, then it has not gotten a mouse up event yet, so
		// we need to send it one
		if em.press != up && em.press != nil {
			em.press.AsWidget().HandleEvent(e)
		}
		em.press = nil

		em.cancelLongPress()
		// a mouse up event acts also acts as a mouse leave
		// event on mobile, as that is needed to clear any
		// hovered state
		if up != nil && TheApp.Platform().IsMobile() {
			up.AsWidget().Send(events.MouseLeave, e)
		}
	case events.Scroll:
		switch {
		case em.slide != nil:
			em.slide.AsWidget().HandleEvent(e)
		case em.drag != nil:
			em.drag.AsWidget().HandleEvent(e)
		case em.press != nil:
			em.press.AsWidget().HandleEvent(e)
		default:
			em.scene.HandleEvent(e)
		}
	}

	// we need to handle cursor after all of the events so that
	// we get the latest cursor if it changes based on the state

	cursorSet := false
	for i := n - 1; i >= 0; i-- {
		w := em.mouseInBBox[i]
		wb := w.AsWidget()
		if !cursorSet && wb.Styles.Cursor != cursors.None {
			em.setCursor(wb.Styles.Cursor)
			cursorSet = true
		}
	}
}

// updateHovers updates the hovered widgets based on current
// widgets in bounding box.
func (em *Events) updateHovers(hov, prev []Widget, e events.Event, enter, leave events.Types) []Widget {
	for _, prv := range prev {
		stillIn := false
		for _, cur := range hov {
			if prv == cur {
				stillIn = true
				break
			}
		}
		if !stillIn && prv.AsTree().This != nil {
			prv.AsWidget().Send(leave, e)
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
			cur.AsWidget().Send(enter, e)
		}
	}
	// todo: detect change in top one, use to update cursor
	return hov
}

// topLongHover returns the top-most LongHoverable widget among the Hovers
func (em *Events) topLongHover() Widget {
	var deep Widget
	for i := len(em.hovers) - 1; i >= 0; i-- {
		h := em.hovers[i]
		if h.AsWidget().AbilityIs(abilities.LongHoverable) {
			deep = h
			break
		}
	}
	return deep
}

// handleLongHover handles long hover events
func (em *Events) handleLongHover(e events.Event) {
	em.handleLong(e, em.topLongHover(), &em.longHoverWidget, &em.longHoverPos, &em.longHoverTimer, events.LongHoverStart, events.LongHoverEnd, DeviceSettings.LongHoverTime, DeviceSettings.LongHoverStopDistance)
}

// handleLongPress handles long press events
func (em *Events) handleLongPress(e events.Event) {
	em.handleLong(e, em.press, &em.longPressWidget, &em.longPressPos, &em.longPressTimer, events.LongPressStart, events.LongPressEnd, DeviceSettings.LongPressTime, DeviceSettings.LongPressStopDistance)
}

// handleLong is the implementation of [Events.handleLongHover] and
// [EventManger.HandleLongPress]. It handles the logic to do with tracking
// long events using the given pointers to event manager fields and
// constant type, time, and distance properties. It should not need to
// be called by anything except for the aforementioned functions.
func (em *Events) handleLong(e events.Event, deep Widget, w *Widget, pos *image.Point, t **time.Timer, styp, etyp events.Types, stime time.Duration, sdist int) {
	em.timerMu.Lock()
	defer em.timerMu.Unlock()

	// fmt.Println("em:", em.Scene.Name)

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
			(*w).AsWidget().Send(etyp, e)
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
		(*w).AsWidget().Send(etyp, e)
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
		(*w).AsWidget().Send(etyp, e)
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
		rc := win.renderContext() // have to get this one first
		rc.lock()
		defer rc.unlock()

		em.timerMu.Lock() // then can get this
		defer em.timerMu.Unlock()
		if *w == nil {
			return
		}
		(*w).AsWidget().Send(styp, e)
		// we are done with the timer, and this indicates that
		// we have sent a start event
		*t = nil
	})
}

func (em *Events) getMouseInBBox(w Widget, pos image.Point) {
	wb := w.AsWidget()
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		// we do not handle disabled here so that
		// we correctly process cursors for disabled elements.
		// it needs to be handled downstream by anyone who needs it.
		if !cwb.IsVisible() {
			return tree.Break
		}
		if !cwb.posInScBBox(pos) {
			return tree.Break
		}
		em.mouseInBBox = append(em.mouseInBBox, cw)
		if cwb.Parts != nil {
			em.getMouseInBBox(cwb.Parts, pos)
		}
		if ly := AsFrame(cw); ly != nil {
			for d := math32.X; d <= math32.Y; d++ {
				if ly.HasScroll[d] {
					sb := ly.scrolls[d]
					em.getMouseInBBox(sb, pos)
				}
			}
		}
		return tree.Continue
	})
}

func (em *Events) cancelLongPress() {
	// if we have sent a long press start event, we send an end
	// event (non-nil widget plus nil timer means we already sent)
	if em.longPressWidget != nil && em.longPressTimer == nil {
		em.longPressWidget.AsWidget().Send(events.LongPressEnd)
	}
	em.longPressWidget = nil
	em.longPressPos = image.Point{}
	if em.longPressTimer != nil {
		em.longPressTimer.Stop()
		em.longPressTimer = nil
	}
}

func (em *Events) cancelRepeatClick() {
	em.repeatClick = nil
	if em.repeatClickTimer != nil {
		em.repeatClickTimer.Stop()
		em.repeatClickTimer = nil
	}
}

func (em *Events) startRepeatClickTimer() {
	if em.repeatClick == nil || !em.repeatClick.AsWidget().IsVisible() {
		return
	}
	delay := DeviceSettings.RepeatClickTime
	if em.repeatClickTimer == nil {
		delay *= 8
	}
	em.repeatClickTimer = time.AfterFunc(delay, func() {
		if em.repeatClick == nil || !em.repeatClick.AsWidget().IsVisible() {
			return
		}
		em.repeatClick.AsWidget().Send(events.Click)
		em.startRepeatClickTimer()
	})
}

func (em *Events) dragStartCheck(e events.Event, dur time.Duration, dist int) bool {
	since := e.SinceStart()
	if since < dur {
		return false
	}
	dst := int(math32.Vector2FromPoint(e.StartDelta()).Length())
	return dst >= dist
}

// dragStart starts a drag event, capturing a sprite image of the given widget
// and storing the data for later use during Drop.
// A drag does not officially start until this is called.
func (em *Events) dragStart(w Widget, data any, e events.Event) {
	ms := em.scene.Stage.Main
	if ms == nil {
		return
	}
	em.drag = w
	em.dragData = data
	sp := NewSprite(dragSpriteName, image.Point{}, e.WindowPos())
	sp.grabRenderFrom(w) // TODO: maybe show the number of items being dragged
	sp.Pixels = clone.AsRGBA(gradient.ApplyOpacity(sp.Pixels, 0.5))
	sp.Active = true
	ms.Sprites.Add(sp)
}

// dragMove is generally handled entirely by the event manager
func (em *Events) dragMove(e events.Event) {
	ms := em.scene.Stage.Main
	if ms == nil {
		return
	}
	sp, ok := ms.Sprites.SpriteByName(dragSpriteName)
	if !ok {
		fmt.Println("Drag sprite not found")
		return
	}
	sp.Geom.Pos = e.WindowPos()
	for _, w := range em.dragHovers {
		w.AsWidget().ScrollToThis()
	}
	em.scene.NeedsRender()
}

func (em *Events) dragClearSprite() {
	ms := em.scene.Stage.Main
	if ms == nil {
		return
	}
	ms.Sprites.InactivateSprite(dragSpriteName)
}

func (em *Events) dragMenuAddModText(m *Scene, mod events.DropMods) {
	text := ""
	switch mod {
	case events.DropCopy:
		text = "Copy (use Shift to move):"
	case events.DropMove:
		text = "Move:"
	}
	NewText(m).SetType(TextLabelLarge).SetText(text).Styler(func(s *styles.Style) {
		s.Margin.Set(units.Em(0.5))
	})
}

// dragDrop sends the [events.Drop] event to the top of the DragHovers stack.
// clearing the current dragging sprite before doing anything.
// It is up to the target to call
func (em *Events) dragDrop(drag Widget, e events.Event) {
	em.dragClearSprite()
	data := em.dragData
	em.drag = nil
	if len(em.dragHovers) == 0 {
		if DebugSettings.EventTrace {
			fmt.Println(drag, "Drop has no target")
		}
		return
	}
	for _, dwi := range em.dragHovers {
		dwi.AsWidget().SetState(false, states.DragHovered)
	}
	targ := em.dragHovers[len(em.dragHovers)-1]
	de := events.NewDragDrop(events.Drop, e.(*events.Mouse)) // gets the actual mod at this point
	de.Data = data
	de.Source = drag
	de.Target = targ
	if DebugSettings.EventTrace {
		fmt.Println(targ, "Drop with mod:", de.DropMod, "source:", de.Source)
	}
	targ.AsWidget().HandleEvent(de)
}

// dropFinalize should be called as the last step in the Drop event processing,
// to send the DropDeleteSource event to the source in case of DropMod == DropMove.
// Otherwise, nothing actually happens.
func (em *Events) dropFinalize(de *events.DragDrop) {
	if de.DropMod != events.DropMove {
		return
	}
	de.Typ = events.DropDeleteSource
	de.ClearHandled()
	de.Source.(Widget).AsWidget().HandleEvent(de)
}

// Clipboard returns the [system.Clipboard], supplying the window context
// if available.
func (em *Events) Clipboard() system.Clipboard {
	var gwin system.Window
	if win := em.RenderWindow(); win != nil {
		gwin = win.SystemWindow
	}
	return system.TheApp.Clipboard(gwin)
}

// setCursor sets the window cursor to the given [cursors.Cursor].
func (em *Events) setCursor(cur cursors.Cursor) {
	win := em.RenderWindow()
	if win == nil {
		return
	}
	if !win.isVisible() {
		return
	}
	errors.Log(system.TheApp.Cursor(win.SystemWindow).Set(cur))
}

// focusClear saves current focus to FocusPrev
func (em *Events) focusClear() bool {
	if em.focus != nil {
		if DebugSettings.FocusTrace {
			fmt.Println(em.scene, "FocusClear:", em.focus)
		}
		em.prevFocus = em.focus
	}
	return em.setFocusEvent(nil)
}

// setFocus sets focus to given item, and returns true if focus changed.
// If item is nil, then nothing has focus.
// This does NOT send the events.Focus event to the widget.
// See [SetFocusEvent] for version that does send event.
func (em *Events) setFocus(w Widget) bool {
	if DebugSettings.FocusTrace {
		fmt.Println(em.scene, "SetFocus:", w)
	}
	got := em.setFocusImpl(w, false) // no event
	if !got {
		if DebugSettings.FocusTrace {
			fmt.Println(em.scene, "SetFocus: Failed", w)
		}
		return false
	}
	if w != nil {
		w.AsWidget().ScrollToThis()
	}
	return got
}

// setFocusEvent sets focus to given item, and returns true if focus changed.
// If item is nil, then nothing has focus.
// This sends the [events.Focus] event to the widget.
// See [SetFocus] for a version that does not.
func (em *Events) setFocusEvent(w Widget) bool {
	if DebugSettings.FocusTrace {
		fmt.Println(em.scene, "SetFocusEvent:", w)
	}
	got := em.setFocusImpl(w, true) // sends event
	if !got {
		if DebugSettings.FocusTrace {
			fmt.Println(em.scene, "SetFocusEvent: Failed", w)
		}
		return false
	}
	if w != nil {
		w.AsWidget().ScrollToThis()
	}
	return got
}

// setFocusImpl sets focus to given item -- returns true if focus changed.
// If item is nil, then nothing has focus.
// sendEvent determines whether the events.Focus event is sent to the focused item.
func (em *Events) setFocusImpl(w Widget, sendEvent bool) bool {
	cfoc := em.focus
	if cfoc == nil {
		em.focus = nil
		cfoc = nil
	}
	if cfoc != nil && w != nil && cfoc == w {
		if DebugSettings.FocusTrace {
			fmt.Println(em.scene, "Already Focus:", cfoc)
		}
		// if sendEvent { // still send event
		// 	w.Send(events.Focus)
		// }
		return false
	}
	if cfoc != nil {
		if DebugSettings.FocusTrace {
			fmt.Println(em.scene, "Losing focus:", cfoc)
		}
		cfoc.AsWidget().Send(events.FocusLost)
	}
	em.focus = w
	if sendEvent && w != nil {
		w.AsWidget().Send(events.Focus)
	}
	return true
}

// focusNext sets the focus on the next item
// that can accept focus after the current Focus item.
// returns true if a focus item found.
func (em *Events) focusNext() bool {
	if em.focus == nil {
		return em.focusFirst()
	}
	return em.FocusNextFrom(em.focus)
}

// FocusNextFrom sets the focus on the next item
// that can accept focus after the given item.
// It returns true if a focus item is found.
func (em *Events) FocusNextFrom(from Widget) bool {
	next := widgetNextFunc(from, func(w Widget) bool {
		wb := w.AsWidget()
		return wb.IsVisible() && !wb.StateIs(states.Disabled) && wb.AbilityIs(abilities.Focusable)
	})
	em.setFocusEvent(next)
	return next != nil
}

// focusOnOrNext sets the focus on the given item, or the next one that can
// accept focus; returns true if a new focus item is found.
func (em *Events) focusOnOrNext(foc Widget) bool {
	cfoc := em.focus
	if cfoc == foc {
		return true
	}
	wb := AsWidget(foc)
	if !wb.IsVisible() {
		return false
	}
	if wb.AbilityIs(abilities.Focusable) {
		em.setFocusEvent(foc)
		return true
	}
	return em.FocusNextFrom(foc)
}

// focusOnOrPrev sets the focus on the given item, or the previous one that can
// accept focus; returns true if a new focus item is found.
func (em *Events) focusOnOrPrev(foc Widget) bool {
	cfoc := em.focus
	if cfoc == foc {
		return true
	}
	wb := AsWidget(foc)
	if !wb.IsVisible() {
		return false
	}
	if wb.AbilityIs(abilities.Focusable) {
		em.setFocusEvent(foc)
		return true
	}
	return em.focusPrevFrom(foc)
}

// focusPrev sets the focus on the previous item before the
// current focus item.
func (em *Events) focusPrev() bool {
	if em.focus == nil {
		return em.focusLast()
	}
	return em.focusPrevFrom(em.focus)
}

// focusPrevFrom sets the focus on the previous item before the given item
// (can be nil).
func (em *Events) focusPrevFrom(from Widget) bool {
	prev := widgetPrevFunc(from, func(w Widget) bool {
		wb := w.AsWidget()
		return wb.IsVisible() && !wb.StateIs(states.Disabled) && wb.AbilityIs(abilities.Focusable)
	})
	em.setFocusEvent(prev)
	return prev != nil
}

// focusFirst sets the focus on the first focusable item in the tree.
// returns true if a focusable item was found.
func (em *Events) focusFirst() bool {
	return em.FocusNextFrom(em.scene.This.(Widget))
}

// focusLast sets the focus on the last focusable item in the tree.
// returns true if a focusable item was found.
func (em *Events) focusLast() bool {
	return em.focusLastFrom(em.scene)
}

// focusLastFrom sets the focus on the last focusable item in the given tree.
// returns true if a focusable item was found.
func (em *Events) focusLastFrom(from Widget) bool {
	last := tree.Last(from).(Widget)
	return em.focusOnOrPrev(last)
}

// SetStartFocus sets the given item to be the first focus when the window opens.
func (em *Events) SetStartFocus(k Widget) {
	em.startFocus = k
}

// activateStartFocus activates start focus if there is no current focus
// and StartFocus is set -- returns true if activated
func (em *Events) activateStartFocus() bool {
	if em.startFocus == nil && !em.startFocusFirst {
		// fmt.Println("no start focus")
		return false
	}
	sf := em.startFocus
	em.startFocus = nil
	if sf == nil {
		em.focusFirst()
	} else {
		// fmt.Println("start focus on:", sf)
		em.setFocusEvent(sf)
	}
	return true
}

// MangerKeyChordEvents handles lower-priority manager-level key events.
// Mainly tab, shift-tab, and Inspector and Settings.
// event will be marked as processed if handled here.
func (em *Events) managerKeyChordEvents(e events.Event) {
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
	sc := em.scene
	cs := e.KeyChord()
	kf := keymap.Of(cs)
	switch kf {
	case keymap.FocusNext: // tab
		if em.focusNext() {
			e.SetHandled()
		}
	case keymap.FocusPrev: // shift-tab
		if em.focusPrev() {
			e.SetHandled()
		}
	case keymap.Menu:
		if tb := sc.GetTopAppBar(); tb != nil {
			ch := tree.ChildByType[*Chooser](tb)
			if ch != nil {
				ch.SetFocusEvent()
				ch.textField.offerComplete()
			} else {
				tb.SetFocusEvent()
			}
			e.SetHandled()
		}
	case keymap.WinSnapshot:
		dstr := time.Now().Format(time.DateOnly + "-" + "15-04-05")
		fnm := filepath.Join(TheApp.AppDataDir(), "screenshot-"+sc.Name+"-"+dstr+".png")
		if errors.Log(imagex.Save(sc.Pixels, fnm)) == nil {
			fmt.Println("Saved screenshot to", strings.ReplaceAll(fnm, " ", `\ `))
		}
		e.SetHandled()
	case keymap.ZoomIn:
		win.stepZoom(1)
		e.SetHandled()
	case keymap.ZoomOut:
		win.stepZoom(-1)
		e.SetHandled()
	case keymap.Refresh:
		e.SetHandled()
		system.TheApp.GetScreens()
		UpdateAll()
		theWindowGeometrySaver.restoreAll()
		// w.FocusInactivate()
		// w.FullReRender()
		// sz := w.SystemWin.Size()
		// w.SetSize(sz)
	case keymap.WinFocusNext:
		e.SetHandled()
		AllRenderWindows.focusNext()
	}
	if !e.IsHandled() {
		em.triggerShortcut(cs)
	}
}

// getShortcuts gathers all [Button]s in the Scene with a shortcut specified.
// It recursively navigates [Button.Menu]s.
func (em *Events) getShortcuts() {
	em.shortcuts = nil
	em.getShortcutsIn(em.scene)
}

// getShortcutsIn gathers all [Button]s in the given parent widget with
// a shortcut specified. It recursively navigates [Button.Menu]s.
func (em *Events) getShortcutsIn(parent Widget) {
	parent.AsWidget().WidgetWalkDown(func(w Widget, wb *WidgetBase) bool {
		bt := AsButton(w)
		if bt == nil {
			return tree.Continue
		}
		if bt.Shortcut != "" {
			em.addShortcut(bt.Shortcut.PlatformChord(), bt)
		}
		if bt.HasMenu() {
			tmps := NewScene()
			bt.Menu(tmps)
			em.getShortcutsIn(tmps)
		}
		return tree.Continue
	})
}

// shortcuts is a map between a key chord and a specific Button that can be
// triggered.  This mapping must be unique, in that each chord has unique
// Button, and generally each Button only has a single chord as well, though
// this is not strictly enforced.  shortcuts are evaluated *after* the
// standard KeyMap event processing, so any conflicts are resolved in favor of
// the local widget's key event processing, with the shortcut only operating
// when no conflicting widgets are in focus.  shortcuts are always window-wide
// and are intended for global window / toolbar buttons.  Widget-specific key
// functions should be handled directly within widget key event
// processing.
type shortcuts map[key.Chord]*Button

// addShortcut adds given shortcut to given button.
func (em *Events) addShortcut(chord key.Chord, bt *Button) {
	if chord == "" {
		return
	}
	if em.shortcuts == nil {
		em.shortcuts = shortcuts{}
	}
	chords := strings.Split(string(chord), "\n")
	for _, c := range chords {
		cc := key.Chord(c)
		if DebugSettings.KeyEventTrace {
			old, exists := em.shortcuts[cc]
			if exists && old != bt {
				slog.Error("Events.AddShortcut: overwriting duplicate shortcut", "shortcut", cc, "originalButton", old, "newButton", bt)
			}
		}
		em.shortcuts[cc] = bt
	}
}

// triggerShortcut attempts to trigger a shortcut, returning true if one was
// triggered, and false otherwise.  Also eliminates any shortcuts with deleted
// buttons, and does not trigger for Disabled buttons.
func (em *Events) triggerShortcut(chord key.Chord) bool {
	if DebugSettings.KeyEventTrace {
		fmt.Printf("Shortcut chord: %v -- looking for button\n", chord)
	}
	if em.shortcuts == nil {
		return false
	}
	sa, exists := em.shortcuts[chord]
	if !exists {
		return false
	}
	if sa == nil || sa.This == nil {
		delete(em.shortcuts, chord)
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

func (em *Events) getSpriteInBBox(sc *Scene, pos image.Point) {
	st := sc.Stage
	for _, kv := range st.Sprites.Names.Order {
		sp := kv.Value
		if !sp.Active {
			continue
		}
		if sp.listeners == nil {
			continue
		}
		r := sp.Geom.Bounds()
		if pos.In(r) {
			em.spriteInBBox = append(em.spriteInBBox, sp)
		}
	}
}

// handleSpriteEvent handles the given event with sprites
// returns true if event was handled
func (em *Events) handleSpriteEvent(e events.Event) bool {
	et := e.Type()
loop:
	for _, sp := range em.spriteInBBox {
		if e.IsHandled() {
			break
		}
		sp.listeners.Call(e) // everyone gets the primary event who is in scope, deepest first
		switch et {
		case events.MouseDown:
			if sp.listeners.HandlesEventType(events.SlideMove) {
				e.SetHandled()
				em.spriteSlide = sp
				em.spriteSlide.send(events.SlideStart, e)
			}
			if sp.listeners.HandlesEventType(events.Click) {
				em.spritePress = sp
			}
			break loop
		case events.MouseUp:
			sp.handleEvent(e)
			if em.spriteSlide == sp {
				sp.send(events.SlideStop, e)
			}
			if em.spritePress == sp {
				sp.send(events.Click, e)
			}
		}
	}
	return e.IsHandled()
}
