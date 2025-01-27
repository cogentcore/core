// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"cogentcore.org/core/paint/path"
	"cogentcore.org/core/styles"
)

// Render represents a collection of render [Item]s to be rendered.
type Render []Item

// Item is a union interface for render items: Path, text.Text, or Image.
type Item interface {
	isRenderItem()
}

// Path is a path drawing render item: responsible for all vector graphics
// drawing functionality.
type Path struct {
	// Path specifies the shape(s) to be drawn, using commands:
	// MoveTo, LineTo, QuadTo, CubeTo, ArcTo, and Close.
	// Each command has the applicable coordinates appended after it,
	// like the SVG path element.
	Path path.Path

	// Style has the styling parameters for rendering the path,
	// including colors, stroke width, etc.
	Style styles.Path
}

// interface assertion.
func (p *Path) isRenderItem() {
}
