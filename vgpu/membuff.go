// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"unsafe"

	"github.com/goki/ki/kit"
	vk "github.com/vulkan-go/vulkan"
)

// MemBuff is a memory buffer holding a particular type of memory
// with staging Host-based memory and Device memory
type MemBuff struct {
	Type       BuffTypes       `desc:"type of memory in this buffer"`
	Size       int             `desc:"allocated buffer size"`
	Host       vk.Buffer       `view:"-" desc:"logical descriptor for host CPU-visible memory, for staging"`
	HostMem    vk.DeviceMemory `view:"-" desc:"host CPU-visible memory, for staging"`
	Dev        vk.Buffer       `view:"-" desc:"logical descriptor for device GPU-local memory, for computation"`
	DevMem     vk.DeviceMemory `view:"-" desc:"device GPU-local memory, for computation"`
	HostPtr    unsafe.Pointer  `view:"-" desc:"memory mapped pointer into host memory -- remains mapped"`
	AlignBytes int             `desc:"alignment of offsets into this buffer"`
	Active     bool            `inactive:"+" desc:"true if memory has been allocated, copied, transfered"`
}

// BuffTypes are memory buffer types managed by the Memory object
type BuffTypes int32

const (
	// VtxIdxBuff is a buffer holding Vertex and Index values
	VtxIdxBuff BuffTypes = iota

	// UniformBuff holds Uniform and UniformTexel objects: read-only, small footprint
	UniformBuff

	// StorageBuff holds Storage and StorageTexel: read-write, larger
	// mostly for compute shaders
	StorageBuff

	// ImageBuff holds Images / Textures -- hardware optimizes allocation
	// on device side, and staging-side is general
	ImageBuff

	BuffTypesN
)

//go:generate stringer -type=BuffTypes

var KiT_BuffTypes = kit.Enums.AddEnum(BuffTypesN, kit.NotBitFlag, nil)

// IsReadOnly returns true if buffer is read-only (most), else read-write (Storage)
func (bt BuffTypes) IsReadOnly() bool {
	if bt == StorageBuff {
		return false
	}
	return true
}

// AlignBytes returns alignment bytes for offsets into given buffer
func (bt BuffTypes) AlignBytes(gp *GPU) int {
	switch bt {
	case StorageBuff:
		return int(gp.GpuProps.Limits.MinStorageBufferOffsetAlignment)
	case UniformBuff, VtxIdxBuff:
		return int(gp.GpuProps.Limits.MinUniformBufferOffsetAlignment)
	case ImageBuff:
		return int(gp.GpuProps.Limits.MinTexelBufferOffsetAlignment)
	}
	return int(gp.GpuProps.Limits.MinUniformBufferOffsetAlignment)
}

// BuffUsages maps BuffTypes into buffer usage flags
var BuffUsages = map[BuffTypes]vk.BufferUsageFlagBits{
	VtxIdxBuff:  vk.BufferUsageVertexBufferBit | vk.BufferUsageIndexBufferBit,
	UniformBuff: vk.BufferUsageUniformBufferBit | vk.BufferUsageUniformTexelBufferBit,
	StorageBuff: vk.BufferUsageStorageBufferBit | vk.BufferUsageStorageTexelBufferBit,
	ImageBuff:   0,
}
