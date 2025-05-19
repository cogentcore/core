// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package render

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/shaped"
)

// Text is a text rendering render item.
type Text struct {
	// Text contains shaped Lines of text to be rendered, as produced by a
	// [shaped.Shaper]. Typically this text is  configured so that the
	// Postion is at the upper left corner of the resulting text rendering.
	Text *shaped.Lines

	// Position to render, which typically specifies the upper left corner of
	// the Text. This is added directly to the offsets and is transformed by the
	// active transform matrix. See also PositionAbs
	Position math32.Vector2

	// Context has the full accumulated style, transform, etc parameters
	// for rendering, combining the current state context (e.g.,
	// from any higher-level groups) with the current element's style parameters.
	Context Context
}

func NewText(txt *shaped.Lines, sty *styles.Paint, ctx *Context, pos math32.Vector2) *Text {
	nt := &Text{Text: txt, Position: pos}
	nt.Context.Init(sty, nil, ctx)
	return nt
}

// interface assertion.
func (tx *Text) IsRenderItem() {}
