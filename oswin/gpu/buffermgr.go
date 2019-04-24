// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

// BufferMgr maintains VectorsBuffer and IndexesBuffer and also the critical
// VAO (Vertex Array Object) for OpenGL which holds these active buffer pointers.
// A typical Shape / Object / Geom will just have this.
// IMPORTANT: BufferMgr cannot be shared across contexts (windows)!
// TheGPU.NewBufferMgr() returns a new buffer manager.
type BufferMgr interface {
	// AddVectorsBuffer makes a new VectorsBuffer to contain Vectors.
	AddVectorsBuffer(usg VectorUsages) VectorsBuffer

	// VectorsBuffer returns the VectorsBuffer for this mgr
	VectorsBuffer() VectorsBuffer

	// AddIndexesBuffer makes a new IndexesBuffer to contain Indexes.
	AddIndexesBuffer(usg VectorUsages) IndexesBuffer

	// IndexesBuffer returns the IndexesBuffer for this mgr
	IndexesBuffer() IndexesBuffer

	// Activate binds buffers as active and configures as needed
	Activate()

	// Handle returns the unique handle for this buffer manager -- only valid after Activate()
	// this is the VAO
	Handle() uint32

	// Transfer transfers all buffer data to GPU (e.g., for initial upload).
	// Activate must have been called with no other such buffers activated in between.
	TransferAll()

	// TransferVectors transfers vectors buffer data to GPU -- if vector data has changed.
	// Activate must have been called with no other such buffers activated in between.
	TransferVectors()

	// TransferIndexes transfers indexes buffer data to GPU -- if indexes data has changed.
	// Activate must have been called with no other such buffers activated in between.
	TransferIndexes()

	// Delete deletes the GPU resources associated with this buffer
	// (requires Activate to re-establish a new one).
	// Should be called prior to Go object being deleted
	// (ref counting can be done externally).
	Delete()
}
