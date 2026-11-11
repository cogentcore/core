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
	NewSourceRenderer func(size math32.Vector2) render.Renderer

	// NewImageRenderer returns a painter renderer for generating
	// images locally in Go regardless of platform.
	NewImageRenderer func(size math32.Vector2) render.Renderer

	// NewSVGRenderer returns a structured SVG renderer that can
	// generate an SVG vector graphics document from painter content.
	NewSVGRenderer func(size math32.Vector2) render.Renderer

	// NewPDFRenderer returns a PDF renderer that can
	// generate a PDF document from painter content.
	NewPDFRenderer func(size math32.Vector2, un *units.Context) render.Renderer
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
// accumulated painter actions to an SVG document using a [NewSVGRenderer]
func RenderToSVG(pc *Painter) []byte {
	rd := NewSVGRenderer(pc.Size)
	return rd.Render(pc.RenderDone()).Source()
}

// RenderToPDF is a convenience function that renders the current
// accumulated painter actions to a PDF document using a [NewPDFRenderer]
func RenderToPDF(pc *Painter) []byte {
	p := pc.RenderDone()
	rd := NewPDFRenderer(pc.Size, &pc.Context().Style.UnitContext)
	return rd.Render(p).Source()
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

	// pushes are the indexes in Render of the [render.ContextPush] items
	// of the contexts of the Stack after the base one.
	pushes []int
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
	rs.pushes = nil
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
	rs.pushes = rs.pushes[:0]
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
	rs.pushes = append(rs.pushes, len(rs.Render))
	rs.Render.Add(&render.ContextPush{Context: *g})
	return g
}

// PushStack pushes the contexts of the stack of the given other state above
// its base context (their clipping and transforms) onto this one, so that the
// rendering on this one continues within them, as for the rendering of a
// part of a scene into another image (in which the widgets that it is in
// pop them). See [State.PopStack].
func (rs *State) PushStack(other *State) {
	for _, g := range other.Stack[1:] {
		rs.Stack = append(rs.Stack, g)
		rs.pushes = append(rs.pushes, len(rs.Render))
		rs.Render.Add(&render.ContextPush{Context: *g})
	}
}

// PopStack pops all of the contexts of the stack above its base context.
func (rs *State) PopStack() {
	for len(rs.Stack) > 1 {
		rs.PopContext()
	}
}

// SetLayer sets the [render.Context.ClipMask], [render.Context.Mask], and
// [render.Context.Opacity] of the context at the top of the stack (pushed
// with [State.PushContext]), with which its rendering is composited as a
// whole (as a layer). The masks are typically rendered with
// [State.RenderSeparately] after the context is pushed, so that they
// have its transform.
func (rs *State) SetLayer(clipMask, mask render.Render, opacity float32) {
	n := len(rs.pushes)
	if n == 0 {
		return
	}
	g := rs.Stack[len(rs.Stack)-1]
	g.ClipMask, g.Mask, g.Opacity = clipMask, mask, opacity
	if i := rs.pushes[n-1]; i < len(rs.Render) {
		if cp, ok := rs.Render[i].(*render.ContextPush); ok {
			cp.Context.ClipMask, cp.Context.Mask, cp.Context.Opacity = clipMask, mask, opacity
		}
	}
}

// RenderSeparately calls the given function, which renders with this
// state, and returns what it renders, which is not added to the current
// rendering (as for the items of a mask, see [State.SetLayer]).
func (rs *State) RenderSeparately(f func()) render.Render {
	saved, savedPushes := rs.Render, rs.pushes
	rs.Render, rs.pushes = nil, nil
	f()
	r := rs.Render
	rs.Render, rs.pushes = saved, savedPushes
	return r
}

// PopContext pops the current Context off of the Stack.
func (rs *State) PopContext() {
	n := len(rs.Stack)
	if n == 1 {
		slog.Error("programmer error: paint.State.PopContext: stack is at base starting point")
		return
	}
	rs.Stack = rs.Stack[:n-1]
	if m := len(rs.pushes); m > 0 {
		rs.pushes = rs.pushes[:m-1]
	}
	rs.Render.Add(&render.ContextPop{})
}
