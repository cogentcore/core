// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/png"
	"io"
	"sync"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
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
// of rendering in the Cogent Core framework.
//
//core:no-new
//core:embedder
type Scene struct {
	Frame

	// Bars contains functions for constructing the control bars for this Scene,
	// attached to different sides of a Scene (e.g., TopAppBar at Top,
	// NavBar at Bottom, etc).  Functions are called in forward order
	// so first added are called first.
	Bars styles.Sides[BarFuncs]

	// BarsInherit determines which of the Bars side functions are inherited
	// from the context widget, for FullWindow Dialogs
	BarsInherit styles.Sides[bool]

	// AppBars contains functions for configuring a top-level App toolbar,
	// (e.g., TopAppBar) for elements contained within this Scene,
	// that should be represented in any app-level toolbar constructed
	// for this Scene.
	AppBars ToolbarFuncs

	// Body provides the main contents of scenes that use control Bars
	// to allow the main window contents to be specified separately
	// from that dynamic control content.  When constructing scenes using
	// a Body, you can operate directly on the [Body], which has wrappers
	// for most major Scene functions.
	Body *Body

	// Data is the optional data value being represented by this scene.
	// Used e.g., for recycling views of a given item instead of creating new one.
	Data any

	// Size and position relative to overall rendering context.
	SceneGeom mat32.Geom2DInt `edit:"-" set:"-"`

	// paint context for rendering
	PaintContext paint.Context `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// live pixels that we render into
	Pixels *image.RGBA `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// event manager for this scene
	EventMgr EventMgr `copier:"-" json:"-" xml:"-" set:"-"`

	// current stage in which this Scene is set
	Stage *Stage `copier:"-" json:"-" xml:"-" set:"-"`

	// RenderBBoxHue is current hue for rendering bounding box in ScRenderBBoxes mode
	RenderBBoxHue float32 `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// the currently selected/hovered widget through the inspect editor selection mode
	// that should be highlighted with a background color
	SelectedWidget Widget

	// the channel on which the selected widget through the inspect editor
	// selection mode is transmitted to the inspect editor after the user is done selecting
	SelectedWidgetChan chan Widget

	// LastRender captures key params from last render.
	// If different then a new ApplyStyleScene is needed.
	LastRender RenderParams `edit:"-" set:"-"`

	// StyleMu is RW mutex protecting access to Style-related global vars
	StyleMu sync.RWMutex `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// ShowIter counts up at start of showing a Scene
	// to trigger Show event and other steps at start of first show
	ShowIter int `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// ReRender items are re-rendered after the current pass
	ReRender []Widget
}

func (sc *Scene) FlagType() enums.BitFlagSetter {
	return (*ScFlags)(&sc.Flags)
}

// NewBodyScene creates a new Scene for use with an associated Body that
// contains the main content of the Scene (e.g., a Window, Dialog, etc).
// It will be constructed from the Bars-configured control bars on each
// side, with the given Body as the central content.
func NewBodyScene(body *Body, name ...string) *Scene {
	sc := &Scene{}
	nm := body.Nm + " scene"
	if len(name) > 0 {
		nm = name[0]
	}
	sc.InitName(sc, nm)
	sc.EventMgr.Scene = sc
	sc.Body = body
	// need to set parent immediately so that SceneConfig works,
	// but can not add it yet because it may go elsewhere due
	// to app bars
	ki.SetParent(body, sc)
	return sc
}

// NewScene creates a new Scene object without a Body, e.g., for use
// in a Menu, Tooltip or other such simple popups or non-control-bar Scenes.
func NewScene(name ...string) *Scene {
	sc := &Scene{}
	sc.InitName(sc, name...)
	sc.EventMgr.Scene = sc
	return sc
}

func (sc *Scene) OnInit() {
	sc.Scene = sc
	sc.WidgetBase.OnInit()
	sc.SetStyles()
	sc.HandleEvents()
	if TheApp.SceneConfig != nil {
		TheApp.SceneConfig(sc)
	}
}

func (sc *Scene) SetStyles() {
	sc.Style(func(s *styles.Style) {
		s.Cursor = cursors.Arrow
		s.Background = colors.C(colors.Scheme.Background)
		s.Color = colors.Scheme.OnBackground
		// we never want borders on scenes
		s.MaxBorder = styles.Border{}
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto) // screen is always scroller of last resort

		// insets and minimum window padding
		if sc.Stage == nil {
			return
		}
		if sc.Stage.Type.IsPopup() || (sc.Stage.Type == DialogStage && !sc.Stage.FullWindow) {
			return
		}

		s.Padding.Set(units.Dp(8))
	})
}

func (sc *Scene) HandleEvents() {
	sc.Frame.HandleEvents()
	sc.OnShow(func(e events.Event) {
		CurRenderWin.SetStageTitle(sc.Stage.Title)
	})
	sc.OnClose(func(e events.Event) {
		sm := sc.MainStageMgr()
		if sm == nil {
			return
		}
		sm.Mu.RLock()
		defer sm.Mu.RUnlock()

		if sm.Stack.Len() < 2 {
			return
		}
		// the stage that will be visible next
		st := sm.Stack.ValueByIndex(sm.Stack.Len() - 2)
		CurRenderWin.SetStageTitle(st.Title)
	})
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

// MainStageMgr returns the Main StageMgr that typically lives in a RenderWin
// and manages all of the MainStage elements (Windows, Dialogs etc),
// which in turn manage their popups.  This Scene could be in a popup
// or in a main stage.
func (sc *Scene) MainStageMgr() *StageMgr {
	return sc.Stage.MainMgr
}

// FitInWindow fits Scene geometry (pos, size) into given window geom.
// Calls resize for the new size.
func (sc *Scene) FitInWindow(winGeom mat32.Geom2DInt) {
	geom := sc.SceneGeom
	// full offscreen windows ignore any window geometry constraints
	// because they must be unbounded by any previous window sizes
	if TheApp.Platform() != goosi.Offscreen || !sc.Stage.FullWindow {
		geom = geom.FitInWindow(winGeom)
	}
	sc.Resize(geom)
	sc.SceneGeom.Pos = geom.Pos
	// fmt.Println("win", winGeom, "geom", geom)
}

// Resize resizes the scene, creating a new image; updates Geom
func (sc *Scene) Resize(geom mat32.Geom2DInt) {
	if geom.Size.X <= 0 || geom.Size.Y <= 0 {
		return
	}
	if sc.PaintContext.State == nil {
		sc.PaintContext.State = &paint.State{}
	}
	if sc.PaintContext.Paint == nil {
		sc.PaintContext.Paint = &styles.Paint{}
	}
	sc.SceneGeom.Pos = geom.Pos
	if sc.Pixels == nil || sc.Pixels.Bounds().Size() != geom.Size {
		sc.Pixels = image.NewRGBA(image.Rectangle{Max: geom.Size})
	}
	sc.PaintContext.Init(geom.Size.X, geom.Size.Y, sc.Pixels)
	sc.SceneGeom.Size = geom.Size // make sure

	sc.ApplyStyleScene()
	// restart the multi-render updating after resize, to get windows to update correctly while
	// resizing on Windows (OS) and Linux (see https://github.com/goki/gi/issues/584), to get
	// windows on Windows (OS) to update after a window snap (see https://github.com/goki/gi/issues/497),
	// and to get FillInsets to overwrite mysterious black bars that otherwise are rendered on both iOS
	// and Android in different contexts.
	// TODO(kai): is there a more efficient way to do this, and do we need to do this on all platforms?
	sc.ShowIter = 0
}

func (sc *Scene) ScIsVisible() bool {
	if sc.RenderCtx() == nil || sc.Pixels == nil {
		return false
	}
	return sc.RenderCtx().HasFlag(RenderVisible)
}

// Close closes the stage associated with this Scene (typically for Dialog)
func (sc *Scene) Close() {
	sc.Send(events.Close, nil)
	mm := sc.Stage.MainMgr
	if mm == nil {
		return // todo: needed, but not sure why
	}
	if sc.Stage.NewWindow && !TheApp.Platform().IsMobile() {
		mm.RenderWin.CloseReq()
		return
	}
	mm.DeleteStage(sc.Stage)
}

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

// UpdateTitle updates the title of the Scene's associated [Stage],
// [RenderWin], and [Body], if applicable.
func (sc *Scene) UpdateTitle(title string) {
	if sc.Scene != nil {
		sc.Stage.Title = title
	}
	if rw := sc.RenderWin(); rw != nil {
		rw.SetTitle(title)
	}
	if sc.Body != nil {
		sc.Body.Title = title
		if tw, ok := sc.Body.ChildByName("title").(*Label); ok {
			tw.SetText(title)
		}
	}
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
