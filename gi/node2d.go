package gi

import (
	"fmt"
	"image"

	"goki.dev/ki/v2"
)

// set our window-level BBox from vp and our bbox
func (nb *Node2DBase) SetWinBBox() {
	nb.BBoxMu.Lock()
	defer nb.BBoxMu.Unlock()
	if nb.Viewport != nil {
		nb.Viewport.BBoxMu.RLock()
		nb.WinBBox = nb.VpBBox.Add(nb.Viewport.WinBBox.Min)
		nb.Viewport.BBoxMu.RUnlock()
	} else {
		nb.WinBBox = nb.VpBBox
	}
}

// ComputeBBox2DBase -- computes the VpBBox and WinBBox from BBox, with
// whatever delta may be in effect
func (nb *Node2DBase) ComputeBBox2DBase(parBBox image.Rectangle, delta image.Point) {
	nb.BBoxMu.Lock()
	nb.ObjBBox = nb.BBox.Add(delta)
	nb.VpBBox = parBBox.Intersect(nb.ObjBBox)
	nb.SetInvisibleState(nb.VpBBox == image.Rectangle{})
	nb.BBoxMu.Unlock()
	nb.SetWinBBox()
}

// BBoxReport reports on all the bboxes for everything in the tree
func (nb *Node2DBase) BBoxReport() string {
	rpt := ""
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		nii, ni := KiToNode2D(k)
		if nii == nil || ni.IsDeleted() || ni.IsDestroyed() {
			return ki.Break
		}
		rpt += fmt.Sprintf("%v: vp: %v, win: %v\n", ni.Nm, ni.VpBBox, ni.WinBBox)
		return ki.Continue
	})
	return rpt
}

// ParentReRenderAnchor returns parent (including this node)
// that is a ReRenderAnchor -- for optimized re-rendering
func (nb *Node2DBase) ParentReRenderAnchor() Node2D {
	var par Node2D
	nb.FuncUp(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		nii, ni := KiToNode2D(k)
		if nii == nil {
			return false // don't keep going up
		}
		if ni.IsReRenderAnchor() {
			par = nii
			return false
		}
		return true
	})
	return par
}
