// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glgpu

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/goki/gi/oswin/gpu"
)

// BufferMgr maintains VectorsBuffer and IndexesBuffer and also the critical
// VAO (Vertex Array Object) for OpenGL which holds these active buffer pointers.
// A typical Shape / Object / Geom will just have this.
// TheGPU.NewBufferMgr() returns a new buffer manager.
type bufferMgr struct {
	init   bool
	handle uint32
	vecs   *vecsBuff
	idxs   *idxsBuff
}

// AddVectorsBuffer makes a new VectorsBuffer to contain Vectors.
func (bm *bufferMgr) AddVectorsBuffer(usg gpu.VectorUsages) gpu.VectorsBuffer {
	bm.vecs = &vecsBuff{usage: usg}
	return bm.vecs
}

// VectorsBuffer returns the VectorsBuffer for this mgr
func (bm *bufferMgr) VectorsBuffer() gpu.VectorsBuffer {
	return bm.vecs
}

// AddIndexesBuffer makes a new IndexesBuffer to contain Indexes.
func (bm *bufferMgr) AddIndexesBuffer(usg gpu.VectorUsages) gpu.IndexesBuffer {
	bm.idxs = &idxsBuff{}
	return bm.idxs
}

// IndexesBuffer returns the IndexesBuffer for this mgr
func (bm *bufferMgr) IndexesBuffer() gpu.IndexesBuffer {
	return bm.idxs
}

// Activate binds buffers as active and configures as needed
func (bm *bufferMgr) Activate() {
	if !bm.init {
		gl.GenVertexArrays(1, &bm.handle)
		bm.init = true
	}
	gl.BindVertexArray(bm.handle)
	if bm.idxs != nil {
		bm.idxs.Activate()
	}
	if bm.vecs != nil {
		bm.vecs.Activate()
	}
}

// Handle returns the unique handle for this buffer manager -- only valid after Activate()
// this is the VAO
func (bm *bufferMgr) Handle() uint32 {
	return bm.handle
}

// Transfer transfers all buffer data to GPU -- Activate must have been called
// with no other such buffers activated in between.
func (bm *bufferMgr) Transfer() {
	if bm.idxs != nil {
		bm.idxs.Transfer()
	}
	if bm.vecs != nil {
		bm.vecs.Transfer()
	}
}

// Delete deletes the GPU resources associated with this buffer
// (requires Activate to re-establish a new one).
// Should be called prior to Go object being deleted
// (ref counting can be done externally).
func (bm *bufferMgr) Delete() {
	if bm.idxs != nil {
		bm.idxs.Delete()
	}
	if bm.vecs != nil {
		bm.vecs.Delete()
	}
}
