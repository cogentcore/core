// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system/composer"
	"cogentcore.org/core/tree"
	"golang.org/x/image/draw"
)

// layersEnabled is whether scenes render their fixed widgets into a layer
// (see [sceneLayer]).
var layersEnabled = true

// sceneLayer is a compositing layer of a [Scene]: a second image, composed
// over the image of the scene (as its own texture of the GPU drawer, as a
// popup is), into which the fixed widgets of the scene are rendered (the
// dialogs, cookie banners and modals of web pages, see
// [WidgetBase.isLayerRoot]), as browsers do. The widgets stay in the one
// widget tree of the scene (with its styles and events); only their painting
// goes to the other image.
//
// The layer holds everything that is painted from the first fixed widget
// on, in the painting order of a render of the whole scene, so that the
// layer over the scene is the same as the one image (whatever is painted
// after the fixed widget, as another fixed one, is in the layer). Each
// render of the whole scene marks the widgets rendered into the layer
// ([widgetInLayer]) and those that have such widgets in them
// ([widgetHasLayer]).
//
// Then a widget under the layer (as a playing video, under the
// translucent backdrop of a dialog) re-renders alone into the image of the
// scene, and the layer, which is unchanged, is composed over it again. A
// widget in the layer that needs to be rendered renders the layer again, as
// a whole: it is cleared, and the render of the scene is replayed (only
// into the widgets that have layer content), with what is painted before
// the layer thrown away.
type sceneLayer struct {

	// state is the painting state of the layer, which the painter of the
	// scene uses while it renders into the layer.
	state *paint.State

	// scratch is the painting state that the painter of the scene uses for
	// what is painted before the layer while the layer is rendered again,
	// which is thrown away.
	scratch *paint.State

	// renderer renders the layer into its image.
	renderer render.Renderer

	// mode is the rendering that is being done.
	mode layerMode

	// started is whether the rendering has reached the layer: the painter
	// of the scene paints into it.
	started bool

	// base is the painting state of the painter of the scene, from before
	// it was switched to the layer.
	base *paint.State

	// active is whether the last render of the whole scene rendered
	// something into the layer, which is then composed over the scene.
	active bool

	// needsRender is whether a widget in the layer needs to be rendered,
	// which renders the layer again.
	needsRender bool

	// renders counts the renders of the layer (for tests).
	renders int
}

// layerMode is the rendering that a [sceneLayer] is in.
type layerMode int32

const (
	// layerOff is no rendering of the scene with a layer.
	layerOff layerMode = iota

	// layerFull is a render of the whole scene, which finds the layer.
	layerFull

	// layerReplay renders the layer again (see [sceneLayer]).
	layerReplay

	// layerPartial is a render of widgets that need to be rendered, into
	// the image of the scene, where those in the layer are not rendered
	// (the layer is rendered again for them).
	layerPartial
)

// isLayerRoot returns whether the widget starts the layer of its scene (see
// [sceneLayer]): it is fixed (positioned relative to the viewport, so its
// layout is independent of the content that scrolls under it), and not under
// the widgets that it is in (of a negative z-index).
func (wb *WidgetBase) isLayerRoot() bool {
	return wb.Styles.Position == styles.PositionFixed && wb.Styles.ZIndex >= 0
}

// layerStart is called by [WidgetBase.StartRender] while the scene is
// rendered with its layer, and returns whether the widget is rendered: it
// switches the painter of the scene to the layer at its first widget, and
// marks the widgets rendered into the layer, or skips the widgets that are
// not rendered in the mode.
func (l *sceneLayer) layerStart(sc *Scene, wb *WidgetBase) bool {
	switch l.mode {
	case layerPartial:
		if wb.hasFlag(widgetInLayer) {
			l.needsRender = true
			return false
		}
		return true
	case layerReplay:
		if !l.started && !wb.hasFlag(widgetInLayer) && !wb.hasFlag(widgetHasLayer) {
			return false // nothing of the layer in it
		}
	}
	if !l.started {
		if !wb.isLayerRoot() {
			return true
		}
		l.start(sc)
	}
	if l.mode == layerFull {
		wb.setFlag(true, widgetInLayer)
		// (all of the widgets that it is in, also those that were rendered
		// before the layer and render the widgets raised to them now)
		for p := wb.parentWidget(); p != nil && !p.hasFlag(widgetInLayer) && !p.hasFlag(widgetHasLayer); p = p.parentWidget() {
			p.setFlag(true, widgetHasLayer)
		}
	}
	return true
}

// start switches the painter of the scene to the layer, which is cleared,
// within the contexts (clipping and transforms) of the widgets that the
// widget that starts the layer is in, as the page that it is in, which
// clips it to the viewport of the page in an app. Those are popped from the
// painting state that it was in, and they pop them from the layer.
func (l *sceneLayer) start(sc *Scene) {
	l.started = true
	l.renders++
	l.base = sc.Painter.State
	l.state.RenderDone() // (anything not yet composed is cleared anyway)
	l.state.Render.Add(pimage.NewClear(colors.Uniform(colors.Transparent), image.Point{}, draw.Src))
	l.state.PushStack(l.base)
	l.base.PopStack()
	sc.Painter.State = l.state
}

// finish ends a rendering of the scene with its layer, switching the
// painter back to the painting state of the scene.
func (l *sceneLayer) finish(sc *Scene) {
	if l.started {
		sc.Painter.State = l.base
		l.base = nil
	}
	l.started = false
	l.mode = layerOff
}

// resize makes the painting states and the renderer of the layer for the
// size of the scene.
func (l *sceneLayer) resize(sc *Scene) {
	sz := math32.FromPoint(sc.SceneGeom.Size)
	if l.state != nil && l.state.Size == sz {
		return
	}
	l.state = &paint.State{}
	l.state.Init(sc.Painter.Paint, sz)
	l.scratch = &paint.State{}
	l.scratch.Init(sc.Painter.Paint, sz)
	l.renderer = paint.NewSourceRenderer(sz)
	l.active = false
}

// renderWhole renders the whole scene, with its layer: the widgets from the
// first fixed widget on are rendered into the layer.
func (sc *Scene) renderWhole() {
	l := &sc.layer
	if !layersEnabled || sc.Painter.State == nil || sc.SceneGeom.Size == (image.Point{}) {
		l.active = false
		sc.RenderWidget()
		return
	}
	l.resize(sc)
	if l.active { // (the flags of the last render)
		sc.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
			cwb.setFlag(false, widgetInLayer, widgetHasLayer)
			return tree.Continue
		})
	}
	l.mode = layerFull
	l.needsRender = false
	sc.RenderWidget()
	l.active = l.started
	l.finish(sc)
}

// renderLayer renders the layer of the scene again (see [sceneLayer]).
func (sc *Scene) renderLayer() {
	l := &sc.layer
	l.needsRender = false
	if !l.active {
		return
	}
	l.mode = layerReplay
	saved := sc.Painter.State
	sc.Painter.State = l.scratch
	sc.RenderWidget()
	l.finish(sc)
	l.scratch.RenderDone() // (thrown away)
	sc.Painter.State = saved
}

// doNeedsRender renders the widgets of the scene that need to be rendered,
// into the image of the scene, and the layer again if any of them is in it
// (see [sceneLayer]), or the whole scene if it needs to be rendered.
func (sc *Scene) doNeedsRender() {
	if sc.WidgetBase.hasFlag(widgetNeedsRender) {
		sc.WidgetBase.setFlag(false, widgetChildNeedsRender)
		sc.renderWhole()
		return
	}
	l := &sc.layer
	if !l.active {
		sc.WidgetBase.doNeedsRender()
		return
	}
	l.mode = layerPartial
	sc.WidgetBase.doNeedsRender()
	l.mode = layerOff
	if l.needsRender {
		sc.renderLayer()
	}
}

// layerSource returns the [composer.Source] for the layer of the scene, to
// be composed over the scene, or nil if it has none.
func (sc *Scene) layerSource() composer.Source {
	l := &sc.layer
	if !l.active || l.renderer == nil {
		return nil
	}
	return &paintSource{render: l.state.RenderDone(), renderer: l.renderer, drawOp: draw.Over, drawPos: sc.SceneGeom.Pos}
}

// addRenderSources adds the sources of the scene to the given composer: its
// image, with the given draw operation, the widgets that render directly
// to the window (for the scene of the window), and its layer over them.
func (sc *Scene) addRenderSources(cp composer.Composer, op draw.Op, direct bool) {
	cp.Add(sc.RenderSource(op), sc)
	if direct {
		for _, dr := range sc.directRenders {
			cp.Add(dr.RenderSource(draw.Over), dr)
		}
	}
	if ls := sc.layerSource(); ls != nil {
		cp.Add(ls, &sc.layer)
	}
}
