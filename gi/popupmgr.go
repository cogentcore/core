// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/goosi"
)

// PopupMgr manages popup Stages within a main Stage element (Window, etc).
// Handles all the logic about stacks of Stage elements
// and routing of events to them.
type PopupMgr struct {
	StageMgrBase
}

// HandleEvent processes Popup events.
// Only gets OnFocus events if in focus.
// requires outer RenderContext mutex!
func (pm *PopupMgr) HandleEvent(evi goosi.Event) {
	top := pm.Top()
	if top == nil {
		return
	}
	if evi.HasPos() {
		pos := evi.Pos()
		if top.IsPtIn(pos) { // stage handles
			top.HandleEvent(evi) // either will be handled or not..
			return
		}
		if top.ClickOff {
			if evi.Type() == goosi.MouseEvent {
				pm.PopDelete()
				// todo: could mark as Handled to absorb
			}
		}
		if top.Modal { // absorb any other events!
			evi.SetHandled()
		}
		// else not Handled, will pass on
	} else { // either focus or other, send it down
		pm.HandleEvent(evi) // either will be handled or not..
	}
}

/*
// CurPopupIsTooltip returns true if current popup is a tooltip
func (pm *PopupMgr) CurPopupIsTooltip() bool {
	return PopupIsTooltip(pm.CurPopup())
}

// DeleteTooltip deletes any tooltip popup (called when hover ends)
func (pm *PopupMgr) DeleteTooltip() {
	pm.Mu.RLock()
	if pm.CurPopupIsTooltip() {
		pm.delPop = true
	}
	pm.Mu.RUnlock()
}

// SetNextPopup sets the next popup, and what to focus on in that popup if non-nil
func (pm *PopupMgr) SetNextPopup(pop *Scene, focus *Stage) {
}

// SetNextPopup sets the next popup, and what to focus on in that popup if non-nil
func (pm *PopupMgr) SetNextPopupImpl(pop, focus *Stage) {
	pm.Mu.Lock()
	pm.NextPopup = pop
	pm.PopupFocus = focus
	pm.Mu.Unlock()
}

// SetDelPopup sets the popup to delete next time through event loop
func (pm *PopupMgr) SetDelPopup(pop *Stage) {
	pm.Mu.Lock()
	pm.DelPopup = pop
	pm.Mu.Unlock()
}

// ShouldDeletePopupMenu returns true if the given popup item should be deleted
func (pm *PopupMgr) ShouldDeletePopupMenu(pop *Stage, me *mouse.Event) bool {
	// if we have a dialog open, close it if we didn't click in it
	if dlg, ok := pop.(*Dialog); ok {
		log.Println("pos", me.Pos(), "bbox", dlg.WinBBox)
		return !me.Pos().In(dlg.WinBBox)
	}
	if !PopupIsMenu(pop) {
		return false
	}
	if pm.NextPopup != nil && PopupIsMenu(pm.NextPopup) { // popping up another menu
		return false
	}
	if me.Button != mouse.Left && pm.EventMgr.Dragging == nil { // probably menu activation in first place
		return false
	}
	return true
}

// DisconnectPopup disconnects given popup -- typically the current one.
func (pm *PopupMgr) DisconnectPopup(pop *Stage) {
	pm.PopDraws.Delete(pop.(Node))
	ki.SetParent(pop, nil) // don't redraw the popup anymore
}

func (pm *PopupMgr) ClosePopup(sc *Scene) bool {
	return false
}

// ClosePopup close given popup -- must be the current one -- returns false if not.
func (pm *PopupMgr) ClosePopupImpl(pop *Stage) bool {
	if pop != pm.CurPopup() {
		return false
	}
	pm.Mu.Lock()
	pm.ResetUpdateRegions()
	if pm.Popup == pm.DelPopup {
		pm.DelPopup = nil
	}
	pm.UpMu.Lock()
	pm.DisconnectPopup(pop)
	pm.UpMu.Unlock()
	popped := pm.PopPopup(pop)
	pm.Mu.Unlock()
	if popped {
		pm.EventMgr.PopFocus()
	}
	pm.UploadAllScenes()
	return true
}
*/
