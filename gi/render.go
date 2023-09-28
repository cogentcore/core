// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"goki.dev/girl/girl"
	"goki.dev/girl/gist"
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
// SetStyle is always called after Config, and after any
// current state of the Widget changes via events, animations, etc
// (e.g., a Hover started or a Button is pushed down).
// These changes should be protected by UpdateStart / End,
// such that SetStyle is only ever called within that scope.
// After the UpdateEnd(updt), call SetNeedsRender(vp, updt)
// which sets the node NeedsRender and ScNeedsRender flags,
// to drive the rendering update at next DoNeedsRender call.
//
// Because Render checks for IsUpdating() flag, and doesn't render
// if so, it should never be the case that a node is being modified
// and rendered at the same time, avoiding need for mutexes.
//
// For nodes with dynamic content that doesn't require styling or config
// a simple SetNeedsRender call will drive re-rendering. UpdateSig does this.
//
// Updating is _always_ driven top-down by RenderWin at FPS sampling rate,
// in the DoUpdate() call on the Scene.
// Three types of updates can be triggered, in order of least impact
// and highest frequency first:
// * ScNeedsRender: does NeedsRender on nodes.
// * ScNeedsLayout: does GetSize, DoLayout, then Render -- after Config.
// * ScNeedsRebuild: Config, Layout with DoRebuild flag set -- for a full
//   rebuild of the scene (e.g., after global style changes, zooming, etc)
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
//
// ki Signals in general should not be used

// UpdateSig just sets NeedsRender flag, in addition to sending
// the standard Ki update signal.  This will drive updating of
// the node on the next DoNUpdate pass.
func (wb *WidgetBase) UpdateSig() bool {
	// note: we do not have the scene here!!
	// this means we need to cache it..
	wb.SetNeedsRender(wb.Sc, true)
	return wb.Node.UpdateSig()
}

// SetNeedsRender sets the NeedsRender and Scene NeedsRender flags,
// if updt is true.  See UpdateEndRender for convenience method.
// This should be called after widget state changes that don't need styling,
// e.g., in event handlers or other update code,
// _after_ calling UpdateEnd(updt) and passing
// that same updt flag from UpdateStart.
func (wb *WidgetBase) SetNeedsRender(sc *Scene, updt bool) {
	if !updt {
		return
	}
	wb.SetFlag(true, NeedsRender)
	sc.SetFlag(true, ScNeedsRender)
}

// UpdateEndRender should be called instead of UpdateEnd
// for any UpdateStart / UpdateEnd block that needs a re-render
// at the end.  Just does SetNeedsRender after UpdateEnd,
// and uses the cached wb.Sc pointer.
func (wb *WidgetBase) UpdateEndRender(updt bool) {
	if !updt {
		return
	}
	wb.UpdateEnd(updt)
	wb.SetNeedsRender(wb.Sc, updt)
}

// note: this is replacement for "SetNeedsFullReRender()" call:

// SetNeedsLayout sets the ScNeedsLayout flag if updt is true.
// See UpdateEndLayout for convenience method.
// This should be called after widget Config call
// _after_ calling UpdateEnd(updt) and passing
// that same updt flag from UpdateStart.
func (wb *WidgetBase) SetNeedsLayout(sc *Scene, updt bool) {
	if !updt || sc == nil {
		return
	}
	sc.SetFlag(true, ScNeedsLayout)
}

// UpdateEndLayout should be called instead of UpdateEnd
// for any UpdateStart / UpdateEnd block that needs a re-layout
// at the end.  Just does SetNeedsLayout after UpdateEnd,
// and uses the cached wb.Sc pointer.
func (wb *WidgetBase) UpdateEndLayout(updt bool) {
	if !updt {
		return
	}
	wb.UpdateEnd(updt)
	wb.SetNeedsLayout(wb.Sc, updt)
}

// ConfigTree calls Config on every Widget in the tree from me.
// Config automatically calls SetStyle.
func (wb *WidgetBase) ConfigTree(sc *Scene) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.ConfigTree." + wb.KiType().Name)
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		wi, w := AsWidget(k)
		if w == nil || w.IsDeleted() || w.IsDestroyed() {
			return ki.Break
		}
		wi.Config(sc)
		return ki.Continue
	})
	pr.End()
}

// SetStyleTree calls SetStyle on every Widget in the tree from me.
// Called during FullRender
func (wb *WidgetBase) SetStyleTree(sc *Scene) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.SetStyleTree." + wb.KiType().Name)
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		wi, w := AsWidget(k)
		if w == nil || w.IsDeleted() || w.IsDestroyed() {
			return ki.Break
		}
		wi.SetStyle(sc)
		return ki.Continue
	})
	pr.End()
}

// GetSizeTree does the sizing as a depth-first pass from me,
// needed for Layout stack.
func (wb *WidgetBase) GetSizeTree(sc *Scene, iter int) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.GetSizeTree." + wb.KiType().Name)
	wb.FuncDownMeLast(0, wb.This(),
		func(k ki.Ki, level int, d any) bool { // tests whether to process node
			_, w := AsWidget(k)
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
			wi.GetSize(sc, iter)
			return ki.Continue
		})
	pr.End()
}

// DoLayoutTree does layout pass for tree from me.
// Each node iterates over children for maximum control,
// Starting with parent ScBBox.
// Handles multiple iterations if needed.
func (wb *WidgetBase) DoLayoutTree(sc *Scene) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("WidgetBase.DoLayoutTree." + wb.KiType().Name)
	parBBox := image.Rectangle{}
	pwi, _ := AsWidget(wb.Par)
	if pwi != nil {
		parBBox = pwi.ChildrenBBoxes(sc)
	} else {
		parBBox = sc.Pixels.Bounds()
	}
	wi := wb.This().(Widget)
	redo := wi.DoLayout(sc, parBBox, 0) // important to use interface version to get interface!
	if redo {
		if LayoutTrace {
			fmt.Printf("Layout: ----------  Redo: %v ----------- \n", wi.Path())
		}
		la := wb.LayState.Alloc
		wb.GetSizeTree(sc, 1)
		wb.LayState.Alloc = la
		wi.DoLayout(sc, parBBox, 1) // todo: multiple iters?
	}
	pr.End()
}

// LayoutRenderTree does a layout and render of the tree:
// GetSize, DoLayout, Render.  Needed after Config.
func (sc *Scene) LayoutRenderTree() {
	sc.Frame.GetSizeTree(sc, 0)
	sc.Frame.LayState.Alloc.Size = mat32.NewVec2FmPoint(sc.Geom.Size)
	sc.Frame.DoLayoutTree(sc)
	sc.Frame.Render(sc)
}

// DoNeedsRender calls Render on tree from me for nodes
// with NeedsRender flags set
func (wb *WidgetBase) DoNeedsRender(sc *Scene) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.DoNeedsRender." + wb.KiType().Name)
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		wi, w := AsWidget(k)
		if w == nil || w.IsDeleted() || w.IsDestroyed() {
			return ki.Break
		}
		if w.HasFlag(NeedsRender) && !w.IsUpdating() {
			w.SetFlag(false, NeedsRender)
			wi.Render(sc)
			return ki.Break // done
		}
		return ki.Continue
	})
	pr.End()
}

//////////////////////////////////////////////////////////////////
//		Scene

// DoUpdate checks scene Needs flags to do whatever updating is required.
// returns false if already updating.
// This is the main update call made by the RenderWin at FPS frequency.
func (sc *Scene) DoUpdate() bool {
	if sc.HasFlag(ScIsUpdating) {
		fmt.Println("scene bail on updt")
		return false
	}
	sc.SetFlag(true, ScIsUpdating) // prevent rendering
	defer sc.SetFlag(false, ScIsUpdating)

	switch {
	case sc.HasFlag(ScNeedsRebuild):
		sc.SetFlag(false, ScNeedsLayout, ScNeedsRender, ScNeedsRebuild)
		sc.DoRebuild()
		sc.SetFlag(true, ScImageUpdated)
	case sc.HasFlag(ScNeedsLayout):
		// fmt.Println("scene layout start")
		sc.SetFlag(false, ScNeedsLayout, ScNeedsRender)
		sc.Fill() // full redraw
		sc.LayoutRenderTree()
		sc.SetFlag(true, ScImageUpdated)
		// fmt.Println("scene layout done")
	case sc.HasFlag(ScNeedsRender):
		// fmt.Println("scene render start")
		sc.SetFlag(false, ScNeedsRender)
		sc.Frame.DoNeedsRender(sc)
		sc.SetFlag(true, ScImageUpdated)
		// fmt.Println("scene render done")
	}
	return true
}

// Config calls Config on all nodes in the tree,
// which will set NeedsLayout to drive subsequent layout and render.
// This is a top-level call, typically only done in a Show function.
func (sc *Scene) Config() {
	sc.SetFlag(true, ScIsUpdating) // prevent rendering
	defer sc.SetFlag(false, ScIsUpdating)
	sc.Frame.ConfigTree(sc)
}

// DoRebuild implements the ScNeedsRebuild case
// Typically not called otherwise, and assumes ScIsUpdating already set.
func (sc *Scene) DoRebuild() {
	sc.Fill() // full redraw
	sc.SetFlag(true, ScRebuild)
	sc.Frame.ConfigTree(sc)
	sc.LayoutRenderTree()
	sc.SetFlag(false, ScRebuild)
}

// Fill fills the scene with BgColor (default transparent)
// which is the starting base level for rendering.
// Typically the root Frame fills its background with color
// but it can e.g., leave corners transparent for popups etc.
func (sc *Scene) Fill() {
	rs := &sc.RenderState
	rs.Lock()
	rs.Paint.FillBox(rs, mat32.Vec2Zero, mat32.NewVec2FmPoint(sc.Geom.Size), &sc.BgColor)
	rs.Unlock()
}

// PrefSize computes the preferred size of the scene based on current contents.
// initSz is the initial size -- e.g., size of screen.
// Used for auto-sizing windows.
func (sc *Scene) PrefSize(initSz image.Point) image.Point {
	sc.SetFlag(true, ScIsUpdating) // prevent rendering
	defer sc.SetFlag(false, ScIsUpdating)

	sc.SetFlag(true, ScPrefSizing)
	sc.Config()

	frame := &sc.Frame
	frame.SetStyleTree(sc) // sufficient to get sizes
	frame.LayState.Alloc.Size.SetPoint(initSz)
	frame.GetSizeTree(sc, 0) // collect sizes

	sc.SetFlag(false, ScPrefSizing)

	vpsz := frame.LayState.Size.Pref.ToPoint()
	// also take into account min size pref
	stw := int(frame.Style.MinWidth.Dots)
	sth := int(frame.Style.MinHeight.Dots)
	// fmt.Printf("dlg stw %v sth %v dpi %v vpsz: %v\n", stw, sth, dlg.Sty.UnContext.DPI, vpsz)
	vpsz.X = max(vpsz.X, stw)
	vpsz.Y = max(vpsz.Y, sth)
	return vpsz
}

//////////////////////////////////////////////////////////////////
//		Widget local rendering

// PushBounds pushes our bounding-box bounds onto the bounds stack if non-empty
// -- this limits our drawing to our own bounding box, automatically -- must
// be called as first step in Render returns whether the new bounds are
// empty or not -- if empty then don't render!
func (wb *WidgetBase) PushBounds(sc *Scene) bool {
	if wb == nil || wb.This() == nil {
		return false
	}
	if !wb.This().(Widget).IsVisible() {
		return false
	}
	if wb.ScBBox.Empty() {
		return false
	}
	rs := &sc.RenderState
	rs.PushBounds(wb.ScBBox)
	if RenderTrace {
		fmt.Printf("Render: %v at %v\n", wb.Path(), wb.ScBBox)
	}
	return true
}

// PopBounds pops our bounding-box bounds -- last step in Render after
// rendering children
func (wb *WidgetBase) PopBounds(sc *Scene) {
	if wb.IsDeleted() || wb.IsDestroyed() || wb.This() == nil {
		return
	}
	rs := &sc.RenderState
	rs.PopBounds()
}

func (wb *WidgetBase) Render(sc *Scene) {
	wi := wb.This().(Widget)
	if wb.PushBounds(sc) {
		wi.FilterEvents()
		wb.RenderParts(sc)
		wb.RenderChildren(sc)
		wb.PopBounds(sc)
	}
}

func (wb *WidgetBase) RenderParts(sc *Scene) {
	if wb.Parts == nil {
		return
	}
	wb.Parts.Render(sc) // is a layout, will do all
}

// RenderChildren renders all of node's children,
// This is the default call at end of Render()
func (wb *WidgetBase) RenderChildren(sc *Scene) {
	for _, kid := range wb.Kids {
		wi, w := AsWidget(kid)
		if w == nil || w.IsDeleted() || w.IsDestroyed() || w.IsUpdating() {
			continue
		}
		wi.Render(sc)
	}
}

/* todo: anything needed here?

// ReRenderTree does a re-render of the tree -- after it has already been
// initialized and styled -- redoes the full stack
func (wb *WidgetBase) ReRenderTree() {
	parBBox := image.Rectangle{}
	pni, _ := KiToWidget(wb.Par)
	if pni != nil {
		parBBox = pni.ChildrenBBoxes(vp)
	}
	delta := wb.LayState.Alloc.Pos.Sub(wb.LayState.Alloc.PosOrig)
	wb.LayState.Alloc.Pos = wb.LayState.Alloc.PosOrig
	ld := wb.LayState // save our current layout data
	updt := wb.UpdateStart()
	wb.ConfigTree()
	wb.SetStyleTree()
	wb.GetSizeTree(0)
	wb.LayState = ld // restore
	wb.DoLayoutTree()
	if !delta.IsNil() {
		wb.Move2D(delta.ToPointFloor(), parBBox)
	}
	wb.RenderTree()
	wb.UpdateEndNoSig(updt)
}
*/

////////////////////////////////////////////////////////////////////////////////
//  Standard Box Model rendering

// RenderLock returns the locked girl.State, Paint, and Style with StyMu locked.
// This should be called at start of widget-level rendering.
func (wb *WidgetBase) RenderLock(sc *Scene) (*girl.State, *girl.Paint, *gist.Style) {
	wb.StyMu.RLock()
	rs := &sc.RenderState
	rs.Lock()
	return rs, &rs.Paint, &wb.Style
}

// RenderUnlock unlocks girl.State and style
func (wb *WidgetBase) RenderUnlock(rs *girl.State) {
	rs.Unlock()
	wb.StyMu.RUnlock()
}

// RenderBoxImpl implements the standard box model rendering -- assumes all
// paint params have already been set
func (wb *WidgetBase) RenderBoxImpl(sc *Scene, pos mat32.Vec2, sz mat32.Vec2, bs gist.Border) {
	rs := &sc.RenderState
	pc := &rs.Paint
	pc.DrawBox(rs, pos, sz, bs)
}

// RenderStdBox draws standard box using given style.
// girl.State and Style must already be locked at this point (RenderLock)
func (wb *WidgetBase) RenderStdBox(sc *Scene, st *gist.Style) {
	// SidesTODO: this is a pretty critical function, so a good place to look if things aren't working
	wb.StyMu.RLock()
	defer wb.StyMu.RUnlock()

	rs := &sc.RenderState
	pc := &rs.Paint

	csp := wb.ParentBackgroundColor()
	pc.DrawStdBox(rs, st, wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size, &csp)
}

// ParentReRenderAnchor returns parent (including this node)
// that is a ReRenderAnchor -- for optimized re-rendering
func (wb *WidgetBase) ParentReRenderAnchor() Widget {
	var par Widget
	wb.FuncUp(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		wi, w := AsWidget(k)
		if w == nil {
			return ki.Break // don't keep going up
		}
		if w.HasFlag(ReRenderAnchor) {
			par = wi
			return ki.Break
		}
		return ki.Continue
	})
	return par
}
