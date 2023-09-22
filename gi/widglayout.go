// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

////////////////////////////////////////////////////////////////////
// 	BBox bounding boxes

// set our window-level BBox from vp and our bbox
func (wb *WidgetBase) SetWinBBox() {
	wb.BBoxMu.Lock()
	defer wb.BBoxMu.Unlock()
	if wb.Viewport != nil {
		wb.Viewport.BBoxMu.RLock()
		wb.WinBBox = wb.VpBBox.Add(wb.Viewport.WinBBox.Min)
		wb.Viewport.BBoxMu.RUnlock()
	} else {
		wb.WinBBox = wb.VpBBox
	}
}

// ComputeBBox2DBase -- computes the VpBBox and WinBBox from BBox, with
// whatever delta may be in effect
func (wb *WidgetBase) ComputeBBox2DBase(parBBox image.Rectangle, delta image.Point) {
	wb.BBoxMu.Lock()
	wb.ObjBBox = wb.BBox.Add(delta)
	wb.VpBBox = parBBox.Intersect(wb.ObjBBox)
	wb.SetInvisibleState(wb.VpBBox == image.Rectangle{})
	wb.BBoxMu.Unlock()
	wb.SetWinBBox()
}

// BBoxReport reports on all the bboxes for everything in the tree
func (wb *WidgetBase) BBoxReport() string {
	rpt := ""
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		nii, ni := AsWidget(k)
		if nii == nil || ni.IsDeleted() || ni.IsDestroyed() {
			return ki.Break
		}
		rpt += fmt.Sprintf("%v: vp: %v, win: %v\n", ni.Nm, ni.VpBBox, ni.WinBBox)
		return ki.Continue
	})
	return rpt
}

// AddParentPos adds the position of our parent to our layout position --
// layout computations are all relative to parent position, so they are
// finally cached out at this stage also returns the size of the parent for
// setting units context relative to parent objects
func (wb *WidgetBase) AddParentPos() mat32.Vec2 {
	if pni, _ := KiToWidget(wb.Par); pni != nil {
		if pw := pni.AsWidget(); pw != nil {
			if !wb.IsField() {
				wb.LayState.Alloc.Pos = pw.LayState.Alloc.PosOrig.Add(wb.LayState.Alloc.PosRel)
			}
			return pw.LayState.Alloc.Size
		}
	}
	return mat32.Vec2Zero
}

// BBoxFromAlloc gets our bbox from Layout allocation.
func (wb *WidgetBase) BBoxFromAlloc() image.Rectangle {
	return mat32.RectFromPosSizeMax(wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size)
}

func (wb *WidgetBase) BBox2D() image.Rectangle {
	return wb.BBoxFromAlloc()
}

func (wb *WidgetBase) ComputeBBox2DParts(parBBox image.Rectangle, delta image.Point) {
	wb.ComputeBBox2DBase(parBBox, delta)
	if wb.Parts != nil {
		wb.Parts.This().(Widget).ComputeBBox2D(parBBox, delta)
	}
}

func (wb *WidgetBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	wb.ComputeBBox2DParts(parBBox, delta)
}

////////////////////////////////////////////////////////////////////
// 	GetSize

// set our LayState.Alloc.Size from constraints
func (wb *WidgetBase) GetSizeFromWH(w, h float32) {
	st := &wb.Style
	if st.Width.Dots > 0 {
		w = mat32.Max(st.Width.Dots, w)
	}
	if st.Height.Dots > 0 {
		h = mat32.Max(st.Height.Dots, h)
	}
	spcsz := st.BoxSpace().Size()
	w += spcsz.X
	h += spcsz.Y
	wb.LayState.Alloc.Size = mat32.Vec2{w, h}
}

// GetSizeAddSpace adds space to existing AllocSize
func (wb *WidgetBase) GetSizeAddSpace() {
	spc := wb.BoxSpace()
	wb.LayState.Alloc.Size.SetAdd(spc.Size())
}

// GetSizeSubSpace returns AllocSize minus 2 * BoxSpace -- the amount avail to the internal elements
func (wb *WidgetBase) GetSizeSubSpace() mat32.Vec2 {
	spc := wb.BoxSpace()
	return wb.LayState.Alloc.Size.Sub(spc.Size())
}

// GetSizeParts sets our size from those of our parts -- default..
func (wb *WidgetBase) GetSizeParts(vp *Viewport, iter int) {
	wb.InitLayout(vp)
	if wb.Parts == nil {
		return
	}
	wb.Parts.GetSizeTree(vp, iter)
	wb.LayState.Alloc.Size = wb.Parts.LayState.Size.Pref // get from parts
	wb.GetSizeAddSpace()
	if LayoutTrace {
		fmt.Printf("Size:   %v size from parts: %v, parts pref: %v\n", wb.Path(), wb.LayState.Alloc.Size, wb.Parts.LayState.Size.Pref)
	}
}

func (wb *WidgetBase) GetSize(vp *Viewport, iter int) {
	wb.GetSizeParts(vp, iter)
}

////////////////////////////////////////////////////////////////////
// 	DoLayout

func (wb *WidgetBase) DoLayoutParts(vp *Viewport, parBBox image.Rectangle, iter int) {
	if wb.Parts == nil {
		return
	}
	spc := wb.BoxSpace()
	wb.Parts.LayState.Alloc.Pos = wb.LayState.Alloc.Pos.Add(spc.Pos())
	wb.Parts.LayState.Alloc.Size = wb.LayState.Alloc.Size.Sub(spc.Size())
	wb.Parts.DoLayout(vp, parBBox, iter)
}

func (wb *WidgetBase) DoLayout(vp *Viewport, parBBox image.Rectangle, iter int) bool {
	wb.DoLayoutBase(vp, parBBox, true, iter) // init style
	wb.DoLayoutParts(vp, parBBox, iter)
	return wb.DoLayoutChildren(vp, iter)
}

func (wb *WidgetBase) InitLayout(vp *Viewport) bool {
	wb.StyMu.Lock()
	defer wb.StyMu.Unlock()
	wb.LayState.SetFromStyle(&wb.Style)
	return false
}

// todo: wtf with initStyle ??

// DoLayoutBase provides basic DoLayout functions -- good for most cases
func (wb *WidgetBase) DoLayoutBase(vp *Viewport, parBBox image.Rectangle, initStyle bool, iter int) {
	wi := wb.This().(Widget)
	psize := wb.AddParentPos()
	wb.LayState.Alloc.PosOrig = wb.LayState.Alloc.Pos
	if initStyle {
		SetUnitContext(&wb.Style, mvp, wb.NodeSize(), psize) // update units with final layout
	}
	wb.BBox = wi.BBox2D() // only compute once, at this point
	// note: if other styles are maintained, they also need to be updated!
	wi.ComputeBBox2D(parBBox, image.Point{}) // other bboxes from BBox
	if LayoutTrace {
		fmt.Printf("Layout: %v alloc pos: %v size: %v vpbb: %v winbb: %v\n", wb.Path(), wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size, wb.VpBBox, wb.WinBBox)
	}
	// typically DoLayoutChildren must be called after this!
}

// DoLayoutChildren does layout on all of node's children, giving them the
// ChildrenBBox -- default call at end of DoLayout.  Passes along whether
// any of the children need a re-layout -- typically DoLayout just returns
// this.
func (wb *WidgetBase) DoLayoutChildren(vp *Viewport, iter int) bool {
	redo := false
	cbb := wb.This().(Widget).ChildrenBBox2D()
	for _, kid := range nb.Kids {
		wi, _ := AsWidget(kid)
		if wi != nil {
			if wi.DoLayout(vp, cbb, iter) {
				redo = true
			}
		}
	}
	return redo
}

// ChildrenBBox2DWidget provides a basic widget box-model subtraction of
// margin and padding to children -- call in ChildrenBBox2D for most widgets
func (wb *WidgetBase) ChildrenBBox2DWidget() image.Rectangle {
	wb := wb.VpBBox
	spc := wb.BoxSpace()
	wb.Min.X += int(spc.Left)
	wb.Min.Y += int(spc.Top)
	wb.Max.X -= int(spc.Right)
	wb.Max.Y -= int(spc.Bottom)
	return wb
}

func (wb *WidgetBase) ChildrenBBox2D() image.Rectangle {
	return wb.ChildrenBBox2DWidget()
}

//////////////////////////////////////////////////////////////////
//		Move2D scrolling

func (wb *WidgetBase) Move2D(vp *Viewport, delta image.Point, parBBox image.Rectangle) {
	wb.Move2DBase(delta, parBBox)
	if wb.Parts != nil {
		wb.Parts.This().(Widget).Move2D(vp, delta, parBBox)
	}
	wb.Move2DChildren(delta)
}

// Move2DBase does the basic move on this node
func (wb *WidgetBase) Move2DBase(delta image.Point, parBBox image.Rectangle) {
	wb.LayState.Alloc.Pos = wb.LayState.Alloc.PosOrig.Add(mat32.NewVec2FmPoint(delta))
	wb.This().(Widget).ComputeBBox2D(parBBox, delta)
}

// Move2DTree does move2d pass -- each node iterates over children for maximum
// control -- this starts with parent ChildrenBBox and current delta -- can be
// called de novo
func (wb *WidgetBase) Move2DTree() {
	if wb.HasNoLayout() {
		return
	}
	parBBox := image.Rectangle{}
	pnii, pn := KiToWidget(wb.Par)
	if pn != nil {
		parBBox = pnii.ChildrenBBox2D()
	}
	delta := wb.LayState.Alloc.Pos.Sub(wb.LayState.Alloc.PosOrig).ToPoint()
	wb.This().(Widget).Move2D(delta, parBBox) // important to use interface version to get interface!
}

// todo: move should just update bboxes with offset from parent
// passed down

// Move2DChildren moves all of node's children, giving them the ChildrenBBox2D
// -- default call at end of Move2D
func (wb *WidgetBase) Move2DChildren(delta image.Point) {
	cbb := wb.This().(Node2D).ChildrenBBox2D()
	for _, kid := range wb.Kids {
		wi, _ := AsWidget(kid)
		if wi != nil {
			wi.Move2D(delta, cbb)
		}
	}
}

// ParentLayout returns the parent layout
func (wb *WidgetBase) ParentLayout() *Layout {
	ly := wb.ParentByType(LayoutType, ki.Embeds)
	if ly == nil {
		return nil
	}
	return ly.Embed(LayoutType).(*Layout) // todo: do this manually
}

// ParentScrollLayout returns the parent layout that has active scrollbars
func (wb *WidgetBase) ParentScrollLayout() *Layout {
	lyk := wb.ParentByType(LayoutType, ki.Embeds)
	if lyk == nil {
		return nil
	}
	ly := lyk.Embed(LayoutType).(*Layout)
	if ly.HasAnyScroll() {
		return ly
	}
	return ly.ParentScrollLayout()
}

// ScrollToMe tells my parent layout (that has scroll bars) to scroll to keep
// this widget in view -- returns true if scrolled
func (wb *WidgetBase) ScrollToMe() bool {
	ly := wb.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollToItem(wb.This().(Node2D))
}
