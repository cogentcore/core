// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

// OptionStates are options for the physical device features
type OptionStates int32 //enums:enum

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
)

// GPUOptions specifies supported options for the vgpu device
// upon initialization.  Several WebGPU device features are
// automatically enabled, which are required for the
// basic functionality of vgpu supported graphics,
// but these are optional and may be required for
// other uses (e.g., compute shaders).
// See also InstanceExts, DeviceExts, and ValidationLayers.
type CPUOptions int32

// const (
// https://www.w3.org/TR/webgpu/#gpufeaturename
//	enum GPUFeatureName {
//	    "depth-clip-control",
//	    "depth32float-stencil8",
//	    "texture-compression-bc",
//	    "texture-compression-bc-sliced-3d",
//	    "texture-compression-etc2",
//	    "texture-compression-astc",
//	    "timestamp-query",
//	    "indirect-first-instance",
//	    "shader-f16",
//	    "rg11b10ufloat-renderable",
//	    "bgra8unorm-storage",
//	    "float32-filterable",
//	    "clip-distances",
//	    "dual-source-blending",
//	};
// )

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

/*
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
		case OptTextureCubeArray:
			hasOpt = (feats.TextureCubeArray == vk.True)
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
		case OptShaderTextureGatherExtended:
			hasOpt = (feats.ShaderTextureGatherExtended == vk.True)
		case OptShaderStorageTextureExtendedFormats:
			hasOpt = (feats.ShaderStorageTextureExtendedFormats == vk.True)
		case OptShaderStorageTextureMultisample:
			hasOpt = (feats.ShaderStorageTextureMultisample == vk.True)
		case OptShaderStorageTextureReadWithoutFormat:
			hasOpt = (feats.ShaderStorageTextureReadWithoutFormat == vk.True)
		case OptShaderStorageTextureWriteWithoutFormat:
			hasOpt = (feats.ShaderStorageTextureWriteWithoutFormat == vk.True)
		case OptShaderUniformBuffererArrayDynamicIndexing:
			hasOpt = (feats.ShaderUniformBuffererArrayDynamicIndexing == vk.True)
		case OptShaderSampledTextureArrayDynamicIndexing:
			hasOpt = (feats.ShaderSampledTextureArrayDynamicIndexing == vk.True)
		case OptShaderStorageBuffererArrayDynamicIndexing:
			hasOpt = (feats.ShaderStorageBuffererArrayDynamicIndexing == vk.True)
		case OptShaderStorageTextureArrayDynamicIndexing:
			hasOpt = (feats.ShaderStorageTextureArrayDynamicIndexing == vk.True)
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
		case OptSparseResidencyTexture2D:
			hasOpt = (feats.SparseResidencyTexture2D == vk.True)
		case OptSparseResidencyTexture3D:
			hasOpt = (feats.SparseResidencyTexture3D == vk.True)
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
		case OptTextureCubeArray:
			feats.TextureCubeArray = vk.True
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
		case OptShaderTextureGatherExtended:
			feats.ShaderTextureGatherExtended = vk.True
		case OptShaderStorageTextureExtendedFormats:
			feats.ShaderStorageTextureExtendedFormats = vk.True
		case OptShaderStorageTextureMultisample:
			feats.ShaderStorageTextureMultisample = vk.True
		case OptShaderStorageTextureReadWithoutFormat:
			feats.ShaderStorageTextureReadWithoutFormat = vk.True
		case OptShaderStorageTextureWriteWithoutFormat:
			feats.ShaderStorageTextureWriteWithoutFormat = vk.True
		case OptShaderUniformBuffererArrayDynamicIndexing:
			feats.ShaderUniformBuffererArrayDynamicIndexing = vk.True
		case OptShaderSampledTextureArrayDynamicIndexing:
			feats.ShaderSampledTextureArrayDynamicIndexing = vk.True
		case OptShaderStorageBuffererArrayDynamicIndexing:
			feats.ShaderStorageBuffererArrayDynamicIndexing = vk.True
		case OptShaderStorageTextureArrayDynamicIndexing:
			feats.ShaderStorageTextureArrayDynamicIndexing = vk.True
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
		case OptSparseResidencyTexture2D:
			feats.SparseResidencyTexture2D = vk.True
		case OptSparseResidencyTexture3D:
			feats.SparseResidencyTexture3D = vk.True
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

*/
