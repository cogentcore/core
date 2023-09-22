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
// which sets the node NeedsRender and VpNeedsRender flags,
// to drive the rendering update at next DoNeedsRender call.
//
// Because Render checks for IsUpdating() flag, and doesn't render
// if so, it should never be the case that a node is being modified
// and rendered at the same time, avoiding need for mutexes.
//
// For nodes with dynamic content that doesn't require styling or config
// a simple SetNeedsRender call will drive re-rendering. UpdateSig does this.
//
// Updating is _always_ driven top-down by Window at FPS sampling rate,
// in the DoUpdate() call on the Viewport.
// Three types of updates can be triggered, in order of least impact
// and highest frequency first:
// * VpNeedsRender: does NeedsRender on nodes.
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
//
// ki Signals in general should not be used

// UpdateSig just sets NeedsRender flag, in addition to sending
// the standard Ki update signal.  This will drive updating of
// the node on the next DoNUpdate pass.
func (wb *WidgetBase) UpdateSig() bool {
	// note: we do not have the viewport here!!
	// this means we need to cache it..
	wb.SetNeedsRender(wb.Vp, true)
	return wb.Node.UpdateSig()
}

// SetNeedsStyle sets the NeedsStyle and Viewport NeedsRender flags,
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
	vp.SetFlag(true, VpNeedsRender)
}

// SetNeedsRender sets the NeedsRender and Viewport NeedsRender flags,
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
	vp.SetFlag(true, VpNeedsRender)
}

// SetNeedsLayout sets the VpNeedsLayout flag if updt is true.
// This should be called after widget Config call
// _after_ calling UpdateEnd(updt) and passing
// that same updt flag from UpdateStart.
func (wb *WidgetBase) SetNeedsLayout(vp *Viewport, updt bool) {
	if !updt {
		return
	}
	wb.SetFlag(true, NeedsStyle)
	vp.SetFlag(true, VpNeedsRender)
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

// DoNeedsRender calls Render on tree from me for nodes
// with NeedsRender flags set
func (wb *WidgetBase) DoNeedsRender(vp *Viewport) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.DoNeedsRender." + wb.Type().Name())
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		wi, w := AsWidget(k)
		if w == nil || w.IsDeleted() || w.IsDestroyed() {
			return ki.Break
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

//////////////////////////////////////////////////////////////////
//		Viewport

// DoUpdate checks viewport Needs flags to do whatever updating is required.
// returns false if already updating.
// This is the main update call made by the Window at FPS frequency.
func (vp *Viewport) DoUpdate() bool {
	if vp.HasFlag(VpIsUpdating) {
		return false
	}
	vp.SetFlag(true, VpIsUpdating) // prevent rendering
	defer vp.SetFlag(false, VpIsUpdating)

	switch {
	case vp.HasFlag(VpNeedsRebuild):
		vp.SetFlag(false, VpNeedsLayout, VpNeedsRender, VpNeedsRebuild)
		vp.DoRebuild()
	case vp.HasFlag(VpNeedsLayout):
		vp.SetFlag(false, VpNeedsLayout, VpNeedsRender)
		vp.Fill() // full redraw
		vp.Frame.LayoutRenderTree(vp)
	case vp.HasFlag(VpNeedsRender):
		vp.SetFlag(false, VpNeedsRender)
		vp.Frame.DoNeedsRender(vp)
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

// DoRebuild implements the VpNeedsRebuild case
// Typically not called otherwise, and assumes VpIsUpdating already set.
func (vp *Viewport) DoRebuild() {
	vp.Fill() // full redraw
	vp.SetFlag(true, VpRebuild)
	vp.Frame.ConfigTree(vp)
	vp.Frame.LayoutRenderTree(vp)
	vp.SetFlag(false, VpRebuild)
}

// Fill fills the viewport with BgColor (default transparent)
// which is the starting base level for rendering.
// Typically the root Frame fills its background with color
// but it can e.g., leave corners transparent for popups etc.
func (vp *Viewport) Fill() {
	rs := &vp.RenderState
	rs.Lock()
	rs.Paint.FillBox(rs, mat32.Vec2Zero, mat32.NewVec2FmPoint(vp.Geom.Size), vp.BgColor)
	rs.Unlock()
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

//////////////////////////////////////////////////////////////////
//		Widget local rendering

// PushBounds pushes our bounding-box bounds onto the bounds stack if non-empty
// -- this limits our drawing to our own bounding box, automatically -- must
// be called as first step in Render returns whether the new bounds are
// empty or not -- if empty then don't render!
func (wb *WidgetBase) PushBounds(vp *Viewport) bool {
	if wb == nil || wb.This() == nil {
		return false
	}
	if !wb.This().(Widget).IsVisible() {
		return false
	}
	if wb.VpBBox.Empty() {
		return false
	}
	rs := &vp.RenderState
	rs.PushBounds(wb.VpBBox)
	if RenderTrace {
		fmt.Printf("Render: %v at %v\n", wb.Path(), wb.VpBBox)
	}
	return true
}

// PopBounds pops our bounding-box bounds -- last step in Render after
// rendering children
func (wb *WidgetBase) PopBounds(vp *Viewport) {
	wb.ClearFullReRender()
	if wb.IsDeleted() || wb.IsDestroyed() || wb.This() == nil {
		return
	}
	rs := &vp.RenderState
	rs.PopBounds()
}

func (wb *WidgetBase) Render(vp *Viewport) {
	wi := wb.This().(Widget)
	if wb.PushBounds(vp) {
		wi.ConnectEvents(vp)
		wb.RenderParts(vp)
		wb.RenderChildren(vp)
		wb.PopBounds(vp)
	} else {
		wb.DisconnectAllEvents(RegPri)
	}
}

func (wb *WidgetBase) RenderParts(vp *Viewport) {
	if wb.Parts == nil {
		return
	}
	wb.Parts.Render(vp) // is a layout, will do all
}

// RenderChildren renders all of node's children,
// This is the default call at end of Render()
func (wb *WidgetBase) RenderChildren(vp *Viewport) {
	for _, kid := range wb.Kids {
		wi, w := AsWidget(kid)
		if w == nil || w.IsDeleted() || w.IsDestroyed() || w.IsUpdating() {
			continue
		}
		wi.Render(vp)
	}
}

/* todo: anything needed here?

// ReRenderTree does a re-render of the tree -- after it has already been
// initialized and styled -- redoes the full stack
func (wb *WidgetBase) ReRenderTree() {
	parBBox := image.Rectangle{}
	pni, _ := KiToWidget(wb.Par)
	if pni != nil {
		parBBox = pni.ChildrenBBox2D()
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
func (wb *WidgetBase) RenderLock() (*girl.State, *girl.Paint, *gist.Style) {
	wb.StyMu.RLock()
	rs := &wb.Viewport.Render
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
func (wb *WidgetBase) RenderBoxImpl(pos mat32.Vec2, sz mat32.Vec2, bs gist.Border) {
	rs := &wb.Viewport.Render
	pc := &rs.Paint
	pc.DrawBorder(rs, pos.X, pos.Y, sz.X, sz.Y, bs)
}

// RenderStdBox draws standard box using given style.
// girl.State and Style must already be locked at this point (RenderLock)
func (wb *WidgetBase) RenderStdBox(st *gist.Style) {
	// SidesTODO: this is a pretty critical function, so a good place to look if things aren't working
	wb.StyMu.RLock()
	defer wb.StyMu.RUnlock()

	rs := &wb.Viewport.Render
	pc := &rs.Paint

	// TODO: maybe implement some version of this to render background color
	// in margin if the parent element doesn't render for us
	// if pwb, ok := wb.Parent().(*WidgetBase); ok {
	// 	if pwb.Embed(LayoutType) != nil && pwb.Embed(TypeFrame) == nil {
	// 		pc.FillBox(rs, wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size, &st.BackgroundColor)
	// 	}
	// }

	pos := wb.LayState.Alloc.Pos.Add(st.EffMargin().Pos())
	sz := wb.LayState.Alloc.Size.Sub(st.EffMargin().Size())
	rad := st.Border.Radius.Dots()

	// the background color we actually use
	bg := st.BackgroundColor
	// the surrounding background color
	sbg := wb.ParentBackgroundColor()
	if bg.IsNil() {
		// we need to do this to prevent
		// elements from rendering over themselves
		// (see https://goki.dev/gi/v2/issues/565)
		bg = sbg
	}

	// We need to fill the whole box where the
	// box shadows / element can go to prevent growing
	// box shadows and borders. We couldn't just
	// do this when there are box shadows, as they
	// may be removed and then need to be covered up.
	// This also fixes https://goki.dev/gi/v2/issues/579.
	// This isn't an ideal solution because of performance,
	// so TODO: maybe come up with a better solution for this.
	// We need to use raw LayState data because we need to clear
	// any box shadow that may have gone in margin.
	mspos, mssz := st.BoxShadowPosSize(wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size)
	pc.FillBox(rs, mspos, mssz, &sbg)

	// first do any shadow
	if st.HasBoxShadow() {
		for _, shadow := range st.BoxShadow {
			pc.StrokeStyle.SetColor(nil)
			pc.FillStyle.SetColor(shadow.Color)

			// TODO: better handling of opacity?
			prevOpacity := pc.FillStyle.Opacity
			pc.FillStyle.Opacity = float32(shadow.Color.A) / 255
			// we only want radius for border, no actual border
			wb.RenderBoxImpl(shadow.BasePos(pos), shadow.BaseSize(sz), gist.Border{Radius: st.Border.Radius})
			// pc.FillStyle.Opacity = 1.0
			if shadow.Blur.Dots != 0 {
				// must divide by 2 like CSS
				pc.BlurBox(rs, shadow.Pos(pos), shadow.Size(sz), shadow.Blur.Dots/2)
			}
			pc.FillStyle.Opacity = prevOpacity
		}
	}

	// then draw the box over top of that.
	// need to set clipping to box first.. (?)
	// we need to draw things twice here because we need to clear
	// the whole area with the background color first so the border
	// doesn't render weirdly
	if rad.IsZero() {
		pc.FillBox(rs, pos, sz, &bg)
	} else {
		pc.FillStyle.SetColorSpec(&bg)
		// no border -- fill only
		pc.DrawRoundedRectangle(rs, pos.X, pos.Y, sz.X, sz.Y, rad)
		pc.Fill(rs)
	}

	// pc.StrokeStyle.SetColor(&st.Border.Color)
	// pc.StrokeStyle.Width = st.Border.Width
	// pc.FillStyle.SetColorSpec(&st.BackgroundColor)
	pos.SetAdd(st.Border.Width.Dots().Pos().MulScalar(0.5))
	sz.SetSub(st.Border.Width.Dots().Size().MulScalar(0.5))
	pc.FillStyle.SetColor(nil)
	// now that we have drawn background color
	// above, we can draw the border
	wb.RenderBoxImpl(pos, sz, st.Border)
}

// ParentReRenderAnchor returns parent (including this node)
// that is a ReRenderAnchor -- for optimized re-rendering
func (wb *WidgetBase) ParentReRenderAnchor() Widget {
	var par Widget
	nb.FuncUp(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		wi, w := AsWidget(k)
		if w == nil {
			return ki.Break // don't keep going up
		}
		if w.IsReRenderAnchor() {
			par = wi
			return ki.Break
		}
		return ki.Continue
	})
	return par
}
