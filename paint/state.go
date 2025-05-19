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
)

var (
	// NewSourceRenderer returns the [composer.Source] renderer
	// for [Painter] rendering, for the current platform.
	NewSourceRenderer func(size math32.Vector2) render.Renderer

	// NewImageRenderer returns a painter renderer for generating
	// images locally in Go regardless of platform.
	NewImageRenderer func(size math32.Vector2) render.Renderer

	// NewSVGRenderer returns a structured SVG renderer that can
	// generate an SVG vector graphics document from painter content.
	NewSVGRenderer func(size math32.Vector2) render.Renderer
)

// RenderToImage is a convenience function that renders the current
// accumulated painter actions to an image using a [NewImageRenderer],
// and returns the Image() call from that renderer.
// The image is wrapped by [imagex.WrapJS] so that it is ready to be
// used efficiently for subsequent rendering actions on the JS (web) platform.
func RenderToImage(pc *Painter) image.Image {
	rd := NewImageRenderer(pc.Size)
	return imagex.WrapJS(rd.Render(pc.RenderDone()).Image())
}

// RenderToSVG is a convenience function that renders the current
// accumulated painter actions to an SVG document using a
// [NewSVGRenderer].n
func RenderToSVG(pc *Painter) []byte {
	rd := NewSVGRenderer(pc.Size)
	return rd.Render(pc.RenderDone()).Source()
}

// The State holds all the current rendering state information used
// while painting. The [Paint] embeds a pointer to this.
type State struct {
	// Size in dots (true pixels) as specified during Init.
	Size math32.Vector2

	// Stack provides the SVG "stacking context" as a stack of [Context]s.
	// There is always an initial base-level Context element for the overall
	// rendering context.
	Stack []*render.Context

	// Render holds the current [render.PaintRender] state that we are building.
	// and has the list of [render.Renderer]s that we render to.
	Render render.Render

	// Path is the current path state we are adding to.
	Path ppath.Path
}

// Init initializes the rendering state, creating a new Stack
// with an initial baseline context using given size and styles.
// Size is used to set the bounds for clipping rendering, assuming
// units are image dots (true pixels), which is typical.
// This should be called whenever the size changes.
func (rs *State) Init(sty *styles.Paint, size math32.Vector2) {
	rs.Size = size
	bounds := render.NewBounds(0, 0, size.X, size.Y, sides.Floats{})
	rs.Stack = []*render.Context{render.NewContext(sty, bounds, nil)}
	rs.Render = nil
	rs.Path = nil
}

// RenderDone should be called when the full set of rendering
// for this painter is done. It returns a self-contained
// [render.Render] representing the entire rendering state,
// suitable for rendering by passing to a [render.Renderer].
// It resets the current painter state so that it is ready for
// new rendering.
func (rs *State) RenderDone() render.Render {
	npr := rs.Render.Clone()
	rs.Render.Reset()
	rs.Path.Reset()
	if len(rs.Stack) > 1 { // ensure back to baseline stack
		rs.Stack = rs.Stack[:1]
	}
	return npr
}

// Context() returns the currently active [render.Context] state (top of Stack).
func (rs *State) Context() *render.Context {
	return rs.Stack[len(rs.Stack)-1]
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
