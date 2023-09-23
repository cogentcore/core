// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"reflect"

	"goki.dev/goosi"
	"goki.dev/ordmap"
	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"
	"golang.org/x/exp/slices"
	"golang.org/x/image/draw"
)

// OSWinList is a list of windows.
type OSWinList []*OSWin

// Add adds a window to the list.
func (wl *OSWinList) Add(w *OSWin) {
	OSWinGlobalMu.Lock()
	*wl = append(*wl, w)
	OSWinGlobalMu.Unlock()
}

// Delete removes a window from the list -- returns true if deleted.
func (wl *OSWinList) Delete(w *OSWin) bool {
	OSWinGlobalMu.Lock()
	defer OSWinGlobalMu.Unlock()
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
func (wl *OSWinList) FindName(name string) (*OSWin, bool) {
	OSWinGlobalMu.Lock()
	defer OSWinGlobalMu.Unlock()
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
func (wl *OSWinList) FindData(data any) (*OSWin, bool) {
	if laser.IfaceIsNil(data) {
		return nil, false
	}
	typ := reflect.TypeOf(data)
	if !typ.Comparable() {
		return nil, false
	}
	OSWinGlobalMu.Lock()
	defer OSWinGlobalMu.Unlock()
	for _, wi := range *wl {
		if wi.Data == data {
			return wi, true
		}
	}
	return nil, false
}

// FindOSWin finds window with given goosi.OSWin on list -- returns
// window and true if found, nil, false otherwise.
func (wl *OSWinList) FindOSWin(osw goosi.OSWin) (*OSWin, bool) {
	OSWinGlobalMu.Lock()
	defer OSWinGlobalMu.Unlock()
	for _, wi := range *wl {
		if wi.OSWin == osw {
			return wi, true
		}
	}
	return nil, false
}

// Len returns the length of the list, concurrent-safe
func (wl *OSWinList) Len() int {
	OSWinGlobalMu.Lock()
	defer OSWinGlobalMu.Unlock()
	return len(*wl)
}

// Win gets window at given index, concurrent-safe
func (wl *OSWinList) Win(idx int) *OSWin {
	OSWinGlobalMu.Lock()
	defer OSWinGlobalMu.Unlock()
	if idx >= len(*wl) || idx < 0 {
		return nil
	}
	return (*wl)[idx]
}

// Focused returns the (first) window in this list that has the WinFlagGotFocus flag set
// and the index in the list (nil, -1 if not present)
func (wl *OSWinList) Focused() (*OSWin, int) {
	OSWinGlobalMu.Lock()
	defer OSWinGlobalMu.Unlock()

	for i, fw := range *wl {
		if fw.HasFlag(int(WinFlagGotFocus)) {
			return fw, i
		}
	}
	return nil, -1
}

// FocusNext focuses on the next window in the list, after the current Focused() one
// skips minimized windows
func (wl *OSWinList) FocusNext() (*OSWin, int) {
	fw, i := wl.Focused()
	if fw == nil {
		return nil, -1
	}
	OSWinGlobalMu.Lock()
	defer OSWinGlobalMu.Unlock()
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

// AllOSWins is the list of all windows that have been created (dialogs, main
// windows, etc).
var AllOSWins OSWinList

// DialogOSWins is the list of only dialog windows that have been created.
var DialogOSWins OSWinList

// MainOSWins is the list of main windows (non-dialogs) that have been
// created.
var MainOSWins OSWinList

// FocusOSWins is a "recents" stack of window names that have focus
// when a window gets focus, it pops to the top of this list
// when a window is closed, it is removed from the list, and the top item
// on the list gets focused.
var FocusOSWins []string

//////////////////////////////////////////////////////////////////////
//  OSWin region updates

// OSWinUpdates are a list of window update regions -- manages
// the index for a window bounding box region, which corresponds to
// the vdraw image index holding this image.
// Automatically handles range issues.
type OSWinUpdates struct {

	// starting index for this set of regions
	StartIdx int `desc:"starting index for this set of regions"`

	// max index (exclusive) for this set of regions
	MaxIdx int `desc:"max index (exclusive) for this set of regions"`

	// order of updates to draw -- when an existing image is updated it goes to the top of the stack.
	Order []int `desc:"order of updates to draw -- when an existing image is updated it goes to the top of the stack."`

	// updates that must be drawn before direct uploads because they fully occlude them
	BeforeDir []int `desc:"updates that must be drawn before direct uploads because they fully occlude them"`

	// ordered map of updates -- order (indx) is the image
	Updates *ordmap.Map[image.Rectangle, *Scene] `desc:"ordered map of updates -- order (indx) is the image"`
}

// SetIdxRange sets the index range based on starting index and n
func (wu *OSWinUpdates) SetIdxRange(st, n int) {
	wu.StartIdx = st
	wu.MaxIdx = st + n
}

// Init checks if ordered map needs to be allocated
func (wu *OSWinUpdates) Init() {
	if wu.Updates == nil {
		wu.Updates = ordmap.New[image.Rectangle, *Scene]()
	}
}

// Reset resets the ordered map
func (wu *OSWinUpdates) Reset() {
	wu.Updates = nil
	wu.Order = nil
	wu.BeforeDir = nil
}

func regPixCnt(r image.Rectangle) int {
	sz := r.Size()
	return sz.X * sz.Y
}

// Add adds a new update, returning index to store for given winBBox
// (could be existing), and bool = true if new index exceeds max range.
// If it is an exact match for an existing bbox, then that is returned.
func (wu *OSWinUpdates) Add(winBBox image.Rectangle, sc *Scene) (int, bool) {
	wu.Init()
	idx, has := wu.Updates.IdxByKeyTry(winBBox)
	if has {
		wu.MoveIdxToTop(idx)
		return wu.Idx(idx), false
	}
	idx = wu.Updates.Len()
	if wu.Idx(idx) >= wu.MaxIdx {
		return idx, true
	}
	wu.Updates.Add(winBBox, sc)
	wu.Order = append(wu.Order, idx)
	return wu.Idx(idx), false
}

// MoveIdxToTop moves the given index to top of the order
func (wu *OSWinUpdates) MoveIdxToTop(idx int) {
	for i, ii := range wu.Order {
		if ii == idx {
			wu.Order = slices.Delete(wu.Order, i, i+1)
			break
		}
	}
	wu.Order = append(wu.Order, idx)
}

// MoveIdxToBeforeDir moves the given index to the BeforeDir list
func (wu *OSWinUpdates) MoveIdxToBeforeDir(idx int) {
	for i, ii := range wu.Order {
		if ii == idx {
			wu.Order = slices.Delete(wu.Order, i, i+1)
			break
		}
	}
	wu.BeforeDir = append(wu.BeforeDir, idx)
}

// Idx returns the given 0-based index plus StartIdx
func (wu *OSWinUpdates) Idx(idx int) int {
	return wu.StartIdx + idx
}

// DrawImages iterates over regions and calls Copy on given
// vdraw.Drawer for each region.  beforeDir calls items on the
// BeforeDir list, else regular Order.
func (wu *OSWinUpdates) DrawImages(drw *vdraw.Drawer, beforeDir bool) {
	if wu.Updates == nil {
		return
	}
	list := wu.Order
	if beforeDir {
		list = wu.BeforeDir
	}
	for _, i := range list {
		kv := wu.Updates.Order[i]
		winBBox := kv.Key
		idx := wu.Idx(i)
		drw.Copy(idx, 0, winBBox.Min, image.Rectangle{}, draw.Src, vgpu.NoFlipY)
	}
}

//////////////////////////////////////////////////////////////////////
//  OSWinDrawers

// OSWinDrawers are a list of gi.Node objects that draw
// directly to the window.  This list manages the index for
// the vdraw image index holding this image.
type OSWinDrawers struct {

	// starting index for this set of Nodes
	StartIdx int `desc:"starting index for this set of Nodes"`

	// max index (exclusive) for this set of Nodes
	MaxIdx int `desc:"max index (exclusive) for this set of Nodes"`

	// set to true to flip Y axis in drawing these images
	FlipY bool `desc:"set to true to flip Y axis in drawing these images"`

	// ordered map of nodes with window bounding box
	Nodes *ordmap.Map[*WidgetBase, image.Rectangle] `desc:"ordered map of nodes with window bounding box"`
}

// SetIdxRange sets the index range based on starting index and n
func (wu *OSWinDrawers) SetIdxRange(st, n int) {
	wu.StartIdx = st
	wu.MaxIdx = st + n
}

// Init checks if ordered map needs to be allocated
func (wu *OSWinDrawers) Init() {
	if wu.Nodes == nil {
		wu.Nodes = ordmap.New[*NodeBase, image.Rectangle]()
	}
}

// Reset resets the ordered map
func (wu *OSWinDrawers) Reset() {
	wu.Nodes = nil
}

// Add adds a new node, returning index to store for given winBBox
// (could be existing), and bool = true if new index exceeds max range
func (wu *OSWinDrawers) Add(node Widget, winBBox image.Rectangle) (int, bool) {
	nb := node.AsGiNode()
	wu.Init()
	idx, has := wu.Nodes.IdxByKeyTry(nb)
	if has {
		return wu.Idx(idx), false
	}
	wu.Nodes.Add(nb, winBBox)
	idx = wu.Idx(wu.Nodes.Len() - 1)
	if idx >= wu.MaxIdx {
		fmt.Printf("gi.OSWinDrawers: ERROR too many nodes of type\n")
		return idx, true
	}
	return idx, false
}

// Delete removes given node from list of drawers
func (wu *OSWinDrawers) Delete(node Widget) {
	nb := node.AsGiNode()
	wu.Nodes.DeleteKey(nb)
}

// Idx returns the given 0-based index plus StartIdx
func (wu *OSWinDrawers) Idx(idx int) int {
	return wu.StartIdx + idx
}

// SetWinBBox sets the window BBox for given element, indexed
// by its allocated index relative to StartIdx
func (wu *OSWinDrawers) SetWinBBox(idx int, bbox image.Rectangle) {
	ei := idx - wu.StartIdx
	wu.Nodes.Order[ei].Val = bbox
}

// DrawImages iterates over regions and calls Copy on given
// vdraw.Drawer for each region
func (wu *OSWinDrawers) DrawImages(drw *vdraw.Drawer) {
	if wu.Nodes == nil {
		return
	}
	for i, kv := range wu.Nodes.Order {
		nb := kv.Key
		if nb.This() == nil {
			continue
		}
		if !nb.This().(Node2D).IsVisible() {
			continue
		}
		winBBox := kv.Val
		idx := wu.Idx(i)
		mvoff := nb.ScBBox.Min.Sub(nb.ObjBBox.Min)
		ibb := winBBox.Sub(winBBox.Min).Add(mvoff)
		drw.Copy(idx, 0, winBBox.Min, ibb, draw.Src, wu.FlipY)
	}
}
