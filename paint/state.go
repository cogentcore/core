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
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
)

// NewDefaultImageRenderer is a function that returns the default image renderer
var NewDefaultImageRenderer func(size math32.Vector2) render.Renderer

// The State holds all the current rendering state information used
// while painting. The [Paint] embeds a pointer to this.
type State struct {

	// Render holds the current [render.PaintRender] state that we are building.
	// and has the list of [render.Renderer]s that we render to.
	Render render.PaintRender

	// Stack provides the SVG "stacking context" as a stack of [Context]s.
	// There is always an initial base-level Context element for the overall
	// rendering context.
	Stack []*render.Context

	// Path is the current path state we are adding to.
	Path ppath.Path
}

// InitImageRaster initializes the [State] and ensures that there is
// at least one image-based renderer present, creating the default type if not,
// using the [NewDefaultImageRenderer] function.
// If renderers exist, then the size is updated for any image-based ones.
// This must be called whenever the image size changes.
func (rs *State) InitImageRaster(sty *styles.Paint, width, height int) {
	sz := math32.Vec2(float32(width), float32(height))
	bounds := render.NewBounds(0, 0, float32(width), float32(height), sides.Floats{})
	if len(rs.Render.Renderers) == 0 {
		rd := NewDefaultImageRenderer(sz)
		rs.Render.Renderers = append(rs.Render.Renderers, rd)
		rs.Stack = []*render.Context{render.NewContext(sty, bounds, nil)}
		return
	}
	ctx := rs.Context()
	ctx.SetBounds(bounds)
	for _, rd := range rs.Render.Renderers {
		if rd.Type() == render.Code {
			continue
		}
		rd.SetSize(units.UnitDot, sz)
	}
}

// Context() returns the currently active [render.Context] state (top of Stack).
func (rs *State) Context() *render.Context {
	return rs.Stack[len(rs.Stack)-1]
}

// RenderImage returns the current render image from the first
// Image renderer present, or nil if none.
// This may be somewhat expensive for some rendering types.
func (rs *State) RenderImage() *image.RGBA {
	return rs.Render.Image()
}

// RenderImageSize returns the size of the current render image
// from the first Image renderer present.
func (rs *State) RenderImageSize() image.Point {
	rd := rs.Render.ImageRenderer()
	if rd == nil {
		return image.Point{}
	}
	_, sz := rd.Size()
	return sz.ToPoint()
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
