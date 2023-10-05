// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/goosi/events"
)

// PopupStageMgr manages popup Stages within a main Stage element (Window, etc).
// Handles all the logic about stacks of Stage elements
// and routing of events to them.
type PopupStageMgr struct {
	StageMgrBase

	// Main is the MainStage that manages us
	Main *MainStage
}

// AsPopupMgr returns underlying PopupStageMgr or nil if a MainStageMgr
func (pm *PopupStageMgr) AsPopupMgr() *PopupStageMgr {
	return pm
}

// HandleEvent processes Popup events.
// Only gets OnFocus events if in focus.
// requires outer RenderContext mutex!
func (pm *PopupStageMgr) HandleEvent(evi events.Event) {
	top := pm.Top()
	if top == nil {
		return
	}
	tb := top.AsBase()
	ts := tb.Scene
	if evi.HasPos() {
		pos := evi.Pos()
		// fmt.Println("pos:", pos, "top geom:", ts.Geom)
		if pos.In(ts.Geom.Bounds()) {
			top.HandleEvent(evi) // either will be handled or not..
			if tb.Modal {
				evi.SetHandled()
			}
			return
		}
		if tb.ClickOff {
			if evi.Type() == events.MouseUp {
				pm.PopDelete()
				// todo: could mark as Handled to absorb
			}
		}
		if tb.Modal { // absorb any other events!
			evi.SetHandled()
		}
		// else not Handled, will pass on
	} else { // either focus or other, send it down
		top.HandleEvent(evi) // either will be handled or not..
	}
}
