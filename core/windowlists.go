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

// RenderWindowList is a list of windows.
type RenderWindowList []*RenderWindow

// Add adds a window to the list.
func (wl *RenderWindowList) Add(w *RenderWindow) {
	RenderWindowGlobalMu.Lock()
	*wl = append(*wl, w)
	RenderWindowGlobalMu.Unlock()
}

// Delete removes a window from the list.
func (wl *RenderWindowList) Delete(w *RenderWindow) {
	RenderWindowGlobalMu.Lock()
	defer RenderWindowGlobalMu.Unlock()
	*wl = slices.DeleteFunc(*wl, func(rw *RenderWindow) bool {
		return rw == w
	})
}

// FindName finds window with given name on list (case sensitive) -- returns
// window and true if found, nil, false otherwise.
func (wl *RenderWindowList) FindName(name string) (*RenderWindow, bool) {
	RenderWindowGlobalMu.Lock()
	defer RenderWindowGlobalMu.Unlock()
	for _, wi := range *wl {
		if wi.Name == name {
			return wi, true
		}
	}
	return nil, false
}

// FindData finds window with given Data on list -- returns
// window and true if found, nil, false otherwise.
// data of type string works fine -- does equality comparison on string contents.
func (wl *RenderWindowList) FindData(data any) (*RenderWindow, bool) {
	if reflectx.AnyIsNil(data) {
		return nil, false
	}
	typ := reflect.TypeOf(data)
	if !typ.Comparable() {
		fmt.Printf("programmer error in RenderWinList.FindData: core.Scene.Data type %s not comparable (value: %v)\n", typ.String(), data)
		return nil, false
	}
	RenderWindowGlobalMu.Lock()
	defer RenderWindowGlobalMu.Unlock()
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

// FindRenderWindow finds window with given system.RenderWindow on list -- returns
// window and true if found, nil, false otherwise.
func (wl *RenderWindowList) FindRenderWindow(osw system.Window) (*RenderWindow, bool) {
	RenderWindowGlobalMu.Lock()
	defer RenderWindowGlobalMu.Unlock()
	for _, wi := range *wl {
		if wi.SystemWindow == osw {
			return wi, true
		}
	}
	return nil, false
}

// Len returns the length of the list, concurrent-safe
func (wl *RenderWindowList) Len() int {
	RenderWindowGlobalMu.Lock()
	defer RenderWindowGlobalMu.Unlock()
	return len(*wl)
}

// Win gets window at given index, concurrent-safe
func (wl *RenderWindowList) Win(idx int) *RenderWindow {
	RenderWindowGlobalMu.Lock()
	defer RenderWindowGlobalMu.Unlock()
	if idx >= len(*wl) || idx < 0 {
		return nil
	}
	return (*wl)[idx]
}

// Focused returns the (first) window in this list that has the WinGotFocus flag set
// and the index in the list (nil, -1 if not present)
func (wl *RenderWindowList) Focused() (*RenderWindow, int) {
	RenderWindowGlobalMu.Lock()
	defer RenderWindowGlobalMu.Unlock()

	for i, fw := range *wl {
		if fw.HasFlag(WindowGotFocus) {
			return fw, i
		}
	}
	return nil, -1
}

// FocusNext focuses on the next window in the list, after the current Focused() one
// skips minimized windows
func (wl *RenderWindowList) FocusNext() (*RenderWindow, int) {
	fw, i := wl.Focused()
	if fw == nil {
		return nil, -1
	}
	RenderWindowGlobalMu.Lock()
	defer RenderWindowGlobalMu.Unlock()
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

// AllRenderWindows is the list of all windows that have been created
// (dialogs, main windows, etc).
var AllRenderWindows RenderWindowList

// DialogRenderWindows is the list of only dialog windows that
// have been created.
var DialogRenderWindows RenderWindowList

// MainRenderWindows is the list of main windows (non-dialogs) that
// have been created.
var MainRenderWindows RenderWindowList
