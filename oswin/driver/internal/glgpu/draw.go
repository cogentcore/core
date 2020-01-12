// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glgpu

import (
	"image"
	"image/draw"

	"github.com/go-gl/gl/v3.3-core/gl"
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
		bits |= gl.STENCIL_BUFFER_BIT
	}
	gl.Clear(bits)
	gl.Enable(gl.POLYGON_OFFSET_FILL)
	gl.Enable(gl.POLYGON_OFFSET_LINE)
	gl.Enable(gl.POLYGON_OFFSET_POINT)
}

// ClearColor sets the color to draw when clear is called
func (dr *Drawing) ClearColor(r, g, b float32) {
	gl.ClearColor(r, g, b, 1)
}

// DepthTest turns on / off depth testing
func (dr *Drawing) DepthTest(on bool) {
	if on {
		gl.Enable(gl.DEPTH_TEST)
		gl.DepthFunc(gl.LEQUAL)
		gl.DepthMask(true)
	} else {
		gl.Disable(gl.DEPTH_TEST)
	}
}

// StencilTest turns on / off stencil testing
func (dr *Drawing) StencilTest(on bool) {
	if on {
		gl.Enable(gl.STENCIL_TEST)
	} else {
		gl.Disable(gl.STENCIL_TEST)
	}
}

// CullFace sets face culling, for front and / or back faces (back typical).
// If you don't do this, rendering of standard Phong model will not work.
func (dr *Drawing) CullFace(front, back, ccw bool) {
	if front || back {
		if ccw {
			gl.FrontFace(gl.CCW)
		} else {
			gl.FrontFace(gl.CW)
		}
		switch {
		case front && back:
			gl.CullFace(gl.FRONT_AND_BACK)
		case front:
			gl.CullFace(gl.FRONT)
		case back:
			gl.CullFace(gl.BACK)
		}
		gl.Enable(gl.CULL_FACE)
	} else {
		gl.Disable(gl.CULL_FACE)
	}
}

// Multisample turns on or off multisampling (antialiasing)
func (dr *Drawing) Multisample(on bool) {
	if on {
		gl.Enable(gl.MULTISAMPLE)
	} else {
		gl.Disable(gl.MULTISAMPLE)
	}
}

// Op sets the blend function based on go standard draw operation
// Src disables blending, and Over uses alpha-blending
func (dr *Drawing) Op(op draw.Op) {
	if op == draw.Over {
		gl.Enable(gl.BLEND)
		gl.BlendEquation(gl.FUNC_ADD)
		gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
	} else {
		gl.Disable(gl.BLEND)
	}
}

// Wireframe sets the rendering to lines instead of fills if on = true
func (dr *Drawing) Wireframe(on bool) {
	if on {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	} else {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	}
}

// Viewport sets the rendering viewport to given rectangle.
// It is important to update this for each render -- cannot assume it.
func (dr *Drawing) Viewport(rect image.Rectangle) {
	gl.Viewport(int32(rect.Min.X), int32(rect.Min.Y), int32(rect.Max.X), int32(rect.Max.Y))
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

// TrianglesIndexed uses all existing settings to draw Triangles Indexed.
// You must have activated an IndexesBuffer that supplies
// the indexes, and start + count determine range of such indexes
// to use, and must be within bounds for that.
func (dr *Drawing) TrianglesIndexed(start, count int) {
	gl.DrawElements(gl.TRIANGLES, int32(count), gl.UNSIGNED_INT, gl.PtrOffset(start*4))
}

// TriangleStripsIndexed uses all existing settings to draw Triangle Strips Indexed.
// You must have activated an IndexesBuffer that supplies
// the indexes, and start + count determine range of such indexes
// to use, and must be within bounds for that.
func (dr *Drawing) TriangleStripsIndexed(start, count int) {
	gl.DrawElements(gl.TRIANGLE_STRIP, int32(count), gl.UNSIGNED_INT, gl.PtrOffset(start*4))
}

// Flush ensures that all rendering is pushed to current render target.
// Especially useful for rendering to framebuffers (Window SwapBuffer
// automatically does a flush)
func (dr *Drawing) Flush() {
	gl.Flush()
}
