// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package vgpu

import vk "github.com/goki/vulkan"

func PlatformDefaults(gp *GPU) {
	gp.DeviceExts = append(gp.DeviceExts, "VK_KHR_portability_subset")
	gp.InstanceExts = append(gp.InstanceExts, vk.KhrGetPhysicalDeviceProperties2ExtensionName)
	gp.InstanceExts = append(gp.InstanceExts, vk.KhrPortabilityEnumerationExtensionName)
}
