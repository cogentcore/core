// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"
	"image/draw"
)

// Draw is the current oswin gpu Drawing instance.
// Call methods as, e.g.: gpu.Draw.Triangles(..) etc..
var Draw Drawing

// Drawing provides commonly-used GPU drawing functions and state settings that affect drawing.
// All operate on the current context with current program, target, etc
type Drawing interface {
	// Clear clears the given properties of the current render target
	Clear(color, depth bool)

	// ClearColor sets the color to draw when clear is called
	ClearColor(r, g, b float32)

	// DepthTest turns on / off depth testing (standard less-than-or-equal depth assumed).
	DepthTest(on bool)

	// StencilTest turns on / off stencil testing
	StencilTest(on bool)

	// CullFace sets face culling, for front and / or back faces (back typical).
	// If you don't do this, rendering of standard Phong model will not work.
	// if ccw = true then standard CCW face ordering is used, else CW (clockwise).
	CullFace(front, back, ccw bool)

	// Multisample turns on or off multisampling (antialiasing)
	Multisample(on bool)

	// Op sets the blend function based on go standard draw operation
	// Src disables blending, and Over uses alpha-blending
	Op(op draw.Op)

	// Wireframe sets the rendering to lines instead of fills if on = true
	Wireframe(on bool)

	// Viewport sets the rendering viewport to given rectangle.
	// It is important to update this for each render -- cannot assume it.
	Viewport(rect image.Rectangle)

	// Triangles uses all existing settings to draw Triangles
	// (non-indexed)
	Triangles(start, count int)

	// TriangleStrips uses all existing settings to draw Triangles Strips
	// (non-indexed)
	TriangleStrips(start, count int)

	// TrianglesIndexed uses all existing settings to draw Triangles Indexed.
	// You must have activated an IndexesBuffer that supplies
	// the indexes, and start + count determine range of such indexes
	// to use, and must be within bounds for that.
	TrianglesIndexed(start, count int)

	// TriangleStripsIndexed uses all existing settings to draw Triangle Strips Indexed.
	// You must have activated an IndexesBuffer that supplies
	// the indexes, and start + count determine range of such indexes
	// to use, and must be within bounds for that.
	TriangleStripsIndexed(start, count int)

	// Flush ensures that all rendering is pushed to current render target.
	// Especially useful for rendering to framebuffers (Window SwapBuffer
	// automatically does a flush)
	Flush()
}
