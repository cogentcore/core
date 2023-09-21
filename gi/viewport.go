// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"sync"

	"goki.dev/enums"
	"goki.dev/girl/girl"
	"goki.dev/girl/gist"
	"goki.dev/ki/v2"
)

// VpFlags has critical state information signaling when rendering,
// styling etc need to be done
type VpFlags int64 //enums:bitflag

const (
	// VpNeedsRender means nodes have flagged that they need a Render
	// and / or SetStyle update
	VpNeedsRender VpFlags = iota

	// VpNeedsFullRender means that this viewport needs to do a full
	// render: SetStyle, GetSize, DoLayout, then Render
	VpNeedsFullRender

	// VpIsRendering means viewport is in the process of rendering,
	// (or any other updating) -- do not trigger another render at this point.
	VpIsRendering

	// todo: remove below:?

	// VpPopupDestroyAll means that if this is a popup, then destroy all
	// the children when it is deleted -- otherwise children below the main
	// layout under the vp will not be destroyed -- it is up to the caller to
	// manage those (typically these are reusable assets)
	VpPopupDestroyAll

	// VpDoRebuild triggers extra rebuilding of elements during
	// Config and FullRender.
	VpDoRebuild

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
	Flags VpFlags `desc:"has critical state information signaling when rendering, styling etc need to be done, and also indicates type of viewport"`

	// fill the viewport with background-color from style
	Fill bool `desc:"fill the viewport with background-color from style"`

	// Viewport-level viewbox within any parent Viewport
	Geom girl.Geom2DInt `desc:"Viewport-level viewbox within any parent Viewport"`

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
	vp.RenderState.Init(nwsz.X, nwsz.Y, vp.Pixels)
	vp.Geom.Size = nwsz // make sure
	// fmt.Printf("vp %v resized to: %v, bounds: %v\n", vp.Path(), nwsz, vp.Pixels.Bounds())
}

// HasFlag checks if flag is set
// using atomic, safe for concurrent access
func (vp *Viewport) HasFlag(f enums.BitFlag) bool {
	return vp.Flags.HasFlag(f)
}

// SetFlag sets the given flag(s) to given state
// using atomic, safe for concurrent access
func (vp *Viewport) SetFlag(on bool, f ...enums.BitFlag) {
	vp.Flags.SetFlag(on, f...)
}

// note: if not a standard viewport in a window, this method must be redefined!
func (vp *Viewport) VpEventMgr() *EventMgr {
	if vp.Win != nil {
		return &vp.Win.EventMgr
	}
	return nil
}

func (vp *Viewport) VpIsVisible() bool {
	if vp.Win == nil || vp.Pixels == nil {
		return false
	}
	return vp.Win.IsVisible()
}

////////////////////////////////////////////////////////////////////////////////////////
//  Main Rendering code

// VpUploadRegion uploads node region of our viewport image
// func (vp *Viewport) VpUploadRegion(vpBBox, winBBox image.Rectangle) {
// 	if !vp.This().(Viewport).VpIsVisible() {
// 		return
// 	}
// 	vpin := vpBBox.Intersect(vp.Pixels.Bounds())
// 	if vpin.Empty() {
// 		return
// 	}
// 	vp.Win.UploadVpRegion(vp, vpin, winBBox)
// }

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

//////////////////////////////////////////////////////////////////
//  Image utilities

// SavePNG encodes the image as a PNG and writes it to disk.
func (vp *Viewport) SavePNG(path string) error {
	return SavePNG(path, vp.Pixels)
}

// EncodePNG encodes the image as a PNG and writes it to the provided io.Writer.
func (vp *Viewport) EncodePNG(w io.Writer) error {
	return png.Encode(w, vp.Pixels)
}
