// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image/draw"

	"github.com/goki/gi/mat32"
)

// TheDraw is the current oswin gpu Draw instance
var TheDraw Draw

// Draw provides commonly-used GPU drawing functions
// All operate on the current context with current program, target, etc
type Draw interface {
	// Clear clears the given properties of the current render target
	Clear(color, depth bool)

	// Op sets the blend function based on go standard draw operation
	// Src disables blending, and Over uses alpha-blending
	Op(op draw.Op)

	// Triangles uses all existing settings to draw Triangles
	// (non-indexed)
	Triangles(start, count int)

	// TriangleStrips uses all existing settings to draw Triangles Strips
	// (non-indexed)
	TriangleStrips(start, count int)

	// TrianglesIndexed uses all existing settings to draw Triangles
	// Indexed
	TrianglesIndexed(count int, idxs mat32.ArrayU32)

	// TriangleStripsIndexed uses all existing settings to draw Triangle Strips
	// Indexed
	TriangleStripsIndexed(count int, idxs mat32.ArrayU32)
}
