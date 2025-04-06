// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"reflect"
	"slices"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/system"
)

// renderWindowList is a list of [renderWindow]s.
type renderWindowList []*renderWindow

// add adds a window to the list.
func (wl *renderWindowList) add(w *renderWindow) {
	renderWindowGlobalMu.Lock()
	*wl = append(*wl, w)
	renderWindowGlobalMu.Unlock()
}

// delete removes a window from the list.
func (wl *renderWindowList) delete(w *renderWindow) {
	renderWindowGlobalMu.Lock()
	defer renderWindowGlobalMu.Unlock()
	*wl = slices.DeleteFunc(*wl, func(rw *renderWindow) bool {
		return rw == w
	})
}

// FindName finds the window with the given name or title
// on the list (case sensitive).
// It returns the window if found and nil otherwise.
func (wl *renderWindowList) FindName(name string) *renderWindow {
	renderWindowGlobalMu.Lock()
	defer renderWindowGlobalMu.Unlock()
	for _, w := range *wl {
		if w.name == name || w.title == name {
			return w
		}
	}
	return nil
}

// findData finds window with given Data on list -- returns
// window and true if found, nil, false otherwise.
// data of type string works fine -- does equality comparison on string contents.
func (wl *renderWindowList) findData(data any) (*renderWindow, bool) {
	if reflectx.IsNil(reflect.ValueOf(data)) {
		return nil, false
	}
	typ := reflect.TypeOf(data)
	if !typ.Comparable() {
		fmt.Printf("programmer error in RenderWinList.FindData: Scene.Data type %s not comparable (value: %v)\n", typ.String(), data)
		return nil, false
	}
	renderWindowGlobalMu.Lock()
	defer renderWindowGlobalMu.Unlock()
	for _, wi := range *wl {
		msc := wi.MainScene()
		if msc == nil {
			continue
		}
		if msc.Data == data {
			return wi, true
		}
	}
	return nil, false
}

// focused returns the (first) window in this list that has the WinGotFocus flag set
// and the index in the list (nil, -1 if not present)
func (wl *renderWindowList) focused() (*renderWindow, int) {
	renderWindowGlobalMu.Lock()
	defer renderWindowGlobalMu.Unlock()

	for i, fw := range *wl {
		if fw.flags.HasFlag(winGotFocus) {
			return fw, i
		}
	}
	return nil, -1
}

// focusNext focuses on the next window in the list, after the current Focused() one.
// It skips minimized windows.
func (wl *renderWindowList) focusNext() (*renderWindow, int) {
	fw, i := wl.focused()
	if fw == nil {
		return nil, -1
	}
	renderWindowGlobalMu.Lock()
	defer renderWindowGlobalMu.Unlock()
	sz := len(*wl)
	if sz == 1 {
		return nil, -1
	}

	for j := 0; j < sz-1; j++ {
		if i == sz-1 {
			i = 0
		} else {
			i++
		}
		fw = (*wl)[i]
		if !fw.SystemWindow.Is(system.Minimized) {
			fw.SystemWindow.Raise()
			break
		}
	}
	return fw, i
}

// AllRenderWindows is the list of all [renderWindow]s that have been created
// (dialogs, main windows, etc).
var AllRenderWindows renderWindowList

// dialogRenderWindows is the list of only dialog [renderWindow]s that
// have been created.
var dialogRenderWindows renderWindowList

// mainRenderWindows is the list of main [renderWindow]s (non-dialogs) that
// have been created.
var mainRenderWindows renderWindowList
