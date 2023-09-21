package gi

import (
	"goki.dev/ki/v2"
)

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
