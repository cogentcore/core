// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/goosi"
	"goki.dev/goosi/mouse"
	"goki.dev/ki/v2"
)

// ConnectEvents is the default event connection function
// for Widget objects. It calls [WidgetEvents], so any Widget
// implementing a custom ConnectEvents function should
// first call [WidgetEvents].
func (wb *WidgetBase) ConnectEvents() {
	wb.WidgetEvents()
}

// WidgetEvents connects the default events for Widget objects.
// Any Widget implementing a custom ConnectEvents function
// should first call this function.
func (wb *WidgetBase) WidgetEvents() {
	// TODO: figure out connect events situation not working
	// nb.WidgetMouseEvent()
	wb.WidgetMouseFocusEvent()
	wb.HoverTooltipEvent()
}

// WidgetMouseFocusEvent does the default handling for mouse click events for the Widget
func (wb *WidgetBase) WidgetMouseEvent() {
	wb.ConnectEvent(goosi.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, data any) {
		if wb.IsDisabled() {
			return
		}

		me := data.(*mouse.Event)
		me.SetProcessed()

		wb.WidgetOnMouseEvent(me)
	})
}

// WidgetOnMouseEvent is the function called on Widget objects
// when they get a mouse click event. If you are declaring a custom
// mouse event function, you should call this function first.
func (wb *WidgetBase) WidgetOnMouseEvent(me *mouse.Event) {
	wb.SetActiveState(me.Action == mouse.Press)
	wb.SetStyleUpdate(wb.Vp)
}

// WidgetMouseFocusEvent does the default handling for mouse focus events for the Widget
func (wb *WidgetBase) WidgetMouseFocusEvent() {
	wb.ConnectEvent(goosi.MouseFocusEvent, RegPri, func(recv, send ki.Ki, sig int64, data any) {
		if wb.IsDisabled() {
			return
		}

		me := data.(*mouse.FocusEvent)
		me.SetProcessed()

		wb.WidgetOnMouseFocusEvent(me)
	})
}

// WidgetOnMouseFocusEvent is the function called on Widget objects
// when they get a mouse foucs event. If you are declaring a custom
// mouse foucs event function, you should call this function first.
func (wb *WidgetBase) WidgetOnMouseFocusEvent(me *mouse.FocusEvent) {
	enter := me.Action == mouse.Enter
	wb.SetHoveredState(enter)
	wb.SetStyleUpdate(wb.Vp)
	// TODO: trigger mouse focus exit after clicking down
	// while leaving; then clear active here
	// // if !enter {
	// // 	nb.ClearActive()
	// }
}

// WidgetMouseEvents connects to either or both mouse events -- IMPORTANT: if
// you need to also connect to other mouse events, you must copy this code --
// all processing of a mouse event must happen within one function b/c there
// can only be one registered per receiver and event type.  sel = Left button
// mouse.Press event, toggles the selected state, and emits a SelectedEvent.
// ctxtMenu = connects to Right button mouse.Press event, and sends a
// WidgetSig WidgetContextMenu signal, followed by calling ContextMenu method
// -- signal can be used to change state prior to generating context menu,
// including setting a CtxtMenuFunc that removes all items and thus negates
// the presentation of any menu
func (wb *WidgetBase) WidgetMouseEvents(sel, ctxtMenu bool) {
	if !sel && !ctxtMenu {
		return
	}
	wb.ConnectEvent(goosi.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		if sel {
			if me.Action == mouse.Press && me.Button == mouse.Left {
				me.SetProcessed()
				wbb := recv.Embed(TypeWidgetBase).(*WidgetBase)
				wbb.SetSelected(!wbb.IsSelected())
				wbb.EmitSelectedSignal()
				wbb.UpdateSig()
			}
		}
		if ctxtMenu {
			if me.Action == mouse.Release && me.Button == mouse.Right {
				me.SetProcessed()
				wbb := recv.Embed(TypeWidgetBase).(*WidgetBase)
				wbb.EmitContextMenuSignal()
				wbb.This().(Widget).ContextMenu()
			}
		}
	})
}

// ConnectEvent connects this node to receive a given type of GUI event
// signal from the parent window -- typically connect only visible nodes, and
// disconnect when not visible
func (wb *WidgetBase) ConnectEvent(et goosi.EventType, pri EventPris, fun ki.RecvFunc) {
	em := wb.EventMgr()
	if em != nil {
		em.ConnectEvent(wb.This(), et, pri, fun)
	}
}

// DisconnectEvent disconnects this receiver from receiving given event
// type -- pri is priority -- pass AllPris for all priorities -- see also
// DisconnectAllEvents
func (wb *WidgetBase) DisconnectEvent(et goosi.EventType, pri EventPris) {
	em := wb.EventMgr()
	if em != nil {
		em.DisconnectEvent(wb.This(), et, pri)
	}
}

// DisconnectAllEvents disconnects node from all window events -- typically
// disconnect when not visible -- pri is priority -- pass AllPris for all priorities.
// This goes down the entire tree from this node on down, as typically everything under
// will not get an explicit disconnect call because no further updating will happen
func (wb *WidgetBase) DisconnectAllEvents(pri EventPris) {
	em := wb.EventMgr()
	if em == nil {
		return
	}
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		_, ni := AsWidget(k)
		if ni == nil || ni.IsDeleted() || ni.IsDestroyed() {
			return ki.Break // going into a different type of thing, bail
		}
		em.DisconnectAllEvents(ni.This(), pri)
		return ki.Continue
	})
}

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

// FirstContainingPoint finds the first node whose WinBBox contains the given
// point -- nil if none.  If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered
func (wb *WidgetBase) FirstContainingPoint(pt image.Point, leavesOnly bool) ki.Ki {
	var rval ki.Ki
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		if k == wb.This() {
			return ki.Continue
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
		}
		_, w := AsWidget(k)
		if w == nil || w.IsDeleted() || w.IsDestroyed() {
			// 3D?
			return ki.Break
		}
		if w.PosInWinBBox(pt) {
			rval = w.This()
			return ki.Break
		}
		return ki.Continue
	})
	return rval
}

// AllWithinBBox returns a list of all nodes whose WinBBox is fully contained
// within the given BBox. If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered.
func (wb *WidgetBase) AllWithinBBox(bbox image.Rectangle, leavesOnly bool) ki.Slice {
	var rval ki.Slice
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		if k == wb.This() {
			return ki.Continue
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
		}
		_, w := AsWidget(k)
		if w == nil || w.IsDeleted() || w.IsDestroyed() {
			// 3D?
			return ki.Break
		}
		if w.WinBBoxInBBox(bbox) {
			rval = append(rval, w.This())
		}
		return ki.Continue
	})
	return rval
}
