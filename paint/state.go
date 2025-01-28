// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"
	"log/slog"

	"cogentcore.org/core/paint/path"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
)

// The State holds all the current rendering state information used
// while painting. The [Paint] embeds a pointer to this.
type State struct {

	// Renderers are the current renderers.
	Renderers []Renderer

	// Stack provides the SVG "stacking context" as a stack of [Context]s.
	// There is always an initial base-level Context element for the overall
	// rendering context.
	Stack []*Context

	// Render is the current render state that we are building.
	Render Render

	// Path is the current path state we are adding to.
	Path path.Path

	// todo: this needs to be removed and replaced with new Image Render recording.
	Image *image.RGBA
}

// InitImageRaster initializes the [State] with the default image-based
// rasterizing renderer, using the given overall styles, size, and image.
// It must be called whenever the image size changes.
func (rs *State) InitImageRaster(sty *styles.Paint, width, height int, img *image.RGBA) {
	// todo: make a default renderer
	rs.Stack = []*Context{NewContext(sty, NewBounds(float32(width), float32(height), sides.Floats{}), nil)}
	rs.Image = img
}

// Context() returns the currently active [Context] state (top of Stack).
func (rs *State) Context() *Context {
	return rs.Stack[len(rs.Stack)-1]
}

// PushContext pushes a new [Context] onto the stack using given styles and bounds.
// The transform from the style will be applied to all elements rendered
// within this group, along with the other group properties.
// This adds the Context to the current Render state as well, so renderers
// that track grouping will track this.
// Must protect within render mutex lock (see Lock version).
func (rs *State) PushContext(sty *styles.Paint, bounds *Bounds) *Context {
	parent := rs.Context()
	g := NewContext(sty, bounds, parent)
	rs.Stack = append(rs.Stack, g)
	rs.Render.Add(&ContextPush{Context: *g})
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
	rs.Render.Add(&ContextPop{})
}
