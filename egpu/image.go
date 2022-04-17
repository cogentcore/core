// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

import vk "github.com/vulkan-go/vulkan"

type ImageResources struct {
	Image                vk.Image
	CmdBuff              vk.CommandBuffer
	GraphicsToPresentCmd vk.CommandBuffer

	View          vk.ImageView
	Framebuffer   vk.Framebuffer
	DescriptorSet vk.DescriptorSet

	UniformBuffer vk.Buffer
	UniformMemory vk.DeviceMemory
}

func (s *ImageResources) Destroy(dev vk.Device, CmdPool ...vk.CommandPool) {
	vk.DestroyFramebuffer(dev, s.Framebuffer, nil)
	vk.DestroyImageView(dev, s.View, nil)
	if len(CmdPool) > 0 {
		vk.FreeCommandBuffers(dev, CmdPool[0], 1, []vk.CommandBuffer{
			s.CmdBuff,
		})
	}
	vk.DestroyBuffer(dev, s.UniformBuffer, nil)
	vk.FreeMemory(dev, s.UniformMemory, nil)
}

func (s *ImageResources) SetImageOwnership(graphicsQueueIndex, presentQueueIndex uint32) {
	ret := vk.BeginCommandBuffer(s.GraphicsToPresentCmd, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
		Flags: vk.CommandBufferUsageFlags(vk.CommandBufferUsageSimultaneousUseBit),
	})
	IfPanic(NewError(ret))

	vk.CmdPipelineBarrier(s.GraphicsToPresentCmd,
		vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{{
			SType:               vk.StructureTypeImageMemoryBarrier,
			DstAccessMask:       vk.AccessFlags(vk.AccessColorAttachmentWriteBit),
			OldLayout:           vk.ImageLayoutPresentSrc,
			NewLayout:           vk.ImageLayoutPresentSrc,
			SrcQueueFamilyIndex: graphicsQueueIndex,
			DstQueueFamilyIndex: presentQueueIndex,
			Image:               s.Image,

			SubresourceRange: vk.ImageSubresourceRange{
				AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
				LevelCount: 1,
				LayerCount: 1,
			},
		}})

	ret = vk.EndCommandBuffer(s.GraphicsToPresentCmd)
	IfPanic(NewError(ret))
}

func (s *ImageResources) SetUniformBuffer(buffer vk.Buffer, mem vk.DeviceMemory) {
	s.UniformBuffer = buffer
	s.UniformMemory = mem
}
