// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"goki.dev/girl/girl"
	"goki.dev/girl/gist"
	"goki.dev/goosi"
	"goki.dev/ki/v2"
	"goki.dev/prof/v2"
)

// FocusChanges are the kinds of changes that can be reported via
// FocusChanged2D method
type FocusChanges int32 //enums:enum

const (
	// FocusLost means that keyboard focus is on a different widget
	// (typically) and this one lost focus
	FocusLost FocusChanges = iota

	// FocusGot means that this widget just got keyboard focus
	FocusGot

	// FocusInactive means that although this widget retains keyboard focus
	// (nobody else has it), the user has clicked on something else and
	// therefore the focus should be considered inactive (distracted), and any
	// changes should be applied as this other action could result in closing
	// of a dialog etc.  Keyboard events will still be sent to the focus
	// widget, but it is up to the widget if or how to process them (e.g., it
	// could reactivate on its own).
	FocusInactive

	// FocusActive means that the user has moved the mouse back into the
	// focused widget to resume active keyboard focus.
	FocusActive
)

func (nb *Node2DBase) MakeContextMenu(m *Menu) {
}

func (nb *Node2DBase) ContextMenuPos() (pos image.Point) {
	nb.BBoxMu.RLock()
	pos.X = (nb.WinBBox.Min.X + nb.WinBBox.Max.X) / 2
	pos.Y = (nb.WinBBox.Min.Y + nb.WinBBox.Max.Y) / 2
	nb.BBoxMu.RUnlock()
	return
}

func (nb *Node2DBase) ContextMenu() {
	var men Menu
	nb.This().(Node2D).MakeContextMenu(&men)
	if len(men) == 0 {
		return
	}
	pos := nb.This().(Node2D).ContextMenuPos()
	mvp := nb.ViewportSafe()
	PopupMenu(men, pos.X, pos.Y, mvp, nb.Nm+"-menu")
}

func (nb *Node2DBase) IsVisible() bool {
	if nb == nil || nb.This() == nil || nb.IsInvisible() {
		return false
	}
	mvp := nb.ViewportSafe()
	if mvp == nil || mvp.This() == nil {
		return false
	}
	if nb.Par == nil || nb.Par.This() == nil {
		return false
	}
	return nb.Par.This().(Node2D).IsVisible()
}

func (nb *Node2DBase) IsDirectWinUpload() bool {
	return false
}

func (nb *Node2DBase) DirectWinUpload() {
}

////////////////////////////////////////////////////////////////////////////////////////
// 2D basic infrastructure code

// Render returns the girl.State from this node's Viewport, using safe lock access
func (nb *Node2DBase) Render() *girl.State {
	mvp := nb.ViewportSafe()
	if mvp == nil {
		return nil
	}
	return &mvp.Render
}

// EventMgr2D() returns the event manager for this node.
// Can be nil.
func (nb *Node2DBase) EventMgr2D() *EventMgr {
	mvp := nb.ViewportSafe()
	if mvp == nil || mvp.This() == nil {
		return nil
	}
	top := mvp.This().(Viewport).VpTop()
	if top == nil {
		return nil
	}
	return top.VpEventMgr()
}

// TopUpdateStart calls UpdateStart on TopNode2D().  Use this
// for TopUpdateStart / End around multiple dispersed updates to
// properly batch everything and prevent redundant updates.
func (nb *Node2DBase) TopUpdateStart() bool {
	mvp := nb.ViewportSafe()
	if mvp == nil || mvp.This() == nil {
		vp := nb.This().(Node2D).AsViewport()
		if vp != nil && vp.This() != nil {
			return vp.This().(Viewport).VpTopUpdateStart()
		}
		return false
	}
	return mvp.This().(Viewport).VpTopUpdateStart()
}

// TopUpdateEnd calls UpdateEnd on TopNode2D().  Use this
// for TopUpdateStart / End around multiple dispersed updates to
// properly batch everything and prevent redundant updates.
func (nb *Node2DBase) TopUpdateEnd(updt bool) {
	if !updt {
		return
	}
	mvp := nb.ViewportSafe()
	if mvp == nil || mvp.This() == nil {
		vp := nb.This().(Node2D).AsViewport()
		if vp != nil && vp.This() != nil {
			vp.This().(Viewport).VpTopUpdateEnd(updt)
		}
		return
	}
	mvp.This().(Viewport).VpTopUpdateEnd(updt)
}

// ParentWindow returns the parent window for this node
func (nb *Node2DBase) ParentWindow() *Window {
	mvp := nb.ViewportSafe()
	if mvp != nil && mvp.Win != nil {
		return mvp.Win
	}
	wini, err := nb.ParentByTypeTry(TypeWindow, ki.Embeds)
	if err != nil {
		// log.Println(err)
		return nil
	}
	return wini.Embed(TypeWindow).(*Window)
}

// ParentViewport returns the parent viewport -- uses AsViewport() method on
// Node2D interface
func (nb *Node2DBase) ParentViewport() *Viewport {
	var parVp *Viewport
	nb.FuncUpParent(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		nii, ok := k.(Node2D)
		if !ok {
			return ki.Break // don't keep going up
		}
		vp := nii.AsViewport()
		if vp != nil {
			parVp = vp
			return ki.Break // done
		}
		return ki.Continue
	})
	return parVp
}

// ConnectEvent connects this node to receive a given type of GUI event
// signal from the parent window -- typically connect only visible nodes, and
// disconnect when not visible
func (nb *Node2DBase) ConnectEvent(et goosi.EventType, pri EventPris, fun ki.RecvFunc) {
	em := nb.EventMgr2D()
	if em != nil {
		em.ConnectEvent(nb.This(), et, pri, fun)
	}
}

// DisconnectEvent disconnects this receiver from receiving given event
// type -- pri is priority -- pass AllPris for all priorities -- see also
// DisconnectAllEvents
func (nb *Node2DBase) DisconnectEvent(et goosi.EventType, pri EventPris) {
	em := nb.EventMgr2D()
	if em != nil {
		em.DisconnectEvent(nb.This(), et, pri)
	}
}

// DisconnectAllEvents disconnects node from all window events -- typically
// disconnect when not visible -- pri is priority -- pass AllPris for all priorities.
// This goes down the entire tree from this node on down, as typically everything under
// will not get an explicit disconnect call because no further updating will happen
func (nb *Node2DBase) DisconnectAllEvents(pri EventPris) {
	em := nb.EventMgr2D()
	if em == nil {
		return
	}
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		_, ni := KiToNode2D(k)
		if ni == nil || ni.IsDeleted() || ni.IsDestroyed() {
			return ki.Break // going into a different type of thing, bail
		}
		ni.DisconnectViewport()
		em.DisconnectAllEvents(ni.This(), pri)
		return ki.Continue
	})
}

// ConnectToViewport connects the node's update signal to the viewport as
// a receiver, so that when the node is updated, it triggers the viewport to
// re-render it -- this is automatically called in PushBounds, and
// disconnected with DisconnectAllEvents, so it only occurs for rendered nodes.
func (nb *Node2DBase) ConnectToViewport() {
	mvp := nb.ViewportSafe()
	if mvp != nil && mvp.This() != nil {
		nb.NodeSig.Connect(mvp.This(), SignalViewport)
	}
}

// DisconnectViewport disconnects the node's update signal to the viewport as
// a receiver
func (nb *Node2DBase) DisconnectViewport() {
	mvp := nb.ViewportSafe()
	if mvp != nil && mvp.This() != nil {
		nb.NodeSig.Disconnect(mvp.This())
	}
}

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

////////////////////////////////////////////////////////////////////////////////////////
// Tree-walking code for the init, style, layout, render passes
//  typically called by Viewport but can be called by others

// FullConfigTree does a full reinitialization of the tree *below this node*
// this should be called whenever the tree is dynamically updated and new
// nodes are added below a given node -- e.g., loading a new SVG graph etc.
// prepares everything to be rendered as usual.
func (nb *Node2DBase) FullConfigTree() {
	for i := range nb.Kids {
		kd := nb.Kids[i].(Node2D).AsNode2D()
		kd.ConfigTree()
		kd.SetStyleTree()
		kd.Size2DTree(0)
		kd.Layout2DTree()
	}
}

// FullRenderTree does a full render of the tree
func (nb *Node2DBase) FullRenderTree() {
	updt := nb.UpdateStart()
	nb.ConfigTree()
	nb.SetStyleTree()
	nb.Size2DTree(0)
	nb.Layout2DTree()
	nb.RenderTree()
	nb.UpdateEndNoSig(updt)
}

// NeedsFullReRenderTree checks the entire tree below this node for any that have
// NeedsFullReRender flag set.
func (nb *Node2DBase) NeedsFullReRenderTree() bool {
	if nb.This() == nil {
		return false
	}
	full := false
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		_, ni := KiToNode2D(k)
		if ni == nil || ni.IsDeleted() || ni.IsDestroyed() {
			return ki.Break
		}
		if ni.NeedsFullReRender() {
			full = true
			ni.ClearFullReRender()
			return ki.Break // done
		}
		return ki.Continue
	})
	return full
}

// ConfigTree initializes scene graph tree from node it is called on -- only
// needs to be done once but must be robust to repeated calls -- use a flag if
// necessary -- needed after structural updates to ensure all nodes are
// updated
func (nb *Node2DBase) ConfigTree() {
	if nb.This() == nil {
		return
	}
	pr := prof.Start("Node2D.ConfigTree." + ki.Type(nb).Name())
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		nii, ni := KiToNode2D(k)
		if nii == nil || ni.IsDeleted() || ni.IsDestroyed() {
			return ki.Break
		}
		// ppr := prof.Start("ConfigTree:" + nii.Type().Name())
		nii.Config()
		// ppr.End()
		return ki.Continue
	})
	pr.End()
}

// SetStyleTree styles scene graph tree from node it is called on -- only needs
// to be done after a structural update in case inherited options changed
func (nb *Node2DBase) SetStyleTree() {
	if nb.This() == nil {
		return
	}
	// fmt.Printf("\n\n###################################\n%v\n", string(debug.Stack()))
	pr := prof.Start("Node2D.SetStyleTree." + ki.Type(nb).Name())
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		nii, ni := KiToNode2D(k)
		if nii == nil || ni.IsDeleted() || ni.IsDestroyed() {
			return ki.Break
		}
		// ppr := prof.Start("SetStyleTree:" + nii.Type().Name())
		nii.SetStyle()
		// ppr.End()
		return ki.Continue
	})
	pr.End()
}

// Size2DTree does the sizing as a depth-first pass
func (nb *Node2DBase) Size2DTree(iter int) {
	if nb.This() == nil {
		return
	}
	pr := prof.Start("Node2D.Size2DTree." + ki.Type(nb).Name())
	nb.FuncDownMeLast(0, nb.This(),
		func(k ki.Ki, level int, d any) bool { // tests whether to process node
			nii, ni := KiToNode2D(k)
			if nii == nil || ni.IsDeleted() || ni.IsDestroyed() {
				return ki.Break
			}
			if ni.HasNoLayout() {
				return ki.Break
			}
			return ki.Continue
		},
		func(k ki.Ki, level int, d any) bool { // this one does the work
			nii, ni := KiToNode2D(k)
			if ni == nil || ni.IsDeleted() || ni.IsDestroyed() {
				return ki.Break
			}
			nii.Size2D(iter)
			return ki.Continue
		})
	pr.End()
}

// Layout2DTree does layout pass -- each node iterates over children for
// maximum control -- this starts with parent VpBBox -- can be called de novo.
// Handles multiple iterations if needed.
func (nb *Node2DBase) Layout2DTree() {
	if nb.This() == nil || nb.HasNoLayout() {
		return
	}
	pr := prof.Start("Node2D.Layout2DTree." + ki.Type(nb).Name())
	parBBox := image.Rectangle{}
	pni, _ := KiToNode2D(nb.Par)
	if pni != nil {
		parBBox = pni.ChildrenBBox2D()
	}
	nbi := nb.This().(Node2D)
	redo := nbi.Layout2D(parBBox, 0) // important to use interface version to get interface!
	if redo {
		if Layout2DTrace {
			fmt.Printf("Layout: ----------  Redo: %v ----------- \n", nbi.Path())
		}
		wb := nbi.AsWidget()
		if wb != nil {
			la := wb.LayState.Alloc
			wb.Size2DTree(1)
			wb.LayState.Alloc = la
		} else {
			nb.Size2DTree(1)
		}
		nbi.Layout2D(parBBox, 1) // todo: multiple iters?
	}
	pr.End()
}

// RenderTree just calls on parent node and it takes full responsibility for
// managing the children -- this allows maximum flexibility for order etc of
// rendering
func (nb *Node2DBase) RenderTree() {
	if nb.This() == nil {
		return
	}
	// pr := prof.Start("Node2D.RenderTree." + ki.Type(nb).Name())
	nb.This().(Node2D).Render() // important to use interface version to get interface!
	// pr.End()
}

// Layout2DChildren does layout on all of node's children, giving them the
// ChildrenBBox2D -- default call at end of Layout2D.  Passes along whether
// any of the children need a re-layout -- typically Layout2D just returns
// this.
func (nb *Node2DBase) Layout2DChildren(iter int) bool {
	redo := false
	cbb := nb.This().(Node2D).ChildrenBBox2D()
	for _, kid := range nb.Kids {
		nii, _ := KiToNode2D(kid)
		if nii != nil {
			if nii.Layout2D(cbb, iter) {
				redo = true
			}
		}
	}
	return redo
}

// Move2DChildren moves all of node's children, giving them the ChildrenBBox2D
// -- default call at end of Move2D
func (nb *Node2DBase) Move2DChildren(delta image.Point) {
	cbb := nb.This().(Node2D).ChildrenBBox2D()
	for _, kid := range nb.Kids {
		nii, _ := KiToNode2D(kid)
		if nii != nil {
			nii.Move2D(delta, cbb)
		}
	}
}

// RenderChildren renders all of node's children -- default call at end of Render()
func (nb *Node2DBase) RenderChildren() {
	for _, kid := range nb.Kids {
		nii, _ := KiToNode2D(kid)
		if nii != nil {
			nii.Render()
		}
	}
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

// ParentActiveStyle returns parent's active style or nil if not avail.
// Calls StyleRLock so must call ParentStyleRUnlock when done.
func (nb *Node2DBase) ParentActiveStyle() *gist.Style {
	if nb.Par == nil {
		return nil
	}
	if ps, ok := nb.Par.(gist.ActiveStyler); ok {
		st := ps.ActiveStyle()
		ps.StyleRLock()
		return st
	}
	return nil
}

// ParentStyleRUnlock unlocks the parent's style
func (nb *Node2DBase) ParentStyleRUnlock() {
	if nb.Par == nil {
		return
	}
	if ps, ok := nb.Par.(gist.ActiveStyler); ok {
		ps.StyleRUnlock()
	}
}

// ParentPaint returns the Paint from parent, if available
func (nb *Node2DBase) ParentPaint() *gist.Paint {
	if nb.Par == nil {
		return nil
	}
	if pp, ok := nb.Par.(gist.Painter); ok {
		return pp.Paint()
	}
	return nil
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

// ParentLayout returns the parent layout
func (nb *Node2DBase) ParentLayout() *Layout {
	ly := nb.ParentByType(TypeLayout, ki.Embeds)
	if ly == nil {
		return nil
	}
	return ly.Embed(TypeLayout).(*Layout)
}

// ParentScrollLayout returns the parent layout that has active scrollbars
func (nb *Node2DBase) ParentScrollLayout() *Layout {
	lyk := nb.ParentByType(TypeLayout, ki.Embeds)
	if lyk == nil {
		return nil
	}
	ly := lyk.Embed(TypeLayout).(*Layout)
	if ly.HasAnyScroll() {
		return ly
	}
	return ly.ParentScrollLayout()
}

// ScrollToMe tells my parent layout (that has scroll bars) to scroll to keep
// this widget in view -- returns true if scrolled
func (nb *Node2DBase) ScrollToMe() bool {
	ly := nb.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollToItem(nb.This().(Node2D))
}
