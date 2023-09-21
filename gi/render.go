// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"goki.dev/ki/v2"
	"goki.dev/prof/v2"
)

// ConfigTree calls Config on every Widget in the tree from this node down.
// Config automatically calls SetStyle.
func (wb *WidgetBase) ConfigTree(vp *Viewport) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.ConfigTree." + wb.Type().Name())
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		wi, w := AsWidget(k)
		if w == nil || w.IsDeleted() || w.IsDestroyed() {
			return ki.Break
		}
		wi.Config(vp)
		return ki.Continue
	})
	pr.End()
}

// SetStyleTree calls SetStyle on every Widget in the tree from this node down.
// Called during FullRender
func (wb *WidgetBase) SetStyleTree(vp *Viewport) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.SetStyleTree." + wb.Type().Name())
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		wi, w := AsWidget(k)
		if w == nil || w.IsDeleted() || w.IsDestroyed() {
			return ki.Break
		}
		wi.SetStyle(vp)
		return ki.Continue
	})
	pr.End()
}

// GetSizeTree does the sizing as a depth-first pass
func (wb *WidgetBase) GetSizeTree(vp *Viewport, iter int) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.GetSizeTree." + wb.Type().Name())
	wb.FuncDownMeLast(0, wb.This(),
		func(k ki.Ki, level int, d any) bool { // tests whether to process node
			wi, w := AsWidget(k)
			if w == nil || w.IsDeleted() || w.IsDestroyed() {
				return ki.Break
			}
			return ki.Continue
		},
		func(k ki.Ki, level int, d any) bool { // this one does the work
			wi, w := AsWidget(k)
			if w == nil || w.IsDeleted() || w.IsDestroyed() {
				return ki.Break
			}
			wi.GetSize(vp, iter)
			return ki.Continue
		})
	pr.End()
}

// DoLayoutTree does layout pass -- each node iterates over children for
// maximum control -- this starts with parent VpBBox -- can be called de novo.
// Handles multiple iterations if needed.
func (wb *WidgetBase) DoLayoutTree(vp *Viewport) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("WidgetBase.DoLayoutTree." + wb.Type().Name())
	parBBox := image.Rectangle{}
	pwi, _ := AsWidget(wb.Par)
	if pwi != nil {
		parBBox = pwi.ChildrenBBox2D()
	}
	wi := wb.This().(Widget)
	redo := wi.DoLayout(vp, parBBox, 0) // important to use interface version to get interface!
	if redo {
		if LayoutTrace {
			fmt.Printf("Layout: ----------  Redo: %v ----------- \n", nbi.Path())
		}
		la := wb.LayState.Alloc
		wb.GetSizeTree(vp, 1)
		wb.LayState.Alloc = la
		wi.DoLayout(vp, parBBox, 1) // todo: multiple iters?
	}
	pr.End()
}

// FullRenderTree does a full render of the tree:
// SetStyle, GetSize, DoLayout, Render
func (wb *WidgetBase) FullRenderTree(vp *Viewport) {
	wb.SetStyleTree(vp)
	wb.GetSizeDTree(vp, 0)
	wb.DoLayoutTree(vp)
	wb.Render(vp)
}

// RenderUpdate calls Style and / or Render on nodes
// with NeedsRender flag set
func (wb *WidgetBase) RenderUpdate(vp *Viewport) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.RenderUpdate." + wb.Type().Name())
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		wi, w := AsWidget(k)
		if w == nil || w.IsDeleted() || w.IsDestroyed() {
			return ki.Break
		}
		if w.HasFlag(NeedsStyle) {
			w.SetFlag(false, NeedsStyle, NeedsRender)
			wi.SetStyle(vp)
			wi.Render(vp)
			return ki.Continue
		}
		if w.HasFlag(NeedsRender) {
			w.SetFlag(false, NeedsRender)
			wi.Render(vp)
			return ki.Continue
		}
		return ki.Continue
	})
	pr.End()
}

// todo: move should just update bboxes with offset from parent
// passed down

// Move2DChildren moves all of node's children, giving them the ChildrenBBox2D
// -- default call at end of Move2D
func (wb *WidgetBase) Move2DChildren(delta image.Point) {
	cbb := wb.This().(Node2D).ChildrenBBox2D()
	for _, kid := range wb.Kids {
		nii, _ := KiToNode2D(kid)
		if nii != nil {
			nii.Move2D(delta, cbb)
		}
	}
}

// RenderChildren renders all of node's children,
// This is the default call at end of Render()
func (wb *WidgetBase) RenderChildren(vp *Viewport) {
	for _, kid := range wb.Kids {
		wi, _ := AsWidget(kid)
		if wi != nil {
			wi.Render(vp)
		}
	}
}

//////////////////////////////////////////////////////////////////
//		Viewport

func (vp *Viewport) Style() {
	vp.Frame.Style.BackgroundColor.SetSolid(ColorScheme.Background)
	vp.Frame.Style.Color = ColorScheme.OnBackground
}

// DoRenderUpdate checks flags to do whatever updating is required
func (vp *Viewport) DoRenderUpdate() {
	vp.SetFlag(true, VpIsRendering) // prevent rendering
	defer vp.SetFlag(false, VpIsRendering)

	switch {
	case vp.HasFlag(VpNeedsFullRender):
		vp.SetFlag(false, VpNeedsFullRender, VpNeedsRender)
		vp.Frame.FullRenderTree(vp)
	case vp.HasFlag(VpNeedsRender):
		vp.SetFlag(false, VpNeedsRender)
		vp.Frame.RenderUpdate(vp)
	}
}

// Config does configuration on the tree,
// and sets VpNeedsFullRender to drive subsequent rendering.
func (vp *Viewport) Config() {
	vp.SetFlag(true, VpIsRendering) // prevent rendering
	defer vp.SetFlag(false, VpIsRendering)

	vp.Frame.ConfigTree(vp)
	vp.SetFlag(true, VpNeedsFullRender)
}

// FullRender does full render
func (vp *Viewport) FullRender() {
	vp.SetFlag(true, VpIsRendering) // prevent rendering
	defer vp.SetFlag(false, VpIsRendering)

	vp.SetFlag(false, VpNeedsFullRender, VpNeedsRender)
	vp.Frame.FullRenderTree(vp)
}

func (vp *Viewport) FillViewport() {
	vp.StyMu.RLock()
	st := &vp.Style
	rs := &vp.Render
	rs.Lock()
	rs.Paint.FillBox(rs, mat32.Vec2Zero, mat32.NewVec2FmPoint(vp.Geom.Size), &st.BackgroundColor)
	rs.Unlock()
	vp.StyMu.RUnlock()
}

// PrefSize computes the preferred size of the viewport based on current contents.
// initSz is the initial size -- e.g., size of screen.
// Used for auto-sizing windows.
func (vp *Viewport) PrefSize(initSz image.Point) image.Point {
	vp.SetFlag(true, VpIsRendering) // prevent rendering
	defer vp.SetFlag(false, VpIsRendering)

	vp.SetFlag(true, VpPrefSizing)
	vp.Config()

	vp.Frame.SetStyleTree(vp) // sufficient to get sizes
	vp.LayState.Alloc.Size.SetPoint(initSz)
	vp.Frame.GetSizeTree(vp, 0) // collect sizes

	vp.ClearFlag(int(VpPrefSizing))

	vpsz := vp.Frame.LayState.Size.Pref.ToPoint()
	// also take into account min size pref
	stw := int(vp.Style.MinWidth.Dots)
	sth := int(vp.Style.MinHeight.Dots)
	// fmt.Printf("dlg stw %v sth %v dpi %v vpsz: %v\n", stw, sth, dlg.Sty.UnContext.DPI, vpsz)
	vpsz.X = max(vpsz.X, stw)
	vpsz.Y = max(vpsz.Y, sth)
	return vpsz
}
