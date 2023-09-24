// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (linux && !android) || dragonfly || openbsd

package vgpu

import vk "github.com/goki/vulkan"

func PlatformDefaults(gp *GPU) {
	gp.DeviceFeaturesNeeded = &vk.PhysicalDeviceVulkan12Features{
		SType:                                        vk.StructureTypePhysicalDeviceVulkan12Features,
		DescriptorBindingVariableDescriptorCount:     vk.True,
		DescriptorBindingPartiallyBound:              vk.True,
		RuntimeDescriptorArray:                       vk.True,
		DescriptorIndexing:                           vk.True, // might not be needed?  not for phong or vdraw
		DescriptorBindingSampledImageUpdateAfterBind: vk.True, // might not be needed?  not for phong or vdraw
	}
}
