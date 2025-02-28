// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package render

import (
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
)

// Path is a path drawing render [Item]: responsible for all vector graphics
// drawing functionality.
type Path struct {
	// Path specifies the shape(s) to be drawn, using commands:
	// MoveTo, LineTo, QuadTo, CubeTo, ArcTo, and Close.
	// Each command has the applicable coordinates appended after it,
	// like the SVG path element. The coordinates are in the original
	// units as specified in the Paint drawing commands, without any
	// transforms applied. See [Path.Transform].
	Path ppath.Path

	// Context has the full accumulated style, transform, etc parameters
	// for rendering the path, combining the current state context (e.g.,
	// from any higher-level groups) with the current element's style parameters.
	Context Context
}

func NewPath(pt ppath.Path, sty *styles.Paint, ctx *Context) *Path {
	pe := &Path{Path: pt}
	pe.Context.Init(sty, nil, ctx)
	return pe
}

// interface assertion.
func (p *Path) IsRenderItem() {}
