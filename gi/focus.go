// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"

	"goki.dev/girl/states"
	"goki.dev/ki/v2"
)

// FocusChanges are the kinds of changes that can be reported via
// FocusChanged method
type FocusChanges int32 //enums:enum

const (
	// FocusLost means that keyboard focus is on a different widget
	// (typically) and this one lost focus
	FocusLost FocusChanges = iota

	// FocusGot means that this widget just got keyboard focus
	FocusGot

	// FocusInactive means that although this widget retains keyboard focus
	// (nobody else has it), the user has clicked on something else and
	// therefore the focus should be considered inactive (distracted), and any
	// changes should be applied as this other action could result in closing
	// of a dialog etc.  Keyboard events will still be sent to the focus
	// widget, but it is up to the widget if or how to process them (e.g., it
	// could reactivate on its own).
	FocusInactive

	// FocusActive means that the user has moved the mouse back into the
	// focused widget to resume active keyboard focus.
	FocusActive
)

// FocusChanged handles the default behavior for node focus changes
// by calling [NodeBase.SetNeedsStyle] and sending an update signal.
func (wb *WidgetBase) FocusChanged(change FocusChanges) {
	wb.ApplyStyleUpdate(wb.Sc)
}

// GrabFocus grabs the keyboard input focus on this item or the first item within it
// that can be focused (if none, then goes ahead and sets focus to this object)
func (wb *WidgetBase) GrabFocus() {
	foc := wb.This().(Widget)
	if !foc.AbilityIs(states.Focusable) {
		wb.WalkPre(func(k ki.Ki) bool {
			kwi, kwb := AsWidget(k)
			if kwb == nil || kwb.This() == nil || kwb.Is(ki.Deleted) || kwb.Is(ki.Destroyed) {
				return ki.Break
			}
			if !kwb.AbilityIs(states.Focusable) {
				return ki.Continue
			}
			foc = kwi
			return ki.Break // done
		})
	}
	em := wb.EventMgr()
	if em != nil {
		fmt.Println("set focus:", foc.Path())
		em.SetFocus(foc.(Widget))
	}
}

// FocusNext moves the focus onto the next item
func (wb *WidgetBase) FocusNext() {
	em := wb.EventMgr()
	if em != nil {
		em.FocusNext(em.Focus)
	}
}

// FocusPrev moves the focus onto the previous item
func (wb *WidgetBase) FocusPrev() {
	em := wb.EventMgr()
	if em != nil {
		em.FocusPrev(em.Focus)
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
	// em := wb.EventMgr()
	// if em == nil {
	// 	return false
	// }
	// cur := em.CurFocus()
	// if cur == nil {
	// 	return false
	// }
	// if cur == wb.This() {
	// 	return true
	// }
	// plev := cur.ParentLevel(wb.This())
	// if plev < 0 {
	// 	return false
	// }
	return true
}
