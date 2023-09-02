// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"log"

	vk "github.com/goki/vulkan"
	"goki.dev/ki/v2/kit"
)

// OptionStates are options for the physical device features
type OptionStates int32

const (
	// Disabled -- option is not enabled
	Disabled OptionStates = iota

	// Optional -- option is enabled if possible
	// and code checks for actual state providing
	// workaround if not supported
	Optional

	// Required -- option is required and GPU.Config
	// fails if not supported by the hardware
	Required

	// Enabled is the state of all options specified
	// during Config, and supported bythe hardware
	Enabled

	OptionStatesN
)

// GPUOptions specifies supported options for the vgpu device
// upon initialization.  Several vulkan device features are
// automatically enabled, which are required for the
// basic functionality of vgpu supported graphics,
// but these are optional and may be required for
// other uses (e.g., compute shaders).
// See also InstanceExts, DeviceExts, and ValidationLayers.
type CPUOptions int

const (
	// OptRobustBufferAccess specifies that accesses to buffers are bounds-checked against the range of the buffer descriptor (as determined by VkDescriptorBufferInfo::range, VkBufferViewCreateInfo::range, or the size of the buffer). Out of bounds accesses must not cause application termination, and the effects of shader loads, stores, and atomics must conform to an implementation-dependent behavior as described below.
	OptRobustBufferAccess CPUOptions = iota

	// OptFullDrawIndexUint32 specifies the full 32-bit range of indices is supported for indexed draw calls when using a VkIndexType of VK_INDEX_TYPE_UINT32. maxDrawIndexedIndexValue is the maximum index value that may be used (aside from the primitive restart index, which is always 232-1 when the VkIndexType is VK_INDEX_TYPE_UINT32). If this feature is supported, maxDrawIndexedIndexValue must be 232-1; otherwise it must be no smaller than 224-1. See maxDrawIndexedIndexValue.
	OptFullDrawIndexUint32

	// OptImageCubeArray specifies whether image views with a VkImageViewType of VK_IMAGE_VIEW_TYPE_CUBE_ARRAY can be created, and that the corresponding SampledCubeArray and ImageCubeArray SPIR-V capabilities can be used in shader code.
	OptImageCubeArray

	// OptIndependentBlend specifies whether the VkPipelineColorBlendAttachmentState settings are controlled independently per-attachment. If this feature is not enabled, the VkPipelineColorBlendAttachmentState settings for all color attachments must be identical. Otherwise, a different VkPipelineColorBlendAttachmentState can be provided for each bound color attachment.
	OptIndependentBlend

	// OptGeometryShader specifies whether geometry shaders are supported. If this feature is not enabled, the VK_SHADER_STAGE_GEOMETRY_BIT and VK_PIPELINE_STAGE_GEOMETRY_SHADER_BIT enum values must not be used. This also specifies whether shader modules can declare the Geometry capability.
	OptGeometryShader

	// OptTessellationShader specifies whether tessellation control and evaluation shaders are supported. If this feature is not enabled, the VK_SHADER_STAGE_TESSELLATION_CONTROL_BIT, VK_SHADER_STAGE_TESSELLATION_EVALUATION_BIT, VK_PIPELINE_STAGE_TESSELLATION_CONTROL_SHADER_BIT, VK_PIPELINE_STAGE_TESSELLATION_EVALUATION_SHADER_BIT, and VK_STRUCTURE_TYPE_PIPELINE_TESSELLATION_STATE_CREATE_INFO enum values must not be used. This also specifies whether shader modules can declare the Tessellation capability.
	OptTessellationShader

	// OptSampleRateShading specifies whether Sample Shading and multisample interpolation are supported. If this feature is not enabled, the sampleShadingEnable member of the VkPipelineMultisampleStateCreateInfo structure must be set to VK_FALSE and the minSampleShading member is ignored. This also specifies whether shader modules can declare the SampleRateShading capability.
	OptSampleRateShading

	// OptDualSrcBlend specifies whether blend operations which take two sources are supported. If this feature is not enabled, the VK_BLEND_FACTOR_SRC1_COLOR, VK_BLEND_FACTOR_ONE_MINUS_SRC1_COLOR, VK_BLEND_FACTOR_SRC1_ALPHA, and VK_BLEND_FACTOR_ONE_MINUS_SRC1_ALPHA enum values must not be used as source or destination blending factors. See https://registry.khronos.org/vulkan/specs/1.3-extensions/html/vkspec.html#framebuffer-dsb.
	OptDualSrcBlend

	// OptLogicOp specifies whether logic operations are supported. If this feature is not enabled, the logicOpEnable member of the VkPipelineColorBlendStateCreateInfo structure must be set to VK_FALSE, and the logicOp member is ignored.
	OptLogicOp

	// OptMultiDrawIndirect specifies whether multiple draw indirect is supported. If this feature is not enabled, the drawCount parameter to the vkCmdDrawIndirect and vkCmdDrawIndexedIndirect commands must be 0 or 1. The maxDrawIndirectCount member of the VkPhysicalDeviceLimits structure must also be 1 if this feature is not supported. See maxDrawIndirectCount.
	OptMultiDrawIndirect

	// OptDrawIndirectFirstInstance specifies whether indirect drawing calls support the firstInstance parameter. If this feature is not enabled, the firstInstance member of all VkDrawIndirectCommand and VkDrawIndexedIndirectCommand structures that are provided to the vkCmdDrawIndirect and vkCmdDrawIndexedIndirect commands must be 0.
	OptDrawIndirectFirstInstance

	// OptDepthClamp specifies whether depth clamping is supported. If this feature is not enabled, the depthClampEnable member of the VkPipelineRasterizationStateCreateInfo structure must be set to VK_FALSE. Otherwise, setting depthClampEnable to VK_TRUE will enable depth clamping.
	OptDepthClamp

	// OptDepthBiasClamp specifies whether depth bias clamping is supported. If this feature is not enabled, the depthBiasClamp member of the VkPipelineRasterizationStateCreateInfo structure must be set to 0.0 unless the VK_DYNAMIC_STATE_DEPTH_BIAS dynamic state is enabled, and the depthBiasClamp parameter to vkCmdSetDepthBias must be set to 0.0.
	OptDepthBiasClamp

	// OptFillModeNonSolid specifies whether point and wireframe fill modes are supported. If this feature is not enabled, the VK_POLYGON_MODE_POINT and VK_POLYGON_MODE_LINE enum values must not be used.
	OptFillModeNonSolid

	// OptDepthBounds specifies whether depth bounds tests are supported. If this feature is not enabled, the depthBoundsTestEnable member of the VkPipelineDepthStencilStateCreateInfo structure must be set to VK_FALSE. When depthBoundsTestEnable is set to VK_FALSE, the minDepthBounds and maxDepthBounds members of the VkPipelineDepthStencilStateCreateInfo structure are ignored.
	OptDepthBounds

	// OptWideLines specifies whether lines with width other than 1.0 are supported. If this feature is not enabled, the lineWidth member of the VkPipelineRasterizationStateCreateInfo structure must be set to 1.0 unless the VK_DYNAMIC_STATE_LINE_WIDTH dynamic state is enabled, and the lineWidth parameter to vkCmdSetLineWidth must be set to 1.0. When this feature is supported, the range and granularity of supported line widths are indicated by the lineWidthRange and lineWidthGranularity members of the VkPhysicalDeviceLimits structure, respectively.
	OptWideLines

	// OptLargePoints specifies whether points with size greater than 1.0 are supported. If this feature is not enabled, only a point size of 1.0 written by a shader is supported. The range and granularity of supported point sizes are indicated by the pointSizeRange and pointSizeGranularity members of the VkPhysicalDeviceLimits structure, respectively.
	OptLargePoints

	// OptAlphaToOne specifies whether the implementation is able to replace the alpha value of the fragment shader color output in the Multisample Coverage fragment operation. If this feature is not enabled, then the alphaToOneEnable member of the VkPipelineMultisampleStateCreateInfo structure must be set to VK_FALSE. Otherwise setting alphaToOneEnable to VK_TRUE will enable alpha-to-one behavior.
	OptAlphaToOne

	// OptMultiViewport specifies whether more than one viewport is supported. If this feature is not enabled: The viewportCount and scissorCount members of the VkPipelineViewportStateCreateInfo structure must be set to 1. The firstViewport and viewportCount parameters to the vkCmdSetViewport command must be set to 0 and 1, respectively. The firstScissor and scissorCount parameters to the vkCmdSetScissor command must be set to 0 and 1, respectively. The exclusiveScissorCount member of the VkPipelineViewportExclusiveScissorStateCreateInfoNV structure must be set to 0 or 1. The firstExclusiveScissor and exclusiveScissorCount parameters to the vkCmdSetExclusiveScissorNV command must be set to 0 and 1, respectively.
	OptMultiViewport

	// OptSamplerAnisotropy specifies whether anisotropic filtering is supported. If this feature is not enabled, the anisotropyEnable member of the VkSamplerCreateInfo structure must be VK_FALSE.
	OptSamplerAnisotropy

	// OptTextureCompressionETC2 specifies whether all of the ETC2 and EAC compressed texture formats are supported. If this feature is enabled, then the VK_FORMAT_FEATURE_SAMPLED_IMAGE_BIT, VK_FORMAT_FEATURE_BLIT_SRC_BIT and VK_FORMAT_FEATURE_SAMPLED_IMAGE_FILTER_LINEAR_BIT features must be supported in optimalTilingFeatures for various formats -- see the Vulkan Spec at https://registry.khronos.org/vulkan/specs/1.3-extensions/man/html/VkPhysicalDeviceFeatures.html
	OptTextureCompressionETC2

	// OptTextureCompressionASTC_LDR specifies whether all of the ASTC LDR compressed texture formats are supported. If this feature is enabled, then the VK_FORMAT_FEATURE_SAMPLED_IMAGE_BIT, VK_FORMAT_FEATURE_BLIT_SRC_BIT and VK_FORMAT_FEATURE_SAMPLED_IMAGE_FILTER_LINEAR_BIT features must be supported in optimalTilingFeatures for various formats -- see the Vulkan Spec at https://registry.khronos.org/vulkan/specs/1.3-extensions/man/html/VkPhysicalDeviceFeatures.html
	OptTextureCompressionASTC_LDR

	// OptTextureCompressionBC specifies whether all of the BC compressed texture formats are supported. If this feature is enabled, then the VK_FORMAT_FEATURE_SAMPLED_IMAGE_BIT, VK_FORMAT_FEATURE_BLIT_SRC_BIT and VK_FORMAT_FEATURE_SAMPLED_IMAGE_FILTER_LINEAR_BIT features must be supported in optimalTilingFeatures for various formats -- see the Vulkan Spec at https://registry.khronos.org/vulkan/specs/1.3-extensions/man/html/VkPhysicalDeviceFeatures.html
	OptTextureCompressionBC

	// OptOcclusionQueryPrecise specifies whether occlusion queries returning actual sample counts are supported. Occlusion queries are created in a VkQueryPool by specifying the queryType of VK_QUERY_TYPE_OCCLUSION in the VkQueryPoolCreateInfo structure which is passed to vkCreateQueryPool. If this feature is enabled, queries of this type can enable VK_QUERY_CONTROL_PRECISE_BIT in the flags parameter to vkCmdBeginQuery. If this feature is not supported, the implementation supports only boolean occlusion queries. When any samples are passed, boolean queries will return a non-zero result value, otherwise a result value of zero is returned. When this feature is enabled and VK_QUERY_CONTROL_PRECISE_BIT is set, occlusion queries will report the actual number of samples passed.
	OptOcclusionQueryPrecise

	// OptPipelineStatisticsQuery specifies whether the pipeline statistics queries are supported. If this feature is not enabled, queries of type VK_QUERY_TYPE_PIPELINE_STATISTICS cannot be created, and none of the VkQueryPipelineStatisticFlagBits bits can be set in the pipelineStatistics member of the VkQueryPoolCreateInfo structure.
	OptPipelineStatisticsQuery

	// OptVertexPipelineStoresAndAtomics specifies whether storage buffers and images support stores and atomic operations in the vertex, tessellation, and geometry shader stages. If this feature is not enabled, all storage image, storage texel buffer, and storage buffer variables used by these stages in shader modules must be decorated with the NonWritable decoration (or the readonly memory qualifier in GLSL).
	OptVertexPipelineStoresAndAtomics

	// OptFragmentStoresAndAtomics specifies whether storage buffers and images support stores and atomic operations in the fragment shader stage. If this feature is not enabled, all storage image, storage texel buffer, and storage buffer variables used by the fragment stage in shader modules must be decorated with the NonWritable decoration (or the readonly memory qualifier in GLSL).
	OptFragmentStoresAndAtomics

	// OptShaderTessellationAndGeometryPointSize specifies whether the PointSize built-in decoration is available in the tessellation control, tessellation evaluation, and geometry shader stages. If this feature is not enabled, members decorated with the PointSize built-in decoration must not be read from or written to and all points written from a tessellation or geometry shader will have a size of 1.0. This also specifies whether shader modules can declare the TessellationPointSize capability for tessellation control and evaluation shaders, or if the shader modules can declare the GeometryPointSize capability for geometry shaders. An implementation supporting this feature must also support one or both of the tessellationShader or geometryShader features.
	OptShaderTessellationAndGeometryPointSize

	// OptShaderImageGatherExtended specifies whether the extended set of image gather instructions are available in shader code. If this feature is not enabled, the OpImage*Gather instructions do not support the Offset and ConstOffsets operands. This also specifies whether shader modules can declare the ImageGatherExtended capability.
	OptShaderImageGatherExtended

	// OptShaderStorageImageExtendedFormats specifies whether all the “storage image extended formats” below are supported; if this feature is supported, then the VK_FORMAT_FEATURE_STORAGE_IMAGE_BIT must be supported in optimalTilingFeatures various formats -- see the Vulkan Spec at https://registry.khronos.org/vulkan/specs/1.3-extensions/man/html/VkPhysicalDeviceFeatures.html
	OptShaderStorageImageExtendedFormats

	// OptShaderStorageImageMultisample specifies whether multisampled storage images are supported. If this feature is not enabled, images that are created with a usage that includes VK_IMAGE_USAGE_STORAGE_BIT must be created with samples equal to VK_SAMPLE_COUNT_1_BIT. This also specifies whether shader modules can declare the StorageImageMultisample and ImageMSArray capabilities.
	OptShaderStorageImageMultisample

	// OptShaderStorageImageReadWithoutFormat specifies whether storage images and storage texel buffers require a format qualifier to be specified when reading. shaderStorageImageReadWithoutFormat applies only to formats listed in the storage without format list.
	OptShaderStorageImageReadWithoutFormat

	// OptShaderStorageImageWriteWithoutFormat specifies whether storage images and storage texel buffers require a format qualifier to be specified when writing. shaderStorageImageWriteWithoutFormat applies only to formats listed in the storage without format list.
	OptShaderStorageImageWriteWithoutFormat

	// OptShaderUniformBufferArrayDynamicIndexing specifies whether arrays of uniform buffers can be indexed by dynamically uniform integer expressions in shader code. If this feature is not enabled, resources with a descriptor type of VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER or VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC must be indexed only by constant integral expressions when aggregated into arrays in shader code. This also specifies whether shader modules can declare the UniformBufferArrayDynamicIndexing capability.
	OptShaderUniformBufferArrayDynamicIndexing

	// OptShaderSampledImageArrayDynamicIndexing specifies whether arrays of samplers or sampled images can be indexed by dynamically uniform integer expressions in shader code. If this feature is not enabled, resources with a descriptor type of VK_DESCRIPTOR_TYPE_SAMPLER, VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER, or VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE must be indexed only by constant integral expressions when aggregated into arrays in shader code. This also specifies whether shader modules can declare the SampledImageArrayDynamicIndexing capability.
	OptShaderSampledImageArrayDynamicIndexing

	// OptShaderStorageBufferArrayDynamicIndexing specifies whether arrays of storage buffers can be indexed by dynamically uniform integer expressions in shader code. If this feature is not enabled, resources with a descriptor type of VK_DESCRIPTOR_TYPE_STORAGE_BUFFER or VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC must be indexed only by constant integral expressions when aggregated into arrays in shader code. This also specifies whether shader modules can declare the StorageBufferArrayDynamicIndexing capability.
	OptShaderStorageBufferArrayDynamicIndexing

	// OptShaderStorageImageArrayDynamicIndexing specifies whether arrays of storage images can be indexed by dynamically uniform integer expressions in shader code. If this feature is not enabled, resources with a descriptor type of VK_DESCRIPTOR_TYPE_STORAGE_IMAGE must be indexed only by constant integral expressions when aggregated into arrays in shader code. This also specifies whether shader modules can declare the StorageImageArrayDynamicIndexing capability.
	OptShaderStorageImageArrayDynamicIndexing

	// OptShaderClipDistance specifies whether clip distances are supported in shader code. If this feature is not enabled, any members decorated with the ClipDistance built-in decoration must not be read from or written to in shader modules. This also specifies whether shader modules can declare the ClipDistance capability.
	OptShaderClipDistance

	// OptShaderCullDistance specifies whether cull distances are supported in shader code. If this feature is not enabled, any members decorated with the CullDistance built-in decoration must not be read from or written to in shader modules. This also specifies whether shader modules can declare the CullDistance capability.
	OptShaderCullDistance

	// OptShaderFloat64 specifies whether 64-bit floats (doubles) are supported in shader code. If this feature is not enabled, 64-bit floating-point types must not be used in shader code. This also specifies whether shader modules can declare the Float64 capability. Declaring and using 64-bit floats is enabled for all storage classes that SPIR-V allows with the Float64 capability.
	OptShaderFloat64

	// OptShaderInt64 specifies whether 64-bit integers (signed and unsigned) are supported in shader code. If this feature is not enabled, 64-bit integer types must not be used in shader code. This also specifies whether shader modules can declare the Int64 capability. Declaring and using 64-bit integers is enabled for all storage classes that SPIR-V allows with the Int64 capability.
	OptShaderInt64

	// OptShaderInt16 specifies whether 16-bit integers (signed and unsigned) are supported in shader code. If this feature is not enabled, 16-bit integer types must not be used in shader code. This also specifies whether shader modules can declare the Int16 capability. However, this only enables a subset of the storage classes that SPIR-V allows for the Int16 SPIR-V capability: Declaring and using 16-bit integers in the Private, Workgroup (for non-Block variables), and Function storage classes is enabled, while declaring them in the interface storage classes (e.g., UniformConstant, Uniform, StorageBuffer, Input, Output, and PushConstant) is not enabled.
	OptShaderInt16

	// OptShaderResourceResidency specifies whether image operations that return resource residency information are supported in shader code. If this feature is not enabled, the OpImageSparse* instructions must not be used in shader code. This also specifies whether shader modules can declare the SparseResidency capability. The feature requires at least one of the sparseResidency* features to be supported.
	OptShaderResourceResidency

	// OptShaderResourceMinLod specifies whether image operations specifying the minimum resource LOD are supported in shader code. If this feature is not enabled, the MinLod image operand must not be used in shader code. This also specifies whether shader modules can declare the MinLod capability.
	OptShaderResourceMinLod

	// OptSparseBinding specifies whether resource memory can be managed at opaque sparse block level instead of at the object level. If this feature is not enabled, resource memory must be bound only on a per-object basis using the vkBindBufferMemory and vkBindImageMemory commands. In this case, buffers and images must not be created with VK_BUFFER_CREATE_SPARSE_BINDING_BIT and VK_IMAGE_CREATE_SPARSE_BINDING_BIT set in the flags member of the VkBufferCreateInfo and VkImageCreateInfo structures, respectively. Otherwise resource memory can be managed as described in Sparse Resource Features.
	OptSparseBinding

	// OptSparseResidencyBuffer specifies whether the device can access partially resident buffers. If this feature is not enabled, buffers must not be created with VK_BUFFER_CREATE_SPARSE_RESIDENCY_BIT set in the flags member of the VkBufferCreateInfo structure.
	OptSparseResidencyBuffer

	// OptSparseResidencyImage2D specifies whether the device can access partially resident 2D images with 1 sample per pixel. If this feature is not enabled, images with an imageType of VK_IMAGE_TYPE_2D and samples set to VK_SAMPLE_COUNT_1_BIT must not be created with VK_IMAGE_CREATE_SPARSE_RESIDENCY_BIT set in the flags member of the VkImageCreateInfo structure.
	OptSparseResidencyImage2D

	// OptSparseResidencyImage3D specifies whether the device can access partially resident 3D images. If this feature is not enabled, images with an imageType of VK_IMAGE_TYPE_3D must not be created with VK_IMAGE_CREATE_SPARSE_RESIDENCY_BIT set in the flags member of the VkImageCreateInfo structure.
	OptSparseResidencyImage3D

	// OptSparseResidency2Samples specifies whether the physical device can access partially resident 2D images with 2 samples per pixel. If this feature is not enabled, images with an imageType of VK_IMAGE_TYPE_2D and samples set to VK_SAMPLE_COUNT_2_BIT must not be created with VK_IMAGE_CREATE_SPARSE_RESIDENCY_BIT set in the flags member of the VkImageCreateInfo structure.
	OptSparseResidency2Samples

	// OptSparseResidency4Samples specifies whether the physical device can access partially resident 2D images with 4 samples per pixel. If this feature is not enabled, images with an imageType of VK_IMAGE_TYPE_2D and samples set to VK_SAMPLE_COUNT_4_BIT must not be created with VK_IMAGE_CREATE_SPARSE_RESIDENCY_BIT set in the flags member of the VkImageCreateInfo structure.
	OptSparseResidency4Samples

	// OptSparseResidency8Samples specifies whether the physical device can access partially resident 2D images with 8 samples per pixel. If this feature is not enabled, images with an imageType of VK_IMAGE_TYPE_2D and samples set to VK_SAMPLE_COUNT_8_BIT must not be created with VK_IMAGE_CREATE_SPARSE_RESIDENCY_BIT set in the flags member of the VkImageCreateInfo structure.
	OptSparseResidency8Samples

	// OptSparseResidency16Samples specifies whether the physical device can access partially resident 2D images with 16 samples per pixel. If this feature is not enabled, images with an imageType of VK_IMAGE_TYPE_2D and samples set to VK_SAMPLE_COUNT_16_BIT must not be created with VK_IMAGE_CREATE_SPARSE_RESIDENCY_BIT set in the flags member of the VkImageCreateInfo structure.
	OptSparseResidency16Samples

	// OptSparseResidencyAliased specifies whether the physical device can correctly access data aliased into multiple locations. If this feature is not enabled, the VK_BUFFER_CREATE_SPARSE_ALIASED_BIT and VK_IMAGE_CREATE_SPARSE_ALIASED_BIT enum values must not be used in flags members of the VkBufferCreateInfo and VkImageCreateInfo structures, respectively.
	OptSparseResidencyAliased

	// OptVariableMultisampleRate specifies whether all pipelines that will be bound to a command buffer during a subpass which uses no attachments must have the same value for VkPipelineMultisampleStateCreateInfo::rasterizationSamples. If set to VK_TRUE, the implementation supports variable multisample rates in a subpass which uses no attachments. If set to VK_FALSE, then all pipelines bound in such a subpass must have the same multisample rate. This has no effect in situations where a subpass uses any attachments.
	OptVariableMultisampleRate

	// OptInheritedQueries specifies whether a secondary command buffer may be executed while a query is active.
	OptInheritedQueries
)

//go:generate stringer -type=OptionStates
//go:generate stringer -type=CPUOptions

var KiT_OptionStates = kit.Enums.AddEnum(OptionStatesN, kit.NotBitFlag, nil)
var KiT_CPUOptions = kit.Enums.AddEnum(OptionStatesN, kit.NotBitFlag, nil)

// GPUOpts is the collection of CPUOption states
type GPUOpts map[CPUOptions]OptionStates

func (co *GPUOpts) Init() {
	if *co == nil {
		*co = make(map[CPUOptions]OptionStates)
	}
}

// Add adds the given option state
func (co *GPUOpts) Add(opt CPUOptions, state OptionStates) {
	co.Init()
	(*co)[opt] = state
}

// CopyFrom copies options from another opts collection,
// overwriting any existing in this map.
func (co *GPUOpts) CopyFrom(fm *GPUOpts) {
	if fm == nil {
		return
	}
	co.Init()
	for opt, state := range *fm {
		(*co)[opt] = state
	}
}

// State returns the state of the given option.
// Any option not explicitly set is assumed to be Disabled.
func (co *GPUOpts) State(opt CPUOptions) OptionStates {
	st, has := (*co)[opt]
	if !has {
		return Disabled
	}
	return st
}

// NewRequiredOpts returns a new GPUOpts with all
// of the given options as Required.
func NewRequiredOpts(opts ...CPUOptions) GPUOpts {
	co := make(map[CPUOptions]OptionStates)
	for _, op := range opts {
		co[op] = Required
	}
	return co
}

// CheckGPUOpts checks if the required options are present.
// if report is true, a message is printed about missing
// features, and the state of the actual
func (gp *GPU) CheckGPUOpts(feats *vk.PhysicalDeviceFeatures, opts *GPUOpts, report bool) bool {
	if opts == nil {
		return true
	}
	ok := true
	for op, st := range *opts {
		hasOpt := false
		switch op {
		case OptRobustBufferAccess:
			hasOpt = (feats.RobustBufferAccess == vk.True)
		case OptFullDrawIndexUint32:
			hasOpt = (feats.FullDrawIndexUint32 == vk.True)
		case OptImageCubeArray:
			hasOpt = (feats.ImageCubeArray == vk.True)
		case OptIndependentBlend:
			hasOpt = (feats.IndependentBlend == vk.True)
		case OptGeometryShader:
			hasOpt = (feats.GeometryShader == vk.True)
		case OptTessellationShader:
			hasOpt = (feats.TessellationShader == vk.True)
		case OptSampleRateShading:
			hasOpt = (feats.SampleRateShading == vk.True)
		case OptDualSrcBlend:
			hasOpt = (feats.DualSrcBlend == vk.True)
		case OptLogicOp:
			hasOpt = (feats.LogicOp == vk.True)
		case OptMultiDrawIndirect:
			hasOpt = (feats.MultiDrawIndirect == vk.True)
		case OptDrawIndirectFirstInstance:
			hasOpt = (feats.DrawIndirectFirstInstance == vk.True)
		case OptDepthClamp:
			hasOpt = (feats.DepthClamp == vk.True)
		case OptDepthBiasClamp:
			hasOpt = (feats.DepthBiasClamp == vk.True)
		case OptFillModeNonSolid:
			hasOpt = (feats.FillModeNonSolid == vk.True)
		case OptDepthBounds:
			hasOpt = (feats.DepthBounds == vk.True)
		case OptWideLines:
			hasOpt = (feats.WideLines == vk.True)
		case OptLargePoints:
			hasOpt = (feats.LargePoints == vk.True)
		case OptAlphaToOne:
			hasOpt = (feats.AlphaToOne == vk.True)
		case OptMultiViewport:
			hasOpt = (feats.MultiViewport == vk.True)
		case OptSamplerAnisotropy:
			hasOpt = (feats.SamplerAnisotropy == vk.True)
		case OptTextureCompressionETC2:
			hasOpt = (feats.TextureCompressionETC2 == vk.True)
		case OptTextureCompressionASTC_LDR:
			hasOpt = (feats.TextureCompressionASTC_LDR == vk.True)
		case OptTextureCompressionBC:
			hasOpt = (feats.TextureCompressionBC == vk.True)
		case OptOcclusionQueryPrecise:
			hasOpt = (feats.OcclusionQueryPrecise == vk.True)
		case OptPipelineStatisticsQuery:
			hasOpt = (feats.PipelineStatisticsQuery == vk.True)
		case OptVertexPipelineStoresAndAtomics:
			hasOpt = (feats.VertexPipelineStoresAndAtomics == vk.True)
		case OptFragmentStoresAndAtomics:
			hasOpt = (feats.FragmentStoresAndAtomics == vk.True)
		case OptShaderTessellationAndGeometryPointSize:
			hasOpt = (feats.ShaderTessellationAndGeometryPointSize == vk.True)
		case OptShaderImageGatherExtended:
			hasOpt = (feats.ShaderImageGatherExtended == vk.True)
		case OptShaderStorageImageExtendedFormats:
			hasOpt = (feats.ShaderStorageImageExtendedFormats == vk.True)
		case OptShaderStorageImageMultisample:
			hasOpt = (feats.ShaderStorageImageMultisample == vk.True)
		case OptShaderStorageImageReadWithoutFormat:
			hasOpt = (feats.ShaderStorageImageReadWithoutFormat == vk.True)
		case OptShaderStorageImageWriteWithoutFormat:
			hasOpt = (feats.ShaderStorageImageWriteWithoutFormat == vk.True)
		case OptShaderUniformBufferArrayDynamicIndexing:
			hasOpt = (feats.ShaderUniformBufferArrayDynamicIndexing == vk.True)
		case OptShaderSampledImageArrayDynamicIndexing:
			hasOpt = (feats.ShaderSampledImageArrayDynamicIndexing == vk.True)
		case OptShaderStorageBufferArrayDynamicIndexing:
			hasOpt = (feats.ShaderStorageBufferArrayDynamicIndexing == vk.True)
		case OptShaderStorageImageArrayDynamicIndexing:
			hasOpt = (feats.ShaderStorageImageArrayDynamicIndexing == vk.True)
		case OptShaderClipDistance:
			hasOpt = (feats.ShaderClipDistance == vk.True)
		case OptShaderCullDistance:
			hasOpt = (feats.ShaderCullDistance == vk.True)
		case OptShaderFloat64:
			hasOpt = (feats.ShaderFloat64 == vk.True)
		case OptShaderInt64:
			hasOpt = (feats.ShaderInt64 == vk.True)
		case OptShaderInt16:
			hasOpt = (feats.ShaderInt16 == vk.True)
		case OptShaderResourceResidency:
			hasOpt = (feats.ShaderResourceResidency == vk.True)
		case OptShaderResourceMinLod:
			hasOpt = (feats.ShaderResourceMinLod == vk.True)
		case OptSparseBinding:
			hasOpt = (feats.SparseBinding == vk.True)
		case OptSparseResidencyBuffer:
			hasOpt = (feats.SparseResidencyBuffer == vk.True)
		case OptSparseResidencyImage2D:
			hasOpt = (feats.SparseResidencyImage2D == vk.True)
		case OptSparseResidencyImage3D:
			hasOpt = (feats.SparseResidencyImage3D == vk.True)
		case OptSparseResidency2Samples:
			hasOpt = (feats.SparseResidency2Samples == vk.True)
		case OptSparseResidency4Samples:
			hasOpt = (feats.SparseResidency4Samples == vk.True)
		case OptSparseResidency8Samples:
			hasOpt = (feats.SparseResidency8Samples == vk.True)
		case OptSparseResidency16Samples:
			hasOpt = (feats.SparseResidency16Samples == vk.True)
		case OptSparseResidencyAliased:
			hasOpt = (feats.SparseResidencyAliased == vk.True)
		case OptVariableMultisampleRate:
			hasOpt = (feats.VariableMultisampleRate == vk.True)
		case OptInheritedQueries:
			hasOpt = (feats.InheritedQueries == vk.True)
		}
		switch st {
		case Required:
			if hasOpt {
				gp.EnabledOpts.Add(op, Enabled)
				if report && Debug {
					log.Printf("INFORMATION: Required vgpu Option: %s is supported\n", op.String())
				}
			} else {
				ok = false
				if report {
					log.Printf("Fatal: vgpu Option: %s is not supported, but is Required -- program cannot be run on this GPU hardware\n", op.String())
				}
			}
		case Optional:
			if hasOpt {
				gp.EnabledOpts.Add(op, Enabled)
				if report && Debug {
					log.Printf("INFORMATION: Optional vgpu Option: %s is supported\n", op.String())
				}
			} else {
				gp.EnabledOpts.Add(op, Disabled)
				if report && Debug {
					log.Printf("INFORMATION: vgpu Option: %s is not supported, but is Optional -- program will use a workaround\n", op.String())
				}
			}
		}
	}
	return ok
}

// SetGPUOpts sets the Enabled optional features in given features struct
func (gp *GPU) SetGPUOpts(feats *vk.PhysicalDeviceFeatures, opts GPUOpts) {
	for op, st := range opts {
		if st != Enabled {
			continue
		}
		switch op {
		case OptRobustBufferAccess:
			feats.RobustBufferAccess = vk.True
		case OptFullDrawIndexUint32:
			feats.FullDrawIndexUint32 = vk.True
		case OptImageCubeArray:
			feats.ImageCubeArray = vk.True
		case OptIndependentBlend:
			feats.IndependentBlend = vk.True
		case OptGeometryShader:
			feats.GeometryShader = vk.True
		case OptTessellationShader:
			feats.TessellationShader = vk.True
		case OptSampleRateShading:
			feats.SampleRateShading = vk.True
		case OptDualSrcBlend:
			feats.DualSrcBlend = vk.True
		case OptLogicOp:
			feats.LogicOp = vk.True
		case OptMultiDrawIndirect:
			feats.MultiDrawIndirect = vk.True
		case OptDrawIndirectFirstInstance:
			feats.DrawIndirectFirstInstance = vk.True
		case OptDepthClamp:
			feats.DepthClamp = vk.True
		case OptDepthBiasClamp:
			feats.DepthBiasClamp = vk.True
		case OptFillModeNonSolid:
			feats.FillModeNonSolid = vk.True
		case OptDepthBounds:
			feats.DepthBounds = vk.True
		case OptWideLines:
			feats.WideLines = vk.True
		case OptLargePoints:
			feats.LargePoints = vk.True
		case OptAlphaToOne:
			feats.AlphaToOne = vk.True
		case OptMultiViewport:
			feats.MultiViewport = vk.True
		case OptSamplerAnisotropy:
			feats.SamplerAnisotropy = vk.True
		case OptTextureCompressionETC2:
			feats.TextureCompressionETC2 = vk.True
		case OptTextureCompressionASTC_LDR:
			feats.TextureCompressionASTC_LDR = vk.True
		case OptTextureCompressionBC:
			feats.TextureCompressionBC = vk.True
		case OptOcclusionQueryPrecise:
			feats.OcclusionQueryPrecise = vk.True
		case OptPipelineStatisticsQuery:
			feats.PipelineStatisticsQuery = vk.True
		case OptVertexPipelineStoresAndAtomics:
			feats.VertexPipelineStoresAndAtomics = vk.True
		case OptFragmentStoresAndAtomics:
			feats.FragmentStoresAndAtomics = vk.True
		case OptShaderTessellationAndGeometryPointSize:
			feats.ShaderTessellationAndGeometryPointSize = vk.True
		case OptShaderImageGatherExtended:
			feats.ShaderImageGatherExtended = vk.True
		case OptShaderStorageImageExtendedFormats:
			feats.ShaderStorageImageExtendedFormats = vk.True
		case OptShaderStorageImageMultisample:
			feats.ShaderStorageImageMultisample = vk.True
		case OptShaderStorageImageReadWithoutFormat:
			feats.ShaderStorageImageReadWithoutFormat = vk.True
		case OptShaderStorageImageWriteWithoutFormat:
			feats.ShaderStorageImageWriteWithoutFormat = vk.True
		case OptShaderUniformBufferArrayDynamicIndexing:
			feats.ShaderUniformBufferArrayDynamicIndexing = vk.True
		case OptShaderSampledImageArrayDynamicIndexing:
			feats.ShaderSampledImageArrayDynamicIndexing = vk.True
		case OptShaderStorageBufferArrayDynamicIndexing:
			feats.ShaderStorageBufferArrayDynamicIndexing = vk.True
		case OptShaderStorageImageArrayDynamicIndexing:
			feats.ShaderStorageImageArrayDynamicIndexing = vk.True
		case OptShaderClipDistance:
			feats.ShaderClipDistance = vk.True
		case OptShaderCullDistance:
			feats.ShaderCullDistance = vk.True
		case OptShaderFloat64:
			feats.ShaderFloat64 = vk.True
		case OptShaderInt64:
			feats.ShaderInt64 = vk.True
		case OptShaderInt16:
			feats.ShaderInt16 = vk.True
		case OptShaderResourceResidency:
			feats.ShaderResourceResidency = vk.True
		case OptShaderResourceMinLod:
			feats.ShaderResourceMinLod = vk.True
		case OptSparseBinding:
			feats.SparseBinding = vk.True
		case OptSparseResidencyBuffer:
			feats.SparseResidencyBuffer = vk.True
		case OptSparseResidencyImage2D:
			feats.SparseResidencyImage2D = vk.True
		case OptSparseResidencyImage3D:
			feats.SparseResidencyImage3D = vk.True
		case OptSparseResidency2Samples:
			feats.SparseResidency2Samples = vk.True
		case OptSparseResidency4Samples:
			feats.SparseResidency4Samples = vk.True
		case OptSparseResidency8Samples:
			feats.SparseResidency8Samples = vk.True
		case OptSparseResidency16Samples:
			feats.SparseResidency16Samples = vk.True
		case OptSparseResidencyAliased:
			feats.SparseResidencyAliased = vk.True
		case OptVariableMultisampleRate:
			feats.VariableMultisampleRate = vk.True
		case OptInheritedQueries:
			feats.InheritedQueries = vk.True
		}
	}

}
