// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package render

// Render represents a collection of render [Item]s to be rendered.
type Render []Item

// Item is a union interface for render items: Path, text.Text, or Image.
type Item interface {
	IsRenderItem()
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

// ContextPush is a [Context] push render item, which can be used by renderers
// that track group structure (e.g., SVG).
type ContextPush struct {
	Context Context
}

// interface assertion.
func (p *ContextPush) IsRenderItem() {
}

// ContextPop is a [Context] pop render item, which can be used by renderers
// that track group structure (e.g., SVG).
type ContextPop struct {
}

// interface assertion.
func (p *ContextPop) IsRenderItem() {
}
