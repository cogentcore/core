// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package render

import "fmt"

// Item is a union interface for render items:
// [Path], [Text], [Image], and [ContextPush].
type Item interface {
	fmt.Stringer

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

func (p *ContextPush) String() string {
	return "ctx-push: " + p.Context.Cumulative.String()
}

// ContextPop is a [Context] pop render item, which can be used by renderers
// that track group structure (e.g., SVG).
type ContextPop struct {
}

func (p *ContextPop) String() string {
	return "ctx-pop"
}

// interface assertion.
func (p *ContextPop) IsRenderItem() {
}
