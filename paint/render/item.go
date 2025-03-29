// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package render

// Item is a union interface for render items:
// [Path], [Text], [Image], and [ContextPush].
type Item interface {
	IsRenderItem()
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
