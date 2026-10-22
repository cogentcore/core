// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"maps"

	"cogentcore.org/core/events"
)

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

// SetAsyncRunner sets the function that runs the listeners of the view
// with the given unique id for an event that is sent from a background
// goroutine, as when the syntax highlighting markup has been updated.
// It must call the given function with whatever lock the listeners need,
// such as the AsyncLock of a widget, which already holds that lock for
// events sent while it handles its own events.
func (ls *Lines) SetAsyncRunner(vid int, run func(f func())) {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	if vw != nil {
		vw.runAsync = run
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

// sendInputAsync is [Lines.sendInput] for a background goroutine:
// the listeners of each view are run by its [Lines.SetAsyncRunner]
// function, if it has one.
func (ls *Lines) sendInputAsync() {
	if ls.batchUpdating {
		return
	}
	e := &events.Base{Typ: events.Input}
	e.Init()
	// The views can be changed by the goroutines of their widgets meanwhile,
	// so a view that has been deleted since is skipped: its widget can be
	// gone. That is checked under the lock of the runner, which the widget
	// holds when it deletes its view.
	ls.Lock()
	views := maps.Clone(ls.views)
	ls.Unlock()
	for vid, vw := range views {
		if vw.runAsync == nil {
			if ls.hasView(vid, vw) {
				vw.listeners.Call(e)
			}
			continue
		}
		vw.runAsync(func() {
			if ls.hasView(vid, vw) {
				vw.listeners.Call(e)
			}
		})
	}
}

// hasView returns whether the given view is still the view with the
// given id, so that it has not been deleted.
func (ls *Lines) hasView(vid int, vw *view) bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.views[vid] == vw
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
