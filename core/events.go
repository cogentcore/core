// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
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

// dragCopySpriteName is the name of the sprite of the green plus on the
// drag sprite, which shows that the drop would copy.
const dragCopySpriteName = "__DragCopySprite__"

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

	// stack of hovered sprites: have mouse pointer in BBox.
	spriteHovers []*Sprite

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

	// widget receiving keyboard events. Use SetFocus.
	focus Widget

	// currently attended widget. Use SetAttend.
	attended Widget

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

	// dragImage is the image of the drag sprite, from DragStart,
	// used for a drag handed off to the operating system.
	dragImage *image.RGBA

	// dragFiles are the files of the current drag, from SetDragFiles.
	dragFiles []string

	// dragOS is set when the current drag has been handed off to the
	// operating system after leaving the window (see [Events.dragToOS]).
	dragOS bool

	// dragMoves is whether the last drag moves by default, instead of
	// copying, as set by its source (see [dragMover]).
	dragMoves bool

	// dragCopy is whether the drag sprite shows that the drop would copy.
	dragCopy bool

	// dragCopyOffset is the offset of the sprite of the green plus from
	// the drag sprite.
	dragCopyOffset image.Point
}

// dragMover is implemented by widgets with an option for whether dragging
// their items moves them by default, instead of copying them, in which
// case the modifier keys that otherwise move them copy them.
type dragMover interface {
	dragMovesDefault() bool
}

// dragMovesFrom returns whether a drag from the given widget moves by
// default, as set by it or the first widget above it that is a [dragMover].
func dragMovesFrom(w Widget) bool {
	for n := tree.Node(w); n != nil; n = n.AsTree().Parent {
		if dm, ok := n.(dragMover); ok {
			return dm.dragMovesDefault()
		}
	}
	return false
}

// mains returns the stack of main stages for our scene.
func (em *Events) mains() *stages {
	if em.scene == nil || em.scene.Stage == nil { // no stage when generating HTML
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
	if de, ok := e.(*events.DragDrop); ok {
		em.handleExternalDrag(de)
		return
	}
	if !tree.IsNil(em.drag) && e.NeedsFocus() { // the modifier keys of the drop
		em.updateDragSprite(dragKeyMods(e))
	}
	switch {
	case e.HasPos():
		em.handlePosEvent(e)
	case e.NeedsFocus():
		em.handleFocusEvent(e)
	default: // eg. os events
		em.scene.HandleEvent(e)
	}
}

func (em *Events) handleFocusEvent(e events.Event) {
	// key down and key up can not give active focus, only key chord
	if tree.IsNil(em.focus) && e.Type() != events.KeyDown && e.Type() != events.KeyUp {
		switch {
		case !tree.IsNil(em.startFocus):
			if DebugSettings.FocusTrace {
				fmt.Println(em.scene, "StartFocus:", em.startFocus)
			}
			em.setFocus(em.startFocus)
		case !tree.IsNil(em.prevFocus):
			if DebugSettings.FocusTrace {
				fmt.Println(em.scene, "PrevFocus:", em.prevFocus)
			}
			em.setFocus(em.prevFocus)
			em.prevFocus = nil
		}
	}
	if !tree.IsNil(em.focus) {
		em.focus.AsTree().WalkUpParent(func(k tree.Node) bool {
			wb := AsWidget(k)
			if !wb.IsDisplayable() {
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
				if !wb.IsDisplayable() {
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
	em.dragOS = false
	em.dragPress = nil
	em.slide = nil
	em.slidePress = nil
	em.spriteSlide = nil
	em.spritePress = nil

	em.cancelRepeatClick()

	// if we have sent a long hover start event, we send an end
	// event (non-nil widget plus nil timer means we already sent)
	if !tree.IsNil(em.longHoverWidget) && em.longHoverTimer == nil {
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

	if em.handleDragOS(e) {
		return
	}

	switch et {
	case events.MouseDown:
		em.resetOnMouseDown()
	case events.MouseDrag:
		if em.spriteSlide != nil {
			em.spriteSlide.handleEvent(e)
			em.spriteSlide.send(events.SlideMove, e)
			e.SetHandled()
			em.setCursorFromStyle()
			return
		}
		if !tree.IsNil(em.slide) {
			if len(sc.selectedText) > 0 && !em.slide.AsWidget().posInScBBox(pos) {
				// allow multi-select
			} else {
				em.slide.AsWidget().HandleEvent(e)
				em.slide.AsWidget().Send(events.SlideMove, e)
				e.SetHandled()
				em.setCursorFromStyle()
				return
			}
		}
	case events.Scroll:
		if !tree.IsNil(em.scroll) {
			scInTime := time.Since(em.lastScrollTime) < TimingSettings.ScrollFocusTime
			if scInTime {
				em.scroll.AsWidget().HandleEvent(e)
				if e.IsHandled() {
					em.lastScrollTime = time.Now()
				}
				em.setCursorFromStyle()
				return
			}
			em.scroll = nil
		}
	}

	em.spriteInBBox = nil
	em.getSpriteInBBox(sc, e.WindowPos())
	if len(em.spriteInBBox) > 0 {
		if em.handleSpriteEvent(e) {
			em.setCursorFromStyle()
			return
		}
	}

	em.mouseInBBox = nil
	em.getMouseInScene(sc, pos)
	em.mouseInBBox = sortByRenderLayer(em.mouseInBBox)

	n := len(em.mouseInBBox)
	if n == 0 {
		if DebugSettings.EventTrace && et != events.MouseMove {
			log.Println("Nothing in bbox:", sc.Geom.TotalBBox, "pos:", pos)
		}
		em.setCursorFromStyle()
		return
	}

	var press, dragPress, slidePress, move, up, repeatClick Widget
	slidePriority := false // see [abilities.SlidePriority]
	for i := n - 1; i >= 0; i-- {
		w := em.mouseInBBox[i]
		wb := w.AsWidget()

		// we need to handle this here and not in [Events.GetMouseInBBox] so that
		// we correctly process cursors for disabled elements.
		// in ScRenderBBoxes, everyone is effectively enabled
		if wb.StateIs(states.Disabled) && !sc.renderBBoxes {
			continue
		}
		// everyone gets the primary event who is in scope, deepest first
		if et == events.Scroll {
			if wb.AbilityIs(abilities.ScrollableUnattended) || (wb.StateIs(states.Focused) || wb.StateIs(states.Attended)) {
				w.AsWidget().HandleEvent(e)
			}
		} else {
			w.AsWidget().HandleEvent(e)
		}
		if tree.IsNil(w) { // died while handling
			continue
		}
		switch et {
		case events.MouseMove:
			em.scroll = nil
			if tree.IsNil(move) && wb.Styles.Abilities.IsHoverable() {
				move = w
			}
		case events.MouseDown:
			em.scroll = nil
			// in ScRenderBBoxes, everyone is effectively pressable
			if tree.IsNil(press) && (wb.Styles.Abilities.IsPressable() || sc.renderBBoxes) {
				press = w
			}
			if tree.IsNil(dragPress) && !slidePriority && wb.AbilityIs(abilities.Draggable) {
				dragPress = w
			}
			if tree.IsNil(slidePress) && wb.AbilityIs(abilities.Slideable) {
				// On mobile, sliding results in scrolling, so we must have the appropriate
				// scrolling attention to allow sliding.
				if TheApp.SystemPlatform().IsMobile() && !wb.AbilityIs(abilities.ScrollableUnattended) && !(wb.StateIs(states.Focused) || wb.StateIs(states.Attended)) {
					continue
				}
				slidePress = w
				slidePriority = wb.AbilityIs(abilities.SlidePriority)
			}
			if repeatClick == nil && wb.Styles.Abilities.Is(abilities.RepeatClickable) {
				repeatClick = w
			}
		case events.MouseUp:
			em.scroll = nil
			// in ScRenderBBoxes, everyone is effectively pressable
			if tree.IsNil(up) && (wb.Styles.Abilities.IsPressable() || sc.renderBBoxes) {
				up = w
			}
		case events.Scroll:
			if !wb.AbilityIs(abilities.ScrollableUnattended) && !(wb.StateIs(states.Focused) || wb.StateIs(states.Attended)) {
				continue
			}
			if e.IsHandled() {
				if tree.IsNil(em.scroll) {
					em.scroll = w
					em.lastScrollTime = time.Now()
				}
			}
		}
	}
	switch et {
	case events.MouseDown:
		if !tree.IsNil(press) {
			em.press = press
			em.setAttend(press)
		}
		if !tree.IsNil(dragPress) {
			em.dragPress = dragPress
		}
		if !tree.IsNil(slidePress) {
			em.slidePress = slidePress
		}
		if !tree.IsNil(repeatClick) {
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
		if !tree.IsNil(em.drag) { // this means we missed the drop
			em.dragHovers = em.updateHovers(hovs, em.dragHovers, e, events.DragEnter, events.DragLeave)
			em.dragDrop(em.drag, e)
			break
		}
		if sc.renderBBoxes {
			pselw := sc.selectedWidget
			if len(em.hovers) > 0 {
				sc.selectedWidget = em.hovers[len(em.hovers)-1]
			} else {
				sc.selectedWidget = nil
			}
			if sc.selectedWidget != pselw {
				if !tree.IsNil(pselw) {
					pselw.AsWidget().NeedsRender()
				}
				if !tree.IsNil(sc.selectedWidget) {
					sc.selectedWidget.AsWidget().NeedsRender()
				}
			}
		}
		em.hovers = em.updateHovers(hovs, em.hovers, e, events.MouseEnter, events.MouseLeave)
		em.handleLongHover(e)
	case events.MouseDrag:
		if !tree.IsNil(em.drag) {
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
			e.SetHandled()
		} else {
			if !tree.IsNil(em.dragPress) && em.dragStartCheck(e, TimingSettings.DragStartTime, TimingSettings.DragStartDistance) {
				em.cancelRepeatClick()
				em.cancelLongPress()
				em.dragPress.AsWidget().Send(events.DragStart, e)
				e.SetHandled()
			} else if !tree.IsNil(em.slidePress) && em.dragStartCheck(e, TimingSettings.SlideStartTime, TimingSettings.DragStartDistance) {
				em.cancelRepeatClick()
				em.cancelLongPress()
				if em.slide != nil { // out of original bbox
					var nextSlide Widget
					for i := n - 1; i >= 0; i-- {
						w := em.mouseInBBox[i]
						if _, ok := w.(*Text); ok {
							nextSlide = w
							break
						}
					}
					slb := em.slide.AsWidget()
					if pos.Y > slb.Geom.TotalBBox.Max.Y || pos.X > slb.Geom.TotalBBox.Max.X {
						ec := e.Clone()
						ec.AsBase().WhereLocal = slb.Geom.TotalBBox.Max
						slb.Send(events.SlideMove, ec)
					}
					if nextSlide != nil {
						em.slide = nextSlide
						slb = em.slide.AsWidget()
						ec := e.Clone()
						ec.AsBase().WhereLocal = slb.Geom.TotalBBox.Min
						slb.Send(events.SlideStart, ec)
					}
				} else {
					em.slide = em.slidePress
					em.slide.AsWidget().Send(events.SlideStart, e)
				}
				e.SetHandled()
			}
		}
		// if we already have a long press widget, we update it based on our dragging movement
		if !tree.IsNil(em.longPressWidget) {
			em.handleLongPress(e)
		}
	case events.MouseUp:
		em.cancelRepeatClick()
		if !tree.IsNil(em.slide) {
			em.slide.AsWidget().Send(events.SlideStop, e)
			e.SetHandled()
			em.slide = nil
			em.press = nil
		}
		if !tree.IsNil(em.drag) {
			em.dragDrop(em.drag, e)
			em.press = nil
		}
		// if we have sent a long press start event, we don't send click
		// events (non-nil widget plus nil timer means we already sent)
		if em.press == up && !tree.IsNil(up) && !(!tree.IsNil(em.longPressWidget) && em.longPressTimer == nil) {
			em.cancelLongPress()
			switch e.MouseButton() {
			case events.Left:
				if sc.selectedWidgetChan != nil {
					sc.selectedWidgetChan <- up
					em.setCursorFromStyle()
					return
				}
				dcInTime := time.Since(em.lastClickTime) < TimingSettings.DoubleClickInterval
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
		if em.press != up && !tree.IsNil(em.press) {
			em.press.AsWidget().HandleEvent(e)
		}
		em.press = nil

		em.cancelLongPress()
		// a mouse up event acts also acts as a mouse leave
		// event on mobile, as that is needed to clear any
		// hovered state
		if !tree.IsNil(up) && TheApp.Platform().IsMobile() {
			up.AsWidget().Send(events.MouseLeave, e)
		}
	case events.Scroll:
		switch {
		case !tree.IsNil(em.slide):
			em.slide.AsWidget().HandleEvent(e)
		case !tree.IsNil(em.drag):
			em.drag.AsWidget().HandleEvent(e)
		case !tree.IsNil(em.press):
			em.press.AsWidget().HandleEvent(e)
		default:
			em.scene.HandleEvent(e)
		}
	}
	em.setCursorFromStyle()
}

// updateHovers updates the hovered widgets based on current
// widgets in bounding box.
func (em *Events) updateHovers(hov, prev []Widget, e events.Event, enter, leave events.Types) []Widget {
	for _, prv := range prev {
		if !slices.Contains(hov, prv) && !tree.IsNil(prv) {
			prv.AsWidget().Send(leave, e)
		}
	}
	for _, cur := range hov {
		if !slices.Contains(prev, cur) {
			cur.AsWidget().Send(enter, e)
		}
	}
	// todo: detect change in top one, use to update cursor
	return hov
}

// updateSpriteHovers updates the hovered sprites based on current
// sprites in bounding box.
func (em *Events) updateSpriteHovers(hov, prev []*Sprite, e events.Event, enter, leave events.Types) []*Sprite {
	for _, prv := range prev {
		if !slices.Contains(hov, prv) {
			prv.send(leave, e)
		}
	}
	for _, cur := range hov {
		if !slices.Contains(prev, cur) {
			cur.send(enter, e)
		}
	}
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
	em.handleLong(e, em.topLongHover(), &em.longHoverWidget, &em.longHoverPos, &em.longHoverTimer, events.LongHoverStart, events.LongHoverEnd, TimingSettings.LongHoverTime, TimingSettings.LongHoverStopDistance)
}

// handleLongPress handles long press events
func (em *Events) handleLongPress(e events.Event) {
	em.handleLong(e, em.press, &em.longPressWidget, &em.longPressPos, &em.longPressTimer, events.LongPressStart, events.LongPressEnd, TimingSettings.LongPressTime, TimingSettings.LongPressStopDistance)
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
	if tree.IsNil(deep) {
		// fmt.Println("no deep")
		if tree.IsNil(*w) {
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
	if !tree.IsNil(*w) {
		(*w).AsWidget().Send(etyp, e)
		clearLong()
		// fmt.Println("lhw, send end, cleared")
		return
	}

	// now we can set it to our new widget
	*w = deep
	// fmt.Println("setting new:", deep)
	*pos = e.WindowPos()
	// The window is found here, in the event loop, as the timer function
	// can only read the stage of the scene under the render context lock,
	// as the scene can be closed in the meantime.
	win := em.RenderWindow()
	*t = time.AfterFunc(stime, func() {
		if win == nil {
			return
		}
		rc := win.renderContext() // have to get this one first
		rc.Lock()
		defer rc.Unlock()

		em.timerMu.Lock() // then can get this
		defer em.timerMu.Unlock()
		if tree.IsNil(*w) || em.mains() == nil { // the scene was closed
			return
		}
		(*w).AsWidget().Send(styp, e)
		// we are done with the timer, and this indicates that
		// we have sent a start event
		*t = nil
	})
}

// getMouseInBBox adds the widgets from the given one down that are under the
// given position to [Events.mouseInBBox], in the order of the tree. If
// layerOnly is true, it only goes into the widgets that are in the layer of
// the scene or have widgets in it (see [Events.getMouseInScene]).
func (em *Events) getMouseInBBox(w Widget, pos image.Point, layerOnly bool) {
	wb := w.AsWidget()
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		// we do not handle disabled here so that
		// we correctly process cursors for disabled elements.
		// it needs to be handled downstream by anyone who needs it.
		if cwb.StateIs(states.Invisible) { // (its ancestors are displayable)
			return tree.Break
		}
		if cwb.Styles.Opacity <= 0 && cwb.Styles.Abilities == 0 {
			// it is not rendered, and neither is anything in it (as for
			// visibility: hidden in CSS): only a transparent widget that
			// is clickable (as a hidden checkbox over a button) gets events
			return tree.Break
		}
		if layerOnly && !cwb.hasFlag(widgetInLayer) && !cwb.hasFlag(widgetHasLayer) {
			return tree.Break // nothing of the layer in it
		}
		if !cwb.posInScBBox(pos) {
			if cwb.noClip() {
				// its children can be outside of it (as positioned ones,
				// as the toggles of the contents of Wikipedia)
				return tree.Continue
			}
			return tree.Break
		}
		if !cwb.Styles.Abilities.HasFlag(abilities.NoPointerEvents) {
			em.mouseInBBox = append(em.mouseInBBox, cw)
		}
		if cwb.Parts != nil {
			em.getMouseInBBox(cwb.Parts, pos, layerOnly)
		}
		if ly := AsFrame(cw); ly != nil {
			for d := math32.X; d <= math32.Y; d++ {
				if ly.HasScroll[d] && ly.Scrolls[d] != nil {
					sb := ly.Scrolls[d]
					em.getMouseInBBox(sb, pos, layerOnly)
				}
			}
		}
		return tree.Continue
	})
}

// getMouseInScene adds the widgets of the scene under the given position to
// [Events.mouseInBBox] (see [Events.getMouseInBBox]), only those in its layer
// and the widgets that they are in if one of those in the layer is under it
// (see [sceneLayer]), as on a dialog of a web page: the layer is rendered
// over all of the other widgets, so one of them is on top, and the widgets
// under it would not get the events (see [sortByRenderLayer]), so the
// widgets of the page under it are not gone through.
func (em *Events) getMouseInScene(sc *Scene, pos image.Point) {
	if sc.layer.active {
		em.getMouseInBBox(sc, pos, true)
		for _, w := range em.mouseInBBox {
			if w.AsWidget().hasFlag(widgetInLayer) {
				return
			}
		}
		em.mouseInBBox = em.mouseInBBox[:0]
	}
	em.getMouseInBBox(sc, pos, false)
}

// WidgetAt returns the widget that mouse events at the given position (in
// the dots of the scene) go to first: the deepest one rendered on top
// there (see [sortByRenderLayer]), as for elementFromPoint in the scripts
// of web pages. It returns nil if there is none.
func (sc *Scene) WidgetAt(pos image.Point) Widget {
	em := &sc.Events
	saved := em.mouseInBBox
	em.mouseInBBox = nil
	em.getMouseInScene(sc, pos)
	ws := sortByRenderLayer(em.mouseInBBox)
	em.mouseInBBox = saved
	if len(ws) == 0 {
		return nil
	}
	return ws[len(ws)-1]
}

// sortByRenderLayer sorts the given widgets under the mouse (in the order
// of the tree, with those that are handled first last) by the order in
// which they are rendered (see [WidgetBase.renderOrder]), so that a widget
// that is rendered over another (as one with a position) gets the events.
// Of two widgets, the one in the child of their nearest common ancestor
// that is rendered later is later, as is a widget in another.
// It returns only the widget that is rendered on top and the widgets that
// it is in, as in CSS: the widgets under one that is rendered over them
// (as those under an open menu) do not get the events (as hover).
func sortByRenderLayer(ws []Widget) []Widget {
	layered := false
	for _, w := range ws {
		if renderLayer(w.AsWidget()) != 0 {
			layered = true
			break
		}
	}
	if !layered {
		return ws
	}
	// path returns the widgets from the root to the given one
	path := func(w *WidgetBase) []*WidgetBase {
		var p []*WidgetBase
		for ; w != nil; w = w.parentWidget() {
			p = append(p, w)
		}
		slices.Reverse(p)
		return p
	}
	paths := make([][]*WidgetBase, len(ws))
	for i, w := range ws {
		paths[i] = path(w.AsWidget())
	}
	idx := make([]int, len(ws))
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int {
		pa, pb := paths[a], paths[b]
		for i := 0; i < len(pa) && i < len(pb); i++ {
			if pa[i] != pb[i] { // the children of their common ancestor
				if c := cmp.Compare(pathLayer(pa, i), pathLayer(pb, i)); c != 0 {
					return c
				}
				break
			}
		}
		return cmp.Compare(a, b) // the order of the tree
	})
	top := paths[idx[len(idx)-1]]
	in := make(map[*WidgetBase]bool, len(top))
	for _, w := range top {
		in[w] = true
	}
	sorted := make([]Widget, 0, len(ws))
	for _, j := range idx {
		if in[ws[j].AsWidget()] {
			sorted = append(sorted, ws[j])
		}
	}
	return sorted
}

// pathLayer returns the layer in which the widget at the given index of the
// given path (from the top) is rendered among its siblings (see
// [renderLayer]), or that of a widget in it that is raised to a stacking
// context above it (see [WidgetBase.raisedTo]), which is rendered over the
// widgets after the ones that it is in, if that is higher.
func pathLayer(path []*WidgetBase, i int) int {
	l := renderLayer(path[i])
	for j := i + 1; j < len(path); j++ {
		r := path[j].raisedTo()
		if r == nil {
			continue
		}
		for k := 0; k < i; k++ {
			if path[k] == r {
				l = max(l, 2+path[j].Styles.ZIndex)
				break
			}
		}
	}
	return l
}

// renderLayer returns the layer in which the widget is rendered among its
// siblings (see [WidgetBase.renderOrder]): 2 plus its z-index if it is
// positioned (or -1 for a negative z-index), 1 if it floats, or else 0.
func renderLayer(wb *WidgetBase) int {
	switch {
	case wb.stacked():
		if wb.Styles.ZIndex < 0 {
			return -1
		}
		return 2 + wb.Styles.ZIndex
	case wb.isFloat():
		return 1
	}
	return 0
}

func (em *Events) cancelLongPress() {
	// if we have sent a long press start event, we send an end
	// event (non-nil widget plus nil timer means we already sent)
	if !tree.IsNil(em.longPressWidget) && em.longPressTimer == nil {
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
	if tree.IsNil(em.repeatClick) || !em.repeatClick.AsWidget().IsVisible() {
		return
	}
	delay := TimingSettings.RepeatClickTime
	if em.repeatClickTimer == nil {
		delay *= 8
	}
	em.repeatClickTimer = time.AfterFunc(delay, func() {
		if tree.IsNil(em.repeatClick) || !em.repeatClick.AsWidget().IsVisible() {
			return
		}
		// note: we are not in event loop here anymore!
		rw := em.repeatClick.AsWidget()
		rw.AsyncLock()
		rw.Send(events.Click)
		rw.AsyncUnlock()
		em.startRepeatClickTimer()
	})
}

func (em *Events) dragStartCheck(e events.Event, dur time.Duration, dist int) bool {
	since := e.SinceStart()
	if since < dur {
		return false
	}
	dst := int(math32.FromPoint(e.StartDelta()).Length())
	return dst >= dist
}

// DragStart starts a drag event, capturing a sprite image of the given widget
// and storing the data for later use during Drop.
// A drag does not officially start until this is called.
func (em *Events) DragStart(w Widget, data any, e events.Event) {
	ms := em.scene.Stage.Main
	if ms == nil {
		return
	}
	em.drag = w
	em.dragData = data
	em.slide = nil // a drag replaces any slide in progress
	em.dragFiles = nil
	if fd, ok := w.(FileDragger); ok {
		em.dragFiles = fd.DragFiles()
	}
	em.dragOS = false
	em.dragMoves = dragMovesFrom(w)
	em.dragCopy = events.DropModFor(e.Modifiers(), em.dragMoves) == events.DropCopy
	// TODO: maybe show the number of items being dragged
	em.dragImage = clone.AsRGBA(gradient.ApplyOpacity(grabRenderFrom(w), 0.5))
	em.addDragSprites(e.WindowPos())
}

// addDragSprites adds the drag sprite, with the drag image, at the given
// position, and the sprite on it of the green plus, which shows that the
// drop would copy, at its start, which stays in the window for wide images.
// It is at the bottom left, below the mouse pointer at the top left, if
// the image is tall enough.
func (em *Events) addDragSprites(pos image.Point) {
	ms := em.scene.Stage.Main
	if ms == nil || em.dragImage == nil {
		return
	}
	sp := NewImageSprite(dragSpriteName, pos, em.dragImage)
	sp.Active = true
	ms.Sprites.Add(sp)

	sz := em.dragImage.Bounds().Size()
	rv := units.Dp(8)
	r := min(int(math32.Ceil(rv.ToDots(&em.scene.Styles.UnitContext))), sz.X/2, sz.Y/2)
	if r < 3 {
		ms.Sprites.Delete(dragCopySpriteName)
		return
	}
	c := image.Pt(r+1, sz.Y-r-1)
	if pointer := 9 * r / 4; c.Y-r < pointer { // too short to be below the pointer, about 18dp
		c.Y = r + 1
	}
	em.dragCopyOffset = c.Sub(image.Pt(r+1, r+1))
	cp := NewImageSprite(dragCopySpriteName, pos.Add(em.dragCopyOffset), dragCopyImage(r))
	cp.Active = em.dragCopy
	ms.Sprites.Add(cp)
}

// dragCopyImage returns the image of the green plus that shows that a drop
// would copy, with the given radius.
func dragCopyImage(r int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 2*r+3, 2*r+3))
	c := image.Pt(r+1, r+1)
	arm := r * 3 / 5
	w := max(1, r/4)
	abs := func(v int) int { return max(v, -v) }
	green := color.RGBA{30, 142, 62, 255}
	for y := c.Y - r; y <= c.Y+r; y++ {
		for x := c.X - r; x <= c.X+r; x++ {
			dx, dy := x-c.X, y-c.Y
			if dx*dx+dy*dy > r*r {
				continue
			}
			plus := (abs(dx) <= arm && abs(dy) < w) || (abs(dy) <= arm && abs(dx) < w)
			if plus {
				img.Set(x, y, color.White)
			} else {
				img.Set(x, y, green)
			}
		}
	}
	return img
}

// updateDragSprite shows or hides the green plus on the drag sprite for
// whether the drop would copy, with the given modifier keys.
func (em *Events) updateDragSprite(mods key.Modifiers) {
	cp := events.DropModFor(mods, em.dragMoves) == events.DropCopy
	if cp == em.dragCopy {
		return
	}
	em.dragCopy = cp
	ms := em.scene.Stage.Main
	if ms == nil {
		return
	}
	if sp, ok := ms.Sprites.SpriteByName(dragCopySpriteName); ok {
		sp.Active = cp
		ms.Sprites.SetModified()
		em.scene.NeedsRender()
	}
}

// dragKeyMods returns the modifier keys after the given key event during a
// drag, as the press or release of a modifier key may not have it yet.
func dragKeyMods(e events.Event) key.Modifiers {
	mods := e.Modifiers()
	ke, ok := e.(*events.Key)
	if !ok {
		return mods
	}
	var m key.Modifiers
	switch ke.Code {
	case key.CodeLeftShift, key.CodeRightShift:
		m = key.Shift
	case key.CodeLeftControl, key.CodeRightControl:
		m = key.Control
	case key.CodeLeftAlt, key.CodeRightAlt:
		m = key.Alt
	case key.CodeLeftMeta, key.CodeRightMeta:
		m = key.Meta
	default:
		return mods
	}
	mods.SetFlag(e.Type() == events.KeyDown, m)
	return mods
}

// dragMove is generally handled entirely by the event manager
func (em *Events) dragMove(e events.Event) {
	ms := em.scene.Stage.Main
	if ms == nil {
		return
	}
	sp, ok := ms.Sprites.SpriteByName(dragSpriteName)
	if !ok || sp == nil {
		fmt.Println("Drag sprite not found")
		return
	}
	em.updateDragSprite(e.Modifiers())
	sp.SetPos(e.WindowPos())
	if cp, ok := ms.Sprites.SpriteByName(dragCopySpriteName); ok {
		cp.SetPos(e.WindowPos().Add(em.dragCopyOffset))
	}
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
	ms.Sprites.Delete(dragSpriteName)
	ms.Sprites.Delete(dragCopySpriteName)
}

// DragMenuAddModText adds info about key modifiers for a drag drop menu.
func (em *Events) DragMenuAddModText(m *Scene, mod events.DropMods) {
	text := ""
	switch {
	case mod == events.DropCopy && em.dragMoves:
		text = "Copy:"
	case mod == events.DropCopy:
		text = "Copy (use Shift to move):"
	case mod == events.DropMove && em.dragMoves:
		text = "Move (use Shift to copy):"
	case mod == events.DropMove:
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
	em.scene.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		cwb.dragStateReset()
		return tree.Continue
	})
	em.scene.Restyle()

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
	de.DropMod = events.DropModFor(de.Mods, em.dragMoves)
	de.Data = data
	de.Source = drag
	de.Target = targ
	if DebugSettings.EventTrace {
		fmt.Println(targ, "Drop with mod:", de.DropMod, "source:", de.Source)
	}
	targ.AsWidget().HandleEvent(de)
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
	// the source can be in another window, for a drop from the operating system
	src := de.Source.(Widget).AsWidget()
	src.runInWindow(em.scene.AsWidget(), func() {
		src.HandleEvent(de)
	})
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

// setCursorFromStyle sets the window cursor to the cursor of the bottom-most
// widget in the mouseInBBox stack that has a Styles.Cursor set.
// This should be called after every event pass.
func (em *Events) setCursorFromStyle() {
	n := len(em.mouseInBBox)
	for i := n - 1; i >= 0; i-- {
		w := em.mouseInBBox[i]
		wb := w.AsWidget()
		if wb.Styles.Cursor != cursors.None {
			em.setCursor(wb.Styles.Cursor)
			break
		}
	}
}

// focusClear saves current focus to FocusPrev
func (em *Events) focusClear() bool {
	if !tree.IsNil(em.focus) {
		if DebugSettings.FocusTrace {
			fmt.Println(em.scene, "FocusClear:", em.focus)
		}
		em.prevFocus = em.focus
	}
	return em.setFocus(nil)
}

// setFocusQuiet sets focus to given item, and returns true if focus changed.
// If item is nil, then nothing has focus.
// This does NOT send the [events.Focus] event to the widget.
// See [Events.setFocus] for version that does send an event.
func (em *Events) setFocusQuiet(w Widget) bool {
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
	return got
}

// setFocus sets focus to given item, and returns true if focus changed.
// If item is nil, then nothing has focus.
// This sends the [events.Focus] event to the widget.
// See [Events.setFocusQuiet] for a version that does not.
func (em *Events) setFocus(w Widget) bool {
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
	if !tree.IsNil(w) {
		w.AsWidget().ScrollToThis()
	}
	return got
}

// setFocusImpl sets focus to given item -- returns true if focus changed.
// If item is nil, then nothing has focus.
// sendEvent determines whether the events.Focus event is sent to the focused item.
func (em *Events) setFocusImpl(w Widget, sendEvent bool) bool {
	cfoc := em.focus
	if tree.IsNil(cfoc) {
		em.focus = nil
		cfoc = nil
	}
	if !tree.IsNil(cfoc) && !tree.IsNil(w) && cfoc == w {
		if DebugSettings.FocusTrace {
			fmt.Println(em.scene, "Already Focus:", cfoc)
		}
		return false
	}
	if !tree.IsNil(cfoc) {
		if DebugSettings.FocusTrace {
			fmt.Println(em.scene, "Losing focus:", cfoc)
		}
		cfoc.AsWidget().Send(events.FocusLost)
	}
	em.focus = w
	if sendEvent && !tree.IsNil(w) {
		w.AsWidget().Send(events.Focus)
	}
	return true
}

// focusNext sets the focus on the next item
// that can accept focus after the current Focus item.
// returns true if a focus item found.
func (em *Events) focusNext() bool {
	if tree.IsNil(em.focus) {
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
		return wb.IsDisplayable() && !wb.StateIs(states.Disabled) && wb.AbilityIs(abilities.Focusable)
	})
	em.setFocus(next)
	return !tree.IsNil(next)
}

// focusOnOrNext sets the focus on the given item, or the next one that can
// accept focus; returns true if a new focus item is found.
func (em *Events) focusOnOrNext(foc Widget) bool {
	cfoc := em.focus
	if cfoc == foc {
		return true
	}
	wb := AsWidget(foc)
	if !wb.IsDisplayable() {
		return false
	}
	if wb.AbilityIs(abilities.Focusable) {
		em.setFocus(foc)
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
	if !wb.IsDisplayable() {
		return false
	}
	if wb.AbilityIs(abilities.Focusable) {
		em.setFocus(foc)
		return true
	}
	return em.focusPrevFrom(foc)
}

// focusPrev sets the focus on the previous item before the
// current focus item.
func (em *Events) focusPrev() bool {
	if tree.IsNil(em.focus) {
		return em.focusLast()
	}
	return em.focusPrevFrom(em.focus)
}

// focusPrevFrom sets the focus on the previous item before the given item
// (can be nil).
func (em *Events) focusPrevFrom(from Widget) bool {
	prev := widgetPrevFunc(from, func(w Widget) bool {
		wb := w.AsWidget()
		return wb.IsDisplayable() && !wb.StateIs(states.Disabled) && wb.AbilityIs(abilities.Focusable)
	})
	em.setFocus(prev)
	return !tree.IsNil(prev)
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
	if tree.IsNil(em.startFocus) && !em.startFocusFirst {
		// fmt.Println("no start focus")
		return false
	}
	sf := em.startFocus
	em.startFocus = nil
	if tree.IsNil(sf) {
		em.focusFirst()
	} else {
		// fmt.Println("start focus on:", sf)
		em.setFocus(sf)
	}
	return true
}

// setAttend sets attended to given item, and returns true if attended changed.
// If item is nil, then nothing is attended.
// This sends the [events.Attend] event to the widget.
func (em *Events) setAttend(w Widget) bool {
	if DebugSettings.FocusTrace {
		fmt.Println(em.scene, "SetAttendEvent:", w)
	}
	got := em.setAttendImpl(w, true) // sends event
	if !got {
		if DebugSettings.FocusTrace {
			fmt.Println(em.scene, "SetAttendEvent: Failed", w)
		}
		return false
	}
	return got
}

// setAttendImpl sets attended to given item, and returns true if attended changed.
// If item is nil, then nothing has attended.
// sendEvent determines whether the events.Attend event is sent to the focused item.
func (em *Events) setAttendImpl(w Widget, sendEvent bool) bool {
	catd := em.attended
	if tree.IsNil(catd) {
		em.attended = nil
		catd = nil
	}
	if catd != nil && !tree.IsNil(w) && catd == w {
		if DebugSettings.FocusTrace {
			fmt.Println(em.scene, "Already Attend:", catd)
		}
		return false
	}
	if catd != nil {
		if DebugSettings.FocusTrace {
			fmt.Println(em.scene, "Losing attend:", catd)
		}
		catd.AsWidget().Send(events.AttendLost)
	}
	em.attended = w
	if sendEvent && !tree.IsNil(w) {
		w.AsWidget().Send(events.Attend)
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
	case keymap.Find:
		Search(sc, LastSearch, LastUseCase)
		e.SetHandled()
	case keymap.ContextMenu:
		// at the widget: the key event has no position (see [Widget.ContextMenuPos])
		if !tree.IsNil(em.focus) {
			em.focus.AsWidget().Send(events.ContextMenu, e)
			e.SetHandled()
		}
	case keymap.WinSnapshot:
		img := sc.Renderer.Image()
		scnm := strings.ReplaceAll(strings.ReplaceAll(strcase.ToKebab(sc.Name), "/", "-"), "•", "-")
		dstr := time.Now().Format(time.DateOnly + "-" + "15-04-05")
		bnm := filepath.Join(TheApp.AppDataDir(), "screenshot-"+scnm+"-"+dstr)
		var sz string
		if img != nil {
			sz = fmt.Sprint(img.Bounds().Size())
			fnm := bnm + ".png"
			if errors.Log(imagex.Save(img, fnm)) == nil {
				MessageSnackbar(sc, "Saved screenshot to: "+strings.ReplaceAll(fnm, " ", `\ `)+sz)
			}
		} else {
			MessageSnackbar(sc, "Save screenshot: no render image")
		}
		sc.RenderWidget()
		rend := sc.Painter.RenderDone()
		svr := paint.NewSVGRenderer(sc.Painter.Size)
		sv := svr.Render(rend).Source()
		fnm := bnm + ".svg"
		errors.Log(os.WriteFile(fnm, sv, 0666))
		pdr := paint.NewPDFRenderer(sc.Painter.Size, &sc.Painter.Context().Style.UnitContext)
		pd := pdr.Render(rend).Source()
		fnm = bnm + ".pdf"
		errors.Log(os.WriteFile(fnm, pd, 0666))

		sc.SetScene(sc)
		sc.Update()

		MessageSnackbar(sc, "Saved SVG, PDF screenshots to: "+strings.ReplaceAll(fnm, " ", `\ `)+sz)
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
	case keymap.WinFocusNext:
		e.SetHandled()
		AllRenderWindows.focusNext()
	}
	if !e.IsHandled() {
		em.triggerShortcut(cs, e)
	}
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
			bt.Menu(tmps, image.Point{})
			em.getShortcutsIn(tmps)
		}
		return tree.Break // there are no buttons in buttons
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

// addShortcut adds the given shortcut for the given button.
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
func (em *Events) triggerShortcut(chord key.Chord, e events.Event) bool {
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
	if tree.IsNil(sa) {
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
	e.SetHandled()
	prev := em.focus
	sa.Send(events.Click)
	if em.focus == sa.This && !tree.IsNil(prev) {
		// the click focused the button, but the shortcut is not
		// for moving the focus to it, as clicking it is
		em.setFocusQuiet(prev)
	}
	return true
}

func (em *Events) getSpriteInBBox(sc *Scene, pos image.Point) {
	st := sc.Stage
	st.Sprites.Do(func(sl *SpriteList) {
		for _, sp := range sl.Values {
			if sp == nil || !sp.Active {
				continue
			}
			if sp.listeners == nil {
				continue
			}
			if pos.In(sp.EventBBox) {
				em.spriteInBBox = append(em.spriteInBBox, sp)
			}
		}
	})
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
	em.spriteHovers = em.updateSpriteHovers(em.spriteInBBox, em.spriteHovers, e, events.MouseEnter, events.MouseLeave)
	return e.IsHandled()
}
