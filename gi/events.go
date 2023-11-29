// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log/slog"

	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/ki/v2"
)

func (wb *WidgetBase) EventMgr() *EventMgr {
	if wb.Sc == nil {
		return nil
	}
	return &wb.Sc.EventMgr
}

// On adds an event listener function for the given event type,
// to the end of the current stack, so that it will be called
// before everything else already on the stack.
func (wb *WidgetBase) On(etype events.Types, fun func(e events.Event)) *WidgetBase {
	wb.Listeners.Add(etype, func(e events.Event) {
		if wb.This() == nil || wb.Is(ki.Deleted) {
			return
		}
		fun(e)
	})
	return wb
}

// OnLast adds an event listener function for the given event type,
// to the start of the current stack, so that it will be called
// after everything else already on the stack.
func (wb *WidgetBase) OnLast(etype events.Types, fun func(e events.Event)) Widget {
	wb.Listeners.AddLastCall(etype, func(e events.Event) {
		if wb.This() == nil || wb.Is(ki.Deleted) {
			return
		}
		fun(e)
	})
	return wb.This().(Widget)
}

// Helper functions for common event types

// OnClick adds an event listener function for [events.Click] events
func (wb *WidgetBase) OnClick(fun func(e events.Event)) *WidgetBase {
	return wb.On(events.Click, fun)
}

// OnDoubleClick adds an event listener function for [events.DoubleClick] events
func (wb *WidgetBase) OnDoubleClick(fun func(e events.Event)) *WidgetBase {
	return wb.On(events.DoubleClick, fun)
}

// OnChange adds an event listener function for [events.Change] events
func (wb *WidgetBase) OnChange(fun func(e events.Event)) *WidgetBase {
	return wb.On(events.Change, fun)
}

// OnInput adds an event listener function for [events.Input] events
func (wb *WidgetBase) OnInput(fun func(e events.Event)) *WidgetBase {
	return wb.On(events.Input, fun)
}

// OnKeyChord adds an event listener function for [events.KeyChord] events
func (wb *WidgetBase) OnKeyChord(fun func(e events.Event)) *WidgetBase {
	return wb.On(events.KeyChord, fun)
}

// OnFocus adds an event listener function for [events.Focus] events
func (wb *WidgetBase) OnFocus(fun func(e events.Event)) *WidgetBase {
	return wb.On(events.Focus, fun)
}

// OnFocusLost adds an event listener function for [events.FocusLost] events
func (wb *WidgetBase) OnFocusLost(fun func(e events.Event)) *WidgetBase {
	return wb.On(events.FocusLost, fun)
}

// OnSelect adds an event listener function for [events.Select] events
func (wb *WidgetBase) OnSelect(fun func(e events.Event)) *WidgetBase {
	return wb.On(events.Select, fun)
}

// OnShow adds an event listener function for [events.Show] events on
// the widget's Scene. Directly listening to Show events for non-scene
// widgets does not work, so it must go through the Scene.
func (wb *WidgetBase) OnShow(fun func(e events.Event)) *WidgetBase {
	return wb.Sc.On(events.Show, fun)
}

// OnClose adds an event listener function for [events.Close] events on
// the widget's Scene. Directly listening to Close events for non-scene
// widgets does not work, so it must go through the Scene.
func (wb *WidgetBase) OnClose(fun func(e events.Event)) *WidgetBase {
	return wb.Sc.On(events.Close, fun)
}

// Send sends an NEW event of given type to this widget,
// optionally starting from values in the given original event
// (recommended to include where possible).
// Do NOT send an existing event using this method if you
// want the Handled state to persist throughout the call chain;
// call HandleEvent directly for any existing events.
func (wb *WidgetBase) Send(typ events.Types, orig ...events.Event) {
	if wb.This() == nil || wb.Is(ki.Deleted) {
		return
	}
	var e events.Event
	if len(orig) > 0 && orig[0] != nil {
		e = orig[0].Clone()
		e.AsBase().Typ = typ
		e.AsBase().ClearHandled()
	} else {
		e = &events.Base{Typ: typ}
		e.Init()
	}
	w, ok := wb.This().(Widget)
	if !ok {
		return
	}
	w.HandleEvent(e)
}

// SendChange sends the [events.Change] event, which is widely used to signal
// updating for most widgets. It takes the event that the new change event
// is derived from, if any.
func (wb *WidgetBase) SendChange(orig ...events.Event) {
	wb.Send(events.Change, orig...)
}

func (wb *WidgetBase) SendKeyFun(kf keyfun.Funs, orig ...events.Event) {
	if wb.This() == nil || wb.Is(ki.Deleted) {
		return
	}
	kc := keyfun.ChordFor(kf)
	wb.SendKeyChord(kc, orig...)
}

func (wb *WidgetBase) SendKeyChord(kc key.Chord, orig ...events.Event) {
	r, code, mods, err := kc.Decode()
	if err != nil {
		fmt.Println("SendKeyChord: Decode error:", err)
		return
	}
	wb.SendKeyChordRune(r, code, mods, orig...)
}

func (wb *WidgetBase) SendKeyChordRune(r rune, code key.Codes, mods key.Modifiers, orig ...events.Event) {
	ke := events.NewKey(events.KeyChord, r, code, mods)
	if len(orig) > 0 && orig[0] != nil {
		kb := *orig[0].AsBase()
		ke.GenTime = kb.GenTime
		ke.ClearHandled()
	} else {
		ke.Init()
	}
	ke.Typ = events.KeyChord
	w, ok := wb.This().(Widget)
	if !ok {
		return
	}
	w.HandleEvent(ke)
}

// AddPriorityEvent adds given event type to the set of priority events for this scene
func (wb *WidgetBase) AddPriorityEvent(etype events.Types) {
	wb.PriorityEvents = append(wb.PriorityEvents, etype)
}

// HandleEvent sends the given event to all Listeners for that event type.
// It also checks if the State has changed and calls ApplyStyle if so.
// If more significant Config level changes are needed due to an event,
// the event handler must do this itself.
func (wb *WidgetBase) HandleEvent(ev events.Event) {
	if EventTrace {
		if ev.Type() != events.MouseMove {
			fmt.Println("Event to Widget:", wb.Path(), ev.String())
		}
	}
	if wb == nil || wb.This() == nil || wb.Is(ki.Deleted) {
		return
	}
	s := &wb.Styles
	state := s.State
	wb.Listeners.Call(ev)
	// widget can be killed after event
	if wb.This() == nil || wb.Is(ki.Deleted) {
		return
	}
	if s.State != state {
		wb.ApplyStyleUpdate()
		// wb.Transition(&s.StateLayer, s.State.StateLayer(), 500*time.Millisecond, LinearTransition)
	}
}

// HandleWidgetEvents adds the default events for Widget objects.
func (wb *WidgetBase) HandleWidgetEvents() {
	wb.HandleWidgetClick()
	wb.HandleWidgetStateFromMouse()
	wb.HandleLongHoverTooltip()
	wb.HandleWidgetStateFromFocus()
	wb.HandleWidgetContextMenu()
	wb.HandleWidgetMagnify()
}

// PosInScBBox returns true if given position is within
// this node's scene bbox (under read lock)
func (wb *WidgetBase) PosInScBBox(pos image.Point) bool {
	wb.BBoxMu.RLock()
	defer wb.BBoxMu.RUnlock()
	return pos.In(wb.Geom.TotalBBox)
}

// HandleWidgetClick handles the Click event for basic Widget behavior.
// For Left button:
// If Checkable, toggles Checked. if Focusable, Focuses or clears,
// If Selectable, updates state and sends Select, Deselect.
// For Right button:
// Sends ContextMenu event that Activates a context menu if present.
func (wb *WidgetBase) HandleWidgetClick() {
	wb.OnClick(func(e events.Event) {
		// fmt.Println("click", wb)
		if wb.AbilityIs(abilities.Checkable) {
			wb.SetState(!wb.StateIs(states.Checked), states.Checked)
		}
		if wb.AbilityIs(abilities.Focusable) {
			wb.SetFocus()
		} else {
			wb.FocusClear()
		}
		// note: ReadOnly items are automatically selectable, for choosers
		if wb.AbilityIs(abilities.Selectable) || wb.IsReadOnly() {
			wb.Send(events.Select, e)
		}
	})
}

// HandleSelectToggle does basic selection handling logic on widget,
// as just a toggle on individual selection state, including ensuring
// consistent selection flagging for parts.
// This is not called by WidgetBase but should be called for simple
// Widget types.  More complex container / View widgets likely implement
// their own more complex selection logic.
func (wb *WidgetBase) HandleSelectToggle() {
	wb.OnSelect(func(e events.Event) {
		wb.SetStateWidget(!wb.StateIs(states.Selected), states.Selected)
	})
}

// HandleWidgetStateFromMouse updates all standard
// State flags based on mouse events,
// such as MouseDown / Up -> Active and MouseEnter / Leave -> Hovered.
// None of these "consume" the event by setting Handled flag, as they are
// designed to work in conjunction with more specific handlers.
// Note that Disabled and Invisible widgets do NOT receive
// these events so it is not necessary to check that.
func (wb *WidgetBase) HandleWidgetStateFromMouse() {
	wb.On(events.MouseDown, func(e events.Event) {
		if wb.AbilityIs(abilities.Activatable) {
			// fmt.Println("active:", wb)
			wb.SetState(true, states.Active)
		}
	})
	wb.On(events.MouseUp, func(e events.Event) {
		if wb.AbilityIs(abilities.Activatable) {
			wb.SetState(false, states.Active)
		}
	})
	wb.On(events.DoubleClick, func(e events.Event) {
		// if we are not double clickable, we just treat
		// it as a click event (as long as we are pressable)
		if !wb.AbilityIs(abilities.DoubleClickable) && wb.Styles.Abilities.IsPressable() {
			wb.Send(events.Click, e)
		}
	})
	wb.On(events.LongPressStart, func(e events.Event) {
		if wb.AbilityIs(abilities.LongPressable) {
			wb.SetState(true, states.LongPressed)
		}
	})
	wb.On(events.LongPressEnd, func(e events.Event) {
		if wb.AbilityIs(abilities.LongPressable) {
			wb.SetState(false, states.LongPressed)
		}
	})
	wb.On(events.MouseEnter, func(e events.Event) {
		if wb.AbilityIs(abilities.Hoverable) {
			wb.SetState(true, states.Hovered)
		}
	})
	wb.On(events.MouseLeave, func(e events.Event) {
		if wb.AbilityIs(abilities.Hoverable) {
			wb.SetState(false, states.Hovered)
		}
	})
	wb.On(events.LongHoverStart, func(e events.Event) {
		if wb.AbilityIs(abilities.LongHoverable) {
			wb.SetState(true, states.LongHovered)
		}
	})
	wb.On(events.LongHoverEnd, func(e events.Event) {
		if wb.AbilityIs(abilities.LongHoverable) {
			wb.SetState(false, states.LongHovered)
		}
	})
	wb.On(events.SlideStart, func(e events.Event) {
		if wb.AbilityIs(abilities.Slideable) {
			// fmt.Println("sliding:", wb)
			wb.SetState(true, states.Sliding)
		}
	})
	wb.On(events.SlideStop, func(e events.Event) {
		if wb.AbilityIs(abilities.Slideable) {
			wb.SetState(false, states.Sliding, states.Active)
			// fmt.Println("done sliding:", wb)
		}
	})
	wb.On(events.DragStart, func(e events.Event) {
		if wb.AbilityIs(abilities.Draggable) {
			wb.SetState(true, states.Dragging)
		}
	})
	wb.On(events.Drop, func(e events.Event) {
		if wb.AbilityIs(abilities.Draggable) {
			wb.SetState(false, states.Dragging, states.Active)
		}
	})
}

// HandleLongHoverTooltip listens for LongHover and LongPress events and
// pops up and deletes tooltips based on those. Most widgets should call
// this as part of their event handler methods.
func (wb *WidgetBase) HandleLongHoverTooltip() {
	wb.On(events.LongHoverStart, func(e events.Event) {
		if wb.Tooltip == "" {
			return
		}
		e.SetHandled()
		NewTooltip(wb, wb.WinBBox().Min).Run()
	})
	wb.On(events.LongHoverEnd, func(e events.Event) {
		if wb.Sc != nil && wb.Sc.MainStageMgr() != nil {
			top := wb.Sc.MainStageMgr().Top()
			if top != nil {
				top.PopupMgr.PopDeleteType(TooltipStage)
			}
		}
	})

	wb.On(events.LongPressStart, func(e events.Event) {
		if wb.Tooltip == "" {
			return
		}
		e.SetHandled()
		NewTooltip(wb, wb.WinBBox().Min).Run()
	})
	wb.On(events.LongPressEnd, func(e events.Event) {
		if wb.Sc != nil && wb.Sc.MainStageMgr() != nil {
			top := wb.Sc.MainStageMgr().Top()
			if top != nil {
				top.PopupMgr.PopDeleteType(TooltipStage)
			}
		}
	})
}

// HandleWidgetStateFromFocus updates standard State flags based on Focus events
func (wb *WidgetBase) HandleWidgetStateFromFocus() {
	wb.OnFocus(func(e events.Event) {
		if wb.AbilityIs(abilities.Focusable) {
			wb.ScrollToMe()
			wb.SetState(true, states.Focused)
		}
	})
	wb.OnFocusLost(func(e events.Event) {
		if wb.AbilityIs(abilities.Focusable) {
			wb.SetState(false, states.Focused)
		}
	})
}

// HandleWidgetMagnifyEvent updates [Prefs.Zoom] on [events.Magnify]
func (wb *WidgetBase) HandleWidgetMagnify() {
	wb.On(events.Magnify, func(e events.Event) {
		ev := e.(*events.TouchMagnify)
		fmt.Println("magnify event", ev.ScaleFactor)
		Prefs.Zoom *= ev.ScaleFactor
		Prefs.UpdateAll()
	})
}

// HandleClickOnEnterSpace adds key event handler for Enter or Space
// to generate a Click action
func (wb *WidgetBase) HandleClickOnEnterSpace() {
	wb.OnKeyChord(func(e events.Event) {
		if KeyEventTrace {
			slog.Info("WidgetBase KeyChordEvent", "widget", wb)
		}
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Enter || e.KeyRune() == ' ' {
			e.SetHandled()
			wb.Send(events.Click, e)
		}
	})
}

///////////////////////////////////////////////////////////////////
//		Focus

// SetFocus sets the keyboard input focus on this item or the first item within it
// that can be focused (if none, then goes ahead and sets focus to this object).
// This does NOT send an [events.Focus] event, which typically results in
// the widget being styled as focused.  See [SetFocusEvent] for one that does.
func (wb *WidgetBase) SetFocus() {
	foc := wb.This().(Widget)
	if !foc.AbilityIs(abilities.Focusable) {
		foc = wb.FocusableInMe()
		if foc == nil {
			foc = wb.This().(Widget)
		}
	}
	em := wb.EventMgr()
	if em != nil {
		// fmt.Println("grab focus:", foc)
		em.SetFocus(foc) // doesn't send event
	}
}

// SetFocusEvent sets the keyboard input focus on this item or the first item within it
// that can be focused (if none, then goes ahead and sets focus to this object).
// This sends an [events.Focus] event, which typically results in
// the widget being styled as focused.  See [SetFocus] for one that does not.
func (wb *WidgetBase) SetFocusEvent() {
	foc := wb.This().(Widget)
	if !foc.AbilityIs(abilities.Focusable) {
		foc = wb.FocusableInMe()
		if foc == nil {
			foc = wb.This().(Widget)
		}
	}
	em := wb.EventMgr()
	if em != nil {
		// fmt.Println("grab focus:", foc)
		em.SetFocusEvent(foc) // doesn't send event
	}
}

// FocusableInMe returns the first Focusable element within this widget
func (wb *WidgetBase) FocusableInMe() Widget {
	var foc Widget
	wb.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		if !wb.AbilityIs(abilities.Focusable) {
			return ki.Continue
		}
		foc = wi
		return ki.Break // done
	})
	return foc
}

// FocusNext moves the focus onto the next item
func (wb *WidgetBase) FocusNext() {
	em := wb.EventMgr()
	if em != nil {
		em.FocusNext()
	}
}

// FocusPrev moves the focus onto the previous item
func (wb *WidgetBase) FocusPrev() {
	em := wb.EventMgr()
	if em != nil {
		em.FocusPrev()
	}
}

// FocusClear resets focus to nil, but keeps the previous focus to pick up next time..
func (wb *WidgetBase) FocusClear() {
	em := wb.EventMgr()
	if em != nil {
		em.FocusClear()
	}
}

// StartFocus specifies this widget to give focus to when the window opens
func (wb *WidgetBase) StartFocus() {
	em := wb.EventMgr()
	if em != nil {
		em.SetStartFocus(wb.This().(Widget))
	}
}

// ContainsFocus returns true if this widget contains the current focus widget
// as maintained in the RenderWin
func (wb *WidgetBase) ContainsFocus() bool {
	em := wb.EventMgr()
	if em == nil {
		return false
	}
	cur := em.Focus
	if cur == nil {
		return false
	}
	if cur.This() == wb.This() {
		return true
	}
	plev := cur.ParentLevel(wb.This())
	// fmt.Println("plev:", plev)
	return plev >= 0
}
