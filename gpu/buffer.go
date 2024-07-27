// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"log/slog"

	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// see: https://toji.dev/webgpu-best-practices/buffer-uploads

// Buffer is a memory buffer holding a particular type of memory.
// Only the device memory is allocated in a Buffer object.
// Host memory is just tracked by size.
type Buffer struct {
	GPU *GPU

	// type of memory in this buffer
	Type BufferTypes

	// allocated buffer size
	Size int

	// device, GPU-local memory, for computation
	Buffer *wgpu.Buffer `display:"-"`

	// alignment of offsets into this buffer
	AlignBytes int

	// true if memory has been allocated, copied, transfered
	Active bool `edit:"-"`
}

// AllocHost allocates memory for this buffer of given size in bytes,
// freeing any existing memory allocated first.
// Host and Dev buffers are made, and host memory is allocated and mapped
// for staging purposes.  Call AllocDev to allocate device memory.
// Returns true if a different memory size was allocated.
func (mb *Buffer) AllocHost(dev *Device, bsz int) bool {
	if bsz == mb.Size {
		return false
	}
	mb.Free(dev)
	mb.Size = bsz
	return true
}

// AllocDev allocates device Buffer for, using the current size.
func (mb *Buffer) AllocDev(dev *Device) error {
	if mb.Size == 0 {
		return
	}
	buf, err := dev.Device.CreateBuffer(&wgpu.BufferDescriptor{
		Size:  uint64(mb.Size),
		Label: mb.Type.String(),
		Usage: wgpu.BufferUsages[mb.Type],
	})
	if err != nil {
		slog.Error(err)
		return err
	}
	mb.Buffer = buf
}

// Free frees all memory for this buffer, including destroying
// buffers which have size associated with them.
func (mb *Buffer) Free(dev *Device) {
	if mb.Buffer == nil {
		return
	}
	mb.Buffer.Release()
	mb.Buffer = nil
	mb.Active = false
}

////////////////////////////////////////////////////////////////

// BufferTypes are memory buffer types managed by the Memory object
type BufferTypes int32 //enums:enum

// https://www.w3.org/TR/webgpu/#bind-group-layout-creation

const (
	// VertexBuffer is a buffer holding Vertex values.
	VertexBuffer BufferTypes = iota

	// IndexBuffer is a buffer holding Index values
	IndexBuffer

	// UniformBuffer holds Uniform objects: read-only, small footprint.
	UniformBuffer

	// StorageBuffer holds Storage data: read-write, larger,
	// mostly for compute shaders.
	StorageBuffer

	// TextureBuff holds StorageTextures (todo: not supported yet?).
	TextureBuff
)

// IsReadOnly returns true if buffer is read-only (most), else read-write (Storage)
func (bt BufferTypes) IsReadOnly() bool {
	if bt == StorageBuffer {
		return false
	}
	return true
}

// AlignBytes returns alignment bytes for offsets into given buffer
func (bt BufferTypes) AlignBytes(gp *GPU) int {
	switch bt {
	case StorageBuffer:
		return int(gp.GPUProperties.Limits.MinStorageBuffererOffsetAlignment)
	case UniformBuffer, IndexBuffer:
		return int(gp.GPUProperties.Limits.MinUniformBuffererOffsetAlignment)
	case TextureBuff:
		return int(gp.GPUProperties.Limits.MinTexelBufferOffsetAlignment)
	}
	return int(gp.GPUProperties.Limits.MinUniformBuffererOffsetAlignment)
}

// BufferUsages maps BufferTypes into buffer usage flags
var BufferUsages = map[BufferTypes]wgpu.BufferUsage{
	VertexBuffer:  wgpu.BufferUsage_Vertex | wgpu.BufferUsage_CopyDst,
	VertexBuffer:  wgpu.BufferUsage_Index | wgpu.BufferUsage_CopyDst,
	UniformBuffer: wgpu.BufferUsage_Uniform | wgpu.BufferUsage_CopyDst,
	StorageBuffer: wgpu.BufferUsage_Storage | wgpu.BufferUsage_CopyDst | wgpu.BufferUsage_CopySrc,
	TextureBuff:   wgpu.BufferUsage_Storage | wgpu.BufferUsage_CopyDst | wgpu.BufferUsage_CopySrc,
}
