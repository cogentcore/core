// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

/*

// StyleProps returns a property that contains another map of properties for a
// given styling selector, such as :normal :active :hover etc -- the
// convention is to prefix this selector with a : and use lower-case names, so
// we follow that.
func (nb *NodeBase) StyleProps(selector string) ki.Props {
	sp, ok := nb.PropInherit(selector, ki.NoInherit, ki.TypeProps) // yeah, use type's
	if !ok {
		return nil
	}
	spm, ok := sp.(ki.Props)
	if ok {
		return spm
	}
	log.Printf("gist.StyleProps: looking for a ki.Props for style selector: %v, instead got type: %T, for node: %v\n", selector, spm, nb.Path())
	return nil
}

// AggCSS aggregates css properties
func AggCSS(agg *ki.Props, css ki.Props) {
	if *agg == nil {
		*agg = make(ki.Props, len(css))
	}
	for key, val := range css {
		(*agg)[key] = val
	}
}

// ParentCSSAgg returns parent's CSSAgg styles or nil if not avail
func (nb *NodeBase) ParentCSSAgg() *ki.Props {
	if nb.Par == nil {
		return nil
	}
	pn := nb.Par.Embed(TypeNodeBase)
	if pn == nil {
		return nil
	}
	return &pn.(*NodeBase).CSSAgg
}

// FirstContainingPoint finds the first node whose WinBBox contains the given
// point -- nil if none.  If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered
func (nb *NodeBase) FirstContainingPoint(pt image.Point, leavesOnly bool) ki.Ki {
	var rval ki.Ki
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		if k == nb.This() {
			return ki.Continue
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
		}
		_, ni := KiToNode2D(k)
		if ni == nil || ni.IsDeleted() || ni.IsDestroyed() {
			// 3D?
			return ki.Break
		}
		if ni.PosInWinBBox(pt) {
			rval = ni.This()
			return ki.Break
		}
		return ki.Continue
	})
	return rval
}

// AllWithinBBox returns a list of all nodes whose WinBBox is fully contained
// within the given BBox. If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered.
func (nb *NodeBase) AllWithinBBox(bbox image.Rectangle, leavesOnly bool) ki.Slice {
	var rval ki.Slice
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		if k == nb.This() {
			return ki.Continue
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
		}
		_, ni := KiToNode2D(k)
		if ni == nil || ni.IsDeleted() || ni.IsDestroyed() {
			// 3D?
			return ki.Break
		}
		if ni.WinBBoxInBBox(bbox) {
			rval = append(rval, ni.This())
		}
		return ki.Continue
	})
	return rval
}

// ElementAndParentSize returns the size
// of this node as a [mat32.Vec2] object.
func (nb *NodeBase) NodeSize() mat32.Vec2 {
	return mat32.NewVec2FmPoint(nb.BBox.Size())
}

*/
