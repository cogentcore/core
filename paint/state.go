// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"
	"log/slog"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
)

var (
	// NewSourceRenderer returns the [composer.Source] renderer
	// for [Painter] rendering, for the current platform.
	// This is created first for Source painters.
	NewSourceRenderer func(size math32.Vector2) render.Renderer

	// NewImageRenderer returns a painter renderer for generating
	// images locally in Go regardless of platform.
	NewImageRenderer func(size math32.Vector2) render.Renderer

	// NewSVGRenderer returns a structured SVG renderer that can
	// generate an SVG vector graphics document from painter content.
	NewSVGRenderer func(size math32.Vector2) render.Renderer
)

// The State holds all the current rendering state information used
// while painting. The [Paint] embeds a pointer to this.
type State struct {

	// Render holds the current [render.PaintRender] state that we are building.
	// and has the list of [render.Renderer]s that we render to.
	Render render.Render

	// Renderers is the list of [Renderer]s that we render to.
	Renderers []render.Renderer

	// Stack provides the SVG "stacking context" as a stack of [Context]s.
	// There is always an initial base-level Context element for the overall
	// rendering context.
	Stack []*render.Context

	// Path is the current path state we are adding to.
	Path ppath.Path
}

// InitImageRender initializes the [State] and ensures that there is
// an image renderer that rasterizes [Painter] items to a Go [image.RGBA]
// (defaults to [rasterx.Renderer]).
// If renderers exist, then the size is updated for the first one
// (no cost if same size). This must be called whenever the image size changes.
func (rs *State) InitImageRender(sty *styles.Paint, width, height int) {
	sz := math32.Vec2(float32(width), float32(height))
	bounds := render.NewBounds(0, 0, float32(width), float32(height), sides.Floats{})
	if len(rs.Renderers) == 0 {
		rd := NewImageRenderer(sz)
		rs.Renderers = append(rs.Renderers, rd)
		rs.Stack = []*render.Context{render.NewContext(sty, bounds, nil)}
		return
	}
	ctx := rs.Context()
	ctx.SetBounds(bounds)
	rs.Renderers[0].SetSize(units.UnitDot, sz)
}

// InitSourceRender initializes the [State] and creates a [composer.Source]
// renderer appropriate for the current platform if none exist.
// If renderers exist, then the size is updated (no cost if size is the same).
// This must be called whenever the image size changes.
func (rs *State) InitSourceRender(sty *styles.Paint, width, height int) {
	sz := math32.Vec2(float32(width), float32(height))
	bounds := render.NewBounds(0, 0, float32(width), float32(height), sides.Floats{})
	if len(rs.Renderers) == 0 {
		rd := NewSourceRenderer(sz)
		rs.Renderers = append(rs.Renderers, rd)
		rs.Stack = []*render.Context{render.NewContext(sty, bounds, nil)}
		return
	}
	ctx := rs.Context()
	ctx.SetBounds(bounds)
	rs.Renderers[0].SetSize(units.UnitDot, sz)
}

// DeleteExtraRenderers deletes any renderers beyond the first one.
func (rs *State) DeleteExtraRenderers() {
	if len(rs.Renderers) <= 1 {
		return
	}
	rs.Renderers = []render.Renderer{rs.Renderers[0]}
}

// Context() returns the currently active [render.Context] state (top of Stack).
func (rs *State) Context() *render.Context {
	return rs.Stack[len(rs.Stack)-1]
}

// RenderImage returns the image.Image from the first [Image] renderer
// if present, else nil.
func (rs *State) RenderImage() image.Image {
	if len(rs.Renderers) == 0 {
		return nil
	}
	rd := rs.Renderers[0]
	// todo: this could be a pure platform-side image (e.g., all on JS or GPU)
	return imagex.WrapJS(rd.Image())
}

// PushContext pushes a new [render.Context] onto the stack using given styles and bounds.
// The transform from the style will be applied to all elements rendered
// within this group, along with the other group properties.
// This adds the Context to the current Render state as well, so renderers
// that track grouping will track this.
// Must protect within render mutex lock (see Lock version).
func (rs *State) PushContext(sty *styles.Paint, bounds *render.Bounds) *render.Context {
	parent := rs.Context()
	g := render.NewContext(sty, bounds, parent)
	rs.Stack = append(rs.Stack, g)
	rs.Render.Add(&render.ContextPush{Context: *g})
	return g
}

// PopContext pops the current Context off of the Stack.
func (rs *State) PopContext() {
	n := len(rs.Stack)
	if n == 1 {
		slog.Error("programmer error: paint.State.PopContext: stack is at base starting point")
		return
	}
	rs.Stack = rs.Stack[:n-1]
	rs.Render.Add(&render.ContextPop{})
}

// todo: units!

// AddSVGRender adds an SVG renderer to the list of renderers.
// Assumes state is already initialized with another renderer.
func (rs *State) AddSVGRenderer(width, height int) render.Renderer {
	sz := math32.Vec2(float32(width), float32(height))
	rd := NewSVGRenderer(sz)
	rs.Renderers = append(rs.Renderers, rd)
	return rd
}
