// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"sync"

	"goki.dev/girl/girl"
	"goki.dev/girl/gist"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// VpFlags has critical state information signaling when rendering,
// styling etc need to be done
type VpFlags int64 //enums:bitflag

const (
	// VpNeedsRender means nodes have flagged that they need rendering
	VpNeedsRender VpFlags = iota

	// VpNeedsStyle means nodes have flagged that they need style updated
	VpNeedsStyle

	// VpIsRendering means viewport is in the process of rendering
	// do not trigger another render at this point.
	VpIsRendering

	// VpIsStyling means viewport is in the process of styling
	// do not trigger another style pass at this point
	VpNeedsStyle

	// todo: remove?

	// VpPopupDestroyAll means that if this is a popup, then destroy all
	// the children when it is deleted -- otherwise children below the main
	// layout under the vp will not be destroyed -- it is up to the caller to
	// manage those (typically these are reusable assets)
	VpPopupDestroyAll

	// VpNeedsFullRender means that this viewport needs to do a full
	// render -- this is set during signal processing and will preempt
	// other lower-level updates etc.
	VpNeedsFullRender

	// VpDoingFullRender means that this viewport is currently doing a
	// full render -- can be used by elements to drive deep rebuild in case
	// underlying data has changed.
	VpDoingFullRender

	// VpPrefSizing means that this viewport is currently doing a
	// PrefSize computation to compute the size of the viewport
	// (for sizing window for example) -- affects layout size computation
	// only for Over
	VpPrefSizing
)

// VpType indicates the type of viewport
type VpType int //enums:enum

const (
	// VpMain means viewport is for a main window
	VpMain VpType = iota

	// VpDialog means viewport is a dialog
	VpDialog

	// VpMenu means viewport is a popup menu
	VpMenu

	// VpCompleter means viewport is a popup menu for completion
	VpCompleter

	// VpCorrector means viewport is a popup menu for spelling correction
	VpCorrector

	// VpTooltip means viewport is serving as a tooltip
	VpTooltip

	// VpPopup means viewport is a popup of some other type
	VpPopup
)

// A Viewport ALWAYS presents its children with a 0,0 - (Size.X, Size.Y)
// rendering area even if it is itself a child of another Viewport.  This is
// necessary for rendering onto the image that it provides.  This creates
// challenges for managing the different geometries in a coherent way, e.g.,
// events come through the Window in terms of the root VP coords.  Thus, nodes
// require a  WinBBox for events and a VpBBox for their parent Viewport.

/*

// Viewport provides an interface for viewports,
// supporting overall management functions that can be
// provided by more embedded viewports for example.
type Viewport interface {
	// VpTop returns the top-level Viewport, which could be this one
	// or a higher one.  VpTopNode and VpEventMgr should be called on
	// on the Viewport returned by this method.  For popups
	// this *not* the popup viewport but rather the window top viewport.
	VpTop() Viewport

	// VpTopNode returns the top node for this viewport.
	// must be called on VpTop()
	VpTopNode() Node

	// VpTopUpdateStart calls UpdateStart on VpTopNode2D().  Use this
	// for TopUpdateStart / End around multiple dispersed updates to
	// properly batch everything and prevent redundant updates.
	VpTopUpdateStart() bool

	// VpTopUpdateEnd calls UpdateEnd on VpTopNode2D().  Use this
	// for TopUpdateStart / End around multiple dispersed updates to
	// properly batch everything and prevent redundant updates.
	VpTopUpdateEnd(updt bool)

	// VpEventMgr returns the event manager for this viewport.
	// Must be called on VpTop().  Can be nil.
	VpEventMgr() *EventMgr

	// VpIsVisible returns true if this viewport is visible.
	// If false, rendering is aborted
	VpIsVisible() bool

	// VpUploadAll is the update call for the main viewport for a window --
	// calls UploadAllViewports in parent window, which uploads the main viewport
	// and any active popups etc over the top of that
	VpUploadAll()

	// VpUploadVp uploads our viewport image into the parent window -- e.g., called
	// by popups when updating separately
	VpUploadVp()

	// VpUploadRegion uploads node region of our viewport image
	VpUploadRegion(vpBBox, winBBox image.Rectangle)
}
*/

// Viewport contains a Widget tree, rooted in a Frame layout,
// which renders into its Pixels image.
type Viewport struct {

	// has critical state information signaling when rendering, styling etc need to be done, and also indicates type of viewport
	Flags Vps `desc:"has critical state information signaling when rendering, styling etc need to be done, and also indicates type of viewport"`

	// fill the viewport with background-color from style
	Fill bool `desc:"fill the viewport with background-color from style"`

	// Viewport-level viewbox within any parent Viewport
	Geom Geom2DInt `desc:"Viewport-level viewbox within any parent Viewport"`

	// [view: -] render state for rendering
	RenderState girl.State `copy:"-" json:"-" xml:"-" view:"-" desc:"render state for rendering"`

	// [view: -] live pixels that we render into
	Pixels *image.RGBA `copy:"-" json:"-" xml:"-" view:"-" desc:"live pixels that we render into"`

	// our parent window that we render into
	Win *Window `copy:"-" json:"-" xml:"-" desc:"our parent window that we render into"`

	// [view: -] CurStyleNode2D is always set to the current node that is being styled used for finding url references -- only active during a Style pass
	// CurStyleNode Node2D `copy:"-" json:"-" xml:"-" view:"-" desc:"CurStyleNode2D is always set to the current node that is being styled used for finding url references -- only active during a Style pass"`

	// Root of the scenegraph for this viewport
	Frame Frame `desc:"Root of the scenegraph for this viewport"`

	// [view: -] UpdtMu is mutex for viewport updates
	UpdtMu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" desc:"UpdtMu is mutex for viewport updates"`

	// [view: -] StackMu is mutex for adding to UpdtStack
	StackMu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" desc:"StackMu is mutex for adding to UpdtStack"`

	// [view: -] StyleMu is RW mutex protecting access to Style-related global vars
	StyleMu sync.RWMutex `copy:"-" json:"-" xml:"-" view:"-" desc:"StyleMu is RW mutex protecting access to Style-related global vars"`
}

func (vp *Viewport) OnInit() {
	vp.Frame.Style.BackgroundColor.SetSolid(ColorScheme.Background)
	vp.Frame.Style.Color = ColorScheme.OnBackground
}

// NewViewport creates a new Pixels Image with the specified width and height,
// and initializes the renderer etc
func NewViewport(width, height int) *Viewport {
	sz := image.Point{width, height}
	vp := &Viewport{
		Geom: Geom2DInt{Size: sz},
	}
	vp.Pixels = image.NewRGBA(image.Rectangle{Max: sz})
	vp.RenderState.Init(width, height, vp.Pixels)
	return vp
}

// Resize resizes the viewport, creating a new image -- updates Geom Size
func (vp *Viewport) Resize(nwsz image.Point) {
	if nwsz.X == 0 || nwsz.Y == 0 {
		return
	}
	if vp.Pixels != nil {
		ib := vp.Pixels.Bounds().Size()
		if ib == nwsz {
			vp.Geom.Size = nwsz // make sure
			return              // already good
		}
	}
	if vp.Pixels != nil {
		vp.Pixels = nil
	}
	vp.Pixels = image.NewRGBA(image.Rectangle{Max: nwsz})
	vp.Render.Init(nwsz.X, nwsz.Y, vp.Pixels)
	vp.Geom.Size = nwsz // make sure
	// fmt.Printf("vp %v resized to: %v, bounds: %v\n", vp.Path(), nwsz, vp.Pixels.Bounds())
}

func (vp *Viewport) IsPopup() bool {
	return vp.HasFlag(int(VpPopup))
}

func (vp *Viewport) IsMenu() bool {
	return vp.HasFlag(int(VpMenu))
}

func (vp *Viewport) IsCompleter() bool {
	return vp.HasFlag(int(VpCompleter))
}

func (vp *Viewport) IsCorrector() bool {
	return vp.HasFlag(int(VpCorrector))
}

func (vp *Viewport) IsTooltip() bool {
	return vp.HasFlag(int(VpTooltip))
}

func (vp *Viewport) IsSVG() bool {
	return vp.HasFlag(int(VpSVG))
}

func (vp *Viewport) IsUpdatingNode() bool {
	return vp.HasFlag(int(VpUpdatingNode))
}

func (vp *Viewport) NeedsFullRender() bool {
	return vp.HasFlag(int(VpNeedsFullRender))
}

func (vp *Viewport) IsDoingFullRender() bool {
	return vp.HasFlag(int(VpDoingFullRender))
}

func (vp *Viewport) IsVisible() bool {
	if vp == nil || vp.This() == nil || vp.IsInvisible() {
		return false
	}
	return vp.This().(Viewport).VpIsVisible()
}

////////////////////////////////////////////////////////////////////////////////////////
//  Viewport interface implementation

func (vp *Viewport) VpTop() Viewport {
	if vp.Win != nil {
		return vp.Win.Viewport
	}
	if vp.Par == nil {
		return vp.This().(Viewport)
	}
	pvp := vp.ParentViewport()
	if pvp != nil {
		return pvp.This().(Viewport)
	}
	return vp.This().(Viewport)
}

func (vp *Viewport) VpTopNode() Node {
	if vp.Win != nil {
		return vp.Win
	}
	return nil
}

func (vp *Viewport) VpTopUpdateStart() bool {
	if vp.Win != nil {
		return vp.Win.UpdateStart()
	}
	return false
}

func (vp *Viewport) VpTopUpdateEnd(updt bool) {
	if !updt {
		return
	}
	if vp.Win != nil {
		vp.Win.UpdateEnd(updt)
	}
}

// note: if not a standard viewport in a window, this method must be redefined!

func (vp *Viewport) VpEventMgr() *EventMgr {
	if vp.Win != nil {
		return &vp.Win.EventMgr
	}
	return nil
}

func (vp *Viewport) VpIsVisible() bool {
	if vp == nil || vp.This() == nil || vp.Win == nil || vp.Pixels == nil {
		return false
	}
	return vp.Win.IsVisible()
}

////////////////////////////////////////////////////////////////////////////////////////
//  Main Rendering code

// VpUploadAll is the update call for the main viewport for a window --
// calls UploadAllViewports in parent window, which uploads the main viewport
// and any active popups etc over the top of that
func (vp *Viewport) VpUploadAll() {
	if !vp.This().(Viewport).VpIsVisible() {
		return
	}
	vp.Win.UploadAllViewports()
}

// VpUploadVp uploads our viewport image into the parent window -- e.g., called
// by popups when updating separately
func (vp *Viewport) VpUploadVp() {
	if !vp.This().(Viewport).VpIsVisible() {
		return
	}
	vp.BBoxMu.RLock()
	vp.Win.UploadVp(vp, vp.WinBBox.Min)
	vp.BBoxMu.RUnlock()
}

// VpUploadRegion uploads node region of our viewport image
func (vp *Viewport) VpUploadRegion(vpBBox, winBBox image.Rectangle) {
	if !vp.This().(Viewport).VpIsVisible() {
		return
	}
	vpin := vpBBox.Intersect(vp.Pixels.Bounds())
	if vpin.Empty() {
		return
	}
	vp.Win.UploadVpRegion(vp, vpin, winBBox)
}

// set our window pointer to point to the current window we are under
func (vp *Viewport) SetCurWin() {
	pwin := vp.ParentWindow()
	if pwin != nil { // only update if non-nil -- otherwise we could be setting
		// temporarily to give access to DPI etc
		vp.Win = pwin
	}
}

// DrawIntoParent draws our viewport image into parent's image -- this is the
// typical way that a sub-viewport renders (e.g., svg boxes, icons, etc -- not popups)
func (vp *Viewport) DrawIntoParent(parVp *Viewport) {
	if parVp.Pixels == nil || vp.Pixels == nil {
		if RenderTrace {
			fmt.Printf("Render: vp DrawIntoParent nil Pixels - no render!: %v parVp: %v\n", vp.Path(), parVp.Path())
		}
		return
	}
	r := vp.Geom.Bounds()
	sp := image.Point{}
	if vp.Par != nil { // use parents children bbox to determine where we can draw
		pni, _ := KiToNode2D(vp.Par)
		nr := r.Intersect(pni.ChildrenBBox2D())
		sp = nr.Min.Sub(r.Min)
		if sp.X < 0 || sp.Y < 0 || sp.X > 10000 || sp.Y > 10000 {
			fmt.Printf("aberrant sp: %v\n", sp)
			return
		}
		r = nr
	}
	if RenderTrace {
		fmt.Printf("Render: vp DrawIntoParent: %v parVp: %v rect: %v sp: %v\n", vp.Path(), parVp.Path(), r, sp)
	}
	draw.Draw(parVp.Pixels, r, vp.Pixels, sp, draw.Over)
}

// ReRenderNode re-renders a specific node, including uploading updated bits to
// the window texture using Window.UploadVpRegion call.
// This should be covered by an outer UpdateStart / End bracket on Window to drive
// publishing changes, with suitable grouping if multiple updates
func (vp *Viewport) ReRenderNode(gni Node2D) {
	if !vp.This().(Viewport).VpIsVisible() {
		return
	}
	gn := gni.AsNode2D()
	if RenderTrace {
		fmt.Printf("Render: vp re-render: %v node: %v\n", vp.Path(), gn.Path())
	}
	// pr := prof.Start("vp.ReRenderNode")
	gn.RenderTree()
	// pr.End()
	gn.BBoxMu.RLock()
	wbb := gn.WinBBox
	gn.BBoxMu.RUnlock()
	vp.This().(Viewport).VpUploadRegion(gn.VpBBox, wbb)
}

// ReRenderAnchor re-renders an anchor node -- the KEY diff from
// ReRenderNode is that it calls ReRenderTree and not just RenderTree!
// uploads updated bits to the window texture using Window.UploadVpRegion call.
// This should be covered by an outer UpdateStart / End bracket on Window to drive
// publishing changes, with suitable grouping if multiple updates
func (vp *Viewport) ReRenderAnchor(gni Node2D) {
	if !vp.This().(Viewport).VpIsVisible() {
		return
	}
	pw := gni.AsWidget()
	if pw == nil {
		return
	}
	if RenderTrace {
		fmt.Printf("Render: vp anchor re-render: %v node: %v\n", vp.Path(), pw.Path())
	}
	// pr := prof.Start("vp.ReRenderNode")
	pw.ReRenderTree()
	// pr.End()
	pw.BBoxMu.RLock()
	wbb := pw.WinBBox
	pw.BBoxMu.RUnlock()
	vp.This().(Viewport).VpUploadRegion(pw.VpBBox, wbb)
}

// Delete this popup viewport -- has already been disconnected from window
// events and parent is nil -- called by window when a popup is deleted -- it
// destroys the vp and its main layout, see VpPopupDestroyAll for whether
// children are destroyed
func (vp *Viewport) DeletePopup() {
	vp.Par = nil // disconnect from window -- it never actually owned us as a child
	vp.Win = nil
	vp.This().SetFlag(int(ki.NodeDeleted)) // prevent further access
	if !vp.HasFlag(int(VpPopupDestroyAll)) {
		// delete children of main layout prior to deleting the popup (e.g., menu items) so they don't get destroyed
		if len(vp.Kids) == 1 {
			cli, _ := KiToNode2D(vp.Child(0))
			ly := cli.AsDoLayout(vp * Viewport)
			if ly != nil {
				ly.DeleteChildren(ki.NoDestroyKids) // do NOT destroy children -- just delete them
			}
		}
	}
	vp.This().Destroy() // nuke everything else in us
}

////////////////////////////////////////////////////////////////////////////////////////
// Node2D interface

func (vp *Viewport) AsViewport() *Viewport {
	return vp
}

func (vp *Viewport) Config() {
	vp.ConfigWidget()
	vp.SetCurWin()
	// note: used to have a NodeSig update here but was redundant -- already handled.
	// also note that SVG viewports require SetNeedsFullRender to repaint!
}

func (vp *Viewport) SetStyle() {
	vp.StyMu.Lock()
	defer vp.StyMu.Unlock()

	vp.SetCurWin()
	vp.SetStyleWidget()
	vp.LayState.SetFromStyle(&vp.Style) // also does reset
}

func (vp *Viewport) GetSize(vp *Viewport, iter int) {
	vp.InitDoLayout(vp * Viewport)
	// we listen to x,y styling for positioning within parent vp, if non-zero -- todo: only popup?
	pos := vp.Style.PosDots().ToPoint()
	if pos != (image.Point{}) {
		vp.Geom.Pos = pos
	}
	if !vp.IsSVG() && vp.Geom.Size != (image.Point{}) {
		vp.LayState.Alloc.Size.SetPoint(vp.Geom.Size)
	}
}

func (vp *Viewport) DoLayout(vp *Viewport, parBBox image.Rectangle, iter int) bool {
	vp.DoLayoutBase(parBBox, true, iter)
	return vp.DoLayoutChildren(iter)
}

func (vp *Viewport) BBox2D() image.Rectangle {
	if vp.Viewport == nil || vp.IsPopup() { // top level viewport
		// viewport ignores any parent parent bbox info!
		if vp.Pixels == nil || !vp.IsPopup() { // non-popups use allocated sizes via layout etc
			if !vp.LayState.Alloc.Size.IsNil() {
				asz := vp.LayState.Alloc.Size.ToPointCeil()
				vp.Resize(asz)
			} else if vp.Pixels == nil {
				vp.Resize(image.Point{64, 64}) // gotta have something..
			}
		}
		return vp.Pixels.Bounds()
	} else {
		bb := vp.BBoxFromAlloc()
		sz := bb.Size()
		if sz != (image.Point{}) {
			vp.Resize(sz)
		} else {
			if vp.Pixels == nil {
				vp.Resize(image.Point{64, 64}) // gotta have something..
			}
			bb.Max = bb.Min.Add(vp.Pixels.Bounds().Size())
		}
		return bb
	}
}

func (vp *Viewport) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	// vp.VpBBox = vp.Pixels.Bounds()
	// vp.SetWinBBox()    // this adds all PARENT offsets
	if vp.Viewport != nil {
		vp.ComputeBBox2DBase(parBBox, delta)
	} else {
		vp.VpBBox = vp.Pixels.Bounds()
		vp.SetWinBBox() // should be same as VpBBox
	}
	if !vp.IsPopup() { // non-popups use allocated positions
		vp.Geom.Pos = vp.LayState.Alloc.Pos.ToPointFloor()
	}
	if vp.Viewport == nil {
		vp.BBoxMu.Lock()
		vp.WinBBox = vp.WinBBox.Add(vp.Geom.Pos)
		vp.BBoxMu.Unlock()
	}
	// fmt.Printf("Viewport: %v bbox: %v vpBBox: %v winBBox: %v\n", vp.Path(), vp.BBox, vp.VpBBox, vp.WinBBox)
}

func (vp *Viewport) ChildrenBBox2D() image.Rectangle {
	if vp.Pixels == nil {
		sz := vp.Geom.Size
		if sz != (image.Point{}) {
			return vp.Geom.Bounds()
		}
		return image.Rectangle{Max: image.Point{100, 100}}
	}
	return vp.Pixels.Bounds() // vp.VpBBox -- this is where we transition to new coordinates!
}

// RenderViewport is the render action for the viewport itself -- either
// uploads image to window or draws into parent viewport
func (vp *Viewport) RenderViewport() {
	if vp.IsPopup() { // popup has a parent that is the window
		vp.SetCurWin()
		if RenderTrace {
			fmt.Printf("Render: %v at Popup VpUploadVp\n", vp.Path())
		}
		vp.This().(Viewport).VpUploadVp()
	} else if vp.Viewport != nil { // sub-vp
		if RenderTrace {
			fmt.Printf("Render: %v at %v DrawIntoParent\n", vp.Path(), vp.VpBBox)
		}
		vp.DrawIntoParent(vp.Viewport)
	} else { // we are the main vp
		if RenderTrace {
			fmt.Printf("Render: %v at %v VpUploadAll\n", vp.Path(), vp.VpBBox)
		}
		vp.This().(Viewport).VpUploadAll()
	}
}

// FullRenderTree is called by window and other places to completely
// re-render -- we set our flag when doing this so valueview elements (and
// anyone else) can do a deep re-build that is typically not otherwise needed
// (e.g., after non-signaling structs have updated)
func (vp *Viewport) FullRenderTree() {
	if vp.IsUpdating() { // already in process!
		return
	}
	vp.SetFlag(int(VpDoingFullRender))
	if RenderTrace {
		fmt.Printf("Render: %v doing full render\n", vp.Path())
	}
	vp.WidgetBase.FullRenderTree()
	vp.ClearFlag(int(VpDoingFullRender))
}

// we use our own render for these -- Viewport member is our parent!
func (vp *Viewport) PushBounds() bool {
	if vp.VpBBox.Empty() {
		return false
	}
	if !vp.This().(Node2D).IsVisible() {
		return false
	}
	// if we are completely invisible, no point in rendering..
	if vp.Viewport != nil {
		vp.BBoxMu.RLock()
		vp.Viewport.BBoxMu.RLock()
		wbi := vp.WinBBox.Intersect(vp.Viewport.WinBBox)
		vp.Viewport.BBoxMu.RUnlock()
		vp.BBoxMu.RUnlock()
		if wbi.Empty() {
			// fmt.Printf("not rendering vp %v bc empty winbox -- ours: %v par: %v\n", vp.Nm, vp.WinBBox, vp.Viewport.WinBBox)
			return false
		}
	}
	rs := &vp.Render
	bb := vp.Pixels.Bounds() // our bounds.. not vp.VpBBox)
	rs.PushBounds(bb)
	if RenderTrace {
		fmt.Printf("Render: %v at %v\n", vp.Path(), bb)
	}
	return true
}

func (vp *Viewport) PopBounds() {
	rs := &vp.Render
	rs.PopBounds()
}

func (vp *Viewport) Move2D(delta image.Point, parBBox image.Rectangle) {
	if vp == nil {
		return
	}
	vp.Move2DBase(delta, parBBox)
	vp.Move2DChildren(image.Point{}) // reset delta here -- we absorb the delta in our placement relative to the parent
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

func (vp *Viewport) FullReRenderIfNeeded() bool {
	vpDoing := false
	if vp.Viewport != nil && vp.Viewport.IsDoingFullRender() {
		vpDoing = true
	}
	if vp.This().(Node2D).IsVisible() && vp.NeedsFullReRender() && !vpDoing {
		if RenderTrace {
			fmt.Printf("Render: NeedsFullReRender for %v at %v\n", vp.Path(), vp.VpBBox)
		}
		vp.ClearFullReRender()
		vp.ReRenderTree()
		return true
	}
	return false
}

func (vp *Viewport) Render(vp *Viewport) {
	if vp.FullReRenderIfNeeded() {
		return
	}
	if vp.PushBounds() {
		if vp.Fill {
			vp.FillViewport()
		}
		vp.RenderChildren() // we must do children first, then us!
		vp.RenderViewport() // update our parent image
		vp.PopBounds()
	}
}

// PrefSize computes the preferred size of the viewport based on current contents.
// initSz is the initial size -- e.g., size of screen.
// Used for auto-sizing windows.
func (vp *Viewport) PrefSize(initSz image.Point) image.Point {
	vp.SetFlag(int(VpPrefSizing))
	vp.ConfigTree()
	vp.SetStyleTree() // sufficient to get sizes
	vp.LayState.Alloc.Size.SetPoint(initSz)
	vp.Size2DTree(0) // collect sizes
	vp.ClearFlag(int(VpPrefSizing))
	ch := vp.ChildByType(TypeLayout, ki.Embeds, 0).Embed(TypeLayout).(*Layout)
	vpsz := ch.LayState.Size.Pref.ToPoint()
	// also take into account min size pref
	stw := int(vp.Style.MinWidth.Dots)
	sth := int(vp.Style.MinHeight.Dots)
	// fmt.Printf("dlg stw %v sth %v dpi %v vpsz: %v\n", stw, sth, dlg.Sty.UnContext.DPI, vpsz)
	vpsz.X = max(vpsz.X, stw)
	vpsz.Y = max(vpsz.Y, sth)
	return vpsz
}

////////////////////////////////////////////////////////////////////////////////////////
//  Signal Handling

// SignalViewport is called by each node in scenegraph through its NodeSig
// signal to notify its parent viewport whenever it changes, causing a
// re-render.
func SignalViewport(vpki, send ki.Ki, sig int64, data any) {
	vpni, ok := vpki.(Node2D)
	if !ok {
		return
	}
	vp := vpni.AsViewport()
	if vp == nil { // should not happen -- should only be called on viewports
		return
	}
	nii, ni := KiToNode2D(send)
	if nii == nil { // should not happen
		return
	}
	if ni.IsDeleted() || ni.IsDestroyed() { // skip these for sure
		return
	}
	if ni.IsUpdating() {
		if Update2DTrace { // this can happen during concurrent update situations
			log.Printf("Update: SignalViewport updating node %v with Updating flag set\n", ni.Path())
		}
		return
	}

	if Update2DTrace {
		fmt.Printf("Update: Viewport: %v NodeUpdated due to signal: %v from node: %v\n", vp.Path(), ki.NodeSignals(sig), send.Path())
	}

	vp.NodeUpdated(nii, sig, data)
}

// NodeUpdated is called from SignalViewport when a valid node's NodeSig sent a signal
// usually after UpdateEnd.
func (vp *Viewport) NodeUpdated(nii Node2D, sig int64, data any) {
	if !vp.NeedsFullRender() {
		vp.StackMu.Lock()
		anchor, full := vp.UpdateLevel(nii, sig, data)
		if anchor != nil {
			already := false
			for _, n := range vp.ReStack {
				if n == anchor {
					already = true
					break
				}
			}
			if !already {
				vp.ReStack = append(vp.ReStack, anchor)
			}
		} else if full {
			vp.SetFlag(int(VpNeedsFullRender))
		} else {
			already := false
			for _, n := range vp.UpdtStack {
				if n == nii {
					already = true
					break
				}
			}
			if !already {
				for _, n := range vp.ReStack {
					if nii.ParentLevel(n) >= 0 {
						already = true
						break
					}
				}
			}
			if !already {
				vp.UpdtStack = append(vp.UpdtStack, nii)
			}
		}
		vp.StackMu.Unlock()
	}

	if !vp.IsUpdatingNode() {
		vp.UpdateNodes() // do all pending nodes
	}
}

// UpdateLevel deteremines what level of updating a node requires
func (vp *Viewport) UpdateLevel(nii Node2D, sig int64, data any) (anchor Node2D, full bool) {
	ni := nii.AsNode2D()
	if sig == int64(ki.NodeSignalUpdated) {
		dflags := data.(int64)
		// todo:
		// vlupdt := bitflag.HasAnyMask(dflags, ki.ValUpdateFlagsMask)
		// strupdt := bitflag.HasAnyMask(dflags, ki.StruUpdateFlagsMask)
		if vlupdt && !strupdt {
			full = false
		} else if strupdt {
			full = true
		}
	} else {
		full = true
	}
	if ni.IsReRenderAnchor() { // only anchors check for any kids that need full rerender
		// so far this just makes things a bit more unpredictable, while fixing *one* case in SymbolsView
		if ni.NeedsFullReRender() { // 2DTree() {
			full = true
		}
	} else if ni.NeedsFullReRender() {
		full = true
	}
	if full {
		ni.ClearFullReRender()
		if Update2DTrace {
			fmt.Printf("Update: Viewport: %v FullRenderTree (structural changes) for node: %v\n", vp.Path(), nii.Path())
		}
		anchor = ni.ParentReRenderAnchor()
		return anchor, full
	}
	return nil, false
}

// SetNeedsFullRender sets the flag indicating that a full render of the viewport is needed
// it will do this immediately pending acquisition of the lock and through the standard
// updating channels, unless already updating.
func (vp *Viewport) SetNeedsFullRender() {
	if !vp.NeedsFullRender() {
		vp.StackMu.Lock()
		vp.SetFlag(int(VpNeedsFullRender))
		vp.StackMu.Unlock()
	}
	if !vp.IsUpdatingNode() {
		vp.UpdateNodes() // do all pending nodes
	}
}

// BlockUpdates uses the UpdtMu lock to block all updates to this viewport.
// This is *ONLY* needed when structural updates to the scenegraph are being
// made from a different goroutine outside of the one this window's event
// loop is running on.  This prevents an update from happening in the
// middle of the construction process and thus attempting to render garbage.
// Must call UnblockUpdates after construction is done.
func (vp *Viewport) BlockUpdates() {
	vp.UpdtMu.Lock()
	vp.SetFlag(int(VpUpdatingNode))
	vp.UpdtMu.Unlock()
}

// UnblockUpdates unblocks updating of this viewport -- see BlockUpdates()
func (vp *Viewport) UnblockUpdates() {
	vp.UpdtMu.Lock()
	vp.ClearFlag(int(VpUpdatingNode))
	vp.UpdtMu.Unlock()
}

// UpdateNodes processes the current update signals and actually does the relevant updating
func (vp *Viewport) UpdateNodes() {
	vp.UpdtMu.Lock()
	vp.SetFlag(int(VpUpdatingNode))
	tn := vp.TopNode2D()
	if tn != nil && tn != vp.This().(Node) {
		wupdt := tn.UpdateStart()
		defer tn.UpdateEnd(wupdt)
	}
	for {
		if vp.NeedsFullRender() {
			vp.StackMu.Lock()
			vp.ReStack = nil
			vp.UpdtStack = nil
			vp.ClearFlag(int(VpNeedsFullRender))
			vp.StackMu.Unlock()
			if vp.Viewport == nil { // top level
				vp.FullRenderTree()
			} else {
				vp.ReRenderTree() // embedded
			}
			break
		}
		vp.StackMu.Lock()
		if len(vp.ReStack) == 0 && len(vp.UpdtStack) == 0 {
			vp.StackMu.Unlock()
			break
		}
		if len(vp.ReStack) > 0 {
			nii := vp.ReStack[0]
			vp.ReStack = vp.ReStack[1:]
			vp.StackMu.Unlock()
			vp.ReRenderAnchor(nii)
			continue
		}
		if len(vp.UpdtStack) > 0 {
			nii := vp.UpdtStack[0]
			vp.UpdtStack = vp.UpdtStack[1:]
			vp.StackMu.Unlock()
			vp.UpdateNode(nii)
			continue
		}
	}

	vp.ClearFlag(int(VpUpdatingNode))
	vp.UpdtMu.Unlock()
}

// UpdateNode is called under UpdtMu lock and does the actual steps to update a given node
func (vp *Viewport) UpdateNode(nii Node2D) {
	if nii.IsDirectWinUpload() {
		if Update2DTrace {
			fmt.Printf("Update: Viewport: %v calling DirectWinUpload on %v\n", vp.Path(), nii.Path())
		}
		nii.DirectWinUpload()
	} else {
		if Update2DTrace {
			fmt.Printf("Update: Viewport: %v ReRender on %v\n", vp.Path(), nii.Path())
		}
		vp.ReRenderNode(nii)
	}
}

//////////////////////////////////////////////////////////////////////////////////
//  Style state

// SetCurStyleNode sets the current styling node to given node, and nil to clear
func (vp *Viewport) SetCurStyleNode(node Node2D) {
	if vp == nil {
		return
	}
	vp.StyleMu.Lock()
	vp.CurStyleNode = node
	vp.StyleMu.Unlock()
}

// SetCurrentColor sets the current color in concurrent-safe way
func (vp *Viewport) SetCurrentColor(clr color.RGBA) {
	if vp == nil {
		return
	}
	vp.StyleMu.Lock()
	vp.CurColor = clr
	vp.StyleMu.Unlock()
}

// ContextColor gets the current color in concurrent-safe way.
// Implements the gist.Context interface
func (vp *Viewport) ContextColor() color.RGBA {
	if vp == nil {
		return color.RGBA{}
	}
	vp.StyleMu.RLock()
	clr := vp.CurColor
	vp.StyleMu.RUnlock()
	return clr
}

// ContextColorSpecByURL finds a Node by an element name (URL-like path), and
// attempts to convert it to a Gradient -- if successful, returns ColorSpec on that.
// Used for colorspec styling based on url() value.
func (vp *Viewport) ContextColorSpecByURL(url string) *gist.ColorSpec {
	return nil
}

//////////////////////////////////////////////////////////////////////////////////
//  Image utilities

// SavePNG encodes the image as a PNG and writes it to disk.
func (vp *Viewport) SavePNG(path string) error {
	return SavePNG(path, vp.Pixels)
}

// EncodePNG encodes the image as a PNG and writes it to the provided io.Writer.
func (vp *Viewport) EncodePNG(w io.Writer) error {
	return png.Encode(w, vp.Pixels)
}
