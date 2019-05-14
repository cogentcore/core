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
	Render       RenderState  `json:"-" xml:"-" view:"-" desc:"render state for rendering"`
	Pixels       *image.RGBA  `json:"-" xml:"-" view:"-" desc:"live pixels that we render into"`
	Win          *Window      `json:"-" xml:"-" desc:"our parent window that we render into"`
	CurStyleNode Node2D       `json:"-" xml:"-" view:"-" desc:"CurStyleNode2D is always set to the current node that is being styled used for finding url references -- only active during a Style pass"`
	CurColor     Color        `json:"-" xml:"-" view:"-" desc:"CurColor is automatically updated from the Color setting of a Style and accessible as a color name in any other style as currentcolor use accessor routines for concurrent-safe access"`
	StyleMu      sync.RWMutex `json:"-" xml:"-" view:"-" desc:"StyleMu is RW mutex protecting access to Style-related global vars"`
}

var KiT_Viewport2D = kit.Types.AddType(&Viewport2D{}, Viewport2DProps)

var Viewport2DProps = ki.Props{
	"color":            &Prefs.Colors.Font,
	"background-color": &Prefs.Colors.Background,
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

// VpFlag flags extend NodeBase NodeFlags to hold viewport state
const (
	// VpFlagPopup means viewport is a popup (menu or dialog) -- does not obey
	// parent bounds (otherwise does)
	VpFlagPopup NodeFlags = NodeFlagsN + iota

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

	// VpFlagDoingFullRender means that this viewport is currently doing a
	// full render -- can be used by elements to drive deep rebuild in case
	// underlying data has changed.
	VpFlagDoingFullRender

	VpFlagN
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
	if vp.IsOverlay() { // don't check for any parent bounds etc -- just draw entire pixels
		if parVp == nil {
			return
		}
		r := vp.Pixels.Bounds()
		pos := vp.LayData.AllocPos.ToPoint() // get updated pos
		r = r.Add(pos)
		draw.Draw(parVp.Pixels, r, vp.Pixels, image.ZP, draw.Over)
		return
	}
	r := vp.Geom.Bounds()
	sp := image.ZP
	if vp.Par != nil { // use parents children bbox to determine where we can draw
		pni, _ := KiToNode2D(vp.Par)
		nr := r.Intersect(pni.ChildrenBBox2D())
		sp = nr.Min.Sub(r.Min)
		r = nr
	}
	if Render2DTrace {
		fmt.Printf("Render: vp DrawIntoParent: %v parVp: %v rect: %v sp: %v\n", vp.PathUnique(), parVp.PathUnique(), r, sp)
	}
	draw.Draw(parVp.Pixels, r, vp.Pixels, sp, draw.Over)
}

// ReRender2DNode re-renders a specific node
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
	if vp.Win != nil {
		updt := vp.Win.UpdateStart()
		vp.Win.UploadVpRegion(vp, gn.VpBBox, gn.WinBBox)
		vp.Win.UpdateEnd(updt)
	}
}

// ReRender2DAnchor re-renders an anchor node -- the KEY diff from
// ReRender2DNode is that it calls ReRender2DTree and not just Render2DTree!
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
	if vp.Win != nil {
		updt := vp.Win.UpdateStart()
		vp.Win.UploadVpRegion(vp, pw.VpBBox, pw.WinBBox)
		vp.Win.UpdateEnd(updt)
	}
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
			if vp.Viewport == nil {
				rvp.FullRender2DTree()
			} else {
				rvp.ReRender2DTree()
			}
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
			if !vp.LayData.AllocSize.IsZero() {
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
	if vp.IsOverlay() {
		return
	}
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
	if vp.IsOverlay() {
		if vp.Viewport != nil {
			vp.Viewport.Render.PushBounds(vp.Viewport.Pixels.Bounds())
		}
		return true
	}
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
	if vp.IsOverlay() {
		if vp.Viewport != nil {
			vp.Viewport.Render.PopBounds()
		}
		return
	}
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
	rs.Paint.FillBox(&vp.Render, Vec2DZero, NewVec2DFmPoint(vp.Geom.Size), &vp.Sty.Font.BgColor)
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

// SignalViewport2D is called by each node in scenegraph through its UpdateSig
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
		fmt.Printf("Update: Viewport2D: %v rendering (next line has specifics) due to signal: %v from node: %v\n", vp.PathUnique(), ki.NodeSignals(sig), send.PathUnique())
	}

	fullRend := false
	if sig == int64(ki.NodeSignalUpdated) {
		dflags := data.(int64)
		vlupdt := bitflag.HasAnyMask(dflags, ki.ValUpdateFlagsMask)
		strupdt := bitflag.HasAnyMask(dflags, ki.StruUpdateFlagsMask)
		if vlupdt && !strupdt {
			fullRend = false
		} else if strupdt {
			fullRend = true
		}
	} else {
		fullRend = true
	}

	if fullRend {
		if Update2DTrace {
			fmt.Printf("Update: Viewport2D: %v FullRender2DTree (structural changes)\n", vp.PathUnique())
		}
		anchor := ni.ParentReRenderAnchor()
		if anchor != nil {
			vp.ReRender2DAnchor(anchor)
		} else {
			vp.FullRender2DTree()
		}
	} else {
		if ni.NeedsFullReRender() {
			ni.ClearFullReRender()
			anchor := ni.ParentReRenderAnchor()
			if anchor != nil {
				if Update2DTrace {
					fmt.Printf("Update: Viewport2D: %v ReRender2D nil, found anchor, styling: %v, then doing ReRender2DTree on: %v\n", vp.PathUnique(), ni.PathUnique(), anchor.PathUnique())
				}
				vp.ReRender2DAnchor(anchor)
			} else {
				if Update2DTrace {
					fmt.Printf("Update: Viewport2D: %v ReRender2D nil, styling: %v, then doing ReRender2DTree on us\n", vp.PathUnique(), ni.PathUnique())
				}
				vp.ReRender2DTree() // need to re-render entirely from us
			}
		} else {
			if nii.DirectWinUpload() {
				if Update2DTrace {
					fmt.Printf("Update: Viewport2D: %v DirectWinUpload on %v\n", vp.PathUnique(), ni.PathUnique())
				}
			} else {
				if Update2DTrace {
					fmt.Printf("Update: Viewport2D: %v ReRender2D on %v\n", vp.PathUnique(), ni.PathUnique())
				}
				vp.ReRender2DNode(nii)
			}
		}
	}
	// don't do anything on deleting or destroying, and
}

////////////////////////////////////////////////////////////////////////////////////////
//  Overlay rendering

// AddOverlay adds the given node as an overlay to be rendered on top of the
// main window viewport -- node must already be initialized as a Ki element
// (e.g., call ki.InitName) -- typically it is a Bitmap and should have
// the bitmap pixels set already.  Sets overlay flag and calls init and style
func (vp *Viewport2D) AddOverlay(nii Node2D) {
	nii.AsNode2D().SetAsOverlay()
	vp.AddChild(nii)
	nii.Init2D()
	nii.Style2D()
}

// RenderOverlays is main call from window for OverlayVp to render overlay nodes within it
func (vp *Viewport2D) RenderOverlays(wsz image.Point) {
	vp.SetAsOverlay()
	vp.Resize(wsz)

	if len(vp.Kids) > 0 {
		// fill to transparent
		draw.Draw(vp.Pixels, vp.Pixels.Bounds(), &image.Uniform{color.Transparent}, image.ZP, draw.Src)

		// just do top-level objects here
		for _, k := range vp.Kids {
			nii, ni := KiToNode2D(k)
			if nii == nil {
				continue
			}
			if !ni.IsOverlay() { // has not been initialized
				ni.SetAsOverlay()
				nii.Init2D()
				nii.Style2D()
			}
			// note: skipping sizing, layout -- use simple elements here, esp Bitmap
			nii.Render2D()
		}
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
