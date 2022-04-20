// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

import (
	"fmt"
	"unsafe"

	"github.com/goki/mat32"
)

// IndexesBuffer manages a buffer of indexes for index-based rendering
// (i.e., GL_ELEMENT_ARRAY_BUFFER for glDrawElements calls in OpenGL).
type IndexesBuffer struct {
	init      bool
	handle    uint32
	ln        int
	idxs      mat32.ArrayU32
	BuffAlloc BuffAlloc `desc:"buffer allocation in terms of vulkan device-side GPU buffer, for vk.CmdBindIndexBuffers"`
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

// MemSize returns total number of bytes of memory needed
func (ib *IndexesBuffer) MemSize() int {
	return ib.ln * 4
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

// example uses staging and dest bits
// https://vulkan-tutorial.com/Vertex_buffers/Index_buffer
// VK_BUFFER_USAGE_TRANSFER_DST_BIT

// Alloc allocates this buffer from overall buffer memory at given offset
func (ib *IndexesBuffer) Alloc(mm *Memory, offset int) int {
	if ib.init {
		fmt.Printf("attempting to allocate already-initialized!\n") // todo: shouldn't happen..
	}

	sz := ib.MemSize()
	ib.BuffAlloc.Set(mm, offset, sz)
	ib.init = true
	return sz
}

// Free nulls the Allocation
func (ib *IndexesBuffer) Free() {
	ib.BuffAlloc.Free()
	ib.init = false
}

// CopyBuffToStaging copies all of the buffer source data into the CPU side staging buffer.
// this does not check for changes -- use for initial configuration.
func (ib *IndexesBuffer) CopyBuffToStaging(bufPtr unsafe.Pointer) {
	BuffMemCopy(&ib.BuffAlloc, bufPtr, unsafe.Pointer(&(ib.idxs[0])))
}

// Activate binds buffer as active one
func (ib *IndexesBuffer) Activate() {
	// if !ib.init {
	// 	gl.GenBuffers(1, &ib.handle)
	// 	ib.init = true
	// }
	// gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ib.handle)
}

// Transfer transfers data to GPU -- Activate must have been called with no other
// such buffers activated in between.  Automatically uses re-specification
// strategy per: https://www.khronos.org/opengl/wiki/Buffer_Object_Streaming
// so it is safe if buffer was still being used from prior GL rendering call.
func (ib *IndexesBuffer) Transfer() {
	// gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, ib.idxs.Bytes(), gl.Ptr(ib.idxs), gl.STATIC_DRAW)
}

// Delete deletes the GPU resources associated with this buffer
// (requires Activate to re-establish a new one).
// Should be called prior to Go object being deleted
// (ref counting can be done externally).
func (ib *IndexesBuffer) Delete() {
	if !ib.init {
		return
	}
	// gl.DeleteBuffers(1, &ib.handle)
	ib.handle = 0
	ib.init = false
}
