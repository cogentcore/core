// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"reflect"
	"slices"

	"cogentcore.org/core/goosi"
	"cogentcore.org/core/laser"
)

// RenderWinList is a list of windows.
type RenderWinList []*RenderWin

// Add adds a window to the list.
func (wl *RenderWinList) Add(w *RenderWin) {
	RenderWinGlobalMu.Lock()
	*wl = append(*wl, w)
	RenderWinGlobalMu.Unlock()
}

// Delete removes a window from the list.
func (wl *RenderWinList) Delete(w *RenderWin) {
	RenderWinGlobalMu.Lock()
	defer RenderWinGlobalMu.Unlock()
	*wl = slices.DeleteFunc(*wl, func(rw *RenderWin) bool {
		return rw == w
	})
}

// FindName finds window with given name on list (case sensitive) -- returns
// window and true if found, nil, false otherwise.
func (wl *RenderWinList) FindName(name string) (*RenderWin, bool) {
	RenderWinGlobalMu.Lock()
	defer RenderWinGlobalMu.Unlock()
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
func (wl *RenderWinList) FindData(data any) (*RenderWin, bool) {
	if laser.AnyIsNil(data) {
		return nil, false
	}
	typ := reflect.TypeOf(data)
	if !typ.Comparable() {
		return nil, false
	}
	RenderWinGlobalMu.Lock()
	defer RenderWinGlobalMu.Unlock()
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

// FindRenderWin finds window with given goosi.RenderWin on list -- returns
// window and true if found, nil, false otherwise.
func (wl *RenderWinList) FindRenderWin(osw goosi.Window) (*RenderWin, bool) {
	RenderWinGlobalMu.Lock()
	defer RenderWinGlobalMu.Unlock()
	for _, wi := range *wl {
		if wi.GoosiWin == osw {
			return wi, true
		}
	}
	return nil, false
}

// Len returns the length of the list, concurrent-safe
func (wl *RenderWinList) Len() int {
	RenderWinGlobalMu.Lock()
	defer RenderWinGlobalMu.Unlock()
	return len(*wl)
}

// Win gets window at given index, concurrent-safe
func (wl *RenderWinList) Win(idx int) *RenderWin {
	RenderWinGlobalMu.Lock()
	defer RenderWinGlobalMu.Unlock()
	if idx >= len(*wl) || idx < 0 {
		return nil
	}
	return (*wl)[idx]
}

// Focused returns the (first) window in this list that has the WinGotFocus flag set
// and the index in the list (nil, -1 if not present)
func (wl *RenderWinList) Focused() (*RenderWin, int) {
	RenderWinGlobalMu.Lock()
	defer RenderWinGlobalMu.Unlock()

	for i, fw := range *wl {
		if fw.HasFlag(WinGotFocus) {
			return fw, i
		}
	}
	return nil, -1
}

// FocusNext focuses on the next window in the list, after the current Focused() one
// skips minimized windows
func (wl *RenderWinList) FocusNext() (*RenderWin, int) {
	fw, i := wl.Focused()
	if fw == nil {
		return nil, -1
	}
	RenderWinGlobalMu.Lock()
	defer RenderWinGlobalMu.Unlock()
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
		if !fw.GoosiWin.Is(goosi.Minimized) {
			fw.GoosiWin.Raise()
			break
		}
	}
	return fw, i
}

// AllRenderWins is the list of all windows that have been created (dialogs, main
// windows, etc).
var AllRenderWins RenderWinList

// DialogRenderWins is the list of only dialog windows that have been created.
var DialogRenderWins RenderWinList

// MainRenderWins is the list of main windows (non-dialogs) that have been
// created.
var MainRenderWins RenderWinList
