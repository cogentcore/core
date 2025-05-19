// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import "cogentcore.org/core/events"

// OnChange adds an event listener function to the view with given
// unique id, for the [events.Change] event.
// This is used for large-scale changes in the text, such as opening a
// new file or setting new text, or EditDone or Save.
func (ls *Lines) OnChange(vid int, fun func(e events.Event)) {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	if vw != nil {
		vw.listeners.Add(events.Change, fun)
	}
}

// OnInput adds an event listener function to the view with given
// unique id, for the [events.Input] event.
// This is sent after every fine-grained change in the text,
// and is used by text widgets to drive updates. It is blocked
// during batchUpdating.
func (ls *Lines) OnInput(vid int, fun func(e events.Event)) {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	if vw != nil {
		vw.listeners.Add(events.Input, fun)
	}
}

// OnClose adds an event listener function to the view with given
// unique id, for the [events.Close] event.
// This event is sent in the Close function.
func (ls *Lines) OnClose(vid int, fun func(e events.Event)) {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	if vw != nil {
		vw.listeners.Add(events.Close, fun)
	}
}

//////// unexported api

// sendChange sends a new [events.Change] event to all views listeners.
// Must never be called with the mutex lock in place!
// This is used to signal that the text has changed, for large-scale changes,
// such as opening a new file or setting new text, or EditoDone or Save.
func (ls *Lines) sendChange() {
	e := &events.Base{Typ: events.Change}
	e.Init()
	for _, vw := range ls.views {
		vw.listeners.Call(e)
	}
}

// sendInput sends a new [events.Input] event to all views listeners.
// Must never be called with the mutex lock in place!
// This is used to signal fine-grained changes in the text,
// and is used by text widgets to drive updates. It is blocked
// during batchUpdating.
func (ls *Lines) sendInput() {
	if ls.batchUpdating {
		return
	}
	e := &events.Base{Typ: events.Input}
	e.Init()
	for _, vw := range ls.views {
		vw.listeners.Call(e)
	}
}

// sendClose sends a new [events.Close] event to all views listeners.
// Must never be called with the mutex lock in place!
// Only sent in the Close function.
func (ls *Lines) sendClose() {
	e := &events.Base{Typ: events.Close}
	e.Init()
	for _, vw := range ls.views {
		vw.listeners.Call(e)
	}
}

// batchUpdateStart call this when starting a batch of updates.
// It calls AutoSaveOff and returns the prior state of that flag
// which must be restored using batchUpdateEnd.
func (ls *Lines) batchUpdateStart() (autoSave bool) {
	ls.batchUpdating = true
	ls.undos.NewGroup()
	autoSave = ls.autoSaveOff()
	return
}

// batchUpdateEnd call to complete BatchUpdateStart
func (ls *Lines) batchUpdateEnd(autoSave bool) {
	ls.autoSaveRestore(autoSave)
	ls.batchUpdating = false
}
