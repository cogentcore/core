// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

package vgpu

import vk "github.com/goki/vulkan"

// Set this variable to true prior to initializing the GPU
// when using the Swiftshader software emulator (used
// by the android studio emulator on macos) or other such
// emulators that fail due to the need for the standard
// DeviceFeaturesNeeded.
var AndroidSoftwareEmulator = false

func PlatformDefaults(gp *GPU) {
	if AndroidSoftwareEmulator {
		gp.DeviceFeaturesNeeded = &vk.PhysicalDeviceVulkan12Features{
			SType: vk.StructureTypePhysicalDeviceVulkan12Features,
		}
	} else {
		gp.DeviceFeaturesNeeded = &vk.PhysicalDeviceVulkan12Features{
			SType:                                        vk.StructureTypePhysicalDeviceVulkan12Features,
			DescriptorBindingVariableDescriptorCount:     vk.True,
			DescriptorBindingPartiallyBound:              vk.True,
			RuntimeDescriptorArray:                       vk.True,
			DescriptorIndexing:                           vk.True, // might not be needed?  not for phong or vdraw
			DescriptorBindingSampledImageUpdateAfterBind: vk.True, // might not be needed?  not for phong or vdraw
		}
	}
}

func Terminate() {
}
