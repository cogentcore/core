// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "cogentcore.org/core/system"

func init() {
	system.HandleBack = handleBack
}

// handleBack handles the back button or gesture of the system (on
// Android; see [system.HandleBack]), returning whether it went back within
// the app: it closes the top popup of the current window (as a menu), or
// else the top stage if it has a back button (see [Stage.hasBackButton])
// or is a dialog, as its back button or Escape does.
func handleBack() bool {
	rw := getCurrentRenderWindow()
	if rw == nil {
		return false
	}
	rc := rw.renderContext()
	if rc == nil {
		return false
	}
	rc.Lock()
	defer rc.Unlock()
	top := rw.mains.top()
	if top == nil || top.Scene == nil {
		return false
	}
	if top.popups != nil {
		if pop := top.popups.top(); pop != nil {
			return pop.ClosePopup()
		}
	}
	if top.Type != DialogStage && !top.hasBackButton(len(rw.mains.list()) > 1) {
		return false
	}
	return top.Scene.Close()
}

// hasBackButton returns whether the stage has a back button in its top
// bar (see [Stage.BackButton]), given whether there are other stages
// below it in its stack.
func (st *Stage) hasBackButton(below bool) bool {
	return st.BackButton.Or(st.FullWindow && !st.NewWindow && below)
}
