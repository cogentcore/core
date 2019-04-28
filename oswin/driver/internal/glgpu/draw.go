// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glgpu

import (
	"image/draw"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin/gpu"
)

func init() {
	gpu.Draw = &Drawing{}
}

// Drawing provides commonly-used GPU drawing functions
// All operate on the current context with current program, target, etc
type Drawing struct {
}

// Clear clears the given properties of the current render target
func (dr *Drawing) Clear(color, depth bool) {
	bits := uint32(0)
	if color {
		bits |= gl.COLOR_BUFFER_BIT
	}
	if depth {
		bits |= gl.DEPTH_BUFFER_BIT
	}
	gl.Clear(bits)
}

// DepthTest turns on / off depth testing
func (dr *Drawing) DepthTest(on bool) {
	if on {
		gl.Enable(gl.DEPTH_TEST)
	} else {
		gl.Disable(gl.DEPTH_TEST)
	}
}

// Op sets the blend function based on go standard draw operation
// Src disables blending, and Over uses alpha-blending
func (dr *Drawing) Op(op draw.Op) {
	if op == draw.Over {
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
	} else {
		gl.Disable(gl.BLEND)
	}
}

// Triangles uses all existing settings to draw Triangles
// (non-indexed)
func (dr *Drawing) Triangles(start, count int) {
	gl.DrawArrays(gl.TRIANGLES, int32(start), int32(count))
}

// TriangleStrips uses all existing settings to draw TriangleStrip
// (non-indexed)
func (dr *Drawing) TriangleStrips(start, count int) {
	gl.DrawArrays(gl.TRIANGLE_STRIP, int32(start), int32(count))
}

// TrianglesIndexed uses all existing settings to draw Triangles
// Indexed
func (dr *Drawing) TrianglesIndexed(count int, idxs mat32.ArrayU32) {
	gl.DrawElements(gl.TRIANGLES, int32(count), gl.UNSIGNED_INT, gl.Ptr(idxs))
}

// TriangleStripsIndexed uses all existing settings to draw Triangles
// Indexed
func (dr *Drawing) TriangleStripsIndexed(count int, idxs mat32.ArrayU32) {
	gl.DrawElements(gl.TRIANGLE_STRIP, int32(count), gl.UNSIGNED_INT, gl.Ptr(idxs))
}
