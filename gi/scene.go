// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"sync"

	"goki.dev/colors"
	"goki.dev/girl/paint"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// see:
//	- render.go for scene-based rendering code
//	- widglayout for layout
//	- style.go for style

// Scene contains a Widget tree, rooted in an embedded Frame layout,
// which renders into its Pixels image.
// The Scene is set in a Stage (pointer retained in Scene).
// Stage has a StageMgr manager for controlling things like Popups
// (Menus and Dialogs, etc).
//
// Each Scene and Widget tree contains state specific to its particular usage
// within a given Stage and overall rendering context (e.g., bounding boxes
// and pointer to current parent Stage), so
type Scene struct {
	Frame

	// name of scene.  User-created scenes can be stored in the global SceneLibrary by name, in which case they must be unique.
	Nm string `desc:"name of scene.  User-created scenes can be stored in the global SceneLibrary by name, in which case they must be unique."`

	// title of the Stage -- generally auto-set based on Scene Title.  used for title of Window and Dialog types
	Title string `desc:"title of the Stage -- generally auto-set based on Scene Title.  used for title of Window and Dialog types"`

	// has critical state information signaling when rendering, styling etc need to be done, and also indicates type of scene
	Flags ScFlags `desc:"has critical state information signaling when rendering, styling etc need to be done, and also indicates type of scene"`

	// Size and position relative to overall rendering context.
	Geom mat32.Geom2DInt

	// Extra decoration, configured by the outer Stage container.  Can be positioned anywhere -- typically uses LayoutNil
	Decor Layout `desc:"Extra decoration, configured by the outer Stage container.  Can be positioned anywhere -- typically uses LayoutNil"`

	// [view: -] render state for rendering
	RenderState paint.State `copy:"-" json:"-" xml:"-" view:"-" desc:"render state for rendering"`

	// [view: -] live pixels that we render into
	Pixels *image.RGBA `copy:"-" json:"-" xml:"-" view:"-" desc:"live pixels that we render into"`

	// background color for filling scene -- defaults to transparent so that popups can have rounded corners
	BgColor colors.Full `desc:"background color for filling scene -- defaults to transparent so that popups can have rounded corners"`

	// event manager for this scene
	EventMgr EventMgr `copy:"-" json:"-" xml:"-" desc:"event manager for this scene"`

	// current stage in which this Scene is set
	Stage Stage `copy:"-" json:"-" xml:"-" desc:"current stage in which this Scene is set"`

	// [view: -] Current color in styling -- used for relative color names
	CurColor color.RGBA `copy:"-" json:"-" xml:"-" view:"-" desc:"Current color in styling -- used for relative color names"`

	// LastRender captures key params from last render.
	// If different then a new ApplyStyleScene is needed.
	LastRender RenderParams

	// [view: -] StyleMu is RW mutex protecting access to Style-related global vars
	StyleMu sync.RWMutex `copy:"-" json:"-" xml:"-" view:"-" desc:"StyleMu is RW mutex protecting access to Style-related global vars"`
}

// StageScene creates a new Scene that will serve as the contents of a Stage
// (e.g., a Window, Dialog, etc).  Scenes can also be added as part of the
// Widget tree within another Scene, where they provide an optimized rendering
// context for areas that tend to update frequently -- use NewScene with a
// parent argument for that.
func StageScene(name string) *Scene {
	sc := &Scene{}
	sc.InitName(sc, name)
	sc.EventMgr.Scene = sc
	sc.BgColor.SetColor(color.Transparent)
	sc.Lay = LayoutVert
	sc.Decor.InitName(&sc.Decor, "decor")
	sc.Decor.Lay = LayoutNil
	sc.SetDefaultStyle()
	return sc
}

func (sc *Scene) SetTitle(title string) *Scene {
	sc.Title = title
	return sc
}

func (sc *Scene) RenderCtx() *RenderContext {
	sm := sc.MainStageMgr()
	if sm == nil {
		log.Println("ERROR: Scene has nil StageMgr:", sc.Nm)
		return nil
	}
	return sm.RenderCtx
}

// MainStageMgr returns the MainStageMgr that typically lives in a RenderWin
// and manages all of the MainStage elements (Windows, Dialogs etc),
// which in turn manage their popups.  This Scene could be in a popup
// or in a main stage.
func (sc *Scene) MainStageMgr() *MainStageMgr {
	if sc.Stage == nil {
		log.Println("ERROR: Scene has nil Stage:", sc.Nm)
		return nil
	}
	return sc.Stage.MainMgr()
}

// PopupStage returns the Stage as a PopupStage.
// nil if it is not a popup.
func (sc *Scene) PopupStage() *PopupStage {
	if sc.Stage == nil {
		return nil
	}
	return sc.Stage.AsPopup()
}

// MainStage returns this Scene's Stage as a MainStage,
// which could be nil if in fact it is in a PopupStage.
func (sc *Scene) MainStage() *MainStage {
	return sc.Stage.AsMain()
}

// FitInWindow fits Scene geometry (pos, size) into given window geom.
// Calls resize for the new size.
func (sc *Scene) FitInWindow(winGeom mat32.Geom2DInt) {
	geom := sc.Geom.FitInWindow(winGeom)
	sc.Resize(geom.Size)
	sc.Geom.Pos = geom.Pos
	// fmt.Println("win", winGeom, "geom", geom)
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
	sc.SetFlag(true, ScNeedsLayout)
	// fmt.Printf("vp %v resized to: %v, bounds: %v\n", vp.Path(), nwsz, vp.Pixels.Bounds())
}

func (sc *Scene) ScIsVisible() bool {
	if sc.RenderCtx == nil || sc.Pixels == nil {
		return false
	}
	return sc.RenderCtx().HasFlag(RenderVisible)
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

// Delete this Scene if not Flagged for preservation.
// Removes Decor and Frame Widgets
func (sc *Scene) Delete(destroy bool) {
	if sc.Flags.HasFlag(ScPreserve) {
		return
	}
	sc.DeleteImpl()
}

// DeleteImpl does the deletion, removing Decor and Frame Widgets.
func (sc *Scene) DeleteImpl() {
	sc.Decor.DeleteChildren(ki.DestroyKids)
	sc.DeleteChildren(ki.DestroyKids)
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
// Implements the styles.Context interface
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
func (sc *Scene) ContextColorSpecByURL(url string) *colors.Full {
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
	// ScUpdating means scene is in the process of updating:
	// set for any kind of tree-level update.
	// skip any further update passes until it goes off.
	ScUpdating ScFlags = ScFlags(WidgetFlagsN) + iota

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

	// ScImageUpdated indicates that the Scene's image has been updated
	// e.g., due to a render or a resize.  This is reset by the
	// global RenderWin rendering pass, so it knows whether it needs to
	// copy the image up to the GPU or not.
	ScImageUpdated

	// ScPrefSizing means that this scene is currently doing a
	// PrefSize computation to compute the size of the scene
	// (for sizing window for example) -- affects layout size computation
	// only for Over
	ScPrefSizing

	// ScPreserve keeps this scene around instead of deleting
	// when it is no longer needed.
	// Set if added to SceneLibrary for example.
	ScPreserve
)
