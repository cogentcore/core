// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import "cogentcore.org/core/events"

// OnChange adds an event listener function for the [events.Change] event.
// This is used for large-scale changes in the text, such as opening a
// new file or setting new text, or EditDone or Save.
func (ls *Lines) OnChange(fun func(e events.Event)) {
	ls.Lock()
	defer ls.Unlock()
	ls.listeners.Add(events.Change, fun)
}

// OnInput adds an event listener function for the [events.Input] event.
func (ls *Lines) OnInput(fun func(e events.Event)) {
	ls.Lock()
	defer ls.Unlock()
	ls.listeners.Add(events.Input, fun)
}

// SendChange sends a new [events.Change] event, which is used to signal
// that the text has changed. This is used for large-scale changes in the
// text, such as opening a new file or setting new text, or EditoDone or Save.
func (ls *Lines) SendChange() {
	ls.Lock()
	defer ls.Unlock()
	ls.SendChange()
}

// SendInput sends a new [events.Input] event, which is used to signal
// that the text has changed. This is sent after every fine-grained change in
// in the text, and is used by text widgets to drive updates. It is blocked
// during batchUpdating and sent at batchUpdateEnd.
func (ls *Lines) SendInput() {
	ls.Lock()
	defer ls.Unlock()
	ls.sendInput()
}

// EditDone finalizes any current editing, sends Changed event.
func (ls *Lines) EditDone() {
	ls.Lock()
	defer ls.Unlock()
	ls.editDone()
}

//////// unexported api

// sendChange sends a new [events.Change] event, which is used to signal
// that the text has changed. This is used for large-scale changes in the
// text, such as opening a new file or setting new text, or EditoDone or Save.
func (ls *Lines) sendChange() {
	e := &events.Base{Typ: events.Change}
	e.Init()
	ls.listeners.Call(e)
}

// sendInput sends a new [events.Input] event, which is used to signal
// that the text has changed. This is sent after every fine-grained change in
// in the text, and is used by text widgets to drive updates. It is blocked
// during batchUpdating and sent at batchUpdateEnd.
func (ls *Lines) sendInput() {
	if ls.batchUpdating {
		return
	}
	e := &events.Base{Typ: events.Input}
	e.Init()
	ls.listeners.Call(e)
}

// editDone finalizes any current editing, sends Changed event.
func (ls *Lines) editDone() {
	ls.autosaveDelete()
	ls.changed = true
	ls.sendChange()
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
	ls.sendInput()
}
