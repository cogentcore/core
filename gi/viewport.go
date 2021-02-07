// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/goki/gi/girl"
	"github.com/goki/gi/gist"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// A Viewport ALWAYS presents its children with a 0,0 - (Size.X, Size.Y)
// rendering area even if it is itself a child of another Viewport.  This is
// necessary for rendering onto the image that it provides.  This creates
// challenges for managing the different geometries in a coherent way, e.g.,
// events come through the Window in terms of the root VP coords.  Thus, nodes
// require a  WinBBox for events and a VpBBox for their parent Viewport.

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

// Viewport2D provides an image and a stack of Paint contexts for drawing onto the image
// with a convenience forwarding of the Paint methods operating on the current Paint
type Viewport2D struct {
	WidgetBase
	Fill         bool         `desc:"fill the viewport with background-color from style"`
	Geom         Geom2DInt    `desc:"Viewport-level viewbox within any parent Viewport2D"`
	Render       girl.State   `copy:"-" json:"-" xml:"-" view:"-" desc:"render state for rendering"`
	Pixels       *image.RGBA  `copy:"-" json:"-" xml:"-" view:"-" desc:"live pixels that we render into"`
	Win          *Window      `copy:"-" json:"-" xml:"-" desc:"our parent window that we render into"`
	CurStyleNode Node2D       `copy:"-" json:"-" xml:"-" view:"-" desc:"CurStyleNode2D is always set to the current node that is being styled used for finding url references -- only active during a Style pass"`
	CurColor     gist.Color   `copy:"-" json:"-" xml:"-" view:"-" desc:"CurColor is automatically updated from the Color setting of a Style and accessible as a color name in any other style as currentcolor use accessor routines for concurrent-safe access"`
	UpdtMu       sync.Mutex   `copy:"-" json:"-" xml:"-" view:"-" desc:"UpdtMu is mutex for viewport updates"`
	UpdtStack    []Node2D     `copy:"-" json:"-" xml:"-" view:"-" desc:"stack of nodes requring basic updating"`
	ReStack      []Node2D     `copy:"-" json:"-" xml:"-" view:"-" desc:"stack of nodes requiring a ReRender (i.e., anchors)"`
	StackMu      sync.Mutex   `copy:"-" json:"-" xml:"-" view:"-" desc:"StackMu is mutex for adding to UpdtStack"`
	StyleMu      sync.RWMutex `copy:"-" json:"-" xml:"-" view:"-" desc:"StyleMu is RW mutex protecting access to Style-related global vars"`
}

var KiT_Viewport2D = kit.Types.AddType(&Viewport2D{}, Viewport2DProps)

var Viewport2DProps = ki.Props{
	"EnumType:Flag":    KiT_VpFlags,
	"color":            &Prefs.Colors.Font,
	"background-color": &Prefs.Colors.Background,
}

func (vp *Viewport2D) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Viewport2D)
	vp.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	vp.Fill = fr.Fill
	vp.Geom = fr.Geom
}

// NewViewport2D creates a new Pixels Image with the specified width and height,
// and initializes the renderer etc
func NewViewport2D(width, height int) *Viewport2D {
	sz := image.Point{width, height}
	vp := &Viewport2D{
		Geom: Geom2DInt{Size: sz},
	}
	vp.Pixels = image.NewRGBA(image.Rectangle{Max: sz})
	vp.Render.Init(width, height, vp.Pixels)
	return vp
}

// Resize resizes the viewport, creating a new image -- updates Geom Size
func (vp *Viewport2D) Resize(nwsz image.Point) {
	if nwsz.X == 0 || nwsz.Y == 0 {
		return
	}
	if vp.Pixels != nil {
		ib := vp.Pixels.Bounds().Size()
		if ib == nwsz {
			vp.BBoxMu.Lock()
			vp.Geom.Size = nwsz // make sure
			vp.BBoxMu.Unlock()
			return // already good
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

// VpFlags extend NodeBase NodeFlags to hold viewport state
type VpFlags int

//go:generate stringer -type=VpFlags

var KiT_VpFlags = kit.Enums.AddEnumExt(KiT_NodeFlags, VpFlagsN, kit.BitFlag, nil)

const (
	// VpFlagPopup means viewport is a popup (menu or dialog) -- does not obey
	// parent bounds (otherwise does)
	VpFlagPopup VpFlags = VpFlags(NodeFlagsN) + iota

	// VpFlagMenu means viewport is serving as a popup menu -- affects how window
	// processes clicks
	VpFlagMenu

	// VpFlagCompleter means viewport is serving as a popup menu for code completion --
	// only applies if the VpFlagMenu is also set
	VpFlagCompleter

	// VpFlagCorrector means viewport is serving as a popup menu for spelling correction --
	// only applies if the VpFlagMenu is also set
	VpFlagCorrector

	// VpFlagTooltip means viewport is serving as a tooltip
	VpFlagTooltip

	// VpFlagPopupDestroyAll means that if this is a popup, then destroy all
	// the children when it is deleted -- otherwise children below the main
	// layout under the vp will not be destroyed -- it is up to the caller to
	// manage those (typically these are reusable assets)
	VpFlagPopupDestroyAll

	// VpFlagSVG means that this viewport is an SVG viewport -- SVG elements
	// look for this for re-rendering
	VpFlagSVG

	// VpFlagUpdatingNode means that this viewport is currently handling the
	// update of a node, and is under the UpdtMu mutex lock.
	// This can be checked to see about whether to add another update or not.
	VpFlagUpdatingNode

	// VpFlagNeedsFullRender means that this viewport needs to do a full
	// render -- this is set during signal processing and will preempt
	// other lower-level updates etc.
	VpFlagNeedsFullRender

	// VpFlagDoingFullRender means that this viewport is currently doing a
	// full render -- can be used by elements to drive deep rebuild in case
	// underlying data has changed.
	VpFlagDoingFullRender

	// VpFlagPrefSizing means that this viewport is currently doing a
	// PrefSize computation to compute the size of the viewport
	// (for sizing window for example) -- affects layout size computation
	// only for Over
	VpFlagPrefSizing

	VpFlagsN
)

func (vp *Viewport2D) IsPopup() bool {
	return vp.HasFlag(int(VpFlagPopup))
}

func (vp *Viewport2D) IsMenu() bool {
	return vp.HasFlag(int(VpFlagMenu))
}

func (vp *Viewport2D) IsCompleter() bool {
	return vp.HasFlag(int(VpFlagCompleter))
}

func (vp *Viewport2D) IsCorrector() bool {
	return vp.HasFlag(int(VpFlagCorrector))
}

func (vp *Viewport2D) IsTooltip() bool {
	return vp.HasFlag(int(VpFlagTooltip))
}

func (vp *Viewport2D) IsSVG() bool {
	return vp.HasFlag(int(VpFlagSVG))
}

func (vp *Viewport2D) IsUpdatingNode() bool {
	return vp.HasFlag(int(VpFlagUpdatingNode))
}

func (vp *Viewport2D) NeedsFullRender() bool {
	return vp.HasFlag(int(VpFlagNeedsFullRender))
}

func (vp *Viewport2D) IsDoingFullRender() bool {
	return vp.HasFlag(int(VpFlagDoingFullRender))
}

func (vp *Viewport2D) IsVisible() bool {
	if vp == nil || vp.This() == nil || vp.IsInvisible() {
		return false
	}
	return vp.This().(Viewport).VpIsVisible()
}

////////////////////////////////////////////////////////////////////////////////////////
//  Viewport interface implementation

func (vp *Viewport2D) VpTop() Viewport {
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

func (vp *Viewport2D) VpTopNode() Node {
	if vp.Win != nil {
		return vp.Win
	}
	return nil
}

func (vp *Viewport2D) VpTopUpdateStart() bool {
	if vp.Win != nil {
		return vp.Win.UpdateStart()
	}
	return false
}

func (vp *Viewport2D) VpTopUpdateEnd(updt bool) {
	if !updt {
		return
	}
	if vp.Win != nil {
		vp.Win.UpdateEnd(updt)
	}
}

// note: if not a standard viewport in a window, this method must be redefined!

func (vp *Viewport2D) VpEventMgr() *EventMgr {
	if vp.Win != nil {
		return &vp.Win.EventMgr
	}
	return nil
}

func (vp *Viewport2D) VpIsVisible() bool {
	if vp.Win == nil {
		return false
	}
	return vp.Win.IsVisible()
}

////////////////////////////////////////////////////////////////////////////////////////
//  Main Rendering code

// VpUploadAll is the update call for the main viewport for a window --
// calls UploadAllViewports in parent window, which uploads the main viewport
// and any active popups etc over the top of that
func (vp *Viewport2D) VpUploadAll() {
	if !vp.This().(Viewport).VpIsVisible() {
		return
	}
	vp.Win.UploadAllViewports()
}

// VpUploadVp uploads our viewport image into the parent window -- e.g., called
// by popups when updating separately
func (vp *Viewport2D) VpUploadVp() {
	if !vp.This().(Viewport).VpIsVisible() {
		return
	}
	vp.BBoxMu.RLock()
	vp.Win.UploadVp(vp, vp.WinBBox.Min)
	vp.BBoxMu.RUnlock()
}

// VpUploadRegion uploads node region of our viewport image
func (vp *Viewport2D) VpUploadRegion(vpBBox, winBBox image.Rectangle) {
	if !vp.This().(Viewport).VpIsVisible() {
		return
	}
	vp.Win.UploadVpRegion(vp, vpBBox, winBBox)
}

// set our window pointer to point to the current window we are under
func (vp *Viewport2D) SetCurWin() {
	pwin := vp.ParentWindow()
	if pwin != nil { // only update if non-nil -- otherwise we could be setting
		// temporarily to give access to DPI etc
		vp.Win = pwin
	}
}

// DrawIntoParent draws our viewport image into parent's image -- this is the
// typical way that a sub-viewport renders (e.g., svg boxes, icons, etc -- not popups)
func (vp *Viewport2D) DrawIntoParent(parVp *Viewport2D) {
	r := vp.Geom.Bounds()
	sp := image.ZP
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
	if Render2DTrace {
		fmt.Printf("Render: vp DrawIntoParent: %v parVp: %v rect: %v sp: %v\n", vp.Path(), parVp.Path(), r, sp)
	}
	draw.Draw(parVp.Pixels, r, vp.Pixels, sp, draw.Over)
}

// ReRender2DNode re-renders a specific node, including uploading updated bits to
// the window texture using Window.UploadVpRegion call.
// This should be covered by an outer UpdateStart / End bracket on Window to drive
// publishing changes, with suitable grouping if multiple updates
func (vp *Viewport2D) ReRender2DNode(gni Node2D) {
	if !vp.This().(Viewport).VpIsVisible() {
		return
	}
	gn := gni.AsNode2D()
	if Render2DTrace {
		fmt.Printf("Render: vp re-render: %v node: %v\n", vp.Path(), gn.Path())
	}
	// pr := prof.Start("vp.ReRender2DNode")
	gn.Render2DTree()
	// pr.End()
	gn.BBoxMu.RLock()
	wbb := gn.WinBBox
	gn.BBoxMu.RUnlock()
	vp.This().(Viewport).VpUploadRegion(gn.VpBBox, wbb)
}

// ReRender2DAnchor re-renders an anchor node -- the KEY diff from
// ReRender2DNode is that it calls ReRender2DTree and not just Render2DTree!
// uploads updated bits to the window texture using Window.UploadVpRegion call.
// This should be covered by an outer UpdateStart / End bracket on Window to drive
// publishing changes, with suitable grouping if multiple updates
func (vp *Viewport2D) ReRender2DAnchor(gni Node2D) {
	if !vp.This().(Viewport).VpIsVisible() {
		return
	}
	pw := gni.AsWidget()
	if pw == nil {
		return
	}
	if Render2DTrace {
		fmt.Printf("Render: vp anchor re-render: %v node: %v\n", vp.Path(), pw.Path())
	}
	// pr := prof.Start("vp.ReRender2DNode")
	pw.ReRender2DTree()
	// pr.End()
	pw.BBoxMu.RLock()
	wbb := pw.WinBBox
	pw.BBoxMu.RUnlock()
	vp.This().(Viewport).VpUploadRegion(pw.VpBBox, wbb)
}

// Delete this popup viewport -- has already been disconnected from window
// events and parent is nil -- called by window when a popup is deleted -- it
// destroys the vp and its main layout, see VpFlagPopupDestroyAll for whether
// children are destroyed
func (vp *Viewport2D) DeletePopup() {
	vp.Par = nil // disconnect from window -- it never actually owned us as a child
	vp.Win = nil
	if !vp.HasFlag(int(VpFlagPopupDestroyAll)) {
		// delete children of main layout prior to deleting the popup (e.g., menu items) so they don't get destroyed
		if len(vp.Kids) == 1 {
			cli, _ := KiToNode2D(vp.Child(0))
			ly := cli.AsLayout2D()
			if ly != nil {
				ly.DeleteChildren(ki.NoDestroyKids) // do NOT destroy children -- just delete them
			}
		}
	}
	vp.This().Destroy() // nuke everything else in us
}

////////////////////////////////////////////////////////////////////////////////////////
// Node2D interface

func (vp *Viewport2D) AsViewport2D() *Viewport2D {
	return vp
}

func (vp *Viewport2D) Init2D() {
	vp.Init2DWidget()
	vp.SetCurWin()
	// note: used to have a NodeSig update here but was redundant -- already handled.
	// also note that SVG viewports require SetNeedsFullRender to repaint!
}

func (vp *Viewport2D) Style2D() {
	vp.StyMu.Lock()
	defer vp.StyMu.Unlock()

	vp.SetCurWin()
	vp.Style2DWidget()
	vp.LayState.SetFromStyle(&vp.Sty.Layout) // also does reset
}

func (vp *Viewport2D) Size2D(iter int) {
	vp.InitLayout2D()
	// we listen to x,y styling for positioning within parent vp, if non-zero -- todo: only popup?
	pos := vp.Sty.Layout.PosDots().ToPoint()
	if pos != image.ZP {
		vp.Geom.Pos = pos
	}
	if !vp.IsSVG() && vp.Geom.Size != image.ZP {
		vp.LayState.Alloc.Size.SetPoint(vp.Geom.Size)
	}
}

func (vp *Viewport2D) Layout2D(parBBox image.Rectangle, iter int) bool {
	vp.Layout2DBase(parBBox, true, iter)
	return vp.Layout2DChildren(iter)
}

func (vp *Viewport2D) BBox2D() image.Rectangle {
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
		if sz != image.ZP {
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

func (vp *Viewport2D) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
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

func (vp *Viewport2D) ChildrenBBox2D() image.Rectangle {
	if vp.Pixels == nil {
		sz := vp.Geom.Size
		if sz != image.ZP {
			return vp.Geom.Bounds()
		}
		return image.Rectangle{Max: image.Point{100, 100}}
	}
	return vp.Pixels.Bounds() // vp.VpBBox -- this is where we transition to new coordinates!
}

// RenderViewport2D is the render action for the viewport itself -- either
// uploads image to window or draws into parent viewport
func (vp *Viewport2D) RenderViewport2D() {
	if vp.IsPopup() { // popup has a parent that is the window
		vp.SetCurWin()
		if Render2DTrace {
			fmt.Printf("Render: %v at Popup VpUploadVp\n", vp.Path())
		}
		vp.This().(Viewport).VpUploadVp()
	} else if vp.Viewport != nil { // sub-vp
		if Render2DTrace {
			fmt.Printf("Render: %v at %v DrawIntoParent\n", vp.Path(), vp.VpBBox)
		}
		vp.DrawIntoParent(vp.Viewport)
	} else { // we are the main vp
		if Render2DTrace {
			fmt.Printf("Render: %v at %v VpUploadAll\n", vp.Path(), vp.VpBBox)
		}
		vp.This().(Viewport).VpUploadAll()
	}
}

// FullRender2DTree is called by window and other places to completely
// re-render -- we set our flag when doing this so valueview elements (and
// anyone else) can do a deep re-build that is typically not otherwise needed
// (e.g., after non-signaling structs have updated)
func (vp *Viewport2D) FullRender2DTree() {
	if vp.IsUpdating() { // already in process!
		return
	}
	vp.SetFlag(int(VpFlagDoingFullRender))
	if Render2DTrace {
		fmt.Printf("Render: %v doing full render\n", vp.Path())
	}
	vp.WidgetBase.FullRender2DTree()
	vp.ClearFlag(int(VpFlagDoingFullRender))
}

// we use our own render for these -- Viewport member is our parent!
func (vp *Viewport2D) PushBounds() bool {
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
	if Render2DTrace {
		fmt.Printf("Render: %v at %v\n", vp.Path(), bb)
	}
	return true
}

func (vp *Viewport2D) PopBounds() {
	rs := &vp.Render
	rs.PopBounds()
}

func (vp *Viewport2D) Move2D(delta image.Point, parBBox image.Rectangle) {
	if vp == nil {
		return
	}
	vp.Move2DBase(delta, parBBox)
	vp.Move2DChildren(image.ZP) // reset delta here -- we absorb the delta in our placement relative to the parent
}

func (vp *Viewport2D) FillViewport() {
	vp.StyMu.RLock()
	st := &vp.Sty
	rs := &vp.Render
	rs.Lock()
	rs.Paint.FillBox(rs, mat32.Vec2Zero, mat32.NewVec2FmPoint(vp.Geom.Size), &st.Font.BgColor)
	rs.Unlock()
	vp.StyMu.RUnlock()
}

func (vp *Viewport2D) FullReRenderIfNeeded() bool {
	vpDoing := false
	if vp.Viewport != nil && vp.Viewport.IsDoingFullRender() {
		vpDoing = true
	}
	if vp.This().(Node2D).IsVisible() && vp.NeedsFullReRender() && !vpDoing {
		if Render2DTrace {
			fmt.Printf("Render: NeedsFullReRender for %v at %v\n", vp.Path(), vp.VpBBox)
		}
		vp.ClearFullReRender()
		vp.ReRender2DTree()
		return true
	}
	return false
}

func (vp *Viewport2D) Render2D() {
	if vp.FullReRenderIfNeeded() {
		return
	}
	if vp.PushBounds() {
		if vp.Fill {
			vp.FillViewport()
		}
		vp.Render2DChildren() // we must do children first, then us!
		vp.RenderViewport2D() // update our parent image
		vp.PopBounds()
	}
}

// PrefSize computes the preferred size of the viewport based on current contents.
// initSz is the initial size -- e.g., size of screen.
// Used for auto-sizing windows.
func (vp *Viewport2D) PrefSize(initSz image.Point) image.Point {
	vp.SetFlag(int(VpFlagPrefSizing))
	vp.Init2DTree()
	vp.Style2DTree() // sufficient to get sizes
	vp.LayState.Alloc.Size.SetPoint(initSz)
	vp.Size2DTree(0) // collect sizes
	vp.ClearFlag(int(VpFlagPrefSizing))
	ch := vp.ChildByType(KiT_Layout, ki.Embeds, 0).Embed(KiT_Layout).(*Layout)
	vpsz := ch.LayState.Size.Pref.ToPoint()
	// also take into account min size pref
	stw := int(vp.Sty.Layout.MinWidth.Dots)
	sth := int(vp.Sty.Layout.MinHeight.Dots)
	// fmt.Printf("dlg stw %v sth %v dpi %v vpsz: %v\n", stw, sth, dlg.Sty.UnContext.DPI, vpsz)
	vpsz.X = ints.MaxInt(vpsz.X, stw)
	vpsz.Y = ints.MaxInt(vpsz.Y, sth)
	return vpsz
}

////////////////////////////////////////////////////////////////////////////////////////
//  Signal Handling

// SignalViewport2D is called by each node in scenegraph through its NodeSig
// signal to notify its parent viewport whenever it changes, causing a
// re-render.
func SignalViewport2D(vpki, send ki.Ki, sig int64, data interface{}) {
	vpni, ok := vpki.(Node2D)
	if !ok {
		return
	}
	vp := vpni.AsViewport2D()
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
			log.Printf("Update: SignalViewport2D updating node %v with Updating flag set\n", ni.Path())
		}
		return
	}

	if Update2DTrace {
		fmt.Printf("Update: Viewport2D: %v NodeUpdated due to signal: %v from node: %v\n", vp.Path(), ki.NodeSignals(sig), send.Path())
	}

	vp.NodeUpdated(nii, sig, data)
}

// NodeUpdated is called from SignalViewport2D when a valid node's NodeSig sent a signal
// usually after UpdateEnd.
func (vp *Viewport2D) NodeUpdated(nii Node2D, sig int64, data interface{}) {
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
			vp.SetFlag(int(VpFlagNeedsFullRender))
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
func (vp *Viewport2D) UpdateLevel(nii Node2D, sig int64, data interface{}) (anchor Node2D, full bool) {
	ni := nii.AsNode2D()
	if sig == int64(ki.NodeSignalUpdated) {
		dflags := data.(int64)
		vlupdt := bitflag.HasAnyMask(dflags, ki.ValUpdateFlagsMask)
		strupdt := bitflag.HasAnyMask(dflags, ki.StruUpdateFlagsMask)
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
			fmt.Printf("Update: Viewport2D: %v FullRender2DTree (structural changes) for node: %v\n", vp.Path(), nii.Path())
		}
		anchor = ni.ParentReRenderAnchor()
		return anchor, full
	}
	return nil, false
}

// SetNeedsFullRender sets the flag indicating that a full render of the viewport is needed
// it will do this immediately pending acquisition of the lock and through the standard
// updating channels, unless already updating.
func (vp *Viewport2D) SetNeedsFullRender() {
	if !vp.NeedsFullRender() {
		vp.StackMu.Lock()
		vp.SetFlag(int(VpFlagNeedsFullRender))
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
func (vp *Viewport2D) BlockUpdates() {
	vp.UpdtMu.Lock()
	vp.SetFlag(int(VpFlagUpdatingNode))
	vp.UpdtMu.Unlock()
}

// UnblockUpdates unblocks updating of this viewport -- see BlockUpdates()
func (vp *Viewport2D) UnblockUpdates() {
	vp.UpdtMu.Lock()
	vp.ClearFlag(int(VpFlagUpdatingNode))
	vp.UpdtMu.Unlock()
}

// UpdateNodes processes the current update signals and actually does the relevant updating
func (vp *Viewport2D) UpdateNodes() {
	vp.UpdtMu.Lock()
	vp.SetFlag(int(VpFlagUpdatingNode))
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
			vp.ClearFlag(int(VpFlagNeedsFullRender))
			vp.StackMu.Unlock()
			if vp.Viewport == nil { // top level
				vp.FullRender2DTree()
			} else {
				vp.ReRender2DTree() // embedded
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
			vp.ReRender2DAnchor(nii)
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

	vp.ClearFlag(int(VpFlagUpdatingNode))
	vp.UpdtMu.Unlock()
}

// UpdateNode is called under UpdtMu lock and does the actual steps to update a given node
func (vp *Viewport2D) UpdateNode(nii Node2D) {
	if nii.DirectWinUpload() {
		if Update2DTrace {
			fmt.Printf("Update: Viewport2D: %v DirectWinUpload on %v\n", vp.Path(), nii.Path())
		}
	} else {
		if Update2DTrace {
			fmt.Printf("Update: Viewport2D: %v ReRender2D on %v\n", vp.Path(), nii.Path())
		}
		vp.ReRender2DNode(nii)
	}
}

//////////////////////////////////////////////////////////////////////////////////
//  Style state

// SetCurStyleNode sets the current styling node to given node, and nil to clear
func (vp *Viewport2D) SetCurStyleNode(node Node2D) {
	if vp == nil {
		return
	}
	vp.StyleMu.Lock()
	vp.CurStyleNode = node
	vp.StyleMu.Unlock()
}

// SetCurrentColor sets the current color in concurrent-safe way
func (vp *Viewport2D) SetCurrentColor(clr gist.Color) {
	if vp == nil {
		return
	}
	vp.StyleMu.Lock()
	vp.CurColor = clr
	vp.StyleMu.Unlock()
}

// ContextColor gets the current color in concurrent-safe way.
// Implements the gist.Context interface
func (vp *Viewport2D) ContextColor() gist.Color {
	if vp == nil {
		return gist.Color{}
	}
	vp.StyleMu.RLock()
	clr := vp.CurColor
	vp.StyleMu.RUnlock()
	return clr
}

// ContextColorSpecByURL finds a Node by an element name (URL-like path), and
// attempts to convert it to a Gradient -- if successful, returns ColorSpec on that.
// Used for colorspec styling based on url() value.
func (vp *Viewport2D) ContextColorSpecByURL(url string) *gist.ColorSpec {
	if vp == nil {
		return nil
	}
	vp.StyleMu.RLock()
	defer vp.StyleMu.RUnlock()

	if vp.CurStyleNode == nil {
		return nil
	}
	val := url[4:]
	val = strings.TrimPrefix(strings.TrimSuffix(val, ")"), "#")
	ne := vp.CurStyleNode.FindNamedElement(val)
	if grad, ok := ne.(*Gradient); ok {
		return &grad.Grad
	}
	return nil
}

//////////////////////////////////////////////////////////////////////////////////
//  Image utilities

// SavePNG encodes the image as a PNG and writes it to disk.
func (vp *Viewport2D) SavePNG(path string) error {
	return SavePNG(path, vp.Pixels)
}

// EncodePNG encodes the image as a PNG and writes it to the provided io.Writer.
func (vp *Viewport2D) EncodePNG(w io.Writer) error {
	return png.Encode(w, vp.Pixels)
}
