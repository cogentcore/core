// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"sync"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/enums"
	"goki.dev/girl/abilities"
	"goki.dev/girl/paint"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// see:
//	- render.go for scene-based rendering code
//	- layimpl.go for layout
//	- style.go for style

// Scene contains a Widget tree, rooted in an embedded Frame layout,
// which renders into its Pixels image.
// The Scene is set in a Stage (pointer retained in Scene).
// Stage has a StageMgr manager for controlling things like Popups
// (Menus and Dialogs, etc).
//
// Each Scene and Widget tree contains state specific to its particular usage
// within a given Stage and overall rendering context, representing the unit
// of rendering in the GoGi framework.
//
//goki:no-new
//goki:embedder
type Scene struct {
	Frame

	// title of the Scene
	Title string `set:"-"`

	// Data is the optional data value being represented by this scene.
	// Used e.g., for recycling views of a given item instead of creating new one.
	Data any

	// TopAppBar is a function used to construct a top app bar at the top of the
	// scene when it is set up. It is copied by default to all FullWindow [DialogStage]s
	// created in the context of this scene.
	TopAppBar func(tb *TopAppBar)

	// Size and position relative to overall rendering context.
	SceneGeom mat32.Geom2DInt `edit:"-" set:"-"`

	// render state for rendering
	RenderState paint.State `copy:"-" json:"-" xml:"-" view:"-" set:"-"`

	// live pixels that we render into
	Pixels *image.RGBA `copy:"-" json:"-" xml:"-" view:"-" set:"-"`

	// background color for filling scene.
	// Defaults to transparent so that popups can have rounded corners
	BgColor colors.Full

	// event manager for this scene
	EventMgr EventMgr `copy:"-" json:"-" xml:"-" set:"-"`

	// current stage in which this Scene is set
	Stage Stage `copy:"-" json:"-" xml:"-" set:"-"`

	// Current color in styling -- used for relative color names
	CurColor color.RGBA `copy:"-" json:"-" xml:"-" view:"-" set:"-"`

	// RenderBBoxHue is current hue for rendering bounding box in ScRenderBBoxes mode
	RenderBBoxHue float32 `copy:"-" json:"-" xml:"-" view:"-" set:"-"`

	// the currently selected widget through the inspect editor selection mode
	SelectedWidget Widget

	// the channel on which the selected widget through the inspect editor
	// selection mode is transmitted to the inspect editor after the user is done selecting
	SelectedWidgetChan chan Widget

	// LastRender captures key params from last render.
	// If different then a new ApplyStyleScene is needed.
	LastRender RenderParams `edit:"-" set:"-"`

	// StyleMu is RW mutex protecting access to Style-related global vars
	StyleMu sync.RWMutex `copy:"-" json:"-" xml:"-" view:"-" set:"-"`

	// ShowLayoutIter counts up at start of showing a Scene
	// for a sequence of Layout passes to ensure proper initial sizing.
	ShowLayoutIter int `copy:"-" json:"-" xml:"-" view:"-" set:"-"`
}

func (sc *Scene) FlagType() enums.BitFlagSetter {
	return (*ScFlags)(&sc.Flags)
}

// NewScene creates a new Scene that will serve as the content of a Stage
// (e.g., a Window, Dialog, etc).  Scenes can also be added as part of the
// Widget tree within another Scene, where they provide an optimized rendering
// context for areas that tend to update frequently -- use NewSubScene with a
// parent argument for that. If no name is provided, it defaults to "scene".
func NewScene(name ...string) *Scene {
	sc := &Scene{}
	sc.InitName(sc, name...)
	sc.EventMgr.Scene = sc
	sc.BgColor.SetSolid(colors.Transparent)
	return sc
}

// NewSubScene creates a new [Scene] that will serve as a sub-scene of another [Scene].
// Scenes can also be added as the content of a [Stage] (without a parent) through the
// [NewScene] function. If no name is provided, it defaults to "scene".
func NewSubScene(par ki.Ki, name ...string) *Scene {
	sc := par.NewChild(SceneType, name...).(*Scene)
	sc.EventMgr.Scene = sc
	sc.BgColor.SetSolid(colors.Transparent)
	return sc
}

func (sc *Scene) OnInit() {
	sc.Sc = sc
	sc.SceneStyles()
	sc.HandleLayoutEvents()
}

func (sc *Scene) SceneStyles() {
	sc.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.FocusWithinable)
		s.Cursor = cursors.Arrow
		s.BackgroundColor.SetSolid(colors.Scheme.Background)
		s.Color = colors.Scheme.OnBackground
		// we never want borders on scenes
		s.MaxBorder = styles.Border{}
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto) // screen is always scroller of last resort

		// insets
		if sc.Stage == nil {
			return
		}
		ms := sc.Stage.AsMain()
		if ms == nil || (ms.Type == DialogStage && !ms.FullWindow) {
			return
		}

		mm := sc.Stage.MainMgr()
		if mm == nil {
			return
		}
		rw := mm.RenderWin
		if rw == nil {
			return
		}

		insets := rw.GoosiWin.Insets()

		uv := func(val float32) units.Value {
			return units.Custom(func(uc *units.Context) float32 {
				return max(val, uc.Dp(8))
			})
		}

		s.Padding.Set(
			uv(insets.Top),
			uv(insets.Right),
			uv(insets.Bottom),
			uv(insets.Left),
		)
	})
}

// SetTitle sets the title
func (sc *Scene) SetTitle(title string) *Scene {
	sc.Title = title
	return sc
}

// RenderCtx returns the current render context.
// This will be nil prior to actual rendering.
func (sc *Scene) RenderCtx() *RenderContext {
	if sc.Stage == nil {
		return nil
	}
	sm := sc.MainStageMgr()
	if sm == nil {
		return nil
	}
	return sm.RenderCtx
}

// RenderWin returns the current render window for this scene.
// In general it is best to go through RenderCtx instead of the window.
// This will be nil prior to actual rendering.
func (sc *Scene) RenderWin() *RenderWin {
	if sc.Stage == nil {
		return nil
	}
	sm := sc.MainStageMgr()
	if sm == nil {
		return nil
	}
	return sm.RenderWin
}

// MainStageMgr returns the MainStageMgr that typically lives in a RenderWin
// and manages all of the MainStage elements (Windows, Dialogs etc),
// which in turn manage their popups.  This Scene could be in a popup
// or in a main stage.
func (sc *Scene) MainStageMgr() *MainStageMgr {
	if sc.Stage == nil {
		slog.Error("Scene has nil Stage", "scene", sc.Nm)
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
	geom := sc.SceneGeom.FitInWindow(winGeom)
	sc.Resize(geom.Size)
	sc.SceneGeom.Pos = geom.Pos
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
			sc.SceneGeom.Size = nwsz // make sure
			return                   // already good
		}
	}
	if sc.Pixels != nil {
		sc.Pixels = nil
	}
	sc.Pixels = image.NewRGBA(image.Rectangle{Max: nwsz})
	sc.RenderState.Init(nwsz.X, nwsz.Y, sc.Pixels)
	sc.SceneGeom.Size = nwsz // make sure
	sc.SetFlag(true, ScNeedsLayout)
	// fmt.Printf("vp %v resized to: %v, bounds: %v\n", vp.Path(), nwsz, vp.Pixels.Bounds())
}

func (sc *Scene) ScIsVisible() bool {
	if sc.RenderCtx() == nil || sc.Pixels == nil {
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
type ScFlags WidgetFlags //enums:bitflag

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

	// ScRenderBBoxes renders the bounding boxes for all objects in scene
	ScRenderBBoxes
)
