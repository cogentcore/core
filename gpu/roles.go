// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import "github.com/cogentcore/webgpu/wgpu"

// VarRoles are the functional roles of variables,
type VarRoles int32 //enums:enum

const (
	UndefVarRole VarRoles = iota

	// Vertex is vertex shader input data: mesh geometry points, normals, etc.
	// These are automatically located in a separate Set, VertexSet (-2),
	// and managed separately.
	Vertex

	// Index is for indexes to access to Vertex data, also located in VertexSet (-2).
	// Only one such Var per VarGroup should be present, and will
	// automatically be used if a value is set.
	Index

	// Push is push constants, NOT CURRENTLY SUPPORTED in WebGPU.
	// They have a minimum of 128 bytes and are
	// stored directly in the command buffer. They do not require any
	// host-device synchronization or buffering, and are fully dynamic.
	// They are ideal for transformation matricies or indexes for accessing data.
	// They are stored in a special PushSet (-1) and managed separately.
	Push

	// Uniform is read-only general purpose data, with a more limited capacity.
	// Compared to Storage, Uniform items can be put in local cache for each
	// shader and thus can be much faster to access. Use for a smaller number
	// of parameters such as transformation matricies.
	Uniform

	// Storage is read-write general purpose data.  This is a larger but slower
	// pool of memory, with more flexible alignment constraints, used primarily
	// for compute data.
	Storage

	// StorageTexture is read-write storage-based texture data, for compute shaders
	// that operate on image data, not for standard use of textures in fragment
	// shader to texturize objects (which is SampledTexture).
	StorageTexture

	// SampledTexture is a Texture + Sampler that is used to texturize objects
	// in the fragment shader.  The variable for this specifies the role for
	// the texture (typically there is just one main such texture), and
	// the different Values of the variable hold each instance, with
	// binding used to switch which texture to use.
	// The Texture takes the first Binding position, and the Sampler is +1.
	SampledTexture
)

// IsDynamic returns true if role has dynamic offset binding
func (vr VarRoles) IsDynamic() bool {
	return vr == Uniform || vr == Storage
}

func (vr VarRoles) BindingType() wgpu.BufferBindingType {
	return RoleBindingTypes[vr]
}

var RoleBindingTypes = map[VarRoles]wgpu.BufferBindingType{
	UndefVarRole:   wgpu.BufferBindingTypeStorage,
	Uniform:        wgpu.BufferBindingTypeUniform,
	Storage:        wgpu.BufferBindingTypeStorage,
	StorageTexture: wgpu.BufferBindingTypeStorage,
	// also defined: wgpu.BufferBindingTypeReadOnlyStorage,
}

func (vr VarRoles) BufferUsages() wgpu.BufferUsage {
	return RoleBufferUsages[vr]
}

// RoleBufferUsages maps VarRoles into buffer usage flags
var RoleBufferUsages = map[VarRoles]wgpu.BufferUsage{
	Vertex:         wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	Index:          wgpu.BufferUsageIndex | wgpu.BufferUsageCopyDst,
	Uniform:        wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	Storage:        wgpu.BufferUsageStorage | wgpu.BufferUsageCopyDst | wgpu.BufferUsageCopySrc,
	StorageTexture: wgpu.BufferUsageStorage | wgpu.BufferUsageCopyDst | wgpu.BufferUsageCopySrc,
}
