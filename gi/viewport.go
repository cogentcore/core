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
	"sync"

	"github.com/goki/gi/mat32"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"
)

// A Viewport ALWAYS presents its children with a 0,0 - (Size.X, Size.Y)
// rendering area even if it is itself a child of another Viewport.  This is
// necessary for rendering onto the image that it provides.  This creates
// challenges for managing the different geometries in a coherent way, e.g.,
// events come through the Window in terms of the root VP coords.  Thus, nodes
// require a  WinBBox for events and a VpBBox for their parent Viewport.

// Viewport2D provides an image and a stack of Paint contexts for drawing onto the image
// with a convenience forwarding of the Paint methods operating on the current Paint
type Viewport2D struct {
	WidgetBase
	Fill         bool         `desc:"fill the viewport with background-color from style"`
	Geom         Geom2DInt    `desc:"Viewport-level viewbox within any parent Viewport2D"`
	Render       RenderState  `copy:"-" json:"-" xml:"-" view:"-" desc:"render state for rendering"`
	Pixels       *image.RGBA  `copy:"-" json:"-" xml:"-" view:"-" desc:"live pixels that we render into"`
	Win          *Window      `copy:"-" json:"-" xml:"-" desc:"our parent window that we render into"`
	CurStyleNode Node2D       `copy:"-" json:"-" xml:"-" view:"-" desc:"CurStyleNode2D is always set to the current node that is being styled used for finding url references -- only active during a Style pass"`
	CurColor     Color        `copy:"-" json:"-" xml:"-" view:"-" desc:"CurColor is automatically updated from the Color setting of a Style and accessible as a color name in any other style as currentcolor use accessor routines for concurrent-safe access"`
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
	// fmt.Printf("vp %v resized to: %v, bounds: %v\n", vp.PathUnique(), nwsz, vp.Pixels.Bounds())
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
	if vp == nil || vp.This() == nil || vp.IsInvisible() || vp.Win == nil {
		return false
	}
	return vp.Win.IsVisible()
}

// set our window pointer to point to the current window we are under
func (vp *Viewport2D) SetCurWin() {
	pwin := vp.ParentWindow()
	if pwin != nil { // only update if non-nil -- otherwise we could be setting
		// temporarily to give access to DPI etc
		vp.Win = pwin
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  Main Rendering code

// UploadMainToWin is the update call for the main viewport for a window --
// calls UploadAllViewports in parent window, which uploads the main viewport
// and any active popups etc over the top of that
func (vp *Viewport2D) UploadMainToWin() {
	if vp.Win == nil {
		return
	}
	vp.Win.UploadAllViewports()
}

// UploadToWin uploads our viewport image into the parent window -- e.g., called
// by popups when updating separately
func (vp *Viewport2D) UploadToWin() {
	if vp.Win == nil {
		return
	}
	vp.Win.UploadVp(vp, vp.WinBBox.Min)
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
		fmt.Printf("Render: vp DrawIntoParent: %v parVp: %v rect: %v sp: %v\n", vp.PathUnique(), parVp.PathUnique(), r, sp)
	}
	draw.Draw(parVp.Pixels, r, vp.Pixels, sp, draw.Over)
}

// ReRender2DNode re-renders a specific node, including uploading updated bits to
// the window texture using Window.UploadVpRegion call.
// This should be covered by an outer UpdateStart / End bracket on Window to drive
// publishing changes, with suitable grouping if multiple updates
func (vp *Viewport2D) ReRender2DNode(gni Node2D) {
	if vp.Win == nil || vp.Win.IsClosed() || vp.Win.IsResizing() { // no node-triggered updates during resize..
		return
	}
	gn := gni.AsNode2D()
	if Render2DTrace {
		fmt.Printf("Render: vp re-render: %v node: %v\n", vp.PathUnique(), gn.PathUnique())
	}
	pr := prof.Start("vp.ReRender2DNode")
	gn.Render2DTree()
	pr.End()
	vp.Win.UploadVpRegion(vp, gn.VpBBox, gn.WinBBox)
}

// ReRender2DAnchor re-renders an anchor node -- the KEY diff from
// ReRender2DNode is that it calls ReRender2DTree and not just Render2DTree!
// uploads updated bits to the window texture using Window.UploadVpRegion call.
// This should be covered by an outer UpdateStart / End bracket on Window to drive
// publishing changes, with suitable grouping if multiple updates
func (vp *Viewport2D) ReRender2DAnchor(gni Node2D) {
	if vp.Win == nil || vp.Win.IsClosed() || vp.Win.IsResizing() { // no node-triggered updates during resize..
		return
	}
	pw := gni.AsWidget()
	if pw == nil {
		return
	}
	if Render2DTrace {
		fmt.Printf("Render: vp anchor re-render: %v node: %v\n", vp.PathUnique(), pw.PathUnique())
	}
	pr := prof.Start("vp.ReRender2DNode")
	pw.ReRender2DTree()
	pr.End()
	vp.Win.UploadVpRegion(vp, pw.VpBBox, pw.WinBBox)
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
				ly.DeleteChildren(false) // do NOT destroy children -- just delete them
			}
		}
	}
	vp.Destroy() // nuke everything else in us
}

////////////////////////////////////////////////////////////////////////////////////////
// Node2D interface

func (vp *Viewport2D) AsViewport2D() *Viewport2D {
	return vp
}

func (vp *Viewport2D) Init2D() {
	vp.Init2DWidget()
	vp.SetCurWin()
	// we update ourselves whenever any node update event happens
	vp.NodeSig.Connect(vp.This(), func(recvp, sendvp ki.Ki, sig int64, data interface{}) {
		rvpi, _ := KiToNode2D(recvp)
		rvp := rvpi.AsViewport2D()
		if Update2DTrace {
			fmt.Printf("Update: Viewport2D: %v full render due to signal: %v from node: %v\n", rvp.PathUnique(), ki.NodeSignals(sig), sendvp.PathUnique())
		}
		if !vp.IsDeleted() && !vp.IsDestroyed() {
			vp.SetNeedsFullRender()
		}
	})
}

func (vp *Viewport2D) Style2D() {
	vp.SetCurWin()
	vp.Style2DWidget()
	vp.LayData.SetFromStyle(&vp.Sty.Layout) // also does reset
}

func (vp *Viewport2D) Size2D(iter int) {
	vp.InitLayout2D()
	// we listen to x,y styling for positioning within parent vp, if non-zero -- todo: only popup?
	pos := vp.Sty.Layout.PosDots().ToPoint()
	if pos != image.ZP {
		vp.Geom.Pos = pos
	}
	if !vp.IsSVG() && vp.Geom.Size != image.ZP {
		vp.LayData.AllocSize.SetPoint(vp.Geom.Size)
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
			if !vp.LayData.AllocSize.IsNil() {
				asz := vp.LayData.AllocSize.ToPointCeil()
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
		vp.Geom.Pos = vp.LayData.AllocPos.ToPointFloor()
	}
	if vp.Viewport == nil {
		vp.WinBBox = vp.WinBBox.Add(vp.Geom.Pos)
	}
	// fmt.Printf("Viewport: %v bbox: %v vpBBox: %v winBBox: %v\n", vp.PathUnique(), vp.BBox, vp.VpBBox, vp.WinBBox)
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
			fmt.Printf("Render: %v at Popup UploadToWin\n", vp.PathUnique())
		}
		vp.UploadToWin()
	} else if vp.Viewport != nil { // sub-vp
		if Render2DTrace {
			fmt.Printf("Render: %v at %v DrawIntoParent\n", vp.PathUnique(), vp.VpBBox)
		}
		vp.DrawIntoParent(vp.Viewport)
	} else { // we are the main vp
		if Render2DTrace {
			fmt.Printf("Render: %v at %v UploadMainToWin\n", vp.PathUnique(), vp.VpBBox)
		}
		vp.UploadMainToWin()
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
		fmt.Printf("Render: %v doing full render\n", vp.PathUnique())
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
		wbi := vp.WinBBox.Intersect(vp.Viewport.WinBBox)
		if wbi.Empty() {
			// fmt.Printf("not rendering vp %v bc empty winbox -- ours: %v par: %v\n", vp.Nm, vp.WinBBox, vp.Viewport.WinBBox)
			return false
		}
	}
	rs := &vp.Render
	bb := vp.Pixels.Bounds() // our bounds.. not vp.VpBBox)
	rs.PushBounds(bb)
	if Render2DTrace {
		fmt.Printf("Render: %v at %v\n", vp.PathUnique(), bb)
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
	rs := &vp.Render
	rs.Lock()
	rs.Paint.FillBox(&vp.Render, mat32.Vec2Zero, mat32.NewVec2FmPoint(vp.Geom.Size), &vp.Sty.Font.BgColor)
	rs.Unlock()
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
		log.Printf("ERROR SignalViewport2D updating node %v with Updating flag set\n", ni.PathUnique())
		return
	}

	if Update2DTrace {
		fmt.Printf("Update: Viewport2D: %v NodeUpdated due to signal: %v from node: %v\n", vp.PathUnique(), ki.NodeSignals(sig), send.PathUnique())
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
	if ni.NeedsFullReRender() {
		ni.ClearFullReRender()
		full = true
	}
	if full {
		if Update2DTrace {
			fmt.Printf("Update: Viewport2D: %v FullRender2DTree (structural changes) for node: %v\n", vp.PathUnique(), nii.PathUnique())
		}
		anchor = ni.ParentReRenderAnchor()
		return anchor, full
	}
	return nil, false
}

// SetNeedsFullRender sets the flag indicating that a full render of the viewport is needed
// it will do this immediately pending aquisition of the lock and through the standard
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
	// vp.UpdtMu.Lock()
}

// UnblockUpdates unblocks updating of this viewport -- see BlockUpdates()
func (vp *Viewport2D) UnblockUpdates() {
	// vp.UpdtMu.Unlock()
}

// UpdateNodes processes the current update signals and actually does the relevant updating
func (vp *Viewport2D) UpdateNodes() {
	vp.UpdtMu.Lock()
	vp.SetFlag(int(VpFlagUpdatingNode))
	if vp.Win != nil {
		wupdt := vp.Win.UpdateStart()
		defer vp.Win.UpdateEnd(wupdt)
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
			fmt.Printf("Update: Viewport2D: %v DirectWinUpload on %v\n", vp.PathUnique(), nii.PathUnique())
		}
	} else {
		if Update2DTrace {
			fmt.Printf("Update: Viewport2D: %v ReRender2D on %v\n", vp.PathUnique(), nii.PathUnique())
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

// CurStyleNodeNamedEl finds element of given name (using FindNamedElement method)
// in current style node, if set -- returns nil if not set or not found.
func (vp *Viewport2D) CurStyleNodeNamedEl(name string) Node2D {
	if vp == nil {
		return nil
	}
	vp.StyleMu.RLock()
	defer vp.StyleMu.RUnlock()

	if vp.CurStyleNode == nil {
		return nil
	}
	ne := vp.CurStyleNode.FindNamedElement(name)
	return ne
}

// SetCurrentColor sets the current color in concurrent-safe way
func (vp *Viewport2D) SetCurrentColor(clr Color) {
	if vp == nil {
		return
	}
	vp.StyleMu.Lock()
	vp.CurColor = clr
	vp.StyleMu.Unlock()
}

// CurrentColor gets the current color in concurrent-safe way
func (vp *Viewport2D) CurrentColor() Color {
	if vp == nil {
		return Color{}
	}
	vp.StyleMu.RLock()
	clr := vp.CurColor
	vp.StyleMu.RUnlock()
	return clr
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
