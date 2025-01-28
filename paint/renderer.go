// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/path"
	"cogentcore.org/core/styles/units"
)

// Renderer is the interface for all backend rendering outputs.
type Renderer interface {
	// RenderSize returns the size of the render target, in its preferred units.
	// For images, it will be [units.UnitDot] to indicate the actual raw pixel size.
	// Direct configuration of the Renderer happens outside of this interface.
	RenderSize() (units.Units, math32.Vector2)

	// Render renders the list of render items.
	Render(r Render)
}

// Render represents a collection of render [Item]s to be rendered.
type Render []Item

// Item is a union interface for render items: Path, text.Text, or Image.
type Item interface {
	isRenderItem()
}

// Add adds item(s) to render.
func (r *Render) Add(item ...Item) Render {
	*r = append(*r, item...)
	return *r
}

// Reset resets back to an empty Render state.
// It preserves the existing slice memory for re-use.
func (r *Render) Reset() Render {
	*r = (*r)[:0]
	return *r
}

// Path is a path drawing render item: responsible for all vector graphics
// drawing functionality.
type Path struct {
	// Path specifies the shape(s) to be drawn, using commands:
	// MoveTo, LineTo, QuadTo, CubeTo, ArcTo, and Close.
	// Each command has the applicable coordinates appended after it,
	// like the SVG path element. The coordinates are in the original
	// units as specified in the Paint drawing commands, without any
	// transforms applied. See [Path.Transform].
	Path path.Path

	// Context has the full accumulated style, transform, etc parameters
	// for rendering the path, combining the current state context (e.g.,
	// from any higher-level groups) with the current element's style parameters.
	Context Context
}

// interface assertion.
func (p *Path) isRenderItem() {
}

// ContextPush is a [Context] push render item, which can be used by renderers
// that track group structure (e.g., SVG).
type ContextPush struct {
	Context Context
}

// interface assertion.
func (p *ContextPush) isRenderItem() {
}

// ContextPop is a [Context] pop render item, which can be used by renderers
// that track group structure (e.g., SVG).
type ContextPop struct {
}

// interface assertion.
func (p *ContextPop) isRenderItem() {
}

// Registry of renderers
var Renderers map[string]Renderer
