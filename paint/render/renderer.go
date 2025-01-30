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

	// IsImage returns true if the renderer generates an image,
	// as in a rasterizer. Others generate structured vector graphics
	// files such as SVG or PDF.
	IsImage() bool

	// Image returns the current rendered image as an image.RGBA,
	// if this is an image-based renderer.
	Image() *image.RGBA

	// Code returns the current rendered image data representation
	// for non-image-based renderers, e.g., the SVG file.
	Code() []byte

	// Size returns the size of the render target, in its preferred units.
	// For image-based (IsImage() == true), it will be [units.UnitDot]
	// to indicate the actual raw pixel size.
	// Direct configuration of the Renderer happens outside of this interface.
	Size() (units.Units, math32.Vector2)

	// SetSize sets the render size in given units. [units.UnitDot] is
	// used for image-based rendering.
	SetSize(un units.Units, size math32.Vector2)

	// Render renders the list of render items.
	Render(r Render)
}

// Registry of renderers
var Renderers map[string]Renderer
