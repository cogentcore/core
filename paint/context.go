// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/path"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
)

// Bounds represents an optimized rounded rectangle form of clipping,
// which is critical for GUI rendering.
type Bounds struct {
	// Rect is a rectangular bounding box.
	Rect math32.Box2

	// Radius is the border radius for rounded rectangles, can be per corner
	// or one value for all.
	Radius sides.Floats

	// Path is the computed clipping path for the Rect and Radius.
	Path path.Path

	// todo: probably need an image here for text
}

func NewBounds(w, h float32, radius sides.Floats) *Bounds {
	return &Bounds{Rect: math32.B2(0, 0, w, h), Radius: radius}
}

// Context contains all of the rendering constraints / filters / masks
// that are applied to elements being rendered.
// For SVG compliant rendering, we need a stack of these Context elements
// that apply to all elements in the group.
// Each level always represents the compounded effects of any parent groups,
// with the compounding being performed when a new Context is pushed on the stack.
// https://www.w3.org/TR/SVG2/render.html#Grouping
type Context struct {

	// Style has the accumulated style values.
	// Individual elements inherit from this style.
	Style styles.Paint

	// Transform is the accumulated transformation matrix.
	Transform math32.Matrix2

	// Bounds is the rounded rectangle clip boundary.
	// This is applied to the effective Path prior to adding to Render.
	Bounds Bounds

	// ClipPath is the current shape-based clipping path,
	// in addition to the Bounds, which is applied to the effective Path
	// prior to adding to Render.
	ClipPath path.Path

	// Mask is the current masking element, as rendered to a separate image.
	// This is composited with the rendering output to produce the final result.
	Mask image.Image

	// Filter // todo add filtering effects here
}

// NewContext returns a new Context using given paint style, bounds, and
// parent Context. See [Context.Init] for details.
func NewContext(sty *styles.Paint, bounds *Bounds, parent *Context) *Context {
	ctx := &Context{Style: *sty}
	ctx.Init(sty, bounds, parent)
	return ctx
}

// Init initializes context based on given style, bounds and parent Context.
// If parent is present, then bounds can be nil, in which
// case it gets the bounds from the parent.
// All the values from the style are used to update the Context,
// accumulating anything from the parent.
func (ctx *Context) Init(sty *styles.Paint, bounds *Bounds, parent *Context) {
	ctx.Style = *sty
	if parent == nil {
		ctx.Transform = sty.Transform
		bsz := bounds.Rect.Size()
		ctx.Bounds = *bounds
		ctx.Bounds.Path = path.RoundedRectangleSides(bounds.Rect.Min.X, bounds.Rect.Min.Y, bsz.X, bsz.Y, bounds.Radius)
		ctx.ClipPath = sty.ClipPath
		ctx.Mask = sty.Mask
		return
	}
	ctx.Transform = parent.Transform.Mul(sty.Transform)
	ctx.Style.InheritFields(&parent.Style)
	ctx.Style.UnitContext = parent.Style.UnitContext
	ctx.Style.ToDots() // update
	if bounds == nil {
		ctx.Bounds = parent.Bounds
	} else {
		ctx.Bounds = *bounds
		// todo: transform bp
		bsz := bounds.Rect.Size()
		bp := path.RoundedRectangleSides(bounds.Rect.Min.X, bounds.Rect.Min.Y, bsz.X, bsz.Y, bounds.Radius)
		ctx.Bounds.Path = bp.And(parent.Bounds.Path) // intersect
	}
	ctx.ClipPath = ctx.Style.ClipPath.And(parent.ClipPath)
	ctx.Mask = parent.Mask // todo: intersect with our own mask
}
