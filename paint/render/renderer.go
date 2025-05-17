// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package render

import (
	"image"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
)

// Renderer is the interface for all backend rendering outputs.
type Renderer interface {

	// Render renders the given Render data.
	Render(r Render) Renderer

	// Size returns the size of the render target, in its preferred units.
	// For [Image] types, it will be [units.UnitDot] to indicate the actual
	// raw pixel size.
	Size() (units.Units, math32.Vector2)

	// SetSize sets the render size in given units. [units.UnitDot] is
	// used for [Image] and [Draw] renderers. Direct configuration of
	// other Renderer properties happens outside of this interface.
	// This is used for resizing [Image] and [Draw] renderers when
	// the relevant Scene size changes.
	SetSize(un units.Units, size math32.Vector2)

	// Image returns the rendered image after rendering.
	// If nil, then images are not supported by this renderer.
	Image() image.Image

	// Source returns the document source code representation
	// of the rendering output. This is supported by e.g., SVG
	// and PDF renderers.
	Source() []byte
}
