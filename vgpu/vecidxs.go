// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"unsafe"

	"github.com/goki/gi/oswin/gpu"
)

// todo: rename VectorsBuffer -> Vectors, Indexes

// VecIdxs maintains related VectorsBuffer and IndexesBuffer
// and corresponds to the VAO (Vertex Array Object) for OpenGL
// which holds these active buffer pointers.
// A typical Shape / Object / Geom will just have this.
// All management, transfer, freeing takes place at level of Memory object!
type VecIdxs struct {
	init bool
	vecs *VectorsBuffer
	idxs *IndexesBuffer
}

// AddVectorsBuffer makes a new VectorsBuffer to contain Vectors.
func (bm *VecIdxs) AddVectorsBuffer(usg gpu.VectorUsages) gpu.VectorsBuffer {
	bm.vecs = &VectorsBuffer{usage: usg}
	return bm.vecs
}

// VectorsBuffer returns the VectorsBuffer for this mgr
func (bm *VecIdxs) VectorsBuffer() gpu.VectorsBuffer {
	return bm.vecs
}

// AddIndexesBuffer makes a new IndexesBuffer to contain Indexes.
func (bm *VecIdxs) AddIndexesBuffer(usg gpu.VectorUsages) gpu.IndexesBuffer {
	bm.idxs = &IndexesBuffer{}
	return bm.idxs
}

// IndexesBuffer returns the IndexesBuffer for this mgr
func (bm *VecIdxs) IndexesBuffer() gpu.IndexesBuffer {
	return bm.idxs
}

// MemSize returns size in bytes of total memory required
func (bm *VecIdxs) MemSize() int {
	sz := 0
	if bm.idxs != nil {
		sz += bm.idxs.MemSize()
	}
	if bm.vecs != nil {
		sz += bm.vecs.MemSize()
	}
	return sz
}

// Alloc allocates subset of Memory Buffer to each sub-buffer
func (bm *VecIdxs) Alloc(mm *Memory, offset int) int {
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

// Free nils buffer allocations
func (bm *VecIdxs) Free() {
	if bm.idxs != nil {
		bm.idxs.Free()
	}
	if bm.vecs != nil {
		bm.vecs.Free()
	}
}

// CopyBuffsToStaging copies all of the buffer source data into the CPU side staging buffer.
// this does not check for changes -- use for initial configuration.
func (bm *VecIdxs) CopyBuffsToStaging(bufPtr unsafe.Pointer) {
	if bm.idxs != nil {
		bm.idxs.CopyBuffToStaging(bufPtr)
	}
	if bm.vecs != nil {
		bm.vecs.CopyBuffToStaging(bufPtr)
	}
}

// SyncBuffsToStaging copies all of the buffer source data into the CPU side staging buffer.
// only for *vector* data marked as changed.  index data is assumed to be static.
// returns true if any was copied.
func (bm *VecIdxs) SyncBuffsToStaging(bufPtr unsafe.Pointer) *BuffAlloc {
	if bm.vecs == nil {
		return nil
	}
	if bm.vecs.SyncBuffToStaging(bufPtr) {
		return &bm.vecs.BuffAlloc
	}
	return nil
}
