// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package render

import (
	"image"
	"image/draw"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
)

//go:generate core generate

// RendererTypes are types of rendererers.
type RendererTypes int32 //enums:enum

const (
	// Image is a rasterizing renderer capable of returning a standard
	// go *image.RGBA image.
	Image RendererTypes = iota

	// Drawer is a direct renderer that uses the [system.Drawer] to
	// render directly, potentially only for a GPU drawer (e.g., [xyz]).
	Drawer

	// Code is a renderer that generates some kind of structured code
	// to represent the render, as in SVG or PDF. The output is []byte.
	Code
)

// Renderer is the interface for all backend rendering outputs.
type Renderer interface {

	// Render renders the given Render data.
	Render(r Render)

	// Type returns the type of renderer this is.
	Type() RendererTypes

	// Image returns the current rendered image as an image.RGBA,
	// if this is an [Image] renderer.
	Image() *image.RGBA

	// Code returns the current rendered image data representation
	// for [Code] renderers, e.g., the SVG file.
	Code() []byte

	// Draw draws the render to the given [system.Drawer] using given
	// compositing operation: Over = alpha blend with current,
	// Src = copy source. This is supported for [Image] and [Drawer]
	// rendering types.
	Draw(r Render, drw system.Drawer, op draw.Op)

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
}

// Registry of renderers
var Renderers map[string]Renderer
