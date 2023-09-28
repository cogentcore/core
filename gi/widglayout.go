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

// ComputeBBoxesBase -- computes the ScBBox and WinBBox from BBox, with
// whatever delta may be in effect
func (wb *WidgetBase) ComputeBBoxesBase(sc *Scene, parBBox image.Rectangle, delta image.Point) {
	wb.BBoxMu.Lock()
	wb.ObjBBox = wb.BBox.Add(delta)
	wb.ScBBox = parBBox.Intersect(wb.ObjBBox)
	wb.SetFlag(wb.ScBBox == image.Rectangle{}, Invisible)
	wb.BBoxMu.Unlock()
}

// BBoxReport reports on all the bboxes for everything in the tree
func (wb *WidgetBase) BBoxReport() string {
	rpt := ""
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		nii, ni := AsWidget(k)
		if nii == nil || ni.IsDeleted() || ni.IsDestroyed() {
			return ki.Break
		}
		rpt += fmt.Sprintf("%v: vp: %v\n", ni.Nm, ni.ScBBox)
		return ki.Continue
	})
	return rpt
}

// AddParentPos adds the position of our parent to our layout position --
// layout computations are all relative to parent position, so they are
// finally cached out at this stage also returns the size of the parent for
// setting units context relative to parent objects
func (wb *WidgetBase) AddParentPos() mat32.Vec2 {
	if pwi, pwb := AsWidget(wb.Par); pwi != nil {
		if !wb.IsField() {
			wb.LayState.Alloc.Pos = pwb.LayState.Alloc.PosOrig.Add(wb.LayState.Alloc.PosRel)
		}
		return pwb.LayState.Alloc.Size
	}
	return mat32.Vec2Zero
}

// BBoxFromAlloc gets our bbox from Layout allocation.
func (wb *WidgetBase) BBoxFromAlloc() image.Rectangle {
	return mat32.RectFromPosSizeMax(wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size)
}

func (wb *WidgetBase) BBoxes() image.Rectangle {
	return wb.BBoxFromAlloc()
}

func (wb *WidgetBase) ComputeBBoxesParts(sc *Scene, parBBox image.Rectangle, delta image.Point) {
	wb.ComputeBBoxesBase(sc, parBBox, delta)
	if wb.Parts != nil {
		// this is happening before parts has done its own layout.
		// set a breakpoint here -- probably do layout first then this
		wb.Parts.This().(Widget).ComputeBBoxes(sc, parBBox, delta)
	}
}

func (wb *WidgetBase) ComputeBBoxes(sc *Scene, parBBox image.Rectangle, delta image.Point) {
	wb.ComputeBBoxesParts(sc, parBBox, delta)
}

// PointToRelPos translates a point in Scene pixel coords
// into relative position within node
func (wb *WidgetBase) PointToRelPos(pt image.Point) image.Point {
	wb.BBoxMu.RLock()
	defer wb.BBoxMu.RUnlock()
	return pt.Sub(wb.ScBBox.Min)
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
func (wb *WidgetBase) GetSizeParts(sc *Scene, iter int) {
	wb.InitLayout(sc)
	if wb.Parts == nil {
		return
	}
	wb.Parts.GetSizeTree(sc, iter)
	wb.LayState.Alloc.Size = wb.Parts.LayState.Size.Pref // get from parts
	wb.GetSizeAddSpace()
	if LayoutTrace {
		fmt.Printf("Size:   %v size from parts: %v, parts pref: %v\n", wb.Path(), wb.LayState.Alloc.Size, wb.Parts.LayState.Size.Pref)
	}
}

func (wb *WidgetBase) GetSize(sc *Scene, iter int) {
	wb.GetSizeParts(sc, iter)
}

// NodeSize returns the size as a [mat32.Vec2] object.
func (wb *WidgetBase) NodeSize() mat32.Vec2 {
	return mat32.NewVec2FmPoint(wb.BBox.Size())
}

////////////////////////////////////////////////////////////////////
// 	DoLayout

func (wb *WidgetBase) DoLayoutParts(sc *Scene, parBBox image.Rectangle, iter int) {
	if wb.Parts == nil {
		return
	}
	spc := wb.BoxSpace()
	wb.Parts.LayState.Alloc.Pos = wb.LayState.Alloc.Pos.Add(spc.Pos())
	wb.Parts.LayState.Alloc.Size = wb.LayState.Alloc.Size.Sub(spc.Size())
	wb.Parts.DoLayout(sc, parBBox, iter)
}

func (wb *WidgetBase) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	wb.DoLayoutBase(sc, parBBox, true, iter) // init style
	wb.DoLayoutParts(sc, parBBox, iter)
	return wb.DoLayoutChildren(sc, iter)
}

func (wb *WidgetBase) InitLayout(sc *Scene) bool {
	wb.StyMu.Lock()
	defer wb.StyMu.Unlock()
	wb.LayState.SetFromStyle(&wb.Style)
	return false
}

// todo: wtf with initStyle ??

// DoLayoutBase provides basic DoLayout functions -- good for most cases
func (wb *WidgetBase) DoLayoutBase(sc *Scene, parBBox image.Rectangle, initStyle bool, iter int) {
	wi := wb.This().(Widget)
	psize := wb.AddParentPos()
	wb.LayState.Alloc.PosOrig = wb.LayState.Alloc.Pos
	if initStyle {
		SetUnitContext(&wb.Style, sc, wb.NodeSize(), psize) // update units with final layout
	}
	wb.BBox = wi.BBoxes() // only compute once, at this point
	// note: if other styles are maintained, they also need to be updated!
	wi.ComputeBBoxes(sc, parBBox, image.Point{}) // other bboxes from BBox
	if LayoutTrace {
		fmt.Printf("Layout: %v alloc pos: %v size: %v scbb: %v\n", wb.Path(), wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size, wb.ScBBox)
	}
	// typically DoLayoutChildren must be called after this!
}

// DoLayoutChildren does layout on all of node's children, giving them the
// ChildrenBBox -- default call at end of DoLayout.  Passes along whether
// any of the children need a re-layout -- typically DoLayout just returns
// this.
func (wb *WidgetBase) DoLayoutChildren(sc *Scene, iter int) bool {
	redo := false
	cbb := wb.This().(Widget).ChildrenBBoxes(sc)
	for _, kid := range wb.Kids {
		wi, _ := AsWidget(kid)
		if wi != nil {
			if wi.DoLayout(sc, cbb, iter) {
				redo = true
			}
		}
	}
	return redo
}

// ChildrenBBoxesWidget provides a basic widget box-model subtraction of
// margin and padding to children -- call in ChildrenBBoxes for most widgets
func (wb *WidgetBase) ChildrenBBoxesWidget(sc *Scene) image.Rectangle {
	bb := wb.ScBBox
	spc := wb.BoxSpace()
	bb.Min.X += int(spc.Left)
	bb.Min.Y += int(spc.Top)
	bb.Max.X -= int(spc.Right)
	bb.Max.Y -= int(spc.Bottom)
	return bb
}

func (wb *WidgetBase) ChildrenBBoxes(sc *Scene) image.Rectangle {
	return wb.ChildrenBBoxesWidget(sc)
}

//////////////////////////////////////////////////////////////////
//		Move2D scrolling

func (wb *WidgetBase) Move2D(sc *Scene, delta image.Point, parBBox image.Rectangle) {
	wb.Move2DBase(sc, delta, parBBox)
	if wb.Parts != nil {
		wb.Parts.This().(Widget).Move2D(sc, delta, parBBox)
	}
	wb.Move2DChildren(sc, delta)
}

// Move2DBase does the basic move on this node
func (wb *WidgetBase) Move2DBase(sc *Scene, delta image.Point, parBBox image.Rectangle) {
	wb.LayState.Alloc.Pos = wb.LayState.Alloc.PosOrig.Add(mat32.NewVec2FmPoint(delta))
	wb.This().(Widget).ComputeBBoxes(sc, parBBox, delta)
}

// Move2DTree does move2d pass -- each node iterates over children for maximum
// control -- this starts with parent ChildrenBBox and current delta -- can be
// called de novo
func (wb *WidgetBase) Move2DTree(sc *Scene) {
	parBBox := image.Rectangle{}
	pwi, pwb := AsWidget(wb.Par)
	if pwb != nil {
		parBBox = pwi.ChildrenBBoxes(sc)
	}
	delta := wb.LayState.Alloc.Pos.Sub(wb.LayState.Alloc.PosOrig).ToPoint()
	wb.This().(Widget).Move2D(sc, delta, parBBox) // important to use interface version to get interface!
}

// todo: move should just update bboxes with offset from parent
// passed down

// Move2DChildren moves all of node's children, giving them the ChildrenBBoxes
// -- default call at end of Move2D
func (wb *WidgetBase) Move2DChildren(sc *Scene, delta image.Point) {
	cbb := wb.This().(Widget).ChildrenBBoxes(sc)
	for _, kid := range wb.Kids {
		cwi, _ := AsWidget(kid)
		if cwi != nil {
			cwi.Move2D(sc, delta, cbb)
		}
	}
}

// ParentLayout returns the parent layout
func (wb *WidgetBase) ParentLayout() *Layout {
	ly := wb.ParentByType(LayoutType, ki.Embeds)
	if ly == nil {
		return nil
	}
	return AsLayout(ly)
}

// ParentScrollLayout returns the parent layout that has active scrollbars
func (wb *WidgetBase) ParentScrollLayout() *Layout {
	lyk := wb.ParentByType(LayoutType, ki.Embeds)
	if lyk == nil {
		return nil
	}
	ly := AsLayout(lyk)
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
	return ly.ScrollToItem(wb.This().(Widget))
}
