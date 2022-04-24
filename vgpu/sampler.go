// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import vk "github.com/vulkan-go/vulkan"

// ImageView represents a vulkan image view
type ImageView struct {
	Name   string       `desc:"view must be uniquely named"`
	VkView vk.ImageView `desc:"the vulkan view"`
}

func (iv *ImageView) Destroy(dev vk.Device) {
	if iv.VkView != nil {
		vk.DestroyImageView(dev, iv.VkView, nil)
		iv.VkView = nil
	}
}

// Sampler represents a vulkan image sampler
type Sampler struct {
	Name      string
	View      string     `desc:"name of the image view used for this sampler"`
	VkSampler vk.Sampler `desc:"the vulkan sampler"`
}

func (sm *Sampler) Destroy(dev vk.Device) {
	if sm.VkSampler != nil {
		vk.DestroySampler(dev, sm.VkSampler, nil)
		sm.VkSampler = nil
	}
}
