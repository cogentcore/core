// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log/slog"

	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
)

func (wb *WidgetBase) EventMgr() *EventMgr {
	if wb.Sc == nil {
		return nil
	}
	return &wb.Sc.EventMgr
}

// On adds an event listener function for the given event type
func (wb *WidgetBase) On(etype events.Types, fun func(e events.Event)) Widget {
	wb.Listeners.Add(etype, fun)
	return wb.This().(Widget)
}

// Helper functions for common event types

// OnClick adds an event listener function for [events.Click] events
func (wb *WidgetBase) OnClick(fun func(e events.Event)) Widget {
	return wb.On(events.Click, fun)
}

// OnDoubleClick adds an event listener function for [events.DoubleClick] events
func (wb *WidgetBase) OnDoubleClick(fun func(e events.Event)) Widget {
	return wb.On(events.DoubleClick, fun)
}

// OnChange adds an event listener function for [events.Change] events
func (wb *WidgetBase) OnChange(fun func(e events.Event)) Widget {
	return wb.On(events.Change, fun)
}

// OnKeyChord adds an event listener function for [events.KeyChord] events
func (wb *WidgetBase) OnKeyChord(fun func(e events.Event)) Widget {
	return wb.On(events.KeyChord, fun)
}

// OnFocus adds an event listener function for [events.Focus] events
func (wb *WidgetBase) OnFocus(fun func(e events.Event)) Widget {
	return wb.On(events.Focus, fun)
}

// OnFocusLost adds an event listener function for [events.FocusLost] events
func (wb *WidgetBase) OnFocusLost(fun func(e events.Event)) Widget {
	return wb.On(events.FocusLost, fun)
}

// OnSelect adds an event listener function for [events.Select] events
func (wb *WidgetBase) OnSelect(fun func(e events.Event)) Widget {
	return wb.On(events.Select, fun)
}

// Send sends an NEW event of given type to this widget,
// optionally starting from values in the given original event
// (recommended to include where possible).
// Do NOT send an existing event using this method if you
// want the Handled state to persist throughout the call chain;
// call HandleEvent directly for any existing events.
func (wb *WidgetBase) Send(typ events.Types, orig ...events.Event) {
	var e events.Event
	if len(orig) > 0 && orig[0] != nil {
		e = orig[0].Clone()
		e.AsBase().Typ = typ
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

// SendChange sends the events.Change event, which is widely used to signal
// updating for most widgets. It takes the event that the new change event
// is derived from, if any.
func (wb *WidgetBase) SendChange(orig ...events.Event) {
	wb.Send(events.Change, orig...)
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
	state := wb.Styles.State
	wb.Listeners.Call(ev)
	if wb.This() == nil || wb.Is(ki.Deleted) || wb.Is(ki.Destroyed) {
		return
	}
	if wb.Styles.State != state {
		wb.ApplyStyleUpdate(wb.Sc)
	}
}

// HandleWidgetEvents adds the default events for Widget objects.
func (wb *WidgetBase) HandleWidgetEvents() {
	wb.HandleWidgetClick()
	wb.HandleWidgetStateFromMouse()
	wb.HandleLongHoverTooltip()
	wb.HandleWidgetStateFromFocus()
	wb.HandleWidgetContextMenu()
}

// PosInScBBox returns true if given position is within
// this node's scene bbox (under read lock)
func (wb *WidgetBase) PosInScBBox(pos image.Point) bool {
	wb.BBoxMu.RLock()
	defer wb.BBoxMu.RUnlock()
	return pos.In(wb.ScBBox)
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
			wb.GrabFocus()
		} else {
			wb.FocusClear()
		}
		if wb.AbilityIs(abilities.Selectable) {
			if wb.StateIs(states.Selected) {
				wb.SetState(false, states.Selected)
				wb.Send(events.Deselect, e)
			} else {
				wb.SetState(true, states.Selected)
				wb.Send(events.Select, e)
			}
		}
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

// HandleLongHoverTooltip listens for LongHoverEvent and pops up a tooltip.
// Most widgets should call this as part of their event handler methods.
func (wb *WidgetBase) HandleLongHoverTooltip() {
	wb.On(events.LongHoverStart, func(e events.Event) {
		if wb.Tooltip == "" {
			return
		}
		e.SetHandled()
		NewTooltip(wb, e.Pos()).Run()
	})
	wb.On(events.LongHoverEnd, func(e events.Event) {
		if wb.Sc != nil && wb.Sc.MainStageMgr() != nil {
			top := wb.Sc.MainStageMgr().Top()
			if top != nil && top.AsMain() != nil {
				top.AsMain().PopupMgr.PopDeleteType(TooltipStage)
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

// HandleClickOnEnterSpace adds key event handler for Enter or Space
// to generate a Click action
func (wb *WidgetBase) HandleClickOnEnterSpace() {
	wb.OnKeyChord(func(e events.Event) {
		if KeyEventTrace {
			slog.Info("WidgetBase KeyChordEvent", "widget", wb)
		}
		kf := KeyFun(e.KeyChord())
		if kf == KeyFunEnter || e.KeyRune() == ' ' {
			// TODO: do we need this?
			// if !(kt.Rune == ' ' && bbb.Sc.Type == ScCompleter) {
			e.SetHandled()
			wb.Send(events.Click, e)
			// }
		}
	})
}

///////////////////////////////////////////////////////////////////
//		Focus

// GrabFocus grabs the keyboard input focus on this item or the first item within it
// that can be focused (if none, then goes ahead and sets focus to this object)
func (wb *WidgetBase) GrabFocus() {
	foc := wb.This().(Widget)
	if !foc.AbilityIs(abilities.Focusable) {
		foc = wb.FocusableInMe()
	}
	em := wb.EventMgr()
	if em != nil {
		// fmt.Println("grab focus:", foc)
		em.GrabFocus(foc) // doesn't send event
	}
}

// FocusableInMe returns the first Focusable element within this widget
func (wb *WidgetBase) FocusableInMe() Widget {
	var foc Widget
	wb.WalkPre(func(k ki.Ki) bool {
		kwi, kwb := AsWidget(k)
		if kwb == nil || kwb.This() == nil || kwb.Is(ki.Deleted) || kwb.Is(ki.Destroyed) {
			return ki.Break
		}
		if !kwb.AbilityIs(abilities.Focusable) {
			return ki.Continue
		}
		foc = kwi
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
	if cur == wb.This() {
		return true
	}
	plev := cur.ParentLevel(wb.This())
	return plev >= 0
}
