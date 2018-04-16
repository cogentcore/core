// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
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
	"os"

	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
)

// these extend NodeBase NodeFlags to hold viewport state
const (
	// viewport is a popup (menu or dialog) -- does not obey parent bounds (otherwise does)
	VpFlagPopup NodeFlags = NodeFlagsN + iota
	// viewport is serving as a popup menu -- affects how window processes clicks
	VpFlagMenu
	// if this is a popup, then destroy all the children when it is deleted -- otherwise children below the main layout under the vp will not be destroyed -- it is up to the caller to manage those (typically these are reusable assets)
	VpFlagPopupDestroyAll
	// this viewport is an SVG viewport -- determines the styling that it uses
	VpFlagSVG
	// draw into window directly instead of drawing into parent -- i.e., as a sprite
	VpFlagDrawIntoWin
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
	Node2DBase
	Fill    bool        `desc:"fill the viewport with background-color from style"`
	ViewBox ViewBox2D   `xml:"viewBox" desc:"viewbox within any parent Viewport2D"`
	Render  RenderState `json:"-" desc:"render state for rendering"`
	Pixels  *image.RGBA `json:"-" desc:"live pixels that we render into"`
	Win     *Window     `json:"-" desc:"our parent window that we render into"`
}

var KiT_Viewport2D = kit.Types.AddType(&Viewport2D{}, nil)

// NewViewport2D creates a new image.RGBA with the specified width and height
func NewViewport2D(width, height int) *Viewport2D {
	return NewViewport2DForRGBA(image.NewRGBA(image.Rect(0, 0, width, height)))
}

// NewViewport2DForImage copies the specified image into a new image.RGBA
// and prepares a context for rendering onto that image.
func NewViewport2DForImage(im image.Image) *Viewport2D {
	return NewViewport2DForRGBA(imageToRGBA(im))
}

// NewViewport2DForRGBA prepares a context for rendering onto the specified image.
// No copy is made.
func NewViewport2DForRGBA(im *image.RGBA) *Viewport2D {
	vp := &Viewport2D{
		ViewBox: ViewBox2D{Size: im.Bounds().Size()},
		Pixels:  im,
	}
	vp.Render.Defaults()
	vp.Render.Image = vp.Pixels
	return vp
}

// resize viewport, creating a new image (no point in trying to resize the image -- need to re-render) -- updates ViewBox Size too -- triggers update -- wrap in other UpdateStart/End calls as appropriate
func (vp *Viewport2D) Resize(width, height int) {
	nwsz := image.Point{width, height}
	if vp.Pixels != nil {
		ib := vp.Pixels.Bounds().Size()
		if ib == nwsz {
			vp.ViewBox.Size = nwsz // make sure
			return                 // already good
		}
	}
	vp.Pixels = image.NewRGBA(image.Rect(0, 0, width, height))
	vp.Render.Defaults()
	vp.ViewBox.Size = nwsz // make sure
	vp.Render.Image = vp.Pixels
	if vp.Viewport == nil { // parent
		vp.FullRender2DTree()
	}
}

func (vp *Viewport2D) IsPopup() bool {
	return bitflag.Has(vp.Flag, int(VpFlagPopup))
}

func (vp *Viewport2D) IsMenu() bool {
	return bitflag.Has(vp.Flag, int(VpFlagMenu))
}

func (vp *Viewport2D) IsSVG() bool {
	return bitflag.Has(vp.Flag, int(VpFlagSVG))
}

// set our window pointer to point to the current window we are under
func (vp *Viewport2D) SetCurWin() {
	vp.Win = vp.ParentWindow()
}

////////////////////////////////////////////////////////////////////////////////////////
//  Main Rendering code

// draw our image into parents -- called at right place in Render
func (vp *Viewport2D) DrawIntoParent(parVp *Viewport2D) {
	r := vp.ViewBox.Bounds()
	sp := image.ZP
	if vp.Par != nil { // use parents children bbox to determine where we can draw
		pgi, _ := KiToNode2D(vp.Par)
		nr := r.Intersect(pgi.ChildrenBBox2D())
		sp = nr.Min.Sub(r.Min)
		r = nr
	}
	draw.Draw(parVp.Pixels, r, vp.Pixels, sp, draw.Src)
}

// draw main viewport into window -- needs to redraw popups over top of it, so does a full update
func (vp *Viewport2D) DrawMainViewport() {
	if vp.Win == nil {
		return
	}
	vp.Win.FullUpdate()
}

// draw a vp into window directly -- for most non-main vp's
func (vp *Viewport2D) DrawIntoWindow() {
	if vp.Win == nil {
		return
	}
	vp.Win.UpdateStart()
	vp.Win.UpdateFullVpRegion(vp, vp.VpBBox, vp.WinBBox)
	vp.Win.UpdateEnd()
}

// draw main window vp into region of this vp
func (vp *Viewport2D) DrawMainVpOverMe() {
	if vp.Win == nil {
		return
	}
	vp.Win.UpdateStart()
	vp.Win.UpdateVpRegionFromMain(vp.WinBBox)
	vp.Win.UpdateEnd()
}

// Delete this viewport -- has already been disconnected from window events
// and parent is nil -- called by window when a popup is deleted -- it
// destroys the vp and its main layout, see VpFlagPopupDestroyAll for whether
// children are destroyed
func (vp *Viewport2D) DeletePopup() {
	vp.Par = nil // disconnect from window -- it never actually owned us as a child
	if !bitflag.Has(vp.Flag, int(VpFlagPopupDestroyAll)) {
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
	vp.SetCurWin()
	vp.Init2DBase()
	// we update oursleves whenever any node update event happens
	vp.NodeSig.Connect(vp.This, func(recvp, sendvp ki.Ki, sig int64, data interface{}) {
		rvpi, _ := KiToNode2D(recvp)
		rvp := rvpi.AsViewport2D()
		if Update2DTrace {
			fmt.Printf("Update: Viewport2D: %v full render due to signal: %v from node: %v\n", rvp.PathUnique(), ki.NodeSignals(sig), sendvp.PathUnique())
		}
		// todo: don't re-render if deleting!
		rvp.FullRender2DTree()
	})
}

func (vp *Viewport2D) Style2D() {
	vp.SetCurWin()
	vp.Style2DWidget(nil)
}

func (vp *Viewport2D) Size2D() {
	vp.InitLayout2D()
	// we listen to x,y styling for positioning within parent vp, if non-zero -- todo: only popup?
	pos := vp.Style.Layout.PosDots().ToPoint()
	if pos != image.ZP {
		vp.ViewBox.Min = pos
	}
	if !vp.IsSVG() && vp.ViewBox.Size != image.ZP {
		vp.LayData.AllocSize.SetPoint(vp.ViewBox.Size)
	}
}

func (vp *Viewport2D) Layout2D(parBBox image.Rectangle) {
	vp.Layout2DBase(parBBox, true)
	vp.Layout2DChildren()
}

func (vp *Viewport2D) BBox2D() image.Rectangle {
	// viewport ignores any parent parent bbox info!
	if vp.Pixels == nil || !vp.IsPopup() { // non-popups use allocated sizes via layout etc
		if !vp.LayData.AllocSize.IsZero() {
			asz := vp.LayData.AllocSize.ToPointCeil()
			vp.Resize(asz.X, asz.Y)
			// fmt.Printf("vp %v resized to %v\n", vp.Nm, asz)
		} else if vp.Pixels == nil {
			vp.Resize(64, 64) // gotta have something..
		}
	}
	return vp.Pixels.Bounds()
}

func (vp *Viewport2D) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	vp.VpBBox = vp.Pixels.Bounds()
	vp.SetWinBBox()    // this adds all PARENT offsets
	if !vp.IsPopup() { // non-popups use allocated positions
		vp.ViewBox.Min = vp.LayData.AllocPos.ToPointFloor()
	}
	vp.WinBBox = vp.WinBBox.Add(vp.ViewBox.Min)
	// fmt.Printf("Viewport: %v bbox: %v vpBBox: %v winBBox: %v\n", vp.PathUnique(), vp.BBox, vp.VpBBox, vp.WinBBox)
}

func (vp *Viewport2D) ChildrenBBox2D() image.Rectangle {
	return vp.VpBBox
}

func (vp *Viewport2D) RenderViewport2D() {
	if vp.IsPopup() { // popup has a parent that is the window
		vp.SetCurWin()
		if Render2DTrace {
			fmt.Printf("Render: %v at %v DrawPopup into Window\n", vp.PathUnique(), vp.VpBBox)
		}
		vp.DrawIntoWindow()
	} else if vp.Viewport != nil { // sub-vp
		if bitflag.Has(vp.Flag, int(VpFlagDrawIntoWin)) {
			if Render2DTrace {
				fmt.Printf("Render: %v at %v DrawIntoWindow\n", vp.PathUnique(), vp.VpBBox)
			}
			vp.DrawIntoWindow()
		} else {
			if Render2DTrace {
				fmt.Printf("Render: %v at %v DrawIntoParent\n", vp.PathUnique(), vp.VpBBox)
			}
			vp.DrawIntoParent(vp.Viewport)
		}
	} else { // we are the main vp
		if Render2DTrace {
			fmt.Printf("Render: %v at %v DrawMainViewport, full render\n", vp.PathUnique(), vp.VpBBox)
		}
		vp.DrawMainViewport()
	}
}

// we use our own render for these -- Viewport member is our parent!
func (vp *Viewport2D) PushBounds() bool {
	if vp.VpBBox.Empty() {
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
	rs.PushBounds(vp.VpBBox)
	if Render2DTrace {
		fmt.Printf("Render: %v at %v\n", vp.PathUnique(), vp.VpBBox)
	}
	return true
}

func (vp *Viewport2D) PopBounds() {
	rs := &vp.Render
	rs.PopBounds()
}

func (vp *Viewport2D) Move2D(delta image.Point, parBBox image.Rectangle) {
	vp.Move2DBase(delta, parBBox)
	vp.Move2DChildren(image.ZP) // reset delta here -- we absorb the delta in our placement relative to the parent
}

func (vp *Viewport2D) Render2D() {
	if vp.PushBounds() {
		if vp.Fill {
			pc := &vp.Paint
			rs := &vp.Render
			pc.FillStyle.SetColor(&vp.Style.Background.Color)
			pc.StrokeStyle.SetColor(nil)
			pc.DrawRectangle(rs, 0.0, 0.0, float64(vp.ViewBox.Size.X), float64(vp.ViewBox.Size.Y))
			pc.FillStrokeClear(rs)
		}
		vp.Render2DChildren() // we must do children first, then us!
		vp.RenderViewport2D() // update our parent image
		vp.PopBounds()
	}
}

func (vp *Viewport2D) ReRender2D() (node Node2D, layout bool) {
	node = vp.This.(Node2D)
	layout = false
	return
}

func (g *Viewport2D) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Viewport2D{}

////////////////////////////////////////////////////////////////////////////////////////
//  Signal Handling

// each node calls this signal method to notify its parent viewport whenever it changes, causing a re-render
func SignalViewport2D(vpki, node ki.Ki, sig int64, data interface{}) {
	vpgi, ok := vpki.(Node2D)
	if !ok {
		return
	}
	vp := vpgi.AsViewport2D()
	if vp == nil { // should not happen -- should only be called on viewports
		return
	}
	gii, gi := KiToNode2D(node)
	if gii == nil { // should not happen
		return
	}
	if gi.IsDeleted() || gi.IsDestroyed() { // skip these for sure
		return
	}
	if gi.UpdateCtr().Value() > 0 {
		log.Printf("ERROR SignalViewport2D updating node %v with Updating counter: %v\n", gi.PathUnique(), gi.UpdateCtr())
		return
	}

	if Update2DTrace {
		fmt.Printf("Update: Viewport2D: %v rendering due to signal: %v from node: %v\n", vp.PathUnique(), ki.NodeSignals(sig), node.PathUnique())
	}

	fullRend := false
	if sig == int64(ki.NodeSignalUpdated) {
		vlupdt := bitflag.HasMask(*(node.Flags()), ki.ValUpdateFlagsMask)
		strupdt := bitflag.HasMask(*(node.Flags()), ki.StruUpdateFlagsMask)
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
		vp.FullRender2DTree()
	} else {
		rr, layout := gii.ReRender2D()
		if rr != nil {
			rrn := rr.AsNode2D()
			if Update2DTrace {
				fmt.Printf("Update: Viewport2D: %v ReRender2D on %v, layout: %v\n", vp.PathUnique(), rrn.PathUnique(), layout)
			}
			if layout {
				rrn.Layout2DTree()
			}
			vp.ReRender2DNode(rr)
		} else {
			anchor := gi.ParentReRenderAnchor()
			if anchor != nil {
				// if Update2DTrace {
				fmt.Printf("Update: Viewport2D: %v ReRender2D nil, found anchor, styling: %v, then doing ReRender2DTree on: %v\n", vp.PathUnique(), gi.PathUnique(), anchor.PathUnique())
				// }
				gi.Style2DTree() // restyle only from affected node downward
				vp.ReRender2DAnchor(anchor.AsNode2D())
			} else {
				// if Update2DTrace {
				fmt.Printf("Update: Viewport2D: %v ReRender2D nil, styling: %v, then doing ReRender2DTree on us\n", vp.PathUnique(), gi.PathUnique())
				// }
				gi.Style2DTree()    // restyle only from affected node downward
				vp.ReRender2DTree() // need to re-render entirely from us
			}
		}
	}
	// don't do anything on deleting or destroying, and
}

////////////////////////////////////////////////////////////////////////////////////////
// Root-level Viewport API -- does all the recursive calls

// re-render a specific node that has said it can re-render
func (vp *Viewport2D) ReRender2DNode(gni Node2D) {
	gn := gni.AsNode2D()
	gn.Render2DTree()
	gnvp := gn.AsViewport2D()
	if !(gnvp != nil && bitflag.Has(gnvp.Flag, int(VpFlagDrawIntoWin))) {
		if vp.Win != nil {
			vp.Win.UpdateStart()
			vp.Win.UpdateVpRegion(vp, gn.VpBBox, gn.WinBBox)
			vp.Win.UpdateEnd()
		}
	}
}

// re-render a specific node that has said it can re-render
func (vp *Viewport2D) ReRender2DAnchor(gni Node2D) {
	gn := gni.AsNode2D()
	gn.ReRender2DTree()
	if vp.Win != nil {
		vp.Win.UpdateStart()
		vp.Win.UpdateVpRegion(vp, gn.VpBBox, gn.WinBBox)
		vp.Win.UpdateEnd()
	}
}

// SavePNG encodes the image as a PNG and writes it to disk.
func (vp *Viewport2D) SavePNG(path string) error {
	return SavePNG(path, vp.Pixels)
}

// EncodePNG encodes the image as a PNG and writes it to the provided io.Writer.
func (vp *Viewport2D) EncodePNG(w io.Writer) error {
	return png.Encode(w, vp.Pixels)
}

// todo:

// DrawPoint is like DrawCircle but ensures that a circle of the specified
// size is drawn regardless of the current transformation matrix. The position
// is still transformed, but not the shape of the point.
// func (vp *Viewport2D) DrawPoint(x, y, r float64) {
// 	pc := vp.PushNewPaint()
// 	p := pc.TransformPoint(x, y)
// 	pc.Identity()
// 	pc.DrawCircle(p.X, p.Y, r)
// 	vp.PopPaint()
// }

//////////////////////////////////////////////////////////////////////////////////
//  Image utilities

func LoadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	im, _, err := image.Decode(file)
	return im, err
}

func LoadPNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return png.Decode(file)
}

func SavePNG(path string, im image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, im)
}

func imageToRGBA(src image.Image) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Rect, src, image.ZP, draw.Src)
	return dst
}
