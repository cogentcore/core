// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package render

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/shaped"
)

// Text is a text rendering render item.
type Text struct {
	// todo: expand to a collection of lines!
	Text *shaped.Lines

	// Position to render, which specifies the baseline of the starting line.
	Position math32.Vector2

	// Context has the full accumulated style, transform, etc parameters
	// for rendering the path, combining the current state context (e.g.,
	// from any higher-level groups) with the current element's style parameters.
	Context Context
}

func NewText(txt *shaped.Lines, ctx *Context, pos math32.Vector2) *Text {
	return &Text{Text: txt, Context: *ctx, Position: pos}
}

// interface assertion.
func (tx *Text) IsRenderItem() {}
