// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"
	"log/slog"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/paint/renderers/rasterx"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
)

var (
	// NewSourceRenderer returns the [composer.Source] renderer
	// for [Painter] source content, for the current platform.
	// This is created first for Source painters.
	NewSourceRenderer func(size math32.Vector2) render.Renderer
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

// InitImageRaster initializes the [State] and ensures that there is
// a [rasterx.Renderer] that rasterizes [Painter] items to a
// Go [image.RGBA].
// If renderers exist, then the size is updated for the first one
// (no cost if same size). This must be called whenever the image size changes.
func (rs *State) InitImageRaster(sty *styles.Paint, width, height int) {
	sz := math32.Vec2(float32(width), float32(height))
	bounds := render.NewBounds(0, 0, float32(width), float32(height), sides.Floats{})
	if len(rs.Renderers) == 0 {
		rd := rasterx.New(sz)
		rs.Renderers = append(rs.Renderers, rd)
		rs.Stack = []*render.Context{render.NewContext(sty, bounds, nil)}
		return
	}
	ctx := rs.Context()
	ctx.SetBounds(bounds)
	rs.Renderers[0].SetSize(units.UnitDot, sz)
}

// InitSourceRaster initializes the [State] and creates a [composer.Source]
// renderer appropriate for the current platform if none exist.
// If renderers exist, then the size is updated (no cost if size is the same).
// This must be called whenever the image size changes.
func (rs *State) InitSourceRaster(sty *styles.Paint, width, height int) {
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

// Context() returns the currently active [render.Context] state (top of Stack).
func (rs *State) Context() *render.Context {
	return rs.Stack[len(rs.Stack)-1]
}

// ImageRenderer returns the [rasterx.Renderer] image rasterizer if it is
// the first renderer, or nil.
func (rs *State) ImageRenderer() render.Renderer {
	if len(rs.Renderers) == 0 {
		return nil
	}
	rd, ok := rs.Renderers[0].(*rasterx.Renderer)
	if !ok {
		return nil
	}
	return rd
}

// RenderImage returns the Go [image.RGBA] from the first [Image] renderer
// if present, else nil.
func (rs *State) RenderImage() *image.RGBA {
	rd := rs.ImageRenderer()
	if rd == nil {
		return nil
	}
	return rd.(*rasterx.Renderer).Image()
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
