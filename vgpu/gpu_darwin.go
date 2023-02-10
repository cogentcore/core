// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package vgpu

import (
	"unsafe"

	vk "github.com/goki/vulkan"
)

func PlatformDefaults(gp *GPU) {
	gp.DeviceExts = append(gp.DeviceExts, []string{"VK_KHR_portability_subset"}...)
	gp.DeviceExts = append(gp.DeviceExts)
	gp.InstanceExts = append(gp.InstanceExts, vk.KhrGetPhysicalDeviceProperties2ExtensionName)
	gp.InstanceExts = append(gp.InstanceExts, vk.KhrPortabilityEnumerationExtensionName)

	gp.PlatformDeviceNext = unsafe.Pointer(&vk.PhysicalDevicePortabilitySubsetFeatures{
		SType:                                  vk.StructureTypePhysicalDevicePortabilitySubsetFeatures,
		ConstantAlphaColorBlendFactors:         vk.True,
		Events:                                 vk.True,
		ImageViewFormatReinterpretation:        vk.True,
		ImageViewFormatSwizzle:                 vk.True,
		ImageView2DOn3DImage:                   vk.False,
		MultisampleArrayImage:                  vk.True,
		MutableComparisonSamplers:              vk.True,
		PointPolygons:                          vk.False,
		SamplerMipLodBias:                      vk.False,
		SeparateStencilMaskRef:                 vk.True,
		ShaderSampleRateInterpolationFunctions: vk.True,
		TessellationIsolines:                   vk.False,
		TessellationPointMode:                  vk.False,
		TriangleFans:                           vk.False,
		VertexAttributeAccessBeyondStride:      vk.True,
	})
}
