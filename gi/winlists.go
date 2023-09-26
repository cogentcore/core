// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"reflect"

	"goki.dev/goosi"
	"goki.dev/laser"
	"goki.dev/ordmap"
	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"
	"golang.org/x/exp/slices"
	"golang.org/x/image/draw"
)

// RenderWinList is a list of windows.
type RenderWinList []*RenderWin

// Add adds a window to the list.
func (wl *RenderWinList) Add(w *RenderWin) {
	RenderWinGlobalMu.Lock()
	*wl = append(*wl, w)
	RenderWinGlobalMu.Unlock()
}

// Delete removes a window from the list -- returns true if deleted.
func (wl *RenderWinList) Delete(w *RenderWin) bool {
	RenderWinGlobalMu.Lock()
	defer RenderWinGlobalMu.Unlock()
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
	// for _, wi := range *wl {
	// if wi.Data == data { // todo: now inside the stage
	// 	return wi, true
	// }
	// }
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

// Focused returns the (first) window in this list that has the WinFlagGotFocus flag set
// and the index in the list (nil, -1 if not present)
func (wl *RenderWinList) Focused() (*RenderWin, int) {
	RenderWinGlobalMu.Lock()
	defer RenderWinGlobalMu.Unlock()

	for i, fw := range *wl {
		if fw.HasFlag(WinFlagGotFocus) {
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
		if !fw.GoosiWin.IsMinimized() {
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

// FocusRenderWins is a "recents" stack of window names that have focus
// when a window gets focus, it pops to the top of this list
// when a window is closed, it is removed from the list, and the top item
// on the list gets focused.
var FocusRenderWins []string

//////////////////////////////////////////////////////////////////////
//  RenderWin region updates

// RenderWinUpdates are a list of window update regions -- manages
// the index for a window bounding box region, which corresponds to
// the vdraw image index holding this image.
// Automatically handles range issues.
type RenderWinUpdates struct {

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
func (wu *RenderWinUpdates) SetIdxRange(st, n int) {
	wu.StartIdx = st
	wu.MaxIdx = st + n
}

// Init checks if ordered map needs to be allocated
func (wu *RenderWinUpdates) Init() {
	if wu.Updates == nil {
		wu.Updates = ordmap.New[image.Rectangle, *Scene]()
	}
}

// Reset resets the ordered map
func (wu *RenderWinUpdates) Reset() {
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
func (wu *RenderWinUpdates) Add(winBBox image.Rectangle, sc *Scene) (int, bool) {
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
func (wu *RenderWinUpdates) MoveIdxToTop(idx int) {
	for i, ii := range wu.Order {
		if ii == idx {
			wu.Order = slices.Delete(wu.Order, i, i+1)
			break
		}
	}
	wu.Order = append(wu.Order, idx)
}

// MoveIdxToBeforeDir moves the given index to the BeforeDir list
func (wu *RenderWinUpdates) MoveIdxToBeforeDir(idx int) {
	for i, ii := range wu.Order {
		if ii == idx {
			wu.Order = slices.Delete(wu.Order, i, i+1)
			break
		}
	}
	wu.BeforeDir = append(wu.BeforeDir, idx)
}

// Idx returns the given 0-based index plus StartIdx
func (wu *RenderWinUpdates) Idx(idx int) int {
	return wu.StartIdx + idx
}

// DrawImages iterates over regions and calls Copy on given
// vdraw.Drawer for each region.  beforeDir calls items on the
// BeforeDir list, else regular Order.
func (wu *RenderWinUpdates) DrawImages(drw *vdraw.Drawer, beforeDir bool) {
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
//  RenderWinDrawers

// RenderWinDrawers are a list of gi.Node objects that draw
// directly to the window.  This list manages the index for
// the vdraw image index holding this image.
type RenderWinDrawers struct {

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
func (wu *RenderWinDrawers) SetIdxRange(st, n int) {
	wu.StartIdx = st
	wu.MaxIdx = st + n
}

// Init checks if ordered map needs to be allocated
func (wu *RenderWinDrawers) Init() {
	if wu.Nodes == nil {
		wu.Nodes = ordmap.New[*WidgetBase, image.Rectangle]()
	}
}

// Reset resets the ordered map
func (wu *RenderWinDrawers) Reset() {
	wu.Nodes = nil
}

// Add adds a new node, returning index to store for given winBBox
// (could be existing), and bool = true if new index exceeds max range
func (wu *RenderWinDrawers) Add(node Widget, winBBox image.Rectangle) (int, bool) {
	nb := node.AsWidget()
	wu.Init()
	idx, has := wu.Nodes.IdxByKeyTry(nb)
	if has {
		return wu.Idx(idx), false
	}
	wu.Nodes.Add(nb, winBBox)
	idx = wu.Idx(wu.Nodes.Len() - 1)
	if idx >= wu.MaxIdx {
		fmt.Printf("gi.RenderWinDrawers: ERROR too many nodes of type\n")
		return idx, true
	}
	return idx, false
}

// Delete removes given node from list of drawers
func (wu *RenderWinDrawers) Delete(node Widget) {
	nb := node.AsWidget()
	wu.Nodes.DeleteKey(nb)
}

// Idx returns the given 0-based index plus StartIdx
func (wu *RenderWinDrawers) Idx(idx int) int {
	return wu.StartIdx + idx
}

// SetWinBBox sets the window BBox for given element, indexed
// by its allocated index relative to StartIdx
func (wu *RenderWinDrawers) SetWinBBox(idx int, bbox image.Rectangle) {
	ei := idx - wu.StartIdx
	wu.Nodes.Order[ei].Val = bbox
}

// DrawImages iterates over regions and calls Copy on given
// vdraw.Drawer for each region
func (wu *RenderWinDrawers) DrawImages(drw *vdraw.Drawer) {
	if wu.Nodes == nil {
		return
	}
	for i, kv := range wu.Nodes.Order {
		nb := kv.Key
		if nb.This() == nil {
			continue
		}
		if !nb.This().(Widget).IsVisible() {
			continue
		}
		winBBox := kv.Val
		idx := wu.Idx(i)
		mvoff := nb.ScBBox.Min.Sub(nb.ObjBBox.Min)
		ibb := winBBox.Sub(winBBox.Min).Add(mvoff)
		drw.Copy(idx, 0, winBBox.Min, ibb, draw.Src, wu.FlipY)
	}
}
