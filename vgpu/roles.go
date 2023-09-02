// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	vk "github.com/goki/vulkan"
	"goki.dev/ki/v2/kit"
)

// VarRoles are the functional roles of variables, corresponding
// to Vertex input vectors and all the different "uniform" types
// as enumerated in vk.DescriptorType.  This does NOT map directly
// to DescriptorType because we combine vertex and uniform data
// and require a different ordering.
type VarRoles int32

const (
	UndefVarRole VarRoles = iota
	Vertex                // vertex shader input data: mesh geometry points, normals, etc.  These are automatically located in a separate Set, VertexSet (-2), and managed separately.
	Index                 // for indexed access to Vertex data, also located in VertexSet (-2) -- only one such Var per VarSet should be present -- will automatically be used if a dynamically bound val is set
	Push                  // for push constants, which have a minimum of 128 bytes and are stored directly in the command buffer -- they do not require any host-device synchronization or buffering, and are fully dynamic.  They are ideal for transformation matricies or indexes for accessing data.  They are stored in a special PushSet (-1) and managed separately.
	Uniform               // read-only general purpose data, uses UniformBufferDynamic with offset specified at binding time, not during initial configuration -- compared to Storage, Uniform items can be put in local cache for each shader and thus can be much faster to access -- use for a smaller number of parameters such as transformation matricies
	Storage               // read-write general purpose data, in StorageBufferDynamic (offset set at binding) -- this is a larger but slower pool of memory, with more flexible alignment constraints, used primarily for compute data
	UniformTexel          // read-only image-formatted data, which cannot be accessed via ImageView or Sampler -- only for rare cases where optimized image format (e.g., rgb values of specific bit count) is useful.  No Dynamic mode is available, so this can only be used for a fixed Val.
	StorageTexel          // read-write image-formatted data, which cannot be accessed via ImageView or Sampler -- only for rare cases where optimized image format (e.g., rgb values of specific bit count) is useful. No Dynamic mode is available, so this can only be used for a fixed Val.
	StorageImage          // read-write access through an ImageView (but not a Sampler) of an Image
	TextureRole           // a Texture is a CombinedImageSampler in Vulkan terminology -- a combination of a Sampler and a specific Image, which appears as a single entity in the shader.
	VarRolesN
)

//go:generate stringer -type=VarRoles

var KiT_VarRoles = kit.Enums.AddEnum(VarRolesN, kit.NotBitFlag, nil)

// IsDynamic returns true if role has dynamic offset binding
func (vr VarRoles) IsDynamic() bool {
	return vr == Uniform || vr == Storage
}

// BuffType returns type of memory buffer for this role
func (vr VarRoles) BuffType() BuffTypes {
	return RoleBuffers[vr]
}

// VkDescriptor returns the vk.DescriptorType
func (vr VarRoles) VkDescriptor() vk.DescriptorType {
	return RoleDescriptors[vr]
}

// VkDescriptorStatic returns the vk.DescriptorType for
// static variable binding type
func (vr VarRoles) VkDescriptorStatic() vk.DescriptorType {
	return StaticRoleDescriptors[vr]
}

var RoleDescriptors = map[VarRoles]vk.DescriptorType{
	Uniform:      vk.DescriptorTypeUniformBufferDynamic,
	Storage:      vk.DescriptorTypeStorageBufferDynamic,
	UniformTexel: vk.DescriptorTypeUniformTexelBuffer,
	StorageTexel: vk.DescriptorTypeStorageTexelBuffer,
	StorageImage: vk.DescriptorTypeStorageImage,
	TextureRole:  vk.DescriptorTypeCombinedImageSampler,
}

// For static variable binding
var StaticRoleDescriptors = map[VarRoles]vk.DescriptorType{
	Uniform:      vk.DescriptorTypeUniformBuffer,
	Storage:      vk.DescriptorTypeStorageBuffer,
	UniformTexel: vk.DescriptorTypeUniformTexelBuffer,
	StorageTexel: vk.DescriptorTypeStorageTexelBuffer,
	StorageImage: vk.DescriptorTypeStorageImage,
	TextureRole:  vk.DescriptorTypeCombinedImageSampler,
}

// RoleBuffers maps VarRoles onto type of memory buffer
var RoleBuffers = map[VarRoles]BuffTypes{
	UndefVarRole: StorageBuff,
	Vertex:       VtxIdxBuff,
	Index:        VtxIdxBuff,
	Uniform:      UniformBuff,
	Storage:      StorageBuff,
	UniformTexel: UniformBuff,
	StorageTexel: StorageBuff,
	StorageImage: StorageBuff,
	TextureRole:  TextureBuff,
}
