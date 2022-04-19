// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

import (
	"github.com/goki/gi/oswin/gpu"
)

// BufferMgr maintains VectorsBuffer and IndexesBuffer and also the critical
// VAO (Vertex Array Object) for OpenGL which holds these active buffer pointers.
// A typical Shape / Object / Geom will just have this.
// TheGPU.NewBufferMgr() returns a new buffer manager.
type BufferMgr struct {
	init bool
	vecs *VectorsBuffer
	idxs *IndexesBuffer
}

// AddVectorsBuffer makes a new VectorsBuffer to contain Vectors.
func (bm *BufferMgr) AddVectorsBuffer(usg gpu.VectorUsages) gpu.VectorsBuffer {
	bm.vecs = &VectorsBuffer{usage: usg}
	return bm.vecs
}

// VectorsBuffer returns the VectorsBuffer for this mgr
func (bm *BufferMgr) VectorsBuffer() gpu.VectorsBuffer {
	return bm.vecs
}

// AddIndexesBuffer makes a new IndexesBuffer to contain Indexes.
func (bm *BufferMgr) AddIndexesBuffer(usg gpu.VectorUsages) gpu.IndexesBuffer {
	bm.idxs = &IndexesBuffer{}
	return bm.idxs
}

// IndexesBuffer returns the IndexesBuffer for this mgr
func (bm *BufferMgr) IndexesBuffer() gpu.IndexesBuffer {
	return bm.idxs
}

// MemSize returns size in bytes of total memory required
func (bm *BufferMgr) MemSize() int {
	sz := 0
	if bm.idxs != nil {
		sz += bm.idxs.MemSize()
	}
	if bm.vecs != nil {
		sz += bm.vecs.MemSize()
	}
	return sz
}

// Alloc allocates BufferView to each sub-buffer
func (bm *BufferMgr) Alloc(mm *Memory, offset int) int {
	sz := 0
	if bm.idxs != nil {
		sz += bm.idxs.Alloc(mm, offset)
	}
	offset += sz
	if bm.vecs != nil {
		sz += bm.vecs.Alloc(mm, offset)
	}
	return sz
}

// Free frees the BufferViews in sub elements
func (bm *BufferMgr) Free(mm *Memory) {
	if bm.idxs != nil {
		bm.idxs.Free(mm)
	}
	if bm.vecs != nil {
		bm.vecs.Free(mm)
	}
}

// note: activate must happen at higher level of entire memory chunk

// Activate binds buffers as active and configures as needed
func (bm *BufferMgr) Activate() {
	// if !bm.init {
	// 	gl.GenVertexArrays(1, &bm.handle)
	// 	bm.init = true
	// }
	// gl.BindVertexArray(bm.handle)
	// if bm.idxs != nil {
	// 	bm.idxs.Activate()
	// }
	// if bm.vecs != nil {
	// 	bm.vecs.Activate()
	// }
}

// TransferAll transfers all buffer data to GPU (e.g., for initial upload).
// Activate must have been called with no other such buffers activated in between.
func (bm *BufferMgr) TransferAll() {
	if bm.idxs != nil {
		bm.idxs.Transfer()
	}
	if bm.vecs != nil {
		bm.vecs.Transfer()
	}
}

// TransferVectors transfers vectors buffer data to GPU -- if vector data has changed.
// Activate must have been called with no other such buffers activated in between.
func (bm *BufferMgr) TransferVectors() {
	if bm.vecs != nil {
		bm.vecs.Transfer()
	}
}

// TransferIndexes transfers indexes buffer data to GPU -- if indexes data has changed.
// Activate must have been called with no other such buffers activated in between.
func (bm *BufferMgr) TransferIndexes() {
	if bm.idxs != nil {
		bm.idxs.Transfer()
	}
}

// Delete deletes the GPU resources associated with this buffer
// (requires Activate to re-establish a new one).
// Should be called prior to Go object being deleted
// (ref counting can be done externally).
func (bm *BufferMgr) Delete() {
	if bm.init {
		// gl.DeleteVertexArrays(1, &bm.handle)
	}
	if bm.idxs != nil {
		bm.idxs.Delete()
	}
	if bm.vecs != nil {
		bm.vecs.Delete()
	}
	bm.init = false
}
