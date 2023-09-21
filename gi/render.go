// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/prof/v2"
)

// Rendering logic:
//
// Key principles:
//
// * Async updates (animation, mouse events, etc) change state, _set only flags_
//   using thread-safe atomic bitflag operations.  Actually rendering async (in V1)
//   is really hard to get right, and requires tons of mutexes etc.
// * Synchronous, full-tree render updates do the layout, rendering,
//   at regular FPS (frames-per-second) rate -- nop unless flag set.
// * Ki UpdateStart / End ensures that _only the highest changed node is flagged_,
//   while each individual state update uses the same Update wrapper calls locally,
//   so that rendering updates automatically happen at this highest common node.
// * UpdateStart starts naturally on the highest node driving a change, causing
//   a cascade of other UpdateStart on lower nodes, but the IsUpdating flag signals
//   that they are not the highest.  Only the highest calls UpdateEnd with true,
//   which is the point at which the change is flagged for render updating.
// * Thus, rendering updates skip any nodes with IsUpdating set, and are only
//   triggered at the highest UpdateEnd, so there shouldn't be conflicts
//   unless a node starts updating again before the next render hits.
//
// Three main steps:
// * Config: (re)configures widgets based on current params
//   typically by making Parts.  Always calls SetStyle.
// * Layout: does GetSize, DoLayout on tree, arranging widgets.
//   Needed for whole tree after any Config changes anywhere
//   (could contain at RenderAnchor nodes).
// * Render: just draws with current config, layout.
//
// SetStyle is always called after Config, and triggered after any
// current state of the Widget changes (e.g., a Hover started),
// by calling SetNeedsStyle(vp, updt) which sets the node NeedsStyle
// and VpNeedsUpdate flags, to drive the styling and rendering update
// at next DoUpdate call.
// If only rendering update is needed, call SetNeedsRender(vp, updt).
//
// For nodes with dynamic content that doesn't require styling or config
// a simple NeedsRender flag will drive re-rendering. UpdateSig does this.
//
// Updating is _always_ driven top-down by Window at FPS sampling rate,
// in the DoUpdate() call on the Viewport.
// Three types of updates can be triggered, in order of least impact
// and highest frequency first:
// * VpNeedsUpdate: does all NeedsStyle, NeedsRender on nodes.
// * VpNeedsLayout: does GetSize, DoLayout, then Render -- after Config.
// * VpNeedsRebuild: Config, Layout with DoRebuild flag set -- for a full
//   rebuild of the viewport (e.g., after global style changes, zooming, etc)
//
// Event handling, styling, etc updates should:
// * Wrap with UpdateStart / End
// * End with: SetNeedsStyle(vp, updt) if needs style updates needed based
//   on state change, or SetNeedsRender(vp, updt)
// * Or, if Config-level changes are needed, the Config(vp) must call
//   SetNeedsLayout(vp, updt) to trigger vp Layout step after.
//
// The one mutex that is still needed is a RWMutex on the BBbox fields
// because they are read by the event manager (and potentially inside
// event handler code) which does not have any flag protection,
// and are also read in rendering and written in Layout.

// UpdateSig just sets NeedsRender flag, in addition to sending
// the standard Ki update signal.  This will drive updating of
// the node on the next DoUpdate pass.
func (wb *WidgetBase) UpdateSig() {
	// note: we do not have the viewport here!!
	// this means we need to cache it..
	wb.SetNeedsRender(wb.Vp, true)
	wb.Node.UpdateSig()
}

// SetNeedsStyle sets the NeedsStyle and Viewport NeedsUpdate flags,
// if updt is true.
// This should be called after widget state changes,
// e.g., in event handlers or other update code,
// _after_ calling UpdateEnd(updt) and passing
// that same updt flag from UpdateStart.
func (wb *WidgetBase) SetNeedsStyle(vp *Viewport, updt bool) {
	if !updt {
		return
	}
	wb.SetFlag(true, NeedsStyle)
	vp.SetFlag(true, VpNeedsUpdate)
}

// SetNeedsRender sets the NeedsRender and Viewport NeedsUpdate flags,
// if updt is true.
// This should be called after widget state changes that don't need styling,
// e.g., in event handlers or other update code,
// _after_ calling UpdateEnd(updt) and passing
// that same updt flag from UpdateStart.
func (wb *WidgetBase) SetNeedsRender(vp *Viewport, updt bool) {
	if !updt {
		return
	}
	wb.SetFlag(true, NeedsStyle)
	vp.SetFlag(true, VpNeedsUpdate)
}

// ConfigTree calls Config on every Widget in the tree from me.
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

// SetStyleTree calls SetStyle on every Widget in the tree from me.
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

// GetSizeTree does the sizing as a depth-first pass from me,
// needed for Layout stack.
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

// DoLayoutTree does layout pass for tree from me.
// Each node iterates over children for maximum control,
// Starting with parent VpBBox.
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
	} else {
		parBBox = vp.Pixels.Bounds()
	}
	wi := wb.This().(Widget)
	redo := wi.DoLayout(vp, parBBox, 0) // important to use interface version to get interface!
	if redo {
		if LayoutTrace {
			fmt.Printf("Layout: ----------  Redo: %v ----------- \n", wi.Path())
		}
		la := wb.LayState.Alloc
		wb.GetSizeTree(vp, 1)
		wb.LayState.Alloc = la
		wi.DoLayout(vp, parBBox, 1) // todo: multiple iters?
	}
	pr.End()
}

// LayoutRenderTree does a layout and render of the tree from me:
// GetSize, DoLayout, Render.  Needed after Config.
func (wb *WidgetBase) LayoutRenderTree(vp *Viewport) {
	wb.GetSizeTree(vp, 0)
	wb.DoLayoutTree(vp)
	wb.Render(vp)
}

// DoUpdateTree calls SetStyle and / or Render on tree from me
// with NeedsStyle or NeedsRender flags set
func (wb *WidgetBase) DoUpdate(vp *Viewport) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.DoUpdate." + wb.Type().Name())
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		wi, w := AsWidget(k)
		if w == nil || w.IsDeleted() || w.IsDestroyed() {
			return ki.Break
		}
		if w.HasFlag(NeedsStyle) && !w.IsUpdating() {
			w.SetFlag(false, NeedsStyle, NeedsRender)
			wi.SetStyleTree(vp) // everybody under me needs restyled
			wi.Render(vp)
			return ki.Break // done
		}
		if w.HasFlag(NeedsRender) && !w.IsUpdating() {
			w.SetFlag(false, NeedsRender)
			wi.Render(vp)
			return ki.Break // done
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

// todo: when?
func (vp *Viewport) SetMyStyle() {
	vp.Frame.Style.BackgroundColor.SetSolid(ColorScheme.Background)
	vp.Frame.Style.Color = ColorScheme.OnBackground
}

// DoUpdate checks viewport Needs flags to do whatever updating is required.
// returns false if already updating.
// This is the main update call made by the Window at FPS frequency.
func (vp *Viewport) DoUpdate() bool {
	if vp.HasFlag(VpIsUpdating) {
		return
	}
	vp.SetFlag(true, VpIsUpdating) // prevent rendering
	defer vp.SetFlag(false, VpIsUpdating)

	switch {
	case vp.HasFlag(VpNeedsRebuild):
		vp.SetFlag(false, VpNeedsLayout, VpNeedsUpdate, VpNeedsRebuild)
		vp.DoRebuild()
	case vp.HasFlag(VpNeedsLayout):
		vp.SetFlag(false, VpNeedsLayout, VpNeedsUpdate)
		vp.Frame.LayoutRenderTree(vp)
	case vp.HasFlag(VpNeedsUpdate):
		vp.SetFlag(false, VpNeedsUpdate)
		vp.Frame.DoUpdate(vp)
	}
	return true
}

// Config calls Config on all nodes in the tree,
// which will set NeedsLayout to drive subsequent layout and render.
// This is a top-level call, typically only done in a Show function.
func (vp *Viewport) Config() {
	vp.SetFlag(true, VpIsUpdating) // prevent rendering
	defer vp.SetFlag(false, VpIsUpdating)
	vp.Frame.ConfigTree(vp)
}

// todo: Probably don't want this anymore - Frame should handle
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
	vp.SetFlag(true, VpIsUpdating) // prevent rendering
	defer vp.SetFlag(false, VpIsUpdating)

	vp.SetFlag(true, VpPrefSizing)
	vp.Config()

	vp.Frame.SetStyleTree(vp) // sufficient to get sizes
	vp.LayState.Alloc.Size.SetPoint(initSz)
	vp.Frame.GetSizeTree(vp, 0) // collect sizes

	vp.SetFlag(false, VpPrefSizing)

	vpsz := vp.Frame.LayState.Size.Pref.ToPoint()
	// also take into account min size pref
	stw := int(vp.Style.MinWidth.Dots)
	sth := int(vp.Style.MinHeight.Dots)
	// fmt.Printf("dlg stw %v sth %v dpi %v vpsz: %v\n", stw, sth, dlg.Sty.UnContext.DPI, vpsz)
	vpsz.X = max(vpsz.X, stw)
	vpsz.Y = max(vpsz.Y, sth)
	return vpsz
}
