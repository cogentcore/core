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

// see render.go for scene-based rendering code.

// Scene contains a Widget tree, rooted in a Frame layout,
// which renders into its Pixels image.
type Scene struct {

	// name of scene.  User-created scenes can be stored in the global SceneLibrary by name, in which case they must be unique.
	Name string `desc:"name of scene.  User-created scenes can be stored in the global SceneLibrary by name, in which case they must be unique."`

	// title of the Stage -- generally auto-set based on Scene Title.  used for title of Window and Dialog types
	Title string `desc:"title of the Stage -- generally auto-set based on Scene Title.  used for title of Window and Dialog types"`

	// has critical state information signaling when rendering, styling etc need to be done, and also indicates type of scene
	Flags ScFlags `desc:"has critical state information signaling when rendering, styling etc need to be done, and also indicates type of scene"`

	// Scene-level viewbox within any parent Scene
	Geom gist.Geom2DInt `desc:"Scene-level viewbox within any parent Scene"`

	// Root of the scenegraph for this scene
	Frame Frame `desc:"Root of the scenegraph for this scene"`

	// Extra decoration, configured by the outer Stage container.  Can be positioned anywhere -- typically uses LayoutNil
	Decor Layout `desc:"Extra decoration, configured by the outer Stage container.  Can be positioned anywhere -- typically uses LayoutNil"`

	// [view: -] render state for rendering
	RenderState girl.State `copy:"-" json:"-" xml:"-" view:"-" desc:"render state for rendering"`

	// [view: -] live pixels that we render into
	Pixels *image.RGBA `copy:"-" json:"-" xml:"-" view:"-" desc:"live pixels that we render into"`

	// todo: remove below:

	// event manager for this scene
	EventMgr EventMgr `copy:"-" json:"-" xml:"-" desc:"event manager for this scene"`

	// our parent window that we render into
	Win *OSWin `copy:"-" json:"-" xml:"-" desc:"our parent window that we render into"`

	// background color for filling scene -- defaults to transparent so that popups can have rounded corners
	BgColor gist.ColorSpec `desc:"background color for filling scene -- defaults to transparent so that popups can have rounded corners"`

	// [view: -] Current color in styling -- used for relative color names
	CurColor color.RGBA `copy:"-" json:"-" xml:"-" view:"-" desc:"Current color in styling -- used for relative color names"`

	// [view: -] CurStyleNode is always set to the current node that is being styled used for finding url references -- only active during a Style pass
	// CurStyleNode WidgetD `copy:"-" json:"-" xml:"-" view:"-" desc:"CurStyleNode is always set to the current node that is being styled used for finding url references -- only active during a Style pass"`

	// [view: -] UpdtMu is mutex for scene updates
	UpdtMu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" desc:"UpdtMu is mutex for scene updates"`

	// [view: -] StackMu is mutex for adding to UpdtStack
	StackMu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" desc:"StackMu is mutex for adding to UpdtStack"`

	// [view: -] StyleMu is RW mutex protecting access to Style-related global vars
	StyleMu sync.RWMutex `copy:"-" json:"-" xml:"-" view:"-" desc:"StyleMu is RW mutex protecting access to Style-related global vars"`
}

// NewScene creates a new Scene with Pixels Image
// of the specified width and height.
func NewScene(width, height int) *Scene {
	sz := image.Point{width, height}
	sc := &Scene{
		Geom: gist.Geom2DInt{Size: sz},
	}
	sc.Pixels = image.NewRGBA(image.Rectangle{Max: sz})
	sc.RenderState.Init(width, height, sc.Pixels)
	sc.BgColor.SetColor(color.Transparent)
	sc.Frame.Lay = LayoutVert
	sc.Decor.Lay = LayoutNil
	return sc
}

// Resize resizes the scene, creating a new image -- updates Geom Size
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

func (sc *Scene) ScIsVisible() bool {
	if sc.Win == nil || sc.Pixels == nil {
		return false
	}
	return sc.Win.IsVisible()
}

// todo: remove

// ScUploadRegion uploads node region of our scene image
// func (vp *Scene) ScUploadRegion(vpBBox, winBBox image.Rectangle) {
// 	if !vp.This().(Scene).ScIsVisible() {
// 		return
// 	}
// 	vpin := vpBBox.Intersect(vp.Pixels.Bounds())
// 	if vpin.Empty() {
// 		return
// 	}
// 	vp.Win.UploadScRegion(vp, vpin, winBBox)
// }

// Delete this popup scene -- has already been disconnected from window
// events and parent is nil -- called by window when a popup is deleted -- it
// destroys the vp and its main layout, see ScPopupDestroyAll for whether
// children are destroyed
func (sc *Scene) DeletePopup() {
	sc.Win = nil
	if sc.HasFlag(ScPopupDestroyAll) {
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

// ScFlags has critical state information signaling when rendering,
// styling etc need to be done
type ScFlags int64 //enums:bitflag

const (
	// ScIsUpdating means scene is in the process of updating:
	// set for any kind of tree-level update.
	// skip any further update passes until it goes off.
	ScIsUpdating ScFlags = iota

	// ScNeedsRender means nodes have flagged that they need a Render
	// update.
	ScNeedsRender

	// ScNeedsLayout means that this scene needs DoLayout stack:
	// GetSize, DoLayout, then Render.  This is true after any Config.
	ScNeedsLayout

	// ScNeedsRebuild means that this scene needs full Rebuild:
	// Config, Layout, Render with DoRebuild flag set
	// (e.g., after global style changes, zooming, etc)
	ScNeedsRebuild

	// ScRebuild triggers extra rebuilding of all elements during
	// Config, including all icons, sprites, cursors, etc.
	// Set by DoRebuild call.
	ScRebuild

	// todo: rename below:

	// ScPrefSizing means that this scene is currently doing a
	// PrefSize computation to compute the size of the scene
	// (for sizing window for example) -- affects layout size computation
	// only for Over
	ScPrefSizing

	// todo: remove below:?

	// ScPopupDestroyAll means that if this is a popup, then destroy all
	// the children when it is deleted -- otherwise children below the main
	// layout under the vp will not be destroyed -- it is up to the caller to
	// manage those (typically these are reusable assets)
	ScPopupDestroyAll
)
