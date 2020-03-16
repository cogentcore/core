// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glgpu

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/goki/mat32"
)

// IndexesBuffer manages a buffer of indexes for index-based rendering
// (i.e., GL_ELEMENT_ARRAY_BUFFER for glDrawElements calls in OpenGL).
type IndexesBuffer struct {
	init   bool
	handle uint32
	ln     int
	idxs   mat32.ArrayU32
}

// SetLen sets the number of indexes in buffer
func (ib *IndexesBuffer) SetLen(ln int) {
	ib.ln = ln
	ib.idxs = make(mat32.ArrayU32, ln)
}

// Len returns the number of indexes in bufer
func (ib *IndexesBuffer) Len() int {
	return ib.ln
}

// Set sets the indexes by copying given data
func (ib *IndexesBuffer) Set(idxs mat32.ArrayU32) {
	if len(idxs) == 0 {
		return
	}
	if ib.ln == 0 {
		ib.ln = len(idxs)
	}
	if ib.idxs == nil {
		ib.idxs = make(mat32.ArrayU32, ib.ln)
	}
	copy(ib.idxs, idxs)
}

// Returns the indexes (direct copy of internal buffer -- can be modified)
func (ib *IndexesBuffer) Indexes() mat32.ArrayU32 {
	return ib.idxs
}

// Activate binds buffer as active one
func (ib *IndexesBuffer) Activate() {
	if !ib.init {
		gl.GenBuffers(1, &ib.handle)
		ib.init = true
	}
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ib.handle)
}

// Handle returns the unique handle for this buffer -- only valid after Activate()
func (ib *IndexesBuffer) Handle() uint32 {
	return ib.handle
}

// Transfer transfers data to GPU -- Activate must have been called with no other
// such buffers activated in between.  Automatically uses re-specification
// strategy per: https://www.khronos.org/opengl/wiki/Buffer_Object_Streaming
// so it is safe if buffer was still being used from prior GL rendering call.
func (ib *IndexesBuffer) Transfer() {
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, ib.idxs.Bytes(), gl.Ptr(ib.idxs), gl.STATIC_DRAW)
}

// Delete deletes the GPU resources associated with this buffer
// (requires Activate to re-establish a new one).
// Should be called prior to Go object being deleted
// (ref counting can be done externally).
func (ib *IndexesBuffer) Delete() {
	if !ib.init {
		return
	}
	gl.DeleteBuffers(1, &ib.handle)
	ib.handle = 0
	ib.init = false
}
