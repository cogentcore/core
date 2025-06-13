// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"slices"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/tree"
)

// Scene contains a [Widget] tree, rooted in an embedded [Frame] layout,
// which renders into its own [paint.Painter]. The [Scene] is set in a
// [Stage], which the [Scene] has a pointer to.
//
// Each [Scene] contains state specific to its particular usage
// within a given [Stage] and overall rendering context, representing the unit
// of rendering in the Cogent Core framework.
type Scene struct { //core:no-new
	Frame

	// Body provides the main contents of scenes that use control Bars
	// to allow the main window contents to be specified separately
	// from that dynamic control content.  When constructing scenes using
	// a [Body], you can operate directly on the [Body], which has wrappers
	// for most major Scene functions.
	Body *Body `json:"-" xml:"-" set:"-"`

	// WidgetInit is a function called on every newly created [Widget].
	// This can be used to set global configuration and styling for all
	// widgets in conjunction with [App.SceneInit].
	WidgetInit func(w Widget) `json:"-" xml:"-" edit:"-"`

	// Bars are functions for creating control bars,
	// attached to different sides of a [Scene]. Functions
	// are called in forward order so first added are called first.
	Bars sides.Sides[BarFuncs] `json:"-" xml:"-" set:"-"`

	// Data is the optional data value being represented by this scene.
	// Used e.g., for recycling views of a given item instead of creating new one.
	Data any

	// Size and position relative to overall rendering context.
	SceneGeom math32.Geom2DInt `edit:"-" set:"-"`

	// painter for rendering all widgets in the scene.
	Painter paint.Painter `copier:"-" json:"-" xml:"-" display:"-" set:"-"`

	// event manager for this scene.
	Events Events `copier:"-" json:"-" xml:"-" set:"-"`

	// current stage in which this Scene is set.
	Stage *Stage `copier:"-" json:"-" xml:"-" set:"-"`

	// Animations are the currently active [Animation]s in this scene.
	Animations []*Animation `json:"-" xml:"-" set:"-"`

	// renderBBoxes indicates to render colored bounding boxes for all of the widgets
	// in the scene. This is enabled by the [Inspector] in select element mode.
	renderBBoxes bool

	// renderBBoxHue is current hue for rendering bounding box in [Scene.RenderBBoxes] mode.
	renderBBoxHue float32

	// selectedWidget is the currently selected/hovered widget through the [Inspector] selection mode
	// that should be highlighted with a background color.
	selectedWidget Widget

	// selectedWidgetChan is the channel on which the selected widget through the inspect editor
	// selection mode is transmitted to the inspect editor after the user is done selecting.
	selectedWidgetChan chan Widget `json:"-" xml:"-"`

	// source renderer for rendering the scene
	renderer render.Renderer `copier:"-" json:"-" xml:"-" display:"-" set:"-"`

	// lastRender captures key params from last render.
	// If different then a new ApplyStyleScene is needed.
	lastRender renderParams

	// showIter counts up at start of showing a Scene
	// to trigger Show event and other steps at start of first show
	showIter int

	// directRenders are widgets that render directly to the [RenderWindow]
	// instead of rendering into the Scene Painter.
	directRenders []Widget

	// flags are atomic bit flags for [Scene] state.
	flags sceneFlags
}

// sceneFlags are atomic bit flags for [Scene] state.
// They must be atomic to prevent race conditions.
type sceneFlags int64 //enums:bitflag -trim-prefix scene

const (
	// sceneHasShown is whether this scene has been shown.
	// This is used to ensure that [events.Show] is only sent once.
	sceneHasShown sceneFlags = iota

	// sceneUpdating means the Scene is in the process of sceneUpdating.
	// It is set for any kind of tree-level update.
	// Skip any further update passes until it goes off.
	sceneUpdating

	// sceneNeedsRender is whether anything in the Scene needs to be re-rendered
	// (but not necessarily the whole scene itself).
	sceneNeedsRender

	// sceneNeedsLayout is whether the Scene needs a new layout pass.
	sceneNeedsLayout

	// sceneHasDeferred is whether the Scene has elements with Deferred functions.
	sceneHasDeferred

	// sceneImageUpdated indicates that the Scene's image has been updated
	// e.g., due to a render or a resize. This is reset by the
	// global [RenderWindow] rendering pass, so it knows whether it needs to
	// copy the image up to the GPU or not.
	sceneImageUpdated

	// sceneContentSizing means that this scene is currently doing a
	// contentSize computation to compute the size of the scene
	// (for sizing window for example). Affects layout size computation.
	sceneContentSizing
)

// hasFlag returns whether the given flag is set.
func (sc *Scene) hasFlag(f sceneFlags) bool {
	return sc.flags.HasFlag(f)
}

// setFlag sets the given flags to the given value.
func (sc *Scene) setFlag(on bool, f ...enums.BitFlag) {
	sc.flags.SetFlag(on, f...)
}

// newBodyScene creates a new Scene for use with an associated Body that
// contains the main content of the Scene (e.g., a Window, Dialog, etc).
// It will be constructed from the Bars-configured control bars on each
// side, with the given Body as the central content.
func newBodyScene(body *Body) *Scene {
	sc := NewScene(body.Name + " scene")
	sc.Body = body
	// need to set parent immediately so that SceneInit works,
	// but can not add it yet because it may go elsewhere due
	// to app bars
	tree.SetParent(body, sc)
	return sc
}

// NewScene creates a new [Scene] object without a [Body], e.g., for use
// in a Menu, Tooltip or other such simple popups or non-control-bar Scenes.
func NewScene(name ...string) *Scene {
	sc := tree.New[Scene]()
	if len(name) > 0 {
		sc.SetName(name[0])
	}
	sc.Events.scene = sc
	return sc
}

func (sc *Scene) Init() {
	sc.Scene = sc
	sc.Frame.Init()
	sc.AddContextMenu(sc.standardContextMenu)
	sc.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable) // this is critical to enable click-off to turn off focus.
		s.Cursor = cursors.Arrow
		s.Background = colors.Scheme.Background
		s.Color = colors.Scheme.OnBackground
		// we never want borders on scenes
		s.MaxBorder = styles.Border{}
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto) // screen is always scroller of last resort

		// insets and minimum window padding
		if sc.Stage == nil {
			return
		}
		if sc.Stage.Type.isPopup() || (sc.Stage.Type == DialogStage && !sc.Stage.FullWindow) {
			return
		}

		s.Padding.Set(units.Dp(8))
	})
	sc.OnShow(func(e events.Event) {
		currentRenderWindow.SetStageTitle(sc.Stage.Title)
	})
	sc.OnClose(func(e events.Event) {
		sm := sc.Stage.Mains
		if sm == nil {
			return
		}
		sm.Lock()
		defer sm.Unlock()

		if sm.stack.Len() < 2 {
			return
		}
		// the stage that will be visible next
		st := sm.stack.ValueByIndex(sm.stack.Len() - 2)
		currentRenderWindow.SetStageTitle(st.Title)
	})
	sc.Updater(func() {
		if TheApp.Platform() == system.Offscreen {
			return
		}
		// At the scene level, we reset the shortcuts and add our context menu
		// shortcuts every time. This clears the way for buttons to add their
		// shortcuts in their own Updaters. We must get the shortcuts every time
		// since buttons may be added or removed dynamically.
		sc.Events.shortcuts = nil
		tmps := NewScene()
		sc.applyContextMenus(tmps)
		sc.Events.getShortcutsIn(tmps)
	})
	if TheApp.SceneInit != nil {
		TheApp.SceneInit(sc)
	}
}

// renderContext returns the current render context.
// This will be nil prior to actual rendering.
func (sc *Scene) renderContext() *renderContext {
	if sc.Stage == nil {
		return nil
	}
	sm := sc.Stage.Mains
	if sm == nil {
		return nil
	}
	return sm.renderContext
}

// TextShaper returns the current [shaped.TextShaper], for text shaping.
// may be nil if not yet initialized.
func (sc *Scene) TextShaper() shaped.Shaper {
	rc := sc.renderContext()
	if rc != nil {
		return rc.textShaper
	}
	return nil
}

// RenderWindow returns the current render window for this scene.
// In general it is best to go through [renderContext] instead of the window.
// This will be nil prior to actual rendering.
func (sc *Scene) RenderWindow() *renderWindow {
	if sc.Stage == nil {
		return nil
	}
	sm := sc.Stage.Mains
	if sm == nil {
		return nil
	}
	return sm.renderWindow
}

// fitInWindow fits Scene geometry (pos, size) into given window geom.
// Calls resize for the new size and returns whether it actually needed to
// be resized.
func (sc *Scene) fitInWindow(winGeom math32.Geom2DInt) bool {
	geom := sc.SceneGeom
	geom = geom.FitInWindow(winGeom)
	return sc.resize(geom)
}

// resize resizes the scene if needed, creating a new image; updates Geom.
// returns false if the scene is already the correct size.
func (sc *Scene) resize(geom math32.Geom2DInt) bool {
	if geom.Size.X <= 0 || geom.Size.Y <= 0 {
		return false
	}
	sz := math32.FromPoint(geom.Size)
	if sc.Painter.State == nil {
		sc.Painter = *paint.NewPainter(sz)
		sc.Painter.Paint.UnitContext = sc.Styles.UnitContext
	}
	sc.SceneGeom.Pos = geom.Pos
	if sc.renderer != nil {
		img := sc.renderer.Image()
		if img != nil {
			isz := img.Bounds().Size()
			if isz == geom.Size {
				return false
			}
		}
	} else {
		sc.renderer = paint.NewSourceRenderer(sz)
	}
	sc.Painter.Paint.UnitContext = sc.Styles.UnitContext
	sc.Painter.State.Init(sc.Painter.Paint, sz)
	sc.renderer.SetSize(units.UnitDot, sz)
	sc.SceneGeom.Size = geom.Size // make sure

	sc.updateScene()
	sc.applyStyleScene()
	// restart the multi-render updating after resize, to get windows to update correctly while
	// resizing on Windows (OS) and Linux (see https://github.com/cogentcore/core/issues/584),
	// to get windows on Windows (OS) to update after a window snap (see
	// https://github.com/cogentcore/core/issues/497),
	// and to get FillInsets to overwrite mysterious black bars that otherwise are rendered
	// on both iOS and Android in different contexts.
	// TODO(kai): is there a more efficient way to do this, and do we need to do this on all platforms?
	sc.showIter = 0
	sc.NeedsLayout()
	return true
}

// ResizeToContent resizes the scene so it fits the current content.
// Only applicable to desktop systems where windows can be resized.
// Optional extra size is added to the amount computed to hold the contents,
// which is needed in cases with wrapped text elements, which don't
// always size accurately. See [Scene.SetGeometry] for a more general way
// to set all window geometry properties.
func (sc *Scene) ResizeToContent(extra ...image.Point) {
	if TheApp.Platform().IsMobile() { // not resizable
		return
	}
	win := sc.RenderWindow()
	if win == nil {
		return
	}
	go func() {
		scsz := system.TheApp.Screen(0).PixelSize
		sz := sc.contentSize(scsz)
		if len(extra) == 1 {
			sz = sz.Add(extra[0])
		}
		win.SystemWindow.SetSize(sz)
	}()
}

// SetGeometry uses [system.Window.SetGeometry] to set all window geometry properties,
// with pos in operating system window manager units and size in raw pixels.
// If pos and/or size is not specified, it defaults to the current value.
// If fullscreen is true, pos and size are ignored, and screen indicates the number
// of the screen on which to fullscreen the window. If fullscreen is false, the
// window is moved to the given pos and size on the given screen. If screen is -1,
// the current screen the window is on is used, and fullscreen/pos/size are all
// relative to that screen. It is only applicable on desktop and web platforms,
// with only fullscreen supported on web. See [Scene.SetFullscreen] for a simpler way
// to set only the fullscreen state. See [Scene.ResizeToContent] to resize the window
// to fit the current content.
func (sc *Scene) SetGeometry(fullscreen bool, pos image.Point, size image.Point, screen int) {
	rw := sc.RenderWindow()
	if rw == nil {
		return
	}
	scr := TheApp.Screen(screen)
	if screen < 0 {
		scr = rw.SystemWindow.Screen()
	}
	rw.SystemWindow.SetGeometry(fullscreen, pos, size, scr)
}

// IsFullscreen returns whether the window associated with this [Scene]
// is in fullscreen mode (true) or window mode (false). This is implemented
// on desktop and web platforms. See [Scene.SetFullscreen] to update the
// current fullscreen state and [Stage.SetFullscreen] to set the initial state.
func (sc *Scene) IsFullscreen() bool {
	rw := sc.RenderWindow()
	if rw == nil {
		return false
	}
	return rw.SystemWindow.Is(system.Fullscreen)
}

// SetFullscreen requests that the window associated with this [Scene]
// be updated to either fullscreen mode (true) or window mode (false).
// This is implemented on desktop and web platforms. See [Scene.IsFullscreen]
// to get the current fullscreen state and [Stage.SetFullscreen] to set the
// initial state. ([Stage.SetFullscreen] sets the initial state, whereas
// this function sets the current state after the [Stage] is already running).
// See [Scene.SetGeometry] for a more general way to set all window
// geometry properties.
func (sc *Scene) SetFullscreen(fullscreen bool) {
	rw := sc.RenderWindow()
	if rw == nil {
		return
	}
	wgp, screen := theWindowGeometrySaver.get(rw.title, "")
	if wgp != nil {
		rw.SystemWindow.SetGeometry(fullscreen, wgp.Pos, wgp.Size, screen)
	} else {
		rw.SystemWindow.SetGeometry(fullscreen, image.Point{}, image.Point{}, rw.SystemWindow.Screen())
	}
}

// Close closes the [Stage] associated with this [Scene].
// This only works for main stages (windows and dialogs).
// It returns whether the [Stage] was successfully closed.
func (sc *Scene) Close() bool {
	if sc == nil {
		return true
	}
	e := &events.Base{Typ: events.Close}
	e.Init()
	sc.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		cw.AsWidget().HandleEvent(e)
		return tree.Continue
	})
	// if they set the event as handled, we do not close the scene
	if e.IsHandled() {
		return false
	}
	mm := sc.Stage.Mains
	if mm == nil {
		return false // todo: needed, but not sure why
	}
	mm.deleteStage(sc.Stage)
	if sc.Stage.NewWindow && !TheApp.Platform().IsMobile() && !mm.renderWindow.flags.HasFlag(winClosing) && !mm.renderWindow.flags.HasFlag(winStopEventLoop) && !TheApp.IsQuitting() {
		mm.renderWindow.closeReq()
	}
	return true
}

func (sc *Scene) ApplyScenePos() {
	sc.Frame.ApplyScenePos()
	if sc.Parts == nil {
		return
	}

	mvi := sc.Parts.ChildByName("move", 1)
	if mvi == nil {
		return
	}
	mv := mvi.(Widget).AsWidget()

	sc.Parts.Geom.Pos.Total.Y = math32.Ceil(0.5 * mv.Geom.Size.Actual.Total.Y)
	sc.Parts.Geom.Size.Actual = sc.Geom.Size.Actual
	sc.Parts.Geom.Size.Alloc = sc.Geom.Size.Alloc
	sc.Parts.setContentPosFromPos()
	sc.Parts.setBBoxesFromAllocs()
	sc.Parts.applyScenePosChildren()

	psz := sc.Parts.Geom.Size.Actual.Content

	mv.Geom.RelPos.X = 0.5*psz.X - 0.5*mv.Geom.Size.Actual.Total.X
	mv.Geom.RelPos.Y = 0
	mv.setPosFromParent()
	mv.setBBoxesFromAllocs()

	rszi := sc.Parts.ChildByName("resize", 1)
	if rszi == nil {
		return
	}
	rsz := rszi.(Widget).AsWidget()
	rsz.Geom.RelPos.X = psz.X // - 0.5*rsz.Geom.Size.Actual.Total.X
	rsz.Geom.RelPos.Y = psz.Y // - 0.5*rsz.Geom.Size.Actual.Total.Y
	rsz.setPosFromParent()
	rsz.setBBoxesFromAllocs()
}

func (sc *Scene) AddDirectRender(w Widget) {
	sc.directRenders = append(sc.directRenders, w)
}

func (sc *Scene) DeleteDirectRender(w Widget) {
	idx := slices.Index(sc.directRenders, w)
	if idx >= 0 {
		sc.directRenders = slices.Delete(sc.directRenders, idx, idx+1)
	}
}
