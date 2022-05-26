// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"reflect"

	"github.com/goki/gi/oswin"
	"github.com/goki/ki/kit"
	"github.com/goki/kigen/ordmap"
	"github.com/goki/vgpu/vdraw"
	"github.com/goki/vgpu/vgpu"
	"golang.org/x/image/draw"
)

// WindowList is a list of windows.
type WindowList []*Window

// Add adds a window to the list.
func (wl *WindowList) Add(w *Window) {
	WindowGlobalMu.Lock()
	*wl = append(*wl, w)
	WindowGlobalMu.Unlock()
}

// Delete removes a window from the list -- returns true if deleted.
func (wl *WindowList) Delete(w *Window) bool {
	WindowGlobalMu.Lock()
	defer WindowGlobalMu.Unlock()
	sz := len(*wl)
	got := false
	for i := sz - 1; i >= 0; i-- {
		wi := (*wl)[i]
		if wi == w {
			copy((*wl)[i:], (*wl)[i+1:])
			(*wl)[sz-1] = nil
			(*wl) = (*wl)[:sz-1]
			sz = len(*wl)
			got = true
		}
	}
	return got
}

// FindName finds window with given name on list (case sensitive) -- returns
// window and true if found, nil, false otherwise.
func (wl *WindowList) FindName(name string) (*Window, bool) {
	WindowGlobalMu.Lock()
	defer WindowGlobalMu.Unlock()
	for _, wi := range *wl {
		if wi.Nm == name {
			return wi, true
		}
	}
	return nil, false
}

// FindData finds window with given Data on list -- returns
// window and true if found, nil, false otherwise.
// data of type string works fine -- does equality comparison on string contents.
func (wl *WindowList) FindData(data interface{}) (*Window, bool) {
	if kit.IfaceIsNil(data) {
		return nil, false
	}
	typ := reflect.TypeOf(data)
	if !typ.Comparable() {
		return nil, false
	}
	WindowGlobalMu.Lock()
	defer WindowGlobalMu.Unlock()
	for _, wi := range *wl {
		if wi.Data == data {
			return wi, true
		}
	}
	return nil, false
}

// FindOSWin finds window with given oswin.Window on list -- returns
// window and true if found, nil, false otherwise.
func (wl *WindowList) FindOSWin(osw oswin.Window) (*Window, bool) {
	WindowGlobalMu.Lock()
	defer WindowGlobalMu.Unlock()
	for _, wi := range *wl {
		if wi.OSWin == osw {
			return wi, true
		}
	}
	return nil, false
}

// Len returns the length of the list, concurrent-safe
func (wl *WindowList) Len() int {
	WindowGlobalMu.Lock()
	defer WindowGlobalMu.Unlock()
	return len(*wl)
}

// Win gets window at given index, concurrent-safe
func (wl *WindowList) Win(idx int) *Window {
	WindowGlobalMu.Lock()
	defer WindowGlobalMu.Unlock()
	if idx >= len(*wl) || idx < 0 {
		return nil
	}
	return (*wl)[idx]
}

// Focused returns the (first) window in this list that has the WinFlagGotFocus flag set
// and the index in the list (nil, -1 if not present)
func (wl *WindowList) Focused() (*Window, int) {
	WindowGlobalMu.Lock()
	defer WindowGlobalMu.Unlock()

	for i, fw := range *wl {
		if fw.HasFlag(int(WinFlagGotFocus)) {
			return fw, i
		}
	}
	return nil, -1
}

// FocusNext focuses on the next window in the list, after the current Focused() one
// skips minimized windows
func (wl *WindowList) FocusNext() (*Window, int) {
	fw, i := wl.Focused()
	if fw == nil {
		return nil, -1
	}
	WindowGlobalMu.Lock()
	defer WindowGlobalMu.Unlock()
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
		if !fw.OSWin.IsMinimized() {
			fw.OSWin.Raise()
			break
		}
	}
	return fw, i
}

// AllWindows is the list of all windows that have been created (dialogs, main
// windows, etc).
var AllWindows WindowList

// DialogWindows is the list of only dialog windows that have been created.
var DialogWindows WindowList

// MainWindows is the list of main windows (non-dialogs) that have been
// created.
var MainWindows WindowList

// FocusWindows is a "recents" stack of window names that have focus
// when a window gets focus, it pops to the top of this list
// when a window is closed, it is removed from the list, and the top item
// on the list gets focused.
var FocusWindows []string

//////////////////////////////////////////////////////////////////////
//  Window region updates

// WindowUpdates are a list of window update regions -- manages
// the index for a window bounding box region, which corresponds to
// the vdraw image index holding this image.
// Automatically handles range issues.
type WindowUpdates struct {
	StartIdx int `desc:"starting index for this set of regions"`
	MaxIdx   int `desc:"max index (exclusive) for this set of regions"`

	Updates *ordmap.Map[image.Rectangle, *Viewport2D] `desc:"ordered map of updates"`
}

// SetIdxRange sets the index range based on starting index and n
func (wu *WindowUpdates) SetIdxRange(st, n int) {
	wu.StartIdx = st
	wu.MaxIdx = st + n
}

// Init checks if ordered map needs to be allocated
func (wu *WindowUpdates) Init() {
	if wu.Updates == nil {
		wu.Updates = ordmap.New[image.Rectangle, *Viewport2D]()
	}
}

// Reset resets the ordered map
func (wu *WindowUpdates) Reset() {
	wu.Updates = nil
}

func regPixCnt(r image.Rectangle) int {
	sz := r.Size()
	return sz.X * sz.Y
}

// Add adds a new update, returning index to store for given winBBox
// (could be existing), and bool = true if new index exceeds max range.
// If it is an exact match for an existing bbox, then that is returned.
func (wu *WindowUpdates) Add(winBBox image.Rectangle, vp *Viewport2D) (int, bool) {
	wu.Init()
	idx, has := wu.Updates.IdxByKey(winBBox)
	if has {
		return wu.Idx(idx), false
	}
	wu.Updates.Add(winBBox, vp)
	idx = wu.Idx(wu.Updates.Len() - 1)
	if idx >= wu.MaxIdx {
		return idx, true
	}
	return idx, false
}

// Idx returns the given 0-based index plus StartIdx
func (wu *WindowUpdates) Idx(idx int) int {
	return wu.StartIdx + idx
}

// DrawImages iterates over regions and calls Copy on given
// vdraw.Drawer for each region
func (wu *WindowUpdates) DrawImages(drw *vdraw.Drawer) {
	if wu.Updates == nil {
		return
	}
	for i, kv := range wu.Updates.Order {
		winBBox := kv.Key
		idx := wu.Idx(i)
		drw.Copy(idx, 0, winBBox.Min, image.ZR, draw.Src, vgpu.NoFlipY)
	}
}

//////////////////////////////////////////////////////////////////////
//  WindowDrawers

// WindowDrawers are a list of gi.Node objects that draw
// directly to the window.  This list manages the index for
// the vdraw image index holding this image.
type WindowDrawers struct {
	StartIdx int  `desc:"starting index for this set of Nodes"`
	MaxIdx   int  `desc:"max index (exclusive) for this set of Nodes"`
	FlipY    bool `desc:"set to true to flip Y axis in drawing these images"`

	Nodes *ordmap.Map[*NodeBase, image.Rectangle] `desc:"ordered map of nodes with window bounding box"`
}

// SetIdxRange sets the index range based on starting index and n
func (wu *WindowDrawers) SetIdxRange(st, n int) {
	wu.StartIdx = st
	wu.MaxIdx = st + n
}

// Init checks if ordered map needs to be allocated
func (wu *WindowDrawers) Init() {
	if wu.Nodes == nil {
		wu.Nodes = ordmap.New[*NodeBase, image.Rectangle]()
	}
}

// Reset resets the ordered map
func (wu *WindowDrawers) Reset() {
	wu.Nodes = nil
}

// Add adds a new node, returning index to store for given winBBox
// (could be existing), and bool = true if new index exceeds max range
func (wu *WindowDrawers) Add(node Node, winBBox image.Rectangle) (int, bool) {
	nb := node.AsGiNode()
	wu.Init()
	idx, has := wu.Nodes.IdxByKey(nb)
	if has {
		return wu.Idx(idx), false
	}
	wu.Nodes.Add(nb, winBBox)
	idx = wu.Idx(wu.Nodes.Len() - 1)
	if idx >= wu.MaxIdx {
		fmt.Printf("gi.WindowDrawers: ERROR too many nodes of type\n")
		return idx, true
	}
	return idx, false
}

// Delete removes given node from list of drawers
func (wu *WindowDrawers) Delete(node Node) {
	nb := node.AsGiNode()
	wu.Nodes.DeleteKey(nb)
}

// Idx returns the given 0-based index plus StartIdx
func (wu *WindowDrawers) Idx(idx int) int {
	return wu.StartIdx + idx
}

// DrawImages iterates over regions and calls Copy on given
// vdraw.Drawer for each region
func (wu *WindowDrawers) DrawImages(drw *vdraw.Drawer) {
	if wu.Nodes == nil {
		return
	}
	for i, kv := range wu.Nodes.Order {
		nb := kv.Key
		if !nb.This().(Node2D).IsVisible() {
			continue
		}
		winBBox := kv.Val
		idx := wu.Idx(i)
		drw.Copy(idx, 0, winBBox.Min, image.ZR, draw.Src, wu.FlipY)
	}
}
