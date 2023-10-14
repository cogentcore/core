// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"log/slog"

	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

////////////////////////////////////////////////////////////////////
// 	BBox bounding boxes

// HasSc checks that the Sc Scene has been set.
// Called prior to using -- logs an error if not.
// todo: need slog Debug mode for this kind of thing.
func (wb *WidgetBase) HasSc() bool {
	if wb.This() == nil || wb.Sc == nil {
		log.Printf("gi.WidgetBase: object or scene is nil\n") // todo: slog.Debug
		return false
	}
	return true
}

// ReConfig is a convenience method for reconfiguring a widget after changes
// have been made.  In general it is more efficient to call Set* methods that
// automatically determine if Config is needed.
// The plain Config method is used during initial configuration,
// called by the Scene and caches the Sc pointer.
func (wb *WidgetBase) ReConfig() {
	if !wb.HasSc() {
		return
	}
	wi := wb.This().(Widget)
	wi.Config(wb.Sc)
	wi.ApplyStyle(wb.Sc)
}

func (wb *WidgetBase) Config(sc *Scene) {
	if wb.This() == nil {
		slog.Error("nil this in config")
		return
	}
	wi := wb.This().(Widget)
	updt := wi.UpdateStart()
	wb.Sc = sc
	wi.ConfigWidget(sc) // where everything actually happens
	wb.UpdateEnd(updt)
	wb.SetNeedsLayoutUpdate(sc, updt)
}

func (wb *WidgetBase) ConfigWidget(sc *Scene) {
	// this must be defined for each widget type
}

// ConfigPartsImpl initializes the parts of the widget if they
// are not already through [WidgetBase.NewParts], calls
// [ki.Node.ConfigChildren] on those parts with the given config,
// and then handles necessary updating logic with the given scene.
func (wb *WidgetBase) ConfigPartsImpl(sc *Scene, config ki.Config, lay Layouts) {
	parts := wb.NewParts(lay)
	mods, updt := parts.ConfigChildren(config)
	if !mods && !wb.NeedsRebuild() {
		parts.UpdateEnd(updt)
		return
	}
	parts.UpdateEnd(updt)
	wb.SetNeedsLayoutUpdate(sc, updt)
}

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
	wb.WalkPre(func(k ki.Ki) bool {
		nii, ni := AsWidget(k)
		if nii == nil || ni.Is(ki.Deleted) || ni.Is(ki.Destroyed) {
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
		if !wb.Is(ki.Field) {
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
	wb.InitLayout(sc)
	wb.GetSizeFromWH(2, 2)
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
	wb.DoLayoutBase(sc, parBBox, iter)
	wb.DoLayoutParts(sc, parBBox, iter)
	return wb.DoLayoutChildren(sc, iter)
}

func (wb *WidgetBase) InitLayout(sc *Scene) bool {
	wb.StyMu.Lock()
	defer wb.StyMu.Unlock()
	wb.LayState.SetFromStyle(&wb.Style)
	return false
}

// todo: wtf with initStyle ??  always true.

// DoLayoutBase provides basic DoLayout functions -- good for most cases
func (wb *WidgetBase) DoLayoutBase(sc *Scene, parBBox image.Rectangle, iter int) {
	if sc == nil {
		sc = wb.Sc
	} else if wb.Sc == nil {
		wb.Sc = sc
	}
	if wb.Sc == nil {
		slog.Error("DoLayoutBase Scene is nil", "widget", wb)
	}
	wi := wb.This().(Widget)
	psize := wb.AddParentPos()
	wb.LayState.Alloc.PosOrig = wb.LayState.Alloc.Pos
	// this is the one point when Style Dots are actually computed!
	SetUnitContext(&wb.Style, sc, wb.NodeSize(), psize) // update units with final layout
	wb.BBox = wi.BBoxes()                               // only compute once, at this point
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
//		LayoutScroll

func (wb *WidgetBase) LayoutScroll(sc *Scene, delta image.Point, parBBox image.Rectangle) {
	wb.LayoutScrollBase(sc, delta, parBBox)
	if wb.Parts != nil {
		wb.Parts.This().(Widget).LayoutScroll(sc, delta, parBBox)
	}
	wb.LayoutScrollChildren(sc, delta)
}

// LayoutScrollBase does the basic move on this node
func (wb *WidgetBase) LayoutScrollBase(sc *Scene, delta image.Point, parBBox image.Rectangle) {
	wb.LayState.Alloc.Pos = wb.LayState.Alloc.PosOrig.Add(mat32.NewVec2FmPoint(delta))
	wb.This().(Widget).ComputeBBoxes(sc, parBBox, delta)
}

// LayoutScrollTree does move2d pass -- each node iterates over children for maximum
// control -- this starts with parent ChildrenBBox and current delta -- can be
// called de novo
func (wb *WidgetBase) LayoutScrollTree(sc *Scene) {
	parBBox := image.Rectangle{}
	pwi, pwb := AsWidget(wb.Par)
	if pwb != nil {
		parBBox = pwi.ChildrenBBoxes(sc)
	} else {
		parBBox.Max = sc.RenderCtx().Size
	}
	delta := wb.LayState.Alloc.Pos.Sub(wb.LayState.Alloc.PosOrig).ToPoint()
	wb.This().(Widget).LayoutScroll(sc, delta, parBBox) // important to use interface version to get interface!
}

// LayoutScrollChildren moves all of node's children, giving them the ChildrenBBoxes
// -- default call at end of LayoutScroll
func (wb *WidgetBase) LayoutScrollChildren(sc *Scene, delta image.Point) {
	cbb := wb.This().(Widget).ChildrenBBoxes(sc)
	for _, kid := range wb.Kids {
		cwi, _ := AsWidget(kid)
		if cwi != nil {
			cwi.LayoutScroll(sc, delta, cbb)
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

//////////////////////////////////////////////////////////////////
//		Widget position functions

// WinBBox returns the RenderWin based bounding box for the widget
// by adding the Scene position to the ScBBox
func (wb *WidgetBase) WinBBox() image.Rectangle {
	if !wb.HasSc() {
		return wb.ScBBox
	}
	return wb.ScBBox.Add(wb.Sc.Geom.Pos)
}

// WinPos returns the RenderWin based position within the
// bounding box of the widget, where the x, y coordinates
// are the proportion across the bounding box to use:
// 0 = left / top, 1 = right / bottom
func (wb *WidgetBase) WinPos(x, y float32) image.Point {
	bb := wb.WinBBox()
	sz := bb.Size()
	var pt image.Point
	pt.X = bb.Min.X + int(mat32.Round(float32(sz.X)*x))
	pt.Y = bb.Min.Y + int(mat32.Round(float32(sz.Y)*y))
	return pt
}
