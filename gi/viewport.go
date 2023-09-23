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

// see render.go for viewport-based rendering code.

// VpFlags has critical state information signaling when rendering,
// styling etc need to be done
type VpFlags int64 //enums:bitflag

const (
	// VpIsUpdating means viewport is in the process of updating:
	// set for any kind of tree-level update.
	// skip any further update passes until it goes off.
	VpIsUpdating VpFlags = iota

	// VpNeedsRender means nodes have flagged that they need a Render
	// update.
	VpNeedsRender

	// VpNeedsLayout means that this viewport needs DoLayout stack:
	// GetSize, DoLayout, then Render.  This is true after any Config.
	VpNeedsLayout

	// VpNeedsRebuild means that this viewport needs full Rebuild:
	// Config, Layout, Render with DoRebuild flag set
	// (e.g., after global style changes, zooming, etc)
	VpNeedsRebuild

	// todo: remove below:?

	// VpPopupDestroyAll means that if this is a popup, then destroy all
	// the children when it is deleted -- otherwise children below the main
	// layout under the vp will not be destroyed -- it is up to the caller to
	// manage those (typically these are reusable assets)
	VpPopupDestroyAll

	// VpRebuild triggers extra rebuilding of all elements during
	// Config, including all icons, sprites, cursors, etc.
	// Set by DoRebuild call.
	VpRebuild

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

// A Scene ALWAYS presents its children with a 0,0 - (Size.X, Size.Y)
// rendering area even if it is itself a child of another Scene.  This is
// necessary for rendering onto the image that it provides.  This creates
// challenges for managing the different geometries in a coherent way, e.g.,
// events come through the Window in terms of the root VP coords.  Thus, nodes
// require a  WinBBox for events and a VpBBox for their parent Scene.

/*

// Scene provides an interface for viewports,
// supporting overall management functions that can be
// provided by more embedded viewports for example.
type Scene interface {
	// VpTop returns the top-level Scene, which could be this one
	// or a higher one.  VpTopNode and VpEventMgr should be called on
	// on the Scene returned by this method.  For popups
	// this *not* the popup viewport but rather the window top viewport.
	VpTop() Scene

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
	// calls UploadAllScenes in parent window, which uploads the main viewport
	// and any active popups etc over the top of that
	VpUploadAll()

	// VpUploadVp uploads our viewport image into the parent window -- e.g., called
	// by popups when updating separately
	VpUploadVp()

	// VpUploadRegion uploads node region of our viewport image
	VpUploadRegion(vpBBox, winBBox image.Rectangle)
}
*/

// Scene contains a Widget tree, rooted in a Frame layout,
// which renders into its Pixels image.
type Scene struct {

	// name of viewport
	Name string `desc:"name of viewport"`

	// has critical state information signaling when rendering, styling etc need to be done, and also indicates type of viewport
	Flags VpFlags `desc:"has critical state information signaling when rendering, styling etc need to be done, and also indicates type of viewport"`

	// type of Scene
	Type VpType `desc:"type of Scene"`

	// Scene-level viewbox within any parent Scene
	Geom gist.Geom2DInt `desc:"Scene-level viewbox within any parent Scene"`

	// Root of the scenegraph for this viewport
	Frame Frame `desc:"Root of the scenegraph for this viewport"`

	// [view: -] render state for rendering
	RenderState girl.State `copy:"-" json:"-" xml:"-" view:"-" desc:"render state for rendering"`

	// [view: -] live pixels that we render into
	Pixels *image.RGBA `copy:"-" json:"-" xml:"-" view:"-" desc:"live pixels that we render into"`

	// our parent window that we render into
	Win *Window `copy:"-" json:"-" xml:"-" desc:"our parent window that we render into"`

	// background color for filling viewport -- defaults to transparent so that popups can have rounded corners
	BgColor gist.ColorSpec `desc:"background color for filling viewport -- defaults to transparent so that popups can have rounded corners"`

	// [view: -] Current color in styling -- used for relative color names
	CurColor color.RGBA `copy:"-" json:"-" xml:"-" view:"-" desc:"Current color in styling -- used for relative color names"`

	// [view: -] CurStyleNode is always set to the current node that is being styled used for finding url references -- only active during a Style pass
	// CurStyleNode WidgetD `copy:"-" json:"-" xml:"-" view:"-" desc:"CurStyleNode is always set to the current node that is being styled used for finding url references -- only active during a Style pass"`

	// [view: -] UpdtMu is mutex for viewport updates
	UpdtMu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" desc:"UpdtMu is mutex for viewport updates"`

	// [view: -] StackMu is mutex for adding to UpdtStack
	StackMu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" desc:"StackMu is mutex for adding to UpdtStack"`

	// [view: -] StyleMu is RW mutex protecting access to Style-related global vars
	StyleMu sync.RWMutex `copy:"-" json:"-" xml:"-" view:"-" desc:"StyleMu is RW mutex protecting access to Style-related global vars"`
}

// NewScene creates a new Pixels Image with the specified width and height,
// and initializes the renderer etc
func NewScene(width, height int) *Scene {
	sz := image.Point{width, height}
	sc := &Scene{
		Geom: gist.Geom2DInt{Size: sz},
	}
	sc.Pixels = image.NewRGBA(image.Rectangle{Max: sz})
	sc.RenderState.Init(width, height, sc.Pixels)
	sc.BgColor.SetColor(color.Transparent)
	sc.Frame.Lay = LayoutVert
	return sc
}

// Resize resizes the viewport, creating a new image -- updates Geom Size
func (sc *Scene) Resize(nwsz image.Point) {
	if nwsz.X == 0 || nwsz.Y == 0 {
		return
	}
	if sc.Pixels != nil {
		ib := sc.Pixels.Bounds().Size()
		if ib == nwsz {
			sc.Geom.Size = nwsz // make sure
			return              // already good
		}
	}
	if sc.Pixels != nil {
		sc.Pixels = nil
	}
	sc.Pixels = image.NewRGBA(image.Rectangle{Max: nwsz})
	sc.RenderState.Init(nwsz.X, nwsz.Y, sc.Pixels)
	sc.Geom.Size = nwsz // make sure
	// fmt.Printf("vp %v resized to: %v, bounds: %v\n", vp.Path(), nwsz, vp.Pixels.Bounds())
}

// HasFlag checks if flag is set
// using atomic, safe for concurrent access
func (sc *Scene) HasFlag(f enums.BitFlag) bool {
	return sc.Flags.HasFlag(f)
}

// SetFlag sets the given flag(s) to given state
// using atomic, safe for concurrent access
func (sc *Scene) SetFlag(on bool, f ...enums.BitFlag) {
	sc.Flags.SetFlag(on, f...)
}

// note: if not a standard viewport in a window, this method must be redefined!
func (sc *Scene) VpEventMgr() *EventMgr {
	if sc.Win != nil {
		return &sc.Win.EventMgr
	}
	return nil
}

func (sc *Scene) VpIsVisible() bool {
	if sc.Win == nil || sc.Pixels == nil {
		return false
	}
	return sc.Win.IsVisible()
}

// todo: remove

// VpUploadRegion uploads node region of our viewport image
// func (vp *Scene) VpUploadRegion(vpBBox, winBBox image.Rectangle) {
// 	if !vp.This().(Scene).VpIsVisible() {
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
func (sc *Scene) DeletePopup() {
	sc.Win = nil
	if sc.HasFlag(VpPopupDestroyAll) {
		sc.Frame.DeleteChildren(ki.DestroyKids)
	} else {
		// delete children of main layout prior to deleting the popup
		// (e.g., menu items) so they don't get destroyed
		sc.Frame.DeleteChildren(ki.NoDestroyKids) // do NOT destroy children -- just delete them
	}
}

// SetCurrentColor sets the current color in concurrent-safe way
func (sc *Scene) SetCurrentColor(clr color.RGBA) {
	if sc == nil {
		return
	}
	sc.StyleMu.Lock()
	sc.CurColor = clr
	sc.StyleMu.Unlock()
}

// ContextColor gets the current color in concurrent-safe way.
// Implements the gist.Context interface
func (sc *Scene) ContextColor() color.RGBA {
	if sc == nil {
		return color.RGBA{}
	}
	sc.StyleMu.RLock()
	clr := sc.CurColor
	sc.StyleMu.RUnlock()
	return clr
}

// ContextColorSpecByURL finds a Node by an element name (URL-like path), and
// attempts to convert it to a Gradient -- if successful, returns ColorSpec on that.
// Used for colorspec styling based on url() value.
func (sc *Scene) ContextColorSpecByURL(url string) *gist.ColorSpec {
	// todo: not currently supported -- see if needed for html / glide
	return nil
}

//////////////////////////////////////////////////////////////////
//  Image utilities

// SavePNG encodes the image as a PNG and writes it to disk.
func (sc *Scene) SavePNG(path string) error {
	return SavePNG(path, sc.Pixels)
}

// EncodePNG encodes the image as a PNG and writes it to the provided io.Writer.
func (sc *Scene) EncodePNG(w io.Writer) error {
	return png.Encode(w, sc.Pixels)
}
