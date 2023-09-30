// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/girl/states"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
)

func (wb *WidgetBase) EventMgr() *EventMgr {
	return &wb.Sc.EventMgr
}

// On adds an event listener function for the given event type
func (wb *WidgetBase) On(etype events.Types, fun func(e events.Event)) Widget {
	wb.Listeners.Add(etype, fun)
	return wb.This().(Widget)
}

// SendMe sends an event of given type to this widget,
// optionally starting from values in the given original event
// (recommended to include where possible).
func (wb *WidgetBase) SendMe(typ events.Types, orig events.Event) {
	var e events.Event
	if orig != nil {
		e = orig.Clone()
		e.AsBase().Typ = typ
	} else {
		e = &events.Base{Typ: typ}
	}
	wb.This().(Widget).HandleEvent(e)
}

// PosInBBox returns true if given position is within
// this node's win bbox (under read lock)
func (wb *WidgetBase) PosInBBox(pos image.Point) bool {
	wb.BBoxMu.RLock()
	defer wb.BBoxMu.RUnlock()
	return pos.In(wb.ScBBox)
}

func (wb *WidgetBase) HandleEvent(ev events.Event) {
	wb.Listeners.Call(ev)
}

// WidgetHandlers adds the default events for Widget objects.
func (wb *WidgetBase) WidgetHandlers() {
	// nb.WidgetMouseEvent() ??
	// wb.WidgetMouseFocusEvent()
	wb.LongHoverTooltip()
}

// LongHoverTooltip listens for LongHoverEvent and pops up a tooltip.
// Most widgets should call this as part of their event handler methods.
func (wb *WidgetBase) LongHoverTooltip() {
	wb.On(events.LongHoverStart, func(e events.Event) {
		if wb.StateIs(states.Disabled) {
			return
		}
		if wb.Tooltip == "" {
			return
		}
		e.SetHandled()
		// pos := wbb.WinBBox.Max
		// pos.X -= 20
		// mvp := wbb.Sc
		// PopupTooltip(wbb.Tooltip, pos.X, pos.Y, mvp, wbb.Nm)
	})
}

/*
// WidgetMouseFocusEvent does the default handling for mouse click events for the Widget
func (wb *WidgetBase) WidgetMouseEvent() {
	we.AddFunc(events.MouseUp, RegPri, func(recv, send ki.Ki, sig int64, data any) {
		wbb := AsWidgetBase(recv)
		if wbb.IsDisabled() {
			return
		}
		me := data.(events.Event)
		me.SetHandled()
		wbb.WidgetOnMouseEvent(me)
	})
}

// WidgetOnMouseEvent is the function called on Widget objects
// when they get a mouse click event. If you are declaring a custom
// mouse event function, you should call this function first.
func (wb *WidgetBase) WidgetOnMouseEvent(me events.Event) {
	// wb.SetFlag(me.Action == events.Press, Active)
	wb.ApplyStyleUpdate(wb.Sc)
}

// WidgetMouseFocusEvent does the default handling for mouse focus events for the Widget
func (wb *WidgetBase) WidgetMouseFocusEvent() {
	we.AddFunc(events.MouseEnter, RegPri, func(recv, send ki.Ki, sig int64, data any) {
		wbb := AsWidgetBase(recv)
		if wbb.IsDisabled() {
			return
		}
		me := data.(events.Event)
		me.SetHandled()
		wbb.WidgetOnMouseFocusEvent(me)
	})
}

// WidgetOnMouseFocusEvent is the function called on Widget objects
// when they get a mouse foucs event. If you are declaring a custom
// mouse foucs event function, you should call this function first.
func (wb *WidgetBase) WidgetOnMouseFocusEvent(me events.Event) {
	enter := me.Action == events.Enter
	wb.SetFlag(enter, Hovered)
	wb.ApplyStyleUpdate(wb.Sc)
	// TODO: trigger mouse focus exit after clicking down
	// while leaving; then clear active here
	// // if !enter {
	// // 	nb.ClearActive()
	// }
}

*/

// WidgetMouseEvents connects to either or both mouse events -- IMPORTANT: if
// you need to also connect to other mouse events, you must copy this code --
// all processing of a mouse event must happen within one function b/c there
// can only be one registered per receiver and event type.  sel = Left button
// events.Press event, toggles the selected state, and emits a SelectedEvent.
// ctxtMenu = connects to Right button events.Press event, and sends a
// WidgetSig WidgetContextMenu signal, followed by calling ContextMenu method
// -- signal can be used to change state prior to generating context menu,
// including setting a CtxtMenuFunc that removes all items and thus negates
// the presentation of any menu
/*
func (wb *WidgetBase) WidgetMouseEvents(sel, ctxtMenu bool) {
	if !sel && !ctxtMenu {
		return
	}
	wbwe.AddFunc(events.MouseUp, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(events.Event)
		if sel {
			if me.Action == events.Press && me.Button == events.Left {
				me.SetHandled()
				_, wbb := AsWidget(recv)
				wbb.SetSelected(!wbb.StateIs(states.Selected))
				wbb.EmitSelectedSignal()
				wbb.ApplyStyleUpdate(wbb.Sc)
			}
		}
		if ctxtMenu {
			if me.Action == events.Release && me.Button == events.Right {
				me.SetHandled()
				wi, wbb := AsWidget(recv)
				wbb.EmitContextMenuSignal()
				wi.ContextMenu()
			}
		}
	})
}
*/

// WidgetSignals are general signals that all widgets can send, via WidgetSig
// signal
type WidgetSignals int64

const (
	// WidgetSelected is triggered when a widget is selected, typically via
	// left mouse button click (see EmitSelectedSignal) -- is NOT contingent
	// on actual IsSelected status -- just reports the click event.
	// The data is the index of the selected item for multi-item widgets
	// (-1 = none / unselected)
	WidgetSelected WidgetSignals = iota

	// WidgetFocused is triggered when a widget receives keyboard focus (see
	// EmitFocusedSignal -- call in FocusChanged for gotFocus
	WidgetFocused

	// WidgetContextMenu is triggered when a widget receives a
	// right-mouse-button press, BEFORE generating and displaying the context
	// menu, so that relevant state can be updated etc (see
	// EmitContextMenuSignal)
	WidgetContextMenu

	WidgetSignalsN
)

/*
// EmitSelectedSignal emits the WidgetSelected signal for this widget
func (wb *WidgetBase) EmitSelectedSignal() {
	wb.WidgetSig.Emit(wb.This(), int64(WidgetSelected), nil)
}

// EmitFocusedSignal emits the WidgetFocused signal for this widget
func (wb *WidgetBase) EmitFocusedSignal() {
	wb.WidgetSig.Emit(wb.This(), int64(WidgetFocused), nil)
}

// EmitContextMenuSignal emits the WidgetContextMenu signal for this widget
func (wb *WidgetBase) EmitContextMenuSignal() {
	wb.WidgetSig.Emit(wb.This(), int64(WidgetContextMenu), nil)
}
*/

// FirstContainingPoint finds the first node whose WinBBox contains the given
// point -- nil if none.  If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered
func (wb *WidgetBase) FirstContainingPoint(pt image.Point, leavesOnly bool) ki.Ki {
	var rval ki.Ki
	wb.WalkPre(func(k ki.Ki) bool {
		if k == wb.This() {
			return ki.Continue
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
		}
		_, w := AsWidget(k)
		if w == nil || w.Is(ki.Deleted) || w.Is(ki.Destroyed) {
			// 3D?
			return ki.Break
		}
		if w.PosInBBox(pt) {
			rval = w.This()
			return ki.Break
		}
		return ki.Continue
	})
	return rval
}
